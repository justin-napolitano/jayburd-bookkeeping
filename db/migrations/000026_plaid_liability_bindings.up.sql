-- #210: reviewed credit-card and student/mortgage liability enrollment.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE OR REPLACE FUNCTION integration.record_connection_manifest(actor uuid,e uuid,b uuid,id uuid,claim_value uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;r jsonb;expiry timestamptz;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);v:=integration.connection_json(raw,32768);PERFORM ledger.keys(v,ARRAY['item_id','accounts','consent_expires_at']);PERFORM integration.sync_text(v->'item_id',256);
 IF jsonb_typeof(v->'accounts')<>'array' OR (CASE WHEN jsonb_typeof(v->'accounts')='array' THEN jsonb_array_length(v->'accounts') ELSE -1 END) NOT BETWEEN 1 AND 25 OR ledger.text_value(v->'consent_expires_at') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$' THEN PERFORM integration.connection_error('P2700');END IF;
 expiry:=integration.connection_expiry(v->'consent_expires_at');IF expiry<=clock_timestamp() THEN PERFORM integration.connection_error('P2703');END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(v->'accounts') LOOP
  IF jsonb_typeof(r) IS DISTINCT FROM 'object' THEN PERFORM integration.connection_error('P2700');END IF;
  IF NOT(r ?& ARRAY['account_id','type','iso_currency_code']) OR EXISTS(SELECT FROM jsonb_object_keys(r) z WHERE z<>ALL(ARRAY['account_id','type','subtype','iso_currency_code'])) THEN PERFORM integration.connection_error('P2700');END IF;
  PERFORM integration.sync_text(r->'account_id',256);IF ledger.text_value(r->'type') NOT IN ('depository','credit','loan','investment','other','payroll') OR r?'subtype' AND r->'subtype'<>'null'::jsonb AND ledger.text_value(r->'subtype') !~ '^[a-z0-9][a-z0-9 -]{0,63}$' OR r->'iso_currency_code'<>'null'::jsonb AND ledger.text_value(r->'iso_currency_code') !~ '^[A-Z]{3}$' THEN PERFORM integration.connection_error('P2700');END IF;
 END LOOP;
 IF EXISTS(SELECT FROM jsonb_array_elements(v->'accounts') z GROUP BY value->>'account_id' HAVING count(*)>1) THEN PERFORM integration.connection_error('P2700');END IF;
 IF a.claim_id IS DISTINCT FROM claim_value OR claim_value IS NULL OR a.mode='remove' THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.manifest IS NOT NULL THEN IF a.manifest<>v THEN PERFORM integration.connection_error('P2702');END IF;RETURN integration.render_connection(a,true);END IF;
 IF a.state NOT IN ('exchange_claimed','credential_recovery_required') THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.expected_item_id IS NOT NULL AND a.expected_item_id<>v->>'item_id' THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.sync_item_id IS NOT NULL THEN
  PERFORM integration.connection_item_pin(a);
  IF EXISTS(SELECT FROM integration.sync_stream t JOIN banking.source_account s USING(entity_id,book_id,source_account_id) JOIN reference.currency c ON c.currency_version_id=s.currency_version_id JOIN ledger.account la ON la.entity_id=s.entity_id AND la.book_id=s.book_id AND la.account_id=s.ledger_account_id WHERE t.entity_id=e AND t.book_id=b AND t.sync_item_id=a.sync_item_id AND t.state='active' AND NOT EXISTS(SELECT FROM jsonb_array_elements(v->'accounts') z WHERE z.value->>'account_id'=t.account_id AND z.value->>'iso_currency_code'=c.alphabetic_code AND ((z.value->>'type'='depository' AND la.category='asset' AND la.normal_side='debit') OR (z.value->>'type'='credit' AND z.value->>'subtype'='credit card' AND la.category='liability' AND la.normal_side='credit') OR (z.value->>'type'='loan' AND z.value->>'subtype' IN ('student','mortgage') AND la.category='liability' AND la.normal_side='credit')))) THEN PERFORM integration.connection_error('P2704');END IF;
 END IF;
 UPDATE integration.connection_attempt SET manifest=v,state='credential_pending' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'manifest',v);RETURN integration.render_connection(a,true);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2600' OR invalid_datetime_format OR datetime_field_overflow THEN PERFORM integration.connection_error('P2700');END $$;
