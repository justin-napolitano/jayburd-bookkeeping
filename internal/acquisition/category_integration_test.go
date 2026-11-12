//go:build integration

package acquisition

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionlabels"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// category48Runner uses a real restricted sync login and synthetic in-memory provider, without a listener.
func category48Runner(t *testing.T, db, lost string, body func(FetchRequest) []byte) (*Runner, *lossPool, *int) {
	t.Helper()
	conn := connect(t, db, "synthetic_sync")
	t.Cleanup(func() { conn.Close(context.Background()) })
	pool := &lossPool{conn: conn, lost: lost}
	calls := 0
	provider := providerFunc(func(_ context.Context, r FetchRequest) ([]byte, error) {
		if pool.active.Load() != 0 {
			t.Fatal("provider inside SQL transaction")
		}
		calls++
		return body(r), nil
	})
	credentials := credentialFunc(func(context.Context, CredentialReference) (ProviderCredentials, error) {
		if pool.active.Load() != 0 {
			t.Fatal("credentials inside SQL transaction")
		}
		return credentialsFixture(), nil
	})
	r, e := NewWithCategories(pool, provider, credentials, localEvidence(t))
	if e != nil {
		t.Fatal(e)
	}
	return r, pool, &calls
}

// category48Page changes only current category evidence, preserving exact0.10 original financial input.
func category48Page(category, cursor string, modified bool) []byte {
	b := categoryPageFixture(category)
	b = []byte(strings.Replace(string(b), `"next_cursor":"C1"`, `"next_cursor":"`+cursor+`"`, 1))
	if modified {
		b = []byte(strings.Replace(string(b), `"added":[`, `"modified":[`, 1))
		b = []byte(strings.Replace(string(b), `"modified":[],`, `"added":[],`, 1))
	}
	return b
}

// TestCategory48CommittedReplay independently proves both categorized commit-loss boundaries with actual SQL.
func TestCategory48CommittedReplay(t *testing.T) {
	for _, lost := range []string{"stage_sync_category_page", "complete_sync_categories"} {
		t.Run(lost, func(t *testing.T) {
			owner, db, stream := setup(t)
			scope := Scope{entity, book, stream["stream_id"].(string)}
			r, p, calls := category48Runner(t, db, lost, func(FetchRequest) []byte {
				return category48Page(`{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH","version":"v2"}`, "C1", false)
			})
			if _, e := r.Run(context.Background(), scope, claimID); e == nil || e.Error() != "commit_unknown" || !p.fired {
				t.Fatal("commit loss not exercised")
			}
			if out, e := r.Run(context.Background(), scope, claimID); e != nil || out.ChangedCount != "1" || *calls != 1 {
				t.Fatal("same-claim recovery failed", e)
			}
			if _, e := r.Run(context.Background(), scope, claimID); e != nil || *calls != 1 {
				t.Fatal("replay refetched", e)
			}
			for q, want := range map[string]string{
				"SELECT count(*)::text FROM banking.provider_category_revision":          "1",
				"SELECT count(*)::text FROM integration.sync_category_page":              "1",
				"SELECT count(*)::text FROM integration.sync_category_completion":        "1",
				"SELECT count(*)::text FROM banking.observation_revision":                "1",
				"SELECT trim_scale(amount)::text FROM banking.observation_revision":      "0.1",
				"SELECT provider_amount_lexeme FROM banking.observation_revision":        "0.10",
				"SELECT category->>'version' FROM banking.provider_category_revision":    "v2",
				"SELECT category->>'confidence' FROM banking.provider_category_revision": "HIGH",
				"SELECT count(*)::text FROM ledger.journal_entry":                        "0",
				"SELECT count(*)::text FROM integration.outbox_message":                  "0",
				"SELECT count(*)::text FROM banking.provider_review_exception":           "0",
				"SELECT cursor_value FROM integration.sync_stream":                       "C1",
				"SELECT count(*)::text FROM banking.provider_category_revision p JOIN integration.sync_page s USING(entity_id,book_id,set_id,page_index) WHERE p.source_page_sha256=s.body->>'sha256'": "1",
			} {
				if got := value(t, owner, q); got != want {
					t.Fatalf("fixed category replay oracle: got %s want %s", got, want)
				}
			}
			exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1", reader)
			if _, e := r.Run(context.Background(), scope, claimID); e == nil || e.Error() != "not_authorized" || *calls != 1 {
				t.Fatal("revoked receipt exposed", e)
			}
		})
	}
}

