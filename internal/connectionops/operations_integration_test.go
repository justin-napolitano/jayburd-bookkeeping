//go:build integration

package connectionops

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

var proof = flag.Bool("connection-ops-proof", false, "Use only the disposable internal registry database")

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
		t.Skip("use scripts/test-connection-state.sh")
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
	db := "connection_ops_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE connection_ops_template")
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

// status uses current interactive identity and exposes no private credential metadata.
func status(t testing.TB, db string, stream any) map[string]any {
	return object(t, call(t, db, "synthetic_runtime", `SELECT integration.connection_stream_status($1,$2,$3,$4)::text`, actor, entity, book, stream))
}

// mutation pins only server-returned revisions; no status label grants authority.
func mutation(v map[string]any, action string) string {
	return raw(map[string]any{"action": action, "expected_item_generation": v["item_generation"], "expected_binding_revision": v["binding_revision"], "expected_stream_revision": v["stream_revision"]})
}

// operate executes one user-reviewed account operation through the runtime login.
func operate(t testing.TB, db string, stream any, key, body string) map[string]any {
	return object(t, call(t, db, "synthetic_runtime", `SELECT integration.mutate_connection_stream($1,$2,$3,$4,$5,$6)::text`, actor, entity, book, stream, key, body))
}

// webhookConfig enrolls the dedicated actual login through the owner boundary.
func webhookConfig(t testing.TB, db string) {
	call(t, db, "synthetic_operator", `SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','webhook-login',true)::text`)
}

// notification models already verified metadata; no signature or body is stored here.
func notification() string {
	return raw(map[string]any{"item_id": "SyntheticItem", "body_sha256": strings.Repeat("c", 64), "issued_at": time.Now().Unix(), "webhook_type": "TRANSACTIONS", "webhook_code": "SYNC_UPDATES_AVAILABLE"})
}

func TestConnectionOpsDiscoveryAndGuard(t *testing.T) {
	c, db := setup(t)
	defer c.Close(context.Background())
	a := begin(t, db, "first", "new", nil, nil, nil)
	_, e := invoke(t, db, "synthetic_runtime", `SELECT integration.begin_connection($1,$2,$3,'other',$4)::text`, actor, entity, book, raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}}))
	requireCode(t, e, "P2804")
	replay := begin(t, db, "first", "new", nil, nil, nil)
	if replay["attempt_id"] != a["attempt_id"] || replay["execute_link"] != false {
		t.Fatal("begin replay regained permission")
	}
	listing := call(t, db, "synthetic_runtime", `SELECT integration.list_connection_attempts($1,$2,$3,1,NULL)::text`, actor, entity, book)
	for _, private := range []string{"nonce", "credential_ref", "token_sha256", "claim_id"} {
		if strings.Contains(listing, private) {
			t.Fatal("private metadata leaked")
		}
	}
	out := object(t, listing)
	if out["unfinished_count"] != "1" {
		t.Fatal("unfinished count")
	}
	exec(t, c, `UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second'`)
	out = object(t, call(t, db, "synthetic_runtime", `SELECT integration.list_connection_attempts($1,$2,$3,1,NULL)::text`, actor, entity, book))
	if out["unfinished_count"] != "0" || out["items"].([]any)[0].(map[string]any)["expired_unclaimed"] != true {
		t.Fatal("expiry projection")
	}
	begin(t, db, "second", "new", nil, nil, nil)
	page := object(t, call(t, db, "synthetic_runtime", `SELECT integration.list_connection_attempts($1,$2,$3,1,NULL)::text`, actor, entity, book))
	if page["next_cursor"] == nil {
		t.Fatal("missing page")
	}
	second := object(t, call(t, db, "synthetic_runtime", `SELECT integration.list_connection_attempts($1,$2,$3,1,$4)::text`, actor, entity, book, page["next_cursor"]))
	if len(second["items"].([]any)) != 1 || second["next_cursor"] != nil || second["unfinished_count"] != "1" {
		t.Fatal("pagination count")
	}
	exec(t, c, `UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='connect'`, actor)
	_, e = invoke(t, db, "synthetic_runtime", `SELECT integration.list_connection_attempts($1,$2,$3,100,NULL)::text`, actor, entity, book)
	requireCode(t, e, "P2701")
}

