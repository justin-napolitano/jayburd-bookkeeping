package connections

import (
	"context"
	"encoding/json"
	"errors"
)

// RegistrationSelection carries only verified account identities; SQL proposes and persists every mapping.
type RegistrationSelection struct {
	AccountIDs []string `json:"account_ids"`
}

// PrepareRegistration persists one reviewed plan without binding, provider calls or credential reads.
func (s *Service) PrepareRegistration(ctx context.Context, scope Scope, id, key, request string, input RegistrationSelection) (json.RawMessage, error) {
	if err := s.ready(scope); err != nil {
		return nil, err
	}
	if !validID(id) || !validID(request) || !keyRE.MatchString(key) || len(input.AccountIDs) < 1 || len(input.AccountIDs) > 25 {
		return nil, failed("invalid_request")
	}
	seen := make(map[string]bool, len(input.AccountIDs))
	for _, account := range input.AccountIDs {
		if !opaqueRE.MatchString(account) || seen[account] {
			return nil, failed("invalid_request")
		}
		seen[account] = true
	}
	// Current private context enforces runtime/profile/key matching before durable work; SQL rechecks at mutation.
	if _, err := s.context(ctx, scope, id); err != nil {
		var f *Failure
		if errors.As(err, &f) && f.Code == "invalid_metadata" {
			return nil, failed("temporarily_unavailable")
		}
		return nil, err
	}
	result, err := s.repo.call(ctx, "prepare_registration", scope.Actor, scope.Entity, scope.Book, id, key, request, jsonText(input))
	var f *Failure
	if errors.As(err, &f) && f.Code == "credential_recovery_required" {
		return nil, failed("registration_not_ready")
	}
	return result, err
}
