//go:build integration

package acquisition

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const autoPolicy = "00000000-0000-4000-8000-000000000091"
const autoExpense = "00000000-0000-4000-8000-000000000092"

// accountingSource sets explicit fictional financial authority independently of provider consent.
func accountingSource(t *testing.T, overlap bool) (*pgx.Conn, string, map[string]any) {
	t.Helper()
	c, db, stream := setup(t)
	if !overlap {
		// Fixture-only attestation: this fictional provider namespace has no overlapping history.
		exec(t, c, "BEGIN; ALTER TABLE banking.source_namespace DISABLE TRIGGER namespace_guard")
		exec(t, c, "UPDATE banking.source_namespace SET overlap_review_required=false WHERE namespace_id=$1", stream["namespace_id"])
		exec(t, c, "ALTER TABLE banking.source_namespace ENABLE TRIGGER namespace_guard; COMMIT")
	}
	exec(t, c, "INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL),($1,$2,'post',NULL)", entity, actor)
	exec(t, c, "INSERT INTO banking.source_permission VALUES($1,$2,$3,$4,'review',NULL)", entity, book, sourceID, actor)
	exec(t, c, "INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())", entity, book, autoPolicy, currency, actor)
	exec(t, c, "INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before)VALUES($1,$2,'00000000-0000-4000-8000-000000000093','2026-01-01','2026-02-01')", entity, book)
	exec(t, c, "INSERT INTO ledger.account VALUES($1,$2,$3,'6000','Fictional supplies','expense','debit',false,'2026-01-01',NULL)", entity, book, autoExpense)
	return c, db, stream
}

// accountingMapping is an explicit recurring-source policy, never a provider category inference.
func accountingMapping(t *testing.T, db, description string) {
	t.Helper()
	rule := map[string]any{"rule_id": "99000000-0000-4000-8000-000000000001", "name": "Fictional approved supplies", "source_account_id": sourceID, "direction": "outflow", "description": description, "offset_account_id": autoExpense, "treatment": "expense"}
	call(t, db, "synthetic_runtime", "SELECT banking.set_automatic_accounting($1,$2,$3,$4)::text", actor, entity, book, raw(map[string]any{"key": "approved-policy", "expected_revision": "0", "enabled": true, "book_policy_id": autoPolicy, "effective_from": "2026-01-01", "effective_to": "2026-02-01", "rules": []any{rule}}))
}

// accountingTurn exercises the real restricted prepare/execute/machine-observe sequence.
func accountingTurn(t *testing.T, db string) map[string]any {
	t.Helper()
	next := object(t, call(t, db, "synthetic_runtime", "SELECT banking.prepare_automatic_accounting($1,$2,$3)::text", actor, entity, book))
	if next["state"] == "idle" || next["state"] == "blocked" {
		return next
	}
	id := next["operation"].(map[string]any)["operation_id"].(string)
	if next["state"] == "prepared" {
		next = object(t, call(t, db, "synthetic_runtime", "SELECT banking.execute_automatic_accounting($1,$2,$3,$4)::text", actor, entity, book, id))
	}
	call(t, db, "synthetic_runtime", "SELECT banking.observe_automatic_accounting($1,$2,$3,$4)::text", actor, entity, book, id)
	return next
}