// TestCategory48MetadataOnly preserves immutable financial rows while retaining a new attributed category revision.
func TestCategory48MetadataOnly(t *testing.T) {
	owner, db, stream := setup(t)
	scope := Scope{entity, book, stream["stream_id"].(string)}
	first := true
	r, _, calls := category48Runner(t, db, "", func(FetchRequest) []byte {
		if first {
			return category48Page(`{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH","version":"v2"}`, "C1", false)
		}
		return category48Page(`{"primary":"TRAVEL","detailed":"TRAVEL_LODGING","confidence_level":"VERY_HIGH","version":"v2"}`, "C2", true)
	})
	if _, e := r.Run(context.Background(), scope, claimID); e != nil {
		t.Fatal(e)
	}
	before := value(t, owner, "SELECT jsonb_agg(to_jsonb(q) ORDER BY revision)::text FROM banking.observation_revision q")
	beforeReview := value(t, owner, "SELECT jsonb_agg(to_jsonb(r) ORDER BY review_id)::text FROM banking.review_item r")
	call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'category-change')::text", entity, book, scope.StreamID)
	first = false
	second := "00000000-0000-4000-8000-000000000085"
	if out, e := r.Run(context.Background(), scope, second); e != nil || out.ChangedCount != "0" {
		t.Fatal("category-only change became financial", e)
	}
	if value(t, owner, "SELECT jsonb_agg(to_jsonb(q) ORDER BY revision)::text FROM banking.observation_revision q") != before || value(t, owner, "SELECT jsonb_agg(to_jsonb(r) ORDER BY review_id)::text FROM banking.review_item r") != beforeReview {
		t.Fatal("category correction mutated financial/review history")
	}
	for q, want := range map[string]string{
		"SELECT source_revision::text FROM banking.source_account":                                                    "1",
		"SELECT count(*)::text FROM banking.provider_category_revision":                                               "2",
		"SELECT max(metadata_revision)::text FROM banking.provider_category_revision":                                 "2",
		"SELECT count(DISTINCT observation_revision)::text FROM banking.provider_category_revision":                   "1",
		"SELECT count(DISTINCT source_page_sha256)::text FROM banking.provider_category_revision":                     "2",
		"SELECT category->>'primary' FROM banking.provider_category_revision ORDER BY metadata_revision DESC LIMIT 1": "TRAVEL",
		"SELECT count(*)::text FROM ledger.journal_entry":                                                             "0",
		"SELECT count(*)::text FROM integration.outbox_message":                                                       "0",
		"SELECT count(*)::text FROM banking.provider_review_exception":                                                "0",
		"SELECT cursor_value FROM integration.sync_stream":                                                            "C2",
	} {
		if value(t, owner, q) != want {
			t.Fatal("fixed category-only oracle", want)
		}
	}
	if _, e := r.Run(context.Background(), scope, second); e != nil || *calls != 2 || value(t, owner, "SELECT count(*)::text FROM banking.provider_category_revision") != "2" {
		t.Fatal("metadata replay duplicated", e)
	}
}

// category48Wire claims a real attempt and derives exact synthetic wire via the reviewed decoder.
func category48Wire(t *testing.T, db string, scope Scope) (attempt, map[string]any) {
	t.Helper()
	c := connect(t, db, "synthetic_sync")
	defer c.Close(context.Background())
	repo := &SQLRepository{DB: c, Categories: true}
	a, e := repo.Claim(context.Background(), scope, claimID)
	if e != nil {
		t.Fatal(e)
	}
	current, e := repo.Context(context.Background(), scope, a)
	if e != nil {
		t.Fatal(e)
	}
	page, cat, e := plaid.DecodePageWithCategories(category48Page(`{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH","version":"v2"}`, "C1", false), current.expected())
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := json.Marshal(pageWire{PageIndex: 0, RequestCursor: current.RequestCursor, Page: page, CategoryAttribution: &cat})
	if e != nil {
		t.Fatal(e)
	}
	return a, object(t, string(encoded))
}

