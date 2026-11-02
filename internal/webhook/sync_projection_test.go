package webhook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/plaidprofile"
)

// TestSignedSyncAdditiveProjection authenticates original bytes and discards additive
// metadata without letting case-insensitive aliases change the authorized projection.
func TestSignedSyncAdditiveProjection(t *testing.T) {
	private, key, now := fixture(t)
	for _, profile := range []plaidprofile.Profile{plaidprofile.Sandbox(), productionProfile()} {
		t.Run(profile.Environment(), func(t *testing.T) {
			for _, extension := range []string{
				`,"synthetic_metadata":"synthetic-private-value"`,
				`,"synthetic_metadata":null`,
				`,"synthetic_metadata":{"nested":[1,true,null,{"value":"synthetic-private-value"}]}`,
				`,"synthetic_metadata":` + strings.Repeat("[", 31) + "0" + strings.Repeat("]", 31),
				`,"ITEM_ID":"foreign-item","ENVIRONMENT":"foreign","Webhook_Code":"ERROR","WEBHOOK_TYPE":"LINK","Historical_Update_Complete":null`,
			} {
				body := []byte(strings.TrimSuffix(strings.Replace(string(bodyFixture()), `"sandbox"`, fmt.Sprintf("%q", profile.Environment()), 1), "}") + extension + "}")
				v, err := NewVerifierForProfile(keyFunc(func(context.Context, string) (JWK, error) { return key, nil }), profile)
				if err != nil {
					t.Fatal(err)
				}
				v.now = func() time.Time { return now }
				calls := 0
				h, err := NewHandler(v, receiverFunc(func(_ context.Context, got Verified) (Receipt, error) {
					calls++
					sum := sha256.Sum256(body)
					if got.ItemID != "synthetic-item" || got.Environment != profile.Environment() || got.IssuedAt != now.Unix() || got.BodySHA256 != hex.EncodeToString(sum[:]) || !digest.MatchString(got.ReceiptKey) || strings.Contains(fmt.Sprintf("%+v", got), "private") {
						t.Fatal("additive fields changed the trusted projection or original digest")
					}
					return Receipt{State: "accepted", ReceiptID: "00000000-0000-4000-8000-000000000001", StreamCount: 1}, nil
				}))
				if err != nil {
					t.Fatal(err)
				}
				token := signBody(t, private, key.ID, body, now.Unix())
				if w := invokeHandler(h, token, body); w.Code != 204 || w.Body.Len() != 0 || calls != 1 {
					t.Fatal("authenticated additive notification was not acknowledged")
				}
				changed := []byte(strings.Replace(string(body), "synthetic-private-value", "tampered", 1))
				if string(changed) == string(body) {
					changed = append(append([]byte{}, body...), ' ')
				}
				if invokeHandler(h, token, changed).Code != 401 || calls != 1 {
					t.Fatal("tampered additive metadata reached SQL")
				}
			}
		})
	}
}

// TestSyncProjectionRejectsAmbiguity retains required routing, finite event,
// duplicate, type, original JSON and resource boundaries before persistence.
func TestSyncProjectionRejectsAmbiguity(t *testing.T) {
	private, key, now := fixture(t)
	base := string(bodyFixture())
	for _, raw := range []string{
		strings.TrimSuffix(base, "}") + `,"item_id":"other"}`,
		strings.TrimSuffix(base, "}") + `,"synthetic_metadata":1,"synthetic_metadata":2}`,
		strings.TrimSuffix(base, "}") + `,"synthetic_metadata":{"a":1,"a":2}}`,
		strings.TrimSuffix(base, "}") + `,"synthetic_metadata":[{"a":1,"\u0061":2}]}`,
		strings.TrimSuffix(base, "}") + `,"synthetic_metadata":` + strings.Repeat("[", 32) + "0" + strings.Repeat("]", 32) + "}",
		strings.TrimSuffix(base, "}") + `,"synthetic_metadata":}`, base + `{}`,
		strings.TrimSuffix(base, "}") + ",\"synthetic_metadata\":\"\xff\"}",
		strings.TrimSuffix(base, "}") + `,"error":null}`,
		strings.TrimSuffix(base, "}") + `,"new_transactions":4}`,
		strings.TrimSuffix(base, "}") + `,"user_id":null}`,
		strings.Replace(base, `"item_id":"synthetic-item",`, "", 1),
		strings.Replace(base, `"item_id":"synthetic-item"`, `"ITEM_ID":"synthetic-item"`, 1),
		strings.Replace(base, `"initial_update_complete":true`, `"initial_update_complete":"true"`, 1),
		strings.Replace(base, `"historical_update_complete":false`, `"historical_update_complete":null`, 1),
		strings.Replace(base, `"SYNC_UPDATES_AVAILABLE"`, `"DEFAULT_UPDATE"`, 1),
		strings.Replace(base, `"sandbox"`, `"production"`, 1), `[]`, `null`,
	} {
		v, _ := verifyFixture(t, key, now)
		h, _ := NewHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) {
			t.Fatal("ambiguous notification reached SQL")
			return Receipt{}, nil
		}))
		body := []byte(raw)
		if w := invokeHandler(h, signBody(t, private, key.ID, body, now.Unix()), body); w.Code != 401 || w.Body.String() != `{"status":"not_accepted"}` {
			t.Fatal("ambiguous notification acknowledged or reflected")
		}
	}
}
