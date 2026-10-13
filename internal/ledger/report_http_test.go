package ledger

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// reportOperations records the verified identity and SQL payload at the transport seam.
type reportOperations struct{ operationsStub }

// FinancialReport reuses the recording stub without implementing accounting in tests.
func (s *reportOperations) FinancialReport(ctx context.Context, c Command) (Result, error) {
	return s.Post(ctx, c)
}

// cursorFixture keeps transport syntax explicit; real SQL membership has separate proof.
func cursorFixture(extra string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(`{"version":"financial-reporting-v1","report":"trial-balance","layout":"account-v1","revision":"1","filter_hash":"abcd","last_key":["100","id"]` + extra + `}`))
}

// TestReportingHTTPContract proves malformed transport cannot reach SQL or leak results.
func TestReportingHTTPContract(t *testing.T) {
	path := "/v1/entities/e/books/b/reports/trial-balance"
	good := `{"contract_version":"financial-reporting-sql-v1","report":"trial-balance","next_cursor":null,"totals":{"total_debits":"0.01"}}`
	for _, tc := range []struct {
		name, method, query, body, token, response string
		err                                        error
		status, calls                              int
	}{
		{name: "first", query: "through_date=2026-01-31", token: "ok", status: 200, calls: 1},
		{name: "cursor", query: "through_date=2026-01-31&snapshot_revision=1&cursor=" + cursorFixture(""), token: "ok", status: 200, calls: 1},
		{name: "unauthenticated", query: "bad=%zz", status: 401},
		{name: "unknown", query: "actor=forged", token: "ok", status: 400},
		{name: "duplicate", query: "limit=1&limit=2", token: "ok", status: 400},
		{name: "bad_escape", query: "through_date=%zz", token: "ok", status: 400},
		{name: "semicolon", query: "through_date=2026-01-31;limit=1", token: "ok", status: 400},
		{name: "body", body: "{}", token: "ok", status: 400},
		{name: "head", method: "HEAD", token: "ok", status: 400},
		{name: "limit", query: "limit=01", token: "ok", status: 400},
		{name: "oversize", query: "cursor=" + strings.Repeat("a", 4096), token: "ok", status: 400},
		{name: "cursor_bound", query: "snapshot_revision=1&cursor=" + strings.Repeat("a", 2049), token: "ok", status: 400},
		{name: "duplicate_cursor", query: "snapshot_revision=1&cursor=" + cursorFixture(`,"revision":"1"`), token: "ok", status: 400},
		{name: "cursor_padding", query: "snapshot_revision=1&cursor=" + cursorFixture("") + "=", token: "ok", status: 400},
		{name: "cursor_crlf", query: "snapshot_revision=1&cursor=" + cursorFixture("") + "%0D%0A", token: "ok", status: 400},
		{name: "cursor_revision", query: "snapshot_revision=2&cursor=" + cursorFixture(""), token: "ok", status: 400},
		{name: "cursor_unknown", query: "snapshot_revision=1&cursor=" + cursorFixture(`,"extra":1`), token: "ok", status: 400},
		{name: "conflict", token: "ok", err: &Failure{409, "report_snapshot_changed"}, status: 409, calls: 1},
		{name: "redaction", token: "ok", err: errors.New("private SQL rows"), status: 500, calls: 1},
		{name: "wrong_contract", token: "ok", response: `{"contract_version":"future","report":"trial-balance","next_cursor":null}`, status: 500, calls: 1},
		{name: "missing_cursor", token: "ok", response: `{"contract_version":"financial-reporting-sql-v1","report":"trial-balance"}`, status: 500, calls: 1},
		{name: "wrong_report", token: "ok", response: `{"contract_version":"financial-reporting-sql-v1","report":"balance-sheet","next_cursor":null}`, status: 500, calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := tc.response
			if response == "" {
				response = good
			}
			ops := &reportOperations{operationsStub{result: Result{Status: 200, Body: []byte(response)}, err: tc.err}}
			method := tc.method
			if method == "" {
				method = "GET"
			}
			r := httptest.NewRequest(method, path+"?"+tc.query, strings.NewReader(tc.body))
			if tc.token != "" {
				r.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			Handler(authStub{}, ops).ServeHTTP(w, r)
			if w.Code != tc.status || ops.calls != tc.calls {
				t.Fatalf("status%d calls%d body%s", w.Code, ops.calls, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "private") {
				t.Fatal("unsafe response")
			}
			if tc.status == 200 {
				if !strings.Contains(w.Body.String(), `"total_debits":"0.01"`) {
					t.Fatal("money changed")
				}
				if ops.command.Actor != "00000000-0000-4000-8000-000000000001" {
					t.Fatal("untrusted actor")
				}
			}
		})
	}

	// Reject case aliases and replacement-prone JSON before any SQL mapping.
	for _, raw := range []string{
		`{"VERSION":"financial-reporting-v1","report":"trial-balance","layout":"account-v1","revision":"1","filter_hash":"abcd","last_key":["100","id"]}`,
		`{"version":"financial-reporting-v1","report":"trial-balance","layout":"posting-v1","revision":"1","filter_hash":"abcd","last_key":["100","id"]}`,
		`{"version":"financial-reporting-v1","report":"trial-balance","layout":"account-v1","revision":"1","filter_hash":"abcd","last_key":["\ud800","id"]}`,
		string([]byte{255}),
		`{} {}`,
	} {
		if _, err := decodeReportCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)), "trial-balance", "1"); err == nil {
			t.Fatal("invalid cursor accepted")
		}
	}
	if _, err := reportResponse([]byte(strings.Repeat("x", (2<<20)+1)), "trial-balance"); err == nil {
		t.Fatal("oversized response accepted")
	}

	for _, escaped := range []string{`\uD000`, `\ud83d\ude00`} {
		raw := `{"version":"financial-reporting-v1","report":"trial-balance","layout":"account-v1","revision":"1","filter_hash":"abcd","last_key":["` + escaped + `","id"]}`
		if _, err := decodeReportCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)), "trial-balance", "1"); err != nil {
			t.Fatal("valid Unicode escape rejected")
		}
	}
	// Round-trip the SQL cursor without allowing envelope metadata to become authority.
	body, err := reportResponse([]byte(`{"contract_version":"financial-reporting-sql-v1","report":"trial-balance","next_cursor":{"revision":"1","filter_hash":"abcd","last_key":["100","id"]}}`), "trial-balance")
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]json.RawMessage
	_ = json.Unmarshal(body, &v)
	var encoded string
	_ = json.Unmarshal(v["next_cursor"], &encoded)
	r := httptest.NewRequest("GET", path+"?snapshot_revision=1&cursor="+url.QueryEscape(encoded), nil)
	raw, err := reportRequest(r, "trial-balance")
	if err != nil || !strings.Contains(raw, `"last_key":["100","id"]`) {
		t.Fatal("cursor roundtrip")
	}
	if _, err = reportRequest(r, "balance-sheet"); err == nil {
		t.Fatal("cross-report cursor accepted")
	}
	r = httptest.NewRequest("GET", path, nil)
	r.TransferEncoding = []string{"chunked"}
	if _, err = reportRequest(r, "trial-balance"); err == nil {
		t.Fatal("chunked body accepted")
	}
}

// TestReportingHTTPResponseBounds rejects oversized wire expansion before writing rows.
func TestReportingHTTPResponseBounds(t *testing.T) {
	base := `{"contract_version":"financial-reporting-sql-v1","report":"trial-balance","next_cursor":{"revision":"1","filter_hash":"abcd","last_key":["100","id"]},"rows":[],"padding":""}`
	expanded := strings.Replace(base, `"padding":""`, `"padding":"`+strings.Repeat("x", (2<<20)-len(base)-1)+`"`, 1)
	for _, body := range []string{strings.Repeat("x", (2<<20)+1), expanded} {
		ops := &reportOperations{operationsStub{result: Result{Status: 200, Body: []byte(body)}}}
		r := httptest.NewRequest("GET", "/v1/entities/e/books/b/reports/trial-balance", nil)
		r.Header.Set("Authorization", "Bearer ok")
		w := httptest.NewRecorder()
		Handler(authStub{}, ops).ServeHTTP(w, r)
		if w.Code != 422 || strings.Contains(w.Body.String(), `"rows"`) || !strings.Contains(w.Body.String(), "result_out_of_range") {
			t.Fatal("oversized result was exposed", w.Code)
		}
	}
}
