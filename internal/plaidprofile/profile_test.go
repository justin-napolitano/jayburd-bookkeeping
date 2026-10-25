package plaidprofile

import (
	"net/http"
	"net/url"
	"testing"
)

// TestClosedPlaidProfiles fixes two literal reviewed destinations and denies malformed/zero configuration.
func TestClosedPlaidProfiles(t *testing.T) {
	for _, tc := range []struct{ name, origin string }{{"sandbox", "https://sandbox.plaid.com"}, {"production", "https://production.plaid.com"}} {
		p, e := New(tc.name)
		if e != nil || !p.Valid() || p.Environment() != tc.name || p.BaseURL() != tc.origin {
			t.Fatal("closed profile mismatch")
		}
		for _, path := range []string{"/link/token/create", "/link/token/get", "/item/public_token/exchange", "/item/get", "/accounts/get", "/item/access_token/invalidate", "/item/remove"} {
			r, _ := http.NewRequest(http.MethodPost, tc.origin+path, nil)
			if !p.AllowsConnection(r) || p.AllowsSync(r) {
				t.Fatal("connection capability mismatch")
			}
		}
		r, _ := http.NewRequest(http.MethodPost, tc.origin+"/transactions/sync", nil)
		if !p.AllowsSync(r) || p.AllowsConnection(r) {
			t.Fatal("sync capability mismatch")
		}
	}
	for _, name := range []string{"", "Sandbox", "Production", "prod", "development", " production", "production\n", "https://production.plaid.com"} {
		p, e := New(name)
		if e != ErrProfile || p.Valid() || p.BaseURL() != "" {
			t.Fatal("unreviewed profile accepted")
		}
	}
	if (Profile{}).Valid() || (Profile{}).BaseURL() != "" {
		t.Fatal("zero profile defaulted")
	}
}

// TestProfileRequestDenials excludes cross-environment, alternate authorities, URI tricks and financial products.
func TestProfileRequestDenials(t *testing.T) {
	p, _ := New("production")
	for _, raw := range []string{
		"https://sandbox.plaid.com/item/get", "http://production.plaid.com/item/get",
		"https://production.plaid.com:443/item/get", "https://production.plaid.com.evil.example/item/get",
		"https://PRODUCTION.plaid.com/item/get", "https://production.plaid.com./item/get",
		"https://synthetic@production.plaid.com/item/get", "https://production.plaid.com/item/get?",
		"https://production.plaid.com/item/get?account_id=x", "https://production.plaid.com/item/get#x",
		"https://production.plaid.com/%69tem/get", "https://production.plaid.com/item/get/",
		"https://production.plaid.com/sandbox/item/reset_login", "https://production.plaid.com/transactions/refresh",
		"https://production.plaid.com/transfer/create", "https://production.plaid.com/payment_initiation/payment/create",
		"https://production.plaid.com/investments/transactions/get", "https://production.plaid.com/liabilities/get",
	} {
		r, e := http.NewRequest(http.MethodPost, raw, nil)
		if e != nil {
			t.Fatal("synthetic request setup")
		}
		if p.AllowsConnection(r) || p.AllowsSync(r) {
			t.Fatal("unreviewed request accepted")
		}
	}
	for _, change := range []string{"method", "host", "opaque", "raw_fragment", "omit_host", "nil_url"} {
		r, _ := http.NewRequest(http.MethodPost, "https://production.plaid.com/item/get", nil)
		switch change {
		case "method":
			r.Method = http.MethodGet
		case "host":
			r.Host = "sandbox.plaid.com"
		case "opaque":
			r.URL.Opaque = "//production.plaid.com/item/get"
		case "raw_fragment":
			r.URL.RawFragment = "x"
		case "omit_host":
			r.URL.OmitHost = true
		case "nil_url":
			r.URL = nil
		}
		if p.AllowsConnection(r) || p.AllowsSync(r) {
			t.Fatal("invalid request metadata accepted")
		}
	}
	if p.AllowsConnection(nil) || p.AllowsSync(nil) {
		t.Fatal("nil request accepted")
	}
	if (Profile{}).AllowsConnection(&http.Request{Method: http.MethodPost, URL: &url.URL{Scheme: "https", Host: "production.plaid.com", Path: "/item/get"}}) {
		t.Fatal("zero profile gained authority")
	}
}
