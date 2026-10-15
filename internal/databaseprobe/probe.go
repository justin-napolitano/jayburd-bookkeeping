// Package databaseprobe proves runtime ACLs without retaining any probe mutations.
package databaseprobe

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrProbe = errors.New("database permission proof failed")

// transaction exposes only bounded execution, a Boolean catalog check and rollback.
type transaction interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	Rollback() error
	allowed(context.Context, string) (bool, error)
}

type sqlTransaction struct{ *sql.Tx }

// allowed consumes only a catalog-derived Boolean, never business rows.
func (t sqlTransaction) allowed(ctx context.Context, identity string) (bool, error) {
	var ok bool
	err := t.QueryRowContext(ctx, `SELECT current_user=$1 AND session_user=$1
 AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls
 AND NOT has_database_privilege(current_user,current_database(),'CREATE')
 AND NOT has_schema_privilege(current_user,'public','CREATE')
 AND has_function_privilege(current_user,'reporting.workspace(uuid)','EXECUTE')
 AND NOT EXISTS (SELECT FROM pg_roles p WHERE p.rolname IN
 ('bookkeeping_owner','bookkeeping_identity_operator','cloudsqlsuperuser','bookkeeping_publisher','bookkeeping_observer','bookkeeping_sync','bookkeeping_webhook')
 AND pg_has_role(current_user,p.oid,'MEMBER'))
 FROM pg_roles WHERE rolname=current_user`, identity).Scan(&ok)
	return ok, err
}

type beginFunc func(context.Context) (transaction, error)

// Run executes through the caller's actual authenticated login, never SET ROLE simulation.
func Run(ctx context.Context, db *sql.DB, identity string) error {
	return run(ctx, func(ctx context.Context) (transaction, error) {
		tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
		if err != nil {
			return nil, err
		}
		return sqlTransaction{tx}, nil
	}, identity)
}

// run always rolls back each transaction, including an unexpected successful DDL.
func run(ctx context.Context, begin beginFunc, identity string) error {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	tx, err := begin(ctx)
	if err != nil {
		return ErrProbe
	}
	ok, checkErr := tx.allowed(ctx, identity)
	rollbackErr := tx.Rollback()
	if checkErr != nil || rollbackErr != nil || !ok {
		return ErrProbe
	}
	for _, p := range probes {
		tx, err := begin(ctx)
		if err != nil {
			return ErrProbe
		}
		_, queryErr := tx.ExecContext(ctx, p.query)
		rollbackErr := tx.Rollback()
		var pgErr *pgconn.PgError
		if rollbackErr != nil || !errors.As(queryErr, &pgErr) || pgErr.Code != p.state {
			return ErrProbe
		}
	}
	return nil
}

// Probe names are fixed; no user-supplied SQL or provider/business payload is accepted.
var probes = []struct{ query, state string }{
	{`CREATE TABLE public.jayburd_permission_probe(id int)`, "42501"},
	{`CREATE SCHEMA jayburd_permission_probe`, "42501"},
	{`CREATE ROLE jayburd_permission_probe`, "42501"},
	{`SELECT * FROM ledger.entity LIMIT 0`, "42501"},
	{`SELECT * FROM deployment.migration_artifact LIMIT 0`, "42501"},
	{`SELECT * FROM integration.work_operation LIMIT 0`, "42501"},
	{`SET ROLE bookkeeping_owner`, "42501"},
	{`SET ROLE bookkeeping_identity_operator`, "42501"},
	{`SET ROLE bookkeeping_publisher`, "42501"},
	{`SET ROLE bookkeeping_observer`, "42501"},
	{`SET ROLE bookkeeping_sync`, "42501"},
	{`SET ROLE bookkeeping_webhook`, "42501"},
	{`SELECT integration.observe_discover('00000000-0000-4000-8000-000000000099',1)`, "42501"},
	{`SELECT reporting.workspace('00000000-0000-4000-8000-000000000099')`, "P2101"},
}
