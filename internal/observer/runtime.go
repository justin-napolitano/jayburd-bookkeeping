package observer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	randv2 "math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Beginner supplies an independently configured publisher or consumer connection pool.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Failure exposes only stable classifications, never SQL or transport diagnostic text.
type Failure struct {
	Code      string
	Retryable bool
}

// Error returns the safe protocol classification only.
func (f *Failure) Error() string { return f.Code }

// reject constructs a fixed nonretryable protocol failure.
func reject() error { return &Failure{Code: "invalid_work", Retryable: false} }

// unavailable marks uncertainty without exposing infrastructure details.
func unavailable() error { return &Failure{Code: "temporarily_unavailable", Retryable: true} }

// safeError maps fixed database states without returning or wrapping the original error.
func safeError(err error) error {
	var f *Failure
	if errors.As(err, &f) {
		return f
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "P2200":
			return reject()
		case "P2201", "42501":
			return &Failure{Code: "work_unavailable"}
		case "P2202", "P2204", "P2205":
			return &Failure{Code: "work_configuration"}
		case "P2203":
			return &Failure{Code: "stale_claim", Retryable: true}
		}
	}
	return unavailable()
}

// retrySQL includes only server-confirmed whole-transaction rollback outcomes.
func retrySQL(err error) bool {
	var p *pgconn.PgError
	return errors.As(err, &p) && (p.Code == "40001" || p.Code == "40P01")
}

// adapter retains one capability, preventing consumer operations through publisher configuration.
type adapter struct {
	db   Beginner
	role string
}

// roleQuery rejects privileged or SET ROLE sessions before calling any work function.
const roleQuery = `SELECT session_user=current_user AND r.rolcanlogin AND NOT (r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication)
 AND pg_has_role(r.oid,$1::text,'USAGE')
 AND NOT EXISTS(SELECT FROM pg_roles p WHERE p.rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_migrator','bookkeeping_identity_operator','bookkeeping_webhook','bookkeeping_sync','bookkeeping_publisher','bookkeeping_observer') AND p.rolname<>$1 AND pg_has_role(r.oid,p.oid,'MEMBER'))
 AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','reporting') AND c.relkind IN ('r','p') AND has_table_privilege(r.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 FROM pg_roles r WHERE r.rolname=session_user`

// run uses one bounded call budget across retries and exposes results only after commit.
func (a adapter) run(parent context.Context, fn func(context.Context, pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(parent, 6*time.Second)
	defer cancel()
	for i := 0; i < 3; i++ {
		if ctx.Err() != nil {
			return unavailable()
		}
		err, unknown := a.attempt(ctx, fn)
		if err == nil {
			return nil
		}
		if unknown {
			return &Failure{Code: "commit_unknown", Retryable: true}
		}
		if !retrySQL(err) || i == 2 {
			return safeError(err)
		}
		timer := time.NewTimer(time.Duration(5+randv2.IntN(20)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return unavailable()
		case <-timer.C:
		}
	}
	return unavailable()
}

// attempt rolls back on an independent cleanup context; no callback may perform network work.
func (a adapter) attempt(ctx context.Context, fn func(context.Context, pgx.Tx) error) (err error, unknown bool) {
	tx, err := a.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err = tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL lock_timeout='1s'; SET LOCAL idle_in_transaction_session_timeout='6s'"); err != nil {
		return err, false
	}
	var valid bool
	if err = tx.QueryRow(ctx, roleQuery, a.role).Scan(&valid); err != nil {
		return err, false
	}
	if !valid {
		return &Failure{Code: "work_configuration"}, false
	}
	if err = fn(ctx, tx); err != nil {
		return err, false
	}
	if err = ctx.Err(); err != nil {
		return err, false
	}
	err = tx.Commit(ctx)
	if err == nil {
		return nil, false
	}
	var p *pgconn.PgError
	if errors.Is(err, pgx.ErrTxCommitRollback) {
		return err, false
	}
	// Only explicit rollback/constraint rejection is known; 08007/40003 and other
	// unclassified server outcomes remain ambiguous even though they are PgError.
	if errors.As(err, &p) && (retrySQL(err) || strings.HasPrefix(p.Code, "23") || strings.HasPrefix(p.Code, "P22") || p.Code == "25P02") {
		return err, false
	}
	return err, true
}

// newHolder creates a fresh canonical UUID once per claim, preserved across SQL retries.
func newHolder() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", unavailable()
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	s := hex.EncodeToString(b[:])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:], nil
}

// taskName independently checks SQL's fixed target-version digest profile.
func taskName(id string, g int64) string {
	sum := sha256.Sum256([]byte(id + "|" + strconv.FormatInt(g, 10) + "|1"))
	return hex.EncodeToString(sum[:])
}

// member checks closed status sets without accepting arbitrary database output.
func member(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}

// leaseValid rejects missing and nonfinite decoded server timestamps without using local time as lease authority.
func leaseValid(t *time.Time) bool {
	return t != nil && !t.IsZero() && t.Year() > 0 && t.Year() <= 9999
}
