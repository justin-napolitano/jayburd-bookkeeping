//go:build integration

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("webhook-proof", false, "Use only the disposable internal registry database")

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
		t.Skip("use scripts/test-webhook.sh")
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
func pgFixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	db := "hook_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_")
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE webhook_template")
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
func pgCall(t testing.TB, db, user, q string, args ...any) string {
	t.Helper()
	out, err := invoke(t, db, user, q, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// object decodes synthetic SQL projections without numeric-money conversion.
func pgObject(t testing.TB, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if e := json.Unmarshal([]byte(s), &v); e != nil {
		t.Fatal(e)
	}
	return v
}

// enrollment constructs an explicit synthetic operator attestation.
func enrollment() map[string]any {
	return map[string]any{"source_account_id": sourceID, "expected_binding_revision": "1", "provider": "plaid", "environment": "sandbox", "item_id": "synthetic-item", "account_id": "Account-A", "credential_ref": "00000000-0000-4000-8000-000000000083", "credential_version": "1", "consent_principal_id": actor, "consent_evidence_ref": "00000000-0000-4000-8000-000000000084", "consent_expires_at": "2099-01-01T00:00:00Z", "service_principal_id": reader, "worker_login": "synthetic_sync", "synthetic": true}
}

// setup installs the standard USD source fixture.
func setup(t *testing.T) (*pgx.Conn, string, map[string]any) { return setupCurrency(t, currency) }

// setupCurrency binds one source to independently installed currency metadata.
func setupCurrency(t *testing.T, currencyValue string) (*pgx.Conn, string, map[string]any) {
	c, db := pgFixture(t)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'import',NULL),($1,$3,'import',NULL)`, entity, actor, reader)
	exec(t, c, `INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by) VALUES($1,$2,$3,$4,$5,'2026-01-05',$6,$7)`, entity, book, sourceID, account, currencyValue, csvNS, actor)
	exec(t, c, `INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by) VALUES($1,$2,$3,$4,'jayburd-synthetic-statement-csv/1',$3,'SYNTHETIC',false,$5)`, entity, book, csvNS, sourceID, actor)
	exec(t, c, `INSERT INTO banking.source_permission SELECT $1,$2,$3,p,cap,NULL FROM unnest(ARRAY[$4::uuid,$5::uuid]) p CROSS JOIN unnest(ARRAY['read','import']) cap`, entity, book, sourceID, actor, reader)
	out := pgCall(t, db, "synthetic_operator", "SELECT integration.enroll_sync_stream($1,$2,$3,$4,$5)::text", actor, entity, book, "enroll", raw(enrollment()))
	return c, db, pgObject(t, out)
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

// signedPostgres composes the real receiver with signed HTTP and an injected fixed key endpoint.
func signedPostgres(t *testing.T, db, lost string) (*Handler, *lossPool, string, []byte) {
	t.Helper()
	c := connect(t, db, "synthetic_webhook")
	t.Cleanup(func() { c.Close(context.Background()) })
	pool := &lossPool{conn: c, lost: lost}
	callOwner(t, db, "SELECT integration.set_webhook_ingress('synthetic_webhook','sandbox','fixture',true)")
	private, key, _ := fixture(t)
	now := time.Now()
	key.Created = now.Add(-time.Hour).Unix()
	keys, _ := NewHTTPKeys(transportFunc(func(r *http.Request) (*http.Response, error) {
		if pool.active.Load() != 0 {
			t.Error("key lookup inside SQL transaction")
		}
		body, _ := json.Marshal(map[string]any{"key": key, "request_id": "synthetic-key-request"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header), Request: r}, nil
	}), credentialsFunc(func(context.Context) (Credentials, error) {
		if pool.active.Load() != 0 {
			t.Error("credential resolution inside SQL")
		}
		return Credentials{"synthetic-client", "synthetic-secret"}, nil
	}))
	verifier, _ := NewVerifier(keys)
	receiver, _ := NewSQLReceiver(pool)
	handler, _ := NewHandler(verifier, receiver)
	body := bodyFixture()
	return handler, pool, signBody(t, private, key.ID, body, now.Unix()), body
}

// callOwner uses the existing explicitly scoped owner fixture path.
func callOwner(t testing.TB, db, q string, args ...any) string {
	t.Helper()
	return pgCall(t, db, "synthetic_operator", q, args...)
}

// TestWebhookPostgresReceipt proves real committed response loss, duplicate delivery and current replay authority.
func TestWebhookPostgresReceipt(t *testing.T) {
	owner, db, _ := setup(t)
	handler, pool, token, body := signedPostgres(t, db, "accept_verified_webhook")
	if w := invokeHandler(handler, token, body); w.Code != 503 || !pool.fired {
		t.Fatalf("unknown commit status %d", w.Code)
	}
	if w := invokeHandler(handler, token, body); w.Code != 204 {
		t.Fatalf("durable replay status %d", w.Code)
	}
	if w := invokeHandler(handler, alternateSignature(t, token), body); w.Code != 204 {
		t.Fatalf("equivalent-signature replay %d", w.Code)
	}
	for query, want := range map[string]string{"SELECT count(*)::text FROM integration.webhook_receipt": "1", "SELECT count(*)::text FROM integration.sync_wakeup": "1", "SELECT count(*)::text FROM banking.source_observation": "0", "SELECT count(*)::text FROM ledger.journal_entry": "0", "SELECT count(*)::text FROM integration.outbox_message": "0"} {
		if value(t, owner, query) != want {
			t.Fatal("receipt/wakeup or no-financial-effect mismatch")
		}
	}
	exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", reader)
	if w := invokeHandler(handler, token, body); w.Code != 403 {
		t.Fatalf("revoked replay status %d", w.Code)
	}
	if value(t, owner, "SELECT count(*)::text FROM integration.sync_wakeup") != "1" {
		t.Fatal("revoked replay created intent")
	}
}

// TestWebhookPostgresCatchup proves bounded missed-notification recovery and stable tick acknowledgement loss.
func TestWebhookPostgresCatchup(t *testing.T) {
	owner, db, stream := setup(t)
	worker := connect(t, db, "synthetic_sync")
	defer worker.Close(context.Background())
	pool := &lossPool{conn: worker, lost: "catchup_sync"}
	catchup, _ := NewCatchup(pool)
	scope := Scope{entity, book}
	if _, e := catchup.Tick(context.Background(), scope, claimID, 1, nil); !errors.Is(e, errUnknown) {
		t.Fatalf("unknown catchup: %v", e)
	}
	out, e := catchup.Tick(context.Background(), scope, claimID, 1, nil)
	if e != nil || len(out.Items) != 1 || out.Items[0].StreamID != stream["stream_id"] || out.NextCursor != nil {
		t.Fatalf("catchup replay: %v", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM integration.sync_wakeup") != "1" {
		t.Fatal("catchup duplicate intent")
	}
	cursor := out.Items[0].StreamID
	if page, e := catchup.Tick(context.Background(), scope, claimID, 1, &cursor); e != nil || len(page.Items) != 0 {
		t.Fatal("cursor repeated scope")
	}
	exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", reader)
	if page, e := catchup.Tick(context.Background(), scope, "00000000-0000-4000-8000-000000000099", 1, nil); e != nil || len(page.Items) != 0 {
		t.Fatal("catchup did not skip revoked authority")
	}
	if value(t, owner, "SELECT count(*)::text FROM integration.sync_wakeup") != "1" || value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("catchup caused unintended effects")
	}
}

// TestWebhookPostgresRoleIsolation denies cross-role and later predefined-role drift without receipts.
func TestWebhookPostgresRoleIsolation(t *testing.T) {
	owner, db, _ := setup(t)
	_, _, _, _ = signedPostgres(t, db, "")
	sum := sha256.Sum256([]byte("synthetic"))
	verified := Verified{Environment: "sandbox", ItemID: "synthetic-item", ReceiptKey: hex.EncodeToString(sum[:]), BodySHA256: hex.EncodeToString(sum[:]), IssuedAt: time.Now().Unix()}
	for _, login := range []string{"synthetic_runtime", "synthetic_sync"} {
		c := connect(t, db, login)
		receiver, _ := NewSQLReceiver(c)
		if _, e := receiver.Receive(context.Background(), verified); !errors.Is(e, errDenied) {
			t.Fatal("cross-role ingress accepted")
		}
		c.Close(context.Background())
	}
	c := connect(t, db, "synthetic_webhook")
	defer c.Close(context.Background())
	receiver, _ := NewSQLReceiver(c)
	verified.ItemID = "unknown-item"
	if _, e := receiver.Receive(context.Background(), verified); !errors.Is(e, errDenied) {
		t.Fatal("unknown Item did not deny uniformly")
	}
	verified.ItemID = "synthetic-item"
	for _, role := range []string{"bookkeeping_api", "pg_execute_server_program", "pg_read_server_files", "pg_write_server_files", "pg_read_all_data", "pg_write_all_data"} {
		// Fixed test-owned identifiers only; never execute a program or access any server file.
		exec(t, owner, "GRANT "+role+" TO synthetic_webhook WITH ADMIN FALSE, INHERIT TRUE, SET FALSE")
		if _, e := receiver.Receive(context.Background(), verified); !errors.Is(e, errDenied) {
			t.Fatal("mixed or predefined role ingress accepted", role)
		}
		exec(t, owner, "REVOKE "+role+" FROM synthetic_webhook")
	}
	if value(t, owner, "SELECT count(*)::text FROM integration.webhook_receipt") != "0" {
		t.Fatal("denied ingress stored receipt")
	}
}
