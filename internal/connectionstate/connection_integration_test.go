//go:build integration

package connectionstate

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

var proof = flag.Bool("connection-proof", false, "Use only the disposable internal registry database")

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
	db := "connection_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE connection_template")
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

func TestConnectionOneShotAndRecovery(t *testing.T) {
	c, db := setup(t)
	a := begin(t, db, "new", "new", nil, nil, nil)
	if a["execute_link"] != true {
		t.Fatal("first Link")
	}
	if begin(t, db, "new", "new", nil, nil, nil)["execute_link"] != false {
		t.Fatal("replayed Link")
	}
	a = link(t, db, a)
	x := claim(t, db, a)
	if x["execute_provider"] != true {
		t.Fatal("first claim")
	}
	if claim(t, db, a)["execute_provider"] != false {
		t.Fatal("duplicate exchange authorized")
	}
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, x["attempt_id"], x["claim_id"])
	requireCode(t, e, "P2704")
	call(t, db, "synthetic_runtime", "SELECT integration.connection_failure($1,$2,$3,$4,$5,'provider_unknown')::text", actor, entity, book, x["attempt_id"], x["claim_id"])
	if claim(t, db, a)["execute_provider"] != false {
		t.Fatal("uncertain exchange retried")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.sync_item") != "0" {
		t.Fatal("unverified Item active")
	}
	x = recordManifest(t, db, x, verified("SyntheticItem", "AccountA"))
	call(t, db, "synthetic_runtime", "SELECT integration.connection_failure($1,$2,$3,$4,$5,'secret_unknown')::text", actor, entity, book, x["attempt_id"], x["claim_id"])
	x = confirm(t, db, x)
	if x["state"] != "binding_review" {
		t.Fatal("secret recovery stage")
	}
	out := bind(t, db, x, binding("AccountA", account))
	if out["state"] != "active" {
		t.Fatal("explicit binding")
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("connection posted")
	}
}

