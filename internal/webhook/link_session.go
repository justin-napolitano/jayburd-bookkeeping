package webhook

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

// verifiedLinkSession contains only the closed operational proof, never raw tokens or ownership.
// Its private type/field prevents other packages from constructing a Link receipt kind.
type verifiedLinkSession struct {
	linkHash, sessionHash, status string
}

// boundedToken accepts visible ASCII within the local profile without normalizing token bytes.
func boundedToken(s string, maximum int) bool {
	if len(s) == 0 || len(s) > maximum {
		return false
	}
	for i := range s {
		if s[i] < '!' || s[i] > '~' {
			return false
		}
	}
	return true
}

// tokenHash reduces one validated opaque identifier to the SQL protocol's lowercase SHA256.
func tokenHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// parseLinkSession validates the modern signed event profile and discards optional public tokens/user ID.
// Provider Item identity and completion time are deliberately absent: SQL owns association and eligibility.
func parseLinkSession(body []byte) (*verifiedLinkSession, error) {
	return parseLinkSessionForProfile(body, plaidprofile.Sandbox())
}

// parseLinkSessionForProfile compares authenticated event origin to trusted runtime metadata before hashing identity.
func parseLinkSessionForProfile(body []byte, profile plaidprofile.Profile) (*verifiedLinkSession, error) {
	if !profile.Valid() {
		return nil, errInvalid
	}
	var event struct {
		Type        string   `json:"webhook_type"`
		Code        string   `json:"webhook_code"`
		Status      string   `json:"status"`
		Link        string   `json:"link_token"`
		Session     string   `json:"link_session_id"`
		Environment string   `json:"environment"`
		Public      *string  `json:"public_token"`
		Publics     []string `json:"public_tokens"`
		User        *string  `json:"user_id"`
	}
	if object(body, []string{"webhook_type", "webhook_code", "status", "link_token", "link_session_id", "environment"},
		[]string{"public_token", "public_tokens", "user_id"}, &event) != nil ||
		event.Type != "LINK" || event.Code != "SESSION_FINISHED" || event.Environment != profile.Environment() ||
		(event.Status != "success" && event.Status != "exited") || !boundedToken(event.Link, 4096) ||
		!boundedToken(event.Session, 256) || event.Public != nil && !boundedToken(*event.Public, 4096) ||
		event.User != nil && !boundedToken(*event.User, 256) || len(event.Publics) > 25 {
		return nil, errInvalid
	}
	for _, token := range event.Publics {
		if !boundedToken(token, 4096) {
			return nil, errInvalid
		}
	}
	return &verifiedLinkSession{linkHash: tokenHash(event.Link), sessionHash: tokenHash(event.Session), status: event.Status}, nil
}
