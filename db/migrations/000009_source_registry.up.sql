-- Synthetic source registry; no upload, observation, journal or acquisition authority.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE banking.source_account (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, source_account_id uuid NOT NULL DEFAULT gen_random_uuid(),
 ledger_account_id uuid NOT NULL, currency_version_id uuid NOT NULL REFERENCES reference.currency,
 binding_date date NOT NULL CHECK(binding_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'), binding_evidence_ref uuid NOT NULL CHECK(binding_evidence_ref<>'00000000-0000-0000-0000-000000000000'),
 created_by uuid NOT NULL REFERENCES ledger.principal, created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)),
 state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','paused','retired')), binding_revision bigint NOT NULL DEFAULT 1 CHECK(binding_revision>0), source_revision bigint NOT NULL DEFAULT 0 CHECK(source_revision=0),
 PRIMARY KEY(entity_id,book_id,source_account_id), UNIQUE(entity_id,source_account_id), UNIQUE(entity_id,book_id,ledger_account_id),
 FOREIGN KEY(entity_id,book_id,ledger_account_id) REFERENCES ledger.account
);
CREATE TABLE banking.source_namespace (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, namespace_id uuid NOT NULL DEFAULT gen_random_uuid(), source_account_id uuid NOT NULL,
 profile text NOT NULL CHECK(profile='jayburd-synthetic-statement-csv/1'), synthetic_dataset_id uuid NOT NULL CHECK(synthetic_dataset_id<>'00000000-0000-0000-0000-000000000000'),
 source_account_ref text COLLATE "C" NOT NULL CHECK(source_account_ref COLLATE "C" ~ '^[A-Za-z0-9_-]{1,64}$'), state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','retired')),
 overlap_review_required boolean NOT NULL, created_by uuid NOT NULL REFERENCES ledger.principal, created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)),
 PRIMARY KEY(entity_id,book_id,namespace_id), UNIQUE(entity_id,profile,synthetic_dataset_id,source_account_ref),
 FOREIGN KEY(entity_id,book_id,source_account_id) REFERENCES banking.source_account
);
CREATE TABLE banking.source_permission (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, source_account_id uuid NOT NULL, principal_id uuid NOT NULL REFERENCES ledger.principal,
 capability text NOT NULL CHECK(capability IN ('read','bind','import','review')), revoked_at timestamptz CHECK(isfinite(revoked_at)),
 PRIMARY KEY(entity_id,book_id,source_account_id,principal_id,capability), FOREIGN KEY(entity_id,book_id,source_account_id) REFERENCES banking.source_account
);
-- One immutable operator manifest serves as replay receipt and provisioning audit.
CREATE TABLE integration.source_registry_receipt (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, operator_login name NOT NULL, operation text NOT NULL CHECK(operation IN ('register','namespace','state','permission')),
 idempotency_key text NOT NULL CHECK(idempotency_key COLLATE "C" ~ '^[A-Za-z0-9._:-]{1,128}$'), approved_by uuid NOT NULL REFERENCES ledger.principal,
 normalized_request jsonb NOT NULL CHECK(octet_length(normalized_request::text)<=16384), result jsonb NOT NULL CHECK(octet_length(result::text)<=65536),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(recorded_at)),
 PRIMARY KEY(entity_id,book_id,operator_login,operation,idempotency_key), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.source_registry_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Fixed diagnostics avoid source metadata in errors.
CREATE FUNCTION banking.source_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='source_registry_rejected'; END $$;
-- Mappings and identity remain immutable even for ordinary owner DML.
CREATE FUNCTION banking.source_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM banking.source_error('P2302'); END IF;
 IF TG_TABLE_NAME='source_account' THEN
  IF (to_jsonb(NEW)-ARRAY['state','binding_revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','binding_revision']) OR OLD.binding_revision=9223372036854775807 OR NEW.binding_revision<>OLD.binding_revision+1 OR OLD.state='retired' AND NEW.state<>'retired' THEN PERFORM banking.source_error('P2302'); END IF;
 ELSE
  IF (to_jsonb(NEW)-'state') IS DISTINCT FROM (to_jsonb(OLD)-'state') OR OLD.state<>'active' OR NEW.state<>'retired' THEN PERFORM banking.source_error('P2302'); END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER source_guard BEFORE UPDATE OR DELETE ON banking.source_account FOR EACH ROW EXECUTE FUNCTION banking.source_guard();
