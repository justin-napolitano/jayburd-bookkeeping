package connections

import (
	"context"
	"encoding/json"
	"regexp"
	"time"
)

var repairDigestRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// repairProof reads owned current SQL evidence only; receipt status never grants bank work by itself.
func (s *Service) repairProof(ctx context.Context, scope Scope, id string) (bool, error) {
	raw, e := s.repo.call(ctx, "connection_link_proof", scope.Actor, scope.Entity, scope.Book, id)
	if e != nil {
		return false, e
	}
	var proof struct {
		Status  string  `json:"status"`
		Receipt *string `json:"receipt_key"`
	}
	if json.Unmarshal(raw, &proof) != nil {
		return false, failed("invalid_metadata")
	}
	switch proof.Status {
	case "pending":
		return false, nil
	case "success":
		if !repairDigestRE.MatchString(value(proof.Receipt)) {
			return false, failed("invalid_metadata")
		}
		return true, nil
	case "exited", "ambiguous":
		return false, failed("connection_operation_blocked")
	default:
		return false, failed("invalid_metadata")
	}
}

// completeHostedRepair checks proof before even decrypting the retained Link handoff.
// An existing claim is recovered explicitly, never replaced by a new Link or exchange.
func (s *Service) completeHostedRepair(ctx context.Context, scope Scope, c attemptContext) (json.RawMessage, error) {
	if support, ok := s.provider.(interface{ SupportsMode(string) bool }); ok && !support.SupportsMode("reconnect") {
		return nil, failed("unsupported_connection_mode")
	}
	if c.State == "active" || c.State == "abandoned" {
		return s.Attempt(ctx, scope, c.AttemptID)
	}
	if c.ClaimID != nil {
		return s.finishRepair(ctx, scope, c)
	}
	if c.State != "link_ready" {
		return s.Attempt(ctx, scope, c.AttemptID)
	}
	ready, e := s.repairProof(ctx, scope, c.AttemptID)
	if e != nil {
		return nil, e
	}
	if !ready {
		return s.Attempt(ctx, scope, c.AttemptID)
	}
	expiry, e := time.Parse(time.RFC3339, c.ExpiresAt)
	if e != nil || !expiry.After(time.Now()) {
		return nil, failed("stale_connection")
	}
	ctx, cancel := context.WithDeadline(ctx, expiry)
	defer cancel()
	handoff, ok := s.secretStore(scope).(LinkHandoffStore)
	if !ok {
		return nil, failed("configuration_error")
	}
	if _, ok := s.secretStore(scope).(RepairCredentialReader); !ok {
		return nil, failed("configuration_error")
	}
	link, e := handoff.ReadLink(ctx, c.AttemptID)
	if e != nil {
		return nil, failed("temporarily_unavailable")
	}
	if !secretValid(link) || hash(link.value) != value(c.LinkHash) {
		return nil, failed("invalid_metadata")
	}
	return s.executeRepair(ctx, scope, c, ExecuteRequest{Nonce: c.Nonce, LinkToken: link.value})
}

// executeRepair obtains the proof-pinned SQL claim without public-token resolution or provider verification.
func (s *Service) executeRepair(ctx context.Context, scope Scope, before attemptContext, input ExecuteRequest) (json.RawMessage, error) {
	if input.PublicToken != "" || input.Nonce != before.Nonce || input.LinkToken == "" || hash(input.LinkToken) != value(before.LinkHash) {
		return nil, failed("connection_conflict")
	}
	if before.ClaimID != nil || before.State != "link_ready" {
		return s.Attempt(ctx, scope, before.AttemptID)
	}
	ready, e := s.repairProof(ctx, scope, before.AttemptID)
	if e != nil {
		return nil, e
	}
	if !ready {
		return s.Attempt(ctx, scope, before.AttemptID)
	}
	// Refuse a known incompatible store before consuming the SQL claim or fencing
	// the Item; this is a capability check, never a bank read or wrapping preflight.
	if _, ok := s.secretStore(scope).(RepairCredentialReader); !ok {
		return nil, failed("configuration_error")
	}
	c, e := s.privateCall(ctx, scope, before.AttemptID, "claim_connection", scope.Actor, scope.Entity, scope.Book, before.AttemptID, input.Nonce,
		jsonText(map[string]any{"link_token_sha256": hash(input.LinkToken), "public_token_sha256": nil}))
	if e != nil {
		return nil, e
	}
	if c.Mode != "reconnect" || c.Nonce != before.Nonce || value(c.LinkHash) != hash(input.LinkToken) {
		return nil, failed("invalid_metadata")
	}
	if !c.ExecuteProvider {
		return s.Attempt(ctx, scope, c.AttemptID)
	}
	return s.finishRepair(ctx, scope, c)
}

// finishRepair repeats only an owned proof-pinned read-only manifest and exact confirmation.
// Missing manifest after a lost claim reply is recoverable; no encrypted credential is written or rotated.
func (s *Service) finishRepair(ctx context.Context, scope Scope, c attemptContext) (json.RawMessage, error) {
	if support, ok := s.provider.(interface{ SupportsMode(string) bool }); ok && !support.SupportsMode("reconnect") {
		return nil, failed("unsupported_connection_mode")
	}
	if c.Mode != "reconnect" || !validID(value(c.ClaimID)) {
		return nil, failed("credential_recovery_required")
	}
	ready, e := s.repairProof(ctx, scope, c.AttemptID)
	if e != nil {
		return nil, e
	}
	if !ready {
		return nil, failed("credential_recovery_required")
	}
	ref, e := reference(c, true)
	if e != nil || value(c.CredentialRef) != ref.ReferenceID || value(c.CredentialVersion) != ref.Version {
		return nil, failed("invalid_metadata")
	}
	ceiling, e := time.Parse(time.RFC3339, c.ItemConsentExpiresAt)
	if e != nil || !ceiling.After(time.Now()) {
		return nil, failed("stale_connection")
	}
	if c.Manifest != nil {
		known, e := time.Parse(time.RFC3339, c.Manifest.ConsentExpiresAt)
		if e != nil || !known.After(time.Now()) {
			return nil, failed("stale_connection")
		}
		if known.Before(ceiling) {
			ceiling = known
		}
	}
	ctx, cancel := context.WithDeadline(ctx, ceiling)
	defer cancel()
	old, e := s.readPrevious(ctx, scope, c)
	if e != nil {
		return nil, e
	}
	if _, e = s.context(ctx, scope, c.AttemptID); e != nil {
		return nil, e
	}
	m, e := s.provider.Manifest(ctx, old)
	if e != nil || ctx.Err() != nil {
		return s.recovery(ctx, scope, c, "provider_unknown")
	}
	if !validManifest(m) || m.ItemID != ref.ItemID {
		return s.recovery(ctx, scope, c, "manifest_rejected")
	}
	expiry, _ := time.Parse(time.RFC3339, m.ConsentExpiresAt)
	if ceiling.Before(expiry) {
		m.ConsentExpiresAt = ceiling.UTC().Format(time.RFC3339Nano)
	}
	if c.Manifest != nil {
		if !sameManifest(m, *c.Manifest) {
			return nil, failed("credential_recovery_required")
		}
		// Preserve the original immutable manifest receipt. A current read-only
		// account-order-independent match needs confirmation, not another record.
		return s.confirm(ctx, scope, c)
	}
	c, e = s.privateCall(ctx, scope, c.AttemptID, "record_connection_manifest", scope.Actor, scope.Entity, scope.Book, c.AttemptID, value(c.ClaimID), jsonText(m))
	if e != nil {
		return nil, e
	}
	return s.confirm(ctx, scope, c)
}
