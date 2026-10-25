package connections

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

// TestProductionConnectionProfile exercises the real pinned SDK on both explicitly selected closed origins.
func TestProductionConnectionProfile(t *testing.T) {
	for _, tc := range []struct{ name, host string }{{"sandbox", "sandbox.plaid.com"}, {"production", "production.plaid.com"}} {
		t.Run(tc.name, func(t *testing.T) {
			profile, _ := plaidprofile.New(tc.name)
			calls := 0
			p, e := NewHTTPProviderForProfile(config(), profile, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.Scheme != "https" || r.URL.Host != tc.host || r.Header.Get("Plaid-Version") != "2020-09-14" || r.Header.Get("PLAID-CLIENT-ID") != "synthetic-client" || r.Header.Get("PLAID-SECRET") != "synthetic-secret" {
					t.Fatal("profile/SDK wire mismatch")
				}
				switch r.URL.Path {
				case "/link/token/create":
					var body struct {
						Products []string `json:"products"`
						Country  []string `json:"country_codes"`
						Language string   `json:"language"`
						User     struct {
							ID string `json:"client_user_id"`
						} `json:"user"`
						Hosted struct {
							Seconds int `json:"url_lifetime_seconds"`
						} `json:"hosted_link"`
						Webhook *string `json:"webhook"`
					}
					if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Products) != 1 || body.Products[0] != "transactions" || len(body.Country) != 1 || body.Country[0] != "US" || body.Language != "en" || body.User.ID != actorID || body.Hosted.Seconds != 900 || body.Webhook != nil {
						t.Fatal("profile broadened Link product/notification scope")
					}
					return response(200, `{"link_token":"synthetic-link","expiration":"2099-01-01T00:00:00Z","hosted_link_url":"https://secure.plaid.com/hl/synthetic"}`), nil
				case "/link/token/get":
					return response(200, `{"link_token":"synthetic-link","link_sessions":[{"finished_at":"`+time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)+`","results":{"item_add_results":[{"public_token":"synthetic-public"}]}}]}`), nil
				case "/item/public_token/exchange":
					return response(200, `{"access_token":"synthetic-access","item_id":"synthetic-item"}`), nil
				case "/item/get":
					return response(200, `{"item":{"item_id":"synthetic-item","error":null,"consent_expiration_time":null}}`), nil
				case "/accounts/get":
					return response(200, `{"item":{"item_id":"synthetic-item"},"accounts":[{"account_id":"one","type":"depository","subtype":"checking","balances":{"iso_currency_code":"USD","unofficial_currency_code":null,"current":1e1000,"available":9007199254740993.01}}]}`), nil
				case "/item/access_token/invalidate":
					return response(200, `{"new_access_token":"synthetic-new-access"}`), nil
				case "/item/remove":
					return response(200, `{"request_id":"synthetic-request"}`), nil
				}
				t.Fatal("unreviewed provider endpoint")
				return nil, nil
			}))
			if e != nil || calls != 0 || p.SupportsMode("reconnect") {
				t.Fatal("constructor performed I/O or enabled repair")
			}
			l, e := p.CreateLink(context.Background(), LinkRequest{ActorID: actorID, Mode: "new"})
			if e != nil {
				t.Fatal("Link construction failed")
			}
			if p.VerifySession(context.Background(), l.Token, NewSecret("synthetic-public"), "") != nil {
				t.Fatal("owned new-session readback failed")
			}
			credential, e := p.Exchange(context.Background(), NewSecret("synthetic-public"))
			if e != nil {
				t.Fatal("exchange failed")
			}
			manifest, e := p.Manifest(context.Background(), credential.AccessToken)
			if e != nil || manifest.ItemID != "synthetic-item" || len(manifest.Accounts) != 1 || manifest.Accounts[0].AccountID != "one" {
				t.Fatal("manifest/float preprojection changed")
			}
			rotated, e := p.Rotate(context.Background(), credential.AccessToken)
			if e != nil || rotated.ItemID != "synthetic-item" {
				t.Fatal("credential lifecycle changed")
			}
			if p.Remove(context.Background(), rotated.AccessToken) != nil || calls != 8 {
				t.Fatal("request retried or escaped lifecycle scope")
			}
			if strings.Contains(fmt.Sprintf("%#v", p), "synthetic-secret") {
				t.Fatal("profile adapter exposed configuration")
			}
		})
	}
}

// TestProductionConnectionConfigurationDenials proves source construction cannot reuse Sandbox notification/repair scope.
func TestProductionConnectionConfigurationDenials(t *testing.T) {
	production, _ := plaidprofile.New("production")
	calls := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("invalid config made request")
		return nil, nil
	})
	for _, change := range []string{"zero_profile", "sandbox_receiver", "other_receiver", "repair", "expired", "missing_secret"} {
		c, p := config(), production
		switch change {
		case "zero_profile":
			p = plaidprofile.Profile{}
		case "sandbox_receiver":
			c.WebhookURL = SandboxTransactionsWebhookURL
		case "other_receiver":
			c.WebhookURL = "https://production.example/webhooks/plaid"
		case "repair":
			c.HostedRepair = true
		case "expired":
			c.ConsentExpiresAt = time.Now().Add(-time.Minute)
		case "missing_secret":
			c.ClientSecret = Secret{}
		}
		if provider, e := NewHTTPProviderForProfile(c, p, transport); e == nil || provider != nil || calls != 0 {
			t.Fatal("invalid Production capability accepted")
		}
	}
	if _, e := NewHTTPProviderForProfile(config(), production, nil); e == nil {
		t.Fatal("missing transport accepted")
	}
}

// TestProfiledSDKFailureRemainsBounded checks the new origin inherits redaction, no redirects and no retries.
func TestProfiledSDKFailureRemainsBounded(t *testing.T) {
	production, _ := plaidprofile.New("production")
	for _, status := range []int{307, 500} {
		calls := 0
		p, e := NewHTTPProviderForProfile(config(), production, roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			r := response(status, `{"error_message":"private-provider-diagnostic"}`)
			r.Header.Set("Location", "https://foreign.example/")
			return r, nil
		}))
		if e != nil {
			t.Fatal("fixture configuration")
		}
		_, e = p.Exchange(context.Background(), NewSecret("synthetic-public"))
		if e == nil || e.Error() != "provider_unavailable" || calls != 1 {
			t.Fatal("profile SDK leaked/retried/followed redirect")
		}
	}
}

// TestProfiledTransportRejectsSubstitution denies before invoking the injected transport, independent of the SDK builder.
func TestProfiledTransportRejectsSubstitution(t *testing.T) {
	production, _ := plaidprofile.New("production")
	calls := 0
	transport := boundedPlaidTransport{profile: production, base: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return response(200, `{}`), nil })}
	for _, target := range []string{"https://sandbox.plaid.com/item/get", "https://production.plaid.com/item/get?x=1", "https://production.plaid.com/transfer/create", "https://production.plaid.com/transactions/refresh"} {
		r, _ := http.NewRequest("POST", target, nil)
		if _, e := transport.RoundTrip(r); e == nil || calls != 0 {
			t.Fatal("substituted request reached dependency")
		}
	}
}
