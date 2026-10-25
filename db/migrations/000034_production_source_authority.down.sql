-- #311: refuse destructive rollback once any explicit environment authority was installed.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN
 IF EXISTS(SELECT FROM integration.provider_scope)
 OR EXISTS(SELECT FROM integration.connection_attempt WHERE environment='production')
 OR EXISTS(SELECT FROM integration.sync_item WHERE environment='production')
 OR EXISTS(SELECT FROM banking.source_namespace WHERE environment='production')
 OR EXISTS(SELECT FROM integration.bank_credential WHERE environment='production')
 THEN RAISE EXCEPTION 'production_scope_rollback_rejected';END IF;
END $$;
DROP FUNCTION integration.connection_provider_context(uuid,uuid,uuid);
DROP FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text);
ALTER FUNCTION integration.begin_connection_before_production34(uuid,uuid,uuid,text,text) RENAME TO begin_connection;
GRANT EXECUTE ON FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) TO bookkeeping_api;
DROP TRIGGER preserve_connection_environment ON integration.connection_attempt;
DROP FUNCTION integration.connection_environment_guard();
DROP TRIGGER preserve_provider_origin ON integration.sync_item;
DROP TRIGGER preserve_provider_origin ON integration.bank_credential;
DROP TRIGGER preserve_provider_origin ON banking.source_namespace;
DROP FUNCTION integration.provider_origin_guard();
DROP FUNCTION integration.connection_authorize(uuid,uuid,uuid);
ALTER FUNCTION integration.connection_authorize_before_production34(uuid,uuid,uuid) RENAME TO connection_authorize;
DROP FUNCTION integration.connection_lock(uuid,uuid,uuid,uuid);
ALTER FUNCTION integration.connection_lock_before_production34(uuid,uuid,uuid,uuid) RENAME TO connection_lock;
DROP FUNCTION integration.connection_item_pin(integration.connection_attempt);
ALTER FUNCTION integration.connection_item_pin_before_production34(integration.connection_attempt) RENAME TO connection_item_pin;
DROP FUNCTION integration.render_connection(integration.connection_attempt,boolean);
ALTER FUNCTION integration.render_connection_before_production34(integration.connection_attempt,boolean) RENAME TO render_connection;
DROP FUNCTION integration.sync_authorize(uuid,uuid,uuid);
ALTER FUNCTION integration.sync_authorize_before_production34(uuid,uuid,uuid) RENAME TO sync_authorize;
CREATE OR REPLACE FUNCTION integration.confirm_connection_credential_before_owned_repair32(actor uuid,e uuid,b uuid,id uuid,claim_value uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;s uuid;w integration.connection_worker;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF claim_value IS NULL OR a.claim_id IS DISTINCT FROM claim_value OR a.mode='remove' THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.state IN ('active','binding_review') THEN RETURN integration.render_connection(a,true);END IF;
 IF a.state NOT IN ('credential_pending','credential_recovery_required') OR a.manifest IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 IF (a.manifest->>'consent_expires_at')::timestamptz<=clock_timestamp() THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.mode='new' THEN
  INSERT INTO integration.sync_item(entity_id,book_id,provider,environment,item_id,credential_ref,credential_version,consent_principal,consent_evidence_ref,consent_expires_at,state,synthetic) VALUES(e,b,'plaid','sandbox',a.manifest->>'item_id',a.credential_ref,a.credential_version,actor,a.attempt_id,(a.manifest->>'consent_expires_at')::timestamptz,'paused',true) RETURNING * INTO i;
 ELSE
  i:=integration.connection_item_pin(a);SELECT * INTO w FROM integration.connection_worker WHERE entity_id=e AND book_id=b;
  FOR s IN SELECT source_account_id FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id AND state='active' ORDER BY source_account_id LOOP
   PERFORM integration.sync_authority(e,b,s,actor,w.service_principal);
  END LOOP;
  UPDATE integration.sync_item SET credential_ref=a.credential_ref,credential_version=a.credential_version,consent_principal=actor,consent_evidence_ref=a.attempt_id,consent_expires_at=(a.manifest->>'consent_expires_at')::timestamptz,state='active',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
 END IF;
 INSERT INTO integration.credential_publication(entity_id,book_id,sync_item_id,credential_version,credential_ref,attempt_id) VALUES(e,b,i.sync_item_id,a.credential_version,a.credential_ref,id) ON CONFLICT(entity_id,book_id,sync_item_id,credential_version) DO NOTHING;
 IF NOT EXISTS(SELECT FROM integration.credential_publication WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id AND credential_version=a.credential_version AND credential_ref=a.credential_ref) THEN PERFORM integration.connection_error('P2702');END IF;
 UPDATE integration.connection_attempt SET sync_item_id=i.sync_item_id,item_generation=i.generation,state=CASE WHEN mode='new' THEN 'binding_review' ELSE 'active' END WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'credential_confirmed',jsonb_build_object('credential_ref',a.credential_ref,'credential_version',a.credential_version::text));RETURN integration.render_connection(a,true);
