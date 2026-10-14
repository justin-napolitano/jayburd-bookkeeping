//go:build integration

package syncstate

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
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

var proof = flag.Bool("sync-proof", false, "Use only the disposable internal registry database")

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
	if !*proof {
		t.Skip("use scripts/test-sync-state.sh")
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
	db := "sync_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE sync_template")
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

// Fixed fictional identifiers belong only to disposable proof databases.
const sourceID = "00000000-0000-4000-8000-000000000080"
const csvNS = "00000000-0000-4000-8000-000000000081"
const claimID = "00000000-0000-4000-8000-000000000082"

// call fails only on unexpected fixed diagnostics; all data below is fictional.
func call(t testing.TB, db, user, q string, args ...any) string {
	t.Helper()
	out, err := invoke(t, db, user, q, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// object decodes synthetic SQL projections without numeric-money conversion.
func object(t testing.TB, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal([]byte(s), &v); e != nil {
		t.Fatal(e)
	}
	return v
}

// enrollment constructs an explicit synthetic operator attestation.
func enrollment() map[string]any {
	return map[string]any{"source_account_id": sourceID, "expected_binding_revision": "1", "provider": "plaid", "environment": "sandbox", "item_id": "synthetic-item", "account_id": "synthetic-account", "credential_ref": "00000000-0000-4000-8000-000000000083", "credential_version": "1", "consent_principal_id": actor, "consent_evidence_ref": "00000000-0000-4000-8000-000000000084", "consent_expires_at": "2099-01-01T00:00:00Z", "service_principal_id": reader, "worker_login": "synthetic_sync", "synthetic": true}
}

// setup installs the standard USD source fixture.
func setup(t *testing.T) (*pgx.Conn, string, map[string]any) { return setupCurrency(t, currency) }

// setupCurrency binds one source to independently installed currency metadata.
func setupCurrency(t *testing.T, currencyValue string) (*pgx.Conn, string, map[string]any) {
	c, db := fixture(t)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'import',NULL),($1,$3,'import',NULL)`, entity, actor, reader)
	exec(t, c, `INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by) VALUES($1,$2,$3,$4,$5,'2026-01-05',$6,$7)`, entity, book, sourceID, account, currencyValue, csvNS, actor)
	exec(t, c, `INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by) VALUES($1,$2,$3,$4,'jayburd-synthetic-statement-csv/1',$3,'SYNTHETIC',false,$5)`, entity, book, csvNS, sourceID, actor)
	exec(t, c, `INSERT INTO banking.source_permission SELECT $1,$2,$3,p,cap,NULL FROM unnest(ARRAY[$4::uuid,$5::uuid]) p CROSS JOIN unnest(ARRAY['read','import']) cap`, entity, book, sourceID, actor, reader)
	out := call(t, db, "synthetic_operator", "SELECT integration.enroll_sync_stream($1,$2,$3,$4,$5)::text", actor, entity, book, "enroll", raw(enrollment()))
	return c, db, object(t, out)
}

// claim obtains a real login-bound fenced acquisition.
func claim(t testing.TB, db string, s map[string]any, k string) map[string]any {
	return object(t, call(t, db, "synthetic_sync", "SELECT integration.claim_sync($1,$2,$3,$4)::text", entity, book, s["stream_id"], k))
}

// row builds independent normalized source facts for SQL boundary tests.
func row(id, amount string) map[string]any {
	return map[string]any{"account_id": "synthetic-account", "transaction_id": id, "pending": false, "pending_transaction_id": nil, "date": "2026-01-05", "authorized_date": nil, "datetime": nil, "authorized_datetime": nil, "iso_currency_code": "USD", "currency_version_id": currency, "amount_lexeme": amount, "amount": amount, "direction": "outflow", "eligible_for_review": true, "name": "Synthetic purchase", "merchant_name": nil, "original_description": nil}
}

// page wraps synthetic rows in the reviewed complete-set protocol.
func page(idx int, cursor any, next string, more bool, added, modified, removed []any) map[string]any {
	if added == nil {
		added = []any{}
	}
	if modified == nil {
		modified = []any{}
	}
	if removed == nil {
		removed = []any{}
	}
	return map[string]any{"page_index": idx, "request_cursor": cursor, "next_cursor": next, "has_more": more, "transactions_update_status": "HISTORICAL_UPDATE_COMPLETE", "request_id": "synthetic-request", "sha256": strings.Repeat("a", 64), "added": added, "modified": modified, "removed": removed}
}

// stage commits one private page through the restricted worker login.
func stage(t testing.TB, db string, s, x map[string]any, p map[string]any) (string, error) {
	return invoke(t, db, "synthetic_sync", "SELECT integration.stage_sync_page($1,$2,$3,$4,$5,$6)::text", entity, book, s["stream_id"], x["set_id"], x["fence"], raw(p))
}

// stageOK requires successful staging while retaining fixed diagnostic failures.
func stageOK(t testing.TB, db string, s, x map[string]any, p map[string]any) {
	t.Helper()
	if _, e := stage(t, db, s, x, p); e != nil {
		t.Fatal(e)
	}
}

// complete attempts atomic publication as the actual worker login.
func complete(t testing.TB, db string, s, x map[string]any) (string, error) {
	return invoke(t, db, "synthetic_sync", "SELECT integration.complete_sync($1,$2,$3,$4,$5)::text", entity, book, s["stream_id"], x["set_id"], x["fence"])
}

// completeOK requires publication and returns the immutable receipt.
func completeOK(t testing.TB, db string, s, x map[string]any) string {
	t.Helper()
	v, e := complete(t, db, s, x)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

// Exact independent expectations prove a complete set creates review evidence only.
func TestSyncPublicationAtomicity(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	p := page(0, nil, "cursor1", true, []any{row("one", "12.5")}, nil, nil)
	stageOK(t, db, s, x, p)
	stageOK(t, db, s, x, p)
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "0" {
		t.Fatal("partial publication")
	}
	_, e := complete(t, db, s, x)
	requireCode(t, e, "P2604")
	stageOK(t, db, s, x, page(1, "cursor1", "cursor2", false, []any{row("two", "12.5")}, nil, nil))
	receipt := completeOK(t, db, s, x)
	if completeOK(t, db, s, x) != receipt {
		t.Fatal("receipt changed")
	}
	for q, w := range map[string]string{"SELECT count(*)::text FROM banking.source_observation": "2", "SELECT count(*)::text FROM banking.review_item WHERE state='overlap_hold'": "2", "SELECT cursor_value FROM integration.sync_stream": "cursor2", "SELECT source_revision::text FROM banking.source_account": "1", "SELECT count(*)::text FROM ledger.journal_entry": "0", "SELECT count(*)::text FROM integration.outbox_message": "0"} {
		if got := value(t, c, q); got != w {
			t.Fatalf("%s got%s want%s", q, got, w)
		}
	}
	y := claim(t, db, s, "00000000-0000-4000-8000-000000000085")
	stageOK(t, db, s, y, page(0, "cursor2", "cursor3", false, []any{row("one", "12.5")}, nil, nil))
	if object(t, completeOK(t, db, s, y))["changed_count"] != "0" {
		t.Fatal("exact repeat")
	}
}

func TestSyncRollbackAndDuplicateIdentity(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	stageOK(t, db, s, x, page(0, nil, "cursor1", false, []any{row("one", "12.5")}, []any{row("unknown", "1")}, nil))
	_, e := complete(t, db, s, x)
	requireCode(t, e, "P2604")
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "0" || value(t, c, "SELECT coalesce(cursor_value,'NULL') FROM integration.sync_stream") != "NULL" {
		t.Fatal("partial failure escaped")
	}
	call(t, db, "synthetic_sync", "SELECT integration.abandon_sync($1,$2,$3,$4,$5,'decoder_rejected')::text", entity, book, s["stream_id"], x["set_id"], x["fence"])
	y := claim(t, db, s, "00000000-0000-4000-8000-000000000085")
	stageOK(t, db, s, y, page(0, nil, "cursor2", false, []any{row("same", "1"), row("same", "1")}, nil, nil))
	_, e = complete(t, db, s, y)
	requireCode(t, e, "P2604")
}

func TestSyncFenceAndWakeup(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	if claim(t, db, s, claimID)["set_id"] != x["set_id"] {
		t.Fatal("claim replay")
	}
	stageOK(t, db, s, x, page(0, nil, "cursor1", true, []any{row("old", "1")}, nil, nil))
	exec(t, c, "UPDATE integration.sync_stream SET lease_until=clock_timestamp()-interval '1 second'")
	y := claim(t, db, s, "00000000-0000-4000-8000-000000000085")
	if y["starting_cursor"] != nil || y["next_page_index"] != float64(0) {
		t.Fatal("partial adoption")
	}
	_, e := stage(t, db, s, x, page(1, "cursor1", "cursor2", false, nil, nil, nil))
	requireCode(t, e, "P2603")
	sig := call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'wake')::text", entity, book, s["stream_id"])
	if call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'wake')::text", entity, book, s["stream_id"]) != sig {
		t.Fatal("signal replay")
	}
	stageOK(t, db, s, y, page(0, nil, "cursor3", false, nil, nil, nil))
	completeOK(t, db, s, y)
	if value(t, c, "SELECT (requested_generation>completed_generation)::text FROM integration.sync_stream") != "true" {
		t.Fatal("lost wakeup")
	}
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "0" {
		t.Fatal("old partial adopted")
	}
}

func TestProviderSourceLifecycle(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	pending := row("pending", "12.5")
	pending["pending"] = true
	pending["eligible_for_review"] = false
	stageOK(t, db, s, x, page(0, nil, "cursor1", false, []any{pending}, nil, nil))
	completeOK(t, db, s, x)
	y := claim(t, db, s, "00000000-0000-4000-8000-000000000085")
	child := row("booked", "13")
	child["pending_transaction_id"] = "pending"
	stageOK(t, db, s, y, page(0, "cursor1", "cursor2", true, []any{child}, nil, nil))
	stageOK(t, db, s, y, page(1, "cursor2", "cursor3", false, nil, nil, []any{map[string]any{"account_id": "synthetic-account", "transaction_id": "pending"}}))
	completeOK(t, db, s, y)
	if value(t, c, "SELECT count(*)::text FROM banking.provider_pending_link") != "1" || value(t, c, "SELECT count(*)::text FROM banking.observation_revision WHERE source_state='removed'") != "1" {
		t.Fatal("pending lineage")
	}
	z := claim(t, db, s, "00000000-0000-4000-8000-000000000086")
	stageOK(t, db, s, z, page(0, "cursor3", "cursor4", false, nil, []any{row("booked", "14")}, nil))
	completeOK(t, db, s, z)
	if value(t, c, "SELECT count(*)::text FROM banking.review_item WHERE state='correction_hold'") != "2" {
		t.Fatal("correction not held")
	}
	_, e := c.Exec(context.Background(), "UPDATE banking.observation_revision SET description='rewrite'")
	requireCode(t, e, "P2108")
}

func TestSyncAuthorityAndState(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	stageOK(t, db, s, x, page(0, nil, "cursor1", false, []any{row("one", "1")}, nil, nil))
	receipt := completeOK(t, db, s, x)
	exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", actor)
	_, e := complete(t, db, s, x)
	requireCode(t, e, "P2601")
	exec(t, c, "UPDATE banking.source_permission SET revoked_at=NULL WHERE principal_id=$1 AND capability='import'", actor)
	if completeOK(t, db, s, x) != receipt {
		t.Fatal("reauthorized receipt")
	}
	state := map[string]any{"expected_generation": "1", "state": "paused"}
	call(t, db, "synthetic_operator", "SELECT integration.set_sync_item_state($1,$2,$3,$4,'pause',$5)::text", actor, entity, book, s["sync_item_id"], raw(state))
	_, e = complete(t, db, s, x)
	requireCode(t, e, "P2601")
	state["expected_generation"] = "2"
	state["state"] = "active"
	call(t, db, "synthetic_operator", "SELECT integration.set_sync_item_state($1,$2,$3,$4,'resume',$5)::text", actor, entity, book, s["sync_item_id"], raw(state))
	if completeOK(t, db, s, x) != receipt {
		t.Fatal("resume receipt")
	}
	exec(t, c, "ALTER ROLE synthetic_sync REPLICATION")
	_, e = complete(t, db, s, x)
	requireCode(t, e, "P2601")
	exec(t, c, "ALTER ROLE synthetic_sync NOREPLICATION")
	exec(t, c, "GRANT bookkeeping_api TO synthetic_sync")
	_, e = complete(t, db, s, x)
	requireCode(t, e, "P2601")
	_, e = invoke(t, db, "synthetic_sync", "SELECT banking.list_sources($1,$2,$3)::text", reader, entity, book)
	requireCode(t, e, "P2501")
	exec(t, c, "REVOKE bookkeeping_api FROM synthetic_sync")
}

func TestSyncMoneyAndPageValidation(t *testing.T) {
	_, db, s := setup(t)
	x := claim(t, db, s, claimID)
	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
		code   string
	}{
		{"rounded", func(v map[string]any) { v["amount"] = "1.001"; v["amount_lexeme"] = "1.001" }, "P2604"},
		{"wrong sign", func(v map[string]any) { v["amount_lexeme"] = "-1" }, "P2604"},
		{"wrong currency", func(v map[string]any) { v["iso_currency_code"] = "JPY" }, "P2604"},
		{"exponent", func(v map[string]any) { v["amount_lexeme"] = "1e101" }, "P2600"},
		{"wrong account", func(v map[string]any) { v["account_id"] = "another" }, "P2604"},
	} {
		t.Run(test.name, func(t *testing.T) {
			v := row("bad", "1")
			test.mutate(v)
			_, e := stage(t, db, s, x, page(0, nil, "cursor1", false, []any{v}, nil, nil))
			requireCode(t, e, test.code)
		})
	}
	zero := row("zero", "0")
	zero["direction"] = "zero"
	zero["eligible_for_review"] = false
	significant := row("exact", "1")
	significant["amount_lexeme"] = "1." + strings.Repeat("0", 50)
	p := page(0, nil, "cursor1", false, []any{zero, significant}, nil, nil)
	stageOK(t, db, s, x, p)
	p["sha256"] = strings.Repeat("b", 64)
	_, e := stage(t, db, s, x, p)
	requireCode(t, e, "P2602")
	completeOK(t, db, s, x)
}

// Decoder bytes, normalized publication and persisted source facts are tested together.
func TestSyncDecoderPublication(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	payload := []byte(`{"accounts":[{"account_id":"synthetic-account","type":"depository"}],"added":[{"account_id":"synthetic-account","transaction_id":"decoded","amount":-1.25e1,"iso_currency_code":"USD","unofficial_currency_code":null,"date":"2026-01-05","datetime":"2026-01-05T12:00:00+23:59","pending":false,"pending_transaction_id":null,"name":"Synthetic decoded\ndeposit","merchant_name":"","original_description":""}],"modified":[],"removed":[],"next_cursor":"Y3Vyc29y","has_more":false,"request_id":"synthetic-request","transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`)
	decoded, err := plaid.DecodePage(payload, plaid.Expected{AccountID: "synthetic-account", Currencies: []plaid.Currency{{Code: "USD", VersionID: currency, MinorUnit: 2, ValidFrom: "2000-01-01"}}})
	if err != nil {
		t.Fatal(err)
	}
	p := object(t, raw(decoded))
	p["page_index"] = 0
	p["request_cursor"] = nil
	stageOK(t, db, s, x, p)
	completeOK(t, db, s, x)
	if value(t, c, "SELECT amount::text||'|'||direction||'|'||provider_amount_lexeme FROM banking.observation_revision") != "12.500000000|inflow|-1.25e1" {
		t.Fatal("decoded source mismatch")
	}
}

