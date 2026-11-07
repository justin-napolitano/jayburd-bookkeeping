//go:build integration

package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const proofTagA = "00000000-0000-4000-8000-000000000301"
const proofTagB = "00000000-0000-4000-8000-000000000302"
const proofTagForeign = "00000000-0000-4000-8000-000000000399"

// independentTagFixture owns only a synthetic clone, never the shared migration template.
func independentTagFixture(t *testing.T) (*pgx.Conn, *pgxpool.Pool, string) {
	t.Helper()
	if !*proof {
		t.Skip("use the disposable strategic tag and recovery runner")
	}
	db := fmt.Sprintf("tagrec_independent_%d", time.Now().UnixNano())
	admin := connect(t, "postgres", "postgres")
	exec(t, admin, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE tagrec_template")
	_ = admin.Close(context.Background())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cleanup, err := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
		if err != nil {
			t.Error("owned synthetic cleanup connection unavailable")
			return
		}
		defer cleanup.Close(ctx)
		if _, err = cleanup.Exec(ctx, "DROP DATABASE "+pgx.Identifier{db}.Sanitize()); err != nil {
			t.Error("owned synthetic database cleanup failed")
		}
	})
	owner := connect(t, db, "postgres")
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	exec(t, owner, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic-independent-tag','owner'),($2,'synthetic-independent-tag','other')`, actor, other)
	exec(t, owner, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic-independent-tag')`, currency)
	pool, err := pgxpool.New(context.Background(), "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=2")
	if err != nil {
		t.Fatal("synthetic restricted pool unavailable")
	}
	t.Cleanup(pool.Close)
	return owner, pool, db
}

// independentV2 invokes only frozen entrypoints through the actual restricted login.
func independentV2(t testing.TB, db, w, operation string, body any) (map[string]any, error) {
	t.Helper()
	return independentV2As(t, db, w, actor, operation, body)
}

// independentV2As keeps a different verified principal out of every JSON request body.
func independentV2As(t testing.TB, db, w, principal, operation string, body any) (map[string]any, error) {
	t.Helper()
	iso, query := "serializable", ""
	args := []any{principal, w}
	switch operation {
	case "catalog":
		iso, query = "repeatable read", "SELECT banking.get_collection_taxonomy_v2($1,$2)"
	case "query":
		iso, query = "repeatable read", "SELECT banking.query_collection_review_v2($1,$2,$3)"
		args = append(args, raw(body))
	case "preview":
		query = "SELECT banking.preview_collection_labels_v2($1,$2,$3)"
		args = append(args, raw(body))
	case "prepare":
		query = "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)"
		args = append(args, raw(body))
	case "pending":
		iso, query = "repeatable read", "SELECT banking.pending_collection_label_command_v2($1,$2)"
	default:
		query = "SELECT banking.transition_collection_label_command_v2($1,$2,$3,$4,'{}')"
		args = append(args, body, operation)
	}
	return call(t, db, "synthetic_runtime", iso, query, args...)
}

// independentTagQuery adds the mandatory v2 tag selector without weakening any v1 filter.
func independentTagQuery() map[string]any {
	v := selector()
	v["profile"] = "private-collection-review-v2"
	v["tag"] = map[string]any{"kind": "all"}
	return v
}

// independentTagPage checks the version rather than treating any successful JSON as the new profile.
func independentTagPage(t *testing.T, db, w string) map[string]any {
	t.Helper()
	r, err := independentV2(t, db, w, "query", independentTagQuery())
	if err != nil || r["profile"] != "private-collection-review-v2" {
		t.Fatal("independent v2 query failed", err)
	}
	return r
}

// independentTagCommand reads only the frozen command wrapper and preserves immutable preview values.
func independentTagCommand(t *testing.T, wrapper map[string]any) map[string]any {
	t.Helper()
	if wrapper["profile"] != "private-collection-command-v2" {
		t.Fatal("unexpected strategic tag command profile")
	}
	c, ok := wrapper["command"].(map[string]any)
	if !ok {
		t.Fatal("expected a persisted strategic tag command")
	}
	return c
}

// independentTagPreview binds exactly the observed current snapshot and explicit selected revisions.
func independentTagPreview(t *testing.T, db, w string, selected []map[string]any, action any) map[string]any {
	t.Helper()
	p, err := independentV2(t, db, w, "preview", map[string]any{"snapshot": independentTagPage(t, db, w)["snapshot"], "selection": selected, "action": action})
	if err != nil || p["profile"] != "private-collection-preview-v2" {
		t.Fatal("independent strategic tag preview failed", err)
	}
	return p
}

// independentTagPrepare uses one fixed key per independently reviewed intent.
func independentTagPrepare(t *testing.T, db, w, key string, preview map[string]any) map[string]any {
	t.Helper()
	r, err := independentV2(t, db, w, "prepare", map[string]any{"preview_id": preview["preview_id"], "prepare_key": key})
	if err != nil {
		t.Fatal("independent strategic tag preparation failed", err)
	}
	return independentTagCommand(t, r)
}

// independentTagApply requires explicit prepare, execute and acknowledgement even for catalog edits.
func independentTagApply(t *testing.T, db, w, key string, selected []map[string]any, action any) map[string]any {
	t.Helper()
	p := independentTagPreview(t, db, w, selected, action)
	c := independentTagPrepare(t, db, w, key, p)
	r, err := independentV2(t, db, w, "execute", c["command_id"])
	if err != nil || independentTagCommand(t, r)["state"] != "committed" {
		t.Fatal("independent strategic tag execution failed", err)
	}
	if _, err = independentV2(t, db, w, "acknowledge", c["command_id"]); err != nil {
		t.Fatal("independent strategic tag acknowledgement failed", err)
	}
	return r
}

// independentTagDefinition uses fixed fictional IDs, never a public merchant taxonomy.
func independentTagDefinition(id, name, status string) map[string]any {
	return map[string]any{"tag_id": id, "name": name, "status": status}
}

// independentTagIDs compares authoritative sets independently of JSON object ordering.
func independentTagIDs(t *testing.T, tags any) []string {
	t.Helper()
	items, ok := tags.([]any)
	if !ok {
		t.Fatal("expected a strategic tag set")
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		tag, ok := item.(map[string]any)
		if !ok {
			t.Fatal("expected an attributed strategic tag")
		}
		ids = append(ids, tag["tag_id"].(string))
	}
	sort.Strings(ids)
	return ids
}

// independentSourceHash includes source evidence, primary-topic decisions and accounting history.
func independentSourceHash(t *testing.T, owner *pgx.Conn) string {
	t.Helper()
	return value(t, owner, `SELECT encode(sha256(convert_to(jsonb_build_array((SELECT coalesce(jsonb_agg(to_jsonb(s)ORDER BY workspace_id,source_account_id),'[]')FROM banking.collection_source s),(SELECT coalesce(jsonb_agg(to_jsonb(o)ORDER BY workspace_id,source_account_id,observation_id),'[]')FROM banking.collection_observation o),(SELECT coalesce(jsonb_agg(to_jsonb(i)ORDER BY collection_item_id),'[]')FROM integration.collection_item i),(SELECT coalesce(jsonb_agg(to_jsonb(q)ORDER BY workspace_id,source_account_id,observation_id,revision),'[]')FROM banking.collection_revision q),(SELECT coalesce(jsonb_agg(to_jsonb(e)ORDER BY workspace_id,taxonomy_revision),'[]')FROM banking.collection_taxonomy_edition e),(SELECT coalesce(jsonb_agg(to_jsonb(d)ORDER BY workspace_id,decision_revision,source_account_id,observation_id),'[]')FROM banking.collection_label_decision d),(SELECT coalesce(jsonb_agg(to_jsonb(j)ORDER BY journal_id),'[]')FROM ledger.journal_entry j),(SELECT coalesce(jsonb_agg(to_jsonb(l)ORDER BY journal_id,line_number),'[]')FROM ledger.journal_line l),(SELECT count(*)FROM integration.outbox_message))::text,'UTF8')),'hex')`)
}

// TestIndependentStrategicTagSets proves tag sets never replace primary topics or source/accounting facts.
func TestIndependentStrategicTagSets(t *testing.T) {
	owner, _, db := independentTagFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "independent-tag-item", 1)
	if _, err := publish(t, db, w, sources[0], "independent-tag-source", "0", nil, "c1", []map[string]any{change("tag-quantity-a", "9007199254740993.01", "USD"), change("tag-quantity-b", "0.10", "USD")}); err != nil {
		t.Fatal(err)
	}
	seedTopics(t, db, w)
	selected := selection(independentTagPage(t, db, w)["items"].([]any))
	// Existing single-valued primary topics remain visible and immutable under all tag-only actions.
	apply(t, db, w, "independent-primary-topic", selected, map[string]any{"kind": "topic", "topic_id": childTopic})
	before := independentSourceHash(t, owner)
	catalogAction := map[string]any{"kind": "tag_catalog", "tags": []map[string]any{independentTagDefinition(proofTagA, "Fictional Essential", "active"), independentTagDefinition(proofTagB, "Fictional Opportunity", "active")}}
	independentTagApply(t, db, w, "independent-tag-catalog", []map[string]any{}, catalogAction)
	added := independentTagApply(t, db, w, "independent-tag-add", selected, map[string]any{"kind": "tags", "operation": "add", "tag_ids": []string{proofTagB, proofTagA}})
	preview := independentTagCommand(t, added)["preview"].(map[string]any)
	results := preview["tag_results"].([]any)
	if len(results) != 2 {
		t.Fatal("tag preview expanded or truncated the explicit selection")
	}
	wantIdentities, actualIdentities := []string{}, []string{}
	for _, item := range selected {
		wantIdentities = append(wantIdentities, raw(item))
	}
	for _, item := range selection(results) {
		actualIdentities = append(actualIdentities, raw(item))
	}
	sort.Strings(wantIdentities)
	sort.Strings(actualIdentities)
	if !reflect.DeepEqual(wantIdentities, actualIdentities) {
		t.Fatal("authoritative tag results drifted from the exact composite revisions")
	}
	for _, item := range results {
		r := item.(map[string]any)
		if !reflect.DeepEqual(independentTagIDs(t, r["tags"]), []string{proofTagA, proofTagB}) {
			t.Fatal("authoritative tag result omitted an explicit tag")
		}
	}
	for _, row := range independentTagPage(t, db, w)["items"].([]any) {
		r := row.(map[string]any)
		if r["owner_topic"].(map[string]any)["topic_id"] != childTopic || r["owner_tags"].(map[string]any)["state"] != "current" || !reflect.DeepEqual(independentTagIDs(t, r["owner_tags"].(map[string]any)["tags"]), []string{proofTagA, proofTagB}) {
			t.Fatal("strategic tags overwrote the primary topic or lost their set")
		}
	}
	filter := independentTagQuery()
	filter["tag"] = map[string]any{"kind": "tag", "tag_id": proofTagA}
	r, err := independentV2(t, db, w, "query", filter)
	if err != nil || r["matching_count"] != "2" {
		t.Fatal("many-to-many tags duplicated source rows/counts", err)
	}
	independentTagApply(t, db, w, "independent-tag-remove", selected, map[string]any{"kind": "tags", "operation": "remove", "tag_ids": []string{proofTagB}})
	for _, row := range independentTagPage(t, db, w)["items"].([]any) {
		if !reflect.DeepEqual(independentTagIDs(t, row.(map[string]any)["owner_tags"].(map[string]any)["tags"]), []string{proofTagA}) {
			t.Fatal("remove changed a different tag")
		}
	}
	independentTagApply(t, db, w, "independent-tag-empty", selected, map[string]any{"kind": "tags", "operation": "replace", "tag_ids": []string{}})
	filter["tag"] = map[string]any{"kind": "untagged"}
	r, err = independentV2(t, db, w, "query", filter)
	if err != nil || r["matching_count"] != "2" || independentSourceHash(t, owner) != before {
		t.Fatal("explicit empty replacement changed original source/topic/accounting state", err)
	}
	// The old version still has precisely its original profile, selector and row fields.
	v1 := page(t, db, w)
	if v1["profile"] != "private-collection-review-v1" {
		t.Fatal("strategic tags silently changed the legacy profile")
	}
	for _, row := range v1["items"].([]any) {
		if _, exists := row.(map[string]any)["owner_tags"]; exists {
			t.Fatal("v2 tag fields leaked into v1 financial JSON")
		}
	}
}

