package connections

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// repairOnlyStore independently forbids generic reads, ciphertext writes and wrapping preflight.
type repairOnlyStore struct {
	*memorySecrets
	purpose, generic, puts, checks, handoffs int
}

// Read fails any accidental fallback to the older generic-purpose resolver.
func (s *repairOnlyStore) Read(context.Context, CredentialReference) (StoredCredential, error) {
	s.generic++
	return StoredCredential{}, errors.New("forbidden generic read")
}

// Put forbids creating or replacing credential ciphertext during repair.
func (s *repairOnlyStore) Put(context.Context, CredentialReference, Secret) error {
	s.puts++
	return errors.New("forbidden write")
}

// Check forbids a fresh wrapping preflight for an existing credential.
func (s *repairOnlyStore) Check(context.Context) error {
	s.checks++
	return errors.New("forbidden preflight")
}

// ReadRepair supplies the fixed existing version through the optional purpose interface only.
func (s *repairOnlyStore) ReadRepair(ctx context.Context, id string, r CredentialReference) (StoredCredential, error) {
	if id != attemptID {
		return StoredCredential{}, errors.New("wrong attempt")
	}
	s.purpose++
	return s.memorySecrets.Read(ctx, r)
}

// ReadLink counts handoff decryption separately from bank credential access.
func (s *repairOnlyStore) ReadLink(ctx context.Context, id string) (Secret, error) {
	s.handoffs++
	return s.memorySecrets.ReadLink(ctx, id)
}

// repairUnit starts at the fixed reviewed link-ready state, with no inherited claim or provider success.
func repairUnit() (*Service, *memoryRepo, *fakeProvider, *repairOnlyStore) {
	s, r, p, base := fixture()
	r.c.Mode, r.c.State = "reconnect", "link_ready"
	r.c.ExpectedItemID, r.c.LinkHash = ptr("synthetic-item"), ptr(hash("synthetic-link"))
	r.c.PreviousCredentialRef, r.c.PreviousCredentialVersion = ptr(oldRefID), ptr("1")
	r.c.ItemConsentExpiresAt = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	base.values[CredentialReference{oldRefID, "1", "synthetic-item", "sandbox"}] = NewSecret("old-access")
	base.links = map[string]Secret{attemptID: NewSecret("synthetic-link")}
	store := &repairOnlyStore{memorySecrets: base}
	s.secrets = store
	return s, r, p, store
}

// TestHostedRepairPendingNoIO proves pending/exited/ambiguous completion cannot decrypt even the Link handoff.
func TestHostedRepairPendingNoIO(t *testing.T) {
	for _, status := range []string{"pending", "exited", "ambiguous", "unknown"} {
		t.Run(status, func(t *testing.T) {
			s, r, p, store := repairUnit()
			r.proofStatus = status
			_, e := s.CompleteHosted(context.Background(), scope, attemptID)
			if status == "pending" && e != nil || status != "pending" && e == nil {
				t.Fatal("wrong proof disposition")
			}
			_, e = s.Execute(context.Background(), scope, attemptID, ExecuteRequest{Nonce: nonceID, LinkToken: "synthetic-link"})
			if status == "pending" && e != nil || status != "pending" && e == nil {
				t.Fatal("direct completion bypassed proof")
			}
			if r.claimed || len(p.calls) != 0 || store.purpose+store.generic+store.puts+store.checks+store.handoffs != 0 {
				t.Fatal("unproved repair reached secret/provider work")
			}
		})
	}
}

// TestHostedRepairCompletion reuses exactly one existing credential with no exchange/session lookup/write.
func TestHostedRepairCompletion(t *testing.T) {
	s, r, p, store := repairUnit()
	r.proofStatus = "success"
	oldCeiling := r.c.ItemConsentExpiresAt
	for range 2 {
		if _, e := s.CompleteHosted(context.Background(), scope, attemptID); e != nil {
			t.Fatal(e)
		}
	}
	if r.c.State != "active" || r.c.Manifest == nil || r.c.Manifest.ConsentExpiresAt != oldCeiling ||
		len(p.calls) != 1 || p.calls[0] != "manifest" || store.purpose != 1 || store.handoffs != 1 ||
		store.generic+store.puts+store.checks != 0 {
		t.Fatal("repair widened consent, repeated provider work or used wrong secret purpose")
	}
}

