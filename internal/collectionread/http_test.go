package collectionread

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

const workspace = "00000000-0000-4000-8000-000000000001"

// testAuth is a synthetic transport adapter, not proof of token verification or SQL authority.
type testAuth struct {
	denied bool
	calls  int
}

// Authenticate returns a fixed independently selected actor, never request-body identity.
func (a *testAuth) Authenticate(context.Context, string) (string, error) {
	a.calls++
	if a.denied {
		return "", errors.New("private auth failure")
	}
	return "verified-actor", nil
}

// testReader records the safe transport envelope without pretending to enforce database grants.
type testReader struct {
	calls     int
	request   Request
	operation string
	body      json.RawMessage
	err       error
}

// read captures exact text and supplies fixed synthetic results.
func (s *testReader) read(r Request, op string) (json.RawMessage, error) {
	s.calls++
	s.request = r
	s.operation = op
	return s.body, s.err
}

// Workspace makes the workspace read observable in the narrow transport fixture.
func (s *testReader) Workspace(_ context.Context, r Request) (json.RawMessage, error) {
	return s.read(r, "workspace")
}

// Sources makes the source enumeration observable in the narrow transport fixture.
func (s *testReader) Sources(_ context.Context, r Request) (json.RawMessage, error) {
	return s.read(r, "sources")
}

// Observations makes the observation query observable in the narrow transport fixture.
func (s *testReader) Observations(_ context.Context, r Request) (json.RawMessage, error) {
	return s.read(r, "observations")
}

// TestCollectionTransport authenticates first, bounds private POSTs and preserves raw SQL-owned JSON.
func TestCollectionTransport(t *testing.T) {
	for _, op := range []string{"sources", "observations"} {
		t.Run(op, func(t *testing.T) {
			body := ` {"search":"fictional","search":"duplicate","amount":"9007199254740993.01","actor":"foreign"} `
			a := &testAuth{}
			profile := map[string]string{"sources": "private-source-page-v1", "observations": "private-source-query-v1"}[op]
			s := &testReader{body: json.RawMessage(`{"profile":"` + profile + `","amount":"9007199254740993.01"}`)}
			r := httptest.NewRequest("POST", "/v1/collections/"+workspace+"/"+op+"/query", strings.NewReader(body))
			r.Header.Set("Authorization", "Bearer synthetic")
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			Handler(a, s).ServeHTTP(w, r)
			if w.Code != 200 || s.calls != 1 || s.request.Actor != "verified-actor" || s.request.Workspace != workspace || s.request.Body != body || s.operation != op || w.Body.String() != string(s.body) {
				t.Fatalf("transport changed authority or bytes: status%d calls%d", w.Code, s.calls)
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("private cache/encoding boundary")
			}
		})
	}
}

// TestCollectionTransportDenies prevents framing shortcuts before restricted operation invocation.
func TestCollectionTransportDenies(t *testing.T) {
	base := "/v1/collections/" + workspace + "/observations/query"
	for _, tc := range []struct {
		name, method, path, body, token, content, encoding, key string
		status                                                  int
	}{
		{"no_auth", "POST", base, "{}", "", "application/json", "", "", 401},
		{"query", "POST", base + "?search=private", "{}", "Bearer synthetic", "application/json", "", "", 400},
		{"empty_query", "POST", base + "?", "{}", "Bearer synthetic", "application/json", "", "", 400},
		{"content", "POST", base, "{}", "Bearer synthetic", "text/plain", "", "", 415},
		{"charset", "POST", base, "{}", "Bearer synthetic", "application/json; charset=utf-8", "", "", 415},
		{"encoding", "POST", base, "{}", "Bearer synthetic", "application/json", "gzip", "", 400},
		{"key", "POST", base, "{}", "Bearer synthetic", "application/json", "", "same-key", 400},
		{"oversize", "POST", base, strings.Repeat("x", 8193), "Bearer synthetic", "application/json", "", "", 413},
		{"nil_id", "POST", "/v1/collections/00000000-0000-0000-0000-000000000000/observations/query", "{}", "Bearer synthetic", "application/json", "", "", 400},
		{"head", "HEAD", "/v1/collection", "", "Bearer synthetic", "", "", "", 405},
		{"get_body", "GET", "/v1/collection", "{}", "Bearer synthetic", "", "", "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &testAuth{}
			s := &testReader{}
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.token != "" {
				r.Header.Set("Authorization", tc.token)
			}
			if tc.content != "" {
				r.Header.Set("Content-Type", tc.content)
			}
			if tc.encoding != "" {
				r.Header.Set("Content-Encoding", tc.encoding)
			}
			if tc.key != "" {
				r.Header.Set("Idempotency-Key", tc.key)
			}
			w := httptest.NewRecorder()
			Handler(a, s).ServeHTTP(w, r)
			if w.Code != tc.status || s.calls != 0 {
				t.Fatalf("status%d calls%d", w.Code, s.calls)
			}
			if tc.method == "HEAD" && w.Header().Get("Allow") != "GET" {
				t.Fatal("HEAD denial lost allowed method")
			}
		})
	}
	a := &testAuth{denied: true}
	s := &testReader{}
	r := httptest.NewRequest("POST", base, strings.NewReader("{}"))
	r.Header.Set("Authorization", "Bearer synthetic")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer second")
	w := httptest.NewRecorder()
	Handler(a, s).ServeHTTP(w, r)
	if w.Code != 401 || a.calls != 0 || s.calls != 0 {
		t.Fatal("duplicate auth reached resolver/SQL")
	}
	r = httptest.NewRequest("POST", base, strings.NewReader("{}"))
	r.Header.Set("Authorization", "Bearer synthetic")
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	Handler(a, s).ServeHTTP(w, r)
	if w.Code != 401 || s.calls != 0 || strings.Contains(w.Body.String(), "private auth") {
		t.Fatal("resolver denial disclosed or reached SQL")
	}
}

// TestCollectionSafeFailures never relays database diagnostics or an unavailable private body.
func TestCollectionSafeFailures(t *testing.T) {
	for _, tc := range []struct {
		err    error
		body   string
		status int
		code   string
	}{
		{&ledger.Failure{Status: 409, Code: "collection_snapshot_changed"}, `{"private":"value"}`, 409, "collection_snapshot_changed"},
		{errors.New("private SQL provider detail"), `{"private":"value"}`, 503, "temporarily_unavailable"},
		{&ledger.Failure{Status: 500, Code: "private diagnostic"}, `{"private":"value"}`, 503, "temporarily_unavailable"},
		{&ledger.Failure{Status: 200, Code: "collection_denied"}, `{"private":"value"}`, 503, "temporarily_unavailable"},
		{nil, `null`, 503, "temporarily_unavailable"},
		{nil, `[]`, 503, "temporarily_unavailable"},
		{nil, `{"profile":"private-source-query-v1"}`, 503, "temporarily_unavailable"},
		{nil, `not-json-private`, 503, "temporarily_unavailable"},
		{nil, strings.Repeat(" ", 1<<20) + "{}", 503, "temporarily_unavailable"},
	} {
		s := &testReader{err: tc.err, body: json.RawMessage(tc.body)}
		r := httptest.NewRequest("GET", "/v1/collection", nil)
		r.Header.Set("Authorization", "Bearer synthetic")
		w := httptest.NewRecorder()
		Handler(&testAuth{}, s).ServeHTTP(w, r)
		var problemBody struct {
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &problemBody); err != nil || !workspaceID.MatchString(problemBody.RequestID) {
			t.Fatal("invalid problem correlation UUID")
		}
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "private") {
			t.Fatalf("unsafe failure status%d", w.Code)
		}
	}
}
