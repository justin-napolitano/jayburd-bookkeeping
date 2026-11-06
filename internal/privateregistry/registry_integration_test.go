//go:build integration

package privateregistry

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("private-registry-proof", false, "Use only disposable internal private registry database")

const actor = "00000000-0000-4000-8000-000000000001"
const other = "00000000-0000-4000-8000-000000000002"
const currency = "00000000-0000-4000-8000-000000000003"
const evidence = "00000000-0000-4000-8000-000000000004"
const credential = "00000000-0000-4000-8000-000000000005"
const entityID = "00000000-0000-4000-8000-000000000010"
const bookID = "00000000-0000-4000-8000-000000000011"
const accountID = "00000000-0000-4000-8000-000000000012"
const sourceID = "00000000-0000-4000-8000-000000000013"
const namespaceID = "00000000-0000-4000-8000-000000000014"
const policyID = "00000000-0000-4000-8000-000000000015"
const equityID = "00000000-0000-4000-8000-000000000016"

// connect accepts only the runner's private database/user names, never inherited credentials.
func connect(t testing.TB, db, user string) *pgx.Conn {
	t.Helper()
	if !*proof {
		t.Skip("use scripts/test-private-registry.sh")
	}
	c, err := pgx.Connect(context.Background(), fmt.Sprintf("host=database port=5432 user=%s dbname=%s sslmode=disable connect_timeout=5", user, db))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// exec is explicit owner-only fictional setup or aggregate inspection.
func exec(t testing.TB, c *pgx.Conn, q string, args ...any) {
	t.Helper()
	if _, err := c.Exec(context.Background(), q, args...); err != nil {
		t.Fatal(err)
	}
}

// value reads a fixed aggregate proof result rather than private source content.
func value(t testing.TB, c *pgx.Conn, q string, args ...any) string {
	t.Helper()
	var v string
	if err := c.QueryRow(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// code compares stable SQLSTATE without relying on input-bearing database diagnostics.
func code(t testing.TB, err error, want string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(err, &p) || p.Code != want {
		t.Fatalf("wanted SQLSTATE%s, got%v", want, err)
	}
}

// raw preserves independently authored synthetic decimal strings.
func raw(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// call runs one real restricted login transaction with an explicit isolation level.
func call(t testing.TB, db, user, iso, q string, args ...any) (map[string]any, error) {
	t.Helper()
	c := connect(t, db, user)
	defer c.Close(context.Background())
	tx, err := c.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.TxIsoLevel(iso)})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if user == "synthetic_operator" {
		if _, err = tx.Exec(context.Background(), "SET LOCAL ROLE bookkeeping_owner"); err != nil {
			return nil, err
		}
	}
	var b []byte
	err = tx.QueryRow(context.Background(), q, args...).Scan(&b)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	var result map[string]any
	err = json.Unmarshal(b, &result)
	return result, err
}

// fixture installs only two fictional enrolled principals and independently dated currency references.
func fixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	hash := sha256.Sum256([]byte(t.Name()))
	db := fmt.Sprintf("private_%x_%d", hash[:4], time.Now().UnixNano())
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE private_template")
	owner := connect(t, db, "postgres")
	t.Cleanup(func() { owner.Close(context.Background()) })
	exec(t, owner, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic','owner-a'),($2,'synthetic','owner-b')`, actor, other)
	exec(t, owner, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000006','JPY','392',0,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000007','KWD','414',3,'2000-01-01',NULL,'synthetic')`, currency)
	return owner, db
}

// provision models separate explicit operator attestation, not book-member self-enrollment.
func provision(t testing.TB, db, subject string) string {
	t.Helper()
	r, err := call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,$2,$3)", subject, "grant", raw(map[string]any{"expected_revision": nil, "capabilities": []string{"read", "acquire", "classify"}, "evidence_ref": evidence}))
	if err != nil {
		t.Fatal(err)
	}
	return r["workspace_id"].(string)
}

// register stores one synthetic verified manifest through the real API login without a provider call.
func register(t testing.TB, db, w, who, item string, count int) (string, []string) {
	t.Helper()
	accounts := []map[string]any{}
	for i := 0; i < count; i++ {
		accounts = append(accounts, map[string]any{"account_id": fmt.Sprintf("account-%d", i), "display_name": fmt.Sprintf("Fictional account%d", i)})
	}
	r, err := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.register_collection_item($1,$2,$3,$4)", who, w, "register-"+item, raw(map[string]any{"provider": "plaid", "environment": "sandbox", "item_id": item, "credential_ref": credential, "credential_version": "1", "consent_evidence_ref": evidence, "consent_expires_at": "2099-01-01T00:00:00Z", "accounts": accounts}))
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, s := range r["sources"].([]any) {
		ids = append(ids, s.(map[string]any)["source_account_id"].(string))
	}
	return r["collection_item_id"].(string), ids
}

// change is an independently fixed normalized source fact; amounts never use float.
func change(id, amount, currencyCode string) map[string]any {
	return map[string]any{"operation": "added", "transaction_id": id, "booked_date": "2026-10-01", "description": "Fictional cafe purchase", "merchant_name": "Fictional Cafe", "source_state": "booked", "currency_code": currencyCode, "amount": amount, "direction": "outflow"}
}

// publish invokes exactly one bounded complete normalized batch from a trusted synthetic backend.
func publish(t testing.TB, db, w, s, key, revision string, start, next any, changes []map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.publish_collection_batch($1,$2,$3,$4,$5)", actor, w, s, key, raw(map[string]any{"expected_source_revision": revision, "expected_item_generation": "1", "starting_cursor": start, "next_cursor": next, "readiness": "HISTORICAL_UPDATE_COMPLETE", "changes": changes}))
}

// selector is the frozen safe query envelope with no implicit current-page aggregation.
func selector() map[string]any {
	return map[string]any{"source_account_id": nil, "starts_on": nil, "through_date": nil, "state": nil, "search": "", "limit": 100, "cursor": nil, "snapshot": nil}
}

// query uses a fresh current restricted snapshot for each page.
func query(t testing.TB, db, w, who string, v map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.query_collection_observations($1,$2,$3)", who, w, raw(v))
}

// TestUnassignedScope proves pre-book storage never fabricates a book/account/journal.
func TestUnassignedScope(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "item-unassigned", 1)
	if _, err := publish(t, db, w, sources[0], "batch", "0", nil, "c1", []map[string]any{change("tx1", "10.01", "USD")}); err != nil {
		t.Fatal(err)
	}
	r, err := query(t, db, w, actor, selector())
	if err != nil {
		t.Fatal(err)
	}
	if r["matching_count"] != "1" {
		t.Fatal("incorrect count")
	}
	row := r["items"].([]any)[0].(map[string]any)
	if row["amount"] != "10.01" || row["observation_revision"] != "1" {
		t.Fatal("exact original amount/revision lost")
	}
	for _, key := range []string{"entity_id", "book_id", "journal_id", "credential_ref", "external_entry_id"} {
		if _, ok := row[key]; ok {
			t.Fatal("unsafe projection", key)
		}
	}
	if value(t, owner, "SELECT((SELECT count(*)FROM ledger.entity)+(SELECT count(*)FROM ledger.book)+(SELECT count(*)FROM ledger.journal_entry)+(SELECT count(*)FROM integration.outbox_message))::text") != "0" {
		t.Fatal("invented accounting state")
	}
}

// TestCrossPrincipalDeny verifies private authority before count, snapshot and guessed source diagnostics.
func TestCrossPrincipalDeny(t *testing.T) {
	_, db := fixture(t)
	w := provision(t, db, actor)
	wb := provision(t, db, other)
	_, sources := register(t, db, w, actor, "item-private", 1)
	_, err := query(t, db, w, other, selector())
	code(t, err, "P2801")
	q := selector()
	q["source_account_id"] = sources[0]
	_, err = query(t, db, wb, other, q)
	code(t, err, "P2801")
	_, err = call(t, db, "synthetic_mixed", "repeatable read", "SELECT banking.get_collection_workspace($1)", actor)
	code(t, err, "P2801")
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.provision_collection_owner($1,$2,$3)", actor, "self-grant", `{}`)
	code(t, err, "42501")
}

// TestRevisionReplay proves same-batch recovery, correction immutability and atomic malformed rollback.
func TestRevisionReplay(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "item-replay", 1)
	s := sources[0]
	r, err := publish(t, db, w, s, "first", "0", nil, "c1", []map[string]any{change("tx1", "100.00", "USD")})
	if err != nil {
		t.Fatal(err)
	}
	again, err := publish(t, db, w, s, "first", "0", nil, "c1", []map[string]any{change("tx1", "100.00", "USD")})
	if err != nil || raw(r) != raw(again) {
		t.Fatal("same-key recovery failed", err)
	}
	_, err = publish(t, db, w, s, "first", "0", nil, "c1", []map[string]any{change("tx1", "99.00", "USD")})
	code(t, err, "P2802")
	c := change("tx1", "15.00", "USD")
	c["operation"] = "modified"
	if _, err = publish(t, db, w, s, "second", "1", "c1", "c2", []map[string]any{c}); err != nil {
		t.Fatal(err)
	}
	if value(t, owner, "SELECT string_agg(amount::text,','ORDER BY revision)FROM banking.collection_revision") != "100.000000000,15.000000000" {
		t.Fatal("history rewritten")
	}
	bad := change("missing", "2.00", "USD")
	bad["operation"] = "modified"
	_, err = publish(t, db, w, s, "bad", "2", "c2", "c3", []map[string]any{change("would-rollback", "3.00", "USD"), bad})
	code(t, err, "P2804")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_observation") != "1" || value(t, owner, "SELECT cursor_value FROM banking.collection_source") != "c2" {
		t.Fatal("partial publication")
	}
	_, err = publish(t, db, w, s, "duplicate", "2", "c2", "c3", []map[string]any{change("duplicate", "1", "USD"), change("duplicate", "2", "USD")})
	code(t, err, "P2802")
}

// TestCurrencyHoldAtomicity preserves unavailable/invalid original lexemes without rounded interpreted money.
func TestCurrencyHoldAtomicity(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "item-money", 1)
	if _, err := publish(t, db, w, sources[0], "currency", "0", nil, "c1", []map[string]any{change("usd", "12.50", "USD"), change("jpy", "1000", "JPY"), change("kwd", "1.234", "KWD"), change("held1", "0.001", "USD"), change("held2", "42.00", "ZZZ")}); err != nil {
		t.Fatal(err)
	}
	r, err := query(t, db, w, actor, selector())
	if err != nil {
		t.Fatal(err)
	}
	held := 0
	for _, item := range r["items"].([]any) {
		row := item.(map[string]any)
		if row["source_state"] == "held" {
			held++
			if row["amount"] != nil || row["currency"] != nil || row["direction"] != nil {
				t.Fatal("fabricated held money")
			}
		} else {
			currency := row["currency"].(map[string]any)
			if currency["valid_from"] != "2000-01-01" || currency["valid_to"] != nil {
				t.Fatal("dated identity missing")
			}
		}
	}
	if held != 2 || value(t, owner, "SELECT amount_lexeme FROM banking.collection_revision WHERE hold_reason='invalid_precision'") != "0.001" {
		t.Fatal("original lexeme/hold lost")
	}
}

// TestRevokeAndReconnect fences siblings, explicit source revoke and stale replay without deleting facts.
func TestRevokeAndReconnect(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	item, sources := register(t, db, w, actor, "item-revoke", 2)
	if _, err := publish(t, db, w, sources[0], "first", "0", nil, "c1", []map[string]any{change("tx", "2", "USD")}); err != nil {
		t.Fatal(err)
	}
	state := map[string]any{"kind": "source", "target_id": sources[1], "expected_revision": "1", "state": "revoked"}
	if _, err := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "source-revoke", raw(state)); err != nil {
		t.Fatal(err)
	}
	r, err := query(t, db, w, actor, selector())
	if err != nil || r["matching_count"] != "1" {
		t.Fatal("sibling incorrectly revoked", err)
	}
	state = map[string]any{"kind": "item", "target_id": item, "expected_revision": "1", "state": "revoked"}
	if _, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "item-revoke", raw(state)); err != nil {
		t.Fatal(err)
	}
	_, err = publish(t, db, w, sources[0], "first", "0", nil, "c1", []map[string]any{change("tx", "2", "USD")})
	code(t, err, "P2801")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_revision") != "1" {
		t.Fatal("revocation erased evidence")
	}
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "reactivate", raw(map[string]any{"kind": "item", "target_id": item, "expected_revision": "2", "state": "active"}))
	code(t, err, "P2804")
}

// TestCollectionPagination3000 proves complete-result counts, fixed page order and original-unit rows.
func TestCollectionPagination3000(t *testing.T) {
	_, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "item-volume", 1)
	for batch := 0; batch < 3; batch++ {
		changes := make([]map[string]any, 1000)
		for j := range changes {
			changes[j] = change(fmt.Sprintf("tx-%04d", batch*1000+j), "0.10", "USD")
		}
		var start any
		if batch > 0 {
			start = fmt.Sprintf("c%d", batch)
		}
		if _, err := publish(t, db, w, sources[0], fmt.Sprintf("batch%d", batch), fmt.Sprint(batch), start, fmt.Sprintf("c%d", batch+1), changes); err != nil {
			t.Fatal(err)
		}
	}
	q := selector()
	seen := map[string]bool{}
	pages := 0
	for {
		r, err := query(t, db, w, actor, q)
		if err != nil {
			t.Fatal(err)
		}
		if r["matching_count"] != "3000" {
			t.Fatal("page-only count")
		}
		for _, z := range r["items"].([]any) {
			row := z.(map[string]any)
			id := row["observation_id"].(string)
			if seen[id] || row["amount"] != "0.1" {
				t.Fatal("duplicate or inexact row")
			}
			seen[id] = true
		}
		pages++
		if r["next_cursor"] == nil {
			break
		}
		q["cursor"] = r["next_cursor"]
		q["snapshot"] = r["snapshot"]
	}
	if len(seen) != 3000 || pages != 30 {
		t.Fatal("incomplete pagination")
	}
	q = selector()
	q["limit"] = 1
	r, err := query(t, db, w, actor, q)
	if err != nil {
		t.Fatal(err)
	}
	q["cursor"] = r["next_cursor"]
	q["snapshot"] = r["snapshot"]
	if _, err = publish(t, db, w, sources[0], "next", "3", "c3", "c4", []map[string]any{change("new", "0.20", "USD")}); err != nil {
		t.Fatal(err)
	}
	_, err = query(t, db, w, actor, q)
	code(t, err, "P2803")
}

// TestCollectionPrivileges verifies closed SQL entrypoints and no new raw-DML/helper authority.
func TestCollectionPrivileges(t *testing.T) {
	owner, _ := fixture(t)
	if value(t, owner, `SELECT count(*)::text FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relname LIKE 'collection_%'AND n.nspname IN('banking','integration','audit')AND has_table_privilege('synthetic_runtime',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')`) != "0" {
		t.Fatal("runtime table access")
	}
	if value(t, owner, `SELECT has_function_privilege('synthetic_runtime','banking.collection_facts(uuid)','EXECUTE')::text`) != "false" {
		t.Fatal("private helper exposed")
	}
	if value(t, owner, `SELECT has_function_privilege('synthetic_runtime','banking.provision_collection_owner(uuid,text,text)','EXECUTE')::text`) != "false" {
		t.Fatal("self grant")
	}
	if value(t, owner, `SELECT has_function_privilege('synthetic_sync','banking.get_collection_workspace(uuid)','EXECUTE')::text`) != "false" {
		t.Fatal("worker read access")
	}
}

// TestCollectionRevocationRace uses actual row locks and a fresh retry after serialization failure.
func TestCollectionRevocationRace(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	tx, err := owner.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(context.Background(), "UPDATE banking.collection_workspace SET revision=revision+1,capabilities='{}'WHERE workspace_id=$1", w); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		rt, er := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
		if er != nil {
			done <- er
			return
		}
		defer rt.Rollback(context.Background())
		var row []byte
		er = rt.QueryRow(context.Background(), "SELECT banking.get_collection_workspace($1)", actor).Scan(&row)
		if er == nil {
			er = rt.Commit(context.Background())
		}
		_ = rt.Rollback(context.Background())
		done <- er
	}()
	waitBlocked(t, owner, runtime.PgConn().PID())
	if err = tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	code(t, <-done, "40001")
	_, err = call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_workspace($1)", actor)
	code(t, err, "P2801")
}

// legacyFixture uses existing source/enrollment/posting SQL and independently fixed old tuples.
func legacyFixture(t *testing.T) (*pgx.Conn, string, map[string]any) {
	t.Helper()
	owner, db := fixture(t)
	exec(t, owner, `INSERT INTO ledger.entity(entity_id,name,kind)VALUES($1,'Fictional legacy LLC','legal')`, entityID)
	exec(t, owner, `INSERT INTO ledger.book(entity_id,book_id,name)VALUES($1,$2,'Fictional legacy book')`, entityID, bookID)
	exec(t, owner, `INSERT INTO ledger.membership SELECT $1,p,permission,NULL FROM unnest(ARRAY[$2::uuid,$3::uuid])p CROSS JOIN unnest(ARRAY['read','import','connect','post'])permission`, entityID, actor, other)
	exec(t, owner, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())`, entityID, bookID, policyID, currency, actor)
	exec(t, owner, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before)VALUES($1,$2,$3,'2026-01-01','2027-01-01')`, entityID, bookID, evidence)
	exec(t, owner, `INSERT INTO ledger.account VALUES($1,$2,$3,'1000','Fictional checking','asset','debit',true,'2026-01-01',NULL),($1,$2,$4,'3000','Fictional equity','equity','credit',false,'2026-01-01',NULL)`, entityID, bookID, accountID, equityID)
	exec(t, owner, `INSERT INTO banking.source_account(entity_id,book_id,source_account_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by)VALUES($1,$2,$3,$4,$5,'2026-01-05',$6,$7)`, entityID, bookID, sourceID, accountID, currency, evidence, actor)
	exec(t, owner, `INSERT INTO banking.source_namespace(entity_id,book_id,namespace_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by)VALUES($1,$2,$3,$4,'jayburd-synthetic-statement-csv/1',$3,'FICTIONAL-01',false,$5)`, entityID, bookID, namespaceID, sourceID, actor)
	exec(t, owner, `INSERT INTO banking.source_permission SELECT $1,$2,$3,p,cap,NULL FROM unnest(ARRAY[$4::uuid,$5::uuid])p CROSS JOIN unnest(ARRAY['read','import'])cap`, entityID, bookID, sourceID, actor, other)
	enrollment := map[string]any{"source_account_id": sourceID, "expected_binding_revision": "1", "provider": "plaid", "environment": "sandbox", "synthetic": true, "item_id": "legacy-item", "account_id": "legacy-account", "credential_ref": credential, "credential_version": "1", "consent_principal_id": actor, "consent_evidence_ref": evidence, "consent_expires_at": "2099-01-01T00:00:00Z", "service_principal_id": other, "worker_login": "synthetic_sync"}
	r, err := call(t, db, "synthetic_operator", "serializable", "SELECT integration.enroll_sync_stream($1,$2,$3,$4,$5)", actor, entityID, bookID, "enroll", raw(enrollment))
	if err != nil {
		t.Fatal(err)
	}
	exec(t, owner, `UPDATE integration.sync_stream SET cursor_value='original-cursor',completed_generation=1 WHERE entity_id=$1 AND book_id=$2`, entityID, bookID)
	for n := 0; n < 2; n++ {
		oid := fmt.Sprintf("00000000-0000-4000-8000-%012d", 100+n)
		tx, err := owner.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(context.Background(), `INSERT INTO banking.source_observation(entity_id,book_id,observation_id,source_account_id,namespace_id,external_entry_id)VALUES($1,$2,$3,$4,$5,$6)`, entityID, bookID, oid, sourceID, namespaceID, fmt.Sprintf("old-%d", n)); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(context.Background(), `INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields)VALUES($1,$2,$3,1,'booked','2026-10-01',$4,'outflow','100.00',100.00,'Fictional old source fact','{}')`, entityID, bookID, oid, currency); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	posting := map[string]any{"expected_revision": value(t, owner, `SELECT revision::text FROM ledger.book WHERE entity_id=$1 AND book_id=$2`, entityID, bookID), "policy_id": policyID, "accounting_date": "2026-10-01", "description": "Fictional original journal", "lines": []map[string]any{{"account_id": accountID, "side": "debit", "transaction_currency": "USD", "transaction_amount": "100.00", "functional_amount": "100.00", "valuation_basis": "identity"}, {"account_id": equityID, "side": "credit", "transaction_currency": "USD", "transaction_amount": "100.00", "functional_amount": "100.00", "valuation_basis": "identity"}}}
	if _, err = call(t, db, "synthetic_runtime", "serializable", "SELECT body FROM ledger.post_journal($1,$2,$3,$4,$5,$6)", actor, entityID, bookID, "original-post", evidence, raw(posting)); err != nil {
		t.Fatal(err)
	}
	return owner, db, r
}

// legacyHash excludes the intentional Item fence but fixes all original source/cursor/journal anchors.
func legacyHash(t testing.TB, owner *pgx.Conn) string {
	t.Helper()
	return value(t, owner, `SELECT encode(sha256(convert_to(jsonb_build_array((SELECT jsonb_agg(to_jsonb(x)ORDER BY source_account_id)FROM banking.source_account x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY observation_id)FROM banking.source_observation x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY observation_id,revision)FROM banking.observation_revision x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY stream_id)FROM integration.sync_stream x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY journal_id)FROM ledger.journal_entry x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY journal_id,line_number)FROM ledger.journal_line x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY idempotency_key)FROM integration.operation_receipt x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY message_id)FROM integration.outbox_message x))::text,'UTF8')),'hex')`)
}

// adopt exercises the real explicit reviewed old-Item transition, never a blind migration grant.
func adopt(t testing.TB, db, w string, item map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.adopt_collection_item($1,$2,$3,$4,$5,$6,$7)", actor, w, entityID, bookID, item["sync_item_id"], "adopt", raw(map[string]any{"expected_generation": "1", "sources": []map[string]any{{"source_account_id": sourceID, "display_name": "Reviewed fictional checking"}}, "evidence_ref": evidence}))
}

// TestExistingBookMigration proves explicit adoption fences acquisition while original identities/history remain exact.
func TestExistingBookMigration(t *testing.T) {
	owner, db, item := legacyFixture(t)
	w := provision(t, db, actor)
	before := legacyHash(t, owner)
	r, err := adopt(t, db, w, item)
	if err != nil {
		t.Fatal(err)
	}
	if before != legacyHash(t, owner) {
		t.Fatal("legacy anchor/history/cursor changed")
	}
	if value(t, owner, `SELECT state||':'||generation::text FROM integration.sync_item WHERE entity_id=$1 AND book_id=$2`, entityID, bookID) != "paused:2" {
		t.Fatal("old acquisition not fenced")
	}
	r2, err := adopt(t, db, w, item)
	if err != nil || raw(r) != raw(r2) {
		t.Fatal("adoption recovery failed", err)
	}
	q := selector()
	q["limit"] = 1
	page, err := query(t, db, w, actor, q)
	if err != nil || page["matching_count"] != "2" {
		t.Fatal("source facts lost", err)
	}
	row := page["items"].([]any)[0].(map[string]any)
	if row["amount"] != "100" || row["merchant_name"] != nil {
		t.Fatal("legacy fact changed")
	}
	for _, field := range []string{"entity_id", "book_id", "journal_id", "ledger_account_id"} {
		if _, ok := row[field]; ok {
			t.Fatal("book metadata leaked")
		}
	}
	_, err = owner.Exec(context.Background(), `UPDATE integration.sync_item SET state='active',generation=generation+1 WHERE entity_id=$1 AND book_id=$2`, entityID, bookID)
	code(t, err, "P2804")
	exec(t, owner, `UPDATE banking.source_account SET source_revision=source_revision+1 WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3`, entityID, bookID, sourceID)
	q["cursor"] = page["next_cursor"]
	q["snapshot"] = page["snapshot"]
	_, err = query(t, db, w, actor, q)
	code(t, err, "P2803")
	exec(t, owner, `UPDATE banking.source_permission SET revoked_at=clock_timestamp()WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3 AND principal_id=$4 AND capability='import'`, entityID, bookID, sourceID, actor)
	_, err = adopt(t, db, w, item)
	code(t, err, "P2801")
	exec(t, owner, `UPDATE ledger.membership SET revoked_at=clock_timestamp()WHERE entity_id=$1 AND principal_id=$2 AND permission='read'`, entityID, actor)
	_, err = adopt(t, db, w, item)
	code(t, err, "P2801")
	if _, err = query(t, db, w, actor, selector()); err != nil {
		t.Fatal("private source read improperly derives book permission", err)
	}
}

// TestDuplicateItemAndAccount proves both native and legacy admission share the canonical identity reservation.
func TestDuplicateItemAndAccount(t *testing.T) {
	owner, db, _ := legacyFixture(t)
	w := provision(t, db, actor)
	manifest := map[string]any{"provider": "plaid", "environment": "sandbox", "item_id": "legacy-item", "credential_ref": credential, "credential_version": "1", "consent_evidence_ref": evidence, "consent_expires_at": "2099-01-01T00:00:00Z", "accounts": []map[string]any{{"account_id": "native-account", "display_name": "Fictional"}}}
	_, err := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "duplicate-legacy", raw(manifest))
	code(t, err, "P2802")
	register(t, db, w, actor, "native-item", 1)
	_, err = owner.Exec(context.Background(), `INSERT INTO integration.sync_item(entity_id,book_id,provider,environment,item_id,credential_ref,credential_version,consent_principal,consent_evidence_ref,consent_expires_at,synthetic)VALUES($1,$2,'plaid','sandbox','native-item',$3,1,$4,$5,'2099-01-01',true)`, entityID, bookID, credential, actor, evidence)
	code(t, err, "P2802")
	if value(t, owner, "SELECT count(*)::text FROM integration.sync_item") != "1" || value(t, owner, "SELECT count(*)::text FROM integration.collection_item") != "2" {
		t.Fatal("duplicate provider authority")
	}
	manifest["item_id"] = "duplicate-account"
	manifest["accounts"] = []map[string]any{{"account_id": "same", "display_name": "One"}, {"account_id": "same", "display_name": "Two"}}
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "duplicate-account", raw(manifest))
	code(t, err, "P2802")
}

