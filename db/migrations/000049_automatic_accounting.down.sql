-- #334: downgrade refuses any retained policy/operation instead of erasing accounting evidence.
BEGIN
  ;

SET LOCAL ROLE bookkeeping_owner;

DO $$BEGIN
  IF EXISTS(SELECT
  FROM banking.automatic_accounting_edition)OR EXISTS(SELECT
  FROM banking.automatic_accounting_operation)THEN
  RAISE
  EXCEPTION  'automatic_accounting_history_retained';

END IF;

END $$;

CREATE OR REPLACE FUNCTION banking.create_review_draft(actor uuid,e uuid,b uuid,id uuid,k text,raw text)
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog AS $$
  DECLARE v jsonb;
  r banking.review_item;
  s banking.source_account;
  q banking.observation_revision;
  d banking.review_draft;
  p ledger.book_policy;
  a ledger.account;
  rev bigint;
  category_id uuid;
  day_value date;
  description_value text;
  code text;
  amount_value text;
  body jsonb;
  norm jsonb;

BEGIN

  v:=banking.import_json(raw,8192);
  PERFORM ledger.keys(v,ARRAY['expected_review_revision','source_revision','binding_revision','expected_book_revision','category_account_id','policy_id','accounting_date','description']);

  IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' THEN
  PERFORM banking.review_error('P2500');

END IF;

  PERFORM banking.import_revision(v->'expected_review_revision');
  PERFORM banking.import_revision(v->'source_revision');
  PERFORM banking.import_revision(v->'binding_revision');
  PERFORM banking.import_revision(v->'expected_book_revision');

  category_id:=ledger.uuid_value(v->'category_account_id');
  PERFORM ledger.uuid_value(v->'policy_id');
  day_value:=ledger.date_value(ledger.text_value(v->'accounting_date'));
  description_value:=ledger.text_value(v->'description');

  IF length(description_value) NOT BETWEEN 1 AND 500 THEN
  PERFORM banking.review_error('P2500');

END IF;

  s:=banking.review_authorize(actor,e,b,id,false);

  norm:=jsonb_build_object('review_id',id,'body',v);

  SELECT * INTO d
  FROM banking.review_draft
  WHERE entity_id=e AND book_id=b AND created_by=actor AND idempotency_key=k;

  IF FOUND THEN
  IF d.normalized_request<>norm THEN
  PERFORM banking.review_error('P2502');

END IF;
  RETURN banking.render_review_draft(d);

END IF;

  SELECT * INTO r
  FROM banking.review_item
  WHERE entity_id=e AND book_id=b AND review_id=id FOR UPDATE;

  SELECT revision INTO rev
  FROM ledger.book
  WHERE entity_id=e AND book_id=b;

  IF r.state NOT IN ('needs_review','draft_ready') OR s.state<>'active' THEN
  PERFORM banking.review_error('P2502');

END IF;

  IF ROW(r.review_revision,s.source_revision,s.binding_revision,rev) IS DISTINCT
  FROM ROW((v->>'expected_review_revision')::bigint,(v->>'source_revision')::bigint,(v->>'binding_revision')::bigint,(v->>'expected_book_revision')::bigint) THEN
  PERFORM banking.review_error('P2503');

END IF;

  IF s.source_revision=9223372036854775807 OR r.review_revision=9223372036854775807 THEN
  PERFORM banking.review_error('P2505');

END IF;

  q:=banking.review_fact(e,b,id);

  IF q.booked_date<>day_value OR q.source_state<>'booked' THEN
  PERFORM banking.review_error('P2500');

END IF;

BEGIN
  SELECT * INTO STRICT p
  FROM ledger.book_policy
  WHERE entity_id=e AND book_id=b AND effective_from<=day_value AND (effective_to IS NULL OR effective_to>day_value);

  EXCEPTION  WHEN no_data_found OR too_many_rows THEN
  PERFORM ledger.fail('P2109');

END;

  IF p.policy_id<>(v->>'policy_id')::uuid THEN
  PERFORM banking.review_error('P2503');

END IF;

  IF q.currency_version_id<>p.functional_currency_version_id THEN
  PERFORM banking.review_error('P2504');

END IF;

  SELECT alphabetic_code INTO code
  FROM reference.currency
  WHERE currency_version_id=q.currency_version_id;

-- Match the existing reviewed posting policy; do not claim a new accounting framework.
  IF p.basis<>'accrual' OR p.timezone<>'UTC' OR p.posting_scale<>2 OR p.rounding_mode<>'half_even' OR p.rate_policy_code<>'synthetic-reviewed-spot-v1' OR code<>'USD' THEN
  PERFORM ledger.fail('P2109');

END IF;

  PERFORM 1
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id IN (s.ledger_account_id,category_id)
  ORDER BY account_id FOR SHARE;

  IF category_id=s.ledger_account_id THEN
  PERFORM banking.review_error('P2500');

