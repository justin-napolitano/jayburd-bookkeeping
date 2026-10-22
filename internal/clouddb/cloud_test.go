package clouddb

import (
	"context"
	"errors"
	"net"
	"testing"
)

// TestConnectorConfiguration proves exact identifiers, pool bounds and absence of raw TCP fallback.
func TestConnectorConfiguration(t *testing.T) {
	for _, component := range []string{"api", "migrator", "publisher", "sync", "webhook", "verifier"} {
		called := false
		cfg, err := PoolConfig(Config{"synthetic-project:us-central1:jayburd-dev", component}, func(context.Context, string, string) (net.Conn, error) {
			called = true
			return nil, errors.New("synthetic failure")
		})
		if err != nil {
			t.Fatal(err)
		}
		want := int32(5)
		if component != "api" {
			want = 2
		}
		if component == "verifier" {
			want = 1
		}
		if cfg.MaxConns != want || cfg.MinConns != 0 || cfg.ConnConfig.Password != "" || cfg.ConnConfig.TLSConfig != nil || len(cfg.ConnConfig.Fallbacks) != 0 || cfg.ConnConfig.User != "jayburd-dev-"+component+"@synthetic-project.iam" {
			t.Fatal("unsafe connector configuration")
		}
		_, err = cfg.ConnConfig.DialFunc(context.Background(), "tcp", "attacker.invalid:5432")
		if !called || err == nil {
			t.Fatal("connector boundary not used")
		}
	}
}

// TestConnectorRejectsAmbiguity verifies malformed metadata cannot reach any credential source.
func TestConnectorRejectsAmbiguity(t *testing.T) {
	for _, c := range []Config{{}, {"synthetic-project:europe-west1:db", "api"}, {"synthetic-project:us-central1:db", "owner"}, {"synthetic-project:us-central1:db password=hidden", "api"}, {"synthetic-project:us-central1:db", "webhook-api"}, {"synthetic-project:us-central1:db", "webhook@other-project.iam"}, {"synthetic-project:us-central1:db", "verifier-api"}, {"synthetic-project:us-central1:db", "verifier@other-project.iam"}} {
		if c.Validate() == nil {
			t.Fatal("unsafe metadata accepted")
		}
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	if _, _, err := Connector(context.Background(), Config{"synthetic-project:us-central1:jayburd-dev", "api"}); err == nil {
		t.Fatal("credential override accepted")
	}
}
