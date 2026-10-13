package observer

import (
	"context"
	"github.com/jackc/pgx/v5"
	"time"
)

// Consumer owns only receipt observation calls through a consumer-bound database pool.
type Consumer struct{ sql adapter }

// NewConsumer validates an injected pool; it never accepts an actor identity or financial payload.
func NewConsumer(db Beginner) (*Consumer, error) {
	if db == nil {
		return nil, reject()
	}
	return &Consumer{adapter{db, "bookkeeping_observer"}}, nil
}

// Claim identifies one committed SQL execution and cannot outlive its server-validated lease.
type Claim struct {
	Status              string
	ExecutionGeneration int64
	Fence               int64
	LeaseUntil          time.Time
	Holder              string
}

// Claim obtains a new execution or a fixed no-data status, keeping holder identity stable across retries.
func (c *Consumer) Claim(ctx context.Context, w Wakeup) (Claim, error) {
	if !canonicalUUID.MatchString(w.OperationID) || w.DispatchGeneration < 1 {
		return Claim{}, reject()
	}
	h, err := newHolder()
	if err != nil {
		return Claim{}, err
	}
	var out Claim
	err = c.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		out = Claim{}
		var g, f *int64
		var lease *time.Time
		if e := tx.QueryRow(ctx, "SELECT status,execution_generation,fence_token,lease_until FROM integration.observe_claim($1,$2,$3)", w.OperationID, w.DispatchGeneration, h).Scan(&out.Status, &g, &f, &lease); e != nil {
			return e
		}
		if member(out.Status, "done", "busy", "not_due") {
			if g != nil || f != nil || lease != nil {
				return reject()
			}
			return nil
		}
		if out.Status != "claimed" || g == nil || *g < 1 || f == nil || *f < 1 || !leaseValid(lease) {
			return reject()
		}
		out = Claim{out.Status, *g, *f, *lease, h}
		return nil
	})
	if err != nil {
		return Claim{}, err
	}
	return out, nil
}

// validClaim rejects malformed metadata locally; SQL remains the authority for current fences and expiry.
func validClaim(id string, c Claim) bool {
	return canonicalUUID.MatchString(id) && canonicalUUID.MatchString(c.Holder) && c.ExecutionGeneration > 0 && c.Fence > 0 && c.Status == "claimed"
}

// Heartbeat cannot extend an expired execution or the fixed ninety-second SQL attempt budget.
func (c *Consumer) Heartbeat(ctx context.Context, id string, claim Claim) (bool, error) {
	if !validClaim(id, claim) {
		return false, reject()
	}
	var ok bool
	err := c.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT integration.observe_heartbeat($1,$2,$3,$4)", id, claim.Holder, claim.ExecutionGeneration, claim.Fence).Scan(&ok)
	})
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Finish observes the immutable journal fact and exposes success only after receipt commit.
func (c *Consumer) Finish(ctx context.Context, id string, claim Claim) (string, error) {
	return c.close(ctx, id, claim, false)
}

// Fail reports only the fixed retryable internal outcome, never arbitrary exception or provider text.
func (c *Consumer) Fail(ctx context.Context, id string, claim Claim) (string, error) {
	return c.close(ctx, id, claim, true)
}

// close restricts SQL function selection to compiled statements and validates fixed return codes.
func (c *Consumer) close(ctx context.Context, id string, claim Claim, failed bool) (string, error) {
	if !validClaim(id, claim) {
		return "", reject()
	}
	query := "SELECT integration.observe_finish($1,$2,$3,$4)"
	allowed := []string{"observed", "done"}
	if failed {
		query = "SELECT integration.observe_fail($1,$2,$3,$4)"
		allowed = []string{"retry_wait", "quarantined", "done"}
	}
	var s string
	err := c.sql.run(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, query, id, claim.Holder, claim.ExecutionGeneration, claim.Fence).Scan(&s); e != nil {
			return e
		}
		if !member(s, allowed...) {
			return reject()
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return s, nil
}

// Handle is receipt-only: committed claim and committed completion, with no external business step between them.
func (c *Consumer) Handle(parent context.Context, raw []byte) error {
	w, err := ParseWakeup(raw)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	claim, err := c.Claim(ctx, w)
	if err != nil {
		return err
	}
	if claim.Status == "done" {
		return nil
	}
	if claim.Status != "claimed" {
		return &Failure{Code: claim.Status, Retryable: true}
	}
	_, err = c.Finish(ctx, w.OperationID, claim)
	return err
}
