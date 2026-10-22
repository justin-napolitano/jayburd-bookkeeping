BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Refuse rollback while exact source evidence depends on the expanded retention profile.
DO $$ BEGIN
 IF EXISTS(SELECT FROM banking.observation_revision q JOIN reference.currency c USING(currency_version_id) WHERE q.adapter_profile='jayburd-plaid-transactions/1' AND scale(trim_scale(q.amount))>c.minor_unit)
 OR EXISTS(SELECT FROM integration.sync_page p CROSS JOIN LATERAL jsonb_array_elements((p.body->'added')||(p.body->'modified')) v JOIN reference.currency c ON c.currency_version_id=(v->>'currency_version_id')::uuid WHERE scale((v->>'amount')::numeric)>c.minor_unit)
 THEN RAISE EXCEPTION 'source_precision_evidence_retained'; END IF;
END $$;
CREATE OR REPLACE FUNCTION integration.sync_row(v jsonb,account_value text,currency_id uuid,removed boolean) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE c reference.currency;bound reference.currency; d date; a numeric; signed_value numeric; lex text; field_name text; text_value text; pending_value boolean; exp_value numeric; coeff text;
BEGIN
 IF removed THEN PERFORM ledger.keys(v,ARRAY['account_id','transaction_id']); ELSE
  IF jsonb_typeof(v)<>'object' OR NOT(v ?& ARRAY['account_id','transaction_id','pending','pending_transaction_id','date','authorized_date','datetime','authorized_datetime','iso_currency_code','currency_version_id','amount_lexeme','amount','direction','eligible_for_review','name','merchant_name','original_description']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['account_id','transaction_id','pending','pending_transaction_id','date','authorized_date','datetime','authorized_datetime','iso_currency_code','currency_version_id','amount_lexeme','amount','direction','eligible_for_review','name','merchant_name','original_description'])) THEN PERFORM integration.sync_error('P2600'); END IF;
 END IF;
 IF integration.sync_text(v->'account_id',256)<>account_value THEN PERFORM integration.sync_error('P2604'); END IF; PERFORM integration.sync_text(v->'transaction_id',256); IF removed THEN RETURN; END IF;
 IF octet_length(v::text)>8192 OR jsonb_typeof(v->'pending')<>'boolean' OR jsonb_typeof(v->'eligible_for_review')<>'boolean' THEN PERFORM integration.sync_error('P2600'); END IF;
 pending_value:=(v->>'pending')::boolean;
 IF v->'pending_transaction_id'<>'null'::jsonb THEN PERFORM integration.sync_text(v->'pending_transaction_id',256); IF pending_value OR v->>'pending_transaction_id'=v->>'transaction_id' THEN PERFORM integration.sync_error('P2604'); END IF; END IF;
 d:=ledger.date_value(ledger.text_value(v->'date'));
 IF v->'authorized_date'<>'null'::jsonb THEN PERFORM ledger.date_value(ledger.text_value(v->'authorized_date')); END IF;
 FOREACH field_name IN ARRAY ARRAY['datetime','authorized_datetime'] LOOP
  IF v->field_name<>'null'::jsonb THEN text_value:=ledger.text_value(v->field_name); IF length(text_value)>40 OR text_value !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$' OR substring(text_value,12,2)>'23' OR substring(text_value,15,2)>'59' OR substring(text_value,18,2)>'59' OR (right(text_value,1)<>'Z' AND (substring(right(text_value,6),2,2)>'23' OR right(text_value,2)>'59')) THEN PERFORM integration.sync_error('P2600'); END IF; PERFORM ledger.date_value(left(text_value,10)); END IF;
 END LOOP;
 FOREACH field_name IN ARRAY ARRAY['name','merchant_name','original_description'] LOOP
  IF v->field_name='null'::jsonb AND field_name<>'name' THEN CONTINUE; END IF;
  text_value:=ledger.text_value(v->field_name); IF length(text_value)>500 OR field_name='name' AND length(text_value)=0 THEN PERFORM integration.sync_error('P2600'); END IF;
 END LOOP;
 -- Binding pins monetary identity; each observation retains its dated version.
 SELECT * INTO bound FROM reference.currency WHERE currency_version_id=currency_id;
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=ledger.uuid_value(v->'currency_version_id');
 IF NOT FOUND OR bound.currency_version_id IS NULL OR c.alphabetic_code<>bound.alphabetic_code OR c.numeric_code IS DISTINCT FROM bound.numeric_code OR ledger.text_value(v->'iso_currency_code')<>c.alphabetic_code OR c.minor_unit IS NULL OR d<c.valid_from OR c.valid_to IS NOT NULL AND d>=c.valid_to THEN PERFORM integration.sync_error('P2604'); END IF;
 text_value:=ledger.text_value(v->'amount'); IF length(text_value)>39 OR text_value !~ '^(0|[1-9][0-9]{0,28})(\.[0-9]{0,8}[1-9])?$' THEN PERFORM integration.sync_error('P2600'); END IF;
 a:=text_value::numeric; IF scale(a)>c.minor_unit THEN PERFORM integration.sync_error('P2604'); END IF;
 lex:=ledger.text_value(v->'amount_lexeme'); IF length(lex)>128 OR lex !~ '^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$' THEN PERFORM integration.sync_error('P2600'); END IF;
 coeff:=regexp_replace(split_part(lower(lex),'e',1),'[-.]','','g'); IF length(trim(both '0' FROM coeff))>38 THEN PERFORM integration.sync_error('P2600'); END IF;
 IF lower(lex) LIKE '%e%' THEN exp_value:=split_part(lower(lex),'e',2)::numeric; IF exp_value NOT BETWEEN -100 AND 100 THEN PERFORM integration.sync_error('P2600'); END IF; END IF;
 signed_value:=lex::numeric;
 IF abs(signed_value)<>a OR ledger.text_value(v->'direction')<>(CASE WHEN signed_value<0 THEN 'inflow' WHEN signed_value>0 THEN 'outflow' ELSE 'zero' END) OR (v->>'eligible_for_review')::boolean<>(NOT pending_value AND a>0) THEN PERFORM integration.sync_error('P2604'); END IF;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range THEN PERFORM integration.sync_error('P2600'); END $$;
