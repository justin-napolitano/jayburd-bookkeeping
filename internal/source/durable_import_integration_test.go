//go:build integration

package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

const importProfile = "jayburd-synthetic-statement-csv/1"
const importRequest = "00000000-0000-4000-8000-000000000041"

// importObject inspects only synthetic function results, never private source data.
func importObject(t testing.TB, s string) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// importSetup exercises audited provisioning with an actual owner-only login.
func importSetup(t *testing.T) (*pgx.Conn, string, string, string) {
	c, db := fixture(t)
	sid, receipt := enroll(t, db)
	nid := importObject(t, receipt)["namespace_id"].(string)
	_, err := mutate(t, db, sid, "set_source_import_permission", "grant-import", map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": true, "entity_import_enabled": true})
	if err != nil {
		t.Fatal(err)
	}
	return c, db, sid, nid
}

// importManifest fixes identity and interval independently from the publication implementation.
func importManifest(sid, nid, ref string) map[string]any {
	return map[string]any{"source_account_id": sid, "namespace_id": nid, "binding_revision": "2", "statement_ref": ref, "starts_on": "2026-01-01", "ends_before": "2026-02-01", "profile": importProfile}
}
func importRegister(t testing.TB, db, sid, nid, ref string) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.register_import_batch($1,$2,$3,$4,$5)::text", reader, entity, book, ref, raw(importManifest(sid, nid, ref))))
}

// importRow contains deliberately identical purchases except their authoritative IDs.
func importRow(id, amount string, record int) map[string]any {
	return map[string]any{"record": record, "source_account_ref": "SYNTH-CASH-01", "entry_id": id, "booked_on": "2026-01-05", "currency": "USD", "direction": "outflow", "amount": amount, "description": "Fictional supplies", "canonical_amount": "12.5", "currency_version_id": currency}
}
func importPublication(rows ...map[string]any) map[string]any {
	return map[string]any{"profile": importProfile, "sha256": strings.Repeat("a", 64), "byte_count": 400, "rows": rows}
}
func importPublish(t testing.TB, db string, batch map[string]any, v any) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], raw(v)))
}
func importCommand(t testing.TB, db, key string, p map[string]any) map[string]any {
	t.Helper()
	preview := p["preview"].(map[string]any)
	body := map[string]any{"batch_id": p["batch_id"], "preview_id": preview["preview_id"], "preview_revision": preview["preview_revision"], "binding_revision": preview["binding_revision"], "source_revision": preview["source_revision"], "profile": importProfile}
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT ledger.register_command($1,$2,$3,$4)::text", reader, key, importRequest, raw(map[string]any{"operation": "import", "entity_id": entity, "book_id": book, "body": body})))
}
func importTransition(t testing.TB, db string, c map[string]any, action string) (string, error) {
	t.Helper()
	return invoke(t, db, "synthetic_runtime", "SELECT ledger.transition_command($1,$2,$3,$4,'{}')::text", reader, c["command_id"], importRequest, action)
}
func importFinish(t testing.TB, db, key string, p map[string]any) map[string]any {
	t.Helper()
	c := importCommand(t, db, key, p)
	result, err := importTransition(t, db, c, "execute")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = importTransition(t, db, c, "acknowledge"); err != nil {
		t.Fatal(err)
	}
	return importObject(t, result)
}

// importNoFinance checks independent SQL facts, including unchanged revision/outbox.
func importNoFinance(t testing.TB, c *pgx.Conn, revision string) {
	t.Helper()
	if got := value(t, c, "SELECT revision::text FROM ledger.book"); got != revision {
		t.Fatalf("book changed %s to %s", revision, got)
	}
	for _, table := range []string{"ledger.journal_entry", "ledger.journal_line", "integration.outbox_message"} {
		if got := value(t, c, "SELECT count(*)::text FROM "+table); got != "0" {
			t.Fatalf("financial effect in %s", table)
		}
	}
}

