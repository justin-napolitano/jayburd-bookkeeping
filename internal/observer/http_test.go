package observer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// verifierFunc injects a verified-claim result without consulting a provider.
type verifierFunc func(context.Context, string, string) (VerifiedToken, error)

// Verify forwards only the injected synthetic verification behavior.
func (f verifierFunc) Verify(c context.Context, t, a string) (VerifiedToken, error) {
	return f(c, t, a)
}

// handlerFunc exposes bounded synthetic consumer behavior.
type handlerFunc func(context.Context, []byte) error

// Handle executes the synthetic completion or fault.
func (f handlerFunc) Handle(c context.Context, b []byte) error { return f(c, b) }

// tickerFunc exposes synthetic partial publisher counts.
type tickerFunc func(context.Context) (TickResult, error)

// Tick executes the synthetic publisher fault or result.
func (f tickerFunc) Tick(c context.Context) (TickResult, error) { return f(c) }

// testPolicy fixes a fictional managed workload identity.
func testPolicy() CallerPolicy {
	return CallerPolicy{"https://observer-synthetic.run.app", "worker@example-project.iam.gserviceaccount.com", "112010400000000710080"}
}

// testClaims produces short-lived authorized synthetic claims.
func testClaims() VerifiedToken {
	p := testPolicy()
	n := time.Now().Unix()
	return VerifiedToken{"https://accounts.google.com", p.Audience, p.Subject, p.Email, true, n - 1, n + 3599}
}

// goodVerifier returns synthetic authorized claims without credential access.
func goodVerifier() TokenVerifier {
	return verifierFunc(func(context.Context, string, string) (VerifiedToken, error) { return testClaims(), nil })
}

// workerRequest supplies the exact private wire protocol.
func workerRequest(path, body string) *http.Request {
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer synthetic-token")
	r.Header.Set("Content-Type", "application/json")
	return r
}

// failRead proves the auth gate runs before request body access.
type failRead struct{ reads int }

// Read records unauthorized body access as a deterministic failure.
func (b *failRead) Read([]byte) (int, error) { b.reads++; return 0, errors.New("private-body") }

// Close satisfies request ownership without reading input.
func (b *failRead) Close() error { return nil }

// TestWorkerHTTPAuthentication rejects invalid identity and headers before body and business work.
func TestWorkerHTTPAuthentication(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*VerifiedToken)
		header func(*http.Request)
		err    error
	}{
		{name: "spoofed-delivery", header: func(r *http.Request) {
			r.Header.Del("Authorization")
			r.Header.Set("X-CloudTasks-TaskName", "trusted-looking-task")
		}},
		{name: "missing", header: func(r *http.Request) { r.Header.Del("Authorization") }},
		{name: "multiple", header: func(r *http.Request) { r.Header.Add("Authorization", "Bearer other") }},
		{name: "alternate", header: func(r *http.Request) { r.Header.Set("X-Serverless-Authorization", "Bearer spoof") }},
		{name: "joined", header: func(r *http.Request) { r.Header.Set("Authorization", "Bearer synthetic,other") }},
		{name: "large", header: func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+strings.Repeat("x", 8192)) }},
		{name: "issuer", mutate: func(c *VerifiedToken) { c.Issuer = "accounts.google.com" }},
		{name: "audience", mutate: func(c *VerifiedToken) { c.Audience = "https://other.run.app" }},
		{name: "subject", mutate: func(c *VerifiedToken) { c.Subject = "112010400000000710081" }},
		{name: "email", mutate: func(c *VerifiedToken) { c.Email = "other@example-project.iam.gserviceaccount.com" }},
		{name: "unverified", mutate: func(c *VerifiedToken) { c.EmailVerified = false }},
		{name: "expired", mutate: func(c *VerifiedToken) { c.Expires = time.Now().Unix() }},
		{name: "future", mutate: func(c *VerifiedToken) { c.IssuedAt = time.Now().Unix() + 30 }},
		{name: "long-lived", mutate: func(c *VerifiedToken) { c.Expires = c.IssuedAt + 3601 }},
		{name: "verify-error", err: errors.New("private-token-or-provider-payload")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			claims := testClaims()
			if tc.mutate != nil {
				tc.mutate(&claims)
			}
			v := verifierFunc(func(ctx context.Context, token, aud string) (VerifiedToken, error) {
				d, ok := ctx.Deadline()
				if !ok || time.Until(d) > 5*time.Second {
					t.Fatal("unbounded verifier")
				}
				return claims, tc.err
			})
			h, e := NewConsumerHTTP(handlerFunc(func(context.Context, []byte) error { calls++; return nil }), v, testPolicy())
			if e != nil {
				t.Fatal(e)
			}
			r := workerRequest(ConsumerPath, "{}")
			if tc.header != nil {
				tc.header(r)
			}
			body := &failRead{}
			r.Body = body
			out := httptest.NewRecorder()
			h.ServeHTTP(out, r)
			if out.Code != 401 || calls != 0 || body.reads != 0 || strings.Contains(out.Body.String(), "private") {
				t.Fatalf("auth boundary: %d %d %d", out.Code, calls, body.reads)
			}
		})
	}
}

