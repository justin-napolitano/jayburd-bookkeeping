//go:build integration

package ledger

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// TestReportReviewHTTPPostgres checks the composed routes with actual scoped SQL and fixed golden values.
func TestReportReviewHTTPPostgres(t *testing.T) {
	c, s, _ := fixtureDB(t, "review_tools_http")
	seedReportMonth(t, c, s)
	h := Handler(authStub{}, s)
	base := "/v1/entities/" + entity + "/books/" + book + "/reports/"
	revision := scalar(t, c, "SELECT revision::text FROM ledger.book WHERE entity_id=$1 AND book_id=$2", entity, book)
	call := func(path string, status int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer synthetic")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("status %d expected %d: %s", w.Code, status, w.Body.String())
		}
		return w
	}
	for _, tc := range []struct{ kind, query, measure, want string }{
		{"trial-balance", "through_date=2026-01-31", "total_debits", "1300"},
		{"balance-sheet", "through_date=2026-01-31", "assets", "1145"},
		{"profit-and-loss", "starts_on=2026-01-01&ends_before=2026-02-01", "net_income", "70"},
		{"general-ledger", "starts_on=2026-01-02&ends_before=2026-01-03&account_id=" + accountA, "closing", "1200"},
	} {
		w := call(base+tc.kind+"/export.csv?"+tc.query+"&snapshot_revision="+revision, 200)
		if w.Header().Get("Content-Type") != "text/csv; charset=utf-8; header=present" || w.Header().Get("Content-Disposition") == "" {
			t.Fatal("missing download headers")
		}
		rows, err := csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
		if err != nil || len(rows) < 2 {
			t.Fatal("invalid export")
		}
		columns := map[string]int{}
		for i, key := range rows[0] {
			columns[key] = i
		}
		found := 0
		for _, row := range rows[1:] {
			if row[columns["row_kind"]] == "summary" && row[columns["measure"]] == tc.measure {
				found++
				if row[columns["current_sign"]] != "positive" || row[columns["current_magnitude"]] != tc.want || row[columns["snapshot_revision"]] != revision {
					t.Fatal("incorrect golden export")
				}
			}
		}
		if found != 1 {
			t.Fatal("missing or duplicated summary measure")
		}
	}
	journal := scalar(t, c, "SELECT journal_id::text FROM ledger.journal_line WHERE entity_id=$1 AND book_id=$2 AND account_id=$3 ORDER BY journal_id LIMIT 1", entity, book, accountA)
	line := scalar(t, c, "SELECT line_number::text FROM ledger.journal_line WHERE entity_id=$1 AND book_id=$2 AND journal_id=$3 AND account_id=$4 ORDER BY line_number LIMIT 1", entity, book, journal, accountA)
	q := url.Values{"account_id": {accountA}, "starts_on": {"2026-01-01"}, "ends_before": {"2026-02-01"}, "line_number": {line}, "snapshot_revision": {revision}}
	drillPath := base + "general-ledger/journals/" + journal + "?" + q.Encode()
	w := call(drillPath, 200)
	var detail struct {
		Revision string `json:"snapshot_revision"`
		Journal  struct {
			ID string `json:"journal_id"`
		} `json:"journal"`
	}
	if json.Unmarshal(w.Body.Bytes(), &detail) != nil || detail.Revision != revision || detail.Journal.ID != journal {
		t.Fatal("wrong pinned journal")
	}
	exportPath := base + "trial-balance/export.csv?through_date=2026-01-31&snapshot_revision=" + revision
	sql(t, c, "UPDATE ledger.book SET revision=revision+1 WHERE entity_id=$1 AND book_id=$2", entity, book)
	for _, path := range []string{drillPath, exportPath} {
		w := call(path, 409)
		if !staleReportProblem(w) {
			t.Fatal("stale report data exposed")
		}
	}
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE principal_id=$1 AND permission='read'", actor)
	call(exportPath, 403)
	call(drillPath, 403)
}

// staleReportProblem proves a closed diagnostic envelope, excluding financial payload fields.
// Numeric substrings in the independent diagnostic UUID are not report amounts.
func staleReportProblem(w *httptest.ResponseRecorder) bool {
	if w.Code != http.StatusConflict || w.Header().Get("Content-Type") != "application/problem+json" || w.Header().Get("Content-Disposition") != "" {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(w.Body.Bytes()))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return false
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return false
		}
		if _, exists := fields[key]; exists {
			return false
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return false
		}
		fields[key] = value
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') {
		return false
	}
	if _, err = d.Token(); err != io.EOF || len(fields) != 5 {
		return false
	}
	for key, want := range map[string]string{"type": "https://bookkeeping.example.invalid/problems/report_snapshot_changed", "title": "Conflict", "code": "report_snapshot_changed"} {
		var got string
		if json.Unmarshal(fields[key], &got) != nil || got != want {
			return false
		}
	}
	var status int
	var id string
	return json.Unmarshal(fields["status"], &status) == nil && status == http.StatusConflict && json.Unmarshal(fields["request_id"], &id) == nil && regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id)
}

// TestStaleReportProblemDiagnosticID reproduces the numeric-sentinel collision deterministically.
func TestStaleReportProblemDiagnosticID(t *testing.T) {
	response := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		problem(w, "00001300-0000-4000-8000-000000000001", &Failure{http.StatusConflict, "report_snapshot_changed"})
		return w
	}
	if w := response(); !strings.Contains(w.Body.String(), "1300") || !staleReportProblem(w) {
		t.Fatal("safe diagnostic identifier rejected")
	}
	for _, field := range []string{"totals", "rows", "journal"} {
		w := response()
		var fields map[string]json.RawMessage
		if json.Unmarshal(w.Body.Bytes(), &fields) != nil {
			t.Fatal("invalid problem fixture")
		}
		fields[field] = json.RawMessage(`{"amount":"1300"}`)
		raw, err := json.Marshal(fields)
		if err != nil {
			t.Fatal("invalid financial fixture")
		}
		w.Body.Reset()
		_, _ = w.Body.Write(raw)
		if staleReportProblem(w) {
			t.Fatal("financial field accepted")
		}
	}
	for _, mutate := range []func(*httptest.ResponseRecorder){
		func(w *httptest.ResponseRecorder) { w.Body.Reset(); _, _ = w.Body.WriteString("amount\r\n1300\r\n") },
		func(w *httptest.ResponseRecorder) { w.Header().Set("Content-Disposition", "attachment") },
		func(w *httptest.ResponseRecorder) { w.Header().Set("Content-Type", "text/csv") },
		func(w *httptest.ResponseRecorder) { w.Code = http.StatusOK },
		func(w *httptest.ResponseRecorder) {
			w.Body.Reset()
			_, _ = w.Body.WriteString(`{"type":"https://bookkeeping.example.invalid/problems/report_snapshot_changed","title":"Conflict","code":"report_snapshot_changed","status":409,"request_id":"00001300-0000-4000-8000-000000000001","status":409}`)
		},
	} {
		w := response()
		mutate(w)
		if staleReportProblem(w) {
			t.Fatal("unsafe response accepted")
		}
	}
}
