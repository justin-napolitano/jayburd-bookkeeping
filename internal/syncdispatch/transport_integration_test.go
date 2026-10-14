//go:build integration

package syncdispatch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
)

// syntheticVerifier supplies fixed verified claims; cryptographic verification is tested separately.
type syntheticVerifier struct{ policy observer.CallerPolicy }

// Verify binds only the fictional bearer to an exact audience and service identity.
func (v syntheticVerifier) Verify(context.Context, string, string) (observer.VerifiedToken, error) {
	return observer.VerifiedToken{Issuer: "https://accounts.google.com", Audience: v.policy.Audience, Email: v.policy.Email, Subject: v.policy.Subject, EmailVerified: true, IssuedAt: time.Now().Add(-time.Minute).Unix(), Expires: time.Now().Add(time.Minute).Unix()}, nil
}

// syntheticAccessToken keeps all queue authentication in memory without credential discovery.
type syntheticAccessToken struct{}

// AccessToken returns an explicit fictional access token to the intercepted queue transport.
func (syntheticAccessToken) AccessToken(context.Context) (string, error) { return "synthetic", nil }

// TestDispatchHTTPComposition traverses private HTTP, queue encoding, SQL claims and the actual acquisition runner.
func TestDispatchHTTPComposition(t *testing.T) {
	owner, db, binding := dispatchSetup(t)
	policy := observer.CallerPolicy{Audience: "https://acquisition-123.us-central1.run.app", Email: "worker@synthetic-project.iam.gserviceaccount.com", Subject: "123456789012345678901"}
	var wakeup []byte
	queueCalls := 0
	queue, err := observer.NewAcquisitionCloudTasksTransport(observer.QueueConfig{
		Project: "synthetic-project", Location: "us-central1", Queue: "acquisition",
		ConsumerURL: policy.Audience + observer.AcquisitionConsumerPath, Audience: policy.Audience, ServiceAccount: policy.Email,
	}, syntheticAccessToken{}, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		queueCalls++
		var body struct {
			Task struct {
				Name             string
				DispatchDeadline string
				HTTPRequest      struct {
					URL  string
					Body []byte
				}
			}
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.Task.DispatchDeadline != "85s" || body.Task.HTTPRequest.URL != policy.Audience+observer.AcquisitionConsumerPath {
			t.Fatal("incorrect queue profile")
		}
		wakeup = append([]byte(nil), body.Task.HTTPRequest.Body...)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"name":"` + body.Task.Name + `"}`))}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	pubConn := connect(t, db, "synthetic_publisher")
	defer pubConn.Close(context.Background())
	pub, err := NewPublisher(&lossPool{conn: pubConn}, binding, queue)
	if err != nil {
		t.Fatal(err)
	}
	pubHTTP, err := observer.NewAcquisitionPublisherHTTP(pub, syntheticVerifier{policy}, policy)
	if err != nil {
		t.Fatal(err)
	}
	request := func(h http.Handler, path, body string, authenticated bool) int {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if authenticated {
			r.Header.Set("Authorization", "Bearer synthetic")
		}
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out.Code
	}
	if status := request(pubHTTP, observer.AcquisitionPublisherPath, "{}", true); status != 200 || queueCalls != 1 || len(wakeup) == 0 {
		t.Fatalf("publisher status %d calls %d", status, queueCalls)
	}
	runner, _, calls := realRunner(t, db, "")
	worker := connect(t, db, "synthetic_sync")
	defer worker.Close(context.Background())
	consumer, err := NewConsumer(&lossPool{conn: worker}, runner)
	if err != nil {
		t.Fatal(err)
	}
	h, err := observer.NewAcquisitionConsumerHTTP(consumer, syntheticVerifier{policy}, policy)
	if err != nil {
		t.Fatal(err)
	}
	if request(h, observer.AcquisitionConsumerPath, string(wakeup), false) != 401 || *calls != 0 {
		t.Fatal("unauthenticated provider access")
	}
	for i := 0; i < 2; i++ {
		if status := request(h, observer.AcquisitionConsumerPath, string(wakeup), true); status != 204 {
			t.Fatalf("consumer status %d", status)
		}
	}
	if *calls != 1 || value(t, owner, "SELECT count(*)::text FROM banking.source_observation") != "1" || value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("duplicate acquisition or automatic posting")
	}
}
