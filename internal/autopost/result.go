package autopost

import (
	"encoding/json"
	"regexp"
	"strconv"
	"time"
)

var uuidPattern = regexp.MustCompile(`^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$`)
var revisionPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,18})$`)
var holdReasons = map[string]bool{"unmapped_rule": true, "ambiguous_rule": true, "outside_policy_interval": true, "manual_draft": true, "source_not_booked": true, "zero_amount": true, "source_held": true, "overlap_review_required": true, "source_correction_required": true, "paired_transfer_required": true, "split_treatment_required": true, "fx_required": true, "unsupported_currency": true, "unsupported_policy": true, "inactive_account": true, "closed_period": true, "source_unavailable": true, "permission_changed": true, "saved_work": true, "stale_input": true}

func validID(v string) bool {
	return uuidPattern.MatchString(v) && v != "00000000-0000-0000-0000-000000000000"
}
func object(raw json.RawMessage, keys ...string) (map[string]json.RawMessage, bool) {
	var m map[string]json.RawMessage
	if len(raw) > 1<<20 || json.Unmarshal(raw, &m) != nil || m == nil || len(m) != len(keys) {
		return nil, false
	}
	for _, key := range keys {
		if _, ok := m[key]; !ok {
			return nil, false
		}
	}
	return m, true
}
func str(raw json.RawMessage) string { var v string; _ = json.Unmarshal(raw, &v); return v }
func flag(raw json.RawMessage) (bool, bool) {
	var v bool
	err := json.Unmarshal(raw, &v)
	return v, err == nil && string(raw) != "null"
}
func revision(raw json.RawMessage) bool {
	s := str(raw)
	if !revisionPattern.MatchString(s) {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
func list(raw json.RawMessage, max int) ([]json.RawMessage, bool) {
	var v []json.RawMessage
	err := json.Unmarshal(raw, &v)
	return v, err == nil && v != nil && len(v) <= max
}
func date(raw json.RawMessage) bool {
	s := str(raw)
	t, err := time.Parse("2006-01-02", s)
	return err == nil && t.Year() > 0 && t.Format("2006-01-02") == s
}
func nullable(raw json.RawMessage, check func(json.RawMessage) bool) bool {
	return string(raw) == "null" || check(raw)
}
func id(raw json.RawMessage) bool { return validID(str(raw)) }
func blockedReason(raw json.RawMessage) bool {
	s := str(raw)
	return string(raw) == "null" || s == "saved_work" || s == "permission_changed" || s == "source_unavailable"
}
func text(raw json.RawMessage, max int) bool {
	s := str(raw)
	return len([]rune(s)) > 0 && len([]rune(s)) <= max
}
func validPolicy(raw json.RawMessage) bool {
	p, ok := object(raw, "book_policy_id", "effective_from", "effective_to", "rules")
	if !ok || !id(p["book_policy_id"]) || !date(p["effective_from"]) || !nullable(p["effective_to"], date) {
		return false
	}
	if string(p["effective_to"]) != "null" && str(p["effective_to"]) <= str(p["effective_from"]) {
		return false
	}
	rules, ok := list(p["rules"], 100)
	if !ok {
		return false
	}
	seen := map[string]bool{}
	names := map[string]bool{}
	for _, raw := range rules {
		r, ok := object(raw, "rule_id", "name", "source_account_id", "direction", "description", "offset_account_id", "treatment")
		if !ok || !id(r["rule_id"]) || !id(r["source_account_id"]) || !id(r["offset_account_id"]) || !text(r["name"], 80) || !text(r["description"], 500) {
			return false
		}
		name, rid := str(r["name"]), str(r["rule_id"])
		if seen[rid] || names[name] {
			return false
		}
		seen[rid] = true
		names[name] = true
		if d := str(r["direction"]); d != "inflow" && d != "outflow" {
			return false
		}
		switch str(r["treatment"]) {
		case "expense", "income", "transfer", "card_payment", "loan":
		default:
			return false
		}
	}
	return true
}
func validOperation(raw json.RawMessage) bool {
	o, ok := object(raw, "operation_id", "state", "review_id", "journal_id", "reason")
	if !ok || !id(o["operation_id"]) || !id(o["review_id"]) || !nullable(o["journal_id"], id) {
		return false
	}
	switch str(o["state"]) {
	case "prepared":
		return string(o["journal_id"]) == "null" && string(o["reason"]) == "null"
	case "committed":
		return id(o["journal_id"]) && string(o["reason"]) == "null"
	case "held", "stale":
		return string(o["journal_id"]) == "null" && holdReasons[str(o["reason"])]
	default:
		return false
	}
}

// validResult rejects malformed or cross-scope SQL output before HTTP can disclose it.
func validResult(raw json.RawMessage, operation string, r Request) bool {
	if operation == "get" || operation == "set" {
		m, ok := object(raw, "profile", "entity_id", "book_id", "revision", "enabled", "policy", "supported_profile")
		if !ok || str(m["profile"]) != "automatic-accounting-policy-v1" || str(m["entity_id"]) != r.Entity || str(m["book_id"]) != r.Book || !revision(m["revision"]) || str(m["supported_profile"]) != "source-identity-usd-v1" {
			return false
		}
		enabled, ok := flag(m["enabled"])
		if !ok {
			return false
		}
		if str(m["revision"]) == "0" {
			return !enabled && string(m["policy"]) == "null"
		}
		return validPolicy(m["policy"])
	}
	if operation != "status" {
		return false
	}
	m, ok := object(raw, "profile", "entity_id", "book_id", "enabled", "pending_count", "attention_count", "blocked_reason", "last_operation", "holds", "holds_truncated")
	if !ok || str(m["profile"]) != "automatic-accounting-status-v1" || str(m["entity_id"]) != r.Entity || str(m["book_id"]) != r.Book || !revision(m["pending_count"]) || !revision(m["attention_count"]) || !blockedReason(m["blocked_reason"]) || !nullable(m["last_operation"], validOperation) {
		return false
	}
	if _, ok = flag(m["enabled"]); !ok {
		return false
	}
	truncated, ok := flag(m["holds_truncated"])
	if !ok {
		return false
	}
	holds, ok := list(m["holds"], 100)
	if !ok {
		return false
	}
	attention, _ := strconv.ParseInt(str(m["attention_count"]), 10, 64)
	if truncated != (attention > int64(len(holds))) || (!truncated && attention != int64(len(holds))) {
		return false
	}
	seen := map[string]bool{}
	for _, raw := range holds {
		h, ok := object(raw, "review_id", "source_account_id", "reason", "operation_id")
		if !ok || !id(h["review_id"]) || !id(h["source_account_id"]) || !id(h["operation_id"]) || !holdReasons[str(h["reason"])] || seen[str(h["review_id"])] {
			return false
		}
		seen[str(h["review_id"])] = true
	}
	return true
}