func TestConnectionOpsPauseResumeFences(t *testing.T) {
	c, db, _, out := connected(t)
	defer c.Close(context.Background())
	streams := out["streams"].([]any)
	id := streams[0].(map[string]any)["stream_id"]
	otherID := streams[1].(map[string]any)["stream_id"]
	initial := status(t, db, id)
	if initial["provider_updated_at"] != nil || initial["last_completed_at"] != nil || initial["readiness"] != "NOT_READY" || initial["stream_revision"] != "1" {
		t.Fatal("fabricated initial status")
	}
	held := object(t, call(t, db, "synthetic_sync", `SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000071')::text`, entity, book, id))
	paused := operate(t, db, id, "pause", mutation(initial, "pause"))
	if paused["state"] != "paused" || paused["stream_revision"] != "2" || paused["binding_revision"] != initial["binding_revision"] || paused["lease_active"] != false {
		t.Fatal("pause pins")
	}
	if status(t, db, otherID)["state"] != "active" {
		t.Fatal("sibling affected")
	}
	_, e := invoke(t, db, "synthetic_sync", `SELECT integration.sync_context($1,$2,$3,$4,$5)::text`, entity, book, id, held["set_id"], held["fence"])
	requireCode(t, e, "P2601")
	replay := operate(t, db, id, "pause", mutation(initial, "pause"))
	if replay["stream_revision"] != "2" {
		t.Fatal("replay changed")
	}
	resumed := operate(t, db, id, "resume", mutation(paused, "resume"))
	if resumed["state"] != "active" || resumed["stream_revision"] != "3" || resumed["requested_generation"] != "2" {
		t.Fatal("resume signal")
	}
	_, e = invoke(t, db, "synthetic_runtime", `SELECT integration.mutate_connection_stream($1,$2,$3,$4,'stale',$5)::text`, actor, entity, book, id, mutation(initial, "pause"))
	requireCode(t, e, "P2803")
	operate(t, db, id, "request", mutation(resumed, "request"))
	if value(t, c, `SELECT count(*)::text FROM integration.sync_wakeup`) != "2" {
		t.Fatal("signal intent count")
	}
	if value(t, c, `SELECT count(*)::text FROM ledger.journal_entry`) != "0" {
		t.Fatal("operation posted")
	}
}

func TestConnectionOpsWebhookAtomicReplay(t *testing.T) {
	c, db, _, out := connected(t)
	defer c.Close(context.Background())
	webhookConfig(t, db)
	body := notification()
	key := strings.Repeat("a", 64)
	first := object(t, call(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, key, body))
	if first["state"] != "accepted" || first["stream_count"] != float64(2) {
		t.Fatal("notification not scoped")
	}
	second := object(t, call(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, key, body))
	if second["state"] != "duplicate" || second["receipt_id"] != first["receipt_id"] {
		t.Fatal("duplicate identity")
	}
	if value(t, c, `SELECT count(*)::text FROM integration.sync_wakeup`) != "2" || value(t, c, `SELECT count(*)::text FROM integration.webhook_receipt`) != "1" {
		t.Fatal("duplicate work")
	}
	call(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("b", 64), body)
	if value(t, c, `SELECT count(*)::text FROM integration.sync_wakeup`) != "4" {
		t.Fatal("identical later body lost")
	}
	bad := strings.Replace(body, strings.Repeat("c", 64), strings.Repeat("d", 64), 1)
	_, e := invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, key, bad)
	requireCode(t, e, "P2802")
	id := out["streams"].([]any)[0].(map[string]any)["stream_id"]
	operate(t, db, id, "pause", mutation(status(t, db, id), "pause"))
	_, e = invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, key, body)
	requireCode(t, e, "P2801")
	partial := object(t, call(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("e", 64), body))
	if partial["stream_count"] != float64(1) {
		t.Fatal("paused stream signalled")
	}
	exec(t, c, `UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='import'`, reader)
	_, e = invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("f", 64), body)
	requireCode(t, e, "P2801")
	if value(t, c, `SELECT count(*)::text FROM integration.webhook_receipt`) != "3" {
		t.Fatal("unauthorized receipt persisted")
	}
}

