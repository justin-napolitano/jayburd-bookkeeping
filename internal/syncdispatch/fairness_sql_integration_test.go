//go:build integration

package syncdispatch

import (
	"fmt"
	"testing"
)

// TestDeliveryCandidateFairness keeps 30 earlier synthetic operations ineligible while a later ready stream remains discoverable.
func TestDeliveryCandidateFairness(t *testing.T) {
	owner, db, binding := dispatchSetup(t)
	stream := value(t, owner, "SELECT stream_id::text FROM integration.sync_stream")
	// Owner-only fixture copies retain production constraints and triggers, without bypassing authorization at the tested boundary.
	for n := 1; n <= 30; n++ {
		src := fmt.Sprintf("10000000-0000-4000-8000-%012d", n)
		ns := fmt.Sprintf("20000000-0000-4000-8000-%012d", n)
		st := fmt.Sprintf("30000000-0000-4000-8000-%012d", n)
		op := fmt.Sprintf("40000000-0000-4000-8000-%012d", n)
		acct := fmt.Sprintf("50000000-0000-4000-8000-%012d", n)
		exec(t, owner, `INSERT INTO ledger.account SELECT (jsonb_populate_record(NULL::ledger.account,to_jsonb(a)||jsonb_build_object('account_id',$1::text,'code',$2::text))).* FROM ledger.account a WHERE account_id=$3`, acct, fmt.Sprintf("F%03d", n), account)
		exec(t, owner, `INSERT INTO banking.source_account SELECT (jsonb_populate_record(NULL::banking.source_account,to_jsonb(s)||jsonb_build_object('source_account_id',$1::text,'ledger_account_id',$2::text))).* FROM banking.source_account s WHERE source_account_id=$3`, src, acct, sourceID)
		exec(t, owner, `INSERT INTO banking.source_namespace SELECT (jsonb_populate_record(NULL::banking.source_namespace,to_jsonb(s)||jsonb_build_object('source_account_id',$1::text,'namespace_id',$2::text,'synthetic_dataset_id',$2::text,'source_account_ref',$2::text))).* FROM banking.source_namespace s WHERE namespace_id=$3`, src, ns, csvNS)
		exec(t, owner, `INSERT INTO banking.source_permission SELECT entity_id,book_id,$1,principal_id,capability,revoked_at FROM banking.source_permission WHERE source_account_id=$2`, src, sourceID)
		exec(t, owner, `INSERT INTO integration.sync_stream SELECT (jsonb_populate_record(NULL::integration.sync_stream,to_jsonb(s)||jsonb_build_object('stream_id',$1::text,'source_account_id',$2::text,'namespace_id',$3::text,'account_id',$1::text))).* FROM integration.sync_stream s WHERE stream_id=$4`, st, src, ns, stream)
		if n <= 15 {
			exec(t, owner, `INSERT INTO integration.sync_delivery_operation(entity_id,book_id,operation_id,binding_id,stream_id,target_generation,state,next_eligible_at) VALUES($1,$2,$3,$4,$5,1,'retry_wait',clock_timestamp()+interval '1 hour')`, entity, book, op, binding, st)
		} else {
			exec(t, owner, `INSERT INTO integration.sync_delivery_operation(entity_id,book_id,operation_id,binding_id,stream_id,target_generation,state,run_holder,run_lease_until,claim_binding_revision) VALUES($1,$2,$3,$4,$5,1,'running',$6,clock_timestamp()+interval '2 minutes',1)`, entity, book, op, binding, st, claimID)
		}
		exec(t, owner, `INSERT INTO integration.sync_delivery_dispatch(entity_id,book_id,operation_id,generation,task_name_hex) VALUES($1,$2,$3,1,encode(sha256(convert_to(($3::uuid)::text||'|1|2','UTF8')),'hex'))`, entity, book, op)
	}
	const due = "f0000000-0000-4000-8000-000000000099"
	exec(t, owner, `INSERT INTO integration.sync_delivery_operation(entity_id,book_id,operation_id,binding_id,stream_id,target_generation) VALUES($1,$2,$3,$4,$5,1)`, entity, book, due, binding, stream)
	exec(t, owner, `INSERT INTO integration.sync_delivery_dispatch(entity_id,book_id,operation_id,generation,task_name_hex) VALUES($1,$2,$3,1,encode(sha256(convert_to(($3::uuid)::text||'|1|2','UTF8')),'hex'))`, entity, book, due)
	got := object(t, call(t, db, "synthetic_publisher", "SELECT integration.sync_delivery_candidates($1,25)", binding))
	ids, ok := got["operation_ids"].([]any)
	if !ok || len(ids) != 1 || ids[0] != due {
		t.Fatalf("ready operation starved: %v", got)
	}
}
