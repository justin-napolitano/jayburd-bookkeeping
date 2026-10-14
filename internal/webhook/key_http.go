package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// Credentials exist only in memory and are never serialized except in the deliberate key request.
type Credentials struct{ ClientID, Secret string }

// String redacts whole-credential formatting.
func (Credentials) String() string { return "[redacted webhook credentials]" }

// GoString redacts Go-syntax whole-credential formatting.
func (c Credentials) GoString() string { return c.String() }

// MarshalJSON prevents incidental credential serialization.
func (Credentials) MarshalJSON() ([]byte, error) { return nil, errUnavailable }

// CredentialSource resolves operator-owned client credentials without request-derived paths.
type CredentialSource interface {
	Resolve(context.Context) (Credentials, error)
}

// HTTPKeys fixes the environment, endpoint, HTTP version profile and response bounds.
type HTTPKeys struct {
	client      *http.Client
	credentials CredentialSource
}

// NewHTTPKeys constructs an inert client; no credentials are resolved until a bounded cache miss.
func NewHTTPKeys(t http.RoundTripper, c CredentialSource) (*HTTPKeys, error) {
	if t == nil || c == nil {
		return nil, errUnavailable
	}
	return &HTTPKeys{&http.Client{Transport: t, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, c}, nil
}

// Get fetches only the requested public key from fixed Sandbox, without redirect or retry.
func (p *HTTPKeys) Get(parent context.Context, id string) (JWK, error) {
	if !keyID.MatchString(id) {
		return JWK{}, errInvalid
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return JWK{}, errUnavailable
	}
	c, e := p.credentials.Resolve(ctx)
	if e != nil || ctx.Err() != nil || len(c.ClientID) == 0 || len(c.ClientID) > 4096 || len(c.Secret) == 0 || len(c.Secret) > 4096 || strings.ContainsAny(c.ClientID+c.Secret, "\r\n\t ") {
		return JWK{}, errUnavailable
	}
	raw, _ := json.Marshal(map[string]string{"client_id": c.ClientID, "secret": c.Secret, "key_id": id})
	req, e := http.NewRequestWithContext(ctx, "POST", "https://sandbox.plaid.com/webhook_verification_key/get", bytes.NewReader(raw))
	if e != nil {
		return JWK{}, errUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Plaid-Version", "2020-09-14")
	response, e := p.client.Do(req)
	if e != nil {
		return JWK{}, errUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return JWK{}, errUnavailable
	}
	body, e := io.ReadAll(io.LimitReader(response.Body, (16<<10)+1))
	if e != nil || len(body) > 16<<10 || ctx.Err() != nil {
		return JWK{}, errUnavailable
	}
	var result struct {
		Key       json.RawMessage `json:"key"`
		RequestID string          `json:"request_id"`
	}
	if object(body, []string{"key", "request_id"}, nil, &result) != nil || !opaque.MatchString(result.RequestID) {
		return JWK{}, errUnavailable
	}
	var k JWK
	if object(result.Key, []string{"alg", "crv", "kid", "kty", "use", "x", "y", "created_at", "expired_at"}, nil, &k) != nil || k.ID != id {
		return JWK{}, errUnavailable
	}
	return k, nil
}
