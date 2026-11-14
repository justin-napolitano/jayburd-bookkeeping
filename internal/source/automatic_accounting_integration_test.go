//go:build integration

package source

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/autopost"
	"strings"
	"testing"
)

const automaticRuleOut = "99000000-0000-4000-8000-000000000001"
const automaticRuleIn = "99000000-0000-4000-8000-000000000002"

// automaticRule is independently authored reviewed mapping input, never generated from expected postings.
func automaticRule(id, sid, direction, offset, treatment string) map[string]any {
	return map[string]any{"rule_id": id, "name": "Fictional " + direction + " " + id, "source_account_id": sid, "direction": direction, "description": "Fictional supplies", "offset_account_id": offset, "treatment": treatment}
}
func automaticPolicy(t testing.TB, db, key, expected, sid string, rules ...map[string]any) map[string]any {
	t.Helper()
	if rules == nil {
		rules = []map[string]any{}
	}
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.set_automatic_accounting($1,$2,$3,$4)::text", reader, entity, book, raw(map[string]any{"key": key, "expected_revision": expected, "enabled": true, "book_policy_id": reviewPolicy, "effective_from": "2026-01-01", "effective_to": "2026-02-01", "rules": rules})))
}
func automaticPrepare(t testing.TB, db string) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.prepare_automatic_accounting($1,$2,$3)::text", reader, entity, book))
}
func automaticExecute(t testing.TB, db, id string) (string, error) {
	t.Helper()
	return invoke(t, db, "synthetic_runtime", "SELECT banking.execute_automatic_accounting($1,$2,$3,$4)::text", reader, entity, book, id)
}
func automaticRound(t testing.TB, db string) {
	t.Helper()
	v := automaticPrepare(t, db)
	if v["state"] == "prepared" {
		if _, e := automaticExecute(t, db, v["operation"].(map[string]any)["operation_id"].(string)); e != nil {
			t.Fatal(e)
		}
	}
}
func automaticStatus(t testing.TB, db string) map[string]any {
	t.Helper()
	return importObject(t, call(t, db, "synthetic_runtime", "SELECT banking.automatic_accounting_status($1,$2,$3)::text", reader, entity, book))
}

