//go:build integration

package connectionops

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// repairHash supplies only deterministic synthetic protocol identities, never a financial oracle.
func repairHash(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(s))) }

// repairStart uses the actual API login and fresh Item/sibling approval, with a unique owned token hash.
func repairStart(t *testing.T, c *pgx.Conn, db, key string) map[string]any {
	t.Helper()
	item := value(t, c, "SELECT sync_item_id::text FROM integration.sync_item")
	a := itemIntent(t, c, db, key, "reconnect", item)
	if a["execute_link"] != true {
		t.Fatal("fresh repair lost its one-shot Link permission")
	}
	if itemIntent(t, c, db, key, "reconnect", item)["execute_link"] != false {
		t.Fatal("repair replay minted Link permission")
	}
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text",
		actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{
			"link_token_sha256": repairHash(key), "expires_at": "2099-01-01T00:00:00Z",
		})))
}

// repairWire models the receiver's already signature-verified closed projection, not a signed event.
func repairWire(a map[string]any, session, status string) string {
	return raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "link_session_sha256": repairHash(session),
		"body_sha256": repairHash(session + ":" + status), "status": status, "issued_at": time.Now().Unix()})
}

// repairDelivery executes only the dedicated receipt function through a real isolated login.
func repairDelivery(t testing.TB, db, key, wire string) map[string]any {
	t.Helper()
	return object(t, call(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash(key), wire))
}

// repairProof reads the current owned status through the API role; it grants no execution by itself.
func repairProof(t testing.TB, db string, a map[string]any) map[string]any {
	t.Helper()
	return object(t, call(t, db, "synthetic_runtime", "SELECT integration.connection_link_proof($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]))
}

// requireNoRepair checks fixed independent effect counts after rejection, not values derived by a generator.
func requireNoRepair(t testing.TB, c *pgx.Conn) {
	t.Helper()
	if value(t, c, "SELECT count(*)::text FROM integration.link_repair_claim") != "0" ||
		value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "0" ||
		value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" ||
		value(t, c, "SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'") != "0" {
		t.Fatal("unconfirmed proof produced an effect")
	}
}

// TestHostedRepairSQLProofAndPublication proves owned receipt→one claim→one repair cause per active stream.
func TestHostedRepairSQLProofAndPublication(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	before := value(t, c, "SELECT jsonb_agg(jsonb_build_array(stream_id,source_account_id,namespace_id,account_id) ORDER BY stream_id)::text FROM integration.sync_stream")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream WHERE requested_generation=1") != "2" {
		t.Fatal("fixture must start with the independently specified initial demand generation1")
	}
	a := repairStart(t, c, db, "repair")
	if repairProof(t, db, a)["status"] != "pending" {
		t.Fatal("fresh repair has proof")
	}
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text",
		actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil}))
	requireCode(t, e, "P2704")
	requireNoRepair(t, c)
	wire := repairWire(a, "owned-session", "success")
	if repairDelivery(t, db, "first", wire)["state"] != "accepted" || repairDelivery(t, db, "first", wire)["state"] != "duplicate" {
		t.Fatal("receipt replay changed association")
	}
	repairDelivery(t, db, "resigned", wire)
	if repairProof(t, db, a)["status"] != "success" || value(t, c, "SELECT count(*)::text FROM integration.link_session_receipt") != "2" {
		t.Fatal("same session was counted as ambiguous")
	}
	requireNoRepair(t, c)
	a = claim(t, db, a)
	if a["execute_provider"] != true || claim(t, db, a)["execute_provider"] != false {
		t.Fatal("claim replay minted another execution")
	}
	a = recordManifest(t, db, a, verified("SyntheticItem", "AccountA", "AccountB"))
	result := confirm(t, db, a)
	if confirm(t, db, result)["attempt_id"] != a["attempt_id"] {
		t.Fatal("confirmation replay changed attempt")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.link_repair_claim") != "1" ||
		value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='resume'") != "2" ||
		value(t, c, "SELECT count(*)::text FROM integration.sync_stream WHERE requested_generation=2") != "2" ||
		value(t, c, "SELECT credential_version::text FROM integration.sync_item") != "1" ||
		value(t, c, "SELECT count(*)::text FROM integration.credential_publication") != "1" ||
		value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" ||
		value(t, c, "SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'") != "0" ||
		value(t, c, "SELECT jsonb_agg(jsonb_build_array(stream_id,source_account_id,namespace_id,account_id) ORDER BY stream_id)::text FROM integration.sync_stream") != before {
		t.Fatal("repair changed identity, credential version or financial effects")
	}
	if repairDelivery(t, db, "late-other-session", repairWire(a, "another-session", "success"))["state"] != "ignored" ||
		value(t, c, "SELECT count(*)::text FROM integration.link_session_receipt") != "2" ||
		value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "2" {
		t.Fatal("late event replaced pinned proof")
	}
}

