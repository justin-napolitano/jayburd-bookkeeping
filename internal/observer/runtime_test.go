package observer

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const testOperation = "00000000-0000-4000-8000-000000000031"
const testBinding = "00000000-0000-4000-8000-000000000032"

// testRow supplies precise nullable SQL results or a fault without a real database.
type testRow struct {
	values []any
	err    error
}

// Scan emulates typed pgx fields while preserving nulls and caller-visible conversion errors.
func (r testRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("wrong fields")
	}
	for i, d := range dest {
		v := reflect.ValueOf(d).Elem()
		if r.values[i] == nil {
			v.SetZero()
			continue
		}
		src := reflect.ValueOf(r.values[i])
		if v.Kind() == reflect.Pointer && src.Type() != v.Type() {
			ptr := reflect.New(v.Type().Elem())
			ptr.Elem().Set(src)
			v.Set(ptr)
		} else {
			v.Set(src)
		}
	}
	return nil
}

// testRows supplies bounded candidate rows and supports the actual Query loop.
type testRows struct {
	pgx.Rows
	ids []string
	i   int
}

// Next advances the deterministic synthetic candidate stream.
func (r *testRows) Next() bool { r.i++; return r.i <= len(r.ids) }

// Scan assigns one candidate identifier.
func (r *testRows) Scan(dest ...any) error { return testRow{values: []any{r.ids[r.i-1]}}.Scan(dest...) }

// Close ends the synthetic cursor without affecting transaction ownership.
func (r *testRows) Close() {}

// Err reports an exhausted successful candidate stream.
func (r *testRows) Err() error { return nil }

// testDB injects statement/commit faults and counts active transactions around transport calls.
type testDB struct {
	begins, commits, active, rollbacks int
	queryErrors                        []error
	commitErrors                       []error
	role                               bool
	badMetadata                        bool
	holders                            []string
	cleanup                            bool
	seenOptions                        []pgx.TxOptions
	setup                              string
}

// BeginTx records the required isolation and starts a tracked transaction.
func (d *testDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	d.begins++
	d.active++
	d.seenOptions = append(d.seenOptions, o)
	return &testTx{db: d}, nil
}

// testTx overrides only the pgx operations used by the adapter.
type testTx struct {
	pgx.Tx
	db     *testDB
	closed bool
}

// Exec records all transaction-local timeout controls.
func (t *testTx) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	t.db.setup = q
	return pgconn.NewCommandTag("SET"), nil
}

// Query returns the one synthetic operation for publisher traversal.
func (t *testTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return &testRows{ids: []string{testOperation}}, nil
}

// QueryRow emits compiled-function outputs and consumes faults only for business calls.
func (t *testTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	d := t.db
	if q == roleQuery {
		return testRow{values: []any{d.role}}
	}
	if strings.Contains(q, "observe_claim") {
		d.holders = append(d.holders, args[2].(string))
	}
	if len(d.queryErrors) > 0 {
		e := d.queryErrors[0]
		d.queryErrors = d.queryErrors[1:]
		if e != nil {
			return testRow{err: e}
		}
	}
	switch {
	case strings.Contains(q, "observe_discover"):
		return testRow{values: []any{1}}
	case strings.Contains(q, "observe_recover"):
		return testRow{values: []any{"no_change"}}
	case strings.Contains(q, "observe_dispatch_claim"):
		name := taskName(testOperation, 1)
		if d.badMetadata {
			name = "bad"
		}
		return testRow{values: []any{"claimed", int64(1), 1, name, int64(1), time.Now().Add(time.Second)}}
	case strings.Contains(q, "observe_dispatch_result"), strings.Contains(q, "observe_heartbeat"):
		return testRow{values: []any{true}}
	case strings.Contains(q, "observe_claim"):
		if d.badMetadata {
			return testRow{values: []any{"busy", int64(1), nil, nil}}
		}
		return testRow{values: []any{"claimed", int64(1), int64(1), time.Now().Add(time.Second)}}
	case strings.Contains(q, "observe_finish"):
		return testRow{values: []any{"observed"}}
	case strings.Contains(q, "observe_fail"):
		return testRow{values: []any{"retry_wait"}}
	}
	return testRow{err: errors.New("unexpected query")}
}

// Commit closes the tracked transaction before optionally discarding its acknowledgement.
func (t *testTx) Commit(context.Context) error {
	t.db.commits++
	t.closed = true
	t.db.active--
	if len(t.db.commitErrors) > 0 {
		e := t.db.commitErrors[0]
		t.db.commitErrors = t.db.commitErrors[1:]
		return e
	}
	return nil
}

