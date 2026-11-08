package collectionactivation

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestApprovedInlineParity keeps the operator delivery choices on one exact
// target/hash/expiry validator, without making either delivery an identity proof.
func TestApprovedInlineParity(t *testing.T) {
	now := time.Now()
	v := authored(now)
	body, _ := json.Marshal(v)
	raw := envelope(body)
	inline, e := ParseApproved(raw, now)
	if e != nil {
		t.Fatal(e)
	}
	file, e := loadBytes(t, raw, now)
	if e != nil || !reflect.DeepEqual(inline, file) {
		t.Fatal("operator delivery changed approved intent")
	}
	// The parser retains no caller-owned bytes that can subsequently change intent.
	wantBody, wantHash := inline.body, inline.Digest()
	for i := range raw {
		raw[i] = 'x'
	}
	if inline.body != wantBody || inline.Digest() != wantHash || inline.value.Actor != a {
		t.Fatal("caller changed accepted plan")
	}
	// Exercise the byte limit with valid approval JSON, not merely invalid input.
	valid := envelope(body)
	atLimit := append(append([]byte{}, valid...), []byte(strings.Repeat(" ", 32768-len(valid)))...)
	if _, e := ParseApproved(atLimit, now); e != nil {
		t.Fatal("valid approval at byte boundary rejected")
	}
	if _, e := ParseApproved(append(atLimit, ' '), now); e == nil {
		t.Fatal("valid approval beyond byte boundary accepted")
	}
}

// TestApprovedInlineClosed proves caller identity, unknown metadata and malformed
// approval bytes cannot enter through the new non-file delivery mechanism.
func TestApprovedInlineClosed(t *testing.T) {
	now := time.Now()
	v := authored(now)
	body, _ := json.Marshal(v)
	good := envelope(body)
	expired := authored(now)
	expired.Expires = now.Add(-time.Second).UTC().Format(time.RFC3339)
	old, _ := json.Marshal(expired)
	foreign := authored(now)
	foreign.Project = "unapproved-project"
	other, _ := json.Marshal(foreign)
	for _, raw := range [][]byte{
		nil, []byte(`null`), []byte(strings.Repeat("x", 32769)), append(append([]byte{}, good...), []byte(` {}`)...),
		envelope([]byte(strings.Replace(string(body), `"actor_id":`, `"actor_id":"`+b+`","actor_id":`, 1))),
		envelope([]byte(strings.Replace(string(body), `"actor_id"`, `"Actor_Id"`, 1))),
		envelope([]byte(strings.TrimSuffix(string(body), "}") + `,"subject":"fictional-untrusted-subject"}`)),
		envelope(old), envelope(other),
		[]byte(strings.Replace(string(good), `"sha256":`, `"sha256":"unapproved","sha256":`, 1)),
		[]byte(strings.Replace(string(good), `"sha256":"`, `"sha256":"f`, 1)),
	} {
		p, e := ParseApproved(raw, now)
		if e == nil || p != nil || e.Error() != "activation rejected" {
			t.Fatal("ambiguous inline approval accepted or diagnostic leaked")
		}
		if _, e = loadBytes(t, raw, now); e == nil {
			t.Fatal("file/inline validation diverged")
		}
	}
}
