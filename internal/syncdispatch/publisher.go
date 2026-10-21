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
	shared    bool
	transport observer.Transport
}

// NewSharedPublisher selects only login-enrolled bindings through SQL on each tick.
func NewSharedPublisher(db acquisition.Beginner, t observer.Transport) (*Publisher, error) {
	if db == nil || t == nil {
		return nil, failure("invalid_request", false)
	}
	return &Publisher{sql: adapter{db, "bookkeeping_publisher"}, shared: true, transport: t}, nil
}

// NewPublisher validates injected configuration without retrieving credentials or opening transactions.
func NewPublisher(db acquisition.Beginner, binding string, t observer.Transport) (*Publisher, error) {
	if db == nil || t == nil || !validID(binding) {
		return nil, failure("invalid_request", false)
	}
	return &Publisher{sql: adapter{db, "bookkeeping_publisher"}, binding: binding, transport: t}, nil
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
	if !p.shared {
		return p.tickBinding(ctx, p.binding, 25)
	}
	var out observer.TickResult
	var selection struct {
		IDs []string `json:"binding_ids"`
	}
	raw, err := p.sql.call(ctx, "SELECT integration.sync_delivery_bindings(5)")
	if err != nil {
		return out, err
	}
	if metadata(raw, &selection) != nil || selection.IDs == nil || len(selection.IDs) > 5 {
		return out, failure("invalid_metadata", false)
	}
	seen := map[string]bool{}
	var first error
	for _, binding := range selection.IDs {
		if !validID(binding) || seen[binding] {
			return out, failure("invalid_metadata", false)
		}
		seen[binding] = true
		part, e := p.tickBinding(ctx, binding, 5)
		out.Discovered += part.Discovered
		out.Candidates += part.Candidates
		out.EnqueueCalls += part.EnqueueCalls
		out.Recorded += part.Recorded
		out.Skipped += part.Skipped
		if f, ok := e.(*Failure); ok && f.Code == "not_authorized" {
			e = nil
		}
		if e != nil && first == nil {
			first = e
		}
		if ctx.Err() != nil {
			break
		}
	}
	return out, first
}

// tickBinding rechecks one selected binding through the existing SQL authority boundary.
func (p *Publisher) tickBinding(ctx context.Context, binding string, limit int) (observer.TickResult, error) {
	var out observer.TickResult
	var discovered struct {
		Created int `json:"created_count"`
	}
	raw, e := p.sql.call(ctx, "SELECT integration.sync_delivery_discover($1,$2)", binding, limit)
	if e != nil {
		return out, e
	}
	if metadata(raw, &discovered) != nil || discovered.Created < 0 || discovered.Created > limit {
		return out, failure("invalid_metadata", false)
	}
	out.Discovered = discovered.Created
	var candidates struct {
		IDs []string `json:"operation_ids"`
	}
	raw, e = p.sql.call(ctx, "SELECT integration.sync_delivery_candidates($1,$2)", binding, limit)
	if e != nil {
		return out, e
	}
	if metadata(raw, &candidates) != nil || candidates.IDs == nil || len(candidates.IDs) > limit {
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
