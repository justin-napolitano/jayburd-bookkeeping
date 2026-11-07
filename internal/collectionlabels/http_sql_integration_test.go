//go:build integration

package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// labelHTTPAuth exists only in this integration-tagged binary and never trusts JSON identity.
type labelHTTPAuth struct{}

func (labelHTTPAuth) Authenticate(_ context.Context, token string) (string, error) {
	switch token {
	case "synthetic-label-owner":
		return actor, nil
	case "synthetic-label-other":
		return other, nil
	}
	return "", errors.New("synthetic identity rejected")
}

// labelHTTPFixture clones only the runner's synthetic template and removes only its own database.
func labelHTTPFixture(t *testing.T) (*pgx.Conn, *pgxpool.Pool, string) {
	t.Helper()
	if !*proof {
		t.Skip("use disposable collection label runner")
	}
	db := fmt.Sprintf("labels_http_%d", time.Now().UnixNano())
	admin := connect(t, "postgres", "postgres")
	exec(t, admin, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE labels_template")
	_ = admin.Close(context.Background())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cleanup, err := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
		if err != nil {
			t.Error("synthetic cleanup connection unavailable")
			return
		}
		defer cleanup.Close(ctx)
		if _, err = cleanup.Exec(ctx, "DROP DATABASE "+pgx.Identifier{db}.Sanitize()); err != nil {
			t.Error("owned synthetic database cleanup failed")
		}
	})
	owner := connect(t, db, "postgres")
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	exec(t, owner, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic-http-label','owner'),($2,'synthetic-http-label','other')`, actor, other)
	exec(t, owner, `INSERT INTO reference.currency VALUES($1,'USD','840',2,'2000-01-01',NULL,'synthetic-http-label'),('00000000-0000-4000-8000-000000000006','EUR',NULL,2,'2000-01-01',NULL,'synthetic-http-label')`, currency)
	pool, err := pgxpool.New(context.Background(), "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=2")
	if err != nil {
		t.Fatal("synthetic runtime pool unavailable")
	}
	t.Cleanup(pool.Close)
	return owner, pool, db
}

// labelHTTPText consumes exact strings only; it never converts financial values to float.
func labelHTTPText(t *testing.T, value json.RawMessage) string {
	t.Helper()
	var result string
	if json.Unmarshal(value, &result) != nil {
		t.Fatal("expected synthetic string")
	}
	return result
}

// labelHTTPObject decodes one JSON object while retaining every nested amount byte as raw JSON.
func labelHTTPObject(t *testing.T, value json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var result map[string]json.RawMessage
	if json.Unmarshal(value, &result) != nil || result == nil {
		t.Fatal("expected synthetic object")
	}
	return result
}

// labelHTTPRows consumes only the real bounded page/preview rows.
func labelHTTPRows(t *testing.T, value json.RawMessage) []map[string]json.RawMessage {
	t.Helper()
	var result []map[string]json.RawMessage
	if json.Unmarshal(value, &result) != nil {
		t.Fatal("expected synthetic rows")
	}
	return result
}

// labelHTTPSelection explicitly fixes all composite identities/revisions from the observed page.
func labelHTTPSelection(t *testing.T, rows []map[string]json.RawMessage) []map[string]string {
	t.Helper()
	result := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		selected := map[string]string{}
		for _, key := range []string{"workspace_id", "source_id", "observation_id", "observation_revision"} {
			selected[key] = labelHTTPText(t, row[key])
		}
		result = append(result, selected)
	}
	return result
}

