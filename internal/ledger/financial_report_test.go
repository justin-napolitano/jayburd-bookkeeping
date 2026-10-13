package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// reportDBStub isolates transaction acknowledgement/retry behavior from SQL tests.
type reportDBStub struct {
	calls                         int
	beginErr, errorRow, commitErr error
	options                       pgx.TxOptions
	raw                           string
}

// BeginTx records the actual adapter isolation and supplies one controlled transaction.
func (d *reportDBStub) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	d.calls++
	d.options = o
	return &reportTxStub{db: d}, d.beginErr
}

// reportTxStub embeds unused methods so accidental adapter expansion fails visibly.
type reportTxStub struct {
	pgx.Tx
	db *reportDBStub
}

// Exec accepts only the adapter timeout configuration in this narrow unit fixture.
func (t *reportTxStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// QueryRow retains raw input to detect accidental request normalization in Go.
func (t *reportTxStub) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	t.db.raw = args[3].(string)
	return reportRowStub{err: t.db.errorRow}
}

// Commit simulates success, retryable database rollback, or lost acknowledgement.
func (t *reportTxStub) Commit(context.Context) error { return t.db.commitErr }

// Rollback supplies bounded cleanup without a real database connection.
func (t *reportTxStub) Rollback(context.Context) error { return nil }

// reportRowStub represents the restricted SQL response or failure.
type reportRowStub struct{ err error }

// Scan returns fixed JSON only when the synthetic database operation succeeded.
func (r reportRowStub) Scan(dest ...any) error {
	if r.err == nil {
		*dest[0].(*json.RawMessage) = json.RawMessage(`{"totals":{}}`)
	}
	return r.err
}

// TestFinancialReportAdapter proves SQL remains authoritative and commit loss leaks no body.
func TestFinancialReportAdapter(t *testing.T) {
	for _, tc := range []struct {
		name               string
		row, commit, begin error
		code               string
		calls              int
	}{
		{name: "success", calls: 1}, {name: "snapshot", row: &pgconn.PgError{Code: "P2114"}, code: "report_snapshot_changed", calls: 1},
		{name: "retry_limit", row: &pgconn.PgError{Code: "40001"}, code: "temporarily_unavailable", calls: 2},
		{name: "commit_lost", commit: errors.New("private transport detail"), code: "temporarily_unavailable", calls: 1},
		{name: "begin", begin: context.Canceled, code: "temporarily_unavailable", calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &reportDBStub{errorRow: tc.row, commitErr: tc.commit, beginErr: tc.begin}
			s := Service{DB: d}
			raw := `{"report":"trial-balance","report":"invalid"}`
			r, err := s.FinancialReport(context.Background(), Command{Body: raw})
			if d.calls != tc.calls || d.options.IsoLevel != pgx.RepeatableRead || d.options.AccessMode == pgx.ReadOnly {
				t.Fatalf("unsafe transaction: %+v", d.options)
			}
			if tc.code != "" {
				var f *Failure
				if !errors.As(err, &f) || f.Code != tc.code || len(r.Body) != 0 {
					t.Fatalf("failure/body: %v %s", err, r.Body)
				}
			} else if err != nil || r.Status != 200 || d.raw != raw {
				t.Fatal("adapter changed boundary", err)
			}
		})
	}
}
