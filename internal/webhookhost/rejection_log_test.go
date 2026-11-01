package webhookhost

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
)

// TestRejectionReporterBounds proves queue/rate/close limits without scheduling or real logging.
func TestRejectionReporterBounds(t *testing.T) {
	now := time.Now()
	r := &rejectionReporter{events: make(chan webhook.Rejection, 16), done: make(chan struct{}), now: func() time.Time { return now }}
	for range 100 {
		r.report(webhook.SignatureRejected)
	}
	if r.count != 20 || len(r.events) != 16 {
		t.Fatal("diagnostic volume unbounded")
	}
	for len(r.events) > 0 {
		<-r.events
	}
	now = now.Add(time.Minute)
	r.report(webhook.KeyLookupRejected)
	if r.count != 1 || len(r.events) != 1 {
		t.Fatal("window did not reset")
	}
	r.closed = true
	r.report(webhook.KeyLookupRejected)
	if r.count != 1 || len(r.events) != 1 {
		t.Fatal("closed reporter accepted new events")
	}
}

// blockedLogWriter simulates a failed trusted logging sink without accepting any request data.
type blockedLogWriter struct {
	started chan struct{}
	release chan struct{}
	written chan string
	once    sync.Once
}

// Write pauses only the single diagnostic writer, never a request goroutine.
func (w *blockedLogWriter) Write(b []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.release
	w.written <- string(b)
	return len(b), nil
}

// TestRejectionReporterBlockedWriter proves reporting/close stay nonblocking and emit only closed labels.
func TestRejectionReporterBlockedWriter(t *testing.T) {
	w := &blockedLogWriter{started: make(chan struct{}), release: make(chan struct{}), written: make(chan string, 32)}
	report, stop := newRejectionReporter(w)
	t.Cleanup(func() {
		stop()
		select {
		case <-w.release:
		default:
			close(w.release)
		}
	})
	report(webhook.Rejection(255))
	select {
	case <-w.started:
	case <-time.After(time.Second):
		t.Fatal("writer not reached")
	}
	done := make(chan struct{})
	go func() {
		for range 1000 {
			report(webhook.SignatureRejected)
		}
		stop()
		stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		close(w.release)
		t.Fatal("logging blocked request/shutdown")
	}
	close(w.release)
	select {
	case line := <-w.written:
		if line != "{\"event\":\"plaid_webhook_rejected\",\"reason\":\"unclassified\"}\n" || strings.Contains(line, "private") {
			t.Fatal("unsafe log projection")
		}
	case <-time.After(time.Second):
		t.Fatal("fixed log missing")
	}
}
