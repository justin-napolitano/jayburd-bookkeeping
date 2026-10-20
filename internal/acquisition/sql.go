package acquisition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SQLRepository owns short role-checked transactions and exposes results only after COMMIT.
type SQLRepository struct{ DB Beginner }

const syncRoleQuery = `SELECT session_user=current_user AND r.rolcanlogin AND NOT(r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication)
AND pg_has_role(r.oid,'bookkeeping_sync','USAGE')
AND NOT EXISTS(SELECT FROM pg_roles p WHERE (p.rolsuper OR p.rolbypassrls OR p.rolcreaterole OR p.rolcreatedb OR p.rolreplication OR p.rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_identity_operator','bookkeeping_webhook','bookkeeping_publisher','bookkeeping_observer','bookkeeping_migrator','cloudsqlsuperuser')) AND pg_has_role(r.oid,p.oid,'MEMBER'))
AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','deployment') AND c.relkind IN ('r','p','v','m','S') AND has_table_privilege(r.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
FROM pg_roles r WHERE r.rolname=session_user`

// sqlError translates only fixed SQLSTATEs and discards raw diagnostic text.
func sqlError(e error) error {
	var p *pgconn.PgError
	if errors.As(e, &p) {
		switch p.Code {
		case "P2600":
			return failure("invalid_request", false)
		case "P2601", "42501":
			return failure("not_authorized", false)
		case "P2602":
			return failure("evidence_conflict", false)
		case "P2603":
			return failure("stale_attempt", false)
		case "P2604":
			return failure("invalid_update", false)
		case "P2605":
			return failure("limit_exceeded", false)
		}
	}
	return failure("temporarily_unavailable", true)
}

// retrySQL includes only server-confirmed whole-transaction rollback failures.
func retrySQL(e error) bool {
	var p *pgconn.PgError
	return errors.As(e, &p) && (p.Code == "40001" || p.Code == "40P01")
}

// knownRollback conservatively distinguishes confirmed rollback from ambiguous COMMIT.
func knownRollback(e error) bool {
	if errors.Is(e, pgx.ErrTxCommitRollback) {
		return true
	}
	var p *pgconn.PgError
	if !errors.As(e, &p) {
		return false
	}
	return retrySQL(e) || p.Code == "25P02" || strings.HasPrefix(p.Code, "23") || strings.HasPrefix(p.Code, "P26")
}

