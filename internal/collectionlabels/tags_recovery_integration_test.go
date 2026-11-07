//go:build integration

package collectionlabels

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
)

const tagA = "00000000-0000-4000-8000-000000003001"
const tagB = "00000000-0000-4000-8000-000000003002"

// tagrecFixture installs fictional principals/units in an isolated latest44 clone.
func tagrecFixture(t *testing.T) (*pgx.Conn, string) {
	t.Helper()
	hash := sha256.Sum256([]byte(t.Name()))
	db := fmt.Sprintf("tagrec_%x_%d", hash[:4], time.Now().UnixNano())
	root := connect(t, "postgres", "postgres")
	defer root.Close(context.Background())
	exec(t, root, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE tagrec_template")
	owner := connect(t, db, "postgres")
	t.Cleanup(func() { owner.Close(context.Background()) })
	exec(t, owner, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic','owner-a'),($2,'synthetic','owner-b')`, actor, other)
	exec(t, owner, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000006','JPY','392',0,'2000-01-01',NULL,'synthetic'),('00000000-0000-4000-8000-000000000007','KWD','414',3,'2000-01-01',NULL,'synthetic')`, currency)
	return owner, db
}

// tagrecSelector carries the complete v2 orthogonal filter, never a browser-side tag expansion.
func tagrecSelector() map[string]any {
	v := selector()
	v["profile"] = "private-collection-review-v2"
	v["tag"] = map[string]any{"kind": "all"}
	return v
}
func tagrecQuery(t testing.TB, db, w, who string, v map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.query_collection_review_v2($1,$2,$3)", who, w, raw(v))
}
func tagrecPage(t testing.TB, db, w string) map[string]any {
	t.Helper()
	r, e := tagrecQuery(t, db, w, actor, tagrecSelector())
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func tagrecCatalog(t testing.TB, db, w string) map[string]any {
	t.Helper()
	r, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_taxonomy_v2($1,$2)", actor, w)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func tagrecPreview(t testing.TB, db, w string, snapshot any, selected []map[string]any, action any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.preview_collection_labels_v2($1,$2,$3)", actor, w, raw(map[string]any{"snapshot": snapshot, "selection": selected, "action": action}))
}
func tagrecPrepare(t testing.TB, db, w, key string, p map[string]any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)", actor, w, raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": key}))
}
func tagrecTransition(t testing.TB, db, w, id, op string) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.transition_collection_label_command_v2($1,$2,$3,$4,'{}')", actor, w, id, op)
}
func tagrecCommand(t testing.TB, r map[string]any) map[string]any {
	t.Helper()
	if r["profile"] != "private-collection-command-v2" {
		t.Fatal("v2 command profile")
	}
	return r["command"].(map[string]any)
}
func tagrecApply(t testing.TB, db, w, key string, selected []map[string]any, action any) map[string]any {
	t.Helper()
	p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], selected, action)
	if e != nil {
		t.Fatal(e)
	}
	r, e := tagrecPrepare(t, db, w, key, p)
	if e != nil {
		t.Fatal(e)
	}
	id := tagrecCommand(t, r)["command_id"].(string)
	r, e = tagrecTransition(t, db, w, id, "execute")
	if e != nil {
		t.Fatal(e)
	}
	if tagrecCommand(t, r)["state"] != "committed" {
		t.Fatal("not committed")
	}
	if _, e = tagrecTransition(t, db, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	return r
}
func tag(id, name, status string) map[string]any {
	return map[string]any{"tag_id": id, "name": name, "status": status}
}
func tagCatalog(tags ...map[string]any) map[string]any {
	if tags == nil {
		tags = []map[string]any{}
	}
	return map[string]any{"kind": "tag_catalog", "tags": tags}
}
func tagAction(op string, ids ...string) map[string]any {
	if ids == nil {
		ids = []string{}
	}
	return map[string]any{"kind": "tags", "operation": op, "tag_ids": ids}
}
func seedTags(t testing.TB, db, w string) {
	t.Helper()
	tagrecApply(t, db, w, "tag-catalog", []map[string]any{}, tagCatalog(tag(tagA, "Travel", "active"), tag(tagB, "Reimburse", "active")))
}
func recoveryStatus(t testing.TB, db, who string) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.collection_recovery_status($1)", who)
}
func recoveryClear(t testing.TB, db, who string, id any) (map[string]any, error) {
	t.Helper()
	return call(t, db, "synthetic_runtime", "serializable", "SELECT banking.clear_collection_recovery($1,$2)", who, raw(map[string]any{"recovery_id": id}))
}
func assertKeys(t testing.TB, v map[string]any, names ...string) {
	t.Helper()
	if len(v) != len(names) {
		t.Fatal("unexpected projection fields")
	}
	for _, name := range names {
		if _, ok := v[name]; !ok {
			t.Fatal("missing projection field", name)
		}
	}
}