// TestDurableImportIdentity proves full canonical equality and separate evidence occurrences.
func TestDurableImportIdentity(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	revision := value(t, c, "SELECT revision::text FROM ledger.book")
	batch := importRegister(t, db, sid, nid, "statement-one")
	again := importRegister(t, db, sid, nid, "statement-one")
	if batch["batch_id"] != again["batch_id"] || batch["object_id"] != again["object_id"] {
		t.Fatal("registration changed identity")
	}
	rows := importPublication(importRow("purchase-1", "0012.500", 2), importRow("purchase-2", "12.50", 3))
	p := importPublish(t, db, batch, rows)
	if p["created_count"] != "2" || p["existing_count"] != "0" {
		t.Fatalf("unexpected preview %v", p)
	}
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "0" {
		t.Fatal("preview created evidence")
	}
	result := importFinish(t, db, "confirm-one", p)
	if result["result"].(map[string]any)["created_count"] != "2" {
		t.Fatal("distinct identical purchases lost")
	}
	replay := importPublish(t, db, batch, rows)
	if replay["state"] != "imported" {
		t.Fatal("imported replay changed")
	}
	second := importRegister(t, db, sid, nid, "statement-two")
	rows2 := importPublication(importRow("purchase-1", "12.5", 2), importRow("purchase-2", "00012.5000", 3))
	p2 := importPublish(t, db, second, rows2)
	if p2["created_count"] != "0" || p2["existing_count"] != "2" {
		t.Fatalf("cross-file duplicate %v", p2)
	}
	importFinish(t, db, "confirm-two", p2)
	for table, want := range map[string]string{"banking.source_observation": "2", "banking.observation_revision": "2", "banking.import_row": "4", "banking.review_item": "2", "banking.review_decision": "2", "banking.review_assignment": "2"} {
		if got := value(t, c, "SELECT count(*)::text FROM "+table); got != want {
			t.Fatalf("%s %s != %s", table, got, want)
		}
	}
	changed := importRow("purchase-1", "12.51", 2)
	changed["canonical_amount"] = "12.51"
	third := importRegister(t, db, sid, nid, "correction")
	rejected := importPublish(t, db, third, importPublication(changed, importRow("new-valid", "12.50", 3)))
	if rejected["state"] != "rejected" || rejected["error_count"] != float64(1) {
		t.Fatalf("correction not rejected %v", rejected)
	}
	// Equal digest cannot authorize replacement fields on an existing batch.
	_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], raw(importPublication(changed)))
	requireCode(t, err, "P2402")
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "2" {
		t.Fatal("partial correction import")
	}
	rid := value(t, c, "SELECT review_id::text FROM banking.review_item ORDER BY review_id LIMIT 1")
	review := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", reader, entity, book, rid))
	if review["state"] != "needs_review" || review["journal_id"] != nil {
		t.Fatal("unapproved bookkeeping")
	}
	importNoFinance(t, c, revision)
}

// TestDurableImportScope checks all current grants before replay or source disclosure.
func TestDurableImportScope(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	batch := importRegister(t, db, sid, nid, "scope")
	p := importPublish(t, db, batch, importPublication(importRow("p1", "12.50", 2)))
	command := importCommand(t, db, "scope-command", p)
	for _, principal := range []string{other, "00000000-0000-4000-8000-000000000099"} {
		_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.import_context($1,$2,$3,$4)::text", principal, entity, book, batch["batch_id"])
		requireCode(t, err, "P2401")
	}
	for _, mutation := range []string{"UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id='" + reader + "' AND permission='import'", "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id='" + reader + "' AND capability='import'", "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id='" + reader + "' AND capability='read'"} {
		exec(t, c, mutation)
		_, err := importTransition(t, db, command, "execute")
		requireCode(t, err, "P2401")
		_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.register_import_batch($1,$2,$3,'scope',$4)::text", reader, entity, book, raw(importManifest(sid, nid, "scope")))
		requireCode(t, err, "P2401")
		exec(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id=$1", reader)
		exec(t, c, "UPDATE banking.source_permission SET revoked_at=NULL WHERE principal_id=$1", reader)
	}
	exec(t, c, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", reader)
	_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.get_import_batch($1,$2,$3,$4)::text", reader, entity, book, batch["batch_id"])
	requireCode(t, err, "P2401")
	exec(t, c, "UPDATE ledger.principal SET disabled_at=NULL WHERE principal_id=$1", reader)
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	for _, sql := range []string{"DELETE FROM banking.import_batch", "UPDATE banking.source_account SET source_revision=9", "SELECT banking.execute_import(NULL,NULL,NULL,'{}')", "SELECT banking.set_source_import_permission(NULL,NULL,NULL,NULL,'x','{}')"} {
		_, err = runtime.Exec(context.Background(), sql)
		requireCode(t, err, "42501")
	}
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.get_import_batch($1,$2,$3,$4)::text", reader, entity, "00000000-0000-4000-8000-000000000099", batch["batch_id"])
	requireCode(t, err, "P2401")
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "0" {
		t.Fatal("denied execution wrote evidence")
	}
}

// TestDurableImportValidation verifies malformed output cannot bypass the decoder boundary.
func TestDurableImportValidation(t *testing.T) {
	_, db, sid, nid := importSetup(t)
	batch := importRegister(t, db, sid, nid, "validation")
	cases := []map[string]any{importRow("p", "12.501", 2), importRow("p", "0", 2), importRow("p", "1e2", 2), importRow("p", "12.50", 3)}
	badDate := importRow("p", "12.50", 2)
	badDate["booked_on"] = "2026-02-01"
	cases = append(cases, badDate)
	badCurrency := importRow("p", "12.50", 2)
	badCurrency["currency"] = "JPY"
	cases = append(cases, badCurrency)
	badScope := importRow("p", "12.50", 2)
	badScope["source_account_ref"] = "OTHER"
	cases = append(cases, badScope)
	badShape := importRow("p", "12.50", 2)
	badShape["extra"] = true
	cases = append(cases, badShape)
	for i, row := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], raw(importPublication(row)))
			requireCode(t, err, "P2400")
		})
	}
	for _, body := range []string{`{"profile":"x","profile":"x"}`, `{"x":1e1000000}`} {
		_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], body)
		requireCode(t, err, "P2400")
	}
	_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], raw(importPublication(importRow("same", "12.50", 2), importRow("same", "12.50", 3))))
	requireCode(t, err, "P2400")
	rejected := importPublish(t, db, batch, map[string]any{"profile": importProfile, "sha256": strings.Repeat("b", 64), "byte_count": 25, "diagnostics": []map[string]any{{"code": "invalid_header", "record": 1, "field": ""}}, "total": 1, "truncated": false})
	if rejected["state"] != "rejected" {
		t.Fatal("decoder rejection lost")
	}
}

