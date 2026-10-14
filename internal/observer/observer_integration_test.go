//go:build integration

package observer

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("observer-proof", false, "Use only the runner's disposable internal database")

const entity = "00000000-0000-4000-8000-000000000002"
const book = "00000000-0000-4000-8000-000000000003"
const actor = "00000000-0000-4000-8000-000000000001"
const service = "00000000-0000-4000-8000-000000000020"
const binding = "00000000-0000-4000-8000-000000000021"
const holder = "00000000-0000-4000-8000-000000000022"
const other = "00000000-0000-4000-8000-000000000023"

// connect never accepts an inherited DSN or credential; this runs on an internal Docker network.
func connect(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if !*proof {
		t.Skip("use scripts/test-observer.sh")
	}
	c, e := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if e != nil {
		t.Fatal(e)
	}
	return c
}

// exec is restricted to explicit synthetic fixture/inspection SQL.
func exec(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal(e)
	}
}

// value inspects exact state without using runtime table privileges.
func value(t testing.TB, c *pgx.Conn, q string, args ...any) string {
	t.Helper()
	var s string
	if e := c.QueryRow(context.Background(), q, args...).Scan(&s); e != nil {
		t.Fatal(e)
	}
	return s
}

// code checks safe SQLSTATE instead of exposing arbitrary database diagnostics.
func code(t testing.TB, e error, want string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(e, &p) || p.Code != want {
		t.Fatalf("want %s got %v", want, e)
	}
}

// fixture creates an owner-provisioned service mandate and posts through the existing function.
func fixture(t *testing.T) (*pgx.Conn, string, string) {
	t.Helper()
	db := "observer_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE observer_template")
	c := connect(t, db, "postgres")
	t.Cleanup(func() { c.Close(context.Background()) })
	exec(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','poster'),($2,'synthetic','observer')`, actor, service)
	exec(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional LLC','legal')`, entity)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'post',NULL),($1,$2,'read',NULL),($1,$3,'read',NULL)`, entity, actor, service)
	exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic book')`, entity, book)
	exec(t, c, `INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000005','USD','840',2,'2000-01-01',NULL,'synthetic')`)
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,'00000000-0000-4000-8000-000000000004','2026-01-01',NULL,'00000000-0000-4000-8000-000000000005','accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$3,now())`, entity, book, actor)
	exec(t, c, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000008','2026-01-01','2026-02-01')`, entity, book)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,'00000000-0000-4000-8000-000000000011','Assets:Checking','Checking','asset','debit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000012','Equity:Capital','Capital','equity','credit',false,'2026-01-01',NULL)`, entity, book)
	exec(t, c, `INSERT INTO integration.observer_binding(binding_id,entity_id,book_id,principal_id,publisher_login,consumer_login) VALUES($1,$2,$3,$4,'synthetic_publisher','synthetic_observer')`, binding, entity, book, service)
	rev := value(t, c, "SELECT revision::text FROM ledger.book")
	body := fmt.Sprintf(`{"expected_revision":%q,"policy_id":"00000000-0000-4000-8000-000000000004","accounting_date":"2026-01-01","description":"Synthetic capital","lines":[{"account_id":"00000000-0000-4000-8000-000000000011","side":"debit","transaction_currency":"USD","transaction_amount":"1.01","functional_amount":"1.01","valuation_basis":"identity"},{"account_id":"00000000-0000-4000-8000-000000000012","side":"credit","transaction_currency":"USD","transaction_amount":"1.01","functional_amount":"1.01","valuation_basis":"identity"}]}`, rev)
	r := connect(t, db, "synthetic_runtime")
	defer r.Close(context.Background())
	tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(context.Background(), `SELECT * FROM ledger.post_journal($1,$2,$3,'fixture-post',gen_random_uuid(),$4)`, actor, entity, book, body)
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	return c, db, value(t, c, "SELECT journal_id::text FROM ledger.journal_entry")
}

// invoke retries complete serializable transactions, preserving all operation identifiers.
func invoke(t testing.TB, db, user, q string, args ...any) (string, error) {
	t.Helper()
	c := connect(t, db, user)
	defer c.Close(context.Background())
	var out string
	var err error
	for i := 0; i < 3; i++ {
		tx, e := c.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return "", e
		}
		_, e = tx.Exec(context.Background(), "SET LOCAL statement_timeout='5s'; SET LOCAL lock_timeout='1s'")
		if e == nil {
			e = tx.QueryRow(context.Background(), q, args...).Scan(&out)
		}
		if e == nil {
			e = tx.Commit(context.Background())
		} else {
			_ = tx.Rollback(context.Background())
		}
		err = e
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "40001" && p.Code != "40P01") {
			return out, e
		}
	}
	return "", err
}

