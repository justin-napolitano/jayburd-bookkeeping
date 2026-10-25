package acquisition

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

// TestProductionSyncProfile preserves literal request scope and independently fixed exact money on both origins.
func TestProductionSyncProfile(t *testing.T) {
	for _, profileCase := range []struct{ name, host string }{{"sandbox", "sandbox.plaid.com"}, {"production", "production.plaid.com"}} {
		for _, money := range []struct {
			code, lexeme, canonical, direction string
			minor                              int
		}{{"USD", "9007199254740993.01", "9007199254740993.01", "outflow", 2}, {"EUR", "-1.255e2", "125.5", "inflow", 2}, {"JPY", "125", "125", "outflow", 0}, {"KWD", "1.005", "1.005", "outflow", 3}} {
			t.Run(profileCase.name+money.code, func(t *testing.T) {
				profile, _ := plaidprofile.New(profileCase.name)
				want := fixturePage("Txn-A", money.lexeme, money.code, "C2", false)
				calls := 0
				cursor := "C1"
				p, e := NewPlaidHTTPForProfile(profile, roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.URL.Scheme != "https" || r.URL.Host != profileCase.host || r.URL.Path != "/transactions/sync" || r.Header.Get("Plaid-Version") != "2020-09-14" || r.Header.Get("Content-Type") != "application/json" {
						t.Fatal("wrong fixed profile request")
					}
					got := providerRequest(t, r)
					if got == nil || *got != "C1" {
						t.Fatal("cursor/filter changed")
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(want)), Header: make(http.Header)}, nil
				}))
				if e != nil || calls != 0 {
					t.Fatal("constructor acquired dependency")
				}
				raw, e := p.Fetch(context.Background(), FetchRequest{Environment: profileCase.name, AccountID: "Account-A", Cursor: &cursor, Credentials: credentialsFixture()})
				if e != nil || calls != 1 || !bytes.Equal(raw, want) {
					t.Fatal("raw lexical bytes changed or repeated")
				}
				c := contextFixture()
				c.Currency.Code = money.code
				c.Currency.MinorUnit = money.minor
				page, e := plaid.DecodePage(raw, c.expected())
				if e != nil || len(page.Added) != 1 || page.Added[0].AmountLexeme != money.lexeme || page.Added[0].Amount != money.canonical || page.Added[0].Direction != money.direction || page.Added[0].ISOCurrencyCode != money.code {
					t.Fatal("profile changed original currency/money semantics")
				}
			})
		}
	}
}

// TestProfileContextMismatchAndLegacyOmission denies untrusted routing metadata before transport and keeps old callers working.
func TestProfileContextMismatchAndLegacyOmission(t *testing.T) {
	for _, name := range []string{"sandbox", "production"} {
		profile, _ := plaidprofile.New(name)
		calls := 0
		p, e := NewPlaidHTTPForProfile(profile, roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
		}))
		if e != nil {
			t.Fatal("fixture setup")
		}
		for _, environment := range []string{"", "Sandbox", "Production", "development", "https://production.plaid.com", " sandbox", map[string]string{"sandbox": "production", "production": "sandbox"}[name]} {
			if _, e := p.Fetch(context.Background(), FetchRequest{Environment: environment, AccountID: "Account-A", Credentials: credentialsFixture()}); e == nil || e.Error() != "invalid_provider_request" || calls != 0 {
				t.Fatal("mixed/missing environment reached dependency")
			}
		}
	}
	legacyCalls := 0
	legacy, e := NewPlaidHTTP(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		legacyCalls++
		if r.URL.Host != "sandbox.plaid.com" {
			t.Fatal("legacy default escaped Sandbox")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	}))
	if e != nil {
		t.Fatal("legacy constructor")
	}
	for _, environment := range []string{"", "sandbox"} {
		if _, e := legacy.Fetch(context.Background(), FetchRequest{Environment: environment, AccountID: "Account-A", Credentials: credentialsFixture()}); e != nil {
			t.Fatal("legacy metadata compatibility lost")
		}
	}
	if _, e := legacy.Fetch(context.Background(), FetchRequest{Environment: "production", AccountID: "Account-A", Credentials: credentialsFixture()}); e == nil || legacyCalls != 2 {
		t.Fatal("legacy wrapper became Production fallback")
	}
	if _, e := NewPlaidHTTPForProfile(plaidprofile.Profile{}, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("zero profile I/O"); return nil, nil })); e == nil {
		t.Fatal("zero profile accepted")
	}
	production, _ := plaidprofile.New("production")
	if _, e := NewPlaidHTTPForProfile(production, nil); e == nil {
		t.Fatal("nil transport accepted")
	}
	if _, e := newPlaidHTTP(production, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("legacy Production I/O"); return nil, nil }), true); e == nil {
		t.Fatal("legacy Production bypass accepted")
	}
}

