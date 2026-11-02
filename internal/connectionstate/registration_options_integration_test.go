//go:build integration

package connectionstate

import (
	"testing"
	"time"
)

// registrationOptions invokes the actual restricted read surface, never trial preparation.
func registrationOptions(t *testing.T, db string, a map[string]any) map[string]any {
	t.Helper()
	return object(t, call(t, db, productionAPI, "SELECT integration.registration_options($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]))
}

// optionAccount uses fixed fixture identity to inspect a closed advisory result.
func optionAccount(t *testing.T, options map[string]any, id string) map[string]any {
	t.Helper()
	for _, row := range options["accounts"].([]any) {
		if row.(map[string]any)["account_id"] == id {
			return row.(map[string]any)
		}
	}
	t.Fatal("missing synthetic option")
	return nil
}

// TestRegistrationOptionsNoEffects proves repeated reads cannot save a command or change financial state.
func TestRegistrationOptionsNoEffects(t *testing.T) {
	c, db, a := registrationFixture(t)
	const snapshot = `SELECT jsonb_build_array((SELECT count(*) FROM ledger.interactive_command),(SELECT count(*) FROM audit.command_action),(SELECT count(*) FROM ledger.account),(SELECT count(*) FROM banking.source_account),(SELECT count(*) FROM integration.sync_stream),(SELECT count(*) FROM integration.connection_receipt),(SELECT count(*) FROM ledger.journal_entry))::text`
	before := value(t, c, snapshot)
	first := registrationOptions(t, db, a)
	second := registrationOptions(t, db, a)
	if raw(first) != raw(second) || value(t, c, snapshot) != before || first["max_sources"] != "1" || first["remaining_sources"] != "1" || first["can_register"] != true || first["history_request_mode"] != "provider_default" || first["history_guaranteed"] != false {
		t.Fatal("read mutated state or misstated capacity/history")
	}
	if optionAccount(t, first, "AccountA")["eligible"] != true || optionAccount(t, first, "AccountB")["eligible"] != true || optionAccount(t, first, "AccountC")["eligible"] != true || optionAccount(t, first, "AccountC")["iso_currency_code"] != "JPY" || optionAccount(t, first, "Unsupported")["reason"] != "unsupported_account" || optionAccount(t, first, "NoCurrency")["reason"] != "currency_unavailable" {
		t.Fatal("SQL eligibility reasons changed")
	}
	plan := registrationPrepare(t, db, a, "from-options", "AccountA")
	binding := plan["body"].(map[string]any)["bindings"].([]any)[0].(map[string]any)
	if binding["new_ledger_account"].(map[string]any)["category"] != "asset" || binding["new_ledger_account"].(map[string]any)["normal_side"] != "debit" || binding["new_ledger_account"].(map[string]any)["code"] != "1004" || binding["currency_version_id"] != currency {
		t.Fatal("factored automatic preparation profile drifted")
	}
	if registrationOptions(t, db, a)["can_register"] != true {
		t.Fatal("advisory read mutated or competed with pending gate")
	}
}

// TestRegistrationOptionsExpiry reports expired setup/consent without making new scope or silently renewing it.
func TestRegistrationOptionsExpiry(t *testing.T) {
	for _, which := range []string{"attempt", "consent"} {
		t.Run(which, func(t *testing.T) {
			c, db, a := registrationFixture(t)
			want := "attempt_expired"
			if which == "attempt" {
				exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()+interval '100 milliseconds'")
			} else {
				exec(t, c, "UPDATE integration.sync_item SET consent_expires_at=clock_timestamp()+interval '100 milliseconds',generation=generation+1; UPDATE integration.connection_attempt SET item_generation=item_generation+1")
				want = "consent_expired"
			}
			time.Sleep(150 * time.Millisecond)
			options := registrationOptions(t, db, a)
			if options["blocked_reason"] != want || options["can_register"] != false || optionAccount(t, options, "AccountA")["reason"] != want {
				t.Fatal("expired authority silently renewed")
			}
		})
	}
}

// TestRegistrationOptionsRegisteredRecovery preserves registered identity and paused capacity across append.
func TestRegistrationOptionsRegisteredRecovery(t *testing.T) {
	_, db, a := registrationFixture(t)
	plan := registrationPrepare(t, db, a, "first", "AccountA")
	out := registrationTransition(t, db, plan, "execute")
	stream := out["result"].(map[string]any)["streams"].([]any)[0].(map[string]any)
	registrationTransition(t, db, plan, "acknowledge")
	options := registrationOptions(t, db, a)
	registered := optionAccount(t, options, "AccountA")
	if registered["registered"] != true || registered["reason"] != "already_registered" || registered["eligible"] != false || registered["source_account_id"] == nil || options["can_register"] != false || optionAccount(t, options, "AccountB")["reason"] != "capacity_exhausted" {
		t.Fatal("existing registration or finite cap lost")
	}
	call(t, db, productionAPI, "SELECT integration.mutate_connection_stream($1,$2,$3,$4,'pause-options',$5)::text", actor, entity, book, stream["stream_id"], raw(map[string]any{"action": "pause", "expected_item_generation": "2", "expected_binding_revision": "1", "expected_stream_revision": "1"}))
	if registrationOptions(t, db, a)["remaining_sources"] != "0" {
		t.Fatal("paused source bypassed capacity")
	}
	registrationLimit(t, db, "opt-in", "2")
	if optionAccount(t, registrationOptions(t, db, a), "AccountB")["eligible"] != true {
		t.Fatal("reviewed fixture limit not reflected")
	}
	call(t, db, productionAPI, "SELECT integration.disconnect_stream($1,$2,$3,$4,'disconnect-options',1)::text", actor, entity, book, stream["stream_id"])
	if optionAccount(t, registrationOptions(t, db, a), "AccountA")["reason"] != "source_disconnected" {
		t.Fatal("disconnected lineage offered as new registration")
	}
}

// TestRegistrationOptionsAuthority rejects foreign actors, wrong runtime logins and revoked scope without payload.
func TestRegistrationOptionsAuthority(t *testing.T) {
	c, db, a := registrationFixture(t)
	_, err := invoke(t, db, productionAPI, "SELECT integration.registration_options($1,$2,$3,$4)::text", other, entity, book, a["attempt_id"])
	requireCode(t, err, "P2701")
	_, err = invoke(t, db, "synthetic_runtime", "SELECT integration.registration_options($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, err, "P2701")
	if value(t, c, "SELECT has_function_privilege('bookkeeping_sync','integration.registration_options(uuid,uuid,uuid,uuid)','EXECUTE')::text") != "false" || value(t, c, "SELECT has_function_privilege('bookkeeping_api','integration.registration_profile(jsonb)','EXECUTE')::text") != "false" {
		t.Fatal("read/helper privilege widened")
	}
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'revoke-options',$4)::text", actor, entity, book, raw(productionScope(false)))
	_, err = invoke(t, db, productionAPI, "SELECT integration.registration_options($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"])
	requireCode(t, err, "P2701")
}

// TestRegistrationOptionsPolicy reports missing current policy without guessing date/currency.
func TestRegistrationOptionsPolicy(t *testing.T) {
	_, db, a := registrationFixtureWithPolicy(t, false)
	options := registrationOptions(t, db, a)
	if options["binding_date"] != nil || options["blocked_reason"] != "policy_unavailable" || options["can_register"] != false || optionAccount(t, options, "AccountA")["reason"] != "policy_unavailable" {
		t.Fatal("missing policy silently defaulted")
	}
}
