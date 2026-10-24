//go:build integration

package webhook

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const repairKeyVersion = "projects/jayburd-synthetic-test/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1"

// repairCredentialWrapper authenticates a fictional KEK while counting prohibited fresh wrapping work.
type repairCredentialWrapper struct {
	t              *testing.T
	pool           *lossPool
	wraps, unwraps int
}

// Wrap is used only to prepare the encrypted owner fixture, never by repair reads.
func (w *repairCredentialWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	w.wraps++
	b, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(b)
	return repairKeyVersion, g.Seal(nil, make([]byte, 12), key, aad), nil
}

// Unwrap proves the production adapter closes its restricted SQL transaction before key work.
func (w *repairCredentialWrapper) Unwrap(_ context.Context, version string, key, aad []byte) ([]byte, error) {
	if w.pool.active.Load() != 0 {
		w.t.Fatal("key unwrap inside SQL transaction")
	}
	w.unwraps++
	if version != repairKeyVersion {
		return nil, bankcredential.ErrUnavailable
	}
	b, _ := aes.NewCipher(bytes.Repeat([]byte{7}, 32))
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, make([]byte, 12), key, aad)
}

// repairCredentialFixture reuses the real owned lifecycle and seeds exactly one existing encrypted version.
// The owner installs only this synthetic envelope; all subsequent reads use production SQL/AEAD and a real API login.
func repairCredentialFixture(t *testing.T, deadline ...string) (*pgx.Conn, string, map[string]any, connections.RepairCredentialReader, connections.CredentialReference, *repairCredentialWrapper) {
	t.Helper()
	var c *pgx.Conn
	var db string
	var ready map[string]any
	if len(deadline) == 0 {
		c, db, ready = repairFixture(t)
	} else {
		c, db, ready = repairFixtureWithConsent(t, deadline[0])
	}
	linkAPI(t, db, "SELECT integration.abandon_connection($1,$2,$3,$4)::text", ready["attempt_id"])
	id := value(t, c, "SELECT stream_id::text FROM integration.sync_stream")
	a := linkAPI(t, db, "SELECT integration.begin_connection($1,$2,$3,'credential-repair',$4)::text", raw(map[string]any{
		"mode": "reconnect", "sync_item_id": value(t, c, "SELECT sync_item_id::text FROM integration.sync_item"),
		"expected_generation": value(t, c, "SELECT generation::text FROM integration.sync_item"), "approved_stream_ids": []any{id}}))
	runtime := connect(t, db, "synthetic_runtime")
	t.Cleanup(func() { runtime.Close(context.Background()) })
	pool := &lossPool{conn: runtime}
	w := &repairCredentialWrapper{t: t, pool: pool}
	ref := connections.CredentialReference{ReferenceID: a["previous_credential_ref"].(string), Version: a["previous_credential_version"].(string),
		ItemID: a["expected_item_id"].(string), Environment: "sandbox"}
	envelope, e := bankcredential.Seal(context.Background(), w, bankcredential.Binding{Entity: entity, Book: book, Reference: ref}, connections.NewSecret("synthetic-existing-access"))
	if e != nil {
		t.Fatal("synthetic envelope setup failed")
	}
	exec(t, c, `INSERT INTO integration.bank_credential(entity_id,book_id,credential_ref,credential_version,item_id,environment,envelope)
 VALUES($1,$2,$3,$4,$5,$6,$7::jsonb)`, entity, book, ref.ReferenceID, ref.Version, ref.ItemID, ref.Environment, raw(envelope))
	w.wraps = 0
	store, e := bankcredential.NewAPIStore(pool, w)
	if e != nil {
		t.Fatal("credential store composition failed")
	}
	r := store.ForScope(connections.Scope{Actor: actor, Entity: entity, Book: book}).(connections.RepairCredentialReader)
	return c, db, a, r, ref, w
}

// credentialLink retains the new owned token after the initial exact-purpose Link-creation read.
func credentialLink(t *testing.T, db string, a map[string]any) map[string]any {
	t.Helper()
	return linkAPI(t, db, "SELECT integration.record_link_session($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"],
		raw(map[string]any{"link_token_sha256": tokenHash("credential-link-synthetic"), "expires_at": "2099-01-01T00:00:00Z"}))
}

