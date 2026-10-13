//go:build integration

package observer

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// proofDB creates actual restricted login transactions and injects commit-reply loss only after real COMMIT.
type proofDB struct {
	t               testing.TB
	database, login string
	active          atomic.Int64
	loseFunction    string
	lost            atomic.Bool
}

// BeginTx uses the adapter's context/isolation while retaining exact session-user role separation.
func (d *proofDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	c := connect(d.t, d.database, d.login)
	tx, e := c.BeginTx(ctx, o)
	if e != nil {
		c.Close(context.Background())
		return nil, e
	}
	d.active.Add(1)
	return &proofTx{Tx: tx, db: d, connection: c}, nil
}

// proofTx tracks connection lifetime and the last owned function used by a transaction.
type proofTx struct {
	pgx.Tx
	db         *proofDB
	connection *pgx.Conn
	closed     bool
	last       string
}

// QueryRow remembers only static SQL text for deterministic fault selection.
func (t *proofTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	t.last = q
	return t.Tx.QueryRow(ctx, q, args...)
}

// end releases the real session once, ensuring queue tests can assert zero active transactions.
func (t *proofTx) end() {
	if !t.closed {
		t.closed = true
		t.db.active.Add(-1)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = t.connection.Close(ctx)
	}
}

// Commit discards one successful acknowledgement for the selected compiled function.
func (t *proofTx) Commit(ctx context.Context) error {
	e := t.Tx.Commit(ctx)
	t.end()
	if e == nil && t.db.loseFunction != "" && strings.Contains(t.last, t.db.loseFunction) && t.db.lost.CompareAndSwap(false, true) {
		return errors.New("injected lost acknowledgement")
	}
	return e
}

// Rollback forwards the adapter's independent cleanup context and always releases the session.
func (t *proofTx) Rollback(ctx context.Context) error { e := t.Tx.Rollback(ctx); t.end(); return e }

// proofClients binds both adapter identities to the fixture's actual independent logins.
func proofClients(t testing.TB, db string, transport Transport) (*Publisher, *Consumer, *proofDB, *proofDB) {
	t.Helper()
	pd := &proofDB{t: t, database: db, login: "synthetic_publisher"}
	cd := &proofDB{t: t, database: db, login: "synthetic_observer"}
	p, e := NewPublisher(pd, binding, transport)
	if e != nil {
		t.Fatal(e)
	}
	c, e := NewConsumer(cd)
	if e != nil {
		t.Fatal(e)
	}
	return p, c, pd, cd
}

// TestObserverAdapterFastConsumer proves committed claims precede enqueue and a fast completion cannot be regressed.
func TestObserverAdapterFastConsumer(t *testing.T) {
	c, db, _ := fixture(t)
	before := value(t, c, "SELECT revision::text FROM ledger.book")
	var p *Publisher
	var consumer *Consumer
	var pd, cd *proofDB
	var tasks []Task
	queue := queueFunc(func(ctx context.Context, task Task) (EnqueueResult, error) {
		if pd.active.Load() != 0 || cd.active.Load() != 0 {
			t.Fatal("queue called during transaction")
		}
		tasks = append(tasks, task)
		if e := consumer.Handle(ctx, task.Body); e != nil {
			t.Fatal(e)
		}
		return Accepted, nil
	})
	p, consumer, pd, cd = proofClients(t, db, queue)
	r, e := p.Tick(context.Background())
	if e != nil || r.Discovered != 1 || r.EnqueueCalls != 1 || r.Recorded != 1 {
		t.Fatal(r, e)
	}
	if e = consumer.Handle(context.Background(), tasks[0].Body); e != nil {
		t.Fatal(e)
	}
	if value(t, c, "SELECT state FROM integration.work_operation") != "succeeded" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "1" || value(t, c, "SELECT revision::text FROM ledger.book") != before {
		t.Fatal("duplicate or financial effect")
	}
}

