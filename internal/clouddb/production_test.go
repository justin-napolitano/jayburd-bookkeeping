package clouddb

import (
	"context"
	"net"
	"testing"
)

// TestProductionIAMProfile checks fixed dedicated identities and mixed-target denial without opening a socket.
func TestProductionIAMProfile(t *testing.T) {
	for _, component := range []string{"api", "sync", "publisher"} {
		c := Config{"fictional-production:us-central1:pilot-db", component}
		p, e := PoolConfigForEnvironment(c, "production", func(context.Context, string, string) (net.Conn, error) {
			t.Fatal("constructor opened network")
			return nil, ErrConfiguration
		})
		if e != nil || p.ConnConfig.User != "jayburd-prod-"+component+"@fictional-production.iam" {
			t.Fatal("wrong Production IAM identity")
		}
	}
	for _, c := range []Config{{"jayburd-bookkeeping-dev:us-central1:pilot-db", "api"}, {"fictional-production:us-central1:jayburd-dev", "api"}, {"fictional-production:us-central1:pilot-db", "verifier"}} {
		if c.ValidateEnvironment("production") == nil {
			t.Fatal("forbidden Production target accepted")
		}
	}
}
