package importer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
)

const testActor = "00000000-0000-4000-8000-000000000001"
const testEntity = "00000000-0000-4000-8000-000000000002"
const testBook = "00000000-0000-4000-8000-000000000003"
const testBatch = "00000000-0000-4000-8000-000000000004"
const testObject = "00000000-0000-4000-8000-000000000005"
const testCurrency = "00000000-0000-4000-8000-000000000006"
const testCSV = "source_account_ref,entry_id,booked_on,currency,direction,amount,description\r\nsynthetic,a,2026-01-02,USD,inflow,0012.500,Fictional deposit\r\n"

var testScope = Scope{testActor, testEntity, testBook}

// fakeRepository records publication calls and scripts fresh authorization failures.
type fakeRepository struct {
	meta             json.RawMessage
	calls, publishes int
	denyAt           int
	published        []byte
	publishErr       error
}

// register preserves fake ownership boundary without exercising SQL registration.
func (r *fakeRepository) register(context.Context, Scope, string, Registration) (json.RawMessage, error) {
	return r.meta, nil
}

// metadata models current SQL authorization independently at every call.
func (r *fakeRepository) metadata(context.Context, Scope, string) (json.RawMessage, error) {
	r.calls++
	if r.calls == r.denyAt {
		return nil, failed("not_found")
	}
	return r.meta, nil
}

// batch returns only the fake public response.
func (r *fakeRepository) batch(context.Context, Scope, string) (json.RawMessage, error) {
	return []byte(`{"state":"received"}`), nil
}

// review returns only the fake public response.
func (r *fakeRepository) review(context.Context, Scope, string) (json.RawMessage, error) {
	return []byte(`{"state":"needs_review"}`), nil
}

// publish captures exact server-owned publication and suppresses failed outcomes.
func (r *fakeRepository) publish(_ context.Context, _ Scope, _ string, body []byte) (json.RawMessage, error) {
	r.publishes++
	r.published = append([]byte(nil), body...)
	if r.publishErr != nil {
		return nil, r.publishErr
	}
	return []byte(`{"state":"preview_ready"}`), nil
}

// ownedSource counts ownership transfers and never wraps an unbounded external reader.
type ownedSource struct {
	reader        *bytes.Reader
	reads, closes int
	cleanupOK     bool
	fault         error
}

// Read respects context and optionally returns a private synthetic fault.
func (s *ownedSource) Read(ctx context.Context, b []byte) (int, error) {
	s.reads++
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	if s.fault != nil {
		return 0, s.fault
	}
	return s.reader.Read(b)
}

// Close proves cleanup has a live independent deadline.
func (s *ownedSource) Close(ctx context.Context) error {
	s.closes++
	d, ok := ctx.Deadline()
	s.cleanupOK = ok && time.Until(d) > 0 && time.Until(d) <= time.Second
	return nil
}

// source creates a context-aware owned in-memory synthetic source.
func source(raw string) *ownedSource { return &ownedSource{reader: bytes.NewReader([]byte(raw))} }

// testMetadata produces a complete authoritative-looking synthetic SQL snapshot.
func testMetadata() json.RawMessage {
	return json.RawMessage(`{"batch_id":"` + testBatch + `","object_id":"` + testObject + `","source_account_id":"00000000-0000-4000-8000-000000000007","namespace_id":"00000000-0000-4000-8000-000000000008","binding_revision":"1","source_revision":"0","state":"received","profile":"` + ingestion.Profile + `","source_account_ref":"synthetic","starts_on":"2026-01-01","ends_before":"2026-02-01","currency":{"code":"USD","version_id":"` + testCurrency + `","minor_unit":2,"valid_from":"1900-01-01","valid_to":""}}`)
}

// fixture uses actual private staging with fake SQL authority for isolated fault injection.
func fixture(t *testing.T) (*Service, *fakeRepository, *staging.Store) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := staging.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	repo := &fakeRepository{meta: testMetadata()}
	return &Service{repo: repo, storage: store}, repo, store
}