func TestConnectionProviderFirstAndOverlap(t *testing.T) {
	c, db, a, out := connected(t)
	if len(out["streams"].([]any)) != 2 || value(t, c, "SELECT count(*)::text FROM banking.source_namespace WHERE overlap_review_required") != "0" {
		t.Fatal("provider-first source")
	}
	if value(t, c, "SELECT count(*)::text FROM banking.source_permission WHERE capability='review'") != "0" {
		t.Fatal("unsolicited review grant")
	}
	status := call(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	for _, key := range []string{"credential_ref", "nonce", "claim_id", "sha256"} {
		if strings.Contains(status, key) {
			t.Fatal("private status leak")
		}
	}
	if len(object(t, call(t, db, "synthetic_runtime", "SELECT integration.list_connections($1,$2,$3)::text", actor, entity, book))["items"].([]any)) != 1 {
		t.Fatal("list")
	}
	// A distinct verified Item explicitly mapped to the existing source must hold overlap.
	b := begin(t, db, "another", "new", nil, nil, nil)
	b = link(t, db, b)
	b = object(t, call(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, b["attempt_id"], b["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "public_token_sha256": strings.Repeat("c", 64)})))
	b = confirm(t, db, recordManifest(t, db, b, verified("AnotherItem", "AnotherAccount")))
	r := binding("AnotherAccount", account)
	first := out["streams"].([]any)[0].(map[string]any)
	sid := value(t, c, "SELECT source_account_id::text FROM banking.source_account WHERE ledger_account_id=$1", account)
	r["source_account_id"] = sid
	r["expected_binding_revision"] = "1"
	_ = first
	other := object(t, call(t, db, "synthetic_runtime", "SELECT integration.bind_connection($1,$2,$3,$4,'overlap',$5)::text", actor, entity, book, b["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": []any{r}})))
	if other["streams"].([]any)[0].(map[string]any)["overlap_review_required"] != true {
		t.Fatal("overlap waived")
	}
}

func TestConnectionRotationAndDisconnect(t *testing.T) {
	c, db, a, out := connected(t)
	streams := out["streams"].([]any)
	one := streams[0].(map[string]any)
	two := streams[1].(map[string]any)
	x := object(t, call(t, db, "synthetic_sync", "SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000082')::text", entity, book, one["stream_id"]))
	y := object(t, call(t, db, "synthetic_sync", "SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000083')::text", entity, book, two["stream_id"]))
	call(t, db, "synthetic_runtime", "SELECT integration.disconnect_stream($1,$2,$3,$4,'one',1)::text", actor, entity, book, one["stream_id"])
	_, e := invoke(t, db, "synthetic_sync", "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, one["stream_id"], x["set_id"], x["fence"])
	requireCode(t, e, "P2601")
	call(t, db, "synthetic_sync", "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, two["stream_id"], y["set_id"], y["fence"])
	rotate := claim(t, db, itemIntent(t, c, db, "rotate", "rotate", a["sync_item_id"]))
	_, e = invoke(t, db, "synthetic_sync", "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, two["stream_id"], y["set_id"], y["fence"])
	requireCode(t, e, "P2601")
	if rotate["credential_ref"] == rotate["previous_credential_ref"] || rotate["credential_version"] != "2" {
		t.Fatal("rotation reference")
	}
	rotate = confirm(t, db, recordManifest(t, db, rotate, verified("SyntheticItem", "AccountA", "AccountB")))
	if rotate["state"] != "active" || value(t, c, "SELECT count(*)::text FROM integration.sync_stream WHERE state='disconnected'") != "1" {
		t.Fatal("rotation resumed disconnected account")
	}
	_, e = invoke(t, db, "synthetic_sync", "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, two["stream_id"], y["set_id"], y["fence"])
	requireCode(t, e, "P2603")
}

// Failed removal acknowledgement keeps every sibling fenced until explicit confirmation.
func TestConnectionRemovalUncertainty(t *testing.T) {
	c, db, a, _ := connected(t)
	remove := claim(t, db, itemIntent(t, c, db, "remove", "remove", a["sync_item_id"]))
	if remove["state"] != "disconnect_pending" {
		t.Fatal("remove reservation")
	}
	call(t, db, "synthetic_runtime", "SELECT integration.connection_failure($1,$2,$3,$4,$5,'provider_unknown')::text", actor, entity, book, remove["attempt_id"], remove["claim_id"])
	if claim(t, db, remove)["execute_provider"] != false {
		t.Fatal("blind remote remove retry")
	}
	if value(t, c, "SELECT state FROM integration.sync_item") != "disconnect_pending" || value(t, c, "SELECT count(*)::text FROM integration.sync_stream WHERE state='active'") != "2" {
		t.Fatal("unknown removal falsely confirmed")
	}
	v := call(t, db, "synthetic_runtime", "SELECT integration.confirm_item_removed($1,$2,$3,$4,$5)::text", actor, entity, book, remove["attempt_id"], remove["claim_id"])
	if call(t, db, "synthetic_runtime", "SELECT integration.confirm_item_removed($1,$2,$3,$4,$5)::text", actor, entity, book, remove["attempt_id"], remove["claim_id"]) != v {
		t.Fatal("removal replay")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream WHERE state='disconnected'") != "2" {
		t.Fatal("siblings not disconnected")
	}
}

// Update mode preserves source identities and rejects replacement Items or missing bound accounts.
func TestConnectionReconnectIdentity(t *testing.T) {
	c, db, a, _ := connected(t)
	before := value(t, c, "SELECT jsonb_agg(jsonb_build_array(namespace_id,source_account_id,account_id) ORDER BY stream_id)::text FROM integration.sync_stream")
	repair := claim(t, db, link(t, db, itemIntent(t, c, db, "repair", "reconnect", a["sync_item_id"])))
	for _, m := range []map[string]any{verified("DifferentItem", "AccountA", "AccountB"), verified("SyntheticItem", "AccountA")} {
		_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, repair["attempt_id"], repair["claim_id"], raw(m))
		requireCode(t, e, "P2704")
	}
	repair = confirm(t, db, recordManifest(t, db, repair, verified("SyntheticItem", "AccountA", "AccountB")))
	if repair["credential_ref"] != repair["previous_credential_ref"] || repair["credential_version"] != "1" || value(t, c, "SELECT jsonb_agg(jsonb_build_array(namespace_id,source_account_id,account_id) ORDER BY stream_id)::text FROM integration.sync_stream") != before {
		t.Fatal("reconnect changed identity")
	}
}

// Lost first-claim application reply and concurrent replays never produce another execution grant.
func TestConnectionClaimCommitRecovery(t *testing.T) {
	c, db := setup(t)
	a := link(t, db, begin(t, db, "new", "new", nil, nil, nil))
	first := claim(t, db, a)
	_ = first // Model a committed response lost before the orchestrator can begin provider I/O.
	type result struct {
		v string
		e error
	}
	ch := make(chan result, 2)
	start := make(chan struct{})
	body := raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "public_token_sha256": strings.Repeat("b", 64)})
	for n := 0; n < 2; n++ {
		go func() {
			<-start
			v, e := invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], body)
			ch <- result{v, e}
		}()
	}
	close(start)
	for n := 0; n < 2; n++ {
		r := <-ch
		if r.e != nil {
			t.Fatal(r.e)
		}
		if object(t, r.v)["execute_provider"] != false {
			t.Fatal("ambiguous claim renewed")
		}
	}
	if value(t, c, "SELECT count(*)::text FROM integration.connection_event WHERE kind='claim'") != "1" {
		t.Fatal("duplicate reservation")
	}
	changed := raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "public_token_sha256": strings.Repeat("c", 64)})
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], changed)
	requireCode(t, e, "P2702")
}