// TestCategory48ForgedSidecar denies malformed or mismatched metadata and rolls back the underlying financial stage.
func TestCategory48ForgedSidecar(t *testing.T) {
	for _, kind := range []string{"profile_null", "hash", "missing", "duplicate", "foreign_account", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			owner, db, stream := setup(t)
			scope := Scope{entity, book, stream["stream_id"].(string)}
			a, wire := category48Wire(t, db, scope)
			cat := wire["category_attribution"].(map[string]any)
			attrs := cat["attributions"].([]any)
			item := attrs[0].(map[string]any)
			code := "P2600"
			switch kind {
			case "profile_null":
				cat["profile"] = nil
			case "hash":
				cat["source_page_sha256"] = strings.Repeat("0", 64)
			case "missing":
				cat["attributions"] = []any{}
			case "duplicate":
				cat["attributions"] = append(attrs, item)
			case "foreign_account":
				item["account_id"] = "Foreign-A"
			case "malformed":
				item["category"].(map[string]any)["version"] = 4
				code = "P2600"
			}
			_, e := invoke(t, db, "synthetic_sync", "SELECT integration.stage_sync_category_page($1,$2,$3,$4,$5,$6)::text", scope.EntityID, scope.BookID, scope.StreamID, a.SetID, a.Fence, raw(wire))
			requireCode(t, e, code)
			if value(t, owner, "SELECT count(*)::text FROM integration.sync_page") != "0" || value(t, owner, "SELECT count(*)::text FROM integration.sync_category_page") != "0" || value(t, owner, "SELECT count(*)::text FROM banking.source_observation") != "0" {
				t.Fatal("forged sidecar partially staged/published")
			}
		})
	}
}

// TestCategory48UnavailableAndLegacy retains unknown attribution without inventing labels or rejecting financial evidence.
func TestCategory48UnavailableAndLegacy(t *testing.T) {
	cases := []struct {
		name, metadata, reason string
		legacy                 bool
	}{
		{"versionless", `{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH"}`, "version_missing", false},
		{"unknown_confidence", `{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"UNKNOWN","version":"v2"}`, "confidence_unknown", false},
		{"new_version", `{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH","version":"v9"}`, "unsupported_version", false},
		{"unknown_code", `{"primary":"FICTIONAL_NEW","detailed":"FICTIONAL_NEW_DETAIL","confidence_level":"HIGH","version":"v2"}`, "", false},
		{"low_confidence", `{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"LOW","version":"v2"}`, "", false},
		{"legacy", `{"version":4}`, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner, db, stream := setup(t)
			scope := Scope{entity, book, stream["stream_id"].(string)}
			r, _, _ := category48Runner(t, db, "", func(FetchRequest) []byte { return category48Page(tc.metadata, "C1", false) })
			if tc.legacy {
				r.categories = false
				r.repo.(*SQLRepository).Categories = false
			}
			if out, e := r.Run(context.Background(), scope, claimID); e != nil || out.ChangedCount != "1" {
				t.Fatal("valid unavailable/legacy source lost", e)
			}
			want := "1"
			if tc.legacy {
				want = "0"
			}
			if value(t, owner, "SELECT count(*)::text FROM banking.provider_category_revision") != want {
				t.Fatal("wrong metadata presence")
			}
			if !tc.legacy && value(t, owner, "SELECT category->>'unavailable_reason' FROM banking.provider_category_revision") != tc.reason {
				t.Fatal("unknown reason changed")
			}
			if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, owner, "SELECT trim_scale(amount)::text FROM banking.observation_revision") != "0.1" {
				t.Fatal("financial source changed")
			}
		})
	}
}

// Retain pgx's actual restricted connection interface at compile time.
var _ Beginner = (*pgx.Conn)(nil)

