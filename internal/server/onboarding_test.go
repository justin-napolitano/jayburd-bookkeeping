package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectiononboarding"
)

// syntheticOnboarding proves route admission only; SQL lifecycle has independent integration proofs.
type syntheticOnboarding struct{ calls int }

func (s *syntheticOnboarding) Catalog(context.Context, collectiononboarding.Request) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{"profile":"private-onboarding-catalog-v1","workspace":{"state":"absent"},"can_setup":true,"can_adopt":false,"candidates":[]}`), nil
}
func (*syntheticOnboarding) Preview(context.Context, collectiononboarding.Request) (json.RawMessage, error) {
	panic("unexpected preview")
}
func (*syntheticOnboarding) Prepare(context.Context, collectiononboarding.Request) (json.RawMessage, error) {
	panic("unexpected prepare")
}
func (*syntheticOnboarding) Pending(context.Context, collectiononboarding.Request) (json.RawMessage, error) {
	panic("unexpected pending")
}
func (*syntheticOnboarding) Clear(context.Context, collectiononboarding.Request) (json.RawMessage, error) {
	panic("unexpected clear")
}
func (*syntheticOnboarding) Transition(context.Context, collectiononboarding.Request, string) (json.RawMessage, error) {
	panic("unexpected transition")
}

// TestOnboardingComposition prevents the new self-service policy escaping the scoped Sandbox gate.
func TestOnboardingComposition(t *testing.T) {
	for _, tc := range []struct {
		name, project, environment string
		enabled, adapter           bool
		want                       int
	}{
		{"sandbox", "jayburd-bookkeeping-dev", "sandbox", true, true, 200},
		{"private disabled", "jayburd-bookkeeping-dev", "sandbox", false, true, 404},
		{"foreign project", "synthetic-foreign", "sandbox", true, true, 404},
		{"production provider", "fictional-production", "production", true, true, 404},
		{"adapter absent", "jayburd-bookkeeping-dev", "sandbox", true, false, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: tc.project, ProviderEnvironment: tc.environment, DatabaseURL: "synthetic", CollectionEnabled: tc.enabled}
			if tc.environment == "production" {
				c.DatabaseURL = ""
				c.DatabaseMode = "cloudsql"
				c.CloudSQLInstance = tc.project + ":us-central1:pilot-db"
			}
			port := &syntheticOnboarding{}
			d := Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }, Collection: &syntheticCollection{}, Labels: &syntheticLabels{}}
			if tc.adapter {
				d.Onboarding = port
			}
			s, err := New(c, d)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("GET", "/v1/collection-onboarding", nil)
			r.Header.Set("Origin", c.Origin)
			r.Header.Set("Authorization", "Bearer synthetic")
			w := httptest.NewRecorder()
			s.HTTP.Handler.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status%d want%d", w.Code, tc.want)
			}
			wantCalls := 0
			if tc.want == 200 {
				wantCalls = 1
			}
			if port.calls != wantCalls {
				t.Fatal("onboarding reached outside scope")
			}
			if tc.want == 200 {
				for _, boundary := range []struct {
					token, origin string
					status        int
				}{{"", c.Origin, 401}, {"Bearer synthetic", "https://foreign.invalid", 403}} {
					r := httptest.NewRequest("GET", "/v1/collection-onboarding", nil)
					r.Header.Set("Origin", boundary.origin)
					if boundary.token != "" {
						r.Header.Set("Authorization", boundary.token)
					}
					w := httptest.NewRecorder()
					s.HTTP.Handler.ServeHTTP(w, r)
					if w.Code != boundary.status {
						t.Fatal("onboarding bypassed identity/origin boundary")
					}
				}
				if port.calls != 1 {
					t.Fatal("rejected caller reached onboarding")
				}
			}
		})
	}
}
