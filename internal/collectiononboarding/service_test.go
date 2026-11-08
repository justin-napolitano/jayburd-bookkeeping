package collectiononboarding

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"reflect"
	"strings"
	"testing"
	"time"
)

const commandID = "00000000-0000-4000-8000-000000000002"
const previewID = "00000000-0000-4000-8000-000000000003"
const recoveryID = "00000000-0000-4000-8000-000000000004"
const workspaceID = "00000000-0000-4000-8000-000000000005"
const candidateID = "00000000-0000-4000-8000-000000000006"
const sourceID = "00000000-0000-4000-8000-000000000007"
const digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func setupPreview() string {
	return `{"profile":"private-onboarding-preview-v1","preview_id":"` + previewID + `","operation":"setup","expires_at":"2026-11-08T12:00:00Z","workspace_id":null,"snapshot":"` + digest + `","review":{"capabilities":["read","acquire","classify"]}}`
}
func prepared() string {
	return `{"profile":"private-onboarding-command-v1","command_id":"` + commandID + `","recovery_id":"` + recoveryID + `","prepare_key":"same-key","state":"prepared","preview":` + setupPreview() + `,"result":null,"acknowledged":false}`
}
func committed() string {
	return strings.Replace(strings.Replace(prepared(), `"state":"prepared"`, `"state":"committed"`, 1), `"result":null`, `"result":{"workspace_id":"`+workspaceID+`","workspace_revision":"1","capabilities":["acquire","classify","read"]}`, 1)
}
func result(operation string) string {
	switch operation {
	case "catalog":
		return `{"profile":"private-onboarding-catalog-v1","workspace":{"state":"absent"},"can_setup":true,"can_adopt":false,"candidates":[]}`
	case "preview":
		return setupPreview()
	case "pending":
		return `{"profile":"private-onboarding-pending-v1","state":"available","command":` + prepared() + `,"recovery_id":"` + recoveryID + `"}`
	case "execute":
		return committed()
	case "cancel":
		return strings.Replace(strings.Replace(prepared(), `"state":"prepared"`, `"state":"cancelled"`, 1), `"result":null`, `"result":{"cancelled":true}`, 1)
	case "acknowledge":
		return strings.Replace(committed(), `"acknowledged":false`, `"acknowledged":true`, 1)
	case "clear":
		return `{"profile":"private-onboarding-clear-v1","recovery_id":"` + recoveryID + `","cleared":true}`
	default:
		return prepared()
	}
}
func request(operation string) Request {
	r := Request{Actor: "verified-principal", Command: commandID, Body: "{}"}
	switch operation {
	case "preview":
		r.Body = ` {"operation":"setup","actor":"untrusted","duplicate":1,"duplicate":2} `
	case "prepare":
		r.Body = ` {"preview_id":"` + previewID + `","prepare_key":"same-key"} `
	case "clear":
		r.Body = `{"recovery_id":"` + recoveryID + `"}`
	}
	return r
}
func invoke(s *Service, ctx context.Context, r Request, operation string) (json.RawMessage, error) {
	switch operation {
	case "catalog":
		return s.Catalog(ctx, r)
	case "preview":
		return s.Preview(ctx, r)
	case "prepare":
		return s.Prepare(ctx, r)
	case "pending":
		return s.Pending(ctx, r)
	case "clear":
		return s.Clear(ctx, r)
	default:
		return s.Transition(ctx, r, operation)
	}
}

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

// TestOnboardingAdapter proves the fixed signatures, original bytes and post-commit response boundary.
func TestOnboardingAdapter(t *testing.T) {
	for _, op := range []string{"catalog", "preview", "prepare", "pending", "clear", "execute", "cancel", "acknowledge"} {
		t.Run(op, func(t *testing.T) {
			d := &fakeDB{body: result(op)}
			r := request(op)
			body, e := invoke(&Service{DB: d}, context.Background(), r, op)
			if e != nil || string(body) != d.body || d.calls != 1 || d.commits != 1 || d.rollbacks != 1 || !d.deadline || !d.cleanupDeadline {
				t.Fatalf("lifecycle %v", e)
			}
			if d.options.IsoLevel != pgx.Serializable || d.options.AccessMode == pgx.ReadOnly {
				t.Fatal("current fence needs SERIALIZABLE row locks")
			}
			q := ""
			args := []any{r.Actor}
			switch op {
			case "catalog":
				q = "select banking.get_collection_onboarding($1::uuid)"
			case "preview":
				q = "select banking.preview_collection_onboarding($1::uuid,$2::text)"
				args = append(args, r.Body)
			case "prepare":
				q = "select banking.prepare_collection_onboarding($1::uuid,$2::text)"
				args = append(args, r.Body)
			case "pending":
				q = "select banking.pending_collection_onboarding($1::uuid)"
			case "clear":
				q = "select banking.clear_collection_onboarding($1::uuid,$2::text)"
				args = append(args, r.Body)
			default:
				q = "select banking.transition_collection_onboarding($1::uuid,$2::uuid,$3::text,$4::text)"
				args = append(args, r.Command, op, r.Body)
			}
			if d.query != q || !reflect.DeepEqual(d.args, args) {
				t.Fatal("statement or original input changed")
			}
		})
	}
}

