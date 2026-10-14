package webhook

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Verifier owns a small environment-fixed key cache; callers cannot select a key URL.
type Verifier struct {
	provider KeyProvider
	mu       sync.Mutex
	cache    map[string]cachedKey
	window   time.Time
	misses   int
	lookup   chan struct{}
	now      func() time.Time
}

// NewVerifier constructs an inert verifier without external I/O.
func NewVerifier(p KeyProvider) (*Verifier, error) {
	if p == nil {
		return nil, errUnavailable
	}
	return &Verifier{provider: p, cache: map[string]cachedKey{}, lookup: make(chan struct{}, 1), now: time.Now}, nil
}

// segment enforces canonical unpadded URL encoding, preventing equivalent alternate receipt identities.
func segment(s string) ([]byte, error) {
	b, e := base64.RawURLEncoding.Strict().DecodeString(s)
	if e != nil || base64.RawURLEncoding.EncodeToString(b) != s {
		return nil, errInvalid
	}
	return b, nil
}

// publicKey validates metadata and coordinates before invoking the standard verifier.
func publicKey(k JWK, id string, issued int64, now time.Time) (*ecdsa.PublicKey, error) {
	if k.ID != id || k.Alg != "ES256" || k.Curve != "P-256" || k.Type != "EC" || k.Use != "sig" || k.Created <= 0 || k.Created > issued || k.Created > now.Unix() || k.Expired != nil && (*k.Expired <= now.Unix() || *k.Expired <= k.Created) {
		return nil, errInvalid
	}
	xb, e := segment(k.X)
	if e != nil || len(xb) != 32 {
		return nil, errInvalid
	}
	yb, e := segment(k.Y)
	if e != nil || len(yb) != 32 {
		return nil, errInvalid
	}
	x, y := new(big.Int).SetBytes(xb), new(big.Int).SetBytes(yb)
	curve := elliptic.P256()
	if !curve.IsOnCurve(x, y) {
		return nil, errInvalid
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

// key bounds cache misses and never falls back to an expired cached key.
func (v *Verifier) key(ctx context.Context, id string) (JWK, error) {
	now := v.now()
	v.mu.Lock()
	if c, ok := v.cache[id]; ok && now.Before(c.until) {
		v.mu.Unlock()
		return c.key, nil
	}
	v.mu.Unlock()
	select {
	case v.lookup <- struct{}{}:
		defer func() { <-v.lookup }()
	case <-ctx.Done():
		return JWK{}, errUnavailable
	default:
		return JWK{}, errUnavailable
	}
	v.mu.Lock()
	now = v.now()
	if c, ok := v.cache[id]; ok && now.Before(c.until) {
		v.mu.Unlock()
		return c.key, nil
	}
	if v.window.IsZero() || !now.Before(v.window.Add(time.Minute)) {
		v.window = now
		v.misses = 0
	}
	if v.misses >= 4 {
		v.mu.Unlock()
		return JWK{}, errUnavailable
	}
	v.misses++
	v.mu.Unlock()
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	k, e := v.provider.Get(bounded, id)
	if e != nil || bounded.Err() != nil {
		return JWK{}, errUnavailable
	}
	if _, e = publicKey(k, id, v.now().Unix(), v.now()); e != nil {
		return JWK{}, errInvalid
	}
	if k.Expired != nil {
		expiry := *k.Expired
		k.Expired = &expiry
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.cache) >= 4 {
		var oldest string
		var until time.Time
		for name, c := range v.cache {
			if oldest == "" || c.until.Before(until) {
				oldest = name
				until = c.until
			}
		}
		delete(v.cache, oldest)
	}
	v.cache[id] = cachedKey{k, v.now().Add(time.Minute)}
	return k, nil
}

// Verify authenticates original bytes and returns only the reviewed metadata projection.
func (v *Verifier) Verify(ctx context.Context, token string, body []byte) (Verified, error) {
	if ctx.Err() != nil {
		return Verified{}, errUnavailable
	}
	if len(token) == 0 || len(token) > 8192 || len(body) == 0 || len(body) > bodyLimit {
		return Verified{}, errInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Verified{}, errInvalid
	}
	header, e := segment(parts[0])
	if e != nil {
		return Verified{}, e
	}
	payload, e := segment(parts[1])
	if e != nil {
		return Verified{}, e
	}
	sig, e := segment(parts[2])
	if e != nil || len(sig) != 64 {
		return Verified{}, errInvalid
	}
	var h struct {
		Alg  string  `json:"alg"`
		Kid  string  `json:"kid"`
		Type *string `json:"typ"`
	}
	if object(header, []string{"alg", "kid"}, []string{"typ"}, &h) != nil || h.Alg != "ES256" || !keyID.MatchString(h.Kid) || h.Type != nil && *h.Type != "JWT" {
		return Verified{}, errInvalid
	}
	var claims struct {
		Issued int64  `json:"iat"`
		Hash   string `json:"request_body_sha256"`
	}
	if object(payload, []string{"iat", "request_body_sha256"}, nil, &claims) != nil || !digest.MatchString(claims.Hash) {
		return Verified{}, errInvalid
	}
	now := v.now()
	if claims.Issued <= 0 || claims.Issued < now.Unix()-300 || claims.Issued > now.Unix()+60 {
		return Verified{}, errInvalid
	}
	k, e := v.key(ctx, h.Kid)
	if e != nil {
		return Verified{}, e
	}
	pub, e := publicKey(k, h.Kid, claims.Issued, v.now())
	if e != nil {
		return Verified{}, e
	}
	if jwt.SigningMethodES256.Verify(parts[0]+"."+parts[1], parts[2], pub) != nil {
		return Verified{}, errInvalid
	}
	sum := sha256.Sum256(body)
	expected, _ := hex.DecodeString(claims.Hash)
	if subtle.ConstantTimeCompare(sum[:], expected) != 1 {
		return Verified{}, errInvalid
	}
	var event struct {
		Type        string  `json:"webhook_type"`
		Code        string  `json:"webhook_code"`
		Item        string  `json:"item_id"`
		User        *string `json:"user_id"`
		Environment string  `json:"environment"`
		Initial     *bool   `json:"initial_update_complete"`
		Historical  *bool   `json:"historical_update_complete"`
	}
	if object(body, []string{"webhook_type", "webhook_code", "item_id", "environment", "initial_update_complete", "historical_update_complete"}, []string{"user_id"}, &event) != nil || event.Type != "TRANSACTIONS" || event.Code != "SYNC_UPDATES_AVAILABLE" || event.Environment != "sandbox" || !opaque.MatchString(event.Item) || event.User != nil && !opaque.MatchString(*event.User) || event.Initial == nil || event.Historical == nil {
		return Verified{}, errInvalid
	}
	if ctx.Err() != nil {
		return Verified{}, errUnavailable
	}
	if current := v.now().Unix(); claims.Issued < current-300 || claims.Issued > current+60 {
		return Verified{}, errInvalid
	}
	fingerprint := deliveryFingerprint(parts, sig)
	return Verified{Environment: event.Environment, ItemID: event.Item, ReceiptKey: hex.EncodeToString(fingerprint[:]), BodySHA256: claims.Hash, IssuedAt: claims.Issued}, nil
}

// deliveryFingerprint normalizes ECDSA S-malleability only for replay identity after library verification.
// It does not replace signature verification or reject valid high-S provider signatures.
func deliveryFingerprint(parts []string, signature []byte) [32]byte {
	canonical := append([]byte(nil), signature...)
	s := new(big.Int).SetBytes(signature[32:])
	opposite := new(big.Int).Sub(elliptic.P256().Params().N, s)
	if opposite.Cmp(s) < 0 {
		opposite.FillBytes(canonical[32:])
	}
	return sha256.Sum256([]byte(parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(canonical)))
}
