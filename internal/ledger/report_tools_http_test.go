package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// reviewOperations records only transport calls; PostgreSQL tests prove accounting.
type reviewOperations struct{ operationsStub }

// FinancialReportCSV exposes a synthetic completed buffer to exercise HTTP framing.
func (s *reviewOperations) FinancialReportCSV(ctx context.Context, c Command) (Result, error) {
	return s.Post(ctx, c)
}

// ReportJournal records the trusted actor and path-bound journal selector.
func (s *reviewOperations) ReportJournal(ctx context.Context, c Command) (Result, error) {
	return s.Post(ctx, c)
}

// TestReportReviewTransport proves authentication, pinning, safe errors and download bounds.
func TestReportReviewTransport(t *testing.T) {
	base := "/v1/entities/e/books/b/reports/"
	export := base + "trial-balance/export.csv?through_date=2026-01-31&snapshot_revision=5"
	drill := base + "general-ledger/journals/j?account_id=a&starts_on=2026-01-01&ends_before=2026-02-01&line_number=1&snapshot_revision=5"
	for _, tc := range []struct {
		name, path, method, body, response string
		anonymous                          bool
		err                                error
		status, calls                      int
	}{
		{name: "export", path: export, response: "\"profile\"\r\n", status: 200, calls: 1},
		{name: "journal", path: drill, response: `{}`, status: 200, calls: 1},
		{name: "auth_before_parse", path: export + "&bad=%zz", anonymous: true, status: 401},
		{name: "unpinned", path: base + "trial-balance/export.csv?through_date=2026-01-31", status: 400},
		{name: "page_limit", path: export + "&limit=100", status: 400},
		{name: "cursor", path: export + "&cursor=abc", status: 400},
		{name: "duplicate", path: export + "&snapshot_revision=5", status: 400},
		{name: "export_body", path: export, body: "{}", status: 400},
		{name: "export_head", path: export, method: "HEAD", status: 400},
		{name: "drill_head", path: drill, method: "HEAD", status: 400},
		{name: "drill_body", path: drill, body: "{}", status: 400},
		{name: "unknown", path: drill + "&actor=forged", status: 400},
		{name: "bad_line", path: strings.Replace(drill, "line_number=1", "line_number=01", 1), status: 400},
		{name: "missing_line", path: strings.Replace(drill, "&line_number=1", "", 1), status: 400},
		{name: "export_size", path: export, response: strings.Repeat("x", (10<<20)+1), status: 422, calls: 1},
		{name: "drill_size", path: drill, response: strings.Repeat("x", (2<<20)+1), status: 422, calls: 1},
		{name: "stale", path: export, err: &Failure{409, "report_snapshot_changed"}, status: 409, calls: 1},
		{name: "redacted", path: drill, err: errors.New("private SQL payload"), status: 500, calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.method
			if method == "" {
				method = "GET"
			}
			r := httptest.NewRequest(method, tc.path, strings.NewReader(tc.body))
			if !tc.anonymous {
				r.Header.Set("Authorization", "Bearer synthetic")
			}
			ops := &reviewOperations{operationsStub{result: Result{Status: 200, Body: []byte(tc.response)}, err: tc.err}}
			w := httptest.NewRecorder()
			Handler(authStub{}, ops).ServeHTTP(w, r)
			if w.Code != tc.status || ops.calls != tc.calls {
				t.Fatalf("status %d calls %d", w.Code, ops.calls)
			}
			if w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "private SQL") {
				t.Fatal("unsafe output")
			}
			if tc.status != 200 && w.Header().Get("Content-Disposition") != "" {
				t.Fatal("failure advertised as a download")
			}
			if tc.name == "export" && (w.Header().Get("Content-Type") != "text/csv; charset=utf-8; header=present" || w.Header().Get("Content-Disposition") != `attachment; filename="financial-reporting-review.csv"` || w.Body.String() != tc.response) {
				t.Fatal("download framing changed")
			}
			if tc.name == "journal" {
				var fields map[string]json.RawMessage
				if json.Unmarshal([]byte(ops.command.Body), &fields) != nil || string(fields["journal_id"]) != `"j"` || string(fields["line_number"]) != "1" || ops.command.Actor != "00000000-0000-4000-8000-000000000001" {
					t.Fatal("untrusted drill selector")
				}
			}
		})
	}
}
