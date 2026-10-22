//go:build integration

package acquisition

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxverification"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// These fixed expectations are authored independently of the adapter, SQL and provider fixture constructors.
//
//go:embed testdata/verification/lifecycle.json
var verificationFixture []byte

//go:embed testdata/verification/identity-map.json
var verificationIdentityMap []byte

const verifierLogin = "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"

// verificationSetup publishes an original synthetic lifecycle through the real HTTP/decoder/SQL boundary.
func verificationSetup(t *testing.T) (*pgx.Conn, string, map[string]any) {
	t.Helper()
	owner, db, stream := setup(t)
	exec(t, owner, "REVOKE ALL ON DATABASE "+pgx.Identifier{db}.Sanitize()+" FROM PUBLIC")
	exec(t, owner, "GRANT CONNECT ON DATABASE "+pgx.Identifier{db}.Sanitize()+" TO synthetic_operator,synthetic_runtime,synthetic_sync,"+pgx.Identifier{verifierLogin}.Sanitize())
	parent := "Pending-A"
	pages := [][]byte{
		lifecyclePage("C1", false, []json.RawMessage{lifecycleRow("Pending-A", "10.00", "2026-01-02", true, nil), lifecycleRow("Hold-B", "9.50", "2026-01-02", true, nil), lifecycleRow("Correct-C", "-0.20", "2026-01-03", false, nil)}, nil, nil),
		lifecyclePage("C2", true, []json.RawMessage{lifecycleRow("Booked-A", "12.50", "2026-01-04", false, &parent), lifecycleRow("Unmatched-D", "-1.25e2", "2026-01-04", false, nil), lifecycleRow("Twin-D", "-1.25e2", "2026-01-04", false, nil)}, nil, nil),
		lifecyclePage("C3", false, nil, []json.RawMessage{lifecycleRow("Correct-C", "-0.21", "2026-01-05", false, nil)}, []string{"Pending-A", "Hold-B"}),
	}
	n := 0
	runner, _, calls := realRunner(t, db, "", func(w http.ResponseWriter, r *http.Request) {
		providerRequest(t, r)
		if n >= len(pages) {
			t.Error("unexpected fixture fetch")
			return
		}
		w.Write(pages[n])
		n++
	})
	scope := Scope{entity, book, stream["stream_id"].(string)}
	if _, e := runner.Run(context.Background(), scope, claimID); e != nil {
		t.Fatal("first fixture acquisition", e)
	}
	call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'verification')::text", entity, book, scope.StreamID)
	if _, e := runner.Run(context.Background(), scope, "00000000-0000-4000-8000-000000000085"); e != nil {
		t.Fatal("second fixture acquisition", e)
	}
	if _, e := runner.Run(context.Background(), scope, "00000000-0000-4000-8000-000000000085"); e != nil || *calls != 3 {
		t.Fatal("fixture replay")
	}
	return owner, db, stream
}

// enrollVerification pins independent raw-byte digests through the actual owner-only metadata operation.
func enrollVerification(t *testing.T, owner *pgx.Conn, db string, stream map[string]any, key string, fixture, identities []byte) string {
	t.Helper()
	fh := sha256.Sum256(fixture)
	ih := sha256.Sum256(identities)
	id := value(t, owner, "SELECT gen_random_uuid()::text")
	body := map[string]any{"mandate_id": id, "source_account_id": sourceID, "namespace_id": stream["namespace_id"], "stream_id": stream["stream_id"], "sync_item_id": value(t, owner, "SELECT sync_item_id::text FROM integration.sync_stream"), "verifier_login": verifierLogin, "binding_revision": value(t, owner, "SELECT binding_revision::text FROM banking.source_account"), "item_generation": "1", "target_generation": "2", "fixture_id": "lifecycle-1", "fixture_sha256": hex.EncodeToString(fh[:]), "identity_map_sha256": hex.EncodeToString(ih[:]), "expires_at": time.Now().UTC().Add(30 * time.Minute).Format("2006-01-02T15:04:05Z")}
	request := raw(body)
	call(t, db, "synthetic_operator", "SELECT integration.enroll_verification($1,$2,$3,$4,$5)::text", actor, entity, book, key, request)
	// Exact enrollment replay must not append another mandate or change the pins.
	call(t, db, "synthetic_operator", "SELECT integration.enroll_verification($1,$2,$3,$4,$5)::text", actor, entity, book, key, request)
	return id
}

