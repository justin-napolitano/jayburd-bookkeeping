package collectiononboarding

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// adoptionDTO is an independently fixed safe full-set review, not a provider payload.
func adoptionDTO() map[string]any {
	return map[string]any{"candidate_id": candidateID, "entity_id": workspaceID, "book_id": commandID, "entity_name": "Fictional entity", "book_name": "Fictional book", "snapshot": digest, "sources": []any{map[string]any{"source_id": sourceID, "display_name": "Fictional checking"}}, "expected_generation": "2", "impact": "pause_legacy_acquisition"}
}
func encoded(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func previewDTO(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{"profile": "private-onboarding-preview-v1", "preview_id": previewID, "operation": "adopt", "expires_at": "2026-11-08T12:00:00Z", "workspace_id": workspaceID, "snapshot": digest, "review": adoptionDTO()}
}
func commandDTO(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{"profile": "private-onboarding-command-v1", "command_id": commandID, "recovery_id": recoveryID, "prepare_key": "same-key", "state": "committed", "preview": previewDTO(t), "result": map[string]any{"collection_item_id": candidateID, "item_generation": "3", "sources": []any{map[string]any{"source_account_id": recoveryID, "source_revision": "0", "revision": "1"}}}, "acknowledged": false}
}

// TestOnboardingAdoptionResponse checks the separate old source identities/new alias receipt without rewriting them.
func TestOnboardingAdoptionResponse(t *testing.T) {
	p := previewDTO(t)
	r := Request{Body: `{"operation":"adopt","candidate_id":"` + candidateID + `","snapshot":"` + digest + `"}`}
	if !validResult(encoded(t, p), "preview", r) {
		t.Fatal("complete adoption review rejected")
	}
	c := commandDTO(t)
	if !validResult(encoded(t, c), "execute", Request{Command: commandID}) {
		t.Fatal("canonical42 adoption result rejected")
	}
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"foreign-item", func(c map[string]any) { c["result"].(map[string]any)["collection_item_id"] = sourceID }},
		{"missing-source", func(c map[string]any) { c["result"].(map[string]any)["sources"] = []any{} }},
		{"noninitial-alias", func(c map[string]any) {
			c["result"].(map[string]any)["sources"].([]any)[0].(map[string]any)["source_revision"] = "1"
		}},
		{"secret-field", func(c map[string]any) {
			c["preview"].(map[string]any)["review"].(map[string]any)["credential_ref"] = "forbidden"
		}},
		{"wrong-impact", func(c map[string]any) {
			c["preview"].(map[string]any)["review"].(map[string]any)["impact"] = "resume_provider"
		}},
		{"case-alias", func(c map[string]any) { c["Prepare_key"] = c["prepare_key"]; delete(c, "prepare_key") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := commandDTO(t)
			tc.mutate(c)
			if validResult(encoded(t, c), "execute", Request{Command: commandID}) {
				t.Fatal("malformed adoption response accepted")
			}
		})
	}
	c = commandDTO(t)
	c["acknowledged"] = true
	c["preview"].(map[string]any)["expires_at"] = "2000-01-01T00:00:00Z"
	if !validResult(encoded(t, c), "prepare", request("prepare")) {
		t.Fatal("same-key acknowledged historical result rejected after expiry")
	}
	r = request("prepare")
	r.Body = strings.Replace(r.Body, "same-key", "other-key", 1)
	if validResult(encoded(t, c), "prepare", r) {
		t.Fatal("prepare key identity drift accepted")
	}
}