// TestAutomaticAccountingPostsOnce checks real source links and fixed 12.50 debit/credit expectations.
func TestAutomaticAccountingPostsOnce(t *testing.T) {
	c, db, sid := reviewSetup(t)
	commands := value(t, c, "SELECT count(*)::text FROM ledger.interactive_command")
	automaticPolicy(t, db, "policy1", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"), automaticRule(automaticRuleIn, sid, "inflow", reviewRevenue, "income"))
	for i := 0; i < 2; i++ {
		p := automaticPrepare(t, db)
		if p["state"] != "prepared" {
			t.Fatal("eligible source was not prepared", p)
		}
		id := p["operation"].(map[string]any)["operation_id"].(string)
		again := automaticPrepare(t, db)
		if again["operation"].(map[string]any)["operation_id"] != id {
			t.Fatal("prepare reply recovery changed identity")
		}
		first, e := automaticExecute(t, db, id)
		if e != nil {
			t.Fatal(e)
		}
		second, e := automaticExecute(t, db, id)
		if e != nil || first != second {
			t.Fatal("same operation replay changed receipt")
		}
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.interactive_command") != commands {
		t.Fatal("automation fabricated interactive commands")
	}
	for _, table := range []string{"ledger.journal_entry", "banking.review_posting", "integration.outbox_message", "integration.operation_receipt"} {
		if value(t, c, "SELECT count(*)::text FROM "+table) != "2" {
			t.Fatal("wrong financial effect count", table)
		}
	}
	if value(t, c, "SELECT string_agg(side||':'||trim_scale(transaction_amount)::text||':'||trim_scale(functional_amount)::text,',' ORDER BY side)FROM ledger.journal_line WHERE account_id=$1", account) != "credit:12.5:12.5,debit:12.5:12.5" {
		t.Fatal("source debit/credit oracle mismatch")
	}
	if value(t, c, "SELECT side||':'||trim_scale(transaction_amount)::text FROM ledger.journal_line WHERE account_id=$1", reviewExpense) != "debit:12.5" {
		t.Fatal("expense oracle mismatch")
	}
	if value(t, c, "SELECT side||':'||trim_scale(transaction_amount)::text FROM ledger.journal_line WHERE account_id=$1", reviewRevenue) != "credit:12.5" {
		t.Fatal("income oracle mismatch")
	}
	if value(t, c, "SELECT count(*)::text FROM banking.review_posting WHERE command_id IS NULL AND automatic_operation_id IS NOT NULL") != "2" {
		t.Fatal("missing automatic source provenance")
	}
	if value(t, c, "SELECT count(*)::text FROM banking.review_decision WHERE reason='automatic_policy'") != "2" {
		t.Fatal("false human draft attribution")
	}
	changed := automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense")
	changed["rule_id"] = "99000000-0000-4000-8000-000000000003"
	automaticPolicy(t, db, "policy2", "1", sid, changed)
	for i := 0; i < 3; i++ {
		automaticRound(t, db)
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "2" {
		t.Fatal("policy/rule identity change reposted evidence")
	}
}

// TestAutomaticAccountingHolds proves manual choices and ambiguous/missing mappings remain unposted.
func TestAutomaticAccountingHolds(t *testing.T) {
	t.Run("manual", func(t *testing.T) {
		c, db, sid := reviewSetup(t)
		rid := reviewID(t, c, "outflow")
		draft := reviewDraft(t, db, rid, "human-draft", reviewExpense)
		automaticPolicy(t, db, "policy", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"))
		for i := 0; i < 4; i++ {
			automaticRound(t, db)
		}
		if reviewDetail(t, db, rid)["draft_id"] != draft["draft_id"] {
			t.Fatal("automatic work replaced human draft")
		}
		if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
			t.Fatal("held work posted")
		}
		if value(t, c, "SELECT count(*)::text FROM banking.automatic_accounting_operation WHERE reason='manual_draft'") != "1" {
			t.Fatal("manual protection hold missing")
		}
	})
	t.Run("ambiguous", func(t *testing.T) {
		c, db, sid := reviewSetup(t)
		automaticPolicy(t, db, "policy", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"), automaticRule(automaticRuleIn, sid, "outflow", reviewExpense, "expense"))
		for i := 0; i < 4; i++ {
			automaticRound(t, db)
		}
		if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
			t.Fatal("ambiguous/unmapped work posted")
		}
		status := automaticStatus(t, db)
		if status["attention_count"] != "2" || status["pending_count"] != "0" {
			t.Fatal("whole-result holds/counts not exact", status)
		}
	})
}

// TestAutomaticAccountingFences checks execute-time permissions and policy editions after durable prepare.
func TestAutomaticAccountingFences(t *testing.T) {
	c, db, sid := reviewSetup(t)
	automaticPolicy(t, db, "policy1", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"), automaticRule(automaticRuleIn, sid, "inflow", reviewRevenue, "income"))
	p := automaticPrepare(t, db)
	id := p["operation"].(map[string]any)["operation_id"].(string)
	exec(t, c, "UPDATE ledger.membership SET revoked_at=clock_timestamp()WHERE principal_id=$1 AND permission='post'", reader)
	_, e := automaticExecute(t, db, id)
	requireCode(t, e, "P2A01")
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("revoked operation posted")
	}
	exec(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id=$1 AND permission='post'", reader)
	automaticPolicy(t, db, "policy2", "1", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"))
	v, e := automaticExecute(t, db, id)
	if e != nil || importObject(t, v)["operation"].(map[string]any)["state"] != "stale" {
		t.Fatal("changed policy did not stale prepared work", e)
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("stale policy posted")
	}
}

// TestAutomaticAccountingOriginAndRollback prevents human-command reservation of automatic drafts and orphan finance.
func TestAutomaticAccountingOriginAndRollback(t *testing.T) {
	c, db, sid := reviewSetup(t)
	automaticPolicy(t, db, "policy", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"), automaticRule(automaticRuleIn, sid, "inflow", reviewRevenue, "income"))
	p := automaticPrepare(t, db)
	op := p["operation"].(map[string]any)
	id := op["operation_id"].(string)
	rid := op["review_id"].(string)
	detail := reviewDetail(t, db, rid)
	d := detail["draft"].(map[string]any)
	if d["origin"] != "automatic" || d["automatic_operation_id"] != id {
		t.Fatal("automatic draft provenance hidden")
	}
	_, e := invoke(t, db, "synthetic_runtime", "SELECT ledger.register_command($1,$2,$3,$4)::text", reader, "cannot-human-post", importRequest, raw(map[string]any{"operation": "source_post", "entity_id": entity, "book_id": book, "body": reviewCommandBody(d)}))
	requireCode(t, e, "P2502")
	if value(t, c, "SELECT count(*)::text FROM ledger.interactive_command_reservation WHERE principal_id=$1", reader) != "0" {
		t.Fatal("automatic draft left stuck human reservation")
	}
	exec(t, c, "CREATE FUNCTION public.fail_auto_link()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION USING ERRCODE='XX000',MESSAGE='fictional_fault';END$$;CREATE TRIGGER fail_auto AFTER INSERT ON banking.review_posting FOR EACH ROW EXECUTE FUNCTION public.fail_auto_link()")
	_, e = automaticExecute(t, db, id)
	requireCode(t, e, "XX000")
	for _, table := range []string{"ledger.journal_entry", "banking.review_posting", "integration.outbox_message", "integration.operation_receipt"} {
		if value(t, c, "SELECT count(*)::text FROM "+table) != "0" {
			t.Fatal("financial effect escaped rollback", table)
		}
	}
	exec(t, c, "DROP TRIGGER fail_auto ON banking.review_posting;DROP FUNCTION public.fail_auto_link()")
	if _, e = automaticExecute(t, db, id); e != nil {
		t.Fatal(e)
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
		t.Fatal("same-operation recovery failed")
	}
}

// TestAutomaticAccountingRunner proves the real restricted Go round actually invokes the SQL engine.
func TestAutomaticAccountingRunner(t *testing.T) {
	c, db, sid := reviewSetup(t)
	automaticPolicy(t, db, "policy", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"), automaticRule(automaticRuleIn, sid, "inflow", reviewRevenue, "income"))
	ctx := context.Background()
	pool, e := pgxpool.New(ctx, "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	service := &autopost.Service{DB: pool}
	for i := 0; i < 3; i++ {
		if e = service.Round(ctx); e != nil {
			t.Fatal(e)
		}
	}
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "2" {
		t.Fatal("background round did not post both eligible sources")
	}
}

// TestAutomaticAccountingTreatmentHolds never substitutes a generic expense for missing pair/split policy.
func TestAutomaticAccountingTreatmentHolds(t *testing.T) {
	for _, tc := range []struct{ kind, reason string }{{"transfer", "paired_transfer_required"}, {"card_payment", "paired_transfer_required"}, {"loan", "split_treatment_required"}} {
		t.Run(tc.kind, func(t *testing.T) {
			c, db, sid := reviewSetup(t)
			automaticPolicy(t, db, "policy", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, tc.kind))
			for i := 0; i < 4; i++ {
				automaticRound(t, db)
			}
			if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, c, "SELECT count(*)::text FROM banking.automatic_accounting_operation WHERE reason=$1", tc.reason) != "1" {
				t.Fatal("unsupported treatment did not hold")
			}
		})
	}
	for _, kind := range []string{"closed", "inactive", "paused"} {
		t.Run(kind, func(t *testing.T) {
			c, db, sid := reviewSetup(t)
			automaticPolicy(t, db, "policy", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"))
			reason := ""
			switch kind {
			case "closed":
				reason = "closed_period"
				exec(t, c, "UPDATE ledger.period SET state='closed',closed_at=clock_timestamp(),closed_by=$1,close_revision=(SELECT revision FROM ledger.book)", actor)
			case "inactive":
				reason = "inactive_account"
				exec(t, c, "UPDATE ledger.account SET active_to='2026-01-03' WHERE account_id=$1", reviewExpense)
			case "paused":
				reason = "source_unavailable"
				_, e := mutate(t, db, sid, "set_source_state", "pause", map[string]any{"expected_binding_revision": value(t, c, "SELECT binding_revision::text FROM banking.source_account WHERE source_account_id=$1", sid), "state": "paused"})
				if e != nil {
					t.Fatal(e)
				}
			}
			for i := 0; i < 4; i++ {
				automaticRound(t, db)
			}
			if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, c, "SELECT count(*)::text FROM banking.automatic_accounting_operation WHERE reason=$1", reason) == "0" {
				t.Fatal("current-state hold missing", kind)
			}
		})
	}
}

// TestAutomaticAccountingForeignUnits retains independently fixed zero/two/three-decimal source quantities.
func TestAutomaticAccountingForeignUnits(t *testing.T) {
	for _, tc := range []struct{ code, cv, amount, canonical string }{{"EUR", "00000000-0000-4000-8000-000000000092", "10.00", "10"}, {"JPY", "00000000-0000-4000-8000-000000000006", "100", "100"}, {"KWD", "00000000-0000-4000-8000-000000000007", "0.125", "0.125"}} {
		t.Run(tc.code, func(t *testing.T) {
			c, db := fixture(t)
			if tc.code == "EUR" {
				exec(t, c, "INSERT INTO reference.currency VALUES($1,'EUR','978',2,'2025-01-01',NULL,'synthetic-current')", tc.cv)
			}
			m := manifest()
			m["currency_version_id"] = tc.cv
			registered := importObject(t, call(t, db, "synthetic_operator", "SELECT banking.register_source($1,$2,$3,'foreign-source',$4)::text", actor, entity, book, raw(m)))
			sid := registered["source_account_id"].(string)
			nid := registered["namespace_id"].(string)
			_, e := mutate(t, db, sid, "set_source_import_permission", "foreign-import", map[string]any{"expected_binding_revision": "1", "principal_id": reader, "enabled": true, "entity_import_enabled": true})
			if e != nil {
				t.Fatal(e)
			}
			exec(t, c, "INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL),($1,$2,'post',NULL)", entity, reader)
			exec(t, c, "INSERT INTO ledger.book_policy VALUES($1,$2,$3,'2026-01-01',NULL,$4,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',$5,now())", entity, book, reviewPolicy, currency, actor)
			exec(t, c, "INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before)VALUES($1,$2,'00000000-0000-4000-8000-000000000061','2026-01-01','2026-02-01')", entity, book)
			exec(t, c, "INSERT INTO ledger.account VALUES($1,$2,$3,'6000','Fictional supplies','expense','debit',false,'2026-01-01',NULL)", entity, book, reviewExpense)
			row := importRow("foreign", tc.amount, 2)
			row["currency"] = tc.code
			row["currency_version_id"] = tc.cv
			row["canonical_amount"] = tc.canonical
			batch := importRegister(t, db, sid, nid, "foreign")
			importFinish(t, db, "foreign-commit", importPublish(t, db, batch, importPublication(row)))
			_, e = mutate(t, db, sid, "set_source_review_permission", "foreign-review", map[string]any{"expected_binding_revision": "2", "principal_id": reader, "enabled": true})
			if e != nil {
				t.Fatal(e)
			}
			before := value(t, c, "SELECT jsonb_agg(to_jsonb(q))::text FROM banking.observation_revision q")
			automaticPolicy(t, db, "policy", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"))
			for i := 0; i < 3; i++ {
				automaticRound(t, db)
			}
			if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || value(t, c, "SELECT count(*)::text FROM integration.outbox_message") != "0" {
				t.Fatal("foreign units fabricated functional posting")
			}
			if value(t, c, "SELECT amount_lexeme||':'||trim_scale(amount)::text||':'||currency_version_id::text FROM banking.observation_revision") != tc.amount+":"+tc.canonical+":"+tc.cv {
				t.Fatal("original unit/precision oracle mismatch")
			}
			if value(t, c, "SELECT jsonb_agg(to_jsonb(q))::text FROM banking.observation_revision q") != before || value(t, c, "SELECT reason FROM banking.automatic_accounting_operation") != "fx_required" {
				t.Fatal("foreign source evidence changed or hold missing")
			}
		})
	}
}

// TestAutomaticAccountingSourcePrivacy rechecks the operation's own source after policy rotation/revocation.
func TestAutomaticAccountingSourcePrivacy(t *testing.T) {
	c, db, sid := reviewSetup(t)
	automaticPolicy(t, db, "policy1", "0", sid, automaticRule(automaticRuleOut, sid, "outflow", reviewExpense, "expense"), automaticRule(automaticRuleIn, sid, "inflow", reviewRevenue, "income"))
	p := automaticPrepare(t, db)
	op := p["operation"].(map[string]any)
	id := op["operation_id"].(string)
	rid := op["review_id"].(string)
	automaticPolicy(t, db, "empty-policy", "1", sid)
	_, e := mutate(t, db, sid, "set_source_permission", "revoke-read", map[string]any{"expected_binding_revision": value(t, c, "SELECT binding_revision::text FROM banking.source_account WHERE source_account_id=$1", sid), "principal_id": reader, "enabled": false})
	if e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"SELECT banking.automatic_accounting_status($1,$2,$3)::text", "SELECT banking.prepare_automatic_accounting($1,$2,$3)::text"} {
		v := call(t, db, "synthetic_runtime", q, reader, entity, book)
		if strings.Contains(v, id) || strings.Contains(v, rid) || strings.Contains(v, sid) {
			t.Fatal("revoked historical source leaked through new policy", q)
		}
	}
	status := automaticStatus(t, db)
	if status["pending_count"] != "0" || status["attention_count"] != "0" || status["last_operation"] != nil {
		t.Fatal("revoked operation count/metadata leaked")
	}
	_, e = automaticExecute(t, db, id)
	requireCode(t, e, "P2A01")
	exec(t, c, "INSERT INTO ledger.membership VALUES($1,$2,'prepare',NULL),($1,$2,'post',NULL)", entity, other)
	_, e = invoke(t, db, "synthetic_runtime", "SELECT banking.execute_automatic_accounting($1,$2,$3,$4)::text", other, entity, book, id)
	requireCode(t, e, "P2A01")
	if value(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" {
		t.Fatal("revoked/foreign actor posted")
	}
}

// TestAutomaticAccountingPrivileges invokes actual denied operations, not just catalog privilege predicates.
func TestAutomaticAccountingPrivileges(t *testing.T) {
	_, db, _ := reviewSetup(t)
	for _, q := range []string{"SELECT count(*)::text FROM banking.automatic_accounting_policy", "SELECT banking.automatic_accounting_policy_wire(NULL,NULL)::text", "SELECT banking.create_source_draft(NULL,NULL,NULL,NULL,NULL,NULL,NULL)::text", "SELECT banking.commit_source_post(NULL,NULL,NULL,NULL,NULL,NULL,NULL)::text", "SELECT banking.source_post_authorize(NULL,NULL,NULL,NULL)::text", "SELECT banking.execute_source_post(NULL,NULL,NULL,NULL,NULL,NULL)::text"} {
		_, e := invoke(t, db, "synthetic_runtime", q)
		requireCode(t, e, "42501")
	}
	for _, login := range []string{"synthetic_observer", "synthetic_publisher"} {
		_, e := invoke(t, db, login, "SELECT banking.get_automatic_accounting($1,$2,$3)::text", reader, entity, book)
		requireCode(t, e, "42501")
	}
}
