package importer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
)

// Scope carries verified server identity; SQL remains the current authority.
type Scope struct{ Actor, Entity, Book string }

// Registration binds statement identity and dates to an existing approved namespace.
type Registration struct {
	SourceAccountID string `json:"source_account_id"`
	NamespaceID     string `json:"namespace_id"`
	BindingRevision string `json:"binding_revision"`
	StatementRef    string `json:"statement_ref"`
	StartsOn        string `json:"starts_on"`
	EndsBefore      string `json:"ends_before"`
	Profile         string `json:"profile"`
}

// Storage accepts generated references only; Put assumes ownership of its Source.
type Storage interface {
	Put(context.Context, staging.Reference, staging.Source) (staging.Receipt, error)
	Read(context.Context, staging.Reference) ([]byte, staging.Receipt, error)
}

// repository hides publication from callers so arbitrary decoded rows cannot bypass Decode.
type repository interface {
	register(context.Context, Scope, string, Registration) (json.RawMessage, error)
	metadata(context.Context, Scope, string) (json.RawMessage, error)
	batch(context.Context, Scope, string) (json.RawMessage, error)
	review(context.Context, Scope, string) (json.RawMessage, error)
	publish(context.Context, Scope, string, []byte) (json.RawMessage, error)
}

// Service separates committed authorization, owned storage and atomic SQL publication.
type Service struct {
	repo    repository
	storage Storage
}

