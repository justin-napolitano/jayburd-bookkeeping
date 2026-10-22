// Package sandboxverification exposes only a finite, hash-pinned Sandbox comparison capability.
package sandboxverification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

const MaxInputBytes = 2 << 20
const Login = "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"

var ErrVerification = errors.New("sandbox_verification_failed")
var ErrUnconfirmed = errors.New("sandbox_verification_unconfirmed")
var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var countPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,3})$`)

// Request is ephemeral synthetic expectation material, never a persisted financial command or log value.
type Request struct {
	MandateID, Revision  string
	Fixture, IdentityMap []byte
}

// Format prevents accidental formatted request/expectation disclosure.
func (Request) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, "[sandbox-verification request redacted]")
}

// MarshalJSON refuses request logging or command persistence.
func (Request) MarshalJSON() ([]byte, error) { return nil, ErrVerification }

// Beginner is an explicitly configured restricted verifier pool, never a caller DSN or role selector.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Result contains only bounded counts and Boolean outcomes; no source identity or financial value.
type Result struct {
	Equal                bool   `json:"equal"`
	MissingCount         string `json:"missing_count"`
	ExtraCount           string `json:"extra_count"`
	FieldMismatchCount   string `json:"field_mismatch_count"`
	ReviewMismatchCount  string `json:"review_mismatch_count"`
	LineageMismatchCount string `json:"lineage_mismatch_count"`
	JournalCount         string `json:"journal_count"`
	JournalOutboxCount   string `json:"journal_outbox_count"`
	NoAutomaticJournals  bool   `json:"no_automatic_journals"`
}

// ValidRequest checks transport bounds/identifiers only; SQL owns fixture semantics and authorization.
func ValidRequest(r Request) bool {
	revision, e := strconv.ParseInt(r.Revision, 10, 64)
	return idPattern.MatchString(r.MandateID) && r.MandateID != "00000000-0000-0000-0000-000000000000" && e == nil && revision > 0 && strconv.FormatInt(revision, 10) == r.Revision && len(r.Fixture) > 0 && len(r.IdentityMap) > 0 && len(r.Fixture)+len(r.IdentityMap) <= MaxInputBytes
}

// DecodeResult rejects malformed/inconsistent dependency output before it can become an evidence artifact.
func DecodeResult(raw []byte) (Result, error) {
	var result Result
	if len(raw) > 4096 {
		return result, ErrVerification
	}
	fields, e := StrictObject(raw)
	if e != nil || len(fields) != 9 {
		return result, ErrVerification
	}
	for _, key := range []string{"equal", "missing_count", "extra_count", "field_mismatch_count", "review_mismatch_count", "lineage_mismatch_count", "journal_count", "journal_outbox_count", "no_automatic_journals"} {
		if v, ok := fields[key]; !ok || bytes.Equal(v, []byte("null")) {
			return result, ErrVerification
		}
	}
	if json.Unmarshal(raw, &result) != nil {
		return Result{}, ErrVerification
	}
	counts := []string{result.MissingCount, result.ExtraCount, result.FieldMismatchCount, result.ReviewMismatchCount, result.LineageMismatchCount, result.JournalCount, result.JournalOutboxCount}
	zero := true
	for _, count := range counts {
		n, e := strconv.Atoi(count)
		if !countPattern.MatchString(count) || e != nil || n > 1001 {
			return Result{}, ErrVerification
		}
		zero = zero && n == 0
	}
	if result.NoAutomaticJournals != (result.JournalCount == "0" && result.JournalOutboxCount == "0") || result.Equal != zero {
		return Result{}, ErrVerification
	}
	return result, nil
}

// StrictObject rejects duplicate/unknown-shape metadata keys; callers validate their own closed field set.
// Financial fixture semantics remain exclusively in SQL and never use this helper for normalization.
func StrictObject(raw []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	token, e := d.Token()
	if e != nil || token != json.Delim('{') {
		return nil, ErrVerification
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		token, e = d.Token()
		key, ok := token.(string)
		if e != nil || !ok {
			return nil, ErrVerification
		}
		if _, seen := fields[key]; seen {
			return nil, ErrVerification
		}
		var v json.RawMessage
		if d.Decode(&v) != nil {
			return nil, ErrVerification
		}
		fields[key] = v
	}
	token, e = d.Token()
	if e != nil || token != json.Delim('}') {
		return nil, ErrVerification
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, ErrVerification
	}
	return fields, nil
}

const identityQuery = `SELECT session_user=$1 AND current_user=$1 AND current_setting('role')='none' AND rolcanlogin AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) FROM pg_roles WHERE rolname=session_user`

// Compare begins its four-second deadline before BeginTx and releases all locks before returning evidence.
// There is no automatic retry; an ambiguous COMMIT discards the prepared result even though comparison is read-only.
func Compare(parent context.Context, db Beginner, r Request) (Result, error) {
	if parent == nil || db == nil || !ValidRequest(r) {
		return Result{}, ErrVerification
	}
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return Result{}, ErrVerification
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		_ = tx.Rollback(cleanup)
	}()
	if _, e = tx.Exec(ctx, "SET LOCAL statement_timeout='3s';SET LOCAL lock_timeout='1s';SET LOCAL idle_in_transaction_session_timeout='4s'"); e != nil {
		return Result{}, ErrVerification
	}
	var identity bool
	if e = tx.QueryRow(ctx, identityQuery, Login).Scan(&identity); e != nil || !identity {
		return Result{}, ErrVerification
	}
	var raw []byte
	if e = tx.QueryRow(ctx, "SELECT integration.verify_sandbox_fixture($1,$2,$3,$4)", r.MandateID, r.Revision, string(r.Fixture), string(r.IdentityMap)).Scan(&raw); e != nil {
		return Result{}, ErrVerification
	}
	result, e := DecodeResult(raw)
	if e != nil || ctx.Err() != nil {
		return Result{}, ErrVerification
	}
	if e = tx.Commit(ctx); e != nil {
		return Result{}, ErrUnconfirmed
	}
	if ctx.Err() != nil {
		return Result{}, ErrUnconfirmed
	}
	return result, nil
}
