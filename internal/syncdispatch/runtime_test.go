package syncdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
	"strings"
	"testing"
	"time"
)

const testID = "11111111-1111-4111-8111-111111111111"

// fakeDB injects per-statement commit faults and records whether external calls overlap transactions.
type fakeDB struct {
	begins, active int
	commitErr      error
	faultQuery     string
	answer         func(string, []any) []byte
	last           string
	denied         bool
}

// BeginTx requires the production adapter's serializable isolation.
func (d *fakeDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable {
		panic("wrong isolation")
	}
	d.begins++
	d.active++
	return &fakeTx{db: d}, nil
}

// fakeTx models the minimal owned transaction surface.
type fakeTx struct {
	pgx.Tx
	db     *fakeDB
	closed bool
}

// Exec accepts bounded local configuration only.
func (t *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// QueryRow resolves fixture role proof or the exact protocol query.
func (t *fakeTx) QueryRow(_ context.Context, q string, a ...any) pgx.Row {
	if q != roleQuery {
		t.db.last = q
	}
	return fakeRow{db: t.db, q: q, args: a}
}

// Commit closes ownership before returning a simulated reply loss.
func (t *fakeTx) Commit(context.Context) error {
	t.closed = true
	t.db.active--
	if t.db.faultQuery == "" || strings.Contains(t.db.last, t.db.faultQuery) {
		return t.db.commitErr
	}
	return nil
}

// Rollback requires independently bounded cleanup even after cancellation.
func (t *fakeTx) Rollback(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok || ctx.Err() != nil {
		panic("invalid cleanup")
	}
	if !t.closed {
		t.db.active--
		t.closed = true
	}
	return nil
}

// fakeRow returns only synthetic metadata.
type fakeRow struct {
	db   *fakeDB
	q    string
	args []any
}

// Scan fills the adapter's exact role/JSON destinations.
func (r fakeRow) Scan(dest ...any) error {
	if r.q == roleQuery {
		*dest[0].(*bool) = !r.db.denied
	} else {
		*dest[0].(*[]byte) = r.db.answer(r.q, r.args)
	}
	return nil
}

// jsonBytes encodes fixture maps without ignoring marshal failures.
func jsonBytes(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}

// fixtureAnswer provides a complete publisher and consumer path with fixed stable identities.
func fixtureAnswer(q string, _ []any) []byte {
	switch {
	case strings.Contains(q, "sync_delivery_bindings"):
		return jsonBytes(map[string]any{"binding_ids": []string{testID}})
	case strings.Contains(q, "discover"):
		return []byte(`{"created_count":1}`)
	case strings.Contains(q, "candidates"):
		return jsonBytes(map[string]any{"operation_ids": []string{testID}})
	case strings.Contains(q, "recover"):
		return []byte(`{"status":"no_change"}`)
	case strings.Contains(q, "dispatch_claim"):
		return jsonBytes(map[string]any{"status": "claimed", "operation_id": testID, "entity_id": testID, "book_id": testID, "dispatch_generation": "1", "target_version": 2, "task_name_hex": taskName(testID, "1"), "dispatch_fence": "1", "lease_until": time.Now().Add(time.Minute)})
	case strings.Contains(q, "dispatch_result"):
		return []byte(`{"recorded":true}`)
	case strings.Contains(q, "delivery_claim"):
		return jsonBytes(map[string]any{"status": "claimed", "operation_id": testID, "entity_id": testID, "book_id": testID, "stream_id": testID, "claim_id": testID, "fence": "1", "lease_until": time.Now().Add(2 * time.Minute), "deadline_at": time.Now().Add(time.Hour), "target_version": 2})
	case strings.Contains(q, "settle"):
		return []byte(`{"status":"done"}`)
	}
	return []byte(`{}`)
}

// transportFunc exposes explicit synthetic enqueue behavior.
type transportFunc func(context.Context, observer.Task) (observer.EnqueueResult, error)

// Enqueue delegates the owned fixture callback.
func (f transportFunc) Enqueue(c context.Context, t observer.Task) (observer.EnqueueResult, error) {
	return f(c, t)
}

// runnerFunc exposes explicit synthetic acquisition behavior.
type runnerFunc func(context.Context, acquisition.Scope, string) (acquisition.Outcome, error)

// Run delegates the owned fixture callback.
func (f runnerFunc) Run(c context.Context, s acquisition.Scope, k string) (acquisition.Outcome, error) {
	return f(c, s, k)
}

// TestUnknownClaimSuppressesExternalAccess covers network and explicit unresolved SQLSTATEs.
func TestUnknownClaimSuppressesExternalAccess(t *testing.T) {
	for _, fault := range []error{errors.New("synthetic-sensitive-detail"), &pgconn.PgError{Code: "08007"}, &pgconn.PgError{Code: "40003"}} {
		for _, consumer := range []bool{false, true} {
			db := &fakeDB{commitErr: fault, answer: fixtureAnswer, faultQuery: "dispatch_claim"}
			calls := 0
			var err error
			if consumer {
				db.faultQuery = "delivery_claim"
				c, _ := NewConsumer(db, runnerFunc(func(context.Context, acquisition.Scope, string) (acquisition.Outcome, error) {
					calls++
					return acquisition.Outcome{}, nil
				}))
				err = c.Handle(context.Background(), encode(testID, "1"))
			} else {
				p, _ := NewPublisher(db, testID, transportFunc(func(context.Context, observer.Task) (observer.EnqueueResult, error) {
					calls++
					return observer.Accepted, nil
				}))
				_, err = p.Tick(context.Background())
			}
			if err == nil || err.Error() != "commit_unknown" || calls != 0 || db.active != 0 {
				t.Fatalf("unsafe claim: %v external=%d", err, calls)
			}
		}
	}
}

// TestConfirmedRollbackBounded verifies only known full rollback gets three retries.
func TestConfirmedRollbackBounded(t *testing.T) {
	for _, code := range []string{"40001", "40P01"} {
		db := &fakeDB{commitErr: &pgconn.PgError{Code: code}, answer: fixtureAnswer}
		_, e := (&adapter{db, "bookkeeping_sync"}).call(context.Background(), "SELECT fixture")
		if e == nil || db.begins != 3 || db.active != 0 {
			t.Fatalf("unbounded retries %v/%d", e, db.begins)
		}
	}
}

// TestExternalCallsOutsideTransactions verifies exact target and stable business claim after commit.
func TestExternalCallsOutsideTransactions(t *testing.T) {
	db := &fakeDB{answer: fixtureAnswer}
	p, _ := NewPublisher(db, testID, transportFunc(func(_ context.Context, task observer.Task) (observer.EnqueueResult, error) {
		if db.active != 0 || task.TargetVersion != 2 || task.Name != taskName(testID, "1") {
			t.Fatal("unsafe transport")
		}
		return observer.AlreadyExists, nil
	}))
	out, e := p.Tick(context.Background())
	if e != nil || out.Recorded != 1 || out.EnqueueCalls != 1 {
		t.Fatalf("relay: %+v %v", out, e)
	}
	c, _ := NewConsumer(db, runnerFunc(func(ctx context.Context, s acquisition.Scope, k string) (acquisition.Outcome, error) {
		if db.active != 0 || k != testID || s.StreamID != testID {
			t.Fatal("unsafe runner")
		}
		d, _ := ctx.Deadline()
		if time.Until(d) > 70*time.Second {
			t.Fatal("settle margin missing")
		}
		return acquisition.Outcome{Status: "completed"}, nil
	}))
	if e = c.Handle(context.Background(), encode(testID, "1")); e != nil {
		t.Fatal(e)
	}
}

// TestSharedPublisherBoundsAndIsolation proves that one failed binding does not block another.
func TestSharedPublisherBoundsAndIsolation(t *testing.T) {
	const second = "22222222-2222-4222-8222-222222222222"
	db := &fakeDB{answer: func(q string, args []any) []byte {
		if strings.Contains(q, "sync_delivery_bindings") {
			return jsonBytes(map[string]any{"binding_ids": []string{testID, second}})
		}
		if strings.Contains(q, "discover") {
			if len(args) != 2 || args[1] != 5 {
				t.Fatal("unbounded shared discovery")
			}
			if args[0] == testID {
				return nil
			}
			return []byte(`{"created_count":0}`)
		}
		if strings.Contains(q, "candidates") {
			return []byte(`{"operation_ids":[]}`)
		}
		return fixtureAnswer(q, args)
	}}
	p, err := NewSharedPublisher(db, transportFunc(func(context.Context, observer.Task) (observer.EnqueueResult, error) {
		t.Fatal("unexpected enqueue")
		return "", nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Tick(context.Background())
	if err == nil || db.active != 0 {
		t.Fatalf("failure not reported or transaction leaked: %v", err)
	}
	if !strings.Contains(db.last, "candidates") {
		t.Fatal("healthy binding was not visited")
	}
}

// TestRetryClassificationRedactsAndPreservesDelay closes arbitrary errors and rounds retry delays upward.
func TestRetryClassificationRedactsAndPreservesDelay(t *testing.T) {
	for _, tc := range []struct {
		code    string
		delay   time.Duration
		want    string
		seconds int
	}{{"provider_transient", 1500 * time.Millisecond, "retryable", 2}, {"provider_transient", 24 * time.Hour, "retryable", 86400}, {"provider_transient", 24*time.Hour + 1, "permanent", 0}, {"retry_deferred", 0, "permanent", 0}, {"commit_unknown", 0, "unknown", 0}, {"synthetic-token-injected-code", 0, "unknown", 0}} {
		s, n := classify(acquisition.Outcome{RetryAfter: tc.delay}, &acquisition.Failure{Code: tc.code, Retryable: true})
		if s != tc.want || n != tc.seconds {
			t.Fatalf("classification %s/%d", s, n)
		}
	}
}

// TestBoundaryAndCancellation prevents malformed wire, denied roles and tiny deadlines from invoking acquisition.
func TestBoundaryAndCancellation(t *testing.T) {
	calls := 0
	db := &fakeDB{answer: fixtureAnswer}
	c, _ := NewConsumer(db, runnerFunc(func(context.Context, acquisition.Scope, string) (acquisition.Outcome, error) {
		calls++
		return acquisition.Outcome{}, nil
	}))
	for _, body := range [][]byte{[]byte(`{}`), []byte(`{"schema_version":1,"schema_version":1}`), encode(testID, "01")} {
		if c.Handle(context.Background(), body) == nil {
			t.Fatal("accepted malformed")
		}
	}
	if db.begins != 0 {
		t.Fatal("SQL before validation")
	}
	db.denied = true
	if e := c.Handle(context.Background(), encode(testID, "1")); e == nil || e.Error() != "not_authorized" {
		t.Fatal(e)
	}
	db.denied = false
	ctx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if c.Handle(ctx, encode(testID, "1")) == nil || calls != 0 {
		t.Fatal("short context invoked runner")
	}
}

// TestUnknownSettleAndTransportProtocol forbids acknowledgements based on uncertain settlement or conflicting transport reports.
func TestUnknownSettleAndTransportProtocol(t *testing.T) {
	db := &fakeDB{answer: fixtureAnswer, commitErr: errors.New("lost"), faultQuery: "settle"}
	c, _ := NewConsumer(db, runnerFunc(func(context.Context, acquisition.Scope, string) (acquisition.Outcome, error) {
		return acquisition.Outcome{Status: "completed"}, nil
	}))
	if e := c.Handle(context.Background(), encode(testID, "1")); e == nil || e.Error() != "commit_unknown" {
		t.Fatal(e)
	}
	db.commitErr = nil
	p, _ := NewPublisher(db, testID, transportFunc(func(context.Context, observer.Task) (observer.EnqueueResult, error) {
		return observer.Accepted, errors.New("sensitive")
	}))
	if _, e := p.Tick(context.Background()); e == nil || e.Error() != "transport_protocol" {
		t.Fatal(e)
	}
}

// TestMetadataAndTerminalAcknowledgement rejects cross-profile scope and acknowledges only durable SQL statuses.
func TestMetadataAndTerminalAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(map[string]any)
		wantErr bool
	}{
		{"wrong_target", func(m map[string]any) { m["target_version"] = 1 }, true},
		{"wrong_operation", func(m map[string]any) { m["operation_id"] = "22222222-2222-4222-8222-222222222222" }, true},
		{"expired_lease", func(m map[string]any) { m["lease_until"] = time.Now().Add(-time.Second) }, true},
		{"unscoped_stream", func(m map[string]any) { m["stream_id"] = "invalid" }, true},
		{"terminal_done", func(m map[string]any) {
			m["status"] = "done"
			m["claim_id"] = nil
			m["fence"] = nil
			m["lease_until"] = nil
		}, false},
		{"terminal_quarantine", func(m map[string]any) {
			m["status"] = "quarantined"
			m["claim_id"] = nil
			m["fence"] = nil
			m["lease_until"] = nil
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			db := &fakeDB{answer: func(q string, a []any) []byte {
				raw := fixtureAnswer(q, a)
				if strings.Contains(q, "delivery_claim") {
					var m map[string]any
					_ = json.Unmarshal(raw, &m)
					tc.change(m)
					return jsonBytes(m)
				}
				return raw
			}}
			c, _ := NewConsumer(db, runnerFunc(func(context.Context, acquisition.Scope, string) (acquisition.Outcome, error) {
				calls++
				return acquisition.Outcome{}, nil
			}))
			e := c.Handle(context.Background(), encode(testID, "1"))
			if (e != nil) != tc.wantErr || calls != 0 {
				t.Fatalf("unsafe metadata %v calls=%d", e, calls)
			}
		})
	}
}

// TestDurableRetryHandoff forwards the full provider delay and waits for settlement COMMIT before acknowledging.
func TestDurableRetryHandoff(t *testing.T) {
	settled := false
	db := &fakeDB{answer: func(q string, a []any) []byte {
		if strings.Contains(q, "delivery_settle") {
			if a[3] != "retryable" || a[4] != 3601 {
				t.Fatalf("shortened retry %v", a[4])
			}
			settled = true
			return []byte(`{"status":"retry_wait"}`)
		}
		return fixtureAnswer(q, a)
	}}
	c, _ := NewConsumer(db, runnerFunc(func(context.Context, acquisition.Scope, string) (acquisition.Outcome, error) {
		return acquisition.Outcome{Status: "abandoned", RetryAfter: time.Hour + time.Millisecond}, &acquisition.Failure{Code: "provider_transient", Retryable: true}
	}))
	if e := c.Handle(context.Background(), encode(testID, "1")); e != nil || !settled || db.active != 0 {
		t.Fatalf("missing durable handoff %v", e)
	}
}
