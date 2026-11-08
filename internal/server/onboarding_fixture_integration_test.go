//go:build integration

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionlabels"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectiononboarding"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionread"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// TestOnboardingBrowserFixturePostgres exercises the exact app bootstrap before any
// listener/browser is needed. Its fixed internal host and explicit proof flag
// cannot consume an inherited managed DSN or enroll a real principal.
func TestOnboardingBrowserFixturePostgres(t *testing.T) {
	if !*appProofOnboarding {
		t.Skip("explicit disposable onboarding proof only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	control, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer control.Close(context.Background())
	db := fmt.Sprintf("onboarding_fixture_%d", time.Now().UnixNano())
	if _, e = control.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE labels_template"); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := control.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{db}.Sanitize()); e != nil {
			t.Error("owned fixture cleanup failed", e)
		}
	}()
	admin, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname="+db+" sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(context.Background())
	if _, e = admin.Exec(ctx, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('00000000-0000-4000-8000-000000000001','https://synthetic.example.invalid','synthetic-browser');INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000002','USD','840',2,'2000-01-01',NULL,'synthetic-browser-fixture')`); e != nil {
		t.Fatal(e)
	}
	pool, e := pgxpool.New(ctx, "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=4")
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	bootstrapOnboardingBrowser(t, ctx, admin, pool)
	ready := func(ctx context.Context) error {
		if e := runtimeReady(ctx, pool); e != nil {
			return e
		}
		if e := collectionReady(ctx, pool); e != nil {
			return e
		}
		return onboardingReady(ctx, pool)
	}
	if e = ready(ctx); e != nil {
		t.Fatal("actual runtime grant gates", e)
	}
	app, e := New(Config{Mode: "managed", Listen: "0.0.0.0:8080", Origin: "http://localhost:39083", Project: "jayburd-bookkeeping-dev", DatabaseURL: "synthetic-injected-test-pool", CollectionEnabled: true}, Dependencies{Auth: onboardingBrowserAuth{}, Operations: &ledger.Service{DB: pool}, Collection: &collectionread.Service{DB: pool}, Labels: &collectionlabels.Service{DB: pool, Rules: collectionlabels.RuleProcessor{}}, Onboarding: &collectiononboarding.Service{DB: pool}, Ready: ready})
	if e != nil {
		t.Fatal("actual server composition", e)
	}
	request := func(method, path string, body any, status int) map[string]any {
		t.Helper()
		text := ""
		if body != nil {
			b, e := json.Marshal(body)
			if e != nil {
				t.Fatal(e)
			}
			text = string(b)
		}
		r := httptest.NewRequest(method, path, strings.NewReader(text))
		r.Header.Set("Authorization", "Bearer synthetic.onboarding.proof")
		r.Header.Set("Origin", "http://localhost:39083")
		r.Header.Set("Sec-Fetch-Site", "same-origin")
		if method == http.MethodPost {
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		app.HTTP.Handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s wanted%d got%d: %s", path, status, w.Code, w.Body.String())
		}
		var out map[string]any
		if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	hash := func() string {
		t.Helper()
		var s string
		e := admin.QueryRow(ctx, `SELECT encode(sha256(convert_to(jsonb_build_array((SELECT jsonb_agg(to_jsonb(x)ORDER BY observation_id)FROM banking.source_observation x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY observation_id,revision)FROM banking.observation_revision x),(SELECT jsonb_agg(to_jsonb(x)ORDER BY stream_id)FROM integration.sync_stream x))::text,'UTF8')),'hex')`).Scan(&s)
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	before := hash()
	const prefix = "/v1/collection-onboarding"
	if request(http.MethodGet, prefix, nil, 200)["can_setup"] != true {
		t.Fatal("fixture did not start absent")
	}
	// Exercise cancellation and acknowledgement without changing the absent state.
	p := request(http.MethodPost, prefix+"/previews", map[string]any{"operation": "setup"}, 200)
	c := request(http.MethodPost, prefix+"/commands", map[string]any{"preview_id": p["preview_id"], "prepare_key": "fixture-cancel"}, 201)
	path := prefix + "/commands/" + c["command_id"].(string)
	request(http.MethodPost, path+"/cancel", map[string]any{}, 200)
	request(http.MethodPost, path+"/acknowledge", map[string]any{}, 200)
	run := func(body any, key string) map[string]any {
		t.Helper()
		p := request(http.MethodPost, prefix+"/previews", body, 200)
		c := request(http.MethodPost, prefix+"/commands", map[string]any{"preview_id": p["preview_id"], "prepare_key": key}, 201)
		pending := request(http.MethodGet, prefix+"/commands/pending", nil, 200)
		if pending["state"] != "available" || pending["recovery_id"] != c["recovery_id"] {
			t.Fatal("fixture current pending identity mismatch")
		}
		path := prefix + "/commands/" + c["command_id"].(string)
		c = request(http.MethodPost, path+"/execute", map[string]any{}, 200)
		request(http.MethodPost, path+"/acknowledge", map[string]any{}, 200)
		return c
	}
	setup := run(map[string]any{"operation": "setup"}, "fixture-setup")
	workspace := setup["result"].(map[string]any)["workspace_id"].(string)
	catalog := request(http.MethodGet, prefix, nil, 200)
	if catalog["can_adopt"] != true || len(catalog["candidates"].([]any)) != 1 {
		t.Fatal("exact bootstrap candidate missing")
	}
	candidate := catalog["candidates"].([]any)[0].(map[string]any)
	run(map[string]any{"operation": "adopt", "candidate_id": candidate["candidate_id"], "snapshot": candidate["snapshot"]}, "fixture-adopt")
	request(http.MethodGet, "/v1/collection", nil, 200)
	page := request(http.MethodPost, "/v3/collections/"+workspace+"/review/query", map[string]any{"profile": "private-collection-review-v3", "source_account_id": nil, "assignment": "all", "starts_on": nil, "through_date": nil, "search": "", "topic": map[string]any{"kind": "all"}, "tag": map[string]any{"kind": "all"}, "sort": map[string]any{"field": "date", "order": "desc"}, "snapshot": nil, "cursor": nil, "limit": 25, "merchant": map[string]any{"kind": "all"}, "amount": nil, "institution": map[string]any{"kind": "all"}, "legal_owner": map[string]any{"kind": "all"}, "book": map[string]any{"kind": "all"}, "status": map[string]any{"source": "all", "journal": "all"}}, 200)
	if page["matching_count"] != "2" || len(page["items"].([]any)) != 2 {
		t.Fatal("exact bootstrap old facts not readable")
	}
	amounts := map[string]bool{}
	for _, v := range page["items"].([]any) {
		r := v.(map[string]any)
		amounts[r["money"].(map[string]any)["amount"].(string)] = true
	}
	if !amounts["0.1"] || !amounts["9007199254740993.01"] || hash() != before {
		t.Fatal("original source precision/history changed")
	}
	var journals, native, claims int
	if e = admin.QueryRow(ctx, `SELECT (SELECT count(*) FROM ledger.journal_entry),(SELECT count(*) FROM banking.collection_observation),(SELECT count(*) FROM ledger.interactive_command_reservation)`).Scan(&journals, &native, &claims); e != nil || journals != 0 || native != 0 || claims != 0 {
		t.Fatal("setup/adoption created journal/native copy or left pending claim", e)
	}
	if pool.Stat().AcquiredConns() != 0 {
		t.Fatal("composed server retained connection")
	}
}
