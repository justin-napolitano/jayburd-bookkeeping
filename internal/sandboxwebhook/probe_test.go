package sandboxwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
)

// transportFunc confines every provider call in this suite to synthetic in-memory responses.
type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// providerFunc records delegation without accessing a provider or SQL database.
type providerFunc func(context.Context, acquisition.FetchRequest) ([]byte, error)

func (f providerFunc) Fetch(ctx context.Context, r acquisition.FetchRequest) ([]byte, error) {
	return f(ctx, r)
}

// request contains only independent synthetic metadata and credentials.
func request() acquisition.FetchRequest {
	return acquisition.FetchRequest{Environment: "sandbox", AccountID: "synthetic-account", Credentials: acquisition.ProviderCredentials{AccessToken: "synthetic-token", ClientID: "synthetic-client", Secret: "synthetic-secret"}}
}

func TestProbeBoundsAndWire(t *testing.T) {
	calls, fetches := 0, 0
	wire := transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.String() != "https://sandbox.plaid.com/sandbox/item/fire_webhook" || r.Header.Get("Plaid-Version") != "2020-09-14" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("wrong fixed request profile")
		}
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 10*time.Second {
			t.Fatal("missing short deadline")
		}
		var got map[string]string
		if json.NewDecoder(r.Body).Decode(&got) != nil || len(got) != 5 || got["client_id"] != "synthetic-client" || got["secret"] != "synthetic-secret" || got["access_token"] != "synthetic-token" || got["webhook_type"] != "TRANSACTIONS" || got["webhook_code"] != "SYNC_UPDATES_AVAILABLE" {
			t.Fatal("wrong synthetic wire fields")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"webhook_fired":true,"request_id":"synthetic-request"}`)), Header: http.Header{}}, nil
	})
	downstream := providerFunc(func(_ context.Context, r acquisition.FetchRequest) ([]byte, error) {
		fetches++
		if r.Environment != "sandbox" || r.AccountID != "synthetic-account" || r.Credentials != request().Credentials {
			t.Fatal("changed filtered request")
		}
		return []byte("synthetic-page"), nil
	})
	p, err := New(downstream, wire, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		b, err := p.Fetch(context.Background(), request())
		if err != nil || string(b) != "synthetic-page" {
			t.Fatal("delegation failed")
		}
	}
	if calls != 1 || fetches != 2 || !p.Requested() {
		t.Fatal("wrong once-only result")
	}
}

func TestProbeFailureRedaction(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		network    bool
	}{
		{"server", `synthetic-private-upstream`, 500, false},
		{"redirect", ``, 302, false},
		{"false", `{"webhook_fired":false,"request_id":"synthetic"}`, 200, false},
		{"duplicate", `{"webhook_fired":true,"webhook_fired":false,"request_id":"synthetic"}`, 200, false},
		{"extra", `{"webhook_fired":true,"request_id":"synthetic","secret":"synthetic-private"}`, 200, false},
		{"missing", `{"webhook_fired":true}`, 200, false},
		{"null", `{"webhook_fired":true,"request_id":null}`, 200, false},
		{"trailing", `{"webhook_fired":true,"request_id":"synthetic"}{}`, 200, false},
		{"oversized", strings.Repeat("x", 4097), 200, false},
		{"network", "", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, fetches := 0, 0
			p, err := New(providerFunc(func(context.Context, acquisition.FetchRequest) ([]byte, error) { fetches++; return nil, nil }), transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.network {
					return nil, errors.New("synthetic-private-network")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{"Location": []string{"https://production.plaid.com/forbidden"}}}, nil
			}), time.Now().Add(time.Minute))
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				_, err = p.Fetch(context.Background(), request())
				if err != ErrProbe || strings.Contains(err.Error(), "private") {
					t.Fatal("failure not closed/redacted")
				}
			}
			if calls != 1 || fetches != 0 || p.Requested() {
				t.Fatal("failed attempt repeated or delegated")
			}
		})
	}
}

func TestProbeAtMostOnceAndExpiry(t *testing.T) {
	calls := 0
	wire := transportFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected") })
	provider := providerFunc(func(context.Context, acquisition.FetchRequest) ([]byte, error) {
		t.Fatal("unexpected delegation")
		return nil, nil
	})
	for _, bad := range []func(*acquisition.FetchRequest){
		func(r *acquisition.FetchRequest) { r.Environment = "production" },
		func(r *acquisition.FetchRequest) { r.Environment = "" },
		func(r *acquisition.FetchRequest) { r.AccountID = "" },
		func(r *acquisition.FetchRequest) { r.Credentials.Secret = "" },
		func(r *acquisition.FetchRequest) { r.Credentials.AccessToken = "synthetic\nprivate" },
	} {
		p, _ := New(provider, wire, time.Now().Add(time.Minute))
		r := request()
		bad(&r)
		if _, err := p.Fetch(context.Background(), r); err != ErrProbe {
			t.Fatal("bad request accepted")
		}
	}
	p, _ := New(provider, wire, time.Now().Add(time.Minute))
	p.now = func() time.Time { return time.Now().Add(time.Hour) }
	if _, err := p.Fetch(context.Background(), request()); err != ErrProbe {
		t.Fatal("expired accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, _ = New(provider, wire, time.Now().Add(time.Minute))
	if _, err := p.Fetch(ctx, request()); err != ErrProbe {
		t.Fatal("canceled accepted")
	}
	for _, deadline := range []time.Time{time.Time{}, time.Now().Add(-time.Second), time.Now().Add(11 * time.Minute)} {
		if _, err := New(provider, wire, deadline); err != ErrProbe {
			t.Fatal("unbounded constructor accepted")
		}
	}
	if calls != 0 {
		t.Fatal("invalid request reached network")
	}
}
