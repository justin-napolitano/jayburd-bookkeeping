//go:build integration

package connectionstate

import (
	"context"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

const registrationRequest = "00000000-0000-4000-8000-000000000080"

// registrationFixture records a verified fictional multi-account manifest before any binding.
func registrationFixture(t *testing.T) (*pgx.Conn, string, map[string]any) {
	t.Helper()
	c, db := productionSetup(t)
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,'00000000-0000-4000-8000-000000000090','2020-01-01',NULL,$3,'accrual','UTC',2,'half_even','synthetic',$4,clock_timestamp())`, entity, book, currency, actor)
	a := productionBegin(t, db)
	a = object(t, call(t, db, productionAPI, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "expires_at": "2099-01-01T00:00:00Z"})))
	a = object(t, call(t, db, productionAPI, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["nonce"], raw(map[string]any{"link_token_sha256": strings.Repeat("a", 64), "public_token_sha256": strings.Repeat("b", 64)})))
	rows := []any{}
	for _, row := range [][4]string{{"AccountA", "depository", "checking", "USD"}, {"AccountB", "depository", "savings", "USD"}, {"AccountC", "credit", "credit card", "JPY"}, {"Unsupported", "investment", "brokerage", "USD"}, {"NoCurrency", "depository", "checking", "ZZZ"}} {
		rows = append(rows, map[string]any{"account_id": row[0], "type": row[1], "subtype": row[2], "iso_currency_code": row[3]})
	}
	a = object(t, call(t, db, productionAPI, "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text", actor, entity, book, a["attempt_id"], a["claim_id"], raw(map[string]any{"item_id": "FictionalItem", "accounts": rows, "consent_expires_at": "2099-01-01T00:00:00Z"})))
	a = object(t, call(t, db, productionAPI, "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], a["claim_id"]))
	return c, db, a
}

// registrationPrepare sends only a selection, never client-generated account/currency proposals.
func registrationPrepare(t testing.TB, db string, a map[string]any, key string, ids ...string) map[string]any {
	t.Helper()
	return object(t, call(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,$5,$6,$7)::text", actor, entity, book, a["attempt_id"], key, registrationRequest, raw(map[string]any{"account_ids": ids})))
}

// registrationTransition uses the existing command gate and explicit terminal acknowledgement.
func registrationTransition(t testing.TB, db string, plan map[string]any, action string) map[string]any {
	t.Helper()
	return object(t, call(t, db, productionAPI, "SELECT ledger.transition_command($1,$2,$3,$4,'{}')::text", actor, plan["command_id"], registrationRequest, action))
}

// registrationLimit is the owner-only synthetic opt-in; it never configures a real scope.
func registrationLimit(t testing.TB, db, key, max string) string {
	t.Helper()
	return call(t, db, "synthetic_operator", "SELECT integration.configure_registration_limit($1,$2,$3,$4,$5)::text", actor, entity, book, key, raw(map[string]any{"max_sources": max}))
}

// TestRegistrationScopeLimit proves compatibility default1, explicit opt-in and immutable receipt replay.
func TestRegistrationScopeLimit(t *testing.T) {
	c, db, a := registrationFixture(t)
	_, err := invoke(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,'too-many',$5,$6)::text", actor, entity, book, a["attempt_id"], registrationRequest, raw(map[string]any{"account_ids": []string{"AccountA", "AccountB"}}))
	requireCode(t, err, "P2705")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "0" {
		t.Fatal("denied capacity created source")
	}
	receipt := registrationLimit(t, db, "allow-two", "2")
	plan := registrationPrepare(t, db, a, "two", "AccountB", "AccountA")
	body := plan["body"].(map[string]any)
	bindings := body["bindings"].([]any)
	if len(bindings) != 2 || bindings[0].(map[string]any)["new_ledger_account"].(map[string]any)["code"] != "1004" || bindings[1].(map[string]any)["new_ledger_account"].(map[string]any)["code"] != "1005" {
		t.Fatal("server did not propose distinct available codes")
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.account") != "5" {
		t.Fatal("prepare created accounts")
	}
	out := registrationTransition(t, db, plan, "execute")
	if out["state"] != "committed" || value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "2" {
		t.Fatal("scoped two-account registration missing")
	}
	registrationTransition(t, db, plan, "acknowledge")
	registrationLimit(t, db, "allow-three", "3")
	if registrationLimit(t, db, "allow-two", "2") != receipt || value(t, c, "SELECT maximum::text FROM integration.registration_limit") != "3" {
		t.Fatal("old receipt reset limit")
	}
	for _, login := range []string{productionAPI, productionWorker, "synthetic_runtime", "synthetic_sync"} {
		_, err = invoke(t, db, login, "SELECT integration.configure_registration_limit($1,$2,$3,'forbidden','{\"max_sources\":\"25\"}')::text", actor, entity, book)
		requireCode(t, err, "42501")
	}
	for _, q := range []string{"SELECT maximum::text FROM integration.registration_limit", "SELECT integration.execute_registration(NULL,NULL,NULL,NULL,'{}')::text", "SELECT integration.apply_registration_bindings(NULL,NULL,NULL,NULL,'{}','[]')::text"} {
		_, err = invoke(t, db, productionAPI, q)
		requireCode(t, err, "42501")
	}
	_, err = invoke(t, db, productionAPI, "SELECT ledger.register_command($1,'forged',$2,$3)::text", actor, registrationRequest, raw(map[string]any{"operation": "register_accounts", "entity_id": entity, "book_id": book, "body": map[string]any{}}))
	requireCode(t, err, "P2100")
	_, err = invoke(t, db, "synthetic_operator", "DELETE FROM integration.registration_limit RETURNING maximum::text")
	requireCode(t, err, "P2702")
	_, err = invoke(t, db, "synthetic_operator", "UPDATE integration.registration_limit SET maximum=2 RETURNING maximum::text")
	requireCode(t, err, "P2702")
}

// TestRegistrationAppendPreservesExisting exercises source publication then preserves its exact lineage on append.
func TestRegistrationAppendPreservesExisting(t *testing.T) {
	c, db, a := registrationFixture(t)
	first := registrationPrepare(t, db, a, "checking", "AccountA")
	initial := registrationTransition(t, db, first, "execute")
	stream := initial["result"].(map[string]any)["streams"].([]any)[0].(map[string]any)
	publishProduction(t, db, stream)
	registrationTransition(t, db, first, "acknowledge")
	snapshot := `SELECT jsonb_build_object('item',(SELECT to_jsonb(i) FROM integration.sync_item i),'attempt',(SELECT to_jsonb(a) FROM integration.connection_attempt a),'stream',(SELECT to_jsonb(s) FROM integration.sync_stream s WHERE stream_id=$1),'source',(SELECT to_jsonb(s) FROM banking.source_account s WHERE source_account_id=$2),'observations',(SELECT jsonb_agg(to_jsonb(o)) FROM banking.source_observation o),'revisions',(SELECT jsonb_agg(to_jsonb(o)) FROM banking.observation_revision o),'review',(SELECT jsonb_agg(to_jsonb(r)) FROM banking.review_item r),'bind',(SELECT jsonb_agg(to_jsonb(r)) FROM integration.connection_receipt r WHERE operation='bind'))::text`
	before := value(t, c, snapshot, stream["stream_id"], stream["source_account_id"])
	registrationLimit(t, db, "allow-savings", "2")
	second := registrationPrepare(t, db, a, "savings", "AccountB")
	appended := registrationTransition(t, db, second, "execute")
	if appended["state"] != "committed" || value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "2" || value(t, c, snapshot, stream["stream_id"], stream["source_account_id"]) != before {
		t.Fatal("append changed existing lineage/cursor/receipt")
	}
	for q, want := range map[string]string{"SELECT count(*)::text FROM banking.source_observation": "1", "SELECT provider_amount_lexeme FROM banking.observation_revision": "50.00", "SELECT count(*)::text FROM ledger.journal_entry": "0", "SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'": "0"} {
		if value(t, c, q) != want {
			t.Fatal("independent source/journal expectation changed")
		}
	}
}

// TestRegistrationPlanReplayAndUnknownCommit resolves a dropped execute reply and concurrent retries to one effect.
func TestRegistrationPlanReplayAndUnknownCommit(t *testing.T) {
	c, db, a := registrationFixture(t)
	plan := registrationPrepare(t, db, a, "checking", "AccountA")
	if raw(plan) != raw(registrationPrepare(t, db, a, "checking", "AccountA")) {
		t.Fatal("prepare retry rebuilt IDs")
	}
	var wg sync.WaitGroup
	results := make(chan string, 2)
	failures := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := invoke(t, db, productionAPI, "SELECT ledger.transition_command($1,$2,$3,'execute','{}')::text", actor, plan["command_id"], registrationRequest)
			results <- out
			failures <- err
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	first := ""
	for out := range results {
		if first != "" && first != out {
			t.Fatal("concurrent replay changed result")
		}
		first = out
	}
	recovered := object(t, call(t, db, productionAPI, "SELECT ledger.pending_command($1)::text", actor))["command"].(map[string]any)
	if recovered["command_id"] != plan["command_id"] || recovered["state"] != "committed" || value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "1" {
		t.Fatal("lost commit not recovered")
	}
	if raw(registrationTransition(t, db, plan, "execute")) != raw(recovered) {
		t.Fatal("execute replay changed receipt")
	}
	_, err := invoke(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,'checking',$5,$6)::text", actor, entity, book, a["attempt_id"], registrationRequest, raw(map[string]any{"account_ids": []string{"AccountB"}}))
	requireCode(t, err, "P2702")
	registrationTransition(t, db, plan, "acknowledge")
	again := registrationPrepare(t, db, a, "checking", "AccountA")
	if again["command_id"] != plan["command_id"] || again["acknowledged"] != true {
		t.Fatal("acknowledged registration replay created new work")
	}
}

// TestRegistrationPlanAtomicity rejects a chart collision without partial accounts, grants or activation.
func TestRegistrationPlanAtomicity(t *testing.T) {
	c, db, a := registrationFixture(t)
	registrationLimit(t, db, "two", "2")
	plan := registrationPrepare(t, db, a, "both", "AccountA", "AccountB")
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,'00000000-0000-4000-8000-000000000099','1005','Fictional competing edit','asset','debit',true,'2020-01-01',NULL)`, entity, book)
	_, err := invoke(t, db, productionAPI, "SELECT ledger.transition_command($1,$2,$3,'execute','{}')::text", actor, plan["command_id"], registrationRequest)
	requireCode(t, err, "P2703")
	for q, want := range map[string]string{"SELECT count(*)::text FROM ledger.account": "6", "SELECT count(*)::text FROM integration.sync_stream": "0", "SELECT count(*)::text FROM banking.source_account": "0", "SELECT count(*)::text FROM banking.source_permission": "0", "SELECT state FROM ledger.interactive_command": "prepared"} {
		if value(t, c, q) != want {
			t.Fatal("failed atomic registration leaked mutation")
		}
	}
	registrationTransition(t, db, plan, "cancel")
	registrationTransition(t, db, plan, "acknowledge")
	// Fail after one new source has been inserted, proving rollback beyond preflight denial.
	plan = registrationPrepare(t, db, a, "retry-after-chart", "AccountA", "AccountB")
	exec(t, c, `CREATE FUNCTION banking.synthetic_registration_fault() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$ BEGIN IF EXISTS(SELECT FROM banking.source_account) THEN RAISE unique_violation;END IF;RETURN NEW;END $$;CREATE TRIGGER synthetic_registration_fault BEFORE INSERT ON banking.source_account FOR EACH ROW EXECUTE FUNCTION banking.synthetic_registration_fault()`)
	_, err = invoke(t, db, productionAPI, "SELECT ledger.transition_command($1,$2,$3,'execute','{}')::text", actor, plan["command_id"], registrationRequest)
	requireCode(t, err, "P2702")
	for q, want := range map[string]string{"SELECT count(*)::text FROM ledger.account": "6", "SELECT count(*)::text FROM integration.sync_stream": "0", "SELECT count(*)::text FROM banking.source_account": "0", "SELECT count(*)::text FROM banking.source_permission": "0", "SELECT state FROM ledger.interactive_command WHERE acknowledged_at IS NULL": "prepared"} {
		if value(t, c, q) != want {
			t.Fatal("injected second-source failure left a partial registration")
		}
	}
}

