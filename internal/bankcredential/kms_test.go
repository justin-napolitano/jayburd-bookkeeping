package bankcredential

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type tokenSource struct{}

func (tokenSource) AccessToken(context.Context) (string, error) { return "fictional-oauth", nil }

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestKMSIntegrity tests wire CRCs and redacted rejection without using a cloud key.
func TestKMSIntegrity(t *testing.T) {
	for _, scenario := range []string{"pass", "unverified", "wrong-name", "corrupt", "http", "bad-decrypt", "duplicate"} {
		t.Run(scenario, func(t *testing.T) {
			key := make([]byte, 32)
			aad := []byte("fictional-scope")
			wrapped := []byte("wrapped-fictional-key")
			transport := roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "cloudkms.googleapis.com" || !strings.HasPrefix(r.URL.Path, "/v1/"+testKey+":") || r.Header.Get("Authorization") != "Bearer fictional-oauth" {
					t.Fatal("wrong routing")
				}
				var in map[string]string
				if json.NewDecoder(r.Body).Decode(&in) != nil {
					t.Fatal("bad request")
				}
				if in["additionalAuthenticatedDataCrc32c"] != crc(aad) {
					t.Fatal("AAD checksum missing")
				}
				result := map[string]any{}
				if strings.HasSuffix(r.URL.Path, ":encrypt") {
					if in["plaintextCrc32c"] != crc(key) {
						t.Fatal("key checksum missing")
					}
					result = map[string]any{"name": testKey + "/cryptoKeyVersions/1", "ciphertext": base64.StdEncoding.EncodeToString(wrapped), "ciphertextCrc32c": crc(wrapped), "verifiedPlaintextCrc32c": true, "verifiedAdditionalAuthenticatedDataCrc32c": true}
					switch scenario {
					case "unverified":
						result["verifiedPlaintextCrc32c"] = false
					case "wrong-name":
						result["name"] = testKey + "-other/cryptoKeyVersions/1"
					case "corrupt":
						result["ciphertextCrc32c"] = "0"
					}
				} else {
					if in["ciphertextCrc32c"] != crc(wrapped) {
						t.Fatal("ciphertext checksum missing")
					}
					result = map[string]any{"plaintext": base64.StdEncoding.EncodeToString(key), "plaintextCrc32c": crc(key)}
					if scenario == "bad-decrypt" {
						result["plaintextCrc32c"] = "1"
					}
				}
				raw, _ := json.Marshal(result)
				if scenario == "duplicate" {
					raw = append(raw[:len(raw)-1], []byte(`,"verifiedPlaintextCrc32c":true}`)...)
				}
				status := 200
				if scenario == "http" {
					status = 403
					raw = []byte("private-upstream-data")
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
			})
			k, e := NewKMS(testKey, tokenSource{}, transport)
			if e != nil {
				t.Fatal(e)
			}
			version, got, e := k.Wrap(context.Background(), key, aad)
			if scenario != "pass" && scenario != "bad-decrypt" {
				if e == nil || e.Error() != "bank_credential_unavailable" {
					t.Fatal("bad wrap accepted or diagnostic exposed")
				}
				return
			}
			plain, e := k.Unwrap(context.Background(), version, got, aad)
			if scenario == "bad-decrypt" {
				if e == nil {
					t.Fatal("bad checksum accepted")
				}
			} else if e != nil || len(plain) != 32 {
				t.Fatal("valid decrypt rejected")
			}
		})
	}
}

// TestKMSCanonicalProject rejects the numeric alias that failed the deployed probe.
func TestKMSCanonicalProject(t *testing.T) {
	for _, key := range []string{
		"projects/123456789/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials",
		"projects/jayburd-synthetic-test/locations/us-central1/keyRings/other/cryptoKeys/bank-credentials",
	} {
		if _, err := NewKMS(key, tokenSource{}, roundTrip(nil)); err == nil {
			t.Fatal("noncanonical or foreign key accepted")
		}
	}
}
