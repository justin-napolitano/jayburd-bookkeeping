-- Preserve accepted registration limits and all reviewed/terminal command history.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN IF EXISTS(SELECT FROM integration.registration_limit) OR EXISTS(SELECT FROM ledger.interactive_command WHERE operation='register_accounts') THEN PERFORM integration.connection_error('P2702');END IF;END $$;
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
 -- #308's bounded Production pilot permits one selected/bound source; paused siblings still count.
 -- A known committed bind receipt was already returned above, before checking new-work limits.
 IF a.environment='production' AND (jsonb_array_length(v->'bindings')<>1
 OR EXISTS(SELECT FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND state<>'disconnected' AND sync_item_id<>a.sync_item_id))
 THEN PERFORM integration.connection_error('P2705');END IF;
 i:=integration.connection_item_pin(a);w:=integration.connection_authorize(actor,e,b);
 IF EXISTS(SELECT FROM jsonb_array_elements(v->'bindings') z GROUP BY value->>'account_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(v->'bindings') z GROUP BY value->>'ledger_account_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(v->'bindings') z WHERE value->'source_account_id'<>'null'::jsonb GROUP BY value->>'source_account_id' HAVING count(*)>1) THEN PERFORM integration.connection_error('P2700');END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(v->'bindings') ORDER BY value->>'ledger_account_id' LOOP
  IF jsonb_typeof(r) IS DISTINCT FROM 'object' THEN PERFORM integration.connection_error('P2700');END IF;
  IF NOT(r ?& ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref']) OR EXISTS(SELECT FROM jsonb_object_keys(r) z WHERE z<>ALL(ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref','new_ledger_account'])) THEN PERFORM integration.connection_error('P2700');END IF;
  PERFORM integration.sync_text(r->'account_id',256);cid:=ledger.uuid_value(r->'currency_version_id');lid:=ledger.uuid_value(r->'ledger_account_id');d:=ledger.date_value(ledger.text_value(r->'binding_date'));IF ledger.uuid_value(r->'binding_evidence_ref')='00000000-0000-0000-0000-000000000000' THEN PERFORM integration.connection_error('P2700');END IF;
  SELECT value INTO provider_row FROM jsonb_array_elements(a.manifest->'accounts') WHERE value->>'account_id'=r->>'account_id';
  IF NOT FOUND OR provider_row->'iso_currency_code'='null'::jsonb THEN PERFORM integration.connection_error('P2704');END IF;
  IF provider_row->>'type'='depository' THEN category_value:='asset';side_value:='debit';
  ELSIF provider_row->>'type'='credit' AND provider_row->>'subtype'='credit card' OR provider_row->>'type'='loan' AND provider_row->>'subtype' IN ('student','mortgage','auto','consumer','line of credit') THEN category_value:='liability';side_value:='credit';
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
  INSERT INTO banking.source_namespace(entity_id,book_id,source_account_id,profile,provider,environment,item_id,provider_account_id,overlap_review_required,created_by) VALUES(e,b,s.source_account_id,'jayburd-plaid-transactions/1','plaid',i.environment,i.item_id,r->>'account_id',NOT new_source,actor) RETURNING namespace_id INTO ns;
  INSERT INTO integration.sync_stream(entity_id,book_id,sync_item_id,source_account_id,namespace_id,account_id,worker_login,service_principal) VALUES(e,b,i.sync_item_id,s.source_account_id,ns,r->>'account_id',w.worker_login,w.service_principal) RETURNING stream_id INTO stream_value;
  result_value:=result_value||jsonb_build_array(jsonb_build_object('stream_id',stream_value,'source_account_id',s.source_account_id,'namespace_id',ns,'binding_revision',s.binding_revision::text,'overlap_review_required',NOT new_source));
 END LOOP;
 UPDATE integration.sync_item SET state='active',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
 UPDATE integration.connection_attempt SET state='active',bind_body=v,streams=result_value,item_generation=i.generation WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 result_value:=integration.render_connection(a,false);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'bind',k,jsonb_build_object('attempt_id',id,'body',v),result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_error('P2700');WHEN SQLSTATE 'P2601' THEN PERFORM integration.connection_error('P2701');WHEN unique_violation THEN PERFORM integration.connection_error('P2702');END $$;
