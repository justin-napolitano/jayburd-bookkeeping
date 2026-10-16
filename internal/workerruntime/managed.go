package workerruntime

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/observer"
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
	transport := DirectTransport()
	tokens, e := NewMetadataTokens(transport)
	if e != nil {
		transport.CloseIdleConnections()
		return nil, ErrConfiguration
	}
	verifier, e := observer.NewGoogleTokenVerifier(transport)
	if e != nil {
		transport.CloseIdleConnections()
		return nil, ErrConfiguration
	}
	var queue observer.Transport
	if c.Mode == "publisher" {
		queue, e = observer.NewAcquisitionCloudTasksTransport(c.Queue, tokens, transport)
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
	config, closeConnector, e := clouddb.Connector(ctx, clouddb.Config{Instance: c.Instance, Component: c.component()})
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
		provider, err := acquisition.NewPlaidHTTP(transport)
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
		runner, e = acquisition.New(pool, provider, credentials, a.Evidence)
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

// inertConsumer validates policy construction only and is never exposed by a runtime listener.
type inertConsumer struct{}

// Handle fails closed if this constructor-only sentinel is ever called.
func (inertConsumer) Handle(context.Context, []byte) error { return ErrConfiguration }