// Every mutation/read rechecks actor/service scope, including replay of a successful connection.
func TestConnectionRevocationAndScope(t *testing.T) {
	c, db, a, out := connected(t)
	for _, permission := range []string{"read", "import", "connect"} {
		exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission=$2", actor, permission)
		_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
		requireCode(t, e, "P2701")
		exec(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id=$1 AND permission=$2", actor, permission)
	}
	stream := out["streams"].([]any)[0].(map[string]any)
	exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND source_account_id=$2 AND capability='bind'", actor, stream["source_account_id"])
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "P2701")
	exec(t, c, "UPDATE banking.source_permission SET revoked_at=NULL WHERE principal_id=$1 AND capability='bind'", actor)
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, other, book, a["attempt_id"])
	requireCode(t, e, "P2701")
	exec(t, c, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", reader)
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "P2701")
}

// Unknown fields, unsupported accounts and stale pins cannot turn browser metadata into bindings.
func TestConnectionManifestAndBindingGates(t *testing.T) {
	c, db := setup(t)
	a := claim(t, db, link(t, db, begin(t, db, "new", "new", nil, nil, nil)))
	m := verified("SyntheticItem", "AccountA")
	m["access_token"] = "fictional-disallowed-field"
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["claim_id"], raw(m))
	requireCode(t, e, "P2700")
	m = verified("SyntheticItem", "AccountA", "AccountA")
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["claim_id"], raw(m))
	requireCode(t, e, "P2700")
	a = confirm(t, db, recordManifest(t, db, a, verified("SyntheticItem", "AccountA")))
	for _, kind := range []string{"unverified", "liability", "currency", "grants", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			r := binding("AccountA", account)
			body := map[string]any{"consent_grants": true, "bindings": []any{r}}
			code := "P2704"
			switch kind {
			case "unverified":
				r["account_id"] = "AccountB"
			case "liability":
				r["ledger_account_id"] = "00000000-0000-4000-8000-000000000012"
			case "currency":
				r["currency_version_id"] = "00000000-0000-4000-8000-000000000006"
			case "grants":
				body["consent_grants"] = false
				code = "P2700"
			case "duplicate":
				body["bindings"] = []any{r, r}
				code = "P2700"
			}
			_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.bind_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], kind, raw(body))
			requireCode(t, e, code)
		})
	}
	if value(t, c, "SELECT count(*)::text FROM banking.source_account") != "0" {
		t.Fatal("invalid selection partially published")
	}
}

// Session role boundaries forbid secret-reference reads outside a current fenced acquisition.
func TestConnectionWorkerContextAndPrivileges(t *testing.T) {
	c, db, _, out := connected(t)
	stream := out["streams"].([]any)[0].(map[string]any)
	key := "00000000-0000-4000-8000-000000000082"
	absent := object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_resume($1,$2,$3,$4)::text", entity, book, stream["stream_id"], key))
	if absent["state"] != "absent" {
		t.Fatal("resume absent")
	}
	x := object(t, call(t, db, "synthetic_sync", "SELECT integration.claim_sync($1,$2,$3,$4)::text", entity, book, stream["stream_id"], key))
	ctx := object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, stream["stream_id"], x["set_id"], x["fence"]))
	if ctx["credential_ref"] == nil || ctx["credential_version"] != "1" || ctx["account_id"] == nil {
		t.Fatal("scoped context")
	}
	worker := connect(t, db, "synthetic_sync")
	defer worker.Close(context.Background())
	for _, q := range []string{"SELECT * FROM integration.connection_attempt", "SELECT integration.connection_context(NULL,NULL,NULL,NULL)", "SELECT integration.confirm_connection_credential(NULL,NULL,NULL,NULL,NULL)"} {
		_, e := worker.Exec(context.Background(), q)
		requireCode(t, e, "42501")
	}
	api := connect(t, db, "synthetic_runtime")
	defer api.Close(context.Background())
	for _, q := range []string{"SELECT * FROM integration.credential_publication", "SELECT integration.set_connection_worker(NULL,NULL,NULL,'x','{}')", "SELECT integration.sync_context(NULL,NULL,NULL,NULL,NULL)"} {
		_, e := api.Exec(context.Background(), q)
		requireCode(t, e, "42501")
	}
	_, e := c.Exec(context.Background(), "UPDATE integration.connection_attempt SET claim_id=gen_random_uuid()")
	requireCode(t, e, "P2702")
}

