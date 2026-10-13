package observer

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"testing"
)

// TestGoogleVerifierSyntheticKeys proves real RS256 checking against injected fictional public keys.
func TestGoogleVerifierSyntheticKeys(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	b64 := base64.RawURLEncoding.EncodeToString
	jwks, _ := json.Marshal(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "synthetic", "n": b64(key.N.Bytes()), "e": b64(big.NewInt(int64(key.E)).Bytes())}}})
	calls := 0
	v, e := NewGoogleTokenVerifier(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "" || r.URL.String() != "https://www.googleapis.com/oauth2/v3/certs" {
			t.Fatal("credential or unexpected URL")
		}
		return queueResponse(200, string(jwks)), nil
	}))
	if e != nil {
		t.Fatal(e)
	}
	c := testClaims()
	payload, _ := json.Marshal(map[string]any{"iss": c.Issuer, "aud": c.Audience, "sub": c.Subject, "email": c.Email, "email_verified": true, "iat": c.IssuedAt, "exp": c.Expires})
	header := b64([]byte(`{"alg":"RS256","kid":"synthetic","typ":"JWT"}`))
	message := header + "." + b64(payload)
	digest := sha256.Sum256([]byte(message))
	sig, e := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if e != nil {
		t.Fatal(e)
	}
	token := message + "." + b64(sig)
	got, e := v.Verify(context.Background(), token, c.Audience)
	if e != nil || got != c || calls != 1 {
		t.Fatalf("valid token %v calls=%d", e, calls)
	}
	for _, bad := range []string{message + "." + b64(make([]byte, 256)), "invalid", strings.Replace(token, header, b64([]byte(`{"alg":"RS256","kid":"unknown"}`)), 1), strings.Replace(token, header, b64([]byte(`{"alg":"ES256","kid":"synthetic"}`)), 1), strings.Replace(token, header, b64([]byte(`{"alg":"none","kid":"synthetic"}`)), 1)} {
		if _, e := v.Verify(context.Background(), bad, c.Audience); e == nil || strings.Contains(e.Error(), "synthetic") {
			t.Fatal("bad signature accepted or raw error exposed")
		}
	}
	if _, e := v.Verify(context.Background(), token, "https://other.run.app"); e == nil {
		t.Fatal("wrong audience accepted")
	}
}

// TestGoogleVerifierKeyResponseLimit rejects oversized valid prefixes rather than accepting truncation.
func TestGoogleVerifierKeyResponseLimit(t *testing.T) {
	for _, size := range []int{65536, 65537} {
		body := `{"keys":[]}` + strings.Repeat(" ", size-len(`{"keys":[]}`))
		transport := keyTransport{roundTripFunc(func(*http.Request) (*http.Response, error) { return queueResponse(200, body), nil })}
		request, _ := http.NewRequestWithContext(context.Background(), "GET", "https://www.googleapis.com/oauth2/v3/certs", nil)
		response, e := transport.RoundTrip(request)
		if size == 65536 {
			if e != nil {
				t.Fatal(e)
			}
			response.Body.Close()
		} else if e == nil {
			response.Body.Close()
			t.Fatal("oversized valid key prefix accepted")
		}
	}
}
