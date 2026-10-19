package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const connectionActor = "00000000-0000-4000-8000-000000000001"
const connectionEntity = "00000000-0000-4000-8000-000000000002"
const connectionBook = "00000000-0000-4000-8000-000000000003"
const connectionAttempt = "00000000-0000-4000-8000-000000000004"
const connectionBase = "/v1/entities/" + connectionEntity + "/books/" + connectionBook

// connectionAuth deliberately supplies a synthetic actor; provider identity verification is separate.
type connectionAuth struct{}

// Authenticate accepts only this test's synthetic bearer credential.
func (connectionAuth) Authenticate(_ context.Context, token string) (string, error) {
	if token != "synthetic" {
		return "", errors.New("private identity detail")
	}
	return connectionActor, nil
}

// connectionStub records trusted scope and invokes no SQL or external service.
type connectionStub struct {
	calls     int
	operation string
	scope     connections.Scope
	err       error
}

// record makes routing and absence of unauthorized calls observable.
func (s *connectionStub) record(scope connections.Scope, op string) (json.RawMessage, error) {
	s.calls++
	s.scope = scope
	s.operation = op
	return json.RawMessage(`{"state":"binding_review"}`), s.err
}

// Begin exposes synthetic ephemeral handoff only in its explicit response.
func (s *connectionStub) Begin(_ context.Context, scope connections.Scope, _ string, _ connections.BeginRequest) (connections.BeginResult, error) {
	raw, e := s.record(scope, "begin")
	return connections.BeginResult{Attempt: raw, Nonce: connectionAttempt, LinkToken: "synthetic-link"}, e
}

// Execute represents the one-shot trusted service boundary.
func (s *connectionStub) Execute(_ context.Context, scope connections.Scope, _ string, _ connections.ExecuteRequest) (json.RawMessage, error) {
	return s.record(scope, "execute")
}

// CompleteHosted checks the scoped saved Link handoff without browser tokens.
func (s *connectionStub) CompleteHosted(_ context.Context, scope connections.Scope, _ string) (json.RawMessage, error) {
	return s.record(scope, "complete")
}

// Reconcile represents status recovery without a new provider mutation.
func (s *connectionStub) Reconcile(_ context.Context, scope connections.Scope, _ string) (json.RawMessage, error) {
	return s.record(scope, "reconcile")
}

// Attempt returns safe metadata for the authenticated scope.
func (s *connectionStub) Attempt(_ context.Context, scope connections.Scope, _ string) (json.RawMessage, error) {
	return s.record(scope, "attempt")
}

// List records the scoped collection route.
func (s *connectionStub) List(_ context.Context, scope connections.Scope) (json.RawMessage, error) {
	return s.record(scope, "list")
}

// Bind records explicit selection, leaving financial checks to SQL proof.
func (s *connectionStub) Bind(_ context.Context, scope connections.Scope, _, _ string, _ connections.BindingRequest) (json.RawMessage, error) {
	return s.record(scope, "bind")
}

// Disconnect records an account-scoped local operation.
func (s *connectionStub) Disconnect(_ context.Context, scope connections.Scope, _, _, _ string) (json.RawMessage, error) {
	return s.record(scope, "disconnect")
}

// Attempts records authenticated discovery without handoff secrets.
func (s *connectionStub) Attempts(_ context.Context, scope connections.Scope, _ int, _ *string) (json.RawMessage, error) {
	return s.record(scope, "attempts")
}

// StreamStatus records a safe current publication read.
func (s *connectionStub) StreamStatus(_ context.Context, scope connections.Scope, _ string) (json.RawMessage, error) {
	return s.record(scope, "status")
}

// OperateStream records a pinned single-stream control.
func (s *connectionStub) OperateStream(_ context.Context, scope connections.Scope, _, _ string, _ connections.StreamOperation) (json.RawMessage, error) {
	return s.record(scope, "operate")
}