CREATE TRIGGER namespace_guard BEFORE UPDATE OR DELETE ON banking.source_namespace FOR EACH ROW EXECUTE FUNCTION banking.source_guard();
-- Validate a bounded namespace independently of JSON storage or client hashes.
CREATE FUNCTION banking.source_namespace_value(v jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE dataset uuid; ref text;
BEGIN
 PERFORM ledger.keys(v,ARRAY['profile','synthetic_dataset_id','source_account_ref']);
 IF ledger.text_value(v->'profile')<>'jayburd-synthetic-statement-csv/1' THEN PERFORM banking.source_error('P2303'); END IF;
 dataset:=ledger.uuid_value(v->'synthetic_dataset_id'); ref:=ledger.text_value(v->'source_account_ref');
 IF dataset='00000000-0000-0000-0000-000000000000' OR ref COLLATE "C" !~ '^[A-Za-z0-9_-]{1,64}$' THEN PERFORM banking.source_error('P2300'); END IF;
 RETURN jsonb_build_object('profile','jayburd-synthetic-statement-csv/1','synthetic_dataset_id',dataset,'source_account_ref',ref);
END $$;
-- Lock every subject before entity/membership/book to serialize operator grants and revocation.
CREATE FUNCTION banking.source_operator_authorize(actor uuid,reader uuid,e uuid,b uuid,reader_active boolean) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE id uuid;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='source_registry_rejected'; END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR actor IS NULL OR reader IS NULL OR e IS NULL OR b IS NULL THEN PERFORM banking.source_error('P2300'); END IF;
 PERFORM set_config('lock_timeout','1s',true);
 FOR id IN SELECT DISTINCT x FROM unnest(ARRAY[actor,reader]) x ORDER BY x LOOP
  PERFORM 1 FROM ledger.principal WHERE principal_id=id AND (disabled_at IS NULL OR id<>actor AND NOT reader_active) FOR SHARE;
  IF NOT FOUND THEN PERFORM banking.source_error('P2301'); END IF;
 END LOOP;
 PERFORM 1 FROM ledger.entity WHERE entity_id=e AND disabled_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM banking.source_error('P2301'); END IF;
 FOR id IN SELECT DISTINCT x FROM unnest(ARRAY[actor,reader]) x ORDER BY x LOOP
  PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=id AND permission='read' AND (revoked_at IS NULL OR id<>actor AND NOT reader_active) FOR SHARE;
  IF NOT FOUND AND (id=actor OR reader_active) THEN PERFORM banking.source_error('P2301'); END IF;
 END LOOP;
 PERFORM 1 FROM ledger.book WHERE entity_id=e AND book_id=b AND disabled_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN PERFORM banking.source_error('P2301'); END IF;
END $$;
-- Internal owner-only mutation boundary; all supported writes append their receipt atomically.
CREATE FUNCTION banking.source_mutate(actor uuid,e uuid,b uuid,sid uuid,k text,raw text,op text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb; norm jsonb; ns jsonb; result_value jsonb; receipt integration.source_registry_receipt;
 s banking.source_account; a ledger.account; c reference.currency; reader uuid:=actor; account_id_value uuid; currency_id uuid; evidence uuid; d date; expected bigint; revision_text text;
 new_source uuid; new_namespace uuid; enabled_value boolean; current_enabled boolean; n integer;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='source_registry_rejected'; END IF;
 IF raw IS NULL OR octet_length(raw)>8192 OR k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) OR op IS NULL OR op NOT IN ('register','namespace','state','permission') THEN PERFORM banking.source_error('P2300'); END IF;
 BEGIN
  v:=raw::jsonb;
  IF op='register' THEN
   PERFORM ledger.keys(v,ARRAY['ledger_account_id','currency_version_id','binding_date','binding_evidence_ref','reader_principal_id','namespace']);
   account_id_value:=ledger.uuid_value(v->'ledger_account_id'); currency_id:=ledger.uuid_value(v->'currency_version_id'); evidence:=ledger.uuid_value(v->'binding_evidence_ref'); reader:=ledger.uuid_value(v->'reader_principal_id');
   d:=ledger.date_value(ledger.text_value(v->'binding_date')); ns:=banking.source_namespace_value(v->'namespace');
   IF evidence='00000000-0000-0000-0000-000000000000' THEN PERFORM banking.source_error('P2300'); END IF;
   v:=jsonb_build_object('ledger_account_id',account_id_value,'currency_version_id',currency_id,'binding_date',to_char(d,'YYYY-MM-DD'),'binding_evidence_ref',evidence,'reader_principal_id',reader,'namespace',ns);
  ELSE
   IF sid IS NULL THEN PERFORM banking.source_error('P2300'); END IF;
   IF op='namespace' THEN PERFORM ledger.keys(v,ARRAY['expected_binding_revision','namespace']); ns:=banking.source_namespace_value(v->'namespace'); v:=jsonb_set(v,'{namespace}',ns);
   ELSIF op='state' THEN PERFORM ledger.keys(v,ARRAY['expected_binding_revision','state']); IF ledger.text_value(v->'state') NOT IN ('active','paused','retired') THEN PERFORM banking.source_error('P2300'); END IF;
   ELSE
    PERFORM ledger.keys(v,ARRAY['expected_binding_revision','principal_id','enabled']); reader:=ledger.uuid_value(v->'principal_id'); v:=jsonb_set(v,'{principal_id}',to_jsonb(reader));
    IF jsonb_typeof(v->'enabled') IS DISTINCT FROM 'boolean' THEN PERFORM banking.source_error('P2300'); END IF; enabled_value:=(v->>'enabled')::boolean;
   END IF;
   revision_text:=ledger.text_value(v->'expected_binding_revision');
   IF revision_text !~ '^[1-9][0-9]{0,18}$' OR revision_text::numeric>9223372036854775807 THEN PERFORM banking.source_error('P2300'); END IF; expected:=revision_text::bigint;
  END IF;
 EXCEPTION WHEN SQLSTATE 'P2100' OR untranslatable_character OR numeric_value_out_of_range THEN PERFORM banking.source_error('P2300'); END;
 PERFORM banking.source_operator_authorize(actor,reader,e,b,CASE WHEN op='permission' THEN enabled_value ELSE true END);
 norm:=jsonb_build_object('approved_by',actor,'entity_id',e,'book_id',b,'source_account_id',sid,'operation',op,'body',v);
 SELECT * INTO receipt FROM integration.source_registry_receipt WHERE entity_id=e AND book_id=b AND operator_login=session_user AND operation=op AND idempotency_key=k;
 IF FOUND THEN
  IF receipt.normalized_request<>norm THEN PERFORM banking.source_error('P2302'); END IF;
  RETURN receipt.result;
 END IF;
 IF op='register' THEN
  SELECT * INTO a FROM ledger.account WHERE entity_id=e AND book_id=b AND ledger.account.account_id=account_id_value FOR SHARE;
  IF NOT FOUND OR a.category<>'asset' OR a.normal_side<>'debit' OR NOT a.monetary OR d<a.active_from OR a.active_to IS NOT NULL AND d>=a.active_to THEN PERFORM banking.source_error('P2303'); END IF;
  SELECT * INTO c FROM reference.currency WHERE currency_version_id=currency_id;
  IF NOT FOUND OR c.minor_unit IS NULL OR d<c.valid_from OR c.valid_to IS NOT NULL AND d>=c.valid_to THEN PERFORM banking.source_error('P2303'); END IF;
  IF EXISTS(SELECT FROM banking.source_account WHERE entity_id=e AND book_id=b AND ledger_account_id=account_id_value) OR EXISTS(SELECT FROM banking.source_namespace WHERE entity_id=e AND profile=ns->>'profile' AND synthetic_dataset_id=(ns->>'synthetic_dataset_id')::uuid AND source_account_ref=ns->>'source_account_ref') THEN PERFORM banking.source_error('P2302'); END IF;
  INSERT INTO banking.source_account(entity_id,book_id,ledger_account_id,currency_version_id,binding_date,binding_evidence_ref,created_by) VALUES(e,b,account_id_value,currency_id,d,evidence,actor) RETURNING * INTO s;
  new_source:=s.source_account_id;
  INSERT INTO banking.source_namespace(entity_id,book_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by) VALUES(e,b,new_source,ns->>'profile',(ns->>'synthetic_dataset_id')::uuid,ns->>'source_account_ref',false,actor) RETURNING namespace_id INTO new_namespace;
  INSERT INTO banking.source_permission VALUES(e,b,new_source,reader,'read',NULL);
 ELSE
  SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR UPDATE;
  IF NOT FOUND THEN PERFORM banking.source_error('P2301'); END IF;
  IF expected<>s.binding_revision THEN PERFORM banking.source_error('P2302'); END IF;
  IF s.binding_revision=9223372036854775807 THEN PERFORM banking.source_error('P2304'); END IF;
  new_source:=sid;
  IF op='namespace' THEN
   IF s.state<>'active' THEN PERFORM banking.source_error('P2302'); END IF;
   SELECT count(*) INTO n FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND source_account_id=sid;
   IF n>=25 THEN PERFORM banking.source_error('P2304'); END IF;
   IF EXISTS(SELECT FROM banking.source_namespace WHERE entity_id=e AND profile=ns->>'profile' AND synthetic_dataset_id=(ns->>'synthetic_dataset_id')::uuid AND source_account_ref=ns->>'source_account_ref') THEN PERFORM banking.source_error('P2302'); END IF;
   INSERT INTO banking.source_namespace(entity_id,book_id,source_account_id,profile,synthetic_dataset_id,source_account_ref,overlap_review_required,created_by) VALUES(e,b,sid,ns->>'profile',(ns->>'synthetic_dataset_id')::uuid,ns->>'source_account_ref',true,actor) RETURNING namespace_id INTO new_namespace;
  ELSIF op='state' THEN
   IF s.state='retired' OR s.state=v->>'state' THEN PERFORM banking.source_error('P2302'); END IF;
   IF v->>'state'='retired' THEN UPDATE banking.source_namespace SET state='retired' WHERE entity_id=e AND book_id=b AND source_account_id=sid AND state='active'; END IF;
  ELSE
   SELECT revoked_at IS NULL INTO current_enabled FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=reader AND capability='read' FOR UPDATE;
   current_enabled:=coalesce(current_enabled,false);
   IF current_enabled=enabled_value THEN PERFORM banking.source_error('P2302'); END IF;
   INSERT INTO banking.source_permission VALUES(e,b,sid,reader,'read',CASE WHEN enabled_value THEN NULL ELSE clock_timestamp() END)
    ON CONFLICT(entity_id,book_id,source_account_id,principal_id,capability) DO UPDATE SET revoked_at=EXCLUDED.revoked_at;
  END IF;
  UPDATE banking.source_account SET binding_revision=binding_revision+1,state=CASE WHEN op='state' THEN v->>'state' ELSE state END WHERE entity_id=e AND book_id=b AND source_account_id=sid RETURNING * INTO s;
 END IF;
 result_value:=jsonb_build_object('source_account_id',new_source,'binding_revision',s.binding_revision::text);
 IF new_namespace IS NOT NULL THEN result_value:=result_value||jsonb_build_object('namespace_id',new_namespace); END IF;
 INSERT INTO integration.source_registry_receipt(entity_id,book_id,operator_login,operation,idempotency_key,approved_by,normalized_request,result) VALUES(e,b,session_user,op,k,actor,norm,result_value);
 RETURN result_value;
EXCEPTION WHEN unique_violation THEN PERFORM banking.source_error('P2302');
END $$;
-- Owner entrypoints compile the operation; callers cannot select arbitrary mutations.
CREATE FUNCTION banking.register_source(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$ SELECT banking.source_mutate(actor,e,b,NULL,k,raw,'register') $$;
CREATE FUNCTION banking.register_source_namespace(actor uuid,e uuid,b uuid,sid uuid,k text,raw text) RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$ SELECT banking.source_mutate(actor,e,b,sid,k,raw,'namespace') $$;
CREATE FUNCTION banking.set_source_state(actor uuid,e uuid,b uuid,sid uuid,k text,raw text) RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$ SELECT banking.source_mutate(actor,e,b,sid,k,raw,'state') $$;
CREATE FUNCTION banking.set_source_permission(actor uuid,e uuid,b uuid,sid uuid,k text,raw text) RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$ SELECT banking.source_mutate(actor,e,b,sid,k,raw,'permission') $$;
-- Runtime reads require both current entity authority and an explicit source read grant.
CREATE FUNCTION banking.get_source(actor uuid,e uuid,b uuid,sid uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE s banking.source_account; result_value jsonb;
BEGIN
 PERFORM set_config('lock_timeout','1s',true);
 BEGIN PERFORM ledger.authorize(actor,e,b,'read',false); EXCEPTION WHEN SQLSTATE 'P2101' OR SQLSTATE 'P2102' THEN PERFORM banking.source_error('P2301'); WHEN SQLSTATE 'P2100' THEN PERFORM banking.source_error('P2300'); END;
 PERFORM set_config('lock_timeout','1s',true);
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR SHARE;
 IF NOT FOUND THEN PERFORM banking.source_error('P2301'); END IF;
 PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=actor AND capability='read' AND revoked_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM banking.source_error('P2301'); END IF;
 SELECT jsonb_build_object('profile','source-registry-v1','entity_id',e,'book_id',b,'source_account_id',sid,'ledger_account_id',s.ledger_account_id,'currency_version_id',s.currency_version_id,'currency',c.alphabetic_code,'minor_units',c.minor_unit,'binding_date',to_char(s.binding_date,'YYYY-MM-DD'),'state',s.state,'binding_revision',s.binding_revision::text,'source_revision',s.source_revision::text,
 'namespaces',(SELECT coalesce(jsonb_agg(jsonb_build_object('namespace_id',n.namespace_id,'profile',n.profile,'synthetic_dataset_id',n.synthetic_dataset_id,'source_account_ref',n.source_account_ref,'state',n.state,'overlap_review_required',n.overlap_review_required) ORDER BY n.namespace_id),'[]'::jsonb) FROM banking.source_namespace n WHERE n.entity_id=e AND n.book_id=b AND n.source_account_id=sid)) INTO result_value FROM reference.currency c WHERE c.currency_version_id=s.currency_version_id;
 IF (SELECT count(*) FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND source_account_id=sid)>25 OR octet_length(result_value::text)>65536 THEN PERFORM banking.source_error('P2304'); END IF;
 RETURN result_value;
END $$;
REVOKE ALL ON banking.source_account,banking.source_namespace,banking.source_permission,integration.source_registry_receipt FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA banking FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher;
GRANT USAGE ON SCHEMA banking TO bookkeeping_api;
GRANT EXECUTE ON FUNCTION banking.get_source(uuid,uuid,uuid,uuid) TO bookkeeping_api;
COMMIT;
