package providersecrets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

var bucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

// GCSRegistry persists metadata only with create-if-absent, never token payloads.
type GCSRegistry struct {
	config GCSConfig
	http   *http.Client
}

// NewGCSRegistry requires explicit private bucket configuration and managed token injection.
func NewGCSRegistry(c GCSConfig) (*GCSRegistry, error) {
	if !bucketName.MatchString(c.Bucket) || !safePath(c.Prefix) || c.Tokens == nil || c.Transport == nil {
		return nil, fail("secret_configuration")
	}
	return &GCSRegistry{c, client(c.Transport)}, nil
}

// get reads a private immutable metadata object; no listing or mutable local cache is used.
func (r *GCSRegistry) get(ctx context.Context, key string, out any) error {
	token, e := r.config.Tokens.AccessToken(ctx)
	if e != nil {
		return fail("secret_auth_failed")
	}
	b, _, e := request(ctx, r.http, "GET", "https://storage.googleapis.com/storage/v1/b/"+r.config.Bucket+"/o/"+escaped(r.config.Prefix+"/"+key)+"?alt=media", token, "", nil)
	if e != nil {
		return e
	}
	return strict(b, out)
}

// create reserves one durable metadata fact; unknown acknowledgement is never retried as mutation.
func (r *GCSRegistry) create(ctx context.Context, key string, v any) error {
	raw, e := json.Marshal(v)
	if e != nil || len(raw) > 16384 {
		return fail("secret_configuration")
	}
	token, e := r.config.Tokens.AccessToken(ctx)
	if e != nil {
		return fail("secret_auth_failed")
	}
	endpoint := "https://storage.googleapis.com/upload/storage/v1/b/" + r.config.Bucket + "/o?uploadType=media&ifGenerationMatch=0&name=" + url.QueryEscape(r.config.Prefix+"/"+key)
	b, _, e := request(ctx, r.http, "POST", endpoint, token, "application/json", raw)
	if e != nil {
		return e
	}
	var out struct{ Name, Generation string }
	if e = strict(b, &out); e != nil {
		return e
	}
	if out.Name != r.config.Prefix+"/"+key || !positive(out.Generation) {
		return fail("secret_response_invalid")
	}
	return nil
}

// mappingKey contains only validated SQL reference identity, not Item names or credentials.
func mappingKey(ref connections.CredentialReference) string {
	return "mappings/" + ref.ReferenceID + "/" + ref.Version + ".json"
}

// Lookup refuses mismatched identity even if an object appears under the expected key.
func (r *GCSRegistry) Lookup(ctx context.Context, ref connections.CredentialReference) (Mapping, error) {
	var m Mapping
	if !validReference(ref) {
		return m, fail("secret_reference_invalid")
	}
	if e := r.get(ctx, mappingKey(ref), &m); e != nil {
		return m, e
	}
	if m.Reference != ref || m.InfisicalSecretID == "" || !positive(m.InfisicalVersion) || !hash.MatchString(m.PayloadSHA256) {
		return Mapping{}, fail("secret_mapping_invalid")
	}
	return m, nil
}

// pin publishes an immutable exact mapping or verifies the existing identical fact.
func (r *GCSRegistry) pin(ctx context.Context, m Mapping) error {
	e := r.create(ctx, mappingKey(m.Reference), m)
	if e == nil {
		return nil
	}
	old, re := r.Lookup(ctx, m.Reference)
	if re != nil {
		return fail("secret_mapping_pending")
	}
	if old != m {
		return fail("secret_mapping_conflict")
	}
	return nil
}