// TestCategory48MalformedPage proves enhanced decode abandonment persists only a fixed existing diagnostic.
func TestCategory48MalformedPage(t *testing.T) {
	owner, db, stream := setup(t)
	scope := Scope{entity, book, stream["stream_id"].(string)}
	r, _, calls := category48Runner(t, db, "", func(FetchRequest) []byte { return category48Page(`{"version":4}`, "C1", false) })
	if _, e := r.Run(context.Background(), scope, claimID); e == nil || e.Error() != "decoder_rejected" {
		t.Fatal("malformed page not held", e)
	}
	for q, want := range map[string]string{
		"SELECT count(*)::text FROM banking.source_observation":                "0",
		"SELECT count(*)::text FROM banking.provider_category_revision":        "0",
		"SELECT count(*)::text FROM integration.sync_page":                     "0",
		"SELECT code FROM integration.sync_decode_diagnostic":                  "invalid_field",
		"SELECT state FROM integration.sync_set":                               "abandoned",
		"SELECT coalesce(cursor_value,'initial') FROM integration.sync_stream": "initial",
	} {
		if value(t, owner, q) != want {
			t.Fatal("malformed page escaped whole-set hold")
		}
	}
	if _, e := r.Run(context.Background(), scope, claimID); e == nil || *calls != 1 {
		t.Fatal("terminal malformed claim refetched")
	}
}

// TestCategory48CatalogEquality binds SQL's complete map to the separately reviewed Go source corpus.
func TestCategory48CatalogEquality(t *testing.T) {
	owner, _, _ := setup(t)
	want := plaid.StarterCategories()
	var pairs []plaid.StarterPair
	if json.Unmarshal([]byte(value(t, owner, `SELECT jsonb_agg(jsonb_build_object('version',version,'primary',primary_code,'detailed',detailed_code)ORDER BY version,primary_code,detailed_code)::text FROM reference.analytic_category_pair`)), &pairs) != nil || !reflect.DeepEqual(pairs, want.Pairs) {
		t.Fatal("SQL map differs from reviewed231 exact versioned pairs")
	}
	var topics []plaid.StarterTopic
	if json.Unmarshal([]byte(value(t, owner, `SELECT jsonb_agg(jsonb_build_object('code',code,'name',name)ORDER BY code)::text FROM reference.analytic_category`)), &topics) != nil || !reflect.DeepEqual(topics, want.Topics) {
		t.Fatal("SQL starter labels differ")
	}
	if value(t, owner, "SELECT count(*)::text FROM reference.analytic_category WHERE map_edition=$1 AND source_sha256=$2", want.Edition, want.SourceSHA256) != "18" {
		t.Fatal("SQL source attribution differs")
	}
}

