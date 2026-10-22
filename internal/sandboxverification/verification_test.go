package sandboxverification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const good = `{"equal":true,"missing_count":"0","extra_count":"0","field_mismatch_count":"0","review_mismatch_count":"0","lineage_mismatch_count":"0","journal_count":"0","journal_outbox_count":"0","no_automatic_journals":true}`

// requestFixture holds synthetic markers to detect accidental formatting or error disclosure.
func requestFixture() Request {
	return Request{"00000000-0000-4000-8000-000000000001", "1", []byte(`{"synthetic-private-fixture":true}`), []byte(`{"synthetic-private-map":true}`)}
}

// fakePool records transaction boundaries without SQL/cloud access.
type fakePool struct {
	tx       *fakeTx
	begins   int
	badBegin bool
}

func (p *fakePool) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.begins++
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 4*time.Second || o.IsoLevel != pgx.Serializable {
		return nil, errors.New("missing budget")
	}
	if p.badBegin {
		return nil, errors.New("synthetic-private-failure")
	}
	return p.tx, nil
}

// fakeTx embeds unused pgx methods while exercising only the actual bounded adapter calls.
type fakeTx struct {
	pgx.Tx
	fault                       string
	raw                         string
	role                        bool
	queries, commits, rollbacks int
	cancel                      context.CancelFunc
	independent                 bool
}

func (t *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if t.fault == "exec" {
		return pgconn.CommandTag{}, errors.New("synthetic-private-failure")
	}
	return pgconn.CommandTag{}, nil
}
func (t *fakeTx) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	t.queries++
	if strings.Contains(q, "session_user") {
		return row{role: &t.role, fail: t.fault == "identity"}
	}
	if t.cancel != nil {
		t.cancel()
	}
	return row{raw: t.raw, fail: t.fault == "query"}
}
func (t *fakeTx) Commit(context.Context) error {
	t.commits++
	if t.fault == "commit" {
		return errors.New("synthetic-private-failure")
	}
	return nil
}
func (t *fakeTx) Rollback(ctx context.Context) error {
	t.rollbacks++
	_, deadline := ctx.Deadline()
	t.independent = deadline && ctx.Err() == nil
	return nil
}

// row scans only the Boolean or JSON result shapes that the capability returns.
type row struct {
	role *bool
	raw  string
	fail bool
}

func (r row) Scan(dest ...any) error {
	if r.fail {
		return errors.New("synthetic-private-failure")
	}
	if r.role != nil {
		*dest[0].(*bool) = *r.role
	} else {
		*dest[0].(*[]byte) = []byte(r.raw)
	}
	return nil
}

func TestCompareBoundaries(t *testing.T) {
	for _, phase := range []string{"success", "begin", "exec", "identity", "denied", "query", "malformed", "commit", "cancel"} {
		t.Run(phase, func(t *testing.T) {
			tx := &fakeTx{fault: phase, raw: good, role: phase != "denied"}
			pool := &fakePool{tx: tx, badBegin: phase == "begin"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if phase == "malformed" {
				tx.raw = `{"source_record":"synthetic-private-fixture"}`
			}
			if phase == "cancel" {
				tx.cancel = cancel
			}
			result, e := Compare(ctx, pool, requestFixture())
			if phase == "success" {
				if e != nil || !result.Equal || tx.commits != 1 {
					t.Fatal("valid comparison failed")
				}
			} else {
				if e == nil || result != (Result{}) || strings.Contains(e.Error(), "private") {
					t.Fatal("failed comparison exposed evidence")
				}
			}
			if phase == "denied" && tx.queries != 1 {
				t.Fatal("identity denial reached fixture SQL")
			}
			if phase != "begin" && (!tx.independent || tx.rollbacks != 1) {
				t.Fatal("missing independent cleanup")
			}
			if phase == "commit" && e != ErrUnconfirmed {
				t.Fatal("ambiguous commit classified as success/retry")
			}
			if pool.begins != 1 {
				t.Fatal("automatic transaction retry")
			}
		})
	}
}
func TestDecodeResultRejectsAmbiguity(t *testing.T) {
	for _, raw := range []string{
		strings.Replace(good, `"missing_count":"0"`, `"missing_count":"00"`, 1),
		strings.Replace(good, `"missing_count":"0"`, `"missing_count":"1002"`, 1),
		strings.Replace(good, `"missing_count":"0"`, `"missing_count":0`, 1),
		strings.Replace(good, `"missing_count":"0"`, `"missing_count":null`, 1),
		strings.Replace(good, `"equal":true`, `"equal":false`, 1),
		strings.Replace(good, `"equal":true`, `"equal":true,"equal":true`, 1),
		strings.Replace(good, `"equal":true`, `"source":"private","equal":true`, 1),
		strings.Replace(good, `"no_automatic_journals":true`, `"no_automatic_journals":false`, 1),
		good + "{}",
	} {
		if result, e := DecodeResult([]byte(raw)); e == nil || result != (Result{}) {
			t.Fatal("ambiguous result accepted")
		}
	}
	unequal := strings.Replace(strings.Replace(good, `"equal":true`, `"equal":false`, 1), `"missing_count":"0"`, `"missing_count":"1001"`, 1)
	if result, e := DecodeResult([]byte(unequal)); e != nil || result.Equal || result.MissingCount != "1001" {
		t.Fatal("bounded mismatch rejected")
	}
}
func TestInputInertnessAndRedaction(t *testing.T) {
	r := requestFixture()
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		if strings.Contains(fmt.Sprintf(format, r), "synthetic-private") {
			t.Fatal("request formatting leaked")
		}
	}
	if _, e := json.Marshal(r); e == nil {
		t.Fatal("request persisted")
	}
	for _, revision := range []string{"0", "01", "9223372036854775808", "-1", "1.0"} {
		bad := r
		bad.Revision = revision
		if ValidRequest(bad) {
			t.Fatal("invalid revision accepted")
		}
	}
	bad := r
	bad.Fixture = make([]byte, MaxInputBytes)
	pool := &fakePool{tx: &fakeTx{}}
	if _, e := Compare(context.Background(), pool, bad); e == nil || pool.begins != 0 {
		t.Fatal("oversized input opened database")
	}
}
