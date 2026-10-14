package providersecrets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// envelope deliberately exists only within the approved stores and memory, never registry metadata.
type envelope struct {
	Schema      string                          `json:"schema"`
	Reference   connections.CredentialReference `json:"reference"`
	AccessToken string                          `json:"access_token"`
}

// decodeEnvelope validates private identity and the exact bytes of the immutable pin.
func decodeEnvelope(b []byte, ref connections.CredentialReference, digest string) (connections.Secret, error) {
	sum := sha256.Sum256(b)
	if digest != "" && hex.EncodeToString(sum[:]) != digest {
		return connections.Secret{}, fail("secret_integrity_failed")
	}
	var v envelope
	if e := strict(b, &v); e != nil {
		return connections.Secret{}, e
	}
	if v.Schema != "jayburd-provider-credential-v1" || v.Reference != ref || !opaque.MatchString(v.AccessToken) {
		return connections.Secret{}, fail("secret_binding_invalid")
	}
	return connections.NewSecret(v.AccessToken), nil
}

// Resolver composes immutable reference pins and managed Google reads without source-store fallback.
type Resolver struct {
	reader   *GCPReader
	registry MappingRegistry
	config   ResolverConfig
}

// NewResolver accepts resource identifiers only, not plaintext client credentials.
func NewResolver(r *GCPReader, m MappingRegistry, c ResolverConfig) (*Resolver, error) {
	if r == nil || m == nil || !r.resource(c.ClientIDVersion, true) || !r.resource(c.ClientSecretVersion, true) || c.ClientIDVersion == c.ClientSecretVersion {
		return nil, fail("secret_configuration")
	}
	return &Resolver{r, m, c}, nil
}

// Resolve revalidates every mapping and payload; credential material never enters diagnostics.
func (r *Resolver) Resolve(ctx context.Context, a acquisition.CredentialReference) (acquisition.ProviderCredentials, error) {
	ref := connections.CredentialReference{ReferenceID: a.ReferenceID, Version: a.Version, ItemID: a.ItemID, Environment: a.Environment}
	var out acquisition.ProviderCredentials
	if !validReference(ref) {
		return out, fail("secret_reference_invalid")
	}
	m, e := r.registry.Lookup(ctx, ref)
	if e != nil {
		return out, fail("secret_mapping_pending")
	}
	if m.Reference != ref || !hash.MatchString(m.PayloadSHA256) {
		return out, fail("secret_mapping_invalid")
	}
	b, e := r.reader.Read(ctx, m.GCPVersion)
	if e != nil {
		return out, e
	}
	token, e := decodeEnvelope(b, ref, m.PayloadSHA256)
	if e != nil {
		return out, e
	}
	id, e := r.reader.Read(ctx, r.config.ClientIDVersion)
	if e != nil {
		return out, e
	}
	secret, e := r.reader.Read(ctx, r.config.ClientSecretVersion)
	if e != nil {
		return out, e
	}
	if !opaque.Match(id) || !opaque.Match(secret) {
		return out, fail("secret_binding_invalid")
	}
	_ = token.WithValue(func(v string) error { out.AccessToken = v; return nil })
	out.ClientID = string(id)
	out.Secret = string(secret)
	return out, nil
}

// String redacts private envelopes outside deliberate store serialization.
func (envelope) String() string { return "[redacted provider envelope]" }

// GoString redacts diagnostic formatting of private envelopes.
func (v envelope) GoString() string { return v.String() }
