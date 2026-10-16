package connections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

const actorID = "00000000-0000-4000-8000-000000000001"
const entityID = "00000000-0000-4000-8000-000000000002"
const bookID = "00000000-0000-4000-8000-000000000003"
const attemptID = "00000000-0000-4000-8000-000000000004"
const nonceID = "00000000-0000-4000-8000-000000000005"
const claimID = "00000000-0000-4000-8000-000000000006"
const refID = "00000000-0000-4000-8000-000000000007"
const oldRefID = "00000000-0000-4000-8000-000000000008"

var scope = Scope{actorID, entityID, bookID}

// ptr builds an explicitly present synthetic string.
func ptr(s string) *string { return &s }

// encode marshals synthetic fixtures without floating-point financial values.
func encode(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

// memoryRepo models one-shot replies only; actual authority is separately tested with PostgreSQL.
type memoryRepo struct {
	c            attemptContext
	denied       bool
	claimed      bool
	begun        bool
	calls        []string
	failOp       string
	claimUnknown bool
	contextReads int
	staleAt      int
}

// call models fixed operation sequencing for unit tests, not SQL authority.
func (r *memoryRepo) call(_ context.Context, op string, args ...any) (json.RawMessage, error) {
	r.calls = append(r.calls, op)
	if op == "connection_context" {
		r.contextReads++
		if r.staleAt > 0 && r.contextReads == r.staleAt {
			return nil, failed("stale_connection")
		}
	}
	if r.denied {
		return nil, failed("not_found")
	}
	if r.failOp == op {
		return nil, failed("commit_unknown")
	}
	c := r.c
	c.ExecuteLink = false
	c.ExecuteProvider = false
	switch op {
	case "begin_connection", "creator_begin_connection":
		var in BeginRequest
		_ = json.Unmarshal([]byte(args[4].(string)), &in)
		r.c.Mode = in.Mode
		c = r.c
		c.ExecuteLink = !r.begun && (in.Mode == "new" || in.Mode == "reconnect")
		r.begun = true
	case "record_link_session":
		var v map[string]string
		_ = json.Unmarshal([]byte(args[5].(string)), &v)
		r.c.LinkHash = ptr(v["link_token_sha256"])
		r.c.State = "link_ready"
		c = r.c
	case "claim_connection":
		var v struct{ Link, Public *string }
		var m map[string]*string
		_ = json.Unmarshal([]byte(args[5].(string)), &m)
		v.Link = m["link_token_sha256"]
		v.Public = m["public_token_sha256"]
		if args[4] != r.c.Nonce || r.c.Mode == "new" && (value(v.Link) != value(r.c.LinkHash) || v.Public == nil) {
			return nil, failed("connection_conflict")
		}
		if !r.claimed {
			r.claimed = true
			r.c.ClaimID = ptr(claimID)
			r.c.State = "exchange_claimed"
			if r.c.Mode == "remove" {
				r.c.State = "disconnect_pending"
			}
			r.c.PublicHash = v.Public
			c = r.c
			c.ExecuteProvider = true
			if r.claimUnknown {
				return nil, failed("commit_unknown")
			}
		}
	case "record_connection_manifest":
		var m Manifest
		_ = json.Unmarshal([]byte(args[5].(string)), &m)
		r.c.Manifest = &m
		r.c.State = "credential_pending"
		c = r.c
	case "confirm_connection_credential":
		r.c.State = "binding_review"
		if r.c.Mode != "new" {
			r.c.State = "active"
		}
		c = r.c
	case "confirm_item_removed":
		r.c.State = "disconnected"
		c = r.c
	case "connection_failure":
		r.c.State = "credential_recovery_required"
		if r.c.Mode == "remove" {
			r.c.State = "disconnect_pending"
		}
		c = r.c
	case "get_connection_attempt":
		return encode(map[string]any{"attempt_id": attemptID, "state": r.c.State, "manifest": r.c.Manifest}), nil
	case "list_connections", "creator_list_connections":
		return encode(map[string]any{"items": []any{}}), nil
	case "bind_connection":
		r.c.State = "active"
		return encode(map[string]any{"state": "active"}), nil
	case "disconnect_stream":
		return encode(map[string]any{"state": "disconnected"}), nil
	}
	return encode(c), nil
}

// TestConnectionStaleContextStopsIO rejects generation changes before the next provider or secret operation.
func TestConnectionStaleContextStopsIO(t *testing.T) {
	for _, at := range []int{2, 3, 4, 5} {
		t.Run(fmt.Sprint(at), func(t *testing.T) {
			s, r, p, store := fixture()
			input := begin(t, s)
			r.contextReads = 0
			r.staleAt = at
			_, e := s.Execute(context.Background(), scope, attemptID, input)
			code(t, e, "stale_connection")
			exchanges, manifests, writes := 0, 0, 0
			if at >= 3 {
				exchanges = 1
			}
			if at >= 4 {
				manifests = 1
			}
			if at >= 5 {
				writes = 1
			}
			if count(p.calls, "verify") != 1 || count(p.calls, "exchange") != exchanges || count(p.calls, "manifest") != manifests || store.writes != writes || store.reads != 0 {
				t.Fatal("stale context permitted another external operation")
			}
			if r.c.State == "binding_review" || r.c.State == "active" {
				t.Fatal("stale context published reference")
			}
		})
	}
}

type fakeProvider struct {
	calls    []string
	manifest Manifest
	fail     string
	hook     func(string)
	token    Secret
	item     string
}

// touch records provider calls and injects deterministic boundary faults.
func (p *fakeProvider) touch(op string) error {
	p.calls = append(p.calls, op)
	if p.hook != nil {
		p.hook(op)
	}
	if p.fail == op {
		return errors.New("private-provider-diagnostic")
	}
	return nil
}

// CreateLink returns a synthetic ephemeral token only.
func (p *fakeProvider) CreateLink(context.Context, LinkRequest) (LinkResult, error) {
	return LinkResult{Token: NewSecret("synthetic-link"), ExpiresAt: time.Now().Add(time.Hour)}, p.touch("link")
}

// VerifySession injects association success or failure without trusting browser metadata.
func (p *fakeProvider) VerifySession(context.Context, Secret, Secret, string) error {
	return p.touch("verify")
}

// Exchange simulates one possibly ambiguous credential exchange.
func (p *fakeProvider) Exchange(context.Context, Secret) (ExchangeResult, error) {
	return ExchangeResult{p.item, p.token}, p.touch("exchange")
}

// Manifest returns fixed independently chosen Item metadata.
func (p *fakeProvider) Manifest(context.Context, Secret) (Manifest, error) {
	return p.manifest, p.touch("manifest")
}

// Rotate simulates shared credential replacement.
func (p *fakeProvider) Rotate(context.Context, Secret) (ExchangeResult, error) {
	return ExchangeResult{p.item, p.token}, p.touch("rotate")
}

// Remove simulates an Item-wide irreversible boundary.
func (p *fakeProvider) Remove(context.Context, Secret) error { return p.touch("remove") }

type memorySecrets struct {
	values     map[CredentialReference]Secret
	writes     int
	reads      int
	lostPut    bool
	wrongRef   bool
	wrongToken bool
	hook       func()
}

// Put simulates immutable version storage and lost acknowledgement.
func (s *memorySecrets) Put(_ context.Context, r CredentialReference, v Secret) error {
	s.writes++
	if old, ok := s.values[r]; ok && old.value != v.value {
		return errors.New("immutable conflict")
	}
	s.values[r] = v
	if s.hook != nil {
		s.hook()
	}
	if s.lostPut {
		return errors.New("private-secret-ack")
	}
	return nil
}

// Read simulates exact-reference lookup and malformed acknowledgements.
func (s *memorySecrets) Read(_ context.Context, r CredentialReference) (StoredCredential, error) {
	s.reads++
	v, ok := s.values[r]
	if !ok {
		return StoredCredential{}, errors.New("private-missing")
	}
	if s.wrongRef {
		r.Version = "99"
	}
	if s.wrongToken {
		v = NewSecret("wrong-token")
	}
	return StoredCredential{r, v}, nil
}

// fixture builds isolated synthetic lifecycle dependencies.
func fixture() (*Service, *memoryRepo, *fakeProvider, *memorySecrets) {
	c := attemptContext{AttemptID: attemptID, EntityID: entityID, BookID: bookID, Mode: "new", State: "link_pending", Nonce: nonceID, ExpiresAt: time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339), CredentialRef: ptr(refID), CredentialVersion: ptr("1")}
	r := &memoryRepo{c: c}
	p := &fakeProvider{item: "synthetic-item", token: NewSecret("synthetic-access"), manifest: Manifest{ItemID: "synthetic-item", Accounts: []Account{{AccountID: "account-one", Type: "depository", Currency: ptr("USD")}}, ConsentExpiresAt: time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)}}
	store := &memorySecrets{values: map[CredentialReference]Secret{}}
	return &Service{repo: r, provider: p, secrets: store}, r, p, store
}

