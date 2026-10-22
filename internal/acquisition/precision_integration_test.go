//go:build integration

package acquisition

import (
	"context"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

// TestSourcePrecisionHold proves exact mixed publication, replay and SQL-only drafting refusal.
func TestSourcePrecisionHold(t *testing.T) {
	owner, db, stream := setupCurrency(t, currency)
	// New fictional sources have no overlapping import to resolve.
	exec(t, owner, "BEGIN; ALTER TABLE banking.source_namespace DISABLE TRIGGER namespace_guard")
	exec(t, owner, "UPDATE banking.source_namespace SET overlap_review_required=false WHERE namespace_id=$1", stream["namespace_id"])
	exec(t, owner, "ALTER TABLE banking.source_namespace ENABLE TRIGGER namespace_guard; COMMIT")
	exec(t, owner, `INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL)`, entity, actor)
	exec(t, owner, `INSERT INTO banking.source_permission VALUES($1,$2,$3,$4,'review',NULL)`, entity, book, sourceID, actor)
	s := Scope{entity, book, stream["stream_id"].(string)}
	runner, _, calls := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
		if providerRequest(t, r) == nil {
			w.Write(fixturePage("Txn-Held", "-1.00100", "USD", "C1", true))
		} else {
			w.Write(fixturePage("Txn-Ordinary", "2.02", "USD", "C2", false))
		}
	})
	out, err := runner.Run(context.Background(), s, claimID)
	if err != nil || out.Status != "completed" || out.ChangedCount != "2" || *calls != 2 {
		t.Fatal("mixed exact publication failed", out, err)
	}
	if _, err = runner.Run(context.Background(), s, claimID); err != nil || *calls != 2 {
		t.Fatal("completed replay acquired again", err)
	}
	for q, want := range map[string]string{
		"SELECT count(*)::text FROM banking.source_observation":                                      "2",
		"SELECT count(*)::text FROM banking.review_item WHERE state='correction_hold'":               "1",
		"SELECT count(*)::text FROM banking.review_item WHERE state='needs_review'":                  "1",
		"SELECT trim_scale(sum(amount))::text FROM banking.observation_revision":                     "3.021",
		"SELECT provider_amount_lexeme FROM banking.observation_revision WHERE direction='inflow'":   "-1.00100",
		"SELECT trim_scale(amount)::text FROM banking.observation_revision WHERE direction='inflow'": "1.001",
		"SELECT count(*)::text FROM ledger.journal_entry":                                            "0",
		"SELECT cursor_value FROM integration.sync_stream":                                           "C2",
	} {
		if got := value(t, owner, q); got != want {
			t.Fatal("source precision or replay changed", got, want)
		}
	}
	id := value(t, owner, "SELECT review_id::text FROM banking.review_item WHERE state='correction_hold'")
	detail := call(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", actor, entity, book, id)
	if !strings.Contains(detail, "source_precision_review_required") || !strings.Contains(detail, `"amount": "1.001"`) {
		t.Fatal("precise source hold missing", detail)
	}
	body := value(t, owner, `SELECT jsonb_build_object('expected_review_revision',r.review_revision::text,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'expected_book_revision',b.revision::text,'category_account_id','00000000-0000-4000-8000-000000000012','policy_id','00000000-0000-4000-8000-000000000091','accounting_date','2026-01-02','description','Fictional precision hold')::text FROM banking.review_item r JOIN banking.source_account s USING(entity_id,book_id,source_account_id) JOIN ledger.book b USING(entity_id,book_id) WHERE review_id=$1`, id)
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'held-draft',$5)::text", actor, entity, book, id, body)
	requireCode(t, err, "P2502")
	if value(t, owner, "SELECT count(*)::text FROM banking.review_draft") != "0" {
		t.Fatal("held source drafted")
	}
	// Retained exact evidence must not be erased or rounded by rollback.
	down, err := fs.ReadFile(migrations.Files, "000029_plaid_source_precision.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = owner.Exec(context.Background(), string(down))
	requireCode(t, err, "P0001")
	exec(t, owner, "ROLLBACK")
}

// TestPrecisionEligibilityForgery independently rejects a caller's false readiness claim or altered quantity.
func TestPrecisionEligibilityForgery(t *testing.T) {
	_, db, _ := setupCurrency(t, currency)
	v := map[string]any{"account_id": "Account-A", "transaction_id": "Txn-A", "pending": false, "pending_transaction_id": nil, "date": "2026-01-02", "authorized_date": nil, "datetime": nil, "authorized_datetime": nil, "iso_currency_code": "USD", "currency_version_id": currency, "amount_lexeme": "-1.00100", "amount": "1.001", "direction": "inflow", "eligible_for_review": false, "name": "Fictional precision", "merchant_name": nil, "original_description": nil}
	if _, err := invoke(t, db, "synthetic_operator", "SELECT integration.sync_row($1::jsonb,'Account-A',$2,false)::text", raw(v), currency); err != nil {
		t.Fatal("exact held row rejected", err)
	}
	v["eligible_for_review"] = true
	_, err := invoke(t, db, "synthetic_operator", "SELECT integration.sync_row($1::jsonb,'Account-A',$2,false)::text", raw(v), currency)
	requireCode(t, err, "P2604")
	v["eligible_for_review"] = false
	v["amount"] = "1"
	_, err = invoke(t, db, "synthetic_operator", "SELECT integration.sync_row($1::jsonb,'Account-A',$2,false)::text", raw(v), currency)
	requireCode(t, err, "P2604")
}