-- Create reviewed cash-like depository accounts atomically with their source bindings.
CREATE OR REPLACE FUNCTION integration.bind_connection(actor uuid,e uuid,b uuid,id uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;w integration.connection_worker;i integration.sync_item;v jsonb;r jsonb;provider_row jsonb;new_account jsonb;s banking.source_account;la ledger.account;c reference.currency;ns uuid;stream_value uuid;d date;cid uuid;lid uuid;sid uuid;new_source boolean;code_value text;name_value text;category_value text;side_value text;result_value jsonb:='[]';p uuid;cap text;rec integration.connection_receipt;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);PERFORM integration.connection_key(k);v:=integration.connection_json(raw,32768);PERFORM ledger.keys(v,ARRAY['consent_grants','bindings']);
 IF v->'consent_grants' IS DISTINCT FROM 'true'::jsonb OR jsonb_typeof(v->'bindings')<>'array' OR (CASE WHEN jsonb_typeof(v->'bindings')='array' THEN jsonb_array_length(v->'bindings') ELSE -1 END) NOT BETWEEN 1 AND 25 THEN PERFORM integration.connection_error('P2700');END IF;
 IF a.mode<>'new' THEN PERFORM integration.connection_error('P2704');END IF;
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='bind' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>jsonb_build_object('attempt_id',id,'body',v) THEN PERFORM integration.connection_error('P2702');END IF;RETURN rec.result;END IF;
 IF a.state='active' THEN PERFORM integration.connection_error('P2702');END IF;
 IF a.state<>'binding_review' OR clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2703');END IF;
 i:=integration.connection_item_pin(a);w:=integration.connection_authorize(actor,e,b);
 IF EXISTS(SELECT FROM jsonb_array_elements(v->'bindings') z GROUP BY value->>'account_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(v->'bindings') z GROUP BY value->>'ledger_account_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(v->'bindings') z WHERE value->'source_account_id'<>'null'::jsonb GROUP BY value->>'source_account_id' HAVING count(*)>1) THEN PERFORM integration.connection_error('P2700');END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(v->'bindings') ORDER BY value->>'ledger_account_id' LOOP
  IF jsonb_typeof(r) IS DISTINCT FROM 'object' THEN PERFORM integration.connection_error('P2700');END IF;
  IF NOT(r ?& ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref']) OR EXISTS(SELECT FROM jsonb_object_keys(r) z WHERE z<>ALL(ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref','new_ledger_account'])) THEN PERFORM integration.connection_error('P2700');END IF;
  PERFORM integration.sync_text(r->'account_id',256);cid:=ledger.uuid_value(r->'currency_version_id');lid:=ledger.uuid_value(r->'ledger_account_id');d:=ledger.date_value(ledger.text_value(r->'binding_date'));IF ledger.uuid_value(r->'binding_evidence_ref')='00000000-0000-0000-0000-000000000000' THEN PERFORM integration.connection_error('P2700');END IF;
  SELECT value INTO provider_row FROM jsonb_array_elements(a.manifest->'accounts') WHERE value->>'account_id'=r->>'account_id';
  IF NOT FOUND OR provider_row->'iso_currency_code'='null'::jsonb THEN PERFORM integration.connection_error('P2704');END IF;
  IF provider_row->>'type'='depository' THEN category_value:='asset';side_value:='debit';
  ELSIF provider_row->>'type'='credit' AND provider_row->>'subtype'='credit card' OR provider_row->>'type'='loan' AND provider_row->>'subtype' IN ('student','mortgage') THEN category_value:='liability';side_value:='credit';
  ELSE PERFORM integration.connection_error('P2704');END IF;
  SELECT * INTO c FROM reference.currency WHERE currency_version_id=cid;
  IF NOT FOUND OR c.alphabetic_code<>provider_row->>'iso_currency_code' OR c.minor_unit IS NULL OR d<c.valid_from OR c.valid_to IS NOT NULL AND d>=c.valid_to THEN PERFORM integration.connection_error('P2704');END IF;
  new_source:=r->'source_account_id'='null'::jsonb;new_account:=r->'new_ledger_account';
  IF new_account IS NOT NULL AND new_account<>'null'::jsonb THEN
   IF jsonb_typeof(new_account)<>'object' OR NOT new_source OR r->'expected_binding_revision'<>'null'::jsonb THEN PERFORM integration.connection_error('P2700');END IF;
   PERFORM ledger.keys(new_account,ARRAY['account_id','code','name','category','normal_side','monetary','active_from']);
   IF ledger.uuid_value(new_account->'account_id')<>lid OR new_account->>'category'<>category_value OR new_account->>'normal_side'<>side_value OR new_account->'monetary' IS DISTINCT FROM 'true'::jsonb OR ledger.date_value(ledger.text_value(new_account->'active_from'))<>d THEN PERFORM integration.connection_error('P2700');END IF;
   IF category_value='asset' AND (provider_row->>'subtype' IS NULL OR provider_row->>'subtype' NOT IN ('checking','savings','money market','cash management')) THEN PERFORM integration.connection_error('P2704');END IF;
   code_value:=ledger.setup_text(new_account->'code',64);name_value:=ledger.setup_text(new_account->'name',200);
   INSERT INTO ledger.account(entity_id,book_id,account_id,code,name,category,normal_side,monetary,active_from,active_to) VALUES(e,b,lid,code_value,name_value,category_value,side_value,true,d,NULL);
  END IF;
  SELECT * INTO la FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id=lid FOR SHARE;
  IF NOT FOUND OR la.category<>category_value OR la.normal_side<>side_value OR NOT la.monetary OR d<la.active_from OR la.active_to IS NOT NULL AND d>=la.active_to THEN PERFORM integration.connection_error('P2704');END IF;
  IF new_source THEN
   IF r->'expected_binding_revision'<>'null'::jsonb OR EXISTS(SELECT FROM banking.source_account WHERE entity_id=e AND book_id=b AND ledger_account_id=lid) THEN PERFORM integration.connection_error('P2702');END IF;
   INSERT INTO banking.source_account(entity_id,book_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by) VALUES(e,b,lid,cid,d,ledger.uuid_value(r->'binding_evidence_ref'),actor) RETURNING * INTO s;
   FOR p IN SELECT DISTINCT x FROM unnest(ARRAY[actor,w.service_principal]) x LOOP
    FOREACH cap IN ARRAY ARRAY['read','import'] LOOP INSERT INTO banking.source_permission VALUES(e,b,s.source_account_id,p,cap,NULL);END LOOP;
   END LOOP;
   INSERT INTO banking.source_permission VALUES(e,b,s.source_account_id,actor,'bind',NULL);
  ELSE
   sid:=ledger.uuid_value(r->'source_account_id');s:=integration.connection_source(actor,e,b,sid);
   IF s.state<>'active' OR s.ledger_account_id<>lid OR s.currency_version_id<>cid OR s.binding_revision<>banking.import_revision(r->'expected_binding_revision') THEN PERFORM integration.connection_error('P2703');END IF;
   PERFORM integration.sync_authority(e,b,sid,actor,w.service_principal);
   IF (SELECT count(*) FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND source_account_id=sid)>=25 THEN PERFORM integration.connection_error('P2705');END IF;
   UPDATE banking.source_account SET binding_revision=binding_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=sid RETURNING * INTO s;
  END IF;
  INSERT INTO banking.source_namespace(entity_id,book_id,source_account_id,profile,provider,environment,item_id,provider_account_id,overlap_review_required,created_by) VALUES(e,b,s.source_account_id,'jayburd-plaid-transactions/1','plaid','sandbox',i.item_id,r->>'account_id',NOT new_source,actor) RETURNING namespace_id INTO ns;
  INSERT INTO integration.sync_stream(entity_id,book_id,sync_item_id,source_account_id,namespace_id,account_id,worker_login,service_principal) VALUES(e,b,i.sync_item_id,s.source_account_id,ns,r->>'account_id',w.worker_login,w.service_principal) RETURNING stream_id INTO stream_value;
  result_value:=result_value||jsonb_build_array(jsonb_build_object('stream_id',stream_value,'source_account_id',s.source_account_id,'namespace_id',ns,'binding_revision',s.binding_revision::text,'overlap_review_required',NOT new_source));
 END LOOP;
 UPDATE integration.sync_item SET state='active',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
 UPDATE integration.connection_attempt SET state='active',bind_body=v,streams=result_value,item_generation=i.generation WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 result_value:=integration.render_connection(a,false);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'bind',k,jsonb_build_object('attempt_id',id,'body',v),result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_error('P2700');WHEN SQLSTATE 'P2601' THEN PERFORM integration.connection_error('P2701');WHEN unique_violation THEN PERFORM integration.connection_error('P2702');END $$;
