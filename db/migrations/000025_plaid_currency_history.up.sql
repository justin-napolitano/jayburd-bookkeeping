-- #225 / docs/contracts/plaid-history-currency.md. Preserve immutable source and ledger facts.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Exact reviewed Sandbox metadata only; no runtime reference-data authority.
CREATE FUNCTION reference.install_sandbox_usd_history() RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE old reference.currency; id constant uuid:='3a184659-cf9b-4b80-b140-3f7a7796cbbd'; provenance constant text:='ISO4217-MA115:2002-05-13+SIX-list-one:2026-09-17;local-coverage-v1';
BEGIN
 PERFORM pg_advisory_xact_lock(77421,hashtext('USD'));
 IF NOT EXISTS(SELECT FROM reference.currency WHERE alphabetic_code='USD' AND numeric_code='840' AND minor_unit=2 AND valid_from=DATE '2026-09-17' AND valid_to=DATE '2027-01-01' AND source_release='SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff') THEN RETURN; END IF;
 SELECT * INTO old FROM reference.currency WHERE currency_version_id=id;
 IF FOUND THEN
  IF ROW(old.alphabetic_code,old.numeric_code,old.minor_unit,old.valid_from,old.valid_to,old.source_release) IS DISTINCT FROM ROW('USD'::text,'840'::text,2::smallint,DATE '2001-01-01',DATE '2026-09-17',provenance) THEN RAISE EXCEPTION 'sandbox_history_conflict'; END IF;
  RETURN;
 END IF;
 -- The currency interval trigger rejects any competing/overlapping history atomically.
 INSERT INTO reference.currency VALUES(id,'USD','840',2,'2001-01-01','2026-09-17',provenance);
END $$;
SELECT reference.install_sandbox_usd_history();

CREATE OR REPLACE FUNCTION integration.sync_context(e uuid,b uuid,id uuid,set_value uuid,f bigint) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;x integration.sync_set;i integration.sync_item;s banking.source_account;c reference.currency;versions jsonb;
BEGIN
 t:=integration.sync_authorize(e,b,id);x:=integration.sync_attempt(t,set_value,f);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id;SELECT * INTO c FROM reference.currency WHERE currency_version_id=s.currency_version_id;
 -- Installed history is independent of the book or source binding start date.
 SELECT jsonb_agg(jsonb_build_object('code',h.alphabetic_code,'version_id',h.currency_version_id,'minor_unit',h.minor_unit,'valid_from',to_char(h.valid_from,'YYYY-MM-DD'),'valid_to',to_char(h.valid_to,'YYYY-MM-DD')) ORDER BY h.valid_from,h.currency_version_id) INTO versions
 FROM (SELECT h.* FROM reference.currency h WHERE h.alphabetic_code=c.alphabetic_code AND h.numeric_code IS NOT DISTINCT FROM c.numeric_code AND h.minor_unit IS NOT NULL ORDER BY h.valid_from,h.currency_version_id LIMIT 33) h;
 IF versions IS NULL OR jsonb_array_length(versions)>32 THEN PERFORM integration.sync_error('P2605'); END IF;
 RETURN jsonb_build_object('stream_id',id,'set_id',set_value,'fence',x.fence::text,'provider',i.provider,'environment',i.environment,'item_id',i.item_id,'account_id',t.account_id,'item_generation',i.generation::text,'credential_ref',i.credential_ref,'credential_version',i.credential_version::text,'binding_revision',x.binding_revision::text,'starting_cursor',x.starting_cursor,'request_cursor',x.continuation_cursor,'next_page_index',x.page_count,'page_count',x.page_count,'byte_count',x.byte_count,'change_count',x.change_count,'final_page',x.final_page,'lease_until',t.lease_until,'currency',jsonb_build_object('code',c.alphabetic_code,'version_id',c.currency_version_id,'minor_unit',c.minor_unit,'valid_from',to_char(c.valid_from,'YYYY-MM-DD'),'valid_to',to_char(c.valid_to,'YYYY-MM-DD')),'currency_versions',versions,'limits',jsonb_build_object('max_pages',10,'max_changes',1000,'max_bytes',2097152,'max_page_changes',500));
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
COMMIT;
