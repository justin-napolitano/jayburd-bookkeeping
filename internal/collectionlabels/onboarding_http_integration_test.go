//go:build integration

package collectionlabels

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectiononboarding"
)

// on47HTTPCountDB observes real transaction admissions across injected lost replies.
type on47HTTPCountDB struct {
	ledger.Beginner
	begins int
}

func (d *on47HTTPCountDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	d.begins++
	return d.Beginner.BeginTx(ctx, o)
}

// on47HTTP uses the actual handler/service and restricted pool without opening
// a host listener. Authentication remains a fixed fictional integration adapter.
func on47HTTP(t testing.TB, h http.Handler, method, path, body, token string, want int) map[string]any {
	t.Helper()
	r := httptest.NewRequest(method, "/v1/collection-onboarding"+path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("onboarding HTTP wanted%d got%d: %s", want, w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("private response headers missing")
	}
	var result map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	if want >= 400 {
		if len(result) != 5 || result["request_id"] == nil || w.Header().Get("Content-Type") != "application/problem+json" {
			t.Fatal("problem profile mismatch")
		}
		if want == 401 && w.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Fatal("missing auth challenge")
		}
	}
	for _, s := range []string{"legacy-item", "legacy-account", "original-cursor", "credential_ref", "synthetic lost commit acknowledgement"} {
		if strings.Contains(w.Body.String(), s) {
			t.Fatal("protected metadata escaped")
		}
	}
	return result
}
func on47HTTPCall(t testing.TB, h http.Handler, method, path string, body any, want int) map[string]any {
	t.Helper()
	s := ""
	if body != nil {
		s = raw(body)
	}
	return on47HTTP(t, h, method, path, s, "synthetic-label-owner", want)
}
func on47HTTPPreview(t testing.TB, h http.Handler, body any) map[string]any {
	t.Helper()
	return on47HTTPCall(t, h, http.MethodPost, "/previews", body, 200)
}
func on47HTTPPrepare(t testing.TB, h http.Handler, p map[string]any, key string, want int) map[string]any {
	t.Helper()
	return on47HTTPCall(t, h, http.MethodPost, "/commands", map[string]any{"preview_id": p["preview_id"], "prepare_key": key}, want)
}
func on47HTTPTransition(t testing.TB, h http.Handler, c map[string]any, op string, want int) map[string]any {
	t.Helper()
	return on47HTTPCall(t, h, http.MethodPost, "/commands/"+c["command_id"].(string)+"/"+op, map[string]any{}, want)
}

func TestOnboarding47HTTPSetupLifecycle(t *testing.T) {
	owner, db := fixture(t)
	pool := activationPool(t, db)
	h := collectiononboarding.Handler(labelHTTPAuth{}, &collectiononboarding.Service{DB: pool})
	catalog := on47HTTPCall(t, h, http.MethodGet, "", nil, 200)
	if catalog["profile"] != "private-onboarding-catalog-v1" || catalog["can_setup"] != true {
		t.Fatal("absent catalog mismatch")
	}
	p := on47HTTPPreview(t, h, map[string]any{"operation": "setup"})
	if p["profile"] != "private-onboarding-preview-v1" {
		t.Fatal("preview mismatch")
	}
	c := on47HTTPPrepare(t, h, p, "cancel-setup", 201)
	pending := on47HTTPCall(t, h, http.MethodGet, "/commands/pending", nil, 200)
	if pending["profile"] != "private-onboarding-pending-v1" || pending["recovery_id"] != c["recovery_id"] || raw(pending["command"]) != raw(c) {
		t.Fatal("pending exact command mismatch")
	}
	c = on47HTTPTransition(t, h, c, "cancel", 200)
	if c["result"].(map[string]any)["cancelled"] != true {
		t.Fatal("cancel receipt mismatch")
	}
	c = on47HTTPTransition(t, h, c, "acknowledge", 200)
	replay := on47HTTPPrepare(t, h, p, "cancel-setup", 201)
	if raw(replay) != raw(c) || c["acknowledged"] != true {
		t.Fatal("acknowledged same-key receipt mismatch")
	}
	p = on47HTTPPreview(t, h, map[string]any{"operation": "setup"})
	c = on47HTTPPrepare(t, h, p, "execute-setup", 201)
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_workspace") != "0" {
		t.Fatal("inert requests granted workspace")
	}
	c = on47HTTPTransition(t, h, c, "execute", 200)
	if c["profile"] != "private-onboarding-command-v1" || c["state"] != "committed" {
		t.Fatal("setup execute profile mismatch")
	}
	on47HTTPTransition(t, h, c, "acknowledge", 200)
	catalog = on47HTTPCall(t, h, http.MethodGet, "", nil, 200)
	if catalog["can_setup"] != false || catalog["workspace"].(map[string]any)["state"] != "ready" || on47HTTPCall(t, h, http.MethodGet, "/commands/pending", nil, 200)["state"] != "none" {
		t.Fatal("setup discovery mismatch")
	}
	if value(t, owner, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("setup posted journal")
	}
}

