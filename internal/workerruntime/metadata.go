package workerruntime

import (
	"context"
	"encoding/json"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// MetadataTokens obtains short-lived service identity tokens from one fixed metadata endpoint.
type MetadataTokens struct{ client *http.Client }

// NewMetadataTokens installs a bounded injected transport without credential or network discovery.
func NewMetadataTokens(t http.RoundTripper) (*MetadataTokens, error) {
	if t == nil || clouddb.ForbiddenEnvironment(func(k string) bool { _, ok := os.LookupEnv(k); return ok }) {
		return nil, ErrConfiguration
	}
	return &MetadataTokens{&http.Client{Transport: t, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// AccessToken requests metadata explicitly; neither ADC files nor inherited proxy settings choose a destination.
func (m *MetadataTokens) AccessToken(parent context.Context) (string, error) {
	if m == nil || m.client == nil {
		return "", ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return "", ErrUnavailable
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	if e != nil {
		return "", ErrUnavailable
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, e := m.client.Do(req)
	if e != nil {
		return "", ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Metadata-Flavor") != "Google" {
		return "", ErrUnavailable
	}
	body, e := io.ReadAll(io.LimitReader(resp.Body, 16385))
	if e != nil || len(body) > 16384 || ctx.Err() != nil {
		return "", ErrUnavailable
	}
	var v struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if json.Unmarshal(body, &v) != nil || v.TokenType != "Bearer" || v.ExpiresIn < 1 || v.ExpiresIn > 3600 || len(v.AccessToken) < 1 || len(v.AccessToken) > 8192 || strings.ContainsAny(v.AccessToken, " \t\r\n") {
		return "", ErrUnavailable
	}
	return v.AccessToken, nil
}

// DirectTransport removes inherited HTTP proxies and gives each worker explicit bounded connection ownership.
func DirectTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.MaxIdleConns = 8
	t.MaxIdleConnsPerHost = 2
	t.MaxConnsPerHost = 2
	t.ResponseHeaderTimeout = 10 * time.Second
	return t
}

// IdentityToken permits reviewed HTTPS audiences or canonical Infisical machine identity UUIDs.
func (m *MetadataTokens) IdentityToken(parent context.Context, audience string) (string, error) {
	u, e := url.Parse(audience)
	validUUID := uuid.MatchString(audience) && audience != "00000000-0000-0000-0000-000000000000"
	validURL := e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && len(audience) <= 2048
	if m == nil || m.client == nil || (!validUUID && !validURL) || strings.ContainsAny(audience, " \t\r\n") {
		return "", ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return "", ErrUnavailable
	}
	q := url.Values{"audience": {audience}, "format": {"full"}}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?"+q.Encode(), nil)
	if e != nil {
		return "", ErrUnavailable
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, e := m.client.Do(req)
	if e != nil {
		return "", ErrUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.Header.Get("Metadata-Flavor") != "Google" {
		return "", ErrUnavailable
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 8193))
	if e != nil || len(b) < 1 || len(b) > 8192 || ctx.Err() != nil || strings.ContainsAny(string(b), " \t\r\n") || len(strings.Split(string(b), ".")) != 3 {
		return "", ErrUnavailable
	}
	return string(b), nil
}