// TestHostedRepairUnknownClaim recovers the exact claimed attempt with no manifest after a lost reply and Link expiry.
func TestHostedRepairUnknownClaim(t *testing.T) {
	s, r, p, store := repairUnit()
	r.proofStatus, r.claimUnknown = "success", true
	_, e := s.CompleteHosted(context.Background(), scope, attemptID)
	code(t, e, "commit_unknown")
	if !r.claimed || r.c.Manifest != nil || store.purpose != 0 || len(p.calls) != 0 {
		t.Fatal("unknown claim performed bank work")
	}
	r.c.ExpiresAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	if _, e = s.CompleteHosted(context.Background(), scope, attemptID); e != nil || r.c.State != "active" || store.purpose != 1 || len(p.calls) != 1 || store.handoffs != 1 {
		t.Fatal("same-claim recovery required another Link or credential")
	}
}

// TestHostedRepairRequestAndScopeDenial keeps caller token/nonce substitution and absent purpose stores outside claims.
func TestHostedRepairRequestAndScopeDenial(t *testing.T) {
	for _, in := range []ExecuteRequest{{Nonce: nonceID, LinkToken: "foreign-link"}, {Nonce: claimID, LinkToken: "synthetic-link"},
		{Nonce: nonceID, LinkToken: "synthetic-link", PublicToken: "never-exchange"}} {
		s, r, p, store := repairUnit()
		r.proofStatus = "success"
		if _, e := s.Execute(context.Background(), scope, attemptID, in); e == nil || r.claimed || len(p.calls) != 0 || store.purpose != 0 {
			t.Fatal("caller substitution reached claim or bank work")
		}
	}
	s, r, p, store := repairUnit()
	r.proofStatus = "success"
	r.denied = true
	if _, e := s.CompleteHosted(context.Background(), scope, attemptID); e == nil || store.purpose+store.handoffs != 0 || len(p.calls) != 0 {
		t.Fatal("revoked scope reached secrets")
	}
}

// TestOwnedHostedRepairWire requires a separate explicit capability and exact bounded Hosted update request.
func TestOwnedHostedRepairWire(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		cfg := config()
		cfg.WebhookURL = SandboxTransactionsWebhookURL
		cfg.HostedRepair = enabled
		calls := 0
		p, e := NewHTTPProvider(cfg, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Host != "sandbox.plaid.com" || r.URL.Path != "/link/token/create" || r.Method != "POST" || r.Header.Get("Plaid-Version") != "2020-09-14" {
				t.Fatal("wrong destination or version")
			}
			var wire map[string]json.RawMessage
			if json.NewDecoder(r.Body).Decode(&wire) != nil || string(wire["access_token"]) != `"old-access"` || string(wire["webhook"]) != `"`+SandboxTransactionsWebhookURL+`"` {
				t.Fatal("wrong existing credential or receiver")
			}
			for _, field := range []string{"products", "additional_consented_products", "required_if_supported_products", "optional_products", "update", "account_filters", "user_id"} {
				if _, ok := wire[field]; ok {
					t.Fatal("Hosted update widened scope")
				}
			}
			if string(wire["hosted_link"]) != `{"url_lifetime_seconds":900}` || string(wire["user"]) != `{"client_user_id":"`+actorID+`"}` {
				t.Fatal("wrong Hosted lifetime or server user")
			}
			return response(200, `{"link_token":"owned-update","expiration":"2099-01-01T00:00:00Z","hosted_link_url":"https://secure.plaid.com/hl/owned"}`), nil
		}))
		if e != nil || calls != 0 || p.SupportsMode("reconnect") != enabled {
			t.Fatal("capability was implicit or constructor made I/O")
		}
		_, e = p.CreateLink(context.Background(), LinkRequest{ActorID: actorID, Mode: "reconnect", AccessToken: NewSecret("old-access")})
		if enabled && (e != nil || calls != 1) || !enabled && (e == nil || calls != 0) {
			t.Fatal("unsafe repair activation")
		}
	}
	cfg := config()
	cfg.HostedRepair = true
	if _, e := NewHTTPProvider(cfg, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unconfigured receiver used transport")
		return nil, nil
	})); e == nil {
		t.Fatal("repair enabled without reviewed receiver")
	}
}

// TestHostedRepairNoAutomaticRestart keeps local unclaimed cancellation free of removal and private expiry context.
func TestHostedRepairNoAutomaticRestart(t *testing.T) {
	s, r, p, store := repairUnit()
	r.c.ExpiresAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	r.failOp = "connection_context"
	if _, e := s.Abandon(context.Background(), scope, attemptID); e != nil || len(p.calls) != 0 || store.purpose+store.handoffs != 0 || strings.Contains(strings.Join(r.calls, ","), "begin_connection") {
		t.Fatal("local cancellation restarted or removed the Item")
	}
}
