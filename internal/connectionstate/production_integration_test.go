//go:build integration

package connectionstate

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

const productionAPI = "synthetic_production_api"
const productionWorker = "synthetic_production_sync"
const productionKey = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/bank"
const productionLinkKey = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/link"

// productionScope is an operator's fictional immutable resource/login selection, never real credentials.
func productionScope(enabled bool) map[string]any {
	return map[string]any{"environment": "production", "api_login": productionAPI, "worker_login": productionWorker, "bank_key": productionKey, "link_key": productionLinkKey, "enabled": enabled}
}

// productionLogins creates fictional restricted identities in the disposable cluster only.
func productionLogins(t testing.TB, c *pgx.Conn) {
	t.Helper()
	exec(t, c, `DO $$ BEGIN
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='synthetic_production_api') THEN CREATE ROLE synthetic_production_api LOGIN IN ROLE bookkeeping_api;END IF;
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='synthetic_production_sync') THEN CREATE ROLE synthetic_production_sync LOGIN IN ROLE bookkeeping_sync;END IF;
END $$`)
}

// productionSetup uses real restricted logins and configures a fresh book before any provider history.
func productionSetup(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	c, db := setup(t)
	productionLogins(t, c)
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'production-scope',$4)::text", actor, entity, book, raw(productionScope(true)))
	call(t, db, "synthetic_operator", "SELECT integration.set_connection_worker($1,$2,$3,'production-worker',$4)::text", actor, entity, book, raw(map[string]any{"service_principal_id": reader, "worker_login": productionWorker, "enabled": true}))
	return c, db
}

// productionBegin pins new intent through the configured API login; no environment is accepted in JSON.
func productionBegin(t testing.TB, db string) map[string]any {
	t.Helper()
	return object(t, call(t, db, productionAPI, "SELECT integration.begin_connection($1,$2,$3,'production-new',$4)::text", actor, entity, book, raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}})))
}

// productionManifest uses fingerprints and fictional identities only, with no actual provider exchange.
func productionManifest(t testing.TB, db string) map[string]any {
	t.Helper()
	return productionManifestFor(t, db, "production-new", "FictionalItem", "AccountA")
}

