//go:build integration

package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// TestRule46ManualAndHistorical protects independent manual fields and historical preview versions.
func TestRule46ManualAndHistorical(t *testing.T) {
	owner, pool, db := rule46Fixture(t)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "manual-rules", 1)
	changes := []map[string]any{}
	for _, id := range []string{"topic-only", "tags-only", "both", "neither"} {
		changes = append(changes, change(id, "1.00", "USD"))
	}
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", changes); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	seedTags(t, db, w)
	rows := tagrecPage(t, db, w)["items"].([]any)
	chosen := selection(rows)
	tagrecApply(t, db, w, "manual-topic", []map[string]any{chosen[0]}, map[string]any{"kind": "topic", "topic_id": rootTopic})
	tagrecApply(t, db, w, "manual-tags", []map[string]any{chosen[1]}, tagAction("replace", tagB))
	tagrecApply(t, db, w, "manual-both-topic", []map[string]any{chosen[2]}, map[string]any{"kind": "topic", "topic_id": rootTopic})
	tagrecApply(t, db, w, "manual-both-tags", []map[string]any{chosen[2]}, tagAction("replace", tagB))
	before := rule46Hash(t, owner)
	rule46Catalog(t, s, w, "rule-edition", rule46Rule("00000000-0000-4000-8000-000000006020", "Cafe", "Fictional Cafe", childTopic, tagA))
	p, e := rule46Preview(t, s, w, chosen, map[string]any{"kind": "rules_apply"})
	if e != nil {
		t.Fatal(e)
	}
	if p["apply_count"] != "3" || p["abstain_count"] != "1" {
		t.Fatal("manual fields changed eligibility", raw(p["rule_results"]))
	}
	preserved := map[string]map[string]bool{}
	for _, x := range p["rule_results"].([]any) {
		v := x.(map[string]any)
		m := v["preserved"].(map[string]any)
		preserved[v["observation_id"].(string)] = map[string]bool{"topic": m["topic"].(bool), "tags": m["tags"].(bool)}
	}
	if !preserved[chosen[0]["observation_id"].(string)]["topic"] || !preserved[chosen[1]["observation_id"].(string)]["tags"] {
		t.Fatal("manual protection disappeared")
	}
	rule46Complete(t, s, w, "apply-fieldwise", p)
	current := rule46Query(t, s, w)
	byID := map[string]map[string]any{}
	for _, x := range current["items"].([]any) {
		r := x.(map[string]any)
		byID[r["observation_id"].(string)] = r
	}
	if byID[chosen[0]["observation_id"].(string)]["owner_topic"].(map[string]any)["topic_id"] != rootTopic {
		t.Fatal("manual topic overwritten")
	}
	tags := byID[chosen[1]["observation_id"].(string)]["owner_tags"].(map[string]any)["tags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["tag_id"] != tagB {
		t.Fatal("manual tags overwritten")
	}
	// New v3 manual preview uses the current nonzero rule pins and effective before-values.
	manual, e := rule46Preview(t, s, w, []map[string]any{chosen[3]}, map[string]any{"kind": "topic", "topic_id": rootTopic})
	if e != nil {
		t.Fatal("nonzero v3 manual preview", e)
	}
	oldrow := manual["review_rows"].([]any)[0].(map[string]any)
	if oldrow["owner_topic"].(map[string]any)["topic_id"] != childTopic {
		t.Fatal("manual review hid applied rule before-value")
	}
	rule46Complete(t, s, w, "manual-after-rule", manual)
	// A newly saved historical v2 preview remains byte-shape identical inside v3 recovery.
	legacy, e := tagrecPreview(t, db, w, tagrecPage(t, db, w)["snapshot"], []map[string]any{chosen[3]}, tagAction("replace", tagB))
	if e != nil {
		t.Fatal(e)
	}
	saved, e := tagrecPrepare(t, db, w, "historical-v2", legacy)
	if e != nil {
		t.Fatal(e)
	}
	id := tagrecCommand(t, saved)["command_id"].(string)
	b, e := s.Pending(context.Background(), Request{Actor: actor, Workspace: w, Version: 3})
	pending := rule46Object(t, b, e)
	if raw(pending["command"].(map[string]any)["preview"]) != raw(legacy) {
		t.Fatal("historical v2 preview changed in v3 wrapper")
	}
	if _, e = rule46Transition(t, s, w, id, "cancel"); e != nil {
		t.Fatal(e)
	}
	if _, e = rule46Transition(t, s, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
	if rule46Hash(t, owner) != before {
		t.Fatal("manual/rule operations changed source or journals")
	}
}

