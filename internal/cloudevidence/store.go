// Package cloudevidence stores acquisition pages in a private, generation-pinned GCS boundary.
package cloudevidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
)

// AccessTokenProvider is injected workload identity; the store never discovers credentials.
type AccessTokenProvider interface {
	AccessToken(context.Context) (string, error)
}

// Store owns one explicit development bucket and never exposes raw provider diagnostics.
type Store struct {
	bucket string
	tokens AccessTokenProvider
	client *http.Client
}

var bucketName = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]-jayburd-dev-acquisition-evidence$`)
var uuid = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

// New fixes the endpoint and credential boundary without network or filesystem access.
func New(bucket string, tokens AccessTokenProvider, transport http.RoundTripper) (*Store, error) {
	if !bucketName.MatchString(bucket) || tokens == nil || transport == nil {
		return nil, failed("configuration_error", false)
	}
	return &Store{bucket, tokens, &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// failed preserves the runner's fixed storage failure vocabulary without wrapping remote text.
func failed(code string, retry bool) error { return &acquisition.Failure{Code: code, Retryable: retry} }

// keyName derives an opaque identity from every server-owned scope/page/cursor field.
func keyName(k acquisition.EvidenceKey) (string, error) {
	for _, id := range []string{k.Scope.EntityID, k.Scope.BookID, k.Scope.StreamID, k.SetID} {
		if !uuid.MatchString(id) || id == "00000000-0000-0000-0000-000000000000" {
			return "", failed("invalid_evidence", false)
		}
	}
	if k.Page < 0 || k.Page >= 10 || (k.Cursor != nil && (len(*k.Cursor) == 0 || len(*k.Cursor) > 256)) {
		return "", failed("invalid_evidence", false)
	}
	b, e := json.Marshal(k)
	if e != nil {
		return "", failed("invalid_evidence", false)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// metadata pins a concrete generation and its independent GCS checksum before downloading.
type metadata struct {
	Name       string `json:"name"`
	Generation string `json:"generation"`
	Size       string `json:"size"`
	CRC32C     string `json:"crc32c"`
}

// request bounds each authenticated call and discards error bodies, including OAuth diagnostics.
func (s *Store) request(parent context.Context, method, address string, body []byte, limit int) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	token, e := s.tokens.AccessToken(ctx)
	if e != nil || len(token) == 0 || len(token) > 16384 || strings.ContainsAny(token, "\r\n") {
		return nil, 0, failed("evidence_unavailable", true)
	}
	r, e := http.NewRequestWithContext(ctx, method, address, bytes.NewReader(body))
	if e != nil {
		return nil, 0, failed("invalid_evidence", false)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/octet-stream")
	response, e := s.client.Do(r)
	if e != nil {
		return nil, 0, failed("evidence_unavailable", true)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 && response.StatusCode != 201 && response.StatusCode != 204 {
		return nil, response.StatusCode, nil
	}
	data, e := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if e != nil || len(data) > limit || ctx.Err() != nil {
		return nil, 0, failed("evidence_unavailable", true)
	}
	return data, response.StatusCode, nil
}

// objectURL permits only hashed application-owned object names in the configured bucket.
func (s *Store) objectURL(name string) string {
	return "https://storage.googleapis.com/storage/v1/b/" + s.bucket + "/o/" + url.PathEscape(name)
}

// meta reads only selected object fields and rejects malformed or oversized records.
func (s *Store) meta(ctx context.Context, name string, max int) (metadata, bool, error) {
	raw, status, e := s.request(ctx, http.MethodGet, s.objectURL(name)+"?fields=name,generation,size,crc32c", nil, 2048)
	if e != nil {
		return metadata{}, false, e
	}
	if status == 404 {
		return metadata{}, false, nil
	}
	if status != 200 {
		return metadata{}, false, failed("evidence_unavailable", true)
	}
	var m metadata
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF {
		return metadata{}, false, failed("invalid_evidence", false)
	}
	generation, e := strconv.ParseUint(m.Generation, 10, 64)
	if e != nil || generation == 0 || strconv.FormatUint(generation, 10) != m.Generation {
		return metadata{}, false, failed("invalid_evidence", false)
	}
	size, e := strconv.Atoi(m.Size)
	checksum, ce := base64.StdEncoding.DecodeString(m.CRC32C)
	if m.Name != name || e != nil || size <= 0 || size > max || ce != nil || len(checksum) != 4 {
		return metadata{}, false, failed("invalid_evidence", false)
	}
	return m, true, nil
}

// deleted treats every valid tombstone as permanent, independent of late raw uploads.
func (s *Store) deleted(ctx context.Context, key string) (bool, error) {
	_, yes, e := s.meta(ctx, "deleted/"+key, 64)
	return yes, e
}

// read pins the current generation so concurrent deletion cannot substitute another body.
func (s *Store) read(ctx context.Context, name string) ([]byte, bool, error) {
	m, yes, e := s.meta(ctx, name, acquisition.MaxRawBytes)
	if e != nil || !yes {
		return nil, yes, e
	}
	raw, status, e := s.request(ctx, http.MethodGet, s.objectURL(name)+"?alt=media&generation="+m.Generation, nil, acquisition.MaxRawBytes)
	if e != nil {
		return nil, false, e
	}
	if status != 200 {
		return nil, false, failed("evidence_unavailable", true)
	}
	sum := crc32.Checksum(raw, crc32.MakeTable(crc32.Castagnoli))
	var c [4]byte
	binary.BigEndian.PutUint32(c[:], sum)
	if strconv.Itoa(len(raw)) != m.Size || base64.StdEncoding.EncodeToString(c[:]) != m.CRC32C {
		return nil, false, failed("invalid_evidence", false)
	}
	return raw, true, nil
}

// create performs a single conditional upload; an ambiguous result is resolved by subsequent reads.
func (s *Store) create(ctx context.Context, name string, body []byte) error {
	q := url.Values{"uploadType": {"media"}, "name": {name}, "ifGenerationMatch": {"0"}, "fields": {"name,generation"}}
	_, status, e := s.request(ctx, http.MethodPost, "https://storage.googleapis.com/upload/storage/v1/b/"+s.bucket+"/o?"+q.Encode(), body, 2048)
	if e != nil {
		return e
	}
	if status != 200 && status != 201 && status != 412 {
		return failed("evidence_unavailable", true)
	}
	return nil
}

// Put creates one immutable page and verifies exact replay; tombstones always win.
func (s *Store) Put(ctx context.Context, k acquisition.EvidenceKey, raw []byte) error {
	key, e := keyName(k)
	if e != nil {
		return e
	}
	if len(raw) == 0 || len(raw) > acquisition.MaxRawBytes {
		return failed("invalid_evidence", false)
	}
	old, yes, e := s.Get(ctx, k)
	if e != nil {
		return e
	}
	if yes {
		if !bytes.Equal(old, raw) {
			return failed("evidence_conflict", false)
		}
		return nil
	}
	if e = s.create(ctx, "raw/"+key, raw); e != nil {
		return e
	}
	stored, yes, e := s.Get(ctx, k)
	if e != nil {
		return e
	}
	if !yes || !bytes.Equal(stored, raw) {
		return failed("evidence_conflict", false)
	}
	return nil
}

// Get checks deletion before and after a generation-pinned read, never returning a known deleted page.
func (s *Store) Get(ctx context.Context, k acquisition.EvidenceKey) ([]byte, bool, error) {
	key, e := keyName(k)
	if e != nil {
		return nil, false, e
	}
	yes, e := s.deleted(ctx, key)
	if e != nil {
		return nil, false, e
	}
	if yes {
		return nil, false, failed("evidence_deleted", false)
	}
	raw, found, e := s.read(ctx, "raw/"+key)
	if e != nil {
		return nil, false, e
	}
	yes, e = s.deleted(ctx, key)
	if e != nil {
		return nil, false, e
	}
	if yes {
		return nil, false, failed("evidence_deleted", false)
	}
	return raw, found, nil
}

// Delete permanently denies reads first; generation-conditional removal cannot erase a newer unrelated object.
func (s *Store) Delete(ctx context.Context, k acquisition.EvidenceKey) error {
	key, e := keyName(k)
	if e != nil {
		return e
	}
	if e = s.create(ctx, "deleted/"+key, []byte("deleted-v1")); e != nil {
		return e
	}
	yes, e := s.deleted(ctx, key)
	if e != nil {
		return e
	}
	if !yes {
		return failed("evidence_unavailable", true)
	}
	m, yes, e := s.meta(ctx, "raw/"+key, acquisition.MaxRawBytes)
	if e != nil || !yes {
		return e
	}
	_, status, e := s.request(ctx, http.MethodDelete, s.objectURL("raw/"+key)+"?ifGenerationMatch="+m.Generation, nil, 2048)
	if e != nil {
		return e
	}
	if status != 204 && status != 404 {
		return failed("evidence_unavailable", true)
	}
	return nil
}
