package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Beginner supplies one separately provisioned restricted login pool.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// SQLReceiver invokes only the verified-notification entrypoint with webhook authority.
type SQLReceiver struct{ db Beginner }

// NewSQLReceiver performs no connection or role mutation.
func NewSQLReceiver(db Beginner) (*SQLReceiver, error) {
	if db == nil {
		return nil, errUnavailable
	}
	return &SQLReceiver{db}, nil
}

// Receive returns only a confirmed receipt; unknown commits remain retryable delivery failures.
func (r *SQLReceiver) Receive(ctx context.Context, v Verified) (Receipt, error) {
	if v.Environment != "sandbox" || !opaque.MatchString(v.ItemID) || !digest.MatchString(v.ReceiptKey) || !digest.MatchString(v.BodySHA256) || v.IssuedAt <= 0 {
		return Receipt{}, errInvalid
	}
	raw, _ := json.Marshal(map[string]any{"item_id": v.ItemID, "body_sha256": v.BodySHA256, "issued_at": v.IssuedAt, "webhook_type": "TRANSACTIONS", "webhook_code": "SYNC_UPDATES_AVAILABLE"})
	out, e := call(ctx, r.db, "bookkeeping_webhook", "SELECT integration.accept_verified_webhook($1,$2,$3)", v.Environment, v.ReceiptKey, string(raw))
	if e != nil {
		return Receipt{}, e
	}
	var receipt Receipt
	if object(out, []string{"state", "receipt_id", "stream_count"}, nil, &receipt) != nil || !validReceipt(receipt) {
		return Receipt{}, errUnavailable
	}
	return receipt, nil
}

// Scope is trusted scheduler composition metadata, never a webhook-provided ownership claim.
type Scope struct{ EntityID, BookID string }

// Wakeup identifies durable intent, not a completed acquisition or posted journal.
type Wakeup struct {
	StreamID            string `json:"stream_id"`
	RequestedGeneration string `json:"requested_generation"`
}

// CatchupResult reports bounded committed progress; callers retain tick identity across pages/replies.
type CatchupResult struct {
	Items      []Wakeup `json:"items"`
	NextCursor *string  `json:"next_cursor"`
}

// Catchup uses only the restricted sync login and performs no provider calls or scheduling.
type Catchup struct{ db Beginner }

// NewCatchup accepts an explicitly configured sync pool without ambient discovery.
func NewCatchup(db Beginner) (*Catchup, error) {
	if db == nil {
		return nil, errUnavailable
	}
	return &Catchup{db}, nil
}

// Tick commits at most maximum eligible streams for one stable scope/tick/cursor page.
func (c *Catchup) Tick(ctx context.Context, s Scope, tick string, maximum int, after *string) (CatchupResult, error) {
	if !validID(s.EntityID) || !validID(s.BookID) || !validID(tick) || maximum < 1 || maximum > 100 || after != nil && !validID(*after) {
		return CatchupResult{}, errInvalid
	}
	raw, e := call(ctx, c.db, "bookkeeping_sync", "SELECT integration.catchup_sync($1,$2,$3,$4,$5)", s.EntityID, s.BookID, tick, maximum, after)
	if e != nil {
		return CatchupResult{}, e
	}
	var wire struct {
		Items      []json.RawMessage `json:"items"`
		NextCursor *string           `json:"next_cursor"`
	}
	if object(raw, []string{"items", "next_cursor"}, nil, &wire) != nil {
		return CatchupResult{}, errUnavailable
	}
	out := CatchupResult{Items: make([]Wakeup, 0, len(wire.Items)), NextCursor: wire.NextCursor}
	for _, item := range wire.Items {
		var value Wakeup
		if object(item, []string{"stream_id", "requested_generation"}, nil, &value) != nil {
			return CatchupResult{}, errUnavailable
		}
		out.Items = append(out.Items, value)
	}
	if wire.Items == nil || out.Items == nil || len(out.Items) > maximum || out.NextCursor != nil && (!validID(*out.NextCursor) || after != nil && *out.NextCursor <= *after) {
		return CatchupResult{}, errUnavailable
	}
	previous := ""
	if after != nil {
		previous = *after
	}
	for _, item := range out.Items {
		n, e := strconv.ParseInt(item.RequestedGeneration, 10, 64)
		if !validID(item.StreamID) || item.StreamID <= previous || e != nil || n <= 0 || strconv.FormatInt(n, 10) != item.RequestedGeneration {
			return CatchupResult{}, errUnavailable
		}
		previous = item.StreamID
	}
	return out, nil
}

