package providersecrets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// Slot names one precreated destination container and a create-only Infisical key; it is never reused.
type Slot struct{ ID, InfisicalKey, GCPSecret string }

// StoreConfig fixes the Sandbox writer's source namespace and finite Terraform-provisioned capacity.
type StoreConfig struct {
	InfisicalOrigin, ProjectID, Environment, IdentityID string
	IdentityTokens                                      IdentityTokenSource
	Transport                                           http.RoundTripper
	Slots                                               []Slot
}

// Store implements the connection writer; acquisition receives only Resolver.
type Store struct {
	config   StoreConfig
	registry *GCSRegistry
	reader   *GCPReader
	http     *http.Client
}

// reservation is an immutable one-shot source-create intent, including an opaque payload digest.
type reservation struct {
	Reference                            connections.CredentialReference `json:"reference"`
	Slot                                 Slot                            `json:"slot"`
	Origin, Project, Environment, Digest string
}

// sourceValue contains source data only in trusted adapter memory.
type sourceValue struct {
	ID, Workspace, Environment, SecretKey, SecretValue, SecretPath string
	Version                                                        json.Number
}

// NewStore validates the bounded pool and freezes its order so concurrent reservation is deterministic.
func NewStore(c StoreConfig, g *GCSRegistry, r *GCPReader) (*Store, error) {
	if g == nil || r == nil || c.Transport == nil || c.IdentityTokens == nil || !uuid.MatchString(c.ProjectID) || !uuid.MatchString(c.IdentityID) || c.Environment != "dev" || len(c.Slots) != 8 {
		return nil, fail("secret_configuration")
	}
	if c.InfisicalOrigin != "https://us.infisical.com" && c.InfisicalOrigin != "https://eu.infisical.com" && c.InfisicalOrigin != "https://app.infisical.com" {
		return nil, fail("secret_configuration")
	}
	c.Slots = append([]Slot(nil), c.Slots...)
	sort.Slice(c.Slots, func(i, j int) bool { return c.Slots[i].ID < c.Slots[j].ID })
	seen := map[string]bool{}
	for i, s := range c.Slots {
		expected := "slot-0" + strconv.Itoa(i)
		if s.ID != expected || s.InfisicalKey != expected || !r.resource(s.GCPSecret, false) || s.GCPSecret != "projects/"+r.config.ProjectID+"/secrets/jayburd-dev-bank-sync-"+expected || seen[s.GCPSecret] {
			return nil, fail("secret_configuration")
		}
		seen[s.GCPSecret] = true
	}
	return &Store{c, g, r, client(c.Transport)}, nil
}

// auth uses audience-bound ID-token login only for the separate source writer identity.
func (s *Store) auth(ctx context.Context) (string, error) {
	jwt, e := s.config.IdentityTokens.IdentityToken(ctx, s.config.IdentityID)
	if e != nil || !opaque.MatchString(jwt) {
		return "", fail("secret_auth_failed")
	}
	// The login endpoint does not use bearer authentication; JWT remains request-body only.
	form := url.Values{"identityId": {s.config.IdentityID}, "jwt": {jwt}}.Encode()
	q, e := http.NewRequestWithContext(ctx, "POST", s.config.InfisicalOrigin+"/api/v1/auth/gcp-auth/login", stringsReader(form))
	if e != nil {
		return "", fail("secret_auth_failed")
	}
	q.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	raw, e := loginResponse(s.http, q)
	if e != nil {
		return "", e
	}
	var v struct {
		AccessToken string `json:"accessToken"`
		TokenType   string `json:"tokenType"`
		ExpiresIn   int    `json:"expiresIn"`
	}
	if e = strict(raw, &v); e != nil {
		return "", e
	}
	if !opaque.MatchString(v.AccessToken) || v.TokenType != "Bearer" || v.ExpiresIn < 1 || v.ExpiresIn > 900 {
		return "", fail("secret_auth_failed")
	}
	return v.AccessToken, nil
}

