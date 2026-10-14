package sandboxqualification

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestHostedScopeAndCleanup proves fixed-host calls, cleanup and no credential/provider output.
func TestHostedScopeAndCleanup(t *testing.T) {
	for _, broken := range []string{"", "exchange", "accounts", "sync", "cleanup"} {
		t.Run(broken, func(t *testing.T) {
			paths := []string{}
			tr := fakeTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "sandbox.plaid.com" || req.URL.Scheme != "https" {
					t.Fatal("non-Sandbox request")
				}
				path := req.URL.Path
				paths = append(paths, path)
				body := ""
				status := 200
				switch path {
				case "/sandbox/public_token/create":
					body = `{"public_token":"public-sandbox-private"}`
				case "/item/public_token/exchange":
					body = `{"access_token":"access-sandbox-private"}`
					if broken == "exchange" {
						status = 500
					}
				case "/accounts/get":
					body = `{"accounts":[{"account_id":"Account-A","balances":{"iso_currency_code":"USD"}}]}`
					if broken == "accounts" {
						status = 500
					}
				case "/transactions/sync":
					body = `{"accounts":[],"added":[],"modified":[],"removed":[],"next_cursor":"C1","has_more":false,"request_id":"private","transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`
					if broken == "sync" {
						status = 500
					}
				case "/item/remove":
					body = `{"request_id":"private"}`
					if broken == "cleanup" {
						status = 500
					}
				default:
					t.Fatal("unexpected endpoint")
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			r := RunHosted(context.Background(), true, []byte("client-private"), []byte("secret-private"), tr)
			b, _ := json.Marshal(r)
			for _, value := range []string{"client-private", "secret-private", "access-sandbox", "public-sandbox", "Account-A", "C1"} {
				if strings.Contains(string(b), value) {
					t.Fatal("leaked data")
				}
			}
			removes := 0
			creates := 0
			for _, p := range paths {
				if p == "/item/remove" {
					removes++
				}
				if p == "/sandbox/public_token/create" {
					creates++
				}
			}
			if creates != 1 {
				t.Fatal("creation retried")
			}
			if broken == "exchange" {
				if removes != 0 {
					t.Fatal("cleanup without token")
				}
			} else if removes != 1 {
				t.Fatal("missing cleanup")
			}
			if broken == "" && r.Status != "partial" {
				t.Fatalf("unexpected status: %s", r.Status)
			}
			if broken == "cleanup" && r.Checks[len(r.Checks)-1].Status != "blocked" {
				t.Fatal("cleanup falsely passed")
			}
		})
	}
}

// TestHostedOptIn rejects before a transport can see any credential.
func TestHostedOptIn(t *testing.T) {
	r := RunHosted(context.Background(), false, []byte("client"), []byte("secret"), fakeTransport(func(*http.Request) (*http.Response, error) { t.Fatal("network without opt-in"); return nil, nil }))
	if r.Status != "blocked" {
		t.Fatal("missing gate")
	}
}