// TestHostedRepairSQLAmbiguous rejects exits, conflicting outcomes and multiple distinct completed sessions.
func TestHostedRepairSQLAmbiguous(t *testing.T) {
	for _, tc := range []struct{ name, first, second, session, want string }{
		{"exit", "exited", "", "", "exited"},
		{"conflict", "success", "exited", "one", "ambiguous"},
		{"sessions", "success", "success", "two", "ambiguous"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, db, _, _ := connected(t)
			webhookConfig(t, db)
			a := repairStart(t, c, db, "repair")
			repairDelivery(t, db, "first", repairWire(a, "one", tc.first))
			if tc.second != "" {
				repairDelivery(t, db, "second", repairWire(a, tc.session, tc.second))
			}
			if repairProof(t, db, a)["status"] != tc.want {
				t.Fatal("wrong proof disposition")
			}
			_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil}))
			requireCode(t, e, "P2704")
			requireNoRepair(t, c)
			call(t, db, "synthetic_runtime", "SELECT integration.abandon_connection($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
			call(t, db, "synthetic_runtime", "SELECT integration.abandon_connection($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
			if value(t, c, "SELECT state FROM integration.sync_item") != "active" || value(t, c, "SELECT count(*)::text FROM integration.connection_attempt") != "2" {
				t.Fatal("local cancellation replaced or removed the Item")
			}
			if repairDelivery(t, db, "late", repairWire(a, "late", "success"))["state"] != "ignored" {
				t.Fatal("cancelled proof accepted")
			}
			requireNoRepair(t, c)
		})
	}
}

// TestHostedRepairSQLScopeDrift checks current authority and scope again after receiving a valid proof.
func TestHostedRepairSQLScopeDrift(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		change     func(*testing.T, *pgx.Conn, string, map[string]any)
	}{
		{"connect", "P2701", func(t *testing.T, c *pgx.Conn, _ string, _ map[string]any) {
			exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='connect'", actor)
		}},
		{"source", "P2701", func(t *testing.T, c *pgx.Conn, _ string, _ map[string]any) {
			exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", reader)
		}},
		{"binding", "P2703", func(t *testing.T, c *pgx.Conn, db string, _ map[string]any) {
			sid := value(t, c, "SELECT source_account_id::text FROM banking.source_account ORDER BY source_account_id LIMIT 1")
			rev := value(t, c, "SELECT binding_revision::text FROM banking.source_account WHERE source_account_id=$1", sid)
			call(t, db, "synthetic_operator", "SELECT integration.set_source_bind_permission($1,$2,$3,$4,'renew-current-bind',$5)::text", actor, entity, book, sid, raw(map[string]any{"expected_binding_revision": rev, "principal_id": actor, "enabled": true}))
		}},
		{"paused", "P2703", func(t *testing.T, c *pgx.Conn, db string, _ map[string]any) {
			sid := value(t, c, "SELECT stream_id::text FROM integration.sync_stream ORDER BY stream_id LIMIT 1")
			operate(t, db, sid, "pause-after-proof", mutation(status(t, db, sid), "pause"))
		}},
		{"expired", "P2703", func(t *testing.T, c *pgx.Conn, _ string, a map[string]any) {
			exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, db, _, _ := connected(t)
			webhookConfig(t, db)
			a := repairStart(t, c, db, "repair")
			repairDelivery(t, db, "proof", repairWire(a, "owned", "success"))
			tc.change(t, c, db, a)
			_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil}))
			requireCode(t, e, tc.code)
			requireNoRepair(t, c)
		})
	}
}