// compareVerification invokes only the externally granted function through its actual restricted login.
func compareVerification(t *testing.T, db, id string, fixture, identities []byte) (string, error) {
	t.Helper()
	return invoke(t, db, verifierLogin, "SELECT integration.verify_sandbox_fixture($1,1,$2,$3)::text", id, string(fixture), string(identities))
}

// TestSandboxVerificationExact proves history/current review/lineage equality without leaking financial rows.
func TestSandboxVerificationExact(t *testing.T) {
	owner, db, stream := verificationSetup(t)
	id := enrollVerification(t, owner, db, stream, "exact", verificationFixture, verificationIdentityMap)
	out, e := compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	if e != nil {
		t.Fatal(e)
	}
	got := object(t, out)
	if got["equal"] != true || got["no_automatic_journals"] != true {
		t.Fatal("complete fixed fixture failed", out)
	}
	actualVerifier := connect(t, db, verifierLogin)
	defer actualVerifier.Close(context.Background())
	adapted, e := sandboxverification.Compare(context.Background(), actualVerifier, sandboxverification.Request{MandateID: id, Revision: "1", Fixture: verificationFixture, IdentityMap: verificationIdentityMap})
	if e != nil || !adapted.Equal || !adapted.NoAutomaticJournals {
		t.Fatal("actual bounded Go adapter failed", e)
	}
	for k, v := range got {
		if k == "equal" || k == "no_automatic_journals" {
			continue
		}
		if v != "0" {
			t.Fatal("nonzero fixed mismatch", k)
		}
	}
	for _, secret := range []string{"Pending-A", "10.00", "Fictional", "lifecycle-1", sourceID} {
		if strings.Contains(out, secret) {
			t.Fatal("verification exposed source metadata")
		}
	}
	if _, e = compareVerification(t, db, id, bytes.Replace(verificationFixture, []byte("12.50"), []byte("12.51"), 1), verificationIdentityMap); e == nil {
		t.Fatal("unpinned probing candidate accepted")
	} else {
		requireCode(t, e, "P3100")
	}
	for _, tc := range []struct{ name, old, next string }{
		{"quantity", `"amount":"12.5"`, `"amount":"12.6"`},
		{"lexeme", `"12.50"`, `"12.500"`},
		{"currency", `"currency_code":"USD"`, `"currency_code":"JPY"`},
		{"version", `"currency_version_id":"00000000-0000-4000-8000-000000000005"`, `"currency_version_id":"00000000-0000-4000-8000-000000000006"`},
		{"date", `"booked_date":"2026-01-04"`, `"booked_date":"2026-01-06"`},
		{"state", `"source_state":"pending"`, `"source_state":"booked"`},
		{"direction", `"direction":"inflow"`, `"direction":"outflow"`},
		{"description", `"Fictional Correct-C"`, `"Fictional correction changed"`},
		{"review", `"review_state":"overlap_hold"`, `"review_state":"needs_review"`},
		{"lineage", `[{"pending_id":"pending","booked_id":"booked"}]`, `[]`},
	} {
		fixture := bytes.Replace(verificationFixture, []byte(tc.old), []byte(tc.next), 1)
		wrong := enrollVerification(t, owner, db, stream, tc.name, fixture, verificationIdentityMap)
		out, e = compareVerification(t, db, wrong, fixture, verificationIdentityMap)
		if e != nil {
			t.Fatal(tc.name, e)
		}
		if object(t, out)["equal"] != false {
			t.Fatal("mismatch escaped", tc.name)
		}
	}
	// Valid but incomplete/extra expectations must fail full-set equality, not merely pass a selected subset.
	for _, mode := range []string{"omit", "phantom"} {
		f := object(t, string(verificationFixture))
		im := object(t, string(verificationIdentityMap))
		rows := f["rows"].([]any)
		items := im["items"].([]any)
		if mode == "omit" {
			kept := []any{}
			for _, r := range rows {
				if r.(map[string]any)["logical_id"] != "unmatched" {
					kept = append(kept, r)
				}
			}
			f["rows"] = kept
			kept = []any{}
			for _, r := range items {
				if r.(map[string]any)["logical_id"] != "unmatched" {
					kept = append(kept, r)
				}
			}
			im["items"] = kept
		} else {
			row := map[string]any{}
			for k, v := range rows[len(rows)-1].(map[string]any) {
				row[k] = v
			}
			row["logical_id"] = "phantom"
			f["rows"] = append(rows, row)
			im["items"] = append(items, map[string]any{"logical_id": "phantom", "transaction_id": "Not-A-Real-Entry"})
		}
		fixture := []byte(raw(f))
		identities := []byte(raw(im))
		wrong := enrollVerification(t, owner, db, stream, mode, fixture, identities)
		mismatch, e := compareVerification(t, db, wrong, fixture, identities)
		if e != nil || object(t, mismatch)["equal"] != false {
			t.Fatal("complete set mismatch escaped", mode, e)
		}
		field := "missing_count"
		if mode == "omit" {
			field = "extra_count"
		}
		if object(t, mismatch)[field] != "1" {
			t.Fatal("wrong complete-set difference count", mode, mismatch)
		}
	}
	down, e := fs.ReadFile(migrations.Files, "000030_sandbox_verification.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	_, e = owner.Exec(context.Background(), string(down))
	requireCode(t, e, "P0001")
	exec(t, owner, "ROLLBACK")
	// Revocation changes current authority even for known successful inputs and receipts.
	call(t, db, "synthetic_operator", "SELECT integration.revoke_verification($1,1,'revoke')::text", id)
	call(t, db, "synthetic_operator", "SELECT integration.revoke_verification($1,1,'revoke')::text", id)
	_, e = compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	requireCode(t, e, "P3101")
	// Explicit controlled posting must invalidate the absolute-zero proof, with no baseline supplied.
	counterID := enrollVerification(t, owner, db, stream, "counter", verificationFixture, verificationIdentityMap)
	exec(t, owner, `INSERT INTO ledger.membership VALUES($1,$2,'post',NULL)`, entity, actor)
	policy := "00000000-0000-4000-8000-000000000091"
	exec(t, owner, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,clock_timestamp())`, entity, book, policy, currency, actor)
	exec(t, owner, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000092','2026-01-01','2026-02-01')`, entity, book)
	body := raw(map[string]any{"expected_revision": value(t, owner, "SELECT revision::text FROM ledger.book"), "policy_id": policy, "accounting_date": "2026-01-05", "description": "Fictional explicit counter proof", "lines": []any{map[string]any{"account_id": account, "side": "debit", "transaction_currency": "USD", "transaction_amount": "1", "functional_amount": "1", "valuation_basis": "identity"}, map[string]any{"account_id": "00000000-0000-4000-8000-000000000012", "side": "credit", "transaction_currency": "USD", "transaction_amount": "1", "functional_amount": "1", "valuation_basis": "identity"}}})
	call(t, db, "synthetic_runtime", "SELECT body::text FROM ledger.post_journal($1,$2,$3,'counter',$4,$5)", actor, entity, book, claimID, body)
	out, e = compareVerification(t, db, counterID, verificationFixture, verificationIdentityMap)
	if e != nil {
		t.Fatal(e)
	}
	got = object(t, out)
	if got["equal"] != false || got["no_automatic_journals"] != false || got["journal_count"] != "1" || got["journal_outbox_count"] != "1" {
		t.Fatal("explicit posting did not invalidate zero proof", out)
	}

}