// TestTagRecActions proves tag sets stay independent from primary topics and immutable source evidence.
func TestTagRecActions(t *testing.T) {
	owner, db := tagrecFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "tags", 1)
	if _, e := publish(t, db, w, sources[0], "one", "0", nil, "c1", []map[string]any{change("one", "9007199254740993.01", "USD"), change("two", "20.02", "USD")}); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	seedTags(t, db, w)
	q := tagrecPage(t, db, w)
	selected := selection(q["items"].([]any))
	tagrecApply(t, db, w, "primary", selected, map[string]any{"kind": "topic", "topic_id": childTopic})
	r := tagrecApply(t, db, w, "add", selected, tagAction("add", tagB, tagA))
	p := tagrecCommand(t, r)["preview"].(map[string]any)
	if len(p["tag_results"].([]any)) != 2 {
		t.Fatal("missing resulting sets")
	}
	q = tagrecPage(t, db, w)
	for _, x := range q["items"].([]any) {
		row := x.(map[string]any)
		tags := row["owner_tags"].(map[string]any)
		if tags["state"] != "current" || len(tags["tags"].([]any)) != 2 || row["owner_topic"].(map[string]any)["topic_id"] != childTopic {
			t.Fatal("tag/topic separation")
		}
	}
	v := tagrecSelector()
	v["tag"] = map[string]any{"kind": "tag", "tag_id": tagA}
	q, e := tagrecQuery(t, db, w, actor, v)
	if e != nil || q["matching_count"] != "2" {
		t.Fatal("tag duplicated matching count", e)
	}
	tagrecApply(t, db, w, "remove", selected[:1], tagAction("remove", tagA))
	tagrecApply(t, db, w, "replace-empty", selected[1:], tagAction("replace"))
	q = tagrecPage(t, db, w)
	v["tag"] = map[string]any{"kind": "untagged"}
	r, e = tagrecQuery(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "1" {
		t.Fatal("explicit empty set", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision") != "4" || value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("tag decision history/journal")
	}
	old := page(t, db, w)
	if _, ok := old["snapshot"].(map[string]any)["tag_revision"]; ok {
		t.Fatal("v1 snapshot widened")
	}
	for _, x := range old["items"].([]any) {
		if _, ok := x.(map[string]any)["owner_tags"]; ok {
			t.Fatal("v1 row widened")
		}
	}
}

// TestTagRecCatalog proves stable identities, terminal retirement and historic preview names.
func TestTagRecCatalog(t *testing.T) {
	owner, db := tagrecFixture(t)
	w := provision(t, db, actor)
	c := tagrecCatalog(t, db, w)
	if c["tag_revision"] != "1" || len(c["tags"].([]any)) != 0 {
		t.Fatal("empty tag edition")
	}
	seedTags(t, db, w)
	tagrecApply(t, db, w, "retire", []map[string]any{}, tagCatalog(tag(tagA, "Travel renamed", "active"), tag(tagB, "Reimburse", "retired")))
	if tagrecCatalog(t, db, w)["tag_revision"] != "3" {
		t.Fatal("tag revision")
	}
	_, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog(tag(tagA, "Travel", "active")))
	code(t, e, "P2802")
	_, e = tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog(tag(tagA, "Travel", "active"), tag(tagB, "Reimburse", "active")))
	code(t, e, "P2802")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_edition") != "3" {
		t.Fatal("catalog history")
	}
}

// TestTagRecRecoveryPrepared proves revocation unlocks only through explicit minimal clear and replay.
func TestTagRecRecoveryPrepared(t *testing.T) {
	owner, db := tagrecFixture(t)
	w := provision(t, db, actor)
	p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog(tag(tagA, "Private name", "active")))
	if e != nil {
		t.Fatal(e)
	}
	r, e := tagrecPrepare(t, db, w, "pending", p)
	if e != nil {
		t.Fatal(e)
	}
	id := tagrecCommand(t, r)["command_id"].(string)
	s, e := recoveryStatus(t, db, actor)
	if e != nil || s["state"] != "available" {
		t.Fatal(e)
	}
	assertKeys(t, s, "profile", "state", "recovery_id")
	token := s["recovery_id"]
	_, e = recoveryClear(t, db, actor, token)
	code(t, e, "P2802")
	_, e = call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'revoke',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{}, "evidence_ref": evidence}))
	if e != nil {
		t.Fatal(e)
	}
	s, e = recoveryStatus(t, db, actor)
	if e != nil || s["state"] != "inaccessible" || s["recovery_id"] != token {
		t.Fatal("revoked status", e)
	}
	r, e = recoveryClear(t, db, actor, token)
	if e != nil {
		t.Fatal(e)
	}
	assertKeys(t, r, "profile", "recovery_id", "cleared")
	again, e := recoveryClear(t, db, actor, token)
	if e != nil || raw(again) != raw(r) {
		t.Fatal("lost clear replay", e)
	}
	if value(t, owner, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM banking.collection_label_command WHERE command_id=$1", id) != "cancelled:true" || value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation") != "0" || value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_edition") != "1" {
		t.Fatal("clear executed or lost claim")
	}
	s, e = recoveryStatus(t, db, actor)
	if e != nil || s["state"] != "none" {
		t.Fatal(e)
	}
}

