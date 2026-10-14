package workerhost

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// listen provides disposable real sockets without cloud or credential dependencies.
func listen(t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

// await bounds test goroutines so failed lifecycle assertions do not hang the suite.
func await(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("host did not return")
		return nil
	}
}

// TestFixedProfile checks the immutable socket policy separately from business handlers.
func TestFixedProfile(t *testing.T) {
	h, _ := New(http.NotFoundHandler())
	s := h.server(context.Background())
	if s.ReadHeaderTimeout != 5*time.Second || s.ReadTimeout != 10*time.Second || s.WriteTimeout != 90*time.Second || s.IdleTimeout != 30*time.Second || s.MaxHeaderBytes != 16384 {
		t.Fatal("transport profile changed")
	}
	if s.ErrorLog == nil || s.ErrorLog.Writer() != io.Discard {
		t.Fatal("raw error logging enabled")
	}
}

// TestCancellationDrains waits for handler cleanup rather than treating socket shutdown as a business receipt.
func TestCancellationDrains(t *testing.T) {
	entered, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	h, _ := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(cancelled)
		<-release
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := listen(t)
	done := make(chan error, 1)
	go func() { done <- h.Serve(ctx, l) }()
	response := make(chan error, 1)
	go func() {
		c := http.Client{Timeout: 15 * time.Second}
		res, err := c.Get("http://" + l.Addr().String())
		if err == nil {
			res.Body.Close()
			if res.StatusCode != 503 {
				err = errors.New("unexpected success")
			}
		}
		response <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("handler not entered")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("request context not cancelled")
	}
	select {
	case <-done:
		t.Fatal("returned before drain")
	default:
	}
	close(release)
	if err := await(t, done); err != nil {
		t.Fatal(err)
	}
	if err := await(t, response); err != nil {
		t.Fatal(err)
	}
	if err := h.Serve(context.Background(), l); err != ErrAlreadyServed {
		t.Fatal(err)
	}
}

// TestForcedClose bounds drain even when handler code ignores cancellation; the test then releases that code explicitly.
func TestForcedClose(t *testing.T) {
	entered, release, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer func() {
		close(release)
		select {
		case <-exited:
		case <-time.After(time.Second):
			t.Error("handler did not exit after explicit release")
		}
	}()
	h, _ := New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; close(exited) }))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := listen(t)
	done := make(chan error, 1)
	go func() { done <- h.Serve(ctx, l) }()
	response := make(chan error, 1)
	go func() {
		c := http.Client{Timeout: 14 * time.Second}
		res, e := c.Get("http://" + l.Addr().String())
		if res != nil {
			res.Body.Close()
		}
		response <- e
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("handler not entered")
	}
	start := time.Now()
	cancel()
	if err := await(t, done); err != ErrShutdown {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 10*time.Second || elapsed > 13*time.Second {
		t.Fatal("unexpected drain duration", elapsed)
	}
	if err := await(t, response); err == nil {
		t.Fatal("forced socket close reported success")
	}
}

// TestValidationAndListenerFailure verifies fixed errors without retaining listener diagnostics.
func TestValidationAndListenerFailure(t *testing.T) {
	if _, err := New(nil); err != ErrConfiguration {
		t.Fatal(err)
	}
	h, _ := New(http.NotFoundHandler())
	if err := h.Serve(context.Background(), nil); err != ErrConfiguration {
		t.Fatal(err)
	}
	l := listen(t)
	_ = l.Close()
	if err := h.Serve(context.Background(), l); err != ErrServe {
		t.Fatal(err)
	}
	if strings.Contains(ErrServe.Error(), l.Addr().String()) {
		t.Fatal("address leaked")
	}
}

// TestCancelledBeforeServe proves startup cancellation still closes the owned listener and joins Serve.
func TestCancelledBeforeServe(t *testing.T) {
	h, _ := New(http.NotFoundHandler())
	l := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.Serve(ctx, l); err != nil {
		t.Fatal(err)
	}
	if c, err := net.DialTimeout("tcp", l.Addr().String(), time.Second); err == nil {
		c.Close()
		t.Fatal("listener remains open")
	}
}
