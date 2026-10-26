package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

// productionProfile is an explicit closed test capability, never ambient provider discovery.
func productionProfile() plaidprofile.Profile { p, _ := plaidprofile.New("production"); return p }

// TestProductionWebhookEnvironmentDeny proves exact raw signatures and runtime environment precede receipt effects.
func TestProductionWebhookEnvironmentDeny(t *testing.T) {
	private, key, now := fixture(t)
	v, e := NewVerifierForProfile(keyFunc(func(context.Context, string) (JWK, error) { return key, nil }), productionProfile())
	if e != nil {
		t.Fatal(e)
	}
	v.now = func() time.Time { return now }
	calls := 0
	h, _ := NewHandler(v, receiverFunc(func(_ context.Context, v Verified) (Receipt, error) {
		calls++
		if v.Environment != "production" {
			t.Fatal("wrong projection")
		}
		return Receipt{State: "accepted", ReceiptID: "00000000-0000-4000-8000-000000000001", StreamCount: 1}, nil
	}))
	body := []byte(strings.Replace(string(bodyFixture()), `"sandbox"`, `"production"`, 1))
	token := signBody(t, private, key.ID, body, now.Unix())
	if got := invokeHandler(h, token, body).Code; got != 204 || calls != 1 {
		t.Fatal("valid Production denied", got)
	}
	for _, c := range []struct {
		name          string
		body          []byte
		iat           int64
		signerForeign bool
	}{{"sandbox", bodyFixture(), now.Unix(), false}, {"tampered", append(append([]byte{}, body...), byte(' ')), now.Unix(), false}, {"old", body, now.Unix() - 301, false}, {"future", body, now.Unix() + 61, false}, {"foreign-key", body, now.Unix(), true}} {
		t.Run(c.name, func(t *testing.T) {
			signed := token
			if c.name != "tampered" {
				signer := private
				if c.signerForeign {
					signer, _, _ = fixture(t)
				}
				signed = signBody(t, signer, key.ID, c.body, c.iat)
			}
			before := calls
			if got := invokeHandler(h, signed, c.body).Code; got == 204 || calls != before {
				t.Fatal("invalid profile reached receipt")
			}
		})
	}
}

// TestProductionKeyTransportAndCache keeps equal key IDs isolated by constructor and verifier ownership.
func TestProductionKeyTransportAndCache(t *testing.T) {
	private, key, now := fixture(t)
	lookups := 0
	secrets := 0
	response, _ := json.Marshal(map[string]any{"key": key, "request_id": "synthetic-production-request"})
	keys, e := NewHTTPKeysForProfile(transportFunc(func(r *http.Request) (*http.Response, error) {
		lookups++
		if r.URL.String() != "https://production.plaid.com/webhook_verification_key/get" || r.Method != "POST" || r.Header.Get("Plaid-Version") != "2020-09-14" {
			t.Fatal("wrong key destination")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(response)))}, nil
	}), credentialsFunc(func(context.Context) (Credentials, error) {
		secrets++
		return Credentials{"synthetic-production-client", "synthetic-production-secret"}, nil
	}), productionProfile())
	if e != nil {
		t.Fatal(e)
	}
	if _, e := NewVerifier(keys); e == nil {
		t.Fatal("Production key capability entered Sandbox verifier")
	}
	sandbox, _ := NewHTTPKeys(transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("foreign key lookup"); return nil, errUnavailable }), credentialsFunc(func(context.Context) (Credentials, error) {
		t.Fatal("foreign secret lookup")
		return Credentials{}, errUnavailable
	}))
	if _, e := NewVerifierForProfile(sandbox, productionProfile()); e == nil {
		t.Fatal("Sandbox keys entered Production verifier")
	}
	v, _ := NewVerifierForProfile(keys, productionProfile())
	v.now = func() time.Time { return now }
	body := []byte(strings.Replace(string(bodyFixture()), `"sandbox"`, `"production"`, 1))
	for i := 0; i < 2; i++ {
		if _, e := v.Verify(context.Background(), signBody(t, private, key.ID, body, now.Unix()), body); e != nil {
			t.Fatal(e)
		}
	}
	if lookups != 1 || secrets != 1 {
		t.Fatal("profile cache not bounded")
	}
	other, _ := NewVerifierForProfile(keys, productionProfile())
	other.now = v.now
	if _, e := other.Verify(context.Background(), signBody(t, private, key.ID, body, now.Unix()), body); e != nil || lookups != 2 {
		t.Fatal("cache shared across verifier instances")
	}
}

// TestProductionSessionProjection authenticates modern session origin without exposing raw Link material.
func TestProductionSessionProjection(t *testing.T) {
	private, key, now := fixture(t)
	v, _ := NewVerifierForProfile(keyFunc(func(context.Context, string) (JWK, error) { return key, nil }), productionProfile())
	v.now = func() time.Time { return now }
	body := []byte(`{"webhook_type":"LINK","webhook_code":"SESSION_FINISHED","environment":"production","status":"success","link_token":"synthetic-production-link","link_session_id":"synthetic-production-session","public_tokens":[]}`)
	got, e := v.Verify(context.Background(), signBody(t, private, key.ID, body, now.Unix()), body)
	if e != nil || got.Environment != "production" || got.ItemID != "" || got.session == nil || got.session.linkHash != tokenHash("synthetic-production-link") {
		t.Fatal("wrong private session projection")
	}
	if _, e := parseLinkSession(body); e == nil {
		t.Fatal("legacy parser accepted Production")
	}
	if _, e := NewHTTPKeysForProfile(nil, nil, plaidprofile.Profile{}); e == nil {
		t.Fatal("zero key profile accepted")
	}
}