// TestObserverAdapterQueueLoss exercises repeated task identity, uncertain enqueue and reversed dispatch delivery order.
func TestObserverAdapterQueueLoss(t *testing.T) {
	c, db, _ := fixture(t)
	var tasks []Task
	results := []EnqueueResult{Timeout, AlreadyExists, Accepted}
	queue := queueFunc(func(_ context.Context, task Task) (EnqueueResult, error) {
		tasks = append(tasks, task)
		return results[len(tasks)-1], nil
	})
	p, consumer, _, _ := proofClients(t, db, queue)
	for i := 0; i < 2; i++ {
		if _, e := p.Tick(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	if tasks[0].Name != tasks[1].Name {
		t.Fatal("uncertain retry changed identity")
	}
	exec(t, c, "UPDATE integration.work_dispatch SET created_at=clock_timestamp()-interval '16 minutes'")
	if _, e := p.Tick(context.Background()); e != nil {
		t.Fatal(e)
	}
	if tasks[2].Name == tasks[0].Name {
		t.Fatal("recovery failed to replace wakeup")
	}
	for _, i := range []int{2, 0, 1, 2} {
		if e := consumer.Handle(context.Background(), tasks[i].Body); e != nil {
			t.Fatal(e)
		}
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_attempt") != "1" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "1" {
		t.Fatal("reordered delivery duplicated execution")
	}
}

// TestObserverAdapterUnknownCommit re-enters scoped SQL after actual committed state with a discarded reply.
func TestObserverAdapterUnknownCommit(t *testing.T) {
	for _, stage := range []string{"discover", "dispatch_claim", "dispatch_result", "claim", "finish"} {
		t.Run(stage, func(t *testing.T) {
			c, db, _ := fixture(t)
			var tasks []Task
			queue := queueFunc(func(_ context.Context, task Task) (EnqueueResult, error) {
				tasks = append(tasks, task)
				return Accepted, nil
			})
			p, consumer, pd, cd := proofClients(t, db, queue)
			if stage == "claim" || stage == "finish" {
				if _, e := p.Tick(context.Background()); e != nil {
					t.Fatal(e)
				}
				cd.loseFunction = "observe_" + stage
				e := consumer.Handle(context.Background(), tasks[0].Body)
				requireCode(t, e, "commit_unknown")
				if stage == "claim" {
					requireCode(t, consumer.Handle(context.Background(), tasks[0].Body), "busy")
					exec(t, c, "UPDATE integration.work_operation SET lease_until=clock_timestamp()-interval '1 second'")
					id := value(t, c, "SELECT operation_id::text FROM integration.work_operation")
					if _, e = p.Recover(context.Background(), id); e != nil {
						t.Fatal(e)
					}
					exec(t, c, "UPDATE integration.work_operation SET next_eligible_at=clock_timestamp()-interval '1 second'")
				}
				if e = consumer.Handle(context.Background(), tasks[0].Body); e != nil {
					t.Fatal(e)
				}
				if value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "1" {
					t.Fatal("commit loss repeated effect")
				}
			} else {
				pd.loseFunction = "observe_" + stage
				r, e := p.Tick(context.Background())
				requireCode(t, e, "commit_unknown")
				if stage != "dispatch_result" && len(tasks) != 0 {
					t.Fatal("uncommitted claim enqueued")
				}
				if stage == "dispatch_result" && r.Recorded != 0 {
					t.Fatal("unknown result acknowledged")
				}
				if stage == "dispatch_claim" {
					if _, e = p.Tick(context.Background()); e != nil {
						t.Fatal(e)
					}
					if len(tasks) != 0 {
						t.Fatal("healthy unknown publisher claim stolen")
					}
					exec(t, c, "UPDATE integration.work_dispatch SET lease_until=clock_timestamp()-interval '1 second'")
				}
				if _, e = p.Tick(context.Background()); e != nil {
					t.Fatal(e)
				}
				if len(tasks) != 1 {
					t.Fatalf("want one enqueue got %d", len(tasks))
				}
				if e = consumer.Handle(context.Background(), tasks[0].Body); e != nil {
					t.Fatal(e)
				}
			}
			if pd.active.Load() != 0 || cd.active.Load() != 0 {
				t.Fatal("leaked transaction")
			}
		})
	}
}

// TestObserverAdapterAuthority rejects mixed SQL identities and durably suppresses revoked consumer work.
func TestObserverAdapterAuthority(t *testing.T) {
	c, db, _ := fixture(t)
	var raw []byte
	queue := queueFunc(func(_ context.Context, task Task) (EnqueueResult, error) { raw = task.Body; return Accepted, nil })
	p, consumer, _, _ := proofClients(t, db, queue)
	for _, login := range []string{"postgres", "synthetic_runtime", "synthetic_observer"} {
		bad, _ := NewPublisher(&proofDB{t: t, database: db, login: login}, binding, queue)
		_, e := bad.Discover(context.Background())
		requireCode(t, e, "work_configuration")
	}
	bad, _ := NewConsumer(&proofDB{t: t, database: db, login: "synthetic_publisher"})
	_, e := bad.Claim(context.Background(), Wakeup{testOperation, 1})
	requireCode(t, e, "work_configuration")
	if _, e = p.Tick(context.Background()); e != nil {
		t.Fatal(e)
	}
	w, e := ParseWakeup(raw)
	if e != nil {
		t.Fatal(e)
	}
	claim, e := consumer.Claim(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", service)
	status, e := consumer.Finish(context.Background(), w.OperationID, claim)
	if e != nil || status != "done" {
		t.Fatal(status, e)
	}
	if e = consumer.Handle(context.Background(), raw); e != nil {
		t.Fatal(e)
	}
	if value(t, c, "SELECT state FROM integration.work_operation") != "blocked" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "0" {
		t.Fatal("revoked observation")
	}
	_, e = p.Tick(context.Background())
	requireCode(t, e, "work_unavailable")
}

// TestObserverAdapterExpiryAndDeadline runs actual cancellation and expired-consumer recovery through production methods.
func TestObserverAdapterExpiryAndDeadline(t *testing.T) {
	c, db, _ := fixture(t)
	var raw []byte
	p, consumer, _, cd := proofClients(t, db, queueFunc(func(_ context.Context, task Task) (EnqueueResult, error) { raw = task.Body; return Accepted, nil }))
	if _, e := p.Tick(context.Background()); e != nil {
		t.Fatal(e)
	}
	w, e := ParseWakeup(raw)
	if e != nil {
		t.Fatal(e)
	}
	claim, e := consumer.Claim(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	if ok, e := consumer.Heartbeat(context.Background(), w.OperationID, claim); e != nil || !ok {
		t.Fatal(ok, e)
	}
	exec(t, c, "UPDATE integration.work_operation SET lease_until=clock_timestamp()-interval '1 second'")
	if ok, e := consumer.Heartbeat(context.Background(), w.OperationID, claim); e != nil || ok {
		t.Fatal("expired heartbeat", e)
	}
	_, e = consumer.Finish(context.Background(), w.OperationID, claim)
	requireCode(t, e, "stale_claim")
	if _, e = p.Recover(context.Background(), w.OperationID); e != nil {
		t.Fatal(e)
	}
	exec(t, c, "UPDATE integration.work_operation SET next_eligible_at=clock_timestamp()-interval '1 second'")
	lock, e := c.Begin(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	_, e = lock.Exec(context.Background(), "SELECT 1 FROM integration.work_operation FOR UPDATE")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Millisecond)
	defer cancel()
	e = consumer.Handle(ctx, raw)
	requireCode(t, e, "temporarily_unavailable")
	if cd.active.Load() != 0 {
		t.Fatal("cancel leaked connection")
	}
	if e = lock.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e = consumer.Handle(context.Background(), raw); e != nil {
		t.Fatal(e)
	}
	if value(t, c, "SELECT failed_attempt_count::text FROM integration.work_operation") != "1" {
		t.Fatal("cancel consumed budget")
	}
}

// TestObserverAdapterFailure closes one retryable execution through the ninth adapter primitive.
func TestObserverAdapterFailure(t *testing.T) {
	c, db, _ := fixture(t)
	var raw []byte
	p, consumer, _, _ := proofClients(t, db, queueFunc(func(_ context.Context, task Task) (EnqueueResult, error) { raw = task.Body; return Accepted, nil }))
	if _, e := p.Tick(context.Background()); e != nil {
		t.Fatal(e)
	}
	w, e := ParseWakeup(raw)
	if e != nil {
		t.Fatal(e)
	}
	claim, e := consumer.Claim(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	state, e := consumer.Fail(context.Background(), w.OperationID, claim)
	if e != nil || state != "retry_wait" {
		t.Fatal(state, e)
	}
	if value(t, c, "SELECT failed_attempt_count::text FROM integration.work_operation") != "1" {
		t.Fatal("failure not counted")
	}
	requireCode(t, consumer.Handle(context.Background(), raw), "not_due")
	exec(t, c, "UPDATE integration.work_operation SET next_eligible_at=clock_timestamp()-interval '1 second'")
	if e = consumer.Handle(context.Background(), raw); e != nil {
		t.Fatal(e)
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_receipt") != "2" {
		t.Fatal("retry history lost")
	}
}
