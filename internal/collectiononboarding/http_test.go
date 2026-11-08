package collectiononboarding

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

type testAuth struct{}

func (testAuth) Authenticate(_ context.Context, token string) (string, error) {
	if token != "synthetic-owner" {
		return "", errors.New("private provider error")
	}
	return "verified-principal", nil
}

// testOperations records transport behavior only; real SQL remains the authority proof.
type testOperations struct {
	calls           int
	request         Request
	operation, body string
	err             error
}

func (o *testOperations) call(r Request, op string) (json.RawMessage, error) {
	o.calls++
	o.request = r
	o.operation = op
	if o.body != "" {
		return json.RawMessage(o.body), o.err
	}
	return json.RawMessage(result(op)), o.err
}
func (o *testOperations) Catalog(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "catalog")
}
func (o *testOperations) Preview(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "preview")
}
func (o *testOperations) Prepare(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "prepare")
}
func (o *testOperations) Pending(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "pending")
}
func (o *testOperations) Clear(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "clear")
}
func (o *testOperations) Transition(_ context.Context, r Request, op string) (json.RawMessage, error) {
	return o.call(r, op)
}
func httpRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer synthetic-owner")
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}
func problemResult(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var p map[string]json.RawMessage
	if w.Code != status || json.Unmarshal(w.Body.Bytes(), &p) != nil || len(p) != 5 || str(p["code"]) != code || !validID(str(p["request_id"])) || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unsafe problem status%d body%s", w.Code, w.Body.String())
	}
	if status == 401 && w.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatal("missing bearer challenge")
	}
	if strings.Contains(w.Body.String(), "sensitive") || strings.Contains(w.Body.String(), "private provider") {
		t.Fatal("private diagnostic escaped")
	}
}

// TestOnboardingHTTPRoutes exercises every frozen mounted route with trusted actor separation.
func TestOnboardingHTTPRoutes(t *testing.T) {
	for _, tc := range []struct{ method, path, operation string }{
		{"GET", "", "catalog"}, {"POST", "/previews", "preview"}, {"POST", "/commands", "prepare"}, {"GET", "/commands/pending", "pending"}, {"POST", "/recovery/clear", "clear"}, {"POST", "/commands/" + commandID + "/execute", "execute"}, {"POST", "/commands/" + commandID + "/cancel", "cancel"}, {"POST", "/commands/" + commandID + "/acknowledge", "acknowledge"},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			o := &testOperations{}
			body := ""
			if tc.method == "POST" {
				body = request(tc.operation).Body
			}
			r := httpRequest(tc.method, "/v1/collection-onboarding"+tc.path, body)
			w := httptest.NewRecorder()
			Handler(testAuth{}, o).ServeHTTP(w, r)
			status := 200
			if tc.operation == "prepare" {
				status = 201
			}
			if w.Code != status || w.Body.String() != result(tc.operation) || o.calls != 1 || o.request.Actor != "verified-principal" || o.request.Body != body || o.operation != tc.operation || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("route status%d body%s", w.Code, w.Body.String())
			}
			if tc.operation == "execute" || tc.operation == "cancel" || tc.operation == "acknowledge" {
				if o.request.Command != commandID {
					t.Fatal("path identity changed")
				}
			}
		})
	}
}

type unreadable struct{ read bool }

func (r *unreadable) Read([]byte) (int, error) {
	r.read = true
	return 0, errors.New("body must not be read")
}
func (r *unreadable) Close() error { return nil }

