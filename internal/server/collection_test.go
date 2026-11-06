package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionread"
)

// syntheticCollection exposes no SQL behavior; it checks only explicit route composition.
type syntheticCollection struct{ calls int }

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
		d := Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }, Collection: port}
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
		if enabled {
			d.Collection = nil
			if _, err = New(c, d); err == nil {
				t.Fatal("enabled feature without adapter")
			}
		}
	}
}
