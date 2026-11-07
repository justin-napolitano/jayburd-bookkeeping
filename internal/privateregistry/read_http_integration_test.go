//go:build integration

package privateregistry

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionread"
)

// readHTTPProof gates this binary to the disposable runner; it never accepts a DSN.
var readHTTPProof = flag.Bool("collection-read-proof", false, "Use only the disposable private registry template")

const readActor = "00000000-0000-4000-8000-000000000101"
const readOther = "00000000-0000-4000-8000-000000000102"

// readProofAuth confines fictional identity to the integration-tagged test binary.
type readProofAuth struct{}

// Authenticate independently maps one of two fixed tokens; request JSON never selects the actor.
func (readProofAuth) Authenticate(_ context.Context, token string) (string, error) {
	switch token {
	case "synthetic-owner":
		return readActor, nil
	case "synthetic-other":
		return readOther, nil
	}
	return "", errors.New("synthetic identity rejected")
}

// readProofConnect accepts only the disposable runner's fixed internal hostname and database.
func readProofConnect(t *testing.T, db, user string) *pgx.Conn {
	t.Helper()
	c, err := pgx.Connect(context.Background(), "host=database port=5432 user="+user+" dbname="+db+" sslmode=disable connect_timeout=5")
	if err != nil {
		t.Fatal("synthetic database unavailable", err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

// readProofCall executes the real restricted entrypoint in an acknowledged bounded transaction.
func readProofCall(t *testing.T, c *pgx.Conn, operator bool, q string, args ...any) map[string]json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if operator {
		if _, err = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner"); err != nil {
			t.Fatal(err)
		}
	}
	var body []byte
	if err = tx.QueryRow(ctx, q, args...).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if json.Unmarshal(body, &result) != nil {
		t.Fatal("invalid synthetic receipt")
	}
	return result
}

// readProofText extracts opaque receipt IDs only; financial expectations remain fixed literals.
func readProofText(t *testing.T, v json.RawMessage) string {
	t.Helper()
	var text string
	if json.Unmarshal(v, &text) != nil {
		t.Fatal("invalid fixture text")
	}
	return text
}

// TestCollectionReadHTTPPostgres proves identity/framing/real role/snapshot/money and revocation together.
func TestCollectionReadHTTPPostgres(t *testing.T) {
	if !*readHTTPProof {
		t.Skip("use disposable private registry runner")
	}
	admin := readProofConnect(t, "postgres", "postgres")
	if _, err := admin.Exec(context.Background(), "CREATE DATABASE private_http_read TEMPLATE private_template"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE private_http_read"); err != nil {
			t.Error("synthetic database cleanup failed", err)
		}
	})
	owner := readProofConnect(t, "private_http_read", "postgres")
	if _, err := owner.Exec(context.Background(), `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic-http','owner'),($2,'synthetic-http','other');`, readActor, readOther); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(context.Background(), `INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000105','USD','840',2,'2000-01-01',NULL,'synthetic-http'),('00000000-0000-4000-8000-000000000110','EUR',NULL,2,'2000-01-01',NULL,'synthetic-http')`); err != nil {
		t.Fatal(err)
	}
	operator := readProofConnect(t, "private_http_read", "synthetic_operator")
	grant := readProofCall(t, operator, true, "SELECT banking.provision_collection_owner($1,'http-owner',$2)", readActor, `{"expected_revision":null,"capabilities":["read","acquire","classify"],"evidence_ref":"00000000-0000-4000-8000-000000000106"}`)
	workspace := readProofText(t, grant["workspace_id"])
	runtime := readProofConnect(t, "private_http_read", "synthetic_runtime")
	registered := readProofCall(t, runtime, false, "SELECT banking.register_collection_item($1,$2,'http-register',$3)", readActor, workspace, `{"provider":"plaid","environment":"sandbox","item_id":"SYNTHETIC-HTTP-ITEM","credential_ref":"00000000-0000-4000-8000-000000000107","credential_version":"1","consent_evidence_ref":"00000000-0000-4000-8000-000000000108","consent_expires_at":"2030-01-01T00:00:00Z","accounts":[{"account_id":"SYNTHETIC-HTTP-ACCOUNT","display_name":"Fictional account"}]}`)
	var sources []map[string]string
	if json.Unmarshal(registered["sources"], &sources) != nil || len(sources) != 1 {
		t.Fatal("invalid registration receipt")
	}
	source := sources[0]["source_account_id"]
	publish := func(key, sourceRevision, operation, amount, cursor string) {
		t.Helper()
		body := `{"expected_item_generation":"1","expected_source_revision":"` + sourceRevision + `","starting_cursor":null,"next_cursor":"` + cursor + `","readiness":"INITIAL_UPDATE_COMPLETE","changes":[{"operation":"` + operation + `","transaction_id":"SYNTHETIC-HTTP-TX","booked_date":"2026-01-02","description":"Fictional source","merchant_name":null,"source_state":"booked","currency_code":"USD","amount":"` + amount + `","direction":"outflow"}]}`
		if sourceRevision != "0" {
			body = strings.Replace(body, `"starting_cursor":null`, `"starting_cursor":"cursor-1"`, 1)
		}
		readProofCall(t, runtime, false, "SELECT banking.publish_collection_batch($1,$2,$3,$4,$5)", readActor, workspace, source, key, body)
	}
	publish("http-publish-1", "0", "added", "9007199254740993.01", "cursor-1")
	pool, err := pgxpool.New(context.Background(), "host=database port=5432 user=synthetic_runtime dbname=private_http_read sslmode=disable connect_timeout=5 pool_max_conns=2")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	handler := collectionread.Handler(readProofAuth{}, &collectionread.Service{DB: pool})
	request := func(token, body string, want int) map[string]json.RawMessage {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, "/v1/collections/"+workspace+"/observations/query", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("HTTP status%d want%d: %s", w.Code, want, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "SYNTHETIC-HTTP-ITEM") || strings.Contains(w.Body.String(), "credential") {
			t.Fatal("provider custody disclosed")
		}
		var result map[string]json.RawMessage
		if json.Unmarshal(w.Body.Bytes(), &result) != nil {
			t.Fatal("invalid JSON result")
		}
		return result
	}
	selector := `{"source_account_id":null,"starts_on":null,"through_date":null,"state":null,"search":"","limit":25,"cursor":null,"snapshot":null}`
	page := request("synthetic-owner", selector, 200)
	if readProofText(t, page["matching_count"]) != "1" {
		t.Fatal("wrong whole matching count")
	}
	var observations []map[string]json.RawMessage
	if json.Unmarshal(page["items"], &observations) != nil || len(observations) != 1 || readProofText(t, observations[0]["amount"]) != "9007199254740993.01" {
		t.Fatal("exact original money lost")
	}
	var unit map[string]json.RawMessage
	if json.Unmarshal(observations[0]["currency"], &unit) != nil || readProofText(t, unit["code"]) != "USD" || readProofText(t, unit["numeric_code"]) != "840" || readProofText(t, unit["valid_from"]) != "2000-01-01" || string(unit["valid_to"]) != "null" || string(unit["minor_unit"]) != "2" {
		t.Fatal("dated original currency metadata lost")
	}
	request("synthetic-other", selector, 404)
	request("synthetic-owner", strings.Replace(selector, `"search":""`, `"search":"","search":"foreign"`, 1), 400)
	request("synthetic-owner", strings.Replace(selector, `"search":""`, `"search":"","actor":"`+readOther+`"`, 1), 400)
	publish("http-publish-2", "1", "modified", "15.00", "cursor-2")
	pinned := strings.Replace(selector, `"snapshot":null`, `"snapshot":`+string(page["snapshot"]), 1)
	request("synthetic-owner", pinned, 409)
	fresh := request("synthetic-owner", selector, 200)
	if json.Unmarshal(fresh["items"], &observations) != nil || readProofText(t, observations[0]["amount"]) != "15" || readProofText(t, observations[0]["observation_revision"]) != "2" {
		t.Fatal("source correction not independently visible")
	}

	// Missing optional numeric identity remains null; dated alphabetic identity still values the fact.
	readProofCall(t, runtime, false, "SELECT banking.publish_collection_batch($1,$2,$3,'http-null-numeric',$4)", readActor, workspace, source, `{"expected_item_generation":"1","expected_source_revision":"2","starting_cursor":"cursor-2","next_cursor":"cursor-3","readiness":"INITIAL_UPDATE_COMPLETE","changes":[{"operation":"modified","transaction_id":"SYNTHETIC-HTTP-TX","booked_date":"2026-01-02","description":"Fictional source","merchant_name":null,"source_state":"booked","currency_code":"EUR","amount":"15.00","direction":"outflow"}]}`)
	nullableUnit := request("synthetic-owner", selector, 200)
	if json.Unmarshal(nullableUnit["items"], &observations) != nil || json.Unmarshal(observations[0]["currency"], &unit) != nil || string(unit["numeric_code"]) != "null" || readProofText(t, unit["code"]) != "EUR" || readProofText(t, observations[0]["amount"]) != "15" {
		t.Fatal("nullable numeric currency code fabricated or denied")
	}
	readProofCall(t, operator, true, "SELECT banking.provision_collection_owner($1,'http-revoke',$2)", readActor, `{"expected_revision":"1","capabilities":[],"evidence_ref":"00000000-0000-4000-8000-000000000109"}`)
	request("synthetic-owner", selector, 404)
	var journals, books int
	if err := owner.QueryRow(context.Background(), "SELECT (SELECT count(*) FROM ledger.journal_entry),(SELECT count(*) FROM ledger.book)").Scan(&journals, &books); err != nil || journals != 0 || books != 0 {
		t.Fatal("source-only fixture created accounting state", err)
	}
}
