// Package collectionlabels transports SQL-owned private query and reviewed label commands.
// It never derives authority, taxonomy decisions or monetary values from caller JSON.
package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// Request keeps independently verified identity and path selectors separate from unchanged JSON.
type Request struct {
	Actor, Workspace, Command, Body string
	Version                         int
}

// Operations exposes only the frozen restricted SQL query and command surface.
type Operations interface {
	Catalog(context.Context, Request) (json.RawMessage, error)
	Query(context.Context, Request) (json.RawMessage, error)
	Preview(context.Context, Request) (json.RawMessage, error)
	Prepare(context.Context, Request) (json.RawMessage, error)
	Pending(context.Context, Request) (json.RawMessage, error)
	Transition(context.Context, Request, string) (json.RawMessage, error)
}

// DetailOperations reads one exact source revision and independently authorized journal anchors.
type DetailOperations interface {
	Detail(context.Context, Request) (json.RawMessage, error)
}

// Detail uses the v3 bounded reader without changing historical command versions.
func (s *Service) Detail(ctx context.Context, r Request) (json.RawMessage, error) {
	if r.Version != 3 {
		return nil, &ledger.Failure{Status: 400, Code: "invalid_collection"}
	}
	return s.run(ctx, r, "detail")
}

// RecoveryOperations exposes payload-free owner recovery independently of workspace grants.
type RecoveryOperations interface {
	Recovery(context.Context, Request) (json.RawMessage, error)
	ClearRecovery(context.Context, Request) (json.RawMessage, error)
}

// Recovery discovers only an opaque owned private reservation; SQL checks the active principal.
func (s *Service) Recovery(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "recovery")
}

// ClearRecovery obtains an exact durable acknowledgement receipt without returning saved payloads.
func (s *Service) ClearRecovery(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "clear_recovery")
}

// Service uses a caller-composed restricted pool; SQL owns every business invariant.
type Service struct {
	DB    ledger.Beginner
	Rules RuleHook
}

// RuleHook evaluates only authoritative SQL inputs inside this operation's transaction.
// Its output still passes the fixed result fence and acknowledged-commit boundary.
type RuleHook interface {
	Process(context.Context, pgx.Tx, Request, string) (json.RawMessage, bool, error)
}

// Catalog reads the complete bounded authorized catalog and source aliases.
func (s *Service) Catalog(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "catalog")
}

// Query reads one current revision-pinned page without interpreting money or filters.
func (s *Service) Query(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "query")
}

// Preview persists a bounded inert server preview in a serializable transaction.
func (s *Service) Preview(ctx context.Context, r Request) (json.RawMessage, error) {
	// v3 can persist rule catalogs: never bypass the trusted Go validation hook.
	if r.Version == 3 && (s == nil || s.Rules == nil) {
		return nil, unavailable()
	}
	return s.run(ctx, r, "preview")
}

// Prepare persists the exact reviewed intent using SQL-owned idempotency semantics.
func (s *Service) Prepare(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "prepare")
}

// Pending discovers existing work without executing or acknowledging it.
func (s *Service) Pending(ctx context.Context, r Request) (json.RawMessage, error) {
	return s.run(ctx, r, "pending")
}

// Transition accepts only explicit execute, cancel or acknowledge operations.
func (s *Service) Transition(ctx context.Context, r Request, operation string) (json.RawMessage, error) {
	if operation != "execute" && operation != "cancel" && operation != "acknowledge" {
		return nil, &ledger.Failure{Status: 400, Code: "invalid_collection"}
	}
	return s.run(ctx, r, operation)
}

// run retries whole known-aborted transactions only; uncertain commit never causes another call.
func (s *Service) run(parent context.Context, r Request, operation string) (json.RawMessage, error) {
	if s == nil || s.DB == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		body, err, unknown := s.attempt(ctx, r, operation)
		if err == nil {
			return body, nil
		}
		if unknown {
			return nil, unavailable()
		}
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || (pg.Code != "40001" && pg.Code != "40P01") {
			return nil, safeFailure(err)
		}
		if attempt < 2 {
			timer := time.NewTimer(time.Duration(attempt+1) * 10 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, unavailable()
			case <-timer.C:
			}
		}
	}
	return nil, unavailable()
}

