package connections

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestWebhookEnrollmentOptIn proves only explicit reviewed configuration changes the outbound Link request.
func TestWebhookEnrollmentOptIn(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy omitted", true: "reviewed receiver"}[configured], func(t *testing.T) {
			cfg := config()
			if configured {
				cfg.WebhookURL = SandboxTransactionsWebhookURL
			}
			calls := 0
			p, err := NewHTTPProvider(cfg, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "sandbox.plaid.com" || r.URL.Path != "/link/token/create" || r.Method != http.MethodPost || r.Header.Get("Plaid-Version") != "2020-09-14" {
					t.Fatal("enrollment destination or version changed")
				}
				var body struct {
					Webhook     json.RawMessage `json:"webhook"`
					Products    []string        `json:"products"`
					AccessToken json.RawMessage `json:"access_token"`
				}
				if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Products) != 1 || body.Products[0] != "transactions" || body.AccessToken != nil {
					t.Fatal("new enrollment authority widened")
				}
				if !configured && body.Webhook != nil {
					t.Fatal("legacy request gained webhook or explicit null")
				}
				if configured && string(body.Webhook) != `"https://jayburd-dev-webhook-9088794002.us-central1.run.app/webhooks/plaid"` {
					t.Fatal("reviewed receiver missing or changed")
				}
				return response(200, `{"link_token":"synthetic-link","expiration":"2099-01-01T00:00:00Z","hosted_link_url":"https://secure.plaid.com/hl/synthetic"}`), nil
			}))
			if err != nil || calls != 0 {
				t.Fatal("constructor rejected valid configuration or made a provider call")
			}
			if _, err = p.CreateLink(context.Background(), LinkRequest{ActorID: actorID, Mode: "new"}); err != nil || calls != 1 {
				t.Fatal("enrollment failed or retried", err)
			}
			if p.SupportsMode("reconnect") {
				t.Fatal("webhook opt-in enabled unqualified reconnect")
			}
		})
	}
}

// TestWebhookEnrollmentRejectsDestinationSubstitution rejects URL variants before construction or I/O.
func TestWebhookEnrollmentRejectsDestinationSubstitution(t *testing.T) {
	for _, target := range []string{
		"http://jayburd-dev-webhook-9088794002.us-central1.run.app/webhooks/plaid",
		"https://attacker.example/webhooks/plaid",
		"https://127.0.0.1/webhooks/plaid",
		strings.Replace(SandboxTransactionsWebhookURL, "9088794002", "1234567890", 1),
		strings.Replace(SandboxTransactionsWebhookURL, "jayburd-dev-webhook", "jayburd-dev-api", 1),
		strings.Replace(SandboxTransactionsWebhookURL, ".run.app/", ".run.app.attacker.example/", 1),
		strings.Replace(SandboxTransactionsWebhookURL, "https://", "https://user@", 1),
		strings.Replace(SandboxTransactionsWebhookURL, ".run.app/", ".run.app:443/", 1),
		strings.Replace(SandboxTransactionsWebhookURL, "/webhooks/plaid", "/webhooks/%70laid", 1),
		SandboxTransactionsWebhookURL + "/", SandboxTransactionsWebhookURL + "?token=private",
		SandboxTransactionsWebhookURL + "#private", " " + SandboxTransactionsWebhookURL,
		SandboxTransactionsWebhookURL + "\n", strings.ToUpper(SandboxTransactionsWebhookURL),
	} {
		cfg := config()
		cfg.WebhookURL = target
		calls := 0
		_, err := NewHTTPProvider(cfg, roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			t.Fatal("invalid URL reached transport")
			return nil, nil
		}))
		if err == nil || err.Error() != "configuration_error" || calls != 0 || strings.Contains(err.Error(), "private") {
			t.Fatal("destination substitution accepted or reflected")
		}
	}
}