// TestHostedRepairCredentialPurpose proves pre-Link access, pending denial and signed-proof/claim recovery with unchanged ciphertext.
func TestHostedRepairCredentialPurpose(t *testing.T) {
	c, db, a, reader, ref, w := repairCredentialFixture(t)
	before := value(t, c, "SELECT envelope::text FROM integration.bank_credential")
	got, e := reader.ReadRepair(context.Background(), a["attempt_id"].(string), ref)
	if e != nil || got.Reference != ref || w.unwraps != 1 || w.wraps != 0 {
		t.Fatal("existing credential unavailable for explicit Link creation")
	}
	_ = got.Token.WithValue(func(s string) error {
		if s != "synthetic-existing-access" {
			t.Fatal("wrong existing token")
		}
		return nil
	})
	a = credentialLink(t, db, a)
	if _, e = reader.ReadRepair(context.Background(), a["attempt_id"].(string), ref); e == nil || w.unwraps != 1 {
		t.Fatal("pending completion unwrapped a bank credential")
	}
	h := signedLinkPostgres(t, db, "")
	if h.send(t, linkBody("credential-link-synthetic", "owned", "success")) != 204 {
		t.Fatal("owned signed receipt failed")
	}
	if _, e = reader.ReadRepair(context.Background(), a["attempt_id"].(string), ref); e == nil || w.unwraps != 1 {
		t.Fatal("unclaimed signed receipt unwrapped a bank credential")
	}
	a = linkAPI(t, db, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"],
		raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil}))
	if a["execute_provider"] != true {
		t.Fatal("proof-pinned claim lost its single permission")
	}
	// A lost claim/manifest reply is explicitly recoverable after the unclaimed Link window,
	// while the original source consent and current scope still authorize this exact claim.
	exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
	for range 2 {
		got, e = reader.ReadRepair(context.Background(), a["attempt_id"].(string), ref)
		if e != nil || got.Reference != ref {
			t.Fatal("exact claimed recovery failed")
		}
	}
	if w.wraps != 0 || w.unwraps != 3 || value(t, c, "SELECT count(*)::text FROM integration.bank_credential") != "1" ||
		value(t, c, "SELECT envelope::text FROM integration.bank_credential") != before ||
		value(t, c, "SELECT credential_version::text FROM integration.sync_item") != "1" ||
		value(t, c, "SELECT count(*)::text FROM integration.credential_publication") != "1" ||
		value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "0" ||
		value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" ||
		value(t, c, "SELECT count(*)::text FROM integration.outbox_message WHERE event_type='journal.posted'") != "0" {
		t.Fatal("read recovery wrote ciphertext, renewed consent or published an effect")
	}
	// Revocation still wins over the proof-pinned claim and skips key work on recovery.
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='connect'", actor)
	if _, e = reader.ReadRepair(context.Background(), a["attempt_id"].(string), ref); e == nil || w.unwraps != 3 {
		t.Fatal("revoked claim reached key unwrap")
	}
}

// TestHostedRepairCredentialCurrentScope denies stale/unclaimed/source-expired/wrong-address work before key access.
func TestHostedRepairCredentialCurrentScope(t *testing.T) {
	for _, change := range []string{"actor", "foreign_actor", "foreign_book", "wrong_attempt", "source", "consent", "claimed_consent", "deadline", "address", "cancel"} {
		t.Run(change, func(t *testing.T) {
			var deadline []string
			if change == "consent" || change == "claimed_consent" {
				deadline = []string{time.Now().Add(5 * time.Second).UTC().Format(time.RFC3339Nano)}
			}
			c, db, a, r, ref, w := repairCredentialFixture(t, deadline...)
			attempt := a["attempt_id"].(string)
			claimed := false
			switch change {
			case "actor":
				exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND permission='connect'", actor)
			case "source":
				exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='import'", reader)
			case "foreign_actor", "foreign_book":
				scope := connections.Scope{Actor: actor, Entity: entity, Book: book}
				if change == "foreign_actor" {
					scope.Actor = other
				} else {
					scope.Book = csvNS
				}
				store, _ := bankcredential.NewAPIStore(w.pool, w)
				r = store.ForScope(scope).(connections.RepairCredentialReader)
			case "wrong_attempt":
				attempt = csvNS
			case "consent", "claimed_consent":
				if change == "claimed_consent" {
					a = credentialLink(t, db, a)
					h := signedLinkPostgres(t, db, "")
					if h.send(t, linkBody("credential-link-synthetic", "owned", "success")) != 204 {
						t.Fatal("owned signed receipt failed")
					}
					linkAPI(t, db, "SELECT integration.claim_connection($1,$2,$3,$4,$5,$6)::text", a["attempt_id"], a["nonce"],
						raw(map[string]any{"link_token_sha256": a["link_token_sha256"], "public_token_sha256": nil}))
					claimed = true
				}
				exec(t, c, "SELECT pg_sleep(greatest(0,extract(epoch FROM consent_expires_at-clock_timestamp()))+0.05) FROM integration.sync_item")
			case "deadline":
				exec(t, c, "UPDATE integration.connection_attempt SET expires_at=clock_timestamp()-interval '1 second' WHERE attempt_id=$1", a["attempt_id"])
			case "address":
				ref.ItemID = "another-item"
			case "cancel":
				linkAPI(t, db, "SELECT integration.abandon_connection($1,$2,$3,$4)::text", a["attempt_id"])
			}
			if _, e := r.ReadRepair(context.Background(), attempt, ref); e == nil || w.unwraps != 0 || w.wraps != 0 {
				t.Fatal("denied scope or wrong addressed result reached key authority")
			}
			wantClaims := "0"
			if claimed {
				wantClaims = "1"
			}
			if value(t, c, "SELECT count(*)::text FROM integration.bank_credential") != "1" ||
				value(t, c, "SELECT count(*)::text FROM integration.link_repair_claim") != wantClaims ||
				value(t, c, "SELECT count(*)::text FROM integration.sync_wakeup") != "0" ||
				value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
				t.Fatal("denied read changed state")
			}
		})
	}
}