// begin exercises the public ephemeral Link handoff.
func begin(t *testing.T, s *Service) ExecuteRequest {
	t.Helper()
	v, e := s.Begin(context.Background(), scope, "begin", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if e != nil {
		t.Fatal(e)
	}
	return ExecuteRequest{v.Nonce, v.LinkToken, "synthetic-public"}
}

// code checks safe classifications without raw error text.
func code(t *testing.T, e error, want string) {
	t.Helper()
	var f *Failure
	if !errors.As(e, &f) || f.Code != want {
		t.Fatalf("got %v want %s", e, want)
	}
}

// count independently counts externally observable operations.
func count(v []string, w string) int {
	n := 0
	for _, s := range v {
		if s == w {
			n++
		}
	}
	return n
}

// TestPlaidConnectionLifecycle proves explicit handoff, single exchange and non-secret public status.
func TestPlaidConnectionLifecycle(t *testing.T) {
	s, r, p, store := fixture()
	input := begin(t, s)
	out, e := s.Execute(context.Background(), scope, attemptID, input)
	if e != nil {
		t.Fatal(e)
	}
	if r.c.State != "binding_review" || store.writes != 1 || count(p.calls, "exchange") != 1 {
		t.Fatal("missing reviewed handoff")
	}
	if _, e = s.Execute(context.Background(), scope, attemptID, input); e != nil {
		t.Fatal(e)
	}
	if count(p.calls, "exchange") != 1 || count(p.calls, "verify") != 1 {
		t.Fatal("replayed provider operation")
	}
	for _, private := range []string{"synthetic-access", "synthetic-public", refID, "claim_id", "nonce"} {
		if strings.Contains(string(out), private) {
			t.Fatal("private result")
		}
	}
	if _, e = s.Bind(context.Background(), scope, attemptID, "bind", BindingRequest{ConsentGrants: true}); e != nil || r.c.State != "active" {
		t.Fatal("explicit binding failed")
	}
}

// TestConnectionUncertainEffects proves no replay of a potentially successful remote mutation.
func TestConnectionUncertainEffects(t *testing.T) {
	for _, phase := range []string{"exchange", "claim", "secret", "manifest"} {
		t.Run(phase, func(t *testing.T) {
			s, r, p, store := fixture()
			input := begin(t, s)
			switch phase {
			case "exchange", "manifest":
				p.fail = phase
			case "claim":
				r.claimUnknown = true
			case "secret":
				store.lostPut = true
			}
			_, e := s.Execute(context.Background(), scope, attemptID, input)
			if phase == "claim" {
				code(t, e, "commit_unknown")
			} else {
				code(t, e, "credential_recovery_required")
			}
			p.fail = ""
			r.claimUnknown = false
			store.lostPut = false
			_, _ = s.Execute(context.Background(), scope, attemptID, input)
			if count(p.calls, "exchange") > 1 {
				t.Fatal("blind exchange retry")
			}
			_, e = s.Reconcile(context.Background(), scope, attemptID)
			if phase == "secret" {
				if e != nil || r.c.State != "binding_review" {
					t.Fatal("lost secret acknowledgement unresolved", e)
				}
			} else {
				code(t, e, "credential_recovery_required")
			}
		})
	}
}

// TestConnectionDenials ensures revoked scope and rejected association never acquire credentials.
func TestConnectionDenials(t *testing.T) {
	t.Run("before read", func(t *testing.T) {
		s, r, p, store := fixture()
		r.denied = true
		_, e := s.Begin(context.Background(), scope, "x", BeginRequest{Mode: "new"})
		code(t, e, "not_found")
		if len(p.calls) != 0 || store.reads != 0 {
			t.Fatal("denied I/O")
		}
	})
	t.Run("association", func(t *testing.T) {
		s, _, p, store := fixture()
		input := begin(t, s)
		p.fail = "verify"
		_, e := s.Execute(context.Background(), scope, attemptID, input)
		code(t, e, "credential_recovery_required")
		if count(p.calls, "exchange") != 0 || store.writes != 0 {
			t.Fatal("unverified exchange")
		}
	})
	t.Run("revoked after exchange", func(t *testing.T) {
		s, r, p, store := fixture()
		input := begin(t, s)
		p.hook = func(op string) {
			if op == "exchange" {
				r.denied = true
			}
		}
		_, e := s.Execute(context.Background(), scope, attemptID, input)
		code(t, e, "not_found")
		if count(p.calls, "manifest") != 0 || store.writes != 0 {
			t.Fatal("revoked publication")
		}
	})
	for _, bad := range []string{"bad\npublic", "bad\uFFFDpublic", ""} {
		t.Run("token"+bad, func(t *testing.T) {
			s, _, p, _ := fixture()
			input := begin(t, s)
			input.PublicToken = bad
			_, e := s.Execute(context.Background(), scope, attemptID, input)
			if e == nil {
				t.Fatal("invalid token accepted")
			}
			if count(p.calls, "exchange") != 0 {
				t.Fatal("invalid token exchanged")
			}
		})
	}
}

// TestConnectionReadback rejects storage identity/value mismatch before publication.
func TestConnectionReadback(t *testing.T) {
	for _, kind := range []string{"reference", "value", "manifest"} {
		t.Run(kind, func(t *testing.T) {
			s, r, p, store := fixture()
			input := begin(t, s)
			switch kind {
			case "reference":
				store.wrongRef = true
			case "value":
				store.wrongToken = true
			case "manifest":
				p.manifest.ItemID = "other-item"
			}
			_, e := s.Execute(context.Background(), scope, attemptID, input)
			code(t, e, "credential_recovery_required")
			if r.c.State == "binding_review" {
				t.Fatal("unverified credential published")
			}
		})
	}
}

// TestPlaidStreamAndCredentials distinguishes Item operations from local account disconnect.
func TestPlaidStreamAndCredentials(t *testing.T) {
	for _, mode := range []string{"reconnect", "rotate", "remove"} {
		t.Run(mode, func(t *testing.T) {
			s, r, p, store := fixture()
			r.c.Mode = mode
			r.c.ExpectedItemID = ptr("synthetic-item")
			r.c.PreviousCredentialRef = ptr(oldRefID)
			r.c.PreviousCredentialVersion = ptr("1")
			r.c.LinkHash = ptr(hash("synthetic-link"))
			store.values[CredentialReference{oldRefID, "1", "synthetic-item", "sandbox"}] = NewSecret("old-access")
			input := ExecuteRequest{Nonce: nonceID}
			if mode == "reconnect" {
				input.LinkToken = "synthetic-link"
			}
			_, e := s.Execute(context.Background(), scope, attemptID, input)
			if e != nil {
				t.Fatal(e)
			}
			want := "active"
			if mode == "remove" {
				want = "disconnected"
			}
			if r.c.State != want {
				t.Fatal(r.c.State)
			}
			_, _ = s.Execute(context.Background(), scope, attemptID, input)
			if count(p.calls, "rotate") > 1 || count(p.calls, "remove") > 1 || count(p.calls, "exchange") != 0 {
				t.Fatal("wrong remote lifecycle")
			}
			before := len(p.calls)
			_, e = s.Disconnect(context.Background(), scope, refID, "disconnect", "1")
			if e != nil || len(p.calls) != before {
				t.Fatal("account disconnect touched provider")
			}
		})
	}
}

// TestProviderPrivacyRetention verifies tokens cannot escape through common serialization paths.
func TestProviderPrivacyRetention(t *testing.T) {
	v := NewSecret("synthetic-private")
	if strings.Contains(fmt.Sprintf("%v %#v", v, v), "synthetic-private") {
		t.Fatal("format leak")
	}
	if _, e := json.Marshal(v); e == nil {
		t.Fatal("JSON secret leak")
	}

	for _, input := range []any{BeginResult{HostedLinkURL: "synthetic-private", LinkToken: "synthetic-private", Nonce: "synthetic-private"}, ExecuteRequest{LinkToken: "synthetic-private", PublicToken: "synthetic-private"}} {
		if strings.Contains(fmt.Sprintf("%v %#v", input, input), "synthetic-private") {
			t.Fatal("handoff diagnostic leak")
		}
	}
}

// TestConnectionUnknownItemMutation leaves ambiguous rotation/removal unresolved without a second call.
func TestConnectionUnknownItemMutation(t *testing.T) {
	for _, mode := range []string{"rotate", "remove"} {
		t.Run(mode, func(t *testing.T) {
			s, r, p, store := fixture()
			r.c.Mode = mode
			r.c.ExpectedItemID = ptr("synthetic-item")
			r.c.PreviousCredentialRef = ptr(oldRefID)
			r.c.PreviousCredentialVersion = ptr("1")
			store.values[CredentialReference{oldRefID, "1", "synthetic-item", "sandbox"}] = NewSecret("old-access")
			p.fail = mode
			input := ExecuteRequest{Nonce: nonceID}
			_, e := s.Execute(context.Background(), scope, attemptID, input)
			code(t, e, "credential_recovery_required")
			p.fail = ""
			_, _ = s.Execute(context.Background(), scope, attemptID, input)
			_, e = s.Reconcile(context.Background(), scope, attemptID)
			code(t, e, "credential_recovery_required")
			if count(p.calls, mode) != 1 || store.writes != 0 {
				t.Fatal("ambiguous Item mutation repeated")
			}
		})
	}
}

// hostedProvider models only server completion; ordinary provider tests retain their SDK contract.
type hostedProvider struct {
	*fakeProvider
	checks  int
	pending bool
}

// CompletedPublicToken never exchanges and exposes delayed provider completion deterministically.
func (p *hostedProvider) CompletedPublicToken(context.Context, Secret) (Secret, error) {
	p.checks++
	if p.pending {
		return Secret{}, failed("link_session_pending")
	}
	return NewSecret("synthetic-public"), nil
}

// TestHostedCompletionClaimBoundary protects pending, scoped authority and replay before provider I/O.
func TestHostedCompletionClaimBoundary(t *testing.T) {
	s, r, p, _ := fixture()
	input := begin(t, s)
	input.PublicToken = ""
	hosted := &hostedProvider{fakeProvider: p, pending: true}
	s.provider = hosted
	raw, e := s.Execute(context.Background(), scope, attemptID, input)
	if e != nil || !strings.Contains(string(raw), "link_ready") || r.claimed {
		t.Fatal("pending consumed claim", e)
	}
	hosted.pending = false
	if _, e = s.Execute(context.Background(), scope, attemptID, input); e != nil {
		t.Fatal(e)
	}
	if !r.claimed || count(p.calls, "exchange") != 1 {
		t.Fatal("missing single exchange")
	}
	if _, e = s.Execute(context.Background(), scope, attemptID, input); e != nil {
		t.Fatal(e)
	}
	if hosted.checks != 2 || count(p.calls, "exchange") != 1 {
		t.Fatal("replay repeated provider work")
	}
	for _, tc := range []string{"nonce", "hash", "expired", "denied"} {
		t.Run(tc, func(t *testing.T) {
			s, r, p, _ := fixture()
			in := begin(t, s)
			in.PublicToken = ""
			hosted := &hostedProvider{fakeProvider: p}
			s.provider = hosted
			switch tc {
			case "nonce":
				in.Nonce = actorID
			case "hash":
				in.LinkToken = "other"
			case "expired":
				r.c.ExpiresAt = time.Now().Add(-time.Minute).Format(time.RFC3339)
			case "denied":
				r.denied = true
			}
			if _, e := s.Execute(context.Background(), scope, attemptID, in); e == nil || hosted.checks != 0 || r.claimed {
				t.Fatal("invalid authority reached provider")
			}
		})
	}
}

// unavailableCipher models a preflight failure with no provider side effect or SQL claim.
type unavailableCipher struct{ *memorySecrets }

func (unavailableCipher) Check(context.Context) error { return errors.New("private KMS failure") }
func TestWrappingPreflightBeforeExchange(t *testing.T) {
	s, r, p, store := fixture()
	s.secrets = unavailableCipher{store}
	in := begin(t, s)
	_, e := s.Execute(context.Background(), scope, attemptID, in)
	code(t, e, "temporarily_unavailable")
	if r.claimed || count(p.calls, "exchange") != 0 || store.writes != 0 {
		t.Fatal("preflight consumed irreversible claim")
	}
}