// Expired handoffs and invalid UTC timestamps cannot gain a provider execution claim.
func TestConnectionExpiryAndAtomicBinding(t *testing.T) {
	c, db := setup(t)
	a := begin(t, db, "old", "new", nil, nil, nil)
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "expires_at": "2099-01-01T24:00:00Z"}))
	requireCode(t, e, "P2700")
	a = link(t, db, a)
	exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "public_token_sha256": strings.Repeat("b", 64)}))
	requireCode(t, e, "P2703")
	a = claim(t, db, link(t, db, begin(t, db, "fresh", "new", nil, nil, nil)))
	a = confirm(t, db, recordManifest(t, db, a, verified("SyntheticItem", "AccountA", "AccountB")))
	good := binding("AccountA", account)
	bad := binding("AccountB", "00000000-0000-4000-8000-000000000099")
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.bind_connection($1,$2,$3,$4,'atomic',$5)::text", actor, entity, book, a["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": []any{good, bad}}))
	requireCode(t, e, "P2704")
	if value(t, c, "SELECT count(*)::text FROM banking.source_account") != "0" || value(t, c, "SELECT count(*)::text FROM banking.source_permission") != "0" {
		t.Fatal("partial binding grant escaped")
	}
}

// Scalar/null containers fail with fixed diagnostics rather than leaking PostgreSQL values.
func TestConnectionMalformedContainers(t *testing.T) {
	_, db := setup(t)
	a := claim(t, db, link(t, db, begin(t, db, "new", "new", nil, nil, nil)))
	for _, bad := range []any{nil, map[string]any{}, "bad", []any{nil}, []any{"bad"}} {
		m := verified("SyntheticItem", "AccountA")
		m["accounts"] = bad
		_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["claim_id"], raw(m))
		requireCode(t, e, "P2700")
	}
	a = confirm(t, db, recordManifest(t, db, a, verified("SyntheticItem", "AccountA")))
	for _, bad := range []any{nil, map[string]any{}, "bad", []any{nil}, []any{"bad"}} {
		_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.bind_connection($1,$2,$3,$4,'malformed',$5)::text", actor, entity, book, a["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": bad}))
		requireCode(t, e, "P2700")
	}
}

// A competing lifecycle reservation invalidates private I/O metadata, not authorized public history.
func TestConnectionPrivateContextFenceAndExclusiveClaim(t *testing.T) {
	c, db, a, _ := connected(t)
	// Expiry is prepared before unresolved lifecycle claims; the fresh-enrollment guard stays intact.
	fresh := begin(t, db, "new-expired", "new", nil, nil, nil)
	exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", fresh["attempt_id"])
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, fresh["attempt_id"])
	requireCode(t, e, "P2703")
	call(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, fresh["attempt_id"])
	first := itemIntent(t, c, db, "rotate-first", "rotate", a["sync_item_id"])
	second := itemIntent(t, c, db, "remove-second", "remove", a["sync_item_id"])
	first = claim(t, db, first)
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, second["attempt_id"])
	requireCode(t, e, "P2703")
	call(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, second["attempt_id"])
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, second["attempt_id"], second["nonce"], raw(map[string]any{"link_token_sha256": nil, "public_token_sha256": nil}))
	requireCode(t, e, "P2703")
	var ids []any
	if e = json.Unmarshal([]byte(value(t, c, "SELECT jsonb_agg(stream_id ORDER BY stream_id)::text FROM integration.sync_stream")), &ids); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"reconnect", "rotate", "remove"} {
		_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.begin_connection($1,$2,$3,$4,$5)::text", actor, entity, book, "blocked-"+mode, raw(map[string]any{"mode": mode, "sync_item_id": a["sync_item_id"], "expected_generation": first["item_generation"], "approved_stream_ids": ids}))
		requireCode(t, e, "P2703")
	}
	// A separately approved owner transition also prevents the first worker using its old reference.
	call(t, db, "synthetic_operator", "SELECT integration.set_sync_item_state($1,$2,$3,$4,'owner-pause',$5)::text", actor, entity, book, a["sync_item_id"], raw(map[string]any{"expected_generation": first["item_generation"], "state": "paused"}))
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, first["attempt_id"])
	requireCode(t, e, "P2703")
	call(t, db, "synthetic_runtime", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, first["attempt_id"])

}
