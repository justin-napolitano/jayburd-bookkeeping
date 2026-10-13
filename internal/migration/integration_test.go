//go:build integration

package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

var cloudSQLProof = flag.Bool("cloud-sql-proof", false, "enable isolated synthetic Docker proof only")

// proofDB hardcodes the disposable network and synthetic roles; no environment or live DSN is used.
func proofDB(t *testing.T, user, db string) *sql.DB {
	t.Helper()
	p, err := sql.Open("pgx", "postgres://"+user+"@database:5432/"+db+"?sslmode=disable&connect_timeout=5&statement_timeout=60000")
	if err != nil {
		t.Fatal(err)
	}
	p.SetMaxOpenConns(2)
	t.Cleanup(func() { p.Close() })
	return p
}

// TestRestrictedMigration applies unchanged product SQL as a non-superuser without role-creation rights.
func TestRestrictedMigration(t *testing.T) {
	if !*cloudSQLProof {
		t.Skip("disposable proof not selected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		if err := Up(ctx, proofDB(t, "synthetic_migrator", "bookkeeping"), migrations.Files); err != nil {
			t.Fatal(err)
		}
	}
	admin := proofDB(t, "postgres", "bookkeeping")
	var count int
	if err := admin.QueryRowContext(ctx, `SELECT count(*) FROM deployment.migration_artifact`).Scan(&count); err != nil || count == 0 {
		t.Fatal("missing artifact receipts")
	}
	runtime := proofDB(t, "synthetic_api", "bookkeeping")
	var allowed bool
	if err := runtime.QueryRowContext(ctx, `SELECT has_function_privilege(current_user,'reporting.workspace(uuid)','EXECUTE')`).Scan(&allowed); err != nil || !allowed {
		t.Fatal("runtime lost its authorized entrypoint")
	}

	for _, query := range []string{`CREATE TABLE public.denied(id int)`, `CREATE SCHEMA denied`, `CREATE ROLE denied`, `SELECT * FROM ledger.entity LIMIT 0`, `SELECT * FROM deployment.migration_artifact LIMIT 0`, `SET ROLE bookkeeping_owner`} {
		_, err := runtime.ExecContext(ctx, query)
		requireSQLState(t, err, "42501")
	}
	// Satisfy the isolation precondition so P2101 proves principal denial, not an unrelated P2100 failure.
	tx, err := runtime.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal("runtime acceptance transaction unavailable")
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `SELECT reporting.workspace('00000000-0000-4000-8000-000000000099')`)
	requireSQLState(t, err, "P2101")
	if err := tx.Rollback(); err != nil {
		t.Fatal("runtime acceptance rollback failed")
	}
	files := copyAssets(t)
	files["000001_ledger.up.sql"].Data = append(files["000001_ledger.up.sql"].Data, []byte("\n-- altered artifact\n")...)
	// Reseal a changed artifact to distinguish durable receipt validation from file checksums.
	manifest := []byte{}
	for name, file := range files {
		if name != "SHA256SUMS" {
			manifest = append(manifest, []byte(fmt.Sprintf("%x  db/migrations/%s\n", sha256.Sum256(file.Data), name))...)
		}
	}
	files["SHA256SUMS"] = &fstest.MapFile{Data: manifest}
	if err := Up(ctx, proofDB(t, "synthetic_migrator", "bookkeeping"), files); err == nil {
		t.Fatal("resealed deployed SQL accepted")
	}
}

// requireSQLState prevents connection, syntax or isolation failures from being mistaken for authorization denial.
func requireSQLState(t *testing.T, err error, expected string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != expected {
		t.Fatalf("expected PostgreSQL rejection %s", expected)
	}
}

// TestMigrationContention rejects a second runner before it records or executes migration work.
func TestMigrationContention(t *testing.T) {
	if !*cloudSQLProof {
		t.Skip("disposable proof not selected")
	}
	db := proofDB(t, "postgres", "bookkeeping")
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(context.Background(), `SELECT pg_advisory_lock(784932617)`); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(784932617)`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := Up(ctx, proofDB(t, "synthetic_migrator", "bookkeeping"), migrations.Files); err == nil {
		t.Fatal("overlapping runner accepted")
	}
}

// TestMigrationDirtyFailure proves rollback, dirty-state retention and blocked retry under restricted identity.
func TestMigrationDirtyFailure(t *testing.T) {
	if !*cloudSQLProof {
		t.Skip("disposable proof not selected")
	}
	files := fstest.MapFS{
		"000001_failure.up.sql":   &fstest.MapFile{Data: []byte("BEGIN; CREATE TABLE deployment.must_rollback(id int); SELECT 1/0; COMMIT;")},
		"000001_failure.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	manifest := []byte{}
	for name, file := range files {
		manifest = append(manifest, []byte(fmt.Sprintf("%x  db/migrations/%s\n", sha256.Sum256(file.Data), name))...)
	}
	files["SHA256SUMS"] = &fstest.MapFile{Data: manifest}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		if err := Up(ctx, proofDB(t, "synthetic_migrator", "bookkeeping_failure"), files); err == nil {
			t.Fatal("failed/dirty migration accepted")
		}
	}
	admin := proofDB(t, "postgres", "bookkeeping_failure")
	var dirty, absent bool
	if err := admin.QueryRowContext(ctx, `SELECT dirty FROM deployment.schema_migrations`).Scan(&dirty); err != nil || !dirty {
		t.Fatal("dirty marker missing")
	}
	if err := admin.QueryRowContext(ctx, `SELECT to_regclass('deployment.must_rollback') IS NULL`).Scan(&absent); err != nil || !absent {
		t.Fatal("failed DDL did not roll back")
	}
}