// TestRegistrationPlanCurrentAuthority proves denial and payload-free recovery after current scope revocation.
func TestRegistrationPlanCurrentAuthority(t *testing.T) {
	c, db, a := registrationFixture(t)
	plan := registrationPrepare(t, db, a, "one", "AccountA")
	_, err := invoke(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,'another',$5,$6)::text", actor, entity, book, a["attempt_id"], registrationRequest, raw(map[string]any{"account_ids": []string{"AccountB"}}))
	requireCode(t, err, "P2112")
	_, err = invoke(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,'foreign',$5,$6)::text", other, entity, book, a["attempt_id"], registrationRequest, raw(map[string]any{"account_ids": []string{"AccountA"}}))
	requireCode(t, err, "P2701")
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'revoke',$4)::text", actor, entity, book, raw(productionScope(false)))
	_, err = invoke(t, db, productionAPI, "SELECT ledger.transition_command($1,$2,$3,'execute','{}')::text", actor, plan["command_id"], registrationRequest)
	requireCode(t, err, "P2102")
	cleared := object(t, call(t, db, productionAPI, "SELECT ledger.clear_inaccessible_command($1,$2,'{}')::text", actor, registrationRequest))
	if raw(cleared) != "{\"cleared\":true}" || value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "0" || value(t, c, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM ledger.interactive_command") != "cancelled:true" {
		t.Fatal("inaccessible recovery exposed or executed intent")
	}
}

