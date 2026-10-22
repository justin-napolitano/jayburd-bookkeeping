// Package webhookhost composes only the verified Sandbox notification boundary.
package webhookhost

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
)

// Fixed errors never contain addresses, requests, credentials or dependency diagnostics.
var (
	ErrConfiguration = errors.New("webhook_host_configuration")
	ErrAlreadyServed = errors.New("webhook_host_already_served")
	ErrServe         = errors.New("webhook_host_serve_failed")
	ErrShutdown      = errors.New("webhook_host_shutdown_failed")
)

// Host owns one explicit listener and optional resource cleanup; it selects no cloud identity.
type Host struct {
	handler *webhook.Handler
	close   func()
	once    sync.Once
	started atomic.Bool
	closed  atomic.Bool
}

// New composes verification and durable receipt capabilities without opening sockets or resolving keys.
// The receiver must commit receipt and wakeup atomically; NewSQL supplies the reviewed adapter.
func New(keys webhook.KeyProvider, receiver webhook.Receiver, close func()) (*Host, error) {
	v, err := webhook.NewVerifier(keys)
	if err != nil {
		return nil, ErrConfiguration
	}
	h, err := webhook.NewHandler(v, receiver)
	if err != nil {
		return nil, ErrConfiguration
	}
	return &Host{handler: h, close: close}, nil
}

// NewSQL uses a separately provisioned webhook-only pool; runtime SQL checks its actual authority.
func NewSQL(keys webhook.KeyProvider, db webhook.Beginner, close func()) (*Host, error) {
	r, err := webhook.NewSQLReceiver(db)
	if err != nil {
		return nil, ErrConfiguration
	}
	return New(keys, r, close)
}

// Close releases explicitly owned dependencies at most once, after Serve has drained.
func (h *Host) Close() {
	if h != nil {
		h.once.Do(func() {
			h.closed.Store(true)
			if h.close != nil {
				h.close()
			}
		})
	}
}

// server bounds header/body socket reads and response writes independently of provider/SQL timeouts.
// These local limits do not establish end-to-end provider delivery timing through a hosted proxy.
func (h *Host) server(ctx context.Context) *http.Server {
	return &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bounded, cancel := context.WithTimeout(r.Context(), 8*time.Second)
			defer cancel()
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
			h.handler.ServeHTTP(w, r.WithContext(bounded))
		}),
		DisableGeneralOptionsHandler: true,
		ReadHeaderTimeout:            2 * time.Second, ReadTimeout: 5 * time.Second,
		WriteTimeout: 8 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 16 << 10,
		BaseContext: func(net.Listener) context.Context { return ctx },
		ErrorLog:    log.New(io.Discard, "", 0),
	}
}

// Serve takes listener ownership once and closes dependencies after cancellation or listener failure.
// TLS termination, public ingress authorization, IAM and enrollment are separate reviewed deployment work.
func (h *Host) Serve(ctx context.Context, l net.Listener) error {
	if h == nil || h.handler == nil || ctx == nil || l == nil {
		return ErrConfiguration
	}
	if h.closed.Load() {
		return ErrAlreadyServed
	}
	if !h.started.CompareAndSwap(false, true) {
		return ErrAlreadyServed
	}
	defer h.Close()
	requests, cancel := context.WithCancel(ctx)
	defer cancel()
	srv := h.server(requests)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(l) }()
	var serveErr error
	joined := false
	select {
	case <-ctx.Done():
	case serveErr = <-done:
		joined = true
	}
	cancel()
	drain, stop := context.WithTimeout(context.Background(), 2*time.Second)
	err := srv.Shutdown(drain)
	stop()
	if err != nil {
		_ = srv.Close()
	}
	if !joined {
		serveErr = <-done
	}
	if err != nil {
		return ErrShutdown
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return ErrServe
	}
	return nil
}
