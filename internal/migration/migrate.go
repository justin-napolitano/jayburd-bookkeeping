// Package migration applies checked, embedded SQL using golang-migrate's locking/dirty-state protocol.
package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"time"

	migrate "github.com/golang-migrate/migrate/v4"
	driver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var ErrMigration = errors.New("migration rejected or failed")
var filename = regexp.MustCompile(`^[0-9]{6}_[a-z][a-z0-9_]*\.(up|down)\.sql$`)

// Verify rejects missing, extra, malformed and modified SQL before any database access.
func Verify(files fs.FS) (map[string]string, error) {
	manifest, err := fs.ReadFile(files, "SHA256SUMS")
	if err != nil {
		return nil, ErrMigration
	}
	hashes := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 || !strings.HasPrefix(parts[1], "db/migrations/") {
			return nil, ErrMigration
		}
		name := strings.TrimPrefix(parts[1], "db/migrations/")
		if !filename.MatchString(name) || hashes[name] != "" {
			return nil, ErrMigration
		}
		raw, err := fs.ReadFile(files, name)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != parts[0] {
			return nil, ErrMigration
		}
		hashes[name] = parts[0]
	}
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, ErrMigration
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") && hashes[e.Name()] == "" {
			return nil, ErrMigration
		}
	}
	for name := range hashes {
		suffix := ".down.sql"
		if strings.HasSuffix(name, ".down.sql") {
			suffix = ".up.sql"
		}
		base := strings.TrimSuffix(strings.TrimSuffix(name, ".up.sql"), ".down.sql")
		if hashes[base+suffix] == "" {
			return nil, ErrMigration
		}
	}
	if len(hashes) == 0 {
		return nil, ErrMigration
	}
	return hashes, nil
}

// Up permits only checked forward migration; a separate advisory lock covers artifact receipts too.
// Receipts register attempted artifacts before execution, retaining their identity through a crash.
// golang-migrate closes the supplied database handle after it constructs its driver.
func Up(ctx context.Context, db *sql.DB, files fs.FS) error {
	hashes, err := Verify(files)
	if err != nil {
		return ErrMigration
	}
	var safe bool
	err = db.QueryRowContext(ctx, `SELECT NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls AND pg_has_role(current_user,'bookkeeping_owner','MEMBER') AND NOT EXISTS(SELECT FROM pg_roles r WHERE r.rolname='cloudsqlsuperuser' AND pg_has_role(current_user,r.oid,'MEMBER')) FROM pg_roles WHERE rolname=current_user`).Scan(&safe)
	if err != nil || !safe {
		return ErrMigration
	}
	lock, err := db.Conn(ctx)
	if err != nil {
		return ErrMigration
	}
	var acquired bool
	if err = lock.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(784932617)`).Scan(&acquired); err != nil || !acquired {
		lock.Close()
		return ErrMigration
	}
	closeDriver := func() {}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = lock.ExecContext(cleanup, `SELECT pg_advisory_unlock(784932617)`)
		_ = lock.Close()
		closeDriver()
	}()
	if _, err = lock.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS deployment.migration_artifact (filename text PRIMARY KEY, sha256 text NOT NULL CHECK(length(sha256)=64), registered_at timestamptz NOT NULL DEFAULT clock_timestamp())`); err != nil {
		return ErrMigration
	}
	rows, err := lock.QueryContext(ctx, `SELECT filename,sha256 FROM deployment.migration_artifact`)
	if err != nil {
		return ErrMigration
	}
	for rows.Next() {
		var name, hash string
		if rows.Scan(&name, &hash) != nil || hashes[name] != hash {
			rows.Close()
			return ErrMigration
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return ErrMigration
	}
	tx, err := lock.BeginTx(ctx, nil)
	if err != nil {
		return ErrMigration
	}
	defer tx.Rollback()
	names := make([]string, 0, len(hashes))
	for name := range hashes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err = tx.ExecContext(ctx, `INSERT INTO deployment.migration_artifact(filename,sha256) VALUES($1,$2) ON CONFLICT DO NOTHING`, name, hashes[name]); err != nil {
			return ErrMigration
		}
	}
	if tx.Commit() != nil {
		return ErrMigration
	}
	src, err := iofs.New(files, ".")
	if err != nil {
		return ErrMigration
	}
	defer src.Close()
	pg, err := driver.WithInstance(db, &driver.Config{DatabaseName: "bookkeeping", SchemaName: "deployment", StatementTimeout: 60 * time.Second})
	if err != nil {
		return ErrMigration
	}
	// Release the outer lock before closing the driver-owned database handle.
	closeDriver = func() { _ = pg.Close() }
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", pg)
	if err != nil {
		return ErrMigration
	}
	m.LockTimeout = 10 * time.Second
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			m.GracefulStop <- true
		case <-stopped:
		}
	}()
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return ErrMigration
	}
	if ctx.Err() != nil {
		return ErrMigration
	}
	return nil
}
