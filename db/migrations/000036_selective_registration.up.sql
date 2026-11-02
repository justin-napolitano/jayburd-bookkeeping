-- #392: docs/contracts/registration-sql.md; default-one Production authority stays unchanged.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE integration.registration_limit (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,maximum smallint NOT NULL CHECK(maximum BETWEEN 1 AND 25),revision bigint NOT NULL CHECK(revision>0),
 PRIMARY KEY(entity_id,book_id),FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
REVOKE ALL ON integration.registration_limit FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
-- Limit history cannot be erased or revised without advancing its authorization revision.
CREATE FUNCTION integration.registration_limit_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.connection_error('P2702');END IF;
 IF ROW(NEW.entity_id,NEW.book_id) IS DISTINCT FROM ROW(OLD.entity_id,OLD.book_id) OR NEW.revision::numeric<>OLD.revision::numeric+1 THEN PERFORM integration.connection_error('P2702');END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_registration_limit BEFORE UPDATE OR DELETE ON integration.registration_limit FOR EACH ROW EXECUTE FUNCTION integration.registration_limit_guard();
-- Only an explicitly authorized operator changes the maximum; a saved receipt cannot re-enable an old limit.
CREATE FUNCTION integration.configure_registration_limit(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;rec integration.connection_receipt;maximum_value integer;lim integration.registration_limit;result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='connection_rejected';END IF;
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.connection_error('P2700');END IF;
 PERFORM integration.connection_key(k);v:=integration.connection_json(raw,1024);PERFORM ledger.keys(v,ARRAY['max_sources']);
 IF jsonb_typeof(v->'max_sources') IS DISTINCT FROM 'string' OR v->>'max_sources' !~ '^([1-9]|1[0-9]|2[0-5])$' THEN PERFORM integration.connection_error('P2700');END IF;
 maximum_value:=(v->>'max_sources')::integer;
 PERFORM banking.source_operator_authorize(actor,actor,e,b,true);
 PERFORM 1 FROM integration.provider_scope WHERE entity_id=e AND book_id=b AND environment='production' FOR SHARE;
 IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='registration_limit' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>v THEN PERFORM integration.connection_error('P2702');END IF;RETURN rec.result;END IF;
 IF (SELECT count(*) FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND state<>'disconnected')>maximum_value THEN PERFORM integration.connection_error('P2705');END IF;
 INSERT INTO integration.registration_limit VALUES(e,b,maximum_value,1)
 ON CONFLICT(entity_id,book_id) DO UPDATE SET maximum=EXCLUDED.maximum,revision=integration.registration_limit.revision+1 RETURNING * INTO lim;
 result_value:=jsonb_build_object('max_sources',lim.maximum::text,'limit_revision',lim.revision::text);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'registration_limit',k,v,result_value);
 RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM integration.connection_error('P2700');END $$;
-- A book lock serializes capacity with binding and scope revocation; paused sources still consume capacity.
CREATE FUNCTION integration.registration_capacity(e uuid,b uuid,i integration.sync_item,bindings_value jsonb) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE maximum_value integer;
BEGIN
 IF jsonb_typeof(bindings_value) IS DISTINCT FROM 'array' OR jsonb_array_length(bindings_value) NOT BETWEEN 1 AND 25 THEN PERFORM integration.connection_error('P2700');END IF;
 IF i.environment='production' THEN
  SELECT maximum INTO maximum_value FROM integration.registration_limit WHERE entity_id=e AND book_id=b FOR SHARE;
  maximum_value:=coalesce(maximum_value,1);
  IF (SELECT count(*) FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND state<>'disconnected')+jsonb_array_length(bindings_value)>maximum_value
  OR EXISTS(SELECT FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND state<>'disconnected' AND sync_item_id<>i.sync_item_id)
  THEN PERFORM integration.connection_error('P2705');END IF;
 END IF;
