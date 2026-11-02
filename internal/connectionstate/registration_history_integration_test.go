//go:build integration

package connectionstate

import (
	"context"
	"io/fs"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

// TestRegistrationHistoricalAccountStart preserves actual binding day while making the ledger available under approved history.
func TestRegistrationHistoricalAccountStart(t *testing.T) {
	c, db, a := registrationFixture(t)
	// An earlier approved policy range is independent of the currently selected policy and of bank consent.
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,'00000000-0000-4000-8000-000000000091','2019-01-01','2020-01-01',$3,'accrual','UTC',2,'half_even','synthetic',$4,clock_timestamp())`, entity, book, currency, actor)
	plan := registrationPrepare(t, db, a, "history-start", "AccountA")
	body := plan["body"].(map[string]any)
	binding := body["bindings"].([]any)[0].(map[string]any)
	newAccount := binding["new_ledger_account"].(map[string]any)
	if newAccount["active_from"] != "2019-01-01" || binding["binding_date"] != body["binding_date"] || binding["binding_date"] == "2019-01-01" {
		t.Fatal("ledger boundary changed source binding date or ignored approved history")
	}
	registrationTransition(t, db, plan, "execute")
	if value(t, c, "SELECT active_from::text FROM ledger.account WHERE account_id=$1", binding["ledger_account_id"]) != "2019-01-01" || value(t, c, "SELECT binding_date::text FROM banking.source_account WHERE ledger_account_id=$1", binding["ledger_account_id"]) != binding["binding_date"] || value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("historical availability posted or changed source dating")
	}
	if value(t, c, "SELECT has_function_privilege('bookkeeping_api','integration.registration_account_start(uuid,uuid,date)','EXECUTE')::text") != "false" {
		t.Fatal("private history helper exposed")
	}
	down, err := fs.ReadFile(migrations.Files, "000038_registration_history_start.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Exec(context.Background(), string(down))
	requireCode(t, err, "P2702")
	exec(t, c, "ROLLBACK")
}

// TestRegistrationHistoricalStartRejectsArbitraryDate preserves strict date authority for legacy direct binding.
func TestRegistrationHistoricalStartRejectsArbitraryDate(t *testing.T) {
	c, db, a := registrationFixture(t)
	plan := registrationPrepare(t, db, a, "review-start", "AccountA")
	body := plan["body"].(map[string]any)
	bindings := body["bindings"].([]any)
	bindings[0].(map[string]any)["new_ledger_account"].(map[string]any)["active_from"] = "2018-01-01"
	_, err := invoke(t, db, productionAPI, "SELECT integration.bind_connection($1,$2,$3,$4,'arbitrary',$5)::text", actor, entity, book, a["attempt_id"], raw(map[string]any{"consent_grants": true, "bindings": bindings}))
	requireCode(t, err, "P2704")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "0" {
		t.Fatal("unreviewed backdate bound a source")
	}
}

// TestRegistrationHistoricalPolicyChange cannot silently widen the saved date after owner policy changes.
func TestRegistrationHistoricalPolicyChange(t *testing.T) {
	c, db, a := registrationFixture(t)
	plan := registrationPrepare(t, db, a, "before-policy", "AccountA")
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,'00000000-0000-4000-8000-000000000091','2019-01-01','2020-01-01',$3,'accrual','UTC',2,'half_even','synthetic',$4,clock_timestamp())`, entity, book, currency, actor)
	_, err := invoke(t, db, productionAPI, "SELECT ledger.transition_command($1,$2,$3,'execute','{}')::text", actor, plan["command_id"], registrationRequest)
	requireCode(t, err, "P2703")
	if value(t, c, "SELECT count(*)::text FROM integration.sync_stream") != "0" {
		t.Fatal("stale historical proposal changed source")
	}
}

// TestRegistrationHistoricalOldPlanCompatibility preserves a plan reviewed under37 after the forward38 upgrade.
func TestRegistrationHistoricalOldPlanCompatibility(t *testing.T) {
	c, db, a := registrationFixture(t)
	down, err := fs.ReadFile(migrations.Files, "000038_registration_history_start.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Exec(context.Background(), string(down)); err != nil {
		t.Fatal(err)
	}
	plan := registrationPrepare(t, db, a, "old-reviewed", "AccountA")
	binding := plan["body"].(map[string]any)["bindings"].([]any)[0].(map[string]any)
	if binding["new_ledger_account"].(map[string]any)["active_from"] != binding["binding_date"] {
		t.Fatal("37 fixture was not legacy proposal")
	}
	up, err := fs.ReadFile(migrations.Files, "000038_registration_history_start.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Exec(context.Background(), string(up)); err != nil {
		t.Fatal(err)
	}
	if raw(registrationPrepare(t, db, a, "old-reviewed", "AccountA")) != raw(plan) {
		t.Fatal("upgrade rewrote saved date or identity")
	}
	registrationTransition(t, db, plan, "execute")
	if value(t, c, "SELECT active_from::text FROM ledger.account WHERE account_id=$1", binding["ledger_account_id"]) != binding["binding_date"] {
		t.Fatal("upgrade silently backdated accepted legacy plan")
	}
}
