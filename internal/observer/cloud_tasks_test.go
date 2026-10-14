package observer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTripFunc injects synthetic HTTP outcomes without any network access.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip executes the supplied single-request fake.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// accessTokenFunc supplies synthetic context-aware access credentials.
type accessTokenFunc func(context.Context) (string, error)

// AccessToken executes only the injected credential provider.
func (f accessTokenFunc) AccessToken(c context.Context) (string, error) { return f(c) }

// queueFixture fixes one fictional queue and deterministic wakeup.
func queueFixture() (QueueConfig, Task) {
	p := testPolicy()
	body, _ := encodeWakeup(Wakeup{testOperation, 1})
	return QueueConfig{"example-project", "us-central1", "observer", p.Audience + ConsumerPath, p.Audience, p.Email}, Task{taskName(testOperation, 1), 1, body}
}

// queueResponse constructs a private in-memory response and body.
func queueResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

// TestCloudTasksWire proves stable routing, exact identifier bytes and one request outside SQL.
func TestCloudTasksWire(t *testing.T) {
	c, task := queueFixture()
	db := &testDB{role: true}
	calls, tokens := 0, 0
	transport, e := NewCloudTasksTransport(c, accessTokenFunc(func(ctx context.Context) (string, error) {
		tokens++
		d, ok := ctx.Deadline()
		if !ok || time.Until(d) > 10*time.Second {
			t.Fatal("unbounded token")
		}
		return "synthetic-access", nil
	}), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if db.active != 0 {
			t.Fatal("queue call in SQL transaction")
		}
		if r.Method != "POST" || r.URL.String() != "https://cloudtasks.googleapis.com/v2/projects/example-project/locations/us-central1/queues/observer/tasks" || r.Header.Get("Authorization") != "Bearer synthetic-access" {
			t.Fatal("wrong request")
		}
		var body struct {
			ResponseView string `json:"responseView"`
			Task         struct {
				Name     string `json:"name"`
				Deadline string `json:"dispatchDeadline"`
				HTTP     struct {
					URL, Body string
					Method    string `json:"httpMethod"`
					Headers   map[string]string
					OIDC      struct {
						Email    string `json:"serviceAccountEmail"`
						Audience string
					} `json:"oidcToken"`
				} `json:"httpRequest"`
			}
		}
		if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
			t.Fatal(e)
		}
		raw, e := base64.StdEncoding.DecodeString(body.Task.HTTP.Body)
		if e != nil || string(raw) != string(task.Body) || body.ResponseView != "BASIC" || body.Task.Deadline != "65s" || body.Task.HTTP.Method != "POST" || body.Task.HTTP.URL != c.ConsumerURL || body.Task.HTTP.OIDC.Email != c.ServiceAccount || body.Task.HTTP.OIDC.Audience != c.Audience || body.Task.HTTP.Headers["Content-Type"] != "application/json" || body.Task.Name != "projects/example-project/locations/us-central1/queues/observer/tasks/"+task.Name {
			t.Fatal("incorrect wire identity")
		}
		return queueResponse(200, `{"name":"`+body.Task.Name+`"}`), nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	p, e := NewPublisher(db, testBinding, transport)
	if e != nil {
		t.Fatal(e)
	}
	out, e := p.Tick(context.Background())
	if e != nil || out.Recorded != 1 || calls != 1 || tokens != 1 {
		t.Fatalf("tick %v %v calls=%d tokens=%d", out, e, calls, tokens)
	}
}

