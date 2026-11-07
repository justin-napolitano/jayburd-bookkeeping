package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// v3Reader exercises the public port without pretending synthetic authentication proves SQL authority.
type v3Reader struct{ fakeOperations }

func (p *v3Reader) Detail(_ context.Context, r Request) (json.RawMessage, error) {
	return p.call(r, "detail")
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
