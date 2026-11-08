package collectiononboarding

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"time"
	"unicode/utf8"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var keyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var revisionPattern = regexp.MustCompile(`^[1-9][0-9]{0,18}$`)

func validID(v string) bool {
	return uuidPattern.MatchString(v) && v != "00000000-0000-0000-0000-000000000000"
}
func str(v json.RawMessage) string { var s string; _ = json.Unmarshal(v, &s); return s }
func flag(v json.RawMessage) (bool, bool) {
	var b bool
	e := json.Unmarshal(v, &b)
	return b, e == nil && string(v) != "null"
}
func null(v json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(v), []byte("null")) }
func object(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, bool) {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil || len(m) != len(keys) {
		return nil, false
	}
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			return nil, false
		}
	}
	return m, true
}
func array(raw json.RawMessage, max int) ([]json.RawMessage, bool) {
	var a []json.RawMessage
	if json.Unmarshal(raw, &a) != nil || a == nil || len(a) > max {
		return nil, false
	}
	return a, true
}
func text(raw json.RawMessage, max int) bool {
	s := str(raw)
	return s != "" && utf8.RuneCountInString(s) <= max
}
func caps(raw json.RawMessage, complete bool) bool {
	a, ok := array(raw, 3)
	if !ok || len(a) == 0 || complete && len(a) != 3 {
		return false
	}
	seen := map[string]bool{}
	for _, v := range a {
		s := str(v)
		if (s != "read" && s != "acquire" && s != "classify") || seen[s] {
			return false
		}
		seen[s] = true
	}
	return seen["read"]
}

// closedJSON rejects duplicate keys and excessive nesting before response projection.
func closedJSON(raw []byte) bool {
	if len(raw) == 0 || len(raw) > 1<<20 || !utf8.Valid(raw) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) bool
	walk = func(depth int) bool {
		if depth > 32 {
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
				s, ok := key.(string)
				if e != nil || !ok || seen[s] {
					return false
				}
				seen[s] = true
				if !walk(depth + 1) {
					return false
				}
			}
			last, e := d.Token()
			return e == nil && last == json.Delim('}')
		case '[':
			for d.More() {
				if !walk(depth + 1) {
					return false
				}
			}
			last, e := d.Token()
			return e == nil && last == json.Delim(']')
		}
		return false
	}
	if !walk(0) {
		return false
	}
	_, e := d.Token()
	return e == io.EOF
}

