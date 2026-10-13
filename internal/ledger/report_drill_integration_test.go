//go:build integration

package ledger

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// drillCommand retains the exact GL account/range and selected immutable line.
func drillCommand(id, revision string) Command {
	body, _ := json.Marshal(map[string]any{"snapshot_revision": revision, "account_id": accountA, "journal_id": id, "starts_on": "2026-01-01", "ends_before": "2026-02-01", "line_number": 1})
	return Command{Actor: actor, Entity: entity, Book: book, Body: string(body)}
}

// TestReportJournalPinned proves a real GL row, exact journal and unchanged durable state.
func TestReportJournalPinned(t *testing.T) {
	c, s, _ := fixtureDB(t, "drill_pinned")
	posted, err := s.Post(context.Background(), command("drill-capital"))
	if err != nil {
		t.Fatal(err)
	}
	gl := reportCall(t, s, map[string]any{"report": "general-ledger", "account_id": accountA, "starts_on": "2026-01-01", "ends_before": "2026-02-01"})
	if len(gl.Rows) != 1 || gl.Revision != "1" {
		t.Fatal("unexpected GL proof")
	}
	id := reportString(t, gl.Rows[0], "journal_id")
	before := ledgerProofSnapshot(t, c)
	r, err := s.ReportJournal(context.Background(), drillCommand(id, gl.Revision))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Version  string          `json:"contract_version"`
		Revision string          `json:"snapshot_revision"`
		Journal  json.RawMessage `json:"journal"`
	}
	if err = json.Unmarshal(r.Body, &out); err != nil {
		t.Fatal(err)
	}
	var actual, want any
	_ = json.Unmarshal(out.Journal, &actual)
	_ = json.Unmarshal(posted.Body, &want)
	if out.Version != "report-journal-v1" || out.Revision != "1" || !reflect.DeepEqual(actual, want) {
		t.Fatal("journal or snapshot changed", string(r.Body))
	}
	if !reflect.DeepEqual(before, ledgerProofSnapshot(t, c)) {
		t.Fatal("drill mutated durable state")
	}
	next := change(command("later-post"), func(v map[string]any) { v["expected_revision"] = "1" })
	if _, err = s.Post(context.Background(), next); err != nil {
		t.Fatal(err)
	}
	_, err = s.ReportJournal(context.Background(), drillCommand(id, "1"))
	requireFailure(t, err, "report_snapshot_changed")
}

