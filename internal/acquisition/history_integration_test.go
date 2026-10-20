//go:build integration

package acquisition

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestHistoricalCurrencyAcquisition publishes both date ranges without widening posting policy.
func TestHistoricalCurrencyAcquisition(t *testing.T) {
	owner, db, stream := setupCurrency(t, currency, true)
	// The legacy enrollment fixture always marks overlap. Model a newly created
	// source (the migration-22 binding outcome) before any observations exist.
	exec(t, owner, "BEGIN; ALTER TABLE banking.source_namespace DISABLE TRIGGER namespace_guard")
	exec(t, owner, "UPDATE banking.source_namespace SET overlap_review_required=false WHERE namespace_id=$1", stream["namespace_id"])
	exec(t, owner, "ALTER TABLE banking.source_namespace ENABLE TRIGGER namespace_guard; COMMIT")
	exec(t, owner, `INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL)`, entity, actor)
	exec(t, owner, `INSERT INTO banking.source_permission VALUES($1,$2,$3,$4,'review',NULL)`, entity, book, sourceID, actor)
	exec(t, owner, `INSERT INTO ledger.book_policy VALUES($1,$2,'00000000-0000-4000-8000-000000000091','2026-09-17',NULL,$3,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$4,clock_timestamp())`, entity, book, currency, actor)
	exec(t, owner, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000092','2026-09-17','2027-01-01')`, entity, book)
	s := Scope{entity, book, stream["stream_id"].(string)}
	runner, _, calls := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
		cursor := providerRequest(t, r)
		if cursor == nil {
			w.Write([]byte(strings.ReplaceAll(string(fixturePage("Txn-Old", "1.01", "USD", "C1", true)), "2026-01-02", "2026-09-16")))
		} else {
			w.Write([]byte(strings.ReplaceAll(string(fixturePage("Txn-New", "2.02", "USD", "C2", false)), "2026-01-02", "2026-10-19")))
		}
	})
	out, err := runner.Run(context.Background(), s, claimID)
	if err != nil || out.Status != "completed" || out.ChangedCount != "2" || *calls != 2 {
		t.Fatal("historical acquisition failed", out, err, *calls)
	}
	if _, err = runner.Run(context.Background(), s, claimID); err != nil || *calls != 2 {
		t.Fatal("completed history replay fetched again", err)
	}
	for q, want := range map[string]string{
		"SELECT count(*)::text FROM banking.source_observation":                              "2",
		"SELECT count(*)::text FROM ledger.journal_entry":                                    "0",
		"SELECT cursor_value FROM integration.sync_stream":                                   "C2",
		"SELECT count(DISTINCT currency_version_id)::text FROM banking.observation_revision": "2",
		"SELECT trim_scale(sum(amount))::text FROM banking.observation_revision":             "3.03",
	} {
		if got := value(t, owner, q); got != want {
			t.Fatal("historical facts or replay changed", got, want)
		}
	}
	oldReview := value(t, owner, `SELECT review_id::text FROM banking.review_item r JOIN banking.review_assignment a USING(entity_id,book_id,review_id) JOIN banking.source_observation o USING(entity_id,book_id,observation_id) WHERE o.external_entry_id='Txn-Old'`)
	newReview := value(t, owner, `SELECT review_id::text FROM banking.review_item r JOIN banking.review_assignment a USING(entity_id,book_id,review_id) JOIN banking.source_observation o USING(entity_id,book_id,observation_id) WHERE o.external_entry_id='Txn-New'`)
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.review_categories($1,$2,$3,$4)::text", actor, entity, book, oldReview)
	requireCode(t, err, "P2109")
	current := object(t, call(t, db, "synthetic_runtime", "SELECT banking.review_categories($1,$2,$3,$4)::text", actor, entity, book, newReview))
	if current["fx_required"] != false || current["accounting_date"] != "2026-10-19" {
		t.Fatal("current review mapping changed")
	}
	// A well-shaped draft for old history cannot bypass the unchanged policy-date gate.
	for _, tc := range []struct{ id, day string }{{oldReview, "2026-09-16"}, {newReview, "2026-10-19"}} {
		body := value(t, owner, `SELECT jsonb_build_object('expected_review_revision',r.review_revision::text,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'expected_book_revision',b.revision::text,'category_account_id','00000000-0000-4000-8000-000000000012','policy_id','00000000-0000-4000-8000-000000000091','accounting_date',$2::text,'description','Fictional history review')::text FROM banking.review_item r JOIN banking.source_account s USING(entity_id,book_id,source_account_id) JOIN ledger.book b USING(entity_id,book_id) WHERE review_id=$1`, tc.id, tc.day)
		_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,$5,$6)::text", actor, entity, book, tc.id, "history-review", body)
		if tc.id == oldReview {
			requireCode(t, err, "P2109")
		} else if err != nil {
			t.Fatal("in-policy draft unavailable", err)
		}
	}
	if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("acquisition or draft posted a journal")
	}
}

