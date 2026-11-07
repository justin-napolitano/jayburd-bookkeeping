//go:build integration

package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// rule46Fixture clones only the empty synthetic46 template and opens the actual restricted API login.
func rule46Fixture(t *testing.T) (*pgx.Conn, *pgxpool.Pool, string) {
	t.Helper()
	db := fmt.Sprintf("rule46_%d", time.Now().UnixNano())
	admin := connect(t, "postgres", "postgres")
	exec(t, admin, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE rules_template")
	_ = admin.Close(context.Background())
	owner := connect(t, db, "postgres")
	query45Cleanup(t, owner, db)
	exec(t, owner, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic-rules','owner'),($2,'synthetic-rules','other')`, actor, other)
	exec(t, owner, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic-rules')`, currency)
	pool, e := pgxpool.New(context.Background(), "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=3")
	if e != nil {
		t.Fatal("synthetic restricted pool unavailable")
	}
	t.Cleanup(pool.Close)
	return owner, pool, db
}
func rule46Object(t testing.TB, b json.RawMessage, e error) map[string]any {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		t.Fatal("synthetic response shape")
	}
	return v
}
func rule46Query(t testing.TB, s *Service, w string) map[string]any {
	t.Helper()
	b, e := s.Query(context.Background(), Request{Actor: actor, Workspace: w, Version: 3, Body: raw(query45Selector())})
	return rule46Object(t, b, e)
}
func rule46Preview(t testing.TB, s *Service, w string, selected []map[string]any, action any) (map[string]any, error) {
	t.Helper()
	p := rule46Query(t, s, w)
	b, e := s.Preview(context.Background(), Request{Actor: actor, Workspace: w, Version: 3, Body: raw(map[string]any{"snapshot": p["snapshot"], "selection": selected, "action": action})})
	if e != nil {
		return nil, e
	}
	return rule46Object(t, b, nil), nil
}
func rule46Prepare(t testing.TB, s *Service, w, key string, p map[string]any) (map[string]any, error) {
	t.Helper()
	b, e := s.Prepare(context.Background(), Request{Actor: actor, Workspace: w, Version: 3, Body: raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": key})})
	if e != nil {
		return nil, e
	}
	return rule46Object(t, b, nil), nil
}
func rule46Transition(t testing.TB, s *Service, w, id, operation string) (map[string]any, error) {
	t.Helper()
	b, e := s.Transition(context.Background(), Request{Actor: actor, Workspace: w, Command: id, Version: 3, Body: "{}"}, operation)
	if e != nil {
		return nil, e
	}
	return rule46Object(t, b, nil), nil
}
func rule46Complete(t testing.TB, s *Service, w, key string, p map[string]any) map[string]any {
	t.Helper()
	c, e := rule46Prepare(t, s, w, key, p)
	if e != nil {
		t.Fatal("prepare phase:", e)
	}
	id := c["command"].(map[string]any)["command_id"].(string)
	c, e = rule46Transition(t, s, w, id, "execute")
	if e != nil {
		t.Fatal("execute phase:", e)
	}
	if c["command"].(map[string]any)["state"] != "committed" {
		t.Fatal("rule command not committed")
	}
	if _, e = rule46Transition(t, s, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	return c
}
func rule46Rule(id, name, merchant, topicID string, tags ...string) map[string]any {
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{"rule_id": id, "name": name, "status": "active", "source_account_id": nil, "priority": 0, "conditions": []map[string]any{{"field": "merchant", "operator": "exact", "value": merchant}}, "output": map[string]any{"kind": "topic", "topic_id": topicID, "tag_ids": tags}}
}
func rule46Catalog(t testing.TB, s *Service, w, key string, rules ...map[string]any) map[string]any {
	t.Helper()
	if rules == nil {
		rules = []map[string]any{}
	}
	p, e := rule46Preview(t, s, w, []map[string]any{}, map[string]any{"kind": "rule_catalog", "rules": rules})
	if e != nil {
		t.Fatal("catalog preview phase:", e)
	}
	return rule46Complete(t, s, w, key, p)
}
func rule46Failure(t testing.TB, e error, status int, code string) {
	t.Helper()
	var f *ledger.Failure
	if !errors.As(e, &f) || f.Status != status || f.Code != code {
		t.Fatalf("wanted fixed %d %s, got %v", status, code, e)
	}
}
func rule46Hash(t testing.TB, c *pgx.Conn) string {
	t.Helper()
	return value(t, c, `SELECT encode(sha256(convert_to(jsonb_build_array((SELECT coalesce(jsonb_agg(to_jsonb(x)ORDER BY workspace_id,source_account_id),'[]')FROM banking.collection_source x),(SELECT coalesce(jsonb_agg(to_jsonb(x)ORDER BY workspace_id,source_account_id,observation_id,revision),'[]')FROM banking.collection_revision x),(SELECT coalesce(jsonb_agg(to_jsonb(x)ORDER BY journal_id),'[]')FROM ledger.journal_entry x),(SELECT coalesce(jsonb_agg(to_jsonb(x)ORDER BY journal_id,line_number),'[]')FROM ledger.journal_line x),(SELECT count(*)FROM integration.outbox_message))::text,'UTF8')),'hex')`)
}

// TestRule46Lifecycle proves real service matching is inert until exact reviewed atomic application.
func TestRule46Lifecycle(t *testing.T) {
	owner, pool, db := rule46Fixture(t)

	s := &Service{DB: pool, Rules: RuleProcessor{}}
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "rule-lifecycle", 1)
	changes := []map[string]any{change("cafe-a", "9007199254740993.01", "USD"), change("cafe-b", "0.10", "USD"), change("market", "3.00", "USD"), change("missing", "4.00", "USD")}
	changes[2]["merchant_name"] = "Fictional Mega"
	changes[3]["merchant_name"] = nil
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", changes); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	seedTags(t, db, w)
	before := rule46Hash(t, owner)
	rule := rule46Rule("00000000-0000-4000-8000-000000006001", "Fictional cafe rule", "Fictional Cafe", childTopic, tagA)
	rule46Catalog(t, s, w, "save-rules", rule)
	p := rule46Query(t, s, w)
	if p["snapshot"].(map[string]any)["ruleset_revision"] != "1" {
		t.Fatal("runtime ruleset remains fabricated zero")
	}
	matched := 0
	for _, x := range p["items"].([]any) {
		r := x.(map[string]any)
		if r["owner_topic"].(map[string]any)["state"] != "unclassified" {
			t.Fatal("read applied suggestion")
		}
		suggestion := r["rule_suggestion"].(map[string]any)
		if suggestion["will_apply"] != false {
			t.Fatal("read promised mutation")
		}
		if suggestion["status"] == "rule" {
			matched++
		}
	}
	if matched != 2 {
		t.Fatal("actual matcher did not identify two Cafe facts")
	}
	selected := selection(p["items"].([]any))
	preview, e := rule46Preview(t, s, w, selected, map[string]any{"kind": "rules_apply"})
	if e != nil {
		t.Fatal(e)
	}
	if preview["apply_count"] != "2" || preview["abstain_count"] != "2" {
		t.Fatal("authoritative eligibility counts", raw(preview["rule_results"]))
	}
	for _, x := range preview["review_rows"].([]any) {
		r := x.(map[string]any)
		if _, ok := r["institution"]; ok {
			t.Fatal("saved preview row coerced to new query shape")
		}
	}
	c, e := rule46Prepare(t, s, w, "apply-rules", preview)
	if e != nil {
		t.Fatal(e)
	}
	id := c["command"].(map[string]any)["command_id"].(string)
	pending, e := s.Pending(context.Background(), Request{Actor: actor, Workspace: w, Version: 3})
	p3 := rule46Object(t, pending, e)
	if raw(p3["command"].(map[string]any)["preview"]) != raw(preview) {
		t.Fatal("immutable preview changed in recovery")
	}
	_, e = call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.pending_collection_label_command_v2($1,$2)", actor, w)
	code(t, e, "P2804")
	if _, e = rule46Transition(t, s, w, id, "execute"); e != nil {
		t.Fatal(e)
	}
	if _, e = rule46Transition(t, s, w, id, "execute"); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_decision") != "2" {
		t.Fatal("duplicate or expanded rule effects")
	}
	if _, e = rule46Transition(t, s, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	p = rule46Query(t, s, w)
	matched = 0
	for _, x := range p["items"].([]any) {
		r := x.(map[string]any)
		if r["owner_topic"].(map[string]any)["state"] == "topic" {
			matched++
			if r["rule_application"].(map[string]any)["state"] != "current" {
				t.Fatal("applied provenance missing")
			}
		}
	}
	if matched != 2 {
		t.Fatal("persisted rule labels absent")
	}
	v := query45Selector()
	v["topic"] = map[string]any{"kind": "topic", "topic_id": childTopic}
	b, e := s.Query(context.Background(), Request{Actor: actor, Workspace: w, Version: 3, Body: raw(v)})
	if rule46Object(t, b, e)["matching_count"] != "2" {
		t.Fatal("filters ignored persisted effective rule labels")
	}
	noop, e := rule46Preview(t, s, w, selection(p["items"].([]any)), map[string]any{"kind": "rules_apply"})
	if e != nil {
		t.Fatal(e)
	}
	if noop["apply_count"] != "0" || noop["abstain_count"] != "4" {
		t.Fatal("same edition application was not idempotent")
	}
	rule46Complete(t, s, w, "no-op", noop)
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_decision") != "2" || rule46Hash(t, owner) != before {
		t.Fatal("rule classification changed financial originals/journals or duplicated decisions")
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != "0" || value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision") != "0" {
		t.Fatal("rules masqueraded as manual decisions")
	}
}

// TestRule46Validation keeps invalid new regex/budgets definite while stored retired outputs remain inert.
func TestRule46Validation(t *testing.T) {
	owner, pool, db := rule46Fixture(t)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "rule-validation", 1)
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("cafe", "1.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	seedTags(t, db, w)
	for _, pattern := range []string{"[", "((a{1000}){1000})"} {
		r := rule46Rule("00000000-0000-4000-8000-000000006010", "Invalid candidate", "Fictional Cafe", childTopic)
		r["conditions"] = []map[string]any{{"field": "merchant", "operator": "regex", "value": pattern}}
		_, e := rule46Preview(t, s, w, []map[string]any{}, map[string]any{"kind": "rule_catalog", "rules": []map[string]any{r}})
		rule46Failure(t, e, 400, "invalid_collection")
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_preview WHERE wire_version=3") != "0" {
		t.Fatal("invalid regex persisted preview")
	}
	large := []map[string]any{}
	for i := 0; i < 600; i++ {
		r := rule46Rule(fmt.Sprintf("00000000-0000-4000-8000-%012d", 7000+i), strings.Repeat("界", 80), strings.Repeat("x", 128), childTopic)
		r["status"] = "retired"
		large = append(large, r)
	}
	_, e := rule46Preview(t, s, w, []map[string]any{}, map[string]any{"kind": "rule_catalog", "rules": large})
	rule46Failure(t, e, 422, "collection_limit")
	rule46Catalog(t, s, w, "valid-catalog", rule46Rule("00000000-0000-4000-8000-000000006011", "Known rule", "Fictional Cafe", childTopic))
	tagrecApply(t, db, w, "retire-output", []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Everyday", nil, "active"), topic(childTopic, "Dining", rootTopic, "retired")}))
	p := rule46Query(t, s, w)
	if p["matching_count"] != "1" || p["items"].([]any)[0].(map[string]any)["rule_suggestion"].(map[string]any)["status"] != "unavailable" {
		t.Fatal("invalid stored output hid basic row or silently fell through")
	}
}
