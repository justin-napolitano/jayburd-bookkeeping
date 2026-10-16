package bankcredential

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// DB exposes only scoped serializable function calls, not a credential-table API.
type DB interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// APIStore binds each operation to the verified caller and selected book at service dispatch.
type APIStore struct {
	db      DB
	wrapper Wrapper
}
type scopedStore struct {
	parent *APIStore
	scope  connections.Scope
}

// NewAPIStore composes an explicitly scoped ciphertext store without startup I/O.
func NewAPIStore(db DB, w Wrapper) (*APIStore, error) {
	if db == nil || w == nil {
		return nil, ErrUnavailable
	}
	return &APIStore{db, w}, nil
}

// ForScope never trusts browser-supplied identity; the service passes its verified scope.
func (s *APIStore) ForScope(scope connections.Scope) connections.SecretStore {
	return &scopedStore{s, scope}
}

// Put without caller scope is deliberately unavailable.
func (*APIStore) Put(context.Context, connections.CredentialReference, connections.Secret) error {
	return ErrUnavailable
}

// Read without caller scope is deliberately unavailable.
func (*APIStore) Read(context.Context, connections.CredentialReference) (connections.StoredCredential, error) {
	return connections.StoredCredential{}, ErrUnavailable
}

// call contains SQL only; no KMS or provider work occurs inside a transaction.
func call(parent context.Context, db DB, query string, args ...any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	for i := 0; i < 3; i++ {
		tx, e := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			return nil, ErrUnavailable
		}
		var raw []byte
		_, e = tx.Exec(ctx, "SET LOCAL statement_timeout='3s'; SET LOCAL lock_timeout='1s'")
		if e == nil {
			e = tx.QueryRow(ctx, query, args...).Scan(&raw)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		cleanup, stop := context.WithTimeout(context.Background(), time.Second)
		_ = tx.Rollback(cleanup)
		stop()
		if e == nil {
			if len(raw) == 0 || len(raw) > 32768 || !json.Valid(raw) {
				return nil, ErrUnavailable
			}
			return raw, nil
		}
		var p *pgconn.PgError
		if !errors.As(e, &p) || (p.Code != "40001" && p.Code != "40P01") {
			return nil, ErrUnavailable
		}
	}
	return nil, ErrUnavailable
}

