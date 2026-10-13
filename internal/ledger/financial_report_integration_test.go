//go:build integration

package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// reportProof is the independent response projection used for fixed-value assertions.
type reportProof struct {
	Revision   string                                         `json:"snapshot_revision"`
	Totals     map[string]string                              `json:"totals"`
	Comparison struct{ Totals, Difference map[string]string } `json:"comparison"`
	Rows       []map[string]json.RawMessage                   `json:"rows"`
	Cursor     json.RawMessage                                `json:"next_cursor"`
}

// reportCall preserves request structure while letting SQL validate every field.
func reportCall(t testing.TB, s *Service, v map[string]any) reportProof {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: string(body)})
	if err != nil {
		t.Fatal(err)
	}
	var out reportProof
	if err = json.Unmarshal(r.Body, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// reportString requires actual JSON strings, never silently coercing numbers.
func reportString(t testing.TB, row map[string]json.RawMessage, key string) string {
	t.Helper()
	var value string
	if err := json.Unmarshal(row[key], &value); err != nil {
		t.Fatalf("%s not string: %v", key, err)
	}
	return value
}

// reportAmounts checks independent decimal strings from an actual response row.
func reportAmounts(t testing.TB, row map[string]json.RawMessage) map[string]string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal(row["amounts"], &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// seedReportMonth posts original corpus entries using runtime SQL, without using expected balances.
func seedReportMonth(t *testing.T, c *pgx.Conn, s *Service) {
	t.Helper()
	raw, err := os.ReadFile("tests/corpus/golden-month.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Books []struct {
			Accounts []string
			Entries  []struct {
				ID, Date, Kind, Reverses string
				Lines                    []struct {
					Account, Side, Currency, Amount string
					Functional                      string `json:"functional_amount"`
				}
			}
		}
	}
	if err = json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{"Assets:Checking": accountA, "Equity:Capital": accountB}
	for _, name := range corpus.Books[0].Accounts {
		if ids[name] != "" {
			continue
		}
		id := requestID()
		ids[name] = id
		category, side := "expense", "debit"
		switch strings.Split(name, ":")[0] {
		case "Assets":
			category = "asset"
		case "Equity":
			category, side = "equity", "credit"
		case "Income":
			category, side = "revenue", "credit"
		case "Liabilities":
			category, side = "liability", "credit"
		}
		sql(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,$4,$4,$5,$6,false,'2026-01-01',NULL)`, entity, book, id, name, category, side)
	}
	sql(t, c, "UPDATE ledger.book SET revision=0")
	journals := map[string]string{}
	for i, e := range corpus.Books[0].Entries {
		cmd := command(e.ID)
		v := map[string]any{"expected_revision": fmt.Sprint(i), "accounting_date": e.Date}
		if e.Kind == "reversal" {
			cmd.Journal = journals[e.Reverses]
			v["reason"] = "Synthetic correction"
		} else {
			v["policy_id"] = policyID
			v["description"] = e.ID
			lines := []any{}
			for _, l := range e.Lines {
				lines = append(lines, map[string]any{"account_id": ids[l.Account], "side": l.Side, "transaction_currency": l.Currency, "transaction_amount": l.Amount, "functional_amount": l.Functional, "valuation_basis": "identity"})
			}
			v["lines"] = lines
		}
		body, _ := json.Marshal(v)
		cmd.Body = string(body)
		var r Result
		if e.Kind == "reversal" {
			r, err = s.Reverse(context.Background(), cmd)
		} else {
			r, err = s.Post(context.Background(), cmd)
		}
		if err != nil {
			t.Fatalf("%s: %v", e.ID, err)
		}
		journals[e.ID] = postedID(t, r)
	}
}

// TestFinancialReportsGoldenMonth ties all four reports to constants independent of report SQL.
func TestFinancialReportsGoldenMonth(t *testing.T) {
	c, s, _ := fixtureDB(t, "reports_golden")
	seedReportMonth(t, c, s)
	before := ledgerProofSnapshot(t, c)
	for _, tc := range []struct {
		kind   string
		totals map[string]string
	}{
		{"trial-balance", map[string]string{"total_debits": "1300", "total_credits": "1300"}},
		{"balance-sheet", map[string]string{"assets": "1145", "liabilities": "0", "direct_equity": "1075", "unclosed_earnings": "70", "total_equity": "1145"}},
		{"profit-and-loss", map[string]string{"revenue": "200", "expense": "130", "net_income": "70"}},
	} {
		v := map[string]any{"report": tc.kind}
		if tc.kind == "profit-and-loss" {
			v["starts_on"] = "2026-01-01"
			v["ends_before"] = "2026-02-01"
			v["compare_starts_on"] = "2026-02-01"
			v["compare_ends_before"] = "2026-03-01"
		} else {
			v["through_date"] = "2026-01-31"
		}
		r := reportCall(t, s, v)
		if !reflect.DeepEqual(r.Totals, tc.totals) {
			t.Fatalf("%s: %v", tc.kind, r.Totals)
		}
		if tc.kind == "profit-and-loss" && (!reflect.DeepEqual(r.Comparison.Totals, map[string]string{"revenue": "0", "expense": "0", "net_income": "0"}) || !reflect.DeepEqual(r.Comparison.Difference, tc.totals)) {
			t.Fatal("comparison", r.Comparison)
		}
	}
	gl := reportCall(t, s, map[string]any{"report": "general-ledger", "starts_on": "2026-01-02", "ends_before": "2026-01-03", "account_id": accountA})
	if !reflect.DeepEqual(gl.Totals, map[string]string{"opening": "1000", "period_debits": "200", "period_credits": "0", "closing": "1200"}) || len(gl.Rows) != 1 || reportAmounts(t, gl.Rows[0])["running_balance"] != "1200" {
		t.Fatal("GL opening/range", gl.Totals)
	}
	empty := reportCall(t, s, map[string]any{"report": "general-ledger", "starts_on": "2026-02-01", "ends_before": "2026-03-01", "account_id": accountA})
	if len(empty.Rows) != 0 || empty.Totals["opening"] != "1045" || empty.Totals["closing"] != "1045" {
		t.Fatal("empty period lost history")
	}
	requireLedgerUnchanged(t, c, before)
}

// TestReportBoundaryAndRevision exercises denied scope, duplicate JSON, cursor binding and helper grants.
func TestReportBoundaryAndRevision(t *testing.T) {
	c, s, db := fixtureDB(t, "reports_boundary")
	first := reportCall(t, s, map[string]any{"report": "trial-balance", "through_date": "2026-01-31", "limit": 1})
	if first.Revision != "0" || string(first.Cursor) == "null" {
		t.Fatal("zero revision/cursor")
	}
	var cursor map[string]any
	if err := json.Unmarshal(first.Cursor, &cursor); err != nil {
		t.Fatal(err)
	}
	second := reportCall(t, s, map[string]any{"report": "trial-balance", "through_date": "2026-01-31", "limit": 1, "snapshot_revision": "0", "cursor": cursor})
	if len(second.Rows) != 1 || reportString(t, first.Rows[0], "id") == reportString(t, second.Rows[0], "id") || string(second.Cursor) != "null" {
		t.Fatal("keyset duplicates")
	}
	before := ledgerProofSnapshot(t, c)
	cases := []struct{ raw, code string }{
		{`{"report":"trial-balance","report":"balance-sheet","through_date":"2026-01-31"}`, "invalid_request"},
		{`{"report":"trial-balance","through_date":"2026-01-31","limit":1.5}`, "invalid_request"},
		{`{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"9223372036854775808"}`, "invalid_request"},
		{`{"report":"trial-balance","through_date":"2026-02-30"}`, "invalid_request"},
		{`{"report":"trial-balance","through_date":"2025-12-31"}`, "unsupported_policy"},
		{`{"report":"general-ledger","starts_on":"2026-01-01","ends_before":"2026-01-01","account_id":"` + accountA + `"}`, "invalid_request"},
		{`{"report":"general-ledger","starts_on":"2026-01-01","ends_before":"2026-02-01","account_id":"00000000-0000-4000-8000-000000000099"}`, "not_found"},
		{`{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"0","cursor":{"revision":"0","revision":"0","filter_hash":"x","last_key":[]}}`, "invalid_request"},
	}
	for _, tc := range cases {
		_, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: tc.raw})
		requireFailure(t, err, tc.code)
	}
	cursor["last_key"] = []string{"not-an-account", accountA}
	bad, _ := json.Marshal(map[string]any{"report": "trial-balance", "through_date": "2026-01-31", "snapshot_revision": "0", "cursor": cursor})
	_, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: string(bad)})
	requireFailure(t, err, "invalid_request")
	_, err = s.FinancialReport(context.Background(), Command{Actor: actor, Entity: requestID(), Book: book, Body: `{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"99"}`})
	requireFailure(t, err, "not_found")
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	_, err = runtime.Exec(context.Background(), `select reporting.report_data($1,$2,'trial-balance','2026-01-01','2026-01-31',NULL,0)`, entity, book)
	requireLedgerSQLState(t, err, "42501")
	requireLedgerUnchanged(t, c, before)
	if _, err = s.Post(context.Background(), command("rev-change")); err != nil {
		t.Fatal(err)
	}
	_, err = s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: `{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"0"}`})
	requireFailure(t, err, "report_snapshot_changed")
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE permission='read'")
	_, err = s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: `{"report":"trial-balance","through_date":"2026-01-31","snapshot_revision":"0"}`})
	requireFailure(t, err, "permission_denied")
}

// TestReportReversalProvenance keeps January values and February offsets tied to original FX.
func TestReportReversalProvenance(t *testing.T) {
	_, s, _ := fixtureDB(t, "reports_fx")
	cmd := change(command("spot"), func(v map[string]any) {
		for _, raw := range v["lines"].([]any) {
			l := raw.(map[string]any)
			l["transaction_currency"] = "EUR"
			l["transaction_amount"] = "10"
			l["functional_amount"] = "11"
			l["valuation_basis"] = "spot"
			l["rate_observation_id"] = rateID
		}
	})
	r, err := s.Post(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	id := postedID(t, r)
	rev := Command{Actor: actor, Entity: entity, Book: book, Journal: id, Key: "reverse", Request: requestID(), Body: `{"expected_revision":"1","accounting_date":"2026-02-01","reason":"Synthetic correction"}`}
	if _, err = s.Reverse(context.Background(), rev); err != nil {
		t.Fatal(err)
	}
	jan := reportCall(t, s, map[string]any{"report": "trial-balance", "through_date": "2026-01-31", "compare_through_date": "2026-02-01"})
	if jan.Totals["total_debits"] != "11" || jan.Comparison.Totals["total_debits"] != "0" {
		t.Fatal("date reversal", jan)
	}
	gl := reportCall(t, s, map[string]any{"report": "general-ledger", "starts_on": "2026-01-01", "ends_before": "2026-02-02", "account_id": accountA})
	if len(gl.Rows) != 2 || gl.Totals["closing"] != "0" {
		t.Fatal("GL FX reversal")
	}
	for _, row := range gl.Rows {
		if reportString(t, row, "transaction_currency") != "EUR" || reportString(t, row, "transaction_amount") != "10" || reportString(t, row, "functional_amount") != "11" || reportString(t, row, "rate_observation_id") != rateID {
			t.Fatal("lost historical FX")
		}
	}
	if reportString(t, gl.Rows[1], "reverses_journal_id") != id {
		t.Fatal("lost reversal link")
	}
}

// TestReportThousandPostings proves complete keyset traversal over actual cent postings, not injected rows.
func TestReportThousandPostings(t *testing.T) {
	c, s, db := fixtureDB(t, "reports_volume")
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	s.DB = runtime
	started := time.Now()
	for i := 0; i < 1024; i++ {
		cmd := change(command(fmt.Sprintf("cent-%d", i)), func(v map[string]any) {
			v["expected_revision"] = fmt.Sprint(i)
			for _, raw := range v["lines"].([]any) {
				l := raw.(map[string]any)
				l["transaction_amount"] = "0.01"
				l["functional_amount"] = "0.01"
			}
		})
		if _, err := s.Post(context.Background(), cmd); err != nil {
			t.Fatalf("post%d: %v", i, err)
		}
	}
	v := map[string]any{"report": "general-ledger", "starts_on": "2026-01-01", "ends_before": "2026-02-01", "account_id": accountA, "limit": 127}
	seen := map[string]bool{}
	pages := 0
	for {
		r := reportCall(t, s, v)
		pages++
		if r.Totals["closing"] != "10.24" || r.Totals["period_debits"] != "10.24" {
			t.Fatal("cent totals", r.Totals)
		}
		for _, row := range r.Rows {
			id := reportString(t, row, "id")
			if seen[id] {
				t.Fatal("duplicate page row")
			}
			seen[id] = true
			want := fmt.Sprintf("%d.%02d", len(seen)/100, len(seen)%100)
			want = strings.TrimRight(strings.TrimRight(want, "0"), ".")
			if reportAmounts(t, row)["running_balance"] != want {
				t.Fatal("running balance crossed page wrong")
			}
		}
		if string(r.Cursor) == "null" {
			break
		}
		var cursor any
		if err := json.Unmarshal(r.Cursor, &cursor); err != nil {
			t.Fatal(err)
		}
		v["cursor"] = cursor
		v["snapshot_revision"] = r.Revision
	}
	if len(seen) != 1024 || pages != 9 || scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1024" {
		t.Fatalf("missing rows/pages: %d/%d", len(seen), pages)
	}
	t.Logf("1024 committed cent journals and nine exact GL pages: %s (synthetic local measurement, not hosted SLA)", time.Since(started))
}

// TestReportCursorScopeCollision binds a cursor to scope even when IDs and revisions collide legitimately.
func TestReportCursorScopeCollision(t *testing.T) {
	c, s, _ := fixtureDB(t, "reports_cursor_scope")
	other := requestID()
	sql(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Other book')`, entity, other)
	sql(t, c, `INSERT INTO ledger.book_policy SELECT entity_id,$1,policy_id,effective_from,effective_to,functional_currency_version_id,basis,timezone,posting_scale,rounding_mode,rate_policy_code,approved_by,approved_at FROM ledger.book_policy WHERE book_id=$2`, other, book)
	sql(t, c, `INSERT INTO ledger.account SELECT entity_id,$1,account_id,code,name,category,normal_side,monetary,active_from,active_to FROM ledger.account WHERE book_id=$2`, other, book)
	sql(t, c, `UPDATE ledger.book SET revision=0`)
	first := reportCall(t, s, map[string]any{"report": "trial-balance", "through_date": "2026-01-31", "limit": 1})
	var cursor any
	if err := json.Unmarshal(first.Cursor, &cursor); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"report": "trial-balance", "through_date": "2026-01-31", "snapshot_revision": "0", "cursor": cursor})
	_, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: other, Body: string(body)})
	requireFailure(t, err, "invalid_request")
}

