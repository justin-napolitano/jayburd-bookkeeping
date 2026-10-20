//go:build integration

package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("acquisition-proof", false, "Use only the disposable internal registry database")

const entity = "00000000-0000-4000-8000-000000000002"
const book = "00000000-0000-4000-8000-000000000003"
const actor = "00000000-0000-4000-8000-000000000001"
const reader = "00000000-0000-4000-8000-000000000020"
const other = "00000000-0000-4000-8000-000000000021"
const account = "00000000-0000-4000-8000-000000000011"
const currency = "00000000-0000-4000-8000-000000000005"

// connect uses only the runner's private synthetic trust network, never an inherited DSN.
func connect(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if !*proof {
		t.Skip("use scripts/test-acquisition.sh")
	}
	c, e := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if e != nil {
		t.Fatal(e)
	}
	return c
}

// exec is owner-only fixture setup or inspection, never product privilege.
func exec(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal(e)
	}
}

// value reads fixed aggregate proof facts with the fixture owner.
func value(t testing.TB, c *pgx.Conn, q string, args ...any) string {
	t.Helper()
	var s string
	if e := c.QueryRow(context.Background(), q, args...).Scan(&s); e != nil {
		t.Fatal(e)
	}
	return s
}

// requireCode compares SQLSTATE without relying on potentially sensitive diagnostics.
func requireCode(t testing.TB, e error, want string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(e, &p) || p.Code != want {
		t.Fatalf("want %s got %v", want, e)
	}
}

// fixture installs only fictional ownership and independently chosen currency metadata.
func fixture(t *testing.T, history ...bool) (*pgx.Conn, string) {
	t.Helper()
	db := "acq_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE acquisition_template")
	c := connect(t, db, "postgres")
	t.Cleanup(func() { c.Close(context.Background()) })
	exec(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','approver'),($2,'synthetic','reader'),($3,'synthetic','other')`, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional LLC','legal')`, entity)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$3,'read',NULL),($1,$4,'read',NULL)`, entity, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic book')`, entity, book)
	usdFrom, usdEnd := "2000-01-01", ""
	if len(history) > 0 && history[0] {
		usdFrom = "2026-09-17"
	}
	exec(t, c, `INSERT INTO reference.currency VALUES($1,'USD','840',2,$2::date,nullif($3,'')::date,'synthetic'),('00000000-0000-4000-8000-000000000006','JPY','392',0,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000007','KWD','414',3,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000008','XXX','999',NULL,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000009','EUR','978',2,'2020-01-01','2025-01-01','synthetic')`, currency, usdFrom, usdEnd)
	if len(history) > 0 && history[0] {
		exec(t, c, `INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000090','USD','840',2,'2001-01-01','2026-09-17','synthetic-history')`)
	}
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Fictional cash','asset','debit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000012','2000','Fictional liability','liability','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000013','1001','Fictional nonmonetary','asset','debit',false,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000014','1002','Fictional contra','asset','credit',true,'2026-01-01',NULL),($1,$2,'00000000-0000-4000-8000-000000000015','1003','Fictional second deposit','asset','debit',true,'2026-01-01',NULL)`, entity, book, account)
	return c, db
}

// raw encodes synthetic exact values without floating-point amounts.
func raw(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return string(b)
}

// invoke uses a real login, bounded transactions and full serialization retry.
func invoke(t testing.TB, db, user, q string, args ...any) (string, error) {
	t.Helper()
	c := connect(t, db, user)
	defer c.Close(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	var result string
	var err error
	for i := 0; i < 3; i++ {
		tx, e := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return "", e
		}
		_, e = tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL lock_timeout='1s'")
		if e == nil && user == "synthetic_operator" {
			_, e = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner")
		}
		if e == nil {
			e = tx.QueryRow(ctx, q, args...).Scan(&result)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		_ = tx.Rollback(context.Background())
		if e == nil {
			return result, nil
		}
		err = e
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "40001" && p.Code != "40P01") {
			break
		}
	}
	return "", err
}

// Fixed fictional identifiers belong only to disposable proof databases.
const sourceID = "00000000-0000-4000-8000-000000000080"
const csvNS = "00000000-0000-4000-8000-000000000081"
const claimID = "00000000-0000-4000-8000-000000000082"

// call fails only on unexpected fixed diagnostics; all data below is fictional.
func call(t testing.TB, db, user, q string, args ...any) string {
	t.Helper()
	out, err := invoke(t, db, user, q, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// object decodes synthetic SQL projections without numeric-money conversion.
func object(t testing.TB, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal([]byte(s), &v); e != nil {
		t.Fatal(e)
	}
	return v
}

// enrollment constructs an explicit synthetic operator attestation.
func enrollment() map[string]any {
	return map[string]any{"source_account_id": sourceID, "expected_binding_revision": "1", "provider": "plaid", "environment": "sandbox", "item_id": "Item-A", "account_id": "Account-A", "credential_ref": "00000000-0000-4000-8000-000000000083", "credential_version": "1", "consent_principal_id": actor, "consent_evidence_ref": "00000000-0000-4000-8000-000000000084", "consent_expires_at": "2099-01-01T00:00:00Z", "service_principal_id": reader, "worker_login": "synthetic_sync", "synthetic": true}
}

// setup installs the standard USD source fixture.
func setup(t *testing.T) (*pgx.Conn, string, map[string]any) { return setupCurrency(t, currency) }

// setupCurrency binds one source to independently installed currency metadata.
func setupCurrency(t *testing.T, currencyValue string, history ...bool) (*pgx.Conn, string, map[string]any) {
	c, db := fixture(t, history...)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'import',NULL),($1,$3,'import',NULL)`, entity, actor, reader)
	exec(t, c, `INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by) VALUES($1,$2,$3,$4,$5,'2026-01-05',$6,$7)`, entity, book, sourceID, account, currencyValue, csvNS, actor)
	exec(t, c, `INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by) VALUES($1,$2,$3,$4,'jayburd-synthetic-statement-csv/1',$3,'SYNTHETIC',false,$5)`, entity, book, csvNS, sourceID, actor)
	exec(t, c, `INSERT INTO banking.source_permission SELECT $1,$2,$3,p,cap,NULL FROM unnest(ARRAY[$4::uuid,$5::uuid]) p CROSS JOIN unnest(ARRAY['read','import']) cap`, entity, book, sourceID, actor, reader)
	out := call(t, db, "synthetic_operator", "SELECT integration.enroll_sync_stream($1,$2,$3,$4,$5)::text", actor, entity, book, "enroll", raw(enrollment()))
	return c, db, object(t, out)
}

