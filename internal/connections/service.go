package connections

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// repository is private so callers cannot submit trusted manifests through the service.
type repository interface {
	call(context.Context, string, ...any) (json.RawMessage, error)
}

// Service commits authority before remote work and publishes references only after readback.
type Service struct {
	repo          repository
	provider      Provider
	secrets       SecretStore
	creatorLogins *WorkerLogins
}

// New composes injected boundaries without startup credential discovery.
func New(db Beginner, provider Provider, secrets SecretStore) *Service {
	return &Service{repo: &sqlRepository{DB: db}, provider: provider, secrets: secrets}
}

type attemptContext struct {
	AttemptID                 string    `json:"attempt_id"`
	EntityID                  string    `json:"entity_id"`
	BookID                    string    `json:"book_id"`
	Mode                      string    `json:"mode"`
	State                     string    `json:"state"`
	Nonce                     string    `json:"nonce"`
	ExpiresAt                 string    `json:"expires_at"`
	SyncItemID                *string   `json:"sync_item_id"`
	ItemGeneration            *string   `json:"item_generation"`
	ExpectedItemID            *string   `json:"expected_item_id"`
	ClaimID                   *string   `json:"claim_id"`
	CredentialRef             *string   `json:"credential_ref"`
	CredentialVersion         *string   `json:"credential_version"`
	PreviousCredentialRef     *string   `json:"previous_credential_ref"`
	PreviousCredentialVersion *string   `json:"previous_credential_version"`
	LinkHash                  *string   `json:"link_token_sha256"`
	PublicHash                *string   `json:"public_token_sha256"`
	Manifest                  *Manifest `json:"manifest"`
	ExecuteLink               bool      `json:"execute_link"`
	ExecuteProvider           bool      `json:"execute_provider"`
}

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var keyRE = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var opaqueRE = regexp.MustCompile(`^[!-~]{1,256}$`)
var tokenRE = regexp.MustCompile(`^[!-~]+$`)
var currencyRE = regexp.MustCompile(`^[A-Z]{3}$`)
var subtypeRE = regexp.MustCompile(`^[a-z0-9][a-z0-9 -]{0,63}$`)

// validID restricts identifiers to canonical nonzero UUIDs.
func validID(v string) bool {
	return uuidRE.MatchString(v) && v != "00000000-0000-0000-0000-000000000000"
}

// validScope checks transport syntax only; SQL owns authorization.
func validScope(s Scope) bool { return validID(s.Actor) && validID(s.Entity) && validID(s.Book) }

