//go:build integration

package server

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

var demoAcquisitionProof = flag.Bool("demo-acquisition-proof", false, "Run only disposable acquisition SQL proof")

// TestDemoAcquisitionGenerator checks independent cent totals and exact identities across all3000 rows.
func TestDemoAcquisitionGenerator(t *testing.T) {
	for i := 0; i < 3; i++ {
		reference := fmt.Sprintf("DEMO-BANK-%d", i+1)
		statement, e := ingestion.Decode(strings.NewReader(string(demoCSV(i, reference))), ingestion.Expectations{SourceAccountRef: reference, StartsOn: "2026-01-01", EndsBefore: "2026-02-01", Currency: ingestion.Currency{Code: "USD", VersionID: demoImportUSD, MinorUnit: 2, ValidFrom: "2000-01-01"}})
		if e != nil {
			t.Fatal(e)
		}
		if len(statement.Rows) != 1000 {
			t.Fatal("missing generated rows")
		}
		inflow, outflow := 0, 0
		total := 0
		for _, row := range statement.Rows {
			parts := strings.Split(row.Amount, ".")
			if len(parts) != 2 || len(parts[1]) != 2 {
				t.Fatal("not exact cents")
			}
			whole, e1 := strconv.Atoi(parts[0])
			fraction, e2 := strconv.Atoi(parts[1])
			if e1 != nil || e2 != nil {
				t.Fatal("invalid generated amount")
			}
			cents := whole*100 + fraction
			if row.Direction == "inflow" {
				inflow++
				total += cents
			} else if row.Direction == "outflow" {
				outflow++
				total -= cents
			} else {
				t.Fatal("unknown direction")
			}
		}
		if inflow != 500 || outflow != 500 || total != 3749500 {
			t.Fatalf("independent cent oracle failed %d/%d/%d", inflow, outflow, total)
		}
	}
}

// TestDemoAcquisitionHTTPBoundary rejects missing identity or arbitrary input before database access.
func TestDemoAcquisitionHTTPBoundary(t *testing.T) {
	d := &demoAcquisition{}
	for _, sample := range []struct {
		method, path, body, token string
		want                      int
	}{{"GET", "/v1/demo/acquisition", "", "", 401}, {"POST", "/v1/demo/acquisition/step", "{}", "wrong", 401}, {"POST", "/v1/demo/acquisition/step", `{"scope":"caller"}`, "synthetic.browser.proof", 400}, {"POST", "/v1/demo/acquisition/step", "{}", "synthetic.browser.proof", 409}, {"GET", "/v1/demo/acquisition?scope=caller", "", "synthetic.browser.proof", 400}} {
		d.gate.Lock()
		r := httptest.NewRequest(sample.method, sample.path, strings.NewReader(sample.body))
		if sample.token != "" {
			r.Header.Set("Authorization", "Bearer "+sample.token)
		}
		w := httptest.NewRecorder()
		d.ServeHTTP(w, r)
		d.gate.Unlock()
		if w.Code != sample.want {
			t.Fatalf("boundary %s: %d", sample.path, w.Code)
		}
	}
}

// demoProofExec creates or inspects fixed synthetic fixtures with the test administrator only.
func demoProofExec(t *testing.T, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, e := c.Exec(context.Background(), q, args...); e != nil {
		t.Fatal(e)
	}
}

// demoLoseCommit tracks an actual successful transition and loses its application reply once.
type demoLoseCommit struct {
	ledger.Beginner
	armed bool
}

// BeginTx wraps only the owned runtime transaction, preserving its isolation options.
func (d *demoLoseCommit) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, e := d.Beginner.BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	return &demoLoseTx{Tx: tx, owner: d}, nil
}

// demoLoseTx selects the real execute transition rather than a read or setup operation.
type demoLoseTx struct {
	pgx.Tx
	owner   *demoLoseCommit
	execute bool
}

// QueryRow recognizes the adapter's exact execute action while delegating actual restricted SQL.
func (t *demoLoseTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	t.execute = strings.Contains(q, "transition_command") && len(args) > 3 && args[3] == "execute"
	return t.Tx.QueryRow(ctx, q, args...)
}

// Commit suppresses one successful reply; no transport or hardware failure guarantee is claimed.
func (t *demoLoseTx) Commit(ctx context.Context) error {
	e := t.Tx.Commit(ctx)
	if e == nil && t.execute && t.owner.armed {
		t.owner.armed = false
		return errors.New("lost synthetic acknowledgement")
	}
	return e
}

