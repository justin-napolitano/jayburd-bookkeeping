//go:build integration

package server

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var serverProof = flag.Bool("server-proof", false, "use only the ledger runner's synthetic internal-network database")

// TestRuntimeReadiness uses isolated fixture roles, never owner-injected connection strings.
func TestRuntimeReadiness(t *testing.T) {
	if !*serverProof {
		t.Fatal("use the disposable synthetic runner; no inherited DSN accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	for _, q := range []string{"CREATE DATABASE server_readiness TEMPLATE ledger_template", "CREATE ROLE synthetic_server LOGIN IN ROLE bookkeeping_api"} {
		if _, err = admin.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	setup, err := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=server_readiness sslmode=disable connect_timeout=5")
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Close(ctx)
	pool, err := pgxpool.New(ctx, "host=database port=5432 user=synthetic_server dbname=server_readiness sslmode=disable connect_timeout=5 pool_max_conns=2")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = runtimeReady(ctx, pool); err != nil {
		t.Fatal("least privilege runtime not ready", err)
	}
	for _, tc := range []struct{ name, grant, revoke string }{
		{"table_read", "GRANT SELECT ON ledger.principal TO synthetic_server", "REVOKE SELECT ON ledger.principal FROM synthetic_server"},
		{"enrollment", "GRANT EXECUTE ON FUNCTION ledger.enroll_principal(text,text) TO synthetic_server", "REVOKE EXECUTE ON FUNCTION ledger.enroll_principal(text,text) FROM synthetic_server"},
		{"deployment_ddl", "CREATE SCHEMA deployment; GRANT CREATE ON SCHEMA deployment TO synthetic_server", "DROP SCHEMA deployment"},
		{"deployment_read", "CREATE SCHEMA deployment; CREATE TABLE deployment.migration_artifact(id int); GRANT USAGE ON SCHEMA deployment TO synthetic_server; GRANT SELECT ON deployment.migration_artifact TO synthetic_server", "DROP SCHEMA deployment CASCADE"},
		{"cloud_admin", "CREATE ROLE cloudsqlsuperuser; GRANT cloudsqlsuperuser TO synthetic_server", "REVOKE cloudsqlsuperuser FROM synthetic_server; DROP ROLE cloudsqlsuperuser"},
		{"schema_ddl", "GRANT CREATE ON SCHEMA reporting TO synthetic_server", "REVOKE CREATE ON SCHEMA reporting FROM synthetic_server"},
		{"owner_role", "GRANT bookkeeping_owner TO synthetic_server", "REVOKE bookkeeping_owner FROM synthetic_server"},
		{"identity_operator", "GRANT bookkeeping_identity_operator TO synthetic_server", "REVOKE bookkeeping_identity_operator FROM synthetic_server"},
		{"publisher_role", "GRANT bookkeeping_publisher TO synthetic_server", "REVOKE bookkeeping_publisher FROM synthetic_server"},
		{"observer_role", "GRANT bookkeeping_observer TO synthetic_server", "REVOKE bookkeeping_observer FROM synthetic_server"},
		{"missing_recovery_register", "REVOKE EXECUTE ON FUNCTION ledger.register_command(uuid,text,uuid,text) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION ledger.register_command(uuid,text,uuid,text) TO bookkeeping_api"},
		{"missing_recovery_pending", "REVOKE EXECUTE ON FUNCTION ledger.pending_command(uuid) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION ledger.pending_command(uuid) TO bookkeeping_api"},
		{"missing_recovery_transition", "REVOKE EXECUTE ON FUNCTION ledger.transition_command(uuid,uuid,uuid,text,text) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION ledger.transition_command(uuid,uuid,uuid,text,text) TO bookkeeping_api"},
		{"missing_workspace", "REVOKE EXECUTE ON FUNCTION reporting.workspace(uuid) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION reporting.workspace(uuid) TO bookkeeping_api"},
		{"missing_list_sources", "REVOKE EXECUTE ON FUNCTION banking.list_sources(uuid,uuid,uuid) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION banking.list_sources(uuid,uuid,uuid) TO bookkeeping_api"},
		{"missing_list_review_items", "REVOKE EXECUTE ON FUNCTION banking.list_review_items(uuid,uuid,uuid,text) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION banking.list_review_items(uuid,uuid,uuid,text) TO bookkeeping_api"},
		{"missing_get_review_item", "REVOKE EXECUTE ON FUNCTION banking.get_review_item(uuid,uuid,uuid,uuid) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION banking.get_review_item(uuid,uuid,uuid,uuid) TO bookkeeping_api"},
		{"missing_review_categories", "REVOKE EXECUTE ON FUNCTION banking.review_categories(uuid,uuid,uuid,uuid) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION banking.review_categories(uuid,uuid,uuid,uuid) TO bookkeeping_api"},
		{"missing_create_review_draft", "REVOKE EXECUTE ON FUNCTION banking.create_review_draft(uuid,uuid,uuid,uuid,text,text) FROM bookkeeping_api", "GRANT EXECUTE ON FUNCTION banking.create_review_draft(uuid,uuid,uuid,uuid,text,text) TO bookkeeping_api"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := setup.Exec(ctx, tc.grant); err != nil {
				t.Fatal(err)
			}
			if err := runtimeReady(ctx, pool); err == nil {
				t.Fatal("unsafe runtime reported ready")
			}
			if _, err := setup.Exec(ctx, tc.revoke); err != nil {
				t.Fatal(err)
			}
			if err := runtimeReady(ctx, pool); err != nil {
				t.Fatal("restored runtime not ready")
			}
		})
	}
}
