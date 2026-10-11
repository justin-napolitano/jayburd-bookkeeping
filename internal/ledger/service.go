// Package ledger calls the restricted PostgreSQL accounting authority. It does
// not calculate balances, normalize financial requests, or retrieve credentials.
package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Beginner lets a caller supply its owned pool without exposing connection details.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Service owns transaction orchestration; the caller owns the connection lifecycle.
type Service struct{ DB Beginner }

// ResolvePrincipal reads only the identity mapping; every operation reauthorizes
// under its own transaction locks. Unknown and disabled identities resolve empty.
func (s *Service) ResolvePrincipal(ctx context.Context, issuer, subject string) (string, error) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", &Failure{503, "temporarily_unavailable"}
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	var id *string
	err = tx.QueryRow(ctx, "select ledger.resolve_principal($1,$2)", issuer, subject).Scan(&id)
	if err != nil {
		return "", failure(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return "", failure(err)
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

// Setup creates a new development entity/book bundle without accepting an actor
// or permissions from its request body; SQL owns atomic setup and receipt replay.
func (s *Service) Setup(ctx context.Context, c Command) (Result, error) {
	return s.execute(ctx, c, "setup")
}

// Command carries trusted identity separately from unchanged user JSON text.
type Command struct {
	Actor, Entity, Book, Journal, Key, Request string
	Body                                       string
}

// Result is returned only after a successful commit, including on durable replay.
type Result struct {
	Status   int
	Body     json.RawMessage
	Replayed bool
}

// Failure is the complete safe error vocabulary; raw database details never escape.
type Failure struct {
	Status int
	Code   string
}

// Error deliberately contains only the stable public code.
func (e *Failure) Error() string { return e.Code }

// failure maps database errors without retaining identifiers, SQL or raw payloads.
func failure(err error) *Failure {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		codes := map[string]Failure{
			"P2100": {400, "invalid_request"}, "P2101": {404, "not_found"}, "P2102": {403, "permission_denied"},
			"P2103": {409, "revision_conflict"}, "P2104": {409, "idempotency_conflict"}, "P2105": {409, "period_closed"},
			"P2106": {409, "already_reversed"}, "P2107": {422, "invalid_amount"}, "P2108": {422, "invalid_journal"},
			"P2109": {422, "unsupported_policy"}, "P2110": {422, "result_out_of_range"}, "P2111": {413, "request_too_large"},
			"22P02": {400, "invalid_request"}, "22021": {400, "invalid_request"}, "40001": {503, "temporarily_unavailable"},
			"40P01": {503, "temporarily_unavailable"}, "57014": {503, "temporarily_unavailable"}, "55P03": {503, "temporarily_unavailable"},
		}
		if strings.HasPrefix(p.Code, "08") || p.Code == "57P01" || p.Code == "57P02" || p.Code == "57P03" {
			return &Failure{503, "temporarily_unavailable"}
		}
		if f, ok := codes[p.Code]; ok {
			return &f
		}
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return &Failure{503, "temporarily_unavailable"}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &Failure{503, "temporarily_unavailable"}
	}
	return &Failure{500, "internal_error"}
}

// retryable covers only errors that require replaying the entire transaction.
func retryable(err error) bool {
	var p *pgconn.PgError
	return errors.As(err, &p) && (p.Code == "40001" || p.Code == "40P01")
}

// Post submits a manual identity/spot journal to the SQL authority.
func (s *Service) Post(ctx context.Context, c Command) (Result, error) {
	return s.execute(ctx, c, "post")
}

// Reverse copies historical values in SQL; callers cannot supply replacement lines.
func (s *Service) Reverse(ctx context.Context, c Command) (Result, error) {
	return s.execute(ctx, c, "reverse")
}

// Journal reads one immutable journal under current scoped authorization.
func (s *Service) Journal(ctx context.Context, c Command) (Result, error) {
	return s.execute(ctx, c, "journal")
}

// TrialBalance returns a consistent cumulative snapshot; Body carries the date.
func (s *Service) TrialBalance(ctx context.Context, c Command) (Result, error) {
	return s.execute(ctx, c, "balance")
}

// execute reserves one second of the five-second budget for rollback cleanup
// and never acknowledges an uncertain commit.
func (s *Service) execute(parent context.Context, c Command, operation string) (Result, error) {
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	iso := pgx.Serializable
	if operation == "journal" || operation == "balance" {
		iso = pgx.RepeatableRead
	}
	for attempt := 0; attempt < 3; attempt++ {
		result, err, commitUnknown := s.attempt(ctx, c, operation, iso)
		if err == nil {
			return result, nil
		}
		if commitUnknown {
			return Result{}, &Failure{503, "temporarily_unavailable"}
		}
		if !retryable(err) {
			return Result{}, failure(err)
		}
		if attempt < 2 {
			timer := time.NewTimer(time.Duration(5+rand.IntN(20)) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return Result{}, failure(ctx.Err())
			case <-timer.C:
			}
		}
	}
	return Result{}, &Failure{503, "temporarily_unavailable"}
}

// attempt rolls back on every non-commit exit, including cancellation; its result
// stays private until COMMIT has been acknowledged by PostgreSQL.
func (s *Service) attempt(ctx context.Context, c Command, op string, iso pgx.TxIsoLevel) (Result, error, bool) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: iso})
	if err != nil {
		return Result{}, err, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	deadline, _ := ctx.Deadline()
	remaining := time.Until(deadline).Milliseconds()
	if remaining < 1 {
		return Result{}, context.DeadlineExceeded, false
	}
	_, err = tx.Exec(ctx, "select set_config('statement_timeout',$1,true),set_config('lock_timeout',$1,true)", (time.Duration(remaining) * time.Millisecond).String())
	if err != nil {
		return Result{}, err, false
	}
	var r Result
	switch op {
	case "setup":
		err = tx.QueryRow(ctx, "select * from ledger.setup_book($1,$2,$3,$4)", c.Actor, c.Key, c.Request, c.Body).Scan(&r.Status, &r.Body, &r.Replayed)
	case "post":
		err = tx.QueryRow(ctx, "select * from ledger.post_journal($1,$2,$3,$4,$5,$6)", c.Actor, c.Entity, c.Book, c.Key, c.Request, c.Body).Scan(&r.Status, &r.Body, &r.Replayed)
	case "reverse":
		err = tx.QueryRow(ctx, "select * from ledger.reverse_journal($1,$2,$3,$4,$5,$6,$7)", c.Actor, c.Entity, c.Book, c.Journal, c.Key, c.Request, c.Body).Scan(&r.Status, &r.Body, &r.Replayed)
	case "journal":
		r.Status = 200
		err = tx.QueryRow(ctx, "select reporting.get_journal($1,$2,$3,$4)", c.Actor, c.Entity, c.Book, c.Journal).Scan(&r.Body)
	case "balance":
		r.Status = 200
		err = tx.QueryRow(ctx, "select reporting.trial_balance($1,$2,$3,$4)", c.Actor, c.Entity, c.Book, c.Body).Scan(&r.Body)
	}
	if err != nil {
		return Result{}, err, false
	}
	err = tx.Commit(ctx)
	if err != nil {
		var p *pgconn.PgError
		return Result{}, err, !errors.As(err, &p) && !errors.Is(err, pgx.ErrTxCommitRollback)
	}
	return r, nil, false
}