// A synthetic owner fixture installs a pre-existing approved provider namespace only
// to exercise correction of an already-posted source. Product enrollment always holds overlap.
func TestSyncPostedEvidenceRemainsOriginal(t *testing.T) {
	c, db, s := setup(t)
	exec(t, c, "ALTER TABLE banking.source_namespace DISABLE TRIGGER namespace_guard")
	exec(t, c, "UPDATE banking.source_namespace SET overlap_review_required=false WHERE namespace_id=$1", s["namespace_id"])
	exec(t, c, "ALTER TABLE banking.source_namespace ENABLE TRIGGER namespace_guard")
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL),($1,$2,'post',NULL)`, entity, reader)
	exec(t, c, `INSERT INTO banking.source_permission VALUES($1,$2,$3,$4,'review',NULL)`, entity, book, sourceID, reader)
	policy := "00000000-0000-4000-8000-000000000060"
	expense := "00000000-0000-4000-8000-000000000051"
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())`, entity, book, policy, currency, actor)
	exec(t, c, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000061','2026-01-01','2026-02-01')`, entity, book)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'6000','Synthetic expense','expense','debit',false,'2026-01-01',NULL)`, entity, book, expense)
	x := claim(t, db, s, claimID)
	stageOK(t, db, s, x, page(0, nil, "cursor1", false, []any{row("posted", "12.5")}, nil, nil))
	completeOK(t, db, s, x)
	rid := value(t, c, "SELECT review_id::text FROM banking.review_item")
	detail := object(t, call(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", reader, entity, book, rid))
	input := map[string]any{"expected_review_revision": detail["review_revision"], "source_revision": detail["source_revision"], "binding_revision": detail["binding_revision"], "expected_book_revision": detail["book_revision"], "category_account_id": expense, "policy_id": policy, "accounting_date": "2026-01-05", "description": "Synthetic reviewed posting"}
	d := object(t, call(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'draft',$5)::text", reader, entity, book, rid, raw(input)))
	body := map[string]any{"review_id": d["review_id"], "draft_id": d["draft_id"], "review_revision": d["review_revision"], "source_revision": d["source_revision"], "binding_revision": d["binding_revision"], "expected_book_revision": d["book_revision"], "posting_body": d["posting_body"]}
	command := object(t, call(t, db, "synthetic_runtime", "SELECT ledger.register_command($1,'post',$2,$3)::text", reader, claimID, raw(map[string]any{"operation": "source_post", "entity_id": entity, "book_id": book, "body": body})))
	call(t, db, "synthetic_runtime", "SELECT ledger.transition_command($1,$2,$3,'execute','{}')::text", reader, command["command_id"], claimID)
	original := call(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", reader, entity, book, rid)
	originalEvidence := raw(object(t, original)["evidence"])
	journal := object(t, original)["journal_id"]
	for n := 0; n < 3; n++ {
		k := fmt.Sprintf("00000000-0000-4000-8000-%012d", 90+n)
		y := claim(t, db, s, k)
		var modified, removed []any
		if n < 2 {
			modified = []any{row("posted", fmt.Sprint(15+n))}
		} else {
			removed = []any{map[string]any{"account_id": "synthetic-account", "transaction_id": "posted"}}
		}
		stageOK(t, db, s, y, page(0, fmt.Sprintf("cursor%d", n+1), fmt.Sprintf("cursor%d", n+2), false, nil, modified, removed))
		completeOK(t, db, s, y)
		next := object(t, call(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", reader, entity, book, rid))
		if next["state"] != "posted" || next["amount"] != "12.5" || next["journal_id"] != journal || raw(next["evidence"]) != originalEvidence {
			t.Fatal("original posted source changed")
		}
		if next["provider_exceptions"].(map[string]any)["total_count"] != fmt.Sprint(n+1) {
			t.Fatal("missing exception")
		}
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" || value(t, c, "SELECT count(*)::text FROM integration.outbox_message") != "1" {
		t.Fatal("correction posted automatically")
	}
}

func TestSyncLimitsAndNullInput(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	nullpage := page(0, nil, "cursor1", false, nil, nil, nil)
	nullpage["transactions_update_status"] = nil
	_, e := stage(t, db, s, x, nullpage)
	requireCode(t, e, "P2600")
	for _, stamp := range []string{"2026-01-05T24:00:00Z", "2026-01-05T23:59:60Z", "2026-01-05T12:00:00+99:00", "2026-02-30T12:00:00Z"} {
		r := row("stamp", "1")
		r["datetime"] = stamp
		_, e = stage(t, db, s, x, page(0, nil, "cursor1", false, []any{r}, nil, nil))
		requireCode(t, e, "P2600")
	}
	nullclaim := map[string]any{"set_id": x["set_id"], "fence": nil}
	_, e = complete(t, db, s, nullclaim)
	requireCode(t, e, "P2603")
	rows := []any{}
	for i := 0; i < 501; i++ {
		rows = append(rows, row(fmt.Sprint(i), "1"))
	}
	_, e = stage(t, db, s, x, page(0, nil, "cursor1", false, rows, nil, nil))
	requireCode(t, e, "P2605")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_page") != "0" {
		t.Fatal("invalid page persisted")
	}
	empty := page(0, nil, "", false, nil, nil, nil)
	empty["transactions_update_status"] = "TRANSACTIONS_UPDATE_STATUS_UNKNOWN"
	stageOK(t, db, s, x, empty)
	stageOK(t, db, s, x, empty)
	completeOK(t, db, s, x)
	if value(t, c, "SELECT coalesce(cursor_value,'NULL') FROM integration.sync_stream") != "NULL" {
		t.Fatal("initialization advanced cursor")
	}
	for _, key := range []string{"synthetic", "provider", "environment", "consent_expires_at"} {
		v := enrollment()
		v[key] = nil
		_, e = invoke(t, db, "synthetic_operator", "SELECT integration.enroll_sync_stream($1,$2,$3,$4,$5)::text", actor, entity, book, "null-"+key, raw(v))
		requireCode(t, e, "P2600")
	}
}

func TestSyncPrivileges(t *testing.T) {
	c, db, _ := setup(t)
	_, guardErr := c.Exec(context.Background(), "UPDATE integration.sync_stream SET account_id='rewritten'")
	requireCode(t, guardErr, "P2602")
	if value(t, c, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname IN ('ledger','banking','integration','reference') AND has_function_privilege('synthetic_sync',p.oid,'EXECUTE')`) != "7" {
		t.Fatal("unexpected worker execute privilege")
	}
	worker := connect(t, db, "synthetic_sync")
	defer worker.Close(context.Background())
	for _, q := range []string{"SELECT * FROM integration.sync_item", "UPDATE banking.source_account SET source_revision=8", "SELECT integration.enroll_sync_stream(NULL,NULL,NULL,'x','{}')", "SELECT integration.sync_login('synthetic_sync')", "SELECT banking.get_review_item(NULL,NULL,NULL,NULL)"} {
		_, e := worker.Exec(context.Background(), q)
		requireCode(t, e, "42501")
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	_, e := runtime.Exec(context.Background(), "SELECT integration.claim_sync(NULL,NULL,NULL,NULL)")
	requireCode(t, e, "42501")
}

// Currency-specific minor units are enforced independently of the provider amount sign.
func TestSyncCurrencies(t *testing.T) {
	for _, tc := range []struct{ code, id, amount, bad string }{{"JPY", "00000000-0000-4000-8000-000000000006", "125", "1.5"}, {"KWD", "00000000-0000-4000-8000-000000000007", "1.234", "1.2345"}} {
		t.Run(tc.code, func(t *testing.T) {
			c, db, s := setupCurrency(t, tc.id)
			x := claim(t, db, s, claimID)
			r := row("currency", tc.bad)
			r["iso_currency_code"] = tc.code
			r["currency_version_id"] = tc.id
			_, e := stage(t, db, s, x, page(0, nil, "cursor1", false, []any{r}, nil, nil))
			requireCode(t, e, "P2604")
			r["amount"] = tc.amount
			r["amount_lexeme"] = "-" + tc.amount
			r["direction"] = "inflow"
			stageOK(t, db, s, x, page(0, nil, "cursor1", false, []any{r}, nil, nil))
			completeOK(t, db, s, x)
			if value(t, c, "SELECT trim_scale(amount)::text FROM banking.observation_revision") != tc.amount {
				t.Fatal("currency amount changed")
			}
		})
	}
}

// Contending workers serialize at the same stream/book boundary; late signals remain pending.
func TestSyncConcurrentCompletionAndSignal(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	stageOK(t, db, s, x, page(0, nil, "cursor1", false, []any{row("one", "1")}, nil, nil))
	type outcome struct {
		v string
		e error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() { <-start; v, e := complete(t, db, s, x); results <- outcome{v, e} }()
	}
	close(start)
	one, two := <-results, <-results
	if one.e != nil || two.e != nil || one.v != two.v {
		t.Fatalf("concurrent receipts %v %v", one.e, two.e)
	}
	y := claim(t, db, s, "00000000-0000-4000-8000-000000000085")
	stageOK(t, db, s, y, page(0, "cursor1", "cursor2", false, nil, nil, nil))
	start = make(chan struct{})
	go func() { <-start; v, e := complete(t, db, s, y); results <- outcome{v, e} }()
	go func() {
		<-start
		v, e := invoke(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'late-wake')::text", entity, book, s["stream_id"])
		results <- outcome{v, e}
	}()
	close(start)
	one, two = <-results, <-results
	if one.e != nil || two.e != nil {
		t.Fatalf("concurrent wakeup %v %v", one.e, two.e)
	}
	if value(t, c, "SELECT (requested_generation>completed_generation)::text FROM integration.sync_stream") != "true" || value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "1" {
		t.Fatal("lost signal or duplicate publication")
	}
}

// Whole-set caps fail before cursor/evidence publication, even when each page is valid.
func TestSyncWholeSetBounds(t *testing.T) {
	for _, mode := range []string{"changes", "bytes", "pages"} {
		t.Run(mode, func(t *testing.T) {
			c, db, s := setup(t)
			x := claim(t, db, s, claimID)
			cursor := any(nil)
			for idx := 0; idx < 11; idx++ {
				rows := []any{}
				n := 0
				if mode == "changes" {
					n = 500
					if idx == 2 {
						n = 1
					}
				}
				if mode == "bytes" {
					n = 300
				}
				for j := 0; j < n; j++ {
					r := row(fmt.Sprintf("r%d_%d", idx, j), "1")
					if mode == "bytes" {
						r["name"] = strings.Repeat("é", 500)
						r["merchant_name"] = strings.Repeat("é", 500)
						r["original_description"] = strings.Repeat("é", 500)
					}
					rows = append(rows, r)
				}
				next := fmt.Sprintf("cursor%d", idx)
				_, e := stage(t, db, s, x, page(idx, cursor, next, true, rows, nil, nil))
				if mode == "pages" && idx == 10 {
					requireCode(t, e, "P2600")
					break
				}
				if e != nil {
					requireCode(t, e, "P2605")
					break
				}
				cursor = next
			}
			if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "0" || value(t, c, "SELECT coalesce(cursor_value,'NULL') FROM integration.sync_stream") != "NULL" {
				t.Fatal("bounded set published")
			}
			_, e := complete(t, db, s, x)
			requireCode(t, e, "P2604")
		})
	}
}

// An empty initialization response can never terminate a previously nonempty loop.
func TestSyncRejectsEmptyInitializationTail(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	stageOK(t, db, s, x, page(0, nil, "cursor1", true, []any{row("one", "1")}, nil, nil))
	tail := page(1, "cursor1", "", false, nil, nil, nil)
	tail["transactions_update_status"] = "NOT_READY"
	_, e := stage(t, db, s, x, tail)
	requireCode(t, e, "P2604")
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "0" {
		t.Fatal("partial initialization published")
	}
	_, e = complete(t, db, s, x)
	requireCode(t, e, "P2604")
}

// Public source metadata uses a tagged identity; no fabricated CSV fields or secret reference.
func TestSyncSourceProjectionAndBinding(t *testing.T) {
	c, db, s := setup(t)
	x := claim(t, db, s, claimID)
	out := object(t, call(t, db, "synthetic_runtime", "SELECT banking.get_source($1,$2,$3,$4)::text", reader, entity, book, sourceID))
	found := false
	for _, entry := range out["namespaces"].([]any) {
		n := entry.(map[string]any)
		if n["profile"] == plaid.Profile {
			found = true
			if n["item_id"] != "synthetic-item" || n["provider_account_id"] != "synthetic-account" || n["overlap_review_required"] != true {
				t.Fatal("provider identity")
			}
			if _, ok := n["synthetic_dataset_id"]; ok {
				t.Fatal("fabricated CSV identity")
			}
		}
	}
	if !found {
		t.Fatal("missing provider namespace")
	}
	exec(t, c, "UPDATE banking.source_account SET binding_revision=binding_revision+1")
	_, e := stage(t, db, s, x, page(0, nil, "cursor1", false, nil, nil, nil))
	requireCode(t, e, "P2603")
	_, e = invoke(t, db, "synthetic_sync", "SELECT integration.claim_sync($1,$2,$3,$4)::text", entity, book, s["stream_id"], claimID)
	requireCode(t, e, "P2603")
	_, e = invoke(t, db, "synthetic_sync", "SELECT integration.claim_sync($1,$2,$3,$4)::text", other, book, s["stream_id"], claimID)
	requireCode(t, e, "P2601")
	exec(t, c, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", reader)
	_, e = complete(t, db, s, x)
	requireCode(t, e, "P2601")
}