EXCEPTION WHEN unique_violation THEN PERFORM integration.connection_error('P2702');END $$;

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
  INSERT INTO banking.source_namespace(entity_id,book_id,source_account_id,profile,provider,environment,item_id,provider_account_id,overlap_review_required,created_by) VALUES(e,b,s.source_account_id,'jayburd-plaid-transactions/1','plaid','sandbox',i.item_id,r->>'account_id',NOT new_source,actor) RETURNING namespace_id INTO ns;
  INSERT INTO integration.sync_stream(entity_id,book_id,sync_item_id,source_account_id,namespace_id,account_id,worker_login,service_principal) VALUES(e,b,i.sync_item_id,s.source_account_id,ns,r->>'account_id',w.worker_login,w.service_principal) RETURNING stream_id INTO stream_value;
  result_value:=result_value||jsonb_build_array(jsonb_build_object('stream_id',stream_value,'source_account_id',s.source_account_id,'namespace_id',ns,'binding_revision',s.binding_revision::text,'overlap_review_required',NOT new_source));
 END LOOP;
 UPDATE integration.sync_item SET state='active',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
 UPDATE integration.connection_attempt SET state='active',bind_body=v,streams=result_value,item_generation=i.generation WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 result_value:=integration.render_connection(a,false);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'bind',k,jsonb_build_object('attempt_id',id,'body',v),result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_error('P2700');WHEN SQLSTATE 'P2601' THEN PERFORM integration.connection_error('P2701');WHEN unique_violation THEN PERFORM integration.connection_error('P2702');END $$;

CREATE OR REPLACE FUNCTION integration.bank_credential_authorize(actor uuid,e uuid,b uuid,r uuid,v bigint,item text,env text,write_value boolean) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);
 IF v IS NULL OR v<1 OR env IS DISTINCT FROM 'sandbox' OR item IS NULL THEN PERFORM integration.connection_error('P2700');END IF;
 SELECT * INTO a FROM integration.connection_attempt x WHERE x.entity_id=e AND x.book_id=b AND x.actor_id=actor
 AND ((x.credential_ref=r AND x.credential_version=v AND x.manifest->>'item_id'=item)
 OR (NOT write_value AND x.previous_credential_ref=r AND x.previous_credential_version=v AND x.expected_item_id=item))
 ORDER BY x.expires_at DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND OR a.claim_id IS NULL THEN PERFORM integration.connection_error('P2701');END IF;
 a:=integration.connection_lock(actor,e,b,a.attempt_id);
 IF a.sync_item_id IS NOT NULL THEN PERFORM integration.connection_item_pin(a);END IF;
 IF write_value AND (a.state NOT IN ('credential_pending','credential_recovery_required') OR a.manifest IS NULL) THEN PERFORM integration.connection_error('P2704');END IF;
END $$;

