package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// transactionFault supplies only transaction behavior used by the narrow adapter.
type transactionFault struct {
	pgx.Tx
	db *faultDB
}

// faultDB counts retries and injects server-confirmed rollback or ambiguous commit.
type faultDB struct {
	begins, rollbacks int
	commitErr         error
}

// BeginTx asserts whole-transaction serializability.
func (d *faultDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable {
		return nil, errors.New("isolation")
	}
	d.begins++
	return &transactionFault{db: d}, nil
}

// Exec accepts only fixture transaction configuration.
func (tx *transactionFault) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// QueryRow returns role proof or an opaque object without real side effects.
func (tx *transactionFault) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	return faultRow{role: q == roleQuery}
}

// Commit injects the selected server or transport outcome.
func (tx *transactionFault) Commit(context.Context) error { return tx.db.commitErr }

// Rollback verifies cleanup retains a live independent deadline.
func (tx *transactionFault) Rollback(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok || ctx.Err() != nil {
		panic("unbounded cleanup")
	}
	tx.db.rollbacks++
	return nil
}

// faultRow separates role booleans from bounded SQL metadata.
type faultRow struct{ role bool }

// Scan populates exact adapter types.
func (r faultRow) Scan(dest ...any) error {
	if r.role {
		*dest[0].(*bool) = true
	} else {
		*dest[0].(*[]byte) = []byte(`{"fixture":true}`)
	}
	return nil
}

// TestSQLCommitFaults prevents replay after ambiguous commit and bounds known rollback retries.
func TestSQLCommitFaults(t *testing.T) {
	for _, tc := range []struct {
		err    error
		want   string
		begins int
	}{{errors.New("synthetic diagnostic"), "commit_unknown", 1}, {&pgconn.PgError{Code: "08007"}, "commit_unknown", 1}, {&pgconn.PgError{Code: "40003"}, "commit_unknown", 1}, {&pgconn.PgError{Code: "40001"}, "temporarily_unavailable", 3}, {&pgconn.PgError{Code: "40P01"}, "temporarily_unavailable", 3}} {
		db := &faultDB{commitErr: tc.err}
		raw, err := call(context.Background(), db, "bookkeeping_webhook", "SELECT fixture")
		if err == nil || err.Error() != tc.want || raw != nil || db.begins != tc.begins || db.rollbacks != tc.begins {
			t.Fatalf("unsafe commit classification %v begins=%d", err, db.begins)
		}
	}
}