// TestHostedRepairSQLPausedSibling requires the paused account in the manifest without resuming its work.
func TestHostedRepairSQLPausedSibling(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	paused := value(t, c, "SELECT stream_id::text FROM integration.sync_stream WHERE account_id='AccountB'")
	operate(t, db, paused, "pause-before-repair", mutation(status(t, db, paused), "pause"))
	a := repairStart(t, c, db, "repair")
	repairDelivery(t, db, "proof", repairWire(a, "owned", "success"))
	a = claim(t, db, a)
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["claim_id"], raw(verified("SyntheticItem", "AccountA")))
	requireCode(t, e, "P2704")
	a = recordManifest(t, db, a, verified("SyntheticItem", "AccountA", "AccountB"))
	confirm(t, db, a)
	if value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "1" || value(t, c, "SELECT state FROM integration.sync_stream WHERE stream_id=$1", paused) != "paused" {
		t.Fatal("repair widened paused scope")
	}
}

// TestHostedRepairSQLRoleAndReceiptBoundary proves narrow actual-login authority, closed fields and disabled ingress.
func TestHostedRepairSQLRoleAndReceiptBoundary(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	a := repairStart(t, c, db, "repair")
	wire := repairWire(a, "owned", "success")
	for _, user := range []string{"synthetic_runtime", "synthetic_sync"} {
		_, e := invoke(t, db, user, "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash("key"), wire)
		requireCode(t, e, "42501")
	}
	for _, user := range []string{"synthetic_runtime", "synthetic_webhook"} {
		_, e := invoke(t, db, user, "SELECT count(*)::text FROM integration.link_session_receipt")
		requireCode(t, e, "42501")
		_, e = invoke(t, db, user, "SELECT integration.claim_connection_before_owned_repair32($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil}))
		requireCode(t, e, "42501")
	}
	_, e := invoke(t, db, "synthetic_webhook", "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "42501")
	_, e = invoke(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('production',$1,$2)::text", repairHash("key"), wire)
	requireCode(t, e, "P2800")
	_, e = invoke(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash("key"), strings.TrimSuffix(wire, "}")+`,"actor_id":"untrusted"}`)
	requireCode(t, e, "P2800")
	if repairDelivery(t, db, "proof", wire)["state"] != "accepted" {
		t.Fatal("dedicated receiver borrowed API authority")
	}
	call(t, db, "synthetic_operator", "SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','disable-proof',false)::text")
	_, e = invoke(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash("later"), wire)
	requireCode(t, e, "P2801")
	requireNoRepair(t, c)
}

// TestHostedRepairSQLConcurrentClaimAndConfirm uses independent sessions and fixed one-effect expectations.
func TestHostedRepairSQLConcurrentClaimAndConfirm(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	a := repairStart(t, c, db, "repair")
	repairDelivery(t, db, "proof", repairWire(a, "owned", "success"))
	type result struct {
		raw string
		err error
	}
	results := make(chan result, 2)
	startID, startNonce, startHash := a["attempt_id"], a["nonce"], a["link_token_sha256"]
	for i := 0; i < 2; i++ {
		go func() {
			r, e := invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, startID, startNonce, raw(map[string]any{"link_token_sha256": startHash, "public_token_sha256": nil}))
			results <- result{r, e}
		}()
	}
	grants := 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		v := object(t, r.raw)
		if v["execute_provider"] == true {
			grants++
		}
		a = v
	}
	if grants != 1 {
		t.Fatal("concurrent claim repeated authority")
	}
	a = recordManifest(t, db, a, verified("SyntheticItem", "AccountA", "AccountB"))
	id, claimID := a["attempt_id"], a["claim_id"]
	for i := 0; i < 2; i++ {
		go func() {
			r, e := invoke(t, db, "synthetic_runtime", "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, id, claimID)
			results <- result{r, e}
		}()
	}
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
	}
	if value(t, c, "SELECT count(*)::text FROM integration.link_repair_claim") != "1" || value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "2" {
		t.Fatal("concurrent confirmation repeated repair")
	}
}

// TestHostedRepairSQLReceiptClosure keeps unknown/new sessions inert and rejects receipt identity substitution.
func TestHostedRepairSQLReceiptClosure(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	a := repairStart(t, c, db, "repair")
	unknown := map[string]any{"link_token_sha256": repairHash("unknown")}
	if repairDelivery(t, db, "unknown", repairWire(unknown, "none", "success"))["state"] != "ignored" {
		t.Fatal("unknown token acquired scope")
	}
	initial := map[string]any{"link_token_sha256": strings.Repeat("a", 64)}
	if repairDelivery(t, db, "new", repairWire(initial, "new-session", "success"))["state"] != "ignored" {
		t.Fatal("new session became repair")
	}
	wire := repairWire(a, "owned", "success")
	repairDelivery(t, db, "proof", wire)
	_, e := invoke(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash("proof"), repairWire(unknown, "other", "exited"))
	requireCode(t, e, "P2802")
	if value(t, c, "SELECT count(*)::text FROM integration.link_session_receipt") != "1" {
		t.Fatal("substituted receipt committed")
	}
	exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
	_, e = invoke(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash("late"), wire)
	requireCode(t, e, "P2703")
	requireNoRepair(t, c)
}

// TestHostedRepairSQLExpiredConsent denies a fresh intent even when its requested generation is current.
func TestHostedRepairSQLExpiredConsent(t *testing.T) {
	c, db, _, _ := connected(t)
	exec(t, c, "UPDATE integration.sync_item SET consent_expires_at=clock_timestamp()-interval '1 second',generation=generation+1")
	ids := object(t, `{"ids":`+value(t, c, "SELECT jsonb_agg(stream_id ORDER BY stream_id)::text FROM integration.sync_stream")+`}`)["ids"]
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.begin_connection($1,$2,$3,'expired-owner',$4)::text", actor, entity, book, raw(map[string]any{"mode": "reconnect", "sync_item_id": value(t, c, "SELECT sync_item_id::text FROM integration.sync_item"), "expected_generation": value(t, c, "SELECT generation::text FROM integration.sync_item"), "approved_stream_ids": ids}))
	requireCode(t, e, "P2703")
	if value(t, c, "SELECT count(*)::text FROM integration.connection_attempt") != "1" {
		t.Fatal("expired intent escaped rollback")
	}
	requireNoRepair(t, c)
}

// TestHostedRepairSQLCredentialPurpose reads existing ciphertext only through this exact current repair.
func TestHostedRepairSQLCredentialPurpose(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	// Independent synthetic ciphertext fixture: this tests addressing/authority, not AEAD correctness or KMS.
	exec(t, c, `INSERT INTO integration.bank_credential(entity_id,book_id,credential_ref,credential_version,item_id,environment,envelope)
 SELECT entity_id,book_id,credential_ref,credential_version,item_id,environment,'{"profile":"synthetic-authorization-only"}'::jsonb FROM integration.sync_item`)
	a := itemIntent(t, c, db, "repair", "reconnect", value(t, c, "SELECT sync_item_id::text FROM integration.sync_item"))
	before := object(t, call(t, db, "synthetic_runtime", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]))
	if before["environment"] != "sandbox" || before["item_id"] != "SyntheticItem" || before["credential_version"] != "1" {
		t.Fatal("unaddressed credential read")
	}
	_, e := invoke(t, db, "synthetic_webhook", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "42501")
	a = object(t, call(t, db, "synthetic_runtime", "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": repairHash("repair"), "expires_at": "2099-01-01T00:00:00Z"})))
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "P2704")
	repairDelivery(t, db, "proof", repairWire(a, "owned", "success"))
	a = claim(t, db, a)
	after := object(t, call(t, db, "synthetic_runtime", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]))
	if after["credential_ref"] != before["credential_ref"] || after["credential_version"] != before["credential_version"] {
		t.Fatal("claim selected another credential")
	}
	exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
	// Link expiry must not strand an already committed proof-pinned read-only recovery.
	late := object(t, call(t, db, "synthetic_runtime", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]))
	if late["credential_ref"] != before["credential_ref"] {
		t.Fatal("expiry selected another credential")
	}
	exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", reader)
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "P2701")
}