// New composes a restricted API database pool and private synthetic storage adapter.
func New(db Beginner, store Storage) *Service {
	return &Service{repo: &SQLRepository{DB: db}, storage: store}
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// validID rejects caller path syntax and the reserved all-zero UUID.
func validID(s string) bool {
	return len(s) == 36 && uuidPattern.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}

// validScope validates syntax only; it never substitutes for SQL authorization.
func validScope(s Scope) bool { return validID(s.Actor) && validID(s.Entity) && validID(s.Book) }

// validRevision enforces the existing nonnegative bigint string wire contract.
func validRevision(s string) bool {
	n, e := strconv.ParseInt(s, 10, 64)
	return e == nil && n >= 0 && strconv.FormatInt(n, 10) == s
}

// Register persists identity before upload; unknown commit requires the same key/body replay.
func (s *Service) Register(ctx context.Context, scope Scope, key string, input Registration) (json.RawMessage, error) {
	if s == nil || s.repo == nil {
		return nil, failed("configuration_error")
	}
	if !validScope(scope) || !validID(input.SourceAccountID) || !validID(input.NamespaceID) || !validRevision(input.BindingRevision) || input.BindingRevision == "0" || len(key) < 1 || len(key) > 128 || len(input.StatementRef) < 1 || len(input.StatementRef) > 64 || len(input.StartsOn) != 10 || len(input.EndsBefore) != 10 || input.Profile != ingestion.Profile {
		return nil, failed("invalid_request")
	}
	return s.repo.register(ctx, scope, key, input)
}

// Batch retrieves currently authorized batch metadata without touching raw evidence.
func (s *Service) Batch(ctx context.Context, scope Scope, batch string) (json.RawMessage, error) {
	if s == nil || s.repo == nil {
		return nil, failed("configuration_error")
	}
	if !validScope(scope) || !validID(batch) {
		return nil, failed("invalid_request")
	}
	return s.repo.batch(ctx, scope, batch)
}

// contextValue carries only SQL-owned fields needed to address and decode evidence.
type contextValue struct {
	BatchID          string `json:"batch_id"`
	ObjectID         string `json:"object_id"`
	SourceAccountID  string `json:"source_account_id"`
	NamespaceID      string `json:"namespace_id"`
	BindingRevision  string `json:"binding_revision"`
	SourceRevision   string `json:"source_revision"`
	State            string `json:"state"`
	Profile          string `json:"profile"`
	SourceAccountRef string `json:"source_account_ref"`
	StartsOn         string `json:"starts_on"`
	EndsBefore       string `json:"ends_before"`
	Currency         struct {
		Code      string  `json:"code"`
		VersionID string  `json:"version_id"`
		MinorUnit *int    `json:"minor_unit"`
		ValidFrom string  `json:"valid_from"`
		ValidTo   *string `json:"valid_to"`
	} `json:"currency"`
}

// expectations translates the SQL currency snapshot without defaults or rounding.
func (c contextValue) expectations() ingestion.Expectations {
	return ingestion.Expectations{SourceAccountRef: c.SourceAccountRef, StartsOn: c.StartsOn, EndsBefore: c.EndsBefore, Currency: ingestion.Currency{Code: c.Currency.Code, VersionID: c.Currency.VersionID, MinorUnit: *c.Currency.MinorUnit, ValidFrom: c.Currency.ValidFrom, ValidTo: *c.Currency.ValidTo}}
}

// readContext validates server metadata before any object operation; malformed metadata fails closed.
func (s *Service) readContext(ctx context.Context, scope Scope, batch string) (contextValue, staging.Reference, error) {
	var c contextValue
	if !validScope(scope) || !validID(batch) {
		return c, staging.Reference{}, failed("invalid_request")
	}
	raw, err := s.repo.metadata(ctx, scope, batch)
	if err != nil {
		return c, staging.Reference{}, err
	}
	if len(raw) > 2<<20 || json.Unmarshal(raw, &c) != nil || c.BatchID != batch || !validID(c.ObjectID) || !validID(c.SourceAccountID) || !validID(c.NamespaceID) || !validRevision(c.BindingRevision) || c.BindingRevision == "0" || !validRevision(c.SourceRevision) || c.Profile != ingestion.Profile || c.Currency.MinorUnit == nil || c.Currency.ValidTo == nil {
		return contextValue{}, staging.Reference{}, failed("invalid_metadata")
	}
	switch c.State {
	case "received", "preview_ready", "rejected", "imported":
	default:
		return contextValue{}, staging.Reference{}, failed("invalid_metadata")
	}
	// Decode rejects invalid expectations before reading; empty input otherwise produces invalid_header.
	_, err = ingestion.Decode(bytes.NewReader(nil), c.expectations())
	var invalid *ingestion.DecodeError
	if !errors.As(err, &invalid) || len(invalid.Diagnostics) != 1 || invalid.Diagnostics[0].Code != "invalid_header" {
		return contextValue{}, staging.Reference{}, failed("invalid_metadata")
	}
	return c, staging.Reference{Scope: staging.Scope{EntityID: scope.Entity, BookID: scope.Book}, ObjectID: c.ObjectID}, nil
}

// closeDenied releases the transferred source with a fresh bounded cleanup context.
func closeDenied(source staging.Source) {
	if source == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = source.Close(ctx)
}

// storageFailure maps only a fixed allowlist and never returns implementation diagnostics.
func storageFailure(err error) error {
	var e *staging.Error
	if errors.As(err, &e) {
		switch e.Code {
		case "storage_unknown":
			return failed("storage_unknown")
		case "object_conflict":
			return failed("statement_identity_conflict")
		case "too_large":
			return failed("request_too_large")
		case "not_found", "object_deleted":
			return failed("evidence_unavailable")
		case "empty_object":
			return failed("invalid_statement")
		}
	}
	return failed("storage_unavailable")
}

// Upload owns source immediately; authorization commits before Put and publication reauthorizes.
func (s *Service) Upload(parent context.Context, scope Scope, batch string, source staging.Source) (json.RawMessage, error) {
	owned := true
	defer func() {
		if owned {
			closeDenied(source)
		}
	}()
	if s == nil || s.repo == nil || s.storage == nil {
		return nil, failed("configuration_error")
	}
	if source == nil {
		return nil, failed("invalid_request")
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	_, ref, err := s.readContext(ctx, scope, batch)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, failed("temporarily_unavailable")
	}
	owned = false
	receipt, err := s.storage.Put(ctx, ref, source)
	if err != nil {
		return nil, storageFailure(err)
	}
	if receipt.Reference != ref || receipt.ByteCount < 1 || receipt.ByteCount > ingestion.MaxBytes {
		return nil, failed("invalid_metadata")
	}
	// Reauthorization after I/O also catches permission or binding changes before the read.
	return s.preview(ctx, scope, batch, &receipt)
}

// Preview reconciles retained immutable evidence after an uncertain upload/publication.
func (s *Service) Preview(parent context.Context, scope Scope, batch string) (json.RawMessage, error) {
	if s == nil || s.repo == nil || s.storage == nil {
		return nil, failed("configuration_error")
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	return s.preview(ctx, scope, batch, nil)
}

// preview decodes only validated retained bytes and sends complete results to reauthorizing SQL.
func (s *Service) preview(ctx context.Context, scope Scope, batch string, put *staging.Receipt) (json.RawMessage, error) {
	metadata, ref, err := s.readContext(ctx, scope, batch)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, failed("temporarily_unavailable")
	}
	raw, receipt, err := s.storage.Read(ctx, ref)
	if err != nil {
		return nil, storageFailure(err)
	}
	if len(raw) < 1 || len(raw) > ingestion.MaxBytes || receipt.Reference != ref || receipt.ByteCount != len(raw) || receipt.SHA256 != sha256.Sum256(raw) || (put != nil && *put != receipt) {
		return nil, failed("invalid_metadata")
	}
	if ctx.Err() != nil {
		return nil, failed("temporarily_unavailable")
	}
	statement, decodeErr := ingestion.Decode(bytes.NewReader(raw), metadata.expectations())
	body, err := publication(receipt, statement, decodeErr)
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, failed("temporarily_unavailable")
	}
	return s.repo.publish(ctx, scope, batch, body)
}

