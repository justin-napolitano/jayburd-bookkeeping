package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

	plaid "github.com/plaid/plaid-go/v49"
	"github.com/plaid/plaid-go/v49/accounts"
	"github.com/plaid/plaid-go/v49/core"
	"github.com/plaid/plaid-go/v49/item"
	"github.com/plaid/plaid-go/v49/link"
	"github.com/plaid/plaid-go/v49/option"
)

// HTTPConfig is explicitly supplied; ConsentExpiresAt is local approval, never invented provider consent.
type HTTPConfig struct {
	ClientID, ClientSecret Secret
	ConsentExpiresAt       time.Time
}

// HTTPProvider is an opt-in fixed Sandbox adapter. Construction never reads credentials or calls a network.
type HTTPProvider struct {
	config   HTTPConfig
	link     *link.Client
	item     *item.Client
	accounts *accounts.Client
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
	clientID, version := config.ClientID.value, "2020-09-14"
	options := core.NewRequestOptions(option.WithBaseURL("https://sandbox.plaid.com"),
		option.WithClientID(&clientID), option.WithSecret(config.ClientSecret.value), option.WithVersion(&version), option.WithoutRetries(),
		option.WithHTTPClient(&http.Client{Transport: boundedPlaidTransport{transport}, Timeout: 3 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
	return &HTTPProvider{config: config, link: link.NewClient(options), item: item.NewClient(options), accounts: accounts.NewClient(options)}, nil
}

// SupportsMode rejects unqualified ordinary update-session proof before creating or claiming an attempt.
func (*HTTPProvider) SupportsMode(mode string) bool {
	return mode == "new" || mode == "rotate" || mode == "remove"
}

// boundedPlaidTransport validates responses before generated decoding; no upstream error escapes.
type boundedPlaidTransport struct{ base http.RoundTripper }

func (t boundedPlaidTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "https" || r.URL.Host != "sandbox.plaid.com" || r.Method != "POST" {
		return nil, failed("invalid_request")
	}
	switch r.URL.Path {
	case "/link/token/create", "/link/token/get", "/item/public_token/exchange", "/item/get", "/accounts/get", "/item/access_token/invalidate", "/item/remove":
	default:
		return nil, failed("invalid_request")
	}
	resp, e := t.base.RoundTrip(r)
	if e != nil {
		return nil, failed("provider_unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, failed("provider_unavailable")
	}
	data, e := io.ReadAll(io.LimitReader(resp.Body, 262145))
	if e != nil || len(data) > 262144 || r.Context().Err() != nil || !validResponseJSON(data) {
		return nil, failed("provider_unavailable")
	}
	if r.URL.Path == "/item/get" {
		var v struct {
			Item map[string]json.RawMessage `json:"item"`
		}
		if json.Unmarshal(data, &v) != nil || string(v.Item["error"]) != "null" {
			return nil, failed("provider_unavailable")
		}
	}
	if r.URL.Path == "/accounts/get" {
		// Discard all balance amounts and unused account data before SDK float models see them.
		var v struct {
			Item struct {
				ID string `json:"item_id"`
			} `json:"item"`
			Accounts []struct {
				ID       string  `json:"account_id"`
				Type     string  `json:"type"`
				Subtype  *string `json:"subtype"`
				Balances struct {
					Currency   *string `json:"iso_currency_code"`
					Unofficial *string `json:"unofficial_currency_code"`
				} `json:"balances"`
			} `json:"accounts"`
		}
		if json.Unmarshal(data, &v) != nil {
			return nil, failed("provider_unavailable")
		}
		data, e = json.Marshal(v)
		if e != nil {
			return nil, failed("provider_unavailable")
		}
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.ContentLength = int64(len(data))
	return resp, nil
}

// CreateLink requests Transactions only; update mode uses an explicitly supplied existing credential.
func (p *HTTPProvider) CreateLink(ctx context.Context, input LinkRequest) (LinkResult, error) {
	if !validID(input.ActorID) || (input.Mode != "new" && input.Mode != "reconnect") {
		return LinkResult{}, failed("invalid_request")
	}
	request := &plaid.LinkTokenCreateRequest{ClientName: "Jayburd", Language: "en", CountryCodes: []plaid.CountryCode{plaid.CountryCodeUs}, User: &plaid.LinkTokenCreateRequestUser{ClientUserID: input.ActorID}}
	if input.Mode == "new" {
		lifetime := 900
		request.Products = []plaid.Products{plaid.ProductsTransactions}
		request.HostedLink = &plaid.LinkTokenCreateHostedLink{URLLifetimeSeconds: &lifetime}
	} else {
		if !secretValid(input.AccessToken) {
			return LinkResult{}, failed("invalid_request")
		}
		request.AccessToken = &input.AccessToken.value
	}
	response, e := p.link.CreateToken(ctx, request)
	if e != nil || response == nil {
		return LinkResult{}, failed("provider_unavailable")
	}

	expiry := response.Expiration
	if !expiry.After(time.Now()) || !secretValid(NewSecret(response.LinkToken)) {
		return LinkResult{}, failed("provider_unavailable")
	}
	if response.HostedLinkURL == nil || !validHostedURL(*response.HostedLinkURL) {
		return LinkResult{}, failed("provider_unavailable")
	}
	return LinkResult{Token: NewSecret(response.LinkToken), ExpiresAt: expiry, HostedURL: NewSecret(*response.HostedLinkURL)}, nil
}

// validHostedURL restricts navigation to the reviewed Plaid-hosted opaque path only.
func validHostedURL(v string) bool {
	return len(v) <= 2048 && regexp.MustCompile(`^https://secure\.plaid\.com/hl/[A-Za-z0-9_-]+$`).MatchString(v)
}

// CompletedPublicToken reads a single completed association without consuming an exchange claim.
// Empty results mean pending; ambiguity and malformed results never choose a first token.
func (p *HTTPProvider) CompletedPublicToken(ctx context.Context, link Secret) (Secret, error) {
	if !secretValid(link) {
		return Secret{}, failed("invalid_request")
	}
	response, e := p.link.GetToken(ctx, &plaid.LinkTokenGetRequest{LinkToken: link.value})
	if e != nil || response == nil || response.LinkToken != link.value || len(response.LinkSessions) > 25 {
		return Secret{}, failed("provider_unavailable")
	}

	var token Secret
	count := 0
	for _, session := range response.LinkSessions {
		if session == nil {
			return Secret{}, failed("provider_unavailable")
		}
		if session.FinishedAt == nil {
			continue
		}
		when := *session.FinishedAt
		if when.After(time.Now()) || time.Since(when) > 6*time.Hour {
			return Secret{}, failed("provider_unavailable")
		}
		if session.Results == nil {
			continue
		}
		if len(session.Results.ItemAddResults) > 25 {
			return Secret{}, failed("provider_unavailable")
		}
		for _, item := range session.Results.ItemAddResults {
			if item == nil {
				return Secret{}, failed("provider_unavailable")
			}
			token = NewSecret(item.PublicToken)
			if !secretValid(token) {
				return Secret{}, failed("provider_unavailable")
			}
			count++
		}
	}
	if count == 0 {
		return Secret{}, failed("link_session_pending")
	}
	if count != 1 {
		return Secret{}, failed("provider_unavailable")
	}
	return token, nil
}

// VerifySession requires exactly one completed provider result matching the claimed token.
func (p *HTTPProvider) VerifySession(ctx context.Context, link, public Secret, expectedItem string) error {
	if expectedItem != "" || !secretValid(public) {
		return failed("provider_unavailable")
	}
	token, e := p.CompletedPublicToken(ctx, link)
	if e != nil || token.value != public.value {
		return failed("provider_unavailable")
	}
	return nil
}

// Exchange never retries a potentially consumed public token.
func (p *HTTPProvider) Exchange(ctx context.Context, public Secret) (ExchangeResult, error) {
	if !secretValid(public) {
		return ExchangeResult{}, failed("invalid_request")
	}
	response, e := p.item.ExchangePublicToken(ctx, &plaid.ItemPublicTokenExchangeRequest{PublicToken: public.value})
	if e != nil || response == nil {
		return ExchangeResult{}, failed("provider_unavailable")
	}

	if !secretValid(NewSecret(response.AccessToken)) || !opaqueRE.MatchString(response.ItemID) {
		return ExchangeResult{}, failed("provider_unavailable")
	}
	return ExchangeResult{response.ItemID, NewSecret(response.AccessToken)}, nil
}

// item checks current provider status without interpreting names as ownership evidence.
func (p *HTTPProvider) itemStatus(ctx context.Context, token Secret) (*plaid.ItemWithConsentFields, error) {
	if !secretValid(token) {
		return nil, failed("invalid_request")
	}
	response, e := p.item.Get(ctx, &plaid.ItemGetRequest{AccessToken: token.value})
	if e != nil || response == nil || response.Item == nil || !opaqueRE.MatchString(response.Item.ItemID) || response.Item.Error != nil {
		return nil, failed("provider_unavailable")
	}
	return response.Item, nil
}

// Manifest uses cached account identity metadata only; balance values are not returned or used.
func (p *HTTPProvider) Manifest(ctx context.Context, token Secret) (Manifest, error) {
	item, e := p.itemStatus(ctx, token)
	if e != nil {
		return Manifest{}, e
	}
	expiry := p.config.ConsentExpiresAt
	if item.ConsentExpirationTime != nil {
		v := *item.ConsentExpirationTime
		if v.Before(expiry) {
			expiry = v
		}
	}
	if !expiry.After(time.Now()) {
		return Manifest{}, failed("provider_unavailable")
	}
	response, e := p.accounts.Get(ctx, &plaid.AccountsGetRequest{AccessToken: token.value})
	if e != nil || response == nil || response.Item == nil || response.Item.ItemID != item.ItemID {
		return Manifest{}, failed("provider_unavailable")
	}

	m := Manifest{ItemID: item.ItemID, ConsentExpiresAt: expiry.UTC().Format(time.RFC3339), Accounts: make([]Account, 0, len(response.Accounts))}
	for _, a := range response.Accounts {
		if a == nil || a.Balances == nil {
			return Manifest{}, failed("provider_unavailable")
		}
		currency := a.Balances.IsoCurrencyCode
		if a.Balances.UnofficialCurrencyCode != nil {
			currency = nil
		}
		var subtype *string
		if a.Subtype != nil {
			value := string(*a.Subtype)
			subtype = &value
		}
		m.Accounts = append(m.Accounts, Account{AccountID: a.AccountID, Type: string(a.Type), Subtype: subtype, Currency: currency})
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
	response, e := p.item.InvalidateAccessToken(ctx, &plaid.ItemAccessTokenInvalidateRequest{AccessToken: token.value})
	if e != nil || response == nil {
		return ExchangeResult{}, failed("provider_unavailable")
	}
	next := NewSecret(response.NewAccessToken)
	if !secretValid(next) {
		return ExchangeResult{}, failed("provider_unavailable")
	}
	item, e := p.itemStatus(ctx, next)
	if e != nil {
		return ExchangeResult{}, e
	}
	return ExchangeResult{item.ItemID, next}, nil
}

// Remove affects the whole Item and is invoked only after an explicit SQL-fenced Item claim.
func (p *HTTPProvider) Remove(ctx context.Context, token Secret) error {
	if !secretValid(token) {
		return failed("invalid_request")
	}
	response, e := p.item.Remove(ctx, &plaid.ItemRemoveRequest{AccessToken: token.value})
	if e != nil || response == nil {
		return failed("provider_unavailable")
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