CREATE OR REPLACE FUNCTION ledger.command_authorize(actor uuid,c ledger.interactive_command) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE e uuid; b uuid; x banking.import_batch;
BEGIN
 IF c.principal_id IS DISTINCT FROM actor THEN PERFORM ledger.fail('P2101'); END IF;
 IF c.operation='source_post' THEN PERFORM banking.source_post_authorize(actor,c.entity_id,c.book_id,c.request_body); RETURN; END IF;
 IF c.operation='import' THEN
  SELECT * INTO x FROM banking.import_batch WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid;
  PERFORM banking.import_authorize(actor,c.entity_id,c.book_id,x.source_account_id,true);
  RETURN;
 END IF;
 IF c.operation='post' THEN e:=c.entity_id; b:=c.book_id;
 ELSIF c.state='committed' THEN e:=c.result_entity_id; b:=c.result_book_id; END IF;
 IF e IS NOT NULL THEN
  PERFORM ledger.authorize(actor,e,b,'read',true);
  PERFORM ledger.authorize(actor,e,b,'post',true);
 END IF;
END $$;
CREATE OR REPLACE FUNCTION ledger.transition_command(actor uuid,id uuid,request uuid,action_name text,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE c ledger.interactive_command; posted record; value jsonb;
BEGIN
 IF id IS NULL OR request IS NULL OR raw IS NULL OR action_name IS NULL OR action_name NOT IN ('execute','cancel','acknowledge') THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN value:=raw::jsonb; EXCEPTION WHEN untranslatable_character THEN PERFORM ledger.fail('P2100'); END;
 IF value<>'{}'::jsonb THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE command_id=id AND principal_id=actor;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 PERFORM ledger.command_authorize(actor,c);
 PERFORM 1 FROM ledger.interactive_command WHERE command_id=id FOR UPDATE;
 IF action_name='execute' THEN
  IF c.state='cancelled' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.state='prepared' THEN
   IF c.operation='source_post' THEN value:=banking.execute_source_post(actor,c.entity_id,c.book_id,c.command_id,request,c.request_body);
   ELSIF c.operation='import' THEN value:=banking.execute_import(actor,c.entity_id,c.book_id,c.request_body);
   ELSE
   IF c.operation='setup' THEN SELECT * INTO posted FROM ledger.setup_book(actor,'command:'||c.command_id,request,c.request_body::text);
   ELSE SELECT * INTO posted FROM ledger.post_journal(actor,c.entity_id,c.book_id,'command:'||c.command_id,request,c.request_body::text); END IF;
   value:=posted.body;
   END IF;
   UPDATE ledger.interactive_command SET state='committed',result=value,result_entity_id=(value->>'entity_id')::uuid,result_book_id=(value->>'book_id')::uuid,terminal_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'execute',request);
  END IF;
 ELSIF action_name='cancel' THEN
  IF c.state='committed' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.state='prepared' THEN
   IF c.operation='import' THEN
    UPDATE banking.import_batch SET state='cancelled' WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid AND state IN ('received','preview_ready') AND (SELECT preview_id FROM banking.import_preview WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid ORDER BY preview_revision DESC LIMIT 1)=(c.request_body->>'preview_id')::uuid;
    IF EXISTS(SELECT FROM banking.import_batch WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid AND state='imported') THEN PERFORM ledger.fail('P2113'); END IF;
   END IF;
   UPDATE ledger.interactive_command SET state='cancelled',terminal_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'cancel',request);
  END IF;
 ELSE
  IF c.state='prepared' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.acknowledged_at IS NULL THEN
   UPDATE ledger.interactive_command SET acknowledged_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'acknowledge',request);
  END IF;
 END IF;
 RETURN ledger.render_command(c);
END $$;
DROP FUNCTION integration.prepare_registration(uuid,uuid,uuid,uuid,text,uuid,text),integration.execute_registration(uuid,uuid,uuid,uuid,jsonb),integration.apply_registration_bindings(uuid,uuid,uuid,integration.sync_item,jsonb,jsonb),integration.registration_capacity(uuid,uuid,integration.sync_item,jsonb),integration.configure_registration_limit(uuid,uuid,uuid,text,text);
DROP TABLE integration.registration_limit;
DROP FUNCTION integration.registration_limit_guard();
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_operation_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_operation_check CHECK(operation IN ('setup','post','import','source_post'));
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_check CHECK((operation='setup' AND entity_id IS NULL AND book_id IS NULL) OR (operation IN ('post','import','source_post') AND entity_id IS NOT NULL AND book_id IS NOT NULL));
COMMIT;