// roleQuery rechecks effective authority per request, including powerful predefined roles without superuser flags.
const roleQuery = `SELECT session_user=current_user AND r.rolcanlogin AND NOT(r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication)
AND pg_has_role(r.oid,$1,'USAGE')
AND NOT EXISTS(SELECT FROM pg_roles p WHERE (p.rolsuper OR p.rolbypassrls OR p.rolcreaterole OR p.rolcreatedb OR p.rolreplication OR p.rolname ~ '^pg_' OR p.rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_identity_operator','bookkeeping_publisher','bookkeeping_observer','bookkeeping_migrator','cloudsqlsuperuser','bookkeeping_sync','bookkeeping_webhook') AND p.rolname<>$1) AND pg_has_role(r.oid,p.oid,'MEMBER'))
AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','deployment') AND c.relkind IN ('r','p','v','m','S') AND has_table_privilege(r.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
FROM pg_roles r WHERE r.rolname=session_user`

// retryable includes only server-confirmed whole-transaction rollback failures.
func retryable(e error) bool {
	var p *pgconn.PgError
	return errors.As(e, &p) && (p.Code == "40001" || p.Code == "40P01")
}

// rollbackKnown excludes unknown-resolution SQLSTATEs even when represented as PgError.
func rollbackKnown(e error) bool {
	var p *pgconn.PgError
	return errors.Is(e, pgx.ErrTxCommitRollback) || errors.As(e, &p) && (retryable(e) || p.Code == "25P02" || strings.HasPrefix(p.Code, "23") || strings.HasPrefix(p.Code, "P28"))
}

// safeSQL discards all raw SQL diagnostics and keeps only uniform denial classification.
func safeSQL(e error) error {
	var p *pgconn.PgError
	if errors.As(e, &p) && (p.Code == "P2801" || p.Code == "P2601" || p.Code == "42501") {
		return errDenied
	}
	return errUnavailable
}

// call retries SQL only, retaining independent cleanup and never replaying ambiguous effects.
func call(parent context.Context, db Beginner, role, q string, args ...any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		if ctx.Err() != nil {
			return nil, errUnavailable
		}
		raw, e, unknown := transaction(ctx, db, role, q, args...)
		if e == nil {
			return raw, nil
		}
		if unknown {
			return nil, errUnknown
		}
		if !retryable(e) || attempt == 2 {
			return nil, safeSQL(e)
		}
		timer := time.NewTimer(time.Duration(10*(attempt+1)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, errUnavailable
		case <-timer.C:
		}
	}
	return nil, errUnavailable
}

// transaction checks the actual session role and exposes metadata only after confirmed COMMIT.
func transaction(ctx context.Context, db Beginner, role, q string, args ...any) (json.RawMessage, error, bool) {
	tx, e := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return nil, e, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='3s'; SET LOCAL lock_timeout='1s'; SET LOCAL idle_in_transaction_session_timeout='4s'"); e != nil {
		return nil, e, false
	}
	var allowed bool
	if e = tx.QueryRow(ctx, roleQuery, role).Scan(&allowed); e != nil {
		return nil, e, false
	}
	if !allowed {
		return nil, &pgconn.PgError{Code: "42501"}, false
	}
	var raw []byte
	if e = tx.QueryRow(ctx, q, args...).Scan(&raw); e != nil {
		return nil, e, false
	}
	if len(raw) == 0 || len(raw) > 32768 || !json.Valid(raw) || bytes.TrimSpace(raw)[0] != '{' {
		return nil, errUnavailable, false
	}
	if ctx.Err() != nil {
		return nil, ctx.Err(), false
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e, !rollbackKnown(e)
	}
	return raw, nil, false
}