END IF;

  SELECT * INTO a
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id=category_id AND active_from<=day_value AND (active_to IS NULL OR active_to>day_value);

  IF NOT FOUND OR NOT EXISTS(SELECT
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id=s.ledger_account_id AND active_from<=day_value AND (active_to IS NULL OR active_to>day_value)) THEN
  PERFORM banking.review_error('P2500');

END IF;

  amount_value:=trim_scale(q.amount)::text;

  body:=jsonb_build_object('expected_revision',rev::text,'policy_id',p.policy_id,'accounting_date',to_char(day_value,'YYYY-MM-DD'),'description',description_value,'lines',jsonb_build_array(
  jsonb_build_object('account_id',s.ledger_account_id,'side',CASE q.direction WHEN 'inflow' THEN
  'debit'
  ELSE  'credit'
END,'transaction_currency',code,'transaction_amount',amount_value,'functional_amount',amount_value,'valuation_basis','identity'),
  jsonb_build_object('account_id',category_id,'side',CASE q.direction WHEN 'inflow' THEN
  'credit'
  ELSE  'debit'
END,'transaction_currency',code,'transaction_amount',amount_value,'functional_amount',amount_value,'valuation_basis','identity')));

  PERFORM ledger.validate_command_body('post',body);

  UPDATE banking.source_account SET source_revision=source_revision+1
  WHERE entity_id=e AND book_id=b AND source_account_id=s.source_account_id RETURNING * INTO s;

  UPDATE banking.review_item SET state='draft_ready',review_revision=review_revision+1
  WHERE entity_id=e AND book_id=b AND review_id=id RETURNING * INTO r;

  INSERT INTO banking.review_draft(entity_id,book_id,review_id,review_revision,source_revision,binding_revision,book_revision,posting_body,normalized_request,created_by,idempotency_key)

  VALUES(e,b,id,r.review_revision,s.source_revision,s.binding_revision,rev,body,norm,actor,k) RETURNING * INTO d;

  INSERT INTO banking.review_decision(entity_id,book_id,review_id,expected_review_revision,kind,actor_id,reason,body)
  VALUES(e,b,id,r.review_revision-1,'draft',actor,'explicit_category',jsonb_build_object('draft_id',d.draft_id,'category_account_id',category_id,'source_revision',s.source_revision::text));

  RETURN banking.render_review_draft(d);

  EXCEPTION  WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR numeric_value_out_of_range THEN
  PERFORM banking.review_error('P2500');
  WHEN unique_violation THEN
  RAISE
  EXCEPTION  USING ERRCODE='40001',MESSAGE='retry_review_draft';

END $$;

CREATE OR REPLACE FUNCTION banking.source_post_authorize(actor uuid,e uuid,b uuid,body jsonb)
  RETURNS banking.review_draft
  LANGUAGE plpgsql
  SET search_path=pg_catalog AS $$
  DECLARE d banking.review_draft;

BEGIN

  PERFORM banking.review_authorize(actor,e,b,(body->>'review_id')::uuid,true);

  SELECT * INTO d
  FROM banking.review_draft
  WHERE entity_id=e AND book_id=b AND review_id=(body->>'review_id')::uuid AND draft_id=(body->>'draft_id')::uuid;

  IF NOT FOUND THEN
  PERFORM banking.review_error('P2501');

END IF;

  IF ROW(d.review_revision,d.source_revision,d.binding_revision,d.book_revision,d.posting_body) IS DISTINCT
  FROM ROW((body->>'review_revision')::bigint,(body->>'source_revision')::bigint,(body->>'binding_revision')::bigint,(body->>'expected_book_revision')::bigint,body->'posting_body') THEN
  PERFORM banking.review_error('P2502');

END IF;

  RETURN d;

END $$;

CREATE OR REPLACE FUNCTION banking.execute_source_post(actor uuid,e uuid,b uuid,command uuid,request uuid,body jsonb)
  RETURNS jsonb
  LANGUAGE plpgsql
  SET search_path=pg_catalog AS $$
  DECLARE d banking.review_draft;
  r banking.review_item;
  s banking.source_account;
  link banking.review_posting;
  rev bigint;
  posted record;
  result_value jsonb;

BEGIN

  d:=banking.source_post_authorize(actor,e,b,body);

  SELECT * INTO link
  FROM banking.review_posting
  WHERE entity_id=e AND book_id=b AND review_id=d.review_id;

  IF FOUND THEN
  IF link.draft_id<>d.draft_id THEN
  PERFORM banking.review_error('P2502');

END IF;
  RETURN link.result;