CREATE OR REPLACE FUNCTION integration.publish_sync_row(t integration.sync_stream,set_value uuid,page_value integer,op text,v jsonb) RETURNS boolean LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE o banking.source_observation; old banking.observation_revision; q banking.observation_revision; r banking.review_item; ns banking.source_namespace; did uuid; next_rev bigint; state_value text; currency_id uuid; material jsonb; jid uuid;
BEGIN
 SELECT * INTO o FROM banking.source_observation WHERE entity_id=t.entity_id AND book_id=t.book_id AND namespace_id=t.namespace_id AND external_entry_id=v->>'transaction_id';
 IF FOUND THEN
  SELECT * INTO old FROM banking.observation_revision WHERE entity_id=t.entity_id AND book_id=t.book_id AND observation_id=o.observation_id AND revision=o.current_revision;
  IF op='added' THEN IF old.provider_fields=v AND old.source_state<>'removed' THEN RETURN false; END IF; PERFORM integration.sync_error('P2602'); END IF;
  IF op='modified' AND old.provider_fields=v AND old.source_state<>'removed' OR op='removed' AND old.source_state='removed' THEN RETURN false; END IF;
  IF old.adapter_profile<>'jayburd-plaid-transactions/1' OR o.current_revision=9223372036854775807 THEN PERFORM integration.sync_error('P2604'); END IF;
  next_rev:=o.current_revision+1;
 ELSE
  IF op<>'added' THEN PERFORM integration.sync_error('P2604'); END IF;
  INSERT INTO banking.source_observation(entity_id,book_id,source_account_id,namespace_id,external_entry_id) VALUES(t.entity_id,t.book_id,t.source_account_id,t.namespace_id,v->>'transaction_id') RETURNING * INTO o;
  next_rev:=1;
 END IF;
 IF op='removed' THEN
  INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,previous_revision,adapter_profile,provider_amount_lexeme,provider_fields)
  VALUES(t.entity_id,t.book_id,o.observation_id,next_rev,'removed',old.booked_date,old.currency_version_id,old.direction,old.amount_lexeme,old.amount,old.description,old.canonical_fields,old.revision,old.adapter_profile,old.provider_amount_lexeme,old.provider_fields) RETURNING * INTO q;
 ELSE
  state_value:=CASE WHEN (v->>'pending')::boolean THEN 'pending' ELSE 'booked' END;
  material:=jsonb_build_object('date',v->'date','currency_version_id',v->'currency_version_id','direction',v->'direction','amount',v->'amount','description',v->'name');
  INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,previous_revision,adapter_profile,provider_amount_lexeme,provider_fields)
  VALUES(t.entity_id,t.book_id,o.observation_id,next_rev,state_value,(v->>'date')::date,(v->>'currency_version_id')::uuid,v->>'direction',v->>'amount',(v->>'amount')::numeric,v->>'name',material,CASE WHEN next_rev>1 THEN next_rev-1 END,'jayburd-plaid-transactions/1',v->>'amount_lexeme',v) RETURNING * INTO q;
 END IF;
 INSERT INTO banking.provider_revision_origin VALUES(t.entity_id,t.book_id,o.observation_id,next_rev,set_value,page_value,op);
 IF next_rev>1 THEN
  SELECT x.* INTO STRICT r FROM banking.review_item x JOIN banking.review_assignment a USING(entity_id,book_id,review_id) WHERE a.entity_id=t.entity_id AND a.book_id=t.book_id AND a.observation_id=o.observation_id;
  IF r.state='posted' THEN
   SELECT journal_id INTO STRICT jid FROM banking.review_posting WHERE entity_id=t.entity_id AND book_id=t.book_id AND review_id=r.review_id;
   INSERT INTO banking.provider_posted_anchor VALUES(t.entity_id,t.book_id,r.review_id,o.observation_id,old.revision,jid) ON CONFLICT DO NOTHING;
   INSERT INTO banking.provider_review_exception(entity_id,book_id,review_id,observation_id,previous_revision,current_revision,set_id,kind) VALUES(t.entity_id,t.book_id,r.review_id,o.observation_id,old.revision,next_rev,set_value,op);
  END IF;
  UPDATE banking.review_item SET state=CASE WHEN state='posted' THEN 'posted' ELSE 'correction_hold' END,review_revision=review_revision+1 WHERE entity_id=t.entity_id AND book_id=t.book_id AND review_id=r.review_id;
  UPDATE banking.source_observation SET current_revision=next_rev WHERE entity_id=t.entity_id AND book_id=t.book_id AND observation_id=o.observation_id;
 ELSE
  SELECT * INTO ns FROM banking.source_namespace WHERE entity_id=t.entity_id AND book_id=t.book_id AND namespace_id=t.namespace_id;
  INSERT INTO banking.review_item(entity_id,book_id,source_account_id,state) VALUES(t.entity_id,t.book_id,t.source_account_id,CASE WHEN q.source_state<>'booked' OR q.amount=0 THEN 'correction_hold' WHEN ns.overlap_review_required THEN 'overlap_hold' ELSE 'needs_review' END) RETURNING * INTO r;
 END IF;
 INSERT INTO banking.review_decision(entity_id,book_id,review_id,expected_review_revision,kind,actor_id,reason,body) VALUES(t.entity_id,t.book_id,r.review_id,CASE WHEN next_rev=1 THEN 0 ELSE r.review_revision END,CASE WHEN next_rev=1 THEN 'initialize' ELSE 'source_update' END,t.service_principal,'provider_sync',jsonb_build_object('set_id',set_value,'observation_id',o.observation_id,'revision',next_rev::text,'operation',op)) RETURNING decision_id INTO did;
 INSERT INTO banking.review_evidence VALUES(t.entity_id,t.book_id,did,o.observation_id,r.review_id,next_rev);
 IF next_rev=1 THEN INSERT INTO banking.review_assignment VALUES(t.entity_id,t.book_id,o.observation_id,r.review_id,did); END IF;
 RETURN true;