// legacyFixture uses existing source/enrollment/posting SQL and independently fixed old tuples.
func tagrecLegacyFixture(t *testing.T) (*pgx.Conn, string, map[string]any) {
	t.Helper()
	owner, db := tagrecFixture(t)
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

// TestTagRecSharedGate races independently connected old/private engines under a primed snapshot.
func TestTagRecSharedGate(t *testing.T) {
	for _, winner := range []string{"ledger", "private"} {
		t.Run(winner, func(t *testing.T) {
			owner, db, _ := tagrecLegacyFixture(t)
			w := provision(t, db, actor)
			before := legacyHash(t, owner)
			p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog())
			if e != nil {
				t.Fatal(e)
			}
			body := ledgerIntent(t, owner)
			first := connect(t, db, "synthetic_runtime")
			defer first.Close(context.Background())
			second := connect(t, db, "synthetic_runtime")
			defer second.Close(context.Background())
			loserTx, e := second.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer loserTx.Rollback(context.Background())
			var count int
			if e = loserTx.QueryRow(context.Background(), "SELECT count(*) FROM pg_catalog.pg_class").Scan(&count); e != nil {
				t.Fatal(e)
			}
			tx, e := first.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(context.Background())
			privateBody := raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "private-race"})
			ledgerSQL := "SELECT ledger.register_command($1,'ledger-race',$2,$3)"
			privateSQL := "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)"
			var bytes []byte
			if winner == "ledger" {
				e = tx.QueryRow(context.Background(), ledgerSQL, actor, evidence, body).Scan(&bytes)
			} else {
				e = tx.QueryRow(context.Background(), privateSQL, actor, w, privateBody).Scan(&bytes)
			}
			if e != nil {
				t.Fatal(e)
			}
			var firstResult map[string]any
			if e = json.Unmarshal(bytes, &firstResult); e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() {
				var result []byte
				var err error
				if winner == "ledger" {
					err = loserTx.QueryRow(context.Background(), privateSQL, actor, w, privateBody).Scan(&result)
				} else {
					err = loserTx.QueryRow(context.Background(), ledgerSQL, actor, evidence, body).Scan(&result)
				}
				if err == nil {
					err = loserTx.Commit(context.Background())
				}
				_ = loserTx.Rollback(context.Background())
				done <- err
			}()
			waitBlocked(t, owner, second.PgConn().PID())
			if e = tx.Commit(context.Background()); e != nil {
				t.Fatal(e)
			}
			code(t, <-done, "40001")
			assertBusy := func() {
				t.Helper()
				if winner == "ledger" {
					_, err := tagrecPrepare(t, db, w, "private-race", p)
					code(t, err, "P2802")
				} else {
					_, err := prepareLedger(t, db, "ledger-race", body)
					code(t, err, "P2112")
				}
				if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation WHERE principal_id=$1", actor) != "1" {
					t.Fatal("global reservation not unique")
				}
			}
			assertBusy()
			var id string
			if winner == "ledger" {
				id = firstResult["command_id"].(string)
				replay, err := prepareLedger(t, db, "ledger-race", body)
				if err != nil || raw(replay) != raw(firstResult) {
					t.Fatal("ledger same-key recovery", err)
				}
				if _, err = transitionLedger(t, db, id, "cancel"); err != nil {
					t.Fatal(err)
				}
			} else {
				id = tagrecCommand(t, firstResult)["command_id"].(string)
				replay, err := tagrecPrepare(t, db, w, "private-race", p)
				if err != nil || raw(replay) != raw(firstResult) {
					t.Fatal("private same-key recovery", err)
				}
				if _, err = tagrecTransition(t, db, w, id, "cancel"); err != nil {
					t.Fatal(err)
				}
			}
			assertBusy()
			if winner == "ledger" {
				if _, e = transitionLedger(t, db, id, "acknowledge"); e != nil {
					t.Fatal(e)
				}
			} else {
				if _, e = tagrecTransition(t, db, w, id, "acknowledge"); e != nil {
					t.Fatal(e)
				}
			}
			if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation WHERE principal_id=$1", actor) != "0" {
				t.Fatal("explicit ack did not release")
			}
			if winner == "ledger" {
				otherCommand, err := tagrecPrepare(t, db, w, "private-race", p)
				if err != nil {
					t.Fatal("private cannot prepare after ledger ack", err)
				}
				replay, err := prepareLedger(t, db, "ledger-race", body)
				if err != nil || replay["acknowledged"] != true {
					t.Fatal("historic ledger replay blocked by private pending", err)
				}
				otherID := tagrecCommand(t, otherCommand)["command_id"].(string)
				if _, err = tagrecTransition(t, db, w, otherID, "cancel"); err != nil {
					t.Fatal(err)
				}
				if _, err = tagrecTransition(t, db, w, otherID, "acknowledge"); err != nil {
					t.Fatal(err)
				}
			} else {
				otherCommand, err := prepareLedger(t, db, "ledger-race", body)
				if err != nil {
					t.Fatal("ledger cannot prepare after private ack", err)
				}
				replay, err := tagrecPrepare(t, db, w, "private-race", p)
				if err != nil || tagrecCommand(t, replay)["acknowledged"] != true {
					t.Fatal("historic private replay blocked by ledger pending", err)
				}
				otherID := otherCommand["command_id"].(string)
				if _, err = transitionLedger(t, db, otherID, "cancel"); err != nil {
					t.Fatal(err)
				}
				if _, err = transitionLedger(t, db, otherID, "acknowledge"); err != nil {
					t.Fatal(err)
				}
			}
			if before != legacyHash(t, owner) {
				t.Fatal("global recovery gate changed original source/journal/history")
			}
		})
	}
}

// TestTagRecStaleAndRetired proves corrections require explicit replacement while retired identities remain removable.
func TestTagRecStaleAndRetired(t *testing.T) {
	_, db := tagrecFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "stale-tags", 1)
	if _, e := publish(t, db, w, sources[0], "one", "0", nil, "c1", []map[string]any{change("tx", "10.01", "USD")}); e != nil {
		t.Fatal(e)
	}
	seedTags(t, db, w)
	q := tagrecPage(t, db, w)
	selected := selection(q["items"].([]any))
	tagrecApply(t, db, w, "tag", selected, tagAction("add", tagA, tagB))
	tagrecApply(t, db, w, "retire", []map[string]any{}, tagCatalog(tag(tagA, "Travel renamed", "active"), tag(tagB, "Reimburse", "retired")))
	q = tagrecPage(t, db, w)
	tags := q["items"].([]any)[0].(map[string]any)["owner_tags"].(map[string]any)["tags"].([]any)
	if len(tags) != 2 || tags[0].(map[string]any)["name"] != "Travel renamed" || tags[1].(map[string]any)["status"] != "retired" {
		t.Fatal("current rename/retirement projection")
	}
	_, e := tagrecPreview(t, db, w, q["snapshot"], selected, tagAction("replace", tagB))
	code(t, e, "P2803")
	tagrecApply(t, db, w, "remove-retired", selected, tagAction("remove", tagB))
	z := change("tx", "15.00", "USD")
	z["operation"] = "modified"
	if _, e = publish(t, db, w, sources[0], "correct", "1", "c1", "c2", []map[string]any{z}); e != nil {
		t.Fatal(e)
	}
	q = tagrecPage(t, db, w)
	row := q["items"].([]any)[0].(map[string]any)
	if row["owner_tags"].(map[string]any)["state"] != "stale" {
		t.Fatal("corrected tags not stale")
	}
	selected = selection(q["items"].([]any))
	for _, op := range []string{"add", "remove"} {
		_, e = tagrecPreview(t, db, w, q["snapshot"], selected, tagAction(op, tagA))
		code(t, e, "P2803")
	}
	tagrecApply(t, db, w, "confirm-replacement", selected, tagAction("replace", tagA))
	q = tagrecPage(t, db, w)
	if q["items"].([]any)[0].(map[string]any)["owner_tags"].(map[string]any)["state"] != "current" {
		t.Fatal("explicit replacement did not revalidate")
	}
}