// TestRule46UnknownReplyAndRevocation resolves a real lost commit with immutable same-command replay.
func TestRule46UnknownReplyAndRevocation(t *testing.T) {
	owner, pool, db := rule46Fixture(t)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "lost-rule", 1)
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("cafe", "1.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	seedTags(t, db, w)
	rule46Catalog(t, s, w, "lost-catalog", rule46Rule("00000000-0000-4000-8000-000000006030", "Cafe", "Fictional Cafe", childTopic))
	p, e := rule46Preview(t, s, w, selection(rule46Query(t, s, w)["items"].([]any)), map[string]any{"kind": "rules_apply"})
	if e != nil {
		t.Fatal(e)
	}
	c, e := rule46Prepare(t, s, w, "lost-prepare", p)
	if e != nil {
		t.Fatal(e)
	}
	id := c["command"].(map[string]any)["command_id"].(string)
	lost := &lostRecoveryCommit{pool: pool}
	lost.lose.Store(true)
	faulty := &Service{DB: lost, Rules: RuleProcessor{}}
	_, e = rule46Transition(t, faulty, w, id, "execute")
	rule46Failure(t, e, 503, "temporarily_unavailable")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_decision") != "1" {
		t.Fatal("lost actual commit did not persist exactly one decision")
	}
	b, e := s.Pending(context.Background(), Request{Actor: actor, Workspace: w, Version: 3})
	recovered := rule46Object(t, b, e)
	if recovered["command"].(map[string]any)["state"] != "committed" {
		t.Fatal("lost reply not recovered")
	}
	replay, e := rule46Transition(t, s, w, id, "execute")
	if e != nil {
		t.Fatal(e)
	}
	if raw(replay["command"]) != raw(recovered["command"]) {
		t.Fatal("terminal receipt changed")
	}
	_, e = call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'rule-revoke',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{}, "evidence_ref": evidence}))
	if e != nil {
		t.Fatal(e)
	}
	_, e = rule46Transition(t, s, w, id, "execute")
	rule46Failure(t, e, 404, "collection_denied")
	status, e := recoveryStatus(t, db, actor)
	if e != nil || status["state"] != "inaccessible" {
		t.Fatal("inaccessible rule recovery", e)
	}
	if _, e = recoveryClear(t, db, actor, status["recovery_id"]); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_decision") != "1" {
		t.Fatal("clear changed accepted effects")
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_command WHERE state='committed' AND acknowledged_at IS NOT NULL") != "4" { // topic catalog, tag catalog, rule catalog, rule apply
		t.Fatal("clear failed to preserve terminal history")
	}
}

