//go:build integration

package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ruleBrowserActor = "00000000-0000-4000-8000-000000000093"
const ruleBrowserFood = "00000000-0000-4000-8000-000000000931"
const ruleBrowserTravel = "00000000-0000-4000-8000-000000000932"

// bootstrapRuleBrowser adds one isolated fictional owner through the existing
// restricted provisioning/publication/command entrypoints. No rule matcher or
// HTTP response is replaced, and actor1's original manual fixture is untouched.
func bootstrapRuleBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, runtime *pgxpool.Pool) {
	t.Helper()
	if _, err := admin.Exec(ctx, "INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'https://synthetic.example.invalid','synthetic-rule-browser')", ruleBrowserActor); err != nil {
		t.Fatal("synthetic rule principal unavailable", err)
	}
	grant := labelBrowserCall(t, ctx, admin, true, "SELECT banking.provision_collection_owner($1,'rule-browser-grant',$2)", ruleBrowserActor, `{"expected_revision":null,"capabilities":["read","acquire","classify"],"evidence_ref":"00000000-0000-4000-8000-000000000933"}`)
	var w string
	if json.Unmarshal(grant["workspace_id"], &w) != nil {
		t.Fatal("synthetic rule workspace malformed")
	}
	registered := labelBrowserCall(t, ctx, runtime, false, "SELECT banking.register_collection_item($1,$2,'rule-browser-register',$3)", ruleBrowserActor, w, `{"provider":"plaid","environment":"sandbox","item_id":"SYNTHETIC-RULE-BROWSER","credential_ref":"00000000-0000-4000-8000-000000000934","credential_version":"1","consent_evidence_ref":"00000000-0000-4000-8000-000000000933","consent_expires_at":"2099-01-01T00:00:00Z","accounts":[{"account_id":"SYNTHETIC-RULE-USD","display_name":"Fictional rules dollar account"}]}`)
	var sources []struct {
		ID string `json:"source_account_id"`
	}
	if json.Unmarshal(registered["sources"], &sources) != nil || len(sources) != 1 {
		t.Fatal("synthetic rule source malformed")
	}
	publish := map[string]any{"expected_item_generation": "1", "expected_source_revision": "0", "starting_cursor": nil, "next_cursor": "synthetic-rules-complete", "readiness": "HISTORICAL_UPDATE_COMPLETE", "changes": []map[string]any{
		{"operation": "added", "transaction_id": "SYNTHETIC-RULE-PROTECTED", "booked_date": "2026-10-03", "description": "Fictional rule protected choice", "merchant_name": "Fictional Rules Cafe", "source_state": "booked", "currency_code": "USD", "amount": "0.20", "direction": "outflow"},
		{"operation": "added", "transaction_id": "SYNTHETIC-RULE-CAFE", "booked_date": "2026-10-02", "description": "Fictional rule cafe purchase", "merchant_name": "Fictional Rules Cafe", "source_state": "booked", "currency_code": "USD", "amount": "0.10", "direction": "outflow"},
		{"operation": "added", "transaction_id": "SYNTHETIC-RULE-HUGE", "booked_date": "2026-10-01", "description": "Fictional rule exact large quantity", "merchant_name": "Fictional Rules Mega", "source_state": "booked", "currency_code": "USD", "amount": "9007199254740993.01", "direction": "outflow"},
	}}
	body, _ := json.Marshal(publish)
	labelBrowserCall(t, ctx, runtime, false, "SELECT banking.publish_collection_batch($1,$2,$3,'rule-browser-publish',$4)", ruleBrowserActor, w, sources[0].ID, string(body))
	// Seed topics and one explicit manual choice through ordinary durable commands.
	// The two rule commands under test must be authored by the real browser later.
	page := ruleBrowserPage(t, ctx, runtime, w)
	ruleBrowserCommand(t, ctx, runtime, w, page["snapshot"], []any{}, map[string]any{"kind": "taxonomy", "topics": []map[string]any{{"topic_id": ruleBrowserFood, "name": "Fictional Rule Food", "parent_id": nil, "status": "active"}, {"topic_id": ruleBrowserTravel, "name": "Fictional Rule Travel", "parent_id": nil, "status": "active"}}}, "rule-browser-topics")
	page = ruleBrowserPage(t, ctx, runtime, w)
	var rows []map[string]json.RawMessage
	if json.Unmarshal(page["items"], &rows) != nil || len(rows) != 3 {
		t.Fatal("synthetic rule source rows malformed")
	}
	var selected map[string]json.RawMessage
	for _, row := range rows {
		var desc string
		_ = json.Unmarshal(row["description"], &desc)
		if desc == "Fictional rule protected choice" {
			selected = map[string]json.RawMessage{"workspace_id": row["workspace_id"], "source_id": row["source_id"], "observation_id": row["observation_id"], "observation_revision": row["observation_revision"]}
		}
	}
	if selected == nil {
		t.Fatal("synthetic protected source missing")
	}
	ruleBrowserCommand(t, ctx, runtime, w, page["snapshot"], []any{selected}, map[string]any{"kind": "topic", "topic_id": ruleBrowserTravel}, "rule-browser-manual")
}

func ruleBrowserPage(t *testing.T, ctx context.Context, pool *pgxpool.Pool, w string) map[string]json.RawMessage {
	t.Helper()
	return labelBrowserCall(t, ctx, pool, false, "SELECT banking.query_collection_review_v2($1,$2,$3)", ruleBrowserActor, w, `{"profile":"private-collection-review-v2","source_account_id":null,"assignment":"all","starts_on":null,"through_date":null,"search":"","topic":{"kind":"all"},"tag":{"kind":"all"},"sort":"date_desc","snapshot":null,"cursor":null,"limit":25}`)
}

func ruleBrowserCommand(t *testing.T, ctx context.Context, pool *pgxpool.Pool, w string, snapshot json.RawMessage, selection []any, action map[string]any, key string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"snapshot": snapshot, "selection": selection, "action": action})
	preview := labelBrowserCall(t, ctx, pool, false, "SELECT banking.preview_collection_labels_v2($1,$2,$3)", ruleBrowserActor, w, string(body))
	body, _ = json.Marshal(map[string]any{"preview_id": preview["preview_id"], "prepare_key": key})
	prepared := labelBrowserCall(t, ctx, pool, false, "SELECT banking.prepare_collection_label_command_v2($1,$2,$3)", ruleBrowserActor, w, string(body))
	var command struct {
		ID string `json:"command_id"`
	}
	if json.Unmarshal(prepared["command"], &command) != nil || command.ID == "" {
		t.Fatal("synthetic rule seed command malformed")
	}
	for _, operation := range []string{"execute", "acknowledge"} {
		labelBrowserCall(t, ctx, pool, false, "SELECT banking.transition_collection_label_command_v2($1,$2,$3,$4,'{}')", ruleBrowserActor, w, command.ID, operation)
	}
}
