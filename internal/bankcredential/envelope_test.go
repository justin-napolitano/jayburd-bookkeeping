package bankcredential

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const testKey = "projects/123456789/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials"

// testWrapper independently authenticates wrapped keys and AAD using a fictional KEK.
type testWrapper struct{}

func (testWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	b, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(b)
	return testKey + "/cryptoKeyVersions/1", g.Seal(nil, make([]byte, 12), key, aad), nil
}
func (testWrapper) Unwrap(_ context.Context, version string, raw, aad []byte) ([]byte, error) {
	if version != testKey+"/cryptoKeyVersions/1" {
		return nil, ErrUnavailable
	}
	b, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, make([]byte, 12), raw, aad)
}
func binding() Binding {
	return Binding{"00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002", connections.CredentialReference{ReferenceID: "00000000-0000-4000-8000-000000000003", Version: "1", ItemID: "fictional-item", Environment: "sandbox"}}
}

// TestEnvelopeBinding proves each scope/reference dimension and encrypted field is authenticated.
func TestEnvelopeBinding(t *testing.T) {
	ctx := context.Background()
	b := binding()
	secret := connections.NewSecret("access-sandbox-fictional-private")
	first, e := Seal(ctx, testWrapper{}, b, secret)
	if e != nil {
		t.Fatal(e)
	}
	second, e := Seal(ctx, testWrapper{}, b, secret)
	if e != nil {
		t.Fatal(e)
	}
	if first.Nonce == second.Nonce || first.Ciphertext == second.Ciphertext || first.WrappedKey == second.WrappedKey {
		t.Fatal("randomness reused")
	}
	raw, _ := json.Marshal(first)
	if bytes.Contains(raw, []byte("access-sandbox")) {
		t.Fatal("plaintext serialized")
	}
	got, e := Open(ctx, testWrapper{}, b, first)
	if e != nil {
		t.Fatal(e)
	}
	_ = got.WithValue(func(v string) error {
		if v != "access-sandbox-fictional-private" {
			t.Fatal("round trip mismatch")
		}
		return nil
	})
	for _, field := range []string{"entity", "book", "reference", "version", "item", "environment"} {
		t.Run(field, func(t *testing.T) {
			wrong := b
			switch field {
			case "entity":
				wrong.Entity = b.Book
			case "book":
				wrong.Book = b.Entity
			case "reference":
				wrong.Reference.ReferenceID = b.Entity
			case "version":
				wrong.Reference.Version = "2"
			case "item":
				wrong.Reference.ItemID = "other-item"
			case "environment":
				wrong.Reference.Environment = "production"
			}
			if _, e := Open(ctx, testWrapper{}, wrong, first); e == nil {
				t.Fatal("wrong binding decrypted")
			}
		})
	}
	for _, field := range []string{"profile", "key", "wrapped", "nonce", "ciphertext"} {
		t.Run(field, func(t *testing.T) {
			bad := first
			flip := func(v string) string {
				b, _ := base64.StdEncoding.DecodeString(v)
				b[0] ^= 1
				return base64.StdEncoding.EncodeToString(b)
			}
			switch field {
			case "profile":
				bad.Profile = "other"
			case "key":
				bad.KeyVersion = testKey + "/cryptoKeyVersions/2"
			case "wrapped":
				bad.WrappedKey = flip(bad.WrappedKey)
			case "nonce":
				bad.Nonce = flip(bad.Nonce)
			case "ciphertext":
				bad.Ciphertext = flip(bad.Ciphertext)
			}
			if _, e := Open(ctx, testWrapper{}, b, bad); e == nil {
				t.Fatal("tamper accepted")
			}
		})
	}
	if e := same(ctx, testWrapper{}, b, first, connections.NewSecret("different-token")); e == nil {
		t.Fatal("different immutable token accepted")
	}
	if e := same(ctx, testWrapper{}, b, first, secret); e != nil {
		t.Fatal(e)
	}
	canceled, stop := context.WithCancel(ctx)
	stop()
	if _, e := Seal(canceled, testWrapper{}, b, secret); e == nil {
		t.Fatal("canceled seal")
	}
	if fmt.Sprintf("%#v", first) != "[encrypted bank credential]" {
		t.Fatal("diagnostic boundary")
	}
}