// call requires a committed result; precommit function returns never count as evidence.
func call(t testing.TB, db, user, q string, args ...any) string {
	t.Helper()
	s, e := invoke(t, db, user, q, args...)
	if e != nil {
		t.Fatal(e)
	}
	return s
}

// discover identifies only the operation created from this fixture's event.
func discover(t testing.TB, c *pgx.Conn, db string) string {
	t.Helper()
	call(t, db, "synthetic_publisher", "SELECT integration.observe_discover($1,25)::text", binding)
	return value(t, c, "SELECT operation_id::text FROM integration.work_operation")
}

// claim returns fence metadata only after claim commit.
func claim(t testing.TB, db, id, h string) (int64, int64) {
	t.Helper()
	s := call(t, db, "synthetic_observer", "SELECT row_to_json(x)::text FROM integration.observe_claim($1,1,$2) x", id, h)
	var v struct {
		Status              string
		ExecutionGeneration int64 `json:"execution_generation"`
		FenceToken          int64 `json:"fence_token"`
	}
	if e := json.Unmarshal([]byte(s), &v); e != nil {
		t.Fatal(e)
	}
	if v.Status != "claimed" {
		t.Fatal(s)
	}
	return v.ExecutionGeneration, v.FenceToken
}

// finish uses precisely the persisted claim tuple; no actor or arbitrary outcome is accepted.
func finish(t testing.TB, db, id, h string, g, f int64) string {
	t.Helper()
	return call(t, db, "synthetic_observer", "SELECT integration.observe_finish($1,$2,$3,$4)", id, h, g, f)
}

