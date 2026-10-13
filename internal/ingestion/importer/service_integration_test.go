//go:build integration

package importer

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

var serviceProof = flag.Bool("import-service-proof", false, "Use only the disposable internal import database")

// proofConnect refuses ambient DSNs and connects only to the disposable synthetic network.
func proofConnect(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if !*serviceProof {
		t.Skip("use the disposable import service proof harness")
	}
	c, e := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

// proofExec is privileged synthetic fixture creation, not application permission.
func proofExec(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal(e)
	}
}

// proofJSON decodes fixed test response metadata without floating-point financial values.
func proofJSON(t testing.TB, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var out map[string]json.RawMessage
	if e := json.Unmarshal(raw, &out); e != nil {
		t.Fatal(e)
	}
	return out
}

// proofString extracts string identity/revision fields from actual SQL output.
func proofString(t testing.TB, raw []byte, key string) string {
	t.Helper()
	var value string
	if e := json.Unmarshal(proofJSON(t, raw)[key], &value); e != nil {
		t.Fatal(e)
	}
	return value
}

// ownerCall invokes only reviewed provisioning with an actual NOINHERIT owner-capability login.
func ownerCall(t testing.TB, c *pgx.Conn, q string, args ...any) []byte {
	t.Helper()
	ctx := context.Background()
	tx, e := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner"); e != nil {
		t.Fatal(e)
	}
	var raw []byte
	if e = tx.QueryRow(ctx, q, args...).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	return raw
}

