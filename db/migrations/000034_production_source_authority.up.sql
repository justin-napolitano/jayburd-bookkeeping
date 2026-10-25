-- #311: explicit Production source authority; docs/contracts/production-source-sql.md.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;

CREATE TABLE integration.provider_scope (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,environment text NOT NULL CHECK(environment IN ('sandbox','production')),
 api_login name NOT NULL,worker_login name NOT NULL,bank_key text,link_key text,
 enabled boolean NOT NULL,revision bigint NOT NULL CHECK(revision>0),
 PRIMARY KEY(entity_id,book_id),FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,
 CHECK(api_login<>worker_login),
 CHECK(environment='sandbox' AND bank_key IS NULL AND link_key IS NULL
 OR environment='production' AND bank_key IS NOT NULL AND link_key IS NOT NULL AND length(bank_key) BETWEEN 1 AND 512 AND length(link_key) BETWEEN 1 AND 512)
);
REVOKE ALL ON integration.provider_scope FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
-- Login identity is independent of a caller-selected actor or SET ROLE.
CREATE FUNCTION integration.provider_api_login(login_value name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolcanlogin AND NOT(rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication))
 AND pg_has_role(login_value,'bookkeeping_api','USAGE')
 AND NOT EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolname IN ('bookkeeping_owner','bookkeeping_sync','bookkeeping_webhook','bookkeeping_identity_operator','bookkeeping_migrator','bookkeeping_observer','bookkeeping_publisher','bookkeeping_verifier','cloudsqlsuperuser')) AND pg_has_role(login_value,oid,'MEMBER'))
 AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','deployment') AND c.relkind IN ('r','p','v','m','S') AND has_table_privilege(login_value,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
