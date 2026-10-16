//go:build integration

package connections

import (
	"context"
	"encoding/base64"
	"testing"
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
	envelope := map[string]any{"profile": "jayburd-bank-aead-v1", "key_version": "projects/123456789/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1", "wrapped_key": enc([]byte("fictional-wrapped-key")), "nonce": enc(make([]byte, 12)), "ciphertext": enc(make([]byte, 32))}
	args := []any{sc.Actor, sc.Entity, sc.Book, value(c.CredentialRef), value(c.CredentialVersion), c.Manifest.ItemID, "sandbox"}
	get := "SELECT integration.get_bank_credential($1,$2,$3,$4,$5,$6,$7)::text"
	put := "SELECT integration.put_bank_credential($1,$2,$3,$4,$5,$6,$7,$8)::text"
	if proofObject(t, proofCall(t, db, "synthetic_runtime", get, args...))["envelope"] != nil {
		t.Fatal("missing row must be explicit")
	}
	first := proofCall(t, db, "synthetic_runtime", put, append(args, proofRaw(envelope))...)
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
