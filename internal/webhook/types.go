// Package webhook verifies bounded provider notifications before durable wakeup acknowledgement.
package webhook

import (
	"context"
	"errors"
	"regexp"
	"time"
)

const bodyLimit = 64 << 10

var opaque = regexp.MustCompile(`^[!-~]{1,256}$`)
var keyID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var identifier = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var errInvalid = errors.New("invalid_webhook")
var errUnavailable = errors.New("temporarily_unavailable")
var errDenied = errors.New("webhook_denied")
var errUnknown = errors.New("commit_unknown")

// Verified holds authenticated notification metadata, never the JWT or raw body.
type Verified struct {
	Environment, ItemID, ReceiptKey, BodySHA256 string
	IssuedAt                                    int64
}

// Receipt is a committed acknowledgement, not proof of transaction acquisition.
type Receipt struct {
	State       string `json:"state"`
	ReceiptID   string `json:"receipt_id"`
	StreamCount int    `json:"stream_count"`
}

// Receiver atomically stores the receipt and authorized durable wakeup intents.
type Receiver interface {
	Receive(context.Context, Verified) (Receipt, error)
}

// JWK is the provider's exact supported public verification-key metadata.
type JWK struct {
	Alg     string `json:"alg"`
	Curve   string `json:"crv"`
	ID      string `json:"kid"`
	Type    string `json:"kty"`
	Use     string `json:"use"`
	X       string `json:"x"`
	Y       string `json:"y"`
	Created int64  `json:"created_at"`
	Expired *int64 `json:"expired_at"`
}

// KeyProvider retrieves a public key through an operator-owned fixed destination.
type KeyProvider interface {
	Get(context.Context, string) (JWK, error)
}

// cachedKey tracks immutable copied metadata and a short independent refresh deadline.
type cachedKey struct {
	key   JWK
	until time.Time
}

// validID permits only nonzero canonical scope identifiers.
func validID(s string) bool {
	return identifier.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}