$$;
-- Scope identity cannot be moved to another environment, login or key after creation.
CREATE FUNCTION integration.provider_scope_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.connection_error('P2702');END IF;
 IF (to_jsonb(NEW)-ARRAY['enabled','revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['enabled','revision'])
 OR NEW.revision::numeric<>OLD.revision::numeric+1 THEN PERFORM integration.connection_error('P2702');END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_provider_scope BEFORE UPDATE OR DELETE ON integration.provider_scope FOR EACH ROW EXECUTE FUNCTION integration.provider_scope_guard();

-- Operator-only initialization/revocation, never exposed as a user API.
CREATE FUNCTION integration.configure_provider_scope(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;p integration.provider_scope;r integration.connection_receipt;result_value jsonb;api name;worker name;env text;bank text;link text;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='connection_rejected';END IF;
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.connection_error('P2700');END IF;
 PERFORM integration.connection_key(k);
 v:=integration.connection_json(raw,4096);
 -- This closed operator input permits JSON null only for Sandbox key references; ledger.keys forbids every null.
 IF NOT(v ?& ARRAY['environment','api_login','worker_login','bank_key','link_key','enabled'])
 OR EXISTS(SELECT FROM jsonb_object_keys(v) z WHERE z<>ALL(ARRAY['environment','api_login','worker_login','bank_key','link_key','enabled']))
 THEN PERFORM integration.connection_error('P2700');END IF;
 env:=ledger.text_value(v->'environment');api:=integration.sync_text(v->'api_login',63)::name;worker:=integration.sync_text(v->'worker_login',63)::name;
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname=api) OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname=worker)
 THEN PERFORM integration.connection_error('P2701');END IF;
 IF env NOT IN ('sandbox','production') OR jsonb_typeof(v->'enabled') IS DISTINCT FROM 'boolean'
 OR NOT integration.provider_api_login(api) OR NOT integration.sync_login(worker)
 THEN PERFORM integration.connection_error('P2701');END IF;
 IF env='sandbox' THEN
  IF v->'bank_key' IS DISTINCT FROM 'null'::jsonb OR v->'link_key' IS DISTINCT FROM 'null'::jsonb THEN PERFORM integration.connection_error('P2700');END IF;
 ELSE
  bank:=ledger.text_value(v->'bank_key');link:=ledger.text_value(v->'link_key');
  IF length(bank)>512 OR length(link)>512
  OR bank !~ '^projects/[a-z][a-z0-9-]{4,28}[a-z0-9]/locations/us-central1/keyRings/[A-Za-z0-9_-]{1,63}/cryptoKeys/[A-Za-z0-9_-]{1,63}$'
  OR link !~ '^projects/[a-z][a-z0-9-]{4,28}[a-z0-9]/locations/us-central1/keyRings/[A-Za-z0-9_-]{1,63}/cryptoKeys/[A-Za-z0-9_-]{1,63}$'
  OR bank LIKE '%/keyRings/jayburd-dev/%' OR link LIKE '%/keyRings/jayburd-dev/%'
  THEN PERFORM integration.connection_error('P2700');END IF;
 END IF;
 PERFORM banking.source_operator_authorize(actor,actor,e,b,true);
 -- Serialize first configuration across books before checking exclusive runtime identities.
 PERFORM pg_advisory_xact_lock(311,34);
 SELECT * INTO p FROM integration.provider_scope WHERE entity_id=e AND book_id=b FOR UPDATE;
 IF FOUND THEN
  IF ROW(p.environment,p.api_login,p.worker_login,p.bank_key,p.link_key) IS DISTINCT FROM ROW(env,api,worker,bank,link)
  THEN PERFORM integration.connection_error('P2702');END IF;
 ELSE
  IF EXISTS(SELECT FROM integration.connection_attempt WHERE entity_id=e AND book_id=b)
  OR EXISTS(SELECT FROM integration.sync_item WHERE entity_id=e AND book_id=b)
  OR EXISTS(SELECT FROM banking.source_namespace WHERE entity_id=e AND book_id=b)
  OR EXISTS(SELECT FROM ledger.journal_entry WHERE entity_id=e AND book_id=b)
  OR EXISTS(SELECT FROM integration.provider_scope WHERE (api_login=api OR worker_login=worker) AND environment<>env)
  THEN PERFORM integration.connection_error('P2702');END IF;
 END IF;
 SELECT * INTO r FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='provider_scope' AND idempotency_key=k;
 IF FOUND THEN IF r.body<>v THEN PERFORM integration.connection_error('P2702');END IF;RETURN r.result;END IF;
 INSERT INTO integration.provider_scope VALUES(e,b,env,api,worker,bank,link,(v->>'enabled')::boolean,1)
 ON CONFLICT(entity_id,book_id) DO UPDATE SET enabled=EXCLUDED.enabled,revision=integration.provider_scope.revision+1 RETURNING * INTO p;
 result_value:=jsonb_build_object('environment',p.environment,'data_origin',CASE WHEN p.environment='sandbox' THEN 'test' ELSE 'provider' END,'scope_revision',p.revision::text,'enabled',p.enabled);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'provider_scope',k,v,result_value);
 RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_error('P2700');END $$;

-- Called only after the established principal/book locks; a configured scope has no fallback.
CREATE FUNCTION integration.provider_environment(e uuid,b uuid) RETURNS text LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE p integration.provider_scope;
BEGIN
 SELECT * INTO p FROM integration.provider_scope WHERE entity_id=e AND book_id=b FOR SHARE;
 IF NOT FOUND THEN
  IF EXISTS(SELECT FROM integration.provider_scope WHERE environment='production' AND (api_login=session_user OR worker_login=session_user))
  THEN PERFORM integration.connection_error('P2701');END IF;
  RETURN 'sandbox';
 END IF;
 IF NOT p.enabled OR NOT (session_user=p.api_login AND integration.provider_api_login(session_user)
 OR session_user=p.worker_login AND integration.sync_login(session_user))
 THEN PERFORM integration.connection_error('P2701');END IF;
 RETURN p.environment;