// TestOnboardingHTTPFraming proves authentication precedes malformed/private request parsing.
func TestOnboardingHTTPFraming(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*http.Request)
		status int
		code   string
	}{
		{"missing-auth", func(r *http.Request) { r.Header.Del("Authorization") }, 401, "unauthenticated"},
		{"wrong-auth", func(r *http.Request) { r.Header.Set("Authorization", "Bearer other") }, 401, "unauthenticated"},
		{"duplicate-auth", func(r *http.Request) { r.Header.Add("Authorization", "Bearer synthetic-owner") }, 401, "unauthenticated"},
		{"cookie-only", func(r *http.Request) { r.Header.Del("Authorization"); r.Header.Set("Cookie", "session=synthetic") }, 401, "unauthenticated"},
		{"query", func(r *http.Request) { r.URL.RawQuery = "actor=sensitive" }, 400, "invalid_onboarding"},
		{"empty-query", func(r *http.Request) { r.URL.ForceQuery = true }, 400, "invalid_onboarding"},
		{"key-header", func(r *http.Request) { r.Header.Set("Idempotency-Key", "unwanted") }, 400, "invalid_onboarding"},
		{"encoding", func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, 400, "invalid_onboarding"},
		{"missing-type", func(r *http.Request) { r.Header.Del("Content-Type") }, 415, "unsupported_media_type"},
		{"type-parameters", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=utf-8") }, 415, "unsupported_media_type"},
		{"duplicate-type", func(r *http.Request) { r.Header.Add("Content-Type", "application/json") }, 415, "unsupported_media_type"},
		{"oversize", func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", 8193)))
			r.ContentLength = 8193
		}, 413, "request_too_large"},
		{"chunked-oversize", func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(strings.Repeat("x", 8193)))
			r.ContentLength = -1
		}, 413, "request_too_large"},
		{"invalid-utf8", func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(string([]byte{0xff})))
			r.ContentLength = 1
		}, 400, "invalid_onboarding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &testOperations{}
			r := httpRequest("POST", "/v1/collection-onboarding/previews", `{"operation":"setup"}`)
			tc.change(r)
			w := httptest.NewRecorder()
			Handler(testAuth{}, o).ServeHTTP(w, r)
			problemResult(t, w, tc.status, tc.code)
			if o.calls != 0 {
				t.Fatal("bad framing reached service")
			}
		})
	}
	body := &unreadable{}
	r := httpRequest("POST", "/v1/collection-onboarding/commands/not-a-uuid/execute", "")
	r.Body = body
	r.Header.Del("Authorization")
	w := httptest.NewRecorder()
	Handler(testAuth{}, &testOperations{}).ServeHTTP(w, r)
	problemResult(t, w, 401, "unauthenticated")
	if body.read {
		t.Fatal("unauthenticated body was read")
	}
	for _, path := range []string{"", "/commands/pending"} {
		for _, method := range []string{"GET", "HEAD"} {
			o := &testOperations{}
			r := httpRequest(method, "/v1/collection-onboarding"+path, "")
			if method == "GET" {
				r.Body = io.NopCloser(strings.NewReader("x"))
				r.ContentLength = 1
			}
			w := httptest.NewRecorder()
			Handler(testAuth{}, o).ServeHTTP(w, r)
			if method == "HEAD" {
				problemResult(t, w, 405, "method_not_allowed")
			} else {
				problemResult(t, w, 400, "invalid_onboarding")
			}
			if o.calls != 0 {
				t.Fatal("unsupported GET/HEAD reached service")
			}
		}
	}
	r = httpRequest("POST", "/v1/collection-onboarding/commands/not-a-uuid/execute", "{}")
	w = httptest.NewRecorder()
	Handler(testAuth{}, &testOperations{}).ServeHTTP(w, r)
	problemResult(t, w, 400, "invalid_onboarding")
}

// TestOnboardingHTTPResponseFailure rejects unsafe service output/diagnostics with independent correlation IDs.
func TestOnboardingHTTPResponseFailure(t *testing.T) {
	ids := map[string]bool{}
	for _, o := range []*testOperations{{err: errors.New("sensitive raw SQL")}, {err: &ledger.Failure{Status: 200, Code: "onboarding_denied"}}, {err: &ledger.Failure{Status: 500, Code: "sensitive"}}, {body: `{"profile":"private-onboarding-catalog-v1","sensitive":"raw"}`}, {body: strings.Repeat(" ", 1<<20+1)}} {
		w := httptest.NewRecorder()
		Handler(testAuth{}, o).ServeHTTP(w, httpRequest("GET", "/v1/collection-onboarding", ""))
		problemResult(t, w, 503, "temporarily_unavailable")
		var p map[string]json.RawMessage
		_ = json.Unmarshal(w.Body.Bytes(), &p)
		id := str(p["request_id"])
		if ids[id] {
			t.Fatal("correlation identity reused")
		}
		ids[id] = true
	}
	w := httptest.NewRecorder()
	Handler(testAuth{}, nil).ServeHTTP(w, httpRequest("GET", "/v1/collection-onboarding", ""))
	problemResult(t, w, 503, "temporarily_unavailable")
}
