//go:build integration

package syncdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("sync-dispatch-proof", false, "Use only the disposable internal registry database")

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
		t.Skip("use scripts/test-sync-dispatch.sh")
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
func fixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	db := "dispatch_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE sync_dispatch_template")
	c := connect(t, db, "postgres")
	t.Cleanup(func() { c.Close(context.Background()) })
	exec(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','approver'),($2,'synthetic','reader'),($3,'synthetic','other')`, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional LLC','legal')`, entity)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$3,'read',NULL),($1,$4,'read',NULL)`, entity, actor, reader, other)
	exec(t, c, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic book')`, entity, book)
	exec(t, c, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000006','JPY','392',0,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000007','KWD','414',3,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000008','XXX','999',NULL,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000009','EUR','978',2,'2020-01-01','2025-01-01','synthetic')`, currency)
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
func setupCurrency(t *testing.T, currencyValue string) (*pgx.Conn, string, map[string]any) {
	c, db := fixture(t)
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

// credentialFunc supplies exclusively fictional in-memory provider credentials.
type credentialFunc func(context.Context, acquisition.CredentialReference) (acquisition.ProviderCredentials, error)

// Resolve delegates the explicitly injected synthetic resolver.
func (f credentialFunc) Resolve(c context.Context, r acquisition.CredentialReference) (acquisition.ProviderCredentials, error) {
	return f(c, r)
}

// roundTripFunc replaces provider networking without changing the actual pinned HTTP adapter.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip delegates the synthetic response transport.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// realRunner uses real SQL and durable private staging with the production exact-money HTTP decoder.
func realRunner(t *testing.T, db, lost string) (*acquisition.Runner, *lossPool, *int) {
	t.Helper()
	conn := connect(t, db, "synthetic_sync")
	t.Cleanup(func() { conn.Close(context.Background()) })
	pool := &lossPool{conn: conn, lost: lost}
	calls := 0
	provider, e := acquisition.NewPlaidHTTP(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if pool.active.Load() != 0 {
			t.Error("provider inside transaction")
		}
		calls++
		if req.URL.String() != "https://sandbox.plaid.com/transactions/sync" || req.Method != "POST" {
			t.Error("unpinned provider")
		}
		b, _ := io.ReadAll(req.Body)
		var body map[string]any
		if json.Unmarshal(b, &body) != nil {
			t.Error("invalid provider request")
		}
		options, _ := body["options"].(map[string]any)
		if options["account_id"] != "Account-A" {
			t.Error("missing account filter")
		}
		page := `{"accounts":[{"account_id":"Account-A","type":"depository"}],"added":[{"account_id":"Account-A","transaction_id":"Txn-1","amount":9007199254740993.01,"iso_currency_code":"USD","unofficial_currency_code":null,"date":"2026-01-02","pending":false,"pending_transaction_id":null,"name":"Fictional supplies"}],"modified":[],"removed":[],"next_cursor":"C1","has_more":false,"request_id":"synthetic-request","transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(page)), Header: make(http.Header)}, nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	path := t.TempDir()
	if e = os.Chmod(path, 0700); e != nil {
		t.Fatal(e)
	}
	store, e := staging.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { store.Close() })
	evidence, e := acquisition.NewLocalEvidence(store)
	if e != nil {
		t.Fatal(e)
	}
	creds := credentialFunc(func(_ context.Context, ref acquisition.CredentialReference) (acquisition.ProviderCredentials, error) {
		if pool.active.Load() != 0 || ref.ItemID != "Item-A" || ref.Environment != "sandbox" {
			t.Fatal("unbound credentials")
		}
		return acquisition.ProviderCredentials{AccessToken: "synthetic-access", ClientID: "synthetic-client", Secret: "synthetic-secret"}, nil
	})
	runner, e := acquisition.New(pool, provider, creds, evidence)
	if e != nil {
		t.Fatal(e)
	}
	return runner, pool, &calls
}

// dispatchSetup explicitly enrolls the owner-approved worker and publisher binding before runtime delivery.
func dispatchSetup(t *testing.T) (*pgx.Conn, string, string) {
	owner, db, _ := setup(t)
	call(t, db, "synthetic_operator", "SELECT integration.set_connection_worker($1,$2,$3,$4,$5)", actor, entity, book, "worker", raw(map[string]any{"service_principal_id": reader, "worker_login": "synthetic_sync", "enabled": true}))
	v := object(t, call(t, db, "synthetic_operator", "SELECT integration.enroll_sync_delivery($1,$2,$3,$4,$5)", actor, entity, book, "delivery", raw(map[string]any{"service_principal_id": reader, "worker_login": "synthetic_sync", "publisher_login": "synthetic_publisher", "enabled": true})))
	return owner, db, v["binding_id"].(string)
}

