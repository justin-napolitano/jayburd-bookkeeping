package server

import (
	"context"
	"errors"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/identity"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// runtimeSafety rejects owner/operator/worker or table authority and requires installed
// reviewed entrypoints. It does not claim provider availability or production approval.
const runtimeSafety = `SELECT
 EXISTS(SELECT FROM pg_roles WHERE rolname=current_user AND rolcanlogin AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
 AND pg_has_role(current_user,'bookkeeping_api','USAGE')
 AND NOT EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolbypassrls OR rolcreatedb OR rolcreaterole OR rolreplication OR rolname IN ('bookkeeping_owner','bookkeeping_identity_operator','bookkeeping_publisher','bookkeeping_observer','bookkeeping_sync','cloudsqlsuperuser')) AND pg_has_role(current_user,oid,'MEMBER'))
 AND NOT has_database_privilege(current_user,current_database(),'CREATE')
 AND NOT EXISTS(SELECT FROM pg_namespace WHERE nspname IN ('public','ledger','reporting','reference','audit','integration','banking','deployment') AND has_schema_privilege(current_user,oid,'CREATE'))
 AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','deployment') AND c.relkind IN ('r','p','v','m','S') AND has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 AND NOT has_function_privilege(current_user,'ledger.enroll_principal(text,text)','EXECUTE')
 AND to_regprocedure('ledger.register_command(uuid,text,uuid,text)') IS NOT NULL
 AND has_function_privilege(current_user,'ledger.register_command(uuid,text,uuid,text)','EXECUTE')
 AND to_regprocedure('ledger.pending_command(uuid)') IS NOT NULL
 AND has_function_privilege(current_user,'ledger.pending_command(uuid)','EXECUTE')
 AND to_regprocedure('ledger.transition_command(uuid,uuid,uuid,text,text)') IS NOT NULL
 AND has_function_privilege(current_user,'ledger.transition_command(uuid,uuid,uuid,text,text)','EXECUTE')
 AND to_regprocedure('reporting.workspace(uuid)') IS NOT NULL
 AND has_function_privilege(current_user,'reporting.workspace(uuid)','EXECUTE')
 AND to_regprocedure('banking.list_sources(uuid,uuid,uuid)') IS NOT NULL
 AND has_function_privilege(current_user,'banking.list_sources(uuid,uuid,uuid)','EXECUTE')
 AND to_regprocedure('banking.list_review_items(uuid,uuid,uuid,text)') IS NOT NULL
 AND has_function_privilege(current_user,'banking.list_review_items(uuid,uuid,uuid,text)','EXECUTE')
 AND to_regprocedure('banking.get_review_item(uuid,uuid,uuid,uuid)') IS NOT NULL
 AND has_function_privilege(current_user,'banking.get_review_item(uuid,uuid,uuid,uuid)','EXECUTE')
 AND to_regprocedure('banking.review_categories(uuid,uuid,uuid,uuid)') IS NOT NULL
 AND has_function_privilege(current_user,'banking.review_categories(uuid,uuid,uuid,uuid)','EXECUTE')
 AND to_regprocedure('banking.create_review_draft(uuid,uuid,uuid,uuid,text,text)') IS NOT NULL
 AND has_function_privilege(current_user,'banking.create_review_draft(uuid,uuid,uuid,uuid,text,text)','EXECUTE')`

// runtimeReady checks current role privileges on each probe using a bounded query.
func runtimeReady(ctx context.Context, p *pgxpool.Pool) error {
	var safe bool
	if err := p.QueryRow(ctx, runtimeSafety).Scan(&safe); err != nil || !safe {
		return errors.New("dependencies unavailable")
	}
	return nil
}

// Managed constructs clients only when explicitly selected. The Admin SDK receives
// a metadata-only token source and an explicit project: no ADC/file discovery.
func Managed(ctx context.Context, c Config) (Dependencies, error) {
	if c.Mode != "managed" || c.Validate() != nil || forbiddenEnvironment(func(k string) bool { _, ok := os.LookupEnv(k); return ok }) {
		return Dependencies{}, errConfiguration
	}
	var cfg *pgxpool.Config
	var err error
	closeTransport := func() {}
	if c.DatabaseMode == "cloudsql" {
		if _, present := os.LookupEnv("JAYBURD_DATABASE_URL"); present {
			return Dependencies{}, errConfiguration
		}
		cfg, closeTransport, err = clouddb.Connector(ctx, clouddb.Config{Instance: c.CloudSQLInstance, Component: "api"})
	} else {
		cfg, err = poolConfig(c.DatabaseURL)
	}
	if err != nil {
		return Dependencies{}, errConfiguration
	}
	start, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(start, cfg)
	if err != nil {
		closeTransport()
		return Dependencies{}, errors.New("dependencies unavailable")
	}
	closeAll := func() { pool.Close(); closeTransport() }
	fail := func() (Dependencies, error) {
		closeAll()
		return Dependencies{}, errors.New("dependencies unavailable")
	}
	if runtimeReady(start, pool) != nil {
		return fail()
	}
	svc := &ledger.Service{DB: pool}
	app, err := firebase.NewApp(start, &firebase.Config{ProjectID: c.Project}, option.WithTokenSource(google.ComputeTokenSource("default", "https://www.googleapis.com/auth/cloud-platform")))
	if err != nil {
		return fail()
	}
	client, err := app.Auth(start)
	if err != nil {
		return fail()
	}
	auth, err := identity.New(client, c.Project, svc)
	if err != nil {
		return fail()
	}
	return Dependencies{Auth: auth, Operations: svc, Ready: func(ctx context.Context) error { return runtimeReady(ctx, pool) }, Close: closeAll}, nil
}
