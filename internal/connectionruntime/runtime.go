// Package connectionruntime composes optional Sandbox enrollment using explicit managed secret capabilities.
package connectionruntime

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
)

// ErrConfiguration rejects identifier or policy metadata without echoing it.
var ErrConfiguration = errors.New("connection_runtime_configuration")

// ErrUnavailable hides managed-store and provider-construction diagnostics.
var ErrUnavailable = errors.New("connection_runtime_unavailable")
var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var project = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
var number = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
var resource = regexp.MustCompile(`^projects/([1-9][0-9]{0,19})/secrets/[A-Za-z0-9_-]{1,128}/versions/([1-9][0-9]{0,18})$`)

// Config contains only explicit identifiers and local authorization policy, never secret values.
type Config struct {
	Enabled, LinkEventsQualified                                               bool
	ProjectID, ProjectNumber                                                   string
	InfisicalOrigin, InfisicalProject, InfisicalEnvironment, InfisicalIdentity string
	RegistryBucket, RegistryPrefix                                             string
	ClientIDVersion, ClientSecretVersion                                       string
	ConsentExpiresAt                                                           string
}

// Tokens separates runtime identity minting from static secrets and permits synthetic tests without discovery.
type Tokens interface {
	AccessToken(context.Context) (string, error)
	IdentityToken(context.Context, string) (string, error)
}

// Validate rejects partial configuration and mutable/cross-project credential references before I/O.
func (c Config) Validate() error {
	if !c.Enabled {
		if c != (Config{}) {
			return ErrConfiguration
		}
		return nil
	}
	expiry, e := time.Parse(time.RFC3339, c.ConsentExpiresAt)
	now := time.Now()
	if !c.LinkEventsQualified || !project.MatchString(c.ProjectID) || !number.MatchString(c.ProjectNumber) || !nonzeroUUID(c.InfisicalProject) || !nonzeroUUID(c.InfisicalIdentity) || c.InfisicalEnvironment != "dev" || c.RegistryBucket != c.ProjectID+"-jayburd-dev-credential-registry" || c.RegistryPrefix != "provider-credentials-v1" || e != nil || !expiry.After(now) || expiry.After(now.Add(30*24*time.Hour)) {
		return ErrConfiguration
	}
	if c.InfisicalOrigin != "https://us.infisical.com" && c.InfisicalOrigin != "https://eu.infisical.com" && c.InfisicalOrigin != "https://app.infisical.com" {
		return ErrConfiguration
	}
	for i, v := range []string{c.ClientIDVersion, c.ClientSecretVersion} {
		name := []string{"jayburd-dev-bank-sync-client-id", "jayburd-dev-bank-sync-client-secret"}[i]
		if !strings.HasPrefix(v, "projects/"+c.ProjectNumber+"/secrets/"+name+"/versions/") {
			return ErrConfiguration
		}
		parts := resource.FindStringSubmatch(v)
		if len(parts) != 3 || parts[1] != c.ProjectNumber {
			return ErrConfiguration
		}
		if _, e := strconv.ParseInt(parts[2], 10, 64); e != nil {
			return ErrConfiguration
		}
	}
	if c.ClientIDVersion == c.ClientSecretVersion {
		return ErrConfiguration
	}
	return nil
}

// nonzeroUUID prevents null identity values from masquerading as an explicit configuration.
func nonzeroUUID(v string) bool {
	return uuid.MatchString(v) && v != "00000000-0000-0000-0000-000000000000"
}

// New is inert when disabled; enabled startup reads only two exact static Secret Manager versions.
// It never opens another DB pool, begins a Link session, writes a secret or calls Plaid at startup.
func New(parent context.Context, c Config, db connections.Beginner, transport http.RoundTripper, tokens Tokens) (*connections.Service, error) {
	if c.Validate() != nil {
		return nil, ErrConfiguration
	}
	if !c.Enabled {
		return nil, nil
	}
	if parent == nil || db == nil || transport == nil || tokens == nil {
		return nil, ErrConfiguration
	}
	reader, e := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: c.ProjectNumber, Tokens: tokens, Transport: transport})
	if e != nil {
		return nil, ErrConfiguration
	}
	registry, e := providersecrets.NewGCSRegistry(providersecrets.GCSConfig{Bucket: c.RegistryBucket, Prefix: c.RegistryPrefix, Tokens: tokens, Transport: transport})
	if e != nil {
		return nil, ErrConfiguration
	}
	slots := make([]providersecrets.Slot, 8)
	for i := range slots {
		name := "slot-0" + strconv.Itoa(i)
		slots[i] = providersecrets.Slot{ID: name, InfisicalKey: name, GCPSecret: "projects/" + c.ProjectNumber + "/secrets/jayburd-dev-bank-sync-" + name}
	}
	store, e := providersecrets.NewStore(providersecrets.StoreConfig{InfisicalOrigin: c.InfisicalOrigin, ProjectID: c.InfisicalProject, Environment: c.InfisicalEnvironment, IdentityID: c.InfisicalIdentity, IdentityTokens: tokens, Transport: transport, Slots: slots}, registry, reader)
	if e != nil {
		return nil, ErrConfiguration
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, ErrUnavailable
	}
	id, e := reader.Read(ctx, c.ClientIDVersion)
	if e != nil {
		return nil, ErrUnavailable
	}
	secret, e := reader.Read(ctx, c.ClientSecretVersion)
	if e != nil || ctx.Err() != nil {
		return nil, ErrUnavailable
	}
	expiry, _ := time.Parse(time.RFC3339, c.ConsentExpiresAt)
	provider, e := connections.NewHTTPProvider(connections.HTTPConfig{ClientID: connections.NewSecret(string(id)), ClientSecret: connections.NewSecret(string(secret)), ConsentExpiresAt: expiry}, transport)
	if e != nil {
		return nil, ErrUnavailable
	}
	return connections.New(db, provider, store), nil
}

// ManagedScope rejects accidental URL database mode or a different hosting project before dependency startup.
func (c Config) ManagedScope(mode, databaseMode, projectID string) error {
	if c.Validate() != nil {
		return ErrConfiguration
	}
	if c.Enabled && (mode != "managed" || databaseMode != "cloudsql" || c.ProjectID != projectID || strings.TrimSpace(projectID) != projectID) {
		return ErrConfiguration
	}
	return nil
}