// TestIndependentStrategicTagConflicts covers bounds, retirement, stale pins and stale source sets.
func TestIndependentStrategicTagConflicts(t *testing.T) {
	owner, _, db := independentTagFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "independent-tag-conflicts", 1)
	if _, err := publish(t, db, w, sources[0], "conflict-source", "0", nil, "c1", []map[string]any{change("tag-conflict-a", "10.01", "USD"), change("tag-conflict-b", "20.02", "USD")}); err != nil {
		t.Fatal(err)
	}
	selected := selection(independentTagPage(t, db, w)["items"].([]any))
	definitions := []map[string]any{independentTagDefinition(proofTagA, "Fictional Essential", "active"), independentTagDefinition(proofTagB, "Fictional Opportunity", "active")}
	ids := []string{proofTagA, proofTagB}
	for i := 0; i < 24; i++ {
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", 400+i)
		definitions = append(definitions, independentTagDefinition(id, fmt.Sprintf("Fictional set member %d", i), "active"))
		ids = append(ids, id)
	}
	independentTagApply(t, db, w, "bounded-catalog", []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": definitions})
	// A real other owner's catalog identity cannot be used as an implicitly shared private tag.
	foreignWorkspace := provision(t, db, other)
	foreignPage, err := independentV2As(t, db, foreignWorkspace, other, "query", independentTagQuery())
	if err != nil {
		t.Fatal(err)
	}
	foreignPreview, err := independentV2As(t, db, foreignWorkspace, other, "preview", map[string]any{"snapshot": foreignPage["snapshot"], "selection": []any{}, "action": map[string]any{"kind": "tag_catalog", "tags": []map[string]any{independentTagDefinition(proofTagForeign, "Fictional other owner tag", "active")}}})
	if err != nil {
		t.Fatal(err)
	}
	foreignPrepared, err := independentV2As(t, db, foreignWorkspace, other, "prepare", map[string]any{"preview_id": foreignPreview["preview_id"], "prepare_key": "foreign-tag-catalog"})
	if err != nil {
		t.Fatal(err)
	}
	foreignID := independentTagCommand(t, foreignPrepared)["command_id"].(string)
	for _, operation := range []string{"execute", "acknowledge"} {
		if _, err = independentV2As(t, db, foreignWorkspace, other, operation, foreignID); err != nil {
			t.Fatal(err)
		}
	}
	pageBefore := independentTagPage(t, db, w)
	for _, tc := range []struct {
		action any
		code   string
	}{
		{map[string]any{"kind": "tags", "operation": "add", "tag_ids": []string{proofTagForeign}}, "P2801"},
		{map[string]any{"kind": "tags", "operation": "add", "tag_ids": []string{proofTagA, proofTagA}}, "P2800"},
		{map[string]any{"kind": "tags", "operation": "replace", "tag_ids": ids}, "P2805"},
	} {
		_, err := independentV2(t, db, w, "preview", map[string]any{"snapshot": pageBefore["snapshot"], "selection": selected, "action": tc.action})
		code(t, err, tc.code)
	}
	independentTagApply(t, db, w, "bounded-full-set", selected, map[string]any{"kind": "tags", "operation": "replace", "tag_ids": ids[:25]})
	_, err = independentV2(t, db, w, "preview", map[string]any{"snapshot": independentTagPage(t, db, w)["snapshot"], "selection": selected, "action": map[string]any{"kind": "tags", "operation": "add", "tag_ids": ids[25:]}})
	code(t, err, "P2805")
	for _, row := range independentTagPage(t, db, w)["items"].([]any) {
		if len(independentTagIDs(t, row.(map[string]any)["owner_tags"].(map[string]any)["tags"])) != 25 {
			t.Fatal("rejected twenty-sixth tag partially changed a selected set")
		}
	}
	// A tag edition change invalidates an otherwise exact prepared tag operation.
	staleCatalog := independentTagPreview(t, db, w, selected, map[string]any{"kind": "tags", "operation": "remove", "tag_ids": []string{proofTagA}})
	definitions[0] = independentTagDefinition(proofTagA, "Fictional Essential Renamed", "active")
	definitions[1] = independentTagDefinition(proofTagB, "Fictional Opportunity", "retired")
	independentTagApply(t, db, w, "retired-catalog", []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": definitions})
	_, err = independentV2(t, db, w, "prepare", map[string]any{"preview_id": staleCatalog["preview_id"], "prepare_key": "stale-tag-catalog"})
	code(t, err, "P2803")
	for _, operation := range []string{"add", "replace"} {
		_, err = independentV2(t, db, w, "preview", map[string]any{"snapshot": independentTagPage(t, db, w)["snapshot"], "selection": selected, "action": map[string]any{"kind": "tags", "operation": operation, "tag_ids": []string{proofTagB}}})
		code(t, err, "P2803")
	}
	independentTagApply(t, db, w, "remove-retired-tag", selected, map[string]any{"kind": "tags", "operation": "remove", "tag_ids": []string{proofTagB}})
	// A separate primary taxonomy change is also a required pin, even for tag-only work.
	taxonomyStale := independentTagPreview(t, db, w, selected, map[string]any{"kind": "tags", "operation": "remove", "tag_ids": []string{proofTagA}})
	seedTopics(t, db, w)
	_, err = independentV2(t, db, w, "prepare", map[string]any{"preview_id": taxonomyStale["preview_id"], "prepare_key": "stale-primary-taxonomy"})
	code(t, err, "P2803")
	// Correcting one source makes its old tag set stale; add/remove cannot silently inherit it.
	correction := change("tag-conflict-a", "15.03", "USD")
	correction["operation"] = "modified"
	if _, err = publish(t, db, w, sources[0], "conflict-correction", "1", "c1", "c2", []map[string]any{correction}); err != nil {
		t.Fatal(err)
	}
	current := independentTagPage(t, db, w)
	currentSelection := selection(current["items"].([]any))
	staleCount := 0
	for _, row := range current["items"].([]any) {
		if row.(map[string]any)["owner_tags"].(map[string]any)["state"] == "stale" {
			staleCount++
		}
	}
	if staleCount != 1 {
		t.Fatal("source correction did not isolate the stale tag set")
	}
	for _, operation := range []string{"add", "remove"} {
		_, err = independentV2(t, db, w, "preview", map[string]any{"snapshot": current["snapshot"], "selection": currentSelection, "action": map[string]any{"kind": "tags", "operation": operation, "tag_ids": []string{proofTagA}}})
		code(t, err, "P2803")
	}
	before := independentSourceHash(t, owner)
	independentTagApply(t, db, w, "explicit-current-replacement", currentSelection, map[string]any{"kind": "tags", "operation": "replace", "tag_ids": []string{proofTagA}})
	if independentSourceHash(t, owner) != before {
		t.Fatal("explicit stale-set replacement rewrote source, topic or accounting history")
	}
}

// TestIndependentTagAtomicExecution fails after one insert attempt and proves no partial durable tag effect.
func TestIndependentTagAtomicExecution(t *testing.T) {
	owner, _, db := independentTagFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "independent-atomic-tag-item", 1)
	if _, err := publish(t, db, w, sources[0], "atomic-tag-source", "0", nil, "c1", []map[string]any{change("atomic-tag-a", "10.01", "USD"), change("atomic-tag-b", "20.02", "USD")}); err != nil {
		t.Fatal(err)
	}
	independentTagApply(t, db, w, "atomic-tag-catalog", []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": []map[string]any{independentTagDefinition(proofTagA, "Fictional atomic tag", "active")}})
	selected := selection(independentTagPage(t, db, w)["items"].([]any))
	p := independentTagPreview(t, db, w, selected, map[string]any{"kind": "tags", "operation": "replace", "tag_ids": []string{proofTagA}})
	c := independentTagPrepare(t, db, w, "atomic-tag-command", p)
	id := c["command_id"].(string)
	before := independentSourceHash(t, owner)
	decisionRevision := value(t, owner, "SELECT decision_revision::text FROM banking.collection_label_state WHERE workspace_id=$1", w)
	// This fault exists only in the owned clone and observes an actual prior insertion attempt.
	exec(t, owner, `CREATE FUNCTION banking.independent_tag_fault()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN IF EXISTS(SELECT FROM banking.collection_tag_decision WHERE workspace_id=NEW.workspace_id AND command_id=NEW.command_id)THEN RAISE EXCEPTION USING ERRCODE='P2805',MESSAGE='synthetic_tag_fault';END IF;RETURN NEW;END$$;CREATE TRIGGER independent_tag_fault BEFORE INSERT ON banking.collection_tag_decision FOR EACH ROW EXECUTE FUNCTION banking.independent_tag_fault();`)
	_, err := independentV2(t, db, w, "execute", id)
	code(t, err, "P2805")
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision WHERE command_id=$1", id) != "0" || value(t, owner, "SELECT state FROM banking.collection_label_command WHERE command_id=$1", id) != "prepared" || value(t, owner, "SELECT count(*)::text FROM audit.collection_label_action WHERE command_id=$1 AND operation='execute'", id) != "0" || value(t, owner, "SELECT decision_revision::text FROM banking.collection_label_state WHERE workspace_id=$1", w) != decisionRevision {
		t.Fatal("late tag failure committed a partial decision, pointer, command or audit")
	}
	exec(t, owner, "DROP TRIGGER independent_tag_fault ON banking.collection_tag_decision;DROP FUNCTION banking.independent_tag_fault();")
	if _, err = independentV2(t, db, w, "execute", id); err != nil {
		t.Fatal("same immutable tag command did not recover after fixture fault removal", err)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision WHERE command_id=$1", id) != "2" || independentSourceHash(t, owner) != before {
		t.Fatal("atomic retry changed a nonselected fact or lost a selected tag set")
	}
	if _, err = independentV2(t, db, w, "acknowledge", id); err != nil {
		t.Fatal(err)
	}
}

