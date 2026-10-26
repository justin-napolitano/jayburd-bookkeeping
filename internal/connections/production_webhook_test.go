package connections

import (
	"context"
	"encoding/json"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
	"net/http"
	"strings"
	"testing"
)

// TestProductionWebhookEnrollment fixes notification routing and keeps repair separately opted in.
func TestProductionWebhookEnrollment(t *testing.T) {
	profile, _ := plaidprofile.New("production")
	for _, repair := range []bool{false, true} {
		cfg := config()
		cfg.WebhookProjectNumber = "123456789012"
		cfg.WebhookURL = "https://jayburd-prod-webhook-123456789012.us-central1.run.app/webhooks/plaid"
		cfg.HostedRepair = repair
		calls := 0
		p, e := NewHTTPProviderForProfile(cfg, profile, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			var body map[string]any
			if json.NewDecoder(r.Body).Decode(&body) != nil || r.URL.String() != "https://production.plaid.com/link/token/create" || body["webhook"] != "https://jayburd-prod-webhook-123456789012.us-central1.run.app/webhooks/plaid" {
				t.Fatal("wrong fixed Production enrollment")
			}
			return response(200, `{"link_token":"synthetic-production-link","expiration":"2099-01-01T00:00:00Z","hosted_link_url":"https://secure.plaid.com/hl/synthetic-production"}`), nil
		}))
		if e != nil || calls != 0 || p.SupportsMode("reconnect") != repair {
			t.Fatal("constructor/repair authority changed")
		}
		if _, e = p.CreateLink(context.Background(), LinkRequest{ActorID: actorID, Mode: "new"}); e != nil || calls != 1 {
			t.Fatal("new enrollment failed", e)
		}
	}
}

// TestProductionWebhookDestinationDeny rejects same-looking URLs and development metadata before transport.
func TestProductionWebhookDestinationDeny(t *testing.T) {
	profile, _ := plaidprofile.New("production")
	base := "https://jayburd-prod-webhook-123456789012.us-central1.run.app/webhooks/plaid"
	for _, url := range []string{SandboxTransactionsWebhookURL, base + "/", base + "?token=synthetic", strings.Replace(base, "123456789012", "999999999999", 1), strings.Replace(base, "https://", "https://user@", 1), strings.Replace(base, ".run.app/", ".run.app:443/", 1), "https://attacker.invalid/webhooks/plaid"} {
		cfg := config()
		cfg.WebhookProjectNumber = "123456789012"
		cfg.WebhookURL = url
		if _, e := NewHTTPProviderForProfile(cfg, profile, roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid destination reached transport")
			return nil, nil
		})); e == nil {
			t.Fatal("invalid destination accepted")
		}
	}
	for _, number := range []string{"", "0", "01", "9088794002", "1/other", "9223372036854775808"} {
		if _, e := ProductionTransactionsWebhookURL(number); e == nil {
			t.Fatal("invalid receiver identifier accepted")
		}
	}
}