// TestAutomaticAccountingProviderCorrection keeps the posted50 while retained evidence changes to55.
func TestAutomaticAccountingProviderCorrection(t *testing.T) {
	c, db, stream := accountingSource(t, false)
	scope := Scope{entity, book, stream["stream_id"].(string)}
	n := 0
	runner, _, _ := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
		providerRequest(t, r)
		n++
		if n == 1 {
			w.Write(lifecyclePage("C1", false, []json.RawMessage{lifecycleRow("Supplies", "50.00", "2026-01-06", false, nil)}, nil, nil))
		} else {
			w.Write(lifecyclePage("C2", false, nil, []json.RawMessage{lifecycleRow("Supplies", "55.00", "2026-01-06", false, nil)}, nil))
		}
	})
	if _, e := runner.Run(context.Background(), scope, claimID); e != nil {
		t.Fatal(e)
	}
	accountingMapping(t, db, "Fictional Supplies")
	accountingTurn(t, db)
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
		t.Fatal("initial50 not posted")
	}
	original := value(t, c, "SELECT result::text FROM banking.review_posting")
	call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'fictional-correction')::text", entity, book, scope.StreamID)
	if _, e := runner.Run(context.Background(), scope, "00000000-0000-4000-8000-000000000085"); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		accountingTurn(t, db)
	}
	status := call(t, db, "synthetic_runtime", "SELECT banking.automatic_accounting_status($1,$2,$3)::text", actor, entity, book)
	if !strings.Contains(status, "source_correction_required") {
		t.Fatal("changed55 lacks correction hold")
	}
	for q, want := range map[string]string{
		"SELECT count(*)::text FROM ledger.journal_entry":                                                   "1",
		"SELECT count(*)::text FROM banking.review_posting":                                                 "1",
		"SELECT count(*)::text FROM integration.operation_receipt":                                          "1",
		"SELECT count(*)::text FROM integration.outbox_message":                                             "1",
		"SELECT result::text FROM banking.review_posting":                                                   original,
		"SELECT string_agg(provider_amount_lexeme,'|' ORDER BY revision) FROM banking.observation_revision": "50.00|55.00",
		"SELECT trim_scale(sum(functional_amount) FILTER(WHERE side='debit'))::text||'|'||trim_scale(sum(functional_amount) FILTER(WHERE side='credit'))::text FROM ledger.journal_line": "50|50",
	} {
		if value(t, c, q) != want {
			t.Fatal("provider correction changed immutable50", q)
		}
	}
}

// TestAutomaticAccountingProviderHolds publishes real synthetic pending/zero/precision/overlap evidence.
// Migration29 already marks its first three cases correction_hold; automatic accounting preserves that hold.
func TestAutomaticAccountingProviderHolds(t *testing.T) {
	for _, tc := range []struct {
		name, amount, reason string
		pending, overlap     bool
	}{
		{"pending", "9.50", "source_correction_required", true, false}, {"zero", "0.00", "source_correction_required", false, false}, {"precision", "1.001", "source_correction_required", false, false}, {"overlap", "9.50", "overlap_review_required", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, db, stream := accountingSource(t, tc.overlap)
			runner, _, _ := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
				providerRequest(t, r)
				w.Write(lifecyclePage("C1", false, []json.RawMessage{lifecycleRow("Held", tc.amount, "2026-01-06", tc.pending, nil)}, nil, nil))
			})
			if _, e := runner.Run(context.Background(), Scope{entity, book, stream["stream_id"].(string)}, claimID); e != nil {
				t.Fatal(e)
			}
			accountingMapping(t, db, "Fictional Held")
			wantEvidence := map[string]string{"pending": "pending|outflow|9.50|2026-01-06", "zero": "booked|zero|0.00|2026-01-06", "precision": "booked|outflow|1.001|2026-01-06", "overlap": "booked|outflow|9.50|2026-01-06"}
			if value(t, c, "SELECT source_state||'|'||direction||'|'||provider_amount_lexeme||'|'||booked_date::text FROM banking.observation_revision") != wantEvidence[tc.name] {
				t.Fatal("provider evidence differs from literal hold oracle", tc.name)
			}
			before := value(t, c, "SELECT row_to_json(q)::text FROM banking.observation_revision q")
			for i := 0; i < 3; i++ {
				accountingTurn(t, db)
			}
			status := object(t, call(t, db, "synthetic_runtime", "SELECT banking.automatic_accounting_status($1,$2,$3)::text", actor, entity, book))
			if status["attention_count"] != "1" || status["pending_count"] != "0" {
				t.Fatal("missing fixed held attention", tc.name, status)
			}
			if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "1" || value(t, c, "SELECT count(*)::text FROM banking.review_item") != "1" || value(t, c, "SELECT count(*)::text FROM banking.automatic_accounting_operation WHERE state='held' AND reason=$1", tc.reason) != "1" {
				t.Fatal("source silently skipped expected hold", tc.name)
			}
			for _, table := range []string{"ledger.journal_entry", "banking.review_posting", "integration.outbox_message"} {
				if value(t, c, "SELECT count(*)::text FROM "+table) != "0" {
					t.Fatal("ineligible source posted", tc.name)
				}
			}
			if value(t, c, "SELECT row_to_json(q)::text FROM banking.observation_revision q") != before {
				t.Fatal("hold changed original evidence")
			}
		})
	}
}
