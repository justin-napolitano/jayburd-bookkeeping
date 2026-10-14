package connections

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip injects a deterministic HTTP response without a network.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// response builds an owned synthetic HTTP body.
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

// config declares explicit synthetic credentials and a fixed local authorization expiry.
func config() HTTPConfig {
	return HTTPConfig{ClientID: NewSecret("synthetic-client"), ClientSecret: NewSecret("synthetic-secret"), ConsentExpiresAt: time.Date(2098, 1, 1, 0, 0, 0, 0, time.UTC)}
}

// TestPlaidHTTPEnrollmentWire independently checks the fixed host, API version, payload and verified session.
func TestPlaidHTTPEnrollmentWire(t *testing.T) {
	paths := []string{}
	p, e := NewHTTPProvider(config(), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Scheme != "https" || r.URL.Host != "sandbox.plaid.com" || r.Method != "POST" || r.Header.Get("Plaid-Version") != "2020-09-14" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("unsafe wire")
		}
		var v map[string]any
		if json.NewDecoder(r.Body).Decode(&v) != nil || v["client_id"] != "synthetic-client" || v["secret"] != "synthetic-secret" {
			t.Fatal("missing injected credentials")
		}
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/link/token/create":
			if v["products"].([]any)[0] != "transactions" || v["user"].(map[string]any)["client_user_id"] != actorID {
				t.Fatal("wrong Link scope")
			}
			return response(200, `{"link_token":"synthetic-link","expiration":"2099-01-01T00:00:00Z"}`), nil
		case "/link/token/get":
			if v["link_token"] != "synthetic-link" {
				t.Fatal("wrong token")
			}
			return response(200, `{"link_token":"synthetic-link","link_sessions":[{"finished_at":"`+time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)+`","results":{"item_add_results":[{"public_token":"synthetic-public"}]}}]}`), nil
		case "/item/public_token/exchange":
			if v["public_token"] != "synthetic-public" {
				t.Fatal("wrong public token")
			}
			return response(200, `{"access_token":"synthetic-access","item_id":"synthetic-item"}`), nil
		case "/item/get":
			return response(200, `{"item":{"item_id":"synthetic-item","error":null,"consent_expiration_time":null}}`), nil
		case "/accounts/get":
			return response(200, `{"item":{"item_id":"synthetic-item"},"accounts":[{"account_id":"one","type":"depository","balances":{"iso_currency_code":"USD","unofficial_currency_code":null,"current":9007199254740993.01}}]}`), nil
		case "/item/access_token/invalidate":
			return response(200, `{"new_access_token":"synthetic-new-access"}`), nil
		case "/item/remove":
			return response(200, `{"request_id":"synthetic-request"}`), nil
		default:
			t.Fatal("unexpected endpoint")
			return nil, errors.New("unexpected")
		}
	}))
	if e != nil {
		t.Fatal(e)
	}
	link, e := p.CreateLink(context.Background(), LinkRequest{ActorID: actorID, Mode: "new"})
	if e != nil {
		t.Fatal(e)
	}
	if e = p.VerifySession(context.Background(), link.Token, NewSecret("synthetic-public"), ""); e != nil {
		t.Fatal(e)
	}
	credential, e := p.Exchange(context.Background(), NewSecret("synthetic-public"))
	if e != nil {
		t.Fatal(e)
	}
	m, e := p.Manifest(context.Background(), credential.AccessToken)
	if e != nil || m.ItemID != "synthetic-item" || m.ConsentExpiresAt != "2098-01-01T00:00:00Z" || len(m.Accounts) != 1 {
		t.Fatal("wrong manifest", e)
	}
	rotated, e := p.Rotate(context.Background(), credential.AccessToken)
	if e != nil || rotated.ItemID != "synthetic-item" || rotated.AccessToken.value != "synthetic-new-access" {
		t.Fatal("rotation", e)
	}
	if e = p.Remove(context.Background(), rotated.AccessToken); e != nil {
		t.Fatal(e)
	}
	if len(paths) != 8 {
		t.Fatal("unexpected retry", paths)
	}
}

