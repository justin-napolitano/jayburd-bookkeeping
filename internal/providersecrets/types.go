// Package providersecrets preserves one-way Infisical delivery and exact immutable Google version pins.
package providersecrets

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// AccessTokenSource supplies managed-identity OAuth tokens without environment or ADC discovery.
type AccessTokenSource interface {
	AccessToken(context.Context) (string, error)
}

// IdentityTokenSource supplies an audience-bound managed identity assertion for the writer only.
type IdentityTokenSource interface {
	IdentityToken(context.Context, string) (string, error)
}

// GCPConfig fixes the only permitted global Secret Manager project and injected transport.
type GCPConfig struct {
	ProjectID string
	Tokens    AccessTokenSource
	Transport http.RoundTripper
}

// GCSConfig fixes a private metadata-only registry namespace; it never contains token values.
type GCSConfig struct {
	Bucket, Prefix string
	Tokens         AccessTokenSource
	Transport      http.RoundTripper
}

// ResolverConfig selects exact immutable versions of the two static Plaid client credentials.
type ResolverConfig struct{ ClientIDVersion, ClientSecretVersion string }

// Mapping pins independently observed source and destination versions, never matching ordinals.
type Mapping struct {
	Reference         connections.CredentialReference `json:"reference"`
	InfisicalSecretID string                          `json:"infisical_secret_id"`
	InfisicalVersion  string                          `json:"infisical_version"`
	GCPVersion        string                          `json:"gcp_version"`
	PayloadSHA256     string                          `json:"payload_sha256"`
}

// MappingRegistry returns only previously verified immutable propagation pins.
type MappingRegistry interface {
	Lookup(context.Context, connections.CredentialReference) (Mapping, error)
}

// Failure contains only a fixed redacted classification, never wrapped HTTP diagnostics.
type Failure struct{ Code string }

// Error exposes the fixed classification only.
func (f *Failure) Error() string { return f.Code }

// fail drops all provider-controlled diagnostic text.
func fail(code string) error { return &Failure{Code: code} }

// code identifies our own fixed errors without matching provider strings.
func code(err error) string {
	var f *Failure
	if errors.As(err, &f) {
		return f.Code
	}
	return "secret_dependency_failed"
}

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// boundedASCII validates opaque credentials without unbounded regexp repetitions.
type boundedASCII struct{}

// MatchString permits bounded printable opaque values only.
func (boundedASCII) MatchString(s string) bool {
	if len(s) < 1 || len(s) > 4096 {
		return false
	}
	for i := range s {
		if s[i] < 33 || s[i] > 126 {
			return false
		}
	}
	return true
}

// Match applies the same bound to raw managed-store payloads.
func (b boundedASCII) Match(s []byte) bool { return b.MatchString(string(s)) }

var opaque boundedASCII
var hash = regexp.MustCompile(`^[0-9a-f]{64}$`)

// positive validates exact decimal metadata without coercion.
func positive(s string) bool {
	n, e := strconv.ParseInt(s, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == s
}

// validReference checks shape only; SQL and the trusted caller supply authorization.
func validReference(r connections.CredentialReference) bool {
	return uuid.MatchString(r.ReferenceID) && r.ReferenceID != "00000000-0000-0000-0000-000000000000" && positive(r.Version) && len(r.ItemID) <= 256 && opaque.MatchString(r.ItemID) && r.Environment == "sandbox"
}
