package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// AutomationOperations configures SQL-owned policy under independently verified identity.
type AutomationOperations interface {
	GetAutomation(context.Context, Request) (json.RawMessage, error)
	SetAutomation(context.Context, Request) (json.RawMessage, error)
}

// AutomationService preserves closed raw settings; it cannot choose computed category results.
type AutomationService struct{ DB ledger.Beginner }

// GetAutomation reads current authorized policy without starting classification.
func (s *AutomationService) GetAutomation(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.settings(ctx, r, false)
}

// SetAutomation leaves idempotency, closed input and current grants to SQL.
func (s *AutomationService) SetAutomation(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.settings(ctx, r, true)
}

// settings returns bounded metadata only after an acknowledged transaction.
func (s *AutomationService) settings(ctx context.Context, r Request, writing bool) (json.RawMessage, error) {
	if s == nil {
		return nil, unavailable()
	}
	iso := pgx.RepeatableRead
	if writing {
		iso = pgx.Serializable
	}
	return automaticTransaction(ctx, s.DB, iso, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var body json.RawMessage
		var err error
		if writing {
			err = tx.QueryRow(ctx, "select banking.set_collection_automation($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
		} else {
			err = tx.QueryRow(ctx, "select banking.get_collection_automation($1::uuid,$2::uuid)", r.Actor, r.Workspace).Scan(&body)
		}
		if err != nil {
			return nil, err
		}
		if !validAutomationStatus(body, r.Workspace) {
			return nil, unavailable()
		}
		return body, nil
	})
}

// AutomaticService runs trusted bounded work; PostgreSQL retains policy, plans and receipts.
// It reuses the existing Go matcher inside the same transaction as inert preparation.
type AutomaticService struct{ DB ledger.Beginner }

type automaticWorkspace struct {
	Principal string `json:"principal_id"`
	Workspace string `json:"workspace_id"`
}

// automaticWorkFailure distinguishes inventory outages from a recoverable workspace phase.
type automaticWorkFailure struct{ cause error }

func (e *automaticWorkFailure) Error() string { return "automatic work unavailable" }
func (e *automaticWorkFailure) Unwrap() error { return e.cause }

// Step processes at most ten SQL-selected workspaces. Any ambiguous phase stops this round;
// later context reads recover durable pending work rather than inventing a replacement key.
func (s *AutomaticService) Step(ctx context.Context) error {
	if s == nil {
		return unavailable()
	}
	var work struct {
		Profile    string               `json:"profile"`
		Workspaces []automaticWorkspace `json:"workspaces"`
	}
	_, err := automaticTransaction(ctx, s.DB, pgx.Serializable, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var raw json.RawMessage
		if err := tx.QueryRow(ctx, "select banking.collection_automation_work()").Scan(&raw); err != nil {
			return nil, err
		}
		if !closedAutomatic(raw, 16384, "profile", "workspaces") || json.Unmarshal(raw, &work) != nil || work.Profile != "private-collection-auto-work-v1" || work.Workspaces == nil || len(work.Workspaces) > 10 {
			return nil, unavailable()
		}
		seen := map[automaticWorkspace]bool{}
		for _, w := range work.Workspaces {
			if !validID(w.Principal) || !validID(w.Workspace) || seen[w] {
				return nil, unavailable()
			}
			seen[w] = true
		}
		return raw, nil
	})
	if err != nil {
		return &automaticWorkFailure{cause: err}
	}
	for _, w := range work.Workspaces {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.process(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

// process resolves one trusted workspace phase and never manufactures interactive work.
func (s *AutomaticService) process(ctx context.Context, w automaticWorkspace) error {
	_, err := automaticTransaction(ctx, s.DB, pgx.Serializable, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var raw json.RawMessage
		if err := tx.QueryRow(ctx, "select banking.collection_automation_context($1::uuid,$2::uuid)", w.Principal, w.Workspace).Scan(&raw); err != nil {
			return nil, err
		}
		var v struct {
			Profile   string          `json:"profile"`
			Principal string          `json:"principal_id"`
			Workspace string          `json:"workspace_id"`
			State     string          `json:"state"`
			RunID     *string         `json:"run_id"`
			Context   json.RawMessage `json:"context"`
			LastRun   json.RawMessage `json:"last_run"`
		}
		if !closedAutomatic(raw, 1<<20, "profile", "principal_id", "workspace_id", "state", "run_id", "context", "last_run") || json.Unmarshal(raw, &v) != nil || v.Profile != "private-collection-auto-context-v1" || v.Principal != w.Principal || v.Workspace != w.Workspace {
			return nil, unavailable()
		}
		// A retained receipt is checked before any new selection; an empty queue
		// never substitutes for confirmed identity after an unknown execute reply.
		if !validAutomaticLastRun(v.LastRun, w.Workspace) {
			return nil, unavailable()
		}
		var result json.RawMessage
		switch v.State {
		case "idle", "blocked":
			if v.RunID != nil || string(v.Context) != "null" {
				return nil, unavailable()
			}
			return raw, nil
		case "pending":
			if v.RunID == nil || !validID(*v.RunID) || string(v.Context) != "null" {
				return nil, unavailable()
			}
			if err := tx.QueryRow(ctx, "select banking.execute_collection_automatic_run($1::uuid,$2::uuid,$3::uuid)", w.Principal, w.Workspace, *v.RunID).Scan(&result); err != nil {
				return nil, err
			}
			if !validAutomaticRun(result, w.Workspace, *v.RunID, false) {
				return nil, unavailable()
			}
		case "ready":
			var c ruleContext
			if v.RunID != nil || json.Unmarshal(v.Context, &c) != nil || c.Profile != "private-rule-context-v1" || c.Principal != w.Principal || c.Workspace != w.Workspace || len(c.Facts) == 0 || len(c.Facts) > 100 || len(c.Rules) > 1000 {
				return nil, unavailable()
			}
			results, err := evaluateRuleContext(ctx, c, false)
			if err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(map[string]any{"results": results})
			if err != nil || len(encoded) > 1<<20 {
				return nil, unavailable()
			}
			if err := tx.QueryRow(ctx, "select banking.prepare_collection_automatic_run($1::uuid,$2::uuid,$3::text,$4::text)", w.Principal, w.Workspace, string(v.Context), string(encoded)).Scan(&result); err != nil {
				return nil, err
			}
			if !validAutomaticRun(result, w.Workspace, "", true) {
				return nil, unavailable()
			}
		default:
			return nil, unavailable()
		}
		return result, nil
	})
	return err
}

// Run periodically recovers durable work; cancellation finishes before the caller closes its pool.
// Fixed transient failures remain private and defer work to the next bounded round.
func (s *AutomaticService) Run(ctx context.Context) {
	for ctx.Err() == nil {
		delay := time.Second
		var inventory *automaticWorkFailure
		if errors.As(s.Step(ctx), &inventory) {
			delay = 10 * time.Second
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// automaticTransaction retries only definite whole-transaction aborts. Unknown COMMIT is never retried.
func automaticTransaction(parent context.Context, db ledger.Beginner, iso pgx.TxIsoLevel, fn func(context.Context, pgx.Tx) (json.RawMessage, error)) (json.RawMessage, error) {
	if db == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		body, err := func() (json.RawMessage, error) {
			tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: iso})
			if err != nil {
				return nil, err
			}
			defer func() {
				cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = tx.Rollback(cleanup)
			}()
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) < time.Millisecond {
				return nil, context.DeadlineExceeded
			}
			remaining := (time.Until(deadline) / time.Millisecond * time.Millisecond).String()
			if _, err := tx.Exec(ctx, "select set_config('statement_timeout',$1,true),set_config('lock_timeout',$1,true)", remaining); err != nil {
				return nil, err
			}
			body, err := fn(ctx, tx)
			if err != nil {
				return nil, err
			}
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			return body, nil
		}()
		if err == nil {
			return body, nil
		}
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || (pg.Code != "40001" && pg.Code != "40P01") {
			return nil, safeFailure(err)
		}
		if ctx.Err() != nil {
			return nil, unavailable()
		}
	}
	return nil, unavailable()
}