// connectionTestServer composes the production origin, admission and route boundaries.
func connectionTestServer(t *testing.T, ops ConnectionOperations) *Server {
	t.Helper()
	s, e := New(Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic"}, Dependencies{Auth: connectionAuth{}, Operations: syntheticOps{}, Connections: ops, Ready: func(context.Context) error { return nil }})
	if e != nil {
		t.Fatal(e)
	}
	return s
}

// connectionRequest sets only synthetic headers and captures the actual server response.
func connectionRequest(s *Server, method, path, body string, change func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer synthetic")
	r.Header.Set("Origin", "http://localhost:3000")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "synthetic-key")
	if change != nil {
		change(r)
	}
	w := httptest.NewRecorder()
	s.HTTP.Handler.ServeHTTP(w, r)
	return w
}

// TestConnectionHTTPBoundary checks all operations through the optional production mount.
func TestConnectionHTTPBoundary(t *testing.T) {
	ops := &connectionStub{}
	s := connectionTestServer(t, ops)
	for _, tc := range []struct{ method, path, body, operation string }{
		{"POST", "/connection-attempts", `{"mode":"new","sync_item_id":null,"expected_generation":null,"approved_stream_ids":[]}`, "begin"},
		{"GET", "/connection-attempts/" + connectionAttempt, "", "attempt"},
		{"POST", "/connection-attempts/" + connectionAttempt + "/execute", `{"nonce":"` + connectionAttempt + `","link_token":"synthetic-link","public_token":"synthetic-public"}`, "execute"},
		{"POST", "/connection-attempts/" + connectionAttempt + "/complete", `{}`, "complete"},
		{"POST", "/connection-attempts/" + connectionAttempt + "/reconcile", `{}`, "reconcile"},
		{"POST", "/connection-attempts/" + connectionAttempt + "/bindings", `{"consent_grants":true,"bindings":[]}`, "bind"},
		{"GET", "/connections", "", "list"},
		{"GET", "/connection-attempts?limit=100&before=" + connectionAttempt, "", "attempts"},
		{"GET", "/connections/streams/" + connectionAttempt + "/status", "", "status"},
		{"POST", "/connections/streams/" + connectionAttempt + "/operations", `{"action":"request","expected_item_generation":"1","expected_binding_revision":"1","expected_stream_revision":"1"}`, "operate"},
		{"POST", "/connections/streams/" + connectionAttempt + "/disconnect", `{"expected_binding_revision":"1"}`, "disconnect"},
	} {
		before := ops.calls
		w := connectionRequest(s, tc.method, connectionBase+tc.path, tc.body, nil)
		if w.Code != 200 || ops.calls != before+1 || ops.operation != tc.operation || ops.scope != (connections.Scope{Actor: connectionActor, Entity: connectionEntity, Book: connectionBook}) {
			t.Fatalf("%s: status=%d operation=%s", tc.operation, w.Code, ops.operation)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable connection data")
		}
	}
	absent := connectionTestServer(t, nil)
	w := connectionRequest(absent, "GET", connectionBase+"/connections", "", nil)
	if w.Code != 404 {
		t.Fatal("connection enabled without dependency")
	}
}

// unreadBody proves rejected authentication cannot cause token-body parsing.
type unreadBody struct{ t *testing.T }

// Read fails the test if unauthorized requests touch private handoff bytes.
func (b unreadBody) Read([]byte) (int, error) { b.t.Fatal("unauthorized body read"); return 0, io.EOF }

// Close satisfies the request body contract without side effects.
func (unreadBody) Close() error { return nil }