// TestIndependentHundredTagHTTP proves the real service deadlines on a bounded selection over 3,000 facts.
func TestIndependentHundredTagHTTP(t *testing.T) {
	owner, pool, db := independentTagFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "independent-http-3000-item", 1)
	for batch := 0; batch < 3; batch++ {
		changes := make([]map[string]any, 0, 1000)
		for i := 0; i < 1000; i++ {
			changes = append(changes, change(fmt.Sprintf("independent-http-%04d", batch*1000+i), "0.10", "USD"))
		}
		var start any
		if batch > 0 {
			start = fmt.Sprintf("c%d", batch)
		}
		if _, err := publish(t, db, w, sources[0], fmt.Sprintf("http-3000-batch-%d", batch), fmt.Sprint(batch), start, fmt.Sprintf("c%d", batch+1), changes); err != nil {
			t.Fatal("independent 3,000-row source publication failed", err)
		}
	}
	independentTagApply(t, db, w, "http-3000-catalog", []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": []map[string]any{independentTagDefinition(proofTagA, "Fictional bounded batch", "active")}})
	handler := Handler(labelHTTPAuth{}, &Service{DB: pool})
	base := "/v2/collections/" + w
	request := func(suffix string, body any) map[string]json.RawMessage {
		t.Helper()
		start := time.Now()
		r := independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodPost, base+suffix, raw(body), 200)
		t.Logf("actual restricted HTTP %s: %s", suffix, time.Since(start).Round(time.Millisecond))
		return r
	}
	query := independentTagQuery()
	query["limit"] = 100
	page := request("/review/query", query)
	rows := labelHTTPRows(t, page["items"])
	if labelHTTPText(t, page["matching_count"]) != "3000" || len(rows) != 100 {
		t.Fatal("large source query did not expose exactly the bounded first selection")
	}
	before := independentSourceHash(t, owner)
	p := request("/labels/preview", map[string]any{"snapshot": page["snapshot"], "selection": labelHTTPSelection(t, rows), "action": map[string]any{"kind": "tags", "operation": "replace", "tag_ids": []string{proofTagA}}})
	if labelHTTPText(t, p["selected_count"]) != "100" || len(labelHTTPRows(t, p["tag_results"])) != 100 {
		t.Fatal("large history preview expanded or truncated the explicit selection")
	}
	prepared := request("/label-commands", map[string]any{"preview_id": labelHTTPText(t, p["preview_id"]), "prepare_key": "http-3000-exact100"})
	id := labelHTTPText(t, labelHTTPObject(t, prepared["command"])["command_id"])
	request("/label-commands/"+id+"/execute", map[string]any{})
	request("/label-commands/"+id+"/acknowledge", map[string]any{})
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_tag_decision WHERE command_id=$1", id) != "100" || independentSourceHash(t, owner) != before {
		t.Fatal("bounded 100-row HTTP action changed unselected or original source/accounting facts")
	}
}

