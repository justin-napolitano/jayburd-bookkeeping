// Package sandboxidentity projects only provider identities against independently authored synthetic tags.
package sandboxidentity

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxverification"
)

var ErrProjection = errors.New("sandbox_identity_projection_failed")
var opaque = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,200}$`)

// ValidIdentity rejects credential-shaped strings as bounded synthetic identity metadata.
func ValidIdentity(value string) bool {
	return opaque.MatchString(value) && !strings.HasPrefix(value, "access-") && !strings.HasPrefix(value, "public-") && !strings.HasPrefix(value, "link-")
}

// text extracts only bounded identity metadata; financial fields are never interpreted here.
func text(fields map[string]json.RawMessage, key string) (string, bool) {
	var value string
	raw, ok := fields[key]
	if !ok || json.Unmarshal(raw, &value) != nil || len(value) == 0 || len(value) > 200 {
		return "", false
	}
	return value, true
}

// objects requires explicit non-null arrays, bounded before inspecting each identity object.
func objects(raw json.RawMessage) ([]json.RawMessage, bool) {
	var values []json.RawMessage
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &values) != nil || len(values) > 1000 {
		return nil, false
	}
	return values, true
}

// Project supports only a complete fresh posted-only page, never cursor/history reconstruction.
// It copies IDs and reads tags only; observed financial fields cannot update the independent fixture.
func Project(fixture, page []byte, account string) ([]byte, int, error) {
	if len(fixture) == 0 || len(page) == 0 || len(fixture)+len(page) > sandboxverification.MaxInputBytes || !ValidIdentity(account) {
		return nil, 0, ErrProjection
	}
	f, err := sandboxverification.StrictObject(fixture)
	if err != nil || len(f) != 4 || string(f["profile"]) != `"jayburd-sandbox-verification/1"` {
		return nil, 0, ErrProjection
	}
	fixtureID, ok := text(f, "fixture_id")
	if !ok || !opaque.MatchString(fixtureID) || len(fixtureID) > 64 {
		return nil, 0, ErrProjection
	}
	links, ok := objects(f["pending_links"])
	if !ok || len(links) != 0 {
		return nil, 0, ErrProjection
	}
	rows, ok := objects(f["rows"])
	if !ok || len(rows) == 0 {
		return nil, 0, ErrProjection
	}
	tags := map[string]string{}
	logicalIDs := map[string]bool{}
	order := make([]string, 0, len(rows))
	for _, raw := range rows {
		r, e := sandboxverification.StrictObject(raw)
		id, idOK := text(r, "logical_id")
		tag, tagOK := text(r, "description")
		if e != nil || !idOK || !tagOK || !opaque.MatchString(id) || logicalIDs[id] || tags[tag] != "" ||
			string(r["revision"]) != `"1"` || string(r["current"]) != "true" || string(r["previous_revision"]) != "null" || string(r["source_state"]) != `"booked"` {
			return nil, 0, ErrProjection
		}
		logicalIDs[id], tags[tag] = true, id
		order = append(order, id)
	}
	p, err := sandboxverification.StrictObject(page)
	if err != nil || string(p["has_more"]) != "false" || string(p["transactions_update_status"]) != `"HISTORICAL_UPDATE_COMPLETE"` {
		return nil, 0, ErrProjection
	}
	for _, key := range []string{"modified", "removed"} {
		values, valid := objects(p[key])
		if !valid || len(values) != 0 {
			return nil, 0, ErrProjection
		}
	}
	added, ok := objects(p["added"])
	if !ok || len(added) != len(rows) {
		return nil, 0, ErrProjection
	}
	identities := map[string]string{}
	seen := map[string]bool{}
	for _, raw := range added {
		r, e := sandboxverification.StrictObject(raw)
		id, idOK := text(r, "transaction_id")
		tag, tagOK := text(r, "name")
		accountID, accountOK := text(r, "account_id")
		logical := tags[tag]
		if e != nil || !idOK || !tagOK || !accountOK || accountID != account || logical == "" || identities[logical] != "" || seen[id] || !ValidIdentity(id) {
			return nil, 0, ErrProjection
		}
		identities[logical], seen[id] = id, true
	}
	// Stable fixture order makes the hash independent of provider response ordering.
	// item carries only the independently named logical identity and provider-assigned identifier.
	type item struct {
		LogicalID     string `json:"logical_id"`
		TransactionID string `json:"transaction_id"`
	}
	result := struct {
		Profile   string `json:"profile"`
		FixtureID string `json:"fixture_id"`
		Items     []item `json:"items"`
	}{Profile: "jayburd-sandbox-identity-map/1", FixtureID: fixtureID}
	for _, logical := range order {
		result.Items = append(result.Items, item{logical, identities[logical]})
	}
	out, e := json.Marshal(result)
	if e != nil {
		return nil, 0, ErrProjection
	}
	return append(out, '\n'), len(order), nil
}
