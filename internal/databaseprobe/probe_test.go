package databaseprobe

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeTx struct {
	allowedValue          bool
	queryErr, rollbackErr error
	rolled                bool
}

// allowed models catalog-only acceptance without a real identity or network.
func (f *fakeTx) allowed(context.Context, string) (bool, error) { return f.allowedValue, nil }

// ExecContext models exact PostgreSQL denial rather than treating any error as sufficient.
func (f *fakeTx) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, f.queryErr
}

// Rollback records cleanup even when a mutation unexpectedly succeeds.
func (f *fakeTx) Rollback() error { f.rolled = true; return f.rollbackErr }

// TestProbeOutcomes ensures unrelated failures and accidental success never become authorization proof.
func TestProbeOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		failAt                int
		queryErr, rollbackErr error
		catalog               bool
		wantOK                bool
	}{
		{"pass", -1, nil, nil, true, true},
		{"unexpected-success", 1, nil, nil, true, false},
		{"missing-table", 1, &pgconn.PgError{Code: "42P01", Message: "protected"}, nil, true, false},
		{"read-only", 1, &pgconn.PgError{Code: "25006"}, nil, true, false},
		{"transport", 1, errors.New("protected"), nil, true, false},
		{"wrong-isolation", len(probes), &pgconn.PgError{Code: "P2100"}, nil, true, false},
		{"rollback-failure", 1, &pgconn.PgError{Code: "42501"}, errors.New("protected"), true, false},
		{"unsafe-catalog", -1, nil, nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var txs []*fakeTx
			err := run(context.Background(), func(context.Context) (transaction, error) {
				i := len(txs)
				f := &fakeTx{allowedValue: tc.catalog}
				if i > 0 {
					f.queryErr = &pgconn.PgError{Code: probes[i-1].state}
				}
				if i == tc.failAt {
					f.queryErr = tc.queryErr
					f.rollbackErr = tc.rollbackErr
				}
				txs = append(txs, f)
				return f, nil
			}, "synthetic_api")
			if (err == nil) != tc.wantOK {
				t.Fatal("incorrect acceptance")
			}
			if err != nil && err.Error() != "database permission proof failed" {
				t.Fatal("error disclosure")
			}
			for _, tx := range txs {
				if !tx.rolled {
					t.Fatal("transaction not rolled back")
				}
			}
		})
	}
}

// TestConnectionFailure confirms connection failures cannot count as permission denial.
func TestConnectionFailure(t *testing.T) {
	if err := run(context.Background(), func(context.Context) (transaction, error) { return nil, errors.New("protected") }, "synthetic_api"); err != ErrProbe {
		t.Fatal("connection failure accepted or exposed")
	}
}
