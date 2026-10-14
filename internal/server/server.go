package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// Dependencies holds already composed adapters; no request can switch providers.
type Dependencies struct {
	Auth        ledger.Authenticator
	Operations  ledger.Operations
	Connections ConnectionOperations
	Ready       func(context.Context) error
	Close       func()
}

// Server owns bounded HTTP lifecycle and distinguishes liveness from dependency readiness.
type Server struct {
	HTTP    *http.Server
	closing atomic.Bool
	close   func()
}

// New mounts accounting only when managed adapters exist. Health-only is never ready.
func New(c Config, d Dependencies) (*Server, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	s := &Server{close: d.Close}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		if c.Mode != "managed" || s.closing.Load() || d.Ready == nil {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"status":"not_ready"}`))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if d.Ready(ctx) != nil {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"status":"not_ready"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	if c.Mode == "managed" {
		if d.Auth == nil || d.Operations == nil || d.Ready == nil {
			return nil, errConfiguration
		}
		application := http.NewServeMux()
		application.Handle("/v1/", ledger.Handler(d.Auth, d.Operations))
		if d.Connections != nil {
			mountConnections(application, d.Auth, d.Connections)
		}
		h, err := ledger.BrowserBoundary(c.Origin, application)
		if err != nil {
			return nil, errConfiguration
		}
		mux.Handle("/v1/", h)
	}
	// Concurrency bounds cap authenticated provider work and queued database pressure.
	slots := make(chan struct{}, 32)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if len(r.RequestURI) > 4096 {
			writeProblem(w, 414, "request_too_large")
			return
		}
		if s.closing.Load() {
			writeProblem(w, 503, "temporarily_unavailable")
			return
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			w.Header().Set("Retry-After", "1")
			writeProblem(w, 503, "temporarily_unavailable")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		mux.ServeHTTP(w, r)
	})
	s.HTTP = &http.Server{Addr: c.Listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10, ErrorLog: log.New(io.Discard, "", 0)}
	return s, nil
}

// writeProblem keeps admission and transport failures in the same safe RFC 9457
// vocabulary as accounting routes; every ID is server generated.
func writeProblem(w http.ResponseWriter, status int, code string) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("request identifier unavailable")
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	id := fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	if status == 503 {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://bookkeeping.example.invalid/problems/" + code, "title": http.StatusText(status), "status": status, "code": code, "request_id": id})
}

// Run drains requests on owner cancellation, then closes the pool; failures are redacted.
func (s *Server) Run(ctx context.Context) error {
	stopped := make(chan struct{})
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		select {
		case <-ctx.Done():
			s.closing.Store(true)
			shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if s.HTTP.Shutdown(shutdown) != nil {
				_ = s.HTTP.Close()
			}
		case <-stopped:
		}
	}()
	err := s.HTTP.ListenAndServe()
	close(stopped)
	<-drained
	s.closing.Store(true)
	if s.close != nil {
		s.close()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New("server stopped")
	}
	return nil
}
