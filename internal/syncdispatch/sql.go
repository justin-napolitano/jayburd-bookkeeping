package syncdispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"math/rand/v2"
	"strings"
	"time"
)

// adapter owns short role-checked transactions and exposes results only after COMMIT.
type adapter struct {
	DB   acquisition.Beginner
	role string
}

const roleQuery = `SELECT session_user=current_user AND r.rolcanlogin AND NOT(r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication)
AND pg_has_role(r.oid,$1,'USAGE')
AND NOT EXISTS(SELECT FROM pg_roles p WHERE (p.rolsuper OR p.rolbypassrls OR p.rolcreaterole OR p.rolcreatedb OR p.rolreplication OR p.rolname IN ('bookkeeping_sync','bookkeeping_owner','bookkeeping_api','bookkeeping_identity_operator','bookkeeping_webhook','bookkeeping_publisher','bookkeeping_observer','bookkeeping_migrator','cloudsqlsuperuser')) AND p.rolname<>$1 AND pg_has_role(r.oid,p.oid,'MEMBER'))
AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','deployment') AND c.relkind IN ('r','p','v','m','S') AND has_table_privilege(r.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
FROM pg_roles r WHERE r.rolname=session_user`

// sqlError translates only fixed SQLSTATEs and discards raw diagnostic text.
func sqlError(e error) error {
	var p *pgconn.PgError
	if errors.As(e, &p) {
		switch p.Code {
		case "P2900":
			return failure("invalid_request", false)
		case "P2901", "42501":
			return failure("not_authorized", false)
		case "P2902":
			return failure("evidence_conflict", false)
		case "P2903":
			return failure("stale_attempt", false)
		case "P2904":
			return failure("invalid_update", false)
		case "P2905":
			return failure("limit_exceeded", false)
		}
	}
	return failure("temporarily_unavailable", true)
}

// retrySQL includes only server-confirmed whole-transaction rollback failures.
func retrySQL(e error) bool {
	var p *pgconn.PgError
	return errors.As(e, &p) && (p.Code == "40001" || p.Code == "40P01")
}

// knownRollback conservatively distinguishes confirmed rollback from ambiguous COMMIT.
func knownRollback(e error) bool {
	if errors.Is(e, pgx.ErrTxCommitRollback) {
		return true
	}
	var p *pgconn.PgError
	if !errors.As(e, &p) {
		return false
	}
	return retrySQL(e) || p.Code == "25P02" || strings.HasPrefix(p.Code, "23") || strings.HasPrefix(p.Code, "P29")
}

// call retries SQL only; network/storage never enters the transaction callback.
func (r *adapter) call(parent context.Context, q string, args ...any) (json.RawMessage, error) {
	if r == nil || r.DB == nil {
		return nil, failure("configuration_error", false)
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for n := 0; n < 3; n++ {
		if ctx.Err() != nil {
			return nil, failure("temporarily_unavailable", true)
		}
		raw, e, unknown := r.transaction(ctx, q, args...)
		if e == nil {
			return raw, nil
		}
		if unknown {
			return nil, failure("commit_unknown", true)
		}
		if !retrySQL(e) || n == 2 {
			return nil, sqlError(e)
		}
		timer := time.NewTimer(time.Duration(5+rand.IntN(20)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, failure("temporarily_unavailable", true)
		case <-timer.C:
		}
	}
	return nil, failure("temporarily_unavailable", true)
}

// transaction owns one serializable transaction and independently bounded cleanup.
func (r *adapter) transaction(ctx context.Context, q string, args ...any) (json.RawMessage, error, bool) {
	tx, e := r.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return nil, e, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='3s'; SET LOCAL lock_timeout='1s'; SET LOCAL idle_in_transaction_session_timeout='4s'"); e != nil {
		return nil, e, false
	}
	var safe bool
	if e = tx.QueryRow(ctx, roleQuery, r.role).Scan(&safe); e != nil {
		return nil, e, false
	}
	if !safe {
		return nil, &pgconn.PgError{Code: "42501"}, false
	}
	var raw []byte
	if e = tx.QueryRow(ctx, q, args...).Scan(&raw); e != nil {
		return nil, e, false
	}
	if len(raw) == 0 || len(raw) > 16384 || !json.Valid(raw) || bytes.TrimSpace(raw)[0] != '{' {
		return nil, errors.New("invalid metadata"), false
	}
	if e = ctx.Err(); e != nil {
		return nil, e, false
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e, !knownRollback(e)
	}
	return raw, nil, false
}
