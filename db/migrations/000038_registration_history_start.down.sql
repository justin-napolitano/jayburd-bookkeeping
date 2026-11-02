-- Never roll back a boundary already present in saved reviewed intent or immutable receipts.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN
 IF EXISTS(SELECT FROM ledger.interactive_command c CROSS JOIN LATERAL jsonb_array_elements(c.request_body->'bindings') z WHERE c.operation='register_accounts' AND z.value->'new_ledger_account'->>'active_from' IS NOT NULL AND z.value->'new_ledger_account'->>'active_from'<>z.value->>'binding_date')
 OR EXISTS(SELECT FROM integration.connection_receipt c CROSS JOIN LATERAL jsonb_array_elements(c.body->'body'->'bindings') z WHERE c.operation='bind' AND z.value->'new_ledger_account'->>'active_from' IS NOT NULL AND z.value->'new_ledger_account'->>'active_from'<>z.value->>'binding_date')
 THEN PERFORM integration.connection_error('P2702');END IF;
END $$;
CREATE OR REPLACE FUNCTION integration.prepare_registration(actor uuid,e uuid,b uuid,id uuid,k text,request uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;w integration.connection_worker;c ledger.interactive_command;v jsonb;selected_value jsonb;bindings_value jsonb:='[]';row_value jsonb;provider_row jsonb;body_value jsonb;policy_value ledger.book_policy;currency_value reference.currency;d date;expiry timestamptz;scope_revision bigint;limit_revision bigint;book_revision bigint;code_value text;label_value text;category_value text;side_value text;base_code integer;ordinal integer:=0;lid uuid;
BEGIN
 IF request IS NULL THEN PERFORM integration.connection_error('P2700');END IF;
 PERFORM ledger.command_principal(actor);a:=integration.connection_lock(actor,e,b,id);PERFORM integration.connection_key(k);
 v:=integration.connection_json(raw,16384);PERFORM ledger.keys(v,ARRAY['account_ids']);
 IF jsonb_typeof(v->'account_ids') IS DISTINCT FROM 'array' OR (CASE WHEN jsonb_typeof(v->'account_ids')='array' THEN jsonb_array_length(v->'account_ids') ELSE -1 END) NOT BETWEEN 1 AND 25 THEN PERFORM integration.connection_error('P2700');END IF;
 FOR row_value IN SELECT value FROM jsonb_array_elements(v->'account_ids') LOOP PERFORM integration.sync_text(row_value,256);END LOOP;
 IF (SELECT count(DISTINCT value) FROM jsonb_array_elements(v->'account_ids'))<>jsonb_array_length(v->'account_ids') THEN PERFORM integration.connection_error('P2700');END IF;
 SELECT jsonb_agg(value ORDER BY value) INTO selected_value FROM jsonb_array_elements(v->'account_ids');
 SELECT * INTO c FROM ledger.interactive_command WHERE principal_id=actor AND registration_key=k;
 IF FOUND THEN
  IF c.operation<>'register_accounts' OR c.entity_id<>e OR c.book_id<>b OR c.request_body->>'attempt_id'<>id::text OR c.request_body->'selection'<>selected_value THEN PERFORM integration.connection_error('P2702');END IF;
  PERFORM ledger.command_authorize(actor,c);RETURN ledger.render_command(c);
 END IF;
 IF EXISTS(SELECT FROM ledger.interactive_command WHERE principal_id=actor AND acknowledged_at IS NULL) THEN PERFORM ledger.fail('P2112');END IF;
 IF a.mode<>'new' OR a.manifest IS NULL OR a.state NOT IN ('binding_review','active') THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.state='binding_review' AND clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2703');END IF;
 i:=integration.connection_item_pin(a);w:=integration.connection_authorize(actor,e,b);
 IF (a.state='binding_review' AND i.state<>'paused' OR a.state='active' AND i.state<>'active') OR clock_timestamp()>=i.consent_expires_at THEN PERFORM integration.connection_error('P2703');END IF;
 SELECT * INTO policy_value FROM ledger.book_policy WHERE entity_id=e AND book_id=b AND effective_from<=(transaction_timestamp() AT TIME ZONE timezone)::date AND (effective_to IS NULL OR effective_to>(transaction_timestamp() AT TIME ZONE timezone)::date);
 IF NOT FOUND OR (SELECT count(*) FROM ledger.book_policy WHERE entity_id=e AND book_id=b AND effective_from<=(transaction_timestamp() AT TIME ZONE timezone)::date AND (effective_to IS NULL OR effective_to>(transaction_timestamp() AT TIME ZONE timezone)::date))<>1 THEN PERFORM integration.connection_error('P2704');END IF;
 d:=(transaction_timestamp() AT TIME ZONE policy_value.timezone)::date;
 FOR row_value IN SELECT value FROM jsonb_array_elements(selected_value) LOOP
  SELECT value INTO provider_row FROM jsonb_array_elements(a.manifest->'accounts') WHERE value->'account_id'=row_value;
  IF NOT FOUND OR provider_row->>'iso_currency_code' IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
  IF EXISTS(SELECT FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id AND account_id=provider_row->>'account_id') THEN PERFORM integration.connection_error('P2702');END IF;
  category_value:=integration.registration_profile(provider_row)->>'category';
  side_value:=integration.registration_profile(provider_row)->>'side';
  base_code:=(integration.registration_profile(provider_row)->>'base_code')::integer;
  label_value:=integration.registration_profile(provider_row)->>'label';
  IF category_value IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
  SELECT * INTO currency_value FROM reference.currency WHERE alphabetic_code=provider_row->>'iso_currency_code' AND minor_unit IS NOT NULL AND valid_from<=d AND (valid_to IS NULL OR valid_to>d);
  IF NOT FOUND OR (SELECT count(*) FROM reference.currency WHERE alphabetic_code=provider_row->>'iso_currency_code' AND minor_unit IS NOT NULL AND valid_from<=d AND (valid_to IS NULL OR valid_to>d))<>1 THEN PERFORM integration.connection_error('P2704');END IF;
  SELECT candidates.code_number::text INTO code_value FROM generate_series(base_code,base_code+9999) AS candidates(code_number) WHERE NOT EXISTS(SELECT FROM ledger.account WHERE entity_id=e AND book_id=b AND code=candidates.code_number::text) AND NOT EXISTS(SELECT FROM jsonb_array_elements(bindings_value) z WHERE z->'new_ledger_account'->>'code'=candidates.code_number::text) ORDER BY candidates.code_number LIMIT 1;
  IF code_value IS NULL THEN PERFORM integration.connection_error('P2705');END IF;
  ordinal:=ordinal+1;lid:=gen_random_uuid();
  bindings_value:=bindings_value||jsonb_build_array(jsonb_build_object('account_id',provider_row->>'account_id','ledger_account_id',lid,'currency_version_id',currency_value.currency_version_id,'source_account_id',NULL,'expected_binding_revision',NULL,'binding_date',d,'binding_evidence_ref',gen_random_uuid(),'new_ledger_account',jsonb_build_object('account_id',lid,'code',code_value,'name',label_value||' '||ordinal::text,'category',category_value,'normal_side',side_value,'monetary',true,'active_from',d)));
 END LOOP;
 PERFORM integration.registration_capacity(e,b,i,bindings_value);
 SELECT revision INTO scope_revision FROM integration.provider_scope WHERE entity_id=e AND book_id=b;
 SELECT revision INTO limit_revision FROM integration.registration_limit WHERE entity_id=e AND book_id=b;
 SELECT revision INTO book_revision FROM ledger.book WHERE entity_id=e AND book_id=b;
 expiry:=least(clock_timestamp()+interval '10 minutes',i.consent_expires_at,CASE WHEN a.state='binding_review' THEN a.expires_at ELSE 'infinity'::timestamptz END);
 body_value:=jsonb_build_object('selection',selected_value,'attempt_id',id,'sync_item_id',i.sync_item_id,'item_generation',i.generation::text,'worker_revision',w.revision::text,'scope_revision',coalesce(scope_revision,0)::text,'limit_revision',coalesce(limit_revision,0)::text,'manifest_digest',encode(sha256(convert_to(a.manifest::text,'UTF8')),'hex'),'book_revision',book_revision::text,'policy_id',policy_value.policy_id,'binding_date',d,'expires_at',expiry,'initial',a.state='binding_review','bindings',bindings_value);
 body_value:=body_value||jsonb_build_object('digest',encode(sha256(convert_to(body_value::text,'UTF8')),'hex'));
 INSERT INTO ledger.interactive_command(command_id,principal_id,registration_key,operation,entity_id,book_id,request_body) VALUES(gen_random_uuid(),actor,k,'register_accounts',e,b,body_value) RETURNING * INTO c;
 INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(c.command_id,actor,'register',request);
 RETURN ledger.render_command(c);
EXCEPTION WHEN unique_violation THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_command_registration';
 WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM integration.connection_error('P2700');END $$;

CREATE OR REPLACE FUNCTION integration.apply_registration_bindings(actor uuid,e uuid,b uuid,i integration.sync_item,manifest_value jsonb,bindings_value jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;r jsonb;provider_row jsonb;new_account jsonb;s banking.source_account;la ledger.account;c reference.currency;ns uuid;stream_value uuid;d date;cid uuid;lid uuid;sid uuid;new_source boolean;code_value text;name_value text;category_value text;side_value text;result_value jsonb:='[]';p uuid;cap text;
BEGIN
 w:=integration.connection_authorize(actor,e,b);
 PERFORM integration.registration_capacity(e,b,i,bindings_value);
 IF EXISTS(SELECT FROM jsonb_array_elements(bindings_value) z GROUP BY value->>'account_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(bindings_value) z GROUP BY value->>'ledger_account_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(bindings_value) z WHERE value->'source_account_id'<>'null'::jsonb GROUP BY value->>'source_account_id' HAVING count(*)>1) THEN PERFORM integration.connection_error('P2700');END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(bindings_value) ORDER BY value->>'ledger_account_id' LOOP
  IF EXISTS(SELECT FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id AND account_id=r->>'account_id') THEN PERFORM integration.connection_error('P2702');END IF;
  IF jsonb_typeof(r) IS DISTINCT FROM 'object' THEN PERFORM integration.connection_error('P2700');END IF;
  IF NOT(r ?& ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref']) OR EXISTS(SELECT FROM jsonb_object_keys(r) z WHERE z<>ALL(ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref','new_ledger_account'])) THEN PERFORM integration.connection_error('P2700');END IF;
  PERFORM integration.sync_text(r->'account_id',256);cid:=ledger.uuid_value(r->'currency_version_id');lid:=ledger.uuid_value(r->'ledger_account_id');d:=ledger.date_value(ledger.text_value(r->'binding_date'));IF ledger.uuid_value(r->'binding_evidence_ref')='00000000-0000-0000-0000-000000000000' THEN PERFORM integration.connection_error('P2700');END IF;
  SELECT value INTO provider_row FROM jsonb_array_elements(manifest_value->'accounts') WHERE value->>'account_id'=r->>'account_id';
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
 RETURN result_value;
END $$;
DROP FUNCTION integration.registration_account_start(uuid,uuid,date);
COMMIT;
