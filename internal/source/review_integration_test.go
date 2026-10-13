//go:build integration

package source

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

const reviewPolicy = "00000000-0000-4000-8000-000000000060"
const reviewExpense = "00000000-0000-4000-8000-000000000051"
const reviewRevenue = "00000000-0000-4000-8000-000000000052"

// reviewSetup creates fictional policy/accounts and explicitly grants prepare/post/review.
func reviewSetup(t *testing.T) (*pgx.Conn, string, string) {
	c, db, sid, nid := importSetup(t)
	exec(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL),($1,$2,'post',NULL)`, entity, reader)
	exec(t, c, `INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())`, entity, book, reviewPolicy, currency, actor)
	exec(t, c, `INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES($1,$2,'00000000-0000-4000-8000-000000000061','2026-01-01','2026-02-01')`, entity, book)
	exec(t, c, `INSERT INTO ledger.account VALUES($1,$2,$3,'6000','Fictional supplies','expense','debit',false,'2026-01-01',NULL),($1,$2,$4,'4000','Fictional sales','revenue','credit',false,'2026-01-01',NULL)`, entity, book, reviewExpense, reviewRevenue)
	batch := importRegister(t, db, sid, nid, "review-fixture")
	out := importRow("outflow", "12.50", 2)
	in := importRow("inflow", "12.50", 3)
	in["direction"] = "inflow"
	importFinish(t, db, "review-import", importPublish(t, db, batch, importPublication(out, in)))
	_, err := mutate(t, db, sid, "set_source_review_permission", "review-grant", map[string]any{"expected_binding_revision": "2", "principal_id": reader, "enabled": true})
	if err != nil {
		t.Fatal(err)
	}
	return c, db, sid
}

// reviewDetail and reviewInput consume only the public SQL projection.
func reviewDetail(t testing.TB, db, id string) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", reader, entity, book, id))
}
func reviewInput(detail map[string]any, category string) map[string]any {
	return map[string]any{"expected_review_revision": detail["review_revision"], "source_revision": detail["source_revision"], "binding_revision": detail["binding_revision"], "expected_book_revision": detail["book_revision"], "category_account_id": category, "policy_id": reviewPolicy, "accounting_date": detail["booked_date"], "description": "Explicit fictional category"}
}
func reviewDraft(t testing.TB, db, id, key, category string) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,$5,$6)::text", reader, entity, book, id, key, raw(reviewInput(reviewDetail(t, db, id), category))))
}
func reviewCommandBody(d map[string]any) map[string]any {
	return map[string]any{"review_id": d["review_id"], "draft_id": d["draft_id"], "review_revision": d["review_revision"], "source_revision": d["source_revision"], "binding_revision": d["binding_revision"], "expected_book_revision": d["book_revision"], "posting_body": d["posting_body"]}
}
func reviewCommand(t testing.TB, db, key string, d map[string]any) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT ledger.register_command($1,$2,$3,$4)::text", reader, key, importRequest, raw(map[string]any{"operation": "source_post", "entity_id": entity, "book_id": book, "body": reviewCommandBody(d)})))
}
func reviewID(t testing.TB, c *pgx.Conn, direction string) string {
	t.Helper()
	return value(t, c, `SELECT a.review_id::text FROM banking.review_assignment a JOIN banking.observation_revision q USING(entity_id,book_id,observation_id) WHERE q.direction=$1`, direction)
}
func reviewList(t testing.TB, db, sid string, limit int, cursor any) (string, error) {
	t.Helper()
	return invoke(t, db, "synthetic_runtime", "SELECT banking.list_review_items($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"source_account_id": sid, "state": nil, "limit": limit, "cursor": cursor}))
}

// TestSourceReviewPosting proves category drafts do not post and both directions remain exact.
func TestSourceReviewPosting(t *testing.T) {
	c, db, sid := reviewSetup(t)
	before := value(t, c, "SELECT revision::text FROM ledger.book")
	sources := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.list_sources($1,$2,$3)::text", reader, entity, book))
	if len(sources["items"].([]any)) != 1 {
		t.Fatal("source selector")
	}
	page, err := reviewList(t, db, sid, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(importObject(t, page)["items"].([]any)) != 2 {
		t.Fatal("review rows")
	}
	for _, direction := range []string{"outflow", "inflow"} {
		id := reviewID(t, c, direction)
		category := reviewExpense
		if direction == "inflow" {
			category = reviewRevenue
		}
		categories := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.review_categories($1,$2,$3,$4)::text", reader, entity, book, id))
		if categories["fx_required"] != false || categories["functional_currency"] != "USD" {
			t.Fatal("category policy")
		}
		for _, a := range categories["accounts"].([]any) {
			if a.(map[string]any)["account_id"] == account {
				t.Fatal("source offered as own offset")
			}
		}
		detail := reviewDetail(t, db, id)
		input := raw(reviewInput(detail, category))
		draft := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,$5,$6)::text", reader, entity, book, id, direction, input))
		again := call(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,$5,$6)::text", reader, entity, book, id, direction, input)
		if importObject(t, again)["draft_id"] != draft["draft_id"] {
			t.Fatal("draft replay")
		}
		if direction == "outflow" {
			importNoFinance(t, c, before)
		}
		command := reviewCommand(t, db, direction, draft)
		pending := importObject(t, call(t, db, "synthetic_runtime", "SELECT ledger.pending_command($1)::text", reader))
		if raw(pending["command"].(map[string]any)["body"]) != raw(command["body"]) {
			t.Fatal("remount lost exact reviewed lines")
		}
		result, err := importTransition(t, db, command, "execute")
		if err != nil {
			t.Fatal(err)
		}
		replay, err := importTransition(t, db, command, "execute")
		if err != nil || result != replay {
			t.Fatal("source post replay")
		}
		if _, err = importTransition(t, db, command, "acknowledge"); err != nil {
			t.Fatal(err)
		}
		after := reviewDetail(t, db, id)
		if after["state"] != "posted" || after["journal_id"] == nil {
			t.Fatal("missing atomic journal link")
		}
		want := "credit"
		if direction == "inflow" {
			want = "debit"
		}
		got := value(t, c, `SELECT side||':'||trim_scale(transaction_amount)::text||':'||trim_scale(functional_amount)::text FROM ledger.journal_line WHERE journal_id=$1 AND account_id=$2`, after["journal_id"], account)
		if got != want+":12.5:12.5" {
			t.Fatal("source leg changed", got)
		}
	}
	for _, table := range []string{"ledger.journal_entry", "banking.review_posting", "integration.outbox_message", "integration.operation_receipt"} {
		if value(t, c, "SELECT count(*)::text FROM "+table) != "2" {
			t.Fatalf("wrong %s effect count", table)
		}
	}
	if value(t, c, "SELECT count(*)::text FROM banking.observation_revision") != "2" {
		t.Fatal("source evidence changed")
	}
}

// TestSourceReviewSnapshots binds cursor filters/scope and invalidates drafts on source or book changes.
func TestSourceReviewSnapshots(t *testing.T) {
	c, db, sid := reviewSetup(t)
	page, err := reviewList(t, db, sid, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	first := importObject(t, page)
	cursor := first["next_cursor"]
	if cursor == nil {
		t.Fatal("missing bounded cursor")
	}
	next, err := reviewList(t, db, sid, 1, cursor)
	if err != nil {
		t.Fatal(err)
	}
	second := importObject(t, next)
	if second["next_cursor"] != nil || raw(first["items"]) == raw(second["items"]) {
		t.Fatal("page repetition")
	}
	out := reviewID(t, c, "outflow")
	in := reviewID(t, c, "inflow")
	draft := reviewDraft(t, db, out, "out", reviewExpense)
	_, err = reviewList(t, db, sid, 1, cursor)
	requireCode(t, err, "P2503")
	// Another draft changes the source-scoped review snapshot and invalidates old pins.
	reviewDraft(t, db, in, "in", reviewRevenue)
	cmd := reviewCommand(t, db, "stale", draft)
	_, err = importTransition(t, db, cmd, "execute")
	requireCode(t, err, "P2503")
	if _, err = importTransition(t, db, cmd, "cancel"); err != nil {
		t.Fatal(err)
	}
	if _, err = importTransition(t, db, cmd, "acknowledge"); err != nil {
		t.Fatal(err)
	}
	if reviewDetail(t, db, out)["state"] != "draft_ready" {
		t.Fatal("cancel destroyed draft")
	}
	newer := reviewDraft(t, db, out, "out-refresh", reviewExpense)
	cmd2 := reviewCommand(t, db, "fresh", newer)
	if _, err = importTransition(t, db, cmd2, "execute"); err != nil {
		t.Fatal(err)
	}
	if value(t, c, "SELECT count(*)::text FROM banking.review_draft WHERE review_id=$1", out) != "2" {
		t.Fatal("old draft overwritten")
	}
}

// TestSourceReviewAuthorization enforces current authority even on immutable receipt replay.
func TestSourceReviewAuthorization(t *testing.T) {
	c, db, _ := reviewSetup(t)
	id := reviewID(t, c, "outflow")
	draft := reviewDraft(t, db, id, "draft", reviewExpense)
	command := reviewCommand(t, db, "post", draft)
	for _, mutation := range []string{"UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id='" + reader + "' AND permission='prepare'", "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id='" + reader + "' AND permission='post'", "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE principal_id='" + reader + "' AND capability='review'"} {
		exec(t, c, mutation)
		_, err := importTransition(t, db, command, "execute")
		requireCode(t, err, "P2501")
		_, err = invoke(t, db, "synthetic_runtime", "SELECT ledger.pending_command($1)::text", reader)
		requireCode(t, err, "P2501")
		exec(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id=$1", reader)
		exec(t, c, "UPDATE banking.source_permission SET revoked_at=NULL WHERE principal_id=$1", reader)
	}
	if _, err := importTransition(t, db, command, "execute"); err != nil {
		t.Fatal(err)
	}
	exec(t, c, "UPDATE banking.source_permission SET revoked_at=clock_timestamp() WHERE capability='review'")
	for _, action := range []string{"execute", "cancel", "acknowledge"} {
		_, err := importTransition(t, db, command, action)
		requireCode(t, err, "P2501")
	}
	for _, principal := range []string{other, "00000000-0000-4000-8000-000000000099"} {
		_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", principal, entity, book, id)
		requireCode(t, err, "P2401")
	}
	runtime := connect(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	for _, q := range []string{"DELETE FROM banking.review_posting", "UPDATE banking.review_item SET state='posted'", "SELECT banking.execute_source_post(NULL,NULL,NULL,NULL,NULL,'{}')", "SELECT banking.set_source_review_permission(NULL,NULL,NULL,NULL,'x','{}')"} {
		_, err := runtime.Exec(context.Background(), q)
		requireCode(t, err, "42501")
	}
}

// TestSourceReviewAtomicity injects failures after ledger.post_journal has begun writing.
func TestSourceReviewAtomicity(t *testing.T) {
	for _, tc := range []struct{ table, event string }{{"banking.review_posting", "INSERT"}, {"banking.review_item", "UPDATE"}, {"banking.source_account", "UPDATE"}, {"audit.command_action", "INSERT"}} {
		t.Run(strings.ReplaceAll(tc.table, ".", "_"), func(t *testing.T) {
			c, db, _ := reviewSetup(t)
			id := reviewID(t, c, "outflow")
			draft := reviewDraft(t, db, id, "draft", reviewExpense)
			command := reviewCommand(t, db, "command", draft)
			rev := value(t, c, "SELECT revision::text FROM ledger.book")
			sourceRev := value(t, c, "SELECT source_revision::text FROM banking.source_account")
			exec(t, c, "CREATE FUNCTION public.fail_review_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION USING ERRCODE='P9999',MESSAGE='synthetic_fault'; END $$")
			exec(t, c, "CREATE TRIGGER inject_review_failure BEFORE "+tc.event+" ON "+tc.table+" FOR EACH ROW EXECUTE FUNCTION public.fail_review_test()")
			_, err := importTransition(t, db, command, "execute")
			requireCode(t, err, "P9999")
			importNoFinance(t, c, rev)
			if value(t, c, "SELECT count(*)::text FROM banking.review_posting") != "0" || value(t, c, "SELECT source_revision::text FROM banking.source_account") != sourceRev || value(t, c, "SELECT state FROM banking.review_item WHERE review_id=$1", id) != "draft_ready" {
				t.Fatal("partial source posting")
			}
			exec(t, c, "DROP TRIGGER inject_review_failure ON "+tc.table)
			if _, err = importTransition(t, db, command, "execute"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestSourceReviewCommandIntegrity rejects altered approved lines and duplicate source posting.
func TestSourceReviewCommandIntegrity(t *testing.T) {
	c, db, _ := reviewSetup(t)
	id := reviewID(t, c, "outflow")
	draft := reviewDraft(t, db, id, "draft", reviewExpense)
	tampered := reviewCommandBody(draft)
	var copyBody map[string]any
	if err := json.Unmarshal([]byte(raw(tampered["posting_body"])), &copyBody); err != nil {
		t.Fatal(err)
	}
	copyBody["description"] = "Unreviewed change"
	tampered["posting_body"] = copyBody
	_, err := invoke(t, db, "synthetic_runtime", "SELECT ledger.register_command($1,'tampered',$2,$3)::text", reader, importRequest, raw(map[string]any{"operation": "source_post", "entity_id": entity, "book_id": book, "body": tampered}))
	requireCode(t, err, "P2502")
	cmd := reviewCommand(t, db, "race", draft)
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 2)
	results := make(chan string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			s, e := importTransition(t, db, cmd, "execute")
			errs <- e
			results <- s
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if a, b := <-results, <-results; a != b {
		t.Fatal("concurrent receipt changed")
	}
	if _, err = importTransition(t, db, cmd, "acknowledge"); err != nil {
		t.Fatal(err)
	}
	again := reviewCommand(t, db, "new-key-same-draft", draft)
	if _, err = importTransition(t, db, again, "execute"); err != nil {
		t.Fatal(err)
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" || value(t, c, "SELECT count(*)::text FROM banking.review_posting") != "1" {
		t.Fatal("second source journal")
	}
	_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'after-post',$5)::text", reader, entity, book, id, raw(reviewInput(reviewDetail(t, db, id), reviewExpense)))
	requireCode(t, err, "P2502")
}

// TestSourceReviewFXGate keeps original units visible and refuses guessed conversion.
func TestSourceReviewFXGate(t *testing.T) {
	for _, tc := range []struct{ code, currency, amount, canonical string }{{"JPY", "00000000-0000-4000-8000-000000000006", "0012.000", "12"}, {"KWD", "00000000-0000-4000-8000-000000000007", "0012.5010", "12.501"}} {
		t.Run(tc.code, func(t *testing.T) {
			c, db, _ := reviewSetup(t)
			v := manifest()
			v["ledger_account_id"] = "00000000-0000-4000-8000-000000000015"
			v["currency_version_id"] = tc.currency
			v["namespace"].(map[string]any)["synthetic_dataset_id"] = "00000000-0000-4000-8000-000000000073"
			receipt := importObject(t, call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'foreign',$4)::text", actor, entity, book, raw(v)))
			sid := receipt["source_account_id"].(string)
			nid := receipt["namespace_id"].(string)
			_, err := mutate(t, db, sid, "set_source_import_permission", "foreign-import", map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": true, "entity_import_enabled": true})
			if err != nil {
				t.Fatal(err)
			}
			batch := importRegister(t, db, sid, nid, "foreign")
			row := importRow("foreign", tc.amount, 2)
			row["canonical_amount"] = tc.canonical
			row["currency"] = tc.code
			row["currency_version_id"] = tc.currency
			importFinish(t, db, "foreign-import", importPublish(t, db, batch, importPublication(row)))
			_, err = mutate(t, db, sid, "set_source_review_permission", "foreign-review", map[string]any{"expected_binding_revision": "2", "principal_id": reader, "enabled": true})
			if err != nil {
				t.Fatal(err)
			}
			id := value(t, c, "SELECT review_id::text FROM banking.review_item WHERE source_account_id=$1", sid)
			detail := reviewDetail(t, db, id)
			if detail["currency"] != tc.code || detail["amount"] != tc.canonical {
				t.Fatal("foreign units lost")
			}
			categories := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.review_categories($1,$2,$3,$4)::text", reader, entity, book, id))
			if categories["fx_required"] != true {
				t.Fatal("FX gate absent")
			}
			_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'foreign-draft',$5)::text", reader, entity, book, id, raw(reviewInput(detail, reviewExpense)))
			requireCode(t, err, "P2504")
			if value(t, c, "SELECT count(*)::text FROM banking.review_draft") != "0" || value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
				t.Fatal("foreign flow bypassed gate")
			}
		})
	}
}

// TestSourceReviewBoundaries preserves overlap holds and existing ledger period authority.
func TestSourceReviewBoundaries(t *testing.T) {
	t.Run("closed_period", func(t *testing.T) {
		c, db, _ := reviewSetup(t)
		exec(t, c, "UPDATE ledger.period SET state='closed',closed_at=now(),closed_by=$3,close_revision=0 WHERE entity_id=$1 AND book_id=$2", entity, book, actor)
		id := reviewID(t, c, "outflow")
		draft := reviewDraft(t, db, id, "draft", reviewExpense)
		cmd := reviewCommand(t, db, "post", draft)
		rev := value(t, c, "SELECT revision::text FROM ledger.book")
		_, err := importTransition(t, db, cmd, "execute")
		requireCode(t, err, "P2105")
		importNoFinance(t, c, rev)
		if value(t, c, "SELECT state FROM ledger.interactive_command WHERE command_id=$1", cmd["command_id"]) != "prepared" {
			t.Fatal("failed posting completed command")
		}
	})
	t.Run("overlap", func(t *testing.T) {
		c, db, sid := reviewSetup(t)
		receipt, err := mutate(t, db, sid, "register_source_namespace", "overlap", map[string]any{"expected_binding_revision": "3", "namespace": map[string]any{"profile": importProfile, "synthetic_dataset_id": "00000000-0000-4000-8000-000000000075", "source_account_ref": "SYNTH-CASH-01"}})
		if err != nil {
			t.Fatal(err)
		}
		nid := importObject(t, receipt)["namespace_id"].(string)
		m := importManifest(sid, nid, "overlap")
		m["binding_revision"] = "4"
		batch := importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.register_import_batch($1,$2,$3,'overlap',$4)::text", reader, entity, book, raw(m)))
		importFinish(t, db, "overlap-import", importPublish(t, db, batch, importPublication(importRow("same-purchase", "12.50", 2))))
		id := value(t, c, "SELECT review_id::text FROM banking.review_item WHERE state='overlap_hold'")
		_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'overlap-draft',$5)::text", reader, entity, book, id, raw(reviewInput(reviewDetail(t, db, id), reviewExpense)))
		requireCode(t, err, "P2502")
	})
	t.Run("shape_scope", func(t *testing.T) {
		c, db, sid := reviewSetup(t)
		id := reviewID(t, c, "outflow")
		detail := reviewDetail(t, db, id)
		for _, category := range []string{account, "00000000-0000-4000-8000-000000000099"} {
			_, err := invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'invalid',$5)::text", reader, entity, book, id, raw(reviewInput(detail, category)))
			requireCode(t, err, "P2500")
		}
		for _, limit := range []int{0, 101} {
			_, err := reviewList(t, db, sid, limit, nil)
			requireCode(t, err, "P2500")
		}
		_, err := reviewList(t, db, sid, 1, "not-a-cursor")
		requireCode(t, err, "P2500")
		_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.get_review_item($1,$2,$3,$4)::text", reader, entity, "00000000-0000-4000-8000-000000000099", id)
		requireCode(t, err, "P2401")
		changed := reviewInput(detail, reviewExpense)
		changed["accounting_date"] = "2026-01-06"
		_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'changed-date',$5)::text", reader, entity, book, id, raw(changed))
		requireCode(t, err, "P2500")
		for _, rawBody := range []string{`{"source_revision":"1","source_revision":"2"}`, `{"x":1e1000000}`} {
			_, err = invoke(t, db, "synthetic_runtime", "SELECT banking.create_review_draft($1,$2,$3,$4,'malformed',$5)::text", reader, entity, book, id, rawBody)
			requireCode(t, err, "P2500")
		}
	})
}
