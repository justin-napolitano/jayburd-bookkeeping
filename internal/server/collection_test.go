package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionlabels"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionread"
)

// syntheticCollection exposes no SQL behavior; it checks only explicit route composition.
type syntheticCollection struct{ calls int }

// syntheticLabels checks route wiring only; real SQL behavior has separate proofs.
type syntheticLabels struct{ calls int }

func (s *syntheticLabels) Catalog(_ context.Context, r collectionlabels.Request) (json.RawMessage, error) {
	s.calls++
	if r.Version == 2 {
		return json.RawMessage(`{"profile":"private-collection-catalog-v2","workspace_id":"` + r.Workspace + `"}`), nil
	}
	return json.RawMessage(`{"profile":"private-collection-catalog-v1","workspace_id":"` + r.Workspace + `"}`), nil
}
func (s *syntheticLabels) Query(context.Context, collectionlabels.Request) (json.RawMessage, error) {
	panic("unexpected query")
}
func (s *syntheticLabels) Preview(context.Context, collectionlabels.Request) (json.RawMessage, error) {
	panic("unexpected preview")
}
func (s *syntheticLabels) Prepare(context.Context, collectionlabels.Request) (json.RawMessage, error) {
	panic("unexpected prepare")
}
func (s *syntheticLabels) Pending(context.Context, collectionlabels.Request) (json.RawMessage, error) {
	panic("unexpected pending")
}
func (s *syntheticLabels) Transition(context.Context, collectionlabels.Request, string) (json.RawMessage, error) {
	panic("unexpected transition")
}

// Workspace supplies a fixed source workspace only when the route reaches this synthetic port.
func (s *syntheticCollection) Workspace(context.Context, collectionread.Request) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{"profile":"private-source-workspace-v1"}`), nil
}

// Sources is unused by this narrow route test.
func (s *syntheticCollection) Sources(context.Context, collectionread.Request) (json.RawMessage, error) {
	panic("unexpected sources call")
}

// Observations is unused by this narrow route test.
func (s *syntheticCollection) Observations(context.Context, collectionread.Request) (json.RawMessage, error) {
	panic("unexpected observations call")
}

// TestCollectionComposition keeps default mode inert and requires explicit reviewed composition.
func TestCollectionComposition(t *testing.T) {
	if _, err := New(Config{Mode: "health", Listen: "127.0.0.1:8080", CollectionEnabled: true}, Dependencies{}); err == nil {
		t.Fatal("health mode accepted private activation")
	}
	for _, enabled := range []bool{false, true} {
		c := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic", CollectionEnabled: enabled}
		port := &syntheticCollection{}
		labels := &syntheticLabels{}
		d := Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }, Collection: port, Labels: labels}
		s, err := New(c, d)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			token, origin string
			status        int
		}{{"Bearer synthetic", c.Origin, 200}, {"", c.Origin, 401}, {"Bearer synthetic", "https://foreign.invalid", 403}} {
			r := httptest.NewRequest("GET", "/v1/collection", nil)
			r.Header.Set("Authorization", tc.token)
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			s.HTTP.Handler.ServeHTTP(w, r)
			want := tc.status
			if !enabled && tc.origin == c.Origin {
				want = 404
			}
			if w.Code != want {
				t.Fatalf("enabled%v: status%d want%d", enabled, w.Code, want)
			}
		}
		want := 0
		if enabled {
			want = 1
		}
		if port.calls != want {
			t.Fatal("private port reached without feature/auth/origin")
		}
		for _, tc := range []struct {
			token, origin string
			status        int
		}{{"Bearer synthetic", c.Origin, 200}, {"", c.Origin, 401}, {"Bearer synthetic", "https://foreign.invalid", 403}} {
			r := httptest.NewRequest("GET", "/v1/collections/00000000-0000-4000-8000-000000000001/taxonomy", nil)
			r.Header.Set("Authorization", tc.token)
			r.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			s.HTTP.Handler.ServeHTTP(w, r)
			status := tc.status
			if !enabled && tc.origin == c.Origin {
				status = 404
			}
			if w.Code != status {
				t.Fatalf("label enabled%v status%d want%d", enabled, w.Code, status)
			}
		}
		if labels.calls != want {
			t.Fatal("label port reached without feature/auth/origin")
		}
		if enabled {
			d.Labels = nil
			if _, err = New(c, d); err == nil {
				t.Fatal("enabled label workflow without adapter")
			}
			d.Labels = labels
			d.Collection = nil
			if _, err = New(c, d); err == nil {
				t.Fatal("enabled feature without adapter")
			}
		}
	}
}

// Recovery supplies a minimal synthetic receipt only for route/origin composition checks.
func (s *syntheticLabels) Recovery(context.Context, collectionlabels.Request) (json.RawMessage, error) {
	return json.RawMessage(`{"profile":"private-collection-recovery-v1","state":"none","recovery_id":null}`), nil
}
func (s *syntheticLabels) ClearRecovery(context.Context, collectionlabels.Request) (json.RawMessage, error) {
	panic("unexpected clear")
}

// TestVersionedPrivateComposition keeps v2/global routes behind the existing feature and origin boundary.
func TestVersionedPrivateComposition(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		c := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic", CollectionEnabled: enabled}
		s, err := New(c, Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }, Collection: &syntheticCollection{}, Labels: &syntheticLabels{}})
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/v2/collections/00000000-0000-4000-8000-000000000001/taxonomy", "/v1/collection-recovery"} {
			for _, origin := range []string{c.Origin, "https://foreign.invalid"} {
				r := httptest.NewRequest("GET", path, nil)
				r.Header.Set("Authorization", "Bearer synthetic")
				r.Header.Set("Origin", origin)
				out := httptest.NewRecorder()
				s.HTTP.Handler.ServeHTTP(out, r)
				want := 200
				if !enabled {
					want = 404
				}
				if origin != c.Origin {
					want = 403
				}
				if out.Code != want {
					t.Fatalf("%s enabled%v origin boundary status%d want%d", path, enabled, out.Code, want)
				}
			}
		}
	}
}