// TestUploadAuthorizationOwnership proves denied/invalid requests consume no bytes and always close.
func TestUploadAuthorizationOwnership(t *testing.T) {
	for _, scenario := range []string{"denied", "invalid_scope", "invalid_metadata", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			service, repo, _ := fixture(t)
			scope := testScope
			ctx := context.Background()
			switch scenario {
			case "denied":
				repo.denyAt = 1
			case "invalid_scope":
				scope.Book = "../private"
			case "invalid_metadata":
				repo.meta = bytes.ReplaceAll(repo.meta, []byte(testObject), []byte("../../file"))
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			input := source(testCSV)
			body, err := service.Upload(ctx, scope, testBatch, input)
			if err == nil || len(body) != 0 || input.reads != 0 || input.closes != 1 || !input.cleanupOK || repo.publishes != 0 {
				t.Fatalf("ownership failure: %v reads=%d closes=%d", err, input.reads, input.closes)
			}
		})
	}
}

// TestUploadExactReplay preserves cents/lexemes and refuses changed bytes at the same object.
func TestUploadExactReplay(t *testing.T) {
	service, repo, _ := fixture(t)
	for i := 0; i < 2; i++ {
		input := source(testCSV)
		if _, err := service.Upload(context.Background(), testScope, testBatch, input); err != nil {
			t.Fatal(err)
		}
		if input.closes != 1 {
			t.Fatal("source double close")
		}
	}
	var decoded struct {
		Rows []rowValue `json:"rows"`
	}
	if err := json.Unmarshal(repo.published, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Rows) != 1 || decoded.Rows[0].Amount != "0012.500" || decoded.Rows[0].CanonicalAmount != "12.5" || decoded.Rows[0].CurrencyVersionID != testCurrency {
		t.Fatal("exact currency provenance lost")
	}
	_, err := service.Upload(context.Background(), testScope, testBatch, source(strings.Replace(testCSV, "0012.500", "12.51", 1)))
	if err == nil || err.Error() != "statement_identity_conflict" || repo.publishes != 2 {
		t.Fatalf("changed evidence replaced: %v", err)
	}
}

