//go:build integration

package webhook

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const productionAPI = "synthetic_production_api"
const productionSync = "synthetic_production_sync"
const productionHook = "synthetic_production_webhook"
const productionBank = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/bank"
const productionLink = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/link"

// productionWrapper independently wraps only fictional key material; envelope AEAD and SQL are real.
type productionWrapper struct {
	key   string
	calls int
}

// Wrap binds the synthetic KEK to exact AAD and one canonical immutable version.
func (w *productionWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	w.calls++
	b, _ := aes.NewCipher(bytes.Repeat([]byte{23}, 32))
	g, _ := cipher.NewGCM(b)
	return w.key + "/cryptoKeyVersions/7", g.Seal(nil, make([]byte, 12), key, aad), nil
}

// Unwrap rejects any foreign key version before independent AEAD authentication.
func (w *productionWrapper) Unwrap(_ context.Context, version string, raw, aad []byte) ([]byte, error) {
	w.calls++
	if version != w.key+"/cryptoKeyVersions/7" {
		return nil, bankcredential.ErrUnavailable
	}
	b, _ := aes.NewCipher(bytes.Repeat([]byte{23}, 32))
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, make([]byte, 12), raw, aad)
}

// productionGraph retains only fictional scope and actual restricted adapters for one disposable fixture.
type productionGraph struct {
	admin, api       *pgx.Conn
	db, stream, item string
	attempt          map[string]any
	store            *bankcredential.APIStore
	bank, link       *productionWrapper
	reference        connections.CredentialReference
}

// productionCall invokes actual scoped functions using the separately provisioned Production API login.
func productionCall(t testing.TB, db, q string, args ...any) map[string]any {
	t.Helper()
	return pgObject(t, pgCall(t, db, productionAPI, q, append([]any{actor, entity, book}, args...)...))
}