END $$;
CREATE OR REPLACE FUNCTION banking.get_review_item(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE r banking.review_item; s banking.source_account; q banking.observation_revision; d banking.review_draft; o banking.source_observation; rev bigint; blockers jsonb:='[]';
BEGIN
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=id;
 s:=banking.import_authorize(actor,e,b,r.source_account_id,false); q:=banking.review_fact(e,b,id);
 SELECT * INTO o FROM banking.source_observation WHERE entity_id=e AND book_id=b AND observation_id=q.observation_id;
 SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b;
 SELECT * INTO d FROM banking.review_draft WHERE entity_id=e AND book_id=b AND review_id=id ORDER BY review_revision DESC LIMIT 1;
 IF r.state='correction_hold' THEN blockers:=jsonb_build_array('provider_correction_required'); END IF;
 IF r.state='overlap_hold' THEN blockers:=jsonb_build_array('overlap_review_required'); END IF;
 IF r.state='draft_ready' AND ROW(d.source_revision,d.binding_revision,d.book_revision,d.review_revision) IS DISTINCT FROM ROW(s.source_revision,s.binding_revision,rev,r.review_revision) THEN blockers:=blockers||jsonb_build_array('draft_stale'); END IF;
 RETURN banking.render_review_summary(r)||banking.render_provider_exceptions(e,b,id)||jsonb_build_object('profile','source-review-v1','source_account_id',r.source_account_id,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'book_revision',rev::text,'ledger_account_id',s.ledger_account_id,'created_at',r.created_at,'blockers',blockers,'draft',banking.render_review_draft(d),
  'evidence',jsonb_build_array(jsonb_build_object('observation_id',o.observation_id,'observation_revision',q.revision::text,'namespace_id',o.namespace_id,'external_entry_id',o.external_entry_id,'source_state',q.source_state,'booked_date',to_char(q.booked_date,'YYYY-MM-DD'),'currency_version_id',q.currency_version_id,'direction',q.direction,'amount',trim_scale(q.amount)::text,'description',q.description)));
END $$;
COMMIT;