// source performs only exact-key create or recovery lookup, never update/delete/list/import.
func (s *Store) source(ctx context.Context, slot Slot, payload *string) (sourceValue, error) {
	var out struct {
		Secret sourceValue `json:"secret"`
	}
	token, e := s.auth(ctx)
	if e != nil {
		return out.Secret, e
	}
	endpoint := s.config.InfisicalOrigin + "/api/v4/secrets/" + slot.InfisicalKey
	method := "GET"
	var body []byte
	if payload != nil {
		method = "POST"
		body, _ = json.Marshal(map[string]any{"projectId": s.config.ProjectID, "environment": s.config.Environment, "secretPath": "/bank-sync", "secretValue": *payload, "type": "shared", "skipMultilineEncoding": true})
	} else {
		endpoint += "?" + url.Values{"projectId": {s.config.ProjectID}, "environment": {s.config.Environment}, "secretPath": {"/bank-sync"}, "type": {"shared"}, "expandSecretReferences": {"false"}, "includeImports": {"false"}, "viewSecretValue": {"true"}}.Encode()
	}
	b, _, e := request(ctx, s.http, method, endpoint, token, "application/json", body)
	if e != nil {
		return out.Secret, e
	}
	if e = strict(b, &out); e != nil {
		return out.Secret, e
	}
	v := out.Secret
	if !uuid.MatchString(v.ID) || v.Workspace != s.config.ProjectID || v.Environment != s.config.Environment || v.SecretKey != slot.InfisicalKey || !positive(string(v.Version)) {
		return sourceValue{}, fail("secret_source_mismatch")
	}
	return v, nil
}

// validateReservation prevents changing namespace or expected payload across process recovery.
func (s *Store) validateReservation(r reservation, slot Slot, ref connections.CredentialReference) error {
	if r.Reference != ref || r.Slot != slot || r.Origin != s.config.InfisicalOrigin || r.Project != s.config.ProjectID || r.Environment != s.config.Environment || !hash.MatchString(r.Digest) {
		return fail("secret_reservation_conflict")
	}
	return nil
}

// reserve commits create intent before external creation; existing/unknown intent never grants another POST.
func (s *Store) reserve(ctx context.Context, ref connections.CredentialReference, digest string) (reservation, bool, error) {
	for _, slot := range s.config.Slots {
		key := "slots/" + slot.ID + ".json"
		var old reservation
		e := s.registry.get(ctx, key, &old)
		if e == nil {
			if old.Reference == ref {
				if e = s.validateReservation(old, slot, ref); e != nil {
					return reservation{}, false, e
				}
				if old.Digest != digest {
					return reservation{}, false, fail("secret_reservation_conflict")
				}
				return old, false, nil
			}
			continue
		}
		if code(e) != "secret_absent" {
			return reservation{}, false, e
		}
		r := reservation{ref, slot, s.config.InfisicalOrigin, s.config.ProjectID, s.config.Environment, digest}
		e = s.registry.create(ctx, key, r)
		if e == nil {
			return r, true, nil
		}
		if code(e) != "secret_conflict" {
			return reservation{}, false, fail("secret_reservation_unknown")
		}
		if e = s.registry.get(ctx, key, &old); e != nil {
			return reservation{}, false, fail("secret_reservation_unknown")
		}
		if old.Reference == ref {
			if e = s.validateReservation(old, slot, ref); e != nil {
				return reservation{}, false, e
			}
			if old.Digest != digest {
				return reservation{}, false, fail("secret_reservation_conflict")
			}
			return old, false, nil
		}
	}
	return reservation{}, false, fail("secret_pool_exhausted")
}

