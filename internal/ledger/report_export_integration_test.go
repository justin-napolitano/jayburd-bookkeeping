//go:build integration

package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// csvCommand requires a pinned revision from the current committed SQL book state.
func csvCommand(t testing.TB, revision string, filters map[string]string) Command {
	t.Helper()
	filters["snapshot_revision"] = revision
	raw, err := json.Marshal(filters)
	if err != nil {
		t.Fatal(err)
	}
	return Command{Actor: actor, Entity: entity, Book: book, Body: string(raw)}
}

// csvAmount finds one exact SQL-derived measure without spreadsheet numeric coercion.
func csvAmount(t testing.TB, rows []map[string]string, kind, measure string) (string, string) {
	t.Helper()
	for _, row := range rows {
		if row["row_kind"] == kind && row["measure"] == measure {
			return row["current_sign"], row["current_magnitude"]
		}
	}
	t.Fatalf("missing %s %s", kind, measure)
	return "", ""
}

// TestReportCSVGoldenSQL proves reviewed fixed financial expectations across actual report SQL.
func TestReportCSVGoldenSQL(t *testing.T) {
	c, s, _ := fixtureDB(t, "csv_golden")
	sql(t, c, `INSERT INTO ledger.account VALUES($1,$2,'00000000-0000-4000-8000-000000000013','@hostile','=hostile()','asset','debit',false,'2026-01-01',NULL)`, entity, book)
	seedReportMonth(t, c, s)
	revision := scalar(t, c, "SELECT revision::text FROM ledger.book")
	before := ledgerProofSnapshot(t, c)
	for _, tc := range []struct {
		kind              string
		filters           map[string]string
		measure, expected string
	}{
		{"trial-balance", map[string]string{"through_date": "2026-01-31", "compare_through_date": "2026-01-01"}, "total_debits", "1300"},
		{"profit-and-loss", map[string]string{"starts_on": "2026-01-01", "ends_before": "2026-02-01", "compare_starts_on": "2026-02-01", "compare_ends_before": "2026-03-01"}, "net_income", "70"},
		{"balance-sheet", map[string]string{"through_date": "2026-01-31"}, "assets", "1145"},
		{"general-ledger", map[string]string{"starts_on": "2026-02-01", "ends_before": "2026-03-01", "account_id": accountA}, "opening", "1045"},
	} {
		tc.filters["report"] = tc.kind
		r, err := s.FinancialReportCSV(context.Background(), csvCommand(t, revision, tc.filters))
		if err != nil {
			t.Fatal(tc.kind, err)
		}
		rows := parseExport(t, r.Body)
		sign, value := csvAmount(t, rows, "summary", tc.measure)
		if sign != "positive" || value != tc.expected {
			t.Fatal(tc.kind, sign, value)
		}
		if strings.Contains(string(r.Body), "hostile") {
			t.Fatal("arbitrary label escaped omission boundary")
		}
		if tc.kind == "general-ledger" {
			if len(rows) != 4 {
				t.Fatal("empty period created postings")
			}
			_, close := csvAmount(t, rows, "summary", "closing")
			if close != "1045" {
				t.Fatal(close)
			}
		}
		if tc.kind == "profit-and-loss" {
			for _, row := range rows {
				if row["row_kind"] == "summary" && row["measure"] == "net_income" && (row["prior_sign"] != "zero" || row["prior_magnitude"] != "0" || row["difference_magnitude"] != "70") {
					t.Fatal("SQL comparison mismatch", row)
				}
			}
		}
	}
	requireLedgerUnchanged(t, c, before)
}

