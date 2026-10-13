package ledger

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ReportJournal verifies pinned GL membership and reads its journal in one transaction.
// SQL owns authorization and revision checks; results require commit acknowledgement.
func (s *Service) ReportJournal(parent context.Context, c Command) (Result, error) {
	ctx, cancel := context.WithTimeout(parent, 9*time.Second)
	defer cancel()
	for attempt := 0; attempt < 2; attempt++ {
		result, err, uncertain := s.reportJournalAttempt(ctx, c)
		if err == nil {
			return result, nil
		}
		if uncertain {
			return Result{}, &Failure{503, "temporarily_unavailable"}
		}
		if !retryable(err) {
			return Result{}, financialReportFailure(err)
		}
	}
	return Result{}, &Failure{503, "temporarily_unavailable"}
}

// reportJournalAttempt never exposes a result before commit acknowledgement;
// ordinary repeatable-read permits the existing authorization row locks.
func (s *Service) reportJournalAttempt(ctx context.Context, c Command) (Result, error, bool) {
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return Result{}, err, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err = tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL lock_timeout='1s'"); err != nil {
		return Result{}, err, false
	}
	r := Result{Status: 200}
	if err = tx.QueryRow(ctx, "select reporting.report_journal($1,$2,$3,$4)", c.Actor, c.Entity, c.Book, c.Body).Scan(&r.Body); err != nil {
		return Result{}, err, false
	}
	if err = tx.Commit(ctx); err != nil {
		var p *pgconn.PgError
		return Result{}, err, !errors.As(err, &p) && !errors.Is(err, pgx.ErrTxCommitRollback)
	}
	return r, nil, false
}
