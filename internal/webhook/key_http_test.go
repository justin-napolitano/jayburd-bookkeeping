package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// transportFunc intercepts every request without opening any network socket.
type transportFunc func(*http.Request) (*http.Response, error)

// RoundTrip forwards only the explicitly injected synthetic request.
func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// credentialsFunc resolves test-only client credentials.
type credentialsFunc func(context.Context) (Credentials, error)

// Resolve forwards bounded credential ownership to the injected implementation.
func (f credentialsFunc) Resolve(c context.Context) (Credentials, error) { return f(c) }

// closeBody observes ownership of provider response cleanup.
type closeBody struct {
	io.Reader
	closed *bool
}

// Close records cleanup even after failure.
func (b closeBody) Close() error { *b.closed = true; return nil }

// TestFixedKeyTransport checks endpoint, credentials/version, strict metadata and body ownership.
func TestFixedKeyTransport(t *testing.T) {
	_, key, _ := fixture(t)
	body, _ := json.Marshal(map[string]any{"key": key, "request_id": "synthetic-request"})
	closed := false
	calls := 0
	p, e := NewHTTPKeys(transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://sandbox.plaid.com/webhook_verification_key/get" || r.Method != "POST" || r.Header.Get("Plaid-Version") != "2020-09-14" {
			t.Fatal("unfixed provider destination")
		}
		var got map[string]string
		if json.NewDecoder(r.Body).Decode(&got) != nil || len(got) != 3 || got["key_id"] != key.ID || got["client_id"] != "synthetic-client" || got["secret"] != "synthetic-secret" {
			t.Fatal("invalid private request")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: closeBody{strings.NewReader(string(body)), &closed}, Request: r}, nil
	}), credentialsFunc(func(context.Context) (Credentials, error) {
		return Credentials{"synthetic-client", "synthetic-secret"}, nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	got, e := p.Get(context.Background(), key.ID)
	if e != nil || got.ID != key.ID || calls != 1 || !closed {
		t.Fatal("key fetch or body cleanup failed")
	}
}

// TestKeyTransportFailures rejects redirects, oversized valid prefixes and duplicate metadata without leaking diagnostics.
func TestKeyTransportFailures(t *testing.T) {
	_, key, _ := fixture(t)
	valid, _ := json.Marshal(map[string]any{"key": key, "request_id": "synthetic-request"})
	for _, tc := range []struct {
		status int
		body   string
	}{{302, string(valid)}, {500, "synthetic-private-error"}, {200, string(valid) + strings.Repeat(" ", 16<<10)}, {200, strings.TrimSuffix(string(valid), "}") + `,"request_id":"duplicate"}`}, {200, strings.Replace(string(valid), `"expired_at":null`, `"expired_at":1.25`, 1)}} {
		closed := false
		calls := 0
		p, _ := NewHTTPKeys(transportFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": []string{"https://evil.invalid"}}, Body: closeBody{strings.NewReader(tc.body), &closed}, Request: r}, nil
		}), credentialsFunc(func(context.Context) (Credentials, error) {
			return Credentials{"synthetic-client", "synthetic-secret"}, nil
		}))
		if _, e := p.Get(context.Background(), key.ID); e == nil || strings.Contains(e.Error(), "synthetic") || calls != 1 || !closed {
			t.Fatal("unsafe transport failure")
		}
	}
	p, _ := NewHTTPKeys(transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("credential failure reached provider")
		return nil, nil
	}), credentialsFunc(func(context.Context) (Credentials, error) { return Credentials{}, errors.New("synthetic-secret") }))
	if _, e := p.Get(context.Background(), key.ID); e == nil || strings.Contains(e.Error(), "synthetic") {
		t.Fatal("credential failure escaped")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := p.Get(ctx, key.ID); e == nil {
		t.Fatal("cancel accepted")
	}
}