// TestReportMaterializationBound checks explicit foundation limits before large JSON allocation.
func TestReportMaterializationBound(t *testing.T) {
	c, s, _ := fixtureDB(t, "reports_row_bound")
	sql(t, c, `INSERT INTO ledger.account(entity_id,book_id,account_id,code,name,category,normal_side,monetary,active_from)
 SELECT $1,$2,gen_random_uuid(),'limit-'||i,'Synthetic','asset','debit',false,'2026-01-01' FROM generate_series(1,4095) i`, entity, book)
	before := ledgerProofSnapshot(t, c)
	_, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: `{"report":"trial-balance","through_date":"2026-01-31","limit":1}`})
	requireFailure(t, err, "result_out_of_range")
	requireLedgerUnchanged(t, c, before)
}

// TestReportPolicyAndUnicode rejects unsupported reference metadata and unsafe JSON representations.
func TestReportPolicyAndUnicode(t *testing.T) {
	c, s, _ := fixtureDB(t, "reports_policy")
	raw := `{"report":"trial-balance","through_date":"2026-01-31","unexpected":"\u0000"}`
	_, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: raw})
	requireFailure(t, err, "invalid_request")
	// Administrative synthetic corruption simulates a legacy currency without minor-unit metadata.
	sql(t, c, `ALTER TABLE reference.currency DISABLE TRIGGER immutable; UPDATE reference.currency SET minor_unit=NULL WHERE alphabetic_code='USD'; ALTER TABLE reference.currency ENABLE TRIGGER immutable`)
	_, err = s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: `{"report":"trial-balance","through_date":"2026-01-31"}`})
	requireFailure(t, err, "unsupported_policy")
}