// productionManifestFor gives each fictional enrollment distinct fingerprints without constructing a real token.
func productionManifestFor(t testing.TB, db, key, item string, accounts ...string) map[string]any {
	t.Helper()
	a := object(t, call(t, db, productionAPI, "SELECT integration.begin_connection($1,$2,$3,$4,$5)::text", actor, entity, book, key, raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}})))
	lh := fmt.Sprintf("%x", sha256.Sum256([]byte("fictional-link:"+a["attempt_id"].(string))))
	ph := fmt.Sprintf("%x", sha256.Sum256([]byte("fictional-public:"+a["attempt_id"].(string))))
	a = object(t, call(t, db, productionAPI, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": lh, "expires_at": "2099-01-01T00:00:00Z"})))
	a = object(t, call(t, db, productionAPI, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": lh, "public_token_sha256": ph})))
	return object(t, call(t, db, productionAPI, "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["claim_id"], raw(verified(item, accounts...))))
}

// productionConnected acknowledges a fictional credential and binds one explicitly reviewed checking account.
func productionConnected(t *testing.T) (*pgx.Conn, string, map[string]any, map[string]any) {
	t.Helper()
	c, db := productionSetup(t)
	a := productionManifest(t, db)
	a = object(t, call(t, db, productionAPI, "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], a["claim_id"]))
	out := object(t, call(t, db, productionAPI, "SELECT integration.bind_connection($1,$2,$3,$4,'production-bind',$5)::text", actor, entity, book, a["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": []any{binding("AccountA", account)}})))
	return c, db, a, out["streams"].([]any)[0].(map[string]any)
}

// productionPage has independently fixed USD50 expectations, not generated from database output.
func productionPage() string {
	return raw(map[string]any{"page_index": 0, "request_cursor": nil, "next_cursor": "C1", "has_more": false, "transactions_update_status": "HISTORICAL_UPDATE_COMPLETE", "request_id": "FictionalRequest", "sha256": strings.Repeat("c", 64), "modified": []any{}, "removed": []any{}, "added": []any{map[string]any{
		"account_id": "AccountA", "transaction_id": "TxnA", "pending": false, "pending_transaction_id": nil, "date": "2026-01-02", "authorized_date": nil, "datetime": nil, "authorized_datetime": nil, "iso_currency_code": "USD", "currency_version_id": currency, "amount_lexeme": "50.00", "amount": "50", "direction": "outflow", "eligible_for_review": true, "name": "Fictional expense", "merchant_name": nil, "original_description": nil,
	}}})
}

// publishProduction exercises actual lease/context/stage/complete SQL and exact committed replay.
func publishProduction(t testing.TB, db string, stream map[string]any) map[string]any {
	t.Helper()
	x := object(t, call(t, db, productionWorker, "SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000082')::text", entity, book, stream["stream_id"]))
	c := object(t, call(t, db, productionWorker, "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, stream["stream_id"], x["set_id"], x["fence"]))
	if c["environment"] != "production" || c["account_id"] != "AccountA" {
		t.Fatal("trusted Production execution context missing")
	}
	call(t, db, productionWorker, "SELECT integration.stage_sync_page($1,$2,$3,$4,$5,$6)::text", entity, book, stream["stream_id"], x["set_id"], x["fence"], productionPage())
	q := "SELECT integration.complete_sync($1,$2,$3,$4,$5)::text"
	first := call(t, db, productionWorker, q, entity, book, stream["stream_id"], x["set_id"], x["fence"])
	if again := call(t, db, productionWorker, q, entity, book, stream["stream_id"], x["set_id"], x["fence"]); first != again {
		t.Fatal("completed acquisition replay changed receipt")
	}
	return x
}

// TestProductionRecordsNotSynthetic proves new origin, exact USD units and zero acquisition journals.
func TestProductionRecordsNotSynthetic(t *testing.T) {
	c, db, a, stream := productionConnected(t)
	if a["environment"] != "production" || a["data_origin"] != "provider" {
		t.Fatal("private attempt origin missing")
	}
	publishProduction(t, db, stream)
	for q, want := range map[string]string{
		"SELECT environment||':'||synthetic::text FROM integration.sync_item":                     "production:false",
		"SELECT environment FROM banking.source_namespace":                                        "production",
		"SELECT count(*)::text FROM banking.source_observation":                                   "1",
		"SELECT provider_amount_lexeme FROM banking.observation_revision":                         "50.00",
		"SELECT trim_scale(amount)::text FROM banking.observation_revision":                       "50",
		"SELECT count(*)::text FROM banking.review_item":                                          "1",
		"SELECT count(*)::text FROM ledger.journal_entry":                                         "0",
		"SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'": "0",
	} {
		if got := value(t, c, q); got != want {
			t.Fatalf("fixed Production source expectation: got %s want %s", got, want)
		}
	}
	// The public attempt projection does not grow an environment selector.
	public := object(t, call(t, db, productionAPI, "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]))
	if _, ok := public["environment"]; ok {
		t.Fatal("internal routing leaked into public projection")
	}
	down, err := fs.ReadFile(migrations.Files, "000034_production_source_authority.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Exec(context.Background(), string(down))
	requireCode(t, err, "P0001")
	exec(t, c, "ROLLBACK")
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "1" {
		t.Fatal("rollback erased retained source")
	}
}

// TestProductionProfileCrossEnvironmentDeny proves actual login/tenant/profile denials before work.
func TestProductionProfileCrossEnvironmentDeny(t *testing.T) {
	c, db := productionSetup(t)
	intent := raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}})
	_, err := invoke(t, db, "synthetic_runtime", "SELECT integration.begin_connection($1,$2,$3,'wrong-runtime',$4)::text", actor, entity, book, intent)
	requireCode(t, err, "P2701")
	_, err = invoke(t, db, productionAPI, "SELECT integration.begin_connection($1,$2,$3,'wrong-tenant',$4)::text", actor, other, book, intent)
	requireCode(t, err, "P2701")
	for _, env := range []any{"sandbox", "production", nil} {
		injected := raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}, "environment": env})
		_, err = invoke(t, db, productionAPI, "SELECT integration.begin_connection($1,$2,$3,'injected',$4)::text", actor, entity, book, injected)
		requireCode(t, err, "P2700")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.connection_attempt") != "0" {
		t.Fatal("denied request created work")
	}
	ctx := object(t, call(t, db, productionAPI, "SELECT integration.connection_provider_context($1,$2,$3)::text", actor, entity, book))
	if ctx["environment"] != "production" || ctx["scope_revision"] != "1" || ctx["bank_key"] != productionKey || ctx["link_key"] != productionLinkKey {
		t.Fatal("313 preflight lost exact scope")
	}
	// A registered Production login must not fall back on another unconfigured book.
	otherBook := "00000000-0000-4000-8000-000000000103"
	exec(t, c, "INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Fictional other book')", entity, otherBook)
	call(t, db, "synthetic_operator", "SELECT integration.set_connection_worker($1,$2,$3,'other-worker',$4)::text", actor, entity, otherBook, raw(map[string]any{"service_principal_id": reader, "worker_login": "synthetic_sync", "enabled": true}))
	_, err = invoke(t, db, productionAPI, "SELECT integration.begin_connection($1,$2,$3,'missing-scope',$4)::text", actor, entity, otherBook, intent)
	requireCode(t, err, "P2701")
}

// TestProductionScopeRevocationAndReplay rechecks current scope, source consent and permissions on retries.
func TestProductionScopeRevocationAndReplay(t *testing.T) {
	c, db, a, stream := productionConnected(t)
	x := object(t, call(t, db, productionWorker, "SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000082')::text", entity, book, stream["stream_id"]))
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'revoke-scope',$4)::text", actor, entity, book, raw(productionScope(false)))
	_, err := invoke(t, db, productionAPI, "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, err, "P2701")
	_, err = invoke(t, db, productionWorker, "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, stream["stream_id"], x["set_id"], x["fence"])
	requireCode(t, err, "P2601")
	// Replaying the old enable command returns its old receipt without re-enabling the scope.
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'production-scope',$4)::text", actor, entity, book, raw(productionScope(true)))
	if value(t, c, "SELECT enabled::text FROM integration.provider_scope") != "false" {
		t.Fatal("old receipt restored revoked authority")
	}
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'enable-again',$4)::text", actor, entity, book, raw(productionScope(true)))
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='connect'", actor)
	_, err = invoke(t, db, productionAPI, "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, err, "P2701")
	exec(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id=$1 AND permission='connect'", actor)
	exec(t, c, "UPDATE integration.sync_item SET consent_expires_at=clock_timestamp()-interval '1 second',generation=generation+1")
	_, err = invoke(t, db, productionWorker, "SELECT integration.sync_context($1,$2,$3,$4,$5)::text", entity, book, stream["stream_id"], x["set_id"], x["fence"])
	requireCode(t, err, "P2601")
}

// testEnvelope is structurally valid fictional ciphertext; crypto/AAD qualification remains #313.
func testEnvelope(key string) string {
	return raw(map[string]any{"profile": "jayburd-bank-aead-v1", "key_version": key + "/cryptoKeyVersions/1", "wrapped_key": base64.StdEncoding.EncodeToString([]byte("fictional-wrapped")), "nonce": base64.StdEncoding.EncodeToString(make([]byte, 12)), "ciphertext": base64.StdEncoding.EncodeToString(make([]byte, 32))})
}

// TestProductionCredentialSQLBinding tests addressed SQL and key pins without claiming successful decryption.
func TestProductionCredentialSQLBinding(t *testing.T) {
	c, db := productionSetup(t)
	a := productionManifest(t, db)
	q := "SELECT integration.put_bank_credential($1,$2,$3,$4,$5,'FictionalItem',$6,$7)::text"
	for _, bad := range []struct{ env, key string }{{"sandbox", productionKey}, {"production", productionLinkKey}, {"production", "projects/fictional-production/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials"}} {
		_, err := invoke(t, db, productionAPI, q, actor, entity, book, a["credential_ref"], a["credential_version"], bad.env, testEnvelope(bad.key))
		if bad.env == "sandbox" {
			requireCode(t, err, "P2701")
		} else {
			requireCode(t, err, "P2700")
		}
	}
	first := call(t, db, productionAPI, q, actor, entity, book, a["credential_ref"], a["credential_version"], "production", testEnvelope(productionKey))
	if again := call(t, db, productionAPI, q, actor, entity, book, a["credential_ref"], a["credential_version"], "production", testEnvelope(productionKey)); again != first {
		t.Fatal("immutable ciphertext winner changed")
	}
	_, err := invoke(t, db, productionAPI, "SELECT integration.get_bank_credential($1,$2,$3,$4,$5,'ForeignItem','production')::text", actor, entity, book, a["credential_ref"], a["credential_version"])
	requireCode(t, err, "P2701")
	if value(t, c, "SELECT environment FROM integration.bank_credential") != "production" {
		t.Fatal("credential origin changed")
	}
	a = object(t, call(t, db, productionAPI, "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], a["claim_id"]))
	out := object(t, call(t, db, productionAPI, "SELECT integration.bind_connection($1,$2,$3,$4,'credential-bind',$5)::text", actor, entity, book, a["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": []any{binding("AccountA", account)}})))
	stream := out["streams"].([]any)[0].(map[string]any)
	x := object(t, call(t, db, productionWorker, "SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000082')::text", entity, book, stream["stream_id"]))
	w := object(t, call(t, db, productionWorker, "SELECT integration.worker_bank_credential($1,$2,$3,$4,$5)::text", entity, book, stream["stream_id"], x["set_id"], x["fence"]))
	if w["environment"] != "production" || w["credential_ref"] != a["credential_ref"] || w["envelope"] == nil {
		t.Fatal("worker credential lost addressed Production binding")
	}
}

// TestProductionHostedHandoffSQLBinding selects the approved Link key without claiming actual AEAD/KMS execution.
func TestProductionHostedHandoffSQLBinding(t *testing.T) {
	_, db := productionSetup(t)
	a := productionBegin(t, db)
	a = object(t, call(t, db, productionAPI, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("d", 64), "expires_at": "2099-01-01T00:00:00Z"})))
	linkEnvelope := func(key string) string {
		return strings.Replace(testEnvelope(key), "jayburd-bank-aead-v1", "jayburd-hosted-link-aead-v1", 1)
	}
	q := "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text"
	_, err := invoke(t, db, productionAPI, q, actor, entity, book, a["attempt_id"], linkEnvelope(productionKey))
	requireCode(t, err, "P2700")
	first := call(t, db, productionAPI, q, actor, entity, book, a["attempt_id"], linkEnvelope(productionLinkKey))
	if again := call(t, db, productionAPI, q, actor, entity, book, a["attempt_id"], linkEnvelope(productionLinkKey)); again != first {
		t.Fatal("immutable Production Link handoff changed")
	}
	got := object(t, call(t, db, productionAPI, "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]))
	if got["envelope"] == nil {
		t.Fatal("current authorized Link handoff missing")
	}
}

// TestProductionScopePrivileges proves no runtime can administer or read raw scope/credential tables.
func TestProductionScopePrivileges(t *testing.T) {
	_, db := productionSetup(t)
	for _, login := range []string{productionAPI, productionWorker, "synthetic_runtime", "synthetic_sync"} {
		c := connect(t, db, login)
		for _, q := range []string{"SELECT * FROM integration.provider_scope", "SELECT * FROM integration.bank_credential", "SELECT integration.configure_provider_scope(NULL,NULL,NULL,'x','{}')", "SELECT integration.provider_environment(NULL,NULL)"} {
			_, err := c.Exec(context.Background(), q)
			requireCode(t, err, "42501")
		}
		c.Close(context.Background())
	}
}

// TestProductionCannotRelabelSandbox protects historical origin and rejects unknown/changed configuration.
func TestProductionCannotRelabelSandbox(t *testing.T) {
	c, db, a, _ := connected(t)
	productionLogins(t, c)
	_, err := invoke(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'relabel',$4)::text", actor, entity, book, raw(productionScope(true)))
	requireCode(t, err, "P2702")
	_, err = c.Exec(context.Background(), "UPDATE integration.connection_attempt SET environment='production' WHERE attempt_id=$1", a["attempt_id"])
	requireCode(t, err, "P2702")
	_, err = c.Exec(context.Background(), "UPDATE integration.sync_item SET environment='production',synthetic=false,generation=generation+1")
	requireCode(t, err, "P2602")
	if value(t, c, "SELECT environment||':'||synthetic::text FROM integration.sync_item") != "sandbox:true" || value(t, c, "SELECT count(*)::text FROM integration.provider_scope") != "0" {
		t.Fatal("Sandbox origin changed")
	}
}

// TestProductionScopeConfigurationDeny keeps operator metadata closed and existing scope identity immutable.
func TestProductionScopeConfigurationDeny(t *testing.T) {
	c, db := productionSetup(t)
	for _, field := range []string{"api_login", "worker_login", "bank_key", "environment"} {
		v := productionScope(true)
		switch field {
		case "api_login":
			v[field] = "synthetic_runtime"
		case "worker_login":
			v[field] = "synthetic_sync"
		case "bank_key":
			v[field] = productionLinkKey
		case "environment":
			v[field] = "sandbox"
			v["bank_key"] = nil
			v["link_key"] = nil
		}
		_, err := invoke(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,$4,$5)::text", actor, entity, book, "changed-"+field, raw(v))
		requireCode(t, err, "P2702")
	}
	v := productionScope(true)
	v["api_login"] = "missing_production_identity"
	_, err := invoke(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'missing-login',$4)::text", actor, entity, book, raw(v))
	requireCode(t, err, "P2701")
	if value(t, c, "SELECT revision::text FROM integration.provider_scope") != "1" {
		t.Fatal("denied configuration changed authority")
	}
}

// TestExplicitSandboxScope preserves deliberate Sandbox composition with absent Production key references.
func TestExplicitSandboxScope(t *testing.T) {
	_, db := setup(t)
	v := map[string]any{"environment": "sandbox", "api_login": "synthetic_runtime", "worker_login": "synthetic_sync", "bank_key": nil, "link_key": nil, "enabled": true}
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'explicit-sandbox',$4)::text", actor, entity, book, raw(v))
	ctx := object(t, call(t, db, "synthetic_runtime", "SELECT integration.connection_provider_context($1,$2,$3)::text", actor, entity, book))
	if ctx["environment"] != "sandbox" || ctx["data_origin"] != "test" || ctx["bank_key"] != nil {
		t.Fatal("explicit Sandbox scope changed provenance")
	}
	a := begin(t, db, "sandbox-new", "new", nil, nil, nil)
	if a["environment"] != "sandbox" {
		t.Fatal("explicit Sandbox attempt changed environment")
	}
}

// TestProductionSingleAccountPilotLimit rejects multi-select and a second Item without partial binding.
func TestProductionSingleAccountPilotLimit(t *testing.T) {
	c, db := productionSetup(t)
	a := productionManifestFor(t, db, "first", "FictionalFirst", "AccountA", "AccountB")
	a = object(t, call(t, db, productionAPI, "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], a["claim_id"]))
	q := "SELECT integration.bind_connection($1,$2,$3,$4,$5,$6)::text"
	_, err := invoke(t, db, productionAPI, q, actor, entity, book, a["attempt_id"], "multi", raw(map[string]any{"consent_grants": true, "bindings": []any{binding("AccountA", account), binding("AccountB", "00000000-0000-4000-8000-000000000015")}}))
	requireCode(t, err, "P2705")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "0" {
		t.Fatal("multi-select partially bound")
	}
	one := raw(map[string]any{"consent_grants": true, "bindings": []any{binding("AccountA", account)}})
	first := call(t, db, productionAPI, q, actor, entity, book, a["attempt_id"], "one", one)
	_, err = invoke(t, db, productionAPI, "SELECT integration.begin_connection($1,$2,$3,'second',$4)::text", actor, entity, book, raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}}))
	requireCode(t, err, "P2705")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "1" || value(t, c, "SELECT count(*)::text FROM integration.sync_item") != "1" || value(t, c, "SELECT count(*)::text FROM integration.connection_attempt") != "1" {
		t.Fatal("second source escaped pilot limit")
	}
	if again := call(t, db, productionAPI, q, actor, entity, book, a["attempt_id"], "one", one); again != first {
		t.Fatal("pilot limit denied committed bind recovery")
	}
}

// TestProductionPostingAuthorityAndReplay preserves authorized exact bookkeeping independently of bank fetch scope.
func TestProductionPostingAuthorityAndReplay(t *testing.T) {
	c, db, _, stream := productionConnected(t)
	publishProduction(t, db, stream)
	policy := "00000000-0000-4000-8000-000000000091"
	expense := "00000000-0000-4000-8000-000000000092"
	// Explicit fictional policy is a test oracle, never the default for a real book.
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())`, entity, book, policy, currency, actor)
	exec(t, c, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000093','2026-01-01','2026-02-01')`, entity, book)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'5000','Fictional expense','expense','debit',false,'2026-01-01',NULL)`, entity, book, expense)
	// Policy/chart creation legitimately advances the book revision; this is a concurrency pin, not a financial oracle.
	bookRevision := value(t, c, "SELECT revision::text FROM ledger.book")
	body := raw(map[string]any{"expected_revision": bookRevision, "policy_id": policy, "accounting_date": "2026-01-02", "description": "Fictional reviewed expense", "lines": []any{
		map[string]any{"account_id": expense, "side": "debit", "transaction_currency": "USD", "transaction_amount": "50", "functional_amount": "50", "valuation_basis": "identity"},
		map[string]any{"account_id": account, "side": "credit", "transaction_currency": "USD", "transaction_amount": "50", "functional_amount": "50", "valuation_basis": "identity"},
	}})
	q := "SELECT body::text FROM ledger.post_journal($1,$2,$3,'production-post','00000000-0000-4000-8000-000000000094',$4)"
	_, err := invoke(t, db, productionAPI, q, actor, entity, book, body)
	requireCode(t, err, "P2102")
	exec(t, c, "INSERT INTO ledger.membership VALUES($1,$2,'post',NULL)", entity, actor)
	first := call(t, db, productionAPI, q, actor, entity, book, body)
	if again := call(t, db, productionAPI, q, actor, entity, book, body); again != first {
		t.Fatal("journal replay changed result")
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" || value(t, c, "SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'") != "1" {
		t.Fatal("authorized post/replay duplicated effects")
	}
	if value(t, c, "SELECT trim_scale(sum(functional_amount))::text FROM ledger.journal_line WHERE side='debit'") != "50" || value(t, c, "SELECT trim_scale(sum(functional_amount))::text FROM ledger.journal_line WHERE side='credit'") != "50" {
		t.Fatal("independent USD50 balanced expectation changed")
	}
	_, err = invoke(t, db, productionAPI, strings.Replace(q, "'production-post'", "'stale-new-post'", 1), actor, entity, book, body)
	requireCode(t, err, "P2103")
	// Expiring provider authority does not erase current authorized local bookkeeping receipts.
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'pause-fetch',$4)::text", actor, entity, book, raw(productionScope(false)))
	if again := call(t, db, productionAPI, q, actor, entity, book, body); again != first {
		t.Fatal("source pause prohibited authorized local receipt recovery")
	}
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='post'", actor)
	_, err = invoke(t, db, productionAPI, q, actor, entity, book, body)
	requireCode(t, err, "P2102")
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
		t.Fatal("revoked posting changed ledger")
	}
}