// TestTagRecBoundsAndForeign enforces scoped IDs, codepoint limits and complete resulting-set bounds.
func TestTagRecBoundsAndForeign(t *testing.T) {
	owner, db := tagrecFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "bounds", 1)
	if _, e := publish(t, db, w, sources[0], "one", "0", nil, "c1", []map[string]any{change("tx", "10", "USD")}); e != nil {
		t.Fatal(e)
	}
	snap := tagrecPage(t, db, w)["snapshot"]
	for _, name := range []string{"", " ", "Bad\nname", strings.Repeat("🪶", 81)} {
		_, e := tagrecPreview(t, db, w, snap, []map[string]any{}, tagCatalog(tag(tagA, name, "active")))
		code(t, e, "P2800")
	}
	_, e := tagrecPreview(t, db, w, snap, []map[string]any{}, tagCatalog(tag(tagA, "same", "active"), tag(tagB, "same", "active")))
	code(t, e, "P2800")
	many := []map[string]any{}
	for i := 0; i < 501; i++ {
		many = append(many, tag(fmt.Sprintf("10000000-0000-4000-8000-%012d", i), fmt.Sprint(i), "active"))
	}
	_, e = tagrecPreview(t, db, w, snap, []map[string]any{}, tagCatalog(many...))
	code(t, e, "P2805")
	catalogTags := []map[string]any{}
	ids := []string{}
	for i := 0; i < 26; i++ {
		id := fmt.Sprintf("10000000-0000-4000-8000-%012d", i)
		ids = append(ids, id)
		name := fmt.Sprint(i)
		if i == 0 {
			name = strings.Repeat("🪶", 80)
		}
		catalogTags = append(catalogTags, tag(id, name, "active"))
	}
	tagrecApply(t, db, w, "catalog", []map[string]any{}, tagCatalog(catalogTags...))
	q := tagrecPage(t, db, w)
	selected := selection(q["items"].([]any))
	_, e = tagrecPreview(t, db, w, q["snapshot"], selected, tagAction("add", tagA))
	code(t, e, "P2801")
	_, e = tagrecPreview(t, db, w, q["snapshot"], selected, tagAction("add", ids[0], ids[0]))
	code(t, e, "P2800")
	_, e = tagrecPreview(t, db, w, q["snapshot"], selected, tagAction("replace", ids...))
	code(t, e, "P2805")
	tagrecApply(t, db, w, "max-set", selected, tagAction("replace", ids[:25]...))
	_, e = tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], selected, tagAction("add", ids[25]))
	code(t, e, "P2805")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision") != "1" {
		t.Fatal("failed set expanded history")
	}
	_, e = tagrecQuery(t, db, w, other, tagrecSelector())
	code(t, e, "P2801")
	_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.preview_collection_labels_v2($1,$2,'{}')", other, w)
	code(t, e, "P2801")
}

// TestTagRecV1Compatibility retains closed old reads and exact immutable old preview recovery through v2.
func TestTagRecV1Compatibility(t *testing.T) {
	_, db := tagrecFixture(t)
	w := provision(t, db, actor)
	before := page(t, db, w)
	oldPreview, e := preview(t, db, w, before["snapshot"], []map[string]any{}, topicsAction([]map[string]any{}))
	if e != nil {
		t.Fatal(e)
	}
	r, e := tagrecPrepare(t, db, w, "old-preview", oldPreview)
	if e != nil {
		t.Fatal(e)
	}
	inner := tagrecCommand(t, r)["preview"].(map[string]any)
	if raw(inner) != raw(oldPreview) || inner["profile"] != "private-collection-preview-v1" {
		t.Fatal("historic v1 preview was widened")
	}
	id := tagrecCommand(t, r)["command_id"].(string)
	if _, e = tagrecTransition(t, db, w, id, "cancel"); e != nil {
		t.Fatal(e)
	}
	if _, e = tagrecTransition(t, db, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	seedTags(t, db, w)
	oldCatalog := catalog(t, db, w)
	if _, ok := oldCatalog["tags"]; ok {
		t.Fatal("v1 catalog widened")
	}
	v := selector()
	v["snapshot"] = before["snapshot"]
	_, e = query(t, db, w, actor, v)
	code(t, e, "P2803")
	p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog(tag(tagA, "Travel", "active"), tag(tagB, "Reimburse", "active")))
	if e != nil {
		t.Fatal(e)
	}
	r, e = tagrecPrepare(t, db, w, "v2-only", p)
	if e != nil {
		t.Fatal(e)
	}
	_, e = call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.pending_collection_label_command($1,$2)", actor, w)
	code(t, e, "P2804")
	_, e = transition(t, db, w, tagrecCommand(t, r)["command_id"].(string), "cancel")
	code(t, e, "P2804")
}

// TestTagRecRecoveryTerminal preserves every prior decision/state while clearing only inaccessible intent.
func TestTagRecRecoveryTerminal(t *testing.T) {
	for _, state := range []string{"prepared", "committed", "cancelled"} {
		t.Run(state, func(t *testing.T) {
			owner, db := tagrecFixture(t)
			w := provision(t, db, actor)
			p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog(tag(tagA, "Private topic-like tag", "active")))
			if e != nil {
				t.Fatal(e)
			}
			r, e := tagrecPrepare(t, db, w, "terminal", p)
			if e != nil {
				t.Fatal(e)
			}
			id := tagrecCommand(t, r)["command_id"].(string)
			if state != "prepared" {
				op := "execute"
				if state == "cancelled" {
					op = "cancel"
				}
				if _, e = tagrecTransition(t, db, w, id, op); e != nil {
					t.Fatal(e)
				}
			}
			before := value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_edition")
			s, e := recoveryStatus(t, db, actor)
			if e != nil {
				t.Fatal(e)
			}
			token := s["recovery_id"]
			_, e = call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'read-only',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{"read"}, "evidence_ref": evidence}))
			if e != nil {
				t.Fatal(e)
			}
			_, e = recoveryClear(t, db, other, token)
			code(t, e, "P2801")
			r, e = recoveryClear(t, db, actor, token)
			if e != nil {
				t.Fatal(e)
			}
			assertKeys(t, r, "profile", "recovery_id", "cleared")
			want := state
			if want == "prepared" {
				want = "cancelled"
			}
			if value(t, owner, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM banking.collection_label_command WHERE command_id=$1", id) != want+":true" || value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_edition") != before {
				t.Fatal("clear rewrote terminal decisions")
			}
			if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
				t.Fatal("clear posted")
			}
			exec(t, owner, "UPDATE ledger.principal SET disabled_at=clock_timestamp()WHERE principal_id=$1", actor)
			_, e = recoveryStatus(t, db, actor)
			code(t, e, "P2801")
			_, e = recoveryClear(t, db, actor, token)
			code(t, e, "P2801")
		})
	}
}

