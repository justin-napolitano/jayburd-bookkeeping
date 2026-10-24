package connectionruntime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// inertDB detects accidental startup transactions; request-time authority remains in the actual connection service.
type inertDB struct{}

// BeginTx is never permitted while composing adapters.
func (inertDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	panic("startup SQL is not permitted")
}

// tokens counts managed authentication without credentials or external calls.
type tokens struct{ access, identity int }

// AccessToken supplies only a synthetic opaque fixture value.
func (t *tokens) AccessToken(context.Context) (string, error) {
	t.access++
	return "synthetic-google-token", nil
}

// IdentityToken must not be used by constructor-only static version reads.
func (t *tokens) IdentityToken(context.Context, string) (string, error) {
	t.identity++
	return "synthetic-identity", nil
}

// transport supplies injected HTTP responses without any network.
type transport func(*http.Request) (*http.Response, error)

// RoundTrip delegates to the explicit synthetic transport.
func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// config constructs explicit fictional identifiers and a finite local policy expiry.
func config() Config {
	return Config{Enabled: true, LinkEventsQualified: true, ProjectID: "synthetic-project", ProjectNumber: "123456789012", InfisicalOrigin: "https://us.infisical.com", InfisicalProject: "00000000-0000-4000-8000-000000000001", InfisicalIdentity: "00000000-0000-4000-8000-000000000002", InfisicalEnvironment: "dev", RegistryBucket: "synthetic-project-jayburd-dev-credential-registry", RegistryPrefix: "provider-credentials-v1", ClientIDVersion: "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", ClientSecretVersion: "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", ConsentExpiresAt: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)}
}

// response encodes a canonical version and independently computed CRC32C for a fictional static secret.
func response(name, value string) *http.Response {
	data := []byte(value)
	body, _ := json.Marshal(map[string]any{"name": name, "payload": map[string]any{"data": base64.StdEncoding.EncodeToString(data), "dataCrc32c": fmt.Sprint(crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)))}})
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body)))}
}

// TestDisabledIsInert proves no dependencies are needed or queried in the default API mode.
func TestDisabledIsInert(t *testing.T) {
	service, e := New(nil, Config{}, nil, nil, nil)
	if e != nil || service != nil {
		t.Fatal("disabled runtime not inert")
	}
	if e = (Config{}).ManagedScope("health", "url", ""); e != nil {
		t.Fatal(e)
	}
}

// TestExpiredEnrollmentStartup proves exact-deadline and cold-restart composition needs no dependency I/O.
func TestExpiredEnrollmentStartup(t *testing.T) {
	deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, now := range []time.Time{deadline, deadline.Add(time.Nanosecond), deadline.Add(365 * 24 * time.Hour)} {
		c := config()
		c.ConsentExpiresAt = deadline.Format(time.RFC3339)
		token := &tokens{}
		calls := 0
		s, err := newWithClock(context.Background(), c, inertDB{}, transport(func(*http.Request) (*http.Response, error) {
			calls++
			t.Fatal("expired startup reached dependency transport")
			return nil, ErrUnavailable
		}), token, func() time.Time { return now })
		if err != nil || s == nil || calls != 0 || token.access != 0 || token.identity != 0 {
			t.Fatalf("expired cold startup unavailable: %v", err)
		}
		_, err = s.Begin(context.Background(), connections.Scope{Actor: "00000000-0000-4000-8000-000000000001", Entity: "00000000-0000-4000-8000-000000000002", Book: "00000000-0000-4000-8000-000000000003"}, "same-key", connections.BeginRequest{Mode: "new"})
		if err == nil || err.Error() != "temporarily_unavailable" {
			t.Fatal("expired enrollment not denied")
		}
	}
	c := config()
	c.ConsentExpiresAt = "2020-01-01T00:00:00Z"
	if c.ManagedScope("managed", "cloudsql", c.ProjectID) != nil {
		t.Fatal("expired optional policy blocked managed startup")
	}
	c.ClientSecretVersion = "projects/999999999999/secrets/foreign/versions/1"
	if c.ManagedScope("managed", "cloudsql", c.ProjectID) != ErrConfiguration {
		t.Fatal("expired policy hid unsafe configuration")
	}
}

// TestEnrollmentExpiresDuringStartup avoids turning a finite secret initialization race into API downtime.
func TestEnrollmentExpiresDuringStartup(t *testing.T) {
	for _, expireAfter := range []int{1, 2} {
		deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		now := deadline.Add(-time.Second)
		c := config()
		c.ConsentExpiresAt = deadline.Format(time.RFC3339)
		calls := 0
		token := &tokens{}
		s, err := newWithClock(context.Background(), c, inertDB{}, transport(func(*http.Request) (*http.Response, error) {
			refs := []string{c.ClientIDVersion, c.ClientSecretVersion}
			name := refs[calls]
			calls++
			if calls == expireAfter {
				now = deadline
			}
			return response(name, "synthetic-static-value"), nil
		}), token, func() time.Time { return now })
		if err != nil || s == nil || calls != expireAfter || token.access != expireAfter {
			t.Fatalf("startup expiry was fatal or loaded another secret: %v", err)
		}
	}
}

