package webhookhost

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
)

// rejectionReporter bounds local diagnostic volume and isolates request latency from the output writer.
type rejectionReporter struct {
	mu     sync.Mutex
	window time.Time
	count  int
	closed bool
	events chan webhook.Rejection
	done   chan struct{}
	once   sync.Once
	now    func() time.Time
}

// newRejectionReporter owns one bounded writer goroutine; only fixed reason labels can enter it.
func newRejectionReporter(out io.Writer) (func(webhook.Rejection), func()) {
	if out == nil {
		out = io.Discard
	}
	r := &rejectionReporter{events: make(chan webhook.Rejection, 16), done: make(chan struct{}), now: time.Now}
	logger := log.New(out, "", 0)
	go func() {
		for {
			select {
			case <-r.done:
				return
			case event := <-r.events:
				logger.Printf(`{"event":"plaid_webhook_rejected","reason":%q}`, event.String())
			}
		}
	}()
	return r.report, func() { r.once.Do(func() { r.mu.Lock(); r.closed = true; close(r.done); r.mu.Unlock() }) }
}

// report never waits for a writer; full queues and the twenty-per-minute cap drop diagnostics only.
func (r *rejectionReporter) report(reason webhook.Rejection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	now := r.now()
	if r.window.IsZero() || !now.Before(r.window.Add(time.Minute)) {
		r.window = now
		r.count = 0
	}
	if r.count >= 20 {
		return
	}
	r.count++
	select {
	case r.events <- reason:
	default:
	}
}