// TestSandboxVerificationRejectsShapes proves bounded input and unsafe authority cannot widen a pinned mandate.
func TestSandboxVerificationRejectsShapes(t *testing.T) {
	owner, db, stream := verificationSetup(t)
	for n, fixture := range [][]byte{
		bytes.Replace(verificationFixture, []byte(`"profile":`), []byte(`"unknown":true,"profile":`), 1),
		bytes.Replace(verificationFixture, []byte(`"profile":`), []byte(`"profile":"duplicate","profile":`), 1),
		bytes.Replace(verificationFixture, []byte(`"revision":"1"`), []byte(`"revision":"0"`), 1),
		[]byte(strings.Repeat(" ", 2<<20)),
	} {
		id := enrollVerification(t, owner, db, stream, fmt.Sprintf("shape%d", n), fixture, verificationIdentityMap)
		_, e := compareVerification(t, db, id, fixture, verificationIdentityMap)
		requireCode(t, e, "P3100")
	}
	// Required profile/fixture IDs are typed strings; null must never become an unknown predicate.
	for _, field := range []string{"profile", "fixture_id"} {
		for _, target := range []string{"fixture", "map"} {
			fixture, identities := verificationFixture, verificationIdentityMap
			if target == "fixture" {
				v := object(t, string(fixture))
				v[field] = nil
				fixture = []byte(raw(v))
			} else {
				v := object(t, string(identities))
				v[field] = nil
				identities = []byte(raw(v))
			}
			id := enrollVerification(t, owner, db, stream, "null-"+target+"-"+field, fixture, identities)
			_, e := compareVerification(t, db, id, fixture, identities)
			requireCode(t, e, "P3100")
		}
	}
	largeFixture := object(t, string(verificationFixture))
	largeMap := object(t, string(verificationIdentityMap))
	rows := []any{}
	items := []any{}
	for n := 0; n < 1001; n++ {
		logical := fmt.Sprintf("bound-%d", n)
		rows = append(rows, map[string]any{"logical_id": logical, "revision": "1", "previous_revision": nil, "current": true, "source_state": "booked", "booked_date": "2026-01-04", "currency_code": "USD", "currency_version_id": currency, "direction": "outflow", "amount": "1", "provider_amount_lexeme": "1.00", "description": "Fictional bound", "review_state": "overlap_hold"})
		items = append(items, map[string]any{"logical_id": logical, "transaction_id": logical})
	}
	largeFixture["rows"] = rows
	largeFixture["pending_links"] = []any{}
	largeMap["items"] = items
	fixtureBytes := []byte(raw(largeFixture))
	mapBytes := []byte(raw(largeMap))
	largeID := enrollVerification(t, owner, db, stream, "array-bound", fixtureBytes, mapBytes)
	_, e := compareVerification(t, db, largeID, fixtureBytes, mapBytes)
	requireCode(t, e, "P3100")
	duplicate := bytes.Replace(verificationIdentityMap, []byte(`"logical_id":"hold"`), []byte(`"logical_id":"pending"`), 1)
	id := enrollVerification(t, owner, db, stream, "duplicate-map", verificationFixture, duplicate)
	_, e = compareVerification(t, db, id, verificationFixture, duplicate)
	requireCode(t, e, "P3100")
	id = enrollVerification(t, owner, db, stream, "authority", verificationFixture, verificationIdentityMap)
	for _, role := range []string{"bookkeeping_api", "pg_execute_server_program"} {
		exec(t, owner, "GRANT "+pgx.Identifier{role}.Sanitize()+" TO "+pgx.Identifier{verifierLogin}.Sanitize()+" WITH INHERIT TRUE, SET FALSE")
		_, e = compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
		requireCode(t, e, "P3101")
		exec(t, owner, "REVOKE "+pgx.Identifier{role}.Sanitize()+" FROM "+pgx.Identifier{verifierLogin}.Sanitize())
	}
	exec(t, owner, "GRANT SELECT (name) ON ledger.book TO "+pgx.Identifier{verifierLogin}.Sanitize())
	_, e = compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	requireCode(t, e, "P3101")
	exec(t, owner, "REVOKE SELECT (name) ON ledger.book FROM "+pgx.Identifier{verifierLogin}.Sanitize())
	exec(t, owner, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id=$1 AND capability='read'", actor)
	_, e = compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	requireCode(t, e, "P3101")
	exec(t, owner, "UPDATE banking.source_permission SET revoked_at=NULL WHERE principal_id=$1 AND capability='read'", actor)
	exec(t, owner, "UPDATE integration.sync_stream SET state='paused',stream_revision=stream_revision+1")
	_, e = compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	requireCode(t, e, "P3101")
	exec(t, owner, "UPDATE integration.sync_stream SET state='active',stream_revision=stream_revision+1")
	// Every call rechecks current principal, consent, source pins and mandate revision.
	_, e = invoke(t, db, verifierLogin, "SELECT integration.verify_sandbox_fixture($1,2,$2,$3)::text", id, string(verificationFixture), string(verificationIdentityMap))
	requireCode(t, e, "P3101")
	exec(t, owner, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", actor)
	_, e = compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	requireCode(t, e, "P3101")
	exec(t, owner, "UPDATE ledger.principal SET disabled_at=NULL WHERE principal_id=$1", actor)
	template := object(t, value(t, owner, "SELECT (body->'body')::text FROM integration.verification_operator_receipt WHERE idempotency_key='authority'"))
	for _, mode := range []string{"expired", "ttl", "foreign", "production"} {
		input := map[string]any{}
		for k, v := range template {
			input[k] = v
		}
		input["mandate_id"] = value(t, owner, "SELECT gen_random_uuid()::text")
		switch mode {
		case "expired":
			input["expires_at"] = time.Now().UTC().Add(-time.Minute).Format("2006-01-02T15:04:05Z")
		case "ttl":
			input["expires_at"] = time.Now().UTC().Add(2 * time.Hour).Format("2006-01-02T15:04:05Z")
		case "foreign":
			input["source_account_id"] = "00000000-0000-4000-8000-000000000099"
		case "production":
			input["verifier_login"] = "jayburd-dev-verifier@production-project.iam"
		}
		_, e = invoke(t, db, "synthetic_operator", "SELECT integration.enroll_verification($1,$2,$3,$4,$5)::text", actor, entity, book, mode, raw(input))
		requireCode(t, e, "P3101")
	}
	// Real expiry is not simulated by modifying the immutable expiry column.
	expiring := map[string]any{}
	for k, v := range template {
		expiring[k] = v
	}
	shortID := value(t, owner, "SELECT gen_random_uuid()::text")
	expiring["mandate_id"] = shortID
	expiring["expires_at"] = time.Now().UTC().Add(2 * time.Second).Format("2006-01-02T15:04:05Z")
	call(t, db, "synthetic_operator", "SELECT integration.enroll_verification($1,$2,$3,'short',$4)::text", actor, entity, book, raw(expiring))
	time.Sleep(2100 * time.Millisecond)
	_, e = compareVerification(t, db, shortID, verificationFixture, verificationIdentityMap)
	requireCode(t, e, "P3101")
	for _, query := range []string{"SELECT count(*) FROM banking.observation_revision", "SELECT integration.verification_authorize('00000000-0000-4000-8000-000000000001',1)", "SELECT integration.revoke_verification('00000000-0000-4000-8000-000000000001',1,'forbidden')"} {
		_, e = invoke(t, db, verifierLogin, query)
		requireCode(t, e, "42501")
	}
	c := connect(t, db, verifierLogin)
	defer c.Close(context.Background())
	if _, e = c.Exec(context.Background(), "SET ROLE bookkeeping_verifier"); e == nil {
		t.Fatal("capability role switch allowed")
	} else {
		requireCode(t, e, "42501")
	}
	if _, e = c.Exec(context.Background(), "SET ROLE "+pgx.Identifier{verifierLogin}.Sanitize()); e != nil {
		t.Fatal(e)
	}
	tx, e := c.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(context.Background(), "SELECT integration.verify_sandbox_fixture($1,1,$2,$3)", id, string(verificationFixture), string(verificationIdentityMap))
	requireCode(t, e, "P3101")
	_ = tx.Rollback(context.Background())
	call(t, db, "synthetic_sync", "SELECT integration.request_sync($1,$2,$3,'newer')::text", entity, book, stream["stream_id"])
	_, e = compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	requireCode(t, e, "P3101")
}

// TestSandboxVerificationReviewIdentity rejects collapsing two equal-valued observations into one review.
func TestSandboxVerificationReviewIdentity(t *testing.T) {
	owner, db, stream := verificationSetup(t)
	id := enrollVerification(t, owner, db, stream, "review-identity", verificationFixture, verificationIdentityMap)
	// Only this disposable owner fixture models a future bad merge; all foreign keys remain enforced.
	// The restricted verifier cannot update assignment/evidence rows or disable the immutable trigger.
	shared := value(t, owner, `SELECT a.review_id::text FROM banking.review_assignment a JOIN banking.source_observation o USING(entity_id,book_id,observation_id) WHERE o.external_entry_id='Unmatched-D'`)
	twin := value(t, owner, `SELECT observation_id::text FROM banking.source_observation WHERE external_entry_id='Twin-D'`)
	decision := value(t, owner, "SELECT gen_random_uuid()::text")
	exec(t, owner, `INSERT INTO banking.review_decision(entity_id,book_id,decision_id,review_id,expected_review_revision,kind,actor_id,reason,body) VALUES($1,$2,$3,$4,0,'initialize',$5,'provider_sync','{}')`, entity, book, decision, shared, actor)
	exec(t, owner, `INSERT INTO banking.review_evidence VALUES($1,$2,$3,$4,$5,1)`, entity, book, decision, twin, shared)
	exec(t, owner, "ALTER TABLE banking.review_assignment DISABLE TRIGGER immutable")
	exec(t, owner, `UPDATE banking.review_assignment SET review_id=$1,decision_id=$2 WHERE observation_id=$3`, shared, decision, twin)
	exec(t, owner, "ALTER TABLE banking.review_assignment ENABLE TRIGGER immutable")
	out, e := compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	if e != nil {
		t.Fatal(e)
	}
	got := object(t, out)
	if got["equal"] != false || got["review_mismatch_count"] == "0" || got["no_automatic_journals"] != true {
		t.Fatal("equal-valued source identities collapsed", out)
	}
}

// TestSandboxVerificationForeignHistory fails complete-source equality for another namespace's retained row.
func TestSandboxVerificationForeignHistory(t *testing.T) {
	owner, db, stream := verificationSetup(t)
	id := enrollVerification(t, owner, db, stream, "foreign-history", verificationFixture, verificationIdentityMap)
	// This fictional CSV row is outside the pinned Plaid namespace but inside the same authorized source.
	exec(t, owner, `BEGIN; INSERT INTO banking.source_observation(entity_id,book_id,observation_id,source_account_id,namespace_id,external_entry_id) VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000095','00000000-0000-4000-8000-000000000080','00000000-0000-4000-8000-000000000081','Other-Namespace'); INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields) VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000095',1,'booked','2026-01-05','00000000-0000-4000-8000-000000000005','outflow','1',1,'Fictional other namespace','{}'); COMMIT;`)
	out, e := compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
	if e != nil {
		t.Fatal(e)
	}
	got := object(t, out)
	if got["equal"] != false || got["extra_count"] != "1" {
		t.Fatal("foreign namespace was silently omitted", out)
	}
}

// TestSandboxVerificationPins checks binding/Item changes and finite consent using separate real scopes.
func TestSandboxVerificationPins(t *testing.T) {
	for _, mode := range []string{"binding", "item", "source", "consent"} {
		t.Run(mode, func(t *testing.T) {
			owner, db, stream := verificationSetup(t)
			id := enrollVerification(t, owner, db, stream, "pin", verificationFixture, verificationIdentityMap)
			switch mode {
			case "binding":
				exec(t, owner, "UPDATE banking.source_account SET binding_revision=binding_revision+1")
			case "item":
				exec(t, owner, "UPDATE integration.sync_item SET generation=generation+1")
			case "source":
				exec(t, owner, "UPDATE banking.source_account SET state='paused',binding_revision=binding_revision+1")
			case "consent":
				exec(t, owner, "UPDATE integration.sync_item SET generation=generation+1,consent_expires_at=clock_timestamp()+interval '10 seconds'")
			}
			_, e := compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
			requireCode(t, e, "P3101")
			if mode == "consent" {
				// Even with the new Item generation pinned, a mandate cannot outlive the shorter consent.
				body := object(t, value(t, owner, "SELECT (body->'body')::text FROM integration.verification_operator_receipt WHERE idempotency_key='pin'"))
				body["mandate_id"] = value(t, owner, "SELECT gen_random_uuid()::text")
				body["item_generation"] = "2"
				_, e = invoke(t, db, "synthetic_operator", "SELECT integration.enroll_verification($1,$2,$3,'beyond-consent',$4)::text", actor, entity, book, raw(body))
				requireCode(t, e, "P3101")
			}
		})
	}
}

// TestSandboxVerificationConcurrentRevoke proves a blocked comparison cannot pass a concurrently committed revoke.
func TestSandboxVerificationConcurrentRevoke(t *testing.T) {
	owner, db, stream := verificationSetup(t)
	id := enrollVerification(t, owner, db, stream, "concurrent", verificationFixture, verificationIdentityMap)
	operator := connect(t, db, "synthetic_operator")
	defer operator.Close(context.Background())
	tx, e := operator.BeginTx(context.Background(), pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(context.Background(), "SET LOCAL ROLE bookkeeping_owner"); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(context.Background(), "SELECT 1 FROM integration.verification_mandate WHERE mandate_id=$1 FOR UPDATE", id); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		_, e := compareVerification(t, db, id, verificationFixture, verificationIdentityMap)
		done <- e
	}()
	blocked := false
	for n := 0; n < 100; n++ {
		if value(t, owner, `SELECT count(*)::text FROM pg_stat_activity WHERE usename='jayburd-dev-verifier@jayburd-bookkeeping-dev.iam' AND datname=current_database() AND wait_event_type='Lock'`) != "0" {
			blocked = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("comparison did not reach mandate lock")
	}
	if _, e = tx.Exec(context.Background(), "SELECT integration.revoke_verification($1,1,'concurrent-revoke')", id); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(context.Background()); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-done:
		requireCode(t, e, "P3101")
	case <-time.After(5 * time.Second):
		t.Fatal("comparison did not resolve revoked mandate")
	}
}