// TestCategory48StageReplayAndContext checks exact staged replay, full-wire accounting and mode-change hold.
func TestCategory48StageReplayAndContext(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "categorized"
		if legacy {
			name = "legacy_mode_hold"
		}
		t.Run(name, func(t *testing.T) {
			owner, db, stream := setup(t)
			scope := Scope{entity, book, stream["stream_id"].(string)}
			a, wire := category48Wire(t, db, scope)
			stage := "SELECT integration.stage_sync_category_page($1,$2,$3,$4,$5,$6)::text"
			if legacy {
				delete(wire, "category_attribution")
				stage = "SELECT integration.stage_sync_page($1,$2,$3,$4,$5,$6)::text"
			}
			body := raw(wire)
			result := call(t, db, "synthetic_sync", stage, entity, book, scope.StreamID, a.SetID, a.Fence, body)
			if legacy {
				_, e := invoke(t, db, "synthetic_sync", "SELECT integration.sync_category_context($1,$2,$3,$4,$5)::text", entity, book, scope.StreamID, a.SetID, a.Fence)
				requireCode(t, e, "P2602")
				if value(t, owner, "SELECT count(*)::text FROM integration.sync_category_page") != "0" {
					t.Fatal("legacy stage invented category")
				}
				return
			}
			if call(t, db, "synthetic_sync", stage, entity, book, scope.StreamID, a.SetID, a.Fence, body) != result {
				t.Fatal("staged replay changed receipt")
			}
			contextJSON := object(t, call(t, db, "synthetic_sync", "SELECT integration.sync_category_context($1,$2,$3,$4,$5)::text", entity, book, scope.StreamID, a.SetID, a.Fence))
			if contextJSON["byte_count"] != float64(len(body)) || contextJSON["page_count"] != float64(1) {
				t.Fatal("categorized context omitted full wire bytes")
			}
			cat := wire["category_attribution"].(map[string]any)
			item := cat["attributions"].([]any)[0].(map[string]any)
			item["category"].(map[string]any)["confidence"] = "VERY_HIGH"
			_, e := invoke(t, db, "synthetic_sync", stage, entity, book, scope.StreamID, a.SetID, a.Fence, raw(wire))
			requireCode(t, e, "P2602")
			complete := "SELECT integration.complete_sync_categories($1,$2,$3,$4,$5)::text"
			receipt := call(t, db, "synthetic_sync", complete, entity, book, scope.StreamID, a.SetID, a.Fence)
			if call(t, db, "synthetic_sync", complete, entity, book, scope.StreamID, a.SetID, a.Fence) != receipt || value(t, owner, "SELECT count(*)::text FROM banking.provider_category_revision") != "1" || value(t, owner, "SELECT category->>'confidence' FROM banking.provider_category_revision") != "HIGH" {
				t.Fatal("stage conflict/replayed completion changed attribution")
			}
		})
	}
}

