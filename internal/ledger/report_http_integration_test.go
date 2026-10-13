//go:build integration

package ledger

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestReportingHTTPPostgres exercises actual scoped SQL through authenticated transport.
func TestReportingHTTPPostgres(t *testing.T) {
	c, s, _ := fixtureDB(t, "report_http")
	seedReportMonth(t, c, s)
	h := Handler(authStub{}, s)
	base := "/v1/entities/" + entity + "/books/" + book + "/reports/"
	call := func(path string, status int) map[string]json.RawMessage {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer synthetic")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("status%d expected%d: %s", w.Code, status, w.Body.String())
		}
		var result map[string]json.RawMessage
		if json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatal("invalid JSON")
		}
		return result
	}
	for _, tc := range []struct{ kind, query, key, want string }{{"trial-balance", "through_date=2026-01-31", "total_debits", "1300"}, {"balance-sheet", "through_date=2026-01-31", "assets", "1145"}, {"profit-and-loss", "starts_on=2026-01-01&ends_before=2026-02-01", "net_income", "70"}, {"general-ledger", "starts_on=2026-01-02&ends_before=2026-01-03&account_id=" + accountA, "closing", "1200"}} {
		v := call(base+tc.kind+"?"+tc.query, 200)
		var totals map[string]string
		if json.Unmarshal(v["totals"], &totals) != nil || totals[tc.key] != tc.want {
			t.Fatal("wrong golden totals")
		}
	}
	query := "through_date=2026-01-31&limit=1"
	v := call(base+"trial-balance?"+query, 200)
	var cursor, revision string
	_ = json.Unmarshal(v["next_cursor"], &cursor)
	_ = json.Unmarshal(v["snapshot_revision"], &revision)
	if cursor == "" {
		t.Fatal("missing continuation")
	}
	next := base + "trial-balance?" + query + "&snapshot_revision=" + revision + "&cursor=" + url.QueryEscape(cursor)
	nextPage := call(next, 200)
	if string(nextPage["rows"]) == string(v["rows"]) {
		t.Fatal("repeated page")
	}
	sql(t, c, "UPDATE ledger.book SET revision=revision+1")
	conflict := call(next, 409)
	if string(conflict["code"]) != `"report_snapshot_changed"` {
		t.Fatal("wrong conflict")
	}
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE permission='read'")
	call(next, 403)
}
