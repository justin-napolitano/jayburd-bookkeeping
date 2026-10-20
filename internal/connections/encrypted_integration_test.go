//go:build integration

package connections

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
)

// TestEncryptedCredentialSQL proves current scoped authority and immutable ciphertext-only persistence.
func TestEncryptedCredentialSQL(t *testing.T) {
	s, admin, db, sc, _ := creatorFixture(t)
	s.secrets.(*memorySecrets).lostPut = true
	out, e := s.Begin(context.Background(), sc, "encrypted-proof", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if e != nil {
		t.Fatal(e)
	}
	a := proofObject(t, string(out.Attempt))
	id := a["attempt_id"].(string)
	_, e = s.Execute(context.Background(), sc, id, ExecuteRequest{Nonce: out.Nonce, LinkToken: out.LinkToken, PublicToken: "synthetic-public"})
	if e == nil {
		t.Fatal("expected staged recovery")
	}
	c, e := s.context(context.Background(), sc, id)
	if e != nil || c.Manifest == nil {
		t.Fatal("missing staged manifest")
	}
	enc := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	keyVersion := "projects/jayburd-synthetic-test/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1"
	envelope := map[string]any{"profile": "jayburd-bank-aead-v1", "key_version": keyVersion, "wrapped_key": enc([]byte("fictional-wrapped-key")), "nonce": enc(make([]byte, 12)), "ciphertext": enc(make([]byte, 32))}
	args := []any{sc.Actor, sc.Entity, sc.Book, value(c.CredentialRef), value(c.CredentialVersion), c.Manifest.ItemID, "sandbox"}
	get := "SELECT integration.get_bank_credential($1,$2,$3,$4,$5,$6,$7)::text"
	put := "SELECT integration.put_bank_credential($1,$2,$3,$4,$5,$6,$7,$8)::text"
	if proofObject(t, proofCall(t, db, "synthetic_runtime", get, args...))["envelope"] != nil {
		t.Fatal("missing row must be explicit")
	}
	for _, invalid := range []string{
		strings.Replace(keyVersion, "jayburd-synthetic-test", "-invalid", 1),
		strings.Replace(keyVersion, "keyRings/jayburd-dev", "keyRings/other", 1),
		keyVersion + "0/extra",
	} {
		envelope["key_version"] = invalid
		if _, err := proofInvoke(t, db, "synthetic_runtime", put, append(args, proofRaw(envelope))...); err == nil {
			t.Fatal("invalid wrapping key metadata accepted")
		}
	}
	envelope["key_version"] = keyVersion
	first := proofCall(t, db, "synthetic_runtime", put, append(args, proofRaw(envelope))...)
	// A downgrade cannot strand a canonical envelope behind numeric-only writers.
	down, err := migrations.Files.ReadFile("000020_canonical_kms_project.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(context.Background(), string(down)); err == nil {
		t.Fatal("canonical envelope downgrade accepted")
	}
	proofExec(t, admin, "ROLLBACK")
	envelope["nonce"] = enc([]byte("differentone"))
	if got := proofCall(t, db, "synthetic_runtime", put, append(args, proofRaw(envelope))...); got != first {
		t.Fatal("concurrent version overwrote immutable winner")
	}
	if _, e := s.Abandon(context.Background(), sc, id); e == nil {
		t.Fatal("stored ciphertext abandoned")
	}
	for _, field := range []int{0, 1, 2, 3, 4, 5, 6} {
		wrong := append([]any(nil), args...)
		switch field {
		case 0:
			wrong[0] = reader
		case 1:
			wrong[1] = other
		case 2:
			wrong[2] = other
		case 3:
			wrong[3] = other
		case 4:
			wrong[4] = "2"
		case 5:
			wrong[5] = "other-item"
		case 6:
			wrong[6] = "production"
		}
		if _, e := proofInvoke(t, db, "synthetic_runtime", get, wrong...); e == nil {
			t.Fatal("foreign binding accepted", field)
		}
	}
	for _, query := range []string{"SELECT envelope::text FROM integration.bank_credential", "SELECT integration.worker_bank_credential($1,$2,$3,$4,1)::text"} {
		var args []any
		if query != "SELECT envelope::text FROM integration.bank_credential" {
			args = []any{sc.Entity, sc.Book, other, other}
		}
		if _, e := proofInvoke(t, db, "synthetic_runtime", query, args...); e == nil {
			t.Fatal("API bypassed capability boundary")
		}
	}
	if _, e := admin.Exec(context.Background(), "DELETE FROM integration.bank_credential"); e == nil {
		t.Fatal("immutable credential deleted")
	}
	proofExec(t, admin, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE entity_id=$1 AND principal_id=$2 AND permission='connect'", sc.Entity, sc.Actor)
	if _, e := proofInvoke(t, db, "synthetic_runtime", get, args...); e == nil {
		t.Fatal("revoked actor read ciphertext")
	}
}

// TestAbandonUnavailableCredential preserves claims while allowing an explicit fresh enrollment.
func TestAbandonUnavailableCredential(t *testing.T) {
	s, admin, db, sc, provider := creatorFixture(t)
	s.secrets.(*memorySecrets).lostPut = true
	out, e := s.Begin(context.Background(), sc, "abandon-proof", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if e != nil {
		t.Fatal(e)
	}
	id := proofObject(t, string(out.Attempt))["attempt_id"].(string)
	_, e = s.Execute(context.Background(), sc, id, ExecuteRequest{Nonce: out.Nonce, LinkToken: out.LinkToken, PublicToken: "synthetic-public"})
	if e == nil {
		t.Fatal("expected credential recovery")
	}
	if _, e := s.Abandon(context.Background(), Scope{Actor: reader, Entity: sc.Entity, Book: sc.Book}, id); e == nil {
		t.Fatal("foreign actor abandoned attempt")
	}
	for i := 0; i < 2; i++ {
		raw, e := s.Abandon(context.Background(), sc, id)
		if e != nil || proofObject(t, string(raw))["state"] != "abandoned" {
			t.Fatal("abandonment failed", e)
		}
	}
	if count(provider.calls, "exchange") != 1 {
		t.Fatal("abandonment repeated exchange")
	}
	page := proofObject(t, proofCall(t, db, "synthetic_runtime", "SELECT integration.list_connection_attempts($1,$2,$3,100,NULL)::text", sc.Actor, sc.Entity, sc.Book))
	if page["unfinished_count"] != "0" {
		t.Fatal("abandoned attempt blocks fresh enrollment")
	}
	if _, e := admin.Exec(context.Background(), "UPDATE integration.connection_attempt SET state='credential_recovery_required' WHERE attempt_id=$1", id); e == nil {
		t.Fatal("abandoned audit state reactivated")
	}
}

// TestCancelUnclaimedSetup proves durable cancellation fences later completion,
// remains idempotent, and releases enrollment without any provider exchange.
func TestCancelUnclaimedSetup(t *testing.T) {
	s, _, db, sc, provider := creatorFixture(t)
	out, err := s.Begin(context.Background(), sc, "cancel-proof", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	id := proofObject(t, string(out.Attempt))["attempt_id"].(string)
	if _, err = s.Abandon(context.Background(), Scope{Actor: reader, Entity: sc.Entity, Book: sc.Book}, id); err == nil {
		t.Fatal("foreign cancellation accepted")
	}
	for i := 0; i < 2; i++ {
		raw, e := s.Abandon(context.Background(), sc, id)
		if e != nil || proofObject(t, string(raw))["state"] != "abandoned" {
			t.Fatal("cancel failed", e)
		}
	}
	_, err = s.Execute(context.Background(), sc, id, ExecuteRequest{Nonce: out.Nonce, LinkToken: out.LinkToken, PublicToken: "synthetic-public"})
	if err == nil {
		t.Fatal("canceled attempt exchanged")
	}
	if count(provider.calls, "exchange") != 0 {
		t.Fatal("cancellation called provider exchange")
	}
	page := proofObject(t, proofCall(t, db, "synthetic_runtime", "SELECT integration.list_connection_attempts($1,$2,$3,100,NULL)::text", sc.Actor, sc.Entity, sc.Book))
	if page["unfinished_count"] != "0" {
		t.Fatal("canceled setup blocks enrollment")
	}
	next, err := s.Begin(context.Background(), sc, "after-cancel", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	nextID := proofObject(t, string(next.Attempt))["attempt_id"].(string)
	if nextID == id {
		t.Fatal("reused canceled setup")
	}
	_, err = s.Execute(context.Background(), sc, nextID, ExecuteRequest{Nonce: next.Nonce, LinkToken: next.LinkToken, PublicToken: "synthetic-public"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Abandon(context.Background(), sc, nextID); err != nil {
		t.Fatal("binding review could not be canceled", err)
	}
	if count(provider.calls, "remove") != 1 {
		t.Fatal("binding cancellation did not remove provider Item")
	}
}

// TestCancelVersusClaim uses competing real SQL sessions; exactly one transition wins.
func TestCancelVersusClaim(t *testing.T) {
	s, _, db, sc, _ := creatorFixture(t)
	out, err := s.Begin(context.Background(), sc, "cancel-race", BeginRequest{Mode: "new", ApprovedStreamIDs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	id := proofObject(t, string(out.Attempt))["attempt_id"].(string)
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, e := proofInvoke(t, db, "synthetic_runtime", "SELECT integration.abandon_connection($1,$2,$3,$4)::text", sc.Actor, sc.Entity, sc.Book, id)
		results <- e
	}()
	go func() {
		<-start
		_, e := proofInvoke(t, db, "synthetic_runtime", "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", sc.Actor, sc.Entity, sc.Book, id, out.Nonce, proofRaw(map[string]any{"link_token_sha256": hash(out.LinkToken), "public_token_sha256": hash("synthetic-public")}))
		results <- e
	}()
	close(start)
	first, second := <-results, <-results
	if (first == nil) == (second == nil) {
		t.Fatal("exactly one cancellation or claim must win", first, second)
	}
}