// lossPool injects an actual committed response loss and detects external work inside a transaction.
type lossPool struct {
	conn   *pgx.Conn
	lost   string
	fired  bool
	active atomic.Int32
}

// BeginTx wraps a real restricted login without impersonation.
func (p *lossPool) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, e := p.conn.BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	p.active.Add(1)
	return &lossTx{Tx: tx, p: p}, nil
}

// lossTx records only static SQL function text to select a synthetic reply loss.
type lossTx struct {
	pgx.Tx
	p      *lossPool
	q      string
	closed bool
}

// QueryRow remembers the last static call without retaining any arguments.
func (tx *lossTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	tx.q = q
	return tx.Tx.QueryRow(ctx, q, args...)
}

// release marks the database transaction as closed once.
func (tx *lossTx) release() {
	if !tx.closed {
		tx.closed = true
		tx.p.active.Add(-1)
	}
}

// Commit first commits real rows and then loses the selected acknowledgement once.
func (tx *lossTx) Commit(ctx context.Context) error {
	e := tx.Tx.Commit(ctx)
	tx.release()
	if e == nil && !tx.p.fired && tx.p.lost != "" && strings.Contains(tx.q, tx.p.lost) {
		tx.p.fired = true
		return errors.New("synthetic lost response")
	}
	return e
}

// Rollback participates in independent cleanup without changing commit truth.
func (tx *lossTx) Rollback(ctx context.Context) error {
	e := tx.Tx.Rollback(ctx)
	tx.release()
	return e
}

// realRunner composes the restricted SQL adapter with actual loopback HTTP and private filesystem evidence.
func realRunner(t *testing.T, db string, lost string, handler http.HandlerFunc) (*Runner, *lossPool, *int) {
	t.Helper()
	c := connect(t, db, "synthetic_sync")
	t.Cleanup(func() { c.Close(context.Background()) })
	p := &lossPool{conn: c, lost: lost}
	calls := 0
	provider := syntheticHTTP(t, func(w http.ResponseWriter, r *http.Request) {
		if p.active.Load() != 0 {
			t.Error("external fetch inside transaction")
		}
		calls++
		handler(w, r)
	})
	resolver := credentialFunc(func(ctx context.Context, ref CredentialReference) (ProviderCredentials, error) {
		if p.active.Load() != 0 {
			t.Error("credential resolution inside transaction")
		}
		if ref.ItemID != "Item-A" || ref.Environment != "sandbox" {
			t.Fatal("unbound credential reference")
		}
		return credentialsFixture(), nil
	})
	runner, e := New(p, provider, resolver, localEvidence(t))
	if e != nil {
		t.Fatal(e)
	}
	return runner, p, &calls
}

