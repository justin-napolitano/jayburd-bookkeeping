package ledger

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const exportTestEntity = "00000000-0000-4000-8000-000000000002"
const exportTestBook = "00000000-0000-4000-8000-000000000003"
const exportTestAccount = "00000000-0000-4000-8000-000000000011"

// exportFixture is independent synthetic SQL output, including deliberately hostile labels.
func exportFixture() (Command, exportPage) {
	c := Command{Entity: exportTestEntity, Book: exportTestBook, Body: `{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"2","compare_through_date":"2026-01-01"}`}
	p := exportPage{Contract: "financial-reporting-sql-v1", Report: "trial-balance", Entity: c.Entity, Book: c.Book, Revision: "2", Generated: "2026-10-13T17:00:00.123456Z", Currency: "USD", CurrencyVersion: "00000000-0000-4000-8000-000000000005", MinorUnits: 2, Basis: "accrual", Timezone: "UTC", Policies: []string{"00000000-0000-4000-8000-000000000004"}, Filters: map[string]string{"report": "trial-balance", "through_date": "2026-01-31", "compare_through_date": "2026-01-01"}, Totals: map[string]string{"total_debits": "10.01", "total_credits": "10.01"}, Comparison: &exportComparison{Totals: map[string]string{"total_debits": "2", "total_credits": "2"}, Difference: map[string]string{"total_debits": "8.01", "total_credits": "8.01"}}, Cursor: json.RawMessage("null")}
	raw := `{"id":"account:` + exportTestAccount + `","account_id":"` + exportTestAccount + `","category":"asset","normal_side":"debit","account_code":"=malicious()","account_name":"\t@hostile","amounts":{"net":"-10.01","debit_balance":"0","credit_balance":"10.01"},"comparison":{"amounts":{"net":"-2","debit_balance":"0","credit_balance":"2"},"difference":{"net":"-8.01","debit_balance":"0","credit_balance":"8.01"}}}`
	var row map[string]json.RawMessage
	_ = json.Unmarshal([]byte(raw), &row)
	p.Rows = []map[string]json.RawMessage{row}
	return c, p
}

// exportResult models a fully committed JSON report response.
func exportResult(p exportPage) Result {
	if p.Rows == nil {
		p.Rows = []map[string]json.RawMessage{}
	}
	raw, _ := json.Marshal(p)
	return Result{Status: 200, Body: raw}
}

// parseExport requires the complete RFC-shaped file and indexes named fields independently.
func parseExport(t testing.TB, raw []byte) []map[string]string {
	t.Helper()
	rows, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
	if err != nil || len(rows) < 2 {
		t.Fatalf("CSV %v", err)
	}
	expected := "profile,report,layout,entity_id,book_id,snapshot_revision,functional_currency,functional_currency_version_id,minor_units,basis,timezone,generated_at,policy_ids,through_date,starts_on,ends_before,compare_through_date,compare_starts_on,compare_ends_before,selected_account_id,row_kind,row_id,account_id,category,normal_side,journal_id,line_number,accounting_date,posted_at,journal_kind,reverses_journal_id,policy_id,side,transaction_currency,transaction_currency_version_id,valuation_basis,rate_observation_id,measure,current_sign,current_magnitude,prior_sign,prior_magnitude,difference_sign,difference_magnitude"
	if strings.Join(rows[0], ",") != expected {
		t.Fatal("wire header changed")
	}
	if !strings.HasSuffix(string(raw), "\r\n") || strings.Contains(strings.ReplaceAll(string(raw), "\r\n", ""), "\n") {
		t.Fatal("non CRLF")
	}
	var out []map[string]string
	for _, row := range rows[1:] {
		if len(row) != 44 {
			t.Fatal("field count")
		}
		v := map[string]string{}
		for i, key := range rows[0] {
			v[key] = row[i]
		}
		out = append(out, v)
	}
	return out
}

// TestReportCSVReviewExport reconstructs fixed signed amounts and preserves prior deltas.
func TestReportCSVReviewExport(t *testing.T) {
	c, p := exportFixture()
	calls := 0
	result, err := exportReport(context.Background(), c, func(_ context.Context, cmd Command) (Result, error) {
		calls++
		if cmd.Body != c.Body {
			t.Fatal("unexpected single-page query")
		}
		return exportResult(p), nil
	}, 10000, 10*1024*1024)
	if err != nil || calls != 2 {
		t.Fatal(err, calls)
	}
	rows := parseExport(t, result.Body)
	if len(rows) != 5 {
		t.Fatal("expected three account measures and two total measures")
	}
	first := rows[0]
	if first["measure"] != "net" || first["current_sign"] != "negative" || first["current_magnitude"] != "10.01" || first["prior_sign"] != "negative" || first["prior_magnitude"] != "2" || first["difference_sign"] != "negative" || first["difference_magnitude"] != "8.01" || first["layout"] != "account-v1" {
		t.Fatal(first)
	}
	if strings.Contains(string(result.Body), "hostile") || strings.Contains(string(result.Body), "malicious") {
		t.Fatal("label exported")
	}
	if rows[3]["row_kind"] != "summary" || rows[3]["current_magnitude"] != "10.01" || rows[1]["current_sign"] != "zero" {
		t.Fatal(rows)
	}
	for _, row := range rows {
		for _, v := range row {
			if len(v) > 0 && strings.ContainsAny(v[:1], "=+-@\t\r\n") {
				t.Fatal("unsafe cell")
			}
		}
	}
}

