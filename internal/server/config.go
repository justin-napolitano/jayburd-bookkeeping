// Package server composes explicit managed adapters; default health mode has no dependencies.
package server

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var errConfiguration = errors.New("invalid server configuration")

// Config carries owner-injected connection material; errors never format its fields.
type Config struct{ Mode, Listen, Origin, Project, DatabaseURL string }

// Validate rejects ambiguous listener and origin configuration before dependencies start.
func (c Config) Validate() error {
	if c.Mode != "health" && c.Mode != "managed" {
		return errConfiguration
	}
	host, port, err := net.SplitHostPort(c.Listen)
	if err != nil || port == "" || (host != "" && host != "localhost" && net.ParseIP(host) == nil) {
		return errConfiguration
	}
	if c.Mode == "health" {
		return nil
	}
	u, err := url.Parse(c.Origin)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) || c.Project == "" || c.DatabaseURL == "" {
		return errConfiguration
	}
	return nil
}

// forbiddenEnvironment checks presence only, preventing SDK emulators, metadata
// redirection, credential-file discovery and inherited PostgreSQL overrides.
func forbiddenEnvironment(present func(string) bool) bool {
	for _, name := range []string{"FIREBASE_AUTH_EMULATOR_HOST", "FIREBASE_CONFIG", "GOOGLE_APPLICATION_CREDENTIALS", "GCE_METADATA_HOST", "GCE_METADATA_IP", "PGHOST", "PGHOSTADDR", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGPASSFILE", "PGSERVICE", "PGSERVICEFILE", "PGSSLMODE", "PGSSLCERT", "PGSSLKEY", "PGSSLROOTCERT", "PGSSLPASSWORD", "PGOPTIONS", "PGAPPNAME", "PGCONNECT_TIMEOUT", "PGTARGETSESSIONATTRS", "PGTZ", "PGMINPROTOCOLVERSION", "PGMAXPROTOCOLVERSION", "PGSSLSNI", "PGSSLNEGOTIATION", "PGCHANNELBINDING", "PGREQUIREAUTH"} {
		if present(name) {
			return true
		}
	}
	return false
}

// poolConfig accepts a narrow explicit URL and overwrites file-loading defaults.
// TLS verifies host and system roots; unencrypted transport is loopback-proxy only.
func poolConfig(raw string) (*pgxpool.Config, error) {
	if forbiddenEnvironment(func(k string) bool { _, ok := os.LookupEnv(k); return ok }) {
		return nil, errConfiguration
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.User == nil || u.User.Username() == "" || u.Hostname() == "" || strings.Contains(u.Hostname(), ",") || u.Path == "" || u.Path == "/" || u.Fragment != "" {
		return nil, errConfiguration
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, errConfiguration
	}
	for k, v := range q {
		if k != "sslmode" || len(v) != 1 {
			return nil, errConfiguration
		}
	}
	mode := q.Get("sslmode")
	if mode == "" {
		mode = "verify-full"
	}
	if mode != "verify-full" && !(mode == "disable" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")) {
		return nil, errConfiguration
	}
	q.Set("sslmode", mode)
	q.Set("sslrootcert", "system")
	q.Set("sslcert", "")
	q.Set("sslkey", "")
	q.Set("passfile", "/dev/null")
	q.Set("connect_timeout", "5")
	u.RawQuery = q.Encode()
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, errConfiguration
	}
	cfg.MaxConns = 8
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = time.Minute
	cfg.ConnConfig.RuntimeParams = map[string]string{"application_name": "jayburd-api", "statement_timeout": "5000", "lock_timeout": "4000", "idle_in_transaction_session_timeout": "5000", "search_path": "pg_catalog"}
	return cfg, nil
}