// TestEnrollmentExpiresAfterStartup keeps the already constructed service closed without a restart.
func TestEnrollmentExpiresAfterStartup(t *testing.T) {
	deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	now := deadline.Add(-time.Second)
	c := config()
	c.ConsentExpiresAt = deadline.Format(time.RFC3339)
	token := &tokens{}
	calls := 0
	s, err := newWithClock(context.Background(), c, inertDB{}, transport(func(*http.Request) (*http.Response, error) {
		refs := []string{c.ClientIDVersion, c.ClientSecretVersion}
		if calls >= len(refs) {
			t.Fatal("unexpected provider or credential I/O")
		}
		name := refs[calls]
		calls++
		return response(name, "synthetic-static-value"), nil
	}), token, func() time.Time { return now })
	if err != nil || s == nil || calls != 2 {
		t.Fatal("approved runtime did not compose")
	}
	for _, instant := range []time.Time{deadline, deadline.Add(time.Nanosecond)} {
		now = instant
		_, err = s.Begin(context.Background(), connections.Scope{Actor: "00000000-0000-4000-8000-000000000001", Entity: "00000000-0000-4000-8000-000000000002", Book: "00000000-0000-4000-8000-000000000003"}, "same-key", connections.BeginRequest{Mode: "new"})
		if err == nil || err.Error() != "temporarily_unavailable" || calls != 2 || token.access != 2 {
			t.Fatal("running service crossed the deadline")
		}
	}
}

// TestApprovalTransportDeadline stops new dependency requests even inside multi-call provider work.
func TestApprovalTransportDeadline(t *testing.T) {
	deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	now := deadline.Add(-time.Nanosecond)
	calls := 0
	tr := approvalTransport{base: transport(func(*http.Request) (*http.Response, error) { calls++; return &http.Response{StatusCode: 200}, nil }), deadline: deadline, now: func() time.Time { return now }}
	req, _ := http.NewRequest(http.MethodPost, "https://sandbox.plaid.com/item/get", nil)
	if _, err := tr.RoundTrip(req); err != nil || calls != 1 {
		t.Fatal("approved transport unavailable")
	}
	for _, expired := range []time.Time{deadline, deadline.Add(time.Nanosecond)} {
		now = expired
		if resp, err := tr.RoundTrip(req); err != ErrUnavailable || resp != nil || calls != 1 {
			t.Fatal("expired request escaped transport fence")
		}
	}
}

// TestExplicitCompleteConfiguration rejects unsupported scope and policy before reading a secret.
func TestExplicitCompleteConfiguration(t *testing.T) {
	cases := []func(*Config){
		func(c *Config) { c.ProjectID = "synthetic-project-that-is-too-long-for-gcp" },
		func(c *Config) { c.RegistryBucket = "other-project-jayburd-dev-credential-registry" },
		func(c *Config) { c.RegistryPrefix = "other-prefix" },
		func(c *Config) { c.ClientIDVersion, c.ClientSecretVersion = c.ClientSecretVersion, c.ClientIDVersion },

		func(c *Config) { c.Enabled = false }, func(c *Config) { c.ProjectNumber = "synthetic-project" }, func(c *Config) { c.InfisicalEnvironment = "prod" }, func(c *Config) { c.InfisicalOrigin = "https://arbitrary.invalid" }, func(c *Config) { c.InfisicalIdentity = "00000000-0000-0000-0000-000000000000" }, func(c *Config) { c.RegistryPrefix = "../outside" }, func(c *Config) { c.ClientIDVersion = "projects/123456789012/secrets/a/versions/latest" }, func(c *Config) { c.ClientIDVersion = "projects/999999999999/secrets/a/versions/1" }, func(c *Config) { c.ClientIDVersion = c.ClientSecretVersion }, func(c *Config) { c.ConsentExpiresAt = "invalid" }, func(c *Config) { c.ConsentExpiresAt = time.Now().Add(31 * 24 * time.Hour).Format(time.RFC3339) },
	}
	for i, mutate := range cases {
		c := config()
		mutate(&c)
		token := &tokens{}
		calls := 0
		s, e := New(context.Background(), c, inertDB{}, transport(func(*http.Request) (*http.Response, error) { calls++; return nil, fmt.Errorf("hidden") }), token)
		if e != ErrConfiguration || s != nil || calls != 0 || token.access != 0 || token.identity != 0 {
			t.Fatalf("configuration case%d reached dependencies", i)
		}
	}
	c := config()
	if c.ManagedScope("managed", "url", c.ProjectID) == nil || c.ManagedScope("health", "cloudsql", c.ProjectID) == nil || c.ManagedScope("managed", "cloudsql", "other-project") == nil {
		t.Fatal("managed scope escaped")
	}
}

