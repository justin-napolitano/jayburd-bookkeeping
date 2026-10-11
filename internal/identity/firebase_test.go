package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"firebase.google.com/go/v4/auth"
)

// verifierStub models SDK outcomes, not cryptographic or live-provider proof.
type verifierStub struct {
	token *auth.Token
	err   error
	calls int
}

// VerifyIDTokenAndCheckRevoked records that the revocation-aware method is used.
func (v *verifierStub) VerifyIDTokenAndCheckRevoked(ctx context.Context, _ string) (*auth.Token, error) {
	v.calls++
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing deadline")
	}
	return v.token, v.err
}

// resolverStub detects mapping before validation and simulates local disable/delete.
type resolverStub struct {
	calls               int
	issuer, subject, id string
	err                 error
}

// ResolvePrincipal records only the verified issuer/subject supplied by the adapter.
func (r *resolverStub) ResolvePrincipal(_ context.Context, i, s string) (string, error) {
	r.calls++
	r.issuer = i
	r.subject = s
	return r.id, r.err
}

// validToken is a synthetic SDK result; it is never a deployable credential.
func validToken() *auth.Token {
	now := time.Now().Unix()
	return &auth.Token{Issuer: "https://securetoken.google.com/synthetic-project", Audience: "synthetic-project", Subject: "synthetic-user", UID: "synthetic-user", AuthTime: now - 60, IssuedAt: now - 30, Expires: now + 300, Claims: map[string]interface{}{"firebase": map[string]interface{}{"sign_in_second_factor": "totp"}, "actor_id": "untrusted", "admin": true}}
}

// TestIdentityBoundary proves validation and revocation precede any local mapping.
func TestIdentityBoundary(t *testing.T) {
	for _, tc := range []struct {
		name        string
		change      func(*auth.Token)
		providerErr bool
		mapping     bool
	}{
		{name: "allowed", mapping: true},
		{name: "revoked", providerErr: true}, {name: "deleted_provider", providerErr: true}, {name: "disabled_provider", providerErr: true}, {name: "provider_unavailable", providerErr: true},
		{name: "issuer", change: func(v *auth.Token) { v.Issuer = "https://attacker.invalid" }},
		{name: "audience", change: func(v *auth.Token) { v.Audience = "other-project" }},
		{name: "subject", change: func(v *auth.Token) { v.Subject = "" }},
		{name: "uid", change: func(v *auth.Token) { v.UID = "another-user" }},
		{name: "expired", change: func(v *auth.Token) { v.Expires = time.Now().Unix() - 1 }},
		{name: "future_issued", change: func(v *auth.Token) { v.IssuedAt = time.Now().Unix() + 60 }},
		{name: "future_auth", change: func(v *auth.Token) { v.AuthTime = time.Now().Unix() + 60 }},
		{name: "missing_auth", change: func(v *auth.Token) { v.AuthTime = 0 }},
		{name: "tenant", change: func(v *auth.Token) { v.Firebase.Tenant = "another-tenant" }},
		{name: "no_mfa", change: func(v *auth.Token) { delete(v.Claims, "firebase") }},
		{name: "sms_mfa", change: func(v *auth.Token) { v.Claims["firebase"] = map[string]interface{}{"sign_in_second_factor": "phone"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := &verifierStub{token: validToken()}
			if tc.change != nil {
				tc.change(v.token)
			}
			if tc.providerErr {
				v.err = errors.New("sensitive provider detail")
			}
			r := &resolverStub{id: "00000000-0000-4000-8000-000000000001"}
			f := &Firebase{v, r, "synthetic-project"}
			got, err := f.Authenticate(context.Background(), "synthetic-token")
			if tc.mapping {
				if err != nil || got != r.id || r.calls != 1 || r.subject != "synthetic-user" || r.issuer != v.token.Issuer {
					t.Fatal("verified identity not mapped")
				}
			} else if err != errUnauthenticated || got != "" || r.calls != 0 {
				t.Fatal("rejected identity reached mapping")
			}
		})
	}
	for _, name := range []string{"not_enrolled", "disabled_local", "deleted_local", "mapping_failure"} {
		t.Run(name, func(t *testing.T) {
			r := &resolverStub{}
			if name == "mapping_failure" {
				r.err = errors.New("private SQL")
			}
			f := &Firebase{&verifierStub{token: validToken()}, r, "synthetic-project"}
			if _, err := f.Authenticate(context.Background(), "synthetic-token"); err != errUnauthenticated {
				t.Fatal("mapping failure leaked or allowed")
			}
		})
	}
	t.Run("emulator", func(t *testing.T) {
		t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "synthetic.invalid")
		v := &verifierStub{token: validToken()}
		f := &Firebase{v, &resolverStub{}, "synthetic-project"}
		if _, err := f.Authenticate(context.Background(), "synthetic-token"); err != errUnauthenticated || v.calls != 0 {
			t.Fatal("emulator accepted")
		}
	})
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f := &Firebase{&verifierStub{token: validToken()}, &resolverStub{id: "synthetic"}, "synthetic-project"}
		if _, err := f.Authenticate(ctx, "synthetic-token"); err != errUnauthenticated {
			t.Fatal("canceled identity accepted")
		}
	})
	if _, err := New(nil, "synthetic-project", &resolverStub{}); err == nil {
		t.Fatal("nil SDK accepted")
	}
}
