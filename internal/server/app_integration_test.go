//go:build integration

package server

import (
	"context"
	"errors"
	"flag"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

var appProofMarker = flag.String("app-proof-marker", "", "public per-invocation synthetic readiness marker")

var appProof = flag.Bool("app-proof", false, "run only the disposable browser integration fixture")
var appProofDemo = flag.Bool("app-proof-demo", false, "seed fictional review books in the disposable fixture")

// appProofAuth exists only in the integration-tagged test executable, never cmd/api.
type appProofAuth struct{}

// Authenticate accepts one fictional token and cannot call a provider or enroll users.
func (appProofAuth) Authenticate(_ context.Context, token string) (string, error) {
	if token != "synthetic.browser.proof" {
		return "", errors.New("synthetic identity rejected")
	}
	return "00000000-0000-4000-8000-000000000001", nil
}

// TestAppIntegrationServer composes real HTTP/SQL boundaries around a test-only identity.
// It accepts no DSN; the companion script owns the fixed isolated Docker hostname.
func TestAppIntegrationServer(t *testing.T) {
	if !*appProof {
		t.Skip("disposable browser runner only")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	setupCtx, stop := context.WithTimeout(ctx, 15*time.Second)
	defer stop()
	admin, err := pgx.Connect(setupCtx, "host=database port=5432 user=postgres dbname=app_proof sslmode=disable connect_timeout=5")
	if err != nil {
		t.Fatal("synthetic database unavailable")
	}
	defer admin.Close(context.Background())
	_, err = admin.Exec(setupCtx, `
 CREATE ROLE synthetic_app LOGIN IN ROLE bookkeeping_api;
 INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES('00000000-0000-4000-8000-000000000001','https://synthetic.example.invalid','synthetic-browser');
 INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000002','USD','840',2,'2000-01-01',NULL,'synthetic-browser-fixture');`)
	if err != nil {
		t.Fatal("synthetic fixture setup failed", err)
	}
	pool, err := pgxpool.New(setupCtx, "host=database port=5432 user=synthetic_app dbname=app_proof sslmode=disable connect_timeout=5 pool_max_conns=4")
	if err != nil {
		t.Fatal("synthetic runtime unavailable")
	}
	defer pool.Close()
	if err = runtimeReady(setupCtx, pool); err != nil {
		t.Fatal("synthetic runtime permissions rejected")
	}
	var acquisition *demoAcquisition
	if *appProofDemo {
		evidenceRoot := t.TempDir()
		if err := os.Chmod(evidenceRoot, 0700); err != nil {
			t.Fatal("private demo storage unavailable")
		}
		store, err := staging.Open(evidenceRoot)
		if err != nil {
			t.Fatal("private demo storage unavailable")
		}
		defer store.Close()
		acquisition, err = bootstrapDemoAcquisition(ctx, admin, pool, store)
		if err != nil {
			t.Fatal("synthetic acquisition bootstrap rejected", err)
		}
		if err := seedDemo(ctx, admin, pool); err != nil {
			t.Fatal("synthetic demo seed rejected", err)
		}
	}
	var connectionFixture *connectionBrowserFixture
	var connectionService *connections.Service
	if *appProofConnections {
		if *appProofDemo {
			t.Fatal("proof fixture modes are exclusive")
		}
		connectionFixture = bootstrapConnectionBrowser(t, ctx, admin, pool)
		connectionService = connectionFixture.service
	}
	app, err := New(Config{Mode: "managed", Listen: "0.0.0.0:8080", Origin: "http://localhost:39083", Project: "synthetic-browser-proof", DatabaseURL: "synthetic-injected-test-pool"}, Dependencies{Connections: connectionService, Auth: appProofAuth{}, Operations: &ledger.Service{DB: pool}, Ready: func(ctx context.Context) error { return runtimeReady(ctx, pool) }})
	if err != nil {
		t.Fatal("synthetic server composition rejected")
	}
	handler := app.HTTP.Handler
	if connectionFixture != nil {
		handler = connectionFixture.wrap(handler)
	}
	if acquisition != nil {
		demoHandler, e := ledger.BrowserBoundary("http://localhost:39083", acquisition)
		if e != nil {
			t.Fatal("synthetic acquisition origin invalid")
		}
		productHandler := handler
		demoSlots := make(chan struct{}, 4)
		// Demo acquisition is absent from cmd/api and bounded independently of production routes.
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v1/demo/") {
				select {
				case demoSlots <- struct{}{}:
					defer func() { <-demoSlots }()
				default:
					writeProblem(w, 503, "temporarily_unavailable")
					return
				}
				r.Body = http.MaxBytesReader(w, r.Body, 64)
				demoHandler.ServeHTTP(w, r)
				return
			}
			productHandler.ServeHTTP(w, r)
		})
	}
	app.HTTP.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Jayburd-Proof", *appProofMarker)
		handler.ServeHTTP(w, r)
	})
	if err := app.Run(ctx); err != nil {
		t.Fatal("synthetic server failed")
	}
}

var appBridge = flag.String("app-bridge", "", "transport only to the current disposable browser API container")

// TestAppIntegrationBridge forwards raw HTTP bytes over docker exec for remote daemons.
// It cannot connect to a database or select an arbitrary command/destination.
func TestAppIntegrationBridge(t *testing.T) {
	if *appBridge == "" {
		t.Skip("disposable browser runner only")
	}
	if !regexp.MustCompile(`^jayburd-app-[0-9]+-[0-9]+-api$`).MatchString(*appBridge) {
		t.Fatal("invalid proof container")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	listener, err := net.Listen("tcp", "127.0.0.1:39081")
	if err != nil {
		t.Fatal("synthetic bridge listener unavailable")
	}
	defer listener.Close()
	go func() { <-ctx.Done(); _ = listener.Close() }()
	var active sync.WaitGroup
	defer active.Wait()
	slots := make(chan struct{}, 32)
	for {
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			t.Fatal("synthetic bridge failed")
		}
		select {
		case slots <- struct{}{}:
		default:
			_ = connection.Close()
			continue
		}
		active.Add(1)
		go func() {
			defer active.Done()
			defer func() { <-slots }()
			defer connection.Close()
			_ = connection.SetDeadline(time.Now().Add(30 * time.Second))
			operation, stop := context.WithTimeout(ctx, 30*time.Second)
			defer stop()
			command := exec.CommandContext(operation, "docker", "exec", "-i", *appBridge, "busybox", "nc", "127.0.0.1", "8080")
			command.Stdin = connection
			command.Stdout = connection
			command.Stderr = io.Discard
			command.WaitDelay = time.Second
			_ = command.Run()
		}()
	}
}