// attempt acknowledges COMMIT before returning a private result and cleans up on every exit.
func (s *Service) attempt(ctx context.Context, r Request, operation string) (json.RawMessage, error, bool) {
	isolation := pgx.Serializable
	if operation == "catalog" || operation == "query" || operation == "pending" || operation == "detail" {
		isolation = pgx.RepeatableRead
	}
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: isolation})
	if err != nil {
		return nil, err, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) < time.Millisecond {
		return nil, context.DeadlineExceeded, false
	}
	remaining := (time.Until(deadline) / time.Millisecond * time.Millisecond).String()
	if _, err = tx.Exec(ctx, "select set_config('statement_timeout',$1,true),set_config('lock_timeout',$1,true)", remaining); err != nil {
		return nil, err, false
	}
	var body json.RawMessage
	handled := false
	if r.Version == 3 && s.Rules != nil && (operation == "query" || operation == "preview" || operation == "detail") {
		body, handled, err = s.Rules.Process(ctx, tx, r, operation)
		if err != nil {
			return nil, err, false
		}
	}
	if !handled {
		switch operation {
		case "catalog":
			err = tx.QueryRow(ctx, labelStatement("get_collection_taxonomy", r.Version), r.Actor, r.Workspace).Scan(&body)
		case "query":
			err = tx.QueryRow(ctx, labelStatement("query_collection_review", r.Version), r.Actor, r.Workspace, r.Body).Scan(&body)
		case "detail":
			err = tx.QueryRow(ctx, "select banking.get_collection_detail($1::uuid,$2::uuid,$3::text)", r.Actor, r.Workspace, r.Body).Scan(&body)
		case "preview":
			err = tx.QueryRow(ctx, labelStatement("preview_collection_labels", r.Version), r.Actor, r.Workspace, r.Body).Scan(&body)
		case "prepare":
			err = tx.QueryRow(ctx, labelStatement("prepare_collection_label_command", r.Version), r.Actor, r.Workspace, r.Body).Scan(&body)
		case "pending":
			err = tx.QueryRow(ctx, labelStatement("pending_collection_label_command", r.Version), r.Actor, r.Workspace).Scan(&body)
		case "execute", "cancel", "acknowledge":
			err = tx.QueryRow(ctx, labelStatement("transition_collection_label_command", r.Version), r.Actor, r.Workspace, r.Command, operation, r.Body).Scan(&body)
		case "recovery":
			err = tx.QueryRow(ctx, "select banking.collection_recovery_status($1::uuid)", r.Actor).Scan(&body)
		case "clear_recovery":
			err = tx.QueryRow(ctx, "select banking.clear_collection_recovery($1::uuid,$2::text)", r.Actor, r.Body).Scan(&body)
		default:
			return nil, unavailable(), false
		}
	}
	if err != nil {
		return nil, err, false
	}
	if !validRequestResult(body, operation, r) {
		return nil, unavailable(), false
	}
	if err = tx.Commit(ctx); err != nil {
		var pg *pgconn.PgError
		unknown := !errors.As(err, &pg) && !errors.Is(err, pgx.ErrTxCommitRollback)
		if pg != nil && (pg.Code == "40003" || strings.HasPrefix(pg.Code, "08")) {
			unknown = true
		}
		return nil, err, unknown
	}
	return body, nil, false
}

