// Package connectionruntime composes optional closed-profile enrollment from explicit managed capabilities.
package connectionruntime

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
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
	Enabled, LinkEventsQualified, HostedRepair                                 bool
	ProjectID, ProjectNumber                                                   string
	InfisicalOrigin, InfisicalProject, InfisicalEnvironment, InfisicalIdentity string
	RegistryBucket, RegistryPrefix                                             string
	ClientIDVersion, ClientSecretVersion                                       string
	ConsentExpiresAt                                                           string
	WebhookURL                                                                 string
	Environment, BankKey, LinkKey                                              string
}

// Tokens separates runtime identity minting from static secrets and permits synthetic tests without discovery.
type Tokens interface {
	AccessToken(context.Context) (string, error)
	IdentityToken(context.Context, string) (string, error)
}

// Validate rejects partial configuration and mutable/cross-project credential references before I/O.
func (c Config) Validate() error {
	return c.validateAt(time.Now())
}

// validateAt separates structural validity from optional provider authority, without extending its deadline.
func (c Config) validateAt(now time.Time) error {
	if !c.Enabled {
		if c != (Config{}) {
			return ErrConfiguration
		}
		return nil
	}
	if c.Environment == "production" {
		return c.validateProduction(now)
	}
	if c.Environment != "" && c.Environment != "sandbox" || c.BankKey != "" || c.LinkKey != "" {
		return ErrConfiguration
	}
	// Notification routing is an explicit single-project capability, never arbitrary HTTPS configuration.
	if c.HostedRepair && c.WebhookURL != connections.SandboxTransactionsWebhookURL {
		return ErrConfiguration
	}
	if c.WebhookURL != "" && (c.WebhookURL != connections.SandboxTransactionsWebhookURL || c.ProjectID != "jayburd-bookkeeping-dev" || c.ProjectNumber != "9088794002") {
		return ErrConfiguration
	}
	expiry, e := time.Parse(time.RFC3339, c.ConsentExpiresAt)
	// Hosted Link has session results by default; ordinary SDK enablement is metadata only.
	if !project.MatchString(c.ProjectID) || !number.MatchString(c.ProjectNumber) || !nonzeroUUID(c.InfisicalProject) || !nonzeroUUID(c.InfisicalIdentity) || c.InfisicalEnvironment != "dev" || c.RegistryBucket != c.ProjectID+"-jayburd-dev-credential-registry" || c.RegistryPrefix != "provider-credentials-v1" || e != nil || expiry.IsZero() || expiry.Year() < 1 || expiry.After(now.Add(30*24*time.Hour)) {
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

// New is inert when disabled; approved startup reads only two exact static Secret Manager versions.
// Expired optional enrollment starts with SQL-only operations and no credential or provider I/O.
// It never opens another DB pool, begins a Link session, writes a secret or calls Plaid at startup.
func New(parent context.Context, c Config, db connections.Beginner, transport http.RoundTripper, tokens Tokens) (*connections.Service, error) {
	return newWithClock(parent, c, db, transport, tokens, time.Now)
}

// newWithClock enables deterministic expiry proofs; production always supplies the real clock through New.
func newWithClock(parent context.Context, c Config, db connections.Beginner, transport http.RoundTripper, tokens Tokens, now func() time.Time) (*connections.Service, error) {
	if now == nil || c.validateAt(now()) != nil {
		return nil, ErrConfiguration
	}
	if !c.Enabled {
		return nil, nil
	}
	if parent == nil || db == nil || transport == nil || tokens == nil {
		return nil, ErrConfiguration
	}
	expiry, _ := time.Parse(time.RFC3339, c.ConsentExpiresAt)
	logins := connections.WorkerLogins{Sync: "jayburd-dev-sync@" + c.ProjectID + ".iam", Publisher: "jayburd-dev-publisher@" + c.ProjectID + ".iam"}
	withoutProvider := func() (*connections.Service, error) {
		if c.Environment == "production" {
			return connections.NewProductionWithApproval(db, nil, nil, expiry, now)
		}
		return connections.NewForBookCreatorsWithApproval(db, nil, nil, logins, expiry, now)
	}
	if !expiry.After(now()) {
		return withoutProvider()
	}
	// Every outgoing dependency request rechecks the same approval, including multi-call provider operations.
	transport = approvalTransport{base: transport, deadline: expiry, now: now}
	reader, e := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: c.ProjectNumber, Tokens: tokens, Transport: transport})
	if e != nil {
		return nil, ErrConfiguration
	}
	wrapper, e := bankcredential.NewKMS("projects/"+c.ProjectID+"/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials", tokens, transport)
	if c.Environment == "production" {
		wrapper, e = bankcredential.NewProductionKMS(c.BankKey, tokens, transport)
	}
	if e != nil {
		return nil, ErrConfiguration
	}
	store, e := bankcredential.NewAPIStore(db, wrapper)
	if c.Environment == "production" {
		link, err := bankcredential.NewProductionKMS(c.LinkKey, tokens, transport)
		if err != nil {
			return nil, ErrConfiguration
		}
		store, e = bankcredential.NewProductionAPIStore(db, wrapper, link, bankcredential.ProductionPolicy{BankKey: c.BankKey, LinkKey: c.LinkKey})
	}
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
		if !expiry.After(now()) {
			return withoutProvider()
		}
		return nil, ErrUnavailable
	}
	if !expiry.After(now()) {
		return withoutProvider()
	}
	secret, e := reader.Read(ctx, c.ClientSecretVersion)
	if e != nil || ctx.Err() != nil {
		if !expiry.After(now()) {
			return withoutProvider()
		}
		return nil, ErrUnavailable
	}
	if !expiry.After(now()) {
		return withoutProvider()
	}
	profile := plaidprofile.Sandbox()
	if c.Environment == "production" {
		profile, _ = plaidprofile.New("production")
	}
	webhookProject := ""
	if c.Environment == "production" && c.WebhookURL != "" {
		webhookProject = c.ProjectNumber
	}
	provider, e := connections.NewHTTPProviderForProfile(connections.HTTPConfig{ClientID: connections.NewSecret(string(id)), ClientSecret: connections.NewSecret(string(secret)), ConsentExpiresAt: expiry, WebhookURL: c.WebhookURL, HostedRepair: c.HostedRepair, WebhookProjectNumber: webhookProject}, profile, transport)
	if e != nil {
		if !expiry.After(now()) {
			return withoutProvider()
		}
		return nil, ErrUnavailable
	}
	if c.Environment == "production" {
		return connections.NewProductionWithApproval(db, provider, store, expiry, now)
	}
	return connections.NewForBookCreatorsWithApproval(db, provider, store, logins, expiry, now)
}

