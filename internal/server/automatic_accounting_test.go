package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/autopost"
)

type syntheticAccounting struct{ calls int }

func (s *syntheticAccounting) GetPolicy(_ context.Context, r autopost.Request) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{"profile":"automatic-accounting-policy-v1","entity_id":"` + r.Entity + `","book_id":"` + r.Book + `","revision":"0","enabled":false,"policy":null,"supported_profile":"source-identity-usd-v1"}`), nil
}
func (s *syntheticAccounting) SetPolicy(c context.Context, r autopost.Request) (json.RawMessage, error) {
	return s.GetPolicy(c, r)
}
func (s *syntheticAccounting) Status(_ context.Context, r autopost.Request) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{"profile":"automatic-accounting-status-v1","entity_id":"` + r.Entity + `","book_id":"` + r.Book + `","enabled":false,"policy_revision":"0","pending_count":"0","attention_count":"0","last_operation":null,"holds":[],"holds_truncated":false,"blocked_reason":null}`), nil
}

// Default-off preserves API48 startup; mounting settings and starting financial work are independent decisions.
func TestAutomaticAccountingComposition(t *testing.T) {
	base := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic"}
	bad := base
	bad.AutomaticSourcePosting = true
	if bad.Validate() == nil {
		t.Fatal("posting accepted without API")
	}
	bad = base
	bad.Mode = "health"
	bad.AutomaticAccountingEnabled = true
	if bad.Validate() == nil {
		t.Fatal("health accepted financial API")
	}
	for _, api := range []bool{false, true} {
		for _, run := range []bool{false, true} {
			if run && !api {
				continue
			}
			c := base
			c.AutomaticAccountingEnabled = api
			c.AutomaticSourcePosting = run
			port := &syntheticAccounting{}
			runner := &syntheticAutomaticRunner{make(chan struct{}), make(chan struct{})}
			deps := Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }, Accounting: port, SourcePosting: runner}
			s, err := New(c, deps)
			if err != nil {
				t.Fatal(err)
			}
			if (s.sourcePosting != nil) != run {
				t.Fatal("settings silently activated runner")
			}
			for _, token := range []string{"", "Bearer synthetic"} {
				r := httptest.NewRequest("GET", "/v1/entities/76000000-0000-4000-8000-000000000001/books/76000000-0000-4000-8000-000000000002/automatic-accounting/status", nil)
				r.Header.Set("Origin", c.Origin)
				r.Header.Set("Authorization", token)
				w := httptest.NewRecorder()
				s.HTTP.Handler.ServeHTTP(w, r)
				want := 404
				if api {
					want = 200
					if token == "" {
						want = 401
					}
				}
				if w.Code != want {
					t.Fatalf("api=%v run=%v got%d want%d", api, run, w.Code, want)
				}
				if want == 200 {
					var out map[string]any
					if json.Unmarshal(w.Body.Bytes(), &out) != nil || out["runtime_enabled"] != run {
						t.Fatal("wrong runtime signal")
					}
				}
			}
			if api {
				missing := deps
				missing.Accounting = nil
				if _, err := New(c, missing); err == nil {
					t.Fatal("missing API port accepted")
				}
			}
			if run {
				missing := deps
				missing.SourcePosting = nil
				if _, err := New(c, missing); err == nil {
					t.Fatal("missing runner accepted")
				}
			}
		}
	}
}
func TestAutomaticAccountingDrainBothRunners(t *testing.T) {
	category := &syntheticAutomaticRunner{make(chan struct{}), make(chan struct{})}
	posting := &syntheticAutomaticRunner{make(chan struct{}), make(chan struct{})}
	s := &Server{automatic: category, sourcePosting: posting}
	stop := s.startAutomatic(context.Background())
	for _, r := range []*syntheticAutomaticRunner{category, posting} {
		select {
		case <-r.started:
		case <-time.After(time.Second):
			t.Fatal("runner not started")
		}
	}
	stop()
	for _, r := range []*syntheticAutomaticRunner{category, posting} {
		select {
		case <-r.stopped:
		default:
			t.Fatal("runner not drained")
		}
	}
}
