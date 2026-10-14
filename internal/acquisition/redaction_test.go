package acquisition

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestInjectedErrorsRedact closes malicious dependency classifications at the public runner boundary.
func TestInjectedErrorsRedact(t *testing.T) {
	for _, injected := range []error{&ProviderFailure{Code: "synthetic-token-private"}, &Failure{Code: "synthetic-token-private"}} {
		repo := &fakeRepository{context: contextFixture()}
		runner := &Runner{repo: repo, provider: providerFunc(func(context.Context, FetchRequest) ([]byte, error) { return nil, injected }), credentials: credentialFunc(func(context.Context, CredentialReference) (ProviderCredentials, error) {
			return credentialsFixture(), nil
		}), evidence: localEvidence(t), slots: make(chan struct{}, 1)}
		_, err := runner.Run(context.Background(), Scope{testEntity, testBook, testStream}, testClaim)
		if err == nil || strings.Contains(err.Error(), "synthetic-token-private") {
			t.Fatalf("unclosed error classification: %v", err)
		}
	}
	c := credentialsFixture()
	for _, formatted := range []string{fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c)} {
		if strings.Contains(formatted, c.AccessToken) || strings.Contains(formatted, c.Secret) {
			t.Fatal("credential formatting exposed token")
		}
	}
	if raw, err := json.Marshal(c); err == nil || len(raw) != 0 {
		t.Fatal("credential serialization accepted")
	}
}

// TestProviderFailures bounds remote bodies, redirects and Retry-After without exposing diagnostics.
func TestProviderFailures(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, tc := range []struct {
		value    string
		delay    time.Duration
		deferred bool
	}{{"3600", time.Hour, false}, {now.Add(time.Hour).Format(http.TimeFormat), time.Hour, false}, {"86401", 0, true}, {"-1", 5 * time.Second, false}, {"garbage", 5 * time.Second, false}, {now.Add(-time.Hour).Format(http.TimeFormat), 5 * time.Second, false}} {
		delay, deferred := retryAfter(tc.value, now)
		if delay != tc.delay || deferred != tc.deferred {
			t.Fatalf("retry policy: %v %v", delay, deferred)
		}
	}
	for _, status := range []int{302, 429, 500} {
		calls := 0
		p, _ := NewPlaidHTTP(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"86401"}, "Location": []string{"https://untrusted.invalid"}}, Body: io.NopCloser(strings.NewReader(`{"error_code":"synthetic-token-private"}`)), Request: r}, nil
		}))
		_, err := p.Fetch(context.Background(), FetchRequest{AccountID: "Account-A", Credentials: credentialsFixture()})
		if err == nil || strings.Contains(err.Error(), "synthetic-token-private") || calls != 1 {
			t.Fatal("unsafe response or automatic retry")
		}
		if status != 302 && err.Error() != "retry_deferred" {
			t.Fatal("long retry shortened")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, _ := NewPlaidHTTP(roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("cancelled fetch reached transport")
		return nil, nil
	}))
	if _, err := p.Fetch(ctx, FetchRequest{AccountID: "Account-A", Credentials: credentialsFixture()}); err == nil {
		t.Fatal("cancel accepted")
	}
}
