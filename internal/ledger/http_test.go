package ledger

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// authStub is an explicit synthetic identity adapter, never a production verifier.
type authStub struct{ err error }

// Authenticate lets transport tests distinguish failed identity from data access.
func (a authStub) Authenticate(context.Context, string) (string, error) {
	return "00000000-0000-4000-8000-000000000001", a.err
}

// operationsStub records transport calls without pretending to execute accounting.
type operationsStub struct {
	calls   int
	command Command
	result  Result
	err     error
}

// Post records the raw body to prove duplicate keys are not lost before SQL.
func (s *operationsStub) Post(_ context.Context, c Command) (Result, error) {
	s.calls++
	s.command = c
	return s.result, s.err
}

// Reverse shares the recording boundary for route-selection tests.
func (s *operationsStub) Reverse(ctx context.Context, c Command) (Result, error) {
	return s.Post(ctx, c)
}

// Journal shares the recording boundary for route-selection tests.
func (s *operationsStub) Journal(ctx context.Context, c Command) (Result, error) {
	return s.Post(ctx, c)
}

// TrialBalance shares the recording boundary for route-selection tests.
func (s *operationsStub) TrialBalance(ctx context.Context, c Command) (Result, error) {
	return s.Post(ctx, c)
}

// TestAPIContract verifies identity-first access, bounded raw input and redaction.
func TestAPIContract(t *testing.T) {
	path := "/v1/entities/e/books/b/journals"
	for _, tc := range []struct {
		name, token, key, body string
		authErr, errorValue    error
		status, calls          int
	}{
		{name: "no_token", key: "k", body: "{}", status: 401},
		{name: "bad_token", token: "bad", key: "k", body: "{}", authErr: errors.New("private identity detail"), status: 401},
		{name: "no_key", token: "ok", body: "{}", status: 400},
		{name: "oversize", token: "ok", key: "k", body: strings.Repeat("x", (1<<20)+1), status: 413},
		{name: "bad_utf8", token: "ok", key: "k", body: string([]byte{255}), status: 400},
		{name: "denied", token: "ok", key: "k", body: "{}", errorValue: &Failure{404, "not_found"}, status: 404, calls: 1},
		{name: "raw_failure", token: "ok", key: "k", body: "{}", errorValue: errors.New("private SQL payload"), status: 500, calls: 1},
		{name: "commit_unknown", token: "ok", key: "k", body: "{}", errorValue: &Failure{503, "temporarily_unavailable"}, status: 503, calls: 1},
		{name: "replay", token: "ok", key: "k", body: `{"duplicate":1,"duplicate":2}`, status: 201, calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ops := &operationsStub{result: Result{Status: 201, Body: []byte(`{"journal_id":"synthetic"}`), Replayed: true}, err: tc.errorValue}
			h := Handler(authStub{tc.authErr}, ops)
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(tc.body))
			if tc.token != "" {
				r.Header.Set("Authorization", "Bearer "+tc.token)
			}
			if tc.key != "" {
				r.Header.Set("Idempotency-Key", tc.key)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || ops.calls != tc.calls {
				t.Fatalf("status %d calls %d", w.Code, ops.calls)
			}
			if w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "private") {
				t.Fatal("unsafe response")
			}
			if tc.name == "replay" && (ops.command.Body != tc.body || w.Header().Get("Idempotent-Replay") != "true") {
				t.Fatal("transport rewrote request or lost replay header")
			}
			if tc.status == 503 && w.Header().Get("Retry-After") == "" {
				t.Fatal("missing retry guidance")
			}
		})
	}
	ops := &operationsStub{}
	w := httptest.NewRecorder()
	Handler(authStub{}, ops).ServeHTTP(w, httptest.NewRequest("POST", "/v1/entities/e/books/b/periods/p/close", nil))
	if w.Code != 404 || ops.calls != 0 {
		t.Fatal("deferred close route mounted")
	}
}
