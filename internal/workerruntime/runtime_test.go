package workerruntime

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
)

// rtFunc injects metadata responses without contacting Google.
type rtFunc func(*http.Request) (*http.Response, error)

// RoundTrip delegates an explicit test transport.
func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// config constructs fictional identifiers with the real fixed acquisition route profile.
func config(mode string) Config {
	c := Config{Mode: mode, Instance: "synthetic-project:us-central1:jayburd-dev", Listen: "127.0.0.1:8080", Caller: observer.CallerPolicy{Audience: "https://worker-example.run.app", Email: "caller@synthetic-project.iam.gserviceaccount.com", Subject: "123456789012345678901"}}
	if mode == "publisher" {
		c.BindingID = "11111111-1111-4111-8111-111111111111"
		c.Queue = observer.QueueConfig{Project: "synthetic-project", Location: "us-central1", Queue: "synthetic-queue", ConsumerURL: "https://consumer-example.run.app/internal/acquisition/consume", Audience: "https://consumer-example.run.app", ServiceAccount: "caller@synthetic-project.iam.gserviceaccount.com"}
	}
	return c
}

// TestSharedPublisherConfiguration pins explicit shared mode and excludes a configured binding.
func TestSharedPublisherConfiguration(t *testing.T) {
	c := config("publisher")
	c.PublisherPolicy = "shared-v1"
	if c.Validate() == nil { t.Fatal("shared mode accepted fixed binding") }
	c.BindingID = ""
	if e := c.Validate(); e != nil { t.Fatal(e) }
	if _, e := New(c, Dependencies{DB: &blockedDB{}, Verifier: &badVerifier{}, Transport: inertTransport{}}); e != nil { t.Fatal(e) }
	c.PublisherPolicy = "unknown"
	if c.Validate() == nil { t.Fatal("unknown policy accepted") }
	c = config("consumer")
	c.PublisherPolicy = "shared-v1"
	if c.Validate() == nil { t.Fatal("consumer accepted publisher policy") }
}

// blockedDB fails if rejected HTTP requests reach the database.
type blockedDB struct{ calls atomic.Int32 }

// BeginTx records forbidden access without opening a database connection.
func (b *blockedDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	b.calls.Add(1)
	return nil, errors.New("synthetic database diagnostic")
}

// badVerifier supplies no trusted identity.
type badVerifier struct{ calls atomic.Int32 }

// Verify ensures signature failure never becomes caller authority.
func (v *badVerifier) Verify(context.Context, string, string) (observer.VerifiedToken, error) {
	v.calls.Add(1)
	return observer.VerifiedToken{}, errors.New("synthetic token diagnostic")
}

// inertRunner is configuration-only and performs no provider work.
type inertRunner struct{}

// Run fails closed if an unauthorized test request crosses the HTTP boundary.
func (inertRunner) Run(context.Context, acquisition.Scope, string) (acquisition.Outcome, error) {
	return acquisition.Outcome{}, ErrUnavailable
}

// inertTransport is configuration-only and performs no queue work.
type inertTransport struct{}

// Enqueue records no transport acceptance.
func (inertTransport) Enqueue(context.Context, observer.Task) (observer.EnqueueResult, error) {
	return observer.Transient, nil
}

// TestMetadataBoundaries verifies fixed destinations, Google headers, size/expiry limits and safe errors.
func TestMetadataBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, body, flavor string
		status             int
		ok                 bool
	}{{"valid", `{"access_token":"synthetic-token","token_type":"Bearer","expires_in":3600}`, "Google", 200, true}, {"oversize", strings.Repeat("x", 16385), "Google", 200, false}, {"expired", `{"access_token":"synthetic-token","token_type":"Bearer","expires_in":0}`, "Google", 200, false}, {"wrong_flavor", `{}`, "", 200, false}, {"redirect", `{}`, "Google", 302, false}, {"bad_token", `{"access_token":"unsafe\nvalue","token_type":"Bearer","expires_in":60}`, "Google", 200, false}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			m, e := NewMetadataTokens(rtFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.String() != "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token" || r.Header.Get("Metadata-Flavor") != "Google" || r.Method != "GET" {
					t.Fatal("unpinned metadata")
				}
				if d, ok := r.Context().Deadline(); !ok || time.Until(d) > 5*time.Second {
					t.Fatal("unbounded metadata")
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Metadata-Flavor": []string{tc.flavor}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			}))
			if e != nil {
				t.Fatal(e)
			}
			token, e := m.AccessToken(context.Background())
			if (e == nil) != tc.ok || calls != 1 || (!tc.ok && token != "") {
				t.Fatalf("metadata outcome %v", e)
			}
			if e != nil && e.Error() != "worker_dependency_unavailable" {
				t.Fatal("raw diagnostics exposed")
			}
		})
	}
}

