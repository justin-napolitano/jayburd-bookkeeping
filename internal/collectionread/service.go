// Package collectionread transports current owner-authorized source projections from PostgreSQL.
// It does not provision private grants, select an actor from input, acquire data or classify/post.
package collectionread

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// Request separates the resolved principal from unchanged private query JSON.
type Request struct{ Actor, Workspace, Body string }

// Reader exposes only source projections; SQL reauthorizes every read and continuation.
type Reader interface {
	Workspace(context.Context, Request) (json.RawMessage, error)
	Sources(context.Context, Request) (json.RawMessage, error)
	Observations(context.Context, Request) (json.RawMessage, error)
}

// Service calls fixed restricted SQL entrypoints through the caller-owned connection pool.
type Service struct{ DB ledger.Beginner }

// Workspace discovers an existing explicit private grant without creating or inferring one.
func (s *Service) Workspace(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.read(ctx, r, "workspace")
}

// Sources returns private aliases only, never legacy book IDs or credential/provider identifiers.
func (s *Service) Sources(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.read(ctx, r, "sources")
}

// Observations preserves original decimal strings and the SQL-owned current source snapshot.
func (s *Service) Observations(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.read(ctx, r, "observations")
}

// read retries only known transaction aborts and never returns a body after uncertain commit.
func (s *Service) read(parent context.Context, r Request, operation string) (json.RawMessage, error) {
	if s == nil || s.DB == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		body, err, unknown := s.attempt(ctx, r, operation)
		if err == nil {
			return body, nil
		}
		if unknown {
			return nil, unavailable()
		}
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || (pg.Code != "40001" && pg.Code != "40P01") {
			return nil, safeFailure(err)
		}
		if attempt < 2 {
			timer := time.NewTimer(time.Duration(attempt+1) * 10 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, unavailable()
			case <-timer.C:
			}
		}
	}
	return nil, unavailable()
}

// attempt uses a single repeatable snapshot with current authority locks before any result escapes.
func (s *Service) attempt(ctx context.Context, r Request, operation string) (json.RawMessage, error, bool) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return nil, err, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) < time.Millisecond {
		return nil, context.DeadlineExceeded, false
	}
	remaining := (time.Until(deadline) / time.Millisecond * time.Millisecond).String()
	if _, err = tx.Exec(ctx, "select set_config('statement_timeout',$1,true),set_config('lock_timeout',$1,true)", remaining); err != nil {
		return nil, err, false
	}
	var body json.RawMessage
	switch operation {
	case "workspace":
		err = tx.QueryRow(ctx, "select banking.get_collection_workspace($1::uuid)", r.Actor).Scan(&body)
	case "sources":
		err = tx.QueryRow(ctx, "select banking.list_collection_sources($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
	case "observations":
		err = tx.QueryRow(ctx, "select banking.query_collection_observations($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
	default:
		return nil, &ledger.Failure{Status: 400, Code: "invalid_collection"}, false
	}
	if err != nil {
		return nil, err, false
	}
	if !validResult(body, operation) {
		return nil, unavailable(), false
	}
	if err = tx.Commit(ctx); err != nil {
		var pg *pgconn.PgError
		unknown := !errors.As(err, &pg) && !errors.Is(err, pgx.ErrTxCommitRollback)
		if pg != nil && (pg.Code == "40003" || strings.HasPrefix(pg.Code, "08")) {
			unknown = true
		}
		return nil, err, unknown
	}
	return body, nil, false
}

// validResult fences the SQL envelope and resource budget without interpreting financial fields.
func validResult(body json.RawMessage, operation string) bool {
	if len(body) > 1<<20 || !json.Valid(body) {
		return false
	}
	var envelope struct {
		Profile string `json:"profile"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	profiles := map[string]string{"workspace": "private-source-workspace-v1", "sources": "private-source-page-v1", "observations": "private-source-query-v1"}
	return profiles[operation] != "" && envelope.Profile == profiles[operation]
}

// unavailable deliberately discards all database, transport and identifier diagnostics.
func unavailable() *ledger.Failure {
	return &ledger.Failure{Status: 503, Code: "temporarily_unavailable"}
}

// safeFailure is the closed public error vocabulary; neither SQL text nor payloads survive it.
func safeFailure(err error) *ledger.Failure {
	var existing *ledger.Failure
	if errors.As(err, &existing) && existing.Status == 503 {
		return unavailable()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P2800", "22P02", "22021":
			return &ledger.Failure{Status: 400, Code: "invalid_collection"}
		case "P2801":
			return &ledger.Failure{Status: 404, Code: "collection_denied"}
		case "P2802":
			return &ledger.Failure{Status: 409, Code: "collection_conflict"}
		case "P2803":
			return &ledger.Failure{Status: 409, Code: "collection_snapshot_changed"}
		case "P2804":
			return &ledger.Failure{Status: 409, Code: "collection_not_ready"}
		case "P2805":
			return &ledger.Failure{Status: 422, Code: "collection_limit"}
		}
	}
	return unavailable()
}
