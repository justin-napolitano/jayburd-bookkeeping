package acquisition

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// PlaidHTTP fixes the Sandbox endpoint and version without credential or environment discovery.
type PlaidHTTP struct{ client *http.Client }

// NewPlaidHTTP accepts an injected network implementation; construction performs no request.
func NewPlaidHTTP(transport http.RoundTripper) (*PlaidHTTP, error) {
	if transport == nil {
		return nil, failure("configuration_error", false)
	}
	return &PlaidHTTP{&http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// credentialText rejects missing, oversized or header-injection credential values without logging them.
func credentialText(v string) bool {
	return len(v) > 0 && len(v) <= 8192 && !strings.ContainsAny(v, "\r\n\t ")
}

// Fetch sends one account-filtered request and returns exact bounded bytes, never SDK floats.
func (p *PlaidHTTP) Fetch(parent context.Context, f FetchRequest) ([]byte, error) {
	if !opaque.MatchString(f.AccountID) || !validCursor(f.Cursor) || !credentialText(f.Credentials.AccessToken) || !credentialText(f.Credentials.ClientID) || !credentialText(f.Credentials.Secret) {
		return nil, failure("invalid_provider_request", false)
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, &ProviderFailure{Code: "provider_transient"}
	}
	raw, _ := json.Marshal(map[string]any{"client_id": f.Credentials.ClientID, "secret": f.Credentials.Secret, "access_token": f.Credentials.AccessToken, "cursor": f.Cursor, "count": 500, "options": map[string]any{"account_id": f.AccountID, "include_original_description": true}})
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, "https://sandbox.plaid.com/transactions/sync", bytes.NewReader(raw))
	if e != nil {
		return nil, failure("invalid_provider_request", false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Plaid-Version", plaid.APIVersion)
	resp, e := p.client.Do(req)
	if e != nil {
		return nil, &ProviderFailure{Code: "provider_transient"}
	}
	defer resp.Body.Close()
	limit := plaid.MaxBytes
	if resp.StatusCode != 200 {
		limit = 8192
	}
	body, e := io.ReadAll(io.LimitReader(resp.Body, int64(limit+1)))
	if ctx.Err() != nil || e != nil {
		return nil, &ProviderFailure{Code: "provider_transient"}
	}
	if len(body) > limit {
		return nil, failure("limit_exceeded", false)
	}
	if resp.StatusCode == 200 {
		return body, nil
	}
	var problem struct {
		Code string `json:"error_code"`
	}
	if json.Unmarshal(body, &problem) == nil && problem.Code == "TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION" {
		return nil, &ProviderFailure{Code: "pagination_mutation"}
	}
	if resp.StatusCode == 429 || resp.StatusCode >= 500 || resp.StatusCode == 408 {
		delay, deferred := retryAfter(resp.Header.Get("Retry-After"), time.Now())
		if deferred {
			return nil, &ProviderFailure{Code: "retry_deferred"}
		}
		return nil, &ProviderFailure{Code: "provider_transient", RetryAfter: delay}
	}
	return nil, &ProviderFailure{Code: "provider_rejected"}
}

// retryAfter preserves a bounded provider delay; an excessive instruction never becomes an earlier retry.
func retryAfter(value string, now time.Time) (time.Duration, bool) {
	delay := 5 * time.Second
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds > 0 {
		if seconds > 86400 {
			return 0, true
		}
		delay = time.Duration(seconds) * time.Second
	} else if date, err := http.ParseTime(value); err == nil && date.After(now) {
		delay = date.Sub(now)
	}
	if delay > 24*time.Hour {
		return 0, true
	}
	return delay, false
}
