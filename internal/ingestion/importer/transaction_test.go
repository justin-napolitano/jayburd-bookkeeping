package importer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeDB scripts failures at transaction boundaries without touching external systems.
type fakeDB struct {
	commits           []error
	begins, rollbacks int
	queryErr          error
	active            bool
	cleanupOK         bool
}

// BeginTx verifies production isolation and returns an independently scoped test transaction.
func (d *fakeDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable {
		panic("wrong isolation")
	}
	d.begins++
	d.active = true
	return &fakeTx{db: d}, nil
}

// fakeTx embeds unused pgx operations so unexpected adapter calls fail loudly.
type fakeTx struct {
	pgx.Tx
	db *fakeDB
}

// Exec accepts only the production timeout setup.
func (t *fakeTx) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	if q != "SET LOCAL statement_timeout='3s'; SET LOCAL lock_timeout='1s'; SET LOCAL idle_in_transaction_session_timeout='4s'" {
		panic("unexpected SQL")
	}
	return pgconn.CommandTag{}, nil
}

// QueryRow returns one bounded synthetic JSON envelope.
func (t *fakeTx) QueryRow(context.Context, string, ...any) pgx.Row { return fakeRow{t.db.queryErr} }

// Commit injects a distinct fault for each transaction attempt.
func (t *fakeTx) Commit(context.Context) error {
	t.db.active = false
	if len(t.db.commits) >= t.db.begins {
		return t.db.commits[t.db.begins-1]
	}
	return nil
}

// Rollback proves independent cleanup remains bounded even after the caller expires.
func (t *fakeTx) Rollback(ctx context.Context) error {
	t.db.rollbacks++
	t.db.active = false
	deadline, ok := ctx.Deadline()
	t.db.cleanupOK = ok && time.Until(deadline) > 0 && time.Until(deadline) <= time.Second
	return nil
}

// fakeRow preserves only SQL output, never a real financial payload.
type fakeRow struct{ err error }

// Scan models the one-column JSON contract.
func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*[]byte) = []byte(`{"state":"receiving"}`)
	return nil
}

// TestSQLCommitBoundary proves ambiguous COMMIT never replays or exposes its result.
func TestSQLCommitBoundary(t *testing.T) {
	for _, code := range []string{"08007", "40003", "57P01"} {
		t.Run(code, func(t *testing.T) {
			db := &fakeDB{commits: []error{&pgconn.PgError{Code: code, Message: "private raw detail"}}}
			body, err := (&SQLRepository{db}).call(context.Background(), "select synthetic()")
			if len(body) != 0 || err == nil || err.Error() != "commit_unknown" || db.begins != 1 || db.rollbacks != 1 || !db.cleanupOK {
				t.Fatalf("unsafe boundary: body=%s err=%v attempts=%d", body, err, db.begins)
			}
		})
	}
}

// TestSQLWholeTransactionRetry proves confirmed rollbacks retry completely and stay capped.
func TestSQLWholeTransactionRetry(t *testing.T) {
	rollback := &pgconn.PgError{Code: "40001"}
	db := &fakeDB{commits: []error{rollback, rollback, nil}}
	body, err := (&SQLRepository{db}).call(context.Background(), "select synthetic()")
	if err != nil || len(body) == 0 || db.begins != 3 || db.rollbacks != 3 {
		t.Fatalf("retry failed: %v", err)
	}
	db = &fakeDB{commits: []error{rollback, rollback, rollback}}
	body, err = (&SQLRepository{db}).call(context.Background(), "select synthetic()")
	if err == nil || len(body) != 0 || db.begins != 3 {
		t.Fatal("unbounded or exposed retry")
	}
}

// TestSQLFailureRedaction verifies pre-COMMIT faults never leak details or retry unsafely.
func TestSQLFailureRedaction(t *testing.T) {
	db := &fakeDB{queryErr: errors.New("private path and CSV value")}
	body, err := (&SQLRepository{db}).call(context.Background(), "select synthetic()")
	if err == nil || err.Error() != "temporarily_unavailable" || len(body) != 0 || db.begins != 1 || !db.cleanupOK {
		t.Fatal("unsafe query failure")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db = &fakeDB{}
	_, err = (&SQLRepository{db}).call(ctx, "select synthetic()")
	if err == nil || db.begins != 0 {
		t.Fatal("cancelled request reached SQL")
	}
}