// validResult fences the closed response profile and request identity before COMMIT or HTTP exposure.
func validResult(raw json.RawMessage, operation string, r Request) bool {
	if !closedJSON(raw) {
		return false
	}
	switch operation {
	case "catalog":
		return validCatalog(raw)
	case "preview":
		if !validPreview(raw) {
			return false
		}
		var req map[string]json.RawMessage
		var v map[string]json.RawMessage
		if json.Unmarshal([]byte(r.Body), &req) != nil || json.Unmarshal(raw, &v) != nil || str(req["operation"]) != str(v["operation"]) {
			return false
		}
		if str(v["operation"]) == "adopt" {
			var review map[string]json.RawMessage
			_ = json.Unmarshal(v["review"], &review)
			return str(req["candidate_id"]) == str(review["candidate_id"]) && str(req["snapshot"]) == str(v["snapshot"])
		}
		return true
	case "pending":
		v, ok := object(raw, "profile", "state", "command", "recovery_id")
		if !ok || str(v["profile"]) != "private-onboarding-pending-v1" {
			return false
		}
		switch str(v["state"]) {
		case "none", "blocked":
			return null(v["command"]) && null(v["recovery_id"])
		case "inaccessible":
			return null(v["command"]) && validID(str(v["recovery_id"]))
		case "available":
			if !validCommand(v["command"], "") || !validID(str(v["recovery_id"])) {
				return false
			}
			var c map[string]json.RawMessage
			_ = json.Unmarshal(v["command"], &c)
			b, _ := flag(c["acknowledged"])
			return !b && str(c["recovery_id"]) == str(v["recovery_id"])
		}
		return false
	case "clear":
		v, ok := object(raw, "profile", "recovery_id", "cleared")
		if !ok || str(v["profile"]) != "private-onboarding-clear-v1" || !validID(str(v["recovery_id"])) {
			return false
		}
		b, ok := flag(v["cleared"])
		var req map[string]json.RawMessage
		return ok && b && json.Unmarshal([]byte(r.Body), &req) == nil && str(req["recovery_id"]) == str(v["recovery_id"])
	default:
		if !validCommand(raw, r.Command) {
			return false
		}
		if operation == "execute" || operation == "cancel" || operation == "acknowledge" {
			var c map[string]json.RawMessage
			_ = json.Unmarshal(raw, &c)
			if str(c["state"]) == "prepared" {
				return false
			}
			if operation == "acknowledge" {
				b, _ := flag(c["acknowledged"])
				if !b {
					return false
				}
			}
		}
		if operation == "prepare" {
			var req, c, p map[string]json.RawMessage
			if json.Unmarshal([]byte(r.Body), &req) != nil || json.Unmarshal(raw, &c) != nil || json.Unmarshal(c["preview"], &p) != nil {
				return false
			}
			return str(req["preview_id"]) == str(p["preview_id"]) && str(req["prepare_key"]) == str(c["prepare_key"])
		}
		return true
	}
}
func validCatalog(raw json.RawMessage) bool {
	v, ok := object(raw, "profile", "workspace", "can_setup", "can_adopt", "candidates")
	if !ok || str(v["profile"]) != "private-onboarding-catalog-v1" {
		return false
	}
	canSetup, ok := flag(v["can_setup"])
	if !ok {
		return false
	}
	canAdopt, ok := flag(v["can_adopt"])
	if !ok {
		return false
	}
	var w map[string]json.RawMessage
	if json.Unmarshal(v["workspace"], &w) != nil {
		return false
	}
	switch str(w["state"]) {
	case "absent", "unavailable":
		if len(w) != 1 {
			return false
		}
	case "ready":
		if len(w) != 4 || !validID(str(w["workspace_id"])) || !revisionPattern.MatchString(str(w["authority_revision"])) || !caps(w["capabilities"], false) {
			return false
		}
	default:
		return false
	}
	candidates, ok := array(v["candidates"], 100)
	if !ok {
		return false
	}
	canAcquire := false
	if str(w["state"]) == "ready" {
		capabilities, _ := array(w["capabilities"], 3)
		for _, c := range capabilities {
			if str(c) == "acquire" {
				canAcquire = true
			}
		}
	}
	if canSetup && str(w["state"]) != "absent" || canAdopt && (!canAcquire || len(candidates) == 0) || !canAcquire && len(candidates) > 0 {
		return false
	}
	seen := map[string]bool{}
	for _, c := range candidates {
		if !validCandidate(c, false) {
			return false
		}
		var m map[string]json.RawMessage
		_ = json.Unmarshal(c, &m)
		id := str(m["candidate_id"])
		if seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
func validCandidate(raw json.RawMessage, review bool) bool {
	keys := []string{"candidate_id", "entity_id", "book_id", "entity_name", "book_name", "snapshot", "sources"}
	if review {
		keys = append(keys, "expected_generation", "impact")
	}
	v, ok := object(raw, keys...)
	if !ok || !validID(str(v["candidate_id"])) || !validID(str(v["entity_id"])) || !validID(str(v["book_id"])) || !text(v["entity_name"], 200) || !text(v["book_name"], 200) || !hashPattern.MatchString(str(v["snapshot"])) {
		return false
	}
	if review && (!revisionPattern.MatchString(str(v["expected_generation"])) || str(v["impact"]) != "pause_legacy_acquisition") {
		return false
	}
	sources, ok := array(v["sources"], 25)
	if !ok || len(sources) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, s := range sources {
		m, ok := object(s, "source_id", "display_name")
		if !ok || !validID(str(m["source_id"])) || !text(m["display_name"], 200) || seen[str(m["source_id"])] {
			return false
		}
		seen[str(m["source_id"])] = true
	}
	return true
}
func validPreview(raw json.RawMessage) bool {
	v, ok := object(raw, "profile", "preview_id", "operation", "expires_at", "workspace_id", "snapshot", "review")
	if !ok || str(v["profile"]) != "private-onboarding-preview-v1" || !validID(str(v["preview_id"])) || !hashPattern.MatchString(str(v["snapshot"])) {
		return false
	}
	if _, e := time.Parse(time.RFC3339Nano, str(v["expires_at"])); e != nil {
		return false
	}
	switch str(v["operation"]) {
	case "setup":
		review, ok := object(v["review"], "capabilities")
		return ok && null(v["workspace_id"]) && caps(review["capabilities"], true)
	case "adopt":
		if !validID(str(v["workspace_id"])) || !validCandidate(v["review"], true) {
			return false
		}
		var review map[string]json.RawMessage
		_ = json.Unmarshal(v["review"], &review)
		return str(review["snapshot"]) == str(v["snapshot"])
	}
	return false
}
func validCommand(raw json.RawMessage, id string) bool {
	v, ok := object(raw, "profile", "command_id", "recovery_id", "prepare_key", "state", "preview", "result", "acknowledged")
	if !ok || str(v["profile"]) != "private-onboarding-command-v1" || !validID(str(v["command_id"])) || !validID(str(v["recovery_id"])) || !keyPattern.MatchString(str(v["prepare_key"])) || id != "" && str(v["command_id"]) != id || !validPreview(v["preview"]) {
		return false
	}
	acknowledged, ok := flag(v["acknowledged"])
	if !ok {
		return false
	}
	switch str(v["state"]) {
	case "prepared":
		return !acknowledged && null(v["result"])
	case "cancelled":
		result, ok := object(v["result"], "cancelled")
		if !ok {
			return false
		}
		b, ok := flag(result["cancelled"])
		return ok && b
	case "committed":
		var p map[string]json.RawMessage
		_ = json.Unmarshal(v["preview"], &p)
		if str(p["operation"]) == "setup" {
			result, ok := object(v["result"], "workspace_id", "workspace_revision", "capabilities")
			return ok && validID(str(result["workspace_id"])) && revisionPattern.MatchString(str(result["workspace_revision"])) && caps(result["capabilities"], true)
		}
		result, ok := object(v["result"], "collection_item_id", "item_generation", "sources")
		if !ok || !validID(str(result["collection_item_id"])) || !revisionPattern.MatchString(str(result["item_generation"])) {
			return false
		}
		var review map[string]json.RawMessage
		_ = json.Unmarshal(p["review"], &review)
		if str(review["candidate_id"]) != str(result["collection_item_id"]) {
			return false
		}
		sources, ok := array(result["sources"], 25)
		expected, _ := array(review["sources"], 25)
		if !ok || len(sources) != len(expected) {
			return false
		}
		seen := map[string]bool{}
		for _, s := range sources {
			m, ok := object(s, "source_account_id", "source_revision", "revision")
			if !ok || !validID(str(m["source_account_id"])) || seen[str(m["source_account_id"])] || str(m["source_revision"]) != "0" || str(m["revision"]) != "1" {
				return false
			}
			seen[str(m["source_account_id"])] = true
		}
		return true
	}
	return false
}
