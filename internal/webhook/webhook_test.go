package webhook

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// keyFunc injects synthetic public metadata, never provider credentials.
type keyFunc func(context.Context, string) (JWK, error)

// Get forwards a synthetic bounded lookup.
func (f keyFunc) Get(c context.Context, id string) (JWK, error) { return f(c, id) }

// receiverFunc injects durable outcome faults.
type receiverFunc func(context.Context, Verified) (Receipt, error)

// Receive forwards already-verified metadata only.
func (f receiverFunc) Receive(c context.Context, v Verified) (Receipt, error) { return f(c, v) }

// fixture builds independent ephemeral P256 signing material without files or network.
func fixture(t testing.TB) (*ecdsa.PrivateKey, JWK, time.Time) {
	t.Helper()
	private, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	return private, JWK{Alg: "ES256", Curve: "P-256", ID: "synthetic-key", Type: "EC", Use: "sig", X: base64.RawURLEncoding.EncodeToString(private.X.FillBytes(make([]byte, 32))), Y: base64.RawURLEncoding.EncodeToString(private.Y.FillBytes(make([]byte, 32))), Created: now.Add(-time.Hour).Unix()}, now
}

// bodyFixture contains an original supported notification rather than transaction or credential data.
func bodyFixture() []byte {
	return []byte(`{"webhook_type":"TRANSACTIONS","webhook_code":"SYNC_UPDATES_AVAILABLE","item_id":"synthetic-item","environment":"sandbox","initial_update_complete":true,"historical_update_complete":false}`)
}

// signRaw signs exact JSON sections through the maintained JWT implementation.
func signRaw(t testing.TB, key *ecdsa.PrivateKey, header, payload string) string {
	t.Helper()
	unsigned := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + base64.RawURLEncoding.EncodeToString([]byte(payload))
	sig, e := jwt.SigningMethodES256.Sign(unsigned, key)
	if e != nil {
		t.Fatal(e)
	}
	return unsigned + "." + sig
}

// signBody pins original raw-byte digest and integer issuance, preserving test control over age.
func signBody(t testing.TB, key *ecdsa.PrivateKey, kid string, body []byte, iat int64) string {
	sum := sha256.Sum256(body)
	return signRaw(t, key, fmt.Sprintf(`{"alg":"ES256","kid":%q,"typ":"JWT"}`, kid), fmt.Sprintf(`{"iat":%d,"request_body_sha256":%q}`, iat, hex.EncodeToString(sum[:])))
}

// verifyFixture fixes the test clock and counts actual cache misses.
func verifyFixture(t testing.TB, k JWK, now time.Time) (*Verifier, *int) {
	t.Helper()
	calls := 0
	v, e := NewVerifier(keyFunc(func(context.Context, string) (JWK, error) { calls++; return k, nil }))
	if e != nil {
		t.Fatal(e)
	}
	v.now = func() time.Time { return now }
	return v, &calls
}

// invokeHandler supplies synthetic HTTP headers and captures redacted results.
func invokeHandler(h http.Handler, token string, body []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/webhooks/plaid", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Plaid-Verification", token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// TestPlaidWebhook proves signature, exact-byte digest and durable receipt replay before acknowledgement.
func TestPlaidWebhook(t *testing.T) {
	private, key, now := fixture(t)
	v, lookups := verifyFixture(t, key, now)
	receipts := map[string]bool{}
	signals := 0
	receiver := receiverFunc(func(_ context.Context, got Verified) (Receipt, error) {
		if got.ItemID != "synthetic-item" || got.Environment != "sandbox" || !digest.MatchString(got.BodySHA256) {
			t.Fatal("unsafe metadata")
		}
		state := "accepted"
		if receipts[got.ReceiptKey] {
			state = "duplicate"
		} else {
			receipts[got.ReceiptKey] = true
			signals++
		}
		return Receipt{state, "00000000-0000-4000-8000-000000000001", 1}, nil
	})
	h, _ := NewHandler(v, receiver)
	body := bodyFixture()
	token := signBody(t, private, key.ID, body, now.Unix())
	for i := 0; i < 2; i++ {
		w := invokeHandler(h, token, body)
		if w.Code != 204 || w.Body.Len() != 0 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("acknowledgement %d", w.Code)
		}
	}
	if signals != 1 || *lookups != 1 {
		t.Fatal("replay repeated effect or key fetch")
	}
	next := signBody(t, private, key.ID, body, now.Unix()+1)
	if invokeHandler(h, next, body).Code != 204 || signals != 2 {
		t.Fatal("body digest permanently suppressed later signed notification")
	}
}