// TestReportCSVPagesAndFailures proves all pages, a final authority read and no partial errors.
func TestReportCSVPagesAndFailures(t *testing.T) {
	for _, scenario := range []string{"success", "revoked_page", "revoked_final", "stale_page", "metadata", "totals", "duplicate_row", "row_limit", "byte_limit", "missing_revision", "client_limit", "malformed_amount", "malformed_uuid", "deadline"} {
		t.Run(scenario, func(t *testing.T) {
			c, p := exportFixture()
			first := p
			first.Cursor = json.RawMessage(`{"revision":"2","filter_hash":"fixed","last_key":["a"]}`)
			calls := 0
			maxRows, maxBytes := 10000, 10*1024*1024
			if scenario == "row_limit" {
				maxRows = 7
			}
			if scenario == "byte_limit" {
				maxBytes = 100
			}
			if scenario == "missing_revision" {
				c.Body = `{"report":"trial-balance","through_date":"2026-01-31"}`
			}
			if scenario == "client_limit" {
				c.Body = strings.TrimSuffix(c.Body, "}") + `,"limit":1}`
			}
			if scenario == "malformed_amount" {
				first.Rows[0]["amounts"] = json.RawMessage(`{"net":"=1","debit_balance":"0","credit_balance":"10.01"}`)
			}
			if scenario == "malformed_uuid" {
				first.CurrencyVersion = "=uuid"
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			read := func(_ context.Context, cmd Command) (Result, error) {
				calls++
				if calls == 1 {
					return exportResult(first), nil
				}
				if calls == 2 {
					var request map[string]json.RawMessage
					_ = json.Unmarshal([]byte(cmd.Body), &request)
					if string(request["limit"]) != "500" || request["cursor"] == nil {
						t.Fatal("not fetching continuation")
					}
					if scenario == "revoked_page" {
						return Result{}, &Failure{403, "permission_denied"}
					}
					if scenario == "stale_page" {
						return Result{}, &Failure{409, "report_snapshot_changed"}
					}
					_, next := exportFixture()
					if scenario != "duplicate_row" {
						next.Rows[0]["account_id"] = json.RawMessage(`"00000000-0000-4000-8000-000000000012"`)
						next.Rows[0]["id"] = json.RawMessage(`"account:00000000-0000-4000-8000-000000000012"`)
					}
					if scenario == "metadata" {
						next.Policies = []string{"00000000-0000-4000-8000-000000000009"}
					}
					if scenario == "totals" {
						next.Totals["total_debits"] = "11"
					}
					if scenario == "deadline" {
						cancel()
					}
					return exportResult(next), nil
				}
				if scenario == "revoked_final" {
					return Result{}, &Failure{403, "permission_denied"}
				}
				return exportResult(p), nil
			}
			result, err := exportReport(ctx, c, read, maxRows, maxBytes)
			if scenario == "success" {
				if err != nil || calls != 3 || len(parseExport(t, result.Body)) != 8 {
					t.Fatal(err, calls)
				}
				return
			}
			if err == nil || len(result.Body) != 0 {
				t.Fatal("partial success", err)
			}
			var f *Failure
			if !errors.As(err, &f) {
				t.Fatal("unsafe error")
			}
			if strings.HasSuffix(scenario, "limit") && scenario != "client_limit" && f.Code != "report_export_limit" {
				t.Fatal(err)
			}
		})
	}
}

// TestReportCSVSignGrammar rejects formulas/noncanonical numerals without float parsing.
func TestReportCSVSignGrammar(t *testing.T) {
	for _, v := range []string{"0.01", "-0.001", "99999999999999999999999999999.999999999", "0"} {
		if _, err := exportSign(v); err != nil {
			t.Fatal(v, err)
		}
	}
	for _, v := range []string{"-0", "+1", "01", "1.0", "1e2", "=1", "1\n", "0.0000000001", "100000000000000000000000000000"} {
		if _, err := exportSign(v); err == nil {
			t.Fatal(v)
		}
	}
}

// TestReportCSVEmptySelection keeps real GL opening/closing and no fabricated posting.
func TestReportCSVEmptySelection(t *testing.T) {
	c, p := exportFixture()
	p.Report = "general-ledger"
	p.Comparison = nil
	p.Rows = nil
	p.Filters = map[string]string{"report": p.Report, "account_id": exportTestAccount, "starts_on": "2026-02-01", "ends_before": "2026-03-01"}
	p.Totals = map[string]string{"opening": "10.01", "period_debits": "0", "period_credits": "0", "closing": "10.01"}
	raw, _ := json.Marshal(p.Filters)
	var req map[string]string
	_ = json.Unmarshal(raw, &req)
	req["snapshot_revision"] = "2"
	raw, _ = json.Marshal(req)
	c.Body = string(raw)
	result, err := exportReport(context.Background(), c, func(context.Context, Command) (Result, error) { return exportResult(p), nil }, 10000, 10*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	rows := parseExport(t, result.Body)
	if len(rows) != 4 || rows[0]["measure"] != "opening" || rows[0]["current_magnitude"] != "10.01" || rows[3]["measure"] != "closing" || rows[3]["current_magnitude"] != "10.01" {
		t.Fatal(rows)
	}
	for _, row := range rows {
		if row["row_kind"] != "summary" || row["prior_sign"] != "" || row["account_id"] != "" || row["selected_account_id"] != exportTestAccount {
			t.Fatal(row)
		}
	}
}

// TestReportCSVRealLimits reaches the production physical-record bound with tiny pages.
func TestReportCSVRealLimits(t *testing.T) {
	c, p := exportFixture()
	p.Comparison = nil
	delete(p.Filters, "compare_through_date")
	c.Body = `{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"2"}`
	p.Rows[0]["comparison"] = json.RawMessage("null")
	template := p.Rows[0]
	p.Rows = nil
	for i := 0; i < 3333; i++ {
		row := map[string]json.RawMessage{}
		for k, v := range template {
			row[k] = v
		}
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1)
		row["account_id"] = json.RawMessage(`"` + id + `"`)
		row["id"] = json.RawMessage(`"account:` + id + `"`)
		p.Rows = append(p.Rows, row)
	}
	result, err := exportReport(context.Background(), c, func(context.Context, Command) (Result, error) { return exportResult(p), nil }, 10000, 10*1024*1024)
	var f *Failure
	if !errors.As(err, &f) || f.Code != "report_export_limit" || len(result.Body) != 0 {
		t.Fatal(err)
	}
}

// TestReportCSVByteCap reaches the real ten-MiB limit through repeated bounded provenance.
func TestReportCSVByteCap(t *testing.T) {
	c, p := exportFixture()
	p.Comparison = nil
	delete(p.Filters, "compare_through_date")
	c.Body = `{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"2"}`
	p.Rows[0]["comparison"] = json.RawMessage("null")
	template := p.Rows[0]
	p.Rows = nil
	p.Policies = nil
	for i := 0; i < 1024; i++ {
		p.Policies = append(p.Policies, fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1))
	}
	for i := 0; i < 100; i++ {
		row := map[string]json.RawMessage{}
		for k, v := range template {
			row[k] = v
		}
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1)
		row["account_id"] = json.RawMessage(`"` + id + `"`)
		row["id"] = json.RawMessage(`"account:` + id + `"`)
		p.Rows = append(p.Rows, row)
	}
	result, err := exportReport(context.Background(), c, func(context.Context, Command) (Result, error) { return exportResult(p), nil }, 10000, 10*1024*1024)
	var f *Failure
	if !errors.As(err, &f) || f.Code != "report_export_limit" || len(result.Body) != 0 {
		t.Fatal(err)
	}
}