// validRevision preserves the positive bigint string contract.
func validRevision(v string) bool {
	n, e := strconv.ParseInt(v, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == v
}

// value reads nullable trusted context fields without implicit defaults.
func value(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// hash fingerprints ephemeral handoff material without storing plaintext.
func hash(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }

// jsonText serializes closed internal structs without numeric conversion.
func jsonText(v any) string { b, _ := json.Marshal(v); return string(b) }

// secretValid bounds opaque tokens and rejects normalized Unicode or control characters.
func secretValid(v Secret) bool { return len(v.value) <= 4096 && tokenRE.MatchString(v.value) }

// ready checks dependencies and syntax before any SQL or external operation.
func (s *Service) ready(scope Scope) error {
	if s == nil || s.repo == nil || s.provider == nil || s.secrets == nil {
		return failed("configuration_error")
	}
	if !validScope(scope) {
		return failed("invalid_request")
	}
	return nil
}

// context validates SQL routing metadata before any provider or secret-store operation.
func (s *Service) context(ctx context.Context, scope Scope, id string) (attemptContext, error) {
	raw, e := s.repo.call(ctx, "connection_context", scope.Actor, scope.Entity, scope.Book, id)
	if e != nil {
		return attemptContext{}, e
	}
	return decodeContext(raw, scope, id)
}

// decodeContext rejects cross-scope or malformed trusted routing metadata.
func decodeContext(raw []byte, scope Scope, id string) (attemptContext, error) {
	var c attemptContext
	if len(raw) > 65536 || json.Unmarshal(raw, &c) != nil || !validID(c.AttemptID) || c.AttemptID != id || c.EntityID != scope.Entity || c.BookID != scope.Book || !validID(c.Nonce) {
		return c, failed("invalid_metadata")
	}
	switch c.Mode {
	case "new", "reconnect", "rotate", "remove":
	default:
		return c, failed("invalid_metadata")
	}
	if _, e := time.Parse(time.RFC3339, c.ExpiresAt); e != nil {
		return c, failed("invalid_metadata")
	}
	return c, nil
}

// privateCall validates private SQL metadata without returning it to the browser.
func (s *Service) privateCall(ctx context.Context, scope Scope, id, op string, args ...any) (attemptContext, error) {
	raw, e := s.repo.call(ctx, op, args...)
	if e != nil {
		return attemptContext{}, e
	}
	return decodeContext(raw, scope, id)
}

// Begin creates one durable attempt before returning an ephemeral Link handoff.
func (s *Service) Begin(parent context.Context, scope Scope, key string, input BeginRequest) (BeginResult, error) {
	var out BeginResult
	if e := s.ready(scope); e != nil {
		return out, e
	}
	if support, ok := s.provider.(interface{ SupportsMode(string) bool }); ok && !support.SupportsMode(input.Mode) {
		return out, failed("unsupported_connection_mode")
	}
	if !keyRE.MatchString(key) {
		return out, failed("invalid_request")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	op := "begin_connection"
	args := []any{scope.Actor, scope.Entity, scope.Book, key, jsonText(input)}
	if s.creatorLogins != nil {
		op = "creator_begin_connection"
		args = append(args, s.creatorLogins.Sync, s.creatorLogins.Publisher)
	}
	raw, e := s.repo.call(ctx, op, args...)
	if e != nil {
		return out, e
	}
	var identity struct {
		AttemptID string `json:"attempt_id"`
	}
	if json.Unmarshal(raw, &identity) != nil {
		return out, failed("invalid_metadata")
	}
	c, e := decodeContext(raw, scope, identity.AttemptID)
	if e != nil {
		return out, e
	}
	if c.ExecuteLink {
		request := LinkRequest{ActorID: scope.Actor, Mode: c.Mode}
		if c.Mode == "reconnect" {
			request.AccessToken, e = s.readPrevious(ctx, scope, c)
			if e != nil {
				return out, e
			}
		}
		if _, e = s.context(ctx, scope, c.AttemptID); e != nil {
			return out, e
		}
		link, e := s.provider.CreateLink(ctx, request)
		if e != nil || ctx.Err() != nil {
			return out, failed("temporarily_unavailable")
		}
		attemptExpiry, _ := time.Parse(time.RFC3339, c.ExpiresAt)
		if attemptExpiry.Before(link.ExpiresAt) {
			link.ExpiresAt = attemptExpiry
		}
		if !secretValid(link.Token) || !link.ExpiresAt.After(time.Now()) {
			return out, failed("invalid_metadata")
		}
		if link.HostedURL.value != "" && !validHostedURL(link.HostedURL.value) {
			return out, failed("invalid_metadata")
		}
		_, e = s.privateCall(ctx, scope, c.AttemptID, "record_link_session", scope.Actor, scope.Entity, scope.Book, c.AttemptID, c.Nonce, jsonText(map[string]any{"link_token_sha256": hash(link.Token.value), "expires_at": link.ExpiresAt.UTC().Format(time.RFC3339)}))
		if e != nil {
			return out, e
		}
		if link.HostedURL.value != "" {
			store, ok := s.secretStore(scope).(LinkHandoffStore)
			if !ok || store.PutLink(ctx, c.AttemptID, link.Token) != nil {
				return out, failed("temporarily_unavailable")
			}
		}
		out.HostedLinkURL = link.HostedURL.value
		out.LinkToken = link.Token.value
		out.ExpiresAt = link.ExpiresAt.UTC().Format(time.RFC3339)
	}
	out.Attempt, e = s.Attempt(ctx, scope, c.AttemptID)
	if e != nil {
		return BeginResult{}, e
	}
	out.Nonce = c.Nonce
	return out, nil
}

// CompleteHosted resolves a retained Link session under current SQL authority after a reload.
// Execute still owns the one-shot provider claim, so an uncertain exchange is never replayed.
func (s *Service) CompleteHosted(ctx context.Context, scope Scope, id string) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if !validID(id) {
		return nil, failed("invalid_request")
	}
	c, e := s.context(ctx, scope, id)
	if e != nil {
		slog.Warn("hosted completion unavailable", "stage", "attempt_context", "attempt_id", id)
		return nil, e
	}
	if c.Mode != "new" || c.State != "link_ready" {
		return s.Attempt(ctx, scope, id)
	}
	store, ok := s.secretStore(scope).(LinkHandoffStore)
	if !ok {
		return nil, failed("configuration_error")
	}
	link, e := store.ReadLink(ctx, id)
	if e != nil {
		slog.Warn("hosted completion unavailable", "stage", "handoff_read", "attempt_id", id)
		return nil, failed("temporarily_unavailable")
	}
	var token string
	if e = link.WithValue(func(v string) error { token = v; return nil }); e != nil || !secretValid(link) || hash(token) != value(c.LinkHash) {
		slog.Warn("hosted completion unavailable", "stage", "handoff_validation", "attempt_id", id)
		return nil, failed("invalid_metadata")
	}
	result, err := s.Execute(ctx, scope, id, ExecuteRequest{Nonce: c.Nonce, LinkToken: token})
	if err != nil {
		slog.Warn("hosted completion unavailable", "stage", "execute", "attempt_id", id)
	}
	return result, err
}

// Execute consumes a confirmed one-shot claim; replay never performs a remote mutation.
func (s *Service) Execute(parent context.Context, scope Scope, id string, input ExecuteRequest) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if !validID(id) || !validID(input.Nonce) || input.LinkToken != "" && !secretValid(NewSecret(input.LinkToken)) || input.PublicToken != "" && !secretValid(NewSecret(input.PublicToken)) {
		return nil, failed("invalid_request")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	before, e := s.context(ctx, scope, id)
	if e != nil {
		return nil, e
	}
	if support, ok := s.provider.(interface{ SupportsMode(string) bool }); ok && !support.SupportsMode(before.Mode) {
		return nil, failed("unsupported_connection_mode")
	}
	// Hosted completion resolves public material only after current scoped SQL authority.
	// A completed/recovery attempt must never perform another provider lookup or exchange.
	if before.Mode == "new" && input.PublicToken == "" {
		if input.Nonce != before.Nonce || input.LinkToken == "" || value(before.LinkHash) != hash(input.LinkToken) {
			return nil, failed("connection_conflict")
		}
		if before.State != "link_ready" {
			return s.Attempt(ctx, scope, id)
		}
		expiry, _ := time.Parse(time.RFC3339, before.ExpiresAt)
		if !expiry.After(time.Now()) {
			return nil, failed("stale_connection")
		}
		resolver, ok := s.provider.(interface {
			CompletedPublicToken(context.Context, Secret) (Secret, error)
		})
		if !ok {
			return nil, failed("unsupported_connection_mode")
		}
		token, err := resolver.CompletedPublicToken(ctx, NewSecret(input.LinkToken))
		if err != nil {
			var failure *Failure
			if errors.As(err, &failure) && failure.Code == "link_session_pending" {
				return s.Attempt(ctx, scope, id)
			}
			slog.Warn("hosted completion unavailable", "stage", "plaid_link_status", "attempt_id", id)
			return nil, failed("temporarily_unavailable")
		}
		if !secretValid(token) {
			return nil, failed("invalid_metadata")
		}
		input.PublicToken = token.value
	}
	// Probe database-store wrapping capability before the irreversible provider claim.
	if before.ClaimID == nil && (before.Mode == "new" || before.Mode == "rotate") {
		if checker, ok := s.secretStore(scope).(interface{ Check(context.Context) error }); ok {
			if checker.Check(ctx) != nil {
				if before.Mode == "new" {
					slog.Warn("hosted completion unavailable", "stage", "credential_preflight", "attempt_id", id)
				}
				return nil, failed("temporarily_unavailable")
			}
		}
	}
	var lh, ph *string
	if input.LinkToken != "" {
		v := hash(input.LinkToken)
		lh = &v
	}
	if input.PublicToken != "" {
		v := hash(input.PublicToken)
		ph = &v
	}
	c, e := s.privateCall(ctx, scope, id, "claim_connection", scope.Actor, scope.Entity, scope.Book, id, input.Nonce, jsonText(map[string]any{"link_token_sha256": lh, "public_token_sha256": ph}))
	if e != nil {
		return nil, e
	}
	if !c.ExecuteProvider {
		return s.Attempt(ctx, scope, id)
	}
	if !validID(value(c.ClaimID)) {
		return nil, failed("invalid_metadata")
	}
	if c.Mode == "new" || c.Mode == "reconnect" {
		if input.LinkToken == "" || c.Mode == "new" && input.PublicToken == "" || value(c.LinkHash) != hash(input.LinkToken) {
			return s.recovery(ctx, scope, c, "manifest_rejected")
		}
		if e = s.provider.VerifySession(ctx, NewSecret(input.LinkToken), NewSecret(input.PublicToken), value(c.ExpectedItemID)); e != nil {
			return s.recovery(ctx, scope, c, "provider_rejected")
		}
	}
	if _, e = s.context(ctx, scope, id); e != nil {
		return nil, e
	}
	var credential ExchangeResult
	if c.Mode == "new" {
		credential, e = s.provider.Exchange(ctx, NewSecret(input.PublicToken))
	} else {
		var old Secret
		old, e = s.readPrevious(ctx, scope, c)
		if e != nil {
			return s.recovery(ctx, scope, c, "secret_unknown")
		}
		if _, e = s.context(ctx, scope, id); e != nil {
			return nil, e
		}
		switch c.Mode {
		case "remove":
			if e = s.provider.Remove(ctx, old); e != nil || ctx.Err() != nil {
				return s.recovery(ctx, scope, c, "provider_unknown")
			}
			if _, e = s.repo.call(ctx, "confirm_item_removed", scope.Actor, scope.Entity, scope.Book, id, value(c.ClaimID)); e != nil {
				return nil, e
			}
			return s.Attempt(ctx, scope, id)
		case "rotate":
			credential, e = s.provider.Rotate(ctx, old)
		case "reconnect":
			credential = ExchangeResult{ItemID: value(c.ExpectedItemID), AccessToken: old}
		}
	}
	if e != nil || ctx.Err() != nil {
		return s.recovery(ctx, scope, c, "provider_unknown")
	}
	if !opaqueRE.MatchString(credential.ItemID) || !secretValid(credential.AccessToken) || c.Mode != "new" && credential.ItemID != value(c.ExpectedItemID) {
		return s.recovery(ctx, scope, c, "manifest_rejected")
	}
	if _, e = s.context(ctx, scope, id); e != nil {
		return nil, e
	}
	manifest, e := s.provider.Manifest(ctx, credential.AccessToken)
	if e != nil {
		return s.recovery(ctx, scope, c, "provider_unknown")
	}
	if !validManifest(manifest) || manifest.ItemID != credential.ItemID {
		return s.recovery(ctx, scope, c, "manifest_rejected")
	}
	c, e = s.privateCall(ctx, scope, id, "record_connection_manifest", scope.Actor, scope.Entity, scope.Book, id, value(c.ClaimID), jsonText(manifest))
	if e != nil {
		return nil, e
	}
	ref, e := reference(c, false)
	if e != nil {
		return nil, e
	}
	if _, e = s.context(ctx, scope, id); e != nil {
		return nil, e
	}
	if e = s.secretStore(scope).Put(ctx, ref, credential.AccessToken); e != nil {
		return s.recovery(ctx, scope, c, "secret_unknown")
	}
	if _, e = s.context(ctx, scope, id); e != nil {
		return nil, e
	}
	stored, e := s.secretStore(scope).Read(ctx, ref)
	if e != nil || !matches(stored, ref) || subtle.ConstantTimeCompare([]byte(stored.Token.value), []byte(credential.AccessToken.value)) != 1 {
		return s.recovery(ctx, scope, c, "secret_unknown")
	}
	return s.confirm(ctx, scope, c)
}

// Reconcile only reads a known immutable secret version; exchange/rotation/removal are never repeated.
func (s *Service) Reconcile(parent context.Context, scope Scope, id string) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if !validID(id) {
		return nil, failed("invalid_request")
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	c, e := s.context(ctx, scope, id)
	if e != nil {
		return nil, e
	}
	if c.State == "active" || c.State == "binding_review" || c.State == "disconnected" {
		return s.Attempt(ctx, scope, id)
	}
	if c.Manifest == nil || !validID(value(c.ClaimID)) || c.Mode == "remove" {
		return nil, failed("credential_recovery_required")
	}
	ref, e := reference(c, false)
	if e != nil {
		return nil, e
	}
	stored, e := s.secretStore(scope).Read(ctx, ref)
	if e != nil || !matches(stored, ref) {
		return nil, failed("credential_recovery_required")
	}
	if _, e = s.context(ctx, scope, id); e != nil {
		return nil, e
	}
	m, e := s.provider.Manifest(ctx, stored.Token)
	if e != nil || !validManifest(m) || !sameManifest(m, *c.Manifest) {
		return nil, failed("credential_recovery_required")
	}
	return s.confirm(ctx, scope, c)
}

// confirm publishes a credential reference only through current SQL authority.
func (s *Service) confirm(ctx context.Context, scope Scope, c attemptContext) (json.RawMessage, error) {
	if _, e := s.repo.call(ctx, "confirm_connection_credential", scope.Actor, scope.Entity, scope.Book, c.AttemptID, value(c.ClaimID)); e != nil {
		return nil, e
	}
	return s.Attempt(ctx, scope, c.AttemptID)
}

// recovery annotates uncertainty without permitting another remote mutation.
func (s *Service) recovery(ctx context.Context, scope Scope, c attemptContext, reason string) (json.RawMessage, error) {
	// The durable claim already forbids replay even if cancellation prevents this status annotation.
	_, _ = s.repo.call(ctx, "connection_failure", scope.Actor, scope.Entity, scope.Book, c.AttemptID, value(c.ClaimID), reason)
	return nil, failed("credential_recovery_required")
}

// reference requires an exact SQL-selected immutable credential version and Item.
func reference(c attemptContext, old bool) (CredentialReference, error) {
	r := CredentialReference{ReferenceID: value(c.CredentialRef), Version: value(c.CredentialVersion), Environment: "sandbox"}
	if c.Manifest != nil {
		r.ItemID = c.Manifest.ItemID
	}
	if old {
		r.ReferenceID = value(c.PreviousCredentialRef)
		r.Version = value(c.PreviousCredentialVersion)
		r.ItemID = value(c.ExpectedItemID)
	}
	if !validID(r.ReferenceID) || !validRevision(r.Version) || !opaqueRE.MatchString(r.ItemID) {
		return r, failed("invalid_metadata")
	}
	return r, nil
}

// matches validates the complete addressed secret identity and bounded token.
func matches(s StoredCredential, r CredentialReference) bool {
	return s.Reference == r && secretValid(s.Token)
}

// readPrevious reauthorizes before resolving a shared Item credential.
func (s *Service) readPrevious(ctx context.Context, scope Scope, c attemptContext) (Secret, error) {
	if _, e := s.context(ctx, scope, c.AttemptID); e != nil {
		return Secret{}, e
	}
	r, e := reference(c, true)
	if e != nil {
		return Secret{}, e
	}
	v, e := s.secretStore(scope).Read(ctx, r)
	if e != nil || !matches(v, r) {
		return Secret{}, failed("credential_recovery_required")
	}
	return v.Token, nil
}

// validManifest checks bounded provider metadata without inferring source ownership.
func validManifest(m Manifest) bool {
	if !opaqueRE.MatchString(m.ItemID) || len(m.Accounts) == 0 || len(m.Accounts) > 25 {
		return false
	}
	expiry, e := time.Parse(time.RFC3339, m.ConsentExpiresAt)
	if e != nil || !expiry.After(time.Now()) {
		return false
	}
	seen := map[string]bool{}
	for _, a := range m.Accounts {
		if !opaqueRE.MatchString(a.AccountID) || seen[a.AccountID] {
			return false
		}
		seen[a.AccountID] = true
		switch a.Type {
		case "depository", "credit", "loan", "investment", "other", "payroll":
		default:
			return false
		}
		if a.Subtype != nil && !subtypeRE.MatchString(*a.Subtype) {
			return false
		}
		if a.Currency != nil && !currencyRE.MatchString(*a.Currency) {
			return false
		}
	}
	return true
}

// sameManifest compares complete metadata independently of account array order.
func sameManifest(a, b Manifest) bool {
	a.Accounts = append([]Account(nil), a.Accounts...)
	b.Accounts = append([]Account(nil), b.Accounts...)
	sort.Slice(a.Accounts, func(i, j int) bool { return a.Accounts[i].AccountID < a.Accounts[j].AccountID })
	sort.Slice(b.Accounts, func(i, j int) bool { return b.Accounts[i].AccountID < b.Accounts[j].AccountID })
	return jsonText(a) == jsonText(b)
}

// Public read/mapping methods never expose the trusted credential context.
func (s *Service) Attempt(ctx context.Context, scope Scope, id string) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if !validID(id) {
		return nil, failed("invalid_request")
	}
	return s.repo.call(ctx, "get_connection_attempt", scope.Actor, scope.Entity, scope.Book, id)
}

// List exposes only the SQL-owned public connection projection.
func (s *Service) List(ctx context.Context, scope Scope) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if s.creatorLogins != nil {
		return s.repo.call(ctx, "creator_list_connections", scope.Actor, scope.Entity, scope.Book, s.creatorLogins.Sync, s.creatorLogins.Publisher)
	}
	return s.repo.call(ctx, "list_connections", scope.Actor, scope.Entity, scope.Book)
}