// TestWebhookRejectedInputs keeps malformed, stale and signed unsupported notifications outside persistence.
func TestWebhookRejectedInputs(t *testing.T) {
	private, key, now := fixture(t)
	body := bodyFixture()
	good := signBody(t, private, key.ID, body, now.Unix())
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	payload := fmt.Sprintf(`{"iat":%d,"request_body_sha256":%q}`, now.Unix(), hash)
	cases := []struct {
		name, token string
		body        []byte
	}{
		{"tamper", good, append(append([]byte{}, body...), ' ')},
		{"stale", signBody(t, private, key.ID, body, now.Unix()-301), body},
		{"future", signBody(t, private, key.ID, body, now.Unix()+61), body},
		{"algorithm", signRaw(t, private, `{"alg":"HS256","kid":"synthetic-key"}`, payload), body},
		{"duplicate header", signRaw(t, private, `{"alg":"ES256","kid":"synthetic-key","kid":"other"}`, payload), body},
		{"key url", signRaw(t, private, `{"alg":"ES256","kid":"synthetic-key","jku":"https://evil.invalid"}`, payload), body},
		{"duplicate claims", signRaw(t, private, `{"alg":"ES256","kid":"synthetic-key"}`, fmt.Sprintf(`{"iat":%d,"iat":%d,"request_body_sha256":%q}`, now.Unix(), now.Unix(), hash)), body},
		{"fractional iat", signRaw(t, private, `{"alg":"ES256","kid":"synthetic-key"}`, fmt.Sprintf(`{"iat":%d.0,"request_body_sha256":%q}`, now.Unix(), hash)), body},
		{"null typ", signRaw(t, private, `{"alg":"ES256","kid":"synthetic-key","typ":null}`, payload), body},
		{"oversize JWT", strings.Repeat("x", 8193), body},
		{"oversize body", good, []byte(strings.Repeat("x", bodyLimit+1))},
	}
	for _, raw := range []string{strings.Replace(string(body), `"sandbox"`, `"production"`, 1), strings.Replace(string(body), `"SYNC_UPDATES_AVAILABLE"`, `"DEFAULT_UPDATE"`, 1), strings.Replace(string(body), `"item_id":"synthetic-item"`, `"item_id":"synthetic-item","item_id":"other"`, 1), strings.Replace(string(body), `"historical_update_complete":false`, `"historical_update_complete":null`, 1), strings.TrimSuffix(string(body), "}") + `,"user_id":null}`} {
		b := []byte(raw)
		cases = append(cases, struct {
			name, token string
			body        []byte
		}{"signed unsupported", signBody(t, private, key.ID, b, now.Unix()), b})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, _ := verifyFixture(t, key, now)
			h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) {
				t.Fatal("invalid input reached SQL")
				return Receipt{}, nil
			}))
			if w := invokeHandler(h, tc.token, tc.body); w.Code < 400 || strings.Contains(w.Body.String(), "synthetic") {
				t.Fatal("invalid input acknowledged or reflected")
			}
		})
	}
}