// TestWorkerHTTPProtocol proves exact envelopes and uncertain completion never acknowledge.
func TestWorkerHTTPProtocol(t *testing.T) {
	raw, _ := encodeWakeup(Wakeup{testOperation, 1})
	for _, tc := range []struct {
		name          string
		alter         func(*http.Request)
		body          string
		err           error
		status, calls int
	}{
		{name: "done", body: string(raw), status: 204, calls: 1},
		{name: "unknown-commit", body: string(raw), err: &Failure{Code: "commit_unknown", Retryable: true}, status: 503, calls: 1},
		{name: "busy", body: string(raw), err: &Failure{Code: "busy", Retryable: true}, status: 503, calls: 1},
		{name: "raw-error", body: string(raw), err: errors.New("private-dsn"), status: 503, calls: 1},
		{name: "method", body: string(raw), alter: func(r *http.Request) { r.Method = "GET" }, status: 405},
		{name: "path", body: string(raw), alter: func(r *http.Request) { r.URL.Path = PublisherPath }, status: 404},
		{name: "query", body: string(raw), alter: func(r *http.Request) { r.URL.RawQuery = "scope=other" }, status: 404},
		{name: "type", body: string(raw), alter: func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, status: 400},
		{name: "encoding", body: string(raw), alter: func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, status: 400},
		{name: "duplicate", body: strings.Replace(string(raw), `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1), status: 400},
		{name: "unknown-field", body: strings.Replace(string(raw), `"schema_version":1`, `"schema_version":1,"actor":"spoof"`, 1), status: 400},
		{name: "large", body: strings.Repeat("x", 1025), status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			h, _ := NewConsumerHTTP(handlerFunc(func(ctx context.Context, b []byte) error {
				calls++
				d, ok := ctx.Deadline()
				if !ok || time.Until(d) > 60*time.Second {
					t.Fatal("unbounded handler")
				}
				return tc.err
			}), goodVerifier(), testPolicy())
			r := workerRequest(ConsumerPath, tc.body)
			if tc.alter != nil {
				tc.alter(r)
			}
			out := httptest.NewRecorder()
			h.ServeHTTP(out, r)
			if out.Code != tc.status || calls != tc.calls || strings.Contains(out.Body.String(), "private-dsn") {
				t.Fatalf("status=%d calls=%d", out.Code, calls)
			}
		})
	}
	calls := 0
	h, _ := NewPublisherHTTP(tickerFunc(func(context.Context) (TickResult, error) {
		calls++
		return TickResult{Discovered: 2, Candidates: 1, EnqueueCalls: 1, Recorded: 1}, nil
	}), goodVerifier(), testPolicy())
	for _, body := range []string{`{"binding":"caller"}`, `{} {}`, `{}`, " {}\n"} {
		r := workerRequest(PublisherPath, body)
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if (strings.TrimSpace(body) == "{}") != (out.Code == 200) {
			t.Fatalf("publisher status %d", out.Code)
		}
	}
	if calls != 2 {
		t.Fatalf("publisher calls %d", calls)
	}
}

// TestWorkerHTTPAdmission proves overlap is bounded and caller cancellation reaches completion.
func TestWorkerHTTPAdmission(t *testing.T) {
	entered := make(chan struct{})
	done := make(chan struct{})
	raw, _ := encodeWakeup(Wakeup{testOperation, 1})
	h, _ := NewConsumerHTTP(handlerFunc(func(ctx context.Context, _ []byte) error { close(entered); <-ctx.Done(); return ctx.Err() }), goodVerifier(), testPolicy())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := httptest.NewRecorder()
	go func() {
		defer close(done)
		h.ServeHTTP(first, workerRequest(ConsumerPath, string(raw)).WithContext(ctx))
	}()
	<-entered
	second := httptest.NewRecorder()
	h.ServeHTTP(second, workerRequest(ConsumerPath, string(raw)))
	if second.Code != 503 {
		t.Fatal(second.Code)
	}
	cancel()
	<-done
	if first.Code != 503 {
		t.Fatal(first.Code)
	}
	expired, stop := context.WithCancel(context.Background())
	stop()
	out := httptest.NewRecorder()
	h.ServeHTTP(out, workerRequest(ConsumerPath, string(raw)).WithContext(expired))
	if out.Code != 401 {
		t.Fatal(out.Code)
	}
}

// TestWorkerConfiguration rejects ambiguous destinations and mixed role guard omissions.
func TestWorkerConfiguration(t *testing.T) {
	for _, aud := range []string{"http://worker.run.app", "https://worker.run.app.evil.invalid", "https://worker.run.app:443", "https://worker.run.app/x", "https://worker.run.app?", "https://worker.run.app#", "https://u@worker.run.app"} {
		p := testPolicy()
		p.Audience = aud
		if _, e := NewConsumerHTTP(handlerFunc(func(context.Context, []byte) error { return nil }), goodVerifier(), p); e == nil {
			t.Fatal("accepted", aud)
		}
	}
	if !strings.Contains(roleQuery, "'bookkeeping_sync'") {
		t.Fatal("sync capability missing from SQL role guard")
	}
	var _ io.ReadCloser = &failRead{}
}
