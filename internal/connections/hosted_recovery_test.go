package connections

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// hostedHTTPFixture composes the real session adapter with intercepted fictional responses and one-shot state.
func hostedHTTPFixture(t *testing.T, result func(int) (int, string)) (*Service, *memoryRepo, *memorySecrets, map[string]int) {
	t.Helper()
	s, repo, _, store := fixture()
	calls := map[string]int{}
	p, err := NewHTTPProvider(config(), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls[r.URL.Path]++
		if r.URL.Host != "sandbox.plaid.com" {
			t.Fatal("unexpected provider destination")
		}
		switch r.URL.Path {
		case "/link/token/create":
			return response(200, `{"link_token":"synthetic-link","expiration":"2099-01-01T00:00:00Z","hosted_link_url":"https://secure.plaid.com/hl/synthetic"}`), nil
		case "/link/token/get":
			status, body := result(calls[r.URL.Path])
			return response(status, body), nil
		case "/item/public_token/exchange":
			return response(200, `{"access_token":"synthetic-access","item_id":"synthetic-item"}`), nil
		case "/item/get":
			return response(200, `{"item":{"item_id":"synthetic-item","error":null}}`), nil
		case "/accounts/get":
			return response(200, `{"item":{"item_id":"synthetic-item"},"accounts":[{"account_id":"account-one","type":"depository","subtype":"checking","balances":{"iso_currency_code":"USD"}}]}`), nil
		default:
			t.Fatal("unexpected provider operation")
			return nil, errors.New("unexpected")
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	s.provider = p
	return s, repo, store, calls
}

// modernCompletion is independently authored provider evidence with a unique completed result.
func modernCompletion() string {
	return `{"link_token":"synthetic-link","link_sessions":[{"link_session_id":"synthetic-session","finished_at":"` + time.Now().Add(-time.Minute).UTC().Format(time.RFC3339) + `","results":{"item_add_results":[{"public_token":"synthetic-public"}]}}]}`
}

// TestHostedRecoverySingleAssociation avoids a second post-claim lookup which could strand a verified session.
func TestHostedRecoverySingleAssociation(t *testing.T) {
	s, repo, store, calls := hostedHTTPFixture(t, func(n int) (int, string) {
		if n > 1 {
			return 503, `{"error_message":"private-second-read-outage"}`
		}
		return 200, modernCompletion()
	})
	if _, err := s.Begin(context.Background(), scope, "single-proof", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteHosted(context.Background(), scope, attemptID); err != nil {
		t.Fatal("verified recovery was stranded by a redundant lookup", err)
	}
	if calls["/link/token/get"] != 1 || calls["/item/public_token/exchange"] != 1 || store.writes != 1 || repo.c.State != "binding_review" {
		t.Fatal("completion did not publish exactly once")
	}
	if _, err := s.CompleteHosted(context.Background(), scope, attemptID); err != nil || calls["/link/token/get"] != 1 || calls["/item/public_token/exchange"] != 1 {
		t.Fatal("completed recovery repeated provider work")
	}
}

// TestHostedRecoveryPreclaimFailures preserves the exchange claim across pending, ambiguous and unavailable results.
func TestHostedRecoveryPreclaimFailures(t *testing.T) {
	for _, scenario := range []string{"pending", "ambiguous", "outage", "claim unknown"} {
		t.Run(scenario, func(t *testing.T) {
			s, repo, store, calls := hostedHTTPFixture(t, func(int) (int, string) {
				switch scenario {
				case "pending":
					return 200, `{"link_token":"synthetic-link","link_sessions":[]}`
				case "ambiguous":
					return 200, strings.Replace(modernCompletion(), `[{"public_token":"synthetic-public"}]`, `[{"public_token":"synthetic-public"},{"public_token":"other"}]`, 1)
				case "outage":
					return 503, `{"error_message":"private-upstream-diagnostic"}`
				default:
					return 200, modernCompletion()
				}
			})
			if _, err := s.Begin(context.Background(), scope, "failure", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}); err != nil {
				t.Fatal(err)
			}
			repo.claimUnknown = scenario == "claim unknown"
			_, err := s.CompleteHosted(context.Background(), scope, attemptID)
			if scenario == "pending" && err != nil || scenario != "pending" && err == nil {
				t.Fatal("unexpected recovery outcome", err)
			}
			if scenario != "claim unknown" && repo.claimed || calls["/item/public_token/exchange"] != 0 || store.writes != 0 {
				t.Fatal("unverified or uncertain completion exchanged/published")
			}
			if err != nil && strings.Contains(err.Error(), "private") {
				t.Fatal("upstream diagnostic escaped")
			}
			if scenario == "claim unknown" {
				if _, err := s.CompleteHosted(context.Background(), scope, attemptID); err != nil || calls["/link/token/get"] != 1 || calls["/item/public_token/exchange"] != 0 {
					t.Fatal("uncertain claim replay repeated provider work")
				}
			}
		})
	}
}

// TestBrowserPublicTokenStillVerified ensures transport-supplied tokens cannot create local server evidence.
func TestBrowserPublicTokenStillVerified(t *testing.T) {
	s, repo, _, calls := hostedHTTPFixture(t, func(int) (int, string) {
		return 200, strings.Replace(modernCompletion(), "synthetic-public", "different-public", 1)
	})
	input := begin(t, s)
	_, err := s.Execute(context.Background(), scope, attemptID, input)
	code(t, err, "credential_recovery_required")
	if !repo.claimed || calls["/link/token/get"] != 1 || calls["/item/public_token/exchange"] != 0 {
		t.Fatal("browser token bypassed independent provider association")
	}
}

// boundedHandoff observes deadline ownership without KMS calls or plaintext persistence outside the fixture.
type boundedHandoff struct {
	*memorySecrets
	read func(context.Context) error
}

// ReadLink observes the inherited deadline before delegating synthetic retained-token access.
func (s boundedHandoff) ReadLink(ctx context.Context, id string) (Secret, error) {
	if err := s.read(ctx); err != nil {
		return Secret{}, err
	}
	return s.memorySecrets.ReadLink(ctx, id)
}

// TestHostedRecoveryReadBounds stops expired/cancelled reads and bounds retained-token access before provider work.
func TestHostedRecoveryReadBounds(t *testing.T) {
	for _, scenario := range []string{"aggregate", "recovery expiry", "expired", "cancelled", "revoked", "revoked during unwrap"} {
		t.Run(scenario, func(t *testing.T) {
			s, repo, p, store := fixture()
			hosted := &hostedProvider{fakeProvider: p, pending: true}
			s.provider = hosted
			if _, err := s.Begin(context.Background(), scope, "bounded", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}}); err != nil {
				t.Fatal(err)
			}
			parent, cancel := context.WithCancel(context.Background())
			defer cancel()
			if scenario == "recovery expiry" {
				repo.c.ExpiresAt = time.Now().Add(3 * time.Second).UTC().Format(time.RFC3339Nano)
			}
			if scenario == "expired" {
				repo.c.ExpiresAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
			}
			if scenario == "cancelled" {
				cancel()
			}
			if scenario == "revoked" {
				repo.denied = true
			}
			reads := 0
			s.secrets = boundedHandoff{memorySecrets: store, read: func(ctx context.Context) error {
				reads++
				deadline, ok := ctx.Deadline()
				expiry, _ := time.Parse(time.RFC3339Nano, repo.c.ExpiresAt)
				if !ok || time.Until(deadline) > 10*time.Second || deadline.After(expiry) {
					t.Fatal("retained token read lacked aggregate/recovery bound")
				}
				if scenario == "revoked during unwrap" {
					repo.denied = true
					return nil
				}
				return errors.New("private-unwrapping-fault")
			}}
			_, err := s.CompleteHosted(parent, scope, attemptID)
			if err == nil || strings.Contains(err.Error(), "private") || hosted.checks != 0 || repo.claimed {
				t.Fatal("failed retained-token read reached provider/claim")
			}
			if (scenario == "expired" || scenario == "cancelled" || scenario == "revoked") && reads != 0 {
				t.Fatal("ineligible attempt unwrapped a retained token")
			}
		})
	}
}
