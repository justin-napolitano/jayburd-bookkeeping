package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionlabels"
)

type syntheticAutomation struct{ calls int }

func (s *syntheticAutomation) GetAutomation(_ context.Context, r collectionlabels.Request) (json.RawMessage, error) {
	s.calls++
	return json.RawMessage(`{"profile":"private-collection-automation-v1","workspace_id":"` + r.Workspace + `","revision":"1","enabled":false,"processing_state":"disabled","blocked_reason":null,"pending_count":"0","attention_count":"0","last_run":null}`), nil
}
func (s *syntheticAutomation) SetAutomation(c context.Context, r collectionlabels.Request) (json.RawMessage, error) {
	return s.GetAutomation(c, r)
}

type syntheticAutomaticRunner struct{ started, stopped chan struct{} }

func (s *syntheticAutomaticRunner) Run(ctx context.Context) {
	close(s.started)
	<-ctx.Done()
	close(s.stopped)
}

// TestAutomaticComposition preserves default-off routes and requires both settings and background composition.
func TestAutomaticComposition(t *testing.T) {
	c := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic", CollectionEnabled: true}
	bad := c
	bad.CollectionEnabled = false
	bad.AutomaticCategories = true
	if bad.Validate() == nil {
		t.Fatal("automation accepted without collection")
	}
	for _, enabled := range []bool{false, true} {
		c.AutomaticCategories = enabled
		port := &syntheticAutomation{}
		runner := &syntheticAutomaticRunner{make(chan struct{}), make(chan struct{})}
		d := Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }, Collection: &syntheticCollection{}, Labels: &syntheticLabels{}, Automation: port, Automatic: runner}
		if enabled {
			bad := d
			bad.Automatic = nil
			if _, e := New(c, bad); e == nil {
				t.Fatal("missing background accepted")
			}
			bad = d
			bad.Automation = nil
			if _, e := New(c, bad); e == nil {
				t.Fatal("missing policy port accepted")
			}
		}
		s, e := New(c, d)
		if e != nil {
			t.Fatal(e)
		}
		for _, token := range []string{"", "Bearer synthetic"} {
			r := httptest.NewRequest("GET", "/v1/collections/00000000-0000-4000-8000-000000000001/automation", nil)
			r.Header.Set("Origin", c.Origin)
			if token != "" {
				r.Header.Set("Authorization", token)
			}
			w := httptest.NewRecorder()
			s.HTTP.Handler.ServeHTTP(w, r)
			want := 404
			if enabled {
				want = 200
				if token == "" {
					want = 401
				}
			}
			if w.Code != want {
				t.Fatalf("enabled%v status%d want%d", enabled, w.Code, want)
			}
		}
		if enabled && port.calls != 1 || !enabled && port.calls != 0 {
			t.Fatal("unconfigured automation reached")
		}
	}
}

// TestAutomaticDrain does not bind a socket: the lifecycle helper must wait for cancellation acknowledgement.
func TestAutomaticDrain(t *testing.T) {
	runner := &syntheticAutomaticRunner{make(chan struct{}), make(chan struct{})}
	s := &Server{automatic: runner}
	stop := s.startAutomatic(context.Background())
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("background not started")
	}
	stop()
	select {
	case <-runner.stopped:
	default:
		t.Fatal("pool could close before background drained")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	(&Server{}).startAutomatic(ctx)()
}
