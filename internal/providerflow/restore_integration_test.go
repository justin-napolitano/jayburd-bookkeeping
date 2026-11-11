//go:build integration

package providerflow

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	osExec "os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/importer"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/server"
)

const restorePolicy = "00000000-0000-4000-8000-000000000060"
const restoreExpense = "00000000-0000-4000-8000-000000000051"
const restoreStatement = "source_account_ref,entry_id,booked_on,currency,direction,amount,description\r\nsynthetic,restore-a,2026-01-05,USD,outflow,0.10,Fictional restore purchase A\r\nsynthetic,restore-b,2026-01-06,USD,outflow,0.20,Fictional restore purchase B\r\n"

// restoreConnect refuses ambient DSNs; only owned databases on the disposable internal network are reachable.
func restoreConnect(t *testing.T, db, user string) *pgx.Conn {
	t.Helper()
	if !*proof {
		t.Skip("use make provider-flow-test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := pgx.Connect(ctx, fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if err != nil {
		t.Fatal("isolated synthetic connection failed")
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

// restoreDatabase creates and cleans only its own random fixture; an empty template is never mutated.
func restoreDatabase(t *testing.T, template bool) string {
	t.Helper()
	admin := restoreConnect(t, "postgres", "postgres")
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal("fixture identity failed")
	}
	db := "provider_restore_" + hex.EncodeToString(suffix[:])
	query := "CREATE DATABASE " + pgx.Identifier{db}.Sanitize()
	if template {
		query += " TEMPLATE provider_flow_pristine"
	}
	exec(t, admin, query)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{db}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Error("owned synthetic database cleanup failed")
		}
	})
	return db
}

// restoreNewDB is shared only by test lanes needing an independently owned empty migrated fixture.
func restoreNewDB(t *testing.T) string { t.Helper(); return restoreDatabase(t, true) }

type restoreFlowFixture struct {
	admin, runtime    *pgx.Conn
	app               *server.Server
	sourceID, batchID string
	store             *staging.Store
	original          []byte
}

// restoreObject decodes identifiers and exact decimal strings without float conversion.
func restoreObject(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var value map[string]json.RawMessage
	if json.Unmarshal(raw, &value) != nil {
		t.Fatal("invalid synthetic response")
	}
	return value
}

// restoreOwner provisions reviewed fixture grants via the actual NOINHERIT operator session.
func restoreOwner(t *testing.T, c *pgx.Conn, query string, args ...any) map[string]json.RawMessage {
	t.Helper()
	ctx := context.Background()
	tx, err := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal("fixture owner begin failed")
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner"); err != nil {
		t.Fatal("fixture owner role failed")
	}
	var raw []byte
	if err = tx.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		t.Fatal("fixture owner operation failed", err)
	}
	if tx.Commit(ctx) != nil {
		t.Fatal("fixture owner commit failed")
	}
	return restoreObject(t, raw)
}

// restoreApp uses the real public artifact-only transport and restricted SQL service; no listener/auth emulator is installed.
func restoreApp(t *testing.T, runtime *pgx.Conn) *server.Server {
	t.Helper()
	app, err := server.New(server.Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "http://localhost:3000", Project: "synthetic-project", DatabaseURL: "synthetic"}, server.Dependencies{Auth: identity{}, Operations: &ledger.Service{DB: runtime}, Ready: func(context.Context) error { return nil }})
	if err != nil {
		t.Fatal("synthetic public composition failed")
	}
	return app
}

// restoreBytes transfers only explicit fictional statement content to the production storage/decoder API.
type restoreBytes struct{ *bytes.Reader }

func (s restoreBytes) Read(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.Reader.Read(p)
}
func (restoreBytes) Close(context.Context) error { return nil }