// TestCollectionLabelsHTTPPostgres proves the actual handler, service and restricted SQL lifecycle.
func TestCollectionLabelsHTTPPostgres(t *testing.T) {
	owner, pool, db := labelHTTPFixture(t)
	w := provision(t, db, actor)
	_, sources := register(t, db, w, actor, "synthetic-http-label-item", 1)
	if _, err := publish(t, db, w, sources[0], "http-initial", "0", nil, "http-c1", []map[string]any{
		change("huge-usd", "9007199254740993.01", "USD"),
		change("nullable-eur", "12.34", "EUR"),
		change("precision-held", "1.001", "USD"),
		change("unsupported-held", "10.00", "XYZ"),
	}); err != nil {
		t.Fatal("synthetic source publication failed", err)
	}
	handler := Handler(labelHTTPAuth{}, &Service{DB: pool})
	base := "/v1/collections/" + w
	request := func(token, method, suffix, body string, want int) map[string]json.RawMessage {
		t.Helper()
		r := httptest.NewRequest(method, base+suffix, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		if method == http.MethodPost {
			r.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, r)
		if response.Code != want {
			t.Fatalf("synthetic HTTP%s%s status%d want%d: %s", method, suffix, response.Code, want, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "no-store" || strings.Contains(response.Body.String(), "synthetic-http-label-item") || strings.Contains(response.Body.String(), "credential_ref") {
			t.Fatal("private transport cache/custody boundary failed")
		}
		result := labelHTTPObject(t, response.Body.Bytes())
		if want != http.StatusOK {
			if !validID(labelHTTPText(t, result["request_id"])) {
				t.Fatal("missing independent correlation UUID")
			}
		}
		return result
	}
	get := func(suffix string) map[string]json.RawMessage {
		return request("synthetic-label-owner", http.MethodGet, suffix, "", 200)
	}
	post := func(suffix string, body any) map[string]json.RawMessage {
		return request("synthetic-label-owner", http.MethodPost, suffix, raw(body), 200)
	}
	commandOf := func(wrapper map[string]json.RawMessage) map[string]json.RawMessage {
		t.Helper()
		if labelHTTPText(t, wrapper["profile"]) != "private-collection-command-v1" {
			t.Fatal("wrong real command profile")
		}
		return labelHTTPObject(t, wrapper["command"])
	}
	initialCatalog := get("/taxonomy")
	if labelHTTPText(t, initialCatalog["taxonomy_revision"]) != "1" || len(labelHTTPRows(t, initialCatalog["topics"])) != 0 || len(labelHTTPRows(t, initialCatalog["sources"])) != 1 {
		t.Fatal("catalog did not expose exact empty edition/source inventory")
	}
	initialPage := post("/review/query", selector())
	rows := labelHTTPRows(t, initialPage["items"])
	if labelHTTPText(t, initialPage["matching_count"]) != "4" || len(rows) != 4 {
		t.Fatal("wrong independent original-row count")
	}
	available, held := 0, 0
	for _, row := range rows {
		money := labelHTTPObject(t, row["money"])
		if labelHTTPText(t, money["state"]) == "held" {
			held++
			if string(money["amount"]) != "null" || string(money["currency"]) != "null" {
				t.Fatal("held evidence invented usable money")
			}
			continue
		}
		available++
		switch labelHTTPText(t, money["currency"]) {
		case "USD":
			if labelHTTPText(t, money["amount"]) != "9007199254740993.01" {
				t.Fatal("huge original USD changed")
			}
		case "EUR":
			if labelHTTPText(t, money["amount"]) != "12.34" || string(money["numeric_code"]) != "null" {
				t.Fatal("original EUR or nullable numeric identifier changed")
			}
		default:
			t.Fatal("unexpected original currency")
		}
	}
	if available != 2 || held != 2 {
		t.Fatal("wrong known/held projections")
	}
	request("synthetic-label-other", http.MethodGet, "/taxonomy", "", 404)
	request("synthetic-label-other", http.MethodPost, "/review/query", raw(selector()), 404)
	request("synthetic-label-owner", http.MethodPost, "/review/query", strings.TrimSuffix(raw(selector()), "}")+`,"search":"duplicate"}`, 400)
	request("synthetic-label-owner", http.MethodPost, "/review/query", strings.TrimSuffix(raw(selector()), "}")+`,"actor":"`+other+`"}`, 400)
	request("synthetic-label-owner", http.MethodPost, "/label-commands/not-a-uuid/execute", `{}`, 400)
	request("synthetic-label-owner", http.MethodPost, "/label-commands/"+commandID+"/execute", `{"unexpected":true}`, 400)
	request("synthetic-label-owner", http.MethodPost, "/review/query", strings.Repeat("x", 8193), 413)

	// Full taxonomy changes are themselves explicitly previewed, prepared, executed and acknowledged.
	taxonomyPreview := post("/labels/preview", map[string]any{"snapshot": initialPage["snapshot"], "selection": []any{}, "action": topicsAction([]map[string]any{topic(rootTopic, "Everyday", nil, "active"), topic(childTopic, "Dining", rootTopic, "active")})})
	if labelHTTPText(t, taxonomyPreview["selected_count"]) != "0" || len(labelHTTPRows(t, taxonomyPreview["review_rows"])) != 0 {
		t.Fatal("taxonomy preview expanded observations")
	}
	taxonomyPrepareBody := map[string]any{"preview_id": labelHTTPText(t, taxonomyPreview["preview_id"]), "prepare_key": "http-taxonomy"}
	taxonomyPrepared := post("/label-commands", taxonomyPrepareBody)
	taxonomyCommand := commandOf(taxonomyPrepared)
	taxonomyID := labelHTTPText(t, taxonomyCommand["command_id"])
	if labelHTTPText(t, commandOf(get("/label-commands/pending"))["command_id"]) != taxonomyID {
		t.Fatal("pending discovery changed command identity")
	}
	taxonomyCommitted := post("/label-commands/"+taxonomyID+"/execute", map[string]any{})
	if labelHTTPText(t, commandOf(taxonomyCommitted)["state"]) != "committed" {
		t.Fatal("taxonomy did not commit")
	}
	if raw(post("/label-commands/"+taxonomyID+"/execute", map[string]any{})) != raw(taxonomyCommitted) || raw(post("/label-commands", taxonomyPrepareBody)) != raw(taxonomyCommitted) {
		t.Fatal("terminal replay changed immutable reviewed intent")
	}
	post("/label-commands/"+taxonomyID+"/acknowledge", map[string]any{})
	if string(get("/label-commands/pending")["command"]) != "null" || labelHTTPText(t, get("/taxonomy")["taxonomy_revision"]) != "2" {
		t.Fatal("taxonomy acknowledgement/catalog refresh failed")
	}
	request("synthetic-label-owner", http.MethodPost, "/labels/preview", raw(map[string]any{"snapshot": initialPage["snapshot"], "selection": labelHTTPSelection(t, rows), "action": map[string]any{"kind": "uncategorized"}}), 409)

	// One explicitly reviewed selection applies atomically; malformed/stale members produce no decisions.
	currentPage := post("/review/query", selector())
	selected := labelHTTPSelection(t, labelHTTPRows(t, currentPage["items"]))
	badSelection := labelHTTPSelection(t, labelHTTPRows(t, currentPage["items"]))
	badSelection[3]["observation_revision"] = "999"
	request("synthetic-label-owner", http.MethodPost, "/labels/preview", raw(map[string]any{"snapshot": currentPage["snapshot"], "selection": badSelection, "action": map[string]any{"kind": "topic", "topic_id": childTopic}}), 409)
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != "0" {
		t.Fatal("malformed bulk selection produced partial decisions")
	}
	labelPreview := post("/labels/preview", map[string]any{"snapshot": currentPage["snapshot"], "selection": selected, "action": map[string]any{"kind": "topic", "topic_id": childTopic}})
	if labelHTTPText(t, labelPreview["selected_count"]) != "4" || len(labelHTTPRows(t, labelPreview["review_rows"])) != 4 {
		t.Fatal("preview did not preserve the exact reviewed selection")
	}
	labelPrepared := post("/label-commands", map[string]any{"preview_id": labelHTTPText(t, labelPreview["preview_id"]), "prepare_key": "http-label-four"})
	labelID := labelHTTPText(t, commandOf(labelPrepared)["command_id"])
	labelCommitted := post("/label-commands/"+labelID+"/execute", map[string]any{})
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision WHERE command_id=$1", labelID) != "4" {
		t.Fatal("actual bulk command was not exact and atomic")
	}
	filter := selector()
	filter["topic"] = map[string]any{"kind": "topic", "topic_id": childTopic}
	if labelHTTPText(t, post("/review/query", filter)["matching_count"]) != "4" {
		t.Fatal("current topic query did not observe all four decisions")
	}
	corrected := change("huge-usd", "15.00", "USD")
	corrected["operation"] = "modified"
	if _, err := publish(t, db, w, sources[0], "http-correction", "1", "http-c1", "http-c2", []map[string]any{corrected}); err != nil {
		t.Fatal("synthetic source correction failed", err)
	}
	if raw(post("/label-commands/"+labelID+"/execute", map[string]any{})) != raw(labelCommitted) {
		t.Fatal("source correction rewrote historical terminal review")
	}
	stale := selector()
	stale["snapshot"] = currentPage["snapshot"]
	request("synthetic-label-owner", http.MethodPost, "/review/query", raw(stale), 409)
	post("/label-commands/"+labelID+"/acknowledge", map[string]any{})
	if string(get("/label-commands/pending")["command"]) != "null" {
		t.Fatal("acknowledgement stranded completed label command")
	}
	// A post-prepare correction prevents every selected label, yet still permits explicit cancellation.
	beforeStale := post("/review/query", selector())
	stalePreview := post("/labels/preview", map[string]any{"snapshot": beforeStale["snapshot"], "selection": labelHTTPSelection(t, labelHTTPRows(t, beforeStale["items"])), "action": map[string]any{"kind": "uncategorized"}})
	stalePrepared := post("/label-commands", map[string]any{"preview_id": labelHTTPText(t, stalePreview["preview_id"]), "prepare_key": "http-stale-source"})
	staleID := labelHTTPText(t, commandOf(stalePrepared)["command_id"])
	correctedEUR := change("nullable-eur", "9.99", "EUR")
	correctedEUR["operation"] = "modified"
	if _, err := publish(t, db, w, sources[0], "http-correction-eur", "2", "http-c2", "http-c3", []map[string]any{correctedEUR}); err != nil {
		t.Fatal("synthetic source correction failed", err)
	}
	request("synthetic-label-owner", http.MethodPost, "/label-commands/"+staleID+"/execute", `{}`, 409)
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_label_decision") != "4" || labelHTTPText(t, commandOf(get("/label-commands/pending"))["state"]) != "prepared" {
		t.Fatal("stale bulk execute partially applied or lost durable prepared intent")
	}
	cancelled := post("/label-commands/"+staleID+"/cancel", map[string]any{})
	if labelHTTPText(t, commandOf(cancelled)["state"]) != "cancelled" {
		t.Fatal("stale preview prevented explicit cancellation")
	}
	post("/label-commands/"+staleID+"/acknowledge", map[string]any{})
	if value(t, owner, "SELECT((SELECT count(*)FROM ledger.book)+(SELECT count(*)FROM ledger.journal_entry)+(SELECT count(*)FROM integration.outbox_message))::text") != "0" || value(t, owner, "SELECT count(*)::text FROM banking.collection_revision") != "6" {
		t.Fatal("analytic commands changed accounting or original source history")
	}
	// Real current-authority denial also applies to immutable terminal replay.
	_, err := call(t, db, "synthetic_operator", "serializable", "SELECT banking.provision_collection_owner($1,'http-revoke',$2)", actor, raw(map[string]any{"expected_revision": "1", "capabilities": []string{}, "evidence_ref": evidence}))
	if err != nil {
		t.Fatal("synthetic private grant revocation failed", err)
	}
	request("synthetic-label-owner", http.MethodGet, "/taxonomy", "", 404)
	request("synthetic-label-owner", http.MethodPost, "/label-commands/"+labelID+"/execute", `{}`, 404)
}

// TestCollectionLabelsHTTPResultCapPostgres injects a wrong SQL result only in its disposable clone.
// It proves the real restricted connection/adapter fails closed before exposing or committing that result.
func TestCollectionLabelsHTTPResultCapPostgres(t *testing.T) {
	owner, pool, db := labelHTTPFixture(t)
	w := provision(t, db, actor)
	// Replacing this one read fixture is explicit fault injection, not a production SQL/profile assertion.
	exec(t, owner, `CREATE OR REPLACE FUNCTION banking.get_collection_taxonomy(actor uuid,w uuid)RETURNS jsonb LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$SELECT jsonb_build_object('profile','private-collection-catalog-v1','workspace_id',w,'padding',repeat('x',1048576))$$`)
	r := httptest.NewRequest(http.MethodGet, "/v1/collections/"+w+"/taxonomy", nil)
	r.Header.Set("Authorization", "Bearer synthetic-label-owner")
	response := httptest.NewRecorder()
	Handler(labelHTTPAuth{}, &Service{DB: pool}).ServeHTTP(response, r)
	body := labelHTTPObject(t, response.Body.Bytes())
	if response.Code != 503 || labelHTTPText(t, body["code"]) != "temporarily_unavailable" || len(response.Body.Bytes()) > 1024 || strings.Contains(response.Body.String(), "padding") {
		t.Fatal("oversized SQL response escaped the fixed503 envelope")
	}
	if !validID(labelHTTPText(t, body["request_id"])) || response.Header().Get("Retry-After") != "1" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("oversized response lost safe failure framing")
	}
	// The failed read released the only acquired transaction and did not create accounting or commands.
	if value(t, owner, "SELECT((SELECT count(*)FROM banking.collection_label_command)+(SELECT count(*)FROM ledger.journal_entry))::text") != "0" {
		t.Fatal("result-fence failure created durable effects")
	}
}