// TestReportCSVPostingGrammar rejects noncanonical lineage fields before serialization.
func TestReportCSVPostingGrammar(t *testing.T) {
	valid := `{"id":"line:00000000-0000-4000-8000-000000000020:1","account_id":"00000000-0000-4000-8000-000000000011","journal_id":"00000000-0000-4000-8000-000000000020","line_number":1,"accounting_date":"2026-01-01","posted_at":"2026-01-01T12:00:00.000001Z","kind":"manual","reverses_journal_id":null,"policy_id":"00000000-0000-4000-8000-000000000004","side":"debit","transaction_currency":"USD","transaction_currency_version_id":"00000000-0000-4000-8000-000000000005","valuation_basis":"identity","rate_observation_id":null,"transaction_amount":"0.01","functional_amount":"0.01","amounts":{"debit":"0.01","credit":"0","running_balance":"0.01"}}`
	for _, scenario := range []string{"valid", "line", "rate_type", "reverse_type", "date", "category"} {
		t.Run(scenario, func(t *testing.T) {
			var row map[string]json.RawMessage
			_ = json.Unmarshal([]byte(valid), &row)
			switch scenario {
			case "line":
				row["line_number"] = json.RawMessage("1001")
				row["id"] = json.RawMessage(`"line:00000000-0000-4000-8000-000000000020:1001"`)
			case "rate_type":
				row["rate_observation_id"] = json.RawMessage("42")
			case "reverse_type":
				row["reverses_journal_id"] = json.RawMessage("false")
			case "date":
				row["accounting_date"] = json.RawMessage(`"2026-02-30"`)
			case "category":
				row["side"] = json.RawMessage(`"=debit"`)
			}
			_, _, _, err := exportRow("general-ledger", row)
			if (err == nil) != (scenario == "valid") {
				t.Fatal(err)
			}
		})
	}
}
