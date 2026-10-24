package webhook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// linkBody supplies fixed fictional modern fields; optional tokens are validated, never exchanged.
func linkBody(link, session, status string) []byte {
	b, _ := json.Marshal(map[string]any{"webhook_type": "LINK", "webhook_code": "SESSION_FINISHED",
		"link_token": link, "link_session_id": session, "status": status, "environment": "sandbox"})
	return b
}

// TestHostedRepairSignature keeps malformed or unauthenticated Link events outside all business persistence.
func TestHostedRepairSignature(t *testing.T) {
	private, key, now := fixture(t)
	good := linkBody("link-synthetic", "session-synthetic", "success")
	for _, tc := range []struct {
		name    string
		body    []byte
		age     int64
		corrupt bool
	}{
		{"wrong_signature", good, 0, true},
		{"stale", good, -301, false}, {"future", good, 61, false},
		{"duplicate", []byte(strings.Replace(string(good), `"status":"success"`, `"status":"success","status":"success"`, 1)), 0, false},
		{"unknown", []byte(strings.TrimSuffix(string(good), "}") + `,"item_id":"foreign"}`), 0, false},
		{"wrong_environment", []byte(strings.Replace(string(good), `"sandbox"`, `"production"`, 1)), 0, false},
		{"unknown_outcome", linkBody("link-synthetic", "session-synthetic", "pending"), 0, false},
		{"empty_link", linkBody("", "session-synthetic", "success"), 0, false},
		{"large_link", linkBody(strings.Repeat("a", 4097), "session-synthetic", "success"), 0, false},
		{"large_session", linkBody("link-synthetic", strings.Repeat("a", 257), "success"), 0, false},
		{"non_ascii", linkBody("link-é", "session-synthetic", "success"), 0, false},
		{"trailing", append(append([]byte(nil), good...), []byte(`{}`)...), 0, false},
		{"wrong_code", []byte(strings.Replace(string(good), "SESSION_FINISHED", "ITEM_ADD_RESULT", 1)), 0, false},
		{"wrong_type", []byte(strings.Replace(string(good), `"LINK"`, `"ITEM"`, 1)), 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, _ := verifyFixture(t, key, now)
			calls := 0
			h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) { calls++; return Receipt{}, nil }))
			token := signBody(t, private, key.ID, tc.body, now.Unix()+tc.age)
			if tc.corrupt {
				other, _, _ := fixture(t)
				token = signBody(t, other, key.ID, tc.body, now.Unix())
			}
			if w := invokeHandler(h, token, tc.body); w.Code != 401 || calls != 0 {
				t.Fatalf("unsafe rejection status=%d persistence=%d", w.Code, calls)
			}
		})
	}
	v, _ := verifyFixture(t, key, now)
	if _, e := v.Verify(context.Background(), signBody(t, private, key.ID, good, now.Unix()), append(good, ' ')); e == nil {
		t.Fatal("raw body alteration accepted")
	}
}

// TestHostedRepairNoSecretAccess proves the projection retains no provider/user/public token identifiers.
func TestHostedRepairNoSecretAccess(t *testing.T) {
	private, key, now := fixture(t)
	for _, status := range []string{"success", "exited"} {
		v, _ := verifyFixture(t, key, now)
		body := []byte(strings.TrimSuffix(string(linkBody("link-synthetic", "session-synthetic", status)), "}") +
			`,"public_token":"deprecated-synthetic","public_tokens":["public-synthetic"],"user_id":"foreign-user"}`)
		got, e := v.Verify(context.Background(), signBody(t, private, key.ID, body, now.Unix()), body)
		if e != nil || got.ItemID != "" || got.session == nil || got.session.linkHash != tokenHash("link-synthetic") ||
			got.session.sessionHash != tokenHash("session-synthetic") || got.session.status != status {
			t.Fatal("wrong hashed operational projection")
		}
		encoded, _ := json.Marshal(got)
		for _, secret := range []string{"link-synthetic", "session-synthetic", "deprecated-synthetic", "public-synthetic", "foreign-user"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatal("raw metadata retained")
			}
		}
		count := 1
		h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) {
			return Receipt{State: "accepted", sessionCount: &count}, nil
		}))
		if invokeHandler(h, signBody(t, private, key.ID, body, now.Unix()), body).Code != 204 {
			t.Fatal("supported event rejected")
		}
	}
}

// TestHostedRepairOptionalProfile enforces closed bounds without assuming update has an empty token array.
func TestHostedRepairOptionalProfile(t *testing.T) {
	base := strings.TrimSuffix(string(linkBody("link-synthetic", "session-synthetic", "success")), "}")
	for _, suffix := range []string{`,"public_token":null}`, `,"public_tokens":null}`, `,"user_id":null}`,
		`,"public_tokens":[null]}`, `,"public_tokens":[""]}`, `,"user_id":""}`, `,"public_token":""}`,
		`,"public_tokens":{}}`, `,"public_tokens":[` + strings.Repeat(`"token",`, 25) + `"token"]}`} {
		if _, e := parseLinkSession([]byte(base + suffix)); e == nil {
			t.Fatal("invalid optional profile accepted")
		}
	}
	for _, suffix := range []string{`}`, `,"public_tokens":[]}`, `,"public_tokens":["token"]}`} {
		if _, e := parseLinkSession([]byte(base + suffix)); e != nil {
			t.Fatal("valid optional profile rejected")
		}
	}
}

// TestHostedRepairReceiptKind prevents an injected receiver from acknowledging the other notification kind.
func TestHostedRepairReceiptKind(t *testing.T) {
	private, key, now := fixture(t)
	count := 0
	for _, tc := range []struct {
		body    []byte
		receipt Receipt
	}{
		{linkBody("link-synthetic", "session-synthetic", "success"), Receipt{State: "accepted", ReceiptID: actorFixtureID, StreamCount: 1}},
		{bodyFixture(), Receipt{State: "ignored", sessionCount: &count}},
	} {
		v, _ := verifyFixture(t, key, now)
		h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) { return tc.receipt, nil }))
		if invokeHandler(h, signBody(t, private, key.ID, tc.body, now.Unix()), tc.body).Code != 503 {
			t.Fatal("mixed receipt acknowledged")
		}
	}
	for _, state := range []string{"accepted", "duplicate", "ignored", "unknown"} {
		for n := -1; n <= 2; n++ {
			want := (state == "accepted" || state == "duplicate") && n == 1 || state == "ignored" && n == 0
			if validReceipt(Receipt{State: state, sessionCount: &n}) != want {
				t.Fatal("invalid receipt closure")
			}
		}
	}
}

const actorFixtureID = "00000000-0000-4000-8000-000000000001"
