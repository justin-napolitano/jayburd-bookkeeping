package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// syntheticAuth checks transport composition only, never token verification claims.
type syntheticAuth struct{}

// Authenticate supplies the explicitly synthetic fixture actor.
func (syntheticAuth) Authenticate(context.Context, string) (string, error) { return "synthetic", nil }

// syntheticOps implements no database behavior; production SQL has separate integration proof.
type syntheticOps struct{ ledger.Operations }

// Workspace makes mounted routing observable without any provider traffic.
func (syntheticOps) Workspace(context.Context, ledger.Command) (ledger.Result, error) {
	return ledger.Result{Status: 200, Body: []byte(`{"entities":[]}`)}, nil
}

// TestServerBoundary proves default inertness, readiness failure and managed origin controls.
func TestServerBoundary(t *testing.T) {
	health, err := New(Config{Mode: "health", Listen: "127.0.0.1:8080"}, Dependencies{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{{"/healthz", 200}, {"/readyz", 503}, {"/v1/workspace", 404}, {"/v1/setup", 404}} {
		r := httptest.NewRequest("GET", tc.path, nil)
		w := httptest.NewRecorder()
		health.HTTP.Handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
		if tc.status == 414 {
			assertTransportProblem(t, w, "request_too_large")
		}
	}
	ready := true
	managed, err := New(Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic"}, Dependencies{Auth: syntheticAuth{}, Operations: syntheticOps{}, Ready: func(context.Context) error {
		if !ready {
			return errors.New("private backend detail")
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, origin, token string
		status              int
	}{{"/readyz", "", "", 200}, {"/v1/workspace", "http://localhost:3000", "Bearer synthetic", 200}, {"/v1/workspace", "https://foreign.invalid", "Bearer synthetic", 403}, {"/v1/workspace", "http://localhost:3000", "", 401}, {"/" + strings.Repeat("x", 4100), "", "", 414}} {
		r := httptest.NewRequest("GET", tc.path, nil)
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		if tc.token != "" {
			r.Header.Set("Authorization", tc.token)
		}
		w := httptest.NewRecorder()
		managed.HTTP.Handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
		if tc.status == 414 {
			assertTransportProblem(t, w, "request_too_large")
		}
	}
	ready = false
	w := httptest.NewRecorder()
	managed.HTTP.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "private") {
		t.Fatal("readiness disclosure")
	}
	if managed.HTTP.ReadHeaderTimeout == 0 || managed.HTTP.WriteTimeout == 0 || managed.HTTP.MaxHeaderBytes != 32<<10 {
		t.Fatal("unbounded server")
	}
	managed.closing.Store(true)
	w = httptest.NewRecorder()
	managed.HTTP.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
	if w.Code != 503 {
		t.Fatal("draining server ready")
	}
}

// TestManagedConfiguration prevents credential-file, emulator and inherited PG loading.
func TestManagedConfiguration(t *testing.T) {
	// This set contains environment variable names only, never their values.
	for name := range map[string]struct{}{
		"FIREBASE_AUTH_EMULATOR_HOST":    {},
		"GOOGLE_APPLICATION_CREDENTIALS": {},
		"GCE_METADATA_HOST":              {},
		"PGSERVICEFILE":                  {},
		"PGPASSWORD":                     {},
		"PGSSLSNI":                       {},
		"PGSSLNEGOTIATION":               {},
		"PGCHANNELBINDING":               {},
		"PGREQUIREAUTH":                  {},
	} {
		if !forbiddenEnvironment(func(k string) bool { return k == name }) {
			t.Fatal("unsafe override permitted")
		}
	}
	for _, raw := range []string{"postgres://synthetic@db.invalid/books?sslmode=disable", "postgres://synthetic@db.invalid/books?service=hidden", "postgres://synthetic@db.invalid/books?sslrootcert=/private/file", "postgres://synthetic@db.invalid/books?sslmode=verify-full&sslmode=disable", "malformed"} {
		if _, err := poolConfig(raw); err == nil {
			t.Fatal("unsafe database configuration accepted")
		}
	}
	for _, raw := range []string{"postgres://synthetic@db.invalid/books", "postgres://synthetic@127.0.0.1/books?sslmode=disable"} {
		cfg, err := poolConfig(raw)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MaxConns != 8 || cfg.ConnConfig.ConnectTimeout == 0 || len(cfg.ConnConfig.Fallbacks) != 0 {
			t.Fatal("pool bounds or fallback invalid")
		}
	}
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	_, err := Managed(context.Background(), Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "https://synthetic.invalid", Project: "synthetic-project", DatabaseURL: "synthetic"})
	if err == nil {
		t.Fatal("emulator presence accepted")
	}
	if _, err := New(Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://public.invalid", Project: "synthetic", DatabaseURL: "synthetic"}, Dependencies{}); err == nil {
		t.Fatal("public HTTP origin accepted")
	}
}

// blockingAuth holds synthetic verification at a channel barrier to test admission.
type blockingAuth struct {
	started chan struct{}
	release chan struct{}
}

// Authenticate permits release or request cancellation; it never contacts a provider.
func (a blockingAuth) Authenticate(ctx context.Context, _ string) (string, error) {
	a.started <- struct{}{}
	select {
	case <-a.release:
		return "synthetic", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// TestServerAdmission proves saturation fails before the next authentication call.
func TestServerAdmission(t *testing.T) {
	auth := blockingAuth{started: make(chan struct{}, 32), release: make(chan struct{})}
	s, err := New(Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "https://synthetic.invalid", Project: "synthetic-project", DatabaseURL: "synthetic"}, Dependencies{Auth: auth, Operations: syntheticOps{}, Ready: func(context.Context) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 32)
	for range 32 {
		go func() {
			r := httptest.NewRequest("GET", "/v1/workspace", nil)
			r.Header.Set("Authorization", "Bearer synthetic")
			w := httptest.NewRecorder()
			s.HTTP.Handler.ServeHTTP(w, r)
			done <- w.Code
		}()
	}
	for range 32 {
		<-auth.started
	}
	r := httptest.NewRequest("GET", "/v1/workspace", nil)
	r.Header.Set("Authorization", "Bearer synthetic")
	w := httptest.NewRecorder()
	s.HTTP.Handler.ServeHTTP(w, r)
	close(auth.release)
	assertTransportProblem(t, w, "temporarily_unavailable")
	if w.Code != 503 || w.Header().Get("Retry-After") != "1" {
		t.Fatal("saturation not rejected")
	}
	for range 32 {
		if <-done != 200 {
			t.Fatal("admitted request failed")
		}
	}
}

// assertTransportProblem rejects opaque/plaintext admission failures and cacheable errors.
func assertTransportProblem(t *testing.T, w *httptest.ResponseRecorder, code string) {
	t.Helper()
	var body struct {
		Code string `json:"code"`
		ID   string `json:"request_id"`
	}
	if w.Header().Get("Content-Type") != "application/problem+json" || w.Header().Get("Cache-Control") != "no-store" || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Code != code || len(body.ID) != 36 {
		t.Fatal("invalid transport problem contract")
	}
}

// TestCloudSQLSelection forbids ambiguous transports and cross-project targets before startup.
func TestCloudSQLSelection(t *testing.T) {
	c := Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "https://synthetic.invalid", Project: "synthetic-project", DatabaseMode: "cloudsql", CloudSQLInstance: "synthetic-project:us-central1:jayburd-dev"}
	if c.Validate() != nil {
		t.Fatal("valid cloud configuration rejected")
	}
	c.DatabaseURL = "synthetic"
	if c.Validate() == nil {
		t.Fatal("mixed transports accepted")
	}
	c.DatabaseURL = ""
	c.Project = "different-project"
	if c.Validate() == nil {
		t.Fatal("cross-project target accepted")
	}
	c.Project = "synthetic-project"
	t.Setenv("JAYBURD_DATABASE_URL", "")
	if _, err := Managed(context.Background(), c); err == nil {
		t.Fatal("ambient DSN presence accepted")
	}
}