// TestHostedRepairSQLLegacyMigration executes actual down/up around a fictional31-era unclaimed intent.
func TestHostedRepairSQLLegacyMigration(t *testing.T) {
	c, db, _, _ := connected(t)
	before := value(t, c, "SELECT jsonb_agg(jsonb_build_array(stream_id,source_account_id,namespace_id) ORDER BY stream_id)::text FROM integration.sync_stream")
	for _, direction := range []string{"down", "up"} {
		data, e := os.ReadFile("/proof/migrations/000032_owned_hosted_repair." + direction + ".sql")
		if e != nil {
			t.Fatal(e)
		}
		exec(t, c, string(data))
		if direction == "down" {
			a := itemIntent(t, c, db, "legacy", "reconnect", value(t, c, "SELECT sync_item_id::text FROM integration.sync_item"))
			call(t, db, "synthetic_runtime", "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": repairHash("legacy"), "expires_at": "2099-01-01T00:00:00Z"}))
		}
	}
	if value(t, c, "SELECT count(*)::text FROM integration.connection_event WHERE kind='repair_window_capped'") != "1" || value(t, c, "SELECT count(*)::text FROM integration.connection_attempt WHERE mode='reconnect' AND repair_scope_sha256 IS NULL") != "1" {
		t.Fatal("legacy state silently reauthorized")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.connection_attempt a WHERE mode='reconnect' AND expires_at>(SELECT min(recorded_at)+interval '15 minutes' FROM integration.connection_event e WHERE e.attempt_id=a.attempt_id AND kind='begin')") != "0" {
		t.Fatal("legacy window extended")
	}
	id := value(t, c, "SELECT attempt_id::text FROM integration.connection_attempt WHERE mode='reconnect'")
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, id)
	requireCode(t, e, "P2703")
	call(t, db, "synthetic_runtime", "SELECT integration.abandon_connection($1,$2,$3,$4)::text", actor, entity, book, id)
	if value(t, c, "SELECT jsonb_agg(jsonb_build_array(stream_id,source_account_id,namespace_id) ORDER BY stream_id)::text FROM integration.sync_stream") != before {
		t.Fatal("legacy migration changed source identity")
	}
	requireNoRepair(t, c)
}