// Bind submits an explicit reviewed account mapping under current SQL authority.
func (s *Service) Bind(ctx context.Context, scope Scope, id, key string, input BindingRequest) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if !validID(id) || !keyRE.MatchString(key) || !validBindingRequest(input) {
		return nil, failed("invalid_request")
	}
	return s.repo.call(ctx, "bind_connection", scope.Actor, scope.Entity, scope.Book, id, key, jsonText(input))
}

// validBindingRequest rejects malformed automatic account definitions before SQL mutation.
func validBindingRequest(input BindingRequest) bool {
	if !input.ConsentGrants || len(input.Bindings) < 1 || len(input.Bindings) > 25 {
		return false
	}
	for _, binding := range input.Bindings {
		if binding.NewLedgerAccount == nil {
			continue
		}
		account := binding.NewLedgerAccount
		if !validID(binding.LedgerAccountID) || account.AccountID != binding.LedgerAccountID ||
			len(account.Code) < 1 || len(account.Code) > 64 || len(account.Name) < 1 || len(account.Name) > 200 ||
			account.Category != "asset" || account.NormalSide != "debit" || !account.Monetary || account.ActiveFrom != binding.BindingDate {
			return false
		}
		if _, err := time.Parse("2006-01-02", account.ActiveFrom); err != nil {
			return false
		}
	}
	return true
}

// Disconnect fences one stream and never performs an Item-wide provider operation.
func (s *Service) Disconnect(ctx context.Context, scope Scope, id, key, revision string) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if !validID(id) || !keyRE.MatchString(key) || !validRevision(revision) {
		return nil, failed("invalid_request")
	}
	return s.repo.call(ctx, "disconnect_stream", scope.Actor, scope.Entity, scope.Book, id, key, revision)
}

// secretStore binds database-backed storage to the current verified operation scope.
func (s *Service) secretStore(scope Scope) SecretStore {
	if factory, ok := s.secrets.(interface{ ForScope(Scope) SecretStore }); ok {
		return factory.ForScope(scope)
	}
	return s.secrets
}

// Abandon serializes explicit new-setup cancellation against exchange and stored credentials.
func (s *Service) Abandon(ctx context.Context, scope Scope, id string) (json.RawMessage, error) {
	if e := s.ready(scope); e != nil {
		return nil, e
	}
	if !validID(id) {
		return nil, failed("invalid_request")
	}
	return s.repo.call(ctx, "abandon_connection", scope.Actor, scope.Entity, scope.Book, id)
}