// TestDemoAcquisitionPostgres loads all3000 through real import/recovery, survives lost replies and restart.
func TestDemoAcquisitionPostgres(t *testing.T) {
	if !*demoAcquisitionProof {
		t.Skip("disposable acquisition proof only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	admin, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(context.Background())
	demoProofExec(t, admin, "CREATE DATABASE demo_acquisition_proof TEMPLATE app_proof")
	demoProofExec(t, admin, "CREATE ROLE demo_acquisition_runtime LOGIN IN ROLE bookkeeping_api")
	owner, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=demo_acquisition_proof sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer owner.Close(context.Background())
	demoProofExec(t, owner, "INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','demo-acquisition')", demoImportActor)
	demoProofExec(t, owner, "INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic')", demoImportUSD)
	pool, e := pgxpool.New(ctx, "host=database port=5432 user=demo_acquisition_runtime dbname=demo_acquisition_proof sslmode=disable connect_timeout=5 pool_max_conns=4")
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	path := t.TempDir()
	if e = os.Chmod(path, 0700); e != nil {
		t.Fatal(e)
	}
	store, e := staging.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	d, e := bootstrapDemoAcquisition(ctx, owner, pool, store)
	if e != nil {
		t.Fatal(e)
	}
	status, e := d.status(ctx)
	if e != nil || status.Imported != "0" || status.Complete || status.Blocked || len(status.Batches) != 3 {
		t.Fatalf("startup must not import: %+v %v", status, e)
	}
	var before string
	if e = owner.QueryRow(ctx, "select string_agg(revision::text,',' order by book_id) from ledger.book").Scan(&before); e != nil {
		t.Fatal(e)
	}
	// An unrelated interactive setup remains prepared; the loader cannot execute or acknowledge it.
	unrelated := map[string]any{"operation": "setup", "entity_id": nil, "book_id": nil, "body": map[string]any{"entity_name": "Unrelated manual request", "entity_kind": "legal", "book_name": "Unrelated", "starts_on": "2026-01-01", "ends_before": "2027-01-01", "functional_currency_version_id": demoImportUSD, "accounts": []map[string]any{demoAccount("1000", "Cash", "asset", "debit", true), demoAccount("3000", "Capital", "equity", "credit", false)}}}
	command, e := d.ledger.Recovery(ctx, ledger.Command{Actor: demoImportActor, Key: "unrelated", Request: demoID(900), Body: demoJSON(unrelated)}, "register")
	if e != nil {
		t.Fatal(e)
	}
	var pending demoPending
	if e = json.Unmarshal(command.Body, &pending); e != nil {
		t.Fatal(e)
	}
	if s, e := d.step(ctx); e == nil || !s.Blocked || s.Imported != "0" {
		t.Fatal("unrelated pending command did not block")
	}
	for _, action := range []string{"cancel", "acknowledge"} {
		if _, e = d.ledger.Recovery(ctx, ledger.Command{Actor: demoImportActor, CommandID: pending.ID, Request: demoID(901), Body: "{}"}, action); e != nil {
			t.Fatal(e)
		}
	}
	lost := &demoLoseCommit{Beginner: pool, armed: true}
	d.ledger = &ledger.Service{DB: lost}
	sawLost := false
	for step := 0; step < 20; step++ {
		status, e = d.step(ctx)
		if e != nil {
			if lost.armed || sawLost {
				t.Fatalf("unexpected step failure: %v", e)
			}
			sawLost = true
			status, e = d.status(ctx)
			if e != nil || status.Imported != "1000" {
				t.Fatalf("lost commit not recoverable: %+v %v", status, e)
			}
			// Reconstruct all identities from durable receipts using the same private object store.
			d, e = bootstrapDemoAcquisition(ctx, owner, pool, store)
			if e != nil {
				t.Fatal(e)
			}
			continue
		}
		if status.Complete {
			break
		}
	}
	if !sawLost || !status.Complete || status.Imported != "3000" || status.Pending != nil {
		t.Fatalf("incomplete durable acquisition: %+v", status)
	}
	if status, e = d.step(ctx); e != nil || !status.Complete {
		t.Fatal("completion replay failed")
	}
	var observations, reviews, links, journals, outbox int
	var net, after string
	e = owner.QueryRow(ctx, `SELECT (SELECT count(*) FROM banking.source_observation),(SELECT count(*) FROM banking.review_item),(SELECT count(*) FROM banking.import_row),(SELECT count(*) FROM ledger.journal_entry),(SELECT count(*) FROM integration.outbox_message),(SELECT sum(CASE direction WHEN 'inflow' THEN amount ELSE -amount END)::text FROM banking.observation_revision),(SELECT string_agg(revision::text,',' order by book_id) FROM ledger.book)`).Scan(&observations, &reviews, &links, &journals, &outbox, &net, &after)
	if e != nil {
		t.Fatal(e)
	}
	if observations != 3000 || reviews != 3000 || links != 3000 || journals != 0 || outbox != 0 || after != before || net != "112485.000000000" {
		t.Fatalf("durable oracle mismatch obs=%d review=%d links=%d journals=%d outbox=%d net=%s rev=%s/%s", observations, reviews, links, journals, outbox, net, before, after)
	}
	var correct bool
	if e = owner.QueryRow(ctx, `SELECT count(*)=3 AND bool_and(n=1000 AND inflows=500 AND outflows=500 AND net=37495.00) FROM (SELECT entity_id,count(*) n,count(*) FILTER(WHERE direction='inflow') inflows,count(*) FILTER(WHERE direction='outflow') outflows,sum(CASE direction WHEN 'inflow' THEN amount ELSE -amount END) net FROM banking.observation_revision GROUP BY entity_id) q`).Scan(&correct); e != nil || !correct {
		t.Fatalf("entity/cents separation failed %v", e)
	}
	// A loaded review is actually categorizable by the restricted runtime actor.
	first := d.batches[0]
	var reviewID string
	if e = owner.QueryRow(ctx, "select review_id::text from banking.review_item where entity_id=$1 and book_id=$2 order by review_id limit 1", first.Entity, first.Book).Scan(&reviewID); e != nil {
		t.Fatal(e)
	}
	detailRaw := demoRuntimeJSON(t, ctx, pool, "select banking.get_review_item($1::uuid,$2::uuid,$3::uuid,$4::uuid)", demoImportActor, first.Entity, first.Book, reviewID)
	var detail struct {
		Review  string `json:"review_revision"`
		Source  string `json:"source_revision"`
		Binding string `json:"binding_revision"`
		Book    string `json:"book_revision"`
	}
	if e = json.Unmarshal(detailRaw, &detail); e != nil {
		t.Fatal(e)
	}
	categoryRaw := demoRuntimeJSON(t, ctx, pool, "select banking.review_categories($1::uuid,$2::uuid,$3::uuid,$4::uuid)", demoImportActor, first.Entity, first.Book, reviewID)
	var categories struct {
		Policy   string `json:"policy_id"`
		Date     string `json:"accounting_date"`
		Accounts []struct {
			ID   string `json:"account_id"`
			Code string `json:"code"`
		} `json:"accounts"`
	}
	if e = json.Unmarshal(categoryRaw, &categories); e != nil {
		t.Fatal(e)
	}
	offset := ""
	for _, account := range categories.Accounts {
		if account.Code == "5000" {
			offset = account.ID
		}
	}
	if offset == "" {
		t.Fatal("missing active category account")
	}
	body := demoJSON(map[string]any{"expected_review_revision": detail.Review, "source_revision": detail.Source, "binding_revision": detail.Binding, "expected_book_revision": detail.Book, "category_account_id": offset, "policy_id": categories.Policy, "accounting_date": categories.Date, "description": "Explicit fictional category review"})
	draft := demoRuntimeJSON(t, ctx, pool, "select banking.create_review_draft($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text,$6::text)", demoImportActor, first.Entity, first.Book, reviewID, "demo-proof-draft", body)
	replay := demoRuntimeJSON(t, ctx, pool, "select banking.create_review_draft($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text,$6::text)", demoImportActor, first.Entity, first.Book, reviewID, "demo-proof-draft", body)
	if string(draft) != string(replay) {
		t.Fatal("draft replay changed immutable receipt")
	}
	if e = owner.QueryRow(ctx, `SELECT (SELECT count(*) FROM ledger.journal_entry)=0 AND (SELECT count(*) FROM integration.outbox_message)=0 AND (SELECT count(*) FROM banking.review_draft)=1 AND (SELECT state FROM banking.review_item WHERE review_id=$1)='draft_ready' AND (SELECT string_agg(revision::text,',' order by book_id) FROM ledger.book)=$2 AND (SELECT sum(CASE direction WHEN 'inflow' THEN amount ELSE -amount END) FROM banking.observation_revision)=112485.00`, reviewID, before).Scan(&correct); e != nil || !correct {
		t.Fatalf("draft changed accounting effects: %v", e)
	}
	// No HTTP request can replace the generated CSV with caller financial input.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/demo/acquisition/step", strings.NewReader(`{"csv":"caller"}`))
	r.Header.Set("Authorization", "Bearer synthetic.browser.proof")
	d.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("arbitrary input accepted")
	}
}

// demoRuntimeJSON exercises restricted SQL APIs without borrowing administrator authority.
func demoRuntimeJSON(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) []byte {
	t.Helper()
	tx, e := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	var result []byte
	if e = tx.QueryRow(ctx, query, args...).Scan(&result); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	return result
}
