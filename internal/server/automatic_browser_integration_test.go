//go:build integration

package server

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionlabels"
)

var appProofAutomatic = flag.Bool("app-proof-automatic", false, "exclusive fictional automatic category browser fixture")

const automaticBrowserActor = "00000000-0000-4000-8000-000000000099"
const automaticBrowserForeign = "00000000-0000-4000-8000-000000000098"
const automaticBrowserFood = "00000000-0000-4000-8000-000000000991"
const automaticBrowserTravel = "00000000-0000-4000-8000-000000000992"

// automaticBrowserAuth exists only in the synthetic integration binary, never application auth.
type automaticBrowserAuth struct{}

func (automaticBrowserAuth) Authenticate(_ context.Context, token string) (string, error) {
	if token == "synthetic.automatic.proof" {
		return automaticBrowserActor, nil
	}
	if token == "synthetic.automatic.other" {
		return automaticBrowserForeign, nil
	}
	return "", errors.New("synthetic automatic identity denied")
}

func automaticBrowserSelector() string {
	return `{"profile":"private-collection-review-v3","source_account_id":null,"assignment":"all","starts_on":null,"through_date":null,"search":"","topic":{"kind":"all"},"tag":{"kind":"all"},"merchant":{"kind":"all"},"amount":null,"institution":{"kind":"all"},"legal_owner":{"kind":"all"},"book":{"kind":"all"},"status":{"source":"all","journal":"all"},"sort":{"field":"date","order":"desc"},"snapshot":null,"cursor":null,"limit":25}`
}

// automaticBrowserPage reads the same actual service used by the browser; no matcher/output is duplicated.
func automaticBrowserPage(t *testing.T, ctx context.Context, s *collectionlabels.Service, w string) map[string]any {
	t.Helper()
	raw, e := s.Query(ctx, collectionlabels.Request{Actor: automaticBrowserActor, Workspace: w, Version: 3, Body: automaticBrowserSelector()})
	if e != nil {
		t.Fatal("automatic fixture query", e)
	}
	var p map[string]any
	if json.Unmarshal(raw, &p) != nil {
		t.Fatal("automatic fixture page shape")
	}
	return p
}

// automaticBrowserCommand seeds explicitly reviewed manual choices through actual durable APIs.
func automaticBrowserCommand(t *testing.T, ctx context.Context, s *collectionlabels.Service, w, key string, selection []any, action any) {
	t.Helper()
	page := automaticBrowserPage(t, ctx, s, w)
	body, _ := json.Marshal(map[string]any{"snapshot": page["snapshot"], "selection": selection, "action": action})
	r := collectionlabels.Request{Actor: automaticBrowserActor, Workspace: w, Version: 3, Body: string(body)}
	p, e := s.Preview(ctx, r)
	if e != nil {
		t.Fatal("automatic fixture review", e)
	}
	var preview map[string]json.RawMessage
	if json.Unmarshal(p, &preview) != nil {
		t.Fatal("automatic fixture preview")
	}
	body, _ = json.Marshal(map[string]any{"preview_id": preview["preview_id"], "prepare_key": key})
	r.Body = string(body)
	c, e := s.Prepare(ctx, r)
	if e != nil {
		t.Fatal("automatic fixture prepare", e)
	}
	var prepared struct {
		Command struct {
			ID string `json:"command_id"`
		} `json:"command"`
	}
	if json.Unmarshal(c, &prepared) != nil || prepared.Command.ID == "" {
		t.Fatal("automatic fixture command")
	}
	r.Command = prepared.Command.ID
	r.Body = "{}"
	for _, op := range []string{"execute", "acknowledge"} {
		if _, e = s.Transition(ctx, r, op); e != nil {
			t.Fatal("automatic fixture transition", op, e)
		}
	}
}

