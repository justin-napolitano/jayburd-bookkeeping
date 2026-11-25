package autopost

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// workScope is trusted SQL-selected authority, never a caller-supplied worker identity.
type workScope struct {
	Actor  string `json:"actor_id"`
	Entity string `json:"entity_id"`
	Book   string `json:"book_id"`
}

// Run owns bounded durable rounds and cancellation only; it never acquires source/provider data.
func (s *Service) Run(ctx context.Context) {
	for ctx.Err() == nil {
		_ = s.Round(ctx)
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// Round is also the synthetic composition proof entrypoint; SQL selects every trusted scope.
func (s *Service) Round(ctx context.Context) error {
	raw, err := s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var raw json.RawMessage
		err := tx.QueryRow(ctx, "select banking.automatic_accounting_work()").Scan(&raw)
		return raw, err
	})
	if err != nil {
		return err
	}
	m, ok := object(raw, "profile", "items")
	if !ok || str(m["profile"]) != "automatic-accounting-work-v1" {
		return unavailable()
	}
	items, ok := list(m["items"], 10)
	if !ok {
		return unavailable()
	}
	var first error
	seen := map[string]bool{}
	for _, raw := range items {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		m, ok := object(raw, "actor_id", "entity_id", "book_id")
		if !ok || !id(m["actor_id"]) || !id(m["entity_id"]) || !id(m["book_id"]) {
			return unavailable()
		}
		var scope workScope
		if json.Unmarshal(raw, &scope) != nil {
			return unavailable()
		}
		key := scope.Entity + "/" + scope.Book
		if seen[key] {
			return unavailable()
		}
		seen[key] = true
		if err := s.scope(ctx, scope); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// scope recovers one durable identity and consumes its exact terminal receipt before new work.
func (s *Service) scope(ctx context.Context, w workScope) error {
	raw, err := s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var raw json.RawMessage
		err := tx.QueryRow(ctx, "select banking.prepare_automatic_accounting($1::uuid,$2::uuid,$3::uuid)", w.Actor, w.Entity, w.Book).Scan(&raw)
		return raw, err
	})
	if err != nil {
		return err
	}
	m, ok := object(raw, "profile", "entity_id", "book_id", "state", "operation", "last_operation", "blocked_reason")
	if !ok || str(m["profile"]) != "automatic-accounting-next-v1" || str(m["entity_id"]) != w.Entity || str(m["book_id"]) != w.Book || !nullable(m["last_operation"], validOperation) || !blockedReason(m["blocked_reason"]) {
		return unavailable()
	}
	state := str(m["state"])
	switch state {
	case "idle", "blocked":
		if string(m["operation"]) != "null" {
			return unavailable()
		}
		return nil
	case "held", "stale", "committed", "prepared":
		if !validOperation(m["operation"]) {
			return unavailable()
		}
	default:
		return unavailable()
	}
	op, _ := object(m["operation"], "operation_id", "state", "review_id", "journal_id", "reason")
	if str(op["state"]) != state {
		return unavailable()
	}
	operationID := str(op["operation_id"])
	if state != "prepared" {
		return s.observe(ctx, w, operationID)
	}
	executed, err := s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var result json.RawMessage
		if err := tx.QueryRow(ctx, "select banking.execute_automatic_accounting($1::uuid,$2::uuid,$3::uuid,$4::uuid)", w.Actor, w.Entity, w.Book, operationID).Scan(&result); err != nil {
			return nil, err
		}
		m, ok := object(result, "profile", "entity_id", "book_id", "operation", "blocked_reason")
		if !ok || str(m["profile"]) != "automatic-accounting-execution-v1" || str(m["entity_id"]) != w.Entity || str(m["book_id"]) != w.Book || !validOperation(m["operation"]) || !blockedReason(m["blocked_reason"]) {
			return nil, unavailable()
		}
		op, _ := object(m["operation"], "operation_id", "state", "review_id", "journal_id", "reason")
		if str(op["operation_id"]) != operationID {
			return nil, unavailable()
		}
		return result, nil
	})
	if err != nil {
		return err
	}
	execution, _ := object(executed, "profile", "entity_id", "book_id", "operation", "blocked_reason")
	completed, _ := object(execution["operation"], "operation_id", "state", "review_id", "journal_id", "reason")
	if str(completed["state"]) == "prepared" {
		return nil
	}
	return s.observe(ctx, w, operationID)
}

// observe records machine consumption only after a validated terminal response committed successfully.
func (s *Service) observe(ctx context.Context, w workScope, operationID string) error {
	_, err := s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var result json.RawMessage
		if err := tx.QueryRow(ctx, "select banking.observe_automatic_accounting($1::uuid,$2::uuid,$3::uuid,$4::uuid)", w.Actor, w.Entity, w.Book, operationID).Scan(&result); err != nil {
			return nil, err
		}
		m, ok := object(result, "profile", "entity_id", "book_id", "operation", "blocked_reason")
		if !ok || str(m["profile"]) != "automatic-accounting-execution-v1" || str(m["entity_id"]) != w.Entity || str(m["book_id"]) != w.Book || !validOperation(m["operation"]) || string(m["blocked_reason"]) != "null" {
			return nil, unavailable()
		}
		op, _ := object(m["operation"], "operation_id", "state", "review_id", "journal_id", "reason")
		if str(op["operation_id"]) != operationID || str(op["state"]) == "prepared" {
			return nil, unavailable()
		}
		return result, nil
	})
	return err
}