// TestWebhookKeyLifecycle checks metadata expiry on cache hits, bounded rotation and miss throttling.
func TestWebhookKeyLifecycle(t *testing.T) {
	private, key, now := fixture(t)
	clock := now
	calls := 0
	expiry := now.Unix() + 10
	key.Expired = &expiry
	v, _ := NewVerifier(keyFunc(func(_ context.Context, id string) (JWK, error) { calls++; k := key; k.ID = id; return k, nil }))
	v.now = func() time.Time { return clock }
	body := bodyFixture()
	if _, e := v.Verify(context.Background(), signBody(t, private, key.ID, body, now.Unix()), body); e != nil {
		t.Fatal(e)
	}
	clock = now.Add(11 * time.Second)
	if _, e := v.Verify(context.Background(), signBody(t, private, key.ID, body, clock.Unix()), body); e == nil || calls != 1 {
		t.Fatal("cached key expiry bypass")
	}
	key.Expired = nil
	clock = now
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("rotated-%d", i)
		if _, e := v.Verify(context.Background(), signBody(t, private, id, body, now.Unix()), body); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := v.Verify(context.Background(), signBody(t, private, "rate-limited", body, now.Unix()), body); !errors.Is(e, errUnavailable) || calls != 4 {
		t.Fatal("unbounded key misses")
	}
	clock = now.Add(time.Minute)
	if _, e := v.Verify(context.Background(), signBody(t, private, "new-window", body, clock.Unix()), body); e != nil || len(v.cache) > 4 {
		t.Fatal("rotation recovery failed")
	}
	for _, mutate := range []func(*JWK){func(k *JWK) { k.Curve = "P-384" }, func(k *JWK) { k.X = "AA" }, func(k *JWK) { k.Use = "enc" }, func(k *JWK) { k.Created = now.Unix() + 1 }} {
		k := key
		mutate(&k)
		bad, _ := verifyFixture(t, k, now)
		if _, e := bad.Verify(context.Background(), signBody(t, private, k.ID, body, now.Unix()), body); e == nil {
			t.Fatal("invalid key metadata")
		}
	}
}

// TestWebhookPersistenceFailure never acknowledges denied, uncertain or malformed receipt outcomes.
func TestWebhookPersistenceFailure(t *testing.T) {
	private, key, now := fixture(t)
	body := bodyFixture()
	token := signBody(t, private, key.ID, body, now.Unix())
	for _, tc := range []struct {
		err    error
		status int
	}{{errDenied, 403}, {errUnknown, 503}, {errors.New("synthetic-private-diagnostic"), 503}, {nil, 503}} {
		v, _ := verifyFixture(t, key, now)
		h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) { return Receipt{}, tc.err }))
		w := invokeHandler(h, token, body)
		if w.Code != tc.status || strings.Contains(w.Body.String(), "synthetic") {
			t.Fatal("unsafe persistence acknowledgement")
		}
	}
	v, _ := NewVerifier(keyFunc(func(context.Context, string) (JWK, error) { return JWK{}, errors.New("synthetic-provider-diagnostic") }))
	v.now = func() time.Time { return now }
	h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) {
		t.Fatal("key failure reached SQL")
		return Receipt{}, nil
	}))
	if w := invokeHandler(h, token, body); w.Code != 503 || strings.Contains(w.Body.String(), "synthetic") {
		t.Fatal("key outage falsely accepted")
	}
}

// TestWebhookRequestBoundary denies alternate routes and duplicate headers before key or SQL work.
func TestWebhookRequestBoundary(t *testing.T) {
	v, _ := NewVerifier(keyFunc(func(context.Context, string) (JWK, error) {
		t.Fatal("invalid request reached key provider")
		return JWK{}, nil
	}))
	h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) {
		t.Fatal("invalid request reached SQL")
		return Receipt{}, nil
	}))
	for _, path := range []string{"/v1/workspace", "/webhooks/plaid?", "/webhooks/plaid?item=synthetic", "/webhooks/%70laid"} {
		r := httptest.NewRequest("POST", path, strings.NewReader("{}"))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 404 {
			t.Fatal("alternate route accepted")
		}
	}
	r := httptest.NewRequest("POST", "/webhooks/plaid", strings.NewReader("{}"))
	r.Header.Add("Plaid-Verification", "one")
	r.Header.Add("Plaid-Verification", "two")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("duplicate verification accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := v.Verify(ctx, "token", []byte("{}")); !errors.Is(e, errUnavailable) {
		t.Fatal("cancelled verifier proceeded")
	}
}

