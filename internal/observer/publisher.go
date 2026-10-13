package observer

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Task contains only a deterministic name and identifier envelope; transport owns allowlisted endpoint configuration.
type Task struct {
	Name          string
	TargetVersion int
	Body          []byte
}

// EnqueueResult records transport evidence, never durable observation success.
type EnqueueResult string

const (
	Accepted      EnqueueResult = "accepted"
	AlreadyExists EnqueueResult = "already_exists"
	Timeout       EnqueueResult = "timeout"
	Transient     EnqueueResult = "transient"
	Rejected      EnqueueResult = "rejected"
)

// Transport must honor context deadlines and never perform bookkeeping side effects.
type Transport interface {
	Enqueue(context.Context, Task) (EnqueueResult, error)
}

// Publisher is bound to one configured book mandate and a publisher-only database login.
type Publisher struct {
	sql       adapter
	binding   string
	transport Transport
}

// NewPublisher validates configuration without opening a transaction or retrieving credentials.
func NewPublisher(db Beginner, binding string, transport Transport) (*Publisher, error) {
	if db == nil || transport == nil || !canonicalUUID.MatchString(binding) {
		return nil, reject()
	}
	return &Publisher{adapter{db, "bookkeeping_publisher"}, binding, transport}, nil
}

// Dispatch is committed enqueue ownership; fields are populated only when Status is claimed.
type Dispatch struct {
	Status        string
	Generation    int64
	TargetVersion int
	TaskName      string
	Fence         int64
	LeaseUntil    time.Time
	Holder        string
}

// Discover atomically creates up to 25 missing sidecars for this configured binding.
func (p *Publisher) Discover(ctx context.Context) (int, error) {
	var n int
	err := p.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, "SELECT integration.observe_discover($1,25)", p.binding).Scan(&n); e != nil {
			return e
		}
		if n < 0 || n > 25 {
			return reject()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// Candidates returns bounded distinct IDs; they are hints and must be authorized again by SQL.
func (p *Publisher) Candidates(ctx context.Context) ([]string, error) {
	var ids []string
	err := p.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		ids = nil
		rows, e := tx.Query(ctx, "SELECT operation_id::text FROM integration.observe_candidates($1,25)", p.binding)
		if e != nil {
			return e
		}
		defer rows.Close()
		seen := map[string]bool{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				return e
			}
			if !canonicalUUID.MatchString(id) || seen[id] || len(ids) >= 25 {
				return reject()
			}
			seen[id] = true
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// Recover commits abandonment, revocation or redispatch before any transport claim.
func (p *Publisher) Recover(ctx context.Context, id string) (string, error) {
	if !canonicalUUID.MatchString(id) {
		return "", reject()
	}
	var s string
	err := p.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, "SELECT integration.observe_recover($1)", id).Scan(&s); e != nil {
			return e
		}
		if !member(s, "no_change", "retry_wait", "redispatched", "blocked", "quarantined", "done") {
			return reject()
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return s, nil
}

// DispatchClaim creates one holder before retries and validates the returned deterministic task identity.
func (p *Publisher) DispatchClaim(ctx context.Context, id string) (Dispatch, error) {
	if !canonicalUUID.MatchString(id) {
		return Dispatch{}, reject()
	}
	h, err := newHolder()
	if err != nil {
		return Dispatch{}, err
	}
	var out Dispatch
	err = p.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		out = Dispatch{}
		var g, f *int64
		var target *int
		var name *string
		var lease *time.Time
		if e := tx.QueryRow(ctx, "SELECT status,dispatch_generation,target_version,task_name_hex,publisher_fence,lease_until FROM integration.observe_dispatch_claim($1,$2)", id, h).Scan(&out.Status, &g, &target, &name, &f, &lease); e != nil {
			return e
		}
		if out.Status == "not_due" {
			if g != nil || target != nil || name != nil || f != nil || lease != nil {
				return reject()
			}
			return nil
		}
		if out.Status != "claimed" || g == nil || *g < 1 || target == nil || *target != 1 || name == nil || *name != taskName(id, *g) || f == nil || *f < 1 || !leaseValid(lease) {
			return reject()
		}
		out = Dispatch{out.Status, *g, *target, *name, *f, *lease, h}
		return nil
	})
	if err != nil {
		return Dispatch{}, err
	}
	return out, nil
}

// DispatchResult records a closed enqueue result with the original committed publisher fence.
func (p *Publisher) DispatchResult(ctx context.Context, id string, d Dispatch, result EnqueueResult) (bool, error) {
	if !canonicalUUID.MatchString(id) || !canonicalUUID.MatchString(d.Holder) || d.Generation < 1 || d.Fence < 1 || !member(string(result), "accepted", "already_exists", "timeout", "transient", "rejected") {
		return false, reject()
	}
	var ok bool
	err := p.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT integration.observe_dispatch_result($1,$2,$3,$4,$5)", id, d.Generation, d.Holder, d.Fence, string(result)).Scan(&ok)
	})
	if err != nil {
		return false, err
	}
	return ok, nil
}

// TickResult reports partial local progress, never total backlog completion or financial effects.
type TickResult struct {
	Discovered   int
	Candidates   int
	EnqueueCalls int
	Recorded     int
	Skipped      int
}

// Tick performs a bounded sequential relay; all queue calls occur after claim COMMIT and before a new SQL transaction.
func (p *Publisher) Tick(parent context.Context) (TickResult, error) {
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	var out TickResult
	var err error
	out.Discovered, err = p.Discover(ctx)
	if err != nil {
		return out, err
	}
	ids, err := p.Candidates(ctx)
	if err != nil {
		return out, err
	}
	out.Candidates = len(ids)
	for _, id := range ids {
		state, e := p.Recover(ctx, id)
		if e != nil {
			return out, e
		}
		if member(state, "retry_wait", "blocked", "quarantined", "done") {
			out.Skipped++
			continue
		}
		d, e := p.DispatchClaim(ctx, id)
		if e != nil {
			return out, e
		}
		if d.Status != "claimed" {
			out.Skipped++
			continue
		}
		raw, e := encodeWakeup(Wakeup{id, d.Generation})
		if e != nil {
			return out, e
		}
		if ctx.Err() != nil {
			return out, unavailable()
		}
		enqueueCtx, stop := context.WithTimeout(ctx, 10*time.Second)
		result, transportErr := p.transport.Enqueue(enqueueCtx, Task{d.TaskName, d.TargetVersion, raw})
		expired := enqueueCtx.Err() != nil
		stop()
		out.EnqueueCalls++
		// An error plus a claimed result is contradictory; don't persist guessed evidence.
		if transportErr != nil {
			if result != "" {
				return out, &Failure{Code: "transport_protocol"}
			}
			if expired {
				result = Timeout
			} else {
				result = Transient
			}
		}
		if !member(string(result), "accepted", "already_exists", "timeout", "transient", "rejected") {
			return out, &Failure{Code: "transport_protocol"}
		}
		recorded, e := p.DispatchResult(ctx, id, d, result)
		if e != nil {
			return out, e
		}
		if recorded {
			out.Recorded++
		} else {
			out.Skipped++
		}
	}
	return out, nil
}
