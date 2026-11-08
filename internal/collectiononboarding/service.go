// Package collectiononboarding transports explicit own-workspace setup and reviewed adoption.
// PostgreSQL owns eligibility, current authority, immutable intent and all effects.
package collectiononboarding

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

// Request keeps verified identity separate from unchanged caller JSON and path identity.
type Request struct{ Actor, Command, Body string }

// Operations is the bounded mounted onboarding port, independent of operator activation plans.
type Operations interface {
	Catalog(context.Context, Request) (json.RawMessage, error)
	Preview(context.Context, Request) (json.RawMessage, error)
	Prepare(context.Context, Request) (json.RawMessage, error)
	Pending(context.Context, Request) (json.RawMessage, error)
	Transition(context.Context, Request, string) (json.RawMessage, error)
	Clear(context.Context, Request) (json.RawMessage, error)
}

// Service uses only the restricted API pool supplied by managed composition.
type Service struct{ DB ledger.Beginner }

// Catalog discovers current eligibility without creating a workspace or adopting sources.
func (s *Service) Catalog(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.execute(ctx, r, "catalog")
}

// Preview stores only the server-reviewed inert intent resolved by SQL.
func (s *Service) Preview(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.execute(ctx, r, "preview")
}

// Prepare reserves the existing principal gate for an immutable preview/key.
func (s *Service) Prepare(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.execute(ctx, r, "prepare")
}

// Pending discovers global onboarding work even before any workspace exists.
func (s *Service) Pending(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.execute(ctx, r, "pending")
}

// Clear acknowledges only the exact inaccessible onboarding identity in SQL.
func (s *Service) Clear(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.execute(ctx, r, "clear")
}

// Transition executes only explicit lifecycle operations on the path command.
func (s *Service) Transition(ctx context.Context, r Request, operation string) (json.RawMessage, error) {
	if operation != "execute" && operation != "cancel" && operation != "acknowledge" {
		return nil, &ledger.Failure{Status: 400, Code: "invalid_onboarding"}
	}
	return s.execute(ctx, r, operation)
}

// execute retries known whole-transaction aborts only; no uncertain commit is automatically repeated.
func (s *Service) execute(parent context.Context, r Request, operation string) (json.RawMessage, error) {
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

// attempt keeps every read/admission under SERIALIZABLE current fences and commits before exposure.
func (s *Service) attempt(ctx context.Context, r Request, operation string) (json.RawMessage, error, bool) {
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
	var body json.RawMessage
	switch operation {
	case "catalog":
		err = tx.QueryRow(ctx, "select banking.get_collection_onboarding($1::uuid)", r.Actor).Scan(&body)
	case "preview":
		err = tx.QueryRow(ctx, "select banking.preview_collection_onboarding($1::uuid,$2::text)", r.Actor, r.Body).Scan(&body)
	case "prepare":
		err = tx.QueryRow(ctx, "select banking.prepare_collection_onboarding($1::uuid,$2::text)", r.Actor, r.Body).Scan(&body)
	case "pending":
		err = tx.QueryRow(ctx, "select banking.pending_collection_onboarding($1::uuid)", r.Actor).Scan(&body)
	case "clear":
		err = tx.QueryRow(ctx, "select banking.clear_collection_onboarding($1::uuid,$2::text)", r.Actor, r.Body).Scan(&body)
	default:
		err = tx.QueryRow(ctx, "select banking.transition_collection_onboarding($1::uuid,$2::uuid,$3::text,$4::text)", r.Actor, r.Command, operation, r.Body).Scan(&body)
	}
	if err != nil {
		return nil, err, false
	}
	if !validResult(body, operation, r) {
		return nil, unavailable(), false
	}
	if ctx.Err() != nil {
		return nil, ctx.Err(), false
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
func unavailable() *ledger.Failure {
	return &ledger.Failure{Status: 503, Code: "temporarily_unavailable"}
}

// safeFailure is the complete SQLSTATE vocabulary; arbitrary errors never enter HTTP diagnostics.
func safeFailure(err error) *ledger.Failure {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P2800", "22P02", "22021":
			return &ledger.Failure{Status: 400, Code: "invalid_onboarding"}
		case "P2801":
			return &ledger.Failure{Status: 404, Code: "onboarding_denied"}
		case "P2802":
			return &ledger.Failure{Status: 409, Code: "onboarding_conflict"}
		case "P2803":
			return &ledger.Failure{Status: 409, Code: "onboarding_snapshot_changed"}
		case "P2804":
			return &ledger.Failure{Status: 409, Code: "onboarding_not_ready"}
		case "P2805":
			return &ledger.Failure{Status: 422, Code: "onboarding_limit"}
		}
	}
	return unavailable()
}
