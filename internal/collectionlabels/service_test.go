package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

const workspace = "00000000-0000-4000-8000-000000000001"
const commandID = "00000000-0000-4000-8000-000000000002"

// fakeDB models adapter failures only; it cannot prove SQL business or permission semantics.
type fakeDB struct {
	beginErr, execErr, rowErr, commitErr error
	firstRowErr                          error
	body                                 string
	calls, commits, rollbacks            int
	options                              pgx.TxOptions
	query                                string
	args                                 []any
	deadline, cleanupDeadline            bool
}

// BeginTx records isolation and the bounded operation context.
func (d *fakeDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	d.calls++
	d.options = o
	deadline, ok := ctx.Deadline()
	d.deadline = ok && time.Until(deadline) <= 4*time.Second
	return &fakeTx{d: d}, d.beginErr
}

type fakeTx struct {
	pgx.Tx
	d *fakeDB
}

// Exec accepts only parameterized transaction-local timeout setup.
func (t *fakeTx) Exec(ctx context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	if q != "select set_config('statement_timeout',$1,true),set_config('lock_timeout',$1,true)" || len(args) != 1 {
		panic("unexpected SQL mutation")
	}
	if ctx.Err() != nil {
		return pgconn.CommandTag{}, ctx.Err()
	}
	return pgconn.CommandTag{}, t.d.execErr
}

// QueryRow captures exact trusted arguments and unchanged selector text.
func (t *fakeTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.d.query = q
	t.d.args = args
	return fakeRow{t.d}
}

// Commit supplies acknowledged success or a synthetic lost/aborted reply.
func (t *fakeTx) Commit(context.Context) error { t.d.commits++; return t.d.commitErr }

// Rollback proves cleanup has its own live bounded context even after cancellation.
func (t *fakeTx) Rollback(ctx context.Context) error {
	t.d.rollbacks++
	deadline, ok := ctx.Deadline()
	t.d.cleanupDeadline = ctx.Err() == nil && ok && time.Until(deadline) <= time.Second
	return nil
}

type fakeRow struct{ d *fakeDB }

// Scan returns fixed safe JSON or the configured SQLSTATE.
func (r fakeRow) Scan(dest ...any) error {
	if r.d.calls == 1 && r.d.firstRowErr != nil {
		return r.d.firstRowErr
	}
	if r.d.rowErr != nil {
		return r.d.rowErr
	}
	*dest[0].(*json.RawMessage) = json.RawMessage(r.d.body)
	return nil
}

// result supplies only the envelope under test, never a pretend SQL permission proof.
func result(op string) string {
	profile := "private-collection-command-v1"
	suffix := `,"command":{}`
	switch op {
	case "catalog":
		profile = "private-collection-catalog-v1"
		suffix = ""
	case "query":
		profile = "private-collection-review-v1"
		suffix = ""
	case "preview":
		profile = "private-collection-preview-v1"
		suffix = ""
	}
	return `{"profile":"` + profile + `","workspace_id":"` + workspace + `"` + suffix + `}`
}

// invoke deliberately uses the same exported adapter methods consumed by server composition.
func invoke(s *Service, ctx context.Context, r Request, op string) (json.RawMessage, error) {
	switch op {
	case "catalog":
		return s.Catalog(ctx, r)
	case "query":
		return s.Query(ctx, r)
	case "preview":
		return s.Preview(ctx, r)
	case "prepare":
		return s.Prepare(ctx, r)
	case "pending":
		return s.Pending(ctx, r)
	default:
		return s.Transition(ctx, r, op)
	}
}

// TestLabelAdapter verifies all fixed SQL signatures, isolation and exact raw-byte separation.
func TestLabelAdapter(t *testing.T) {
	for _, op := range []string{"catalog", "query", "preview", "prepare", "pending", "execute", "cancel", "acknowledge"} {
		t.Run(op, func(t *testing.T) {
			d := &fakeDB{body: result(op)}
			s := &Service{DB: d}
			r := Request{Actor: "verified-principal", Workspace: workspace, Command: commandID, Body: ` {"actor":"foreign","duplicate":1,"duplicate":2,"amount":"9007199254740993.01"} `}
			body, err := invoke(s, context.Background(), r, op)
			if err != nil || string(body) != d.body || d.calls != 1 || d.commits != 1 || d.rollbacks != 1 || !d.deadline || !d.cleanupDeadline {
				t.Fatalf("adapter lifecycle: %v", err)
			}
			isolation := pgx.Serializable
			if op == "catalog" || op == "query" || op == "pending" {
				isolation = pgx.RepeatableRead
			}
			if d.options.IsoLevel != isolation || d.options.AccessMode == pgx.ReadOnly {
				t.Fatal("wrong transaction isolation or row-lock mode")
			}
			args := []any{r.Actor, r.Workspace}
			var q string
			switch op {
			case "catalog":
				q = "select banking.get_collection_taxonomy($1::uuid,$2::uuid)"
			case "query":
				q = "select banking.query_collection_review($1::uuid,$2::uuid,$3::text)"
				args = append(args, r.Body)
			case "preview":
				q = "select banking.preview_collection_labels($1::uuid,$2::uuid,$3::text)"
				args = append(args, r.Body)
			case "prepare":
				q = "select banking.prepare_collection_label_command($1::uuid,$2::uuid,$3::text)"
				args = append(args, r.Body)
			case "pending":
				q = "select banking.pending_collection_label_command($1::uuid,$2::uuid)"
			default:
				q = "select banking.transition_collection_label_command($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)"
				args = append(args, r.Command, op, r.Body)
			}
			if d.query != q || !reflect.DeepEqual(d.args, args) {
				t.Fatal("changed SQL signature or caller bytes")
			}
		})
	}
}

