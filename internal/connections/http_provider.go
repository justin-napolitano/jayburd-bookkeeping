package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
	"unicode/utf8"
)

// HTTPConfig is explicitly supplied; ConsentExpiresAt is local approval, never invented provider consent.
type HTTPConfig struct {
	ClientID, ClientSecret Secret
	ConsentExpiresAt       time.Time
}

// HTTPProvider is an opt-in fixed Sandbox adapter. Construction never reads credentials or calls a network.
type HTTPProvider struct {
	config HTTPConfig
	client *http.Client
}

// String prevents formatting an adapter from traversing its credential configuration.
func (*HTTPProvider) String() string { return "[connection provider redacted]" }

// GoString prevents detailed formatting from exposing injected client credentials.
func (*HTTPProvider) GoString() string { return "[connection provider redacted]" }

// NewHTTPProvider requires explicit synthetic/provider credentials and a bounded transport.
func NewHTTPProvider(config HTTPConfig, transport http.RoundTripper) (*HTTPProvider, error) {
	if transport == nil || !secretValid(config.ClientID) || !secretValid(config.ClientSecret) || !config.ConsentExpiresAt.After(time.Now()) || config.ConsentExpiresAt.Year() > 9999 {
		return nil, failed("configuration_error")
	}
	return &HTTPProvider{config: config, client: &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// SupportsMode rejects unqualified ordinary update-session proof before creating or claiming an attempt.
func (*HTTPProvider) SupportsMode(mode string) bool {
	return mode == "new" || mode == "rotate" || mode == "remove"
}

// post performs one bounded request, no automatic retry, and strips all upstream diagnostics.
func (p *HTTPProvider) post(parent context.Context, path string, body map[string]any, out any) error {
	if p == nil || p.client == nil {
		return failed("configuration_error")
	}
	switch path {
	case "/link/token/create", "/link/token/get", "/item/public_token/exchange", "/item/get", "/accounts/get", "/item/access_token/invalidate", "/item/remove":
	default:
		return failed("invalid_request")
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	body["client_id"] = p.config.ClientID.value
	body["secret"] = p.config.ClientSecret.value
	raw, e := json.Marshal(body)
	if e != nil || len(raw) > 16384 {
		return failed("invalid_request")
	}
	req, e := http.NewRequestWithContext(ctx, "POST", "https://sandbox.plaid.com"+path, bytes.NewReader(raw))
	if e != nil {
		return failed("invalid_request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Plaid-Version", "2020-09-14")
	resp, e := p.client.Do(req)
	if e != nil {
		return failed("provider_unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return failed("provider_unavailable")
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, 262145))
	if e != nil || len(data) > 262144 || ctx.Err() != nil || !validResponseJSON(data) || json.Unmarshal(data, out) != nil {
		return failed("provider_unavailable")
	}
	return nil
}

// CreateLink requests Transactions only; update mode uses an explicitly supplied existing credential.
func (p *HTTPProvider) CreateLink(ctx context.Context, input LinkRequest) (LinkResult, error) {
	if !validID(input.ActorID) || (input.Mode != "new" && input.Mode != "reconnect") {
		return LinkResult{}, failed("invalid_request")
	}
	body := map[string]any{"client_name": "Jayburd", "language": "en", "country_codes": []string{"US"}, "user": map[string]string{"client_user_id": input.ActorID}}
	if input.Mode == "new" {
		body["products"] = []string{"transactions"}
	} else {
		if !secretValid(input.AccessToken) {
			return LinkResult{}, failed("invalid_request")
		}
		body["access_token"] = input.AccessToken.value
	}
	var response struct {
		Token      string `json:"link_token"`
		Expiration string `json:"expiration"`
	}
	if e := p.post(ctx, "/link/token/create", body, &response); e != nil {
		return LinkResult{}, e
	}
	expiry, e := time.Parse(time.RFC3339, response.Expiration)
	if e != nil || !expiry.After(time.Now()) || !secretValid(NewSecret(response.Token)) {
		return LinkResult{}, failed("provider_unavailable")
	}
	return LinkResult{NewSecret(response.Token), expiry}, nil
}

// VerifySession requires one completed server-reported new Item result matching the public token.
// Ordinary update-mode session proof is deliberately unavailable until its hosted profile is qualified.
func (p *HTTPProvider) VerifySession(ctx context.Context, link, public Secret, expectedItem string) error {
	if !secretValid(link) || !secretValid(public) || expectedItem != "" {
		return failed("provider_unavailable")
	}
	var response struct {
		Token    string `json:"link_token"`
		Sessions []struct {
			Finished *string `json:"finished_at"`
			Results  *struct {
				Added []struct {
					PublicToken string `json:"public_token"`
				} `json:"item_add_results"`
			} `json:"results"`
		} `json:"link_sessions"`
	}
	if e := p.post(ctx, "/link/token/get", map[string]any{"link_token": link.value}, &response); e != nil {
		return e
	}
	if response.Token != link.value || len(response.Sessions) == 0 || len(response.Sessions) > 25 {
		return failed("provider_unavailable")
	}
	matches := 0
	for _, session := range response.Sessions {
		if session.Finished == nil || session.Results == nil {
			continue
		}
		when, e := time.Parse(time.RFC3339, *session.Finished)
		if e != nil || when.After(time.Now()) || time.Since(when) > 6*time.Hour {
			return failed("provider_unavailable")
		}
		if len(session.Results.Added) > 25 {
			return failed("provider_unavailable")
		}
		for _, item := range session.Results.Added {
			if item.PublicToken == public.value {
				matches++
			}
		}
	}
	if matches != 1 {
		return failed("provider_unavailable")
	}
	return nil
}

// Exchange never retries a potentially consumed public token.
func (p *HTTPProvider) Exchange(ctx context.Context, public Secret) (ExchangeResult, error) {
	if !secretValid(public) {
		return ExchangeResult{}, failed("invalid_request")
	}
	var response struct {
		Token string `json:"access_token"`
		Item  string `json:"item_id"`
	}
	if e := p.post(ctx, "/item/public_token/exchange", map[string]any{"public_token": public.value}, &response); e != nil {
		return ExchangeResult{}, e
	}
	if !secretValid(NewSecret(response.Token)) || !opaqueRE.MatchString(response.Item) {
		return ExchangeResult{}, failed("provider_unavailable")
	}
	return ExchangeResult{response.Item, NewSecret(response.Token)}, nil
}

type providerItem struct {
	ID      string          `json:"item_id"`
	Consent *string         `json:"consent_expiration_time"`
	Error   json.RawMessage `json:"error"`
}

// item checks current provider status without interpreting names as ownership evidence.
func (p *HTTPProvider) item(ctx context.Context, token Secret) (providerItem, error) {
	var response struct {
		Item providerItem `json:"item"`
	}
	if !secretValid(token) {
		return response.Item, failed("invalid_request")
	}
	if e := p.post(ctx, "/item/get", map[string]any{"access_token": token.value}, &response); e != nil {
		return response.Item, e
	}
	if !opaqueRE.MatchString(response.Item.ID) || len(response.Item.Error) == 0 || string(response.Item.Error) != "null" {
		return response.Item, failed("provider_unavailable")
	}
	return response.Item, nil
}

// Manifest uses cached account identity metadata only; balance values are not returned or used.
func (p *HTTPProvider) Manifest(ctx context.Context, token Secret) (Manifest, error) {
	item, e := p.item(ctx, token)
	if e != nil {
		return Manifest{}, e
	}
	expiry := p.config.ConsentExpiresAt
	if item.Consent != nil {
		v, e := time.Parse(time.RFC3339, *item.Consent)
		if e != nil {
			return Manifest{}, failed("provider_unavailable")
		}
		if v.Before(expiry) {
			expiry = v
		}
	}
	if !expiry.After(time.Now()) {
		return Manifest{}, failed("provider_unavailable")
	}
	var response struct {
		Item struct {
			ID string `json:"item_id"`
		} `json:"item"`
		Accounts []struct {
			ID       string `json:"account_id"`
			Type     string `json:"type"`
			Balances struct {
				Currency   *string `json:"iso_currency_code"`
				Unofficial *string `json:"unofficial_currency_code"`
			} `json:"balances"`
		} `json:"accounts"`
	}
	if e = p.post(ctx, "/accounts/get", map[string]any{"access_token": token.value}, &response); e != nil {
		return Manifest{}, e
	}
	if response.Item.ID != item.ID {
		return Manifest{}, failed("provider_unavailable")
	}
	m := Manifest{ItemID: item.ID, ConsentExpiresAt: expiry.UTC().Format(time.RFC3339), Accounts: make([]Account, 0, len(response.Accounts))}
	for _, a := range response.Accounts {
		currency := a.Balances.Currency
		if a.Balances.Unofficial != nil {
			currency = nil
		}
		m.Accounts = append(m.Accounts, Account{a.ID, a.Type, currency})
	}
	if !validManifest(m) {
		return Manifest{}, failed("provider_unavailable")
	}
	return m, nil
}

// Rotate verifies the returned credential's Item before allowing reference publication.
func (p *HTTPProvider) Rotate(ctx context.Context, token Secret) (ExchangeResult, error) {
	if !secretValid(token) {
		return ExchangeResult{}, failed("invalid_request")
	}
	var response struct {
		Token string `json:"new_access_token"`
	}
	if e := p.post(ctx, "/item/access_token/invalidate", map[string]any{"access_token": token.value}, &response); e != nil {
		return ExchangeResult{}, e
	}
	next := NewSecret(response.Token)
	if !secretValid(next) {
		return ExchangeResult{}, failed("provider_unavailable")
	}
	item, e := p.item(ctx, next)
	if e != nil {
		return ExchangeResult{}, e
	}
	return ExchangeResult{item.ID, next}, nil
}

// Remove affects the whole Item and is invoked only after an explicit SQL-fenced Item claim.
func (p *HTTPProvider) Remove(ctx context.Context, token Secret) error {
	if !secretValid(token) {
		return failed("invalid_request")
	}
	var response struct {
		RequestID string `json:"request_id"`
	}
	if e := p.post(ctx, "/item/remove", map[string]any{"access_token": token.value}, &response); e != nil {
		return e
	}
	if !opaqueRE.MatchString(response.RequestID) {
		return failed("provider_unavailable")
	}
	return nil
}

// validResponseJSON rejects duplicate keys, trailing documents, invalid UTF8 and excessive nesting.
func validResponseJSON(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if !scanResponseValue(d, 0) {
		return false
	}
	_, e := d.Token()
	return e == io.EOF
}

// scanResponseValue rejects duplicate decoded keys recursively before field interpretation.
func scanResponseValue(d *json.Decoder, depth int) bool {
	if depth > 32 {
		return false
	}
	token, e := d.Token()
	if e != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			name, ok := k.(string)
			if e != nil || !ok || seen[name] {
				return false
			}
			seen[name] = true
			if !scanResponseValue(d, depth+1) {
				return false
			}
		}
		end, e := d.Token()
		return e == nil && end == json.Delim('}')
	case '[':
		for d.More() {
			if !scanResponseValue(d, depth+1) {
				return false
			}
		}
		end, e := d.Token()
		return e == nil && end == json.Delim(']')
	}
	return false
}