// TestReportJournalBoundary rejects malformed input and forged inclusion without leaking state.
func TestReportJournalBoundary(t *testing.T) {
	c, s, db := fixtureDB(t, "drill_boundary")
	posted, err := s.Post(context.Background(), command("drill-boundary"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := drillCommand(postedID(t, posted), "1")
	before := ledgerProofSnapshot(t, c)
	for _, tc := range []struct {
		name, key string
		value     any
		code      string
	}{
		{"other_line", "line_number", 2, "not_found"}, {"missing_line", "line_number", 3, "not_found"},
		{"other_account", "account_id", accountB, "not_found"}, {"other_journal", "journal_id", requestID(), "not_found"},
		{"outside_range", "starts_on", "2026-01-02", "not_found"},
		{"wrong_revision", "snapshot_revision", "0", "report_snapshot_changed"},
		{"numeric_revision", "snapshot_revision", 1, "invalid_request"}, {"big_revision", "snapshot_revision", "9223372036854775808", "invalid_request"},
		{"leading_revision", "snapshot_revision", "01", "invalid_request"}, {"unknown", "unknown", true, "invalid_request"},
		{"null", "account_id", nil, "invalid_request"}, {"line_string", "line_number", "1", "invalid_request"},
		{"line_zero", "line_number", 0, "invalid_request"}, {"line_large", "line_number", 1001, "invalid_request"},
		{"line_fraction", "line_number", 1.5, "invalid_request"}, {"empty_range", "ends_before", "2026-01-01", "invalid_request"},
		{"long_range", "ends_before", "2027-02-01", "invalid_request"}, {"bad_date", "starts_on", "2026-02-30", "invalid_request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.ReportJournal(context.Background(), change(cmd, func(v map[string]any) { v[tc.key] = tc.value }))
			requireFailure(t, err, tc.code)
		})
	}
	for _, raw := range []string{`{}`, `{"snapshot_revision":"1","snapshot_revision":"1"}`, strings.Repeat(" ", 8193), strings.Replace(cmd.Body, accountA, `\u0000`, 1)} {
		bad := cmd
		bad.Body = raw
		_, err = s.ReportJournal(context.Background(), bad)
		requireFailure(t, err, "invalid_request")
	}
	bad := cmd
	bad.Book = requestID()
	_, err = s.ReportJournal(context.Background(), bad)
	requireFailure(t, err, "not_found")
	bad = cmd
	bad.Entity = requestID()
	_, err = s.ReportJournal(context.Background(), bad)
	requireFailure(t, err, "not_found")
	if !reflect.DeepEqual(before, ledgerProofSnapshot(t, c)) {
		t.Fatal("rejected drill mutated durable state")
	}
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	if scalar(t, runtime, "SELECT has_function_privilege(current_user,'reporting.report_journal(uuid,uuid,uuid,text)','EXECUTE')::text") != "true" {
		t.Fatal("missing wrapper grant")
	}
	if scalar(t, runtime, "SELECT has_function_privilege(current_user,'ledger.render_journal(uuid,uuid,uuid)','EXECUTE')::text") != "false" {
		t.Fatal("private renderer exposed")
	}
	if scalar(t, c, "SELECT EXISTS(SELECT FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid='reporting.report_journal(uuid,uuid,uuid,text)'::regprocedure AND a.grantee=0)::text") != "false" {
		t.Fatal("PUBLIC grant")
	}
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE permission='read'")
	_, err = s.ReportJournal(context.Background(), cmd)
	requireFailure(t, err, "permission_denied")
}

// TestReportJournalReversal keeps original and correction provenance on their own accounting dates.
func TestReportJournalReversal(t *testing.T) {
	_, s, _ := fixtureDB(t, "drill_reversal")
	original := change(command("drill-original"), func(v map[string]any) {
		for _, raw := range v["lines"].([]any) {
			l := raw.(map[string]any)
			l["transaction_currency"] = "EUR"
			l["transaction_amount"] = "10"
			l["functional_amount"] = "11"
			l["valuation_basis"] = "spot"
			l["rate_observation_id"] = rateID
		}
	})
	posted, err := s.Post(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	id := postedID(t, posted)
	reverse := Command{Actor: actor, Entity: entity, Book: book, Journal: id, Key: "drill-reverse", Request: requestID(), Body: `{"expected_revision":"1","accounting_date":"2026-02-01","reason":"Synthetic correction"}`}
	reversed, err := s.Reverse(context.Background(), reverse)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id, start, end string
		want           Result
	}{{id, "2026-01-01", "2026-02-01", posted}, {postedID(t, reversed), "2026-02-01", "2026-03-01", reversed}} {
		cmd := change(drillCommand(tc.id, "2"), func(v map[string]any) { v["starts_on"] = tc.start; v["ends_before"] = tc.end })
		r, err := s.ReportJournal(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]json.RawMessage
		_ = json.Unmarshal(r.Body, &out)
		var got, want any
		_ = json.Unmarshal(out["journal"], &got)
		_ = json.Unmarshal(tc.want.Body, &want)
		if !reflect.DeepEqual(got, want) {
			t.Fatal("changed reversal provenance")
		}
	}
}

// TestReportJournalRevocationRace orders revocation before the reader's authority lock.
func TestReportJournalRevocationRace(t *testing.T) {
	c, s, db := fixtureDB(t, "drill_revoke_race")
	posted, err := s.Post(context.Background(), command("drill-race"))
	if err != nil {
		t.Fatal(err)
	}
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
		_, err := s.ReportJournal(context.Background(), drillCommand(postedID(t, posted), "1"))
		done <- err
	}()
	waitBlocked(t, c, runtime.PgConn().PID())
	if err = tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	requireFailure(t, <-done, "permission_denied")
}
