//go:build integration

package webhook

import (
	"context"
	"crypto/ecdsa"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// linkAPI supplies fixed synthetic actor/book scope through the actual restricted API login.
func linkAPI(t testing.TB, db, q string, args ...any) map[string]any {
	t.Helper()
	return pgObject(t, pgCall(t, db, "synthetic_runtime", q, append([]any{actor, entity, book}, args...)...))
}

// repairFixture establishes one bound Item through the real lifecycle before a fresh reviewed repair.
// Owner access installs only fictional identities/reference rows, never bypassing the repair proof gate.
func repairFixture(t *testing.T) (*pgx.Conn, string, map[string]any) {
	t.Helper()
	c, db := pgFixture(t)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'import',NULL),($1,$3,'import',NULL)`, entity, actor, reader)
	callOwner(t, db, "SELECT integration.set_connect_permission($1,$2,$3,$1,'connect',true)::text", actor, entity, book)
	callOwner(t, db, "SELECT integration.set_connection_worker($1,$2,$3,'worker',$4)::text", actor, entity, book,
		raw(map[string]any{"service_principal_id": reader, "worker_login": "synthetic_sync", "enabled": true}))
	a := linkAPI(t, db, "SELECT integration.begin_connection($1,$2,$3,'new',$4)::text", raw(map[string]any{
		"mode": "new", "sync_item_id": nil, "expected_generation": nil, "approved_stream_ids": []any{}}))
	a = linkAPI(t, db, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"],
		raw(map[string]any{"link_token_sha256": tokenHash("new-synthetic"), "expires_at": "2099-01-01T00:00:00Z"}))
	a = linkAPI(t, db, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"],
		raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": tokenHash("public-synthetic")}))
	a = linkAPI(t, db, "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["claim_id"],
		raw(map[string]any{"item_id": "synthetic-item", "accounts": []any{map[string]any{
			"account_id": "Account-A", "type": "depository", "iso_currency_code": "USD"}}, "consent_expires_at": "2099-01-01T00:00:00Z"}))
	a = linkAPI(t, db, "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", a["attempt_id"], a["claim_id"])
	bound := linkAPI(t, db, "SELECT integration.bind_connection($1,$2,$3,$4,'bind',$5)::text", a["attempt_id"], raw(map[string]any{
		"consent_grants": true, "bindings": []any{map[string]any{"account_id": "Account-A", "ledger_account_id": account,
			"currency_version_id": currency, "source_account_id": nil, "expected_binding_revision": nil,
			"binding_date": "2026-01-05", "binding_evidence_ref": csvNS}}}))
	stream := bound["streams"].([]any)[0].(map[string]any)["stream_id"]
	a = linkAPI(t, db, "SELECT integration.begin_connection($1,$2,$3,'repair',$4)::text", raw(map[string]any{
		"mode": "reconnect", "sync_item_id": value(t, c, "SELECT sync_item_id::text FROM integration.sync_item"),
		"expected_generation": value(t, c, "SELECT generation::text FROM integration.sync_item"), "approved_stream_ids": []any{stream}}))
	a = linkAPI(t, db, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"],
		raw(map[string]any{"link_token_sha256": tokenHash("link-synthetic"), "expires_at": "2099-01-01T00:00:00Z"}))
	callOwner(t, db, "SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','fixture',true)")
	return c, db, a
}

// linkHTTP signs exact synthetic bytes and uses a real isolated receiver session, with no bank/vault clients.
type linkHTTP struct {
	handler *Handler
	pool    *lossPool
	private *ecdsa.PrivateKey
	key     JWK
}

// signedLinkPostgres constructs only ephemeral signature material and the restricted SQL adapter.
func signedLinkPostgres(t *testing.T, db, lost string) linkHTTP {
	t.Helper()
	c := connect(t, db, "synthetic_webhook")
	t.Cleanup(func() { c.Close(context.Background()) })
	pool := &lossPool{conn: c, lost: lost}
	private, key, _ := fixture(t)
	key.Created = time.Now().Add(-time.Hour).Unix()
	v, _ := NewVerifier(keyFunc(func(context.Context, string) (JWK, error) {
		if pool.active.Load() != 0 {
			t.Error("signature-key lookup inside business transaction")
		}
		return key, nil
	}))
	r, _ := NewSQLReceiver(pool)
	h, _ := NewHandler(v, r)
	return linkHTTP{h, pool, private, key}
}

// send returns only the fixed HTTP classification, never event or database diagnostics.
func (h linkHTTP) send(t testing.TB, body []byte) int {
	t.Helper()
	return invokeHandler(h.handler, signBody(t, h.private, h.key.ID, body, time.Now().Unix()), body).Code
}

// noLinkEffects independently asserts that signed receipt acceptance cannot execute a repair or financial mutation.
func noLinkEffects(t testing.TB, c *pgx.Conn) {
	t.Helper()
	for _, q := range []string{"SELECT count(*)::text FROM integration.link_repair_claim",
		"SELECT count(*)::text FROM integration.sync_wakeup", "SELECT count(*)::text FROM ledger.journal_entry",
		"SELECT count(*)::text FROM banking.source_observation", "SELECT count(*)::text FROM integration.bank_credential",
		"SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'"} {
		if value(t, c, q) != "0" {
			t.Fatal("receipt authorized an unintended effect")
		}
	}
	if value(t, c, "SELECT credential_version::text FROM integration.sync_item") != "1" ||
		value(t, c, "SELECT count(*)::text FROM integration.credential_publication") != "1" ||
		value(t, c, "SELECT requested_generation::text FROM integration.sync_stream") != "1" {
		t.Fatal("receipt changed existing identity or acquisition generation")
	}
}

// TestHostedRepairSignedReplay proves actual committed reply loss, equivalent-signature replay and hashed persistence.
func TestHostedRepairSignedReplay(t *testing.T) {
	c, db, a := repairFixture(t)
	h := signedLinkPostgres(t, db, "accept_verified_link_session")
	body := []byte(strings.TrimSuffix(string(linkBody("link-synthetic", "session-synthetic", "success")), "}") +
		`,"user_id":"foreign-user","public_tokens":["never-exchange-synthetic"]}`)
	token := signBody(t, h.private, h.key.ID, body, time.Now().Unix())
	if invokeHandler(h.handler, token, body).Code != http.StatusServiceUnavailable || !h.pool.fired {
		t.Fatal("unknown commit was acknowledged")
	}
	for _, delivery := range []string{token, token, alternateSignature(t, token), signBody(t, h.private, h.key.ID, body, time.Now().Unix())} {
		if invokeHandler(h.handler, delivery, body).Code != http.StatusNoContent {
			t.Fatal("durable replay failed")
		}
	}
	if linkAPI(t, db, "SELECT integration.connection_link_proof($1,$2,$3,$4)::text", a["attempt_id"])["status"] != "success" {
		t.Fatal("same session became ambiguous")
	}
	if value(t, c, "SELECT count(DISTINCT link_session_sha256)::text FROM integration.link_session_receipt") != "1" ||
		value(t, c, "SELECT count(*)::text FROM integration.link_session_receipt WHERE link_token_sha256=$1 AND link_session_sha256=$2", tokenHash("link-synthetic"), tokenHash("session-synthetic")) != "2" {
		t.Fatal("hashed session or replay identity mismatch")
	}
	stored := value(t, c, "SELECT jsonb_agg(to_jsonb(r))::text FROM integration.link_session_receipt r")
	for _, secret := range []string{"link-synthetic", "session-synthetic", "foreign-user", "never-exchange-synthetic", token} {
		if strings.Contains(stored, secret) {
			t.Fatal("raw token or user identifier retained")
		}
	}
	noLinkEffects(t, c)
	// The actual ingress session still cannot borrow API identity or direct table reads.
	_, e := invoke(t, db, "synthetic_webhook", "SELECT integration.connection_link_proof($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "42501")
	_, e = invoke(t, db, "synthetic_webhook", "SELECT count(*)::text FROM integration.bank_credential")
	requireCode(t, e, "42501")
}

// TestHostedRepairSignedForeignScope acknowledges unmatched/new-mode events without disclosing or changing an attempt.
func TestHostedRepairSignedForeignScope(t *testing.T) {
	c, db, a := repairFixture(t)
	h := signedLinkPostgres(t, db, "")
	for _, link := range []string{"unknown-synthetic", "new-synthetic"} {
		if h.send(t, linkBody(link, "foreign-session", "success")) != http.StatusNoContent {
			t.Fatal("ignored event exposed lookup result")
		}
	}
	if value(t, c, "SELECT count(*)::text FROM integration.link_session_receipt") != "0" ||
		linkAPI(t, db, "SELECT integration.connection_link_proof($1,$2,$3,$4)::text", a["attempt_id"])["status"] != "pending" {
		t.Fatal("foreign or new-mode token authorized repair")
	}
	noLinkEffects(t, c)
}

// TestHostedRepairSignedAmbiguous retains outcomes without selecting an exited/conflicting/second session for execution.
func TestHostedRepairSignedAmbiguous(t *testing.T) {
	for _, tc := range []struct{ name, first, second, session, want string }{
		{"exit", "exited", "", "", "exited"}, {"conflict", "success", "exited", "one", "ambiguous"},
		{"sessions", "success", "success", "two", "ambiguous"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, db, a := repairFixture(t)
			h := signedLinkPostgres(t, db, "")
			if h.send(t, linkBody("link-synthetic", "one", tc.first)) != 204 {
				t.Fatal("first receipt rejected")
			}
			if tc.second != "" && h.send(t, linkBody("link-synthetic", tc.session, tc.second)) != 204 {
				t.Fatal("second receipt rejected")
			}
			if linkAPI(t, db, "SELECT integration.connection_link_proof($1,$2,$3,$4)::text", a["attempt_id"])["status"] != tc.want {
				t.Fatal("wrong ambiguity disposition")
			}
			noLinkEffects(t, c)
		})
	}
}

// TestHostedRepairSignedRevoked rechecks current authority and deadline at signed delivery, including replay.
func TestHostedRepairSignedRevoked(t *testing.T) {
	for _, change := range []string{"connect", "worker", "expired", "ingress"} {
		t.Run(change, func(t *testing.T) {
			c, db, a := repairFixture(t)
			h := signedLinkPostgres(t, db, "")
			body := linkBody("link-synthetic", "one", "success")
			token := signBody(t, h.private, h.key.ID, body, time.Now().Unix())
			if invokeHandler(h.handler, token, body).Code != 204 {
				t.Fatal("initial proof rejected")
			}
			switch change {
			case "connect":
				exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='connect'", actor)
			case "worker":
				exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", reader)
			case "expired":
				exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
			case "ingress":
				callOwner(t, db, "SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','disabled',false)")
			}
			if invokeHandler(h.handler, token, body).Code != 403 {
				t.Fatal("revoked or expired replay acknowledged")
			}
			if h.send(t, linkBody("link-synthetic", "second", "success")) != 403 ||
				value(t, c, "SELECT count(*)::text FROM integration.link_session_receipt") != "1" {
				t.Fatal("current denial failed atomically")
			}
			noLinkEffects(t, c)
		})
	}
}