// TestOnboardingTransactionFailures distinguishes known aborts from ambiguous commits without exposing errors.
func TestOnboardingTransactionFailures(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		row, commit, begin, exec error
		calls                    int
		code                     string
	}{
		{name: "invalid", row: &pgconn.PgError{Code: "P2800", Message: "sensitive"}, calls: 1, code: "invalid_onboarding"},
		{name: "denied", row: &pgconn.PgError{Code: "P2801"}, calls: 1, code: "onboarding_denied"},
		{name: "conflict", row: &pgconn.PgError{Code: "P2802"}, calls: 1, code: "onboarding_conflict"},
		{name: "stale", row: &pgconn.PgError{Code: "P2803"}, calls: 1, code: "onboarding_snapshot_changed"},
		{name: "busy", row: &pgconn.PgError{Code: "P2804"}, calls: 1, code: "onboarding_not_ready"},
		{name: "limit", row: &pgconn.PgError{Code: "P2805"}, calls: 1, code: "onboarding_limit"},
		{name: "known-row-abort", row: &pgconn.PgError{Code: "40001"}, calls: 3, code: "temporarily_unavailable"},
		{name: "known-commit-abort", commit: &pgconn.PgError{Code: "40P01"}, calls: 3, code: "temporarily_unavailable"},
		{name: "lost-commit", commit: errors.New("sensitive commit reply"), calls: 1, code: "temporarily_unavailable"},
		{name: "indeterminate", commit: &pgconn.PgError{Code: "40003"}, calls: 1, code: "temporarily_unavailable"},
		{name: "network-commit", commit: &pgconn.PgError{Code: "08006"}, calls: 1, code: "temporarily_unavailable"},
		{name: "begin", begin: errors.New("sensitive database"), calls: 1, code: "temporarily_unavailable"},
		{name: "timeout", exec: context.DeadlineExceeded, calls: 1, code: "temporarily_unavailable"},
		{name: "unknown-row", row: &pgconn.PgError{Code: "XX000", Message: "sensitive"}, calls: 1, code: "temporarily_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &fakeDB{body: result("execute"), rowErr: tc.row, commitErr: tc.commit, beginErr: tc.begin, execErr: tc.exec}
			body, e := invoke(&Service{DB: d}, context.Background(), request("execute"), "execute")
			var f *ledger.Failure
			if body != nil || !errors.As(e, &f) || f.Code != tc.code || d.calls != tc.calls || strings.Contains(e.Error(), "sensitive") {
				t.Fatalf("unsafe failure %v calls%d", e, d.calls)
			}
		})
	}
	d := &fakeDB{body: result("prepare"), firstRowErr: &pgconn.PgError{Code: "40001"}}
	r := request("prepare")
	first, e := invoke(&Service{DB: d}, context.Background(), r, "prepare")
	if e != nil || d.calls != 2 || d.commits != 1 {
		t.Fatal("known abort did not replay exact intent")
	}
	second, e := invoke(&Service{DB: d}, context.Background(), r, "prepare")
	if e != nil || string(first) != string(second) || d.args[1] != r.Body {
		t.Fatal("identical explicit replay changed")
	}
}

// TestOnboardingResultFence prevents malformed, foreign and excessive SQL results from committing.
func TestOnboardingResultFence(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{}`, strings.Repeat(" ", 1<<20+1), strings.Replace(prepared(), commandID, previewID, 1), strings.Replace(prepared(), `"state":"prepared"`, `"state":"committed"`, 1), strings.Replace(prepared(), `"acknowledged":false`, `"acknowledged":true`, 1), strings.Replace(prepared(), `"profile":"private-onboarding-command-v1"`, `"profile":"private-onboarding-command-v1","profile":"private-onboarding-command-v1"`, 1), strings.Replace(prepared(), `"result":null`, `"result":null,"Actor":"sensitive"`, 1)} {
		d := &fakeDB{body: body}
		raw, e := invoke(&Service{DB: d}, context.Background(), request("execute"), "execute")
		if raw != nil || e == nil || d.commits != 0 || d.rollbacks != 1 {
			t.Fatal("invalid SQL result escaped")
		}
	}
	d := &fakeDB{body: result("clear")}
	r := request("clear")
	r.Body = `{"recovery_id":"` + previewID + `"}`
	if _, e := invoke(&Service{DB: d}, context.Background(), r, "clear"); e == nil || d.commits != 0 {
		t.Fatal("foreign clear receipt escaped")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d = &fakeDB{body: prepared()}
	if _, e := invoke(&Service{DB: d}, ctx, request("execute"), "execute"); e == nil || d.commits != 0 || !d.cleanupDeadline {
		t.Fatal("cancelled work or cleanup failed")
	}
	if _, e := (&Service{}).Transition(context.Background(), Request{}, "invented"); e == nil {
		t.Fatal("unknown transition accepted")
	}
}
