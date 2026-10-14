// Package workerhost supplies bounded HTTP lifecycle without selecting routes or credentials.
package workerhost

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// Fixed errors exclude listener addresses, requests, tokens and underlying diagnostics.
var (
	ErrConfiguration = errors.New("worker_host_configuration")
	ErrAlreadyServed = errors.New("worker_host_already_served")
	ErrServe         = errors.New("worker_host_serve_failed")
	ErrShutdown      = errors.New("worker_host_shutdown_failed")
)

// Host owns one listener lifecycle; injected handlers retain authentication and durable-work responsibility.
type Host struct {
	handler http.Handler
	started atomic.Bool
}

// New creates an inert single-use host; it opens no sockets and installs no routes.
func New(handler http.Handler) (*Host, error) {
	if handler == nil {
		return nil, ErrConfiguration
	}
	return &Host{handler: handler}, nil
}

// server fixes transport bounds and propagates cancellation to every ordinary request.
func (h *Host) server(ctx context.Context) *http.Server {
	return &http.Server{
		Handler: h.handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 90 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10,
		BaseContext: func(net.Listener) context.Context { return ctx },
		ErrorLog:    log.New(io.Discard, "", 0),
	}
}

// Serve takes listener ownership after validation and waits for shutdown before returning.
// Cancellation is cooperative: Close interrupts sockets, not arbitrary handler code or hijacked connections.
func (h *Host) Serve(ctx context.Context, listener net.Listener) error {
	if h == nil || h.handler == nil || ctx == nil || listener == nil {
		return ErrConfiguration
	}
	if !h.started.CompareAndSwap(false, true) {
		return ErrAlreadyServed
	}
	requests, cancel := context.WithCancel(ctx)
	defer cancel()
	srv := h.server(requests)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	var serveErr error
	joined := false
	select {
	case <-ctx.Done():
	case serveErr = <-done:
		joined = true
	}
	// A listener failure also cancels in-flight work; cleanup never inherits an expired caller context.
	cancel()
	drain, stop := context.WithTimeout(context.Background(), 10*time.Second)
	shutdownErr := srv.Shutdown(drain)
	stop()
	if shutdownErr != nil {
		_ = srv.Close()
	}
	if !joined {
		serveErr = <-done
	}
	if shutdownErr != nil {
		return ErrShutdown
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return ErrServe
	}
	return nil
}