// TestMetadataIdentityAndOverrides encodes an explicit audience and rejects ambient credential sources.
func TestMetadataIdentityAndOverrides(t *testing.T) {
	m, e := NewMetadataTokens(rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/computeMetadata/v1/instance/service-accounts/default/identity" || r.URL.Query().Get("audience") != "https://app.infisical.com" || r.URL.Query().Get("format") != "full" {
			t.Fatal("identity target")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Metadata-Flavor": []string{"Google"}}, Body: io.NopCloser(strings.NewReader("synthetic.payload.signature"))}, nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.IdentityToken(context.Background(), "https://app.infisical.com"); e != nil {
		t.Fatal(e)
	}
	for _, aud := range []string{"", "http://attacker.invalid", "https://u:p@attacker.invalid", "https://example.com/?query"} {
		if _, e = m.IdentityToken(context.Background(), aud); e == nil {
			t.Fatal("invalid audience")
		}
	}
	t.Setenv("GCE_METADATA_HOST", "synthetic.invalid")
	if _, e = NewMetadataTokens(http.DefaultTransport); e == nil {
		t.Fatal("metadata override accepted")
	}
}

// TestWorkerConfiguration rejects role substitution, ambiguous flags and unmanaged consumer dependencies before network access.
func TestWorkerConfiguration(t *testing.T) {
	for _, mode := range []string{"publisher", "consumer"} {
		if e := config(mode).Validate(); e != nil {
			t.Fatal(e)
		}
	}
	for _, mutate := range []func(*Config){func(c *Config) { c.Mode = "api" }, func(c *Config) { c.Listen = "attacker.invalid:8080" }, func(c *Config) { c.Listen = "0.0.0.0:08080" }, func(c *Config) { c.Instance = "synthetic-project:europe-west1:db" }, func(c *Config) { c.Queue.Project = "foreign-project" }, func(c *Config) { c.BindingID = "" }} {
		c := config("publisher")
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal("unsafe configuration")
		}
	}
	if _, e := Managed(context.Background(), config("consumer"), ConsumerAdapters{}); e == nil {
		t.Fatal("consumer missing durable adapters")
	}
	d := DirectTransport()
	defer d.CloseIdleConnections()
	if d.Proxy != nil || d.MaxConnsPerHost != 2 {
		t.Fatal("ambient transport")
	}
}

// TestHostedAuthBeforeSQL starts the real bounded host and proves rejected calls cannot access the database.
func TestHostedAuthBeforeSQL(t *testing.T) {
	for _, mode := range []string{"publisher", "consumer"} {
		t.Run(mode, func(t *testing.T) {
			db := &blockedDB{}
			v := &badVerifier{}
			var closed atomic.Int32
			d := Dependencies{DB: db, Verifier: v, Close: func() { closed.Add(1) }}
			if mode == "publisher" {
				d.Transport = inertTransport{}
			} else {
				d.Runner = inertRunner{}
			}
			r, e := New(config(mode), d)
			if e != nil {
				t.Fatal(e)
			}
			l, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- r.Serve(ctx, l) }()
			client := &http.Client{Timeout: time.Second, Transport: DirectTransport()}
			defer client.CloseIdleConnections()
			path := "/internal/acquisition/consume"
			if mode == "publisher" {
				path = "/internal/acquisition/publish"
			}
			u := url.URL{Scheme: "http", Host: l.Addr().String(), Path: path}
			req, _ := http.NewRequest("POST", u.String(), strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer synthetic")
			req.Header.Set("Content-Type", "application/json")
			resp, e := client.Do(req)
			if e != nil {
				cancel()
				<-done
				t.Fatal(e)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode < 400 || strings.Contains(string(body), "diagnostic") || db.calls.Load() != 0 || v.calls.Load() != 1 {
				t.Fatal("unsafe authentication boundary")
			}
			cancel()
			if e = <-done; e != nil {
				t.Fatal(e)
			}
			r.Close()
			if closed.Load() != 1 {
				t.Fatal("cleanup ownership")
			}
		})
	}
}

// TestMetadataCancellation never returns a token after cancellation or starts I/O for a canceled caller.
func TestMetadataCancellation(t *testing.T) {
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())
	m, e := NewMetadataTokens(rtFunc(func(*http.Request) (*http.Response, error) {
		calls++
		cancel()
		return &http.Response{StatusCode: 200, Header: http.Header{"Metadata-Flavor": []string{"Google"}}, Body: io.NopCloser(strings.NewReader(`{"access_token":"synthetic-token","token_type":"Bearer","expires_in":60}`))}, nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	if token, e := m.AccessToken(ctx); e == nil || token != "" || calls != 1 {
		t.Fatal("canceled token exposed")
	}
	if _, e = m.AccessToken(ctx); e == nil || calls != 1 {
		t.Fatal("canceled caller started request")
	}
}

// TestInfisicalIdentityAudience permits only the reviewed canonical machine UUID outside HTTPS audiences.
func TestInfisicalIdentityAudience(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	calls := 0
	m, e := NewMetadataTokens(rtFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("audience") != id || r.URL.Host != "metadata.google.internal" {
			t.Fatal("audience binding")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Metadata-Flavor": []string{"Google"}}, Body: io.NopCloser(strings.NewReader("synthetic.payload.signature"))}, nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.IdentityToken(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{"00000000-0000-0000-0000-000000000000", "11111111-1111-4111-8111-11111111111G", id + "\n", "other-string"} {
		if _, e = m.IdentityToken(context.Background(), bad); e == nil {
			t.Fatal("invalid identity accepted")
		}
	}
	if calls != 1 {
		t.Fatal("invalid identity reached metadata")
	}
}
