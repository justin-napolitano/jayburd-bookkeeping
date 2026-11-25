package autopost

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

const httpEntity = "f1000000-0000-4000-8000-000000000001"
const httpBook = "f1000000-0000-4000-8000-000000000002"
const httpBase = "/v1/entities/" + httpEntity + "/books/" + httpBook + "/automatic-accounting"

type boundaryAuth struct{}

func (boundaryAuth) Authenticate(_ context.Context, token string) (string, error) {
	if token != "synthetic-owner" {
		return "", errors.New("PRIVATE_SENTINEL")
	}
	return "f1000000-0000-4000-8000-000000000003", nil
}

// boundaryOperations observes raw transport only; it does not replace restricted SQL acceptance.
type boundaryOperations struct {
	calls     int
	operation string
	request   Request
	body      string
	err       error
}

func (o *boundaryOperations) call(r Request, operation string) (json.RawMessage, error) {
	o.calls++
	o.request, o.operation = r, operation
	if o.body != "" || o.err != nil {
		return json.RawMessage(o.body), o.err
	}
	return json.RawMessage(boundaryResponse(operation)), nil
}
func (o *boundaryOperations) GetPolicy(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "get")
}
func (o *boundaryOperations) SetPolicy(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "set")
}
func (o *boundaryOperations) Status(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "status")
}
func boundaryResponse(operation string) string {
	if operation == "status" {
		return `{"profile":"automatic-accounting-status-v1","entity_id":"` + httpEntity + `","book_id":"` + httpBook + `","enabled":false,"policy_revision":"0","pending_count":"0","attention_count":"0","last_operation":null,"holds":[],"holds_truncated":false,"blocked_reason":null}`
	}
	return `{"profile":"automatic-accounting-policy-v1","entity_id":"` + httpEntity + `","book_id":"` + httpBook + `","revision":"0","enabled":false,"policy":null,"supported_profile":"source-identity-usd-v1"}`
}
func boundaryHTTPResponse(operation string, runtime bool) string {
	if operation != "status" {
		return boundaryResponse(operation)
	}
	var value map[string]any
	_ = json.Unmarshal([]byte(boundaryResponse(operation)), &value)
	value["runtime_enabled"] = runtime
	raw, _ := json.Marshal(value)
	return string(raw)
}
func boundaryRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer synthetic-owner")
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}
func checkBoundaryProblem(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var body map[string]json.RawMessage
	if response.Code != status || json.Unmarshal(response.Body.Bytes(), &body) != nil || len(body) != 5 {
		t.Fatalf("problem framing status%d expected%d: %s", response.Code, status, response.Body.String())
	}
	var actualCode, requestID string
	_ = json.Unmarshal(body["code"], &actualCode)
	_ = json.Unmarshal(body["request_id"], &requestID)
	if actualCode != code || !pathID.MatchString(requestID) || strings.Contains(response.Body.String(), "PRIVATE_SENTINEL") {
		t.Fatal("problem revealed diagnostics or lacked a fixed code/correlation ID")
	}
	if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("private response caching boundary changed")
	}
}

func TestAutomaticAccountingHTTPRoutes(t *testing.T) {
	for _, tc := range []struct{ method, suffix, operation string }{{"GET", "", "get"}, {"POST", "", "set"}, {"GET", "/status", "status"}} {
		t.Run(tc.operation, func(t *testing.T) {
			operations := &boundaryOperations{}
			raw := ""
			if tc.method == "POST" {
				// The original bytes go to SQL; the browser cannot supply trusted actor or evaluation results.
				raw = " {\"key\":\"synthetic-policy\"} \n"
			}
			w := httptest.NewRecorder()
			Handler(boundaryAuth{}, operations, false).ServeHTTP(w, boundaryRequest(tc.method, httpBase+tc.suffix, raw))
			if w.Code != 200 || w.Body.String() != boundaryHTTPResponse(tc.operation, false) || operations.calls != 1 || operations.operation != tc.operation || operations.request.Actor != "f1000000-0000-4000-8000-000000000003" || operations.request.Entity != httpEntity || operations.request.Book != httpBook || operations.request.Body != raw {
				t.Fatalf("route changed trusted identity/raw request/status: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAutomaticAccountingHTTPRuntimeSignal(t *testing.T) {
	for _, runtime := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			body := strings.Replace(boundaryResponse("status"), `"enabled":false`, `"enabled":true`, 1)
			if !enabled {
				body = boundaryResponse("status")
			}
			w := httptest.NewRecorder()
			Handler(boundaryAuth{}, &boundaryOperations{body: body}, runtime).ServeHTTP(w, boundaryRequest("GET", httpBase+"/status", ""))
			var value struct {
				Runtime bool `json:"runtime_enabled"`
				Enabled bool `json:"enabled"`
			}
			if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &value) != nil || value.Runtime != runtime || value.Enabled != enabled {
				t.Fatal("runtime flag was confused with policy enablement")
			}
		}
	}
}

type unreadPolicyBody struct{ read bool }

func (b *unreadPolicyBody) Read([]byte) (int, error) {
	b.read = true
	return 0, errors.New("PRIVATE_SENTINEL")
}
func (b *unreadPolicyBody) Close() error { return nil }

func TestAutomaticAccountingHTTPFraming(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*http.Request)
		status int
		code   string
	}{
		{"missing-auth", func(r *http.Request) { r.Header.Del("Authorization") }, 401, "unauthenticated"},
		{"wrong-auth", func(r *http.Request) { r.Header.Set("Authorization", "Bearer denied") }, 401, "unauthenticated"},
		{"duplicate-auth", func(r *http.Request) { r.Header.Add("Authorization", "Bearer synthetic-owner") }, 401, "unauthenticated"},
		{"empty-bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") }, 401, "unauthenticated"},
		{"cookie-only", func(r *http.Request) { r.Header.Del("Authorization"); r.Header.Set("Cookie", "owner=synthetic") }, 401, "unauthenticated"},
		{"query", func(r *http.Request) { r.URL.RawQuery = "actor=PRIVATE_SENTINEL" }, 400, "invalid_automatic_accounting"},
		{"empty-query", func(r *http.Request) { r.URL.ForceQuery = true }, 400, "invalid_automatic_accounting"},
		{"key-header", func(r *http.Request) { r.Header.Set("Idempotency-Key", "synthetic") }, 400, "invalid_automatic_accounting"},
		{"encoding", func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, 400, "invalid_automatic_accounting"},
		{"media-type", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, 415, "unsupported_media_type"},
		{"charset", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=utf-8") }, 415, "unsupported_media_type"},
		{"duplicate-type", func(r *http.Request) { r.Header.Add("Content-Type", "application/json") }, 415, "unsupported_media_type"},
		{"known-size", func(r *http.Request) { r.ContentLength = policyBodyLimit + 1 }, 413, "request_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operations := &boundaryOperations{}
			r := boundaryRequest("POST", httpBase, "{}")
			body := &unreadPolicyBody{}
			r.Body = body
			tc.change(r)
			w := httptest.NewRecorder()
			Handler(boundaryAuth{}, operations, false).ServeHTTP(w, r)
			checkBoundaryProblem(t, w, tc.status, tc.code)
			if body.read || operations.calls != 0 {
				t.Fatal("invalid framing reached private body/SQL")
			}
		})
	}
}