// Put creates at most once after durable intent and an empty precreated destination check.
func (s *Store) Put(ctx context.Context, ref connections.CredentialReference, secret connections.Secret) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if !validReference(ref) {
		return fail("secret_reference_invalid")
	}
	var raw []byte
	e := secret.WithValue(func(v string) error {
		if !opaque.MatchString(v) {
			return fail("secret_binding_invalid")
		}
		raw, _ = json.Marshal(envelope{"jayburd-provider-credential-v1", ref, v})
		return nil
	})
	if e != nil {
		return e
	}
	sum := sha256.Sum256(raw)
	r, fresh, e := s.reserve(ctx, ref, hex.EncodeToString(sum[:]))
	if e != nil {
		return e
	}
	if !fresh {
		_, e = s.Read(ctx, ref)
		return e
	}
	versions, e := s.reader.versions(ctx, r.Slot.GCPSecret)
	if e != nil || len(versions) != 0 {
		return fail("secret_destination_not_empty")
	}
	payload := string(raw)
	v, e := s.source(ctx, r.Slot, &payload)
	if e != nil {
		return fail("secret_create_unknown")
	}
	if _, e = decodeEnvelope([]byte(v.SecretValue), ref, r.Digest); e != nil {
		return e
	}
	return s.propagate(ctx, r, v)
}

// propagate pins an observed numeric destination only when every visible version has identical expected bytes.
func (s *Store) propagate(ctx context.Context, r reservation, v sourceValue) error {
	if _, e := decodeEnvelope([]byte(v.SecretValue), r.Reference, r.Digest); e != nil {
		return e
	}
	versions, e := s.reader.versions(ctx, r.Slot.GCPSecret)
	if e != nil {
		return e
	}
	if len(versions) == 0 {
		return fail("secret_propagation_pending")
	}
	sort.Slice(versions, func(i, j int) bool { return versionNumber(versions[i]) < versionNumber(versions[j]) })
	for _, name := range versions {
		b, e := s.reader.Read(ctx, name)
		if e != nil {
			return e
		}
		if _, e = decodeEnvelope(b, r.Reference, r.Digest); e != nil {
			return e
		}
	}
	return s.registry.pin(ctx, Mapping{r.Reference, v.ID, string(v.Version), versions[0], r.Digest})
}

// Read recovers propagation without repeating source creation, then reads the pinned Google payload.
func (s *Store) Read(ctx context.Context, ref connections.CredentialReference) (connections.StoredCredential, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var out connections.StoredCredential
	if !validReference(ref) {
		return out, fail("secret_reference_invalid")
	}
	m, e := s.registry.Lookup(ctx, ref)
	if e != nil {
		if code(e) != "secret_absent" {
			return out, e
		}
		found := false
		for _, slot := range s.config.Slots {
			var r reservation
			e = s.registry.get(ctx, "slots/"+slot.ID+".json", &r)
			if code(e) == "secret_absent" {
				continue
			}
			if e != nil {
				return out, e
			}
			if r.Reference != ref {
				continue
			}
			if e = s.validateReservation(r, slot, ref); e != nil {
				return out, e
			}
			found = true
			v, e := s.source(ctx, slot, nil)
			if e != nil {
				return out, fail("secret_recovery_pending")
			}
			if e = s.propagate(ctx, r, v); e != nil {
				return out, e
			}
			break
		}
		if !found {
			return out, fail("secret_mapping_pending")
		}
		m, e = s.registry.Lookup(ctx, ref)
		if e != nil {
			return out, fail("secret_mapping_pending")
		}
	}
	b, e := s.reader.Read(ctx, m.GCPVersion)
	if e != nil {
		return out, e
	}
	token, e := decodeEnvelope(b, ref, m.PayloadSHA256)
	if e != nil {
		return out, e
	}
	return connections.StoredCredential{Reference: ref, Token: token}, nil
}

// String prevents routine formatting of a source response from exposing its private envelope.
func (sourceValue) String() string { return "[redacted source value]" }

// GoString prevents diagnostic Go formatting from exposing the source payload.
func (v sourceValue) GoString() string { return v.String() }
