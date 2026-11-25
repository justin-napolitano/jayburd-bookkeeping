// Package autopost runs explicit approved source-backed accounting mappings.
// PostgreSQL owns current authority, exact money, policy selection and durable effects.
package autopost

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// Request keeps trusted verified identity and path scope separate from untouched JSON.
type Request struct{ Actor, Entity, Book, Body string }

// Operations exposes configuration and inert status only; no public execution port exists.
type Operations interface {
	GetPolicy(context.Context, Request) (json.RawMessage, error)
	SetPolicy(context.Context, Request) (json.RawMessage, error)
	Status(context.Context, Request) (json.RawMessage, error)
}

// Service receives an already restricted API pool; it discovers no credentials or providers.
type Service struct{ DB ledger.Beginner }

func (s *Service) GetPolicy(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.operation(ctx, r, "get")
}
func (s *Service) SetPolicy(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.operation(ctx, r, "set")
}
func (s *Service) Status(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.operation(ctx, r, "status")
}
func (s *Service) operation(ctx context.Context, r Request, op string) (json.RawMessage, error) {
	return s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var raw json.RawMessage
		var err error
		switch op {
		case "get":
			err = tx.QueryRow(ctx, "select banking.get_automatic_accounting($1::uuid,$2::uuid,$3::uuid)", r.Actor, r.Entity, r.Book).Scan(&raw)
		case "set":
			err = tx.QueryRow(ctx, "select banking.set_automatic_accounting($1::uuid,$2::uuid,$3::uuid,$4::text)", r.Actor, r.Entity, r.Book, r.Body).Scan(&raw)
		case "status":
			err = tx.QueryRow(ctx, "select banking.automatic_accounting_status($1::uuid,$2::uuid,$3::uuid)", r.Actor, r.Entity, r.Book).Scan(&raw)
		default:
			return nil, &ledger.Failure{Status: 400, Code: "invalid_automatic_accounting"}
		}
		if err != nil {
			return nil, err
		}
		if !validResult(raw, op, r) {
			return nil, unavailable()
		}
		return raw, nil
	})
}

// transaction retries only known complete aborts, never an uncertain COMMIT or changed request.
func (s *Service) transaction(parent context.Context, fn func(context.Context, pgx.Tx) (json.RawMessage, error)) (json.RawMessage, error) {
	if s == nil || s.DB == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		result, err, unknown := s.attempt(ctx, fn)
		if err == nil {
			return result, nil
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
func (s *Service) attempt(ctx context.Context, fn func(context.Context, pgx.Tx) (json.RawMessage, error)) (json.RawMessage, error, bool) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
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
	raw, err := fn(ctx, tx)
	if err != nil {
		return nil, err, false
	}
	if ctx.Err() != nil {
		return nil, ctx.Err(), false
	}
	if err = tx.Commit(ctx); err != nil {
		var pg *pgconn.PgError
		unknown := !errors.As(err, &pg) && !errors.Is(err, pgx.ErrTxCommitRollback)
		if pg != nil && (pg.Code == "40003" || len(pg.Code) >= 2 && pg.Code[:2] == "08") {
			unknown = true
		}
		return nil, err, unknown
	}
	return raw, nil, false
}
func unavailable() *ledger.Failure {
	return &ledger.Failure{Status: 503, Code: "temporarily_unavailable"}
}
func safeFailure(err error) *ledger.Failure {
	var f *ledger.Failure
	if errors.As(err, &f) && f.Status == 503 {
		return unavailable()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P2A00", "22P02", "22021":
			return &ledger.Failure{Status: 400, Code: "invalid_automatic_accounting"}
		case "P2A01":
			return &ledger.Failure{Status: 404, Code: "automatic_accounting_denied"}
		case "P2A02":
			return &ledger.Failure{Status: 409, Code: "automatic_accounting_conflict"}
		case "P2A03":
			return &ledger.Failure{Status: 409, Code: "automatic_accounting_snapshot_changed"}
		case "P2A04":
			return &ledger.Failure{Status: 422, Code: "automatic_accounting_limit"}
		}
	}
	return unavailable()
}