// TestHostedRepairSQLHandoffRecovery keeps one encrypted winner under live, owned, unclaimed scope.
func TestHostedRepairSQLHandoffRecovery(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	a := repairStart(t, c, db, "repair")
	// Fixed envelope syntax only; existing bankcredential tests own AEAD/AAD correctness.
	envelope := map[string]any{"profile": "jayburd-hosted-link-aead-v1", "key_version": "projects/9088794002/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1", "wrapped_key": "AQ==", "nonce": strings.Repeat("A", 16), "ciphertext": strings.Repeat("A", 23) + "="}
	first := call(t, db, "synthetic_runtime", "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], raw(envelope))
	envelope["wrapped_key"] = "Ag=="
	if call(t, db, "synthetic_runtime", "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], raw(envelope)) != first || call(t, db, "synthetic_runtime", "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]) != first {
		t.Fatal("handoff winner changed")
	}
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", other, entity, book, a["attempt_id"])
	requireCode(t, e, "P2701")
	_, e = invoke(t, db, "synthetic_webhook", "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "42501")
	repairDelivery(t, db, "proof", repairWire(a, "owned", "success"))
	a = claim(t, db, a)
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "P2704")
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], raw(envelope))
	requireCode(t, e, "P2704")
	exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "P2704")
	if value(t, c, "SELECT count(*)::text FROM integration.link_handoff") != "1" || value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "0" {
		t.Fatal("handoff recovery created an effect")
	}
}

