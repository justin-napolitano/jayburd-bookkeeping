//go:build integration

package collectionlabels

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
	"github.com/jackc/pgx/v5/pgxpool"
)

var automaticCategorizationProof = flag.Bool("automatic-categorization-proof", false, "use only the disposable schema48 automatic proof template")

// auto48Fixture never borrows the legacy label runner's schema43/46 resources.
func auto48Fixture(t *testing.T) (*pgx.Conn, *pgxpool.Pool, string) {
	t.Helper()
	if !*automaticCategorizationProof {
		t.Skip("use scripts/test-automatic-categorization.sh")
	}
	return rule46Fixture(t)
}

// auto48Loss reports an ambiguous commit only after PostgreSQL has durably accepted the phase.
type auto48Loss struct {
	pool  *pgxpool.Pool
	phase string
	fired bool
}
type auto48LossTx struct {
	pgx.Tx
	parent   *auto48Loss
	selected bool
}

func (p *auto48Loss) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, e := p.pool.BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	return &auto48LossTx{Tx: tx, parent: p}, nil
}
func (tx *auto48LossTx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	if strings.Contains(q, tx.parent.phase) {
		tx.selected = true
	}
	return tx.Tx.QueryRow(ctx, q, args...)
}
func (tx *auto48LossTx) Commit(ctx context.Context) error {
	e := tx.Tx.Commit(ctx)
	if e == nil && tx.selected && !tx.parent.fired {
		tx.parent.fired = true
		return errors.New("synthetic lost committed reply")
	}
	return e
}

// TestAuto48UnknownCommit proves recovery against durable SQL, rather than a simulated empty queue.
func TestAuto48UnknownCommit(t *testing.T) {
	for _, phase := range []string{"prepare_collection_automatic_run", "execute_collection_automatic_run"} {
		t.Run(phase, func(t *testing.T) {
			owner, pool, db := auto48Fixture(t)
			w := provision(t, db, actor)
			_, sources := register(t, db, w, actor, "unknown48", 1)
			if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("one", "0.10", "USD")}); e != nil {
				t.Fatal(e)
			}
			loss := &auto48Loss{pool: pool, phase: phase}
			worker := &AutomaticService{DB: loss}
			if phase == "execute_collection_automatic_run" {
				if e := worker.Step(context.Background()); e != nil {
					t.Fatal(e)
				}
			}
			if e := worker.Step(context.Background()); e == nil || !loss.fired {
				t.Fatal("lost reply not exercised")
			}
			id := value(t, owner, "SELECT run_id::text FROM banking.collection_automatic_run WHERE workspace_id=$1", w)
			if e := worker.Step(context.Background()); e != nil {
				t.Fatal("same-run recovery", e)
			}
			if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "1" || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "1" || value(t, owner, "SELECT last_run_id::text FROM banking.collection_automation_state WHERE workspace_id=$1", w) != id {
				t.Fatal("unknown reply created replacement effects")
			}
			if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command") != "0" || value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
				t.Fatal("automatic work fabricated interactive/accounting work")
			}
		})
	}
}

// auto48ProjectionEqual guards semantic drift between whole-history and selected-row SQL projections.
func auto48ProjectionEqual(t testing.TB, owner *pgx.Conn, w string) {
	t.Helper()
	if value(t, owner, `WITH complete AS MATERIALIZED(
 SELECT source_id,observation_id,observation_revision,body FROM banking.collection_query_rows($1::uuid,$2::uuid)
), selected AS MATERIALIZED(
 SELECT r.source_id,r.observation_id,r.observation_revision,
 banking.collection_automatic_project_row($1::uuid,$2::uuid,r.source_id,r.observation_id,r.observation_revision,r.body)AS body
 FROM banking.collection_tag_rows($1::uuid,$2::uuid)r
)
SELECT count(*)::text FROM complete c FULL JOIN selected s USING(source_id,observation_id,observation_revision)
WHERE c.source_id IS NULL OR s.source_id IS NULL OR c.body IS DISTINCT FROM s.body`, actor, w) != "0" {
		t.Fatal("selected-row and complete SQL projections disagree")
	}
}

