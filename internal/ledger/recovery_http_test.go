package ledger

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// recoveryStub observes transport only; real command authority is integration tested.
type recoveryStub struct {
	*operationsStub
	action string
}

// Recovery records the explicit action and untouched body for the SQL boundary.
func (s *recoveryStub) Recovery(ctx context.Context, c Command, action string) (Result, error) {
	s.action = action
	return s.Post(ctx, c)
}

// TestCommandHTTPBoundary covers origin/auth ordering, raw JSON and command route identity.
func TestCommandHTTPBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, key, origin, token, action string
		status                                               int
	}{
		{"register", "POST", "/v1/commands", `{"body":{"x":1,"x":2}}`, "registration", "https://synthetic.invalid", "Bearer synthetic", "register", 201},
		{"pending", "GET", "/v1/commands/pending", "", "", "https://synthetic.invalid", "Bearer synthetic", "pending", 200},
		{"clear", "POST", "/v1/commands/pending/clear", `{}`, "", "https://synthetic.invalid", "Bearer synthetic", "clear", 200},
		{"execute", "POST", "/v1/commands/synthetic-id/execute", `{}`, "", "https://synthetic.invalid", "Bearer synthetic", "execute", 200},
		{"cancel", "POST", "/v1/commands/synthetic-id/cancel", `{}`, "", "https://synthetic.invalid", "Bearer synthetic", "cancel", 200},
		{"acknowledge", "POST", "/v1/commands/synthetic-id/acknowledge", `{}`, "", "https://synthetic.invalid", "Bearer synthetic", "acknowledge", 200},
		{"foreign_origin", "POST", "/v1/commands/synthetic-id/execute", `{}`, "", "https://foreign.invalid", "Bearer synthetic", "", 403},
		{"missing_bearer", "GET", "/v1/commands/pending", "", "", "https://synthetic.invalid", "", "", 401},
		{"registration_key", "POST", "/v1/commands", `{}`, "", "https://synthetic.invalid", "Bearer synthetic", "", 400},
		{"pending_body", "GET", "/v1/commands/pending", `{}`, "", "https://synthetic.invalid", "Bearer synthetic", "", 400},
		{"query", "POST", "/v1/commands/synthetic-id/execute?actor=spoof", `{}`, "", "https://synthetic.invalid", "Bearer synthetic", "", 400},
		{"oversize", "POST", "/v1/commands/synthetic-id/execute", strings.Repeat("x", (1<<20)+1), "", "https://synthetic.invalid", "Bearer synthetic", "", 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ops := &recoveryStub{operationsStub: &operationsStub{result: Result{Status: tc.status, Body: []byte(`{}`)}}}
			h, err := BrowserBoundary("https://synthetic.invalid", Handler(authStub{}, ops))
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Origin", tc.origin)
			if tc.token != "" {
				r.Header.Set("Authorization", tc.token)
			}
			if tc.key != "" {
				r.Header.Set("Idempotency-Key", tc.key)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("recovery transport contract", w.Code)
			}
			if tc.action == "" {
				if ops.calls != 0 {
					t.Fatal("denied request reached SQL")
				}
				return
			}
			if ops.calls != 1 || ops.action != tc.action || ops.command.Body != tc.body || ops.command.Actor != "00000000-0000-4000-8000-000000000001" {
				t.Fatal("command transport changed")
			}
			if tc.action == "execute" || tc.action == "cancel" || tc.action == "acknowledge" {
				if ops.command.CommandID != "synthetic-id" || ops.command.Key != "" {
					t.Fatal("action identity changed")
				}
			}
		})
	}
	ops := &recoveryStub{operationsStub: &operationsStub{err: errors.New("private financial detail")}}
	r := httptest.NewRequest("GET", "/v1/commands/pending", nil)
	r.Header.Set("Authorization", "Bearer synthetic")
	w := httptest.NewRecorder()
	Handler(authStub{}, ops).ServeHTTP(w, r)
	if w.Code != 500 || strings.Contains(w.Body.String(), "private") || w.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatal("recovery error detail leaked")
	}
}