// proofFixture provisions current synthetic source ownership through the restricted operator path.
func proofFixture(t *testing.T) (*Service, *pgx.Conn, *pgx.Conn, Registration) {
	t.Helper()
	db := "importer_" + strings.ToLower(t.Name())
	admin := proofConnect(t, "postgres", "postgres")
	proofExec(t, admin, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE import_template")
	owner := proofConnect(t, db, "postgres")
	proofExec(t, owner, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','importer')`, testActor)
	proofExec(t, owner, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional import LLC','legal')`, testEntity)
	proofExec(t, owner, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL)`, testEntity, testActor)
	proofExec(t, owner, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Synthetic import book')`, testEntity, testBook)
	proofExec(t, owner, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic')`, testCurrency)
	proofExec(t, owner, `INSERT INTO ledger.account VALUES($1,$2,'00000000-0000-4000-8000-000000000011','1000','Fictional cash','asset','debit',true,'2026-01-01',NULL)`, testEntity, testBook)
	operator := proofConnect(t, db, "synthetic_operator")
	manifest := `{"ledger_account_id":"00000000-0000-4000-8000-000000000011","currency_version_id":"` + testCurrency + `","binding_date":"2026-01-01","binding_evidence_ref":"00000000-0000-4000-8000-000000000030","reader_principal_id":"` + testActor + `","namespace":{"profile":"` + ingestion.Profile + `","synthetic_dataset_id":"00000000-0000-4000-8000-000000000031","source_account_ref":"synthetic"}}`
	enrolled := ownerCall(t, operator, "select banking.register_source($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)", testActor, testEntity, testBook, "source", manifest)
	sid := proofString(t, enrolled, "source_account_id")
	var nid string
	if e := owner.QueryRow(context.Background(), "select namespace_id::text from banking.source_namespace where source_account_id=$1", sid).Scan(&nid); e != nil {
		t.Fatal(e)
	}
	permission := `{"expected_binding_revision":"1","principal_id":"` + testActor + `","enabled":true,"entity_import_enabled":true}`
	granted := ownerCall(t, operator, "select banking.set_source_import_permission($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text,$6::text)", testActor, testEntity, testBook, sid, "grant", permission)
	runtime := proofConnect(t, db, "synthetic_runtime")
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	store, e := staging.Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = store.Close() })
	request := Registration{sid, nid, proofString(t, granted, "binding_revision"), "JANUARY", "2026-01-01", "2026-02-01", ingestion.Profile}
	return New(runtime, store), owner, runtime, request
}

// TestImportServicePostgresStages connects actual authorization/storage/decoder/publication without posting.
func TestImportServicePostgresStages(t *testing.T) {
	service, owner, _, request := proofFixture(t)
	registered, e := service.Register(context.Background(), testScope, "statement", request)
	if e != nil {
		t.Fatal(e)
	}
	batch := proofString(t, registered, "batch_id")
	result, e := service.Upload(context.Background(), testScope, batch, source(testCSV))
	if e != nil {
		t.Fatal(e)
	}
	if proofString(t, result, "state") != "preview_ready" {
		t.Fatal("not preview ready")
	}
	replay, e := service.Upload(context.Background(), testScope, batch, source(testCSV))
	if e != nil {
		t.Fatal(e)
	}
	if proofString(t, replay, "preview_id") != proofString(t, result, "preview_id") {
		t.Fatal("identical replay created another preview")
	}
	var effects int
	if e = owner.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM ledger.journal_entry)+(SELECT count(*) FROM banking.source_observation)+(SELECT count(*) FROM banking.review_item)+(SELECT count(*) FROM integration.outbox_message)`).Scan(&effects); e != nil {
		t.Fatal(e)
	}
	if effects != 0 {
		t.Fatal("preview caused financial/import side effects")
	}
	public, e := service.Batch(context.Background(), testScope, batch)
	if e != nil {
		t.Fatal(e)
	}
	for _, private := range []string{"object_id", "currency", "source_account_ref"} {
		if _, ok := proofJSON(t, public)[private]; ok {
			t.Fatal("private metadata in batch wrapper")
		}
	}
	_, e = service.Upload(context.Background(), testScope, batch, source(strings.Replace(testCSV, "0012.500", "12.51", 1)))
	if e == nil || e.Error() != "statement_identity_conflict" {
		t.Fatalf("changed bytes accepted: %v", e)
	}
}

// TestImportServicePostgresRejection persists bounded diagnostics with no usable decoder rows.
func TestImportServicePostgresRejection(t *testing.T) {
	service, owner, _, request := proofFixture(t)
	registered, e := service.Register(context.Background(), testScope, "statement", request)
	if e != nil {
		t.Fatal(e)
	}
	batch := proofString(t, registered, "batch_id")
	result, e := service.Upload(context.Background(), testScope, batch, source(strings.Replace(testCSV, "0012.500", "12.501", 1)))
	if e != nil {
		t.Fatal(e)
	}
	if proofString(t, result, "state") != "rejected" {
		t.Fatal("bad cents accepted")
	}
	var rows int
	if e = owner.QueryRow(context.Background(), "select jsonb_array_length(rows) from banking.import_preview where batch_id=$1", batch).Scan(&rows); e != nil {
		t.Fatal(e)
	}
	if rows != 0 {
		t.Fatal("partial usable rows")
	}
}

// TestImportServicePostgresRevocation checks actual current permission before consuming source bytes.
func TestImportServicePostgresRevocation(t *testing.T) {
	service, owner, _, request := proofFixture(t)
	registered, e := service.Register(context.Background(), testScope, "statement", request)
	if e != nil {
		t.Fatal(e)
	}
	proofExec(t, owner, "update banking.source_permission set revoked_at=clock_timestamp() where principal_id=$1 and capability='import'", testActor)
	input := source(testCSV)
	_, e = service.Upload(context.Background(), testScope, proofString(t, registered, "batch_id"), input)
	if e == nil || input.reads != 0 || input.closes != 1 {
		t.Fatal("revoked actor consumed evidence")
	}
}

// lostReply wraps actual COMMIT; it never claims a network-level fault injection.
type lostReply struct {
	Beginner
	operation string
	armed     bool
	active    bool
}

// BeginTx tracks actual SQL lifetime so storage tests can reject transaction-spanning I/O.
func (d *lostReply) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, e := d.Beginner.BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	d.active = true
	return &lostReplyTx{Tx: tx, owner: d}, nil
}

// lostReplyTx suppresses a selected committed reply once without replaying its operation.
type lostReplyTx struct {
	pgx.Tx
	owner    *lostReply
	selected bool
}

// QueryRow identifies the selected boundary while executing the real restricted SQL function.
func (t *lostReplyTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	t.selected = strings.Contains(q, t.owner.operation)
	return t.Tx.QueryRow(ctx, q, args...)
}

// Commit first commits PostgreSQL, then injects the lost application acknowledgement.
func (t *lostReplyTx) Commit(ctx context.Context) error {
	e := t.Tx.Commit(ctx)
	t.owner.active = false
	if e == nil && t.owner.armed && t.selected {
		t.owner.armed = false
		return errors.New("synthetic lost COMMIT reply")
	}
	return e
}

// Rollback clears lifecycle tracking and uses the production adapter's cleanup context.
func (t *lostReplyTx) Rollback(ctx context.Context) error {
	e := t.Tx.Rollback(ctx)
	t.owner.active = false
	return e
}

// TestImportServicePostgresUnknownCommit reconciles the same immutable preview after a real lost reply.
func TestImportServicePostgresUnknownCommit(t *testing.T) {
	service, owner, runtime, request := proofFixture(t)
	lost := &lostReply{Beginner: runtime, operation: "publish_import_preview", armed: true}
	service.repo = &SQLRepository{DB: lost}
	base := service.storage
	checkOutside := func(context.Context) {
		if lost.active {
			t.Fatal("SQL transaction spans storage")
		}
	}
	service.storage = checkedStorage{base: base, onRead: checkOutside, onPut: checkOutside}
	registered, e := service.Register(context.Background(), testScope, "statement", request)
	if e != nil {
		t.Fatal(e)
	}
	batch := proofString(t, registered, "batch_id")
	result, e := service.Upload(context.Background(), testScope, batch, source(testCSV))
	if e == nil || e.Error() != "commit_unknown" || len(result) != 0 {
		t.Fatalf("unknown commit escaped: %v", e)
	}
	result, e = service.Preview(context.Background(), testScope, batch)
	if e != nil || proofString(t, result, "state") != "preview_ready" {
		t.Fatalf("reconciliation failed: %v", e)
	}
	var count int
	if e = owner.QueryRow(context.Background(), "select count(*) from banking.import_preview where batch_id=$1", batch).Scan(&count); e != nil {
		t.Fatal(e)
	}
	if count != 1 {
		t.Fatal("unknown reply duplicated preview")
	}
}

// TestImportServiceExplicitRecovery proves confirmation creates review evidence once and no journals.
func TestImportServiceExplicitRecovery(t *testing.T) {
	service, owner, runtime, request := proofFixture(t)
	var before int64
	if e := owner.QueryRow(context.Background(), "select revision from ledger.book where entity_id=$1 and book_id=$2", testEntity, testBook).Scan(&before); e != nil {
		t.Fatal(e)
	}
	registered, e := service.Register(context.Background(), testScope, "statement", request)
	if e != nil {
		t.Fatal(e)
	}
	batch := proofString(t, registered, "batch_id")
	preview, e := service.Upload(context.Background(), testScope, batch, source(testCSV))
	if e != nil {
		t.Fatal(e)
	}
	value := proofJSON(t, preview)
	body := map[string]any{"batch_id": batch, "profile": ingestion.Profile}
	for _, key := range []string{"preview_id", "preview_revision", "binding_revision", "source_revision"} {
		body[key] = value[key]
	}
	wire, e := json.Marshal(map[string]any{"operation": "import", "entity_id": testEntity, "book_id": testBook, "body": body})
	if e != nil {
		t.Fatal(e)
	}
	recovery := &ledger.Service{DB: runtime}
	command, e := recovery.Recovery(context.Background(), ledger.Command{Actor: testActor, Key: "explicit-import", Request: "00000000-0000-4000-8000-000000000040", Body: string(wire)}, "register")
	if e != nil {
		t.Fatal(e)
	}
	cid := proofString(t, command.Body, "command_id")
	for i := 0; i < 2; i++ {
		result, e := recovery.Recovery(context.Background(), ledger.Command{Actor: testActor, CommandID: cid, Request: fmt.Sprintf("00000000-0000-4000-8000-%012d", 41+i), Body: "{}"}, "execute")
		if e != nil {
			t.Fatal(e)
		}
		if proofString(t, result.Body, "state") != "committed" {
			t.Fatalf("unexpected terminal command: %s", result.Body)
		}
	}
	var observations, reviews, links, journals, outbox int
	var after int64
	var reviewID string
	if e = owner.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM banking.source_observation),(SELECT count(*) FROM banking.review_item),(SELECT count(*) FROM banking.import_row),(SELECT count(*) FROM ledger.journal_entry),(SELECT count(*) FROM integration.outbox_message),(SELECT revision FROM ledger.book),(SELECT review_id::text FROM banking.review_item LIMIT 1)`).Scan(&observations, &reviews, &links, &journals, &outbox, &after, &reviewID); e != nil {
		t.Fatal(e)
	}
	if observations != 1 || reviews != 1 || links != 1 || journals != 0 || outbox != 0 || after != before {
		t.Fatalf("explicit import effects mismatch %d/%d/%d journals=%d outbox=%d rev=%d/%d", observations, reviews, links, journals, outbox, before, after)
	}
	review, e := service.Review(context.Background(), testScope, reviewID)
	if e != nil {
		t.Fatal(e)
	}
	if proofString(t, review, "state") != "needs_review" {
		t.Fatal("import did not create review")
	}
	var evidence []struct {
		Amount string `json:"amount"`
	}
	if e = json.Unmarshal(proofJSON(t, review)["evidence"], &evidence); e != nil {
		t.Fatal(e)
	}
	if len(evidence) != 1 || evidence[0].Amount != "12.5" {
		t.Fatal("review cents mismatch")
	}
	proofExec(t, owner, "update banking.source_permission set revoked_at=clock_timestamp() where principal_id=$1 and capability='read'", testActor)
	if result, e := service.Review(context.Background(), testScope, reviewID); e == nil || len(result) != 0 {
		t.Fatal("revoked read leaked review")
	}
}