// TestRegistrationCurrencyAndProfile checks missing/unsupported references and exact original-unit currency choices.
func TestRegistrationCurrencyAndProfile(t *testing.T) {
	c, db, a := registrationFixture(t)
	for _, ids := range [][]string{{"Unsupported"}, {"NoCurrency"}, {"Missing"}, {"AccountA", "AccountA"}} {
		_, err := invoke(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,'invalid',$5,$6)::text", actor, entity, book, a["attempt_id"], registrationRequest, raw(map[string]any{"account_ids": ids}))
		want := "P2704"
		if len(ids) > 1 {
			want = "P2700"
		}
		requireCode(t, err, want)
	}
	registrationLimit(t, db, "multicurrency", "2")
	plan := registrationPrepare(t, db, a, "currencies", "AccountA", "AccountC")
	body := plan["body"].(map[string]any)
	bindings := body["bindings"].([]any)
	if bindings[0].(map[string]any)["currency_version_id"] != currency || bindings[1].(map[string]any)["currency_version_id"] != "00000000-0000-4000-8000-000000000006" || bindings[1].(map[string]any)["new_ledger_account"].(map[string]any)["category"] != "liability" {
		t.Fatal("exact currency/profile proposal lost")
	}
	registrationTransition(t, db, plan, "execute")
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("registration posted journal")
	}
	down, err := fs.ReadFile(migrations.Files, "000036_selective_registration.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Exec(context.Background(), string(down))
	requireCode(t, err, "P2702")
	exec(t, c, "ROLLBACK")
}

