package collectionread

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// testDB isolates transport/transaction behavior; real role/snapshot proofs use disposable SQL.
type testDB struct {
	beginErr, rowErr, commitErr error
	body                        string
	calls, commits, rollbacks   int
	options                     pgx.TxOptions
	query                       string
	args                        []any
}

// BeginTx records actual isolation and exposes only methods this adapter needs.
func (d *testDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	d.calls++
	d.options = o
	return &testTx{d: d}, d.beginErr
}

// testTx deliberately embeds unused methods so accidental adapter expansion fails visibly.
type testTx struct {
	pgx.Tx
	d *testDB
}

// Exec accepts timeout setup only, without any product mutation or external effect.
func (t *testTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// QueryRow captures the fixed entrypoint and trusted actor separately from unchanged JSON.
func (t *testTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.d.query = q
	t.d.args = args
	return testRow{t.d}
}

// Commit simulates exact acknowledgement or an uncertain reply.
func (t *testTx) Commit(context.Context) error { t.d.commits++; return t.d.commitErr }

// Rollback records cleanup on every exit, including failed/unknown commits.
func (t *testTx) Rollback(context.Context) error { t.d.rollbacks++; return nil }

// testRow is a fixed synthetic JSON result or SQLSTATE error.
type testRow struct{ d *testDB }

// Scan returns only the configured fixture without interpreting amounts.
func (r testRow) Scan(dest ...any) error {
	if r.d.rowErr == nil {
		*dest[0].(*json.RawMessage) = json.RawMessage(r.d.body)
	}
	return r.d.rowErr
}

// TestCollectionReadAdapter preserves the exact request/body and acknowledges only after COMMIT.
func TestCollectionReadAdapter(t *testing.T) {
	for _, op := range []string{"workspace", "sources", "observations"} {
		profile := map[string]string{"workspace": "private-source-workspace-v1", "sources": "private-source-page-v1", "observations": "private-source-query-v1"}[op]
		d := &testDB{body: `{"profile":"` + profile + `","amount":"9007199254740993.01"}`}
		s := &Service{DB: d}
		r := Request{Actor: "trusted", Workspace: workspace, Body: ` {"search":"one","search":"two"} `}
		var body json.RawMessage
		var err error
		switch op {
		case "workspace":
			body, err = s.Workspace(context.Background(), r)
		case "sources":
			body, err = s.Sources(context.Background(), r)
		case "observations":
			body, err = s.Observations(context.Background(), r)
		}
		if err != nil || string(body) != d.body || d.calls != 1 || d.commits != 1 || d.rollbacks != 1 || d.options.IsoLevel != pgx.RepeatableRead || d.options.AccessMode == pgx.ReadOnly {
			t.Fatalf("unsafe adapter %s: %v", op, err)
		}
		expected := []any{r.Actor}
		if op != "workspace" {
			expected = append(expected, r.Workspace, r.Body)
		}
		if !reflect.DeepEqual(d.args, expected) {
			t.Fatal("adapter changed trusted actor or raw selector")
		}
		q := map[string]string{"workspace": "select banking.get_collection_workspace($1::uuid)", "sources": "select banking.list_collection_sources($1::uuid,$2::uuid,$3::text)", "observations": "select banking.query_collection_observations($1::uuid,$2::uuid,$3::text)"}[op]
		if d.query != q {
			t.Fatal("unexpected SQL entrypoint")
		}
	}
}

// TestCollectionReadFailure tests denial/snapshot/limit, bounded abort retry and unknown commit loss.
func TestCollectionReadFailure(t *testing.T) {
	for _, tc := range []struct {
		name               string
		row, commit, begin error
		body, code         string
		calls              int
	}{
		{name: "denied", row: &pgconn.PgError{Code: "P2801", Message: "private"}, code: "collection_denied", calls: 1},
		{name: "snapshot", row: &pgconn.PgError{Code: "P2803"}, code: "collection_snapshot_changed", calls: 1},
		{name: "invalid", row: &pgconn.PgError{Code: "P2800"}, code: "invalid_collection", calls: 1},
		{name: "limit", row: &pgconn.PgError{Code: "P2805"}, code: "collection_limit", calls: 1},
		{name: "abort", row: &pgconn.PgError{Code: "40001"}, code: "temporarily_unavailable", calls: 3},
		{name: "deadlock", commit: &pgconn.PgError{Code: "40P01"}, body: `{"profile":"private-source-query-v1"}`, code: "temporarily_unavailable", calls: 3},
		{name: "lost_commit", commit: errors.New("private lost reply"), body: `{"profile":"private-source-query-v1"}`, code: "temporarily_unavailable", calls: 1},
		{name: "unknown_commit", commit: &pgconn.PgError{Code: "40003"}, body: `{"profile":"private-source-query-v1"}`, code: "temporarily_unavailable", calls: 1},
		{name: "connection_commit", commit: &pgconn.PgError{Code: "08006"}, body: `{"profile":"private-source-query-v1"}`, code: "temporarily_unavailable", calls: 1},
		{name: "invalid_result", body: `private malformed`, code: "temporarily_unavailable", calls: 1},
		{name: "null_result", body: `null`, code: "temporarily_unavailable", calls: 1},
		{name: "array_result", body: `[]`, code: "temporarily_unavailable", calls: 1},
		{name: "wrong_profile", body: `{"profile":"private-source-workspace-v1"}`, code: "temporarily_unavailable", calls: 1},
		{name: "begin", begin: context.Canceled, code: "temporarily_unavailable", calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &testDB{rowErr: tc.row, commitErr: tc.commit, beginErr: tc.begin, body: tc.body}
			s := &Service{DB: d}
			body, err := s.Observations(context.Background(), Request{Actor: "trusted", Workspace: workspace, Body: `{}`})
			var f *ledger.Failure
			if !errors.As(err, &f) || f.Code != tc.code || len(body) != 0 || d.calls != tc.calls {
				t.Fatalf("bad failure: calls%d error%v", d.calls, err)
			}
		})
	}
	var s *Service
	if body, err := s.Workspace(context.Background(), Request{}); len(body) != 0 || err == nil {
		t.Fatal("nil adapter available")
	}
}
