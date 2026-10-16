// Package bankcredential stores only scoped, authenticated ciphertext in PostgreSQL.
package bankcredential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

var ErrUnavailable = errors.New("bank_credential_unavailable")
var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

const profile = "jayburd-bank-aead-v1"

// Binding is server-derived identity authenticated by both local AEAD and the wrapping key.
type Binding struct {
	Entity, Book string
	Reference    connections.CredentialReference
}

// Envelope contains ciphertext and wrapping metadata only, never a data key or plaintext token.
type Envelope struct {
	Profile    string `json:"profile"`
	KeyVersion string `json:"key_version"`
	WrappedKey string `json:"wrapped_key"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// Wrapper holds the remote key capability; implementations must bind the supplied AAD.
type Wrapper interface {
	Wrap(context.Context, []byte, []byte) (string, []byte, error)
	Unwrap(context.Context, string, []byte, []byte) ([]byte, error)
}

// validID rejects zero and malformed scope identities before cryptographic or SQL work.
func validID(s string) bool {
	return idPattern.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}

// aad serializes a fixed-order tuple so field concatenation cannot alias another scope.
func (b Binding) aad() ([]byte, error) {
	r := b.Reference
	n, e := strconv.ParseInt(r.Version, 10, 64)
	if !validID(b.Entity) || !validID(b.Book) || !validID(r.ReferenceID) || e != nil || n < 1 || strconv.FormatInt(n, 10) != r.Version || r.Environment != "sandbox" || !tokenValid(r.ItemID) || len(r.ItemID) > 256 {
		return nil, ErrUnavailable
	}
	return json.Marshal([]string{profile, b.Entity, b.Book, r.ReferenceID, r.Version, r.ItemID, r.Environment})
}

// tokenValid permits the bounded opaque provider grammar without logging rejected material.
func tokenValid(s string) bool {
	if len(s) < 1 || len(s) > 4096 {
		return false
	}
	for i := range s {
		if s[i] < 33 || s[i] > 126 {
			return false
		}
	}
	return true
}

// Seal generates an independent data key and nonce for this immutable credential version.
func Seal(ctx context.Context, w Wrapper, b Binding, token connections.Secret) (Envelope, error) {
	var out Envelope
	aad, e := b.aad()
	if e != nil || w == nil || ctx.Err() != nil {
		return out, ErrUnavailable
	}
	key := make([]byte, 32)
	defer clear(key)
	nonce := make([]byte, 12)
	if _, e = rand.Read(key); e != nil {
		return out, ErrUnavailable
	}
	if _, e = rand.Read(nonce); e != nil {
		return out, ErrUnavailable
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return out, ErrUnavailable
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return out, ErrUnavailable
	}
	version, wrapped, e := w.Wrap(ctx, key, aad)
	if e != nil || version == "" || len(wrapped) < 1 || len(wrapped) > 8192 || ctx.Err() != nil {
		return out, ErrUnavailable
	}
	payloadAAD, _ := json.Marshal([]string{string(aad), version})
	var encrypted []byte
	e = token.WithValue(func(v string) error {
		if !tokenValid(v) {
			return ErrUnavailable
		}
		raw := []byte(v)
		defer clear(raw)
		encrypted = gcm.Seal(nil, nonce, raw, payloadAAD)
		return nil
	})
	if e != nil {
		return out, ErrUnavailable
	}
	return Envelope{profile, version, base64.StdEncoding.EncodeToString(wrapped), base64.StdEncoding.EncodeToString(nonce), base64.StdEncoding.EncodeToString(encrypted)}, nil
}

// Open authenticates the complete binding and envelope before returning a memory-only Secret.
func Open(ctx context.Context, w Wrapper, b Binding, v Envelope) (connections.Secret, error) {
	empty := connections.Secret{}
	aad, e := b.aad()
	if e != nil || w == nil || ctx.Err() != nil || v.Profile != profile || len(v.KeyVersion) > 512 || len(v.WrappedKey) > 11000 || len(v.Nonce) > 16 || len(v.Ciphertext) > 5500 {
		return empty, ErrUnavailable
	}
	wrapped, e := base64.StdEncoding.Strict().DecodeString(v.WrappedKey)
	if e != nil || len(wrapped) < 1 || len(wrapped) > 8192 {
		return empty, ErrUnavailable
	}
	nonce, e := base64.StdEncoding.Strict().DecodeString(v.Nonce)
	if e != nil || len(nonce) != 12 {
		return empty, ErrUnavailable
	}
	ciphertext, e := base64.StdEncoding.Strict().DecodeString(v.Ciphertext)
	if e != nil || len(ciphertext) < 17 || len(ciphertext) > 4112 {
		return empty, ErrUnavailable
	}
	key, e := w.Unwrap(ctx, v.KeyVersion, wrapped, aad)
	defer clear(key)
	if e != nil || len(key) != 32 || ctx.Err() != nil {
		return empty, ErrUnavailable
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return empty, ErrUnavailable
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return empty, ErrUnavailable
	}
	payloadAAD, _ := json.Marshal([]string{string(aad), v.KeyVersion})
	plain, e := gcm.Open(nil, nonce, ciphertext, payloadAAD)
	defer clear(plain)
	if e != nil || !tokenValid(string(plain)) {
		return empty, ErrUnavailable
	}
	return connections.NewSecret(string(plain)), nil
}

// String keeps ciphertext and provider-binding metadata out of incidental diagnostics.
func (Envelope) String() string { return "[encrypted bank credential]" }

// GoString preserves the same diagnostic boundary for detailed formatting.
func (v Envelope) GoString() string { return v.String() }

// keyVersionMatches accepts only numeric immutable versions under the configured wrapping key.
func keyVersionMatches(key, version string) bool {
	prefix := key + "/cryptoKeyVersions/"
	if !strings.HasPrefix(version, prefix) {
		return false
	}
	v := strings.TrimPrefix(version, prefix)
	n, e := strconv.ParseInt(v, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == v
}

// Probe verifies wrapping and decryption before a one-shot provider mutation, without storing any token.
func Probe(ctx context.Context, w Wrapper, binding []byte) error {
	key := make([]byte, 32)
	defer clear(key)
	if _, e := rand.Read(key); e != nil {
		return ErrUnavailable
	}
	version, wrapped, e := w.Wrap(ctx, key, binding)
	if e != nil {
		return ErrUnavailable
	}
	plain, e := w.Unwrap(ctx, version, wrapped, binding)
	defer clear(plain)
	if e != nil || len(plain) != 32 || subtle.ConstantTimeCompare(key, plain) != 1 {
		return ErrUnavailable
	}
	return nil
}