// TestRule46LegacyVersionFences rejects v3 work before legacy code can reserve or render it.
func TestRule46LegacyVersionFences(t *testing.T) {
	owner, pool, db := rule46Fixture(t)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	w := provision(t, db, actor)
	seedTopics(t, db, w)
	p, e := rule46Preview(t, s, w, []map[string]any{}, map[string]any{"kind": "rule_catalog", "rules": []map[string]any{rule46Rule("00000000-0000-4000-8000-000000006040", "Cafe", "Fictional Cafe", childTopic)}})
	if e != nil {
		t.Fatal(e)
	}
	if p["snapshot"].(map[string]any)["ruleset_revision"] != "0" {
		t.Fatal("fixture did not target initial-zero legacy guard")
	}
	audit := value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action")
	_, e = prepare(t, db, w, "legacy-must-deny", p)
	code(t, e, "P2804")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_command") != "1" || value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation") != "0" || value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action") != audit {
		t.Fatal("rejected legacy prepare changed command/reservation/audit")
	}
	c, e := rule46Prepare(t, s, w, "correct-v3", p)
	if e != nil {
		t.Fatal(e)
	}
	id := c["command"].(map[string]any)["command_id"].(string)
	for _, phase := range []string{"prepared", "committed"} {
		if phase == "committed" {
			if _, e = rule46Transition(t, s, w, id, "execute"); e != nil {
				t.Fatal(e)
			}
		}
		before := value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action")
		_, e = call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.pending_collection_label_command($1,$2)", actor, w)
		code(t, e, "P2804")
		if value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action") != before || value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation") != "1" {
			t.Fatal("rejected legacy pending changed durable state")
		}
	}
	if _, e = rule46Transition(t, s, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
}

// TestRule46StalePins never promotes an old preview after any classification/source edition changes.
func TestRule46StalePins(t *testing.T) {
	for _, mode := range []string{"source_execute", "taxonomy", "tags", "rules", "manual"} {
		t.Run(mode, func(t *testing.T) {
			owner, pool, db := rule46Fixture(t)
			s := &Service{DB: pool, Rules: RuleProcessor{}}
			w := provision(t, db, actor)
			_, sources := register(t, db, w, actor, "stale-"+mode, 1)
			_, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("cafe-a", "1.00", "USD"), change("cafe-b", "2.00", "USD")})
			if e != nil {
				t.Fatal(e)
			}
			seedTopics(t, db, w)
			seedTags(t, db, w)
			rule := rule46Rule("00000000-0000-4000-8000-000000006050", "Cafe", "Fictional Cafe", childTopic)
			rule46Catalog(t, s, w, "rule-catalog", rule)
			chosen := selection(rule46Query(t, s, w)["items"].([]any))
			p, e := rule46Preview(t, s, w, chosen, map[string]any{"kind": "rules_apply"})
			if e != nil {
				t.Fatal(e)
			}
			var id string
			if mode == "source_execute" {
				c, e := rule46Prepare(t, s, w, "old-intent", p)
				if e != nil {
					t.Fatal(e)
				}
				id = c["command"].(map[string]any)["command_id"].(string)
			}
			switch mode {
			case "source_execute":
				corrected := change("cafe-b", "2.01", "USD")
				corrected["operation"] = "modified"
				if _, e = publish(t, db, w, sources[0], "correction", "1", "c1", "c2", []map[string]any{corrected}); e != nil {
					t.Fatal(e)
				}
			case "taxonomy":
				tagrecApply(t, db, w, "topic-edit", []map[string]any{}, topicsAction([]map[string]any{topic(rootTopic, "Everyday renamed", nil, "active"), topic(childTopic, "Dining", rootTopic, "active")}))
			case "tags":
				tagrecApply(t, db, w, "tag-edit", []map[string]any{}, tagCatalog(tag(tagA, "Travel renamed", "active"), tag(tagB, "Reimburse", "active")))
			case "rules":
				rule["name"] = "Cafe renamed"
				rule46Catalog(t, s, w, "rule-edit", rule)
			case "manual":
				tagrecApply(t, db, w, "manual-edit", []map[string]any{chosen[0]}, map[string]any{"kind": "topic", "topic_id": rootTopic})
			}
			before := rule46Hash(t, owner)
			if mode == "source_execute" {
				_, e = rule46Transition(t, s, w, id, "execute")
				rule46Failure(t, e, 409, "collection_snapshot_changed")
				if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_command WHERE state='prepared'") != "1" {
					t.Fatal("stale execute changed command")
				}
				if _, e = rule46Transition(t, s, w, id, "cancel"); e != nil {
					t.Fatal("stale cancel", e)
				}
				if _, e = rule46Transition(t, s, w, id, "acknowledge"); e != nil {
					t.Fatal("stale acknowledgement", e)
				}
			} else {
				_, e = rule46Prepare(t, s, w, "stale-prepare", p)
				rule46Failure(t, e, 409, "collection_snapshot_changed")
			}
			if value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_decision") != "0" || rule46Hash(t, owner) != before {
				t.Fatal("stale intent had partial effects")
			}
		})
	}
}