// Rollback confirms cleanup has a live independent bounded context even after cancellation.
func (t *testTx) Rollback(ctx context.Context) error {
	t.db.rollbacks++
	_, ok := ctx.Deadline()
	t.db.cleanup = ok && ctx.Err() == nil
	if !t.closed {
		t.closed = true
		t.db.active--
	}
	return nil
}

// queueFunc injects deterministic queue behavior without provider configuration.
type queueFunc func(context.Context, Task) (EnqueueResult, error)

// Enqueue delegates only to the test's controlled behavior.
func (f queueFunc) Enqueue(ctx context.Context, task Task) (EnqueueResult, error) {
	return f(ctx, task)
}

// requireCode checks fixed errors without retaining underlying fault messages.
func requireCode(t testing.TB, err error, want string) {
	t.Helper()
	var f *Failure
	if !errors.As(err, &f) || f.Code != want {
		t.Fatalf("want %s, got %v", want, err)
	}
}

// TestWakeupBoundary validates the exact existing message profile before any SQL or queue call.
func TestWakeupBoundary(t *testing.T) {
	raw, err := encodeWakeup(Wakeup{testOperation, 1})
	if err != nil {
		t.Fatal(err)
	}
	w, err := ParseWakeup(raw)
	if err != nil || w.OperationID != testOperation || w.DispatchGeneration != 1 {
		t.Fatal("valid envelope rejected")
	}
	for _, s := range []string{`null`, `{}`, `[]`, string(raw) + `{}`, strings.Replace(string(raw), `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1), strings.Replace(string(raw), `"schema_version":1`, `"schema_version":1.0`, 1), strings.Replace(string(raw), `"schema_version":1`, `"schema_version":"1"`, 1), strings.Replace(string(raw), `"dispatch_generation":"1"`, `"dispatch_generation":"01"`, 1), strings.Replace(string(raw), `"dispatch_generation":"1"`, `"dispatch_generation":1`, 1), strings.Replace(string(raw), `"dispatch_generation":"1"`, `"dispatch_generation":"9223372036854775808"`, 1), strings.Replace(string(raw), `"operation_id"`, `"actor"`, 1), strings.Replace(string(raw), testOperation, strings.ToUpper("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"), 1), string(raw) + strings.Repeat(" ", 1025)} {
		if _, e := ParseWakeup([]byte(s)); e == nil {
			t.Fatalf("accepted malformed envelope %q", s)
		}
	}
}

// TestAdapterTransactionFaults proves bounded full retries, stable holder IDs and unknown-commit suppression.
func TestAdapterTransactionFaults(t *testing.T) {
	for _, tc := range []struct {
		name          string
		query, commit []error
		begins        int
		want          string
	}{{"serialization", []error{&pgconn.PgError{Code: "40001"}, nil}, nil, 2, ""}, {"deadlockcommit", nil, []error{&pgconn.PgError{Code: "40P01"}, nil}, 2, ""}, {"exhausted", []error{&pgconn.PgError{Code: "40001"}, &pgconn.PgError{Code: "40001"}, &pgconn.PgError{Code: "40001"}}, nil, 3, "temporarily_unavailable"}, {"unknown", nil, []error{errors.New("protected transport diagnostic")}, 1, "commit_unknown"}, {"transaction_unknown", nil, []error{&pgconn.PgError{Code: "08007"}}, 1, "commit_unknown"}, {"statement_unknown", nil, []error{&pgconn.PgError{Code: "40003"}}, 1, "commit_unknown"}, {"denied", []error{&pgconn.PgError{Code: "P2201", Message: "protected SQL text"}}, nil, 1, "work_unavailable"}} {
		t.Run(tc.name, func(t *testing.T) {
			d := &testDB{role: true, queryErrors: tc.query, commitErrors: tc.commit}
			c, _ := NewConsumer(d)
			got, e := c.Claim(context.Background(), Wakeup{testOperation, 1})
			if tc.want != "" {
				requireCode(t, e, tc.want)
				if got != (Claim{}) {
					t.Fatal("exposed uncommitted metadata")
				}
			} else if e != nil {
				t.Fatal(e)
			}
			if d.begins != tc.begins || d.active != 0 || !d.cleanup {
				t.Fatal("transaction lifetime")
			}
			for _, o := range d.seenOptions {
				if o.IsoLevel != pgx.Serializable {
					t.Fatal("wrong isolation")
				}
			}
			for _, h := range d.holders {
				if h != d.holders[0] {
					t.Fatal("changed retry identity")
				}
			}
			if !strings.Contains(d.setup, "statement_timeout='5s'") || !strings.Contains(d.setup, "lock_timeout='1s'") {
				t.Fatal("missing timeouts")
			}
		})
	}
}

// TestPublisherCommitBeforeTransport rejects invalid metadata and observes no active transaction at enqueue.
func TestPublisherCommitBeforeTransport(t *testing.T) {
	d := &testDB{role: true}
	calls := 0
	q := queueFunc(func(ctx context.Context, task Task) (EnqueueResult, error) {
		calls++
		if d.active != 0 {
			t.Fatal("network call in transaction")
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 10*time.Second {
			t.Fatal("unbounded queue call")
		}
		if task.Name != taskName(testOperation, 1) || task.TargetVersion != 1 {
			t.Fatal("task identity")
		}
		if _, e := ParseWakeup(task.Body); e != nil {
			t.Fatal(e)
		}
		return Accepted, nil
	})
	p, _ := NewPublisher(d, testBinding, q)
	r, e := p.Tick(context.Background())
	if e != nil || r.EnqueueCalls != 1 || r.Recorded != 1 || calls != 1 {
		t.Fatal(r, e)
	}
	d.badMetadata = true
	_, e = p.Tick(context.Background())
	requireCode(t, e, "invalid_work")
	if calls != 1 {
		t.Fatal("invalid digest enqueued")
	}
}

// TestPublisherUnknownCommitNeverEnqueues stops immediately at every unknown transaction acknowledgement.
func TestPublisherUnknownCommitNeverEnqueues(t *testing.T) {
	for _, at := range []int{0, 1, 2, 3} {
		t.Run(string(rune('a'+at)), func(t *testing.T) {
			faults := make([]error, at+1)
			faults[at] = errors.New("lost ack")
			d := &testDB{role: true, commitErrors: faults}
			calls := 0
			p, _ := NewPublisher(d, testBinding, queueFunc(func(context.Context, Task) (EnqueueResult, error) { calls++; return Accepted, nil }))
			_, e := p.Tick(context.Background())
			requireCode(t, e, "commit_unknown")
			if calls != 0 {
				t.Fatal("enqueued without known claim commit")
			}
		})
	}
}

// TestTransportContradiction fails closed instead of persisting invented successful queue evidence.
func TestTransportContradiction(t *testing.T) {
	for _, result := range []EnqueueResult{Accepted, "unknown"} {
		d := &testDB{role: true}
		p, _ := NewPublisher(d, testBinding, queueFunc(func(context.Context, Task) (EnqueueResult, error) { return result, errors.New("private details") }))
		r, e := p.Tick(context.Background())
		requireCode(t, e, "transport_protocol")
		if r.Recorded != 0 {
			t.Fatal("contradiction recorded")
		}
	}
}

// TestAdapterRoleAndMetadata suppresses invalid capability and inconsistent nullable claim outputs.
func TestAdapterRoleAndMetadata(t *testing.T) {
	d := &testDB{}
	c, _ := NewConsumer(d)
	_, e := c.Claim(context.Background(), Wakeup{testOperation, 1})
	requireCode(t, e, "work_configuration")
	if d.commits != 0 {
		t.Fatal("invalid role committed")
	}
	d.role = true
	d.badMetadata = true
	_, e = c.Claim(context.Background(), Wakeup{testOperation, 1})
	requireCode(t, e, "invalid_work")
}

// TestPublisherUnknownSQLStateNeverEnqueues treats server-reported completion ambiguity like a lost commit reply.
func TestPublisherUnknownSQLStateNeverEnqueues(t *testing.T) {
	for _, sqlstate := range []string{"08007", "40003"} {
		t.Run(sqlstate, func(t *testing.T) {
			d := &testDB{role: true, commitErrors: []error{nil, nil, nil, &pgconn.PgError{Code: sqlstate}}}
			calls := 0
			p, _ := NewPublisher(d, testBinding, queueFunc(func(context.Context, Task) (EnqueueResult, error) { calls++; return Accepted, nil }))
			r, e := p.Tick(context.Background())
			requireCode(t, e, "commit_unknown")
			if calls != 0 || r.EnqueueCalls != 0 || d.begins != 4 {
				t.Fatal("ambiguous SQLSTATE replayed or enqueued")
			}
		})
	}
}