// TestCloudTasksUncertainty keeps uncertain create outcomes recoverable without retries or raw errors.
func TestCloudTasksUncertainty(t *testing.T) {
	c, task := queueFixture()
	name := "projects/example-project/locations/us-central1/queues/observer/tasks/" + task.Name
	for _, tc := range []struct {
		name   string
		status int
		body   string
		err    error
		want   EnqueueResult
	}{
		{"accepted", 201, `{"name":"` + name + `"}`, nil, Accepted},
		{"duplicate", 409, "private-provider-body", nil, AlreadyExists},
		{"throttled", 429, "private-provider-body", nil, Transient},
		{"server", 500, "private-provider-body", nil, Transient},
		{"timeout", 504, "private-provider-body", nil, Timeout},
		{"denied", 403, "private-provider-body", nil, Rejected},
		{"redirect", 302, "", nil, Transient},
		{"wrong-name", 200, `{"name":"other"}`, nil, Transient},
		{"lost-reply", 0, "", errors.New("private-address"), Transient},
		{"bad-json", 200, "private-provider-body", nil, Transient},
		{"huge", 200, strings.Repeat("x", 16385), nil, Transient},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, tokens := 0, 0
			q, _ := NewCloudTasksTransport(c, accessTokenFunc(func(context.Context) (string, error) { tokens++; return "synthetic", nil }), roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.err != nil {
					return nil, tc.err
				}
				resp := queueResponse(tc.status, tc.body)
				resp.Header.Set("Location", "https://attacker.invalid/secret")
				return resp, nil
			}))
			got, e := q.Enqueue(context.Background(), task)
			if e != nil || got != tc.want || calls != 1 || tokens != 1 {
				t.Fatalf("got %s err=%v calls=%d tokens=%d", got, e, calls, tokens)
			}
		})
	}
}

// TestCloudTasksCancellationAndValidation proves malformed input precedes credentials and cancellation is bounded.
func TestCloudTasksCancellationAndValidation(t *testing.T) {
	c, task := queueFixture()
	calls, tokens := 0, 0
	q, _ := NewCloudTasksTransport(c, accessTokenFunc(func(ctx context.Context) (string, error) { tokens++; return "synthetic", nil }), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		<-r.Context().Done()
		return nil, r.Context().Err()
	}))
	bad := task
	bad.Name = "wrong"
	if _, e := q.Enqueue(context.Background(), bad); e == nil || tokens != 0 || calls != 0 {
		t.Fatal("invalid task acquired credentials")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	got, e := q.Enqueue(ctx, task)
	if e != nil || got != Timeout || calls != 1 {
		t.Fatalf("cancellation %s %v", got, e)
	}
	for _, mutate := range []func(*QueueConfig){func(c *QueueConfig) { c.Project = "../other" }, func(c *QueueConfig) { c.Queue = "a/b" }, func(c *QueueConfig) { c.ConsumerURL = "http://localhost" }, func(c *QueueConfig) { c.Audience = "https://other.run.app" }, func(c *QueueConfig) { c.ServiceAccount = "worker@other-project.iam.gserviceaccount.com" }} {
		bad := c
		mutate(&bad)
		if _, e := NewCloudTasksTransport(bad, q.tokens, q.client.Transport); e == nil {
			t.Fatal("invalid routing accepted")
		}
	}
}

// TestCloudTasksCredentialFailure never sends when credentials fail, contain header injection or expire in flight.
func TestCloudTasksCredentialFailure(t *testing.T) {
	c, task := queueFixture()
	for _, tc := range []struct {
		name, token string
		err         error
		cancel      bool
		want        EnqueueResult
	}{
		{name: "provider-error", err: errors.New("private-credential-location"), want: Transient},
		{name: "empty", want: Rejected},
		{name: "header-injection", token: "secret\r\nOther: value", want: Rejected},
		{name: "cancelled", cancel: true, want: Timeout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			q, _ := NewCloudTasksTransport(c, accessTokenFunc(func(context.Context) (string, error) {
				if tc.cancel {
					cancel()
				}
				return tc.token, tc.err
			}), roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected") }))
			got, e := q.Enqueue(ctx, task)
			if got != tc.want || e != nil || calls != 0 {
				t.Fatalf("got %s err=%v calls=%d", got, e, calls)
			}
		})
	}
}