// TestRegistrationPausedAndDisconnected keeps paused capacity and immutable disconnected identity fences.
func TestRegistrationPausedAndDisconnected(t *testing.T) {
	c, db, a := registrationFixture(t)
	plan := registrationPrepare(t, db, a, "one", "AccountA")
	out := registrationTransition(t, db, plan, "execute")
	stream := out["result"].(map[string]any)["streams"].([]any)[0].(map[string]any)
	registrationTransition(t, db, plan, "acknowledge")
	call(t, db, productionAPI, "SELECT integration.mutate_connection_stream($1,$2,$3,$4,'pause',$5)::text", actor, entity, book, stream["stream_id"], raw(map[string]any{"action": "pause", "expected_item_generation": "2", "expected_binding_revision": "1", "expected_stream_revision": "1"}))
	_, err := invoke(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,'still-full',$5,$6)::text", actor, entity, book, a["attempt_id"], registrationRequest, raw(map[string]any{"account_ids": []string{"AccountB"}}))
	requireCode(t, err, "P2705")
	call(t, db, productionAPI, "SELECT integration.disconnect_stream($1,$2,$3,$4,'disconnect',1)::text", actor, entity, book, stream["stream_id"])
	_, err = invoke(t, db, productionAPI, "SELECT integration.prepare_registration($1,$2,$3,$4,'duplicate-history',$5,$6)::text", actor, entity, book, a["attempt_id"], registrationRequest, raw(map[string]any{"account_ids": []string{"AccountA"}}))
	requireCode(t, err, "P2702")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "1" {
		t.Fatal("disconnected identity was duplicated")
	}
}

// TestRegistrationStalePins rechecks current limits/consent and the actual prepared expiry before new work.
func TestRegistrationStalePins(t *testing.T) {
	for _, change := range []string{"limit", "consent", "expiry"} {
		t.Run(change, func(t *testing.T) {
			c, db, a := registrationFixture(t)
			if change == "expiry" {
				exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()+interval '2 seconds'")
			}
			if change == "consent" {
				exec(t, c, "UPDATE integration.sync_item SET consent_expires_at=clock_timestamp()+interval '2 seconds',generation=generation+1; UPDATE integration.connection_attempt SET item_generation=item_generation+1")
			}
			plan := registrationPrepare(t, db, a, "one", "AccountA")
			switch change {
			case "limit":
				registrationLimit(t, db, "new-revision", "1")
			case "consent", "expiry":
				time.Sleep(2100 * time.Millisecond)
			}
			_, err := invoke(t, db, productionAPI, "SELECT ledger.transition_command($1,$2,$3,'execute','{}')::text", actor, plan["command_id"], registrationRequest)
			requireCode(t, err, "P2703")
			if value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "0" {
				t.Fatal("stale plan mutated source")
			}
			registrationTransition(t, db, plan, "cancel")
			registrationTransition(t, db, plan, "acknowledge")
		})
	}
}