// TestHostedRepairSQLKnownConsentExpiry denies ciphertext/confirmation after a known narrower manifest expires.
func TestHostedRepairSQLKnownConsentExpiry(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	exec(t, c, `INSERT INTO integration.bank_credential(entity_id,book_id,credential_ref,credential_version,item_id,environment,envelope)
 SELECT entity_id,book_id,credential_ref,credential_version,item_id,environment,'{"profile":"synthetic-authorization-only"}'::jsonb FROM integration.sync_item`)
	a := repairStart(t, c, db, "repair")
	repairDelivery(t, db, "proof", repairWire(a, "owned", "success"))
	a = claim(t, db, a)
	m := verified("SyntheticItem", "AccountA", "AccountB")
	expiry := value(t, c, `SELECT to_char((clock_timestamp()+interval '5 seconds') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`)
	m["consent_expires_at"] = expiry
	a = recordManifest(t, db, a, m)
	// Wait only for this recorded server-clock deadline (bounded5s), never rewrite immutable manifest evidence.
	exec(t, c, `SELECT pg_sleep(least(greatest(extract(epoch FROM ($1::timestamptz-clock_timestamp())),0),5)+0.01)`, expiry)
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, e, "P2703")
	_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], a["claim_id"])
	requireCode(t, e, "P2703")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "0" || value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("expired manifest produced an effect")
	}
}

// TestHostedRepairSQLCancelRaces serializes receipt/claim/cancel without automatic resume or replacement.
func TestHostedRepairSQLCancelRaces(t *testing.T) {
	for _, which := range []string{"receipt-cancel", "claim-cancel", "receipt-claim"} {
		t.Run(which, func(t *testing.T) {
			c, db, _, _ := connected(t)
			webhookConfig(t, db)
			a := repairStart(t, c, db, "repair")
			wire := repairWire(a, "owned", "success")
			if which == "claim-cancel" {
				repairDelivery(t, db, "proof", wire)
			}
			id, nonce, h := a["attempt_id"], a["nonce"], a["link_token_sha256"]
			type outcome struct {
				which string
				err   error
			}
			out := make(chan outcome, 2)
			claimCall := func() {
				_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, id, nonce, raw(map[string]any{"link_token_sha256": h, "public_token_sha256": nil}))
				out <- outcome{"claim", e}
			}
			receiptCall := func() {
				_, e := invoke(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash("proof"), wire)
				out <- outcome{"receipt", e}
			}
			cancelCall := func() {
				_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.abandon_connection($1,$2,$3,$4)::text", actor, entity, book, id)
				out <- outcome{"cancel", e}
			}
			switch which {
			case "receipt-cancel":
				go receiptCall()
				go cancelCall()
			case "claim-cancel":
				go claimCall()
				go cancelCall()
			case "receipt-claim":
				go receiptCall()
				go claimCall()
			}
			for i := 0; i < 2; i++ {
				r := <-out
				if r.err != nil {
					if r.which == "receipt" {
						t.Fatal(r.err)
					}
					requireCode(t, r.err, "P2704")
				}
			}
			state := value(t, c, "SELECT state FROM integration.connection_attempt WHERE attempt_id=$1", id)
			claims := value(t, c, "SELECT count(*)::text FROM integration.link_repair_claim")
			if (state == "exchange_claimed" && claims != "1") || (state != "exchange_claimed" && claims != "0") || (state != "exchange_claimed" && state != "abandoned" && state != "link_ready") {
				t.Fatal("race escaped one durable outcome")
			}
			if which == "receipt-cancel" && state != "abandoned" {
				t.Fatal("receipt overrode cancellation")
			}
			if value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "0" || value(t, c, "SELECT count(*)::text FROM integration.connection_attempt") != "2" || value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
				t.Fatal("race resumed, replaced or posted automatically")
			}
		})
	}
}