// labelStatement chooses only fixed reviewed SQL names, never caller-controlled identifiers.
func labelStatement(name string, version int) string {
	switch name {
	case "get_collection_taxonomy":
		if version == 3 {
			return "select banking.get_collection_taxonomy_v3($1::uuid,$2::uuid)"
		}
		if version == 2 {
			return "select banking.get_collection_taxonomy_v2($1::uuid,$2::uuid)"
		}
		return "select banking.get_collection_taxonomy($1::uuid,$2::uuid)"
	case "query_collection_review":
		if version == 3 {
			return "select banking.query_collection_review_v3($1::uuid,$2::uuid,$3::text)"
		}
		if version == 2 {
			return "select banking.query_collection_review_v2($1::uuid,$2::uuid,$3::text)"
		}
		return "select banking.query_collection_review($1::uuid,$2::uuid,$3::text)"
	case "preview_collection_labels":
		if version == 3 {
			return "select banking.preview_collection_labels_v3($1::uuid,$2::uuid,$3::text)"
		}
		if version == 2 {
			return "select banking.preview_collection_labels_v2($1::uuid,$2::uuid,$3::text)"
		}
		return "select banking.preview_collection_labels($1::uuid,$2::uuid,$3::text)"
	case "prepare_collection_label_command":
		if version == 3 {
			return "select banking.prepare_collection_label_command_v3($1::uuid,$2::uuid,$3::text)"
		}
		if version == 2 {
			return "select banking.prepare_collection_label_command_v2($1::uuid,$2::uuid,$3::text)"
		}
		return "select banking.prepare_collection_label_command($1::uuid,$2::uuid,$3::text)"
	case "pending_collection_label_command":
		if version == 3 {
			return "select banking.pending_collection_label_command_v3($1::uuid,$2::uuid)"
		}
		if version == 2 {
			return "select banking.pending_collection_label_command_v2($1::uuid,$2::uuid)"
		}
		return "select banking.pending_collection_label_command($1::uuid,$2::uuid)"
	case "transition_collection_label_command":
		if version == 3 {
			return "select banking.transition_collection_label_command_v3($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)"
		}
		if version == 2 {
			return "select banking.transition_collection_label_command_v2($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)"
		}
		return "select banking.transition_collection_label_command($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)"
	}
	return ""
}

// validRequestResult keeps versioned payloads and opaque recovery receipts within their exact boundary.
func validRequestResult(body json.RawMessage, operation string, r Request) bool {
	if r.Version == 3 && (operation == "catalog" || operation == "query" || operation == "detail") {
		return validV3ReadResult(body, operation, r)
	}
	if operation == "recovery" || operation == "clear_recovery" {
		if len(body) > 4096 || !json.Valid(body) {
			return false
		}
		var v map[string]json.RawMessage
		if json.Unmarshal(body, &v) != nil || len(v) != 3 {
			return false
		}
		var profile, id string
		if json.Unmarshal(v["profile"], &profile) != nil {
			return false
		}
		if operation == "recovery" {
			var state string
			if profile != "private-collection-recovery-v1" || json.Unmarshal(v["state"], &state) != nil {
				return false
			}
			if state == "none" {
				return string(v["recovery_id"]) == "null"
			}
			return (state == "available" || state == "inaccessible") && json.Unmarshal(v["recovery_id"], &id) == nil && validID(id)
		}
		var cleared bool
		var request struct {
			RecoveryID string `json:"recovery_id"`
		}
		return profile == "private-collection-recovery-clear-v1" && json.Unmarshal(v["recovery_id"], &id) == nil && validID(id) && json.Unmarshal(v["cleared"], &cleared) == nil && cleared && json.Unmarshal([]byte(r.Body), &request) == nil && request.RecoveryID == id
	}
	if r.Version != 2 && r.Version != 3 {
		return validResult(body, operation, r.Workspace)
	}
	if len(body) > 1<<20 || !json.Valid(body) {
		return false
	}
	var v map[string]json.RawMessage
	if json.Unmarshal(body, &v) != nil {
		return false
	}
	var profile string
	suffix := "-v2"
	if r.Version == 3 {
		suffix = "-v3"
	}
	if json.Unmarshal(v["profile"], &profile) != nil || !strings.HasSuffix(profile, suffix) {
		return false
	}
	// Reuse the same envelope/nullability fence while keeping immutable saved previews untouched.
	v["profile"], _ = json.Marshal(strings.TrimSuffix(profile, suffix) + "-v1")
	normalized, err := json.Marshal(v)
	return err == nil && validResult(normalized, operation, r.Workspace)
}

