//go:build integration

package ledger

import (
	"encoding/csv"
	"encoding/json"
	"net/http/httptest"
	"net/url"
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
		if w.Header().Get("Content-Disposition") != "" || strings.Contains(w.Body.String(), "1300") {
			t.Fatal("stale report data exposed")
		}
	}
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE principal_id=$1 AND permission='read'", actor)
	call(exportPath, 403)
	call(drillPath, 403)
}