// TestSyncHistoricalCurrencyValidation exercises independent SQL identity/date/precision checks.
func TestSyncHistoricalCurrencyValidation(t *testing.T) {
	_, db, _ := setupCurrency(t, currency, true)
	for _, tc := range []struct{ code, version, date, amount string }{
		{"USD", "00000000-0000-4000-8000-000000000090", "2026-09-16", "1.01"},
		{"USD", currency, "2026-09-17", "1.01"},
		{"USD", currency, "2026-09-16", "1.01"},
		{"USD", "00000000-0000-4000-8000-000000000090", "2026-09-17", "1.01"},
		{"USD", "00000000-0000-4000-8000-000000000006", "2026-09-16", "1.01"},
		{"USD", "00000000-0000-4000-8000-000000000090", "2026-09-16", "1.001"},
	} {
		v := raw(map[string]any{"account_id": "Account-A", "transaction_id": "Txn-A", "pending": false, "pending_transaction_id": nil, "date": tc.date, "authorized_date": nil, "datetime": nil, "authorized_datetime": nil, "iso_currency_code": tc.code, "currency_version_id": tc.version, "amount_lexeme": tc.amount, "amount": tc.amount, "direction": "outflow", "eligible_for_review": true, "name": "Fictional history", "merchant_name": nil, "original_description": nil})
		_, err := invoke(t, db, "synthetic_operator", "SELECT integration.sync_row($1::jsonb,'Account-A',$2,false)::text", v, currency)
		valid := tc.date == "2026-09-16" && tc.version == "00000000-0000-4000-8000-000000000090" && tc.amount == "1.01" || tc.date == "2026-09-17" && tc.version == currency
		if valid && err != nil || !valid && err == nil {
			t.Fatal("SQL dated currency validation mismatch", err)
		}
	}
}

// TestSandboxUSDHistory proves owner-only exact replay without editing current reference provenance.
func TestSandboxUSDHistory(t *testing.T) {
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	db := "acq_history_reference"
	exec(t, root, "CREATE DATABASE "+db+" TEMPLATE acquisition_template")
	owner := connect(t, db, "postgres")
	defer owner.Close(context.Background())
	exec(t, owner, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2026-09-17','2027-01-01','SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff')`, currency)
	for i := 0; i < 2; i++ {
		call(t, db, "synthetic_operator", "SELECT reference.install_sandbox_usd_history()::text")
	}
	if value(t, owner, "SELECT count(*)::text FROM reference.currency") != "2" || value(t, owner, "SELECT valid_from::text FROM reference.currency WHERE currency_version_id=$1", currency) != "2026-09-17" {
		t.Fatal("reference history duplicated or rewritten")
	}
	_, err := invoke(t, db, "synthetic_sync", "SELECT reference.install_sandbox_usd_history()::text")
	requireCode(t, err, "42501")
	_, err = invoke(t, db, "synthetic_runtime", "SELECT reference.install_sandbox_usd_history()::text")
	requireCode(t, err, "42501")
	if _, err = owner.Exec(context.Background(), "UPDATE reference.currency SET minor_unit=3 WHERE currency_version_id=$1", currency); err == nil {
		t.Fatal("immutable current reference changed")
	}
	// An occupied installer ID cannot silently overwrite a different reference.
	conflictDB := "acq_history_reference_conflict"
	exec(t, root, "CREATE DATABASE "+conflictDB+" TEMPLATE acquisition_template")
	conflict := connect(t, conflictDB, "postgres")
	defer conflict.Close(context.Background())
	exec(t, conflict, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2026-09-17','2027-01-01','SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff'),('3a184659-cf9b-4b80-b140-3f7a7796cbbd','EUR','978',2,'2001-01-01',NULL,'synthetic-conflict')`, currency)
	_, err = invoke(t, conflictDB, "synthetic_operator", "SELECT reference.install_sandbox_usd_history()::text")
	requireCode(t, err, "P0001")
	if value(t, conflict, "SELECT alphabetic_code FROM reference.currency WHERE currency_version_id='3a184659-cf9b-4b80-b140-3f7a7796cbbd'") != "EUR" {
		t.Fatal("conflicting reference overwritten")
	}
}