END IF;

  SELECT * INTO r
  FROM banking.review_item
  WHERE entity_id=e AND book_id=b AND review_id=d.review_id FOR UPDATE;

  SELECT * INTO s
  FROM banking.source_account
  WHERE entity_id=e AND book_id=b AND source_account_id=r.source_account_id;

  SELECT revision INTO rev
  FROM ledger.book
  WHERE entity_id=e AND book_id=b;

  IF r.state<>'draft_ready' OR s.state<>'active' THEN
  PERFORM banking.review_error('P2502');

END IF;

  IF ROW(r.review_revision,s.source_revision,s.binding_revision,rev) IS DISTINCT
  FROM ROW(d.review_revision,d.source_revision,d.binding_revision,d.book_revision) THEN
  PERFORM banking.review_error('P2503');

END IF;

  IF r.review_revision=9223372036854775807 OR s.source_revision=9223372036854775807 THEN
  PERFORM banking.review_error('P2505');

END IF;

  SELECT * INTO posted
  FROM ledger.post_journal(actor,e,b,'source-command:'||command,request,d.posting_body::text);

  UPDATE banking.source_account SET source_revision=source_revision+1
  WHERE entity_id=e AND book_id=b AND source_account_id=s.source_account_id RETURNING * INTO s;

  UPDATE banking.review_item SET state='posted',review_revision=review_revision+1
  WHERE entity_id=e AND book_id=b AND review_id=r.review_id RETURNING * INTO r;

  result_value:=jsonb_build_object('entity_id',e,'book_id',b,'review_id',r.review_id,'draft_id',d.draft_id,'journal_id',posted.body->>'journal_id','book_revision',posted.body->>'book_revision','journal',posted.body,'review_revision',r.review_revision::text,'source_revision',s.source_revision::text);

  INSERT INTO banking.review_posting(entity_id,book_id,review_id,draft_id,command_id,journal_id,result)
  VALUES(e,b,r.review_id,d.draft_id,command,(posted.body->>'journal_id')::uuid,result_value);

  RETURN result_value;

END $$;

CREATE OR REPLACE FUNCTION banking.render_review_draft(d banking.review_draft)
  RETURNS jsonb
  LANGUAGE sql STABLE
  SET search_path=pg_catalog AS $$
  SELECT CASE WHEN d.draft_id IS NULL THEN
  NULL
  ELSE  jsonb_build_object('draft_id',d.draft_id,'review_id',d.review_id,'review_revision',d.review_revision::text,'source_revision',d.source_revision::text,'binding_revision',d.binding_revision::text,'book_revision',d.book_revision::text,'posting_body',d.posting_body)
END
  $$;

DROP FUNCTION banking.get_automatic_accounting(uuid,uuid,uuid),banking.set_automatic_accounting(uuid,uuid,uuid,text),banking.automatic_accounting_status(uuid,uuid,uuid),banking.automatic_accounting_work(),banking.prepare_automatic_accounting(uuid,uuid,uuid),banking.execute_automatic_accounting(uuid,uuid,uuid,uuid);

DROP FUNCTION banking.automatic_accounting_authorize(uuid,uuid,uuid,boolean),banking.automatic_accounting_source(uuid,uuid,uuid,uuid),banking.automatic_accounting_sources(uuid,uuid,uuid,jsonb),banking.automatic_accounting_policy_wire(uuid,uuid),banking.create_source_draft(uuid,uuid,uuid,uuid,text,text,uuid),banking.commit_source_post(uuid,uuid,uuid,uuid,uuid,uuid,jsonb),banking.authorize_source_post_core(uuid,uuid,uuid,jsonb,uuid),banking.automatic_accounting_inputs(uuid,uuid,uuid,bigint),banking.automatic_accounting_visible(uuid,uuid,uuid,banking.automatic_accounting_operation),banking.render_automatic_accounting_operation(banking.automatic_accounting_operation);

ALTER TABLE banking.review_posting DROP CONSTRAINT review_posting_origin;

ALTER TABLE banking.review_posting DROP COLUMN automatic_operation_id;

ALTER TABLE banking.review_posting ALTER COLUMN command_id SET NOT NULL;

ALTER TABLE banking.review_draft DROP COLUMN automatic_operation_id;

ALTER TABLE banking.review_decision DROP CONSTRAINT review_decision_reason_check;

ALTER TABLE banking.review_decision ADD CONSTRAINT review_decision_reason_check CHECK(reason IN('synthetic_import','explicit_category','provider_sync'));

ALTER TABLE banking.automatic_accounting_policy DROP CONSTRAINT automatic_accounting_policy_entity_id_book_id_revision_fkey;

DROP TABLE banking.automatic_accounting_operation,banking.automatic_accounting_setting,banking.automatic_accounting_edition,banking.automatic_accounting_cursor,banking.automatic_accounting_policy;

DROP FUNCTION banking.automatic_accounting_guard(),banking.automatic_accounting_error(text);

COMMIT;