// TestOnboardingCatalogPendingDTOs accepts explicit unavailable states without manufacturing absence.
func TestOnboardingCatalogPendingDTOs(t *testing.T) {
	candidate := adoptionDTO()
	delete(candidate, "expected_generation")
	delete(candidate, "impact")
	for _, workspace := range []any{map[string]any{"state": "absent"}, map[string]any{"state": "unavailable"}, map[string]any{"state": "ready", "workspace_id": workspaceID, "authority_revision": "2", "capabilities": []string{"read"}}} {
		v := map[string]any{"profile": "private-onboarding-catalog-v1", "workspace": workspace, "can_setup": false, "can_adopt": false, "candidates": []any{}}
		if !validResult(encoded(t, v), "catalog", Request{}) {
			t.Fatal("safe complete catalog rejected")
		}
	}
	for _, tc := range []struct {
		workspace    any
		setup, adopt bool
		items        []any
	}{
		{map[string]any{"state": "ready", "workspace_id": workspaceID, "authority_revision": "1", "capabilities": []string{"read"}}, true, false, []any{}},
		{map[string]any{"state": "unavailable"}, true, false, []any{}},
		{map[string]any{"state": "absent"}, false, true, []any{}},
		{map[string]any{"state": "ready", "workspace_id": workspaceID, "authority_revision": "1", "capabilities": []string{"read"}}, false, false, []any{candidate}},
		{map[string]any{"state": "ready", "workspace_id": workspaceID, "authority_revision": "1", "capabilities": []string{"read", "acquire"}}, false, true, []any{}},
	} {
		v := map[string]any{"profile": "private-onboarding-catalog-v1", "workspace": tc.workspace, "can_setup": tc.setup, "can_adopt": tc.adopt, "candidates": tc.items}
		if validResult(encoded(t, v), "catalog", Request{}) {
			t.Fatal("contradictory catalog admitted")
		}
	}
	for _, state := range []string{"none", "blocked", "inaccessible", "available"} {
		v := map[string]any{"profile": "private-onboarding-pending-v1", "state": state, "command": nil, "recovery_id": nil}
		if state == "available" {
			v["command"] = json.RawMessage(prepared())
		}
		if state == "available" || state == "inaccessible" {
			v["recovery_id"] = recoveryID
		}
		if !validResult(encoded(t, v), "pending", Request{}) {
			t.Fatal("explicit pending state rejected", state)
		}
		if state != "available" {
			v["command"] = commandDTO(t)
			if validResult(encoded(t, v), "pending", Request{}) {
				t.Fatal("payload leaked through unavailable pending state")
			}
		}
	}
	candidates := []any{}
	for i := 0; i < 101; i++ {
		c := adoptionDTO()
		delete(c, "expected_generation")
		delete(c, "impact")
		c["candidate_id"] = fmt.Sprintf("00000000-0000-4000-8000-%012d", i+100)
		candidates = append(candidates, c)
	}
	catalog := map[string]any{"profile": "private-onboarding-catalog-v1", "workspace": map[string]any{"state": "ready", "workspace_id": workspaceID, "authority_revision": "1", "capabilities": []string{"read", "acquire"}}, "can_setup": false, "can_adopt": true, "candidates": candidates[:100]}
	if !validResult(encoded(t, catalog), "catalog", Request{}) {
		t.Fatal("complete candidate boundary rejected")
	}
	catalog["candidates"] = candidates
	if validResult(encoded(t, catalog), "catalog", Request{}) {
		t.Fatal("candidate limit exceeded")
	}
	if closedJSON([]byte(`{"review":{"capabilities":[],"capabilities":[]}}`)) || closedJSON([]byte(strings.Repeat("[", 40)+"0"+strings.Repeat("]", 40))) {
		t.Fatal("duplicate/deep response admitted")
	}
}

// TestOnboardingStableRecoveryIdentity rejects mismatched pending ownership before payload exposure.
func TestOnboardingStableRecoveryIdentity(t *testing.T) {
	pending := map[string]any{"profile": "private-onboarding-pending-v1", "state": "available", "command": json.RawMessage(prepared()), "recovery_id": recoveryID}
	if !validResult(encoded(t, pending), "pending", Request{}) {
		t.Fatal("stable pending identity rejected")
	}
	pending["recovery_id"] = previewID
	if validResult(encoded(t, pending), "pending", Request{}) {
		t.Fatal("foreign pending recovery identity accepted")
	}
	c := commandDTO(t)
	delete(c, "recovery_id")
	if validResult(encoded(t, c), "execute", Request{Command: commandID}) {
		t.Fatal("missing durable recovery identity accepted")
	}
	c = commandDTO(t)
	c["recovery_id"] = "00000000-0000-0000-0000-000000000000"
	if validResult(encoded(t, c), "execute", Request{Command: commandID}) {
		t.Fatal("zero recovery identity accepted")
	}
	c = commandDTO(t)
	c["acknowledged"] = true
	if !validResult(encoded(t, c), "acknowledge", Request{Command: commandID}) {
		t.Fatal("acknowledged closure lost stable identity")
	}
}
