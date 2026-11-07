//go:build integration

package collectionlabels

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var proof = flag.Bool("collection-label-proof", false, "Use only disposable internal private registry database")

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
		t.Skip("use scripts/test-collection-labels.sh")
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
	db := fmt.Sprintf("labels_%x_%d", hash[:4], time.Now().UnixNano())
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE labels_template")
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

const rootTopic = "00000000-0000-4000-8000-000000000101"
const childTopic = "00000000-0000-4000-8000-000000000102"

// selector is the exact closed review wire; all filtering and paging remain SQL-owned.
func selector() map[string]any {
	return map[string]any{"profile": "private-collection-review-v1", "source_account_id": nil, "assignment": "all", "starts_on": nil, "through_date": nil, "search": "", "topic": map[string]any{"kind": "all"}, "sort": "date_desc", "snapshot": nil, "cursor": nil, "limit": 100}
}
func query(t testing.TB, db, w, who string, v map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.query_collection_review($1,$2,$3)", who, w, raw(v))
}
func page(t testing.TB, db, w string) map[string]any {
	t.Helper()
	r, e := query(t, db, w, actor, selector())
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func catalog(t testing.TB, db, w string) map[string]any {
	t.Helper()
	r, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_taxonomy($1,$2)", actor, w)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func topic(id, name string, parent any, status string) map[string]any {
	return map[string]any{"topic_id": id, "name": name, "parent_id": parent, "status": status}
}
func topicsAction(topics []map[string]any) map[string]any {
	return map[string]any{"kind": "taxonomy", "topics": topics}
}
func selection(rows []any) []map[string]any {
	result := []map[string]any{}
	for _, x := range rows {
		r := x.(map[string]any)
		result = append(result, map[string]any{"workspace_id": r["workspace_id"], "source_id": r["source_id"], "observation_id": r["observation_id"], "observation_revision": r["observation_revision"]})
	}
	return result
}
func preview(t testing.TB, db, w string, snapshot any, selected []map[string]any, action any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.preview_collection_labels($1,$2,$3)", actor, w, raw(map[string]any{"snapshot": snapshot, "selection": selected, "action": action}))
}
func prepare(t testing.TB, db, w, key string, p map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.prepare_collection_label_command($1,$2,$3)", actor, w, raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": key}))
}
func transition(t testing.TB, db, w, id, op string) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.transition_collection_label_command($1,$2,$3,$4,'{}')", actor, w, id, op)
}
func command(t testing.TB, wrapper map[string]any) map[string]any {
	t.Helper()
	if wrapper["profile"] != "private-collection-command-v1" {
		t.Fatal("command profile")
	}
	return wrapper["command"].(map[string]any)
}
func apply(t testing.TB, db, w, key string, selected []map[string]any, action any) map[string]any {
	t.Helper()
	p, e := preview(t, db, w, page(t, db, w)["snapshot"], selected, action)
	if e != nil {
		t.Fatal(e)
	}
	r, e := prepare(t, db, w, key, p)
	if e != nil {
		t.Fatal(e)
	}
	id := command(t, r)["command_id"].(string)
	r, e = transition(t, db, w, id, "execute")
	if e != nil {
		t.Fatal(e)
	}
	if command(t, r)["state"] != "committed" {
		t.Fatal("not committed")
	}
	if _, e = transition(t, db, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	return r
}
func seedTopics(t testing.TB, db, w string) {
	t.Helper()
	apply(t, db, w, "taxonomy-initial", []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Everyday", nil, "active"), topic(childTopic, "Dining", rootTopic, "active")}))
}

// TestTaxonomyLifecycle proves persisted hierarchy, exact retry and explicit terminal acknowledgement.
func TestTaxonomyLifecycle(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	c := catalog(t, db, w)
	if c["taxonomy_revision"] != "1" || len(c["topics"].([]any)) != 0 {
		t.Fatal("empty initial edition")
	}
	if value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action") != "0" {
		t.Fatal("read wrote audit")
	}
	p, e := preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Everyday", nil, "active"), topic(childTopic, "Dining", rootTopic, "active")}))
	if e != nil {
		t.Fatal(e)
	}
	if p["selected_count"] != "0" || len(p["review_rows"].([]any)) != 0 {
		t.Fatal("taxonomy selection")
	}
	a, e := prepare(t, db, w, "stable", p)
	if e != nil {
		t.Fatal(e)
	}
	b, e := prepare(t, db, w, "stable", p)
	if e != nil || raw(a) != raw(b) {
		t.Fatal("lost prepare replay", e)
	}
	id := command(t, a)["command_id"].(string)
	_, e = transition(t, db, w, id, "acknowledge")
	code(t, e, "P2802")
	a, e = transition(t, db, w, id, "execute")
	if e != nil {
		t.Fatal(e)
	}
	b, e = transition(t, db, w, id, "cancel")
	if e != nil || raw(a) != raw(b) {
		t.Fatal("cancel cannot change committed", e)
	}
	b, e = transition(t, db, w, id, "execute")
	if e != nil || raw(a) != raw(b) {
		t.Fatal("terminal replay", e)
	}
	if catalog(t, db, w)["taxonomy_revision"] != "2" {
		t.Fatal("edition not advanced")
	}
	a, e = transition(t, db, w, id, "acknowledge")
	if e != nil {
		t.Fatal(e)
	}
	b, e = transition(t, db, w, id, "acknowledge")
	if e != nil || raw(a) != raw(b) {
		t.Fatal("ack replay", e)
	}
	apply(t, db, w, "retire", []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Everyday renamed", nil, "active"), topic(childTopic, "Dining", rootTopic, "retired")}))
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_taxonomy_edition") != "3" {
		t.Fatal("edition history")
	}
	_, e = preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Everyday", nil, "active"), topic(childTopic, "Dining", rootTopic, "active")}))
	code(t, e, "P2802")
	_, e = preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Everyday", nil, "active")}))
	code(t, e, "P2802")
}

// TestTaxonomyMalformed independently specifies foreign parents, cycles, sibling names, bounds and retired parents.
func TestTaxonomyMalformed(t *testing.T) {
	_, db := fixture(t)
	w := provision(t, db, actor)
	snap := page(t, db, w)["snapshot"]
	cases := [][]map[string]any{{topic(rootTopic, "Dining", childTopic, "active")}, {topic(rootTopic, "A", childTopic, "active"), topic(childTopic, "B", rootTopic, "active")}, {topic(rootTopic, "same", nil, "active"), topic(childTopic, "same", nil, "active")}, {topic(rootTopic, "retired", nil, "retired"), topic(childTopic, "active", rootTopic, "active")}, {topic(rootTopic, "", nil, "active")}, {topic(rootTopic, "A", nil, "active"), topic(rootTopic, "B", nil, "active")}}
	for i, v := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, e := preview(t, db, w, snap, []map[string]any{}, topicsAction(v))
			code(t, e, "P2800")
		})
	}
	deep := []map[string]any{}
	var parent any
	for i := 0; i < 9; i++ {
		id := fmt.Sprintf("10000000-0000-4000-8000-%012d", i)
		deep = append(deep, topic(id, fmt.Sprint(i), parent, "active"))
		parent = id
	}
	_, e := preview(t, db, w, snap, []map[string]any{}, topicsAction(deep))
	code(t, e, "P2800")
	many := []map[string]any{}
	for i := 0; i < 501; i++ {
		many = append(many, topic(fmt.Sprintf("10000000-0000-4000-8000-%012d", i), fmt.Sprint(i), nil, "active"))
	}
	_, e = preview(t, db, w, snap, []map[string]any{}, topicsAction(many))
	code(t, e, "P2805")
}

