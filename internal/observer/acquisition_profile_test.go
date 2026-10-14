package observer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestAcquisitionProfileIsolation prevents shared transport code from selecting another worker.
func TestAcquisitionProfileIsolation(t *testing.T) {
	raw, _ := encodeWakeup(Wakeup{testOperation, 1})
	calls := 0
	consumer := handlerFunc(func(ctx context.Context, body []byte) error {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 80*time.Second || time.Until(deadline) < 75*time.Second {
			t.Fatal("acquisition settlement budget missing")
		}
		if string(body) != string(raw) {
			t.Fatal("wakeup changed")
		}
		return nil
	})
	h, e := NewAcquisitionConsumerHTTP(consumer, goodVerifier(), testPolicy())
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		path   string
		status int
	}{{AcquisitionConsumerPath, 204}, {ConsumerPath, 404}, {AcquisitionPublisherPath, 404}, {"/v1/workspace", 404}, {AcquisitionConsumerPath + "?scope=other", 404}} {
		before := calls
		out := httptest.NewRecorder()
		h.ServeHTTP(out, workerRequest(tc.path, string(raw)))
		if out.Code != tc.status || tc.status != 204 && calls != before {
			t.Fatalf("cross-profile route: %s %d", tc.path, out.Code)
		}
	}
	// The reverse route is equally isolated; syntax is never a handler selector.
	old, _ := NewConsumerHTTP(consumer, goodVerifier(), testPolicy())
	out := httptest.NewRecorder()
	old.ServeHTTP(out, workerRequest(AcquisitionConsumerPath, string(raw)))
	if out.Code != 404 {
		t.Fatal("observer admitted acquisition path")
	}
	publisherCalls := 0
	pub, e := NewAcquisitionPublisherHTTP(tickerFunc(func(context.Context) (TickResult, error) { publisherCalls++; return TickResult{Recorded: 1}, nil }), goodVerifier(), testPolicy())
	if e != nil {
		t.Fatal(e)
	}
	out = httptest.NewRecorder()
	pub.ServeHTTP(out, workerRequest(AcquisitionPublisherPath, "{}"))
	if out.Code != 200 || publisherCalls != 1 {
		t.Fatal("publisher route failed")
	}
	out = httptest.NewRecorder()
	pub.ServeHTTP(out, workerRequest(PublisherPath, "{}"))
	if out.Code != 404 || publisherCalls != 1 {
		t.Fatal("publisher routes mixed")
	}
}

// TestAcquisitionHTTPRejectsBeforeWork exercises the new profile's identity, body and unknown-outcome gates.
func TestAcquisitionHTTPRejectsBeforeWork(t *testing.T) {
	raw, _ := encodeWakeup(Wakeup{testOperation, 1})
	calls := 0
	h, _ := NewAcquisitionConsumerHTTP(handlerFunc(func(context.Context, []byte) error { calls++; return errors.New("private unknown commit") }), goodVerifier(), testPolicy())
	r := workerRequest(AcquisitionConsumerPath, string(raw))
	r.Header.Del("Authorization")
	body := &failRead{}
	r.Body = body
	out := httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 401 || calls != 0 || body.reads != 0 {
		t.Fatal("identity did not precede body")
	}
	for _, bad := range []string{`{"schema_version":1,"operation_id":"` + testOperation + `","dispatch_generation":"1","scope":"x"}`, `{"schema_version":1,"schema_version":1}`, strings.Repeat(" ", 1025)} {
		out = httptest.NewRecorder()
		h.ServeHTTP(out, workerRequest(AcquisitionConsumerPath, bad))
		if out.Code != 400 || calls != 0 {
			t.Fatal("bad envelope reached worker")
		}
	}
	out = httptest.NewRecorder()
	h.ServeHTTP(out, workerRequest(AcquisitionConsumerPath, string(raw)))
	if out.Code != 503 || strings.Contains(out.Body.String(), "private") {
		t.Fatal("unknown completion acknowledged or disclosed")
	}
}

// TestAcquisitionQueueProfile checks a disjoint task name, deadline and route before token retrieval.
func TestAcquisitionQueueProfile(t *testing.T) {
	c, legacy := queueFixture()
	c.ConsumerURL = testPolicy().Audience + AcquisitionConsumerPath
	task := Task{transportTaskName(testOperation, 1, 2), 2, legacy.Body}
	tokens, calls := 0, 0
	credentials := accessTokenFunc(func(context.Context) (string, error) { tokens++; return "synthetic", nil })
	transport, e := NewAcquisitionCloudTasksTransport(c, credentials, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var v struct {
			Task struct {
				Name     string
				Deadline string               `json:"dispatchDeadline"`
				HTTP     struct{ URL string } `json:"httpRequest"`
			}
		}
		if json.NewDecoder(r.Body).Decode(&v) != nil || v.Task.Deadline != "85s" || v.Task.HTTP.URL != c.ConsumerURL || !strings.HasSuffix(v.Task.Name, "/"+task.Name) {
			t.Fatal("wrong acquisition queue profile")
		}
		return queueResponse(200, `{"name":"`+v.Task.Name+`"}`), nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = transport.Enqueue(context.Background(), legacy); e == nil || tokens != 0 || calls != 0 {
		t.Fatal("wrong target profile reached credentials")
	}
	if result, e := transport.Enqueue(context.Background(), task); e != nil || result != Accepted || calls != 1 {
		t.Fatal("acquisition task failed")
	}
	if task.Name == legacy.Name {
		t.Fatal("task identity profiles collided")
	}
	if _, e := NewCloudTasksTransport(c, credentials, roundTripFunc(nil)); e == nil {
		t.Fatal("observer accepted acquisition destination")
	}
	c.ConsumerURL = testPolicy().Audience + ConsumerPath
	if _, e := NewAcquisitionCloudTasksTransport(c, credentials, roundTripFunc(nil)); e == nil {
		t.Fatal("acquisition accepted observer destination")
	}
}