// TestLabelTransactionFailures proves closed diagnostics, bounded retries and lost-commit inertness.
func TestLabelTransactionFailures(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		row, commit, begin, exec error
		code                     string
		calls                    int
	}{
		{"invalid", &pgconn.PgError{Code: "P2800", Message: "private"}, nil, nil, nil, "invalid_collection", 1},
		{"denied", &pgconn.PgError{Code: "P2801"}, nil, nil, nil, "collection_denied", 1},
		{"conflict", &pgconn.PgError{Code: "P2802"}, nil, nil, nil, "collection_conflict", 1},
		{"snapshot", &pgconn.PgError{Code: "P2803"}, nil, nil, nil, "collection_snapshot_changed", 1},
		{"not_ready", &pgconn.PgError{Code: "P2804"}, nil, nil, nil, "collection_not_ready", 1},
		{"limit", &pgconn.PgError{Code: "P2805"}, nil, nil, nil, "collection_limit", 1},
		{"abort", &pgconn.PgError{Code: "40001"}, nil, nil, nil, "temporarily_unavailable", 3},
		{"deadlock", nil, &pgconn.PgError{Code: "40P01"}, nil, nil, "temporarily_unavailable", 3},
		{"lost_commit", nil, errors.New("private reply lost"), nil, nil, "temporarily_unavailable", 1},
		{"commit_unknown", nil, &pgconn.PgError{Code: "40003"}, nil, nil, "temporarily_unavailable", 1},
		{"commit_connection", nil, &pgconn.PgError{Code: "08006"}, nil, nil, "temporarily_unavailable", 1},
		{"commit_rollback", nil, pgx.ErrTxCommitRollback, nil, nil, "temporarily_unavailable", 1},
		{"lock_timeout", &pgconn.PgError{Code: "55P03"}, nil, nil, nil, "temporarily_unavailable", 1},
		{"begin", nil, nil, context.Canceled, nil, "temporarily_unavailable", 1},
		{"timeout_setup", nil, nil, nil, errors.New("private"), "temporarily_unavailable", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &fakeDB{body: result("execute"), rowErr: tc.row, commitErr: tc.commit, beginErr: tc.begin, execErr: tc.exec}
			body, err := (&Service{DB: d}).Transition(context.Background(), Request{Workspace: workspace}, "execute")
			var f *ledger.Failure
			if !errors.As(err, &f) || f.Code != tc.code || len(body) != 0 || d.calls != tc.calls {
				t.Fatalf("unexpected safe outcome: %v calls%d", err, d.calls)
			}
			if tc.begin == nil && (d.rollbacks != tc.calls || !d.cleanupDeadline) {
				t.Fatal("missing bounded cleanup")
			}
		})
	}
	d := &fakeDB{body: result("preview"), firstRowErr: &pgconn.PgError{Code: "40001"}}
	if _, err := (&Service{DB: d}).Preview(context.Background(), Request{Workspace: workspace}); err != nil || d.calls != 2 || d.commits != 1 {
		t.Fatal("known abort did not recover once", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d = &fakeDB{body: result("query")}
	if body, err := (&Service{DB: d}).Query(ctx, Request{Workspace: workspace}); err == nil || len(body) != 0 || d.commits != 0 || !d.cleanupDeadline {
		t.Fatal("cancelled operation escaped or cleanup cancelled")
	}
	var nilService *Service
	if _, err := nilService.Query(context.Background(), Request{}); err == nil {
		t.Fatal("nil service available")
	}
	d = &fakeDB{}
	if _, err := (&Service{DB: d}).Transition(context.Background(), Request{}, "arbitrary"); err == nil || d.calls != 0 {
		t.Fatal("invalid operation reached database")
	}
}

// TestLabelResultFence rejects wrong scope/profile, oversized results and invalid command wrappers.
func TestLabelResultFence(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{}`, `invalid`, strings.Replace(result("query"), workspace, commandID, 1), result("catalog"), strings.Repeat(" ", 1<<20) + result("query")} {
		d := &fakeDB{body: body}
		got, err := (&Service{DB: d}).Query(context.Background(), Request{Workspace: workspace})
		if err == nil || len(got) != 0 || d.commits != 0 {
			t.Fatal("invalid result committed")
		}
	}
	for _, op := range []string{"prepare", "pending", "execute", "cancel", "acknowledge"} {
		for _, command := range []string{`null`, `[]`, `"opaque"`, `false`, `{}`} {
			body := strings.Replace(result(op), `"command":{}`, `"command":`+command, 1)
			want := command == `{}` || (command == `null` && op == "pending")
			if validResult(json.RawMessage(body), op, workspace) != want {
				t.Fatalf("wrong command-null fence %s %s", op, command)
			}
		}
		if validResult(json.RawMessage(strings.Replace(result(op), `,"command":{}`, "", 1)), op, workspace) {
			t.Fatal("missing command accepted")
		}
	}
}
