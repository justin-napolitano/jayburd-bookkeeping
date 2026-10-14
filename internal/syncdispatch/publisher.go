package syncdispatch

import (
	"context"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
	"time"
)

// Publisher relays one SQL-enrolled book binding through a dedicated publisher login.
type Publisher struct {
	sql       adapter
	binding   string
	transport observer.Transport
}

// NewPublisher validates injected configuration without retrieving credentials or opening transactions.
func NewPublisher(db acquisition.Beginner, binding string, t observer.Transport) (*Publisher, error) {
	if db == nil || t == nil || !validID(binding) {
		return nil, failure("invalid_request", false)
	}
	return &Publisher{adapter{db, "bookkeeping_publisher"}, binding, t}, nil
}

// dispatch contains committed enqueue ownership, never authority supplied by a queue request.
type dispatch struct {
	Status        string     `json:"status"`
	OperationID   string     `json:"operation_id"`
	EntityID      string     `json:"entity_id"`
	BookID        string     `json:"book_id"`
	Generation    string     `json:"dispatch_generation"`
	TargetVersion int        `json:"target_version"`
	TaskName      string     `json:"task_name_hex"`
	Fence         *string    `json:"dispatch_fence"`
	LeaseUntil    *time.Time `json:"lease_until"`
}

// Tick performs bounded sequential recovery and relay, with no transport inside a transaction.
func (p *Publisher) Tick(parent context.Context) (observer.TickResult, error) {
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	var out observer.TickResult
	var discovered struct {
		Created int `json:"created_count"`
	}
	raw, e := p.sql.call(ctx, "SELECT integration.sync_delivery_discover($1,25)", p.binding)
	if e != nil {
		return out, e
	}
	if metadata(raw, &discovered) != nil || discovered.Created < 0 || discovered.Created > 25 {
		return out, failure("invalid_metadata", false)
	}
	out.Discovered = discovered.Created
	var candidates struct {
		IDs []string `json:"operation_ids"`
	}
	raw, e = p.sql.call(ctx, "SELECT integration.sync_delivery_candidates($1,25)", p.binding)
	if e != nil {
		return out, e
	}
	if metadata(raw, &candidates) != nil || candidates.IDs == nil || len(candidates.IDs) > 25 {
		return out, failure("invalid_metadata", false)
	}
	seen := map[string]bool{}
	for _, id := range candidates.IDs {
		if !validID(id) || seen[id] {
			return out, failure("invalid_metadata", false)
		}
		seen[id] = true
	}
	out.Candidates = len(candidates.IDs)
	for _, id := range candidates.IDs {
		var recovery struct {
			Status string `json:"status"`
		}
		raw, e = p.sql.call(ctx, "SELECT integration.sync_delivery_recover($1)", id)
		if e != nil {
			return out, e
		}
		if metadata(raw, &recovery) != nil || !oneOf(recovery.Status, "no_change", "retry_wait", "redispatched", "done", "quarantined") {
			return out, failure("invalid_metadata", false)
		}
		if oneOf(recovery.Status, "retry_wait", "done", "quarantined") {
			out.Skipped++
			continue
		}
		h, e := holder()
		if e != nil {
			return out, e
		}
		raw, e = p.sql.call(ctx, "SELECT integration.sync_delivery_dispatch_claim($1,$2)", id, h)
		if e != nil {
			return out, e
		}
		var d dispatch
		if metadata(raw, &d) != nil || d.OperationID != id || !validID(d.EntityID) || !validID(d.BookID) || d.TargetVersion != 2 || !positive(d.Generation) || d.TaskName != taskName(id, d.Generation) {
			return out, failure("invalid_metadata", false)
		}
		if oneOf(d.Status, "not_due", "done", "quarantined") {
			if d.Fence != nil || d.LeaseUntil != nil {
				return out, failure("invalid_metadata", false)
			}
			out.Skipped++
			continue
		}
		if d.Status != "claimed" || d.Fence == nil || !positive(*d.Fence) || !liveLease(d.LeaseUntil) {
			return out, failure("invalid_metadata", false)
		}
		if ctx.Err() != nil {
			return out, failure("temporarily_unavailable", true)
		}
		enqueue, stop := context.WithTimeout(ctx, 10*time.Second)
		result, transportErr := p.transport.Enqueue(enqueue, observer.Task{Name: d.TaskName, TargetVersion: 2, Body: encode(id, d.Generation)})
		expired := enqueue.Err() != nil
		stop()
		out.EnqueueCalls++
		if transportErr != nil {
			if result != "" {
				return out, failure("transport_protocol", false)
			}
			if expired {
				result = observer.Timeout
			} else {
				result = observer.Transient
			}
		}
		if !oneOf(string(result), "accepted", "already_exists", "timeout", "transient", "rejected") {
			return out, failure("transport_protocol", false)
		}
		raw, e = p.sql.call(ctx, "SELECT integration.sync_delivery_dispatch_result($1,$2,$3,$4,$5)", id, d.Generation, h, *d.Fence, string(result))
		if e != nil {
			return out, e
		}
		var r struct {
			Recorded *bool `json:"recorded"`
		}
		if metadata(raw, &r) != nil || r.Recorded == nil {
			return out, failure("invalid_metadata", false)
		}
		if *r.Recorded {
			out.Recorded++
		} else {
			out.Skipped++
		}
	}
	return out, nil
}
