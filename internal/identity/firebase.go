// Package identity maps provider-verified identities to enrolled ledger principals.
// It never initializes credentials or grants permissions from token claims.
package identity

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
)

var errUnauthenticated = errors.New("unauthenticated")
var projectPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

// Resolver resolves only the verified issuer/subject; SQL rechecks authorization.
type Resolver interface {
	ResolvePrincipal(context.Context, string, string) (string, error)
}

// tokenVerifier allows synthetic boundary tests; public construction needs the SDK.
type tokenVerifier interface {
	VerifyIDTokenAndCheckRevoked(context.Context, string) (*auth.Token, error)
}

// Firebase requires managed TOTP sign-in and an explicitly enrolled principal.
type Firebase struct {
	client   tokenVerifier
	resolver Resolver
	project  string
}

// New binds a trusted non-emulator Admin client to one explicit project.
// Composition owns workload identity, client configuration and network lifecycle.
func New(client *auth.Client, project string, resolver Resolver) (*Firebase, error) {
	if client == nil || resolver == nil || !projectPattern.MatchString(project) || emulatorPresent() {
		return nil, errUnauthenticated
	}
	return &Firebase{client: client, resolver: resolver, project: project}, nil
}

// emulatorPresent checks presence only; it never returns or records an env value.
func emulatorPresent() bool {
	_, present := os.LookupEnv("FIREBASE_AUTH_EMULATOR_HOST")
	return present
}

// Authenticate checks revocation every time and suppresses provider/database details.
func (f *Firebase) Authenticate(parent context.Context, raw string) (string, error) {
	if f == nil || f.client == nil || f.resolver == nil || !projectPattern.MatchString(f.project) || emulatorPresent() || len(raw) == 0 || len(raw) > 16384 || strings.ContainsAny(raw, " \r\n\t") {
		return "", errUnauthenticated
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	token, err := f.client.VerifyIDTokenAndCheckRevoked(ctx, raw)
	if err != nil || token == nil {
		return "", errUnauthenticated
	}
	now := time.Now().Unix()
	if token.Issuer != "https://securetoken.google.com/"+f.project || token.Audience != f.project || token.Subject == "" || len(token.Subject) > 128 || token.UID != token.Subject || token.Firebase.Tenant != "" || token.Expires <= now || token.IssuedAt <= 0 || token.IssuedAt > now || token.AuthTime <= 0 || token.AuthTime > token.IssuedAt || token.Expires <= token.IssuedAt {
		return "", errUnauthenticated
	}
	firebase, ok := token.Claims["firebase"].(map[string]interface{})
	if !ok || firebase["sign_in_second_factor"] != "totp" {
		return "", errUnauthenticated
	}
	principal, err := f.resolver.ResolvePrincipal(ctx, token.Issuer, token.Subject)
	if err != nil || principal == "" || ctx.Err() != nil {
		return "", errUnauthenticated
	}
	return principal, nil
}