// TestTagRecRecoveryRestoredAndNewClaim handles ordinary acknowledgement after a lost clear without touching newer work.
func TestTagRecRecoveryRestoredAndNewClaim(t *testing.T) {
	owner, db, _ := tagrecLegacyFixture(t)
	w := provision(t, db, actor)
	p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog())
	if e != nil {
		t.Fatal(e)
	}
	r, e := tagrecPrepare(t, db, w, "a", p)
	if e != nil {
		t.Fatal(e)
	}
	a := tagrecCommand(t, r)["command_id"].(string)
	s, e := recoveryStatus(t, db, actor)
	if e != nil {
		t.Fatal(e)
	}
	tokenA := s["recovery_id"]
	for i, caps := range [][]string{{"read"}, {"read", "acquire", "classify"}} {
		_, e = call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,$2,$3)", actor, fmt.Sprintf("caps%d", i), raw(map[string]any{"expected_revision": fmt.Sprint(i + 1), "capabilities": caps, "evidence_ref": evidence}))
		if e != nil {
			t.Fatal(e)
		}
	}
	_, e = recoveryClear(t, db, actor, tokenA)
	code(t, e, "P2802")
	// The clear response was never proof. Another explicit ordinary cancel/ack supplies durable closure.
	if _, e = tagrecTransition(t, db, w, a, "cancel"); e != nil {
		t.Fatal(e)
	}
	if _, e = tagrecTransition(t, db, w, a, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	p, e = tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog())
	if e != nil {
		t.Fatal(e)
	}
	r, e = tagrecPrepare(t, db, w, "b", p)
	if e != nil {
		t.Fatal(e)
	}
	b := tagrecCommand(t, r)["command_id"].(string)
	s, e = recoveryStatus(t, db, actor)
	if e != nil {
		t.Fatal(e)
	}
	tokenB := s["recovery_id"]
	if tokenA == tokenB {
		t.Fatal("reused claim identity")
	}
	r, e = recoveryClear(t, db, actor, tokenA)
	if e != nil || r["cleared"] != true {
		t.Fatal("ordinary ack closure replay", e)
	}
	if value(t, owner, "SELECT label_command_id::text FROM ledger.interactive_command_reservation") != b {
		t.Fatal("old clear released new private claim")
	}
	if _, e = tagrecTransition(t, db, w, b, "cancel"); e != nil {
		t.Fatal(e)
	}
	if _, e = tagrecTransition(t, db, w, b, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	ledger, e := prepareLedger(t, db, "new-book", ledgerIntent(t, owner))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = recoveryClear(t, db, actor, tokenA); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT legacy_command_id::text FROM ledger.interactive_command_reservation") != ledger["command_id"] {
		t.Fatal("old clear released new book claim")
	}
	s, e = recoveryStatus(t, db, actor)
	if e != nil || s["state"] != "none" || s["recovery_id"] != nil {
		t.Fatal("book claim exposed through private recovery", e)
	}
}

// TestTagRecRecoveryPrimedStatus proves no false absence after a claim commits behind a stale principal lock wait.
func TestTagRecRecoveryPrimedStatus(t *testing.T) {
	owner, db := tagrecFixture(t)
	w := provision(t, db, actor)
	p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog())
	if e != nil {
		t.Fatal(e)
	}
	reader := connect(t, db, "synthetic_runtime")
	defer reader.Close(context.Background())
	readTx, e := reader.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer readTx.Rollback(context.Background())
	var count int
	if e = readTx.QueryRow(context.Background(), "SELECT count(*)FROM pg_catalog.pg_class").Scan(&count); e != nil {
		t.Fatal(e)
	}
	writer := connect(t, db, "synthetic_runtime")
	defer writer.Close(context.Background())
	tx, e := writer.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	var result []byte
	if e = tx.QueryRow(context.Background(), "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)", actor, w, raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "race"})).Scan(&result); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		var b []byte
		err := readTx.QueryRow(context.Background(), "SELECT banking.collection_recovery_status($1)", actor).Scan(&b)
		_ = readTx.Rollback(context.Background())
		done <- err
	}()
	waitBlocked(t, owner, reader.PgConn().PID())
	if e = tx.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	code(t, <-done, "40001")
	r, e := recoveryStatus(t, db, actor)
	if e != nil || r["state"] != "available" {
		t.Fatal("fresh claim absent", e)
	}
}

