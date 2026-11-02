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
