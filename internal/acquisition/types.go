// Package acquisition connects bounded synthetic provider acquisition to SQL-owned evidence publication.
package acquisition

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

const MaxRawBytes = 2 << 20

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var opaque = regexp.MustCompile(`^[!-~]{1,256}$`)

// Scope is a server-owned stream identity, never a caller-supplied provider account filter.
type Scope struct {
	EntityID string `json:"entity_id"`
	BookID   string `json:"book_id"`
	StreamID string `json:"stream_id"`
}

// Beginner supplies the independently configured, restricted sync pool.
type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Failure contains fixed operational classifications, never infrastructure or financial data.
type Failure struct {
	Code      string
	Retryable bool
}

// Error returns only the fixed classification.
func (e *Failure) Error() string { return e.Code }

// failure creates a redacted failure without retaining an underlying exception.
func failure(code string, retry bool) error { return &Failure{code, retry} }

// validID rejects zero IDs and all pathname syntax.
func validID(s string) bool {
	return uuid.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}

// validScope checks identifiers without claiming authority.
func validScope(s Scope) bool { return validID(s.EntityID) && validID(s.BookID) && validID(s.StreamID) }

// positive checks exact signed-bigint metadata, with no coercion or floating point.
func positive(s string) bool {
	n, e := strconv.ParseInt(s, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == s
}

// validCursor permits the pinned opaque cursor grammar and initial absence.
func validCursor(s *string) bool { return s == nil || opaque.MatchString(*s) }

// CredentialReference is SQL-authorized metadata, never the token itself.
type CredentialReference struct{ ReferenceID, Version, ItemID, Environment string }

// ProviderCredentials exists in memory only; composition must never log or persist it.
type ProviderCredentials struct{ AccessToken, ClientID, Secret string }

// String suppresses whole-object credential formatting.
func (c ProviderCredentials) String() string { return "[redacted provider credentials]" }

// GoString suppresses Go-syntax whole-object credential formatting.
func (c ProviderCredentials) GoString() string { return c.String() }

// MarshalJSON forbids accidental serialization outside the deliberate provider request builder.
func (c ProviderCredentials) MarshalJSON() ([]byte, error) {
	return nil, failure("credential_serialization_forbidden", false)
}

// CredentialResolver verifies immutable version and Item/environment binding before returning credentials.
type CredentialResolver interface {
	Resolve(context.Context, CredentialReference) (ProviderCredentials, error)
}

// FetchRequest fixes one authorized account and cursor; it accepts no destination URL.
type FetchRequest struct {
	AccountID   string
	Cursor      *string
	Credentials ProviderCredentials
}

// Provider returns original response bytes or a bounded classified provider failure.
type Provider interface {
	Fetch(context.Context, FetchRequest) ([]byte, error)
}

// ProviderFailure carries only safe retry/mutation information, never the provider response body.
type ProviderFailure struct {
	Code       string
	RetryAfter time.Duration
}

// Error exposes only the closed provider classification.
func (e *ProviderFailure) Error() string { return e.Code }

// EvidenceKey binds immutable raw bytes to a complete authorized attempt and page.
type EvidenceKey struct {
	Scope  Scope   `json:"scope"`
	SetID  string  `json:"set_id"`
	Page   int     `json:"page"`
	Cursor *string `json:"request_cursor"`
}

// EvidenceStore retains exact bytes privately; implementations must honor context and reject byte conflicts.
type EvidenceStore interface {
	Put(context.Context, EvidenceKey, []byte) error
	Get(context.Context, EvidenceKey) ([]byte, bool, error)
	Delete(context.Context, EvidenceKey) error
}

// Outcome exposes a committed receipt or a safe acquisition state; never credentials or raw pages.
type Outcome struct {
	Status       string
	ChangedCount string
	RetryAfter   time.Duration
}

// attempt identifies one SQL-owned fenced acquisition set.
type attempt struct {
	SetID string
	Fence string
}

// completion is the immutable SQL publication receipt.
type completion struct {
	StreamID            string  `json:"stream_id"`
	SetID               string  `json:"set_id"`
	Cursor              *string `json:"cursor"`
	CompletedGeneration string  `json:"completed_generation"`
	SourceRevision      string  `json:"source_revision"`
	ChangedCount        string  `json:"changed_count"`
	Readiness           string  `json:"readiness"`
}

// resume permits safe recovery before considering credentials or provider access.
type resume struct {
	State      string      `json:"state"`
	SetID      *string     `json:"set_id"`
	Fence      *string     `json:"fence"`
	Reason     *string     `json:"reason"`
	Completion *completion `json:"completion"`
}

// workContext contains only fresh SQL-authorized metadata for one live attempt.
type workContext struct {
	StreamID          string    `json:"stream_id"`
	SetID             string    `json:"set_id"`
	Fence             string    `json:"fence"`
	Provider          string    `json:"provider"`
	Environment       string    `json:"environment"`
	ItemID            string    `json:"item_id"`
	AccountID         string    `json:"account_id"`
	ItemGeneration    string    `json:"item_generation"`
	CredentialRef     string    `json:"credential_ref"`
	CredentialVersion string    `json:"credential_version"`
	BindingRevision   string    `json:"binding_revision"`
	StartingCursor    *string   `json:"starting_cursor"`
	RequestCursor     *string   `json:"request_cursor"`
	NextPageIndex     int       `json:"next_page_index"`
	PageCount         int       `json:"page_count"`
	ChangeCount       int       `json:"change_count"`
	ByteCount         int       `json:"byte_count"`
	FinalPage         bool      `json:"final_page"`
	LeaseUntil        time.Time `json:"lease_until"`
	Currency          struct {
		Code      string  `json:"code"`
		VersionID string  `json:"version_id"`
		MinorUnit int     `json:"minor_unit"`
		ValidFrom string  `json:"valid_from"`
		ValidTo   *string `json:"valid_to"`
	} `json:"currency"`
	Limits struct {
		MaxPages       int `json:"max_pages"`
		MaxChanges     int `json:"max_changes"`
		MaxBytes       int `json:"max_bytes"`
		MaxPageChanges int `json:"max_page_changes"`
	} `json:"limits"`
}

// expected translates trusted currency metadata without deriving it from provider amounts.
func (c workContext) expected() plaid.Expected {
	to := ""
	if c.Currency.ValidTo != nil {
		to = *c.Currency.ValidTo
	}
	return plaid.Expected{AccountID: c.AccountID, Currencies: []plaid.Currency{{Code: c.Currency.Code, VersionID: c.Currency.VersionID, MinorUnit: c.Currency.MinorUnit, ValidFrom: c.Currency.ValidFrom, ValidTo: to}}}
}

// reference exposes only the currently authorized secret-store lookup key.
func (c workContext) reference() CredentialReference {
	return CredentialReference{c.CredentialRef, c.CredentialVersion, c.ItemID, c.Environment}
}

// pageWire preserves decoder values and adds the SQL-owned cursor position.
type pageWire struct {
	PageIndex     int     `json:"page_index"`
	RequestCursor *string `json:"request_cursor"`
	plaid.Page
}

// repository is private so callers cannot publish arbitrary rows through Runner.
type repository interface {
	Resume(context.Context, Scope, string) (resume, error)
	Claim(context.Context, Scope, string) (attempt, error)
	Context(context.Context, Scope, attempt) (workContext, error)
	Stage(context.Context, Scope, attempt, int, json.RawMessage) error
	Complete(context.Context, Scope, attempt) (completion, error)
	Abandon(context.Context, Scope, attempt, string) error
	RejectDecode(context.Context, Scope, attempt, string) error
}