// TestDurableImportRecovery requires explicit confirmation after a changed source snapshot.
func TestDurableImportRecovery(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	revision := value(t, c, "SELECT revision::text FROM ledger.book")
	one := importRegister(t, db, sid, nid, "first")
	two := importRegister(t, db, sid, nid, "second")
	pub := importPublication(importRow("p1", "12.50", 2))
	p1 := importPublish(t, db, one, pub)
	p2 := importPublish(t, db, two, pub)
	command := importCommand(t, db, "first-command", p1)
	// The existing one-unacknowledged-principal gate includes import.
	body := raw(map[string]any{"operation": "import", "entity_id": entity, "book_id": book, "body": command["body"]})
	_, err := invoke(t, db, "synthetic_runtime", "SELECT ledger.register_command($1,'blocked',$2,$3)::text", reader, importRequest, body)
	requireCode(t, err, "P2112")
	result, err := importTransition(t, db, command, "execute")
	if err != nil {
		t.Fatal(err)
	}
	again, err := importTransition(t, db, command, "execute")
	if err != nil || again != result {
		t.Fatal("commit replay changed")
	}
	if _, err = importTransition(t, db, command, "acknowledge"); err != nil {
		t.Fatal(err)
	}
	stale := importCommand(t, db, "stale", p2)
	_, err = importTransition(t, db, stale, "execute")
	requireCode(t, err, "P2403")
	refreshed := importPublish(t, db, two, pub)
	if refreshed["preview_revision"] != "2" || refreshed["existing_count"] != "1" {
		t.Fatal("refresh did not pin new snapshot")
	}
	_, err = importTransition(t, db, stale, "execute")
	requireCode(t, err, "P2403")
	// Refresh wins first: cancel the obsolete intent, preserve the new preview, then reconfirm.
	if _, err = importTransition(t, db, stale, "cancel"); err != nil {
		t.Fatal(err)
	}
	if _, err = importTransition(t, db, stale, "acknowledge"); err != nil {
		t.Fatal(err)
	}
	importFinish(t, db, "refreshed-confirmation", refreshed)
	// Current-preview cancellation wins first: the batch cannot later refresh or execute.
	third := importRegister(t, db, sid, nid, "cancel-current")
	p3 := importPublish(t, db, third, pub)
	cancelled := importCommand(t, db, "cancel-current", p3)
	if _, err = importTransition(t, db, cancelled, "cancel"); err != nil {
		t.Fatal(err)
	}
	_, err = importTransition(t, db, cancelled, "execute")
	requireCode(t, err, "P2113")
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, third["batch_id"], raw(pub))
	requireCode(t, err, "P2402")
	importNoFinance(t, c, revision)
}

