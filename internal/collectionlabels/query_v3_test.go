package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// v3Reader exercises the public port without pretending synthetic authentication proves SQL authority.
type v3Reader struct{ fakeOperations }

func (p *v3Reader) Detail(_ context.Context, r Request) (json.RawMessage, error) {
	return p.call(r, "detail")
}

type rejectedRuleHook struct{ err error }

func (h rejectedRuleHook) Process(context.Context, pgx.Tx, Request, string) (json.RawMessage, bool, error) {
	return nil, false, h.err
}

// A definite regex/budget rejection stays editable; unapproved hook errors remain unavailable.
func TestRuleHookFixedValidationFailures(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{&ledger.Failure{Status: 400, Code: "invalid_collection"}, 400, "invalid_collection"},
		{&ledger.Failure{Status: 422, Code: "collection_limit"}, 422, "collection_limit"},
		{&ledger.Failure{Status: 403, Code: "private diagnostic"}, 503, "temporarily_unavailable"},
		{&ledger.Failure{Status: 500, Code: "invalid_collection"}, 503, "temporarily_unavailable"},
	} {
		db := &fakeDB{}
		_, err := (&Service{DB: db, Rules: rejectedRuleHook{err: tc.err}}).Preview(context.Background(), Request{Version: 3, Actor: "verified", Workspace: workspace, Body: `{}`})
		var f *ledger.Failure
		if !errors.As(err, &f) || f.Status != tc.status || f.Code != tc.code || db.commits != 0 || db.query != "" {
			t.Fatal("hook error escaped the fixed validation boundary")
		}
	}
}

func TestV3ReadTransport(t *testing.T) {
	for _, tc := range []struct{ method, suffix, body, result, sql string }{
		{"GET", "taxonomy", "", `{"profile":"private-collection-catalog-v3","workspace_id":"` + workspace + `"}`, "select banking.get_collection_taxonomy_v3($1::uuid,$2::uuid)"},
		{"POST", "review/query", "{}", `{"profile":"private-collection-review-v3","workspace_id":"` + workspace + `"}`, "select banking.query_collection_review_v3($1::uuid,$2::uuid,$3::text)"},
		{"POST", "review/detail", `{"source_id":"` + commandID + `","observation_id":"` + workspace + `","observation_revision":"7"}`, `{"profile":"private-collection-detail-v1","workspace_id":"` + workspace + `","row":{"source_id":"` + commandID + `","observation_id":"` + workspace + `","observation_revision":"7"}}`, "select banking.get_collection_detail($1::uuid,$2::uuid,$3::text)"},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			db := &fakeDB{body: tc.result}
			service := &Service{DB: db}
			r := httptest.NewRequest(tc.method, "/v3/collections/"+workspace+"/"+tc.suffix, strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer fictional")
			if tc.method == "POST" {
				r.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			Handler(&fixedAuth{}, service).ServeHTTP(w, r)
			if w.Code != 200 || db.query != tc.sql || db.options.IsoLevel != pgx.RepeatableRead || db.commits != 1 || db.args[0] != "independently-verified" {
				t.Fatalf("status=%d query=%s commits=%d", w.Code, db.query, db.commits)
			}
			for _, token := range []string{"", "Basic fictional"} {
				r := httptest.NewRequest(tc.method, "/v3/collections/"+workspace+"/"+tc.suffix, strings.NewReader(tc.body))
				r.Header.Set("Authorization", token)
				w := httptest.NewRecorder()
				Handler(&fixedAuth{}, service).ServeHTTP(w, r)
				if w.Code != 401 || db.calls != 1 {
					t.Fatal("unverified identity reached SQL")
				}
			}
		})
	}
}

func TestV3DetailRejectsForeignRevisionResult(t *testing.T) {
	r := Request{Version: 3, Workspace: workspace, Body: `{"source_id":"` + commandID + `","observation_id":"` + workspace + `","observation_revision":"7"}`}
	for _, result := range []string{
		`{"profile":"private-collection-detail-v1","workspace_id":"` + workspace + `","row":{"source_id":"` + commandID + `","observation_id":"` + workspace + `","observation_revision":"8"}}`,
		`{"profile":"private-collection-detail-v1","workspace_id":"` + commandID + `"}`,
		`{"profile":"private-collection-command-v3","workspace_id":"` + workspace + `","command":null}`,
	} {
		db := &fakeDB{body: result}
		_, err := (&Service{DB: db}).Detail(context.Background(), r)
		if err == nil || db.commits != 0 || db.rollbacks != 1 {
			t.Fatal("foreign result committed")
		}
	}
}

type failingRuleHook struct{ calls int }

func (h *failingRuleHook) Process(_ context.Context, _ pgx.Tx, _ Request, _ string) (json.RawMessage, bool, error) {
	h.calls++
	return nil, false, errors.New("private matcher diagnostic")
}

// A hook failure cannot silently fall back to a page with an invented ruleset result.
func TestRuleHookFailureClosesTransaction(t *testing.T) {
	h := &failingRuleHook{}
	db := &fakeDB{}
	_, err := (&Service{DB: db, Rules: h}).Query(context.Background(), Request{Version: 3, Actor: "verified", Workspace: workspace, Body: "{}"})
	if err == nil || h.calls != 1 || db.commits != 0 || db.query != "" || db.rollbacks != 1 {
		t.Fatal("hook failure escaped transaction fence")
	}
}

// Missing composition must never fall back to a rule catalog SQL call without Go validation.
func TestV3PreviewRequiresRuleHook(t *testing.T) {
	db := &fakeDB{}
	_, err := (&Service{DB: db}).Preview(context.Background(), Request{Version: 3, Actor: "verified", Workspace: workspace, Body: `{}`})
	if err == nil || db.calls != 0 {
		t.Fatal("v3 preview bypassed the rule validation hook")
	}
}

func TestV3CommandsKeepExactLegacyPreviewBytes(t *testing.T) {
	body := `{"profile":"private-collection-command-v3","workspace_id":"` + workspace + `","command":{"preview":{"profile":"private-collection-preview-v2","historical":"untouched"}}}`
	for _, operation := range []string{"prepare", "pending", "execute", "cancel", "acknowledge"} {
		db := &fakeDB{body: body}
		r := Request{Version: 3, Actor: "verified", Workspace: workspace, Command: commandID, Body: `{}`}
		var result json.RawMessage
		var err error
		s := &Service{DB: db}
		switch operation {
		case "prepare":
			result, err = s.Prepare(context.Background(), r)
		case "pending":
			result, err = s.Pending(context.Background(), r)
		default:
			result, err = s.Transition(context.Background(), r, operation)
		}
		if err != nil || string(result) != body || db.commits != 1 || !strings.Contains(db.query, "_v3(") {
			t.Fatal("v3 command changed a historical preview or selected a legacy SQL entrypoint")
		}
	}
}
