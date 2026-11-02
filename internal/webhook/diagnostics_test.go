package webhook

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// TestRejectionStages uses independently signed synthetic cases and retains exact public errors/results.
func TestRejectionStages(t *testing.T) {
	private, key, now := fixture(t)
	valid := bodyFixture()
	for _, tc := range []struct {
		name string
		body []byte
		kind Rejection
	}{
		{"valid", valid, Unclassified},
		{"json", []byte("{"), BodyJSONRejected},
		{"error", []byte(strings.TrimSuffix(string(valid), "}") + `,"error":null}`), SyncUnexpectedError},
		{"count", []byte(strings.TrimSuffix(string(valid), "}") + `,"new_transactions":5}`), SyncUnexpectedCount},
		{"null user", []byte(strings.TrimSuffix(string(valid), "}") + `,"user_id":null}`), SyncNullUser},
		{"missing initial", []byte(strings.Replace(string(valid), `"initial_update_complete":true,`, "", 1)), SyncMissingInitial},
		{"missing historical", []byte(strings.Replace(string(valid), `,"historical_update_complete":false`, "", 1)), SyncMissingHistorical},
		{"unknown", []byte(strings.TrimSuffix(string(valid), "}") + `,"synthetic-private-field":"synthetic-private-value"}`), SyncUnexpectedField},
		{"environment", []byte(strings.Replace(string(valid), `"sandbox"`, `"production"`, 1)), EnvironmentRejected},
		{"identifier", []byte(strings.Replace(string(valid), `"synthetic-item"`, `"synthetic private identifier"`, 1)), ItemRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, _ := verifyFixture(t, key, now)
			token := signBody(t, private, key.ID, tc.body, now.Unix())
			got, err, stage := v.verify(context.Background(), token, tc.body)
			plain, plainErr := v.Verify(context.Background(), token, tc.body)
			if stage != tc.kind || err != plainErr || !reflect.DeepEqual(got, plain) || (err == nil) != (stage == Unclassified) {
				t.Fatal("classification changed validation")
			}
			if strings.Contains(stage.String(), "private") || Rejection(255).String() != "unclassified" {
				t.Fatal("diagnostic reflected input")
			}
		})
	}
	for _, tc := range []struct {
		name, token string
		body        []byte
		stage       Rejection
	}{
		{"framing", "synthetic-private-token", valid, FramingRejected},
		{"header", signRaw(t, private, `{"alg":"HS256","kid":"synthetic-key"}`, `{}`), valid, HeaderRejected},
		{"issued type", signRaw(t, private, `{"alg":"ES256","kid":"synthetic-key"}`, `{"iat":"synthetic-private","request_body_sha256":"synthetic"}`), valid, IssuedAtTypeRejected},
		{"time", signBody(t, private, key.ID, valid, now.Unix()-301), valid, IssuedAtRejected},
		{"hash", signBody(t, private, key.ID, valid, now.Unix()), append(append([]byte{}, valid...), ' '), BodyHashRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, _ := verifyFixture(t, key, now)
			_, err, stage := v.verify(context.Background(), tc.token, tc.body)
			if err == nil || stage != tc.stage {
				t.Fatal("wrong rejection stage")
			}
		})
	}
	badKey := key
	badKey.Use = "synthetic-invalid-use"
	v, _ := verifyFixture(t, badKey, now)
	if _, err, stage := v.verify(context.Background(), signBody(t, private, key.ID, valid, now.Unix()), valid); err == nil || stage != KeyMetadataRejected {
		t.Fatal("key rejection misclassified")
	}
	other, _, _ := fixture(t)
	v, _ = verifyFixture(t, key, now)
	if _, err, stage := v.verify(context.Background(), signBody(t, other, key.ID, valid, now.Unix()), valid); err == nil || stage != SignatureRejected {
		t.Fatal("signature rejection misclassified")
	}
}

// TestObservedHandlerNeverPersistsRejected proves diagnostics are private and never grant receipt authority.
func TestObservedHandlerNeverPersistsRejected(t *testing.T) {
	_, key, now := fixture(t)
	v, _ := verifyFixture(t, key, now)
	var observed []Rejection
	h, err := NewObservedHandler(v, receiverFunc(func(context.Context, Verified) (Receipt, error) {
		t.Fatal("rejected input reached SQL")
		return Receipt{}, nil
	}), func(r Rejection) { observed = append(observed, r) })
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"", "synthetic-private-token"} {
		w := invokeHandler(h, token, []byte(`{"synthetic-private":"synthetic-private"}`))
		if w.Code != 401 || w.Body.String() != `{"status":"not_accepted"}` || strings.Contains(w.Body.String(), "private") {
			t.Fatal("public response changed")
		}
	}
	if !reflect.DeepEqual(observed, []Rejection{VerificationHeaderRejected, FramingRejected}) {
		t.Fatal("missing fixed classifications")
	}
	private, public, clock := fixture(t)
	acceptedVerifier, _ := verifyFixture(t, public, clock)
	receipts := 0
	accepted, _ := NewObservedHandler(acceptedVerifier, receiverFunc(func(context.Context, Verified) (Receipt, error) {
		receipts++
		return Receipt{State: "accepted", ReceiptID: "00000000-0000-4000-8000-000000000001", StreamCount: 1}, nil
	}), func(Rejection) { t.Fatal("accepted request emitted rejection data") })
	if invokeHandler(accepted, signBody(t, private, public.ID, bodyFixture(), clock.Unix()), bodyFixture()).Code != 204 || receipts != 1 {
		t.Fatal("observer changed accepted path")
	}
}