// TestObserverSchemaBoundary proves exact restricted roles, transaction mode and immutable evidence.
func TestObserverSchemaBoundary(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	for _, user := range []string{"synthetic_publisher", "synthetic_observer", "synthetic_runtime"} {
		r := connect(t, db, user)
		for _, q := range []string{"SELECT * FROM integration.work_operation", "INSERT INTO integration.work_receipt DEFAULT VALUES", "CREATE TABLE integration.evil(id int)", "SELECT integration.observer_error('P2200')"} {
			_, e := r.Exec(context.Background(), q)
			code(t, e, "42501")
		}
		r.Close(context.Background())
	}
	_, e := invoke(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_claim($1,1,$2)", id, holder)
	code(t, e, "42501")
	_, e = invoke(t, db, "synthetic_observer", "SELECT integration.observe_discover($1,1)::text", binding)
	code(t, e, "42501")
	r := connect(t, db, "synthetic_observer")
	defer r.Close(context.Background())
	_, e = r.Exec(context.Background(), "SELECT * FROM integration.observe_claim($1,1,$2)", id, holder)
	code(t, e, "P2200")
	_, e = invoke(t, db, "postgres", "SELECT status FROM integration.observe_claim($1,1,$2)", id, holder)
	code(t, e, "P2201")
	if value(t, c, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='integration' AND starts_with(p.proname,'observe_') AND (NOT p.prosecdef OR p.provolatile<>'v' OR p.proparallel<>'u' OR p.proconfig<>ARRAY['search_path=pg_catalog, pg_temp'])`) != "0" {
		t.Fatal("unsafe function attributes")
	}
	// A caller-controlled temporary object cannot intercept a fully qualified source read.
	exec(t, r, "CREATE TEMP TABLE work_operation(operation_id uuid)")
	g, f := claim(t, db, id, holder)
	if finish(t, db, id, holder, g, f) != "observed" {
		t.Fatal("no observation")
	}
	_, e = c.Exec(context.Background(), "UPDATE integration.work_receipt SET outcome='retryable'")
	code(t, e, "P2108")
	_, e = c.Exec(context.Background(), "UPDATE integration.work_operation SET state='ready',terminal_code=NULL,finished_at=NULL")
	code(t, e, "P2205")
}

// TestObserverDiscoverAtomicity rolls back discovery and races two independent publisher sessions.
func TestObserverDiscoverAtomicity(t *testing.T) {
	c, db, _ := fixture(t)
	r := connect(t, db, "synthetic_publisher")
	defer r.Close(context.Background())
	tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(context.Background(), "SELECT integration.observe_discover($1,25)", binding)
	if e != nil {
		t.Fatal(e)
	}
	_ = tx.Rollback(context.Background())
	for _, table := range []string{"work_operation", "outbox_delivery", "work_dispatch"} {
		if value(t, c, "SELECT count(*)::text FROM integration."+table) != "0" {
			t.Fatal("partial discovery")
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Go(func() {
			_, e := invoke(t, db, "synthetic_publisher", "SELECT integration.observe_discover($1,25)::text", binding)
			errs <- e
		})
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	for _, table := range []string{"work_operation", "outbox_delivery", "work_dispatch"} {
		if value(t, c, "SELECT count(*)::text FROM integration."+table) != "1" {
			t.Fatal("duplicate discovery")
		}
	}
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_discover($1,25)::text", binding) != "0" {
		t.Fatal("history duplicated")
	}
}

// TestObserverReceiptAtomicity rolls back completion and repeats a committed receipt without financial changes.
func TestObserverReceiptAtomicity(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	before := value(t, c, "SELECT revision::text FROM ledger.book")
	g, f := claim(t, db, id, holder)
	r := connect(t, db, "synthetic_observer")
	defer r.Close(context.Background())
	tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(context.Background(), "SELECT integration.observe_finish($1,$2,$3,$4)", id, holder, g, f)
	if e != nil {
		t.Fatal(e)
	}
	_ = tx.Rollback(context.Background())
	if value(t, c, "SELECT count(*)::text FROM integration.work_receipt") != "0" {
		t.Fatal("receipt survived rollback")
	}
	if finish(t, db, id, holder, g, f) != "observed" || finish(t, db, id, holder, g, f) != "done" {
		t.Fatal("ack replay")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "1" || value(t, c, "SELECT revision::text FROM ledger.book") != before || value(t, c, "SELECT count(*)::text FROM integration.outbox_message") != "1" {
		t.Fatal("financial side effect")
	}
}

// TestObserverPublisherRace commits consumer success before a delayed publisher acknowledgement.
func TestObserverPublisherRace(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	if call(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_dispatch_claim($1,$2)", id, other) != "claimed" {
		t.Fatal("publisher not claimed")
	}
	g, f := claim(t, db, id, holder)
	finish(t, db, id, holder, g, f)
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_dispatch_result($1,1,$2,1,'accepted')::text", id, other) != "true" {
		t.Fatal("late ack")
	}
	if value(t, c, "SELECT state FROM integration.work_operation") != "succeeded" {
		t.Fatal("regressed")
	}
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_dispatch_result($1,1,$2,1,'timeout')::text", id, other) != "false" {
		t.Fatal("stale publisher accepted")
	}
}

// TestObserverRetryBudget checks exact jitter bounds and the eight-failure terminal budget.
func TestObserverRetryBudget(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	for i := 1; i <= 8; i++ {
		g, f := claim(t, db, id, holder)
		s := call(t, db, "synthetic_observer", "SELECT integration.observe_fail($1,$2,$3,$4)", id, holder, g, f)
		want := "retry_wait"
		if i == 8 {
			want = "quarantined"
		}
		if s != want {
			t.Fatal(s)
		}
		if value(t, c, "SELECT failed_attempt_count::text FROM integration.work_operation") != fmt.Sprint(i) {
			t.Fatal("failure count")
		}
		if i < 8 {
			if value(t, c, fmt.Sprintf("SELECT (extract(epoch FROM next_eligible_at-(SELECT max(recorded_at) FROM integration.work_receipt)) BETWEEN %f AND %f)::text FROM integration.work_operation", float64(5*(int64(1)<<(i-1)))/2, float64(5*(int64(1)<<(i-1))))) != "true" {
				t.Fatal("jitter out of bounds")
			}
			exec(t, c, "UPDATE integration.work_operation SET next_eligible_at=clock_timestamp()-interval '1 second'")
		}
	}
	if call(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,1,$2)", id, other) != "done" {
		t.Fatal("terminal reopened")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_attempt") != "8" {
		t.Fatal("duplicate attempt")
	}
}

// TestObserverFenceClock waits past actual lease expiry and races two recovery sessions.
func TestObserverFenceClock(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	g, f := claim(t, db, id, holder)
	if call(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,1,$2)", id, other) != "busy" {
		t.Fatal("healthy claim stolen")
	}
	// Owner shortens only the synthetic lease, preserving the immutable attempt start.
	exec(t, c, "UPDATE integration.work_operation SET lease_until=clock_timestamp()+interval '120 milliseconds'")
	lock, e := c.Begin(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	_, e = lock.Exec(context.Background(), "SELECT 1 FROM integration.work_operation FOR UPDATE")
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		_, e := invoke(t, db, "synthetic_observer", "SELECT integration.observe_finish($1,$2,$3,$4)", id, holder, g, f)
		done <- e
	}()
	time.Sleep(180 * time.Millisecond)
	if e = lock.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	code(t, <-done, "P2203")
	if call(t, db, "synthetic_observer", "SELECT integration.observe_heartbeat($1,$2,$3,$4)::text", id, holder, g, f) != "false" {
		t.Fatal("resurrected")
	}
	var wg sync.WaitGroup
	results := make(chan string, 2)
	for i := 0; i < 2; i++ {
		wg.Go(func() { results <- call(t, db, "synthetic_publisher", "SELECT integration.observe_recover($1)", id) })
	}
	wg.Wait()
	close(results)
	if value(t, c, "SELECT failed_attempt_count::text FROM integration.work_operation") != "1" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='abandoned'") != "1" {
		t.Fatal("double abandoned")
	}
	exec(t, c, "UPDATE integration.work_operation SET next_eligible_at=clock_timestamp()-interval '1 second'")
	ng, nf := claim(t, db, id, other)
	_, e = invoke(t, db, "synthetic_observer", "SELECT integration.observe_finish($1,$2,$3,$4)", id, holder, g, f)
	code(t, e, "P2203")
	finish(t, db, id, other, ng, nf)
}

// TestObserverAuthorizationRace checks service revocation and independence from the original poster.
func TestObserverAuthorizationRace(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	g, f := claim(t, db, id, holder)
	// The service mandate is independent of the journal poster.
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", actor)
	if call(t, db, "synthetic_observer", "SELECT integration.observe_heartbeat($1,$2,$3,$4)::text", id, holder, g, f) != "true" {
		t.Fatal("poster revoked service")
	}
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", service)
	if finish(t, db, id, holder, g, f) != "done" || value(t, c, "SELECT state FROM integration.work_operation") != "blocked" || value(t, c, "SELECT failed_attempt_count::text FROM integration.work_operation") != "0" {
		t.Fatal("revocation not durable")
	}
	if finish(t, db, id, holder, g, f) != "done" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt") != "1" {
		t.Fatal("repeated denial appended")
	}
	_, e := invoke(t, db, "synthetic_publisher", "SELECT integration.observe_discover($1,25)::text", binding)
	code(t, e, "P2201")
}

// TestObserverQueueLoss retains uncertain enqueue evidence and deduplicates old/new dispatch arrivals.
func TestObserverQueueLoss(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	for _, r := range []string{"timeout", "already_exists", "rejected"} {
		call(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_dispatch_claim($1,$2)", id, other)
		f := value(t, c, "SELECT publisher_fence::text FROM integration.work_dispatch")
		if call(t, db, "synthetic_publisher", "SELECT integration.observe_dispatch_result($1,1,$2,$3,$4)::text", id, other, f, r) != "true" {
			t.Fatal("result failed")
		}
	}
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_recover($1)", id) != "redispatched" {
		t.Fatal("no new wakeup")
	}
	if value(t, c, "SELECT dispatch_generation::text FROM integration.work_operation") != "2" {
		t.Fatal("generation")
	}
	g, f := claim(t, db, id, holder)
	finish(t, db, id, holder, g, f)
	if call(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,2,$2)", id, other) != "done" {
		t.Fatal("new delivery duplicated old")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "1" {
		t.Fatal("duplicate effect")
	}
}

// TestObserverRevocationOrder proves both actual two-session lock/commit orderings.
func TestObserverRevocationOrder(t *testing.T) {
	for _, first := range []string{"completion", "revocation"} {
		t.Run(first, func(t *testing.T) {
			c, db, _ := fixture(t)
			id := discover(t, c, db)
			g, f := claim(t, db, id, holder)
			if first == "completion" {
				r := connect(t, db, "synthetic_observer")
				defer r.Close(context.Background())
				tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
				if e != nil {
					t.Fatal(e)
				}
				var got string
				e = tx.QueryRow(context.Background(), "SELECT integration.observe_finish($1,$2,$3,$4)", id, holder, g, f).Scan(&got)
				if e != nil {
					t.Fatal(e)
				}
				result := make(chan error, 1)
				go func() {
					a := connect(t, db, "postgres")
					defer a.Close(context.Background())
					_, e := a.Exec(context.Background(), "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", service)
					result <- e
				}()
				select {
				case e := <-result:
					t.Fatalf("revocation bypassed lock: %v", e)
				case <-time.After(80 * time.Millisecond):
				}
				if e = tx.Commit(context.Background()); e != nil {
					t.Fatal(e)
				}
				if e = <-result; e != nil {
					t.Fatal(e)
				}
				if value(t, c, "SELECT state FROM integration.work_operation") != "succeeded" || finish(t, db, id, holder, g, f) != "done" {
					t.Fatal("completed receipt suppression")
				}
			} else {
				tx, e := c.Begin(context.Background())
				if e != nil {
					t.Fatal(e)
				}
				_, e = tx.Exec(context.Background(), "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1", service)
				if e != nil {
					t.Fatal(e)
				}
				result := make(chan error, 1)
				go func() {
					_, e := invoke(t, db, "synthetic_observer", "SELECT integration.observe_finish($1,$2,$3,$4)", id, holder, g, f)
					result <- e
				}()
				select {
				case e := <-result:
					t.Fatalf("finish bypassed revoke lock: %v", e)
				case <-time.After(80 * time.Millisecond):
				}
				if e = tx.Commit(context.Background()); e != nil {
					t.Fatal(e)
				}
				if e = <-result; e != nil {
					t.Fatal(e)
				}
				if value(t, c, "SELECT state FROM integration.work_operation") != "blocked" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "0" {
					t.Fatal("revocation lost")
				}
			}
		})
	}
}

// TestObserverDisableMatrix checks current binding/principal/entity/book authority before first claim.
func TestObserverDisableMatrix(t *testing.T) {
	for _, scope := range []string{"binding", "principal", "entity", "book"} {
		t.Run(scope, func(t *testing.T) {
			c, db, _ := fixture(t)
			id := discover(t, c, db)
			switch scope {
			case "binding":
				exec(t, c, "UPDATE integration.observer_binding SET enabled=false,binding_revision=binding_revision+1,updated_at=clock_timestamp()")
			case "principal":
				exec(t, c, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", service)
			case "entity":
				exec(t, c, "UPDATE ledger.entity SET disabled_at=clock_timestamp()")
			case "book":
				exec(t, c, "UPDATE ledger.book SET disabled_at=clock_timestamp()")
			}
			if call(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,1,$2)", id, holder) != "done" || value(t, c, "SELECT state FROM integration.work_operation") != "blocked" {
				t.Fatal("disabled authority not blocked")
			}
			if value(t, c, "SELECT count(*)::text FROM integration.work_attempt") != "0" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt") != "0" {
				t.Fatal("fabricated attempt")
			}
		})
	}
}

// TestObserverCommitAcknowledgementLoss injects a lost application acknowledgement after actual COMMIT.
func TestObserverCommitAcknowledgementLoss(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	g, f := claim(t, db, id, holder)
	r := connect(t, db, "synthetic_observer")
	tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(context.Background(), "SELECT integration.observe_finish($1,$2,$3,$4)", id, holder, g, f)
	if e != nil {
		t.Fatal(e)
	}
	// Simulate the handler losing its successful commit response; disconnect before it can acknowledge.
	if e = tx.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	_ = r.Close(context.Background())
	if call(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,1,$2)", id, other) != "done" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "1" {
		t.Fatal("lost acknowledgement duplicated effect")
	}
}

// TestObserverEventMismatch uses owner-only corruption injection to test quarantine, never a runtime mutation route.
func TestObserverEventMismatch(t *testing.T) {
	for _, stage := range []string{"discovery", "completion"} {
		t.Run(stage, func(t *testing.T) {
			c, db, _ := fixture(t)
			var id string
			var g, f int64
			if stage == "completion" {
				id = discover(t, c, db)
				g, f = claim(t, db, id, holder)
			}
			exec(t, c, "ALTER TABLE integration.outbox_message DISABLE TRIGGER immutable; UPDATE integration.outbox_message SET book_revision=book_revision+1; ALTER TABLE integration.outbox_message ENABLE TRIGGER immutable")
			if stage == "discovery" {
				id = discover(t, c, db)
			} else {
				if finish(t, db, id, holder, g, f) != "done" {
					t.Fatal("mismatch result")
				}
			}
			if value(t, c, "SELECT terminal_code FROM integration.work_operation") != "event_mismatch" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "0" {
				t.Fatal("bad event observed")
			}
		})
	}
}

// TestObserverStaleDispatchExpiry proves fifteen-minute recovery and publisher-fence takeover independently.
func TestObserverStaleDispatchExpiry(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	call(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_dispatch_claim($1,$2)", id, holder)
	exec(t, c, "UPDATE integration.work_dispatch SET lease_until=clock_timestamp()-interval '1 second'")
	if call(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_dispatch_claim($1,$2)", id, other) != "claimed" {
		t.Fatal("publisher takeover")
	}
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_dispatch_result($1,1,$2,1,'accepted')::text", id, holder) != "false" {
		t.Fatal("stale publisher fenced")
	}
	call(t, db, "synthetic_publisher", "SELECT integration.observe_dispatch_result($1,1,$2,2,'already_exists')::text", id, other)
	exec(t, c, "UPDATE integration.work_dispatch SET created_at=clock_timestamp()-interval '16 minutes'")
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_recover($1)", id) != "redispatched" {
		t.Fatal("queue tombstone not recovered")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_dispatch") != "2" {
		t.Fatal("dispatch history lost")
	}
}

// TestObserverAbandonedEighthFailure checks terminal status when a duplicate delivery closes the final expired claim.
func TestObserverAbandonedEighthFailure(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	for i := 0; i < 7; i++ {
		g, f := claim(t, db, id, holder)
		call(t, db, "synthetic_observer", "SELECT integration.observe_fail($1,$2,$3,$4)", id, holder, g, f)
		exec(t, c, "UPDATE integration.work_operation SET next_eligible_at=clock_timestamp()-interval '1 second'")
	}
	claim(t, db, id, holder)
	exec(t, c, "UPDATE integration.work_operation SET lease_until=clock_timestamp()-interval '1 second'")
	if call(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,1,$2)", id, other) != "done" || value(t, c, "SELECT terminal_code FROM integration.work_operation") != "retry_exhausted" || value(t, c, "SELECT attempt_count::text||':'||failed_attempt_count::text FROM integration.work_operation") != "8:8" {
		t.Fatal("eighth abandonment not terminal")
	}
}

// TestObserverDeadlineAndOverflow creates only owner-controlled expired metadata; runtime clocks remain server-owned.
func TestObserverDeadlineAndOverflow(t *testing.T) {
	t.Run("deadline", func(t *testing.T) {
		c, db, _ := fixture(t)
		id := discover(t, c, db)
		// Construct a historical failed first execution, then allow a current second claim just before its 24h deadline.
		exec(t, c, `BEGIN;
 INSERT INTO integration.work_attempt(entity_id,book_id,operation_id,execution_generation,fence_token,holder_id,binding_revision,started_at) SELECT entity_id,book_id,operation_id,1,1,'00000000-0000-4000-8000-000000000022',1,clock_timestamp()-interval '24 hours'+interval '800 milliseconds' FROM integration.work_operation;
 INSERT INTO integration.work_receipt SELECT entity_id,book_id,operation_id,1,1,'retryable',request_digest,1,clock_timestamp()-interval '23 hours' FROM integration.work_operation;
 UPDATE integration.work_operation SET state='retry_wait',attempt_count=1,failed_attempt_count=1,execution_generation=1,fence_token=2,first_claimed_at=(SELECT started_at FROM integration.work_attempt),deadline_at=(SELECT started_at+interval '24 hours' FROM integration.work_attempt);
 COMMIT;`)
		g, f := claim(t, db, id, holder)
		time.Sleep(900 * time.Millisecond)
		if finish(t, db, id, holder, g, f) != "done" || value(t, c, "SELECT terminal_code FROM integration.work_operation") != "age_exhausted" || value(t, c, "SELECT failed_attempt_count::text FROM integration.work_operation") != "2" {
			t.Fatal("deadline did not override healthy lease")
		}
	})
	t.Run("overflow", func(t *testing.T) {
		c, db, _ := fixture(t)
		id := discover(t, c, db)
		exec(t, c, "UPDATE integration.work_operation SET fence_token=9223372036854775807")
		_, e := invoke(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,1,$2)", id, holder)
		code(t, e, "P2204")
		if value(t, c, "SELECT count(*)::text FROM integration.work_attempt") != "0" {
			t.Fatal("overflow claimed")
		}
	})
}

// TestObserverExactPrivileges audits every application function/table against the explicit capability allowlist.
func TestObserverExactPrivileges(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	for _, role := range []string{"bookkeeping_publisher", "bookkeeping_observer"} {
		allowed := []string{"observe_discover", "observe_candidates", "observe_dispatch_claim", "observe_dispatch_result", "observe_recover", "sync_delivery_discover", "sync_delivery_candidates", "sync_delivery_dispatch_claim", "sync_delivery_dispatch_result", "sync_delivery_recover"}
		if role == "bookkeeping_observer" {
			allowed = []string{"observe_claim", "observe_heartbeat", "observe_finish", "observe_fail"}
		}
		if value(t, c, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','reporting') AND has_function_privilege($1,p.oid,'EXECUTE') <> (n.nspname='integration' AND p.proname=ANY($2::text[]))`, role, allowed) != "0" {
			t.Fatal("function grant widened")
		}
		if value(t, c, `SELECT count(*)::text FROM pg_class x JOIN pg_namespace n ON n.oid=x.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','reporting') AND x.relkind IN ('r','p') AND has_table_privilege($1,x.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')`, role) != "0" {
			t.Fatal("table grant widened")
		}
	}
	r := connect(t, db, "synthetic_observer")
	defer r.Close(context.Background())
	_, e := r.Exec(context.Background(), "SET ROLE bookkeeping_owner")
	code(t, e, "42501")
	for _, limit := range []int{0, 26} {
		_, e = invoke(t, db, "synthetic_publisher", "SELECT integration.observe_discover($1,$2)::text", binding, limit)
		code(t, e, "P2200")
	}
	for _, bad := range []string{other, "00000000-0000-4000-8000-000000000099"} {
		_, e = invoke(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,1,$2)", bad, holder)
		code(t, e, "P2201")
	}
	_, e = invoke(t, db, "synthetic_observer", "SELECT status FROM integration.observe_claim($1,99,$2)", id, holder)
	code(t, e, "P2200")
	// Even ordinary owner writes cannot insert incomplete terminal timestamps/codes.
	_, e = c.Exec(context.Background(), "UPDATE integration.work_operation SET state='quarantined',finished_at=clock_timestamp(),terminal_code=NULL")
	code(t, e, "23514")
}