// call retries SQL only; network/storage never enters the transaction callback.
func (r *SQLRepository) call(parent context.Context, q string, args ...any) (json.RawMessage, error) {
	if r == nil || r.DB == nil {
		return nil, failure("configuration_error", false)
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for n := 0; n < 3; n++ {
		if ctx.Err() != nil {
			return nil, failure("temporarily_unavailable", true)
		}
		raw, e, unknown := r.transaction(ctx, q, args...)
		if e == nil {
			return raw, nil
		}
		if unknown {
			return nil, failure("commit_unknown", true)
		}
		if !retrySQL(e) || n == 2 {
			return nil, sqlError(e)
		}
		timer := time.NewTimer(time.Duration(5+rand.IntN(20)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, failure("temporarily_unavailable", true)
		case <-timer.C:
		}
	}
	return nil, failure("temporarily_unavailable", true)
}

// transaction owns one serializable transaction and independently bounded cleanup.
func (r *SQLRepository) transaction(ctx context.Context, q string, args ...any) (json.RawMessage, error, bool) {
	tx, e := r.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
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
	var safe bool
	if e = tx.QueryRow(ctx, syncRoleQuery).Scan(&safe); e != nil {
		return nil, e, false
	}
	if !safe {
		return nil, &pgconn.PgError{Code: "42501"}, false
	}
	var raw []byte
	if e = tx.QueryRow(ctx, q, args...).Scan(&raw); e != nil {
		return nil, e, false
	}
	if len(raw) == 0 || len(raw) > 16384 || !json.Valid(raw) || bytes.TrimSpace(raw)[0] != '{' {
		return nil, errors.New("invalid metadata"), false
	}
	if e = ctx.Err(); e != nil {
		return nil, e, false
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e, !knownRollback(e)
	}
	return raw, nil, false
}

// decodeMetadata rejects malformed SQL results without returning their content.
func decodeMetadata(raw []byte, out any) error {
	if json.Unmarshal(raw, out) != nil {
		return failure("invalid_metadata", false)
	}
	return nil
}

// validCompletion ensures only a scoped, bounded receipt becomes acquisition success.
func validCompletion(c completion, s Scope, a attempt) bool {
	n, e := strconv.Atoi(c.ChangedCount)
	return c.StreamID == s.StreamID && c.SetID == a.SetID && positive(c.CompletedGeneration) && positive(c.SourceRevision) && e == nil && n >= 0 && n <= 1000 && strconv.Itoa(n) == c.ChangedCount && validCursor(c.Cursor) && oneOf(c.Readiness, "TRANSACTIONS_UPDATE_STATUS_UNKNOWN", "NOT_READY", "INITIAL_UPDATE_COMPLETE", "HISTORICAL_UPDATE_COMPLETE")
}

// oneOf compares only closed protocol enums.
func oneOf(v string, allowed ...string) bool {
	for _, s := range allowed {
		if v == s {
			return true
		}
	}
	return false
}

// Resume rechecks current authority before exposing a prior attempt or its committed receipt.
func (r *SQLRepository) Resume(ctx context.Context, s Scope, key string) (resume, error) {
	var out resume
	raw, e := r.call(ctx, "SELECT integration.sync_resume($1,$2,$3,$4)", s.EntityID, s.BookID, s.StreamID, key)
	if e != nil {
		return out, e
	}
	if e = decodeMetadata(raw, &out); e != nil {
		return resume{}, e
	}
	if out.State == "absent" && out.SetID == nil && out.Fence == nil && out.Reason == nil && out.Completion == nil {
		return out, nil
	}
	if !oneOf(out.State, "staging", "completed", "abandoned", "quarantined") || out.SetID == nil || !validID(*out.SetID) || out.Fence == nil || !positive(*out.Fence) {
		return resume{}, failure("invalid_metadata", false)
	}
	if out.State == "completed" {
		if out.Completion == nil || !validCompletion(*out.Completion, s, attempt{*out.SetID, *out.Fence}) {
			return resume{}, failure("invalid_metadata", false)
		}
	} else if out.Completion != nil {
		return resume{}, failure("invalid_metadata", false)
	}
	return out, nil
}

// Claim returns only committed ownership under a stable caller-persisted claim UUID.
func (r *SQLRepository) Claim(ctx context.Context, s Scope, key string) (attempt, error) {
	var out struct {
		StreamID string `json:"stream_id"`
		SetID    string `json:"set_id"`
		Fence    string `json:"fence"`
	}
	raw, e := r.call(ctx, "SELECT integration.claim_sync($1,$2,$3,$4)", s.EntityID, s.BookID, s.StreamID, key)
	if e != nil {
		return attempt{}, e
	}
	if decodeMetadata(raw, &out) != nil || out.StreamID != s.StreamID || !validID(out.SetID) || !positive(out.Fence) {
		return attempt{}, failure("invalid_metadata", false)
	}
	return attempt{out.SetID, out.Fence}, nil
}

// Context authorizes each credential/provider/evidence access using current SQL pins.
func (r *SQLRepository) Context(ctx context.Context, s Scope, a attempt) (workContext, error) {
	var out workContext
	raw, e := r.call(ctx, "SELECT integration.sync_context($1,$2,$3,$4,$5)", s.EntityID, s.BookID, s.StreamID, a.SetID, a.Fence)
	if e != nil {
		return out, e
	}
	if decodeMetadata(raw, &out) != nil || !validContext(out, s, a) {
		return workContext{}, failure("invalid_metadata", false)
	}
	return out, nil
}

// validCurrency checks reference shape without treating publication as currency inception.
func validCurrency(c currencyMetadata) bool {
	from, e := time.Parse("2006-01-02", c.ValidFrom)
	toOK := true
	if c.ValidTo != nil {
		to, x := time.Parse("2006-01-02", *c.ValidTo)
		toOK = x == nil && to.After(from)
	}
	return validID(c.VersionID) && regexpCurrency(c.Code) && c.MinorUnit >= 0 && c.MinorUnit <= 9 && e == nil && toOK
}

// validCurrencies requires bounded ordered history and an unchanged binding reference.
func validCurrencies(c workContext) bool {
	if !validCurrency(c.Currency) {
		return false
	}
	if c.CurrencyVersions == nil {
		return true // Older SQL exposes the exact singleton reference only.
	}
	if len(c.CurrencyVersions) == 0 || len(c.CurrencyVersions) > 32 {
		return false
	}
	seen := make(map[string]bool)
	found := false
	for i, v := range c.CurrencyVersions {
		if !validCurrency(v) || v.Code != c.Currency.Code || seen[v.VersionID] {
			return false
		}
		seen[v.VersionID] = true
		if i > 0 {
			previous := c.CurrencyVersions[i-1]
			if previous.ValidTo == nil || *previous.ValidTo > v.ValidFrom {
				return false
			}
		}
		if v.VersionID == c.Currency.VersionID {
			toEqual := v.ValidTo == nil && c.Currency.ValidTo == nil || v.ValidTo != nil && c.Currency.ValidTo != nil && *v.ValidTo == *c.Currency.ValidTo
			found = v.ValidFrom == c.Currency.ValidFrom && v.MinorUnit == c.Currency.MinorUnit && toEqual
		}
	}
	return found
}

// validContext fails closed on malformed scope, references, lease and engineering limits.
func validContext(c workContext, s Scope, a attempt) bool {
	return c.StreamID == s.StreamID && c.SetID == a.SetID && c.Fence == a.Fence && c.Provider == "plaid" && c.Environment == "sandbox" && opaque.MatchString(c.AccountID) && opaque.MatchString(c.ItemID) && validID(c.CredentialRef) && positive(c.CredentialVersion) && positive(c.ItemGeneration) && positive(c.BindingRevision) && validCursor(c.StartingCursor) && validCursor(c.RequestCursor) && c.PageCount == c.NextPageIndex && c.PageCount >= 0 && c.PageCount <= 10 && c.ChangeCount >= 0 && c.ChangeCount <= 1000 && c.ByteCount >= 0 && c.ByteCount <= 2<<20 && c.LeaseUntil.After(time.Now()) && c.LeaseUntil.Before(time.Now().Add(91*time.Second)) && validCurrencies(c) && c.Limits.MaxPages == 10 && c.Limits.MaxChanges == 1000 && c.Limits.MaxBytes == 2<<20 && c.Limits.MaxPageChanges == 500
}

// regexpCurrency checks the fixed ISO alphabetic shape without inventing a catalog.
func regexpCurrency(s string) bool {
	return len(s) == 3 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z' && s[2] >= 'A' && s[2] <= 'Z'
}

// Stage sends only the runner's exact-decoder result after private evidence publication.
func (r *SQLRepository) Stage(ctx context.Context, s Scope, a attempt, index int, body json.RawMessage) error {
	raw, e := r.call(ctx, "SELECT integration.stage_sync_page($1,$2,$3,$4,$5,$6)", s.EntityID, s.BookID, s.StreamID, a.SetID, a.Fence, string(body))
	if e != nil {
		return e
	}
	var v struct {
		SetID     string `json:"set_id"`
		PageCount int    `json:"page_count"`
	}
	if decodeMetadata(raw, &v) != nil || v.SetID != a.SetID || v.PageCount != index+1 {
		return failure("invalid_metadata", false)
	}
	return nil
}

// Complete exposes a result only after SQL atomically publishes evidence and cursor.
func (r *SQLRepository) Complete(ctx context.Context, s Scope, a attempt) (completion, error) {
	var out completion
	raw, e := r.call(ctx, "SELECT integration.complete_sync($1,$2,$3,$4,$5)", s.EntityID, s.BookID, s.StreamID, a.SetID, a.Fence)
	if e != nil {
		return out, e
	}
	if decodeMetadata(raw, &out) != nil || !validCompletion(out, s, a) {
		return completion{}, failure("invalid_metadata", false)
	}
	return out, nil
}

// Abandon fences the entire update before cleanup or a new C0 attempt.
func (r *SQLRepository) Abandon(ctx context.Context, s Scope, a attempt, reason string) error {
	return r.abandon(ctx, s, a, "SELECT integration.abandon_sync($1,$2,$3,$4,$5,$6)", reason)
}

// RejectDecode commits only a closed decoder code together with fenced abandonment.
func (r *SQLRepository) RejectDecode(ctx context.Context, s Scope, a attempt, code string) error {
	return r.abandon(ctx, s, a, "SELECT integration.reject_sync_decode($1,$2,$3,$4,$5,$6)", code)
}

// abandon validates the committed result for either fixed SQL abandonment entrypoint.
func (r *SQLRepository) abandon(ctx context.Context, s Scope, a attempt, query, reason string) error {
	raw, e := r.call(ctx, query, s.EntityID, s.BookID, s.StreamID, a.SetID, a.Fence, reason)
	if e != nil {
		return e
	}
	var v struct {
		SetID string `json:"set_id"`
		State string `json:"state"`
	}
	if decodeMetadata(raw, &v) != nil || v.SetID != a.SetID || v.State != "abandoned" {
		return failure("invalid_metadata", false)
	}
	return nil
}