func TestConnectionOpsCatchupAndPrivileges(t *testing.T) {
	c, db, _, _ := connected(t)
	defer c.Close(context.Background())
	webhookConfig(t, db)
	first := object(t, call(t, db, "synthetic_sync", `SELECT integration.catchup_sync($1,$2,'00000000-0000-4000-8000-000000000091',1,NULL)::text`, entity, book))
	if len(first["items"].([]any)) != 1 || first["next_cursor"] == nil {
		t.Fatal("catchup page")
	}
	call(t, db, "synthetic_sync", `SELECT integration.catchup_sync($1,$2,'00000000-0000-4000-8000-000000000091',1,NULL)::text`, entity, book)
	call(t, db, "synthetic_sync", `SELECT integration.catchup_sync($1,$2,'00000000-0000-4000-8000-000000000091',1,$3)::text`, entity, book, first["next_cursor"])
	if value(t, c, `SELECT count(*)::text FROM integration.sync_wakeup`) != "2" {
		t.Fatal("tick duplicated generation")
	}
	for _, u := range []string{"synthetic_runtime", "synthetic_sync"} {
		_, e := invoke(t, db, u, `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("a", 64), notification())
		requireCode(t, e, "42501")
	}
	_, e := invoke(t, db, "synthetic_webhook", `SELECT integration.catchup_sync($1,$2,'00000000-0000-4000-8000-000000000091',1,NULL)::text`, entity, book)
	requireCode(t, e, "42501")
	if value(t, c, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname IN ('ledger','banking','integration') AND has_function_privilege('synthetic_webhook',p.oid,'EXECUTE')`) != "1" {
		t.Fatal("webhook function surface")
	}
	exec(t, c, `GRANT bookkeeping_api TO synthetic_webhook`)
	_, e = invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("a", 64), notification())
	requireCode(t, e, "P2801")
	exec(t, c, `REVOKE bookkeeping_api FROM synthetic_webhook`)
}

// emptyPage independently specifies a complete, unchanged provider result without invented transaction data.
func emptyPage() string {
	return raw(map[string]any{"page_index": 0, "request_cursor": nil, "next_cursor": "synthetic-completed-cursor", "has_more": false, "transactions_update_status": "HISTORICAL_UPDATE_COMPLETE", "request_id": "synthetic-ops-request", "sha256": strings.Repeat("d", 64), "added": []any{}, "modified": []any{}, "removed": []any{}})
}

func TestConnectionOpsCompletionSignalRace(t *testing.T) {
	c, db, _, out := connected(t)
	defer c.Close(context.Background())
	webhookConfig(t, db)
	id := out["streams"].([]any)[0].(map[string]any)["stream_id"]
	held := object(t, call(t, db, "synthetic_sync", `SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000070')::text`, entity, book, id))
	call(t, db, "synthetic_sync", `SELECT integration.stage_sync_page($1,$2,$3,$4,$5,$6)::text`, entity, book, id, held["set_id"], held["fence"], emptyPage())
	errs := make(chan error, 2)
	start := make(chan struct{})
	go func() {
		<-start
		_, e := invoke(t, db, "synthetic_sync", `SELECT integration.complete_sync($1,$2,$3,$4,$5)::text`, entity, book, id, held["set_id"], held["fence"])
		errs <- e
	}()
	go func() {
		<-start
		_, e := invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("e", 64), notification())
		errs <- e
	}()
	close(start)
	for range 2 {
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
	}
	current := status(t, db, id)
	if current["completed_generation"] != "1" || current["requested_generation"] != "2" || current["queued"] != true || current["last_completed_at"] == nil || current["provider_updated_at"] != nil || current["readiness"] != "HISTORICAL_UPDATE_COMPLETE" {
		t.Fatal("completion lost later wakeup", current)
	}
	if value(t, c, `SELECT count(*)::text FROM integration.sync_wakeup`) != "2" || value(t, c, `SELECT count(*)::text FROM integration.sync_completion`) != "1" {
		t.Fatal("atomic wakeup/completion receipts")
	}
	if value(t, c, `SELECT count(*)::text FROM banking.source_observation`) != "0" || value(t, c, `SELECT count(*)::text FROM ledger.journal_entry`) != "0" {
		t.Fatal("empty sync manufactured finance")
	}
}

