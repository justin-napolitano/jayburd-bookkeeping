// Package collectionlabels transports SQL-owned private query and reviewed label commands.
// It never derives authority, taxonomy decisions or monetary values from caller JSON.
package collectionlabels

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

// Request keeps independently verified identity and path selectors separate from unchanged JSON.
type Request struct{ Actor, Workspace, Command, Body string }

// Operations exposes only the frozen restricted SQL query and command surface.
type Operations interface {
	Catalog(context.Context, Request) (json.RawMessage, error)
	Query(context.Context, Request) (json.RawMessage, error)
	Preview(context.Context, Request) (json.RawMessage, error)
	Prepare(context.Context, Request) (json.RawMessage, error)
	Pending(context.Context, Request) (json.RawMessage, error)
	Transition(context.Context, Request, string) (json.RawMessage, error)
}

// Service uses a caller-composed restricted pool; SQL owns every business invariant.
type Service struct{ DB ledger.Beginner }

// Catalog reads the complete bounded authorized catalog and source aliases.
func (s *Service) Catalog(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "catalog")
}

// Query reads one current revision-pinned page without interpreting money or filters.
func (s *Service) Query(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "query")
}

// Preview persists a bounded inert server preview in a serializable transaction.
func (s *Service) Preview(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "preview")
}

// Prepare persists the exact reviewed intent using SQL-owned idempotency semantics.
func (s *Service) Prepare(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "prepare")
}

// Pending discovers existing work without executing or acknowledging it.
func (s *Service) Pending(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "pending")
}

// Transition accepts only explicit execute, cancel or acknowledge operations.
func (s *Service) Transition(ctx context.Context, r Request, operation string) (json.RawMessage, error) {
	if operation != "execute" && operation != "cancel" && operation != "acknowledge" {
		return nil, &ledger.Failure{Status: 400, Code: "invalid_collection"}
	}
	return s.run(ctx, r, operation)
}

// run retries whole known-aborted transactions only; uncertain commit never causes another call.
func (s *Service) run(parent context.Context, r Request, operation string) (json.RawMessage, error) {
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

// attempt acknowledges COMMIT before returning a private result and cleans up on every exit.
func (s *Service) attempt(ctx context.Context, r Request, operation string) (json.RawMessage, error, bool) {
	isolation := pgx.Serializable
	if operation == "catalog" || operation == "query" || operation == "pending" {
		isolation = pgx.RepeatableRead
	}
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: isolation})
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
		err = tx.QueryRow(ctx, "select banking.get_collection_taxonomy($1::uuid,$2::uuid)", r.Actor, r.Workspace).Scan(&body)
	case "query":
		err = tx.QueryRow(ctx, "select banking.query_collection_review($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
	case "preview":
		err = tx.QueryRow(ctx, "select banking.preview_collection_labels($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
	case "prepare":
		err = tx.QueryRow(ctx, "select banking.prepare_collection_label_command($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
	case "pending":
		err = tx.QueryRow(ctx, "select banking.pending_collection_label_command($1::uuid,$2::uuid)", r.Actor, r.Workspace).Scan(&body)
	case "execute", "cancel", "acknowledge":
		err = tx.QueryRow(ctx, "select banking.transition_collection_label_command($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)", r.Actor, r.Workspace, r.Command, operation, r.Body).Scan(&body)
	default:
		return nil, unavailable(), false
	}
	if err != nil {
		return nil, err, false
	}
	if !validResult(body, operation, r.Workspace) {
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

// validResult fences profile, workspace, command-nullability and budget, leaving business fields to SQL.
func validResult(body json.RawMessage, operation, workspace string) bool {
	if len(body) > 1<<20 || !json.Valid(body) {
		return false
	}
	var envelope struct {
		Profile   string          `json:"profile"`
		Workspace string          `json:"workspace_id"`
		Command   json.RawMessage `json:"command"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Workspace != workspace {
		return false
	}
	profile := "private-collection-command-v1"
	switch operation {
	case "catalog":
		profile = "private-collection-catalog-v1"
	case "query":
		profile = "private-collection-review-v1"
	case "preview":
		profile = "private-collection-preview-v1"
	case "prepare", "pending", "execute", "cancel", "acknowledge":
		command := strings.TrimSpace(string(envelope.Command))
		if command == "null" {
			if operation != "pending" {
				return false
			}
		} else if len(command) < 2 || command[0] != '{' {
			return false
		}
	default:
		return false
	}
	return envelope.Profile == profile
}

// unavailable discards all database, transport and private identifier diagnostics.
func unavailable() *ledger.Failure {
	return &ledger.Failure{Status: 503, Code: "temporarily_unavailable"}
}

// safeFailure translates only the fixed SQLSTATE vocabulary; unknown errors fail closed.
func safeFailure(err error) *ledger.Failure {
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
