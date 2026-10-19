package bankcredential

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

// TestHostedLinkEnvelope rejects cross-actor, cross-attempt and ciphertext substitution.
func TestHostedLinkEnvelope(t *testing.T) {
	ctx := context.Background()
	scope := connections.Scope{Actor: "00000000-0000-4000-8000-000000000001", Entity: "00000000-0000-4000-8000-000000000002", Book: "00000000-0000-4000-8000-000000000003"}
	id := "00000000-0000-4000-8000-000000000004"
	token := connections.NewSecret("link-sandbox-fictional")
	first, err := sealLink(ctx, testWrapper{}, scope, id, token)
	if err != nil {
		t.Fatal(err)
	}
	second, err := sealLink(ctx, testWrapper{}, scope, id, token)
	if err != nil {
		t.Fatal(err)
	}
	if first.Nonce == second.Nonce || first.WrappedKey == second.WrappedKey {
		t.Fatal("envelope randomness reused")
	}
	raw, _ := json.Marshal(first)
	if bytes.Contains(raw, []byte("link-sandbox-fictional")) {
		t.Fatal("plaintext serialized")
	}
	got, err := openLink(ctx, testWrapper{}, scope, id, first)
	if err != nil {
		t.Fatal(err)
	}
	if err = got.WithValue(func(value string) error {
		if value != "link-sandbox-fictional" {
			t.Fatal("wrong token")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	other := scope
	other.Actor = "00000000-0000-4000-8000-000000000005"
	if _, err = openLink(ctx, testWrapper{}, other, id, first); err == nil {
		t.Fatal("cross-actor decryption")
	}
	if _, err = openLink(ctx, testWrapper{}, scope, other.Actor, first); err == nil {
		t.Fatal("cross-attempt decryption")
	}
	first.Profile = profile
	if _, err = openLink(ctx, testWrapper{}, scope, id, first); err == nil {
		t.Fatal("cross-profile decryption")
	}
}
