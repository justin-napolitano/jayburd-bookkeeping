package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// registrationStub records transport identity only; restricted PostgreSQL owns actual authorization proof.
type registrationStub struct {
	connectionStub
	attempt, key, request string
	input                 connections.RegistrationSelection
}

// RegistrationOptions records only the authenticated scope and owned attempt, with no client-selected policy.
func (s *registrationStub) RegistrationOptions(_ context.Context, scope connections.Scope, id string) (json.RawMessage, error) {
	s.attempt = id
	return s.record(scope, "registration_options")
}

// TestRegistrationOptionsHTTPBoundary rejects alternate authority, query and request bodies before SQL.
func TestRegistrationOptionsHTTPBoundary(t *testing.T) {
	path := connectionBase + "/connection-attempts/" + connectionAttempt + "/registration"
	for _, tc := range []struct {
		name, method, path, body string
		change                   func(*http.Request)
		status, calls            int
	}{
		{name: "valid", method: "GET", path: path, status: 200, calls: 1},
		{name: "query", method: "GET", path: path + "?actor=spoof", status: 400},
		{name: "body", method: "GET", path: path, body: `{"actor":"spoof"}`, status: 400},
		{name: "head", method: "HEAD", path: path, status: 405},
		{name: "identity", method: "GET", path: path, status: 401, change: func(r *http.Request) { r.Header.Del("Authorization") }},
		{name: "origin", method: "GET", path: path, status: 403, change: func(r *http.Request) { r.Header.Set("Origin", "https://foreign.invalid") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &registrationStub{}
			w := connectionRequest(connectionTestServer(t, s), tc.method, tc.path, tc.body, tc.change)
			if w.Code != tc.status || s.calls != tc.calls || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("options status%d calls%d", w.Code, s.calls)
			}
			if tc.calls > 0 && (s.scope.Actor != connectionActor || s.attempt != connectionAttempt || s.operation != "registration_options") {
				t.Fatal("options identity changed")
			}
		})
	}
	if w := connectionRequest(connectionTestServer(t, &connectionStub{}), "GET", path, "", nil); w.Code != 503 {
		t.Fatal("missing options capability enabled")
	}
}

// PrepareRegistration observes the optional service boundary without simulating SQL authorization.
func (s *registrationStub) PrepareRegistration(_ context.Context, scope connections.Scope, id, key, request string, in connections.RegistrationSelection) (json.RawMessage, error) {
	s.attempt = id
	s.key = key
	s.request = request
	s.input = in
	raw, err := s.record(scope, "prepare_registration")
	return raw, err
}

// TestRegistrationHTTPBoundary enforces verified scope, closed selection and server-owned audit identity.
func TestRegistrationHTTPBoundary(t *testing.T) {
	path := connectionBase + "/connection-attempts/" + connectionAttempt + "/registration"
	for _, tc := range []struct {
		name, path, body string
		change           func(*http.Request)
		status, calls    int
	}{
		{name: "valid", path: path, body: `{"account_ids":["one"]}`, status: 201, calls: 1, change: func(r *http.Request) { r.Header.Set("X-Request-ID", connectionActor) }},
		{name: "duplicate-key", path: path, body: `{"account_ids":["one"],"account_ids":["two"]}`, status: 400},
		{name: "spoof-actor", path: path, body: `{"account_ids":["one"],"actor":"spoof"}`, status: 400},
		{name: "spoof-mapping", path: path, body: `{"account_ids":["one"],"currency_version_id":"spoof"}`, status: 400},
		{name: "query", path: path + "?environment=production", body: `{"account_ids":["one"]}`, status: 400},
		{name: "oversize", path: path, body: strings.Repeat("x", (16<<10)+1), status: 413},
		{name: "key", path: path, body: `{"account_ids":["one"]}`, status: 400, change: func(r *http.Request) { r.Header.Del("Idempotency-Key") }},
		{name: "identity", path: path, body: `{}`, status: 401, change: func(r *http.Request) { r.Header.Del("Authorization") }},
		{name: "origin", path: path, body: `{}`, status: 403, change: func(r *http.Request) { r.Header.Set("Origin", "https://foreign.invalid") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &registrationStub{}
			w := connectionRequest(connectionTestServer(t, s), "POST", tc.path, tc.body, tc.change)
			if w.Code != tc.status || s.calls != tc.calls || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("registration boundary: status%d calls%d", w.Code, s.calls)
			}
			if tc.calls > 0 && (s.scope.Actor != connectionActor || s.attempt != connectionAttempt || s.key != "synthetic-key" || !connectionUUID.MatchString(s.request) || s.request == connectionActor || len(s.input.AccountIDs) != 1) {
				t.Fatal("request identity came from caller or selection changed")
			}
		})
	}
	w := connectionRequest(connectionTestServer(t, &connectionStub{}), "POST", path, `{"account_ids":["one"]}`, nil)
	if w.Code != 503 {
		t.Fatal("missing preparer silently enabled operation")
	}
}