// TestCollectionLabelsUpgradeAcknowledgementRace proves old snapshots cannot strand a backfilled claim.
func TestCollectionLabelsUpgradeAcknowledgementRace(t *testing.T) {
	owner, _, db := labelHTTPFixture(t)
	// Only source artifacts copied by the disposable runner are read; no inherited path or DSN.
	down, err := os.ReadFile("/proof/migrations/000043_collection_labels.down.sql")
	if err != nil {
		t.Fatal("synthetic migration source unavailable")
	}
	up, err := os.ReadFile("/proof/migrations/000043_collection_labels.up.sql")
	if err != nil {
		t.Fatal("synthetic migration source unavailable")
	}
	exec(t, owner, string(down))
	const legacyID = "00000000-0000-4000-8000-000000000901"
	// A fictional already-reviewed42-era setup intent needs no invented book or journal.
	exec(t, owner, `INSERT INTO ledger.interactive_command(command_id,principal_id,registration_key,operation,request_body)VALUES($1,$2,'http-upgrade-legacy','setup','{}')`, legacyID, actor)
	if _, err = call(t, db, "synthetic_runtime", "serializable", "SELECT ledger.transition_command($1,$2,$3,'cancel','{}')", actor, legacyID, evidence); err != nil {
		t.Fatal("synthetic legacy cancellation failed", err)
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stale, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer stale.Rollback(context.Background())
	var count int
	if err = stale.QueryRow(ctx, "SELECT count(*)FROM pg_catalog.pg_class").Scan(&count); err != nil {
		t.Fatal("synthetic old snapshot unavailable", err)
	}
	// Strip only the script's outer transaction so the real backfill remains uncommitted during ACK.
	source := string(up)
	body := strings.Replace(source, "\nBEGIN;\n", "\n", 1)
	trimmed := strings.TrimSpace(body)
	if body == source || !strings.HasSuffix(trimmed, "COMMIT;") {
		t.Fatal("unexpected synthetic migration transaction framing")
	}
	body = strings.TrimSuffix(trimmed, "COMMIT;")
	migration, err := owner.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer migration.Rollback(context.Background())
	if _, err = migration.Exec(ctx, body); err != nil {
		t.Fatal("synthetic43 upgrade failed", err)
	}
	done := make(chan error, 1)
	go func() {
		var result []byte
		err := stale.QueryRow(ctx, "SELECT ledger.transition_command($1,$2,$3,'acknowledge','{}')", actor, legacyID, evidence).Scan(&result)
		if err == nil {
			err = stale.Commit(ctx)
		}
		_ = stale.Rollback(context.Background())
		done <- err
	}()
	waitBlocked(t, owner, runtime.PgConn().PID())
	if err = migration.Commit(ctx); err != nil {
		t.Fatal("synthetic43 upgrade commit failed", err)
	}
	code(t, <-done, "40001")
	if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation WHERE principal_id=$1 AND legacy_command_id=$2", actor, legacyID) != "1" || value(t, owner, "SELECT (acknowledged_at IS NULL)::text FROM ledger.interactive_command WHERE command_id=$1", legacyID) != "true" {
		t.Fatal("stale acknowledgement released or stranded a partially acknowledged claim")
	}
	if _, err = call(t, db, "synthetic_runtime", "serializable", "SELECT ledger.transition_command($1,$2,$3,'acknowledge','{}')", actor, legacyID, evidence); err != nil {
		t.Fatal("fresh legacy acknowledgement failed", err)
	}
	if value(t, owner, "SELECT count(*)::text FROM ledger.interactive_command_reservation") != "0" || value(t, owner, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM ledger.interactive_command WHERE command_id=$1", legacyID) != "cancelled:true" || value(t, owner, "SELECT count(*)::text FROM audit.command_action WHERE command_id=$1 AND action='acknowledge'", legacyID) != "1" {
		t.Fatal("fresh acknowledgement failed exact release or changed historical command")
	}
	if value(t, owner, "SELECT((SELECT count(*)FROM ledger.book)+(SELECT count(*)FROM ledger.journal_entry))::text") != "0" {
		t.Fatal("upgrade recovery fabricated accounting effects")
	}
}
