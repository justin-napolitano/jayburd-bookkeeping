package workerruntime

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

// ConsumerAdapters are explicitly constructed durable storage and credential resolution capabilities.
type ConsumerAdapters struct {
	Credentials       acquisition.CredentialResolver
	CredentialFactory func(acquisition.Beginner) (acquisition.CredentialResolver, error)
	Evidence          acquisition.EvidenceStore
}

// Managed constructs actual Cloud SQL/queue/provider clients using only metadata identity and explicit adapters.
// It does not create bindings, fetch bank data, migrate or start a listener.
func Managed(parent context.Context, c Config, a ConsumerAdapters) (*Runtime, error) {
	if c.Validate() != nil || c.Mode == "consumer" && ((a.Credentials == nil && a.CredentialFactory == nil) || (a.Credentials != nil && a.CredentialFactory != nil) || a.Evidence == nil) || c.Mode == "publisher" && (a.Credentials != nil || a.CredentialFactory != nil || a.Evidence != nil) {
		return nil, ErrConfiguration
	}
	if !c.ApprovedAt(time.Now()) {
		return nil, ErrUnavailable
	}
	transport := DirectTransport()
	dependency := c.ApprovedTransport(transport)
	tokens, e := NewMetadataTokens(dependency)
	if e != nil {
		transport.CloseIdleConnections()
		return nil, ErrConfiguration
	}
	verifier, e := observer.NewGoogleTokenVerifier(dependency)
	if e != nil {
		transport.CloseIdleConnections()
		return nil, ErrConfiguration
	}
	var queue observer.Transport
	if c.Mode == "publisher" {
		queue, e = observer.NewAcquisitionCloudTasksTransport(c.Queue, tokens, dependency)
		if e != nil {
			transport.CloseIdleConnections()
			return nil, ErrConfiguration
		}
	}
	// Validate the exact caller policy before opening any database transport.
	if _, e = observer.NewAcquisitionConsumerHTTP(inertConsumer{}, verifier, c.Caller); e != nil {
		transport.CloseIdleConnections()
		return nil, ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	config, closeConnector, e := clouddb.ConnectorForEnvironment(ctx, clouddb.Config{Instance: c.Instance, Component: c.component()}, c.ProviderEnvironment())
	if e != nil {
		transport.CloseIdleConnections()
		return nil, ErrUnavailable
	}
	pool, e := pgxpool.NewWithConfig(ctx, config)
	if e != nil {
		closeConnector()
		transport.CloseIdleConnections()
		return nil, ErrUnavailable
	}
	closeAll := func() { pool.Close(); closeConnector(); transport.CloseIdleConnections() }
	if e = pool.Ping(ctx); e != nil {
		closeAll()
		return nil, ErrUnavailable
	}
	var runner *acquisition.Runner
	if c.Mode == "consumer" {
		profile, _ := plaidprofile.New(c.ProviderEnvironment())
		providerConstructor := acquisition.NewPlaidHTTPForProfile
		if c.Categories {
			providerConstructor = acquisition.NewPlaidHTTPForProfileWithCategories
		}
		provider, err := providerConstructor(profile, dependency)
		if err != nil {
			closeAll()
			return nil, ErrConfiguration
		}
		credentials := a.Credentials
		if a.CredentialFactory != nil {
			credentials, e = a.CredentialFactory(pool)
			if e != nil || credentials == nil {
				closeAll()
				return nil, ErrConfiguration
			}
		}
		runnerConstructor := acquisition.New
		if c.Categories {
			runnerConstructor = acquisition.NewWithCategories
		}
		runner, e = runnerConstructor(pool, provider, credentials, a.Evidence)
		if e != nil {
			closeAll()
			return nil, ErrConfiguration
		}
	}
	d := Dependencies{DB: pool, Verifier: verifier, Transport: queue, Close: closeAll}
	if runner != nil {
		d.Runner = runner
	}
	r, e := New(c, d)
	if e != nil {
		closeAll()
		return nil, ErrConfiguration
	}
	return r, nil
}

// ApprovedAt keeps a finite Production worker inert after approval expires, including cold restart.
func (c Config) ApprovedAt(now time.Time) bool {
	if c.ProviderEnvironment() == "sandbox" {
		return true
	}
	if c.ProviderEnvironment() != "production" {
		return false
	}
	expiry, e := time.Parse(time.RFC3339, c.ApprovalExpiresAt)
	return e == nil && expiry.After(now)
}

// ApprovedTransport fences each new dependency request during multi-call worker work.
func (c Config) ApprovedTransport(base http.RoundTripper) http.RoundTripper {
	return approvedTransport{config: c, base: base}
}

// approvedTransport checks local approval without minting identity or opening a dependency connection.
type approvedTransport struct {
	config Config
	base   http.RoundTripper
}

// RoundTrip never extends authority on a warm instance or reveals rejected configuration.
func (t approvedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !t.config.ApprovedAt(time.Now()) {
		return nil, ErrUnavailable
	}
	return t.base.RoundTrip(r)
}

// inertConsumer validates policy construction only and is never exposed by a runtime listener.
type inertConsumer struct{}

// Handle fails closed if this constructor-only sentinel is ever called.
func (inertConsumer) Handle(context.Context, []byte) error { return ErrConfiguration }