// TestAuto48WholeResult proves bounded batches finish a larger history and filters/counts use saved output.
func TestAuto48WholeResult(t *testing.T) {
	owner, pool, db := auto48Fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "history48", 1)
	for batch := 0; batch < 3; batch++ {
		changes := []map[string]any{}
		for n := 0; n < 1000; n++ {
			c := change(fmt.Sprintf("row-%04d", batch*1000+n), "0.10", "USD")
			if n%2 == 1 {
				c["merchant_name"] = "Fictional unknown"
			}
			changes = append(changes, c)
		}
		var start any
		if batch > 0 {
			start = fmt.Sprintf("c%d", batch)
		}
		if _, e := publish(t, db, w, sources[0], fmt.Sprintf("batch%d", batch), fmt.Sprint(batch), start, fmt.Sprintf("c%d", batch+1), changes); e != nil {
			t.Fatal(e)
		}
	}
	seedTopics(t, db, w)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	rule46Catalog(t, s, w, "history-rule", rule46Rule("00000000-0000-4000-8000-000000006001", "Fictional cafe rule", "Fictional Cafe", childTopic))
	before := rule46Hash(t, owner)
	worker := &AutomaticService{DB: pool}
	var longestPhase time.Duration
	for i := 0; i < 60; i++ {
		started := time.Now()
		if e := worker.Step(context.Background()); e != nil {
			t.Fatalf("bounded phase%d after %s: %v", i, time.Since(started), e)
		}
		if elapsed := time.Since(started); elapsed > longestPhase {
			longestPhase = elapsed
		}
	}
	t.Logf("3000 facts / 30 runs; longest bounded workspace phase %s", longestPhase)
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "3000" || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "30" {
		t.Fatal("bounded backfill did not finish exactly")
	}
	auto48ProjectionEqual(t, owner, w)
	q := query45Selector()
	q["topic"] = map[string]any{"kind": "topic", "topic_id": childTopic}
	p := query45Page(t, db, w, q)
	if p["matching_count"] != "1500" || len(p["items"].([]any)) != 100 {
		t.Fatal("category filter/count is page-local")
	}
	if _, e := s.Query(context.Background(), Request{Actor: actor, Workspace: w, Version: 3, Body: raw(q)}); e != nil {
		t.Fatal("bounded service query", e)
	}
	q = query45Selector()
	q["topic"] = map[string]any{"kind": "needs_attention"}
	q["status"] = map[string]any{"source": "booked", "journal": "all"}
	p = query45Page(t, db, w, q)
	if p["matching_count"] != "1500" {
		t.Fatal("exceptions count is page-local")
	}
	status, e := (&AutomationService{DB: pool}).GetAutomation(context.Background(), Request{Actor: actor, Workspace: w})
	v := rule46Object(t, status, e)
	if v["pending_count"] != "0" || v["attention_count"] != "1500" {
		t.Fatal("whole-result status wrong")
	}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "30" || before != rule46Hash(t, owner) {
		t.Fatal("unchanged history repeated or source/accounting mutated")
	}
}

// TestAuto48SourceFenceAndSharedWork prevents stale effects and leaves human work untouched.
func TestAuto48SourceFenceAndSharedWork(t *testing.T) {
	owner, pool, db := auto48Fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "fence48", 1)
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("one", "1.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	worker := &AutomaticService{DB: pool}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	run := value(t, owner, "SELECT run_id::text FROM banking.collection_automatic_run WHERE workspace_id=$1", w)
	modified := change("one", "2.00", "USD")
	modified["operation"] = "modified"
	if _, e := publish(t, db, w, sources[0], "modified", "1", "c1", "c2", []map[string]any{modified}); e != nil {
		t.Fatal(e)
	}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT state FROM banking.collection_automatic_run WHERE run_id=$1", run) != "stale" || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "0" {
		t.Fatal("stale source plan applied")
	}
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	p, e := rule46Preview(t, s, w, selection(rule46Query(t, s, w)["items"].([]any)), map[string]any{"kind": "uncategorized"})
	if e != nil {
		t.Fatal(e)
	}
	c, e := rule46Prepare(t, s, w, "held-human", p)
	if e != nil {
		t.Fatal(e)
	}
	human := c["command"].(map[string]any)["command_id"].(string)
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT state FROM banking.collection_label_command WHERE command_id=$1", human) != "prepared" || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "1" {
		t.Fatal("automation touched saved human work")
	}
	status, e := (&AutomationService{DB: pool}).GetAutomation(context.Background(), Request{Actor: actor, Workspace: w})
	v := rule46Object(t, status, e)
	if v["processing_state"] != "blocked" || v["blocked_reason"] != "saved_work" {
		t.Fatal("pending human work was hidden")
	}
}