// relay commits publisher intent and captures the exact identifier-only task without contacting any queue.
func relay(t *testing.T, db, binding string) []byte {
	t.Helper()
	conn := connect(t, db, "synthetic_publisher")
	defer conn.Close(context.Background())
	pool := &lossPool{conn: conn}
	var body []byte
	p, e := NewPublisher(pool, binding, transportFunc(func(_ context.Context, task observer.Task) (observer.EnqueueResult, error) {
		if pool.active.Load() != 0 || task.TargetVersion != 2 {
			t.Fatal("unsafe queue call")
		}
		body = append([]byte(nil), task.Body...)
		return observer.Accepted, nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	out, e := p.Tick(context.Background())
	if e != nil || out.EnqueueCalls != 1 || out.Recorded != 1 || len(body) == 0 {
		t.Fatalf("relay %+v %v", out, e)
	}
	return body
}

// TestDispatchRealAcquisition commits one exact source observation and recovers duplicate/lost receipts without another provider call.
func TestDispatchRealAcquisition(t *testing.T) {
	for _, lost := range []string{"", "complete_sync", "sync_delivery_settle"} {
		t.Run("loss_"+lost, func(t *testing.T) {
			owner, db, binding := dispatchSetup(t)
			body := relay(t, db, binding)
			runner, runnerDB, calls := realRunner(t, db, lost)
			worker := connect(t, db, "synthetic_sync")
			defer worker.Close(context.Background())
			consumerDB := &lossPool{conn: worker, lost: lost}
			consumer, e := NewConsumer(consumerDB, runner)
			if e != nil {
				t.Fatal(e)
			}
			e = consumer.Handle(context.Background(), body)
			if lost == "sync_delivery_settle" {
				if e == nil || e.Error() != "commit_unknown" || !consumerDB.fired {
					t.Fatalf("settle uncertainty %v", e)
				}
			} else if e != nil {
				t.Fatal(e)
			}
			if lost == "complete_sync" && !runnerDB.fired {
				t.Fatal("completion loss missing")
			}
			if e = consumer.Handle(context.Background(), body); e != nil {
				t.Fatal(e)
			}
			if *calls != 1 {
				t.Fatalf("duplicate provider effects %d", *calls)
			}
			for q, want := range map[string]string{"SELECT count(*)::text FROM banking.source_observation": "1", "SELECT count(*)::text FROM ledger.journal_entry": "0", "SELECT count(*)::text FROM integration.outbox_message": "0", "SELECT cursor_value FROM integration.sync_stream": "C1", "SELECT trim_scale(amount)::text FROM banking.observation_revision": "9007199254740993.01"} {
				if got := value(t, owner, q); got != want {
					t.Fatalf("expected %s got %s", want, got)
				}
			}
			exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1", reader)
			if e = consumer.Handle(context.Background(), body); e == nil || *calls != 1 {
				t.Fatalf("revoked replay %v", e)
			}
		})
	}
}

// TestDispatchRealUnknownClaim cannot call a provider after an actual committed delivery claim loses its reply.
func TestDispatchRealUnknownClaim(t *testing.T) {
	owner, db, binding := dispatchSetup(t)
	body := relay(t, db, binding)
	runner, _, calls := realRunner(t, db, "")
	worker := connect(t, db, "synthetic_sync")
	defer worker.Close(context.Background())
	pool := &lossPool{conn: worker, lost: "sync_delivery_claim"}
	c, _ := NewConsumer(pool, runner)
	if e := c.Handle(context.Background(), body); e == nil || e.Error() != "commit_unknown" || !pool.fired || *calls != 0 {
		t.Fatalf("unsafe unknown claim %v", e)
	}
	if e := c.Handle(context.Background(), body); e == nil || *calls != 0 {
		t.Fatalf("healthy claimed lease replay %v", e)
	}
	if got := value(t, owner, "SELECT count(*)::text FROM integration.sync_set"); got != "0" {
		t.Fatal("unknown claim started acquisition")
	}
}
