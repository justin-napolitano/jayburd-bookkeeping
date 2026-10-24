package connectionruntime

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// repairConfig supplies only the exact public Sandbox identifiers, never real credential values.
func repairConfig() Config {
	c := config()
	c.ProjectID, c.ProjectNumber = "jayburd-bookkeeping-dev", "9088794002"
	c.RegistryBucket = "jayburd-bookkeeping-dev-jayburd-dev-credential-registry"
	c.ClientIDVersion = "projects/9088794002/secrets/jayburd-dev-bank-sync-client-id/versions/1"
	c.ClientSecretVersion = "projects/9088794002/secrets/jayburd-dev-bank-sync-client-secret/versions/2"
	c.WebhookURL = connections.SandboxTransactionsWebhookURL
	c.LinkEventsQualified = false
	return c
}

// repairDB counts request-time SQL while refusing all real database access.
type repairDB struct{ calls int }

// BeginTx proves the capability reached current SQL authority, never a provider or key fallback.
func (d *repairDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	d.calls++
	return nil, errors.New("synthetic SQL unavailable")
}

var repairScope = connections.Scope{Actor: "00000000-0000-4000-8000-000000000001", Entity: "00000000-0000-4000-8000-000000000002", Book: "00000000-0000-4000-8000-000000000003"}

const repairAttempt = "00000000-0000-4000-8000-000000000004"

// TestHostedRepairRuntimeOptIn proves existing webhook opt-in alone stays disabled and explicit repair reaches SQL.
func TestHostedRepairRuntimeOptIn(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		c := repairConfig()
		c.HostedRepair = enabled
		db, token := &repairDB{}, &tokens{}
		calls := 0
		s, e := New(context.Background(), c, db, transport(func(r *http.Request) (*http.Response, error) {
			refs := []string{c.ClientIDVersion, c.ClientSecretVersion}
			if calls >= len(refs) || r.URL.Host != "secretmanager.googleapis.com" {
				t.Fatal("constructor or denied request reached provider/KMS")
			}
			name := refs[calls]
			calls++
			return response(name, "synthetic-static"), nil
		}), token)
		if e != nil || s == nil || calls != 2 || db.calls != 0 {
			t.Fatal("runtime construction changed startup authority")
		}
		_, e = s.Begin(context.Background(), repairScope, "repair-runtime", connections.BeginRequest{Mode: "reconnect"})
		want, wantDB := "unsupported_connection_mode", 0
		if enabled {
			want, wantDB = "temporarily_unavailable", 1
		}
		if e == nil || e.Error() != want || db.calls != wantDB || calls != 2 || token.access != 2 || token.identity != 0 {
			t.Fatal("repair permission was implicit or bypassed SQL")
		}
	}
}

// TestHostedRepairRuntimeValidation denies partial/foreign configuration before tokens, secrets, SQL or provider I/O.
func TestHostedRepairRuntimeValidation(t *testing.T) {
	for _, change := range []string{"disabled", "missing_receiver", "foreign_project", "foreign_number", "substituted_receiver", "malformed_approval", "far_future"} {
		t.Run(change, func(t *testing.T) {
			c := repairConfig()
			c.HostedRepair = true
			switch change {
			case "disabled":
				c = Config{HostedRepair: true}
			case "missing_receiver":
				c.WebhookURL = ""
			case "foreign_project":
				c.ProjectID = "foreign-project"
			case "foreign_number":
				c.ProjectNumber = "123456789012"
			case "substituted_receiver":
				c.WebhookURL = "https://wrong.invalid/webhooks/plaid"
			case "malformed_approval":
				c.ConsentExpiresAt = "tomorrow"
			case "far_future":
				c.ConsentExpiresAt = time.Now().Add(31 * 24 * time.Hour).UTC().Format(time.RFC3339)
			}
			db, token := &repairDB{}, &tokens{}
			calls := 0
			_, e := New(context.Background(), c, db, transport(func(*http.Request) (*http.Response, error) { calls++; return nil, ErrUnavailable }), token)
			if e != ErrConfiguration || calls+db.calls+token.access+token.identity != 0 {
				t.Fatal("invalid repair composition acquired authority")
			}
		})
	}
}

// TestHostedRepairRuntimeExpired preserves cold SQL bookkeeping inspection while closing every provider entry.
func TestHostedRepairRuntimeExpired(t *testing.T) {
	c := repairConfig()
	c.HostedRepair = true
	c.ConsentExpiresAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	db, token := &repairDB{}, &tokens{}
	calls := 0
	s, e := New(context.Background(), c, db, transport(func(*http.Request) (*http.Response, error) { calls++; return nil, ErrUnavailable }), token)
	if e != nil || s == nil || calls+db.calls+token.access+token.identity != 0 {
		t.Fatal("expired repair blocked startup or acquired dependencies")
	}
	_, e = s.Begin(context.Background(), repairScope, "expired", connections.BeginRequest{Mode: "reconnect"})
	if e == nil || e.Error() != "temporarily_unavailable" || db.calls != 0 {
		t.Fatal("expired repair reached SQL or provider")
	}
	_, e = s.Attempt(context.Background(), repairScope, repairAttempt)
	if e == nil || db.calls != 1 || calls+token.access+token.identity != 0 {
		t.Fatal("SQL-only recovery lost current-authority inspection")
	}
}
