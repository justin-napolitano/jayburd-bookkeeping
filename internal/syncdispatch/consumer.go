package syncdispatch

import (
	"context"
	"errors"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
	"time"
)

// Consumer bridges committed delivery ownership to the existing acquisition runner.
type Consumer struct {
	sql    adapter
	runner Runner
}

// NewConsumer accepts only independently injected sync authority and an acquisition runner.
func NewConsumer(db acquisition.Beginner, r Runner) (*Consumer, error) {
	if db == nil || r == nil {
		return nil, failure("invalid_request", false)
	}
	return &Consumer{adapter{db, "bookkeeping_sync"}, r}, nil
}

// claim is the SQL-owned scope and stable business identity for one execution lease.
type claim struct {
	Status        string     `json:"status"`
	OperationID   string     `json:"operation_id"`
	EntityID      string     `json:"entity_id"`
	BookID        string     `json:"book_id"`
	StreamID      string     `json:"stream_id"`
	ClaimID       *string    `json:"claim_id"`
	Fence         *string    `json:"fence"`
	LeaseUntil    *time.Time `json:"lease_until"`
	DeadlineAt    *time.Time `json:"deadline_at"`
	TargetVersion int        `json:"target_version"`
}

// Handle acknowledges only a committed terminal result or SQL-owned durable retry handoff.
func (c *Consumer) Handle(parent context.Context, body []byte) error {
	w, e := observer.ParseWakeup(body)
	if e != nil || !validID(w.OperationID) {
		return failure("invalid_request", false)
	}
	ctx, cancel := context.WithTimeout(parent, 75*time.Second)
	defer cancel()
	h, e := holder()
	if e != nil {
		return e
	}
	raw, e := c.sql.call(ctx, "SELECT integration.sync_delivery_claim($1,$2,$3)", w.OperationID, w.DispatchGeneration, h)
	if e != nil {
		return e
	}
	var cl claim
	if metadata(raw, &cl) != nil || cl.OperationID != w.OperationID || !validID(cl.EntityID) || !validID(cl.BookID) || !validID(cl.StreamID) || cl.TargetVersion != 2 {
		return failure("invalid_metadata", false)
	}
	if oneOf(cl.Status, "done", "quarantined", "busy", "not_due") {
		if cl.ClaimID != nil || cl.Fence != nil || cl.LeaseUntil != nil {
			return failure("invalid_metadata", false)
		}
		if oneOf(cl.Status, "done", "quarantined") {
			return nil
		}
		return failure(cl.Status, true)
	}
	if cl.Status != "claimed" || cl.ClaimID == nil || !validID(*cl.ClaimID) || cl.Fence == nil || !positive(*cl.Fence) || !liveLease(cl.LeaseUntil) || cl.DeadlineAt == nil || !cl.DeadlineAt.After(time.Now()) || cl.DeadlineAt.After(time.Now().Add(24*time.Hour+time.Minute)) {
		return failure("invalid_metadata", false)
	}
	deadline, _ := ctx.Deadline()
	runUntil := deadline.Add(-5 * time.Second)
	if cl.LeaseUntil.Before(runUntil) {
		runUntil = cl.LeaseUntil.Add(-5 * time.Second)
	}
	if cl.DeadlineAt.Before(runUntil) {
		runUntil = cl.DeadlineAt.Add(-5 * time.Second)
	}
	// A short caller deadline cannot authorize starting external acquisition without settlement time.
	if time.Until(runUntil) < time.Second {
		return failure("temporarily_unavailable", true)
	}
	runCtx, stop := context.WithDeadline(ctx, runUntil)
	out, runErr := c.runner.Run(runCtx, acquisition.Scope{EntityID: cl.EntityID, BookID: cl.BookID, StreamID: cl.StreamID}, *cl.ClaimID)
	stopped := runCtx.Err() != nil
	stop()
	disposition, delay := classify(out, runErr)
	if stopped {
		disposition = "unknown"
		delay = 0
	}
	raw, e = c.sql.call(ctx, "SELECT integration.sync_delivery_settle($1,$2,$3,$4,$5)", w.OperationID, h, *cl.Fence, disposition, delay)
	if e != nil {
		return e
	}
	var settled struct {
		Status string `json:"status"`
	}
	if metadata(raw, &settled) != nil || !oneOf(settled.Status, "done", "retry_wait", "quarantined", "stale") {
		return failure("invalid_metadata", false)
	}
	if settled.Status == "stale" {
		return failure("stale_attempt", true)
	}
	return nil
}

// classify reduces runner outcomes to hints; SQL independently inspects completion and live acquisition leases.
func classify(out acquisition.Outcome, err error) (string, int) {
	if err == nil {
		if out.Status == "completed" {
			return "completed", 0
		}
		return "unknown", 0
	}
	var f *acquisition.Failure
	if !errors.As(err, &f) {
		return "unknown", 0
	}
	switch f.Code {
	case "commit_unknown", "cancelled", "temporarily_unavailable", "busy", "stale_attempt", "cleanup_required":
		return "unknown", 0
	case "provider_transient", "pagination_mutation", "credentials_unavailable", "evidence_unavailable":
		if out.RetryAfter < 0 || out.RetryAfter > 24*time.Hour {
			return "permanent", 0
		}
		seconds := int((out.RetryAfter + time.Second - 1) / time.Second)
		return "retryable", seconds
	case "invalid_request", "not_authorized", "evidence_conflict", "evidence_deleted", "invalid_evidence", "evidence_missing", "invalid_metadata", "invalid_update", "limit_exceeded", "attempt_terminal", "decoder_rejected", "provider_rejected", "configuration_error", "invalid_provider_request", "retry_deferred":
		return "permanent", 0
	default:
		return "unknown", 0
	}
}