// validV3ReadResult fences the new reader without relabeling immutable v1/v2 commands.
func validV3ReadResult(body json.RawMessage, operation string, r Request) bool {
	if len(body) > 1<<20 || !json.Valid(body) {
		return false
	}
	var envelope struct {
		Profile   string `json:"profile"`
		Workspace string `json:"workspace_id"`
		Row       struct {
			Source      string `json:"source_id"`
			Observation string `json:"observation_id"`
			Revision    string `json:"observation_revision"`
		} `json:"row"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Workspace != r.Workspace {
		return false
	}
	switch operation {
	case "catalog":
		return envelope.Profile == "private-collection-catalog-v3"
	case "query":
		return envelope.Profile == "private-collection-review-v3"
	case "detail":
		var request struct {
			Source      string `json:"source_id"`
			Observation string `json:"observation_id"`
			Revision    string `json:"observation_revision"`
		}
		return envelope.Profile == "private-collection-detail-v1" && json.Unmarshal([]byte(r.Body), &request) == nil &&
			validID(request.Source) && validID(request.Observation) && request.Revision != "" &&
			envelope.Row.Source == request.Source && envelope.Row.Observation == request.Observation && envelope.Row.Revision == request.Revision
	}
	return false
}

// validResult fences profile, workspace, command-nullability and budget, leaving business fields to SQL.
func validResult(body json.RawMessage, operation, workspace string) bool {
	if len(body) > 1<<20 || !json.Valid(body) {
		return false
	}
	var envelope struct {
		Profile   string          `json:"profile"`
		Workspace string          `json:"workspace_id"`
		Command   json.RawMessage `json:"command"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Workspace != workspace {
		return false
	}
	profile := "private-collection-command-v1"
	switch operation {
	case "catalog":
		profile = "private-collection-catalog-v1"
	case "query":
		profile = "private-collection-review-v1"
	case "preview":
		profile = "private-collection-preview-v1"
	case "prepare", "pending", "execute", "cancel", "acknowledge":
		command := strings.TrimSpace(string(envelope.Command))
		if command == "null" {
			if operation != "pending" {
				return false
			}
		} else if len(command) < 2 || command[0] != '{' {
			return false
		}
	default:
		return false
	}
	return envelope.Profile == profile
}

// unavailable discards all database, transport and private identifier diagnostics.
func unavailable() *ledger.Failure {
	return &ledger.Failure{Status: 503, Code: "temporarily_unavailable"}
}

// safeFailure translates only the fixed SQLSTATE vocabulary; unknown errors fail closed.
func safeFailure(err error) *ledger.Failure {
	// Only these definite Go validation failures may cross the hook boundary;
	// copy fixed fields so an internal error cannot choose arbitrary HTTP output.
	var f *ledger.Failure
	if errors.As(err, &f) && f != nil &&
		((f.Status == 400 && f.Code == "invalid_collection") || (f.Status == 422 && f.Code == "collection_limit")) {
		return &ledger.Failure{Status: f.Status, Code: f.Code}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P2800", "22P02", "22021":
			return &ledger.Failure{Status: 400, Code: "invalid_collection"}
		case "P2801":
			return &ledger.Failure{Status: 404, Code: "collection_denied"}
		case "P2802":
			return &ledger.Failure{Status: 409, Code: "collection_conflict"}
		case "P2803":
			return &ledger.Failure{Status: 409, Code: "collection_snapshot_changed"}
		case "P2804":
			return &ledger.Failure{Status: 409, Code: "collection_not_ready"}
		case "P2805":
			return &ledger.Failure{Status: 422, Code: "collection_limit"}
		}
	}
	return unavailable()
}