// TestTagRecStaleIsNotInaccessible keeps changed source pins in ordinary review rather than bypassing it.
func TestTagRecStaleIsNotInaccessible(t *testing.T) {
	_, db := tagrecFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "stale-recovery", 1)
	if _, e := publish(t, db, w, sources[0], "one", "0", nil, "c1", []map[string]any{change("tx", "10", "USD")}); e != nil {
		t.Fatal(e)
	}
	q := tagrecPage(t, db, w)
	p, e := tagrecPreview(t, db, w, q["snapshot"], selection(q["items"].([]any)), map[string]any{"kind": "uncategorized"})
	if e != nil {
		t.Fatal(e)
	}
	r, e := tagrecPrepare(t, db, w, "a", p)
	if e != nil {
		t.Fatal(e)
	}
	id := tagrecCommand(t, r)["command_id"].(string)
	z := change("tx", "15", "USD")
	z["operation"] = "modified"
	if _, e = publish(t, db, w, sources[0], "two", "1", "c1", "c2", []map[string]any{z}); e != nil {
		t.Fatal(e)
	}
	s, e := recoveryStatus(t, db, actor)
	if e != nil || s["state"] != "available" {
		t.Fatal("stale ordinary work misclassified", e)
	}
	_, e = recoveryClear(t, db, actor, s["recovery_id"])
	code(t, e, "P2802")
	_, e = tagrecTransition(t, db, w, id, "execute")
	code(t, e, "P2803")
	if _, e = tagrecTransition(t, db, w, id, "cancel"); e != nil {
		t.Fatal(e)
	}
	if _, e = tagrecTransition(t, db, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
}

// TestTagRecPostedHistoryAndBookRecovery links independently posted evidence and proves current book metadata gating.
func TestTagRecPostedHistoryAndBookRecovery(t *testing.T) {
	owner, db, item := tagrecLegacyFixture(t)
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
	seedTags(t, db, w)
	before := legacyHash(t, owner)
	r := tagrecPage(t, db, w)
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
	tagrecApply(t, db, w, "retag-posted", selection(r["items"].([]any)), tagAction("add", tagA, tagB))
	if legacyHash(t, owner) != before {
		t.Fatal("retag changed original source/journal/outbox")
	}
	r = tagrecPage(t, db, w)
	p, e := tagrecPreview(t, db, w, r["snapshot"], selection(r["items"].([]any)), tagAction("add", tagA, tagB))
	if e != nil {
		t.Fatal(e)
	}
	cmd, e := tagrecPrepare(t, db, w, "pending-legacy", p)
	if e != nil {
		t.Fatal(e)
	}
	exec(t, owner, `UPDATE ledger.membership SET revoked_at=clock_timestamp()WHERE entity_id=$1 AND principal_id=$2 AND permission='read'`, entityID, actor)
	fresh := tagrecPage(t, db, w)
	for _, x := range fresh["items"].([]any) {
		row := x.(map[string]any)
		if row["assignment"].(map[string]any)["state"] != "unavailable" || row["journal_state"] != "unavailable" {
			t.Fatal("legacy metadata leaked")
		}
	}
	_, e = tagrecTransition(t, db, w, tagrecCommand(t, cmd)["command_id"].(string), "cancel")
	code(t, e, "P2801")
	status, er := recoveryStatus(t, db, actor)
	if er != nil || status["state"] != "inaccessible" {
		t.Fatal("book-revoked recovery", er)
	}
	if _, er = recoveryClear(t, db, actor, status["recovery_id"]); er != nil {
		t.Fatal(er)
	}
	if before != legacyHash(t, owner) {
		t.Fatal("payload-free book recovery changed journal/source")
	}
	v := tagrecSelector()
	v["snapshot"] = r["snapshot"]
	_, e = tagrecQuery(t, db, w, actor, v)
	code(t, e, "P2803")
	v = tagrecSelector()
	v["assignment"] = "unassigned"
	q, e := tagrecQuery(t, db, w, actor, v)
	if e != nil || q["matching_count"] != "0" {
		t.Fatal("unavailable mistaken for unassigned", e)
	}
}