// TestIndependentStrategicTagHTTP proves the real v2 HTTP/service/SQL surface and v1 isolation.
func TestIndependentStrategicTagHTTP(t *testing.T) {
	owner, pool, db := independentTagFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "independent-http-tag-item", 1)
	if _, err := publish(t, db, w, sources[0], "independent-http-tag-source", "0", nil, "c1", []map[string]any{change("http-tag-a", "9007199254740993.01", "USD"), change("http-tag-b", "0.10", "USD")}); err != nil {
		t.Fatal(err)
	}
	handler := Handler(labelHTTPAuth{}, &Service{DB: pool})
	base := "/v2/collections/" + w
	request := func(method, suffix string, body any, status int) map[string]json.RawMessage {
		t.Helper()
		encoded := ""
		if method == http.MethodPost {
			encoded = raw(body)
		}
		return independentRecoveryRequest(t, handler, "synthetic-label-owner", method, base+suffix, encoded, status)
	}
	beforeV1 := page(t, db, w)
	before := independentSourceHash(t, owner)
	catalog := request(http.MethodGet, "/taxonomy", nil, 200)
	if labelHTTPText(t, catalog["profile"]) != "private-collection-catalog-v2" || string(catalog["tags"]) != "[]" || len(catalog["tag_revision"]) == 0 {
		t.Fatal("new catalog omitted its explicit version or independent tag edition")
	}
	independentRecoveryRequest(t, handler, "synthetic-label-other", http.MethodGet, base+"/taxonomy", "", 404)
	missingTagSelector := independentTagQuery()
	delete(missingTagSelector, "tag")
	request(http.MethodPost, "/review/query", missingTagSelector, 400)
	// Catalog editing travels through the same explicit durable command lifecycle as a tag set.
	current := request(http.MethodPost, "/review/query", independentTagQuery(), 200)
	preview := request(http.MethodPost, "/labels/preview", map[string]any{"snapshot": current["snapshot"], "selection": []any{}, "action": map[string]any{"kind": "tag_catalog", "tags": []map[string]any{independentTagDefinition(proofTagA, "Fictional HTTP strategy", "active")}}}, 200)
	if labelHTTPText(t, preview["profile"]) != "private-collection-preview-v2" || string(preview["tag_results"]) != "[]" {
		t.Fatal("catalog preview invented per-observation tag effects")
	}
	prepared := request(http.MethodPost, "/label-commands", map[string]any{"preview_id": labelHTTPText(t, preview["preview_id"]), "prepare_key": "independent-http-catalog"}, 200)
	command := labelHTTPObject(t, prepared["command"])
	id := labelHTTPText(t, command["command_id"])
	if labelHTTPText(t, prepared["profile"]) != "private-collection-command-v2" {
		t.Fatal("new command changed or omitted its explicit version")
	}
	request(http.MethodGet, "/label-commands/pending", nil, 200)
	independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodGet, "/v1/collections/"+w+"/label-commands/pending", "", 409)
	committed := request(http.MethodPost, "/label-commands/"+id+"/execute", map[string]any{}, 200)
	if raw(request(http.MethodPost, "/label-commands/"+id+"/execute", map[string]any{}, 200)) != raw(committed) {
		t.Fatal("catalog command replay changed the saved review")
	}
	request(http.MethodPost, "/label-commands/"+id+"/acknowledge", map[string]any{}, 200)
	current = request(http.MethodPost, "/review/query", independentTagQuery(), 200)
	rows := labelHTTPRows(t, current["items"])
	preview = request(http.MethodPost, "/labels/preview", map[string]any{"snapshot": current["snapshot"], "selection": labelHTTPSelection(t, rows), "action": map[string]any{"kind": "tags", "operation": "add", "tag_ids": []string{proofTagA}}}, 200)
	if len(labelHTTPRows(t, preview["tag_results"])) != 2 || labelHTTPText(t, preview["selected_count"]) != "2" {
		t.Fatal("HTTP tag preview expanded or truncated the explicit selection")
	}
	prepared = request(http.MethodPost, "/label-commands", map[string]any{"preview_id": labelHTTPText(t, preview["preview_id"]), "prepare_key": "independent-http-set"}, 200)
	id = labelHTTPText(t, labelHTTPObject(t, prepared["command"])["command_id"])
	request(http.MethodPost, "/label-commands/"+id+"/execute", map[string]any{}, 200)
	request(http.MethodPost, "/label-commands/"+id+"/acknowledge", map[string]any{}, 200)
	filtered := independentTagQuery()
	filtered["tag"] = map[string]any{"kind": "tag", "tag_id": proofTagA}
	result := request(http.MethodPost, "/review/query", filtered, 200)
	if labelHTTPText(t, result["matching_count"]) != "2" || len(labelHTTPRows(t, result["items"])) != 2 {
		t.Fatal("HTTP tag filter duplicated original financial rows")
	}
	for _, row := range labelHTTPRows(t, result["items"]) {
		money := labelHTTPObject(t, row["money"])
		amount := labelHTTPText(t, money["amount"])
		if amount != "9007199254740993.01" && amount != "0.1" {
			t.Fatal("tag action changed exact original money")
		}
	}
	if independentSourceHash(t, owner) != before || raw(beforeV1["items"]) != raw(page(t, db, w)["items"]) {
		t.Fatal("new HTTP tags changed v1 source/topic/accounting projection")
	}
	// Historical v1 work remains recoverable through the v2 wrapper without changing its preview.
	p, err := previewV1Independent(t, db, w)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := prepare(t, db, w, "independent-historical-v1", p)
	if err != nil {
		t.Fatal(err)
	}
	legacyID := commandV1Independent(t, legacy)["command_id"].(string)
	union := request(http.MethodGet, "/label-commands/pending", nil, 200)
	unionCommand := labelHTTPObject(t, union["command"])
	if string(unionCommand["preview"]) == "null" || !independentJSONEqual(t, unionCommand["preview"], json.RawMessage(raw(p))) {
		t.Fatal("v2 recovery rewrote a historical v1 preview")
	}
	request(http.MethodPost, "/label-commands/"+legacyID+"/cancel", map[string]any{}, 200)
	request(http.MethodPost, "/label-commands/"+legacyID+"/acknowledge", map[string]any{}, 200)
}