// TestImportServicePublicationAuthority rechecks real SQL permission after storage has completed.
func TestImportServicePublicationAuthority(t *testing.T) {
	service, owner, _, request := proofFixture(t)
	registered, e := service.Register(context.Background(), testScope, "statement", request)
	if e != nil {
		t.Fatal(e)
	}
	service.storage = checkedStorage{base: service.storage, onRead: func(context.Context) {
		proofExec(t, owner, "update banking.source_permission set revoked_at=clock_timestamp() where principal_id=$1 and capability='import'", testActor)
	}}
	body, e := service.Upload(context.Background(), testScope, proofString(t, registered, "batch_id"), source(testCSV))
	if e == nil || e.Error() != "not_found" || len(body) != 0 {
		t.Fatalf("revoked publication acknowledged: %v", e)
	}
	var count int
	if e = owner.QueryRow(context.Background(), "select count(*) from banking.import_preview").Scan(&count); e != nil {
		t.Fatal(e)
	}
	if count != 0 {
		t.Fatal("revoked preview committed")
	}
}

// TestImportServicePostgresDeadline cancels an actual blocked authorization query without reading bytes.
func TestImportServicePostgresDeadline(t *testing.T) {
	service, owner, _, request := proofFixture(t)
	registered, e := service.Register(context.Background(), testScope, "statement", request)
	if e != nil {
		t.Fatal(e)
	}
	tx, e := owner.Begin(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(context.Background(), "select 1 from ledger.principal where principal_id=$1 for update", testActor); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	input := source(testCSV)
	body, e := service.Upload(ctx, testScope, proofString(t, registered, "batch_id"), input)
	if e == nil || len(body) != 0 || input.reads != 0 || input.closes != 1 {
		t.Fatalf("deadline admission escaped: %v", e)
	}
}
