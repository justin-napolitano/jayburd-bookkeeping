package observer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

// GoogleTokenVerifier uses the pinned Google signature verifier, never parse-only identity.
type GoogleTokenVerifier struct{ validator *idtoken.Validator }

// NewGoogleTokenVerifier injects public-key HTTP transport without credential discovery.
func NewGoogleTokenVerifier(transport http.RoundTripper) (*GoogleTokenVerifier, error) {
	if transport == nil {
		return nil, reject()
	}
	client := &http.Client{Transport: keyTransport{transport}, Timeout: 5 * time.Second, CheckRedirect: noRedirect}
	v, e := idtoken.NewValidator(context.Background(), option.WithHTTPClient(client))
	if e != nil {
		return nil, reject()
	}
	return &GoogleTokenVerifier{v}, nil
}

// Verify accepts only service-account RS256 tokens; handlers additionally enforce caller policy.
func (v *GoogleTokenVerifier) Verify(parent context.Context, token, audience string) (VerifiedToken, error) {
	if v == nil || v.validator == nil || len(token) > 8192 || audience == "" {
		return VerifiedToken{}, reject()
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return VerifiedToken{}, reject()
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[0])
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if e != nil || json.Unmarshal(raw, &header) != nil || header.Algorithm != "RS256" || header.KeyID == "" || len(header.KeyID) > 128 {
		return VerifiedToken{}, reject()
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	p, e := v.validator.Validate(ctx, token, audience)
	if e != nil || p == nil || ctx.Err() != nil {
		return VerifiedToken{}, reject()
	}
	email, _ := p.Claims["email"].(string)
	verified, _ := p.Claims["email_verified"].(bool)
	return VerifiedToken{p.Issuer, p.Audience, p.Subject, email, verified, p.IssuedAt, p.Expires}, nil
}

// keyTransport permits only the fixed Google service-account JWKS endpoint and caps its body.
type keyTransport struct{ base http.RoundTripper }

// RoundTrip disallows endpoint substitution even if the upstream verifier changes behavior.
func (t keyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != "GET" || r.URL.String() != "https://www.googleapis.com/oauth2/v3/certs" {
		return nil, reject()
	}
	resp, e := t.base.RoundTrip(r)
	if e != nil {
		return nil, unavailable()
	}
	if resp == nil || resp.Body == nil {
		return nil, unavailable()
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(body) > 65536 || r.Context().Err() != nil {
		return nil, unavailable()
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

// noRedirect prevents credentials and verification requests from following redirects.
func noRedirect(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
