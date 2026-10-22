package webhookhost

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
)

// configFixture contains explicitly synthetic deployment identifiers, never actual resource references.
func configFixture() Config {
	return Config{Listen: "127.0.0.1:8080", Instance: "synthetic-project:us-central1:synthetic-db", ProjectNumber: "123456", ClientIDVersion: "projects/123456/secrets/jayburd-dev-bank-sync-client-id/versions/1", ClientSecretVersion: "projects/123456/secrets/jayburd-dev-bank-sync-client-secret/versions/2"}
}

// inertPool rejects any accidental SQL during constructor-only tests.
type inertPool struct{ t *testing.T }

func (p inertPool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.t.Error("constructor performed SQL authority/business operation")
	return nil, errors.New("synthetic-private-sql")
}

// TestManagedInvalidConfigurationHasNoIO rejects destinations, versions and ambiguous binds before factories.
func TestManagedInvalidConfigurationHasNoIO(t *testing.T) {
	mutations := []func(*Config){
		func(c *Config) { c.Instance = "synthetic-project:europe-west1:synthetic-db" },
		func(c *Config) { c.Instance = "postgres://synthetic-db" },
		func(c *Config) { c.ProjectNumber = "synthetic-project" },
		func(c *Config) { c.ProjectNumber = "0123456" },
		func(c *Config) { c.ProjectNumber = "0" },
		func(c *Config) { c.ProjectNumber = "9223372036854775808" },
		func(c *Config) { c.Listen = ":8080" },
		func(c *Config) { c.Listen = "localhost:8080" },
		func(c *Config) { c.Listen = "0.0.0.0:0" },
		func(c *Config) { c.Listen = "127.0.0.1:08080" },
		func(c *Config) { c.Listen = "127.0.0.1:65536" },
		func(c *Config) { c.ClientIDVersion = strings.Replace(c.ClientIDVersion, "/1", "/latest", 1) },
		func(c *Config) { c.ClientIDVersion = strings.Replace(c.ClientIDVersion, "/1", "/01", 1) },
		func(c *Config) { c.ClientIDVersion = strings.Replace(c.ClientIDVersion, "123456", "654321", 1) },
		func(c *Config) {
			c.ClientIDVersion = strings.Replace(c.ClientIDVersion, "client-id", "access-token", 1)
		},
		func(c *Config) { c.ClientSecretVersion = c.ClientIDVersion },
		func(c *Config) { c.ClientSecretVersion += "?version=1" },
	}
	f := factories{keys: func(Config) (webhook.KeyProvider, func(), error) {
		t.Fatal("invalid configuration reached keys factory")
		return nil, nil, nil
	}, pool: func(context.Context, clouddb.Config) (webhook.Beginner, func(), error) {
		t.Fatal("invalid configuration reached pool factory")
		return nil, nil, nil
	}}
	for i, mutate := range mutations {
		c := configFixture()
		mutate(&c)
		if _, err := managed(context.Background(), c, f); err != ErrConfiguration {
			t.Fatalf("invalid configuration accepted: %d", i)
		}
	}
	if _, err := managed(nil, configFixture(), f); err != ErrConfiguration {
		t.Fatal("nil context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := managed(ctx, configFixture(), f); err != ErrServe {
		t.Fatal("cancelled initialization started")
	}
}

// TestManagedOwnershipAndDedicatedPool proves bounded fixed capability composition and reverse cleanup once.
func TestManagedOwnershipAndDedicatedPool(t *testing.T) {
	keys, _, _ := signedFixture(t)
	order := []string{}
	r, err := managed(context.Background(), configFixture(), factories{
		keys: func(c Config) (webhook.KeyProvider, func(), error) {
			if c != configFixture() {
				t.Fatal("configuration changed")
			}
			return keys, func() { order = append(order, "keys") }, nil
		},
		pool: func(ctx context.Context, c clouddb.Config) (webhook.Beginner, func(), error) {
			if c.Component != "webhook" || c.Instance != configFixture().Instance {
				t.Fatal("borrowed database authority")
			}
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 20*time.Second {
				t.Fatal("startup unbounded")
			}
			return inertPool{t}, func() { order = append(order, "pool") }, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if keys.calls.Load() != 0 || len(order) != 0 {
		t.Fatal("constructor resolved key or closed live dependency")
	}
	r.Close()
	r.Close()
	if !reflect.DeepEqual(order, []string{"pool", "keys"}) {
		t.Fatal("cleanup repeated or reordered")
	}
	if err := r.Run(context.Background()); err != ErrAlreadyServed {
		t.Fatal("closed runtime reused dependencies")
	}
}

// TestManagedFailureCleanup covers partial construction, unavailable pools and cancellation between phases.
func TestManagedFailureCleanup(t *testing.T) {
	for _, failure := range []string{"keys", "nilkeys", "pool", "nilpool", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			keys, _, _ := signedFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			closedKeys, closedPool := 0, 0
			_, err := managed(ctx, configFixture(), factories{
				keys: func(Config) (webhook.KeyProvider, func(), error) {
					close := func() { closedKeys++ }
					if failure == "keys" {
						return keys, close, errors.New("synthetic-private-key")
					}
					if failure == "nilkeys" {
						return nil, close, nil
					}
					if failure == "cancel" {
						cancel()
					}
					return keys, close, nil
				},
				pool: func(context.Context, clouddb.Config) (webhook.Beginner, func(), error) {
					close := func() { closedPool++ }
					if failure == "pool" {
						return inertPool{t}, close, errors.New("synthetic-private-db")
					}
					if failure == "nilpool" {
						return nil, close, nil
					}
					t.Fatal("unexpected pool initialization")
					return nil, nil, nil
				},
			})
			if err == nil || strings.Contains(err.Error(), "synthetic") {
				t.Fatal("failure accepted or diagnostic exposed")
			}
			expectedPool := 0
			if failure == "pool" || failure == "nilpool" {
				expectedPool = 1
			}
			if closedKeys != 1 || closedPool != expectedPool {
				t.Fatal("partial cleanup incorrect")
			}
		})
	}
}

// readerFunc supplies synthetic secret results without a network transport or real secret access.
type readerFunc func(context.Context, string) ([]byte, error)

func (f readerFunc) Read(ctx context.Context, name string) ([]byte, error) { return f(ctx, name) }

// TestPinnedCredentialResolution proves exact resource selection, lazy construction and redacted faults.
func TestPinnedCredentialResolution(t *testing.T) {
	c := configFixture()
	seen := []string{}
	source := clientCredentials{id: c.ClientIDVersion, secret: c.ClientSecretVersion, reader: readerFunc(func(_ context.Context, name string) ([]byte, error) {
		seen = append(seen, name)
		return []byte("synthetic-only"), nil
	})}
	if len(seen) != 0 {
		t.Fatal("constructor resolved secret")
	}
	value, err := source.Resolve(context.Background())
	if err != nil || value.ClientID != "synthetic-only" || value.Secret != "synthetic-only" || !reflect.DeepEqual(seen, []string{c.ClientIDVersion, c.ClientSecretVersion}) {
		t.Fatal("pinned resources changed")
	}
	for _, failAt := range []int{1, 2} {
		calls := 0
		source.reader = readerFunc(func(context.Context, string) ([]byte, error) {
			calls++
			if calls == failAt {
				return nil, errors.New("synthetic-private-secret-diagnostic")
			}
			return []byte("synthetic-only"), nil
		})
		value, err := source.Resolve(context.Background())
		if err != ErrServe || value != (webhook.Credentials{}) || calls != failAt {
			t.Fatal("credential fault leaked partial values or retried")
		}
	}
	keys, close, err := managedKeys(c)
	if err != nil || keys == nil || close == nil {
		t.Fatal("inert concrete keys constructor failed")
	}
	close()
}