CREATE OR REPLACE FUNCTION integration.put_bank_credential(actor uuid,e uuid,b uuid,r uuid,v bigint,item text,env text,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE value jsonb;result_value jsonb;
BEGIN
 PERFORM integration.bank_credential_authorize(actor,e,b,r,v,item,env,true);
 value:=integration.connection_json(raw,32768);
 PERFORM ledger.keys(value,ARRAY['profile','key_version','wrapped_key','nonce','ciphertext']);
 IF value->>'profile' IS DISTINCT FROM 'jayburd-bank-aead-v1'
 OR jsonb_typeof(value->'key_version') IS DISTINCT FROM 'string'
 OR length(value->>'key_version') NOT BETWEEN 1 AND 512
 OR (value->>'key_version') !~ '^projects/([1-9][0-9]*|[a-z][a-z0-9-]{4,28}[a-z0-9])/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$'
 OR jsonb_typeof(value->'wrapped_key') IS DISTINCT FROM 'string'
 OR jsonb_typeof(value->'nonce') IS DISTINCT FROM 'string'
 OR jsonb_typeof(value->'ciphertext') IS DISTINCT FROM 'string'
 THEN PERFORM integration.connection_error('P2700');END IF;
 IF octet_length(decode(value->>'wrapped_key','base64')) NOT BETWEEN 1 AND 8192
 OR octet_length(decode(value->>'nonce','base64'))<>12
 OR octet_length(decode(value->>'ciphertext','base64')) NOT BETWEEN 17 AND 4112
 THEN PERFORM integration.connection_error('P2700');END IF;
 INSERT INTO integration.bank_credential(entity_id,book_id,credential_ref,credential_version,item_id,environment,envelope)
 VALUES(e,b,r,v,item,env,value) ON CONFLICT(entity_id,book_id,credential_ref,credential_version) DO NOTHING;
 SELECT envelope INTO result_value FROM integration.bank_credential WHERE entity_id=e AND book_id=b AND credential_ref=r AND credential_version=v AND item_id=item AND environment=env;
 IF result_value IS NULL THEN PERFORM integration.connection_error('P2702');END IF;
 RETURN jsonb_build_object('envelope',result_value);
EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');END $$;

CREATE OR REPLACE FUNCTION integration.put_link_handoff_before_owned_repair32(actor uuid,e uuid,b uuid,id uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;winner jsonb;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.mode<>'new' OR a.state<>'link_ready' OR a.claim_id IS NOT NULL OR a.link_token_sha256 IS NULL OR clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2704');END IF;
 v:=integration.connection_json(raw,32768);
 PERFORM ledger.keys(v,ARRAY['profile','key_version','wrapped_key','nonce','ciphertext']);
 IF v->>'profile' IS DISTINCT FROM 'jayburd-hosted-link-aead-v1'
 OR jsonb_typeof(v->'key_version') IS DISTINCT FROM 'string'
 OR length(v->>'key_version') NOT BETWEEN 1 AND 512
 OR (v->>'key_version') !~ '^projects/([1-9][0-9]*|[a-z][a-z0-9-]{4,28}[a-z0-9])/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$'
 OR jsonb_typeof(v->'wrapped_key') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v->'nonce') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v->'ciphertext') IS DISTINCT FROM 'string'
 OR octet_length(decode(v->>'wrapped_key','base64')) NOT BETWEEN 1 AND 8192
 OR octet_length(decode(v->>'nonce','base64'))<>12
 OR octet_length(decode(v->>'ciphertext','base64')) NOT BETWEEN 17 AND 4112
 THEN PERFORM integration.connection_error('P2700');END IF;
 INSERT INTO integration.link_handoff(entity_id,book_id,attempt_id,envelope) VALUES(e,b,id,v)
 ON CONFLICT(entity_id,book_id,attempt_id) DO NOTHING;
 SELECT envelope INTO winner FROM integration.link_handoff WHERE entity_id=e AND book_id=b AND attempt_id=id;
 RETURN jsonb_build_object('envelope',winner);
EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');END $$;
DROP FUNCTION integration.provider_key_matches(uuid,uuid,text,text,boolean);
DROP FUNCTION integration.provider_environment(uuid,uuid);
DROP FUNCTION integration.configure_provider_scope(uuid,uuid,uuid,text,text);
DROP TRIGGER preserve_provider_scope ON integration.provider_scope;
DROP FUNCTION integration.provider_scope_guard();
DROP FUNCTION integration.provider_api_login(name);
DROP TABLE integration.provider_scope;
ALTER TABLE integration.connection_attempt DROP COLUMN environment;
ALTER TABLE integration.sync_item DROP CONSTRAINT sync_item_environment_check;
ALTER TABLE integration.sync_item DROP CONSTRAINT sync_item_synthetic_check;
ALTER TABLE integration.sync_item ADD CONSTRAINT sync_item_environment_check CHECK(environment='sandbox');
ALTER TABLE integration.sync_item ADD CONSTRAINT sync_item_synthetic_check CHECK(synthetic);
ALTER TABLE integration.bank_credential DROP CONSTRAINT bank_credential_environment_check;
ALTER TABLE integration.bank_credential ADD CONSTRAINT bank_credential_environment_check CHECK(environment='sandbox');
ALTER TABLE banking.source_namespace DROP CONSTRAINT source_namespace_profile_check;
ALTER TABLE banking.source_namespace ADD CONSTRAINT source_namespace_profile_check CHECK(
 (profile='jayburd-synthetic-statement-csv/1' AND synthetic_dataset_id IS NOT NULL AND source_account_ref IS NOT NULL AND provider IS NULL AND environment IS NULL AND item_id IS NULL AND provider_account_id IS NULL)
 OR (profile='jayburd-plaid-transactions/1' AND synthetic_dataset_id IS NULL AND source_account_ref IS NULL AND provider IS NOT NULL AND environment IS NOT NULL AND provider='plaid' AND environment='sandbox' AND item_id IS NOT NULL AND provider_account_id IS NOT NULL));
COMMIT;
