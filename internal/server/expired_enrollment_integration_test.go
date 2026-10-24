//go:build integration

package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connectionruntime"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// expiredEnrollmentTripwire makes any managed dependency access fail the isolated browser proof.
type expiredEnrollmentTripwire struct{ t *testing.T }

// AccessToken cannot be needed to serve bookkeeping with an expired optional provider approval.
func (p expiredEnrollmentTripwire) AccessToken(context.Context) (string, error) {
	p.t.Fatal("expired approval reached token minting")
	return "", connectionruntime.ErrUnavailable
}

// IdentityToken cannot be needed by this expired optional capability.
func (p expiredEnrollmentTripwire) IdentityToken(context.Context, string) (string, error) {
	p.t.Fatal("expired approval reached identity minting")
	return "", connectionruntime.ErrUnavailable
}

// RoundTrip rejects secret, KMS and provider I/O rather than simulating a successful bank call.
func (p expiredEnrollmentTripwire) RoundTrip(*http.Request) (*http.Response, error) {
	p.t.Fatal("expired approval reached dependency transport")
	return nil, connectionruntime.ErrUnavailable
}

// expiredEnrollmentProof composes the production runtime around the real disposable restricted SQL pool.
func expiredEnrollmentProof(t *testing.T, ctx context.Context, db connections.Beginner) *connections.Service {
	t.Helper()
	c := connectionruntime.Config{Enabled: true, ProjectID: "synthetic-project", ProjectNumber: "123456789012", InfisicalOrigin: "https://us.infisical.com", InfisicalProject: "00000000-0000-4000-8000-000000000001", InfisicalIdentity: "00000000-0000-4000-8000-000000000002", InfisicalEnvironment: "dev", RegistryBucket: "synthetic-project-jayburd-dev-credential-registry", RegistryPrefix: "provider-credentials-v1", ClientIDVersion: "projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", ClientSecretVersion: "projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/2", ConsentExpiresAt: "2020-01-01T00:00:00Z"}
	if c.ManagedScope("managed", "cloudsql", c.ProjectID) != nil {
		t.Fatal("expired optional policy rejected startup")
	}
	tripwire := expiredEnrollmentTripwire{t}
	s, err := connectionruntime.New(ctx, c, db, tripwire, tripwire)
	if err != nil || s == nil {
		t.Fatal("expired optional runtime rejected bookkeeping composition")
	}
	return s
}
