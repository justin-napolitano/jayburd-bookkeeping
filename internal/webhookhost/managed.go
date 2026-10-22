package webhookhost

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
)

// Config contains explicit public identifiers only, never credentials, DSNs or role overrides.
type Config struct {
	Listen, Instance, ProjectNumber, ClientIDVersion, ClientSecretVersion string
}

// numeric requires a canonical positive identifier/version, excluding aliases and leading zeros.
func numeric(value string) bool {
	n, err := strconv.ParseUint(value, 10, 63)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == value
}

// secretVersion fixes a single approved resource name in the explicit numeric project.
func secretVersion(value, project, name string) bool {
	prefix := "projects/" + project + "/secrets/" + name + "/versions/"
	return strings.HasPrefix(value, prefix) && numeric(strings.TrimPrefix(value, prefix))
}

// Validate rejects role, destination and secret substitution before any external access.
func (c Config) Validate() error {
	if (clouddb.Config{Instance: c.Instance, Component: "webhook"}).Validate() != nil || !numeric(c.ProjectNumber) {
		return ErrConfiguration
	}
	host, port, err := net.SplitHostPort(c.Listen)
	n, pe := strconv.Atoi(port)
	if err != nil || pe != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port || (host != "127.0.0.1" && host != "0.0.0.0") {
		return ErrConfiguration
	}
	if !secretVersion(c.ClientIDVersion, c.ProjectNumber, "jayburd-dev-bank-sync-client-id") || !secretVersion(c.ClientSecretVersion, c.ProjectNumber, "jayburd-dev-bank-sync-client-secret") {
		return ErrConfiguration
	}
	return nil
}

// Runtime retains the fixed listener configuration and explicitly owned host dependencies.
type Runtime struct {
	host    *Host
	listen  string
	started atomic.Bool
}

// Close releases dependencies once; callers stop/join Run before explicit cleanup.
func (r *Runtime) Close() {
	if r != nil {
		r.host.Close()
	}
}

// Run opens only the validated bind and serves the verified route; it performs no migration/enrollment.
func (r *Runtime) Run(ctx context.Context) error {
	if r == nil || r.host == nil || ctx == nil {
		return ErrConfiguration
	}
	if r.host.closed.Load() {
		return ErrAlreadyServed
	}
	if !r.started.CompareAndSwap(false, true) {
		return ErrAlreadyServed
	}
	defer r.Close()
	if ctx.Err() != nil {
		return ErrServe
	}
	l, err := net.Listen("tcp", r.listen)
	if err != nil {
		return ErrServe
	}
	defer l.Close()
	return r.host.Serve(ctx, l)
}

// factories isolate initialization faults without a test auth route or ambient runtime override.
type factories struct {
	keys func(Config) (webhook.KeyProvider, func(), error)
	pool func(context.Context, clouddb.Config) (webhook.Beginner, func(), error)
}

// Managed constructs concrete metadata-only adapters; only startup transport Ping contacts Cloud SQL.
// Secrets and provider keys are resolved lazily for bounded JWT key-ID cache misses, never during startup.
func Managed(ctx context.Context, c Config) (*Runtime, error) {
	return managed(ctx, c, factories{keys: managedKeys, pool: managedPool})
}

// managed validates first and unwinds partial ownership in reverse acquisition order.
func managed(ctx context.Context, c Config, f factories) (*Runtime, error) {
	if ctx == nil || c.Validate() != nil || f.keys == nil || f.pool == nil {
		return nil, ErrConfiguration
	}
	if ctx.Err() != nil {
		return nil, ErrServe
	}
	keys, closeKeys, err := f.keys(c)
	if closeKeys == nil {
		closeKeys = func() {}
	}
	if err != nil || keys == nil {
		closeKeys()
		return nil, ErrConfiguration
	}
	if ctx.Err() != nil {
		closeKeys()
		return nil, ErrServe
	}
	startup, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	db, closeDB, err := f.pool(startup, clouddb.Config{Instance: c.Instance, Component: "webhook"})
	if closeDB == nil {
		closeDB = func() {}
	}
	cleanup := func() { closeDB(); closeKeys() }
	if err != nil || db == nil || startup.Err() != nil {
		cleanup()
		return nil, ErrServe
	}
	h, err := NewSQL(keys, db, cleanup)
	if err != nil {
		cleanup()
		return nil, ErrConfiguration
	}
	return &Runtime{host: h, listen: c.Listen}, nil
}

// managedPool owns a dedicated webhook connector/pool and tests transport only, not SQL role authority.
func managedPool(ctx context.Context, c clouddb.Config) (webhook.Beginner, func(), error) {
	p, closeConnector, err := clouddb.Connector(ctx, c)
	if err != nil {
		return nil, nil, ErrServe
	}
	pool, err := pgxpool.NewWithConfig(ctx, p)
	if err != nil {
		closeConnector()
		return nil, nil, ErrServe
	}
	closeAll := func() { pool.Close(); closeConnector() }
	if err = pool.Ping(ctx); err != nil {
		closeAll()
		return nil, nil, ErrServe
	}
	return pool, closeAll, nil
}

// secretReader is the existing integrity-checked secret capability, never a configurable request path.
type secretReader interface {
	Read(context.Context, string) ([]byte, error)
}

// clientCredentials retains only pinned resource identifiers; resolved values stay in memory.
type clientCredentials struct {
	reader     secretReader
	id, secret string
}

// Resolve performs no database work and never returns provider/secret diagnostics.
func (c clientCredentials) Resolve(ctx context.Context) (webhook.Credentials, error) {
	id, err := c.reader.Read(ctx, c.id)
	if err != nil {
		return webhook.Credentials{}, ErrServe
	}
	secret, err := c.reader.Read(ctx, c.secret)
	if err != nil {
		return webhook.Credentials{}, ErrServe
	}
	return webhook.Credentials{ClientID: string(id), Secret: string(secret)}, nil
}

// managedKeys creates inert bounded adapters with fixed secret paths and direct proxy-free transport.
func managedKeys(c Config) (webhook.KeyProvider, func(), error) {
	t := workerruntime.DirectTransport()
	closeAll := func() { t.CloseIdleConnections() }
	tokens, err := workerruntime.NewMetadataTokens(t)
	if err != nil {
		closeAll()
		return nil, nil, ErrConfiguration
	}
	reader, err := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: c.ProjectNumber, Tokens: tokens, Transport: t})
	if err != nil {
		closeAll()
		return nil, nil, ErrConfiguration
	}
	keys, err := webhook.NewHTTPKeys(t, clientCredentials{reader: reader, id: c.ClientIDVersion, secret: c.ClientSecretVersion})
	if err != nil {
		closeAll()
		return nil, nil, ErrConfiguration
	}
	return keys, closeAll, nil
}
