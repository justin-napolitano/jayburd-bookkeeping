// Package clouddb owns metadata-only Cloud SQL transport and bounded connection configuration.
package clouddb

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2/google"
)

var ErrConfiguration = errors.New("invalid cloud database configuration")
var instancePattern = regexp.MustCompile(`^([a-z][a-z0-9-]{4,28}[a-z0-9]):us-central1:[a-z][a-z0-9-]{0,96}[a-z0-9]$`)

// Config contains identifiers only; user/database are derived to prevent role substitution.
type Config struct{ Instance, Component string }

// Validate confines callers to the reviewed region and the explicitly implemented identities.
func (c Config) Validate() error {
	if !instancePattern.MatchString(c.Instance) || (c.Component != "api" && c.Component != "migrator" && c.Component != "publisher" && c.Component != "sync" && c.Component != "webhook" && c.Component != "verifier") {
		return ErrConfiguration
	}
	return nil
}

// Project returns the explicit instance project after configuration validation.
func (c Config) Project() string { return strings.Split(c.Instance, ":")[0] }

// PoolConfig disables pgx TLS only inside the connector's encrypted tunnel, never for raw TCP.
// The injected dial function also makes the network boundary independently testable.
func PoolConfig(c Config, dial func(context.Context, string, string) (net.Conn, error)) (*pgxpool.Config, error) {
	return PoolConfigForEnvironment(c, "sandbox", dial)
}

// ValidateEnvironment reserves a separate IAM naming profile without granting or provisioning roles.
func (c Config) ValidateEnvironment(environment string) error {
	if c.Validate() != nil || (environment != "sandbox" && environment != "production") {
		return ErrConfiguration
	}
	if environment == "production" && (c.Project() == "jayburd-bookkeeping-dev" || strings.HasSuffix(c.Instance, ":jayburd-dev") || (c.Component != "api" && c.Component != "sync" && c.Component != "publisher" && c.Component != "webhook")) {
		return ErrConfiguration
	}
	return nil
}

// PoolConfigForEnvironment derives a dedicated role from a closed profile, never a caller login string.
func PoolConfigForEnvironment(c Config, environment string, dial func(context.Context, string, string) (net.Conn, error)) (*pgxpool.Config, error) {
	if c.ValidateEnvironment(environment) != nil || dial == nil || ForbiddenEnvironment(func(k string) bool { _, ok := os.LookupEnv(k); return ok }) {
		return nil, ErrConfiguration
	}
	prefix := "jayburd-dev-"
	if environment == "production" {
		prefix = "jayburd-prod-"
	}
	u := url.URL{Scheme: "postgres", Host: "localhost", Path: "/bookkeeping", User: url.User(prefix + c.Component + "@" + c.Project() + ".iam")}
	q := url.Values{"sslmode": {"disable"}, "passfile": {"/dev/null"}, "sslcert": {""}, "sslkey": {""}, "sslrootcert": {""}, "connect_timeout": {"15"}}
	u.RawQuery = q.Encode()
	p, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, ErrConfiguration
	}
	p.ConnConfig.DialFunc = dial
	p.ConnConfig.Fallbacks = nil
	p.MaxConns = 5
	if c.Component != "api" {
		p.MaxConns = 2
	}
	// The finite verifier performs one controlled comparison, never parallel acquisition or posting.
	if c.Component == "verifier" {
		p.MaxConns = 1
	}
	p.MinConns = 0
	p.MaxConnLifetime = 30 * time.Minute
	p.MaxConnLifetimeJitter = 5 * time.Minute
	p.MaxConnIdleTime = 5 * time.Minute
	p.HealthCheckPeriod = time.Minute
	p.ConnConfig.RuntimeParams = map[string]string{"application_name": "jayburd-" + c.Component, "search_path": "pg_catalog", "statement_timeout": "5000", "lock_timeout": "4000", "idle_in_transaction_session_timeout": "5000"}
	if c.Component == "migrator" {
		p.ConnConfig.RuntimeParams = map[string]string{"application_name": "jayburd-migrator", "search_path": "deployment,pg_catalog", "statement_timeout": "60000", "lock_timeout": "10000", "idle_in_transaction_session_timeout": "60000"}
	}
	return p, nil
}

// Connector returns an explicit pgx configuration and a close hook; it never discovers ADC files.
func Connector(ctx context.Context, c Config) (*pgxpool.Config, func(), error) {
	return ConnectorForEnvironment(ctx, c, "sandbox")
}

// ConnectorForEnvironment keeps metadata-only authentication and rejects mixed profiles before SDK I/O.
func ConnectorForEnvironment(ctx context.Context, c Config, environment string) (*pgxpool.Config, func(), error) {
	if c.ValidateEnvironment(environment) != nil || ForbiddenEnvironment(func(k string) bool { _, ok := os.LookupEnv(k); return ok }) {
		return nil, nil, ErrConfiguration
	}
	d, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN(), cloudsqlconn.WithLazyRefresh(), cloudsqlconn.WithRefreshTimeout(15*time.Second), cloudsqlconn.WithIAMAuthNTokenSources(
		google.ComputeTokenSource("default", "https://www.googleapis.com/auth/sqlservice.admin", "https://www.googleapis.com/auth/cloud-platform"),
		google.ComputeTokenSource("default", "https://www.googleapis.com/auth/sqlservice.login")))
	if err != nil {
		return nil, nil, errors.New("database transport unavailable")
	}
	close := func() { _ = d.Close() }
	cfg, err := PoolConfigForEnvironment(c, environment, func(ctx context.Context, _, _ string) (net.Conn, error) {
		return d.Dial(ctx, c.Instance, cloudsqlconn.WithPublicIP())
	})
	if err != nil {
		close()
		return nil, nil, ErrConfiguration
	}
	return cfg, close, nil
}