// TestCategory48PrivateProviderFallback proves ingestion→adoption→automatic service→filtered labels without journals.
func TestCategory48PrivateProviderFallback(t *testing.T) {
	owner, db, stream := setup(t)
	scope := Scope{entity, book, stream["stream_id"].(string)}
	ingest, _, _ := category48Runner(t, db, "", func(FetchRequest) []byte {
		return category48Page(`{"primary":"FOOD_AND_DRINK","detailed":"FOOD_AND_DRINK_COFFEE","confidence_level":"HIGH","version":"v2"}`, "C1", false)
	})
	if _, e := ingest.Run(context.Background(), scope, claimID); e != nil {
		t.Fatal(e)
	}
	evidence := "00000000-0000-4000-8000-000000000084"
	workspace := object(t, call(t, db, "synthetic_operator", "SELECT banking.provision_collection_owner($1,'category-owner',$2)::text", actor, raw(map[string]any{"expected_revision": nil, "capabilities": []string{"read", "acquire", "classify"}, "evidence_ref": evidence})))["workspace_id"].(string)
	exec(t, owner, "INSERT INTO ledger.membership VALUES($1,$2,'connect',NULL)", entity, actor)
	item := value(t, owner, "SELECT sync_item_id::text FROM integration.sync_stream")
	adopted := object(t, call(t, db, "synthetic_runtime", "SELECT banking.adopt_collection_item($1,$2,$3,$4,$5,'category-adopt',$6)::text", actor, workspace, entity, book, item, raw(map[string]any{"expected_generation": "1", "sources": []map[string]any{{"source_account_id": sourceID, "display_name": "Fictional reviewed source"}}, "evidence_ref": evidence})))
	privateSource := adopted["sources"].([]any)[0].(map[string]any)["source_account_id"]
	conn := connect(t, db, "synthetic_runtime")
	defer conn.Close(context.Background())
	worker := &collectionlabels.AutomaticService{DB: conn}
	service := &collectionlabels.Service{DB: conn, Rules: collectionlabels.RuleProcessor{}}
	financialBefore := value(t, owner, "SELECT jsonb_agg(to_jsonb(q) ORDER BY revision)::text FROM banking.observation_revision q")
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal("provider prepare", e)
	}
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_automatic_decision") != "0" {
		t.Fatal("inert preparation applied")
	}
	if e := worker.Step(context.Background()); e != nil {
		t.Fatal("provider execute", e)
	}
	selector := map[string]any{"profile": "private-collection-review-v3", "source_account_id": nil, "assignment": "all", "starts_on": nil, "through_date": nil, "search": "", "topic": map[string]any{"kind": "all"}, "tag": map[string]any{"kind": "all"}, "sort": map[string]any{"field": "date", "order": "desc"}, "snapshot": nil, "cursor": nil, "limit": 100, "merchant": map[string]any{"kind": "all"}, "amount": nil, "institution": map[string]any{"kind": "all"}, "legal_owner": map[string]any{"kind": "all"}, "book": map[string]any{"kind": "all"}, "status": map[string]any{"source": "all", "journal": "all"}}
	query := func() map[string]any {
		b, e := service.Query(context.Background(), collectionlabels.Request{Actor: actor, Workspace: workspace, Version: 3, Body: raw(selector)})
		if e != nil {
			t.Fatal("provider query", e)
		}
		return object(t, string(b))
	}
	page := query()
	rows := page["items"].([]any)
	if len(rows) != 1 {
		t.Fatal("source not visible after automatic fallback")
	}
	row := rows[0].(map[string]any)
	classification := row["categorization"].(map[string]any)
	provider := classification["provider"].(map[string]any)
	topic := value(t, owner, "SELECT topic_id::text FROM reference.analytic_category WHERE code='FOOD_AND_DRINK'")
	if classification["state"] != "automatic" || classification["origin"] != "provider" || provider["name"] != "Plaid" || provider["version"] != "v2" || provider["detailed"] != "FOOD_AND_DRINK_COFFEE" || provider["confidence"] != "HIGH" || row["source_id"] != privateSource || row["owner_topic"].(map[string]any)["topic_id"] != topic {
		t.Fatal("provider attribution/default topic missing")
	}
	selector["topic"] = map[string]any{"kind": "topic", "topic_id": topic}
	filtered := query()
	if len(filtered["items"].([]any)) != 1 || filtered["matching_count"] != "1" {
		t.Fatal("whole-query category count differs from applied projection")
	}
	selector["topic"] = map[string]any{"kind": "all"}
	selected := []map[string]any{{"workspace_id": workspace, "source_id": row["source_id"], "observation_id": row["observation_id"], "observation_revision": row["observation_revision"]}}
	preview, e := service.Preview(context.Background(), collectionlabels.Request{Actor: actor, Workspace: workspace, Version: 3, Body: raw(map[string]any{"snapshot": page["snapshot"], "selection": selected, "action": map[string]any{"kind": "uncategorized"}})})
	if e != nil {
		t.Fatal("manual correction preview", e)
	}
	prepared, e := service.Prepare(context.Background(), collectionlabels.Request{Actor: actor, Workspace: workspace, Version: 3, Body: raw(map[string]any{"preview_id": object(t, string(preview))["preview_id"], "prepare_key": "category-manual"})})
	if e != nil {
		t.Fatal("manual correction prepare", e)
	}
	command := object(t, string(prepared))["command"].(map[string]any)["command_id"].(string)
	for _, op := range []string{"execute", "acknowledge"} {
		if _, e := service.Transition(context.Background(), collectionlabels.Request{Actor: actor, Workspace: workspace, Command: command, Version: 3, Body: "{}"}, op); e != nil {
			t.Fatal("manual correction transition", e)
		}
	}
	for i := 0; i < 2; i++ {
		if e := worker.Step(context.Background()); e != nil {
			t.Fatal("manual protection", e)
		}
	}
	final := query()["items"].([]any)[0].(map[string]any)
	if final["owner_topic"].(map[string]any)["state"] != "uncategorized" || final["categorization"].(map[string]any)["origin"] != "manual" {
		t.Fatal("automatic fallback overwrote explicit manual decision")
	}
	if value(t, owner, "SELECT jsonb_agg(to_jsonb(q) ORDER BY revision)::text FROM banking.observation_revision q") != financialBefore || value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, owner, "SELECT count(*)::text FROM integration.outbox_message") != "0" || value(t, owner, "SELECT count(*)::text FROM banking.review_draft") != "0" {
		t.Fatal("automatic labels changed source money or posted accounting")
	}
}
