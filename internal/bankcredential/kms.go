package bankcredential

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"hash/crc32"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Tokens supplies workload identity OAuth without credential files or environment discovery.
type Tokens interface {
	AccessToken(context.Context) (string, error)
}

// KMS uses only the fixed Google endpoint and a reviewed development wrapping-key resource.
type KMS struct {
	key    string
	tokens Tokens
	client *http.Client
}

// Cloud KMS uses the canonical project ID, independently of numeric GSM references.
var keyPattern = regexp.MustCompile(`^projects/[a-z][a-z0-9-]{4,28}[a-z0-9]/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials$`)

// NewKMS validates configuration and performs no network access.
func NewKMS(key string, tokens Tokens, transport http.RoundTripper) (*KMS, error) {
	if !keyPattern.MatchString(key) || tokens == nil || transport == nil {
		return nil, ErrUnavailable
	}
	return &KMS{key, tokens, &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// crc is request/response integrity metadata, never a stored plaintext digest.
func crc(v []byte) string {
	return strconv.FormatUint(uint64(crc32.Checksum(v, crc32.MakeTable(crc32.Castagnoli))), 10)
}

// post strips upstream diagnostics and never retries a wrapping or unwrapping operation automatically.
func (k *KMS) post(ctx context.Context, action string, in, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	token, e := k.tokens.AccessToken(ctx)
	if e != nil || !tokenValid(token) {
		return ErrUnavailable
	}
	body, e := json.Marshal(in)
	if e != nil {
		return ErrUnavailable
	}
	req, e := http.NewRequestWithContext(ctx, "POST", "https://cloudkms.googleapis.com/v1/"+k.key+":"+action, bytes.NewReader(body))
	if e != nil {
		return ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	response, e := k.client.Do(req)
	if e != nil {
		return ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return ErrUnavailable
	}
	raw, e := io.ReadAll(io.LimitReader(response.Body, 32769))
	if e != nil || len(raw) > 32768 || ctx.Err() != nil {
		return ErrUnavailable
	}
	if !validJSON(raw) {
		return ErrUnavailable
	}
	if e = json.Unmarshal(raw, out); e != nil {
		return ErrUnavailable
	}
	return nil
}

// Wrap validates server CRC acknowledgements, returned ciphertext CRC and immutable version name.
func (k *KMS) Wrap(ctx context.Context, key, aad []byte) (string, []byte, error) {
	if len(key) != 32 || len(aad) > 2048 {
		return "", nil, ErrUnavailable
	}
	var out struct {
		Name       string `json:"name"`
		Ciphertext string `json:"ciphertext"`
		CRC        string `json:"ciphertextCrc32c"`
		Plain      bool   `json:"verifiedPlaintextCrc32c"`
		AAD        bool   `json:"verifiedAdditionalAuthenticatedDataCrc32c"`
	}
	e := k.post(ctx, "encrypt", map[string]string{"plaintext": base64.StdEncoding.EncodeToString(key), "additionalAuthenticatedData": base64.StdEncoding.EncodeToString(aad), "plaintextCrc32c": crc(key), "additionalAuthenticatedDataCrc32c": crc(aad)}, &out)
	if e != nil || !out.Plain || !out.AAD || !keyVersionMatches(k.key, out.Name) {
		return "", nil, ErrUnavailable
	}
	wrapped, e := base64.StdEncoding.Strict().DecodeString(out.Ciphertext)
	if e != nil || len(wrapped) < 1 || len(wrapped) > 8192 || crc(wrapped) != out.CRC {
		return "", nil, ErrUnavailable
	}
	return out.Name, wrapped, nil
}

// Unwrap accepts only this key's numeric versions; AEAD further verifies scope and token integrity.
func (k *KMS) Unwrap(ctx context.Context, version string, wrapped, aad []byte) ([]byte, error) {
	if !keyVersionMatches(k.key, version) || len(wrapped) < 1 || len(wrapped) > 8192 || len(aad) > 2048 {
		return nil, ErrUnavailable
	}
	var out struct {
		Plain string `json:"plaintext"`
		CRC   string `json:"plaintextCrc32c"`
	}
	e := k.post(ctx, "decrypt", map[string]string{"ciphertext": base64.StdEncoding.EncodeToString(wrapped), "additionalAuthenticatedData": base64.StdEncoding.EncodeToString(aad), "ciphertextCrc32c": crc(wrapped), "additionalAuthenticatedDataCrc32c": crc(aad)}, &out)
	if e != nil {
		return nil, ErrUnavailable
	}
	key, e := base64.StdEncoding.Strict().DecodeString(out.Plain)
	if e != nil || len(key) != 32 || crc(key) != out.CRC {
		clear(key)
		return nil, ErrUnavailable
	}
	return key, nil
}

// validJSON rejects duplicate/case-alias fields, trailing documents and excessive nesting.
func validJSON(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) bool
	walk = func(depth int) bool {
		if depth > 16 {
			return false
		}
		token, e := d.Token()
		if e != nil {
			return false
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return true
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, e := d.Token()
				name, ok := key.(string)
				if e != nil || !ok {
					return false
				}
				name = strings.ToLower(name)
				if seen[name] {
					return false
				}
				seen[name] = true
				if !walk(depth + 1) {
					return false
				}
			}
			end, e := d.Token()
			return e == nil && end == json.Delim('}')
		case '[':
			for d.More() {
				if !walk(depth + 1) {
					return false
				}
			}
			end, e := d.Token()
			return e == nil && end == json.Delim(']')
		}
		return false
	}
	if !walk(0) {
		return false
	}
	_, e := d.Token()
	return e == io.EOF
}

// String never exposes workload token internals through incidental formatting.
func (*KMS) String() string { return "[bank credential wrapping capability]" }

// GoString applies the same redaction to detailed formatting.
func (k *KMS) GoString() string { return k.String() }