// TestPlaidHTTPFailsClosed rejects ambiguity, malformed bodies and redirects without retries or diagnostics.
func TestPlaidHTTPFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"duplicate", `{"access_token":"one","access_token":"two","item_id":"x"}`, 200},
		{"trailing", `{"access_token":"one","item_id":"x"}{}`, 200},
		{"oversize", strings.Repeat(" ", 262145), 200},
		{"surrogate", `{"access_token":"\ud800","item_id":"x"}`, 200},
		{"linebreak", `{"access_token":"a\nb","item_id":"x"}`, 200},
		{"redirect", `{}`, 302},
		{"denied", `{"error_message":"private-provider-body"}`, 403},
		{"ambiguous", `{"access_token":"one","item_id":""}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			p, _ := NewHTTPProvider(config(), roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return response(tc.status, tc.body), nil }))
			_, e := p.Exchange(context.Background(), NewSecret("synthetic-public"))
			if e == nil || strings.Contains(e.Error(), "private") || calls != 1 {
				t.Fatal("invalid response accepted or retried")
			}
		})
	}
}

// TestPlaidHTTPSessionAssociation does not accept callback tokens without one exact completed provider result.
func TestPlaidHTTPSessionAssociation(t *testing.T) {
	for _, body := range []string{`{"link_token":"other","link_sessions":[]}`, `{"link_token":"synthetic-link","link_sessions":[]}`, `{"link_token":"synthetic-link","link_sessions":[{"results":{"item_add_results":[{"public_token":"synthetic-public"}]}}]}`} {
		p, _ := NewHTTPProvider(config(), roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, body), nil }))
		if p.VerifySession(context.Background(), NewSecret("synthetic-link"), NewSecret("synthetic-public"), "") == nil {
			t.Fatal("unverified association")
		}
	}
	s, r, _, _ := fixture()
	calls := 0
	p, _ := NewHTTPProvider(config(), roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not call") }))
	s.provider = p
	_, e := s.Begin(context.Background(), scope, "reconnect", BeginRequest{Mode: "reconnect"})
	code(t, e, "unsupported_connection_mode")
	if calls != 0 || len(r.calls) != 0 {
		t.Fatal("unsupported mode mutated")
	}
	r.c.Mode = "reconnect"
	_, e = s.Execute(context.Background(), scope, attemptID, ExecuteRequest{Nonce: nonceID})
	code(t, e, "unsupported_connection_mode")
	if r.claimed || calls != 0 {
		t.Fatal("unsupported mode claimed")
	}
}

// TestPlaidHTTPConsentBound distinguishes an explicit fixed local limit from an actual earlier provider expiry.
func TestPlaidHTTPConsentBound(t *testing.T) {
	cfg := config()
	cfg.ConsentExpiresAt = time.Now().Add(-time.Minute)
	if _, e := NewHTTPProvider(cfg, roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, nil })); e == nil {
		t.Fatal("expired policy")
	}
	p, _ := NewHTTPProvider(config(), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/item/get" {
			return response(200, `{"item":{"item_id":"item","error":null,"consent_expiration_time":"2090-01-01T00:00:00Z"}}`), nil
		}
		return response(200, `{"item":{"item_id":"item"},"accounts":[{"account_id":"one","type":"depository","balances":{"iso_currency_code":"JPY","unofficial_currency_code":null}}]}`), nil
	}))
	m, e := p.Manifest(context.Background(), NewSecret("access"))
	if e != nil || m.ConsentExpiresAt != "2090-01-01T00:00:00Z" {
		t.Fatal("provider expiry ignored", e)
	}
	p.config.ConsentExpiresAt = time.Now().Add(-time.Minute)
	if _, e = p.Manifest(context.Background(), NewSecret("access")); e == nil {
		t.Fatal("expired config extended")
	}
}

// TestPlaidHTTPCancellation uses a transport that honors cancellation and returns no raw error.
func TestPlaidHTTPCancellation(t *testing.T) {
	p, _ := NewHTTPProvider(config(), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, errors.New("private cancellation")
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := p.Exchange(ctx, NewSecret("synthetic-public"))
	if e == nil || strings.Contains(e.Error(), "private") {
		t.Fatal("cancellation escaped")
	}
}
