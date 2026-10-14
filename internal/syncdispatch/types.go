// Package syncdispatch relays SQL-owned acquisition demand without posting journals.
package syncdispatch

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Failure exposes a closed operational classification, never infrastructure diagnostics.
type Failure struct {
	Code      string
	Retryable bool
}

// Error returns only the adapter-owned classification.
func (e *Failure) Error() string { return e.Code }

// failure constructs only locally chosen public codes.
func failure(s string, retry bool) error { return &Failure{s, retry} }

// validID rejects null identity and noncanonical UUID syntax before SQL.
func validID(s string) bool {
	return uuid.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}

// positive accepts only canonical positive signed-bigint strings.
func positive(s string) bool {
	n, e := strconv.ParseInt(s, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == s
}

// oneOf compares bounded protocol vocabulary.
func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}

// holder gives one transaction call a stable ownership token across confirmed retries.
func holder() (string, error) {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", failure("temporarily_unavailable", true)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	s := hex.EncodeToString(b[:])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:], nil
}

// taskName separates acquisition delivery from receipt-only observer transport.
func taskName(id, generation string) string {
	b := sha256.Sum256([]byte(id + "|" + generation + "|2"))
	return hex.EncodeToString(b[:])
}

// metadata rejects unexpected SQL response fields and trailing values without exposing them.
func metadata(raw []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return failure("invalid_metadata", false)
	}
	if d.Decode(new(any)) != io.EOF {
		return failure("invalid_metadata", false)
	}
	return nil
}

// liveLease rejects expired or implausibly distant lease metadata; SQL remains authority.
func liveLease(t *time.Time) bool {
	return t != nil && t.After(time.Now()) && t.Before(time.Now().Add(3*time.Minute))
}

// encode emits the existing three-key wakeup without acquisition or financial payloads.
func encode(id, generation string) []byte {
	return []byte(fmt.Sprintf(`{"schema_version":1,"operation_id":%q,"dispatch_generation":%q}`, id, generation))
}

// Runner owns account-filtered acquisition and receives only committed SQL scope and stable claim identity.
type Runner interface {
	Run(context.Context, acquisition.Scope, string) (acquisition.Outcome, error)
}