// TestPublicationGenerationFence rejects an old prepared batch after pause/resume even with unchanged cursor/source revision.
func TestPublicationGenerationFence(t *testing.T) {
	_, db := fixture(t)
	w := provision(t, db, actor)
	item, sources := register(t, db, w, actor, "generation-item", 1)
	for n, state := range []string{"paused", "active"} {
		_, err := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, fmt.Sprintf("state%d", n), raw(map[string]any{"kind": "item", "target_id": item, "expected_revision": fmt.Sprint(n + 1), "state": state}))
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := publish(t, db, w, sources[0], "stale", "0", nil, "c1", []map[string]any{change("stale", "1", "USD")})
	code(t, err, "P2803")
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "state0", raw(map[string]any{"kind": "item", "target_id": item, "expected_revision": "1", "state": "paused"}))
	code(t, err, "P2803")
}

// waitBlocked observes the actual database lock instead of assuming a sleep produced the intended schedule.
func waitBlocked(t testing.TB, observer *pgx.Conn, pid uint32) {
	t.Helper()
	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := observer.QueryRow(context.Background(), "SELECT EXISTS(SELECT FROM pg_locks WHERE pid=$1 AND NOT granted)", pid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("synthetic contender never reached the owned lock")
}

// itemManifest is a fixed fictional provider identity; no token or provider request is involved.
func itemManifest(item string) string {
	return raw(map[string]any{"provider": "plaid", "environment": "sandbox", "item_id": item, "credential_ref": credential, "credential_version": "1", "consent_evidence_ref": evidence, "consent_expires_at": "2099-01-01T00:00:00Z", "accounts": []map[string]any{{"account_id": "race-account", "display_name": "Fictional race account"}}})
}

// TestLegacyNativeIdentityRace proves both possible reservation winners, with real blocked transactions.
func TestLegacyNativeIdentityRace(t *testing.T) {
	owner, db, _ := legacyFixture(t)
	w := provision(t, db, actor)
	legacySQL := `INSERT INTO integration.sync_item(entity_id,book_id,provider,environment,item_id,credential_ref,credential_version,consent_principal,consent_evidence_ref,consent_expires_at,synthetic)VALUES($1,$2,'plaid','sandbox',$3,$4,1,$5,$6,'2099-01-01',true)`
	legacyTx, err := owner.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer legacyTx.Rollback(context.Background())
	if _, err = legacyTx.Exec(context.Background(), legacySQL, entityID, bookID, "race-legacy", credential, actor, evidence); err != nil {
		t.Fatal(err)
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	observer := connect(t, db, "postgres")
	defer observer.Close(context.Background())
	done := make(chan error, 1)
	go func() {
		tx, er := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
		if er != nil {
			done <- er
			return
		}
		defer tx.Rollback(context.Background())
		var b []byte
		er = tx.QueryRow(context.Background(), "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "race-legacy", itemManifest("race-legacy")).Scan(&b)
		if er == nil {
			er = tx.Commit(context.Background())
		}
		_ = tx.Rollback(context.Background())
		done <- er
	}()
	waitBlocked(t, observer, runtime.PgConn().PID())
	if err = legacyTx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	err = <-done
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "40001" {
		_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "race-legacy", itemManifest("race-legacy"))
	}
	code(t, err, "P2802")
	// Reverse schedule: native authority commits while old enrollment waits on its same reservation.
	nativeTx, err := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer nativeTx.Rollback(context.Background())
	var result []byte
	if err = nativeTx.QueryRow(context.Background(), "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "race-native", itemManifest("race-native")).Scan(&result); err != nil {
		t.Fatal(err)
	}
	legacyConn := connect(t, db, "postgres")
	defer legacyConn.Close(context.Background())
	go func() {
		_, er := legacyConn.Exec(context.Background(), legacySQL, entityID, bookID, "race-native", credential, actor, evidence)
		done <- er
	}()
	waitBlocked(t, observer, legacyConn.PgConn().PID())
	if err = nativeTx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	code(t, <-done, "P2802")
	if value(t, owner, "SELECT count(*)::text FROM integration.collection_item WHERE item_id IN('race-native','race-legacy')") != "2" || value(t, owner, "SELECT count(*)::text FROM integration.sync_item WHERE item_id IN('race-native','race-legacy')") != "1" {
		t.Fatal("two provider authorities survived race")
	}
}

// TestCurrentReplayAuthority verifies old successful receipts cannot bypass later source or Item revocation.
func TestCurrentReplayAuthority(t *testing.T) {
	_, db := fixture(t)
	w := provision(t, db, actor)
	item, sources := register(t, db, w, actor, "replay-authority", 1)
	_, err := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "pause", raw(map[string]any{"kind": "item", "target_id": item, "expected_revision": "1", "state": "paused"}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "revoke-item", raw(map[string]any{"kind": "item", "target_id": item, "expected_revision": "2", "state": "revoked"}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "pause", raw(map[string]any{"kind": "item", "target_id": item, "expected_revision": "1", "state": "paused"}))
	code(t, err, "P2801")
	// Distinct Item remains readable, but its registration receipt may not return a revoked alias.
	_, sources = register(t, db, w, actor, "source-replay", 1)
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "revoke-source", raw(map[string]any{"kind": "source", "target_id": sources[0], "expected_revision": "1", "state": "revoked"}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "register-source-replay", itemManifest("source-replay"))
	code(t, err, "P2802")
	// Replay exactly the originally reviewed payload to test authority rather than body mismatch.
	manifest := map[string]any{"provider": "plaid", "environment": "sandbox", "item_id": "source-replay", "credential_ref": credential, "credential_version": "1", "consent_evidence_ref": evidence, "consent_expires_at": "2099-01-01T00:00:00Z", "accounts": []map[string]any{{"account_id": "account-0", "display_name": "Fictional account0"}}}
	_, err = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.register_collection_item($1,$2,$3,$4)", actor, w, "register-source-replay", raw(manifest))
	code(t, err, "P2801")
}

// TestCurrentSourceReadLocks primes RR before each lifecycle commit and observes the actual blocked reader.
func TestCurrentSourceReadLocks(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		owner, db, item := legacyFixture(t)
		w := provision(t, db, actor)
		adopted, err := adopt(t, db, w, item)
		if err != nil {
			t.Fatal(err)
		}
		alias := adopted["sources"].([]any)[0].(map[string]any)["source_account_id"]
		reader := connect(t, db, "synthetic_runtime")
		defer reader.Close(context.Background())
		rt, err := reader.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if err != nil {
			t.Fatal(err)
		}
		defer rt.Rollback(context.Background())
		var count int
		if err = rt.QueryRow(context.Background(), "SELECT count(*)FROM pg_catalog.pg_class").Scan(&count); err != nil {
			t.Fatal(err)
		}
		writer, err := owner.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Rollback(context.Background())
		if _, err = writer.Exec(context.Background(), `UPDATE banking.source_account SET state='paused',binding_revision=binding_revision+1 WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3`, entityID, bookID, sourceID); err != nil {
			t.Fatal(err)
		}
		observer := connect(t, db, "postgres")
		defer observer.Close(context.Background())
		done := make(chan error, 1)
		q := selector()
		q["source_account_id"] = alias
		go func() {
			var body []byte
			er := rt.QueryRow(context.Background(), "SELECT banking.query_collection_observations($1,$2,$3)", actor, w, raw(q)).Scan(&body)
			_ = rt.Rollback(context.Background())
			done <- er
		}()
		waitBlocked(t, observer, reader.PgConn().PID())
		if err = writer.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
		code(t, <-done, "40001")
		_, err = query(t, db, w, actor, q)
		code(t, err, "P2801")
	})
	t.Run("native", func(t *testing.T) {
		owner, db := fixture(t)
		w := provision(t, db, actor)
		item, _ := register(t, db, w, actor, "primed-native", 1)
		reader := connect(t, db, "synthetic_runtime")
		defer reader.Close(context.Background())
		rt, err := reader.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
		if err != nil {
			t.Fatal(err)
		}
		defer rt.Rollback(context.Background())
		var count int
		if err = rt.QueryRow(context.Background(), "SELECT count(*)FROM pg_catalog.pg_class").Scan(&count); err != nil {
			t.Fatal(err)
		}
		writeConn := connect(t, db, "synthetic_runtime")
		defer writeConn.Close(context.Background())
		writer, err := writeConn.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Rollback(context.Background())
		var result []byte
		if err = writer.QueryRow(context.Background(), "SELECT banking.set_collection_state($1,$2,$3,$4)", actor, w, "revoke", raw(map[string]any{"kind": "item", "target_id": item, "expected_revision": "1", "state": "revoked"})).Scan(&result); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			var body []byte
			er := rt.QueryRow(context.Background(), "SELECT banking.list_collection_sources($1,$2,$3)", actor, w, `{"limit":100,"cursor":null,"snapshot":null}`).Scan(&body)
			_ = rt.Rollback(context.Background())
			done <- er
		}()
		waitBlocked(t, owner, reader.PgConn().PID())
		if err = writer.Commit(context.Background()); err != nil {
			t.Fatal(err)
		}
		code(t, <-done, "40001")
		r, err := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.list_collection_sources($1,$2,$3)", actor, w, `{"limit":100,"cursor":null,"snapshot":null}`)
		if err != nil || len(r["items"].([]any)) != 0 {
			t.Fatal("fresh read retained revoked aliases", err)
		}
	})
}