func TestOnboarding47HTTPAdoptionAndRecovery(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(map[bool]string{false: "prepared", true: "committed"}[committed], func(t *testing.T) {
			owner, db, _ := legacyFixture(t)
			on47Setup(t, db)
			before := legacyHash(t, owner)
			pool := activationPool(t, db)
			h := collectiononboarding.Handler(labelHTTPAuth{}, &collectiononboarding.Service{DB: pool})
			catalog := on47HTTPCall(t, h, http.MethodGet, "", nil, 200)
			candidate := catalog["candidates"].([]any)[0].(map[string]any)
			p := on47HTTPPreview(t, h, map[string]any{"operation": "adopt", "candidate_id": candidate["candidate_id"], "snapshot": candidate["snapshot"]})
			if p["review"].(map[string]any)["impact"] != "pause_legacy_acquisition" {
				t.Fatal("adoption review mismatch")
			}
			c := on47HTTPPrepare(t, h, p, "http-adopt", 201)
			on47HTTP(t, h, http.MethodPost, "/commands/"+c["command_id"].(string)+"/execute", "{}", "synthetic-label-other", 404)
			if committed {
				c = on47HTTPTransition(t, h, c, "execute", 200)
				if c["result"].(map[string]any)["item_generation"] != "2" {
					t.Fatal("adoption result mismatch")
				}
			}
			if legacyHash(t, owner) != before {
				t.Fatal("HTTP adoption changed original history")
			}
			on47HTTPCall(t, h, http.MethodPost, "/recovery/clear", map[string]any{"recovery_id": c["recovery_id"]}, 409)
			exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND book_id=$2 AND source_account_id=$3 AND principal_id=$4 AND capability='read'", entityID, bookID, sourceID, actor)
			pending := on47HTTPCall(t, h, http.MethodGet, "/commands/pending", nil, 200)
			if pending["state"] != "inaccessible" || pending["command"] != nil || pending["recovery_id"] != c["recovery_id"] {
				t.Fatal("revoked pending leaked payload")
			}
			on47HTTPTransition(t, h, c, "execute", 404)
			cleared := on47HTTPCall(t, h, http.MethodPost, "/recovery/clear", map[string]any{"recovery_id": c["recovery_id"]}, 200)
			if len(cleared) != 3 || cleared["profile"] != "private-onboarding-clear-v1" || cleared["cleared"] != true {
				t.Fatal("payload-free closure mismatch")
			}
			if raw(on47HTTPCall(t, h, http.MethodPost, "/recovery/clear", map[string]any{"recovery_id": c["recovery_id"]}, 200)) != raw(cleared) || legacyHash(t, owner) != before {
				t.Fatal("closure replay changed history")
			}
		})
	}
}

func TestOnboarding47HTTPFramingAndUnknownCommit(t *testing.T) {
	owner, db, _ := legacyFixture(t)
	on47Setup(t, db)
	before := legacyHash(t, owner)
	pool := activationPool(t, db)
	counted := &on47HTTPCountDB{Beginner: pool}
	lost := &activationLostDB{db: counted}
	h := collectiononboarding.Handler(labelHTTPAuth{}, &collectiononboarding.Service{DB: lost})
	for _, tc := range []struct {
		body, token string
		status      int
	}{
		{`{"operation":`, "synthetic-unenrolled", 401},
		{`{"operation":"setup","actor":"foreign"}`, "synthetic-label-owner", 400},
		{`{"operation":"setup","operation":"setup"}`, "synthetic-label-owner", 400},
		{`{"operation":`, "synthetic-label-owner", 400},
	} {
		on47HTTP(t, h, http.MethodPost, "/previews", tc.body, tc.token, tc.status)
	}
	on47HTTP(t, h, http.MethodGet, "?unexpected=1", "", "synthetic-label-owner", 400)
	if value(t, owner, "SELECT count(*)::text FROM banking.collection_onboarding_preview WHERE operation='adopt'") != "0" {
		t.Fatal("invalid requests persisted intent")
	}
	candidate := on47HTTPCall(t, h, http.MethodGet, "", nil, 200)["candidates"].([]any)[0].(map[string]any)
	p := on47HTTPPreview(t, h, map[string]any{"operation": "adopt", "candidate_id": candidate["candidate_id"], "snapshot": candidate["snapshot"]})
	attempts := counted.begins
	lost.lose.Store(true)
	on47HTTPPrepare(t, h, p, "lost-prepare", 503)
	if counted.begins != attempts+1 {
		t.Fatal("unknown prepare automatically retried")
	}
	c := on47HTTPPrepare(t, h, p, "lost-prepare", 201)
	if on47HTTPCall(t, h, http.MethodGet, "/commands/pending", nil, 200)["command"].(map[string]any)["command_id"] != c["command_id"] {
		t.Fatal("unknown prepare receipt not discoverable")
	}
	attempts = counted.begins
	lost.lose.Store(true)
	on47HTTPTransition(t, h, c, "execute", 503)
	if counted.begins != attempts+1 {
		t.Fatal("unknown execute automatically retried")
	}
	c = on47HTTPTransition(t, h, c, "execute", 200)
	if c["state"] != "committed" || value(t, owner, "SELECT count(*)::text FROM integration.collection_receipt WHERE operation='adopt'") != "1" || legacyHash(t, owner) != before {
		t.Fatal("unknown execute duplicated or altered history")
	}
	attempts = counted.begins
	lost.lose.Store(true)
	on47HTTPTransition(t, h, c, "acknowledge", 503)
	if counted.begins != attempts+1 {
		t.Fatal("unknown acknowledgement automatically retried")
	}
	exec(t, owner, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND principal_id=$2 AND permission='read'", entityID, actor)
	on47HTTPTransition(t, h, c, "acknowledge", 404)
	on47HTTPCall(t, h, http.MethodPost, "/recovery/clear", map[string]any{"recovery_id": c["recovery_id"]}, 200)
	if on47HTTPCall(t, h, http.MethodGet, "/commands/pending", nil, 200)["state"] != "none" || legacyHash(t, owner) != before {
		t.Fatal("lost acknowledgement recovery mismatch")
	}
	if pool.Stat().AcquiredConns() != 0 {
		t.Fatal("HTTP service retained pool connection")
	}
}
