package autopost

import (
	"encoding/json"
	"testing"
)

const testEntity = "91000000-0000-4000-8000-000000000001"
const testBook = "91000000-0000-4000-8000-000000000002"

func encoded(v any) json.RawMessage { raw, _ := json.Marshal(v); return raw }
func statusFixture() map[string]any {
	return map[string]any{"profile": "automatic-accounting-status-v1", "entity_id": testEntity, "book_id": testBook, "policy_revision": "0", "enabled": false, "pending_count": "0", "attention_count": "0", "blocked_reason": nil, "last_operation": nil, "holds": []any{}, "holds_truncated": false}
}

// TestStatusDisclosureClosure rejects cross-scope or contradictory counts before public disclosure.
func TestStatusDisclosureClosure(t *testing.T) {
	r := Request{Entity: testEntity, Book: testBook}
	if !validResult(encoded(statusFixture()), "status", r) {
		t.Fatal("valid absent status rejected")
	}
	for name, change := range map[string]func(map[string]any){"foreign book": func(v map[string]any) { v["book_id"] = testEntity }, "unknown field": func(v map[string]any) { v["private_payload"] = "fixture" }, "unbounded revision": func(v map[string]any) { v["policy_revision"] = "9223372036854775808" }, "false empty holds": func(v map[string]any) { v["attention_count"] = "1" }, "wrong truncation": func(v map[string]any) { v["holds_truncated"] = true }, "unknown reason": func(v map[string]any) { v["blocked_reason"] = "arbitrary" }} {
		t.Run(name, func(t *testing.T) {
			v := statusFixture()
			change(v)
			if validResult(encoded(v), "status", r) {
				t.Fatal("malformed status accepted")
			}
		})
	}
}

// TestOperationReceiptStates requires a journal only for a committed terminal receipt.
func TestOperationReceiptStates(t *testing.T) {
	v := map[string]any{"operation_id": testEntity, "state": "prepared", "review_id": testBook, "journal_id": nil, "reason": nil}
	if !validOperation(encoded(v)) {
		t.Fatal("prepared receipt rejected")
	}
	v["state"] = "committed"
	if validOperation(encoded(v)) {
		t.Fatal("committed receipt missing journal accepted")
	}
	v["journal_id"] = testEntity
	if !validOperation(encoded(v)) {
		t.Fatal("committed receipt rejected")
	}
	v["state"] = "held"
	v["reason"] = "fx_required"
	if validOperation(encoded(v)) {
		t.Fatal("held receipt claimed journal")
	}
	v["journal_id"] = nil
	if !validOperation(encoded(v)) {
		t.Fatal("FX hold rejected")
	}
}