// bootstrapAutomaticBrowser preserves a manual choice while the real background runner later handles routine facts.
// All three original facts are independently fixed; no provider or financial posting is involved.
func bootstrapAutomaticBrowser(t *testing.T, ctx context.Context, admin *pgx.Conn, runtime *pgxpool.Pool) string {
	t.Helper()
	if _, e := admin.Exec(ctx, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'https://synthetic.example.invalid','synthetic-automatic-browser'),($2,'https://synthetic.example.invalid','synthetic-automatic-other')`, automaticBrowserActor, automaticBrowserForeign); e != nil {
		t.Fatal(e)
	}
	grant := labelBrowserCall(t, ctx, admin, true, "SELECT banking.provision_collection_owner($1,'automatic-browser-grant',$2)", automaticBrowserActor, `{"expected_revision":null,"capabilities":["read","acquire","classify"],"evidence_ref":"00000000-0000-4000-8000-000000000993"}`)
	var w string
	if json.Unmarshal(grant["workspace_id"], &w) != nil {
		t.Fatal("automatic fixture workspace")
	}
	registered := labelBrowserCall(t, ctx, runtime, false, "SELECT banking.register_collection_item($1,$2,'automatic-browser-register',$3)", automaticBrowserActor, w, `{"provider":"plaid","environment":"sandbox","item_id":"SYNTHETIC-AUTOMATIC-BROWSER","credential_ref":"00000000-0000-4000-8000-000000000994","credential_version":"1","consent_evidence_ref":"00000000-0000-4000-8000-000000000993","consent_expires_at":"2099-01-01T00:00:00Z","accounts":[{"account_id":"SYNTHETIC-AUTOMATIC-USD","display_name":"Fictional automatic dollar account"}]}`)
	var sources []struct {
		ID string `json:"source_account_id"`
	}
	if json.Unmarshal(registered["sources"], &sources) != nil || len(sources) != 1 {
		t.Fatal("automatic fixture source")
	}
	changes := []map[string]any{}
	for _, row := range []struct{ id, date, description, merchant, amount string }{{"SYNTHETIC-AUTO-PROTECTED", "2026-10-03", "Fictional automatic protected choice", "Fictional Automatic Cafe", "0.20"}, {"SYNTHETIC-AUTO-CAFE", "2026-10-02", "Fictional automatic cafe purchase", "Fictional Automatic Cafe", "0.10"}, {"SYNTHETIC-AUTO-UNKNOWN", "2026-10-01", "Fictional automatic unknown purchase", "Fictional Unknown Shop", "0.30"}} {
		changes = append(changes, map[string]any{"operation": "added", "transaction_id": row.id, "booked_date": row.date, "description": row.description, "merchant_name": row.merchant, "source_state": "booked", "currency_code": "USD", "amount": row.amount, "direction": "outflow"})
	}
	body, _ := json.Marshal(map[string]any{"expected_item_generation": "1", "expected_source_revision": "0", "starting_cursor": nil, "next_cursor": "synthetic-automatic-complete", "readiness": "HISTORICAL_UPDATE_COMPLETE", "changes": changes})
	labelBrowserCall(t, ctx, runtime, false, "SELECT banking.publish_collection_batch($1,$2,$3,'automatic-browser-publish',$4)", automaticBrowserActor, w, sources[0].ID, string(body))
	s := &collectionlabels.Service{DB: runtime, Rules: collectionlabels.RuleProcessor{}, AutomaticCategories: true}
	automaticBrowserCommand(t, ctx, s, w, "automatic-browser-topics", []any{}, map[string]any{"kind": "taxonomy", "topics": []map[string]any{{"topic_id": automaticBrowserFood, "name": "Fictional Automatic Food", "parent_id": nil, "status": "active"}, {"topic_id": automaticBrowserTravel, "name": "Fictional Automatic Travel", "parent_id": nil, "status": "active"}}})
	page := automaticBrowserPage(t, ctx, s, w)
	var selected []any
	for _, x := range page["items"].([]any) {
		row := x.(map[string]any)
		if row["description"] == "Fictional automatic protected choice" {
			selected = append(selected, map[string]any{"workspace_id": w, "source_id": row["source_id"], "observation_id": row["observation_id"], "observation_revision": row["observation_revision"]})
		}
	}
	if len(selected) != 1 {
		t.Fatal("automatic protected source missing")
	}
	automaticBrowserCommand(t, ctx, s, w, "automatic-browser-manual", selected, map[string]any{"kind": "topic", "topic_id": automaticBrowserTravel})
	automaticBrowserCommand(t, ctx, s, w, "automatic-browser-rules", []any{}, map[string]any{"kind": "rule_catalog", "rules": []map[string]any{{"rule_id": "00000000-0000-4000-8000-000000000995", "name": "Fictional Automatic Cafe rule", "status": "active", "source_account_id": nil, "priority": 0, "conditions": []map[string]string{{"field": "merchant", "operator": "exact", "value": "Fictional Automatic Cafe"}}, "output": map[string]any{"kind": "topic", "topic_id": automaticBrowserFood, "tag_ids": []string{}}}}})
	return w
}

// TestAuto48BrowserBootstrap catches exact composed-fixture SQL/type errors without binding a host port.
func TestAuto48BrowserBootstrap(t *testing.T) {
	if !*automaticServerProof {
		t.Skip("explicit disposable automatic server proof only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	control, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer control.Close(context.Background())
	db := fmt.Sprintf("automatic_browser_%d", time.Now().UnixNano())
	if _, e = control.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE rules_template"); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := control.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{db}.Sanitize()); e != nil {
			t.Error("automatic browser clone cleanup", e)
		}
	}()
	admin, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname="+db+" sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(context.Background())
	if _, e = admin.Exec(ctx, `INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000002','USD','840',2,'2000-01-01',NULL,'synthetic-automatic-browser')`); e != nil {
		t.Fatal(e)
	}
	pool, e := pgxpool.New(ctx, "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=3")
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	w := bootstrapAutomaticBrowser(t, ctx, admin, pool)
	worker := &collectionlabels.AutomaticService{DB: pool}
	for i := 0; i < 2; i++ {
		if e = worker.Step(ctx); e != nil {
			t.Fatal(e)
		}
	}
	page := automaticBrowserPage(t, ctx, &collectionlabels.Service{DB: pool, Rules: collectionlabels.RuleProcessor{}, AutomaticCategories: true}, w)
	if page["matching_count"] != "3" || len(page["items"].([]any)) != 3 {
		t.Fatal("independent three-row fixture count changed")
	}
	for _, x := range page["items"].([]any) {
		row := x.(map[string]any)
		c := row["categorization"].(map[string]any)
		topic := row["owner_topic"].(map[string]any)
		switch row["description"] {
		case "Fictional automatic cafe purchase":
			if c["state"] != "automatic" || c["origin"] != "rule" || topic["topic_id"] != automaticBrowserFood {
				t.Fatal("routine classification missing")
			}
		case "Fictional automatic protected choice":
			if c["origin"] != "manual" || topic["topic_id"] != automaticBrowserTravel {
				t.Fatal("owner choice changed")
			}
		case "Fictional automatic unknown purchase":
			if c["state"] != "needs_attention" {
				t.Fatal("unknown hidden")
			}
		default:
			t.Fatal("unexpected source")
		}
	}
}
