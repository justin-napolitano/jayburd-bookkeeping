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

// TestExplicitCompleteConfiguration rejects unsupported scope and policy before reading a secret.
func TestExplicitCompleteConfiguration(t *testing.T) {
	cases := []func(*Config){
		func(c *Config) { c.ProjectID = "synthetic-project-that-is-too-long-for-gcp" },
		func(c *Config) { c.RegistryBucket = "other-project-jayburd-dev-credential-registry" },
		func(c *Config) { c.RegistryPrefix = "other-prefix" },
		func(c *Config) { c.ClientIDVersion, c.ClientSecretVersion = c.ClientSecretVersion, c.ClientIDVersion },

		func(c *Config) { c.Enabled = false }, func(c *Config) { c.LinkEventsQualified = false }, func(c *Config) { c.ProjectNumber = "synthetic-project" }, func(c *Config) { c.InfisicalEnvironment = "prod" }, func(c *Config) { c.InfisicalOrigin = "https://arbitrary.invalid" }, func(c *Config) { c.InfisicalIdentity = "00000000-0000-0000-0000-000000000000" }, func(c *Config) { c.RegistryPrefix = "../outside" }, func(c *Config) { c.ClientIDVersion = "projects/123456789012/secrets/a/versions/latest" }, func(c *Config) { c.ClientIDVersion = "projects/999999999999/secrets/a/versions/1" }, func(c *Config) { c.ClientIDVersion = c.ClientSecretVersion }, func(c *Config) { c.ConsentExpiresAt = time.Now().Add(-time.Second).Format(time.RFC3339) }, func(c *Config) { c.ConsentExpiresAt = time.Now().Add(31 * 24 * time.Hour).Format(time.RFC3339) },
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
