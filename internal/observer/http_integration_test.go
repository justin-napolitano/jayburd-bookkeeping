//go:build integration

package observer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestObserverHTTPComposition proves real SQL receipt commit precedes HTTP acknowledgement despite two lost replies.
func TestObserverHTTPComposition(t *testing.T) {
	admin, db, _ := fixture(t)
	before := value(t, admin, "SELECT revision::text FROM ledger.book")
	pd := &proofDB{t: t, database: db, login: "synthetic_publisher"}
	cd := &proofDB{t: t, database: db, login: "synthetic_observer", loseFunction: "observe_finish"}
	consumer, e := NewConsumer(cd)
	if e != nil {
		t.Fatal(e)
	}
	consumerHTTP, e := NewConsumerHTTP(consumer, goodVerifier(), testPolicy())
	if e != nil {
		t.Fatal(e)
	}
	deliveries, creates := 0, 0
	config, _ := queueFixture()
	queue, e := NewCloudTasksTransport(config, accessTokenFunc(func(context.Context) (string, error) { return "synthetic-access", nil }), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		creates++
		if pd.active.Load() != 0 || cd.active.Load() != 0 {
			t.Fatal("network entered with SQL transaction")
		}
		var request struct {
			Task struct {
				HTTP struct {
					Body []byte `json:"body"`
				} `json:"httpRequest"`
			} `json:"task"`
		}
		if e := json.NewDecoder(r.Body).Decode(&request); e != nil {
			t.Fatal(e)
		}
		for _, want := range []int{503, 204, 204} {
			out := httptest.NewRecorder()
			consumerHTTP.ServeHTTP(out, workerRequest(ConsumerPath, string(request.Task.HTTP.Body)))
			deliveries++
			if out.Code != want {
				t.Fatalf("delivery status %d want %d", out.Code, want)
			}
		}
		// The queue accepted and delivered; its create acknowledgement is deliberately lost.
		return nil, errors.New("synthetic lost create acknowledgement")
	}))
	if e != nil {
		t.Fatal(e)
	}
	publisher, e := NewPublisher(pd, binding, queue)
	if e != nil {
		t.Fatal(e)
	}
	publisherHTTP, e := NewPublisherHTTP(publisher, goodVerifier(), testPolicy())
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		out := httptest.NewRecorder()
		publisherHTTP.ServeHTTP(out, workerRequest(PublisherPath, "{}"))
		if out.Code != 200 {
			t.Fatalf("publisher status %d", out.Code)
		}
	}
	if creates != 1 || deliveries != 3 || !cd.lost.Load() || pd.active.Load() != 0 || cd.active.Load() != 0 {
		t.Fatal("unexpected transport/transaction count")
	}
	if value(t, admin, "SELECT count(*)::text FROM integration.work_receipt WHERE outcome='observed'") != "1" || value(t, admin, "SELECT count(*)::text FROM integration.work_attempt") != "1" || value(t, admin, "SELECT state FROM integration.work_operation") != "succeeded" || value(t, admin, "SELECT revision::text FROM ledger.book") != before || value(t, admin, "SELECT count(*)::text FROM ledger.journal_entry") != "1" || value(t, admin, "SELECT count(*)::text FROM integration.outbox_message") != "1" {
		t.Fatal("duplicate receipt or financial effect")
	}
}

// TestObserverHTTPMixedSyncRole proves the adapter refuses mixed worker capability before discovery.
func TestObserverHTTPMixedSyncRole(t *testing.T) {
	admin, db, _ := fixture(t)
	exec(t, admin, `DO $x$ BEGIN IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_sync') THEN CREATE ROLE bookkeeping_sync NOLOGIN; END IF; END $x$`)
	exec(t, admin, "GRANT bookkeeping_sync TO synthetic_publisher")
	t.Cleanup(func() { exec(t, admin, "REVOKE bookkeeping_sync FROM synthetic_publisher") })
	publisher, _, _, _ := proofClients(t, db, queueFunc(func(context.Context, Task) (EnqueueResult, error) { t.Fatal("mixed role enqueued"); return "", nil }))
	h, e := NewPublisherHTTP(publisher, goodVerifier(), testPolicy())
	if e != nil {
		t.Fatal(e)
	}
	out := httptest.NewRecorder()
	h.ServeHTTP(out, workerRequest(PublisherPath, "{}"))
	if out.Code != 503 || value(t, admin, "SELECT count(*)::text FROM integration.work_operation") != "0" {
		t.Fatal("mixed role performed discovery")
	}
}