// productionFixture enrolls and binds one original fictional account, persisting real AEAD before confirmation.
func productionFixture(t *testing.T) productionGraph {
	t.Helper()
	admin, db := pgFixture(t)
	exec(t, admin, `DO $$ BEGIN
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='synthetic_production_api') THEN CREATE ROLE synthetic_production_api LOGIN IN ROLE bookkeeping_api;END IF;
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='synthetic_production_sync') THEN CREATE ROLE synthetic_production_sync LOGIN IN ROLE bookkeeping_sync;END IF;
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='synthetic_production_webhook') THEN CREATE ROLE synthetic_production_webhook LOGIN IN ROLE bookkeeping_webhook;END IF;
END $$`)
	exec(t, admin, `INSERT INTO ledger.membership VALUES($1,$2,'import',NULL),($1,$3,'import',NULL)`, entity, actor, reader)
	callOwner(t, db, "SELECT integration.set_connect_permission($1,$2,$3,$1,'production-connect',true)::text", actor, entity, book)
	callOwner(t, db, "SELECT integration.configure_provider_scope($1,$2,$3,'production-scope',$4)::text", actor, entity, book, raw(map[string]any{"environment": "production", "api_login": productionAPI, "worker_login": productionSync, "bank_key": productionBank, "link_key": productionLink, "enabled": true}))
	callOwner(t, db, "SELECT integration.set_connection_worker($1,$2,$3,'production-worker',$4)::text", actor, entity, book, raw(map[string]any{"service_principal_id": reader, "worker_login": productionSync, "enabled": true}))
	a := productionCall(t, db, "SELECT integration.begin_connection($1,$2,$3,'production-new',$4)::text", raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}}))
	a = productionCall(t, db, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": tokenHash("production-new-link"), "expires_at": "2099-01-01T00:00:00Z"}))
	a = productionCall(t, db, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": tokenHash("production-new-public")}))
	a = productionCall(t, db, "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["claim_id"], raw(productionManifest()))
	api := connect(t, db, productionAPI)
	t.Cleanup(func() { api.Close(context.Background()) })
	bank, link := &productionWrapper{key: productionBank}, &productionWrapper{key: productionLink}
	store, e := bankcredential.NewProductionAPIStore(api, bank, link, bankcredential.ProductionPolicy{BankKey: productionBank, LinkKey: productionLink})
	if e != nil {
		t.Fatal(e)
	}
	ref := connections.CredentialReference{ReferenceID: a["credential_ref"].(string), Version: a["credential_version"].(string), ItemID: "synthetic-item", Environment: "production"}
	if e := store.ForScope(connections.Scope{Actor: actor, Entity: entity, Book: book}).Put(context.Background(), ref, connections.NewSecret("synthetic-production-access")); e != nil {
		t.Fatal("synthetic ciphertext setup failed", e)
	}
	a = productionCall(t, db, "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", a["attempt_id"], a["claim_id"])
	bound := productionCall(t, db, "SELECT integration.bind_connection($1,$2,$3,$4,'production-bind',$5)::text", a["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": []any{map[string]any{"account_id": "Account-A", "ledger_account_id": account, "currency_version_id": currency, "source_account_id": nil, "expected_binding_revision": nil, "binding_date": "2026-01-05", "binding_evidence_ref": csvNS}}}))
	stream := bound["streams"].([]any)[0].(map[string]any)["stream_id"].(string)
	callOwner(t, db, "SELECT integration.set_webhook_ingress('synthetic_production_webhook','production','production-ingress',true)::text")
	return productionGraph{admin: admin, api: api, db: db, stream: stream, item: value(t, admin, "SELECT sync_item_id::text FROM integration.sync_item"), attempt: a, store: store, bank: bank, link: link, reference: ref}
}

// productionManifest fixes identity/currency/consent independently of the receiver and posting code.
func productionManifest() map[string]any {
	return map[string]any{"item_id": "synthetic-item", "accounts": []any{map[string]any{"account_id": "Account-A", "type": "depository", "iso_currency_code": "USD"}}, "consent_expires_at": "2099-01-01T00:00:00Z"}
}

// signedProductionHTTP exercises the fixed key host, real signature/raw-body verification and restricted SQL receiver.
func signedProductionHTTP(t *testing.T, db, lost string) linkHTTP {
	t.Helper()
	conn := connect(t, db, productionHook)
	t.Cleanup(func() { conn.Close(context.Background()) })
	pool := &lossPool{conn: conn, lost: lost}
	private, key, _ := fixture(t)
	key.Created = time.Now().Add(-time.Hour).Unix()
	keyResponse, _ := json.Marshal(map[string]any{"key": key, "request_id": "synthetic-production-key-request"})
	keys, e := NewHTTPKeysForProfile(transportFunc(func(r *http.Request) (*http.Response, error) {
		if pool.active.Load() != 0 || r.URL.String() != "https://production.plaid.com/webhook_verification_key/get" {
			t.Fatal("key fetch crossed SQL/profile boundary")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(keyResponse)))}, nil
	}), credentialsFunc(func(context.Context) (Credentials, error) {
		return Credentials{"synthetic-production-client", "synthetic-production-secret"}, nil
	}), productionProfile())
	if e != nil {
		t.Fatal(e)
	}
	v, _ := NewVerifierForProfile(keys, productionProfile())
	receiver, _ := NewSQLReceiverForProfile(pool, productionProfile())
	h, e := NewHandler(v, receiver)
	if e != nil {
		t.Fatal(e)
	}
	return linkHTTP{h, pool, private, key}
}

// productionTransactionBody uses fixed supported metadata, never private transaction payloads.
func productionTransactionBody() []byte {
	return []byte(strings.Replace(string(bodyFixture()), `"sandbox"`, `"production"`, 1))
}

// TestProductionWebhookSQLReceipt proves committed reply loss, exact replay and reordered distinct notifications.
func TestProductionWebhookSQLReceipt(t *testing.T) {
	f := productionFixture(t)
	h := signedProductionHTTP(t, f.db, "accept_verified_webhook")
	body := productionTransactionBody()
	token := signBody(t, h.private, h.key.ID, body, time.Now().Unix())
	if invokeHandler(h.handler, token, body).Code != 503 || !h.pool.fired {
		t.Fatal("committed reply loss not modeled")
	}
	for i := 0; i < 2; i++ {
		if invokeHandler(h.handler, token, body).Code != 204 {
			t.Fatal("exact signed replay rejected")
		}
	}
	if value(t, f.admin, "SELECT count(*)::text FROM integration.webhook_receipt") != "1" || value(t, f.admin, "SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='webhook'") != "1" {
		t.Fatal("duplicate delivery changed demand")
	}
	older := []byte(strings.Replace(string(body), `"historical_update_complete":false`, `"historical_update_complete":true`, 1))
	olderToken := signBody(t, h.private, h.key.ID, older, time.Now().Add(-time.Minute).Unix())
	if invokeHandler(h.handler, olderToken, older).Code != 204 || invokeHandler(h.handler, token, body).Code != 204 {
		t.Fatal("reordered metadata rejected")
	}
	if value(t, f.admin, "SELECT count(*)::text FROM integration.webhook_receipt") != "2" || value(t, f.admin, "SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='webhook'") != "2" || value(t, f.admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, f.admin, "SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'") != "0" {
		t.Fatal("unexpected receipt/demand/accounting effects")
	}
	if _, e := invoke(t, f.db, productionHook, "SELECT integration.set_webhook_ingress('synthetic_production_webhook','production','unauthorized',false)::text"); e == nil {
		t.Fatal("receiver gained owner configuration")
	}
}