END $$;

ALTER FUNCTION integration.connection_authorize(uuid,uuid,uuid) RENAME TO connection_authorize_before_production34;
CREATE FUNCTION integration.connection_authorize(actor uuid,e uuid,b uuid) RETURNS integration.connection_worker LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;p integration.provider_scope;env text;
BEGIN
 w:=integration.connection_authorize_before_production34(actor,e,b);
 env:=integration.provider_environment(e,b);
 SELECT * INTO p FROM integration.provider_scope WHERE entity_id=e AND book_id=b;
 IF FOUND AND (session_user<>p.api_login OR w.worker_login<>p.worker_login)
 THEN PERFORM integration.connection_error('P2701');END IF;
 RETURN w;
END $$;

-- A finite Production pilot cannot mint another Item while its first Item still exists.
ALTER FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) RENAME TO begin_connection_before_production34;
REVOKE ALL ON FUNCTION integration.begin_connection_before_production34(uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api;
CREATE FUNCTION integration.begin_connection(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);
 v:=integration.connection_json(raw,16384);
 IF integration.provider_environment(e,b)='production' AND v->>'mode'='new'
 AND NOT EXISTS(SELECT FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND actor_id=actor AND idempotency_key=k)
 AND EXISTS(SELECT FROM integration.sync_item WHERE entity_id=e AND book_id=b AND environment='production' AND state<>'disconnected')
 THEN PERFORM integration.connection_error('P2705');END IF;
 RETURN integration.begin_connection_before_production34(actor,e,b,k,raw);
END $$;
REVOKE ALL ON FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) TO bookkeeping_api;

-- New creation derives immutable routing from current authority; existing rows retain Sandbox.
ALTER TABLE integration.connection_attempt ADD COLUMN environment text NOT NULL DEFAULT 'sandbox' CHECK(environment IN ('sandbox','production'));
CREATE FUNCTION integration.connection_environment_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='INSERT' THEN NEW.environment:=integration.provider_environment(NEW.entity_id,NEW.book_id);
 ELSIF NEW.environment IS DISTINCT FROM OLD.environment THEN PERFORM integration.connection_error('P2702');END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_connection_environment BEFORE INSERT OR UPDATE ON integration.connection_attempt FOR EACH ROW EXECUTE FUNCTION integration.connection_environment_guard();

ALTER FUNCTION integration.connection_lock(uuid,uuid,uuid,uuid) RENAME TO connection_lock_before_production34;
CREATE FUNCTION integration.connection_lock(actor uuid,e uuid,b uuid,id uuid) RETURNS integration.connection_attempt LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 a:=integration.connection_lock_before_production34(actor,e,b,id);
 IF a.environment IS DISTINCT FROM integration.provider_environment(e,b) THEN PERFORM integration.connection_error('P2701');END IF;
 RETURN a;
END $$;
ALTER FUNCTION integration.connection_item_pin(integration.connection_attempt) RENAME TO connection_item_pin_before_production34;
CREATE FUNCTION integration.connection_item_pin(a integration.connection_attempt) RETURNS integration.sync_item LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE i integration.sync_item;
BEGIN
 i:=integration.connection_item_pin_before_production34(a);
 IF i.environment IS DISTINCT FROM a.environment THEN PERFORM integration.connection_error('P2701');END IF;
 RETURN i;
END $$;
ALTER FUNCTION integration.render_connection(integration.connection_attempt,boolean) RENAME TO render_connection_before_production34;
CREATE FUNCTION integration.render_connection(a integration.connection_attempt,private_value boolean) RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;
BEGIN
 v:=integration.render_connection_before_production34(a,private_value);
 IF private_value THEN v:=v||jsonb_build_object('environment',a.environment,'data_origin',CASE WHEN a.environment='sandbox' THEN 'test' ELSE 'provider' END);END IF;
 RETURN v;
END $$;

