//go:build integration

package syncdispatch_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// init shares the opt-in proof flag with adapter tests when composed.
func init() {
	if flag.Lookup("sync-dispatch-proof") == nil {
		flag.Bool("sync-dispatch-proof", false, "Use only the disposable dispatch database")
	}
}

const entity = "00000000-0000-4000-8000-000000000002"
const book = "00000000-0000-4000-8000-000000000003"
const actor = "00000000-0000-4000-8000-000000000001"
const reader = "00000000-0000-4000-8000-000000000020"
const other = "00000000-0000-4000-8000-000000000021"
const account = "00000000-0000-4000-8000-000000000011"
const currency = "00000000-0000-4000-8000-000000000005"

// connect uses only the runner's private synthetic trust network, never an inherited DSN.
func connect(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if flag.Lookup("sync-dispatch-proof").Value.String() != "true" {
		t.Skip("use scripts/test-sync-dispatch.sh")
	}
	c, e := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if e != nil {
		t.Fatal(e)
	}
	return c
}

// exec is owner-only fixture setup or inspection, never product privilege.
func exec(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal(e)
	}
}

// value reads fixed aggregate proof facts with the fixture owner.
func value(t testing.TB, c *pgx.Conn, q string, args ...any) string {
	t.Helper()
	var s string
	if e := c.QueryRow(context.Background(), q, args...).Scan(&s); e != nil {
		t.Fatal(e)
	}
	return s
}

// requireCode compares SQLSTATE without relying on potentially sensitive diagnostics.
func requireCode(t testing.TB, e error, want string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(e, &p) || p.Code != want {
		t.Fatalf("want %s got %v", want, e)
	}
}

// fixture installs only fictional ownership and independently chosen currency metadata.
func fixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	db := "ds_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE sync_dispatch_template")
	c := connect(t, db, "postgres")
	t.Cleanup(func() { c.Close(context.Background()) })
	exec(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','approver'),($2,'synthetic','reader'),($3,'synthetic','other')`, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional LLC','legal')`, entity)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$3,'read',NULL),($1,$4,'read',NULL)`, entity, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic book')`, entity, book)
	exec(t, c, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000006','JPY','392',0,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000007','KWD','414',3,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000008','XXX','999',NULL,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000009','EUR','978',2,'2020-01-01','2025-01-01','synthetic')`, currency)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Fictional cash','asset','debit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000012','2000','Fictional liability','liability','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000013','1001','Fictional nonmonetary','asset','debit',false,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000014','1002','Fictional contra','asset','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000015','1003','Fictional second deposit','asset','debit',true,'2026-01-01',NULL)`, entity, book, account)
	return c, db
}

