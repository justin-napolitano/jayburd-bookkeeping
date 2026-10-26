package bankcredential

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// ReadRepair without verified caller scope cannot acquire any SQL or KMS authority.
func (*APIStore) ReadRepair(context.Context, string, connections.CredentialReference) (connections.StoredCredential, error) {
	return connections.StoredCredential{}, ErrUnavailable
}

// ReadRepair obtains the existing ciphertext through the exact-attempt SQL purpose gate before unwrapping.
// SQL owns live scope/proof/claim/consent checks; addressed metadata and unchanged AEAD bind the result.
func (s *scopedStore) ReadRepair(ctx context.Context, attempt string, expected connections.CredentialReference) (connections.StoredCredential, error) {
	var empty connections.StoredCredential
	b := Binding{s.scope.Entity, s.scope.Book, expected}
	// Production owned repair remains a separately qualified SQL/runtime capability.
	if expected.Environment != "sandbox" || s.authorizeReference(ctx, expected) != nil {
		return empty, ErrUnavailable
	}
	if !validID(s.scope.Actor) || !validID(attempt) || ctx.Err() != nil {
		return empty, ErrUnavailable
	}
	if _, e := b.aad(); e != nil {
		return empty, ErrUnavailable
	}
	raw, e := call(ctx, s.parent.db, "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text",
		s.scope.Actor, s.scope.Entity, s.scope.Book, attempt)
	if e != nil {
		return empty, ErrUnavailable
	}
	var out struct {
		Envelope    *Envelope `json:"envelope"`
		Reference   string    `json:"credential_ref"`
		Version     string    `json:"credential_version"`
		Item        string    `json:"item_id"`
		Environment string    `json:"environment"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil || out.Envelope == nil {
		return empty, ErrUnavailable
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return empty, ErrUnavailable
	}
	ref := connections.CredentialReference{ReferenceID: out.Reference, Version: out.Version, ItemID: out.Item, Environment: out.Environment}
	if ref != expected {
		return empty, ErrUnavailable
	}
	token, e := Open(ctx, s.parent.wrapper, b, *out.Envelope)
	if e != nil {
		return empty, ErrUnavailable
	}
	return connections.StoredCredential{Reference: ref, Token: token}, nil
}