ALTER TABLE integration.sync_item DROP CONSTRAINT sync_item_environment_check;
ALTER TABLE integration.sync_item DROP CONSTRAINT sync_item_synthetic_check;
ALTER TABLE integration.sync_item ADD CONSTRAINT sync_item_environment_check CHECK(environment IN ('sandbox','production'));
ALTER TABLE integration.sync_item ADD CONSTRAINT sync_item_synthetic_check CHECK(environment='sandbox' AND synthetic OR environment='production' AND NOT synthetic);
ALTER TABLE integration.bank_credential DROP CONSTRAINT bank_credential_environment_check;
ALTER TABLE integration.bank_credential ADD CONSTRAINT bank_credential_environment_check CHECK(environment IN ('sandbox','production'));
ALTER TABLE banking.source_namespace DROP CONSTRAINT source_namespace_profile_check;
ALTER TABLE banking.source_namespace ADD CONSTRAINT source_namespace_profile_check CHECK(
 (profile='jayburd-synthetic-statement-csv/1' AND synthetic_dataset_id IS NOT NULL AND source_account_ref IS NOT NULL AND provider IS NULL AND environment IS NULL AND item_id IS NULL AND provider_account_id IS NULL)
 OR (profile='jayburd-plaid-transactions/1' AND synthetic_dataset_id IS NULL AND source_account_ref IS NULL AND provider IS NOT NULL AND environment IS NOT NULL AND provider='plaid' AND environment IN ('sandbox','production') AND item_id IS NOT NULL AND provider_account_id IS NOT NULL));
-- Inserts cannot mix a scoped book's origin. Existing immutable guards protect later updates.
CREATE FUNCTION integration.provider_origin_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE env text;
BEGIN
 SELECT environment INTO env FROM integration.provider_scope WHERE entity_id=NEW.entity_id AND book_id=NEW.book_id;
 env:=coalesce(env,'sandbox');
 IF TG_TABLE_NAME='source_namespace' AND NEW.environment IS NULL THEN
  IF env='production' THEN PERFORM integration.connection_error('P2701');END IF;
 ELSIF NEW.environment IS DISTINCT FROM env THEN PERFORM integration.connection_error('P2701');END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_provider_origin BEFORE INSERT ON integration.sync_item FOR EACH ROW EXECUTE FUNCTION integration.provider_origin_guard();
CREATE TRIGGER preserve_provider_origin BEFORE INSERT ON integration.bank_credential FOR EACH ROW EXECUTE FUNCTION integration.provider_origin_guard();
CREATE TRIGGER preserve_provider_origin BEFORE INSERT ON banking.source_namespace FOR EACH ROW EXECUTE FUNCTION integration.provider_origin_guard();

ALTER FUNCTION integration.sync_authorize(uuid,uuid,uuid) RENAME TO sync_authorize_before_production34;
CREATE FUNCTION integration.sync_authorize(e uuid,b uuid,id uuid) RETURNS integration.sync_stream LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;env text;item_env text;
BEGIN
 t:=integration.sync_authorize_before_production34(e,b,id);
 env:=integration.provider_environment(e,b);
 SELECT environment INTO item_env FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 IF env IS DISTINCT FROM item_env OR NOT EXISTS(SELECT FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=t.namespace_id AND environment=env)
 THEN PERFORM integration.sync_error('P2601');END IF;
 RETURN t;
EXCEPTION WHEN SQLSTATE 'P2701' THEN PERFORM integration.sync_error('P2601');END $$;

-- #313 preflight returns only approved public routing/key metadata under current connection authority.
CREATE FUNCTION integration.connection_provider_context(actor uuid,e uuid,b uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE env text;p integration.provider_scope;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);
 env:=integration.provider_environment(e,b);
 SELECT * INTO p FROM integration.provider_scope WHERE entity_id=e AND book_id=b;
 RETURN jsonb_build_object('environment',env,'data_origin',CASE WHEN env='sandbox' THEN 'test' ELSE 'provider' END,'scope_revision',coalesce(p.revision,0)::text,'bank_key',p.bank_key,'link_key',p.link_key);
