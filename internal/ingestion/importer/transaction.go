// Package importer connects synthetic staging and decoding to SQL-owned review.
// It never creates journals or executes import commands implicitly.
package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Beginner supplies a caller-owned restricted API pool without connection strings.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Failure carries a fixed public classification without raw infrastructure details.
type Failure struct{ Code string }

// Error deliberately excludes payloads, paths, identifiers and database messages.
func (e *Failure) Error() string { return e.Code }

// failed constructs a safe internal classification.
func failed(code string) error { return &Failure{Code: code} }

// SQLRepository owns bounded SQL calls; no transaction callback performs storage I/O.
type SQLRepository struct{ DB Beginner }

// retryable recognizes only confirmed whole-transaction rollback outcomes.
func retryable(err error) bool {
	var p *pgconn.PgError
	return errors.As(err, &p) && (p.Code == "40001" || p.Code == "40P01")
}

// sqlFailure exposes only reviewed classifications; unknown errors are redacted.
func sqlFailure(err error) error {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "P2400", "P2300", "22P02", "22021":
			return failed("invalid_request")
		case "P2401", "P2301", "P2101", "P2102":
			return failed("not_found")
		case "P2402":
			return failed("import_conflict")
		case "P2403":
			return failed("stale_preview")
		case "P2404":
			return failed("import_limit")
		case "P2302":
			return failed("source_conflict")
		case "P2303":
			return failed("unsupported_profile")
		case "P2304":
			return failed("source_limit")
		}
	}
	return failed("temporarily_unavailable")
}

// call retries complete SQL transactions only, exposing JSON after confirmed COMMIT.
func (r *SQLRepository) call(parent context.Context, query string, args ...any) (json.RawMessage, error) {
	if r == nil || r.DB == nil {
		return nil, failed("configuration_error")
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		if ctx.Err() != nil {
			return nil, failed("temporarily_unavailable")
		}
		body, err, unknown := r.attempt(ctx, query, args...)
		if err == nil {
			return body, nil
		}
		if unknown {
			return nil, failed("commit_unknown")
		}
		if !retryable(err) || attempt == 2 {
			return nil, sqlFailure(err)
		}
		timer := time.NewTimer(time.Duration(5+rand.IntN(20)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, failed("temporarily_unavailable")
		case <-timer.C:
		}
	}
	return nil, failed("temporarily_unavailable")
}

// attempt always cleans up with an independent budget and suppresses uncertain results.
func (r *SQLRepository) attempt(ctx context.Context, query string, args ...any) (json.RawMessage, error, bool) {
	tx, err := r.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err = tx.Exec(ctx, "SET LOCAL statement_timeout='3s'; SET LOCAL lock_timeout='1s'; SET LOCAL idle_in_transaction_session_timeout='4s'"); err != nil {
		return nil, err, false
	}
	var body []byte
	if err = tx.QueryRow(ctx, query, args...).Scan(&body); err != nil {
		return nil, err, false
	}
	if len(body) == 0 || len(body) > 2<<20 || !json.Valid(body) || bytes.TrimSpace(body)[0] != '{' {
		return nil, errors.New("invalid SQL response"), false
	}
	if err = ctx.Err(); err != nil {
		return nil, err, false
	}
	if err = tx.Commit(ctx); err != nil {
		var p *pgconn.PgError
		known := errors.Is(err, pgx.ErrTxCommitRollback) || retryable(err)
		if errors.As(err, &p) && (strings.HasPrefix(p.Code, "23") || strings.HasPrefix(p.Code, "P2") || p.Code == "25P02") {
			known = true
		}
		return nil, err, !known
	}
	return json.RawMessage(body), nil, false
}
