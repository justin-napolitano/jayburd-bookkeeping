package connections

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

// sqlRepository owns bounded SQL calls; no transaction callback performs storage I/O.
type sqlRepository struct{ DB Beginner }

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
		case "P2700", "P2800", "22P02", "22021":
			return failed("invalid_request")
		case "P2701", "P2801", "P2101", "P2102":
			return failed("not_found")
		case "P2702", "P2802":
			return failed("connection_conflict")
		case "P2703", "P2803":
			return failed("stale_connection")
		case "P2704":
			return failed("credential_recovery_required")
		case "P2804":
			return failed("connection_operation_blocked")
		case "P2705", "P2805":
			return failed("connection_limit")
		}
	}
	return failed("temporarily_unavailable")
}

// call retries complete SQL transactions only, exposing JSON after confirmed COMMIT.
func (r *sqlRepository) call(parent context.Context, operation string, args ...any) (json.RawMessage, error) {
	query, ok := queries[operation]
	if !ok {
		return nil, failed("configuration_error")
	}
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
func (r *sqlRepository) attempt(ctx context.Context, query string, args ...any) (json.RawMessage, error, bool) {
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

// queries is a closed trusted-backend SQL surface; request strings never select SQL.
var queries = map[string]string{
	"abandon_connection":               "SELECT integration.abandon_connection($1,$2,$3,$4)::text",
	"begin_binding_abandon":            "SELECT integration.begin_binding_abandon($1,$2,$3,$4)::text",
	"confirm_binding_abandoned":        "SELECT integration.confirm_binding_abandoned($1,$2,$3,$4,$5)::text",
	"creator_begin_connection":         "SELECT integration.creator_begin_connection($1,$2,$3,$4,$5,$6,$7)::text",
	"creator_list_connections":         "SELECT integration.creator_list_connections($1,$2,$3,$4,$5)::text",
	"creator_list_connection_attempts": "SELECT integration.creator_list_connection_attempts($1,$2,$3,$4,$5,$6,$7)::text",
	"list_connection_attempts":         "SELECT integration.list_connection_attempts($1,$2,$3,$4,$5)::text",
	"connection_stream_status":         "SELECT integration.connection_stream_status($1,$2,$3,$4)::text",
	"mutate_connection_stream":         "SELECT integration.mutate_connection_stream($1,$2,$3,$4,$5,$6)::text",
	"begin_connection":                 "SELECT integration.begin_connection($1,$2,$3,$4,$5)::text",
	"record_link_session":              "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text",
	"claim_connection":                 "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text",
	"record_connection_manifest":       "SELECT integration.record_connection_manifest($1,$2,$3,$4,$5,$6)::text",
	"confirm_connection_credential":    "SELECT integration.confirm_connection_credential($1,$2,$3,$4,$5)::text",
	"connection_failure":               "SELECT integration.connection_failure($1,$2,$3,$4,$5,$6)::text",
	"connection_context":               "SELECT integration.connection_context($1,$2,$3,$4)::text",
	"get_connection_attempt":           "SELECT integration.get_connection_attempt($1,$2,$3,$4)::text",
	"bind_connection":                  "SELECT integration.bind_connection($1,$2,$3,$4,$5,$6)::text",
	"list_connections":                 "SELECT integration.list_connections($1,$2,$3)::text",
	"disconnect_stream":                "SELECT integration.disconnect_stream($1,$2,$3,$4,$5,$6)::text",
	"confirm_item_removed":             "SELECT integration.confirm_item_removed($1,$2,$3,$4,$5)::text",
}