// TestReportAggregateOverflow rejects a one-cent overflow instead of rounding or emitting partial rows.
func TestReportAggregateOverflow(t *testing.T) {
	c, s, _ := fixtureDB(t, "reports_overflow")
	for i, amount := range []string{"99999999999999999999999999999.99", "0.01"} {
		cmd := change(command(fmt.Sprintf("overflow-%d", i)), func(v map[string]any) {
			v["expected_revision"] = fmt.Sprint(i)
			for _, raw := range v["lines"].([]any) {
				l := raw.(map[string]any)
				l["transaction_amount"] = amount
				l["functional_amount"] = amount
			}
		})
		if _, err := s.Post(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}
	before := ledgerProofSnapshot(t, c)
	for _, kind := range []string{"trial-balance", "balance-sheet", "general-ledger"} {
		v := map[string]any{"report": kind}
		if kind == "general-ledger" {
			v["starts_on"] = "2026-01-01"
			v["ends_before"] = "2026-02-01"
			v["account_id"] = accountA
		} else {
			v["through_date"] = "2026-01-31"
		}
		body, _ := json.Marshal(v)
		r, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: string(body)})
		requireFailure(t, err, "result_out_of_range")
		if len(r.Body) != 0 {
			t.Fatal("partial overflow response")
		}
	}
	requireLedgerUnchanged(t, c, before)
}