// TestUploadAtomicRejection ensures malformed records publish diagnostics and no usable rows.
func TestUploadAtomicRejection(t *testing.T) {
	service, repo, _ := fixture(t)
	_, err := service.Upload(context.Background(), testScope, testBatch, source(strings.Replace(testCSV, "0012.500", "12.501", 1)))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err = json.Unmarshal(repo.published, &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["rows"]; ok || len(body["diagnostics"]) == 0 || string(body["total"]) != "1" || bytes.Contains(body["diagnostics"], []byte("12.501")) {
		t.Fatal("partial rows or raw values escaped")
	}
}

// TestUploadRevocationAndUnknownCommit preserves private evidence for safe later reconciliation.
func TestUploadRevocationAndUnknownCommit(t *testing.T) {
	for _, scenario := range []string{"revoke_after_put", "publish_denied", "unknown_commit"} {
		t.Run(scenario, func(t *testing.T) {
			service, repo, _ := fixture(t)
			if scenario == "revoke_after_put" {
				repo.denyAt = 2
			} else if scenario == "publish_denied" {
				repo.publishErr = failed("not_found")
			} else {
				repo.publishErr = failed("commit_unknown")
			}
			input := source(testCSV)
			body, err := service.Upload(context.Background(), testScope, testBatch, input)
			if err == nil || len(body) != 0 || input.closes != 1 {
				t.Fatal("unconfirmed result escaped")
			}
			if scenario == "revoke_after_put" && repo.publishes != 0 {
				t.Fatal("revoked authority published")
			}
			repo.denyAt = 0
			repo.publishErr = nil
			if _, err = service.Preview(context.Background(), testScope, testBatch); err != nil {
				t.Fatalf("lost retained evidence: %v", err)
			}
		})
	}
}

// TestUploadLimitsAndSafeErrors ensures oversized/private reader errors do not publish partial evidence.
func TestUploadLimitsAndSafeErrors(t *testing.T) {
	for _, scenario := range []string{"limit", "reader"} {
		t.Run(scenario, func(t *testing.T) {
			service, repo, _ := fixture(t)
			input := source(strings.Repeat("x", ingestion.MaxBytes+1))
			if scenario == "reader" {
				input.fault = errors.New("private account at /private/secret")
			}
			body, err := service.Upload(context.Background(), testScope, testBatch, input)
			if err == nil || len(body) != 0 || repo.publishes != 0 || input.closes != 1 || strings.Contains(err.Error(), "private") {
				t.Fatalf("unsafe limit: %v", err)
			}
		})
	}
}

// checkedStorage injects a read-boundary fault while preserving actual private writes.
type checkedStorage struct {
	base      Storage
	onRead    func(context.Context)
	onPut     func(context.Context)
	corrupt   bool
	readError error
}

// Put delegates ownership exactly once to the real storage adapter.
func (s checkedStorage) Put(ctx context.Context, r staging.Reference, source staging.Source) (staging.Receipt, error) {
	if s.onPut != nil {
		s.onPut(ctx)
	}
	return s.base.Put(ctx, r, source)
}

// Read injects only explicit synthetic corruption/cancellation points.
func (s checkedStorage) Read(ctx context.Context, r staging.Reference) ([]byte, staging.Receipt, error) {
	if s.onRead != nil {
		s.onRead(ctx)
	}
	if s.readError != nil {
		return nil, staging.Receipt{}, s.readError
	}
	raw, receipt, err := s.base.Read(ctx, r)
	if s.corrupt {
		receipt.SHA256[0] ^= 1
	}
	return raw, receipt, err
}

// TestPreviewIntegrityAndCancellation never publishes corrupt receipts or work past cancellation.
func TestPreviewIntegrityAndCancellation(t *testing.T) {
	for _, scenario := range []string{"digest", "cancel", "read_error"} {
		t.Run(scenario, func(t *testing.T) {
			service, repo, store := fixture(t)
			ref := staging.Reference{Scope: staging.Scope{EntityID: testEntity, BookID: testBook}, ObjectID: testObject}
			if _, err := store.Put(context.Background(), ref, source(testCSV)); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			shim := checkedStorage{base: store}
			switch scenario {
			case "digest":
				shim.corrupt = true
			case "cancel":
				shim.onRead = func(context.Context) { cancel() }
			case "read_error":
				shim.readError = errors.New("private filesystem detail")
			}
			service.storage = shim
			body, err := service.Preview(ctx, testScope, testBatch)
			if err == nil || len(body) != 0 || repo.publishes != 0 || strings.Contains(err.Error(), "private") {
				t.Fatalf("unsafe read: %v", err)
			}
		})
	}
}

// TestMetadataValidationBeforeIO rejects malformed SQL currency snapshots instead of applying defaults.
func TestMetadataValidationBeforeIO(t *testing.T) {
	for _, pair := range [][2]string{{`"minor_unit":2`, `"minor_unit":null`}, {`"valid_to":""`, `"valid_to":null`}, {`"source_revision":"0"`, `"source_revision":"00"`}, {`"starts_on":"2026-01-01"`, `"starts_on":"2026-02-30"`}, {`"code":"USD"`, `"code":"usd"`}, {`"state":"received"`, `"state":"cancelled"`}} {
		t.Run(pair[1], func(t *testing.T) {
			service, repo, _ := fixture(t)
			repo.meta = bytes.ReplaceAll(repo.meta, []byte(pair[0]), []byte(pair[1]))
			input := source(testCSV)
			_, err := service.Upload(context.Background(), testScope, testBatch, input)
			if err == nil || err.Error() != "invalid_metadata" || input.reads != 0 || input.closes != 1 || repo.publishes != 0 {
				t.Fatalf("invalid SQL metadata accepted: %v", err)
			}
		})
	}
}