// closedAutomatic fences the SQL response envelope; public settings remain unchanged for SQL validation.
func closedAutomatic(raw json.RawMessage, limit int, keys ...string) bool {
	if len(raw) > limit || !json.Valid(raw) {
		return false
	}
	var v map[string]json.RawMessage
	if json.Unmarshal(raw, &v) != nil || len(v) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := v[key]; !ok {
			return false
		}
	}
	return true
}

// automaticCount retains exact bigint metadata as canonical decimal strings.
func automaticCount(v string) bool {
	n, err := strconv.ParseUint(v, 10, 63)
	return len(v) <= 19 && err == nil && strconv.FormatUint(n, 10) == v
}

// validAutomaticLastRun recognizes only an exact scoped receipt or explicit absence.
func validAutomaticLastRun(raw json.RawMessage, workspace string) bool {
	return string(raw) == "null" || validAutomaticRun(raw, workspace, "", true) || validAutomaticRun(raw, workspace, "", false)
}

// validAutomaticRun checks phase/result identity before allowing its transaction to commit.
func validAutomaticRun(raw json.RawMessage, workspace, id string, prepared bool) bool {
	if !closedAutomatic(raw, 4096, "profile", "workspace_id", "run_id", "state", "processed_count", "attention_count") {
		return false
	}
	var v struct {
		Profile   string `json:"profile"`
		Workspace string `json:"workspace_id"`
		ID        string `json:"run_id"`
		State     string `json:"state"`
		Processed string `json:"processed_count"`
		Attention string `json:"attention_count"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Profile != "private-collection-auto-run-v1" || v.Workspace != workspace || !validID(v.ID) || (id != "" && v.ID != id) || !automaticCount(v.Processed) || !automaticCount(v.Attention) {
		return false
	}
	return (prepared && v.State == "prepared") || (!prepared && (v.State == "committed" || v.State == "stale"))
}

// validAutomationStatus exposes a closed bounded status without source or command payloads.
func validAutomationStatus(raw json.RawMessage, workspace string) bool {
	if !closedAutomatic(raw, 16384, "profile", "workspace_id", "revision", "enabled", "processing_state", "blocked_reason", "pending_count", "attention_count", "last_run") {
		return false
	}
	var v struct {
		Profile   string          `json:"profile"`
		Workspace string          `json:"workspace_id"`
		Revision  string          `json:"revision"`
		Enabled   *bool           `json:"enabled"`
		State     string          `json:"processing_state"`
		Blocked   *string         `json:"blocked_reason"`
		Pending   string          `json:"pending_count"`
		Attention string          `json:"attention_count"`
		Last      json.RawMessage `json:"last_run"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Profile != "private-collection-automation-v1" || v.Workspace != workspace || v.Enabled == nil || !automaticCount(v.Revision) || !automaticCount(v.Pending) || !automaticCount(v.Attention) {
		return false
	}
	if v.State != "disabled" && v.State != "idle" && v.State != "pending" && v.State != "blocked" {
		return false
	}
	if v.Blocked != nil && *v.Blocked != "saved_work" && *v.Blocked != "category_catalog_unavailable" && *v.Blocked != "permission_changed" {
		return false
	}
	return validAutomaticLastRun(v.Last, workspace)
}