// lookup requires SQL to authorize the reference before any key-unwrapping request.
func (s *scopedStore) lookup(ctx context.Context, r connections.CredentialReference) (*Envelope, error) {
	b := Binding{s.scope.Entity, s.scope.Book, r}
	if _, e := b.aad(); e != nil {
		return nil, ErrUnavailable
	}
	raw, e := call(ctx, s.parent.db, "SELECT integration.get_bank_credential($1,$2,$3,$4,$5,$6,$7)::text", s.scope.Actor, s.scope.Entity, s.scope.Book, r.ReferenceID, r.Version, r.ItemID, r.Environment)
	if e != nil {
		return nil, e
	}
	var out struct {
		Envelope *Envelope `json:"envelope"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return nil, ErrUnavailable
	}
	return out.Envelope, nil
}

// same checks a concurrent immutable winner without persisting plaintext hashes.
func same(ctx context.Context, w Wrapper, b Binding, v Envelope, token connections.Secret) error {
	old, e := Open(ctx, w, b, v)
	if e != nil {
		return e
	}
	equal := false
	_ = old.WithValue(func(a string) error {
		return token.WithValue(func(z string) error { equal = subtle.ConstantTimeCompare([]byte(a), []byte(z)) == 1; return nil })
	})
	if !equal {
		return ErrUnavailable
	}
	return nil
}

// Put verifies existing or newly committed ciphertext by decrypting it under the exact binding.
func (s *scopedStore) Put(ctx context.Context, r connections.CredentialReference, token connections.Secret) error {
	b := Binding{s.scope.Entity, s.scope.Book, r}
	old, e := s.lookup(ctx, r)
	if e != nil {
		return e
	}
	if old != nil {
		return same(ctx, s.parent.wrapper, b, *old, token)
	}
	v, e := Seal(ctx, s.parent.wrapper, b, token)
	if e != nil {
		return e
	}
	body, e := json.Marshal(v)
	if e != nil {
		return ErrUnavailable
	}
	raw, e := call(ctx, s.parent.db, "SELECT integration.put_bank_credential($1,$2,$3,$4,$5,$6,$7,$8)::text", s.scope.Actor, s.scope.Entity, s.scope.Book, r.ReferenceID, r.Version, r.ItemID, r.Environment, string(body))
	if e != nil {
		return e
	}
	var out struct {
		Envelope *Envelope `json:"envelope"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Envelope == nil {
		return ErrUnavailable
	}
	return same(ctx, s.parent.wrapper, b, *out.Envelope, token)
}

// Read supports committed-write recovery without performing another external mutation.
func (s *scopedStore) Read(ctx context.Context, r connections.CredentialReference) (connections.StoredCredential, error) {
	v, e := s.lookup(ctx, r)
	if e != nil || v == nil {
		return connections.StoredCredential{}, ErrUnavailable
	}
	secret, e := Open(ctx, s.parent.wrapper, Binding{s.scope.Entity, s.scope.Book, r}, *v)
	if e != nil {
		return connections.StoredCredential{}, e
	}
	return connections.StoredCredential{Reference: r, Token: secret}, nil
}

// Resolver combines authorized database ciphertext with separately managed static client credentials.
type Resolver struct {
	db      DB
	wrapper Wrapper
	static  func(context.Context) (acquisition.ProviderCredentials, error)
}

// NewResolver performs no I/O; only a current scoped lease can subsequently unwrap a token.
func NewResolver(db DB, w Wrapper, static func(context.Context) (acquisition.ProviderCredentials, error)) (*Resolver, error) {
	if db == nil || w == nil || static == nil {
		return nil, ErrUnavailable
	}
	return &Resolver{db, w, static}, nil
}

// Resolve refuses legacy unscoped token resolution; it cannot fall back to an old vault mapping.
func (*Resolver) Resolve(context.Context, acquisition.CredentialReference) (acquisition.ProviderCredentials, error) {
	return acquisition.ProviderCredentials{}, ErrUnavailable
}

// ResolveScoped derives stored identity from sync_context before decrypting, then checks the runner's pin.
func (r *Resolver) ResolveScoped(ctx context.Context, s acquisition.Scope, setID, fence string, expected acquisition.CredentialReference) (acquisition.ProviderCredentials, error) {
	var empty acquisition.ProviderCredentials
	if !validID(s.EntityID) || !validID(s.BookID) || !validID(s.StreamID) || !validID(setID) {
		return empty, ErrUnavailable
	}
	raw, e := call(ctx, r.db, "SELECT integration.worker_bank_credential($1,$2,$3,$4,$5)::text", s.EntityID, s.BookID, s.StreamID, setID, fence)
	if e != nil {
		return empty, e
	}
	var out struct {
		Envelope    Envelope `json:"envelope"`
		Ref         string   `json:"credential_ref"`
		Version     string   `json:"credential_version"`
		Item        string   `json:"item_id"`
		Environment string   `json:"environment"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Ref != expected.ReferenceID || out.Version != expected.Version || out.Item != expected.ItemID || out.Environment != expected.Environment {
		return empty, ErrUnavailable
	}
	ref := connections.CredentialReference{ReferenceID: out.Ref, Version: out.Version, ItemID: out.Item, Environment: out.Environment}
	token, e := Open(ctx, r.wrapper, Binding{s.EntityID, s.BookID, ref}, out.Envelope)
	if e != nil {
		return empty, e
	}
	result, e := r.static(ctx)
	if e != nil || !tokenValid(result.ClientID) || !tokenValid(result.Secret) {
		return empty, ErrUnavailable
	}
	_ = token.WithValue(func(v string) error { result.AccessToken = v; return nil })
	return result, nil
}

// Check prevents known KMS permission/configuration failures from consuming a public token.
func (s *scopedStore) Check(ctx context.Context) error {
	if !validID(s.scope.Actor) || !validID(s.scope.Entity) || !validID(s.scope.Book) {
		return ErrUnavailable
	}
	// Check the additive SQL surface before a token can be consumed, even outside the hosted release pipeline.
	raw, e := call(ctx, s.parent.db, `SELECT jsonb_build_object('ready',
 coalesce(has_function_privilege(current_user,to_regprocedure('integration.get_bank_credential(uuid,uuid,uuid,uuid,bigint,text,text)'),'EXECUTE'),false)
 AND coalesce(has_function_privilege(current_user,to_regprocedure('integration.put_bank_credential(uuid,uuid,uuid,uuid,bigint,text,text,text)'),'EXECUTE'),false)
 AND to_regclass('integration.bank_credential') IS NOT NULL)::text`)
	var status struct {
		Ready bool `json:"ready"`
	}
	if e != nil || json.Unmarshal(raw, &status) != nil || !status.Ready {
		return ErrUnavailable
	}
	aad, _ := json.Marshal([]string{"jayburd-bank-preflight-v1", s.scope.Entity, s.scope.Book})
	return Probe(ctx, s.parent.wrapper, aad)
}
