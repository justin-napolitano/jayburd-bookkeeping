// Package sandboxwebhook requests one supported test event inside an authorized Sandbox acquisition.
package sandboxwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

var ErrProbe = errors.New("sandbox_webhook_probe_failed")

// Probe wraps only the normal runner's authorized provider call; it owns no credential lookup.
type Probe struct {
	mu                   sync.Mutex
	provider             acquisition.Provider
	client               *http.Client
	expires              time.Time
	now                  func() time.Time
	attempted, requested bool
	failed               bool
}

// New is inert and requires a short explicit deadline before accepting a network capability.
func New(p acquisition.Provider, transport http.RoundTripper, expires time.Time) (*Probe, error) {
	now := time.Now()
	if p == nil || transport == nil || !expires.After(now) || expires.After(now.Add(10*time.Minute)) {
		return nil, ErrProbe
	}
	return &Probe{provider: p, client: &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, expires: expires, now: time.Now}, nil
}

// Requested reports only the confirmed test-endpoint acknowledgement, never receiver acceptance.
func (p *Probe) Requested() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.requested
}

// token permits bounded visible credential/account bytes without normalizing or exposing them.
func token(s string) bool {
	if len(s) == 0 || len(s) > 4096 {
		return false
	}
	for i := range s {
		if s[i] < '!' || s[i] > '~' {
			return false
		}
	}
	return true
}

// Fetch fires at most once, then delegates unchanged filtered acquisition; failed/expired calls stay closed.
func (p *Probe) Fetch(parent context.Context, r acquisition.FetchRequest) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if parent.Err() != nil || !p.expires.After(p.now()) || p.failed || r.Environment != "sandbox" || !token(r.AccountID) || len(r.AccountID) > 256 || !token(r.Credentials.AccessToken) || !token(r.Credentials.ClientID) || !token(r.Credentials.Secret) {
		return nil, ErrProbe
	}
	ctx, cancel := context.WithDeadline(parent, p.expires)
	defer cancel()
	if !p.attempted {
		p.attempted = true
		if p.fire(ctx, r.Credentials) != nil {
			p.failed = true
			return nil, ErrProbe
		}
		p.requested = true
	}
	if ctx.Err() != nil || !p.expires.After(p.now()) {
		return nil, ErrProbe
	}
	return p.provider.Fetch(ctx, r)
}

// fire has one fixed Sandbox destination/code and discards all provider diagnostics and identifiers.
func (p *Probe) fire(parent context.Context, c acquisition.ProviderCredentials) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]string{"client_id": c.ClientID, "secret": c.Secret, "access_token": c.AccessToken, "webhook_type": "TRANSACTIONS", "webhook_code": "SYNC_UPDATES_AVAILABLE"})
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sandbox.plaid.com/sandbox/item/fire_webhook", bytes.NewReader(body))
	if err != nil {
		return ErrProbe
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Plaid-Version", plaidprofile.APIVersion)
	response, err := p.client.Do(r)
	if err != nil {
		return ErrProbe
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ErrProbe
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil || len(raw) > 4096 || ctx.Err() != nil {
		return ErrProbe
	}
	// Token-level parsing rejects duplicate/unknown fields before decoding the two fixed values.
	d := json.NewDecoder(bytes.NewReader(raw))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return ErrProbe
	}
	seen := map[string]bool{}
	var fired bool
	for d.More() {
		t, err = d.Token()
		k, ok := t.(string)
		if err != nil || !ok || seen[k] {
			return ErrProbe
		}
		seen[k] = true
		switch k {
		case "webhook_fired":
			if d.Decode(&fired) != nil {
				return ErrProbe
			}
		case "request_id":
			var id string
			if d.Decode(&id) != nil || !token(id) || len(id) > 256 {
				return ErrProbe
			}
		default:
			return ErrProbe
		}
	}
	if t, err = d.Token(); err != nil || t != json.Delim('}') || len(seen) != 2 || !fired {
		return ErrProbe
	}
	if _, err = d.Token(); err != io.EOF {
		return ErrProbe
	}
	return nil
}
