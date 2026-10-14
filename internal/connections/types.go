// Package connections owns bounded, explicitly authorized provider credential handoffs.
// Database state contains references and verified metadata, never provider tokens.
package connections

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Scope comes from authenticated server identity; SQL rechecks current permissions.
type Scope struct{ Actor, Entity, Book string }

// BeginRequest records explicit Item-wide approval for every affected stream.
type BeginRequest struct {
	Mode               string   `json:"mode"`
	SyncItemID         *string  `json:"sync_item_id"`
	ExpectedGeneration *string  `json:"expected_generation"`
	ApprovedStreamIDs  []string `json:"approved_stream_ids"`
}

// BeginResult contains an ephemeral browser handoff. Do not persist or log it.
type BeginResult struct {
	Attempt   json.RawMessage `json:"attempt"`
	Nonce     string          `json:"nonce"`
	LinkToken string          `json:"link_token,omitempty"`
	ExpiresAt string          `json:"expires_at,omitempty"`
}

// ExecuteRequest carries memory-only Link return material; rotate/remove need only nonce.
type ExecuteRequest struct {
	Nonce       string `json:"nonce"`
	LinkToken   string `json:"link_token,omitempty"`
	PublicToken string `json:"public_token,omitempty"`
}

// Binding requires explicit account, currency, source and evidence selection.
type Binding struct {
	AccountID               string  `json:"account_id"`
	LedgerAccountID         string  `json:"ledger_account_id"`
	CurrencyVersionID       string  `json:"currency_version_id"`
	SourceAccountID         *string `json:"source_account_id"`
	ExpectedBindingRevision *string `json:"expected_binding_revision"`
	BindingDate             string  `json:"binding_date"`
	BindingEvidenceRef      string  `json:"binding_evidence_ref"`
}

// BindingRequest explicitly grants the selected source capabilities; it never posts journals.
type BindingRequest struct {
	ConsentGrants bool      `json:"consent_grants"`
	Bindings      []Binding `json:"bindings"`
}

// Secret prevents accidental formatting or JSON serialization of credential material.
type Secret struct{ value string }

// NewSecret takes ownership of an in-memory credential value.
func NewSecret(value string) Secret { return Secret{value: value} }

// String redacts routine formatting.
func (Secret) String() string { return "[redacted]" }

// GoString redacts diagnostic formatting.
func (Secret) GoString() string { return "[redacted]" }

// MarshalJSON rejects accidental credential serialization.
func (Secret) MarshalJSON() ([]byte, error) { return nil, errors.New("secret_serialization_forbidden") }

// CredentialReference names one immutable version, scoped to an expected provider Item.
type CredentialReference struct{ ReferenceID, Version, ItemID, Environment string }

// StoredCredential proves the exact addressed version and Item, not merely successful storage.
type StoredCredential struct {
	Reference CredentialReference
	Token     Secret
}

// SecretStore implements immutable version writes and exact readback without discovering credentials.
type SecretStore interface {
	Put(context.Context, CredentialReference, Secret) error
	Read(context.Context, CredentialReference) (StoredCredential, error)
}

// Account is provider-verified metadata. Unsupported accounts remain visible but cannot bind.
type Account struct {
	AccountID string  `json:"account_id"`
	Type      string  `json:"type"`
	Currency  *string `json:"iso_currency_code"`
}

// Manifest is server-verified Item metadata, never client Link metadata or transaction balances.
type Manifest struct {
	ItemID           string    `json:"item_id"`
	Accounts         []Account `json:"accounts"`
	ConsentExpiresAt string    `json:"consent_expires_at"`
}

// LinkRequest selects a new or existing Item without exposing tokens to SQL.
type LinkRequest struct {
	ActorID, Mode string
	AccessToken   Secret
}

// LinkResult is an ephemeral provider token with a bounded lifetime.
type LinkResult struct {
	Token     Secret
	ExpiresAt time.Time
}

// ExchangeResult retains the provider Item identity beside its memory-only token.
type ExchangeResult struct {
	ItemID      string
	AccessToken Secret
}

// Provider is a context-bound transport, never an authority to activate sources.
// VerifySession requires completed Hosted-Link-compatible association; unavailable evidence rejects.
type Provider interface {
	CreateLink(context.Context, LinkRequest) (LinkResult, error)
	VerifySession(context.Context, Secret, Secret, string) error
	Exchange(context.Context, Secret) (ExchangeResult, error)
	Manifest(context.Context, Secret) (Manifest, error)
	Rotate(context.Context, Secret) (ExchangeResult, error)
	Remove(context.Context, Secret) error
}

// Failure exposes only fixed public classifications, never wrapped transport diagnostics.
type Failure struct{ Code string }

// Error returns only the safe public classification.
func (f *Failure) Error() string { return f.Code }

// failed constructs a redacted failure without wrapping infrastructure details.
func failed(code string) error { return &Failure{Code: code} }
