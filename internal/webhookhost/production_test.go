package webhookhost

import (
	"context"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// productionConfigFixture contains fictional numeric static pins and a finite deployment capability.
func productionConfigFixture() Config {
	c := configFixture()
	c.Environment = "production"
	c.ApprovalExpiresAt = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	c.ClientIDVersion = strings.Replace(c.ClientIDVersion, "jayburd-dev", "jayburd-prod", 1)
	c.ClientSecretVersion = strings.Replace(c.ClientSecretVersion, "jayburd-dev", "jayburd-prod", 1)
	return c
}

// TestProductionReceiverConfiguration fences mixed references/identity and cold expiry before factories.
func TestProductionReceiverConfiguration(t *testing.T) {
	if productionConfigFixture().Validate() != nil {
		t.Fatal("valid explicit profile rejected")
	}
	f := factories{keys: func(Config) (webhook.KeyProvider, func(), error) {
		t.Fatal("rejected config reached keys")
		return nil, nil, nil
	}, pool: func(context.Context, clouddb.Config) (webhook.Beginner, func(), error) {
		t.Fatal("rejected config reached SQL")
		return nil, nil, nil
	}}
	for _, mutate := range []func(*Config){func(c *Config) { c.Environment = "Production" }, func(c *Config) { c.Instance = "jayburd-bookkeeping-dev:us-central1:jayburd-dev" }, func(c *Config) {
		c.ClientIDVersion = strings.Replace(c.ClientIDVersion, "jayburd-prod", "jayburd-dev", 1)
	}, func(c *Config) {
		c.ClientSecretVersion = strings.Replace(c.ClientSecretVersion, "/versions/2", "/versions/latest", 1)
	}, func(c *Config) { c.ApprovalExpiresAt = "" }} {
		c := productionConfigFixture()
		mutate(&c)
		if _, e := managed(context.Background(), c, f); e != ErrConfiguration {
			t.Fatal("mixed receiver accepted")
		}
	}
	c := productionConfigFixture()
	c.ApprovalExpiresAt = "2020-01-01T00:00:00Z"
	if _, e := managed(context.Background(), c, f); e != ErrServe {
		t.Fatal("expired receiver constructed dependencies")
	}
}

// TestProductionReceiverWarmExpiry denies even cached-key traffic before body verification or business SQL.
func TestProductionReceiverWarmExpiry(t *testing.T) {
	keys, token, body := signedFixture(t)
	received := 0
	h, e := New(keys, receiverFunc(func(context.Context, webhook.Verified) (webhook.Receipt, error) {
		received++
		return webhook.Receipt{}, nil
	}), nil)
	if e != nil {
		t.Fatal(e)
	}
	h.approvalDeadline = time.Now().Add(-time.Second)
	request := httptest.NewRequest("POST", "/webhooks/plaid", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Plaid-Verification", token)
	w := httptest.NewRecorder()
	h.server(context.Background()).Handler.ServeHTTP(w, request)
	if w.Code != 503 || received != 0 || keys.calls.Load() != 0 {
		t.Fatal("expired host reached verification/receipt")
	}
}