// independentJSONEqual ignores object ordering/whitespace without converting numeric values to floats.
func independentJSONEqual(t *testing.T, left, right json.RawMessage) bool {
	t.Helper()
	decode := func(value json.RawMessage) any {
		decoder := json.NewDecoder(strings.NewReader(string(value)))
		decoder.UseNumber()
		var result any
		if decoder.Decode(&result) != nil {
			t.Fatal("invalid synthetic JSON comparison")
		}
		return result
	}
	return reflect.DeepEqual(decode(left), decode(right))
}

// previewV1Independent keeps old wire construction explicit rather than projecting v2 fields backward.
func previewV1Independent(t *testing.T, db, w string) (map[string]any, error) {
	t.Helper()
	return preview(t, db, w, page(t, db, w)["snapshot"], []map[string]any{}, topicsAction([]map[string]any{}))
}

// commandV1Independent rejects an accidental silent version upgrade in the compatibility fixture.
func commandV1Independent(t *testing.T, wrapper map[string]any) map[string]any {
	t.Helper()
	return command(t, wrapper)
}

// lostRecoveryCommit acknowledges a real commit internally but deliberately loses one transport reply.
type lostRecoveryCommit struct {
	pool *pgxpool.Pool
	lose atomic.Bool
}

func (d *lostRecoveryCommit) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := d.pool.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &lostRecoveryTx{Tx: tx, lose: &d.lose}, nil
}