END $$;
-- Production keys must exactly match operator metadata; Sandbox retains the installed grammar.
CREATE FUNCTION integration.provider_key_matches(e uuid,b uuid,env text,key_version text,link_value boolean) RETURNS boolean LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE key_value text;
BEGIN
 IF env='sandbox' THEN RETURN key_version ~ '^projects/([1-9][0-9]*|[a-z][a-z0-9-]{4,28}[a-z0-9])/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$';END IF;
 SELECT CASE WHEN link_value THEN link_key ELSE bank_key END INTO key_value FROM integration.provider_scope WHERE entity_id=e AND book_id=b AND environment='production' AND enabled;
 RETURN coalesce(env='production' AND key_value IS NOT NULL AND key_version ~ '/cryptoKeyVersions/[1-9][0-9]*$' AND regexp_replace(key_version,'/cryptoKeyVersions/[1-9][0-9]*$','')=key_value,false);
END $$;

-- Explicit replacements of reviewed installed implementations; no dynamic SQL patching.
CREATE OR REPLACE FUNCTION integration.confirm_connection_credential_before_owned_repair32(actor uuid,e uuid,b uuid,id uuid,claim_value uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;s uuid;w integration.connection_worker;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF claim_value IS NULL OR a.claim_id IS DISTINCT FROM claim_value OR a.mode='remove' THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.state IN ('active','binding_review') THEN RETURN integration.render_connection(a,true);END IF;
 IF a.state NOT IN ('credential_pending','credential_recovery_required') OR a.manifest IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 IF (a.manifest->>'consent_expires_at')::timestamptz<=clock_timestamp() THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.mode='new' THEN
  INSERT INTO integration.sync_item(entity_id,book_id,provider,environment,item_id,credential_ref,credential_version,consent_principal,consent_evidence_ref,consent_expires_at,state,synthetic) VALUES(e,b,'plaid',a.environment,a.manifest->>'item_id',a.credential_ref,a.credential_version,actor,a.attempt_id,(a.manifest->>'consent_expires_at')::timestamptz,'paused',a.environment='sandbox') RETURNING * INTO i;
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

CREATE OR REPLACE FUNCTION integration.bank_credential_authorize(actor uuid,e uuid,b uuid,r uuid,v bigint,item text,env text,write_value boolean) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);
 IF v IS NULL OR v<1 OR env NOT IN ('sandbox','production') OR item IS NULL THEN PERFORM integration.connection_error('P2700');END IF;
 SELECT * INTO a FROM integration.connection_attempt x WHERE x.entity_id=e AND x.book_id=b AND x.actor_id=actor
 AND ((x.credential_ref=r AND x.credential_version=v AND x.manifest->>'item_id'=item)
 OR (NOT write_value AND x.previous_credential_ref=r AND x.previous_credential_version=v AND x.expected_item_id=item))
 ORDER BY x.expires_at DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND OR a.environment IS DISTINCT FROM env OR a.claim_id IS NULL THEN PERFORM integration.connection_error('P2701');END IF;
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
 OR NOT integration.provider_key_matches(e,b,env,value->>'key_version',false)
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
 OR NOT integration.provider_key_matches(e,b,a.environment,v->>'key_version',true)
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
REVOKE ALL ON FUNCTION integration.provider_api_login(name) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.provider_scope_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.configure_provider_scope(uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.provider_environment(uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_authorize(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_environment_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_lock(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_item_pin(integration.connection_attempt) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.render_connection(integration.connection_attempt,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.provider_origin_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_authorize(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_provider_context(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.provider_key_matches(uuid,uuid,text,text,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.connection_provider_context(uuid,uuid,uuid) TO bookkeeping_api;
COMMIT;