// restoreFixture creates metadata only with privileged fixture setup; every import/post effect uses restricted APIs.
func restoreFixture(t *testing.T, db, statement string) restoreFlowFixture {
	t.Helper()
	admin := restoreConnect(t, db, "postgres")
	exec(t, admin, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','restore-owner')`, actor)
	exec(t, admin, `INSERT INTO ledger.entity(entity_id,name,kind) VALUES($1,'Fictional restore LLC','legal')`, entity)
	exec(t, admin, `INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$2,'prepare',NULL),($1,$2,'post',NULL)`, entity, actor)
	exec(t, admin, `INSERT INTO ledger.book(entity_id,book_id,name) VALUES($1,$2,'Fictional restored book')`, entity, book)
	exec(t, admin, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic')`, currency)
	exec(t, admin, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Fictional bank','asset','debit',true,'2026-01-01',NULL),($1,$2,$4,'5000','Fictional expenses','expense','debit',false,'2026-01-01',NULL)`, entity, book, account, restoreExpense)
	exec(t, admin, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())`, entity, book, restorePolicy, currency, actor)
	exec(t, admin, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000061','2026-01-01','2026-02-01')`, entity, book)
	operator := restoreConnect(t, db, "synthetic_operator")
	manifest := `{"ledger_account_id":"` + account + `","currency_version_id":"` + currency + `","binding_date":"2026-01-01","binding_evidence_ref":"` + claim + `","reader_principal_id":"` + actor + `","namespace":{"profile":"` + ingestion.Profile + `","synthetic_dataset_id":"00000000-0000-4000-8000-000000000031","source_account_ref":"synthetic"}}`
	registered := restoreOwner(t, operator, "select banking.register_source($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)", actor, entity, book, "restore-source", manifest)
	sid := stringField(t, registered["source_account_id"])
	var namespace string
	if admin.QueryRow(context.Background(), "SELECT namespace_id::text FROM banking.source_namespace WHERE source_account_id=$1", sid).Scan(&namespace) != nil {
		t.Fatal("fixture namespace missing")
	}
	permission := `{"expected_binding_revision":"1","principal_id":"` + actor + `","enabled":true,"entity_import_enabled":true}`
	granted := restoreOwner(t, operator, "select banking.set_source_import_permission($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text,$6::text)", actor, entity, book, sid, "restore-import-grant", permission)
	runtime := restoreConnect(t, db, "synthetic_runtime")
	var forbidden int
	err := runtime.QueryRow(context.Background(), "SELECT count(*) FROM ledger.journal_entry").Scan(&forbidden)
	var denied *pgconn.PgError
	if !errors.As(err, &denied) || denied.Code != "42501" {
		t.Fatal("runtime unexpectedly has direct financial table access")
	}
	dir := t.TempDir()
	if os.Chmod(dir, 0700) != nil {
		t.Fatal("synthetic staging mode failed")
	}
	store, err := staging.Open(dir)
	if err != nil {
		t.Fatal("synthetic staging failed")
	}
	t.Cleanup(func() { _ = store.Close() })
	service := importer.New(runtime, store)
	scope := importer.Scope{Actor: actor, Entity: entity, Book: book}
	batch, err := service.Register(context.Background(), scope, "restore-statement", importer.Registration{SourceAccountID: sid, NamespaceID: namespace, BindingRevision: stringField(t, granted["binding_revision"]), StatementRef: "RESTORE", StartsOn: "2026-01-01", EndsBefore: "2026-02-01", Profile: ingestion.Profile})
	if err != nil {
		t.Fatal("restricted import registration failed", err)
	}
	batchID := stringField(t, restoreObject(t, batch)["batch_id"])
	preview, err := service.Upload(context.Background(), scope, batchID, restoreBytes{bytes.NewReader([]byte(statement))})
	if err != nil {
		t.Fatal("restricted source decoding failed", err)
	}
	v := restoreObject(t, preview)
	if stringField(t, v["state"]) != "preview_ready" {
		t.Fatal("source did not reach reviewed preview")
	}
	app := restoreApp(t, runtime)
	body := map[string]any{"batch_id": batchID, "profile": ingestion.Profile}
	for _, key := range []string{"preview_id", "preview_revision", "binding_revision", "source_revision"} {
		body[key] = v[key]
	}
	command := request(t, app, "POST", "/v1/commands", "restore-import", map[string]any{"operation": "import", "entity_id": entity, "book_id": book, "body": body}, 201)
	id := stringField(t, command["command_id"])
	for _, action := range []string{"execute", "acknowledge"} {
		request(t, app, "POST", "/v1/commands/"+id+"/"+action, "", map[string]any{}, 200)
	}
	restoreOwner(t, operator, "select banking.set_source_review_permission($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text,$6::text)", actor, entity, book, sid, "restore-review-grant", `{"expected_binding_revision":"2","principal_id":"`+actor+`","enabled":true}`)
	return restoreFlowFixture{admin: admin, runtime: runtime, app: app, sourceID: sid, batchID: batchID, store: store, original: []byte(statement)}
}

// restorePrepare reviews current API facts before saving an immutable source_post command, never a manual post.
func restorePrepare(t *testing.T, f restoreFlowFixture, reviewID, key string) map[string]json.RawMessage {
	t.Helper()
	base := "/v1/entities/" + entity + "/books/" + book + "/review-items/" + reviewID
	detail := request(t, f.app, "GET", base, "", nil, 200)
	categories := request(t, f.app, "GET", base+"/categories", "", nil, 200)
	if stringField(t, categories["policy_id"]) != restorePolicy || stringField(t, categories["functional_currency"]) != "USD" || string(categories["fx_required"]) != "false" {
		t.Fatal("unexpected source category authority")
	}
	input := map[string]any{"category_account_id": restoreExpense, "policy_id": restorePolicy, "description": "Fictional reviewed source posting", "accounting_date": detail["booked_date"]}
	for _, key := range []string{"review_revision", "source_revision", "binding_revision", "book_revision"} {
		name := key
		if key == "review_revision" || key == "book_revision" {
			name = "expected_" + key
		}
		input[name] = detail[key]
	}
	draft := request(t, f.app, "POST", base+"/drafts", key+"-draft", input, 201)
	body := map[string]any{"expected_book_revision": draft["book_revision"]}
	for _, field := range []string{"review_id", "draft_id", "review_revision", "source_revision", "binding_revision", "posting_body"} {
		body[field] = draft[field]
	}
	return request(t, f.app, "POST", "/v1/commands", key, map[string]any{"operation": "source_post", "entity_id": entity, "book_id": book, "body": body}, 201)
}

// restoreDump restores only the named owned synthetic database; dumps never reach disk or logs.
func restoreDump(t *testing.T, source, target string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dump := osExec.CommandContext(ctx, "/usr/local/bin/pg_dump", "-h", "database", "-p", "5432", "-U", "postgres", "--no-password", "-Fc", source)
	dump.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=/tmp"}
	raw, err := dump.Output()
	if err != nil {
		t.Fatal("owned synthetic dump failed")
	}
	if len(raw) == 0 || len(raw) > 64<<20 {
		t.Fatal("synthetic dump bounds")
	}
	restore := osExec.CommandContext(ctx, "/usr/local/bin/pg_restore", "-h", "database", "-p", "5432", "-U", "postgres", "--no-password", "--exit-on-error", "-d", target)
	restore.Env = dump.Env
	restore.Stdin = bytes.NewReader(raw)
	if restore.Run() != nil {
		t.Fatal("owned synthetic restore failed")
	}
}

// restoreSignature compares retained identities and evidence independently of display or generator assertions.
func restoreSignature(t *testing.T, c *pgx.Conn) string {
	t.Helper()
	sum := sha256.New()
	for _, table := range []string{"banking.source_account", "banking.source_namespace", "banking.source_permission", "banking.import_batch", "banking.import_preview", "banking.source_observation", "banking.observation_revision", "banking.import_row", "banking.review_item", "banking.review_decision", "banking.review_evidence", "banking.review_assignment", "banking.review_draft", "banking.review_posting", "ledger.book", "ledger.interactive_command", "ledger.interactive_command_reservation", "ledger.journal_entry", "ledger.journal_line", "integration.operation_receipt", "integration.outbox_message", "audit.command_action"} {
		value := scalar(t, c, "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY to_jsonb(r)::text),'[]'::jsonb)::text FROM "+table+" r")
		_, _ = io.WriteString(sum, table+"\n"+value+"\n")
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// restoreCounts supplies fixed expectations; identity equality alone cannot prove correct money or effects.
func restoreCounts(t *testing.T, c *pgx.Conn, posted bool) {
	t.Helper()
	want := "0"
	if posted {
		want = "1"
	}
	for _, table := range []string{"ledger.journal_entry", "banking.review_posting", "integration.operation_receipt", "integration.outbox_message"} {
		if scalar(t, c, "SELECT count(*)::text FROM "+table) != want {
			t.Fatal("unexpected source posting count", table)
		}
	}
	if scalar(t, c, "SELECT count(*)::text FROM banking.source_observation") != "2" || scalar(t, c, "SELECT count(*)::text FROM banking.review_draft") != "1" || scalar(t, c, "SELECT trim_scale(sum(amount))::text FROM banking.observation_revision") != "0.3" {
		t.Fatal("restored independent source oracle failed")
	}
	if scalar(t, c, "SELECT string_agg(amount_lexeme,',' ORDER BY amount_lexeme) FROM banking.observation_revision") != "0.10,0.20" {
		t.Fatal("original source units changed")
	}
	if posted && (scalar(t, c, "SELECT count(*)::text FROM ledger.journal_line") != "2" || scalar(t, c, "SELECT trim_scale(sum(functional_amount))::text FROM ledger.journal_line WHERE side='debit'") != "0.1" || scalar(t, c, "SELECT trim_scale(sum(functional_amount))::text FROM ledger.journal_line WHERE side='credit'") != "0.1") {
		t.Fatal("restored posting cents changed")
	}
}

// restoreBalance uses a currently authorized real report; exact strings are fixed independently of source decoding.
func restoreBalance(t *testing.T, app *server.Server, posted bool) map[string]json.RawMessage {
	t.Helper()
	r := request(t, app, "GET", "/v1/entities/"+entity+"/books/"+book+"/reports/trial-balance?through_date=2026-01-31", "", nil, 200)
	want := "0"
	if posted {
		want = "0.1"
	}
	var totals map[string]string
	if json.Unmarshal(r["totals"], &totals) != nil || stringField(t, r["functional_currency"]) != "USD" || totals["total_debits"] != want || totals["total_credits"] != want {
		t.Fatal("independent restored balance oracle failed")
	}
	delete(r, "generated_at") // Report generation time is not retained financial identity.
	return r
}

// TestProviderSourceRestore proves the complete retained evidence/command/linked posting boundary across synthetic restore.
func TestProviderSourceRestore(t *testing.T) {
	for _, state := range []string{"prepared", "committed"} {
		t.Run(state, func(t *testing.T) {
			original := restoreNewDB(t)
			f := restoreFixture(t, original, restoreStatement)
			base := "/v1/entities/" + entity + "/books/" + book
			page := request(t, f.app, "GET", base+"/review-items?source_account_id="+f.sourceID+"&state=needs_review", "", nil, 200)
			var items []map[string]json.RawMessage
			if json.Unmarshal(page["items"], &items) != nil || len(items) != 2 {
				t.Fatal("source review inventory changed")
			}
			var reviewID string
			for _, item := range items {
				if stringField(t, item["description"]) == "Fictional restore purchase A" {
					reviewID = stringField(t, item["review_id"])
				}
			}
			if reviewID == "" {
				t.Fatal("retained source review missing")
			}
			command := restorePrepare(t, f, reviewID, "restore-source-post")
			id := stringField(t, command["command_id"])
			posted := state == "committed"
			if posted {
				command = request(t, f.app, "POST", "/v1/commands/"+id+"/execute", "", map[string]any{}, 200)
			}
			restoreCounts(t, f.admin, posted)
			balance := restoreBalance(t, f.app, posted)
			before := restoreSignature(t, f.admin)
			var objectID, expectedHash string
			if f.admin.QueryRow(context.Background(), "SELECT object_id::text,byte_sha256 FROM banking.import_batch WHERE batch_id=$1", f.batchID).Scan(&objectID, &expectedHash) != nil {
				t.Fatal("retained object identity missing")
			}
			ref := staging.Reference{Scope: staging.Scope{EntityID: entity, BookID: book}, ObjectID: objectID}
			evidence, receipt, err := f.store.Read(context.Background(), ref)
			if err != nil || !bytes.Equal(evidence, []byte(restoreStatement)) || hex.EncodeToString(receipt.SHA256[:]) != expectedHash {
				t.Fatal("retained original statement evidence changed")
			}
			target := restoreDatabase(t, false)
			restoreDump(t, original, target)
			admin := restoreConnect(t, target, "postgres")
			if restoreSignature(t, admin) != before {
				t.Fatal("source/evidence/command/journal identities changed on restore")
			}
			restoreCounts(t, admin, posted)
			runtime := restoreConnect(t, target, "synthetic_runtime")
			app := restoreApp(t, runtime)
			if !reflect.DeepEqual(balance, restoreBalance(t, app, posted)) {
				t.Fatal("report identity or exact money changed on restore")
			}
			// The DB backup retains the object reference/hash; local object bytes are a separate explicit synthetic recovery input.
			dir := t.TempDir()
			if os.Chmod(dir, 0700) != nil {
				t.Fatal("restored synthetic storage mode failed")
			}
			store, err := staging.Open(dir)
			if err != nil {
				t.Fatal("restored synthetic storage failed")
			}
			defer store.Close()
			if _, err = store.Put(context.Background(), ref, restoreBytes{bytes.NewReader(evidence)}); err != nil {
				t.Fatal("restored original object failed")
			}
			raw, recoveredReceipt, err := store.Read(context.Background(), ref)
			if err != nil || !bytes.Equal(raw, []byte(restoreStatement)) || recoveredReceipt != receipt {
				t.Fatal("restored raw evidence integrity failed")
			}
			pending := request(t, app, "GET", "/v1/commands/pending", "", nil, 200)
			if !reflect.DeepEqual(restoreObject(t, pending["command"]), command) {
				t.Fatal("prepared or unacknowledged command changed on restore")
			}
			for range 2 {
				executed := request(t, app, "POST", "/v1/commands/"+id+"/execute", "", map[string]any{}, 200)
				if posted && !reflect.DeepEqual(executed, command) {
					t.Fatal("historical source receipt replay changed")
				}
				command = executed
				posted = true
				restoreCounts(t, admin, true)
				restoreBalance(t, app, true)
			}
			request(t, app, "POST", "/v1/commands/"+id+"/acknowledge", "", map[string]any{}, 200)
			if string(request(t, app, "GET", "/v1/commands/pending", "", nil, 200)["command"]) != "null" {
				t.Fatal("acknowledged recovery did not release gate")
			}
			request(t, app, "POST", "/v1/commands/"+id+"/execute", "", map[string]any{}, 200)
			restoreCounts(t, admin, true)
			journal := restoreObject(t, command["result"])
			jid := stringField(t, journal["journal_id"])
			if !reflect.DeepEqual(journal, request(t, app, "GET", base+"/journals/"+jid, "", nil, 200)) {
				t.Fatal("restored immutable journal differs from receipt")
			}
		})
	}
}
