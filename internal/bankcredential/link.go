package bankcredential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const linkProfile = "jayburd-hosted-link-aead-v1"

// linkAAD binds a recoverable Link token to the authenticated actor and one attempt.
func linkAAD(scope connections.Scope, id string) ([]byte, error) {
	if !validID(scope.Actor) || !validID(scope.Entity) || !validID(scope.Book) || !validID(id) {
		return nil, ErrUnavailable
	}
	return json.Marshal([]string{linkProfile, scope.Actor, scope.Entity, scope.Book, id})
}

// sealLink uses a distinct profile and AAD from long-lived bank credentials.
func sealLink(ctx context.Context, w Wrapper, scope connections.Scope, id string, token connections.Secret) (Envelope, error) {
	var empty Envelope
	aad, err := linkAAD(scope, id)
	if err != nil || w == nil || ctx.Err() != nil {
		return empty, ErrUnavailable
	}
	key, nonce := make([]byte, 32), make([]byte, 12)
	defer clear(key)
	if _, err = rand.Read(key); err != nil {
		return empty, ErrUnavailable
	}
	if _, err = rand.Read(nonce); err != nil {
		return empty, ErrUnavailable
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return empty, ErrUnavailable
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return empty, ErrUnavailable
	}
	version, wrapped, err := w.Wrap(ctx, key, aad)
	if err != nil || version == "" || len(wrapped) < 1 || len(wrapped) > 8192 {
		return empty, ErrUnavailable
	}
	payloadAAD, _ := json.Marshal([]string{string(aad), version})
	var encrypted []byte
	err = token.WithValue(func(value string) error {
		if !tokenValid(value) {
			return ErrUnavailable
		}
		raw := []byte(value)
		defer clear(raw)
		encrypted = gcm.Seal(nil, nonce, raw, payloadAAD)
		return nil
	})
	if err != nil {
		return empty, ErrUnavailable
	}
	return Envelope{linkProfile, version, base64.StdEncoding.EncodeToString(wrapped), base64.StdEncoding.EncodeToString(nonce), base64.StdEncoding.EncodeToString(encrypted)}, nil
}

// openLink authenticates the attempt binding before a server-side Plaid status lookup.
func openLink(ctx context.Context, w Wrapper, scope connections.Scope, id string, value Envelope) (connections.Secret, error) {
	var empty connections.Secret
	aad, err := linkAAD(scope, id)
	if err != nil || w == nil || ctx.Err() != nil || value.Profile != linkProfile || len(value.KeyVersion) > 512 || len(value.WrappedKey) > 11000 || len(value.Nonce) > 16 || len(value.Ciphertext) > 5500 {
		return empty, ErrUnavailable
	}
	wrapped, err := base64.StdEncoding.Strict().DecodeString(value.WrappedKey)
	if err != nil || len(wrapped) < 1 || len(wrapped) > 8192 {
		return empty, ErrUnavailable
	}
	nonce, err := base64.StdEncoding.Strict().DecodeString(value.Nonce)
	if err != nil || len(nonce) != 12 {
		return empty, ErrUnavailable
	}
	ciphertext, err := base64.StdEncoding.Strict().DecodeString(value.Ciphertext)
	if err != nil || len(ciphertext) < 17 || len(ciphertext) > 4112 {
		return empty, ErrUnavailable
	}
	key, err := w.Unwrap(ctx, value.KeyVersion, wrapped, aad)
	defer clear(key)
	if err != nil || len(key) != 32 {
		return empty, ErrUnavailable
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return empty, ErrUnavailable
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return empty, ErrUnavailable
	}
	payloadAAD, _ := json.Marshal([]string{string(aad), value.KeyVersion})
	plain, err := gcm.Open(nil, nonce, ciphertext, payloadAAD)
	defer clear(plain)
	if err != nil || !tokenValid(string(plain)) {
		return empty, ErrUnavailable
	}
	return connections.NewSecret(string(plain)), nil
}

// PutLink stores only authenticated ciphertext and checks a concurrent immutable winner.
func (s *scopedStore) PutLink(ctx context.Context, id string, token connections.Secret) error {
	value, err := sealLink(ctx, s.parent.wrapper, s.scope, id, token)
	if err != nil {
		return err
	}
	body, err := json.Marshal(value)
	if err != nil {
		return ErrUnavailable
	}
	raw, err := call(ctx, s.parent.db, "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text", s.scope.Actor, s.scope.Entity, s.scope.Book, id, string(body))
	if err != nil {
		return err
	}
	var result struct {
		Envelope Envelope `json:"envelope"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return ErrUnavailable
	}
	stored, err := openLink(ctx, s.parent.wrapper, s.scope, id, result.Envelope)
	if err != nil {
		return err
	}
	match := false
	_ = stored.WithValue(func(a string) error {
		return token.WithValue(func(b string) error { match = subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1; return nil })
	})
	if !match {
		return ErrUnavailable
	}
	return nil
}

// ReadLink reauthorizes the exact attempt in SQL before unwrapping its token.
func (s *scopedStore) ReadLink(ctx context.Context, id string) (connections.Secret, error) {
	var empty connections.Secret
	if _, err := linkAAD(s.scope, id); err != nil {
		return empty, err
	}
	raw, err := call(ctx, s.parent.db, "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", s.scope.Actor, s.scope.Entity, s.scope.Book, id)
	if err != nil {
		return empty, err
	}
	var result struct {
		Envelope *Envelope `json:"envelope"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Envelope == nil {
		return empty, ErrUnavailable
	}
	return openLink(ctx, s.parent.wrapper, s.scope, id, *result.Envelope)
}
