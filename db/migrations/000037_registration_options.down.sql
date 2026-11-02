-- Restore exact36 preparation before removing read-only option helpers.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
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

DROP FUNCTION integration.registration_options(uuid,uuid,uuid,uuid);
DROP FUNCTION integration.registration_profile(jsonb);
COMMIT;
