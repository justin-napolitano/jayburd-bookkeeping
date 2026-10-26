//go:build integration

package connectionstate

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// productionWrapper is an independent fictional KEK; real envelope AEAD still executes unchanged.
type productionWrapper struct {
	key   string
	calls int
}

// Wrap authenticates only the test's independent key and exact AAD.
func (w *productionWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	w.calls++
	b, _ := aes.NewCipher(bytes.Repeat([]byte{17}, 32))
	g, _ := cipher.NewGCM(b)
	return w.key + "/cryptoKeyVersions/7", g.Seal(nil, make([]byte, 12), key, aad), nil
}

// Unwrap rejects foreign version metadata before authenticating synthetic ciphertext.
func (w *productionWrapper) Unwrap(_ context.Context, version string, raw, aad []byte) ([]byte, error) {
	w.calls++
	if version != w.key+"/cryptoKeyVersions/7" {
		return nil, bankcredential.ErrUnavailable
	}
	b, _ := aes.NewCipher(bytes.Repeat([]byte{17}, 32))
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, make([]byte, 12), raw, aad)
}

// productionProvider supplies fixed original synthetic identity and counts one-shot enrollment effects.
type productionProvider struct{ calls, exchanges int }

// CreateLink supplies a fictional Hosted handoff so the real encrypted Link persistence path executes.
func (p *productionProvider) CreateLink(context.Context, connections.LinkRequest) (connections.LinkResult, error) {
	p.calls++
	return connections.LinkResult{Token: connections.NewSecret("synthetic-production-link"), HostedURL: connections.NewSecret("https://secure.plaid.com/hl/synthetic-production"), ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// VerifySession models the already separately tested provider association boundary without network.
func (p *productionProvider) VerifySession(context.Context, connections.Secret, connections.Secret, string) error {
	p.calls++
	return nil
}

// Exchange counts the one-shot mutation independently of the SQL result.
func (p *productionProvider) Exchange(context.Context, connections.Secret) (connections.ExchangeResult, error) {
	p.calls++
	p.exchanges++
	return connections.ExchangeResult{ItemID: "FictionalItem", AccessToken: connections.NewSecret("synthetic-production-access")}, nil
}

// Manifest supplies independently authored identity, currency and consent metadata.
func (p *productionProvider) Manifest(context.Context, connections.Secret) (connections.Manifest, error) {
	p.calls++
	code, subtype := "USD", "checking"
	return connections.Manifest{ItemID: "FictionalItem", Accounts: []connections.Account{{AccountID: "AccountA", Type: "depository", Subtype: &subtype, Currency: &code}}, ConsentExpiresAt: "2099-01-01T00:00:00Z"}, nil
}

// Rotate cannot be used by this enrollment fixture.
func (p *productionProvider) Rotate(context.Context, connections.Secret) (connections.ExchangeResult, error) {
	p.calls++
	return connections.ExchangeResult{}, fmt.Errorf("synthetic unsupported")
}

// Remove cannot be mistaken for local disconnect or credential read.
func (p *productionProvider) Remove(context.Context, connections.Secret) error {
	p.calls++
	return fmt.Errorf("synthetic unsupported")
}

// TestProductionCredentialRuntimeSQL proves real service/current SQL/AAD storage and worker resolution together.
func TestProductionCredentialRuntimeSQL(t *testing.T) {
	admin, db := productionSetup(t)
	api := connect(t, db, productionAPI)
	defer api.Close(context.Background())
	sync := connect(t, db, productionWorker)
	defer sync.Close(context.Background())
	bank, link := &productionWrapper{key: productionKey}, &productionWrapper{key: productionLinkKey}
	policy := bankcredential.ProductionPolicy{BankKey: productionKey, LinkKey: productionLinkKey}
	store, e := bankcredential.NewProductionAPIStore(api, bank, link, policy)
	if e != nil {
		t.Fatal(e)
	}
	p := &productionProvider{}
	s, e := connections.NewProductionWithApproval(api, p, store, time.Now().Add(time.Hour), time.Now)
	if e != nil {
		t.Fatal(e)
	}
	scope := connections.Scope{Actor: actor, Entity: entity, Book: book}
	// A valid but unapproved configured key cannot create an attempt or consume a token.
	wrong := policy
	wrong.BankKey = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/other"
	badStore, _ := bankcredential.NewProductionAPIStore(api, bank, link, wrong)
	bad, _ := connections.NewProductionWithApproval(api, p, badStore, time.Now().Add(time.Hour), time.Now)
	if _, e = bad.Begin(context.Background(), scope, "wrong-runtime-key", connections.BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}); e == nil || p.calls != 0 || bank.calls != 0 || link.calls != 0 {
		t.Fatal("wrong runtime key created work")
	}
	if value(t, admin, "SELECT count(*) FROM integration.connection_attempt") != "0" {
		t.Fatal("preflight failure created durable intent")
	}
	out, e := s.Begin(context.Background(), scope, "production-runtime", connections.BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if e != nil {
		t.Fatal("Production service begin failed", e)
	}
	a := object(t, string(out.Attempt))
	id := a["attempt_id"].(string)
	// Recovery decrypts the persisted Production Link handoff under current SQL authority.
	linkReader := store.ForScope(scope).(connections.LinkHandoffStore)
	if token, e := linkReader.ReadLink(context.Background(), id); e != nil || token != connections.NewSecret("synthetic-production-link") {
		t.Fatal("Production handoff recovery failed", e)
	}
	request := connections.ExecuteRequest{Nonce: out.Nonce, LinkToken: out.LinkToken, PublicToken: "synthetic-production-public"}
	if _, e = s.Execute(context.Background(), scope, id, request); e != nil {
		t.Fatal("Production credential exchange failed", e)
	}
	if _, e = s.Execute(context.Background(), scope, id, request); e != nil || p.exchanges != 1 {
		t.Fatal("one-shot exchange replay changed effects", e)
	}
	c := object(t, call(t, db, productionAPI, "SELECT integration.connection_context($1,$2,$3,$4)::text", actor, entity, book, id))
	ref := connections.CredentialReference{ReferenceID: c["credential_ref"].(string), Version: c["credential_version"].(string), ItemID: "FictionalItem", Environment: "production"}
	scoped := store.ForScope(scope)
	if got, e := scoped.Read(context.Background(), ref); e != nil || got.Token != connections.NewSecret("synthetic-production-access") {
		t.Fatal("addressed Production read failed", e)
	}
	bound := object(t, call(t, db, productionAPI, "SELECT integration.bind_connection($1,$2,$3,$4,'runtime-bind',$5)::text", actor, entity, book, id, raw(map[string]any{"consent_grants": true, "bindings": []any{binding("AccountA", account)}})))
	stream := bound["streams"].([]any)[0].(map[string]any)
	x := object(t, call(t, db, productionWorker, "SELECT integration.claim_sync($1,$2,$3,'00000000-0000-4000-8000-000000000082')::text", entity, book, stream["stream_id"]))
	static := 0
	resolver, _ := bankcredential.NewProductionResolver(sync, bank, policy, func(context.Context) (acquisition.ProviderCredentials, error) {
		static++
		return acquisition.ProviderCredentials{ClientID: "synthetic-production-client", Secret: "synthetic-production-secret"}, nil
	})
	ar := acquisition.CredentialReference{ReferenceID: ref.ReferenceID, Version: ref.Version, ItemID: ref.ItemID, Environment: ref.Environment}
	as := acquisition.Scope{EntityID: entity, BookID: book, StreamID: stream["stream_id"].(string)}
	if got, e := resolver.ResolveScoped(context.Background(), as, x["set_id"].(string), x["fence"].(string), ar); e != nil || got.AccessToken != "synthetic-production-access" || static != 1 {
		t.Fatal("worker addressed resolution failed", e)
	}
	before := bank.calls
	for _, field := range []string{"environment", "item", "version", "reference"} {
		wrong := ar
		switch field {
		case "environment":
			wrong.Environment = "sandbox"
		case "item":
			wrong.ItemID = "OtherItem"
		case "version":
			wrong.Version = "2"
		case "reference":
			wrong.ReferenceID = actor
		}
		if _, e := resolver.ResolveScoped(context.Background(), as, x["set_id"].(string), x["fence"].(string), wrong); e == nil || bank.calls != before || static != 1 {
			t.Fatal("mixed worker binding reached unwrap/static")
		}
	}
	wrongResolver, _ := bankcredential.NewProductionResolver(sync, bank, wrong, func(context.Context) (acquisition.ProviderCredentials, error) {
		static++
		return acquisition.ProviderCredentials{}, nil
	})
	if _, e := wrongResolver.ResolveScoped(context.Background(), as, x["set_id"].(string), x["fence"].(string), ar); e == nil || bank.calls != before || static != 1 {
		t.Fatal("wrong worker key reached unwrap/static")
	}
	call(t, db, "synthetic_operator", "SELECT integration.configure_provider_scope($1,$2,$3,'disable-runtime',$4)::text", actor, entity, book, raw(productionScope(false)))
	before = bank.calls
	if _, e := scoped.Read(context.Background(), ref); e == nil || bank.calls != before {
		t.Fatal("revoked scope reached unwrap")
	}
	if value(t, admin, "SELECT count(*) FROM ledger.journal_entry") != "0" {
		t.Fatal("credential setup created journal")
	}
}