// validateProduction requires explicit separated metadata; it never discovers actual resources or values.
func (c Config) validateProduction(now time.Time) error {
	expiry, e := time.Parse(time.RFC3339, c.ConsentExpiresAt)
	// The approved shared project must use its matching numeric identity, never mixed metadata.
	sharedMismatch := (c.ProjectID == "jayburd-bookkeeping-dev") != (c.ProjectNumber == "9088794002")
	if !project.MatchString(c.ProjectID) || sharedMismatch || !number.MatchString(c.ProjectNumber) || !nonzeroUUID(c.InfisicalProject) || !nonzeroUUID(c.InfisicalIdentity) || c.InfisicalEnvironment != "prod" || c.RegistryBucket != c.ProjectID+"-jayburd-prod-credential-registry" || c.RegistryPrefix != "provider-credentials-v1" || e != nil || expiry.IsZero() || expiry.Year() < 1 || expiry.After(now.Add(30*24*time.Hour)) {
		return ErrConfiguration
	}
	if c.WebhookURL != "" {
		expected, e := connections.ProductionTransactionsWebhookURL(c.ProjectNumber)
		if e != nil || c.WebhookURL != expected {
			return ErrConfiguration
		}
	} else if c.HostedRepair {
		return ErrConfiguration
	}
	if c.InfisicalOrigin != "https://us.infisical.com" && c.InfisicalOrigin != "https://eu.infisical.com" && c.InfisicalOrigin != "https://app.infisical.com" {
		return ErrConfiguration
	}
	p := bankcredential.ProductionPolicy{BankKey: c.BankKey, LinkKey: c.LinkKey}
	if !p.Valid() || !strings.HasPrefix(c.BankKey, "projects/"+c.ProjectID+"/") || !strings.HasPrefix(c.LinkKey, "projects/"+c.ProjectID+"/") {
		return ErrConfiguration
	}
	for i, v := range []string{c.ClientIDVersion, c.ClientSecretVersion} {
		name := []string{"jayburd-prod-bank-sync-client-id", "jayburd-prod-bank-sync-client-secret"}[i]
		parts := resource.FindStringSubmatch(v)
		if !strings.HasPrefix(v, "projects/"+c.ProjectNumber+"/secrets/"+name+"/versions/") || len(parts) != 3 || !positiveNumber(parts[2]) {
			return ErrConfiguration
		}
	}
	return nil
}

// positiveNumber excludes overflow and aliases from immutable destination version pins.
func positiveNumber(s string) bool {
	n, e := strconv.ParseInt(s, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == s
}

// approvalTransport denies new dependency requests at expiry without invoking the underlying transport.
type approvalTransport struct {
	base     http.RoundTripper
	deadline time.Time
	now      func() time.Time
}

// RoundTrip never extends approval while a multi-request operation or secret initialization is in progress.
func (t approvalTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !t.deadline.After(t.now()) {
		return nil, ErrUnavailable
	}
	return t.base.RoundTrip(r)
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