func TestAutomaticAccountingHTTPBoundedBodiesAndSelectors(t *testing.T) {
	for _, tc := range []struct {
		method, path, body string
		length             int64
		status             int
	}{
		{"POST", httpBase, strings.Repeat(" ", int(policyBodyLimit)+1), -1, 413},
		{"POST", httpBase, string([]byte{0xff}), -1, 400},
		{"GET", httpBase, "x", 0, 400},
		{"GET", httpBase + "/status", "{}", 2, 400},
		{"GET", strings.Replace(httpBase, httpEntity, "00000000-0000-0000-0000-000000000000", 1), "", 0, 400},
		{"GET", strings.Replace(httpBase, httpBook, "not-a-book", 1), "", 0, 400},
	} {
		operations := &boundaryOperations{}
		r := boundaryRequest(tc.method, tc.path, tc.body)
		r.ContentLength = tc.length
		w := httptest.NewRecorder()
		Handler(boundaryAuth{}, operations, false).ServeHTTP(w, r)
		if w.Code != tc.status || operations.calls != 0 {
			t.Fatalf("unsafe body/selectors: status%d expected%d", w.Code, tc.status)
		}
	}
	for _, suffix := range []string{"/run", "/prepare", "/execute", "/acknowledge"} {
		operations := &boundaryOperations{}
		w := httptest.NewRecorder()
		Handler(boundaryAuth{}, operations, false).ServeHTTP(w, boundaryRequest("POST", httpBase+suffix, "{}"))
		if w.Code != 404 || operations.calls != 0 {
			t.Fatal("public automatic execution route exposed")
		}
	}
	operations := &boundaryOperations{}
	w := httptest.NewRecorder()
	Handler(boundaryAuth{}, operations, false).ServeHTTP(w, boundaryRequest("HEAD", httpBase, ""))
	if w.Code != 405 || operations.calls != 0 {
		t.Fatal("HEAD reached policy read")
	}
}

func TestAutomaticAccountingHTTPErrorsAndResponseClosure(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{&ledger.Failure{Status: 400, Code: "invalid_automatic_accounting"}, 400, "invalid_automatic_accounting"},
		{&ledger.Failure{Status: 404, Code: "automatic_accounting_denied"}, 404, "automatic_accounting_denied"},
		{&ledger.Failure{Status: 409, Code: "automatic_accounting_conflict"}, 409, "automatic_accounting_conflict"},
		{&ledger.Failure{Status: 409, Code: "automatic_accounting_snapshot_changed"}, 409, "automatic_accounting_snapshot_changed"},
		{&ledger.Failure{Status: 422, Code: "automatic_accounting_limit"}, 422, "automatic_accounting_limit"},
		{errors.New("PRIVATE_SENTINEL"), 503, "temporarily_unavailable"},
		{&ledger.Failure{Status: 400, Code: "PRIVATE_SENTINEL"}, 503, "temporarily_unavailable"},
	} {
		w := httptest.NewRecorder()
		Handler(boundaryAuth{}, &boundaryOperations{err: tc.err}, false).ServeHTTP(w, boundaryRequest("GET", httpBase, ""))
		checkBoundaryProblem(t, w, tc.status, tc.code)
	}
	for _, body := range []string{`{}`, strings.Replace(boundaryResponse("get"), httpBook, httpEntity, 1), strings.Replace(boundaryResponse("get"), `"enabled":false`, `"enabled":false,"extra":"PRIVATE_SENTINEL"`, 1)} {
		w := httptest.NewRecorder()
		Handler(boundaryAuth{}, &boundaryOperations{body: body}, false).ServeHTTP(w, boundaryRequest("GET", httpBase, ""))
		checkBoundaryProblem(t, w, 503, "temporarily_unavailable")
	}
}