// TestAcquisitionCommittedReplyLoss proves claim, page and completion recovery against actual committed PostgreSQL state.
func TestAcquisitionCommittedReplyLoss(t *testing.T) {
	for _, lost := range []string{"claim_sync", "stage_sync_page", "complete_sync"} {
		t.Run(lost, func(t *testing.T) {
			owner, db, stream := setup(t)
			s := Scope{entity, book, stream["stream_id"].(string)}
			runner, p, calls := realRunner(t, db, lost, func(w http.ResponseWriter, r *http.Request) {
				providerRequest(t, r)
				w.Write(fixturePage("Txn-1", "9007199254740993.01", "USD", "C1", false))
			})
			if _, err := runner.Run(context.Background(), s, claimID); err == nil || err.Error() != "commit_unknown" {
				t.Fatalf("loss classification: %v", err)
			}
			if !p.fired {
				t.Fatal("loss not exercised")
			}
			if lost == "claim_sync" && *calls != 0 {
				t.Fatal("ambiguous claim accessed provider")
			}
			out, err := runner.Run(context.Background(), s, claimID)
			if err != nil || out.Status != "completed" || out.ChangedCount != "1" || *calls != 1 {
				t.Fatalf("recovery: %+v %v calls=%d", out, err, *calls)
			}
			if _, err = runner.Run(context.Background(), s, claimID); err != nil || *calls != 1 {
				t.Fatal("receipt replay refetched")
			}
			for q, want := range map[string]string{"SELECT count(*)::text FROM banking.source_observation": "1", "SELECT count(*)::text FROM ledger.journal_entry": "0", "SELECT count(*)::text FROM integration.outbox_message": "0", "SELECT cursor_value FROM integration.sync_stream": "C1", "SELECT trim_scale(amount)::text FROM banking.observation_revision": "9007199254740993.01"} {
				if got := value(t, owner, q); got != want {
					t.Fatalf("aggregate mismatch %s %s", got, want)
				}
			}
			exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1", reader)
			if _, err = runner.Run(context.Background(), s, claimID); err == nil || err.Error() != "not_authorized" || *calls != 1 {
				t.Fatalf("revoked receipt replay %v", err)
			}
		})
	}
}

// TestAcquisitionMutationRestart proves whole-loop C0 restart and original-wakeup replay without partial publication.
func TestAcquisitionMutationRestart(t *testing.T) {
	owner, db, stream := setup(t)
	s := Scope{entity, book, stream["stream_id"].(string)}
	n := 0
	runner, _, calls := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
		cursor := providerRequest(t, r)
		n++
		switch n {
		case 1, 3:
			if cursor != nil {
				t.Error("restart was not C0")
			}
			w.Write(fixturePage("Txn-1", "12.50", "USD", "C1", true))
		case 2:
			w.WriteHeader(400)
			w.Write([]byte(`{"error_code":"TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION"}`))
		case 4:
			w.Write(fixturePage("Txn-2", "-1.25e2", "USD", "C2", false))
		default:
			t.Error("unexpected provider call")
		}
	})
	out, err := runner.Run(context.Background(), s, claimID)
	if err != nil || out.ChangedCount != "2" || *calls != 4 {
		t.Fatalf("mutation %+v %v calls=%d", out, err, *calls)
	}
	if _, err = runner.Run(context.Background(), s, claimID); err != nil || *calls != 4 {
		t.Fatalf("original identity replay: %v", err)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.source_observation") != "2" || value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("partial or financial publication")
	}
}

// TestAcquisitionRevocationBeforePublication denies a fetched page after current source authority is revoked.
func TestAcquisitionRevocationBeforePublication(t *testing.T) {
	owner, db, stream := setup(t)
	s := Scope{entity, book, stream["stream_id"].(string)}
	runner, _, _ := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
		exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1", reader)
		w.Write(fixturePage("Txn-1", "12.50", "USD", "C1", false))
	})
	if _, err := runner.Run(context.Background(), s, claimID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("revocation: %v", err)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.source_observation") != "0" {
		t.Fatal("revoked publication")
	}
}

// TestAcquisitionExpiredClaimAndDeadline forbids provider access after lease expiry or caller cancellation.
func TestAcquisitionExpiredClaimAndDeadline(t *testing.T) {
	owner, db, stream := setup(t)
	s := Scope{entity, book, stream["stream_id"].(string)}
	runner, _, calls := realRunner(t, db, "claim_sync", func(http.ResponseWriter, *http.Request) { t.Error("expired claim fetched") })
	if _, err := runner.Run(context.Background(), s, claimID); err == nil || err.Error() != "commit_unknown" {
		t.Fatal("claim fault absent")
	}
	exec(t, owner, "UPDATE integration.sync_stream SET lease_until=clock_timestamp()-interval '1 second'")
	if _, err := runner.Run(context.Background(), s, claimID); err == nil || err.Error() != "stale_attempt" || *calls != 0 {
		t.Fatalf("expired claim %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runner.Run(ctx, s, claimID); err == nil || *calls != 0 {
		t.Fatal("cancelled caller reached provider")
	}
}

// TestAcquisitionRoleIsolation rejects API login and a worker with mixed runtime capability.
func TestAcquisitionRoleIsolation(t *testing.T) {
	owner, db, stream := setup(t)
	s := Scope{entity, book, stream["stream_id"].(string)}
	api := connect(t, db, "synthetic_runtime")
	defer api.Close(context.Background())
	if _, err := (&SQLRepository{DB: api}).Resume(context.Background(), s, claimID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("API worker authority %v", err)
	}
	exec(t, owner, "GRANT bookkeeping_api TO synthetic_sync")
	defer exec(t, owner, "REVOKE bookkeeping_api FROM synthetic_sync")
	worker := connect(t, db, "synthetic_sync")
	defer worker.Close(context.Background())
	if _, err := (&SQLRepository{DB: worker}).Resume(context.Background(), s, claimID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("mixed worker authority %v", err)
	}
}