// TestObserverCancellation rolls back a real blocked query; no claim or receipt survives cancellation.
func TestObserverCancellation(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	lock, e := c.Begin(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Rollback(context.Background())
	_, e = lock.Exec(context.Background(), "SELECT 1 FROM integration.work_operation FOR UPDATE")
	if e != nil {
		t.Fatal(e)
	}
	r := connect(t, db, "synthetic_observer")
	defer r.Close(context.Background())
	tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	_, e = tx.Exec(ctx, "SELECT * FROM integration.observe_claim($1,1,$2)", id, holder)
	if e == nil {
		t.Fatal("cancelled query succeeded")
	}
	_ = tx.Rollback(context.Background())
	if e = lock.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_attempt") != "0" || value(t, c, "SELECT count(*)::text FROM integration.work_receipt") != "0" {
		t.Fatal("cancelled attempt persisted")
	}
}

// TestObserverReversalHistory proves setup revision gaps and reversed journals remain discoverable facts.
func TestObserverReversalHistory(t *testing.T) {
	c, db, j := fixture(t)
	rev := value(t, c, "SELECT revision::text FROM ledger.book")
	body := fmt.Sprintf(`{"expected_revision":%q,"accounting_date":"2026-01-02","reason":"Synthetic correction"}`, rev)
	r := connect(t, db, "synthetic_runtime")
	defer r.Close(context.Background())
	tx, e := r.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(context.Background(), "SELECT * FROM ledger.reverse_journal($1,$2,$3,$4,'fixture-reversal',gen_random_uuid(),$5)", actor, entity, book, j, body)
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_discover($1,25)::text", binding) != "2" {
		t.Fatal("reversal missing")
	}
	rows, e := c.Query(context.Background(), "SELECT operation_id::text FROM integration.work_operation ORDER BY operation_id")
	if e != nil {
		t.Fatal(e)
	}
	var ids []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			t.Fatal(e)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		g, f := claim(t, db, id, holder)
		finish(t, db, id, holder, g, f)
	}
	if value(t, c, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "2" || value(t, c, "SELECT min(book_revision)::text FROM integration.outbox_message") == "1" {
		t.Fatal("history/gaps fixture")
	}
}