func TestConnectionOpsMalformedAndRevokedReplay(t *testing.T) {
	c, db, _, out := connected(t)
	defer c.Close(context.Background())
	webhookConfig(t, db)
	id := out["streams"].([]any)[0].(map[string]any)["stream_id"]
	body := notification()
	v := object(t, body)
	for _, field := range []string{"item_id", "body_sha256", "issued_at", "webhook_type", "webhook_code"} {
		old := v[field]
		v[field] = nil
		_, e := invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("a", 64), raw(v))
		requireCode(t, e, "P2800")
		v[field] = old
	}
	v["issued_at"] = time.Now().Add(-time.Hour).Unix()
	_, e := invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("a", 64), raw(v))
	requireCode(t, e, "P2803")
	v = object(t, body)
	v["item_id"] = "UnknownItem"
	_, e = invoke(t, db, "synthetic_webhook", `SELECT integration.accept_verified_webhook('sandbox',$1,$2)::text`, strings.Repeat("a", 64), raw(v))
	requireCode(t, e, "P2801")
	initial := status(t, db, id)
	request := mutation(initial, "request")
	operate(t, db, id, "request", request)
	exec(t, c, `UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE source_account_id=$1 AND principal_id=$2 AND capability='bind'`, initial["source_account_id"], actor)
	_, e = invoke(t, db, "synthetic_runtime", `SELECT integration.mutate_connection_stream($1,$2,$3,$4,'request',$5)::text`, actor, entity, book, id, request)
	requireCode(t, e, "P2701")
	_, e = invoke(t, db, "synthetic_runtime", `SELECT integration.connection_stream_status($1,$2,$3,$4)::text`, actor, entity, "00000000-0000-4000-8000-000000000099", id)
	requireCode(t, e, "P2701")
	if value(t, c, `SELECT count(*)::text FROM integration.webhook_receipt`) != "0" || value(t, c, `SELECT count(*)::text FROM integration.sync_wakeup`) != "1" {
		t.Fatal("failed request mutated intent")
	}
}

func TestConnectionOpsExpiredBindingVersusUnknownClaim(t *testing.T) {
	c, db := setup(t)
	defer c.Close(context.Background())
	a := claim(t, db, link(t, db, begin(t, db, "new", "new", nil, nil, nil)))
	a = recordManifest(t, db, a, verified("SyntheticItem", "AccountA"))
	a = confirm(t, db, a)
	exec(t, c, `UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1`, a["attempt_id"])
	listing := object(t, call(t, db, "synthetic_runtime", `SELECT integration.list_connection_attempts($1,$2,$3,100,NULL)::text`, actor, entity, book))
	old := listing["items"].([]any)[0].(map[string]any)
	if listing["unfinished_count"] != "0" || old["binding_expired"] != true || old["expired_unclaimed"] != false {
		t.Fatal("known published expired binding blocks fresh start")
	}
	fresh := begin(t, db, "fresh", "new", nil, nil, nil)
	fresh = link(t, db, fresh)
	fresh = object(t, call(t, db, "synthetic_runtime", `SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text`, actor, entity, book, fresh["attempt_id"], fresh["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "public_token_sha256": strings.Repeat("e", 64)})))
	exec(t, c, `UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1`, fresh["attempt_id"])
	listing = object(t, call(t, db, "synthetic_runtime", `SELECT integration.list_connection_attempts($1,$2,$3,100,NULL)::text`, actor, entity, book))
	if listing["unfinished_count"] != "1" {
		t.Fatal("unknown claimed expiry escaped recovery")
	}
	_, e := invoke(t, db, "synthetic_runtime", `SELECT integration.begin_connection($1,$2,$3,'unsafe',$4)::text`, actor, entity, book, raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}}))
	requireCode(t, e, "P2804")
}

func TestConnectionOpsConcurrentFreshBegin(t *testing.T) {
	c, db := setup(t)
	defer c.Close(context.Background())
	errs := make(chan error, 2)
	start := make(chan struct{})
	body := raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}})
	for _, key := range []string{"a", "b"} {
		go func(k string) {
			<-start
			_, e := invoke(t, db, "synthetic_runtime", `SELECT integration.begin_connection($1,$2,$3,$4,$5)::text`, actor, entity, book, k, body)
			errs <- e
		}(key)
	}
	close(start)
	ok, blocked := 0, 0
	for range 2 {
		e := <-errs
		if e == nil {
			ok++
		} else {
			requireCode(t, e, "P2804")
			blocked++
		}
	}
	if ok != 1 || blocked != 1 || value(t, c, `SELECT count(*)::text FROM integration.connection_attempt`) != "1" {
		t.Fatal("concurrent enrollment guard")
	}
}