// TestEnabledReadsOnlyExactStaticVersions proves no Link, write, identity login or SQL side effect occurs at startup.
func TestEnabledReadsOnlyExactStaticVersions(t *testing.T) {
	c := config()
	token := &tokens{}
	calls := 0
	s, e := New(context.Background(), c, inertDB{}, transport(func(r *http.Request) (*http.Response, error) {
		refs := []string{c.ClientIDVersion, c.ClientSecretVersion}
		if calls >= 2 || r.Method != "GET" || r.URL.String() != "https://secretmanager.googleapis.com/v1/"+refs[calls]+":access" {
			t.Fatal("unexpected startup request")
		}
		if r.Header.Get("Authorization") != "Bearer synthetic-google-token" {
			t.Fatal("missing explicit managed identity")
		}
		name := refs[calls]
		calls++
		return response(name, "synthetic-static-value"), nil
	}), token)
	if e != nil || s == nil || calls != 2 || token.access != 2 || token.identity != 0 {
		t.Fatalf("unexpected composition result %v", e)
	}
}

// TestStaticReadFailuresAreRedacted never exposes a dependency error or returns a partially configured service.
func TestStaticReadFailuresAreRedacted(t *testing.T) {
	for _, which := range []string{"transport", "wrong_version", "invalid_secret", "cancelled"} {
		t.Run(which, func(t *testing.T) {
			c := config()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if which == "cancelled" {
				cancel()
			}
			calls := 0
			s, e := New(ctx, c, inertDB{}, transport(func(r *http.Request) (*http.Response, error) {
				calls++
				if which == "transport" {
					return nil, fmt.Errorf("private-secret")
				}
				name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/"), ":access")
				if which == "wrong_version" {
					name = c.ClientIDVersion + "0"
				}
				v := "synthetic-value"
				if which == "invalid_secret" {
					v = "hidden\nsecret"
				}
				return response(name, v), nil
			}), &tokens{})
			if e != ErrUnavailable || s != nil || strings.Contains(e.Error(), "private") {
				t.Fatal("failure not fixed")
			}
			if which == "cancelled" && calls != 0 {
				t.Fatal("cancelled startup reached storage")
			}
		})
	}
}

// TestHostedLinkDoesNotRequireSDKEnablement retains all other explicit configuration gates.
func TestHostedLinkDoesNotRequireSDKEnablement(t *testing.T) {
	c := config()
	c.LinkEventsQualified = false
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
}

// webhookConfig uses fixed public deployment identifiers with synthetic credentials and no network.
func webhookConfig() Config {
	c := config()
	c.ProjectID = "jayburd-bookkeeping-dev"
	c.ProjectNumber = "9088794002"
	c.RegistryBucket = c.ProjectID + "-jayburd-dev-credential-registry"
	c.ClientIDVersion = "projects/9088794002/secrets/jayburd-dev-bank-sync-client-id/versions/1"
	c.ClientSecretVersion = "projects/9088794002/secrets/jayburd-dev-bank-sync-client-secret/versions/2"
	c.WebhookURL = connections.SandboxTransactionsWebhookURL
	return c
}

// TestWebhookConfigurationScope rejects partial, foreign-project and substituted receivers before dependencies.
func TestWebhookConfigurationScope(t *testing.T) {
	for _, mutate := range []func(*Config){
		func(c *Config) { c.WebhookURL = "https://attacker.example/webhooks/plaid" },
		func(c *Config) { c.WebhookURL += "?private=value" },
		func(c *Config) {
			c.ProjectID = "synthetic-project"
			c.RegistryBucket = c.ProjectID + "-jayburd-dev-credential-registry"
		},
		func(c *Config) {
			c.ProjectNumber = "123456789012"
			c.ClientIDVersion = strings.Replace(c.ClientIDVersion, "9088794002", c.ProjectNumber, 1)
			c.ClientSecretVersion = strings.Replace(c.ClientSecretVersion, "9088794002", c.ProjectNumber, 1)
		},
		func(c *Config) { *c = Config{WebhookURL: connections.SandboxTransactionsWebhookURL} },
	} {
		c := webhookConfig()
		mutate(&c)
		token := &tokens{}
		calls := 0
		service, err := New(context.Background(), c, inertDB{}, transport(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, fmt.Errorf("private-diagnostic")
		}), token)
		if err != ErrConfiguration || service != nil || calls != 0 || token.access != 0 || token.identity != 0 {
			t.Fatal("invalid webhook scope reached managed dependencies")
		}
	}
}

// TestWebhookOptInStartup remains limited to existing static credential reads, never enrollment or SQL.
func TestWebhookOptInStartup(t *testing.T) {
	c := webhookConfig()
	token := &tokens{}
	calls := 0
	service, err := New(context.Background(), c, inertDB{}, transport(func(r *http.Request) (*http.Response, error) {
		refs := []string{c.ClientIDVersion, c.ClientSecretVersion}
		if calls >= len(refs) || r.Method != http.MethodGet || r.URL.String() != "https://secretmanager.googleapis.com/v1/"+refs[calls]+":access" {
			t.Fatal("webhook opt-in performed unexpected startup operation")
		}
		name := refs[calls]
		calls++
		return response(name, "synthetic-static-value"), nil
	}), token)
	if err != nil || service == nil || calls != 2 || token.access != 2 || token.identity != 0 {
		t.Fatal("valid opt-in composition rejected or performed extra operations", err)
	}
}