// TestObserverRepeatedUncertainty cannot refresh the fixed dispatch age indefinitely, including publisher crashes.
func TestObserverRepeatedUncertainty(t *testing.T) {
	c, db, _ := fixture(t)
	id := discover(t, c, db)
	exec(t, c, "UPDATE integration.work_dispatch SET created_at=clock_timestamp()-interval '16 minutes'")
	call(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_dispatch_claim($1,$2)", id, holder)
	exec(t, c, "UPDATE integration.work_dispatch SET lease_until=clock_timestamp()-interval '1 second'")
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_recover($1)", id) != "no_change" {
		t.Fatal("first publisher expiry did not preserve identity")
	}
	call(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_dispatch_claim($1,$2)", id, other)
	exec(t, c, "UPDATE integration.work_dispatch SET lease_until=clock_timestamp()-interval '1 second'")
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_recover($1)", id) != "redispatched" {
		t.Fatal("repeated publisher crashes postponed recovery")
	}
	for i, r := range []string{"already_exists", "timeout", "already_exists"} {
		call(t, db, "synthetic_publisher", "SELECT status FROM integration.observe_dispatch_claim($1,$2)", id, holder)
		if call(t, db, "synthetic_publisher", "SELECT integration.observe_dispatch_result($1,2,$2,$3,$4)::text", id, holder, i+1, r) != "true" {
			t.Fatal("uncertainty not recorded")
		}
	}
	exec(t, c, "UPDATE integration.work_dispatch SET created_at=clock_timestamp()-interval '16 minutes' WHERE dispatch_generation=2")
	if call(t, db, "synthetic_publisher", "SELECT integration.observe_recover($1)", id) != "redispatched" || value(t, c, "SELECT dispatch_generation::text FROM integration.work_operation") != "3" {
		t.Fatal("fresh uncertainty reset fixed age")
	}
}