// TestTagRecBulkAtomicity forces a late tag insert failure and then applies exactly100 explicit observations.
func TestTagRecBulkAtomicity(t *testing.T) {
	owner, db := tagrecFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "bulk-tags", 1)
	changes := []map[string]any{}
	for i := 0; i < 100; i++ {
		changes = append(changes, change(fmt.Sprintf("tx-%03d", i), "1.01", "USD"))
	}
	if _, e := publish(t, db, w, sources[0], "one", "0", nil, "c1", changes); e != nil {
		t.Fatal(e)
	}
	seedTags(t, db, w)
	q := tagrecPage(t, db, w)
	p, e := tagrecPreview(t, db, w, q["snapshot"], selection(q["items"].([]any)), tagAction("add", tagA, tagB))
	if e != nil {
		t.Fatal(e)
	}
	if len(p["tag_results"].([]any)) != 100 {
		t.Fatal("not exact100 result sets")
	}
	r, e := tagrecPrepare(t, db, w, "bulk", p)
	if e != nil {
		t.Fatal(e)
	}
	id := tagrecCommand(t, r)["command_id"].(string)
	exec(t, owner, "CREATE SEQUENCE banking.proof_tag_attempts")
	exec(t, owner, `CREATE FUNCTION banking.proof_tag_fail()RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$BEGIN IF nextval('banking.proof_tag_attempts')>=2 THEN RAISE EXCEPTION USING ERRCODE='P2802',MESSAGE='synthetic_tag_failure';END IF;RETURN NEW;END$$`)
	exec(t, owner, "REVOKE ALL ON FUNCTION banking.proof_tag_fail()FROM PUBLIC")
	exec(t, owner, "CREATE TRIGGER proof_tag_fail BEFORE INSERT ON banking.collection_tag_decision FOR EACH ROW EXECUTE FUNCTION banking.proof_tag_fail()")
	_, e = tagrecTransition(t, db, w, id, "execute")
	code(t, e, "P2802")
	if value(t, owner, "SELECT last_value::text FROM banking.proof_tag_attempts") != "2" || value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision") != "0" || value(t, owner, "SELECT decision_revision::text FROM banking.collection_label_state") != "0" || value(t, owner, "SELECT state FROM banking.collection_label_command WHERE command_id=$1", id) != "prepared" || value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action WHERE command_id=$1 AND operation='execute'", id) != "0" {
		t.Fatal("late failure partially committed")
	}
	exec(t, owner, "DROP TRIGGER proof_tag_fail ON banking.collection_tag_decision")
	exec(t, owner, "DROP FUNCTION banking.proof_tag_fail()")
	exec(t, owner, "DROP SEQUENCE banking.proof_tag_attempts")
	if _, e = tagrecTransition(t, db, w, id, "execute"); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision") != "100" {
		t.Fatal("bulk decision count")
	}
	v := tagrecSelector()
	v["tag"] = map[string]any{"kind": "tag", "tag_id": tagA}
	r, e = tagrecQuery(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "100" {
		t.Fatal("multi-tag double counted source rows", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("tags posted")
	}
}

// TestTagRecPrivileges checks exact v2/recovery API exposure and denies helper, base-alias and table access.
func TestTagRecPrivileges(t *testing.T) {
	owner, db := tagrecFixture(t)
	w := provision(t, db, actor)
	for _, role := range []string{"bookkeeping_api", "bookkeeping_sync", "bookkeeping_webhook", "bookkeeping_observer", "bookkeeping_publisher", "bookkeeping_verifier", "bookkeeping_identity_operator"} {
		allowed := []string{}
		if role == "bookkeeping_api" {
			allowed = []string{"get_collection_taxonomy_v2(uuid, uuid)", "query_collection_review_v2(uuid, uuid, text)", "preview_collection_labels_v2(uuid, uuid, text)", "prepare_collection_label_command_v2(uuid, uuid, text)", "pending_collection_label_command_v2(uuid, uuid)", "transition_collection_label_command_v2(uuid, uuid, uuid, text, text)", "collection_recovery_status(uuid)", "clear_collection_recovery(uuid, text)"}
		}
		if value(t, owner, `SELECT count(*)::text FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='banking'AND(p.proname LIKE 'collection_tag_%'OR p.proname LIKE 'collection_recovery_%'OR p.proname IN('get_collection_taxonomy_v2','query_collection_review_v2','preview_collection_labels_v2','prepare_collection_label_command_v2','pending_collection_label_command_v2','transition_collection_label_command_v2','clear_collection_recovery'))AND has_function_privilege($1,p.oid,'EXECUTE')<>((p.proname||'('||oidvectortypes(p.proargtypes)||')')=ANY($2::text[]))`, role, allowed) != "0" {
			t.Fatal("function privileges", role)
		}
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	for _, q := range []string{"SELECT * FROM banking.collection_tag_edition", "SELECT * FROM banking.collection_tag_decision", "SELECT * FROM banking.collection_tag_preview_result", "SELECT * FROM ledger.collection_recovery_state", "SELECT * FROM ledger.collection_recovery_receipt", "SELECT banking.collection_tag_prepare_v1_base(NULL,NULL,'{}')", "SELECT banking.collection_recovery_principal(NULL)", "SELECT banking.collection_label_snapshot(NULL)"} {
		_, e := runtime.Exec(context.Background(), q)
		code(t, e, "42501")
	}
	_, e := call(t, db, "synthetic_mixed", "serializable", "SELECT banking.collection_recovery_status($1)", actor)
	code(t, e, "P2801")
	_, e = call(t, db, "synthetic_mixed", "repeatable read", "SELECT banking.get_collection_taxonomy_v2($1,$2)", actor, w)
	code(t, e, "P2801")
}

// TestTagRecPagination3000 proves complete counts, both stable date orders and exact source filters.
func TestTagRecPagination3000(t *testing.T) {
	owner, db := tagrecFixture(t)
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
	seedTags(t, db, w)
	basePage := tagrecPage(t, db, w)
	began := time.Now()
	preparedPreview, e := tagrecDeadline(t, db, "SELECT banking.preview_collection_labels_v2($1,$2,$3)", actor, w, raw(map[string]any{"snapshot": basePage["snapshot"], "selection": selection(basePage["items"].([]any)), "action": tagAction("add", tagA, tagB)}))
	if e != nil {
		t.Fatal("100-of3000 preview exceeded bounded service deadline", e)
	}
	t.Logf("actual100-of3000 preview under3500ms statement/4s context: %s", time.Since(began))
	began = time.Now()
	prepared, e := tagrecDeadline(t, db, "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)", actor, w, raw(map[string]any{"preview_id": preparedPreview["preview_id"], "prepare_key": "first100-tags"}))
	if e != nil {
		t.Fatal("bounded prepare", e)
	}
	id := tagrecCommand(t, prepared)["command_id"].(string)
	t.Logf("actual100-of3000 prepare under3500ms statement/4s context: %s", time.Since(began))
	began = time.Now()
	if _, e = tagrecDeadline(t, db, "SELECT banking.transition_collection_label_command_v2($1,$2,$3,'execute','{}')", actor, w, id); e != nil {
		t.Fatal("100-of3000 atomic execute exceeded service deadline", e)
	}
	t.Logf("actual100-of3000 atomic execute under3500ms statement/4s context: %s", time.Since(began))
	if _, e = tagrecTransition(t, db, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	filter := tagrecSelector()
	filter["tag"] = map[string]any{"kind": "tag", "tag_id": tagA}
	filtered, err := tagrecQuery(t, db, w, actor, filter)
	if err != nil || filtered["matching_count"] != "100" {
		t.Fatal("tag whole count", err)
	}
	filter["tag"] = map[string]any{"kind": "untagged"}
	filtered, err = tagrecQuery(t, db, w, actor, filter)
	if err != nil || filtered["matching_count"] != "2900" {
		t.Fatal("untagged whole count", err)
	}
	for _, order := range []string{"date_desc", "date_asc"} {
		t.Run(order, func(t *testing.T) {
			began := time.Now()
			v := tagrecSelector()
			v["sort"] = order
			v["source_account_id"] = sources[0]
			seen := map[string]bool{}
			pages := 0
			lastDay := ""
			for {
				r, e := tagrecQuery(t, db, w, actor, v)
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
	v := tagrecSelector()
	v["source_account_id"] = sources[1]
	r, e := tagrecQuery(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "0" || r["start_index"] != "0" {
		t.Fatal("empty account filter", e)
	}
	v = tagrecSelector()
	v["assignment"] = "assigned"
	r, e = tagrecQuery(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "0" {
		t.Fatal("native assignment", e)
	}
	v = tagrecSelector()
	v["search"] = "cafe"
	v["starts_on"] = "2026-10-01"
	v["through_date"] = "2026-10-01"
	r, e = tagrecQuery(t, db, w, actor, v)
	if e != nil || r["matching_count"] != "108" {
		t.Fatal("full date/search filter", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("query posted")
	}
}

// TestTagRecAstralCursor proves a maximal-codepoint selector survives both bounded wire pages.
func TestTagRecAstralCursor(t *testing.T) {
	_, db := tagrecFixture(t)
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
	tagrecApply(t, db, w, "labels", selection(tagrecPage(t, db, w)["items"].([]any)), map[string]any{"kind": "topic", "topic_id": childTopic})
	seedTags(t, db, w)
	tagrecApply(t, db, w, "tag-rows", selection(tagrecPage(t, db, w)["items"].([]any)), tagAction("add", tagA, tagB))
	v := tagrecSelector()
	v["source_account_id"] = sources[0]
	v["tag"] = map[string]any{"kind": "tag", "tag_id": tagA}
	v["assignment"] = "unassigned"
	v["starts_on"] = "2026-01-01"
	v["through_date"] = "2026-12-31"
	v["search"] = search
	v["topic"] = map[string]any{"kind": "topic", "topic_id": childTopic}
	v["sort"] = "date_asc"
	v["limit"] = 1
	first, e := tagrecQuery(t, db, w, actor, v)
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
	second, e := tagrecQuery(t, db, w, actor, v)
	if e != nil {
		t.Fatal(e)
	}
	if second["matching_count"] != "2" || second["start_index"] != "2" || second["next_cursor"] != nil || len(second["items"].([]any)) != 1 || first["items"].([]any)[0].(map[string]any)["observation_id"] == second["items"].([]any)[0].(map[string]any)["observation_id"] {
		t.Fatal("astral second page identity/count")
	}
	t.Logf("100 astral codepoints, account/topic/date/assignment selector, seven pins: cursor %d hexchars; resumed query %d bytes", len(cursor), len([]byte(raw(v))))
}

// tagrecDeadline exercises the actual SQL boundary under the service's transaction time budget.
func tagrecDeadline(t testing.TB, db, query string, args ...any) (map[string]any, error) {
	t.Helper()
	c := connect(t, db, "synthetic_runtime")
	defer c.Close(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	tx, e := c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='3500ms';SET LOCAL lock_timeout='1s'"); e != nil {
		return nil, e
	}
	var b []byte
	if e = tx.QueryRow(ctx, query, args...).Scan(&b); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	var v map[string]any
	e = json.Unmarshal(b, &v)
	return v, e
}

// TestTagRecUnusedPrincipalFence permits unused identity deletion while retaining all protected history.
func TestTagRecUnusedPrincipalFence(t *testing.T) {
	owner, db := tagrecFixture(t)
	_, e := owner.Exec(context.Background(), "DELETE FROM ledger.collection_recovery_state WHERE principal_id=$1", actor)
	code(t, e, "P2802")
	if value(t, owner, "SELECT count(*)::text FROM ledger.collection_recovery_state WHERE principal_id=$1", actor) != "1" {
		t.Fatal("direct deletion lost current fence")
	}
	w := provision(t, db, actor)
	p, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{}, tagCatalog())
	if e != nil {
		t.Fatal(e)
	}
	r, e := tagrecPrepare(t, db, w, "retained", p)
	if e != nil {
		t.Fatal(e)
	}
	id := tagrecCommand(t, r)["command_id"].(string)
	if _, e = tagrecTransition(t, db, w, id, "cancel"); e != nil {
		t.Fatal(e)
	}
	if _, e = tagrecTransition(t, db, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	_, e = owner.Exec(context.Background(), "DELETE FROM ledger.principal WHERE principal_id=$1", actor)
	code(t, e, "23503")
	if value(t, owner, "SELECT count(*)::text FROM ledger.principal WHERE principal_id=$1", actor) != "1" || value(t, owner, "SELECT count(*)::text FROM ledger.collection_recovery_receipt WHERE principal_id=$1", actor) != "1" || value(t, owner, "SELECT count(*)::text FROM ledger.collection_recovery_state WHERE principal_id=$1", actor) != "1" {
		t.Fatal("retained history or its fence was deleted")
	}
	if value(t, owner, `SELECT count(*)::text FROM pg_constraint WHERE contype='f'AND confrelid='ledger.principal'::regclass AND confdeltype='c'AND conrelid<>'ledger.collection_recovery_state'::regclass`) != "0" {
		t.Fatal("cascade broadened into retained principal history")
	}
	unused := "00000000-0000-4000-8000-000000000090"
	exec(t, owner, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic','unused-principal')`, unused)
	reader := connect(t, db, "synthetic_runtime")
	defer reader.Close(context.Background())
	readTx, e := reader.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer readTx.Rollback(context.Background())
	var count int
	if e = readTx.QueryRow(context.Background(), "SELECT count(*)FROM pg_catalog.pg_class").Scan(&count); e != nil {
		t.Fatal(e)
	}
	writer := connect(t, db, "postgres")
	defer writer.Close(context.Background())
	tx, e := writer.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(context.Background(), "DELETE FROM ledger.principal WHERE principal_id=$1", unused); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		var b []byte
		err := readTx.QueryRow(context.Background(), "SELECT banking.collection_recovery_status($1)", unused).Scan(&b)
		_ = readTx.Rollback(context.Background())
		done <- err
	}()
	waitBlocked(t, owner, reader.PgConn().PID())
	if e = tx.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	code(t, <-done, "40001")
	if value(t, owner, "SELECT count(*)::text FROM ledger.principal WHERE principal_id=$1", unused) != "0" || value(t, owner, "SELECT count(*)::text FROM ledger.collection_recovery_state WHERE principal_id=$1", unused) != "0" {
		t.Fatal("unused parent/fence deletion blocked")
	}
	_, e = recoveryStatus(t, db, unused)
	code(t, e, "P2801")
}