// rowValue preserves original and canonical exact-money strings in the SQL publication wire.
type rowValue struct {
	Record            int    `json:"record"`
	SourceAccountRef  string `json:"source_account_ref"`
	EntryID           string `json:"entry_id"`
	BookedOn          string `json:"booked_on"`
	Currency          string `json:"currency"`
	Direction         string `json:"direction"`
	Amount            string `json:"amount"`
	Description       string `json:"description"`
	CanonicalAmount   string `json:"canonical_amount"`
	CurrencyVersionID string `json:"currency_version_id"`
}

// diagnosticValue contains only decoder-owned fixed codes and ordinals.
type diagnosticValue struct {
	Code   string `json:"code"`
	Record int    `json:"record"`
	Field  string `json:"field"`
}

// publication has no public caller: only strict Decode produces rows or rejection evidence.
func publication(receipt staging.Receipt, statement ingestion.Statement, err error) ([]byte, error) {
	value := map[string]any{"profile": ingestion.Profile, "sha256": hex.EncodeToString(receipt.SHA256[:]), "byte_count": receipt.ByteCount}
	if err != nil {
		var d *ingestion.DecodeError
		if !errors.As(err, &d) {
			return nil, failed("invalid_statement")
		}
		diagnostics := make([]diagnosticValue, len(d.Diagnostics))
		for i, v := range d.Diagnostics {
			diagnostics[i] = diagnosticValue{v.Code, v.Record, v.Field}
		}
		value["diagnostics"] = diagnostics
		value["total"] = d.Total
		value["truncated"] = d.Truncated
	} else {
		rows := make([]rowValue, len(statement.Rows))
		for i, r := range statement.Rows {
			rows[i] = rowValue{r.Record, r.SourceAccountRef, r.EntryID, r.BookedOn, r.Currency, r.Direction, r.Amount, r.Description, r.CanonicalAmount, r.CurrencyVersionID}
		}
		value["rows"] = rows
	}
	body, e := json.Marshal(value)
	if e != nil || len(body) > 2<<20 {
		return nil, failed("invalid_statement")
	}
	return body, nil
}

// register calls the exact scoped statement-registration function with unchanged typed metadata.
func (r *SQLRepository) register(ctx context.Context, s Scope, key string, v Registration) (json.RawMessage, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, failed("invalid_request")
	}
	return r.call(ctx, "select banking.register_import_batch($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)", s.Actor, s.Entity, s.Book, key, string(body))
}

// metadata obtains an import-authorized SQL snapshot before any evidence access.
func (r *SQLRepository) metadata(ctx context.Context, s Scope, batch string) (json.RawMessage, error) {
	return r.call(ctx, "select banking.import_context($1::uuid,$2::uuid,$3::uuid,$4::uuid)", s.Actor, s.Entity, s.Book, batch)
}

// publish commits a complete decoder result with fresh authorization and immutable identity checks.
func (r *SQLRepository) publish(ctx context.Context, s Scope, batch string, body []byte) (json.RawMessage, error) {
	return r.call(ctx, "select banking.publish_import_preview($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text)", s.Actor, s.Entity, s.Book, batch, string(body))
}

// Review retrieves one currently readable review item without reading raw evidence.
func (s *Service) Review(ctx context.Context, scope Scope, id string) (json.RawMessage, error) {
	if s == nil || s.repo == nil {
		return nil, failed("configuration_error")
	}
	if !validScope(scope) || !validID(id) {
		return nil, failed("invalid_request")
	}
	return s.repo.review(ctx, scope, id)
}

// batch reads the public wrapper without disclosing private object or decoder metadata.
func (r *SQLRepository) batch(ctx context.Context, s Scope, id string) (json.RawMessage, error) {
	return r.call(ctx, "select banking.get_import_batch($1::uuid,$2::uuid,$3::uuid,$4::uuid)", s.Actor, s.Entity, s.Book, id)
}

// review reads only the scoped SQL review projection, never a raw source object.
func (r *SQLRepository) review(ctx context.Context, s Scope, id string) (json.RawMessage, error) {
	return r.call(ctx, "select banking.get_review_item($1::uuid,$2::uuid,$3::uuid,$4::uuid)", s.Actor, s.Entity, s.Book, id)
}