CREATE OR REPLACE FUNCTION integration.sync_context(e uuid,b uuid,id uuid,set_value uuid,f bigint) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;x integration.sync_set;i integration.sync_item;s banking.source_account;c reference.currency;versions jsonb;account_type text;account_subtype text;account_category text;
BEGIN
 t:=integration.sync_authorize(e,b,id);x:=integration.sync_attempt(t,set_value,f);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id;SELECT * INTO c FROM reference.currency WHERE currency_version_id=s.currency_version_id;
 SELECT z.value->>'type',z.value->>'subtype' INTO account_type,account_subtype FROM integration.connection_attempt a CROSS JOIN LATERAL jsonb_array_elements(a.manifest->'accounts') z WHERE a.entity_id=e AND a.book_id=b AND a.sync_item_id=t.sync_item_id AND a.state='active' AND z.value->>'account_id'=t.account_id ORDER BY a.item_generation DESC NULLS LAST LIMIT 1;
 SELECT category INTO account_category FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id=s.ledger_account_id;
 IF account_category='liability' AND (account_type='credit' AND account_subtype='credit card' OR account_type='loan' AND account_subtype IN ('student','mortgage')) IS NOT TRUE THEN PERFORM integration.sync_error('P2605');END IF;
 -- Installed history is independent of the book or source binding start date.
 SELECT jsonb_agg(jsonb_build_object('code',h.alphabetic_code,'version_id',h.currency_version_id,'minor_unit',h.minor_unit,'valid_from',to_char(h.valid_from,'YYYY-MM-DD'),'valid_to',to_char(h.valid_to,'YYYY-MM-DD')) ORDER BY h.valid_from,h.currency_version_id) INTO versions
 FROM (SELECT h.* FROM reference.currency h WHERE h.alphabetic_code=c.alphabetic_code AND h.numeric_code IS NOT DISTINCT FROM c.numeric_code AND h.minor_unit IS NOT NULL ORDER BY h.valid_from,h.currency_version_id LIMIT 33) h;
 IF versions IS NULL OR jsonb_array_length(versions)>32 THEN PERFORM integration.sync_error('P2605'); END IF;
 RETURN jsonb_build_object('stream_id',id,'set_id',set_value,'fence',x.fence::text,'provider',i.provider,'environment',i.environment,'item_id',i.item_id,'account_id',t.account_id,'account_type',coalesce(account_type,'depository'),'account_subtype',account_subtype,'item_generation',i.generation::text,'credential_ref',i.credential_ref,'credential_version',i.credential_version::text,'binding_revision',x.binding_revision::text,'starting_cursor',x.starting_cursor,'request_cursor',x.continuation_cursor,'next_page_index',x.page_count,'page_count',x.page_count,'byte_count',x.byte_count,'change_count',x.change_count,'final_page',x.final_page,'lease_until',t.lease_until,'currency',jsonb_build_object('code',c.alphabetic_code,'version_id',c.currency_version_id,'minor_unit',c.minor_unit,'valid_from',to_char(c.valid_from,'YYYY-MM-DD'),'valid_to',to_char(c.valid_to,'YYYY-MM-DD')),'currency_versions',versions,'limits',jsonb_build_object('max_pages',10,'max_changes',1000,'max_bytes',2097152,'max_page_changes',500));
END $$;

-- The existing source-review draft formula is asset-only. Keep liability
-- observations read-only until a separate posting-direction policy is reviewed.
CREATE FUNCTION banking.reject_liability_review_draft() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE category_value text; side_value text;
BEGIN
 SELECT a.category,a.normal_side INTO category_value,side_value
 FROM banking.review_item r JOIN banking.source_account s USING(entity_id,book_id,source_account_id)
 JOIN ledger.account a ON a.entity_id=s.entity_id AND a.book_id=s.book_id AND a.account_id=s.ledger_account_id
 WHERE r.entity_id=NEW.entity_id AND r.book_id=NEW.book_id AND r.review_id=NEW.review_id;
 IF category_value IS DISTINCT FROM 'asset' OR side_value IS DISTINCT FROM 'debit' THEN
  PERFORM banking.review_error('P2504');
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER liability_review_draft_guard BEFORE INSERT ON banking.review_draft
 FOR EACH ROW EXECUTE FUNCTION banking.reject_liability_review_draft();
COMMIT;
