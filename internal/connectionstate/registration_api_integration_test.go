//go:build integration

package connectionstate

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/server"
)

// registrationIdentity is confined to the disposable integration binary, not a managed identity claim.
type registrationIdentity struct{}

// Authenticate maps only this explicit fixture credential to its pre-provisioned synthetic owner.
func (registrationIdentity) Authenticate(_ context.Context, token string) (string, error) {
	if token != "synthetic-registration" {
		return "", errors.New("synthetic identity rejected")
	}
	return actor, nil
}

// lostRegistrationReply fails only HTTP delivery after SQL commits; it is not database commit uncertainty.
type lostRegistrationReply struct {
	header  http.Header
	written bool
}

// Header retains ordinary headers while delivery is faulted after commit.
func (w *lostRegistrationReply) Header() http.Header { return w.header }

// WriteHeader discards transport metadata without altering SQL state.
func (w *lostRegistrationReply) WriteHeader(int) {}

// Write fails only the final HTTP response body, after the handler has committed.
func (w *lostRegistrationReply) Write([]byte) (int, error) {
	w.written = true
	return 0, io.ErrClosedPipe
}

// TestRegistrationAPIRecovery composes HTTP, Go services and restricted PostgreSQL for first/append recovery.
func TestRegistrationAPIRecovery(t *testing.T) {
	admin, db, a := registrationFixture(t)
	api := connect(t, db, productionAPI)
	defer api.Close(context.Background())
	bank, link := &productionWrapper{key: productionKey}, &productionWrapper{key: productionLinkKey}
	store, err := bankcredential.NewProductionAPIStore(api, bank, link, bankcredential.ProductionPolicy{BankKey: productionKey, LinkKey: productionLinkKey})
	if err != nil {
		t.Fatal(err)
	}
	provider := &productionProvider{}
	service, err := connections.NewProductionWithApproval(api, provider, store, time.Now().Add(time.Hour), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	application, err := server.New(server.Config{Mode: "managed", Listen: "127.0.0.1:8080", Origin: "https://synthetic.invalid", Project: "synthetic-registration", DatabaseURL: "synthetic-injected-pool"}, server.Dependencies{Auth: registrationIdentity{}, Operations: &ledger.Service{DB: api}, Connections: service, Ready: func(context.Context) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	path := "/v1/entities/" + entity + "/books/" + book + "/connection-attempts/" + a["attempt_id"].(string) + "/registration"
	request := func(method, path, body, key string) *http.Request {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Origin", "https://synthetic.invalid")
		r.Header.Set("Authorization", "Bearer synthetic-registration")
		if method == "POST" {
			r.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		return r
	}
	callHTTP := func(method, path, body, key string, want int) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		application.HTTP.Handler.ServeHTTP(w, request(method, path, body, key))
		if w.Code != want || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("HTTP registration status%d want%d", w.Code, want)
		}
		return object(t, w.Body.String())
	}
	options := callHTTP("GET", path, "", "", 200)
	if options["can_register"] != true || optionAccount(t, options, "AccountA")["eligible"] != true || value(t, admin, "SELECT count(*)::text FROM ledger.interactive_command") != "0" {
		t.Fatal("HTTP options persisted work or lost SQL eligibility")
	}
	first := callHTTP("POST", path, `{"account_ids":["AccountA"]}`, "checking", 201)
	if first["operation"] != "register_accounts" || first["state"] != "prepared" || value(t, admin, "SELECT count(*)::text FROM integration.sync_stream") != "0" {
		t.Fatal("prepare bound or lost typed command")
	}
	commandPath := "/v1/commands/" + first["command_id"].(string)
	lost := &lostRegistrationReply{header: make(http.Header)}
	application.HTTP.Handler.ServeHTTP(lost, request("POST", commandPath+"/execute", `{}`, ""))
	if !lost.written {
		t.Fatal("lost response injection not reached")
	}
	pending := callHTTP("GET", "/v1/commands/pending", "", "", 200)["command"].(map[string]any)
	if pending["state"] != "committed" || pending["command_id"] != first["command_id"] {
		t.Fatal("committed HTTP loss was not recoverable")
	}
	options = callHTTP("GET", path, "", "", 200)
	if optionAccount(t, options, "AccountA")["reason"] != "already_registered" || optionAccount(t, options, "AccountB")["reason"] != "capacity_exhausted" {
		t.Fatal("active HTTP snapshot lost existing registration/current cap")
	}
	replay := callHTTP("POST", commandPath+"/execute", `{}`, "", 200)
	if raw(replay) != raw(pending) || value(t, admin, "SELECT count(*)::text FROM integration.sync_stream") != "1" {
		t.Fatal("unknown-outcome retry duplicated source")
	}
	callHTTP("POST", path, `{"account_ids":["AccountB"]}`, "pending-blocked", 409)
	callHTTP("POST", commandPath+"/acknowledge", `{}`, "", 200)
	stream := pending["result"].(map[string]any)["streams"].([]any)[0].(map[string]any)
	snapshot := value(t, admin, "SELECT to_jsonb(s)::text FROM integration.sync_stream s WHERE stream_id=$1", stream["stream_id"])
	callHTTP("POST", path, `{"account_ids":["AccountB"]}`, "savings", 422)
	registrationLimit(t, db, "allow-second", "2")
	second := callHTTP("POST", path, `{"account_ids":["AccountB"]}`, "savings", 201)
	secondPath := "/v1/commands/" + second["command_id"].(string)
	callHTTP("POST", secondPath+"/execute", `{}`, "", 200)
	if value(t, admin, "SELECT count(*)::text FROM integration.sync_stream") != "2" || value(t, admin, "SELECT to_jsonb(s)::text FROM integration.sync_stream s WHERE stream_id=$1", stream["stream_id"]) != snapshot || value(t, admin, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("append changed first stream or posted")
	}
	if provider.calls != 0 || bank.calls != 0 || link.calls != 0 {
		t.Fatal("registration invoked provider or cryptographic I/O")
	}
	callHTTP("POST", secondPath+"/acknowledge", `{}`, "", 200)
	callHTTP("POST", path, `{"account_ids":["AccountA"],"actor":"spoof"}`, "spoof", 400)
}
