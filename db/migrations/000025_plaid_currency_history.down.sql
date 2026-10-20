-- Refuse rollback that would orphan or erase reviewed historical reference evidence.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN IF EXISTS(SELECT FROM reference.currency WHERE currency_version_id='3a184659-cf9b-4b80-b140-3f7a7796cbbd') THEN RAISE EXCEPTION 'currency_history_rollback_requires_preservation'; END IF; END $$;
CREATE OR REPLACE FUNCTION integration.sync_row(v jsonb,account_value text,currency_id uuid,removed boolean) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE c reference.currency; d date; a numeric; signed_value numeric; lex text; field_name text; text_value text; pending_value boolean; exp_value numeric; coeff text;
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
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=currency_id;
 IF ledger.uuid_value(v->'currency_version_id')<>currency_id OR ledger.text_value(v->'iso_currency_code')<>c.alphabetic_code OR c.minor_unit IS NULL OR d<c.valid_from OR c.valid_to IS NOT NULL AND d>=c.valid_to THEN PERFORM integration.sync_error('P2604'); END IF;
 text_value:=ledger.text_value(v->'amount'); IF length(text_value)>39 OR text_value !~ '^(0|[1-9][0-9]{0,28})(\.[0-9]{0,8}[1-9])?$' THEN PERFORM integration.sync_error('P2600'); END IF;
 a:=text_value::numeric; IF scale(a)>c.minor_unit THEN PERFORM integration.sync_error('P2604'); END IF;
 lex:=ledger.text_value(v->'amount_lexeme'); IF length(lex)>128 OR lex !~ '^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$' THEN PERFORM integration.sync_error('P2600'); END IF;
 coeff:=regexp_replace(split_part(lower(lex),'e',1),'[-.]','','g'); IF length(trim(both '0' FROM coeff))>38 THEN PERFORM integration.sync_error('P2600'); END IF;
 IF lower(lex) LIKE '%e%' THEN exp_value:=split_part(lower(lex),'e',2)::numeric; IF exp_value NOT BETWEEN -100 AND 100 THEN PERFORM integration.sync_error('P2600'); END IF; END IF;
 signed_value:=lex::numeric;
 IF abs(signed_value)<>a OR ledger.text_value(v->'direction')<>(CASE WHEN signed_value<0 THEN 'inflow' WHEN signed_value>0 THEN 'outflow' ELSE 'zero' END) OR (v->>'eligible_for_review')::boolean<>(NOT pending_value AND a>0) THEN PERFORM integration.sync_error('P2604'); END IF;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range THEN PERFORM integration.sync_error('P2600'); END $$;

CREATE OR REPLACE FUNCTION integration.sync_context(e uuid,b uuid,id uuid,set_value uuid,f bigint) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;x integration.sync_set;i integration.sync_item;s banking.source_account;c reference.currency;
BEGIN
 t:=integration.sync_authorize(e,b,id);x:=integration.sync_attempt(t,set_value,f);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id;SELECT * INTO c FROM reference.currency WHERE currency_version_id=s.currency_version_id;
 RETURN jsonb_build_object('stream_id',id,'set_id',set_value,'fence',x.fence::text,'provider',i.provider,'environment',i.environment,'item_id',i.item_id,'account_id',t.account_id,'item_generation',i.generation::text,'credential_ref',i.credential_ref,'credential_version',i.credential_version::text,'binding_revision',x.binding_revision::text,'starting_cursor',x.starting_cursor,'request_cursor',x.continuation_cursor,'next_page_index',x.page_count,'page_count',x.page_count,'byte_count',x.byte_count,'change_count',x.change_count,'final_page',x.final_page,'lease_until',t.lease_until,'currency',jsonb_build_object('code',c.alphabetic_code,'version_id',c.currency_version_id,'minor_unit',c.minor_unit,'valid_from',to_char(c.valid_from,'YYYY-MM-DD'),'valid_to',to_char(c.valid_to,'YYYY-MM-DD')),'limits',jsonb_build_object('max_pages',10,'max_changes',1000,'max_bytes',2097152,'max_page_changes',500));
END $$;
DROP FUNCTION reference.install_sandbox_usd_history();
COMMIT;