// TestDurableImportAtomicity injects late failures into every effect family.
func TestDurableImportAtomicity(t *testing.T) {
	for _, table := range []string{"banking.source_observation", "banking.observation_revision", "banking.import_row", "banking.review_item", "banking.review_decision", "banking.review_evidence", "banking.review_assignment", "audit.command_action"} {
		t.Run(strings.ReplaceAll(table, ".", "_"), func(t *testing.T) {
			c, db, sid, nid := importSetup(t)
			revision := value(t, c, "SELECT revision::text FROM ledger.book")
			batch := importRegister(t, db, sid, nid, "atomic")
			p := importPublish(t, db, batch, importPublication(importRow("p1", "12.50", 2), importRow("p2", "12.50", 3)))
			command := importCommand(t, db, "atomic", p)
			exec(t, c, "CREATE FUNCTION public.fail_import_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION USING ERRCODE='P9999',MESSAGE='synthetic_fault'; END $$")
			exec(t, c, "CREATE TRIGGER inject_import_failure BEFORE INSERT ON "+table+" FOR EACH ROW EXECUTE FUNCTION public.fail_import_test()")
			_, err := importTransition(t, db, command, "execute")
			requireCode(t, err, "P9999")
			for _, check := range []string{"banking.source_observation", "banking.observation_revision", "banking.import_row", "banking.review_item", "banking.review_decision", "banking.review_evidence", "banking.review_assignment"} {
				if value(t, c, "SELECT count(*)::text FROM "+check) != "0" {
					t.Fatalf("partial %s", check)
				}
			}
			if value(t, c, "SELECT source_revision::text FROM banking.source_account") != "0" || value(t, c, "SELECT state FROM ledger.interactive_command") != "prepared" {
				t.Fatal("partial revision or receipt")
			}
			importNoFinance(t, c, revision)
			exec(t, c, "DROP TRIGGER inject_import_failure ON "+table)
			if _, err = importTransition(t, db, command, "execute"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestDurableImportConcurrency proves competing confirmation of one command converges.
func TestDurableImportConcurrency(t *testing.T) {
	c, db, sid, nid := importSetup(t)
	batch := importRegister(t, db, sid, nid, "race")
	p := importPublish(t, db, batch, importPublication(importRow("p1", "12.50", 2)))
	command := importCommand(t, db, "race", p)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	results := make(chan string, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			s, e := importTransition(t, db, command, "execute")
			errs <- e
			results <- s
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first string
	for result := range results {
		if first == "" {
			first = result
		} else if first != result {
			t.Fatal("race result differs")
		}
	}
	if value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "1" || value(t, c, "SELECT count(*)::text FROM banking.review_item") != "1" {
		t.Fatal("duplicate effect")
	}
}

// TestDurableImportMoneyAndBounds preserves currency units and tests the 1000-row boundary.
func TestDurableImportMoneyAndBounds(t *testing.T) {
	for _, tc := range []struct{ code, id, amount, canonical, bad string }{{"JPY", "00000000-0000-4000-8000-000000000006", "0012.000", "12", "12.1"}, {"KWD", "00000000-0000-4000-8000-000000000007", "0012.5010", "12.501", "12.5011"}} {
		t.Run(tc.code, func(t *testing.T) {
			c, db := fixture(t)
			v := manifest()
			v["currency_version_id"] = tc.id
			receipt := importObject(t, call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'money',$4)::text", actor, entity, book, raw(v)))
			sid := receipt["source_account_id"].(string)
			nid := receipt["namespace_id"].(string)
			_, err := mutate(t, db, sid, "set_source_import_permission", "grant", map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": true, "entity_import_enabled": true})
			if err != nil {
				t.Fatal(err)
			}
			batch := importRegister(t, db, sid, nid, "money")
			r := importRow("money", tc.bad, 2)
			r["currency"] = tc.code
			r["currency_version_id"] = tc.id
			r["canonical_amount"] = tc.bad
			_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], raw(importPublication(r)))
			requireCode(t, err, "P2400")
			r["amount"] = tc.amount
			r["canonical_amount"] = tc.canonical
			p := importPublish(t, db, batch, importPublication(r))
			importFinish(t, db, "money", p)
			if got := value(t, c, "SELECT trim_scale(amount)::text FROM banking.observation_revision"); got != tc.canonical {
				t.Fatal("source units changed", got)
			}
		})
	}
	t.Run("records", func(t *testing.T) {
		c, db, sid, nid := importSetup(t)
		batch := importRegister(t, db, sid, nid, "large")
		rows := make([]map[string]any, 1001)
		for i := range rows {
			rows[i] = importRow(fmt.Sprintf("entry-%04d", i), "12.50", i+2)
		}
		_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], raw(importPublication(rows...)))
		requireCode(t, err, "P2404")
		p := importPublish(t, db, batch, importPublication(rows[:1000]...))
		result := importFinish(t, db, "large", p)
		if result["result"].(map[string]any)["created_count"] != "1000" || value(t, c, "SELECT count(*)::text FROM banking.import_row") != "1000" {
			t.Fatal("1000-row atomic import failed")
		}
	})
}

// TestDurableImportLifecycle verifies bounded I/O admission and overlap gates.
func TestDurableImportLifecycle(t *testing.T) {
	t.Run("overlap", func(t *testing.T) {
		c, db, sid, nid := importSetup(t)
		first := importRegister(t, db, sid, nid, "first")
		pub := importPublication(importRow("shared", "12.50", 2))
		importFinish(t, db, "first", importPublish(t, db, first, pub))
		result, err := mutate(t, db, sid, "register_source_namespace", "second-ns", map[string]any{"expected_binding_revision": "2", "namespace": map[string]any{"profile": importProfile, "synthetic_dataset_id": "00000000-0000-4000-8000-000000000055", "source_account_ref": "SYNTH-CASH-01"}})
		if err != nil {
			t.Fatal(err)
		}
		nid2 := importObject(t, result)["namespace_id"].(string)
		m := importManifest(sid, nid2, "overlap")
		m["binding_revision"] = "3"
		batch := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.register_import_batch($1,$2,$3,'overlap',$4)::text", reader, entity, book, raw(m)))
		importFinish(t, db, "overlap", importPublish(t, db, batch, pub))
		if value(t, c, "SELECT count(*)::text FROM banking.review_item WHERE state='overlap_hold'") != "1" || value(t, c, "SELECT count(*)::text FROM banking.source_observation") != "2" {
			t.Fatal("cross-namespace auto-match")
		}
	})
	t.Run("expiry", func(t *testing.T) {
		c, db, sid, nid := importSetup(t)
		batch := importRegister(t, db, sid, nid, "expires")
		exec(t, c, "ALTER TABLE banking.import_batch DISABLE TRIGGER preserve_batch")
		exec(t, c, "UPDATE banking.import_batch SET created_at=clock_timestamp()-interval '8 days',expires_at=clock_timestamp()-interval '1 day'")
		exec(t, c, "ALTER TABLE banking.import_batch ENABLE TRIGGER preserve_batch")
		_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.import_context($1,$2,$3,$4)::text", reader, entity, book, batch["batch_id"])
		requireCode(t, err, "P2402")
		_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.publish_import_preview($1,$2,$3,$4,$5)::text", reader, entity, book, batch["batch_id"], raw(importPublication(importRow("p", "12.50", 2))))
		requireCode(t, err, "P2402")
	})
	t.Run("paused_replay", func(t *testing.T) {
		c, db, sid, nid := importSetup(t)
		batch := importRegister(t, db, sid, nid, "paused")
		p := importPublish(t, db, batch, importPublication(importRow("p", "12.50", 2)))
		command := importCommand(t, db, "paused", p)
		result, err := importTransition(t, db, command, "execute")
		if err != nil {
			t.Fatal(err)
		}
		_, err = mutate(t, db, sid, "set_source_state", "pause", map[string]any{"expected_binding_revision": "2", "state": "paused"})
		if err != nil {
			t.Fatal(err)
		}
		again, err := importTransition(t, db, command, "execute")
		if err != nil || result != again {
			t.Fatal("paused committed recovery failed")
		}
		exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE capability='import'")
		for _, action := range []string{"execute", "cancel", "acknowledge"} {
			_, err = importTransition(t, db, command, action)
			requireCode(t, err, "P2401")
		}
		_, err = invoke(t, db, "synthetic_runtime", "SELECT ledger.pending_command($1)::text", reader)
		requireCode(t, err, "P2401")
		// Read grant remains independently useful for historical review.
		if _, err = invoke(t, db, "synthetic_runtime", "SELECT banking.get_import_batch($1,$2,$3,$4)::text", reader, entity, book, batch["batch_id"]); err != nil {
			t.Fatal(err)
		}
	})
}