// TestProductionWebhookCurrentScope reauthorizes each signed delivery and never creates a receipt on denial.
func TestProductionWebhookCurrentScope(t *testing.T) {
	for _, change := range []string{"unknown-item", "source", "consent", "scope", "ingress", "wrong-profile"} {
		t.Run(change, func(t *testing.T) {
			f := productionFixture(t)
			h := signedProductionHTTP(t, f.db, "")
			body := productionTransactionBody()
			switch change {
			case "unknown-item":
				body = []byte(strings.Replace(string(body), "synthetic-item", "foreign-item", 1))
			case "source":
				exec(t, f.admin, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", reader)
			case "consent":
				exec(t, f.admin, "UPDATE integration.sync_item SET generation=generation+1,consent_expires_at=clock_timestamp()-interval '1 second'")
			case "scope":
				callOwner(t, f.db, "SELECT integration.configure_provider_scope($1,$2,$3,'production-disable',$4)::text", actor, entity, book, raw(map[string]any{"environment": "production", "api_login": productionAPI, "worker_login": productionSync, "bank_key": productionBank, "link_key": productionLink, "enabled": false}))
			case "ingress":
				callOwner(t, f.db, "SELECT integration.set_webhook_ingress('synthetic_production_webhook','production','production-disable',false)::text")
			case "wrong-profile":
				body = bodyFixture()
			}
			if h.send(t, body) == 204 || value(t, f.admin, "SELECT count(*)::text FROM integration.webhook_receipt") != "0" || value(t, f.admin, "SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='webhook'") != "0" {
				t.Fatal("current denial produced receipt/demand")
			}
		})
	}
}

// TestProductionIngressIdentityCannotMove preserves immutable environment binding after replacement/disable.
func TestProductionIngressIdentityCannotMove(t *testing.T) {
	f := productionFixture(t)
	callOwner(t, f.db, "SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','sandbox-ingress',true)::text")
	if _, e := invoke(t, f.db, "synthetic_operator", "SELECT integration.set_webhook_ingress('synthetic_production_webhook','sandbox','cross-profile',true)::text"); e == nil {
		t.Fatal("Production login moved to Sandbox")
	}
	callOwner(t, f.db, "SELECT integration.set_webhook_ingress('synthetic_production_webhook','production','disabled',false)::text")
	if _, e := invoke(t, f.db, "synthetic_operator", "SELECT integration.set_webhook_ingress('synthetic_production_webhook','sandbox','cross-disabled',true)::text"); e == nil {
		t.Fatal("disabled identity moved profile")
	}
	down, e := migrations.Files.ReadFile("000035_production_webhooks.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.admin.Exec(context.Background(), string(down)); e == nil {
		t.Fatal("downgrade discarded Production ingress history")
	}
	exec(t, f.admin, "ROLLBACK")
}

// TestProductionOwnedSessionSQL binds signed proof to one owned environment/attempt and existing immutable credential.
func TestProductionOwnedSessionSQL(t *testing.T) {
	f := productionFixture(t)
	a := productionCall(t, f.db, "SELECT integration.begin_connection($1,$2,$3,'production-repair',$4)::text", raw(map[string]any{"mode": "reconnect", "sync_item_id": f.item, "expected_generation": value(t, f.admin, "SELECT generation::text FROM integration.sync_item"), "approved_stream_ids": []any{f.stream}}))
	a = productionCall(t, f.db, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": tokenHash("production-repair-link"), "expires_at": "2099-01-01T00:00:00Z"}))
	addSandboxLinkCollision(t, f, tokenHash("production-repair-link"))
	scoped := f.store.ForScope(connections.Scope{Actor: actor, Entity: entity, Book: book})
	before := f.bank.calls
	reader := scoped.(connections.RepairCredentialReader)
	if _, e := reader.ReadRepair(context.Background(), a["attempt_id"].(string), f.reference); e == nil || f.bank.calls != before {
		t.Fatal("pending proof unwrapped credential")
	}
	h := signedProductionHTTP(t, f.db, "")
	body := []byte(`{"webhook_type":"LINK","webhook_code":"SESSION_FINISHED","environment":"production","status":"success","link_token":"production-repair-link","link_session_id":"production-repair-session","public_tokens":[]}`)
	if h.send(t, body) != 204 {
		t.Fatal("owned Production proof rejected")
	}
	if value(t, f.admin, "SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='webhook'") != "0" || f.bank.calls != before {
		t.Fatal("ingress performed acquisition or token work")
	}
	proof := productionCall(t, f.db, "SELECT integration.connection_link_proof($1,$2,$3,$4)::text", a["attempt_id"])
	if proof["status"] != "success" {
		t.Fatal("owned proof missing")
	}
	a = productionCall(t, f.db, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil}))
	got, e := reader.ReadRepair(context.Background(), a["attempt_id"].(string), f.reference)
	if e != nil || got.Reference != f.reference || got.Token != connections.NewSecret("synthetic-production-access") {
		t.Fatal("Production purpose credential read failed", e)
	}
	a = productionCall(t, f.db, "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["claim_id"], raw(productionManifest()))
	for i := 0; i < 2; i++ {
		productionCall(t, f.db, "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", a["attempt_id"], a["claim_id"])
	}
	if value(t, f.admin, "SELECT count(*)::text FROM integration.link_repair_claim WHERE environment='production'") != "1" || value(t, f.admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, f.admin, "SELECT count(*)::text FROM integration.bank_credential") != "1" || value(t, f.admin, "SELECT requested_generation::text FROM integration.sync_stream") != "2" {
		t.Fatal("repair changed immutable credential/accounting history")
	}
}

// addSandboxLinkCollision creates a separately authorized Sandbox attempt with an independently fixed identical hash.
func addSandboxLinkCollision(t *testing.T, f productionGraph, hash string) {
	t.Helper()
	exec(t, f.admin, "INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Fictional Sandbox collision')", entity, other)
	callOwner(t, f.db, "SELECT integration.set_connect_permission($1,$2,$3,$1,'sandbox-collision-connect',true)::text", actor, entity, other)
	callOwner(t, f.db, "SELECT integration.set_connection_worker($1,$2,$3,'sandbox-collision-worker',$4)::text", actor, entity, other, raw(map[string]any{"service_principal_id": reader, "worker_login": "synthetic_sync", "enabled": true}))
	a := pgObject(t, pgCall(t, f.db, "synthetic_runtime", "SELECT integration.begin_connection($1,$2,$3,'sandbox-collision',$4)::text", actor, entity, other, raw(map[string]any{"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}})))
	pgCall(t, f.db, "synthetic_runtime", "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, other, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": hash, "expires_at": "2099-01-01T00:00:00Z"}))
}

// TestProductionIngressHistoricalBackfill proves a retired pre35 Sandbox login cannot be rebound to Production.
func TestProductionIngressHistoricalBackfill(t *testing.T) {
	admin, db := pgFixture(t)
	down, e := migrations.Files.ReadFile("000035_production_webhooks.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	exec(t, admin, string(down))
	exec(t, admin, `DO $$ BEGIN IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='synthetic_replacement_webhook') THEN CREATE ROLE synthetic_replacement_webhook LOGIN IN ROLE bookkeeping_webhook;END IF;END $$`)
	callOwner(t, db, "SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','historical-original',true)::text")
	callOwner(t, db, "SELECT integration.set_webhook_ingress('synthetic_replacement_webhook','sandbox','historical-replacement',true)::text")
	up, e := migrations.Files.ReadFile("000035_production_webhooks.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	exec(t, admin, string(up))
	if _, e := invoke(t, db, "synthetic_operator", "SELECT integration.set_webhook_ingress('synthetic_webhook','production','illegal-reuse',true)::text"); e == nil {
		t.Fatal("retired historical login changed environment")
	}
	if value(t, admin, "SELECT count(*)::text FROM integration.webhook_login_profile WHERE login_name='synthetic_webhook' AND environment='sandbox'") != "1" {
		t.Fatal("historical identity was not preserved")
	}
}

// TestProductionWebhookRoleIsolation denies API, worker and Sandbox ingress identities at actual login boundaries.
func TestProductionWebhookRoleIsolation(t *testing.T) {
	f := productionFixture(t)
	callOwner(t, f.db, "SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','sandbox-config',true)::text")
	v := Verified{Environment: "production", ItemID: "synthetic-item", ReceiptKey: strings.Repeat("a", 64), BodySHA256: strings.Repeat("b", 64), IssuedAt: time.Now().Unix()}
	for _, login := range []string{productionAPI, productionSync, "synthetic_webhook"} {
		conn := connect(t, f.db, login)
		r, _ := NewSQLReceiverForProfile(conn, productionProfile())
		if _, e := r.Receive(context.Background(), v); e != errDenied {
			t.Fatal("foreign role entered Production ingress")
		}
		conn.Close(context.Background())
	}
	for _, q := range []string{"SELECT envelope::text FROM integration.bank_credential", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text"} {
		args := []any{}
		if strings.Contains(q, "get_repair") {
			args = []any{actor, entity, book, f.attempt["attempt_id"]}
		}
		_, e := invoke(t, f.db, productionHook, q, args...)
		requireCode(t, e, "42501")
	}
	if value(t, f.admin, "SELECT count(*)::text FROM integration.webhook_receipt") != "0" {
		t.Fatal("role denial created receipt")
	}
	for _, grant := range []struct{ add, remove string }{
		{"GRANT bookkeeping_verifier TO " + productionHook, "REVOKE bookkeeping_verifier FROM " + productionHook},
		{"GRANT pg_read_all_data TO " + productionHook, "REVOKE pg_read_all_data FROM " + productionHook},
		{"GRANT SELECT ON integration.bank_credential TO " + productionHook, "REVOKE SELECT ON integration.bank_credential FROM " + productionHook},
	} {
		func() {
			exec(t, f.admin, grant.add)
			defer exec(t, f.admin, grant.remove)
			conn := connect(t, f.db, productionHook)
			defer conn.Close(context.Background())
			r, _ := NewSQLReceiverForProfile(conn, productionProfile())
			if _, e := r.Receive(context.Background(), v); e != errDenied {
				t.Fatal("role drift entered Go receiver")
			}
			_, e := invoke(t, f.db, productionHook, "SELECT integration.accept_verified_webhook('production',$1,'{}')::text", v.ReceiptKey)
			requireCode(t, e, "P2801")
		}()
	}
}

// TestProductionSessionAmbiguity retains distinct sessions without selecting a repair or creating acquisition demand.
func TestProductionSessionAmbiguity(t *testing.T) {
	f := productionFixture(t)
	h := signedProductionHTTP(t, f.db, "")
	link := func(token, session string) []byte {
		body := map[string]any{"webhook_type": "LINK", "webhook_code": "SESSION_FINISHED", "environment": "production", "status": "success", "link_token": token, "link_session_id": session, "public_tokens": []string{}}
		raw, _ := json.Marshal(body)
		return raw
	}
	for _, token := range []string{"unknown-production-link", "production-new-link"} {
		if h.send(t, link(token, "ignored")) != 204 {
			t.Fatal("unmatched event not ignored")
		}
	}
	if value(t, f.admin, "SELECT count(*)::text FROM integration.link_session_receipt") != "0" {
		t.Fatal("unmatched session created proof")
	}
	a := productionCall(t, f.db, "SELECT integration.begin_connection($1,$2,$3,'ambiguous-repair',$4)::text", raw(map[string]any{"mode": "reconnect", "sync_item_id": f.item, "expected_generation": value(t, f.admin, "SELECT generation::text FROM integration.sync_item"), "approved_stream_ids": []any{f.stream}}))
	a = productionCall(t, f.db, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": tokenHash("ambiguous-production-link"), "expires_at": "2099-01-01T00:00:00Z"}))
	for _, session := range []string{"one", "two"} {
		if h.send(t, link("ambiguous-production-link", session)) != 204 {
			t.Fatal("signed session rejected")
		}
	}
	if productionCall(t, f.db, "SELECT integration.connection_link_proof($1,$2,$3,$4)::text", a["attempt_id"])["status"] != "ambiguous" {
		t.Fatal("ambiguous proof selected")
	}
	if _, e := invoke(t, f.db, productionAPI, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil})); e == nil {
		t.Fatal("ambiguous session claimed")
	}
	if value(t, f.admin, "SELECT count(*)::text FROM integration.link_repair_claim") != "0" || value(t, f.admin, "SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='webhook'") != "0" {
		t.Fatal("ambiguous proof created authority")
	}
}
