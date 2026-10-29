package clouddb

import (
	"context"
	"net"
	"testing"
)

// TestSharedRuntimeRouting proves distinct database/login derivation without network access.
func TestSharedRuntimeRouting(t *testing.T) {
	dial := func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("constructor opened network")
		return nil, ErrConfiguration
	}
	for _, component := range []string{"api", "sync", "publisher", "webhook"} {
		c := Config{"jayburd-bookkeeping-dev:us-central1:jayburd-dev", component}
		prod, err := PoolConfigForEnvironment(c, "production", dial)
		if err != nil || prod.ConnConfig.Database != "bookkeeping_prod" || prod.ConnConfig.User != "jayburd-prod-"+component+"@jayburd-bookkeeping-dev.iam" {
			t.Fatalf("Production target mismatch for %s", component)
		}
		sandbox, err := PoolConfigForEnvironment(c, "sandbox", dial)
		if err != nil || sandbox.ConnConfig.Database != "bookkeeping" || sandbox.ConnConfig.User != "jayburd-dev-"+component+"@jayburd-bookkeeping-dev.iam" {
			t.Fatalf("Sandbox target mismatch for %s", component)
		}
	}
	for _, component := range []string{"migrator", "verifier"} {
		c := Config{"jayburd-bookkeeping-dev:us-central1:jayburd-dev", component}
		if _, err := PoolConfigForEnvironment(c, "production", dial); err == nil {
			t.Fatalf("implicit privileged route accepted for %s", component)
		}
	}
}
