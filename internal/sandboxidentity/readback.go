package sandboxidentity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxverification"
)

var ErrReadback = errors.New("sandbox_identity_readback_failed")

// MaxCredentialBytes bounds one private injection envelope without any credential discovery.
const MaxCredentialBytes = 16 << 10

// ValidReadbackExpiry requires a fresh finite operator window; it never renews application or Item consent.
func ValidReadbackExpiry(expiry, now time.Time) bool {
	return expiry.After(now) && !expiry.After(now.Add(time.Hour))
}

// privateText accepts bounded visible ASCII only; callers must never format the returned credential values.
func privateText(fields map[string]json.RawMessage, key string) (string, bool) {
	var value string
	if json.Unmarshal(fields[key], &value) != nil || len(value) == 0 || len(value) > 4096 {
		return "", false
	}
	for _, c := range value {
		if c < 33 || c > 126 {
			return "", false
		}
	}
	return value, true
}

// Readback performs one fixed Sandbox request and returns only the existing identity projection.
// It neither exposes raw provider bytes nor discovers credentials, persists cursors or retries.
func Readback(parent context.Context, fixture, credential []byte, account string, expiry time.Time, transport http.RoundTripper) ([]byte, int, error) {
	if parent == nil || transport == nil || !ValidReadbackExpiry(expiry, time.Now()) || !ValidIdentity(account) || ValidateReadbackFixture(fixture) != nil || len(credential) == 0 || len(credential) > MaxCredentialBytes {
		return nil, 0, ErrReadback
	}
	fields, err := sandboxverification.StrictObject(credential)
	if err != nil || len(fields) != 3 {
		return nil, 0, ErrReadback
	}
	id, idOK := privateText(fields, "client_id")
	secret, secretOK := privateText(fields, "secret")
	token, tokenOK := privateText(fields, "access_token")
	if !idOK || !secretOK || !tokenOK || !strings.HasPrefix(token, "access-sandbox-") || len(token) <= len("access-sandbox-") {
		return nil, 0, ErrReadback
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	ctx, end := context.WithDeadline(ctx, expiry)
	defer end()
	if ctx.Err() != nil {
		return nil, 0, ErrReadback
	}
	// Only the exact account's initial stream is requested; no cursor, refresh or history expansion.
	body, err := json.Marshal(struct {
		ClientID    string `json:"client_id"`
		Secret      string `json:"secret"`
		AccessToken string `json:"access_token"`
		Count       int    `json:"count"`
		Options     struct {
			AccountID string `json:"account_id"`
		} `json:"options"`
	}{ClientID: id, Secret: secret, AccessToken: token, Count: 500, Options: struct {
		AccountID string `json:"account_id"`
	}{account}})
	if err != nil {
		return nil, 0, ErrReadback
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sandbox.plaid.com/transactions/sync", bytes.NewReader(body))
	if err != nil {
		return nil, 0, ErrReadback
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Plaid-Version", "2020-09-14")
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Timeout: 15 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, 0, ErrReadback
	}
	defer response.Body.Close()
	contentType, _, parseErr := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if response.StatusCode != http.StatusOK || parseErr != nil || contentType != "application/json" || ctx.Err() != nil {
		return nil, 0, ErrReadback
	}
	maximum := sandboxverification.MaxInputBytes - len(fixture)
	page, err := io.ReadAll(io.LimitReader(response.Body, int64(maximum)+1))
	if err != nil || len(page) > maximum || ctx.Err() != nil || !expiry.After(time.Now()) {
		return nil, 0, ErrReadback
	}
	result, count, err := Project(fixture, page, account)
	if err != nil || ctx.Err() != nil {
		return nil, 0, ErrReadback
	}
	return result, count, nil
}