// TestRunnerPropagatesTrustedEnvironment checks SQL-returned routing metadata reaches the provider without entering request JSON.
func TestRunnerPropagatesTrustedEnvironment(t *testing.T) {
	repo := &fakeRepository{context: contextFixture()}
	calls := 0
	runner := runnerFixture(t, repo, providerFunc(func(_ context.Context, f FetchRequest) ([]byte, error) {
		calls++
		if f.Environment != "sandbox" || f.AccountID != "Account-A" {
			t.Fatal("runner lost trusted execution environment")
		}
		return fixturePage("Txn-A", "0.10", "USD", "C1", false), nil
	}))
	out, e := runner.Run(context.Background(), scopeFixture(), testClaim)
	if e != nil || out.Status != "completed" || calls != 1 {
		t.Fatal("runner fixture failed")
	}
	if _, e = runner.Run(context.Background(), scopeFixture(), testClaim); e != nil || calls != 1 {
		t.Fatal("completed replay repeated provider work")
	}
}

// TestProfiledSyncFailures retains pagination/retry/body/redaction/redirect behavior without automatic network retries.
func TestProfiledSyncFailures(t *testing.T) {
	production, _ := plaidprofile.New("production")
	for _, name := range []string{"pagination", "retry_deferred", "retry_bounded", "redirect", "oversize", "cancel"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p, e := NewPlaidHTTPForProfile(production, roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				status, body := 400, `{"error_code":"synthetic-rejection","error_message":"private-diagnostic"}`
				header := make(http.Header)
				switch name {
				case "pagination":
					body = `{"error_code":"TRANSACTIONS_SYNC_MUTATION_DURING_PAGINATION","error_message":"private-diagnostic"}`
				case "retry_deferred":
					status = 429
					header.Set("Retry-After", "86401")
				case "retry_bounded":
					status = 503
					header.Set("Retry-After", "10")
				case "redirect":
					status = 307
					header.Set("Location", "https://foreign.example/")
				case "oversize":
					status = 200
					body = strings.Repeat("x", plaid.MaxBytes+1)
				case "cancel":
					status = 200
					cancel()
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
			}))
			if e != nil {
				t.Fatal("fixture constructor")
			}
			_, e = p.Fetch(ctx, FetchRequest{Environment: "production", AccountID: "Account-A", Credentials: credentialsFixture()})
			if e == nil || calls != 1 || strings.Contains(e.Error(), "private") {
				t.Fatal("profile failed open/retried/exposed diagnostic")
			}
			want := map[string]string{"pagination": "pagination_mutation", "retry_deferred": "retry_deferred", "retry_bounded": "provider_transient", "redirect": "provider_rejected", "oversize": "limit_exceeded", "cancel": "provider_transient"}[name]
			if e.Error() != want {
				t.Fatal("provider classification changed")
			}
			if name == "retry_bounded" {
				failure, ok := e.(*ProviderFailure)
				if !ok || failure.RetryAfter != 10*time.Second {
					t.Fatal("retry delay changed")
				}
			}
		})
	}
}

// TestEnvironmentIsInternalMetadata verifies routing pins never become an undocumented Plaid financial field.
func TestEnvironmentIsInternalMetadata(t *testing.T) {
	production, _ := plaidprofile.New("production")
	p, _ := NewPlaidHTTPForProfile(production, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var fields map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&fields) != nil || len(fields) != 6 || fields["environment"] != nil {
			t.Fatal("internal routing leaked into wire schema")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	}))
	if _, e := p.Fetch(context.Background(), FetchRequest{Environment: "production", AccountID: "Account-A", Credentials: credentialsFixture()}); e != nil {
		t.Fatal("profile fixture request failed")
	}
}

// TestSourceSQLContextRemainsSandboxOnly prevents a transport component from silently broadening durable authority.
func TestSourceSQLContextRemainsSandboxOnly(t *testing.T) {
	c := contextFixture()
	a := attempt{SetID: c.SetID, Fence: c.Fence}
	if !validContext(c, scopeFixture(), a) {
		t.Fatal("positive current SQL context fixture invalid")
	}
	c.Environment = "production"
	if validContext(c, scopeFixture(), a) {
		t.Fatal("transport source change broadened SQL authority")
	}
}