// TestSharedPublisherSelection proves login scope, bounded wraparound and disabled exclusion in SQL.
func TestSharedPublisherSelection(t *testing.T) {
	c, db := fixture(t)
	for i := 1; i <= 8; i++ {
		bid := fmt.Sprintf("00000000-0000-4000-8000-%012x", 0x300+i)
		bookID := fmt.Sprintf("00000000-0000-4000-8000-%012x", 0x400+i)
		exec(t, c, "INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,$3)", entity, bookID, fmt.Sprintf("Fictional shared book %d", i))
		login := "synthetic_publisher"
		if i == 7 {
			login = "bookkeeping_publisher"
		}
		principal := reader
		if i%2 == 0 {
			principal = other
		}
		exec(t, c, "INSERT INTO integration.sync_delivery_binding(entity_id,book_id,binding_id,service_principal,publisher_login,worker_login,enabled) VALUES($1,$2,$3,$4,$5,'synthetic_sync',$6)", entity, bookID, bid, principal, login, i != 8)
	}
	page := func() []any {
		return object(t, call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_bindings(5)::text"))["binding_ids"].([]any)
	}
	first, second := page(), page()
	unique := map[any]bool{}
	for _, id := range append(first, second...) {
		unique[id] = true
		if id == "00000000-0000-4000-8000-000000000307" || id == "00000000-0000-4000-8000-000000000308" {
			t.Fatal("foreign or disabled binding selected")
		}
	}
	if len(first) != 5 || len(second) != 5 || len(unique) != 6 {
		t.Fatalf("unfair or unbounded selection: %v %v", first, second)
	}
	if value(t, c, "SELECT count(DISTINCT service_principal)::text FROM integration.sync_delivery_binding WHERE publisher_login='synthetic_publisher' AND enabled") != "2" {
		t.Fatal("distinct service principals not enrolled")
	}
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.sync_delivery_bindings(5)::text")
	requireCode(t, e, "42501")
}

// raw encodes synthetic exact values without floating-point amounts.
func raw(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return string(b)
}

// invoke uses a real login, bounded transactions and full serialization retry.
func invoke(t testing.TB, db, user, q string, args ...any) (string, error) {
	t.Helper()
	c := connect(t, db, user)
	defer c.Close(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	var result string
	var err error
	for i := 0; i < 3; i++ {
		tx, e := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return "", e
		}
		_, e = tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL lock_timeout='1s'")
		if e == nil && user == "synthetic_operator" {
			_, e = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner")
		}
		if e == nil {
			e = tx.QueryRow(ctx, q, args...).Scan(&result)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		_ = tx.Rollback(context.Background())
		if e == nil {
			return result, nil
		}
		err = e
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "40001" && p.Code != "40P01") {
			break
		}
	}
	return "", err
}

// call requires a committed result from an actual restricted login.
func call(t testing.TB, db, user, q string, args ...any) string {
	t.Helper()
	v, e := invoke(t, db, user, q, args...)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

// object decodes only synthetic fixed SQL projection fields.
func object(t testing.TB, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal([]byte(s), &v); e != nil {
		t.Fatal(e)
	}
	return v
}

// setup explicitly provisions connection management separately from acquisition.
func setup(t *testing.T) (*pgx.Conn, string) {
	c, db := fixture(t)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'import',NULL),($1,$3,'import',NULL)`, entity, actor, reader)
	call(t, db, "synthetic_operator", "SELECT integration.set_connect_permission($1,$2,$3,$1,'connect',true)::text", actor, entity, book)
	call(t, db, "synthetic_operator", "SELECT integration.set_connection_worker($1,$2,$3,'worker',$4)::text", actor, entity, book, raw(map[string]any{"service_principal_id": reader, "worker_login": "synthetic_sync", "enabled": true}))
	return c, db
}

// begin pins intent before any synthetic provider or secret effect.
func begin(t testing.TB, db, key, mode string, item, generation any, ids []any) map[string]any {
	t.Helper()
	if ids == nil {
		ids = []any{}
	}
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.begin_connection($1,$2,$3,$4,$5)::text", actor, entity, book, key, raw(map[string]any{"mode": mode, "sync_item_id": item, "expected_generation": generation, "approved_stream_ids": ids})))
}

// link records only an independently chosen fingerprint, not a token.
func link(t testing.TB, db string, a map[string]any) map[string]any {
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "expires_at": "2099-01-01T00:00:00Z"})))
}

// claim performs the one-shot exchange reservation through a real API login.
func claim(t testing.TB, db string, a map[string]any) map[string]any {
	var l, p any
	if a["mode"] == "new" || a["mode"] == "reconnect" {
		l = strings.Repeat("a", 64)
	}
	if a["mode"] == "new" {
		p = strings.Repeat("b", 64)
	}
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": l, "public_token_sha256": p})))
}

// verified represents synthetic server-verified metadata, distinct from a browser selection.
func verified(item string, accounts ...string) map[string]any {
	rows := []any{}
	for _, id := range accounts {
		rows = append(rows, map[string]any{"account_id": id, "type": "depository", "iso_currency_code": "USD"})
	}
	return map[string]any{"item_id": item, "accounts": rows, "consent_expires_at": "2099-01-01T00:00:00Z"}
}

// recordManifest commits verified identity before any secret write could occur.
func recordManifest(t testing.TB, db string, a map[string]any, v map[string]any) map[string]any {
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["claim_id"], raw(v)))
}

// confirm models only the trusted service's confirmed opaque-reference acknowledgement.
func confirm(t testing.TB, db string, a map[string]any) map[string]any {
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], a["claim_id"]))
}

// binding explicitly approves a new source; no grant or identity is inferred from its name.
func binding(id, ledgerID string) map[string]any {
	return map[string]any{"account_id": id, "ledger_account_id": ledgerID, "currency_version_id": currency, "source_account_id": nil, "expected_binding_revision": nil, "binding_date": "2026-01-05", "binding_evidence_ref": "00000000-0000-4000-8000-000000000080"}
}

// bind publishes exactly the reviewed selected accounts and capabilities.
func bind(t testing.TB, db string, a map[string]any, rows ...any) map[string]any {
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.bind_connection($1,$2,$3,$4,'bind',$5)::text", actor, entity, book, a["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": rows})))
}

// connected sets up a two-account Item using only runtime lifecycle operations.
func connected(t *testing.T) (*pgx.Conn, string, map[string]any, map[string]any) {
	c, db := setup(t)
	a := claim(t, db, link(t, db, begin(t, db, "new", "new", nil, nil, nil)))
	a = recordManifest(t, db, a, verified("SyntheticItem", "AccountA", "AccountB"))
	a = confirm(t, db, a)
	out := bind(t, db, a, binding("AccountA", account), binding("AccountB", "00000000-0000-4000-8000-000000000015"))
	return c, db, a, out
}

// itemIntent loads exact current sibling IDs/generation for an explicitly approved operation.
func itemIntent(t testing.TB, c *pgx.Conn, db, key, mode string, item any) map[string]any {
	ids := []any{}
	var parsed []any
	if e := json.Unmarshal([]byte(value(t, c, "SELECT jsonb_agg(stream_id ORDER BY stream_id)::text FROM integration.sync_stream")), &parsed); e != nil {
		t.Fatal(e)
	}
	ids = append(ids, parsed...)
	return begin(t, db, key, mode, item, value(t, c, "SELECT generation::text FROM integration.sync_item"), ids)
}

// delivery installs one owner-enrolled publisher mandate and discovers initial demand.
func delivery(t *testing.T) (*pgx.Conn, string, string, string) {
	c, db, _, _ := connected(t)
	b := object(t, call(t, db, "synthetic_operator", "SELECT integration.enroll_sync_delivery($1,$2,$3,'delivery',$4)::text", actor, entity, book, raw(map[string]any{"publisher_login": "synthetic_publisher", "worker_login": "synthetic_sync", "service_principal_id": reader, "enabled": true})))
	bid := b["binding_id"].(string)
	call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_discover($1,25)::text", bid)
	op := value(t, c, "SELECT operation_id::text FROM integration.sync_delivery_operation ORDER BY operation_id LIMIT 1")
	return c, db, bid, op
}

// TestDeliveryInitialDemand proves initial demand does not require a webhook wakeup.
func TestDeliveryInitialDemand(t *testing.T) {
	c, db, bid, op := delivery(t)
	if value(t, c, "SELECT count(*)::text FROM integration.sync_delivery_operation") != "2" {
		t.Fatal("missing initial demand")
	}
	call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_candidates($1,25)::text", bid)
	h := "00000000-0000-4000-8000-000000000091"
	q := object(t, call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_dispatch_claim($1,$2)::text", op, h))
	if q["status"] != "claimed" {
		t.Fatal(q)
	}
	call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_dispatch_result($1,1,$2,$3,'accepted')::text", op, h, q["dispatch_fence"])
	r := object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_delivery_claim($1,1,$2)::text", op, h))
	if r["status"] != "claimed" {
		t.Fatal(r)
	}
	out := object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_delivery_settle($1,$2,$3,'completed',0)::text", op, h, r["fence"]))
	if out["status"] != "retry_wait" {
		t.Fatal("Go completion hint became false SQL completion", out)
	}
	if value(t, c, "SELECT dispatch_generation::text FROM integration.sync_delivery_operation WHERE operation_id=$1", op) != "2" {
		t.Fatal("retry did not issue new task identity")
	}
}

const holder = "00000000-0000-4000-8000-000000000091"
const holder2 = "00000000-0000-4000-8000-000000000092"

// take claims one fixed delivery through the real worker session.
func take(t *testing.T, db, op string, g any, h string) map[string]any {
	return object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_delivery_claim($1,$2,$3)::text", op, g, h))
}

// acquire reserves the underlying authoritative acquisition lease.
func acquire(t *testing.T, db string, r map[string]any, key any) map[string]any {
	return object(t, call(t, db, "synthetic_sync", "SELECT integration.claim_sync($1,$2,$3,$4)::text", entity, book, r["stream_id"], key))
}

// abandon records known worker failure without inventing a provider retry classification.
func abandon(t *testing.T, db string, r, x map[string]any, reason string) {
	call(t, db, "synthetic_sync", "SELECT integration.abandon_sync($1,$2,$3,$4,$5,$6)::text", entity, book, r["stream_id"], x["set_id"], x["fence"], reason)
}

// settle submits only a hint, which SQL compares against authoritative acquisition facts.
func settle(t *testing.T, db, op string, r map[string]any, outcome string, delay int) map[string]any {
	return object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_delivery_settle($1,$2,$3,$4,$5)::text", op, holder, r["fence"], outcome, delay))
}

// finish publishes an empty complete set; its durable receipt is independently authoritative.
func finish(t *testing.T, db string, r, x map[string]any) {
	p := map[string]any{"page_index": 0, "request_cursor": nil, "next_cursor": "synthetic-C1", "has_more": false, "transactions_update_status": "HISTORICAL_UPDATE_COMPLETE", "request_id": "synthetic-request", "sha256": strings.Repeat("a", 64), "added": []any{}, "modified": []any{}, "removed": []any{}}
	call(t, db, "synthetic_sync", "SELECT integration.stage_sync_page($1,$2,$3,$4,$5,$6)::text", entity, book, r["stream_id"], x["set_id"], x["fence"], raw(p))
	call(t, db, "synthetic_sync", "SELECT integration.complete_sync($1,$2,$3,$4,$5)::text", entity, book, r["stream_id"], x["set_id"], x["fence"])
}
func TestDeliveryDerivedLeaseAndReceipt(t *testing.T) {
	c, db, _, op := delivery(t)
	r := take(t, db, op, 1, holder)
	x := acquire(t, db, r, r["claim_id"])
	abandon(t, db, r, x, "pagination_mutation")
	derived := value(t, c, "SELECT integration.sync_restart_claim($1,$2,$3,$4)::text", entity, book, r["stream_id"], r["claim_id"])
	x = acquire(t, db, r, derived)
	exec(t, c, "UPDATE integration.sync_delivery_operation SET run_lease_until=clock_timestamp()-interval '1 second' WHERE operation_id=$1", op)
	if v := take(t, db, op, 1, holder2); v["status"] != "busy" {
		t.Fatal(v)
	}
	if v := object(t, call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_recover($1)::text", op)); v["status"] != "no_change" {
		t.Fatal(v)
	}
	finish(t, db, r, x)
	if v := take(t, db, op, 1, holder2); v["status"] != "done" {
		t.Fatal(v)
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("dispatch wrote journals")
	}
}
func TestDeliveryUnknownFailureQuarantines(t *testing.T) {
	c, db, bid, op := delivery(t)
	r := take(t, db, op, 1, holder)
	x := acquire(t, db, r, r["claim_id"])
	abandon(t, db, r, x, "provider_failure")
	exec(t, c, "UPDATE integration.sync_delivery_operation SET run_lease_until=clock_timestamp()-interval '1 second' WHERE operation_id=$1", op)
	v := object(t, call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_recover($1)::text", op))
	if v["status"] != "quarantined" {
		t.Fatal(v)
	}
	if value(t, c, "SELECT terminal_code FROM integration.sync_delivery_operation WHERE operation_id=$1", op) != "outcome_unknown" {
		t.Fatal("lost provider delay was guessed")
	}
	call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_discover($1,25)::text", bid)
	if value(t, c, "SELECT count(*)::text FROM integration.sync_delivery_operation WHERE stream_id=$1", r["stream_id"]) != "1" {
		t.Fatal("quarantine automatically reset")
	}
}
func TestDeliveryRetryAfterAndBudget(t *testing.T) {
	c, db, _, op := delivery(t)
	for n := 1; n <= 8; n++ {
		g := value(t, c, "SELECT dispatch_generation::text FROM integration.sync_delivery_operation WHERE operation_id=$1", op)
		r := take(t, db, op, g, holder)
		x := acquire(t, db, r, r["claim_id"])
		abandon(t, db, r, x, "provider_failure")
		v := settle(t, db, op, r, "retryable", 3600)
		if n == 8 {
			if v["status"] != "quarantined" {
				t.Fatal(v)
			}
			break
		}
		if v["status"] != "retry_wait" {
			t.Fatal(v)
		}
		if value(t, c, "SELECT (next_eligible_at>clock_timestamp()+interval '3595 seconds')::text FROM integration.sync_delivery_operation WHERE operation_id=$1", op) != "true" {
			t.Fatal("Retry-After shortened")
		}
		if value(t, c, "SELECT (claim_id<>$2)::text FROM integration.sync_delivery_operation WHERE operation_id=$1", op, r["claim_id"]) != "true" {
			t.Fatal("known failed claim reused")
		}
		// Synthetic clock advance changes only scheduling time, not identity or failure facts.
		exec(t, c, "UPDATE integration.sync_delivery_operation SET next_eligible_at=clock_timestamp()-interval '1 second' WHERE operation_id=$1", op)
	}
	if value(t, c, "SELECT failed_count::text||':'||terminal_code FROM integration.sync_delivery_operation WHERE operation_id=$1", op) != "8:retry_exhausted" {
		t.Fatal("retry budget escaped")
	}
}
func TestDeliveryPermanentFencesLatePublication(t *testing.T) {
	c, db, _, op := delivery(t)
	r := take(t, db, op, 1, holder)
	x := acquire(t, db, r, r["claim_id"])
	if v := settle(t, db, op, r, "permanent", 0); v["status"] != "quarantined" {
		t.Fatal(v)
	}
	if value(t, c, "SELECT state FROM integration.sync_set WHERE set_id=$1", x["set_id"]) != "abandoned" {
		t.Fatal("live acquisition survived quarantine")
	}
	_, e := invoke(t, db, "synthetic_sync", "SELECT integration.complete_sync($1,$2,$3,$4,$5)::text", entity, book, r["stream_id"], x["set_id"], x["fence"])
	if e == nil {
		t.Fatal("late publication allowed")
	}
}
func TestDeliveryExpiredLeaseNewClaim(t *testing.T) {
	c, db, _, op := delivery(t)
	r := take(t, db, op, 1, holder)
	x := acquire(t, db, r, r["claim_id"])
	exec(t, c, "UPDATE integration.sync_delivery_operation SET run_lease_until=clock_timestamp()-interval '1 second' WHERE operation_id=$1", op)
	exec(t, c, "UPDATE integration.sync_stream SET lease_until=clock_timestamp()-interval '1 second' WHERE stream_id=$1", r["stream_id"])
	if v := take(t, db, op, 1, holder2); v["status"] != "not_due" {
		t.Fatal(v)
	}
	if value(t, c, "SELECT state FROM integration.sync_set WHERE set_id=$1", x["set_id"]) != "abandoned" {
		t.Fatal("expired acquisition not fenced")
	}
	if value(t, c, "SELECT (claim_id<>$2)::text FROM integration.sync_delivery_operation WHERE operation_id=$1", op, r["claim_id"]) != "true" {
		t.Fatal("expired claim not replaced")
	}
}
func TestDeliveryAuthorityAndImmutableReceipts(t *testing.T) {
	c, db, bid, op := delivery(t)
	r := take(t, db, op, 1, holder)
	settle(t, db, op, r, "unknown", 0)
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='import'", reader)
	_, e := invoke(t, db, "synthetic_sync", "SELECT integration.sync_delivery_settle($1,$2,$3,'unknown',0)::text", op, holder, r["fence"])
	requireCode(t, e, "P2901")
	_, e = invoke(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_candidates($1,25)::text", bid)
	requireCode(t, e, "P2901")
	_, e = c.Exec(context.Background(), "DELETE FROM integration.sync_delivery_run_receipt")
	if e == nil {
		t.Fatal("receipt mutable")
	}
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.sync_delivery_claim($1,1,$2)::text", op, holder)
	requireCode(t, e, "42501")
}

// TestDeliveryNullStateConstraints rejects partial lease, age and terminal metadata.
func TestDeliveryNullStateConstraints(t *testing.T) {
	c, _, _, op := delivery(t)
	for _, assignment := range []string{"first_claimed_at=clock_timestamp()", "deadline_at=clock_timestamp()", "run_holder='00000000-0000-4000-8000-000000000091'", "run_lease_until=clock_timestamp()", "claim_binding_revision=1", "finished_at=clock_timestamp()", "terminal_code='completed'"} {
		_, e := c.Exec(context.Background(), "UPDATE integration.sync_delivery_operation SET "+assignment+" WHERE operation_id=$1", op)
		requireCode(t, e, "23514")
	}
}

// TestDeliveryManualRecovery retains terminal receipts and starts a distinct business claim.
func TestDeliveryManualRecovery(t *testing.T) {
	for _, diagnostic := range []string{"unsupported_currency", "amount_magnitude_exceeded", "amount_scale_exceeded", "currency_precision_exceeded"} {
		t.Run(diagnostic, func(t *testing.T) {
			c, db, bid, op := delivery(t)
			r := take(t, db, op, 1, holder)
			x := acquire(t, db, r, r["claim_id"])
			_, err := invoke(t, db, "synthetic_sync", "SELECT integration.reject_sync_decode($1,$2,$3,$4,$5,'raw-provider-value')::text", entity, book, r["stream_id"], x["set_id"], x["fence"])
			requireCode(t, err, "P2600")
			if value(t, c, "SELECT state FROM integration.sync_set WHERE set_id=$1", x["set_id"]) != "staging" {
				t.Fatal("invalid diagnostic mutated set")
			}
			call(t, db, "synthetic_sync", "SELECT integration.reject_sync_decode($1,$2,$3,$4,$5,$6)::text", entity, book, r["stream_id"], x["set_id"], x["fence"], diagnostic)
			settle(t, db, op, r, "permanent", 0)
			st := object(t, call(t, db, "synthetic_runtime", "SELECT integration.connection_stream_status($1,$2,$3,$4)::text", actor, entity, book, r["stream_id"]))
			if st["delivery_state"] != "quarantined" || st["decoder_error"] != diagnostic {
				t.Fatal("failure hidden", st)
			}
			body := raw(map[string]any{"action": "request", "expected_item_generation": st["item_generation"], "expected_binding_revision": st["binding_revision"], "expected_stream_revision": st["stream_revision"]})
			q := "SELECT integration.mutate_connection_stream($1,$2,$3,$4,$5,$6)::text"
			_, err = invoke(t, db, "synthetic_runtime", q, other, entity, book, r["stream_id"], "recovery", body)
			requireCode(t, err, "P2701")
			results := make(chan struct {
				receipt string
				err     error
			}, 2)
			for i := 0; i < 2; i++ {
				go func() {
					receipt, err := invoke(t, db, "synthetic_runtime", q, actor, entity, book, r["stream_id"], "recovery", body)
					results <- struct {
						receipt string
						err     error
					}{receipt, err}
				}()
			}
			first, again := <-results, <-results
			if first.err != nil || again.err != nil || first.receipt != again.receipt {
				t.Fatal("recovery replay changed receipt")
			}
			if value(t, c, "SELECT count(*)::text FROM integration.sync_delivery_recovery WHERE operation_id=$1", op) != "1" {
				t.Fatal("missing recovery")
			}
			if value(t, c, "SELECT state||':'||terminal_code FROM integration.sync_delivery_operation WHERE operation_id=$1", op) != "quarantined:permanent_failure" {
				t.Fatal("terminal history rewritten")
			}
			_, err = c.Exec(context.Background(), "DELETE FROM integration.sync_delivery_recovery WHERE operation_id=$1", op)
			requireCode(t, err, "P2108")
			_, err = c.Exec(context.Background(), "UPDATE integration.sync_decode_diagnostic SET code='invalid_field' WHERE set_id=$1", x["set_id"])
			requireCode(t, err, "P2108")
			call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_discover($1,25)::text", bid)
			fresh := value(t, c, "SELECT operation_id::text FROM integration.sync_delivery_operation WHERE stream_id=$1 AND state='ready'", r["stream_id"])
			next := take(t, db, fresh, 1, holder2)
			if next["claim_id"] == r["claim_id"] {
				t.Fatal("failed claim reused")
			}
			// Old queued work cannot acquire or publish into the replacement.
			old := take(t, db, op, 1, holder)
			if old["status"] != "quarantined" {
				t.Fatal("old task resumed")
			}
			nx := acquire(t, db, next, next["claim_id"])
			finish(t, db, next, nx)
			if settleWithHolder(t, db, fresh, next, holder2)["status"] != "done" {
				t.Fatal("replacement failed")
			}
		})
	}
}

// settleWithHolder finishes the replacement through its independently fenced delivery holder.
func settleWithHolder(t *testing.T, db, op string, r map[string]any, h string) map[string]any {
	return object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_delivery_settle($1,$2,$3,'completed',0)::text", op, h, r["fence"]))
}

// TestDeliveryManualRecoveryRejectsUnknown preserves quarantine when a result is uncertain.
func TestDeliveryManualRecoveryRejectsUnknown(t *testing.T) {
	c, db, _, op := delivery(t)
	r := take(t, db, op, 1, holder)
	x := acquire(t, db, r, r["claim_id"])
	abandon(t, db, r, x, "provider_failure")
	exec(t, c, "UPDATE integration.sync_delivery_operation SET run_lease_until=clock_timestamp()-interval '1 second' WHERE operation_id=$1", op)
	call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_recover($1)::text", op)
	st := object(t, call(t, db, "synthetic_runtime", "SELECT integration.connection_stream_status($1,$2,$3,$4)::text", actor, entity, book, r["stream_id"]))
	body := raw(map[string]any{"action": "request", "expected_item_generation": st["item_generation"], "expected_binding_revision": st["binding_revision"], "expected_stream_revision": st["stream_revision"]})
	_, err := invoke(t, db, "synthetic_runtime", "SELECT integration.mutate_connection_stream($1,$2,$3,$4,'unknown-recovery',$5)::text", actor, entity, book, r["stream_id"], body)
	requireCode(t, err, "P2804")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_delivery_recovery") != "0" {
		t.Fatal("unknown recovered")
	}
}