// TestRule46AtomicFailure forces a later insert failure and proves the whole reviewed batch rolls back.
func TestRule46AtomicFailure(t *testing.T) {
	owner, pool, db := rule46Fixture(t)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "atomic-rules", 1)
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("cafe-a", "1.00", "USD"), change("cafe-b", "2.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	seedTags(t, db, w)
	rule46Catalog(t, s, w, "rules", rule46Rule("00000000-0000-4000-8000-000000006060", "Cafe", "Fictional Cafe", childTopic))
	chosen := selection(rule46Query(t, s, w)["items"].([]any))
	p, e := rule46Preview(t, s, w, chosen, map[string]any{"kind": "rules_apply"})
	if e != nil {
		t.Fatal(e)
	}
	c, e := rule46Prepare(t, s, w, "atomic-prepare", p)
	if e != nil {
		t.Fatal(e)
	}
	id := c["command"].(map[string]any)["command_id"].(string)
	before := rule46Hash(t, owner)
	revision := value(t, owner, "SELECT decision_revision::text FROM banking.collection_label_state WHERE workspace_id=$1", w)
	// This test-only constraint affects only this disposable clone; production SQL is unchanged.
	_, e = owner.Exec(context.Background(), "ALTER TABLE banking.collection_rule_decision ADD CONSTRAINT synthetic_later_row_failure CHECK(observation_id<>"+"'"+chosen[1]["observation_id"].(string)+"'::uuid) NOT VALID")
	if e != nil {
		t.Fatal(e)
	}
	_, e = rule46Transition(t, s, w, id, "execute")
	rule46Failure(t, e, 503, "temporarily_unavailable")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_decision") != "0" || value(t, owner, "SELECT decision_revision::text FROM banking.collection_label_state WHERE workspace_id=$1", w) != revision || value(t, owner, "SELECT state FROM banking.collection_label_command WHERE command_id=$1", id) != "prepared" {
		t.Fatal("failed batch partially committed")
	}
	exec(t, owner, "ALTER TABLE banking.collection_rule_decision DROP CONSTRAINT synthetic_later_row_failure")
	if _, e = rule46Transition(t, s, w, id, "execute"); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_decision") != "2" || rule46Hash(t, owner) != before {
		t.Fatal("same reviewed retry did not commit exactly once")
	}
	if _, e = rule46Transition(t, s, w, id, "acknowledge"); e != nil {
		t.Fatal(e)
	}
}

// TestRule46RetainedRollback refuses to erase even an unprepared immutable v3 review.
func TestRule46RetainedRollback(t *testing.T) {
	owner, pool, db := rule46Fixture(t)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	w := provision(t, db, actor)
	seedTopics(t, db, w)
	if _, e := rule46Preview(t, s, w, []map[string]any{}, map[string]any{"kind": "rule_catalog", "rules": []map[string]any{rule46Rule("00000000-0000-4000-8000-000000006070", "Cafe", "Fictional Cafe", childTopic)}}); e != nil {
		t.Fatal(e)
	}
	down, e := os.ReadFile("/proof/migrations/000046_collection_rules.down.sql")
	if e != nil {
		t.Fatal("synthetic fixture migration source unavailable")
	}
	ddl := strings.Replace(string(down), "BEGIN;", "", 1)
	ddl = strings.TrimSuffix(strings.TrimSpace(ddl), "COMMIT;")
	tx, e := owner.Begin(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(context.Background(), ddl)
	code(t, e, "P2802")
	_ = tx.Rollback(context.Background())
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_preview WHERE wire_version=3") != "1" || value(t, owner, "SELECT count(*)::text FROM banking.collection_rule_edition") != "1" {
		t.Fatal("refused downgrade lost retained review/history")
	}
}
