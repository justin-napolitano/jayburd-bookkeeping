package connections

import (
	"context"
	"encoding/json"
)

// StreamOperation pins independent stream lifecycle, Item and source binding revisions.
type StreamOperation struct {
	Action                  string `json:"action"`
	ExpectedItemGeneration  string `json:"expected_item_generation"`
	ExpectedBindingRevision string `json:"expected_binding_revision"`
	ExpectedStreamRevision  string `json:"expected_stream_revision"`
}

// readReady needs only SQL; status never requires provider credentials or network access.
func (s *Service) readReady(scope Scope) error {
	if s == nil || s.repo == nil {
		return failed("configuration_error")
	}
	if !validScope(scope) {
		return failed("invalid_request")
	}
	return nil
}

// Attempts discovers this actor's durable attempts without exposing private handoff material.
func (s *Service) Attempts(ctx context.Context, scope Scope, limit int, after *string) (json.RawMessage, error) {
	if err := s.readReady(scope); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 || after != nil && !validID(*after) {
		return nil, failed("invalid_request")
	}
	if s.creatorLogins != nil {
		return s.repo.call(ctx, "creator_list_connection_attempts", scope.Actor, scope.Entity, scope.Book, limit, after, s.creatorLogins.Sync, s.creatorLogins.Publisher)
	}
	return s.repo.call(ctx, "list_connection_attempts", scope.Actor, scope.Entity, scope.Book, limit, after)
}

// StreamStatus reads publication evidence and current authority without starting acquisition.
func (s *Service) StreamStatus(ctx context.Context, scope Scope, stream string) (json.RawMessage, error) {
	if err := s.readReady(scope); err != nil {
		return nil, err
	}
	if !validID(stream) {
		return nil, failed("invalid_request")
	}
	return s.repo.call(ctx, "connection_stream_status", scope.Actor, scope.Entity, scope.Book, stream)
}

// OperateStream persists an explicit scoped control; a request receipt never means data was fetched.
func (s *Service) OperateStream(ctx context.Context, scope Scope, stream, key string, input StreamOperation) (json.RawMessage, error) {
	if err := s.readReady(scope); err != nil {
		return nil, err
	}
	if !validID(stream) || !keyRE.MatchString(key) || !validRevision(input.ExpectedItemGeneration) || !validRevision(input.ExpectedBindingRevision) || !validRevision(input.ExpectedStreamRevision) {
		return nil, failed("invalid_request")
	}
	switch input.Action {
	case "pause", "resume", "request":
	default:
		return nil, failed("invalid_request")
	}
	return s.repo.call(ctx, "mutate_connection_stream", scope.Actor, scope.Entity, scope.Book, stream, key, jsonText(input))
}
