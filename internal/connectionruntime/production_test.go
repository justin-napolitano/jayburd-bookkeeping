package connectionruntime

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// productionConfig contains only fictional approved-reference metadata, never an actual credential.
func productionConfig() Config {
	c := config()
	c.Environment = "production"
	c.InfisicalEnvironment = "prod"
	c.RegistryBucket = c.ProjectID + "-jayburd-prod-credential-registry"
	c.ClientIDVersion = "projects/123456789012/secrets/jayburd-prod-bank-sync-client-id/versions/4"
	c.ClientSecretVersion = "projects/123456789012/secrets/jayburd-prod-bank-sync-client-secret/versions/9"
	c.BankKey = "projects/synthetic-project/locations/us-central1/keyRings/pilot/cryptoKeys/bank"
	c.LinkKey = "projects/synthetic-project/locations/us-central1/keyRings/pilot/cryptoKeys/link"
	return c
}

// TestProductionRuntimeStaticPins proves startup reads only the two exact reviewed static versions.
func TestProductionRuntimeStaticPins(t *testing.T) {
	c := productionConfig()
	calls := 0
	tok := &tokens{}
	s, e := New(context.Background(), c, inertDB{}, transport(func(r *http.Request) (*http.Response, error) {
		refs := []string{c.ClientIDVersion, c.ClientSecretVersion}
		if calls >= 2 || r.Method != "GET" || r.URL.String() != "https://secretmanager.googleapis.com/v1/"+refs[calls]+":access" {
			t.Fatal("unexpected dependency")
		}
		name := refs[calls]
		calls++
		return response(name, "synthetic-production-static"), nil
	}), tok)
	if e != nil || s == nil || calls != 2 || tok.identity != 0 {
		t.Fatal("Production startup rejected", e)
	}
}

// TestProductionRuntimeConfigurationDeny fences aliases, key/environment/reference substitution before I/O.
func TestProductionRuntimeConfigurationDeny(t *testing.T) {
	for _, mutate := range []func(*Config){func(c *Config) { c.Environment = "Production" }, func(c *Config) { c.InfisicalEnvironment = "dev" }, func(c *Config) {
		c.ClientSecretVersion = "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/9"
	}, func(c *Config) {
		c.ClientIDVersion = "projects/123456789012/secrets/jayburd-prod-bank-sync-client-id/versions/latest"
	}, func(c *Config) { c.BankKey = c.LinkKey }, func(c *Config) {
		c.BankKey = "projects/synthetic-project/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials"
	}, func(c *Config) {
		c.LinkKey = "projects/foreign-project/locations/us-central1/keyRings/pilot/cryptoKeys/link"
	}, func(c *Config) { c.RegistryBucket = c.ProjectID + "-jayburd-dev-credential-registry" }, func(c *Config) { c.HostedRepair = true }, func(c *Config) { c.WebhookURL = "https://attacker.invalid/webhooks/plaid" }} {
		c := productionConfig()
		mutate(&c)
		tok := &tokens{}
		calls := 0
		s, e := New(context.Background(), c, inertDB{}, transport(func(*http.Request) (*http.Response, error) { calls++; return nil, ErrUnavailable }), tok)
		if e != ErrConfiguration || s != nil || calls != 0 || tok.access != 0 {
			t.Fatal("unsafe profile reached dependency")
		}
	}
}

// TestProductionColdExpiredStartup retains SQL-only recovery with no dependency I/O.
func TestProductionColdExpiredStartup(t *testing.T) {
	c := productionConfig()
	deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c.ConsentExpiresAt = deadline.Format(time.RFC3339)
	tok := &tokens{}
	s, e := newWithClock(context.Background(), c, inertDB{}, transport(func(*http.Request) (*http.Response, error) {
		t.Fatal("expired startup reached dependency")
		return nil, ErrUnavailable
	}), tok, func() time.Time { return deadline })
	if e != nil || s == nil || tok.access != 0 {
		t.Fatal("expired SQL-only composition unavailable")
	}
}