// TestConnectionHTTPFailures checks strict input, current denial forwarding and error redaction.
func TestConnectionHTTPFailures(t *testing.T) {
	for _, tc := range []struct {
		name, method, suffix, body string
		status                     int
		change                     func(*http.Request)
	}{
		{"no auth", "POST", "/connection-attempts", `{}`, 401, func(r *http.Request) { r.Header.Del("Authorization"); r.Body = unreadBody{t} }},
		{"origin", "POST", "/connection-attempts", `{}`, 403, func(r *http.Request) { r.Header.Set("Origin", "https://foreign.invalid") }},
		{"double auth", "POST", "/connection-attempts", `{}`, 401, func(r *http.Request) { r.Header.Add("Authorization", "Bearer synthetic") }},
		{"query", "GET", "/connections?secret=synthetic", "", 400, nil},
		{"discovery duplicate", "GET", "/connection-attempts?limit=1&limit=2", "", 400, nil},
		{"discovery unknown", "GET", "/connection-attempts?actor=other", "", 400, nil},
		{"discovery overflow", "GET", "/connection-attempts?limit=101", "", 400, nil},
		{"discovery malformed escape", "GET", "/connection-attempts?before=%xx", "", 400, nil},
		{"discovery empty cursor", "GET", "/connection-attempts?before=", "", 400, nil},
		{"discovery no auth", "GET", "/connection-attempts", "", 401, func(r *http.Request) { r.Header.Del("Authorization") }},
		{"head", "HEAD", "/connections", "", 405, nil},
		{"get body", "GET", "/connections", `{}`, 400, nil},
		{"missing key", "POST", "/connection-attempts", `{}`, 400, func(r *http.Request) { r.Header.Del("Idempotency-Key") }},
		{"content type", "POST", "/connection-attempts", `{}`, 400, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }},
		{"unknown", "POST", "/connection-attempts", `{"actor":"other"}`, 400, nil},
		{"duplicate", "POST", "/connection-attempts", `{"mode":"new","mode":"rotate"}`, 400, nil},
		{"nested duplicate", "POST", "/connection-attempts/" + connectionAttempt + "/bindings", `{"bindings":[{"account_id":"a","account_id":"b"}]}`, 400, nil},
		{"null", "POST", "/connection-attempts", `null`, 400, nil},
		{"trailing", "POST", "/connection-attempts", `{} {}`, 400, nil},
		{"large", "POST", "/connection-attempts", strings.Repeat(" ", 32<<10) + "{}", 413, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ops := &connectionStub{}
			s := connectionTestServer(t, ops)
			w := connectionRequest(s, tc.method, connectionBase+tc.suffix, tc.body, tc.change)
			if w.Code != tc.status || ops.calls != 0 {
				t.Fatalf("status=%d calls=%d", w.Code, ops.calls)
			}
		})
	}
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{{&connections.Failure{Code: "permission_denied"}, 403, "permission_denied"}, {&connections.Failure{Code: "commit_unknown"}, 503, "commit_unknown"}, {&connections.Failure{Code: "credential_recovery_required"}, 409, "credential_recovery_required"}, {errors.New("synthetic-private-token"), 500, "internal_error"}, {&connections.Failure{Code: "synthetic-private-token"}, 500, "internal_error"}} {
		ops := &connectionStub{err: tc.err}
		w := connectionRequest(connectionTestServer(t, ops), "GET", connectionBase+"/connections", "", nil)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "synthetic-private-token") {
			t.Fatal("unsafe failure projection")
		}
	}
}

// Abandon models the explicit SQL-only terminal recovery transition.
func (s *connectionStub) Abandon(_ context.Context, scope connections.Scope, _ string) (json.RawMessage, error) {
	return s.record(scope, "abandon")
}

// TestAbandonAcknowledgement rejects omission, false and unknown fields before any service call.
func TestAbandonAcknowledgement(t *testing.T) {
	for _, body := range []string{`{}`, `{"confirm_unrecoverable":false}`, `{"confirm_unrecoverable":true,"extra":true}`, `{"confirm_unrecoverable":true}`} {
		ops := &connectionStub{}
		request := httptest.NewRequest("POST", connectionBase+"/connection-attempts/"+connectionAttempt+"/abandon", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer synthetic")
		request.Header.Set("Content-Type", "application/json")
		request.SetPathValue("entity", connectionEntity)
		request.SetPathValue("book", connectionBook)
		request.SetPathValue("attempt", connectionAttempt)
		response := httptest.NewRecorder()
		connectionEndpoint(connectionAuth{}, ops, "abandon")(response, request)
		want := 400
		if body == `{"confirm_unrecoverable":true}` {
			want = 200
		}
		if response.Code != want || (want == 400 && ops.calls != 0) || (want == 200 && ops.operation != "abandon") {
			t.Fatal("acknowledgement boundary", response.Code, ops.calls)
		}
	}
}