// TestHostedRepairSQLOriginalOwner cannot silently transfer source consent to another fully capable member.
func TestHostedRepairSQLOriginalOwner(t *testing.T) {
	c, db, _, _ := connected(t)
	// Independent fixture grants deliberately satisfy generic membership/source checks.
	exec(t, c, "INSERT INTO ledger.membership VALUES($1,$2,'connect',NULL)", entity, reader)
	exec(t, c, "INSERT INTO banking.source_permission SELECT entity_id,book_id,source_account_id,$1,'bind',NULL FROM banking.source_account", reader)
	ids := object(t, `{"ids":`+value(t, c, "SELECT jsonb_agg(stream_id ORDER BY stream_id)::text FROM integration.sync_stream")+`}`)["ids"]
	_, e := invoke(t, db, "synthetic_runtime", "SELECT integration.begin_connection($1,$2,$3,'delegated-repair',$4)::text", reader, entity, book, raw(map[string]any{"mode": "reconnect", "sync_item_id": value(t, c, "SELECT sync_item_id::text FROM integration.sync_item"), "expected_generation": value(t, c, "SELECT generation::text FROM integration.sync_item"), "approved_stream_ids": ids}))
	requireCode(t, e, "P2701")
	if value(t, c, "SELECT consent_principal::text FROM integration.sync_item") != actor || value(t, c, "SELECT count(*)::text FROM integration.connection_attempt") != "1" {
		t.Fatal("generic grants silently transferred source ownership")
	}
	requireNoRepair(t, c)
}

// TestHostedRepairSQLDuplicateRace forces both snapshots to reach the same owned book before release.
func TestHostedRepairSQLDuplicateRace(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	a := repairStart(t, c, db, "repair")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	blocker := connect(t, db, "postgres")
	defer blocker.Close(context.Background())
	tx, e := blocker.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SELECT 1 FROM ledger.book WHERE entity_id=$1 AND book_id=$2 FOR UPDATE", entity, book); e != nil {
		t.Fatal(e)
	}
	wire := repairWire(a, "owned", "success")
	type outcome struct {
		raw string
		err error
	}
	results := make(chan outcome, 2)
	for i := 0; i < 2; i++ {
		go func() {
			r, e := invoke(t, db, "synthetic_webhook", "SELECT integration.accept_verified_link_session('sandbox',$1,$2)::text", repairHash("proof"), wire)
			results <- outcome{r, e}
		}()
	}
	// Bounded synchronization on real lock waits, not an arbitrary ordering delay.
	for {
		if value(t, c, "SELECT count(*)::text FROM pg_stat_activity WHERE datname=current_database() AND usename='synthetic_webhook' AND wait_event_type='Lock'") == "2" {
			break
		}
		if ctx.Err() != nil {
			t.Fatal("duplicate deliveries did not reach lock barrier")
		}
		time.Sleep(time.Millisecond)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		state := object(t, r.raw)["state"]
		if state != "accepted" && state != "duplicate" {
			t.Fatal("duplicate delivery did not resolve")
		}
	}
	if value(t, c, "SELECT count(*)::text FROM integration.link_session_receipt") != "1" {
		t.Fatal("duplicate receipt stored twice")
	}
	requireNoRepair(t, c)
}

// TestHostedRepairSQLClaimedExpiryRecovery inspects the same lost claim before any manifest was stored.
func TestHostedRepairSQLClaimedExpiryRecovery(t *testing.T) {
	c, db, _, _ := connected(t)
	webhookConfig(t, db)
	a := repairStart(t, c, db, "repair")
	repairDelivery(t, db, "proof", repairWire(a, "owned", "success"))
	a = claim(t, db, a)
	id, claimID := a["attempt_id"], a["claim_id"]
	exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", id)
	recovered := object(t, call(t, db, "synthetic_runtime", "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, id))
	if recovered["claim_id"] != claimID || recovered["manifest"] != nil || recovered["execute_provider"] != false {
		t.Fatal("lost claim was replaced or could execute again")
	}
	a = recordManifest(t, db, recovered, verified("SyntheticItem", "AccountA", "AccountB"))
	confirm(t, db, a)
	confirm(t, db, a)
	if value(t, c, "SELECT count(*)::text FROM integration.link_repair_claim") != "1" || value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "2" || value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, c, "SELECT credential_version::text FROM integration.sync_item") != "1" {
		t.Fatal("late reconciliation duplicated or widened an effect")
	}
}
