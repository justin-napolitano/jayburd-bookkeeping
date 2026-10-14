package acquisition

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
)

// LocalEvidence is synthetic-only durable raw storage under an already opened private staging root.
type LocalEvidence struct{ store *staging.Store }

// NewLocalEvidence composes the existing secure local store without opening caller paths.
func NewLocalEvidence(store *staging.Store) (*LocalEvidence, error) {
	if store == nil {
		return nil, failure("configuration_error", false)
	}
	return &LocalEvidence{store}, nil
}

// evidenceManifest publishes scope, cursor and exact immutable chunk metadata last.
type evidenceManifest struct {
	Profile string      `json:"profile"`
	Key     EvidenceKey `json:"key"`
	Bytes   int         `json:"bytes"`
	SHA256  string      `json:"sha256"`
	Chunks  []int       `json:"chunks"`
}

// stableID derives an internal UUID from a domain-separated server-owned identity.
func stableID(parts ...string) string {
	h := sha256.New()
	for _, s := range parts {
		_, _ = fmt.Fprintf(h, "%d:%s", len(s), s)
	}
	b := h.Sum(nil)[:16]
	b[6] = (b[6] & 15) | 0x80
	b[8] = (b[8] & 63) | 0x80
	s := hex.EncodeToString(b)
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}

// validKey accepts only full scope and bounded server-side page identifiers.
func validKey(k EvidenceKey) bool {
	return validScope(k.Scope) && validID(k.SetID) && k.Page >= 0 && k.Page < 10 && validCursor(k.Cursor)
}

// evidenceRef prevents cross-stream or cross-page object reuse without accepting filesystem paths.
func evidenceRef(k EvidenceKey, part string) staging.Reference {
	return staging.Reference{Scope: staging.Scope{EntityID: k.Scope.EntityID, BookID: k.Scope.BookID}, ObjectID: stableID("acquisition-raw-v1", k.Scope.EntityID, k.Scope.BookID, k.Scope.StreamID, k.SetID, strconv.Itoa(k.Page), part)}
}

// byteSource transfers bounded in-memory bytes into the context-aware staging API.
type byteSource struct{ reader *bytes.Reader }

// Read checks cancellation before reading already owned memory.
func (s *byteSource) Read(ctx context.Context, b []byte) (int, error) {
	if e := ctx.Err(); e != nil {
		return 0, e
	}
	return s.reader.Read(b)
}

// Close owns no external resource and is safe during cancellation.
func (s *byteSource) Close(context.Context) error { return nil }

// storageError removes storage paths and distinguishes safe replay conflicts.
func storageError(e error) error {
	var s *staging.Error
	if errors.As(e, &s) {
		switch s.Code {
		case "object_conflict":
			return failure("evidence_conflict", false)
		case "object_deleted":
			return failure("evidence_deleted", false)
		}
	}
	return failure("evidence_unavailable", true)
}

// Put stores complete raw bytes before exposing their immutable manifest.
func (s *LocalEvidence) Put(ctx context.Context, k EvidenceKey, raw []byte) error {
	if !validKey(k) || len(raw) == 0 || len(raw) > MaxRawBytes {
		return failure("invalid_evidence", false)
	}
	existing, found, e := s.Get(ctx, k)
	if e != nil {
		return e
	}
	if found {
		if !bytes.Equal(existing, raw) {
			return failure("evidence_conflict", false)
		}
		return nil
	}
	sum := sha256.Sum256(raw)
	m := evidenceManifest{Profile: "acquisition-raw-v1", Key: k, Bytes: len(raw), SHA256: hex.EncodeToString(sum[:]), Chunks: []int{}}
	for pos, index := 0, 0; pos < len(raw); index++ {
		end := pos + staging.MaxBytes
		if end > len(raw) {
			end = len(raw)
		}
		chunk := raw[pos:end]
		if _, e = s.store.Put(ctx, evidenceRef(k, "chunk-"+strconv.Itoa(index)), &byteSource{bytes.NewReader(chunk)}); e != nil {
			return storageError(e)
		}
		m.Chunks = append(m.Chunks, len(chunk))
		pos = end
	}
	body, _ := json.Marshal(m)
	if _, e = s.store.Put(ctx, evidenceRef(k, "manifest"), &byteSource{bytes.NewReader(body)}); e != nil {
		return storageError(e)
	}
	stored, found, e := s.Get(ctx, k)
	if e != nil {
		return e
	}
	if !found || !bytes.Equal(stored, raw) {
		return failure("evidence_conflict", false)
	}
	return nil
}

// Get verifies complete scoped bytes; a digest alone never establishes replay equality.
func (s *LocalEvidence) Get(ctx context.Context, k EvidenceKey) ([]byte, bool, error) {
	if !validKey(k) {
		return nil, false, failure("invalid_evidence", false)
	}
	body, _, e := s.store.Read(ctx, evidenceRef(k, "manifest"))
	if e != nil {
		var se *staging.Error
		if errors.As(e, &se) && se.Code == "not_found" {
			return nil, false, nil
		}
		return nil, false, storageError(e)
	}
	if len(body) > 4096 {
		return nil, false, failure("invalid_evidence", false)
	}
	var m evidenceManifest
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil {
		return nil, false, failure("invalid_evidence", false)
	}
	var tail any
	if d.Decode(&tail) != io.EOF {
		return nil, false, failure("invalid_evidence", false)
	}
	expected, _ := json.Marshal(k)
	actual, _ := json.Marshal(m.Key)
	if m.Profile != "acquisition-raw-v1" || !bytes.Equal(expected, actual) || m.Bytes <= 0 || m.Bytes > MaxRawBytes || len(m.Chunks) < 1 || len(m.Chunks) > 2 {
		return nil, false, failure("invalid_evidence", false)
	}
	out := make([]byte, 0, m.Bytes)
	for i, n := range m.Chunks {
		if n <= 0 || n > staging.MaxBytes || len(out)+n > m.Bytes {
			return nil, false, failure("invalid_evidence", false)
		}
		chunk, _, e := s.store.Read(ctx, evidenceRef(k, "chunk-"+strconv.Itoa(i)))
		if e != nil {
			return nil, false, storageError(e)
		}
		if len(chunk) != n {
			return nil, false, failure("invalid_evidence", false)
		}
		out = append(out, chunk...)
	}
	sum := sha256.Sum256(out)
	if len(out) != m.Bytes || hex.EncodeToString(sum[:]) != m.SHA256 {
		return nil, false, failure("invalid_evidence", false)
	}
	return out, true, nil
}

// Delete tombstones the manifest first, preventing later partial writes from becoming readable.
func (s *LocalEvidence) Delete(ctx context.Context, k EvidenceKey) error {
	if !validKey(k) {
		return failure("invalid_evidence", false)
	}
	for _, part := range []string{"manifest", "chunk-0", "chunk-1"} {
		if e := s.store.Delete(ctx, evidenceRef(k, part)); e != nil {
			return storageError(e)
		}
	}
	return nil
}