// TestCredentialFormatting prohibits incidental key-provider credential disclosure.
func TestCredentialFormatting(t *testing.T) {
	c := Credentials{"synthetic-client", "synthetic-secret"}
	for _, s := range []string{fmt.Sprintf("%v", c), fmt.Sprintf("%#v", c)} {
		if strings.Contains(s, "synthetic") {
			t.Fatal("credentials formatted")
		}
	}
	if b, e := json.Marshal(c); e == nil || len(b) != 0 {
		t.Fatal("credentials serialized")
	}
}

// alternateSignature constructs the mathematically equivalent ECDSA S value without the private key.
func alternateSignature(t testing.TB, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	signature, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil || len(signature) != 64 {
		t.Fatal("signature fixture")
	}
	s := new(big.Int).SetBytes(signature[32:])
	new(big.Int).Sub(elliptic.P256().Params().N, s).FillBytes(signature[32:])
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	return strings.Join(parts, ".")
}

// TestEquivalentECDSADelivery proves valid high/low-S representations share one receipt identity.
func TestEquivalentECDSADelivery(t *testing.T) {
	private, key, now := fixture(t)
	v, _ := verifyFixture(t, key, now)
	body := bodyFixture()
	token := signBody(t, private, key.ID, body, now.Unix())
	alternate := alternateSignature(t, token)
	if alternate == token {
		t.Fatal("malleability fixture unchanged")
	}
	original, e := v.Verify(context.Background(), token, body)
	if e != nil {
		t.Fatal(e)
	}
	changed, e := v.Verify(context.Background(), alternate, body)
	if e != nil || original.ReceiptKey != changed.ReceiptKey {
		t.Fatal("equivalent signature created another delivery")
	}
}

// TestWebhookAdmission bounds concurrent durable receivers and releases capacity after completion.
func TestWebhookAdmission(t *testing.T) {
	private, key, now := fixture(t)
	v, _ := verifyFixture(t, key, now)
	body := bodyFixture()
	token := signBody(t, private, key.ID, body, now.Unix())
	if _, e := v.Verify(context.Background(), token, body); e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{}, 16)
	release := make(chan struct{})
	results := make(chan int, 16)
	h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) {
		entered <- struct{}{}
		<-release
		return Receipt{"duplicate", "00000000-0000-4000-8000-000000000001", 1}, nil
	}))
	for i := 0; i < 16; i++ {
		go func() { results <- invokeHandler(h, token, body).Code }()
	}
	for i := 0; i < 16; i++ {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			close(release)
			t.Fatal("admission fixture stalled")
		}
	}
	if w := invokeHandler(h, token, body); w.Code != 503 {
		close(release)
		t.Fatal("unbounded concurrent admission")
	}
	close(release)
	for i := 0; i < 16; i++ {
		if <-results != 204 {
			t.Fatal("accepted receiver lost acknowledgement")
		}
	}
}

// TestKeyLookupCancellation bounds in-flight provider work and rejects parallel cache-miss amplification.
func TestKeyLookupCancellation(t *testing.T) {
	private, key, now := fixture(t)
	entered := make(chan struct{})
	v, _ := NewVerifier(keyFunc(func(ctx context.Context, _ string) (JWK, error) {
		close(entered)
		<-ctx.Done()
		return JWK{}, ctx.Err()
	}))
	v.now = func() time.Time { return now }
	body := bodyFixture()
	token := signBody(t, private, key.ID, body, now.Unix())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { _, e := v.Verify(ctx, token, body); finished <- e }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("lookup did not start")
	}
	if _, e := v.Verify(context.Background(), token, body); !errors.Is(e, errUnavailable) {
		t.Fatal("parallel miss bypassed admission")
	}
	cancel()
	select {
	case e := <-finished:
		if !errors.Is(e, errUnavailable) {
			t.Fatal("cancel classified unsafely")
		}
	case <-time.After(time.Second):
		t.Fatal("provider ignored cancellation")
	}
}