END $$;
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_operation_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_operation_check CHECK(operation IN ('setup','post','import','source_post','register_accounts'));
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_check CHECK((operation='setup' AND entity_id IS NULL AND book_id IS NULL) OR (operation IN ('post','import','source_post','register_accounts') AND entity_id IS NOT NULL AND book_id IS NOT NULL));

-- Shared strict binding mutation: both legacy first-bind and reviewed append use the same rules.
CREATE FUNCTION integration.apply_registration_bindings(actor uuid,e uuid,b uuid,i integration.sync_item,manifest_value jsonb,bindings_value jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
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
CREATE OR REPLACE FUNCTION integration.bind_connection(actor uuid,e uuid,b uuid,id uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;v jsonb;result_value jsonb;rec integration.connection_receipt;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);PERFORM integration.connection_key(k);v:=integration.connection_json(raw,32768);PERFORM ledger.keys(v,ARRAY['consent_grants','bindings']);
 IF v->'consent_grants' IS DISTINCT FROM 'true'::jsonb OR jsonb_typeof(v->'bindings')<>'array' OR (CASE WHEN jsonb_typeof(v->'bindings')='array' THEN jsonb_array_length(v->'bindings') ELSE -1 END) NOT BETWEEN 1 AND 25 THEN PERFORM integration.connection_error('P2700');END IF;
 IF a.mode<>'new' THEN PERFORM integration.connection_error('P2704');END IF;
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='bind' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>jsonb_build_object('attempt_id',id,'body',v) THEN PERFORM integration.connection_error('P2702');END IF;RETURN rec.result;END IF;
 IF a.state='active' THEN PERFORM integration.connection_error('P2702');END IF;
 IF a.state<>'binding_review' OR clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2703');END IF;

 i:=integration.connection_item_pin(a);
 result_value:=integration.apply_registration_bindings(actor,e,b,i,a.manifest,v->'bindings');
 UPDATE integration.sync_item SET state='active',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
 UPDATE integration.connection_attempt SET state='active',bind_body=v,streams=result_value,item_generation=i.generation WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 result_value:=integration.render_connection(a,false);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'bind',k,jsonb_build_object('attempt_id',id,'body',v),result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_error('P2700');WHEN SQLSTATE 'P2601' THEN PERFORM integration.connection_error('P2701');WHEN unique_violation THEN PERFORM integration.connection_error('P2702');END $$;
-- Only this preparer may create register_accounts commands; generic command registration remains closed.
CREATE FUNCTION integration.prepare_registration(actor uuid,e uuid,b uuid,id uuid,k text,request uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
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
  category_value:=NULL;
  IF provider_row->>'type'='depository' AND provider_row->>'subtype' IN ('checking','savings','money market','cash management') THEN category_value:='asset';side_value:='debit';base_code:=1000;label_value:=initcap(provider_row->>'subtype');
  ELSIF provider_row->>'type'='credit' AND provider_row->>'subtype'='credit card' THEN category_value:='liability';side_value:='credit';base_code:=2100;label_value:='Credit card';
  ELSIF provider_row->>'type'='loan' AND provider_row->>'subtype' IN ('student','mortgage','auto','consumer','line of credit') THEN category_value:='liability';side_value:='credit';base_code:=2200;label_value:='Loan';END IF;
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

-- A committed command is recovered by the existing transition before this new-work executor runs.
CREATE FUNCTION integration.execute_registration(actor uuid,e uuid,b uuid,command_value uuid,body_value jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;w integration.connection_worker;scope_revision bigint;limit_revision bigint;book_revision bigint;policy_value ledger.book_policy;result_value jsonb;
BEGIN
 a:=integration.connection_lock(actor,e,b,(body_value->>'attempt_id')::uuid);i:=integration.connection_item_pin(a);w:=integration.connection_authorize(actor,e,b);
 SELECT revision INTO scope_revision FROM integration.provider_scope WHERE entity_id=e AND book_id=b;
 SELECT revision INTO limit_revision FROM integration.registration_limit WHERE entity_id=e AND book_id=b;
 SELECT revision INTO book_revision FROM ledger.book WHERE entity_id=e AND book_id=b;
 SELECT * INTO policy_value FROM ledger.book_policy WHERE entity_id=e AND book_id=b AND policy_id=(body_value->>'policy_id')::uuid;
 IF NOT FOUND OR clock_timestamp()>=(body_value->>'expires_at')::timestamptz OR clock_timestamp()>=i.consent_expires_at
 OR ((body_value->>'initial')::boolean AND i.state<>'paused' OR NOT (body_value->>'initial')::boolean AND i.state<>'active')
 OR (body_value->>'digest') IS DISTINCT FROM encode(sha256(convert_to((body_value-'digest')::text,'UTF8')),'hex')
 OR i.sync_item_id::text<>body_value->>'sync_item_id' OR i.generation::text<>body_value->>'item_generation'
 OR w.revision::text<>body_value->>'worker_revision' OR coalesce(scope_revision,0)::text<>body_value->>'scope_revision'
 OR coalesce(limit_revision,0)::text<>body_value->>'limit_revision' OR book_revision::text<>body_value->>'book_revision'
 OR encode(sha256(convert_to(a.manifest::text,'UTF8')),'hex')<>body_value->>'manifest_digest'
 OR (transaction_timestamp() AT TIME ZONE policy_value.timezone)::date<>(body_value->>'binding_date')::date
 OR policy_value.effective_from>(body_value->>'binding_date')::date OR policy_value.effective_to IS NOT NULL AND policy_value.effective_to<=(body_value->>'binding_date')::date
 THEN PERFORM integration.connection_error('P2703');END IF;
 IF (body_value->>'initial')::boolean THEN
  IF a.state<>'binding_review' THEN PERFORM integration.connection_error('P2703');END IF;
  result_value:=integration.bind_connection(actor,e,b,a.attempt_id,'registration:'||command_value::text,jsonb_build_object('consent_grants',true,'bindings',body_value->'bindings')::text);
 ELSE
  IF a.state<>'active' THEN PERFORM integration.connection_error('P2703');END IF;
  result_value:=jsonb_build_object('entity_id',e,'book_id',b,'sync_item_id',i.sync_item_id,'streams',integration.apply_registration_bindings(actor,e,b,i,a.manifest,body_value->'bindings'));
 END IF;
 RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_error('P2700');WHEN unique_violation THEN PERFORM integration.connection_error('P2702');END $$;

CREATE OR REPLACE FUNCTION ledger.command_authorize(actor uuid,c ledger.interactive_command) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE e uuid; b uuid; x banking.import_batch;
BEGIN
 IF c.principal_id IS DISTINCT FROM actor THEN PERFORM ledger.fail('P2101'); END IF;
 IF c.operation='register_accounts' THEN
  BEGIN
   PERFORM integration.connection_lock(actor,c.entity_id,c.book_id,(c.request_body->>'attempt_id')::uuid);
  EXCEPTION WHEN SQLSTATE 'P2701' OR SQLSTATE 'P2703' THEN PERFORM ledger.fail('P2102');END;
  RETURN;
 END IF;
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
   IF c.operation='register_accounts' THEN value:=integration.execute_registration(actor,c.entity_id,c.book_id,c.command_id,c.request_body);
   ELSIF c.operation='source_post' THEN value:=banking.execute_source_post(actor,c.entity_id,c.book_id,c.command_id,request,c.request_body);
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
REVOKE ALL ON FUNCTION integration.registration_limit_guard(),integration.configure_registration_limit(uuid,uuid,uuid,text,text),integration.registration_capacity(uuid,uuid,integration.sync_item,jsonb),integration.apply_registration_bindings(uuid,uuid,uuid,integration.sync_item,jsonb,jsonb),integration.prepare_registration(uuid,uuid,uuid,uuid,text,uuid,text),integration.execute_registration(uuid,uuid,uuid,uuid,jsonb) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.prepare_registration(uuid,uuid,uuid,uuid,text,uuid,text) TO bookkeeping_api;
COMMIT;