type lostRecoveryTx struct {
	pgx.Tx
	lose *atomic.Bool
}

func (t *lostRecoveryTx) Commit(ctx context.Context) error {
	if err := t.Tx.Commit(ctx); err != nil {
		return err
	}
	if t.lose.Swap(false) {
		return errors.New("synthetic committed reply unavailable")
	}
	return nil
}

// independentRecoveryRequest crosses the actual handler/service and preserves JSON field types.
func independentRecoveryRequest(t *testing.T, handler http.Handler, token, method, path, body string, status int) map[string]json.RawMessage {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+token)
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != status {
		t.Fatalf("synthetic recovery HTTP %s status %d want %d: %s", path, w.Code, status, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("recovery response can be cached")
	}
	return labelHTTPObject(t, w.Body.Bytes())
}

// independentRecoveryFields rejects every accidental payload-bearing response field.
func independentRecoveryFields(t *testing.T, object map[string]json.RawMessage, fields ...string) {
	t.Helper()
	wanted := append([]string(nil), fields...)
	sort.Strings(wanted)
	actual := make([]string, 0, len(object))
	for key := range object {
		actual = append(actual, key)
	}
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, wanted) {
		t.Fatal("payload-free recovery exposed an unexpected field")
	}
}

// independentLedgerSetup is valid reviewed old-engine intent; gate proofs never execute it.
func independentLedgerSetup() string {
	return raw(map[string]any{"operation": "setup", "entity_id": nil, "book_id": nil, "body": map[string]any{"entity_name": "Fictional recovery scope", "entity_kind": "personal", "book_name": "Fictional recovery book", "starts_on": "2026-01-01", "ends_before": "2027-01-01", "functional_currency_version_id": currency, "accounts": []map[string]any{{"code": "1000", "name": "Fictional asset", "category": "asset", "normal_side": "debit", "monetary": true}, {"code": "3000", "name": "Fictional equity", "category": "equity", "normal_side": "credit", "monetary": false}}}})
}

// independentGrant changes only the fictional owner's explicit private capabilities.
func independentGrant(t *testing.T, db, key, expected string, capabilities []string) {
	t.Helper()
	if _, err := call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,$2,$3)", actor, key, raw(map[string]any{"expected_revision": expected, "capabilities": capabilities, "evidence_ref": evidence})); err != nil {
		t.Fatal("synthetic capability change failed", err)
	}
}

// independentRecoveryStatus requires an exact payload-free object, never a saved preview projection.
func independentRecoveryStatus(t *testing.T, handler http.Handler, expected string) string {
	t.Helper()
	r := independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodGet, "/v1/collection-recovery", "", 200)
	independentRecoveryFields(t, r, "profile", "state", "recovery_id")
	if labelHTTPText(t, r["profile"]) != "private-collection-recovery-v1" || labelHTTPText(t, r["state"]) != expected {
		t.Fatal("unexpected payload-free recovery state")
	}
	if string(r["recovery_id"]) == "null" {
		if expected == "inaccessible" {
			t.Fatal("inaccessible work lacks its opaque retry identity")
		}
		return ""
	}
	id := labelHTTPText(t, r["recovery_id"])
	if !validID(id) || expected == "none" {
		t.Fatal("invalid payload-free recovery identity")
	}
	return id
}

// independentClear verifies the exact minimal durable receipt and excludes hidden payload fields.
func independentClear(t *testing.T, handler http.Handler, id string) map[string]json.RawMessage {
	t.Helper()
	r := independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodPost, "/v1/collection-recovery/clear", raw(map[string]any{"recovery_id": id}), 200)
	independentRecoveryFields(t, r, "profile", "recovery_id", "cleared")
	if labelHTTPText(t, r["profile"]) != "private-collection-recovery-clear-v1" || labelHTTPText(t, r["recovery_id"]) != id || string(r["cleared"]) != "true" {
		t.Fatal("unexpected payload-free durable clear receipt")
	}
	return r
}

