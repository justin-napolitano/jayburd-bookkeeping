//go:build integration

package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// repairProviderWire supplies fixed synthetic provider metadata through the production pinned SDK transport.
// No real provider is contacted; every unsupported path fails instead of simulating an exchange or rotation.
type repairProviderWire struct {
	t                       *testing.T
	pool                    *lossPool
	calls                   []string
	account, currency, kind string
	extra                   bool
}

// RoundTrip checks the exact owned update request and serves only Link creation and read-only manifest endpoints.
func (p *repairProviderWire) RoundTrip(r *http.Request) (*http.Response, error) {
	if p.pool.active.Load() != 0 {
		p.t.Fatal("provider call inside SQL transaction")
	}
	if r.URL.Scheme != "https" || r.URL.Host != "sandbox.plaid.com" || r.Method != "POST" || r.Header.Get("Plaid-Version") != "2020-09-14" {
		p.t.Fatal("wrong provider destination or version")
	}
	var v map[string]json.RawMessage
	if json.NewDecoder(r.Body).Decode(&v) != nil || string(v["access_token"]) != `"synthetic-existing-access"` {
		p.t.Fatal("repair used another credential")
	}
	p.calls = append(p.calls, r.URL.Path)
	var body string
	switch r.URL.Path {
	case "/link/token/create":
		if string(v["hosted_link"]) != `{"url_lifetime_seconds":900}` || string(v["webhook"]) != `"`+connections.SandboxTransactionsWebhookURL+`"` {
			p.t.Fatal("repair is not the explicit bounded Hosted flow")
		}
		for _, field := range []string{"products", "additional_consented_products", "optional_products", "required_if_supported_products", "account_filters", "update", "user_id"} {
			if _, ok := v[field]; ok {
				p.t.Fatal("repair widened requested scope")
			}
		}
		body = `{"link_token":"service-owned-link","expiration":"2099-01-01T00:00:00Z","hosted_link_url":"https://secure.plaid.com/hl/serviceOwned"}`
	case "/item/get":
		body = `{"item":{"item_id":"synthetic-item","error":null,"consent_expiration_time":null}}`
	case "/accounts/get":
		accounts := []any{map[string]any{"account_id": p.account, "type": p.kind, "balances": map[string]any{"iso_currency_code": p.currency}}}
		if p.extra {
			accounts = append(accounts, map[string]any{"account_id": "new-unapproved", "type": "depository", "balances": map[string]any{"iso_currency_code": "USD"}})
		}
		body = raw(map[string]any{"item": map[string]any{"item_id": "synthetic-item"}, "accounts": accounts})
	default:
		p.t.Fatal("repair attempted exchange, rotation, removal or Link result lookup")
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

// repairServiceProof composes current SQL scope, encrypted bank/Link handoff and the concrete provider adapter.
type repairServiceProof struct {
	owner    *pgx.Conn
	db, id   string
	scope    connections.Scope
	service  *connections.Service
	key      *repairCredentialWrapper
	provider *repairProviderWire
	begin    connections.BeginResult
}

// repairServiceFixture creates a fresh service-owned repair after the seed attempt, preserving all audit history.
func repairServiceFixture(t *testing.T, paused bool, deadline ...string) repairServiceProof {
	t.Helper()
	c, db, a, _, _, w := repairCredentialFixture(t, deadline...)
	linkAPI(t, db, "SELECT integration.abandon_connection($1,$2,$3,$4)::text", a["attempt_id"])
	stream := value(t, c, "SELECT stream_id::text FROM integration.sync_stream")
	if paused {
		status := linkAPI(t, db, "SELECT integration.connection_stream_status($1,$2,$3,$4)::text", stream)
		linkAPI(t, db, "SELECT integration.mutate_connection_stream($1,$2,$3,$4,'pause-before-repair',$5)::text", stream,
			raw(map[string]any{"action": "pause", "expected_item_generation": status["item_generation"], "expected_binding_revision": status["binding_revision"], "expected_stream_revision": status["stream_revision"]}))
	}
	p := &repairProviderWire{t: t, pool: w.pool, account: "Account-A", currency: "USD", kind: "depository"}
	provider, e := connections.NewHTTPProvider(connections.HTTPConfig{ClientID: connections.NewSecret("synthetic-client"), ClientSecret: connections.NewSecret("synthetic-secret"),
		ConsentExpiresAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC), WebhookURL: connections.SandboxTransactionsWebhookURL, HostedRepair: true}, p)
	if e != nil {
		t.Fatal("provider composition failed")
	}
	store, e := bankcredential.NewAPIStore(w.pool, w)
	if e != nil {
		t.Fatal("store composition failed")
	}
	s := connections.New(w.pool, provider, store)
	scope := connections.Scope{Actor: actor, Entity: entity, Book: book}
	item, generation := value(t, c, "SELECT sync_item_id::text FROM integration.sync_item"), value(t, c, "SELECT generation::text FROM integration.sync_item")
	begin, e := s.Begin(context.Background(), scope, "service-repair", connections.BeginRequest{Mode: "reconnect", SyncItemID: &item, ExpectedGeneration: &generation, ApprovedStreamIDs: []string{stream}})
	if e != nil {
		t.Fatal("owned Hosted repair begin failed", e)
	}
	id := pgObject(t, string(begin.Attempt))["attempt_id"].(string)
	if begin.HostedLinkURL != "https://secure.plaid.com/hl/serviceOwned" || begin.LinkToken != "service-owned-link" || len(p.calls) != 1 {
		t.Fatal("wrong Hosted handoff")
	}
	if got, e := time.Parse(time.RFC3339, begin.ExpiresAt); e != nil || got.After(time.Now().Add(15*time.Minute)) {
		t.Fatal("repair widened its local deadline")
	}
	return repairServiceProof{c, db, id, scope, s, w, p, begin}
}

// signedSuccess retains actual cryptographically verified ownership proof, never a direct normalized SQL fixture.
func (f repairServiceProof) signedSuccess(t *testing.T, session, status string) {
	t.Helper()
	h := signedLinkPostgres(t, f.db, "")
	if h.send(t, linkBody("service-owned-link", session, status)) != 204 {
		t.Fatal("signed owned completion rejected")
	}
}

// requireRepairStable independently checks original source/credential identities and bounded authorized effects.
func (f repairServiceProof) requireRepairStable(t *testing.T, wakeups, claims string) {
	t.Helper()
	for q, want := range map[string]string{
		"SELECT count(*)::text FROM integration.bank_credential":                                  "1",
		"SELECT count(*)::text FROM integration.credential_publication":                           "1",
		"SELECT credential_version::text FROM integration.sync_item":                              "1",
		"SELECT count(*)::text FROM integration.sync_stream":                                      "1",
		"SELECT count(*)::text FROM banking.source_account":                                       "1",
		"SELECT count(*)::text FROM integration.link_repair_claim":                                claims,
		"SELECT count(*)::text FROM integration.sync_wakeup WHERE reason='resume'":                wakeups,
		"SELECT count(*)::text FROM banking.source_observation":                                   "0",
		"SELECT count(*)::text FROM ledger.journal_entry":                                         "0",
		"SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'": "0",
	} {
		if value(t, f.owner, q) != want {
			t.Fatal("repair changed a fixed identity or publication invariant")
		}
	}
}

// TestHostedRepairServicePublication proves encrypted Hosted creation, proof-before-any-completion-I/O and one same-version resume.
func TestHostedRepairServicePublication(t *testing.T) {
	f := repairServiceFixture(t, false)
	envelope := value(t, f.owner, "SELECT envelope::text FROM integration.bank_credential")
	consent := value(t, f.owner, "SELECT consent_expires_at::text FROM integration.sync_item")
	wraps, unwraps := f.key.wraps, f.key.unwraps
	for range 2 {
		if _, e := f.service.CompleteHosted(context.Background(), f.scope, f.id); e != nil {
			t.Fatal("pending completion failed")
		}
	}
	if f.key.wraps != wraps || f.key.unwraps != unwraps || len(f.provider.calls) != 1 {
		t.Fatal("pending completion read handoff/key/provider")
	}
	f.requireRepairStable(t, "0", "0")
	f.signedSuccess(t, "one", "success")
	f.provider.extra = true
	for range 2 {
		if _, e := f.service.CompleteHosted(context.Background(), f.scope, f.id); e != nil {
			t.Fatal("repair completion failed", e)
		}
	}
	if f.key.wraps != wraps || len(f.provider.calls) != 3 || value(t, f.owner, "SELECT state FROM integration.sync_item") != "active" ||
		value(t, f.owner, "SELECT requested_generation::text FROM integration.sync_stream") != "2" ||
		value(t, f.owner, "SELECT envelope::text FROM integration.bank_credential") != envelope || value(t, f.owner, "SELECT consent_expires_at::text FROM integration.sync_item") != consent {
		t.Fatal("completion repeated work, wrote a credential or renewed consent")
	}
	f.requireRepairStable(t, "1", "1")
}

// TestHostedRepairServiceUnknownReply recovers committed claim/manifest/confirmation reply loss without new Link or exchange.
func TestHostedRepairServiceUnknownReply(t *testing.T) {
	for _, tc := range []struct{ name, operation string }{{"claim", "claim_connection"}, {"manifest", "record_connection_manifest"}, {"confirm", "confirm_connection_credential"}} {
		t.Run(tc.name, func(t *testing.T) {
			f := repairServiceFixture(t, false)
			f.signedSuccess(t, "one", "success")
			wraps := f.key.wraps
			f.key.pool.lost = tc.operation
			if _, e := f.service.CompleteHosted(context.Background(), f.scope, f.id); e == nil || !f.key.pool.fired {
				t.Fatal("committed reply loss was not preserved")
			}
			if tc.name == "claim" {
				if value(t, f.owner, "SELECT count(*)::text FROM integration.connection_attempt WHERE attempt_id=$1 AND manifest IS NULL", f.id) != "1" || len(f.provider.calls) != 1 {
					t.Fatal("lost claim touched bank manifest")
				}
				exec(t, f.owner, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", f.id)
			}
			for range 2 {
				if _, e := f.service.Reconcile(context.Background(), f.scope, f.id); e != nil {
					t.Fatal("same-claim recovery failed", e)
				}
			}
			if _, e := f.service.CompleteHosted(context.Background(), f.scope, f.id); e != nil {
				t.Fatal("terminal repair replay failed")
			}
			if f.key.wraps != wraps || value(t, f.owner, "SELECT state FROM integration.sync_item") != "active" {
				t.Fatal("recovery wrote or failed to activate")
			}
			links := 0
			for _, path := range f.provider.calls {
				if path == "/link/token/create" {
					links++
				}
			}
			if links != 1 {
				t.Fatal("recovery restarted Hosted Link")
			}
			f.requireRepairStable(t, "1", "1")
		})
	}
}

// TestHostedRepairServiceRejectedScope keeps invalid proof/current scope and reduced account manifests from resuming.
func TestHostedRepairServiceRejectedScope(t *testing.T) {
	for _, change := range []string{"exit", "ambiguous", "revoked", "deadline", "consent", "account", "currency", "type"} {
		t.Run(change, func(t *testing.T) {
			var deadline []string
			if change == "consent" {
				deadline = []string{time.Now().Add(5 * time.Second).UTC().Format(time.RFC3339Nano)}
			}
			f := repairServiceFixture(t, false, deadline...)
			status := "success"
			if change == "exit" {
				status = "exited"
			}
			f.signedSuccess(t, "one", status)
			if change == "ambiguous" {
				f.signedSuccess(t, "two", "success")
			}
			wraps, unwraps := f.key.wraps, f.key.unwraps
			claims := "0"
			switch change {
			case "revoked":
				exec(t, f.owner, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='connect'", actor)
			case "deadline":
				exec(t, f.owner, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", f.id)
			case "consent":
				exec(t, f.owner, "SELECT pg_sleep(greatest(0,extract(epoch FROM consent_expires_at-clock_timestamp()))+0.05) FROM integration.sync_item")
			case "account":
				f.provider.account = "not-approved"
				claims = "1"
			case "currency":
				f.provider.currency = "EUR"
				claims = "1"
			case "type":
				f.provider.kind = "credit"
				claims = "1"
			}
			if _, e := f.service.CompleteHosted(context.Background(), f.scope, f.id); e == nil {
				t.Fatal("invalid repair completed")
			}
			if claims == "0" && (f.key.unwraps != unwraps || len(f.provider.calls) != 1) {
				t.Fatal("denied scope/proof reached bank or handoff")
			}
			if f.key.wraps != wraps {
				t.Fatal("rejection wrote new encrypted material")
			}
			f.requireRepairStable(t, "0", claims)
		})
	}
}

// TestHostedRepairServicePaused preserves an explicitly paused approved stream across a successful same-Item repair.
func TestHostedRepairServicePaused(t *testing.T) {
	f := repairServiceFixture(t, true)
	f.signedSuccess(t, "one", "success")
	if _, e := f.service.CompleteHosted(context.Background(), f.scope, f.id); e != nil {
		t.Fatal("paused-scope repair failed", e)
	}
	if value(t, f.owner, "SELECT state FROM integration.sync_stream") != "paused" || value(t, f.owner, "SELECT requested_generation::text FROM integration.sync_stream") != "1" {
		t.Fatal("repair unpaused a stream")
	}
	f.requireRepairStable(t, "0", "1")
}