// TestAuto48AtomicFailure proves a later-row rejection rolls back the entire batch and retains its exact plan.
func TestAuto48AtomicFailure(t *testing.T) {
	owner, pool, db := auto48Fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "atomic48", 1)
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("one", "1.00", "USD"), change("two", "2.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	worker := &AutomaticService{DB: pool}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	run := value(t, owner, "SELECT run_id::text FROM banking.collection_automatic_run WHERE workspace_id=$1", w)
	rejected := value(t, owner, "SELECT context->'selection'->1->>'observation_id' FROM banking.collection_automatic_run WHERE run_id=$1", run)
	exec(t, owner, "ALTER TABLE banking.collection_automatic_decision ADD CONSTRAINT synthetic_later_row_failure CHECK(observation_id<> '"+rejected+"'::uuid) NOT VALID")
	before := value(t, owner, "SELECT projection_revision::text FROM banking.collection_label_state WHERE workspace_id=$1", w)
	if e := worker.Step(context.Background()); e == nil {
		t.Fatal("later row failure not exercised")
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "0" || value(t, owner, "SELECT state FROM banking.collection_automatic_run WHERE run_id=$1", run) != "prepared" || value(t, owner, "SELECT projection_revision::text FROM banking.collection_label_state WHERE workspace_id=$1", w) != before {
		t.Fatal("partial automatic effects committed")
	}
	exec(t, owner, "ALTER TABLE banking.collection_automatic_decision DROP CONSTRAINT synthetic_later_row_failure")
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "2" || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "1" {
		t.Fatal("same-plan atomic recovery failed")
	}
}

// TestAuto48ExactAttention excludes queued, zero and no-longer-booked manual facts from the exception count/view.
func TestAuto48ExactAttention(t *testing.T) {
	owner, pool, db := auto48Fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "attention48", 1)
	zero := change("zero", "0", "USD")
	zero["direction"] = "zero"
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("unknown", "1.00", "USD"), change("manual", "2.00", "USD"), zero}); e != nil {
		t.Fatal(e)
	}
	q := query45Selector()
	q["topic"] = map[string]any{"kind": "needs_attention"}
	if p := query45Page(t, db, w, q); p["matching_count"] != "0" {
		t.Fatal("queued routine rows appeared as exceptions")
	}
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	rows := rule46Query(t, s, w)["items"].([]any)
	var selected []map[string]any
	for _, x := range rows {
		r := x.(map[string]any)
		if r["money"].(map[string]any)["amount"] == "2" {
			selected = selection([]any{x})
		}
	}
	if len(selected) != 1 {
		t.Fatal("manual fixture row selection")
	}
	p, e := rule46Preview(t, s, w, selected, map[string]any{"kind": "uncategorized"})
	if e != nil {
		t.Fatal(e)
	}
	rule46Complete(t, s, w, "manual", p)
	auto48ProjectionEqual(t, owner, w)
	worker := &AutomaticService{DB: pool}
	for i := 0; i < 2; i++ {
		if e := worker.Step(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	pending := change("manual", "2.00", "USD")
	pending["operation"] = "modified"
	pending["source_state"] = "pending"
	if _, e := publish(t, db, w, sources[0], "pending", "1", "c1", "c2", []map[string]any{pending}); e != nil {
		t.Fatal(e)
	}
	status, e := (&AutomationService{DB: pool}).GetAutomation(context.Background(), Request{Actor: actor, Workspace: w})
	v := rule46Object(t, status, e)
	p = query45Page(t, db, w, q)
	if v["attention_count"] != "1" || p["matching_count"] != "1" || len(p["items"].([]any)) != 1 {
		t.Fatal("exception view/count included pending or unavailable records")
	}
	auto48ProjectionEqual(t, owner, w)
}

// TestAuto48Lifecycle exercises actual trusted matcher/service phases through the restricted login.
func TestAuto48Lifecycle(t *testing.T) {
	owner, pool, db := auto48Fixture(t)
	s := &Service{DB: pool, Rules: RuleProcessor{}}
	settings := &AutomationService{DB: pool}
	worker := &AutomaticService{DB: pool}
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "auto48", 1)
	changes := []map[string]any{change("routine", "0.10", "USD"), change("exception", "3.00", "USD")}
	changes[1]["merchant_name"] = "Fictional unknown"
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", changes); e != nil {
		t.Fatal(e)
	}
	seedTopics(t, db, w)
	rule46Catalog(t, s, w, "rule", rule46Rule("00000000-0000-4000-8000-000000006001", "Fictional cafe rule", "Fictional Cafe", childTopic))
	before := rule46Hash(t, owner)
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal("prepare", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "0" {
		t.Fatal("prepare applied categories")
	}
	run := value(t, owner, "SELECT run_id::text FROM banking.collection_automatic_run WHERE workspace_id=$1", w)
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal("execute", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "2" || before != rule46Hash(t, owner) {
		t.Fatal("automatic effects escaped analytic boundary")
	}
	p := rule46Query(t, s, w)
	for _, x := range p["items"].([]any) {
		r := x.(map[string]any)
		a := r["categorization"].(map[string]any)
		if r["merchant"] == "Fictional Cafe" {
			if a["state"] != "automatic" || a["origin"] != "rule" || r["owner_topic"].(map[string]any)["topic_id"] != childTopic {
				t.Fatal("routine label missing")
			}
		} else if a["state"] != "needs_attention" {
			t.Fatal("unknown hidden as success")
		}
	}
	first, e := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.execute_collection_automatic_run($1,$2,$3)", actor, w, run)
	if e != nil {
		t.Fatal(e)
	}
	again, e := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.execute_collection_automatic_run($1,$2,$3)", actor, w, run)
	if e != nil || raw(first) != raw(again) {
		t.Fatal("terminal replay changed")
	}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal("restart", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "1" {
		t.Fatal("restart reclassified unchanged rows")
	}
	b, e := settings.GetAutomation(context.Background(), Request{Actor: actor, Workspace: w})
	status := rule46Object(t, b, e)
	if status["pending_count"] != "0" || status["attention_count"] != "1" || status["last_run"].(map[string]any)["run_id"] != run {
		t.Fatal("whole-result status or receipt missing")
	}
	rows := p["items"].([]any)
	var chosen []map[string]any
	for _, x := range rows {
		r := x.(map[string]any)
		if r["merchant"] == "Fictional Cafe" {
			chosen = selection([]any{x})
		}
	}
	preview, e := rule46Preview(t, s, w, chosen, map[string]any{"kind": "uncategorized"})
	if e != nil {
		t.Fatal(e)
	}
	rule46Complete(t, s, w, "manual-correction", preview)
	for i := 0; i < 2; i++ {
		if e := worker.Step(context.Background()); e != nil {
			t.Fatal("manual preservation", e)
		}
	}
	p = rule46Query(t, s, w)
	for _, x := range p["items"].([]any) {
		r := x.(map[string]any)
		if r["merchant"] == "Fictional Cafe" && (r["owner_topic"].(map[string]any)["state"] != "uncategorized" || r["categorization"].(map[string]any)["origin"] != "manual") {
			t.Fatal("manual correction overwritten")
		}
	}
	if before != rule46Hash(t, owner) {
		t.Fatal("categories changed source, money or accounting")
	}
}

// TestAuto48PolicyStale protects retained work against a pause and proves exact settings recovery.
func TestAuto48PolicyStale(t *testing.T) {
	owner, pool, db := auto48Fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "pause48", 1)
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("one", "1.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	worker := &AutomaticService{DB: pool}
	settings := &AutomationService{DB: pool}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	run := value(t, owner, "SELECT run_id::text FROM banking.collection_automatic_run WHERE workspace_id=$1", w)
	request := Request{Actor: actor, Workspace: w, Body: `{"key":"pause","expected_revision":"0","enabled":false}`}
	a, e := settings.SetAutomation(context.Background(), request)
	if e != nil {
		t.Fatal(e)
	}
	b, e := settings.SetAutomation(context.Background(), request)
	if e != nil || string(a) != string(b) {
		t.Fatal("settings replay changed")
	}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal(e)
	}
	if value(t, owner, "SELECT state FROM banking.collection_automatic_run WHERE run_id=$1", run) != "stale" || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "0" {
		t.Fatal("paused plan applied")
	}
	request.Body = `{"key":"pause","expected_revision":"0","enabled":true}`
	_, e = settings.SetAutomation(context.Background(), request)
	rule46Failure(t, e, 409, "collection_conflict")
	request.Actor = other
	_, e = settings.GetAutomation(context.Background(), request)
	rule46Failure(t, e, 404, "collection_denied")
	request.Actor = actor
	request.Body = `{"key":"duplicate","expected_revision":"1","enabled":true,"enabled":false}`
	_, e = settings.SetAutomation(context.Background(), request)
	rule46Failure(t, e, 400, "invalid_collection")
}

// TestAuto48InventoryFairness rotates scheduling even when no selected workspace phase succeeds.
func TestAuto48InventoryFairness(t *testing.T) {
	owner, _, db := auto48Fixture(t)
	for n := 0; n < 11; n++ {
		principal := fmt.Sprintf("00000000-0000-4000-8000-%012d", 48000+n)
		exec(t, owner, "INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic-auto48-fairness',$2)", principal, fmt.Sprintf("owner-%d", n))
		provision(t, db, principal)
	}
	seen := map[string]bool{}
	for round := 0; round < 2; round++ {
		inventory, err := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.collection_automation_work()")
		if err != nil {
			t.Fatal("inventory", err)
		}
		workspaces, ok := inventory["workspaces"].([]any)
		if !ok || len(workspaces) == 0 || len(workspaces) > 10 {
			t.Fatal("unbounded or empty work inventory")
		}
		for _, item := range workspaces {
			workspace, ok := item.(map[string]any)["workspace_id"].(string)
			if !ok || workspace == "" {
				t.Fatal("work inventory lost identity")
			}
			seen[workspace] = true
		}
		// Do not read context or execute work: this models every phase failing after inventory commit.
	}
	if len(seen) != 11 {
		t.Fatal("persistent first ten workspaces starved the remaining workspace")
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "0" ||
		value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "0" ||
		value(t, owner, "SELECT count(*)::text FROM banking.collection_automation_state WHERE initialized") != "0" {
		t.Fatal("scheduling inventory created classification effects")
	}
}

// TestAuto48RetainedRollback refuses to erase accepted automatic evidence on downgrade.
func TestAuto48RetainedRollback(t *testing.T) {
	owner, pool, db := auto48Fixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "retained48", 1)
	if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("one", "1.00", "USD")}); e != nil {
		t.Fatal(e)
	}
	worker := &AutomaticService{DB: pool}
	for i := 0; i < 2; i++ {
		if e := worker.Step(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	down, e := os.ReadFile("/proof/migrations/000048_automatic_categorization.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	_, e = owner.Exec(context.Background(), string(down))
	if e == nil || !strings.Contains(e.Error(), "automatic_categorization_retained_records") {
		t.Fatal("retained automatic evidence was not protected")
	}
	exec(t, owner, "ROLLBACK")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "1" || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run WHERE state='committed'") != "1" {
		t.Fatal("failed downgrade erased evidence")
	}
}

// TestAuto48ContextCache proves the retained SQL context is an authority boundary, not caller-supplied facts.
func TestAuto48ContextCache(t *testing.T) {
	for _, scenario := range []string{"private_cache_and_read_only_status", "changed_context", "source_changed_before_prepare", "same_context_prepare_replay"} {
		t.Run(scenario, func(t *testing.T) {
			owner, _, db := auto48Fixture(t)
			w := provision(t, db, actor)
			_, sources := register(t, db, w, actor, "cache48", 1)
			if _, e := publish(t, db, w, sources[0], "facts", "0", nil, "c1", []map[string]any{change("one", "0.10", "USD")}); e != nil {
				t.Fatal(e)
			}
			if scenario == "private_cache_and_read_only_status" {
				before := value(t, owner, "SELECT to_jsonb(s)::text FROM banking.collection_automation_state s WHERE workspace_id=$1", w)
				if _, e := call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_automation($1,$2)", actor, w); e != nil {
					t.Fatal(e)
				}
				if before != value(t, owner, "SELECT to_jsonb(s)::text FROM banking.collection_automation_state s WHERE workspace_id=$1", w) {
					t.Fatal("public status changed private automation state")
				}
			}
			wrapper, e := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.collection_automation_context($1,$2)", actor, w)
			if e != nil || wrapper["state"] != "ready" {
				t.Fatal("SQL context unavailable", e)
			}
			c := wrapper["context"].(map[string]any)
			var typed ruleContext
			if e = json.Unmarshal([]byte(raw(c)), &typed); e != nil {
				t.Fatal(e)
			}
			results, e := evaluateRuleContext(context.Background(), typed, false)
			if e != nil || len(results) != 1 {
				t.Fatal("existing matcher unavailable", e)
			}
			payload := raw(map[string]any{"results": results})
			switch scenario {
			case "private_cache_and_read_only_status":
				runtime := connect(t, db, "synthetic_runtime")
				defer runtime.Close(context.Background())
				for _, q := range []string{
					"SELECT context_cache FROM banking.collection_automation_state",
					"UPDATE banking.collection_automation_state SET context_cache='{}'::jsonb",
					"SELECT banking.collection_automatic_rule_context('00000000-0000-4000-8000-000000000001'::uuid,'00000000-0000-4000-8000-000000000002'::uuid,'{}'::text,true,'[]'::jsonb)",
					"SELECT banking.collection_automatic_project_row('00000000-0000-4000-8000-000000000001'::uuid,'00000000-0000-4000-8000-000000000002'::uuid,'00000000-0000-4000-8000-000000000003'::uuid,'00000000-0000-4000-8000-000000000004'::uuid,1::bigint,'{}'::jsonb)",
				} {
					_, e = runtime.Exec(context.Background(), q)
					code(t, e, "42501")
				}
				before := value(t, owner, "SELECT to_jsonb(s)::text FROM banking.collection_automation_state s WHERE workspace_id=$1", w)
				if _, e = call(t, db, "synthetic_runtime", "repeatable read", "SELECT banking.get_collection_automation($1,$2)", actor, w); e != nil {
					t.Fatal(e)
				}
				if before != value(t, owner, "SELECT to_jsonb(s)::text FROM banking.collection_automation_state s WHERE workspace_id=$1", w) {
					t.Fatal("status mutated or cleared an authorized retained context")
				}
			case "changed_context":
				c["facts"].([]any)[0].(map[string]any)["description"] = "Fictional caller-altered context"
				_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.prepare_collection_automatic_run($1,$2,$3,$4)", actor, w, raw(c), payload)
				code(t, e, "P2803")
				if value(t, owner, "SELECT context_cache#>>'{context,facts,0,description}' FROM banking.collection_automation_state WHERE workspace_id=$1", w) != "Fictional cafe purchase" {
					t.Fatal("caller replaced the SQL-owned context")
				}
			case "source_changed_before_prepare":
				modified := change("one", "0.20", "USD")
				modified["operation"] = "modified"
				if _, e = publish(t, db, w, sources[0], "changed", "1", "c1", "c2", []map[string]any{modified}); e != nil {
					t.Fatal(e)
				}
				_, e = call(t, db, "synthetic_runtime", "serializable", "SELECT banking.prepare_collection_automatic_run($1,$2,$3,$4)", actor, w, raw(c), payload)
				code(t, e, "P2803")
			case "same_context_prepare_replay":
				first, e := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.prepare_collection_automatic_run($1,$2,$3,$4)", actor, w, raw(c), payload)
				if e != nil || first["state"] != "prepared" {
					t.Fatal("inert plan unavailable", e)
				}
				again, e := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.prepare_collection_automatic_run($1,$2,$3,$4)", actor, w, raw(c), payload)
				if e != nil || raw(first) != raw(again) || value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != "1" {
					t.Fatal("same context/result did not recover its exact inert plan", e)
				}
			}
			wantRuns := "0"
			if scenario == "same_context_prepare_replay" {
				wantRuns = "1"
			}
			if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_run") != wantRuns ||
				value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "0" ||
				value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
				t.Fatal("cache validation created unintended effects")
			}
		})
	}
}