// TestReportRevocationRace proves a waiting report cannot use membership revoked before its lock succeeds.
func TestReportRevocationRace(t *testing.T) {
	c, s, db := fixtureDB(t, "reports_revoke_race")
	admin := connection(t, db, "postgres")
	defer admin.Close(context.Background())
	tx, err := admin.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(context.Background(), "UPDATE ledger.membership SET revoked_at=now() WHERE permission='read'"); err != nil {
		t.Fatal(err)
	}
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	s.DB = runtime
	done := make(chan error, 1)
	go func() {
		_, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: `{"report":"trial-balance","through_date":"2026-01-31"}`})
		done <- err
	}()
	waitBlocked(t, c, runtime.PgConn().PID())
	if err = tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	requireFailure(t, <-done, "permission_denied")
}

// TestReportPolicyCountBound caps historical policy provenance before constructing its response array.
func TestReportPolicyCountBound(t *testing.T) {
	c, s, _ := fixtureDB(t, "reports_policy_bound")
	// Controlled synthetic setup only; no posted history exists during this fixture rewrite.
	sql(t, c, `ALTER TABLE ledger.book_policy DISABLE TRIGGER immutable; UPDATE ledger.book_policy SET effective_to='2026-01-02'; ALTER TABLE ledger.book_policy ENABLE TRIGGER immutable`)
	sql(t, c, `INSERT INTO ledger.book_policy SELECT entity_id,book_id,gen_random_uuid(),date '2026-01-01'+i,date '2026-01-02'+i,functional_currency_version_id,basis,timezone,posting_scale,rounding_mode,rate_policy_code,approved_by,approved_at FROM ledger.book_policy CROSS JOIN generate_series(1,1024) i WHERE policy_id=$1`, policyID)
	sql(t, c, "UPDATE ledger.book SET revision=0")
	if _, err := s.Post(context.Background(), command("historical-opening")); err != nil {
		t.Fatal(err)
	}
	date := scalar(t, c, "SELECT (date '2026-01-01'+1024)::text")
	raw, _ := json.Marshal(map[string]any{"report": "trial-balance", "through_date": date})
	_, err := s.FinancialReport(context.Background(), Command{Actor: actor, Entity: entity, Book: book, Body: string(raw)})
	requireFailure(t, err, "result_out_of_range")
}