// TestReportCSVAllPagesSQL exports 101 actual committed journals, not only the first page.
func TestReportCSVAllPagesSQL(t *testing.T) {
	c, s, _ := fixtureDB(t, "csv_pages")
	for i := 0; i < 101; i++ {
		cmd := change(command(fmt.Sprintf("csv-cent-%d", i)), func(v map[string]any) {
			v["expected_revision"] = fmt.Sprint(i)
			for _, raw := range v["lines"].([]any) {
				line := raw.(map[string]any)
				line["transaction_amount"] = "0.01"
				line["functional_amount"] = "0.01"
			}
		})
		if _, err := s.Post(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}
	before := ledgerProofSnapshot(t, c)
	cmd := csvCommand(t, "101", map[string]string{"report": "general-ledger", "starts_on": "2026-01-01", "ends_before": "2026-02-01", "account_id": accountA})
	r, err := s.FinancialReportCSV(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	rows := parseExport(t, r.Body)
	if len(rows) != 509 {
		t.Fatal("expected 101*5 posting measures+4 totals", len(rows))
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row["row_kind"] == "posting" && row["measure"] == "transaction_amount" {
			if seen[row["row_id"]] || row["current_magnitude"] != "0.01" || row["current_sign"] != "positive" {
				t.Fatal("duplicate/inexact cent")
			}
			seen[row["row_id"]] = true
		}
	}
	if len(seen) != 101 {
		t.Fatal("missing journals")
	}
	sign, value := csvAmount(t, rows, "summary", "closing")
	if sign != "positive" || value != "1.01" {
		t.Fatal(sign, value)
	}
	requireLedgerUnchanged(t, c, before)
}

// TestReportCSVAuthoritySQL uses real committed revocation and posting between export reads.
func TestReportCSVAuthoritySQL(t *testing.T) {
	for _, scenario := range []string{"repeat_revoked", "during_revoked", "during_post"} {
		t.Run(scenario, func(t *testing.T) {
			c, s, _ := fixtureDB(t, "csv_"+scenario)
			cmd := csvCommand(t, "0", map[string]string{"report": "trial-balance", "through_date": "2026-01-31"})
			if _, err := s.FinancialReportCSV(context.Background(), cmd); err != nil {
				t.Fatal(err)
			}
			if scenario == "repeat_revoked" {
				sql(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE permission='read'")
			}
			calls := 0
			read := func(ctx context.Context, q Command) (Result, error) {
				calls++
				r, err := s.FinancialReport(ctx, q)
				if calls == 1 && err == nil {
					switch scenario {
					case "during_revoked":
						sql(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE permission='read'")
					case "during_post":
						if _, err := s.Post(context.Background(), command("csv-change")); err != nil {
							t.Fatal(err)
						}
					}
				}
				return r, err
			}
			result, err := exportReport(context.Background(), cmd, read, 10000, 10*1024*1024)
			var f *Failure
			expected := "permission_denied"
			if scenario == "during_post" {
				expected = "report_snapshot_changed"
			}
			if !errors.As(err, &f) || f.Code != expected || len(result.Body) != 0 {
				t.Fatal("authority bypass/partial body", err)
			}
		})
	}
}

// TestReportCSVFXReversalSQL retains original amounts/rates and correction linkage.
func TestReportCSVFXReversalSQL(t *testing.T) {
	_, s, _ := fixtureDB(t, "csv_fx")
	cmd := change(command("csv-spot"), func(v map[string]any) {
		for _, raw := range v["lines"].([]any) {
			line := raw.(map[string]any)
			line["transaction_currency"] = "EUR"
			line["transaction_amount"] = "10"
			line["functional_amount"] = "11"
			line["valuation_basis"] = "spot"
			line["rate_observation_id"] = rateID
		}
	})
	posted, err := s.Post(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	id := postedID(t, posted)
	if _, err = s.Reverse(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Journal: id, Key: "csv-reverse", Request: requestID(), Body: `{"expected_revision":"1","accounting_date":"2026-02-01","reason":"Synthetic correction"}`}); err != nil {
		t.Fatal(err)
	}
	result, err := s.FinancialReportCSV(context.Background(), csvCommand(t, "2", map[string]string{"report": "general-ledger", "account_id": accountA, "starts_on": "2026-01-01", "ends_before": "2026-02-02"}))
	if err != nil {
		t.Fatal(err)
	}
	rows := parseExport(t, result.Body)
	original, reversal := false, false
	for _, row := range rows {
		if row["row_kind"] != "posting" {
			continue
		}
		if row["transaction_currency"] != "EUR" || row["rate_observation_id"] != rateID {
			t.Fatal("lost FX provenance")
		}
		if row["measure"] == "transaction_amount" && row["current_magnitude"] != "10" {
			t.Fatal("lost original units")
		}
		if row["measure"] == "functional_amount" && row["current_magnitude"] != "11" {
			t.Fatal("lost functional units")
		}
		if row["journal_kind"] == "manual" {
			original = true
		} else if row["journal_kind"] == "reversal" && row["reverses_journal_id"] == id {
			reversal = true
		}
	}
	if !original || !reversal {
		t.Fatal("lost journal provenance")
	}
	sign, value := csvAmount(t, rows, "summary", "closing")
	if sign != "zero" || value != "0" {
		t.Fatal(sign, value)
	}
}