// TestIndependentRecoveryUnknownCommit proves an unknown clear cannot release a newer old-engine claim.
func TestIndependentRecoveryUnknownCommit(t *testing.T) {
	owner, pool, db := independentTagFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "independent-private-recovery", 1)
	if _, err := publish(t, db, w, sources[0], "private-recovery-source", "0", nil, "c1", []map[string]any{change("private-recovery-quantity", "12.34", "USD")}); err != nil {
		t.Fatal(err)
	}
	selected := selection(independentTagPage(t, db, w)["items"].([]any))
	p := independentTagPreview(t, db, w, selected, map[string]any{"kind": "uncategorized"})
	c := independentTagPrepare(t, db, w, "private-recovery-prepared", p)
	id := c["command_id"].(string)
	before := independentSourceHash(t, owner)
	handler := Handler(labelHTTPAuth{}, &Service{DB: pool})
	independentRecoveryStatus(t, handler, "available")
	independentGrant(t, db, "private-recovery-read-only", "1", []string{"read"})
	recoveryID := independentRecoveryStatus(t, handler, "inaccessible")
	if recoveryID == id || recoveryID == w || recoveryID == sources[0] {
		t.Fatal("opaque recovery identity disclosed private command or scope identity")
	}
	independentRecoveryRequest(t, handler, "synthetic-label-other", http.MethodPost, "/v1/collection-recovery/clear", raw(map[string]any{"recovery_id": recoveryID}), 404)
	independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodPost, "/v1/collection-recovery/clear", `{"recovery_id":"`+recoveryID+`","workspace_id":"`+w+`"}`, 400)
	independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodPost, "/v1/collection-recovery/clear", `{"recovery_id":"`+recoveryID+`","recovery_id":"`+recoveryID+`"}`, 400)
	// Eligibility is a current authority decision, not a capability granted by the old status result.
	independentGrant(t, db, "private-recovery-restored", "2", []string{"read", "acquire", "classify"})
	independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodPost, "/v1/collection-recovery/clear", raw(map[string]any{"recovery_id": recoveryID}), 409)
	if value(t, owner, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM banking.collection_label_command WHERE command_id=$1", id) != "prepared:false" {
		t.Fatal("restored current authority still allowed opaque destructive clearance")
	}
	independentGrant(t, db, "private-recovery-read-only-again", "3", []string{"read"})
	if independentRecoveryStatus(t, handler, "inaccessible") != recoveryID {
		t.Fatal("the same inaccessible work changed its opaque retry identity")
	}
	lost := &lostRecoveryCommit{pool: pool}
	lost.lose.Store(true)
	lostHandler := Handler(labelHTTPAuth{}, &Service{DB: lost})
	independentRecoveryRequest(t, lostHandler, "synthetic-label-owner", http.MethodPost, "/v1/collection-recovery/clear", raw(map[string]any{"recovery_id": recoveryID}), 503)
	if value(t, owner, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM banking.collection_label_command WHERE command_id=$1", id) != "cancelled:true" || independentSourceHash(t, owner) != before {
		t.Fatal("unknown clear executed labels, lost its durable outcome or rewrote original history")
	}
	newBook, err := prepareLedger(t, db, "new-book-after-private-clear", independentLedgerSetup())
	if err != nil {
		t.Fatal("exact private release did not permit a new old-engine command", err)
	}
	newBookID := newBook["command_id"].(string)
	independentRecoveryStatus(t, handler, "none")
	firstReceipt := independentClear(t, handler, recoveryID)
	if raw(independentClear(t, handler, recoveryID)) != raw(firstReceipt) || value(t, owner, "SELECT legacy_command_id::text FROM ledger.interactive_command_reservation WHERE principal_id=$1", actor) != newBookID {
		t.Fatal("old clear replay released or replaced a newer cross-engine claim")
	}
	if _, err = transitionLedger(t, db, newBookID, "cancel"); err != nil {
		t.Fatal(err)
	}
	if _, err = transitionLedger(t, db, newBookID, "acknowledge"); err != nil {
		t.Fatal(err)
	}
	exec(t, owner, "UPDATE ledger.principal SET disabled_at=clock_timestamp()WHERE principal_id=$1", actor)
	independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodGet, "/v1/collection-recovery", "", 404)
	independentRecoveryRequest(t, handler, "synthetic-label-owner", http.MethodPost, "/v1/collection-recovery/clear", raw(map[string]any{"recovery_id": recoveryID}), 404)
}

// TestIndependentRecoveryCommittedPreservation acknowledges inaccessible results without cancelling effects.
func TestIndependentRecoveryCommittedPreservation(t *testing.T) {
	owner, pool, db := independentTagFixture(t)
	w := provision(t, db, actor)
	p := independentTagPreview(t, db, w, []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": []map[string]any{independentTagDefinition(proofTagA, "Fictional private retained name", "active")}})
	c := independentTagPrepare(t, db, w, "committed-private-recovery", p)
	id := c["command_id"].(string)
	if _, err := independentV2(t, db, w, "execute", id); err != nil {
		t.Fatal(err)
	}
	commandHash := value(t, owner, "SELECT encode(sha256(convert_to((to_jsonb(c)-'acknowledged_at')::text,'UTF8')),'hex')FROM banking.collection_label_command c WHERE command_id=$1", id)
	before := independentSourceHash(t, owner)
	independentGrant(t, db, "committed-recovery-revoked", "1", []string{})
	handler := Handler(labelHTTPAuth{}, &Service{DB: pool})
	recoveryID := independentRecoveryStatus(t, handler, "inaccessible")
	independentClear(t, handler, recoveryID)
	if value(t, owner, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM banking.collection_label_command WHERE command_id=$1", id) != "committed:true" || commandHash != value(t, owner, "SELECT encode(sha256(convert_to((to_jsonb(c)-'acknowledged_at')::text,'UTF8')),'hex')FROM banking.collection_label_command c WHERE command_id=$1", id) || independentSourceHash(t, owner) != before {
		t.Fatal("inaccessible committed recovery rewrote the immutable result or original history")
	}
	independentGrant(t, db, "committed-recovery-restored", "2", []string{"read", "acquire", "classify"})
	catalog, err := independentV2(t, db, w, "catalog", nil)
	if err != nil || !reflect.DeepEqual(independentTagIDs(t, catalog["tags"]), []string{proofTagA}) {
		t.Fatal("acknowledgement erased the committed tag catalog", err)
	}
}

// unavailableRecoveryBegin simulates an unavailable delivery before a clear transaction can start.
type unavailableRecoveryBegin struct{}

func (unavailableRecoveryBegin) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("synthetic clear delivery unavailable")
}

