package connectionruntime

import (
	"context"
	"net/http"
	"testing"
)

// TestProductionWebhookRuntimeOptIn permits only the configured project's canonical receiver and static startup reads.
func TestProductionWebhookRuntimeOptIn(t *testing.T) {
	c := productionConfig()
	c.WebhookURL = "https://jayburd-prod-webhook-123456789012.us-central1.run.app/webhooks/plaid"
	c.HostedRepair = true
	calls := 0
	s, e := New(context.Background(), c, inertDB{}, transport(func(r *http.Request) (*http.Response, error) {
		refs := []string{c.ClientIDVersion, c.ClientSecretVersion}
		if calls >= 2 || r.URL.String() != "https://secretmanager.googleapis.com/v1/"+refs[calls]+":access" {
			t.Fatal("unexpected startup dependency")
		}
		name := refs[calls]
		calls++
		return response(name, "synthetic-production-static"), nil
	}), &tokens{})
	if e != nil || s == nil || calls != 2 {
		t.Fatal("matching receiver rejected", e)
	}
	c.WebhookURL = "https://jayburd-prod-webhook-999999999999.us-central1.run.app/webhooks/plaid"
	if c.Validate() != ErrConfiguration {
		t.Fatal("foreign receiver accepted")
	}
}