// TestBulkTopicAtomicity proves explicit100-row effects, exact money, independent topic states and stale corrections.
func TestBulkTopicAtomicity(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "bulk", 1)
	changes := []map[string]any{}
	for i := 0; i < 100; i++ {
		changes = append(changes, change(fmt.Sprintf("tx-%03d", i), "9007199254740993.01", "USD"))
	}
	if _, e := publish(t, db, w, sources[0], "batch", "0", nil, "c1", changes); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	before := page(t, db, w)
	selected := selection(before["items"].([]any))
	if before["items"].([]any)[0].(map[string]any)["money"].(map[string]any)["amount"] != "9007199254740993.01" {
		t.Fatal("decimal loss")
	}
	apply(t, db, w, "bulk", selected, map[string]any{"kind": "topic", "topic_id": childTopic})
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != "100" {
		t.Fatal("bulk count")
	}
	v := selector()
	v["snapshot"] = before["snapshot"]
	_, e := query(t, db, w, actor, v)
	code(t, e, "P2803")
	v = selector()
	v["topic"] = map[string]any{"kind": "topic", "topic_id": childTopic}
	r, e := query(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "100" {
		t.Fatal("topic filter", e)
	}
	apply(t, db, w, "uncategorized", selected[:50], map[string]any{"kind": "uncategorized"})
	v["topic"] = map[string]any{"kind": "uncategorized"}
	r, e = query(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "50" {
		t.Fatal("explicit uncategorized", e)
	}
	changed := change("tx-000", "15.00", "USD")
	changed["operation"] = "modified"
	if _, e = publish(t, db, w, sources[0], "correct", "1", "c1", "c2", []map[string]any{changed}); e != nil {
		t.Fatal(e)
	}
	v = selector()
	v["topic"] = map[string]any{"kind": "unclassified"}
	r, e = query(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "1" || r["items"].([]any)[0].(map[string]any)["owner_topic"].(map[string]any)["state"] != "stale" {
		t.Fatal("source-corrected override not stale", e)
	}
	invalid := append([]map[string]any{}, selected[:2]...)
	invalid[1] = map[string]any{"workspace_id": w, "source_id": sources[0], "observation_id": evidence, "observation_revision": "1"}
	count := value(t, owner, "SELECT count(*)::text FROM banking.collection_label_preview")
	_, e = preview(t, db, w, page(t, db, w)["snapshot"], invalid, map[string]any{"kind": "uncategorized"})
	code(t, e, "P2801")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_preview") != count {
		t.Fatal("partial preview")
	}
	if value(t, owner, "SELECT (SELECT count(*) FROM ledger.journal_entry)::text||':'||(SELECT count(*)FROM ledger.journal_line)::text||':'||(SELECT count(*)FROM integration.outbox_message)::text") != "0:0:0" {
		t.Fatal("analytic label posted")
	}
}

// TestAllTransactionsPrincipalDeny proves private capability cannot be inferred from guessed identities.
func TestAllTransactionsPrincipalDeny(t *testing.T) {
	_, db := fixture(t)
	w := provision(t, db, actor)
	provision(t, db, other)
	_, e := query(t, db, w, other, selector())
	code(t, e, "P2801")
	_, e = call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_taxonomy($1,$2)", other, w)
	code(t, e, "P2801")
	_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.preview_collection_labels($1,$2,'{}')", other, w)
	code(t, e, "P2801")
}

// TestLabelCommandStaleRecovery proves stale source pins block execution but leave explicit cancel/ack available.
func TestLabelCommandStaleRecovery(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "stale", 1)
	if _, e := publish(t, db, w, sources[0], "one", "0", nil, "c1", []map[string]any{change("tx", "10.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	r := page(t, db, w)
	p, e := preview(t, db, w, r["snapshot"], selection(r["items"].([]any)), map[string]any{"kind": "uncategorized"})
	if e != nil {
		t.Fatal(e)
	}
	a, e := prepare(t, db, w, "stable", p)
	if e != nil {
		t.Fatal(e)
	}
	id := command(t, a)["command_id"].(string)
	z := change("tx", "15.00", "USD")
	z["operation"] = "modified"
	if _, e = publish(t, db, w, sources[0], "two", "1", "c1", "c2", []map[string]any{z}); e != nil {
		t.Fatal(e)
	}
	_, e = transition(t, db, w, id, "execute")
	code(t, e, "P2803")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != "0" {
		t.Fatal("stale side effect")
	}
	pending, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.pending_collection_label_command($1,$2)", actor, w)
	if e != nil || command(t, pending)["command_id"] != id {
		t.Fatal("lost execute discovery", e)
	}
	a, e = transition(t, db, w, id, "cancel")
	if e != nil || command(t, a)["state"] != "cancelled" {
		t.Fatal("cancel stale", e)
	}
	if _, e = transition(t, db, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	if _, e = call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'revoke',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{"read"}, "evidence_ref": evidence})); e != nil {
		t.Fatal(e)
	}
	_, e = transition(t, db, w, id, "execute")
	code(t, e, "P2801")
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

// TestAllTransactionsPagination3000 proves complete counts, both stable date orders and exact source filters.
func TestAllTransactionsPagination3000(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "pages", 2)
	var start any
	for batch := 0; batch < 3; batch++ {
		changes := []map[string]any{}
		for i := 0; i < 1000; i++ {
			z := change(fmt.Sprintf("tx-%04d", batch*1000+i), "0.10", "USD")
			z["booked_date"] = fmt.Sprintf("2026-10-%02d", 1+i%28)
			changes = append(changes, z)
		}
		next := fmt.Sprintf("c%d", batch+1)
		if _, e := publish(t, db, w, sources[0], fmt.Sprint(batch), fmt.Sprint(batch), start, next, changes); e != nil {
			t.Fatal(e)
		}
		start = next
	}
	for _, order := range []string{"date_desc", "date_asc"} {
		t.Run(order, func(t *testing.T) {
			began := time.Now()
			v := selector()
			v["sort"] = order
			v["source_account_id"] = sources[0]
			seen := map[string]bool{}
			pages := 0
			lastDay := ""
			for {
				r, e := query(t, db, w, actor, v)
				if e != nil {
					t.Fatal(e)
				}
				if r["matching_count"] != "3000" || r["start_index"] != fmt.Sprint(pages*100+1) {
					t.Fatal("whole count/index")
				}
				for _, x := range r["items"].([]any) {
					row := x.(map[string]any)
					id := row["observation_id"].(string)
					if seen[id] || row["source_id"] != sources[0] || row["money"].(map[string]any)["amount"] != "0.1" {
						t.Fatal("duplicate/filter/decimal")
					}
					day := row["booked_date"].(string)
					if lastDay != "" && (order == "date_desc" && day > lastDay || order == "date_asc" && day < lastDay) {
						t.Fatal("global sort")
					}
					lastDay = day
					seen[id] = true
				}
				pages++
				if r["next_cursor"] == nil {
					break
				}
				v["cursor"] = r["next_cursor"]
				v["snapshot"] = r["snapshot"]
			}
			if len(seen) != 3000 || pages != 30 {
				t.Fatal("page coverage")
			}
			t.Logf("3000 exact rows /30 unique pages: %s; synthetic local PG17, no latency SLA", time.Since(began))
		})
	}
	v := selector()
	v["source_account_id"] = sources[1]
	r, e := query(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "0" || r["start_index"] != "0" {
		t.Fatal("empty account filter", e)
	}
	v = selector()
	v["assignment"] = "assigned"
	r, e = query(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "0" {
		t.Fatal("native assignment", e)
	}
	v = selector()
	v["search"] = "cafe"
	v["starts_on"] = "2026-10-01"
	v["through_date"] = "2026-10-01"
	r, e = query(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "108" {
		t.Fatal("full date/search filter", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("query posted")
	}
}

// TestPostedRetagNoJournalMutation links independently posted evidence and proves current book metadata gating.
func TestPostedRetagNoJournalMutation(t *testing.T) {
	owner, db, item := legacyFixture(t)
	w := provision(t, db, actor)
	if _, e := adopt(t, db, w, item); e != nil {
		t.Fatal(e)
	}
	// Attach an independently posted journal through the existing immutable review relations.
	exec(t, owner, `INSERT INTO banking.review_item(entity_id,book_id,review_id,source_account_id,review_revision,state)VALUES($1,$2,$3,$4,1,'posted')`, entityID, bookID, evidence, sourceID)
	exec(t, owner, `INSERT INTO banking.review_decision(entity_id,book_id,decision_id,review_id,expected_review_revision,kind,actor_id,reason,body)VALUES($1,$2,$3,$3,0,'initialize',$4,'synthetic_import','{}')`, entityID, bookID, evidence, actor)
	exec(t, owner, `INSERT INTO banking.review_evidence(entity_id,book_id,decision_id,observation_id,review_id,linked_revision)VALUES($1,$2,$3,'00000000-0000-4000-8000-000000000100',$3,1)`, entityID, bookID, evidence)
	exec(t, owner, `INSERT INTO banking.review_assignment VALUES($1,$2,'00000000-0000-4000-8000-000000000100',$3,$3)`, entityID, bookID, evidence)
	exec(t, owner, `INSERT INTO banking.review_draft(entity_id,book_id,draft_id,review_id,review_revision,source_revision,binding_revision,book_revision,posting_body,normalized_request,created_by,idempotency_key)VALUES($1,$2,$3,$3,1,0,1,1,'{}','{}',$4,'fixture-draft')`, entityID, bookID, evidence, actor)
	exec(t, owner, `INSERT INTO ledger.interactive_command(command_id,principal_id,registration_key,operation,entity_id,book_id,request_body,state,result,result_entity_id,result_book_id,terminal_at,acknowledged_at)VALUES($1,$2,'fixture-command','post',$3,$4,'{}','committed','{}',$3,$4,clock_timestamp(),clock_timestamp())`, evidence, actor, entityID, bookID)
	exec(t, owner, `INSERT INTO banking.review_posting(entity_id,book_id,review_id,draft_id,command_id,journal_id,result)SELECT $1,$2,$3,$3,$3,journal_id,'{}'FROM ledger.journal_entry WHERE entity_id=$1 AND book_id=$2`, entityID, bookID, evidence)
	before := legacyHash(t, owner)
	r := page(t, db, w)
	if r["matching_count"] != "2" {
		t.Fatal("legacy count")
	}
	found := false
	for _, x := range r["items"].([]any) {
		row := x.(map[string]any)
		if row["assignment"].(map[string]any)["state"] != "assigned" {
			t.Fatal("authorized assignment")
		}
		if row["observation_id"] == "00000000-0000-4000-8000-000000000100" && row["journal_state"] == "posted" {
			found = true
		}
	}
	if !found {
		t.Fatal("actual journal link unavailable")
	}
	apply(t, db, w, "retag-posted", selection(r["items"].([]any)), map[string]any{"kind": "uncategorized"})
	if legacyHash(t, owner) != before {
		t.Fatal("retag changed original source/journal/outbox")
	}
	r = page(t, db, w)
	p, e := preview(t, db, w, r["snapshot"], selection(r["items"].([]any)), map[string]any{"kind": "uncategorized"})
	if e != nil {
		t.Fatal(e)
	}
	cmd, e := prepare(t, db, w, "pending-legacy", p)
	if e != nil {
		t.Fatal(e)
	}
	exec(t, owner, `UPDATE ledger.membership SET revoked_at=clock_timestamp()WHERE entity_id=$1 AND principal_id=$2 AND permission='read'`, entityID, actor)
	fresh := page(t, db, w)
	for _, x := range fresh["items"].([]any) {
		row := x.(map[string]any)
		if row["assignment"].(map[string]any)["state"] != "unavailable" || row["journal_state"] != "unavailable" {
			t.Fatal("legacy metadata leaked")
		}
	}
	_, e = transition(t, db, w, command(t, cmd)["command_id"].(string), "cancel")
	code(t, e, "P2801")
	v := selector()
	v["snapshot"] = r["snapshot"]
	_, e = query(t, db, w, actor, v)
	code(t, e, "P2803")
	v = selector()
	v["assignment"] = "unassigned"
	q, e := query(t, db, w, actor, v)
	if e != nil || q["matching_count"] != "0" {
		t.Fatal("unavailable mistaken for unassigned", e)
	}
}

// TestLabelPrimedSnapshotFences observes source insertion and grant revocation with real blocked RR row locks.
func TestLabelPrimedSnapshotFences(t *testing.T) {
	for _, mode := range []string{"source_insert", "book_grant_revoke", "taxonomy"} {
		t.Run(mode, func(t *testing.T) {
			var owner *pgx.Conn
			var db, w string
			if mode == "book_grant_revoke" {
				var item map[string]any
				owner, db, item = legacyFixture(t)
				w = provision(t, db, actor)
				if _, e := adopt(t, db, w, item); e != nil {
					t.Fatal(e)
				}
			} else {
				owner, db = fixture(t)
				w = provision(t, db, actor)
			}
			runtime := connect(t, db, "synthetic_runtime")
			defer runtime.Close(context.Background())
			readTx, e := runtime.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
			if e != nil {
				t.Fatal(e)
			}
			defer readTx.Rollback(context.Background())
			var prime int
			if e = readTx.QueryRow(context.Background(), "SELECT count(*) FROM pg_catalog.pg_class").Scan(&prime); e != nil {
				t.Fatal(e)
			}
			writer := connect(t, db, "postgres")
			defer writer.Close(context.Background())
			tx, e := writer.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(context.Background())
			switch mode {
			case "source_insert":
				_, e = tx.Exec(context.Background(), `WITH i AS(INSERT INTO integration.collection_item(provider,environment,item_id,workspace_id,credential_ref,credential_version,consent_evidence_ref,consent_expires_at)VALUES('plaid','sandbox','insert-race',$1,$2,1,$3,'2099-01-01')RETURNING collection_item_id)INSERT INTO banking.collection_source(workspace_id,collection_item_id,account_id,display_name,origin)SELECT $1,collection_item_id,'race-account','Fictional inserted source','native'FROM i`, w, credential, evidence)
			case "book_grant_revoke":
				_, e = tx.Exec(context.Background(), `UPDATE ledger.membership SET revoked_at=clock_timestamp()WHERE entity_id=$1 AND principal_id=$2 AND permission='read'`, entityID, actor)
			case "taxonomy":
				_, e = tx.Exec(context.Background(), `INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics)VALUES($1,2,'[]')`, w)
				if e == nil {
					_, e = tx.Exec(context.Background(), `UPDATE banking.collection_label_state SET taxonomy_revision=2 WHERE workspace_id=$1`, w)
				}
			}
			if e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() {
				var b []byte
				err := readTx.QueryRow(context.Background(), "SELECT banking.query_collection_review($1,$2,$3)", actor, w, raw(selector())).Scan(&b)
				_ = readTx.Rollback(context.Background())
				done <- err
			}()
			waitBlocked(t, owner, runtime.PgConn().PID())
			if e = tx.Commit(context.Background()); e != nil {
				t.Fatal(e)
			}
			code(t, <-done, "40001")
			if _, e = query(t, db, w, actor, selector()); e != nil {
				t.Fatal("fresh read", e)
			}
		})
	}
}

// TestLabelPrivileges proves only the six reviewed exact APIs cross the private storage boundary.
func TestLabelPrivileges(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	for _, role := range []string{"bookkeeping_api", "bookkeeping_sync", "bookkeeping_webhook", "bookkeeping_observer", "bookkeeping_publisher", "bookkeeping_verifier", "bookkeeping_identity_operator"} {
		allow := []string{}
		if role == "bookkeeping_api" {
			allow = []string{"get_collection_taxonomy(uuid, uuid)", "query_collection_review(uuid, uuid, text)", "preview_collection_labels(uuid, uuid, text)", "prepare_collection_label_command(uuid, uuid, text)", "pending_collection_label_command(uuid, uuid)", "transition_collection_label_command(uuid, uuid, uuid, text, text)"}
		}
		if value(t, owner, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='banking' AND(p.proname LIKE 'collection_label_%'OR p.proname=ANY(ARRAY['get_collection_taxonomy','query_collection_review','preview_collection_labels','prepare_collection_label_command','pending_collection_label_command','transition_collection_label_command']))AND has_function_privilege($1,p.oid,'EXECUTE')<>((p.proname||'('||oidvectortypes(p.proargtypes)||')')=ANY($2::text[]))`, role, allow) != "0" {
			t.Fatal("function privileges", role)
		}
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	for _, q := range []string{"SELECT * FROM banking.collection_label_state", "SELECT * FROM banking.collection_label_preview", "SELECT * FROM banking.collection_taxonomy_edition", "SELECT * FROM banking.collection_label_command", "SELECT * FROM banking.collection_label_decision", "SELECT * FROM audit.collection_label_action", "SELECT banking.collection_label_snapshot(NULL)", "SET ROLE bookkeeping_owner"} {
		_, e := runtime.Exec(context.Background(), q)
		code(t, e, "42501")
	}
	_, e := call(t, db, "synthetic_mixed", "repeatable read", "SELECT banking.get_collection_taxonomy($1,$2)", actor, w)
	code(t, e, "P2801")
}

// TestReviewCoverageAndLegacyPrecision distinguishes history coverage from assignment and safe held projections.
func TestReviewCoverageAndLegacyPrecision(t *testing.T) {
	t.Run("native_empty_coverage", func(t *testing.T) {
		owner, db := fixture(t)
		w := provision(t, db, actor)
		_, sources := register(t, db, w, actor, "coverage", 1)
		q := page(t, db, w)
		if q["partial"] != true || q["matching_count"] != "0" {
			t.Fatal("empty NOT_READY incorrectly complete")
		}
		for i, state := range []string{"INITIAL_UPDATE_COMPLETE", "HISTORICAL_UPDATE_COMPLETE"} {
			_, e := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.publish_collection_batch($1,$2,$3,$4,$5)", actor, w, sources[0], fmt.Sprint(i), raw(map[string]any{"expected_source_revision": "0", "expected_item_generation": "1", "starting_cursor": func() any {
				if i == 0 {
					return nil
				}
				return "c0"
			}(), "next_cursor": fmt.Sprintf("c%d", i), "readiness": state, "changes": []map[string]any{}}))
			if e != nil {
				t.Fatal(e)
			}
			if page(t, db, w)["partial"] != (i == 0) {
				t.Fatal("history readiness projection", state)
			}
		}
		if value(t, owner, "SELECT count(*)::text FROM banking.collection_observation") != "0" {
			t.Fatal("empty coverage invented fact")
		}
	})
	t.Run("legacy_original_precision", func(t *testing.T) {
		owner, db, item := legacyFixture(t)
		w := provision(t, db, actor)
		if _, e := adopt(t, db, w, item); e != nil {
			t.Fatal(e)
		}
		tx, e := owner.Begin(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(context.Background())
		_, e = tx.Exec(context.Background(), `INSERT INTO banking.source_observation(entity_id,book_id,observation_id,source_account_id,namespace_id,external_entry_id)VALUES($1,$2,'00000000-0000-4000-8000-000000000777',$3,$4,'precision')`, entityID, bookID, sourceID, namespaceID)
		if e != nil {
			t.Fatal(e)
		}
		_, e = tx.Exec(context.Background(), `INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields)VALUES($1,$2,'00000000-0000-4000-8000-000000000777',1,'booked','2026-10-01',$3,'outflow','1.001',1.001,'Fictional precision hold','{}')`, entityID, bookID, currency)
		if e != nil {
			t.Fatal(e)
		}
		if e = tx.Commit(context.Background()); e != nil {
			t.Fatal(e)
		}
		exec(t, owner, `UPDATE banking.source_account SET source_revision=source_revision+1 WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3`, entityID, bookID, sourceID)
		before := legacyHash(t, owner)
		q := page(t, db, w)
		found := false
		for _, x := range q["items"].([]any) {
			r := x.(map[string]any)
			if r["observation_id"] != "00000000-0000-4000-8000-000000000777" {
				continue
			}
			found = true
			m := r["money"].(map[string]any)
			if r["source_state"] != "held" || m["state"] != "held" || m["reason"] != "invalid_precision" {
				t.Fatal("legacy precision exposed as usable")
			}
			for _, k := range []string{"amount", "direction", "currency", "currency_version_id", "numeric_code", "minor_unit", "valid_from", "valid_to"} {
				if m[k] != nil {
					t.Fatal("held financial field", k)
				}
			}
		}
		if !found {
			t.Fatal("original hold absent")
		}
		if before != legacyHash(t, owner) {
			t.Fatal("projection rewrote original precision/source/journal")
		}
		exec(t, owner, `UPDATE ledger.membership SET revoked_at=clock_timestamp()WHERE entity_id=$1 AND principal_id=$2 AND permission='read'`, entityID, actor)
		after := page(t, db, w)
		if q["partial"] != after["partial"] || after["partial"] != true {
			t.Fatal("book metadata redefined history coverage")
		}
	})
}

// TestLabelExecutionAtomicRollback forces failure after the first insert and proves no partial durable effect.
func TestLabelExecutionAtomicRollback(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "rollback", 1)
	if _, e := publish(t, db, w, sources[0], "batch", "0", nil, "c1", []map[string]any{change("one", "10", "USD"), change("two", "20", "USD"), change("three", "30", "USD")}); e != nil {
		t.Fatal(e)
	}
	q := page(t, db, w)
	p, e := preview(t, db, w, q["snapshot"], selection(q["items"].([]any)), map[string]any{"kind": "uncategorized"})
	if e != nil {
		t.Fatal(e)
	}
	r, e := prepare(t, db, w, "atomic", p)
	if e != nil {
		t.Fatal(e)
	}
	id := command(t, r)["command_id"].(string)
	exec(t, owner, "CREATE SEQUENCE banking.proof_attempts")
	exec(t, owner, `CREATE FUNCTION banking.proof_fail_second()RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$BEGIN IF nextval('banking.proof_attempts')>=2 THEN RAISE EXCEPTION USING ERRCODE='P2802',MESSAGE='synthetic_failure';END IF;RETURN NEW;END$$`)
	exec(t, owner, "REVOKE ALL ON FUNCTION banking.proof_fail_second()FROM PUBLIC")
	exec(t, owner, "CREATE TRIGGER proof_fail_second BEFORE INSERT ON banking.collection_label_decision FOR EACH ROW EXECUTE FUNCTION banking.proof_fail_second()")
	_, e = transition(t, db, w, id, "execute")
	code(t, e, "P2802")
	if value(t, owner, "SELECT last_value::text FROM banking.proof_attempts") != "2" {
		t.Fatal("late failure schedule not reached")
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != "0" || value(t, owner, "SELECT decision_revision::text FROM banking.collection_label_state") != "0" || value(t, owner, "SELECT state FROM banking.collection_label_command") != "prepared" || value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action WHERE operation='execute'") != "0" {
		t.Fatal("partial committed effect")
	}
	exec(t, owner, "DROP TRIGGER proof_fail_second ON banking.collection_label_decision")
	exec(t, owner, "DROP FUNCTION banking.proof_fail_second()")
	exec(t, owner, "DROP SEQUENCE banking.proof_attempts")
	r, e = transition(t, db, w, id, "execute")
	if e != nil || command(t, r)["state"] != "committed" || value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != "3" {
		t.Fatal("same command recovery after atomic abort", e)
	}
}

// TestLabelExecuteCancelRace proves both terminal winners under actual workspace/state locks.
func TestLabelExecuteCancelRace(t *testing.T) {
	for _, winner := range []string{"execute", "cancel"} {
		t.Run(winner, func(t *testing.T) {
			owner, db := fixture(t)
			w := provision(t, db, actor)
			_, sources := register(t, db, w, actor, "race", 1)
			if _, e := publish(t, db, w, sources[0], "batch", "0", nil, "c1", []map[string]any{change("tx", "10", "USD")}); e != nil {
				t.Fatal(e)
			}
			q := page(t, db, w)
			p, e := preview(t, db, w, q["snapshot"], selection(q["items"].([]any)), map[string]any{"kind": "uncategorized"})
			if e != nil {
				t.Fatal(e)
			}
			r, e := prepare(t, db, w, "race", p)
			if e != nil {
				t.Fatal(e)
			}
			id := command(t, r)["command_id"].(string)
			first := connect(t, db, "synthetic_runtime")
			defer first.Close(context.Background())
			second := connect(t, db, "synthetic_runtime")
			defer second.Close(context.Background())
			tx, e := first.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(context.Background())
			var result []byte
			if e = tx.QueryRow(context.Background(), "SELECT banking.transition_collection_label_command($1,$2,$3,$4,'{}')", actor, w, id, winner).Scan(&result); e != nil {
				t.Fatal(e)
			}
			loser := "cancel"
			state := "committed"
			wantCount := "1"
			if winner == "cancel" {
				loser = "execute"
				state = "cancelled"
				wantCount = "0"
			}
			done := make(chan error, 1)
			go func() {
				rt, err := second.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
				if err == nil {
					var b []byte
					err = rt.QueryRow(context.Background(), "SELECT banking.transition_collection_label_command($1,$2,$3,$4,'{}')", actor, w, id, loser).Scan(&b)
					if err == nil {
						err = rt.Commit(context.Background())
					}
					_ = rt.Rollback(context.Background())
				}
				done <- err
			}()
			waitBlocked(t, owner, second.PgConn().PID())
			if e = tx.Commit(context.Background()); e != nil {
				t.Fatal(e)
			}
			code(t, <-done, "40001")
			r, e = transition(t, db, w, id, loser)
			if e != nil || command(t, r)["state"] != state {
				t.Fatal("lost response changed terminal winner", e)
			}
			if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != wantCount {
				t.Fatal("duplicate/false effect")
			}
		})
	}
}

// TestTopicRevisionConflict rejects a preview prepared against an earlier taxonomy edition.
func TestTopicRevisionConflict(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	snap := page(t, db, w)["snapshot"]
	old, e := preview(t, db, w, snap, []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Old", nil, "active")}))
	if e != nil {
		t.Fatal(e)
	}
	apply(t, db, w, "new-edition", []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "New", nil, "active")}))
	_, e = prepare(t, db, w, "stale", old)
	code(t, e, "P2803")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_taxonomy_edition") != "2" {
		t.Fatal("stale edition wrote")
	}
}

// TestLabelClosedWire rejects unknown/duplicate keys and foreign/stale explicit selections without effects.
func TestLabelClosedWire(t *testing.T) {
	owner, db := fixture(t)
	w := provision(t, db, actor)
	for _, body := range []string{`{}`, `null`, `{"profile":"x","profile":"y"}`, raw(selector()) + `{}`} {
		_, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.query_collection_review($1,$2,$3)", actor, w, body)
		code(t, e, "P2800")
	}
	v := selector()
	v["unexpected"] = true
	_, e := query(t, db, w, actor, v)
	code(t, e, "P2800")
	_, sources := register(t, db, w, actor, "closed", 1)
	if _, e = publish(t, db, w, sources[0], "batch", "0", nil, "c1", []map[string]any{change("tx", "1", "USD")}); e != nil {
		t.Fatal(e)
	}
	q := page(t, db, w)
	selected := selection(q["items"].([]any))
	_, e = preview(t, db, w, q["snapshot"], append(selected, selected[0]), map[string]any{"kind": "uncategorized"})
	code(t, e, "P2800")
	selected[0]["observation_revision"] = "2"
	_, e = preview(t, db, w, q["snapshot"], selected, map[string]any{"kind": "uncategorized"})
	code(t, e, "P2803")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_preview") != "0" {
		t.Fatal("invalid preview persisted")
	}
}

// TestReadOnlyCollectionDiscovery permits idle read-only browsing but never discloses inaccessible saved intent.
func TestReadOnlyCollectionDiscovery(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprint(pending), func(t *testing.T) {
			_, db := fixture(t)
			w := provision(t, db, actor)
			if pending {
				p, e := preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{}))
				if e != nil {
					t.Fatal(e)
				}
				if _, e = prepare(t, db, w, "pending", p); e != nil {
					t.Fatal(e)
				}
			}
			_, e := call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'read-only',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{"read"}, "evidence_ref": evidence}))
			if e != nil {
				t.Fatal(e)
			}
			if catalog(t, db, w)["authority_revision"] != "2" || page(t, db, w)["matching_count"] != "0" {
				t.Fatal("read-only catalog/query denied")
			}
			r, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.pending_collection_label_command($1,$2)", actor, w)
			if pending {
				code(t, e, "P2801")
			} else if e != nil || r["command"] != nil {
				t.Fatal("idle read-only pending discovery", e)
			}
			_, e = preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{}))
			code(t, e, "P2801")
		})
	}
}

// TestReviewAstralSelectorCursor proves a maximal-codepoint selector survives both bounded wire pages.
func TestReviewAstralSelectorCursor(t *testing.T) {
	_, db := fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "astral-selector", 1)
	search := strings.Repeat("🪶", 100)
	one := change("one", "10.01", "USD")
	one["description"] = search + " A"
	two := change("two", "20.02", "USD")
	two["description"] = search + " B"
	if _, e := publish(t, db, w, sources[0], "batch", "0", nil, "c1", []map[string]any{one, two}); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	apply(t, db, w, "labels", selection(page(t, db, w)["items"].([]any)), map[string]any{"kind": "topic", "topic_id": childTopic})
	v := selector()
	v["source_account_id"] = sources[0]
	v["assignment"] = "unassigned"
	v["starts_on"] = "2026-01-01"
	v["through_date"] = "2026-12-31"
	v["search"] = search
	v["topic"] = map[string]any{"kind": "topic", "topic_id": childTopic}
	v["sort"] = "date_asc"
	v["limit"] = 1
	first, e := query(t, db, w, actor, v)
	if e != nil {
		t.Fatal(e)
	}
	if first["matching_count"] != "2" || first["start_index"] != "1" || len(first["items"].([]any)) != 1 {
		t.Fatal("astral first page")
	}
	cursor := first["next_cursor"].(string)
	if len(cursor) == 0 || len(cursor) > 4096 {
		t.Fatalf("cursor exceeds frozen bound: %d", len(cursor))
	}
	v["cursor"] = cursor
	v["snapshot"] = first["snapshot"]
	if len([]byte(raw(v))) > 8192 {
		t.Fatal("query envelope exceeds wire bound")
	}
	second, e := query(t, db, w, actor, v)
	if e != nil {
		t.Fatal(e)
	}
	if second["matching_count"] != "2" || second["start_index"] != "2" || second["next_cursor"] != nil || len(second["items"].([]any)) != 1 || first["items"].([]any)[0].(map[string]any)["observation_id"] == second["items"].([]any)[0].(map[string]any)["observation_id"] {
		t.Fatal("astral second page identity/count")
	}
	t.Logf("100 astral codepoints, account/topic/date/assignment selector, sixpins: cursor %d hexchars; resumed query %d bytes", len(cursor), len([]byte(raw(v))))
}