// TestIndependentRecoveryOrdinaryAckReceipt proves ordinary acknowledgement resolves an earlier unknown clear.
func TestIndependentRecoveryOrdinaryAckReceipt(t *testing.T) {
	owner, pool, db := independentTagFixture(t)
	w := provision(t, db, actor)
	p := independentTagPreview(t, db, w, []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": []map[string]any{}})
	c := independentTagPrepare(t, db, w, "ordinary-ack-after-unknown", p)
	id := c["command_id"].(string)
	independentGrant(t, db, "ordinary-ack-revoke", "1", []string{"read"})
	handler := Handler(labelHTTPAuth{}, &Service{DB: pool})
	recoveryID := independentRecoveryStatus(t, handler, "inaccessible")
	independentRecoveryRequest(t, Handler(labelHTTPAuth{}, &Service{DB: unavailableRecoveryBegin{}}), "synthetic-label-owner", http.MethodPost, "/v1/collection-recovery/clear", raw(map[string]any{"recovery_id": recoveryID}), 503)
	independentGrant(t, db, "ordinary-ack-restore", "2", []string{"read", "acquire", "classify"})
	if _, err := independentV2(t, db, w, "cancel", id); err != nil {
		t.Fatal(err)
	}
	if _, err := independentV2(t, db, w, "acknowledge", id); err != nil {
		t.Fatal(err)
	}
	nextPreview := independentTagPreview(t, db, w, []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": []map[string]any{}})
	next := independentTagPrepare(t, db, w, "new-private-after-ordinary-ack", nextPreview)
	nextID := next["command_id"].(string)
	independentClear(t, handler, recoveryID)
	if value(t, owner, "SELECT label_command_id::text FROM ledger.interactive_command_reservation WHERE principal_id=$1", actor) != nextID || value(t, owner, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM banking.collection_label_command WHERE command_id=$1", nextID) != "prepared:false" {
		t.Fatal("old ordinary-ack receipt touched a newer private command")
	}
	if _, err := independentV2(t, db, w, "cancel", nextID); err != nil {
		t.Fatal(err)
	}
	if _, err := independentV2(t, db, w, "acknowledge", nextID); err != nil {
		t.Fatal(err)
	}
}

// TestIndependentRecoveryPrimedAdmission observes both command engines against an old absent-status snapshot.
func TestIndependentRecoveryPrimedAdmission(t *testing.T) {
	for _, engine := range []string{"private", "book"} {
		t.Run(engine, func(t *testing.T) {
			owner, _, db := independentTagFixture(t)
			w := provision(t, db, actor)
			p := independentTagPreview(t, db, w, []map[string]any{}, map[string]any{"kind": "tag_catalog", "tags": []map[string]any{}})
			runtime := connect(t, db, "synthetic_runtime")
			defer runtime.Close(context.Background())
			winner := connect(t, db, "synthetic_runtime")
			defer winner.Close(context.Background())
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			stale, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			defer stale.Rollback(context.Background())
			var count int
			if err = stale.QueryRow(ctx, "SELECT count(*)FROM pg_catalog.pg_class").Scan(&count); err != nil {
				t.Fatal(err)
			}
			tx, err := winner.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(context.Background())
			var encoded []byte
			if engine == "private" {
				err = tx.QueryRow(ctx, "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)", actor, w, raw(map[string]any{"preview_id": p["preview_id"], "prepare_key": "primed-private"})).Scan(&encoded)
			} else {
				err = tx.QueryRow(ctx, "SELECT ledger.register_command($1,'primed-book',$2,$3)", actor, evidence, independentLedgerSetup()).Scan(&encoded)
			}
			if err != nil {
				t.Fatal(err)
			}
			var prepared map[string]any
			if json.Unmarshal(encoded, &prepared) != nil {
				t.Fatal("malformed synthetic prepared command")
			}
			done := make(chan error, 1)
			go func() {
				var result []byte
				err := stale.QueryRow(ctx, "SELECT banking.collection_recovery_status($1)", actor).Scan(&result)
				if err == nil {
					err = stale.Commit(ctx)
				}
				_ = stale.Rollback(context.Background())
				done <- err
			}()
			waitBlocked(t, owner, runtime.PgConn().PID())
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			code(t, <-done, "40001")
			fresh, err := call(t, db, "synthetic_runtime", "serializable", "SELECT banking.collection_recovery_status($1)", actor)
			if err != nil || len(fresh) != 3 || fresh["profile"] != "private-collection-recovery-v1" {
				t.Fatal("fresh minimal status failed", err)
			}
			if engine == "private" {
				if fresh["state"] != "available" {
					t.Fatal("primed discovery missed the new private command")
				}
				id := independentTagCommand(t, prepared)["command_id"].(string)
				if _, err = independentV2(t, db, w, "cancel", id); err != nil {
					t.Fatal(err)
				}
				if _, err = independentV2(t, db, w, "acknowledge", id); err != nil {
					t.Fatal(err)
				}
			} else {
				if fresh["state"] != "none" || fresh["recovery_id"] != nil {
					t.Fatal("private recovery disclosed another command engine")
				}
				id := prepared["command_id"].(string)
				if _, err = transitionLedger(t, db, id, "cancel"); err != nil {
					t.Fatal(err)
				}
				if _, err = transitionLedger(t, db, id, "acknowledge"); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
