// Package plaidprofile defines closed provider destinations, never credential or runtime authority.
package plaidprofile

import (
	"errors"
	"net/http"
)

// APIVersion pins the reviewed wire profile shared by enrollment and acquisition.
const APIVersion = "2020-09-14"

var ErrProfile = errors.New("invalid_plaid_profile")

// Profile has private immutable state; callers cannot substitute an arbitrary URL.
type Profile struct{ environment string }

// New accepts only the two exact reviewed environment names, without normalization or fallback.
func New(environment string) (Profile, error) {
	if environment != "sandbox" && environment != "production" {
		return Profile{}, ErrProfile
	}
	return Profile{environment: environment}, nil
}

// Sandbox supplies the explicitly fixed legacy constructor profile without ambient configuration.
func Sandbox() Profile { return Profile{environment: "sandbox"} }

// Valid also rejects the zero value, which must never become an implicit Production destination.
func (p Profile) Valid() bool { return p.environment == "sandbox" || p.environment == "production" }

// Environment returns nonsecret routing metadata; it is not proof of credential or SQL provenance.
func (p Profile) Environment() string { return p.environment }

// BaseURL returns only a reviewed HTTPS origin, or empty for an invalid profile.
func (p Profile) BaseURL() string {
	if !p.Valid() {
		return ""
	}
	return "https://" + p.environment + ".plaid.com"
}

// request checks exact authority and URI spelling before any transport can receive credentials.
func (p Profile) request(r *http.Request) bool {
	if !p.Valid() || r == nil || r.URL == nil {
		return false
	}
	u := r.URL
	host := p.environment + ".plaid.com"
	return r.Method == http.MethodPost && u.Scheme == "https" && u.Host == host &&
		(r.Host == "" || r.Host == host) && u.User == nil && u.Opaque == "" &&
		u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.RawFragment == "" &&
		u.RawPath == "" && !u.OmitHost
}

// AllowsConnection keeps Item/credential lifecycle separate from unapproved financial products.
func (p Profile) AllowsConnection(r *http.Request) bool {
	if !p.request(r) {
		return false
	}
	switch r.URL.Path {
	case "/link/token/create", "/link/token/get", "/item/public_token/exchange", "/item/get", "/accounts/get", "/item/access_token/invalidate", "/item/remove":
		return true
	}
	return false
}

// AllowsSync permits only the account-filtered sync transport, never refresh or Sandbox test calls.
func (p Profile) AllowsSync(r *http.Request) bool {
	return p.request(r) && r.URL.Path == "/transactions/sync"
}
