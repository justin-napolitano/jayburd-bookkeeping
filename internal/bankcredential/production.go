package bankcredential

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// ProductionPolicy is immutable runtime metadata; current SQL still owns scope and permissions.
type ProductionPolicy struct{ BankKey, LinkKey string }

// Valid confines both capabilities to explicit canonical nondevelopment resources.
func (p ProductionPolicy) Valid() bool {
	return ValidProductionKey(p.BankKey) && ValidProductionKey(p.LinkKey) && p.BankKey != p.LinkKey
}

// NewProductionAPIStore keeps bank and Link capabilities separate and performs no startup I/O.
func NewProductionAPIStore(db DB, bank, link Wrapper, p ProductionPolicy) (*APIStore, error) {
	if db == nil || bank == nil || link == nil || !p.Valid() {
		return nil, ErrUnavailable
	}
	return &APIStore{db: db, wrapper: bank, linkWrapper: link, policy: &p}, nil
}

// NewProductionResolver requires current addressed SQL evidence before any unwrap or static read.
func NewProductionResolver(db DB, bank Wrapper, p ProductionPolicy, static func(context.Context) (acquisition.ProviderCredentials, error)) (*Resolver, error) {
	if db == nil || bank == nil || !p.Valid() || static == nil {
		return nil, ErrUnavailable
	}
	return &Resolver{db: db, wrapper: bank, policy: &p, static: static}, nil
}

// environment never promotes a legacy resolver when a caller supplies Production metadata.
func (r *Resolver) environment() string {
	if r.policy != nil {
		return "production"
	}
	return "sandbox"
}

// AuthorizeProvider compares current SQL authority to both exact runtime keys before provider I/O.
func (s *scopedStore) AuthorizeProvider(ctx context.Context) error {
	if s.parent.policy == nil {
		return nil
	}
	raw, e := call(ctx, s.parent.db, "SELECT integration.connection_provider_context($1,$2,$3)::text", s.scope.Actor, s.scope.Entity, s.scope.Book)
	var c struct {
		Environment string `json:"environment"`
		Origin      string `json:"data_origin"`
		Revision    string `json:"scope_revision"`
		BankKey     string `json:"bank_key"`
		LinkKey     string `json:"link_key"`
	}
	if e != nil || json.Unmarshal(raw, &c) != nil || c.Environment != "production" || c.Origin != "provider" || !positiveVersion(c.Revision) || c.BankKey != s.parent.policy.BankKey || c.LinkKey != s.parent.policy.LinkKey {
		return ErrUnavailable
	}
	return nil
}

// positiveVersion shares the existing canonical numeric-version grammar without parsing financial values.
func positiveVersion(v string) bool {
	n, e := strconv.ParseInt(v, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == v
}

// authorizeReference prevents legacy stores or mismatched runtimes from reaching addressed token SQL.
func (s *scopedStore) authorizeReference(ctx context.Context, r connections.CredentialReference) error {
	want := "sandbox"
	if s.parent.policy != nil {
		want = "production"
	}
	if r.Environment != want {
		return ErrUnavailable
	}
	return s.AuthorizeProvider(ctx)
}

// bankEnvelope checks the exact configured key before its wrapper is allowed to authenticate.
func (s *scopedStore) bankEnvelope(v *Envelope) error {
	if v != nil && s.parent.policy != nil && !keyVersionMatches(s.parent.policy.BankKey, v.KeyVersion) {
		return ErrUnavailable
	}
	return nil
}

// environment keeps the legacy Link identity unchanged without a Production policy.
func (s *scopedStore) environment() string {
	if s.parent.policy != nil {
		return "production"
	}
	return "sandbox"
}

// linkEnvelope denies substitution across the separately configured wrapping capabilities.
func (s *scopedStore) linkEnvelope(v Envelope) error {
	if s.parent.policy != nil && !keyVersionMatches(s.parent.policy.LinkKey, v.KeyVersion) {
		return ErrUnavailable
	}
	return nil
}
