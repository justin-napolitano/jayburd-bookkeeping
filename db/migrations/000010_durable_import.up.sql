-- Synthetic durable import. Confirmation creates review evidence, never journals.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
ALTER TABLE ledger.membership DROP CONSTRAINT membership_permission_check;
ALTER TABLE ledger.membership ADD CONSTRAINT membership_permission_check CHECK(permission IN ('read','prepare','post','close','import'));
ALTER TABLE banking.source_account DROP CONSTRAINT source_account_source_revision_check;
ALTER TABLE banking.source_account ADD CONSTRAINT source_account_source_revision_check CHECK(source_revision>=0);
ALTER TABLE banking.source_namespace ADD UNIQUE(entity_id,book_id,source_account_id,namespace_id);
ALTER TABLE integration.source_registry_receipt DROP CONSTRAINT source_registry_receipt_operation_check;
ALTER TABLE integration.source_registry_receipt ADD CONSTRAINT source_registry_receipt_operation_check CHECK(operation IN ('register','namespace','state','permission','import_permission'));
-- Binding edits and evidence publication use independent monotonic revisions.
CREATE OR REPLACE FUNCTION banking.source_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM banking.source_error('P2302'); END IF;
 IF TG_TABLE_NAME='source_account' THEN
  IF (to_jsonb(NEW)-ARRAY['state','binding_revision','source_revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','binding_revision','source_revision'])
   OR NOT ((NEW.binding_revision::numeric=OLD.binding_revision::numeric+1 AND NEW.source_revision=OLD.source_revision)
    OR (NEW.binding_revision=OLD.binding_revision AND NEW.source_revision::numeric=OLD.source_revision::numeric+1 AND NEW.state=OLD.state))
   OR OLD.state='retired' AND NEW.state<>'retired' THEN PERFORM banking.source_error('P2302'); END IF;
 ELSE
  IF (to_jsonb(NEW)-'state') IS DISTINCT FROM (to_jsonb(OLD)-'state') OR OLD.state<>'active' OR NEW.state<>'retired' THEN PERFORM banking.source_error('P2302'); END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TABLE banking.import_batch (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, batch_id uuid NOT NULL DEFAULT gen_random_uuid(), source_account_id uuid NOT NULL, namespace_id uuid NOT NULL,
 statement_ref text COLLATE "C" NOT NULL CHECK(statement_ref COLLATE "C" ~ '^[A-Za-z0-9_-]{1,64}$'), binding_revision bigint NOT NULL CHECK(binding_revision>0),
 profile text NOT NULL CHECK(profile='jayburd-synthetic-statement-csv/1'), parser_version text NOT NULL DEFAULT '1' CHECK(parser_version='1'),
 starts_on date NOT NULL CHECK(starts_on BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'), ends_before date NOT NULL CHECK(ends_before BETWEEN DATE '0001-01-01' AND DATE '9999-12-31' AND ends_before>starts_on),
 object_id uuid NOT NULL DEFAULT gen_random_uuid(), byte_sha256 text CHECK(byte_sha256 COLLATE "C" ~ '^[0-9a-f]{64}$'), byte_count integer CHECK(byte_count BETWEEN 1 AND 1048576),
 state text NOT NULL DEFAULT 'received' CHECK(state IN ('received','preview_ready','rejected','imported','cancelled')),
 created_by uuid NOT NULL REFERENCES ledger.principal, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '7 days',
 raw_deleted_at timestamptz, publication jsonb CHECK(octet_length(publication::text)<=2097152), imported_body jsonb, imported_result jsonb,
 PRIMARY KEY(entity_id,book_id,batch_id), UNIQUE(entity_id,book_id,namespace_id,statement_ref), UNIQUE(entity_id,book_id,object_id),
 FOREIGN KEY(entity_id,book_id,source_account_id,namespace_id) REFERENCES banking.source_namespace(entity_id,book_id,source_account_id,namespace_id),
 CHECK(isfinite(created_at) AND isfinite(expires_at) AND expires_at>created_at), CHECK(raw_deleted_at IS NULL OR isfinite(raw_deleted_at)),
 CHECK((byte_sha256 IS NULL)=(byte_count IS NULL)), CHECK(state IN ('received','cancelled') OR byte_sha256 IS NOT NULL), CHECK((state='imported')=(imported_result IS NOT NULL))
);
CREATE TABLE integration.import_registration (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, principal_id uuid NOT NULL REFERENCES ledger.principal, idempotency_key text NOT NULL CHECK(idempotency_key COLLATE "C" ~ '^[A-Za-z0-9._:-]{1,128}$'),
 batch_id uuid NOT NULL, normalized_request jsonb NOT NULL CHECK(octet_length(normalized_request::text)<=8192), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,principal_id,idempotency_key), FOREIGN KEY(entity_id,book_id,batch_id) REFERENCES banking.import_batch
);
CREATE TABLE banking.import_preview (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, preview_id uuid NOT NULL DEFAULT gen_random_uuid(), batch_id uuid NOT NULL,
 preview_revision bigint NOT NULL CHECK(preview_revision>0), source_revision bigint NOT NULL CHECK(source_revision>=0), binding_revision bigint NOT NULL CHECK(binding_revision>0),
 rows jsonb NOT NULL CHECK(jsonb_typeof(rows)='array' AND jsonb_array_length(rows)<=1000 AND octet_length(rows::text)<=2097152),
 diagnostics jsonb NOT NULL CHECK(jsonb_typeof(diagnostics)='array' AND jsonb_array_length(diagnostics)<=100), error_count integer NOT NULL CHECK(error_count>=0), truncated boolean NOT NULL,
 created_count integer NOT NULL CHECK(created_count BETWEEN 0 AND 1000), existing_count integer NOT NULL CHECK(existing_count BETWEEN 0 AND 1000),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,preview_id), UNIQUE(entity_id,book_id,batch_id,preview_revision), FOREIGN KEY(entity_id,book_id,batch_id) REFERENCES banking.import_batch,
 CHECK((error_count=0 AND jsonb_array_length(rows) BETWEEN 1 AND 1000 AND created_count+existing_count=jsonb_array_length(rows) AND diagnostics='[]'::jsonb AND NOT truncated)
 OR (error_count>0 AND rows='[]'::jsonb AND created_count=0 AND existing_count=0 AND jsonb_array_length(diagnostics)>0))
);
CREATE TABLE banking.source_observation (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, observation_id uuid NOT NULL DEFAULT gen_random_uuid(), source_account_id uuid NOT NULL, namespace_id uuid NOT NULL,
 external_entry_id text COLLATE "C" NOT NULL CHECK(external_entry_id COLLATE "C" ~ '^[A-Za-z0-9_-]{1,64}$'), current_revision bigint NOT NULL DEFAULT 1 CHECK(current_revision=1), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,observation_id), UNIQUE(entity_id,source_account_id,namespace_id,external_entry_id),
 FOREIGN KEY(entity_id,book_id,source_account_id,namespace_id) REFERENCES banking.source_namespace(entity_id,book_id,source_account_id,namespace_id)
);
CREATE TABLE banking.observation_revision (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, observation_id uuid NOT NULL, revision bigint NOT NULL CHECK(revision=1), source_state text NOT NULL CHECK(source_state='booked'),
 booked_date date NOT NULL CHECK(booked_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'), currency_version_id uuid NOT NULL REFERENCES reference.currency,
 direction text NOT NULL CHECK(direction IN ('inflow','outflow')), amount_lexeme text NOT NULL CHECK(amount_lexeme COLLATE "C" ~ '^[0-9]{1,29}(\.[0-9]{1,9})?$'), amount ledger.amount NOT NULL CHECK(amount>0), description text NOT NULL CHECK(length(description) BETWEEN 1 AND 500),
 canonical_fields jsonb NOT NULL CHECK(octet_length(canonical_fields::text)<=4096), recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,observation_id,revision), FOREIGN KEY(entity_id,book_id,observation_id) REFERENCES banking.source_observation
);
ALTER TABLE banking.source_observation ADD FOREIGN KEY(entity_id,book_id,observation_id,current_revision) REFERENCES banking.observation_revision(entity_id,book_id,observation_id,revision) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE banking.import_row (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, batch_id uuid NOT NULL, record_ordinal integer NOT NULL CHECK(record_ordinal BETWEEN 2 AND 1001), observation_id uuid NOT NULL, observation_revision bigint NOT NULL,
 original_fields jsonb NOT NULL CHECK(octet_length(original_fields::text)<=4096), PRIMARY KEY(entity_id,book_id,batch_id,record_ordinal),
 FOREIGN KEY(entity_id,book_id,batch_id) REFERENCES banking.import_batch, FOREIGN KEY(entity_id,book_id,observation_id,observation_revision) REFERENCES banking.observation_revision
);
CREATE TABLE banking.review_item (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, review_id uuid NOT NULL DEFAULT gen_random_uuid(), source_account_id uuid NOT NULL, review_revision bigint NOT NULL DEFAULT 1 CHECK(review_revision=1),
 state text NOT NULL CHECK(state IN ('needs_review','overlap_hold')), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,review_id), FOREIGN KEY(entity_id,book_id,source_account_id) REFERENCES banking.source_account
);
CREATE TABLE banking.review_decision (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, decision_id uuid NOT NULL DEFAULT gen_random_uuid(), review_id uuid NOT NULL, expected_review_revision bigint NOT NULL CHECK(expected_review_revision=0), kind text NOT NULL CHECK(kind='initialize'),
 actor_id uuid NOT NULL REFERENCES ledger.principal, reason text NOT NULL CHECK(reason='synthetic_import'), body jsonb NOT NULL CHECK(octet_length(body::text)<=65536), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,decision_id), UNIQUE(entity_id,book_id,decision_id,review_id), FOREIGN KEY(entity_id,book_id,review_id) REFERENCES banking.review_item
);
CREATE TABLE banking.review_evidence (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, decision_id uuid NOT NULL, observation_id uuid NOT NULL, review_id uuid NOT NULL, linked_revision bigint NOT NULL,
 PRIMARY KEY(entity_id,book_id,decision_id,observation_id), UNIQUE(entity_id,book_id,decision_id,observation_id,review_id),
 FOREIGN KEY(entity_id,book_id,decision_id,review_id) REFERENCES banking.review_decision(entity_id,book_id,decision_id,review_id), FOREIGN KEY(entity_id,book_id,observation_id,linked_revision) REFERENCES banking.observation_revision
);
CREATE TABLE banking.review_assignment (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, observation_id uuid NOT NULL, review_id uuid NOT NULL, decision_id uuid NOT NULL,
 PRIMARY KEY(entity_id,book_id,observation_id), FOREIGN KEY(entity_id,book_id,decision_id,observation_id,review_id) REFERENCES banking.review_evidence(entity_id,book_id,decision_id,observation_id,review_id)
);
-- Every evidence row is immutable; future correction/review transitions require reviewed migrations.
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['import_preview','source_observation','observation_revision','import_row','review_item','review_decision','review_evidence','review_assignment'] LOOP
 EXECUTE format('CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.%I FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row()',t);
 END LOOP;
END $$;
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.import_registration FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Bound errors and JSON before casts; no source fields leak in PostgreSQL diagnostics.
CREATE FUNCTION banking.import_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$ BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='import_rejected'; END $$;
CREATE FUNCTION banking.import_json(raw text,max_bytes integer) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF raw IS NULL OR octet_length(raw)>max_bytes OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM banking.import_error('P2400'); END IF;
 RETURN raw::jsonb;
EXCEPTION WHEN untranslatable_character OR numeric_value_out_of_range THEN PERFORM banking.import_error('P2400');
END $$;
CREATE FUNCTION banking.import_revision(v jsonb) RETURNS bigint LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE x text;
BEGIN x:=ledger.text_value(v); IF x !~ '^(0|[1-9][0-9]{0,18})$' OR x::numeric>9223372036854775807 THEN PERFORM banking.import_error('P2400'); END IF; RETURN x::bigint;
EXCEPTION WHEN SQLSTATE 'P2100' THEN PERFORM banking.import_error('P2400'); END $$;
-- Owner attests entity and source grants explicitly; revocation also works for dormant subjects.
CREATE FUNCTION banking.set_source_import_permission(actor uuid,e uuid,b uuid,sid uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE v jsonb; subject uuid; expected bigint; enabled boolean; entity_enabled boolean; s banking.source_account; receipt integration.source_registry_receipt; norm jsonb; result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='import_rejected'; END IF;
 v:=banking.import_json(raw,8192); PERFORM ledger.keys(v,ARRAY['expected_binding_revision','principal_id','enabled','entity_import_enabled']);
 subject:=ledger.uuid_value(v->'principal_id'); expected:=banking.import_revision(v->'expected_binding_revision');
 IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' OR jsonb_typeof(v->'enabled')<>'boolean' OR jsonb_typeof(v->'entity_import_enabled')<>'boolean' THEN PERFORM banking.import_error('P2400'); END IF;
 enabled:=(v->>'enabled')::boolean; entity_enabled:=(v->>'entity_import_enabled')::boolean;
 PERFORM banking.source_operator_authorize(actor,subject,e,b,enabled OR entity_enabled);
 norm:=jsonb_build_object('approved_by',actor,'source_account_id',sid,'body',v);
 SELECT * INTO receipt FROM integration.source_registry_receipt WHERE entity_id=e AND book_id=b AND operator_login=session_user AND operation='import_permission' AND idempotency_key=k;
 IF FOUND THEN IF receipt.normalized_request<>norm THEN PERFORM banking.import_error('P2402'); END IF; RETURN receipt.result; END IF;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR UPDATE;
 IF NOT FOUND THEN PERFORM banking.import_error('P2401'); END IF;
 IF s.binding_revision<>expected OR expected=9223372036854775807 THEN PERFORM banking.import_error('P2403'); END IF;
 INSERT INTO ledger.membership VALUES(e,subject,'import',CASE WHEN entity_enabled THEN NULL ELSE clock_timestamp() END) ON CONFLICT(entity_id,principal_id,permission) DO UPDATE SET revoked_at=EXCLUDED.revoked_at;
 INSERT INTO banking.source_permission VALUES(e,b,sid,subject,'import',CASE WHEN enabled THEN NULL ELSE clock_timestamp() END) ON CONFLICT(entity_id,book_id,source_account_id,principal_id,capability) DO UPDATE SET revoked_at=EXCLUDED.revoked_at;
 UPDATE banking.source_account SET binding_revision=binding_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=sid RETURNING * INTO s;
 result_value:=jsonb_build_object('source_account_id',sid,'binding_revision',s.binding_revision::text);
 INSERT INTO integration.source_registry_receipt(entity_id,book_id,operator_login,operation,idempotency_key,approved_by,normalized_request,result) VALUES(e,b,session_user,'import_permission',k,actor,norm,result_value);
 RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' THEN PERFORM banking.import_error('P2400'); END $$;
-- Same principal/entity/membership/book/source order for grants, publication and recovery.
CREATE FUNCTION banking.import_authorize(actor uuid,e uuid,b uuid,sid uuid,writing boolean) RETURNS banking.source_account LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE s banking.source_account; cap text;
BEGIN
 PERFORM set_config('lock_timeout','1s',true);
 PERFORM ledger.authorize(actor,e,b,'read',writing);
 IF writing THEN
  PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='import' AND revoked_at IS NULL FOR SHARE;
  IF NOT FOUND THEN PERFORM banking.import_error('P2401'); END IF;
 END IF;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR UPDATE;
 IF NOT FOUND THEN PERFORM banking.import_error('P2401'); END IF;
 FOREACH cap IN ARRAY CASE WHEN writing THEN ARRAY['read','import'] ELSE ARRAY['read'] END LOOP
  PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=actor AND capability=cap AND revoked_at IS NULL FOR SHARE;
  IF NOT FOUND THEN PERFORM banking.import_error('P2401'); END IF;
 END LOOP;
 RETURN s;
EXCEPTION WHEN SQLSTATE 'P2100' THEN PERFORM banking.import_error('P2400'); WHEN SQLSTATE 'P2101' OR SQLSTATE 'P2102' THEN PERFORM banking.import_error('P2401'); END $$;
-- Public rendering excludes opaque object identity and private attribution.
CREATE FUNCTION banking.render_import(x banking.import_batch) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE p banking.import_preview; rev bigint;
BEGIN
 SELECT * INTO p FROM banking.import_preview WHERE entity_id=x.entity_id AND book_id=x.book_id AND batch_id=x.batch_id ORDER BY preview_revision DESC LIMIT 1;
 SELECT source_revision INTO rev FROM banking.source_account WHERE entity_id=x.entity_id AND book_id=x.book_id AND source_account_id=x.source_account_id;
 RETURN jsonb_build_object('profile','durable-import-v1','batch_id',x.batch_id,'source_account_id',x.source_account_id,'namespace_id',x.namespace_id,'state',x.state,
 'binding_revision',x.binding_revision::text,'source_revision',rev::text,'preview_id',p.preview_id,'preview_revision',p.preview_revision::text,
 'created_count',p.created_count::text,'existing_count',p.existing_count::text,'diagnostics',coalesce(p.diagnostics,'[]'::jsonb),'error_count',coalesce(p.error_count,0),'truncated',coalesce(p.truncated,false),
 'raw_available',x.byte_sha256 IS NOT NULL AND x.raw_deleted_at IS NULL,'created_at',x.created_at,'expires_at',x.expires_at,
 'preview',CASE WHEN p.preview_id IS NULL THEN NULL ELSE jsonb_build_object('preview_id',p.preview_id,'preview_revision',p.preview_revision::text,'binding_revision',p.binding_revision::text,'source_revision',p.source_revision::text,'created_count',p.created_count::text,'existing_count',p.existing_count::text) END);
END $$;
CREATE FUNCTION banking.get_import_batch(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE x banking.import_batch;
BEGIN
 SELECT * INTO x FROM banking.import_batch WHERE entity_id=e AND book_id=b AND batch_id=id;
 PERFORM banking.import_authorize(actor,e,b,x.source_account_id,false);
 RETURN banking.render_import(x);
END $$;
-- Service-only context: same runtime login, but no HTTP endpoint accepts publication rows.
CREATE FUNCTION banking.import_context(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE x banking.import_batch; s banking.source_account; n banking.source_namespace; c reference.currency;
BEGIN
 SELECT * INTO x FROM banking.import_batch WHERE entity_id=e AND book_id=b AND batch_id=id;
 s:=banking.import_authorize(actor,e,b,x.source_account_id,true);
 SELECT * INTO n FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=x.namespace_id;
 IF x.state='cancelled' OR x.state<>'imported' AND (clock_timestamp()>=x.expires_at OR x.raw_deleted_at IS NOT NULL) THEN PERFORM banking.import_error('P2402'); END IF;
 IF x.state IN ('received','preview_ready') AND (s.state<>'active' OR n.state<>'active' OR s.binding_revision<>x.binding_revision) THEN PERFORM banking.import_error('P2403'); END IF;
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=s.currency_version_id;
 RETURN banking.render_import(x)||jsonb_build_object('profile',x.profile,'object_id',x.object_id,'source_account_ref',n.source_account_ref,'starts_on',to_char(x.starts_on,'YYYY-MM-DD'),'ends_before',to_char(x.ends_before,'YYYY-MM-DD'),
 'currency',jsonb_build_object('code',c.alphabetic_code,'version_id',c.currency_version_id,'minor_unit',c.minor_unit,'valid_from',to_char(c.valid_from,'YYYY-MM-DD'),'valid_to',coalesce(to_char(c.valid_to,'YYYY-MM-DD'),'')));
END $$;
-- Idempotency is actor-scoped; statement identity is shared within its registered namespace.
CREATE FUNCTION banking.register_import_batch(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; sid uuid; nid uuid; expected bigint; d date; ending date; s banking.source_account; n banking.source_namespace; x banking.import_batch; r integration.import_registration;
BEGIN
 v:=banking.import_json(raw,8192); PERFORM ledger.keys(v,ARRAY['source_account_id','namespace_id','binding_revision','statement_ref','starts_on','ends_before','profile']);
 sid:=ledger.uuid_value(v->'source_account_id'); nid:=ledger.uuid_value(v->'namespace_id'); expected:=banking.import_revision(v->'binding_revision');
 d:=ledger.date_value(ledger.text_value(v->'starts_on')); ending:=ledger.date_value(ledger.text_value(v->'ends_before'));
 IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' OR ledger.text_value(v->'profile')<>'jayburd-synthetic-statement-csv/1' OR ledger.text_value(v->'statement_ref') COLLATE "C" !~ '^[A-Za-z0-9_-]{1,64}$' OR d>=ending THEN PERFORM banking.import_error('P2400'); END IF;
 s:=banking.import_authorize(actor,e,b,sid,true);
 SELECT * INTO r FROM integration.import_registration WHERE entity_id=e AND book_id=b AND principal_id=actor AND idempotency_key=k;
 IF FOUND THEN IF r.normalized_request<>v THEN PERFORM banking.import_error('P2402'); END IF; RETURN banking.import_context(actor,e,b,r.batch_id); END IF;
 SELECT * INTO x FROM banking.import_batch WHERE entity_id=e AND book_id=b AND namespace_id=nid AND statement_ref=v->>'statement_ref';
 IF FOUND THEN
  IF ROW(x.source_account_id,x.binding_revision,x.profile,x.starts_on,x.ends_before) IS DISTINCT FROM ROW(sid,expected,v->>'profile',d,ending) THEN PERFORM banking.import_error('P2402'); END IF;
 ELSE
  SELECT * INTO n FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=nid AND source_account_id=sid;
  IF NOT FOUND OR n.state<>'active' OR s.state<>'active' THEN PERFORM banking.import_error('P2401'); END IF;
  IF s.binding_revision<>expected THEN PERFORM banking.import_error('P2403'); END IF;
  INSERT INTO banking.import_batch(entity_id,book_id,source_account_id,namespace_id,statement_ref,binding_revision,profile,starts_on,ends_before,created_by)
   VALUES(e,b,sid,nid,v->>'statement_ref',expected,v->>'profile',d,ending,actor) RETURNING * INTO x;
 END IF;
 INSERT INTO integration.import_registration(entity_id,book_id,principal_id,idempotency_key,batch_id,normalized_request) VALUES(e,b,actor,k,x.batch_id,v);
 RETURN banking.import_context(actor,e,b,x.batch_id);
EXCEPTION WHEN SQLSTATE 'P2100' THEN PERFORM banking.import_error('P2400'); WHEN unique_violation THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_import_registration'; END $$;
-- Canonical equality deliberately excludes record position and original amount lexeme.
CREATE FUNCTION banking.import_canonical(r jsonb) RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$ SELECT r-ARRAY['record','amount'] $$;
-- Revalidate trusted decoder output against SQL-owned source/currency metadata.
CREATE FUNCTION banking.validate_import_rows(rows_value jsonb,x banking.import_batch,s banking.source_account,n banking.source_namespace) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE r jsonb; c reference.currency; d date; a text; ordinal integer:=1; seen text[]:=ARRAY[]::text[];
BEGIN
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=s.currency_version_id;
 IF jsonb_typeof(rows_value) IS DISTINCT FROM 'array' THEN PERFORM banking.import_error('P2400'); END IF;
 IF jsonb_array_length(rows_value) NOT BETWEEN 1 AND 1000 OR octet_length(rows_value::text)>2097152 THEN PERFORM banking.import_error('P2404'); END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(rows_value) LOOP
  ordinal:=ordinal+1;
  PERFORM ledger.keys(r,ARRAY['record','source_account_ref','entry_id','booked_on','currency','direction','amount','description','canonical_amount','currency_version_id']);
  IF jsonb_typeof(r->'record')<>'number' OR r->>'record'<>ordinal::text OR ledger.text_value(r->'source_account_ref')<>n.source_account_ref
   OR ledger.text_value(r->'entry_id') COLLATE "C" !~ '^[A-Za-z0-9_-]{1,64}$' OR r->>'entry_id'=ANY(seen)
   OR ledger.text_value(r->'currency')<>c.alphabetic_code OR ledger.text_value(r->'currency_version_id')<>c.currency_version_id::text
   OR ledger.text_value(r->'direction') NOT IN ('inflow','outflow') THEN PERFORM banking.import_error('P2400'); END IF;
  seen:=array_append(seen,r->>'entry_id');
  d:=ledger.date_value(ledger.text_value(r->'booked_on'));
  IF d<x.starts_on OR d>=x.ends_before OR d<c.valid_from OR c.valid_to IS NOT NULL AND d>=c.valid_to THEN PERFORM banking.import_error('P2400'); END IF;
  a:=ledger.text_value(r->'amount');
  IF a COLLATE "C" !~ '^[0-9]{1,29}(\.[0-9]{1,9})?$' THEN PERFORM banking.import_error('P2400'); END IF;
  IF a::numeric<=0 OR scale(trim_scale(a::numeric))>c.minor_unit OR c.minor_unit IS NULL OR ledger.text_value(r->'canonical_amount')<>trim_scale(a::numeric)::text THEN PERFORM banking.import_error('P2400'); END IF;
  a:=ledger.text_value(r->'description');
  IF length(a) NOT BETWEEN 1 AND 500 OR a COLLATE "C" !~ '^[ -~]+$' THEN PERFORM banking.import_error('P2400'); END IF;
 END LOOP;
EXCEPTION WHEN SQLSTATE 'P2100' OR numeric_value_out_of_range THEN PERFORM banking.import_error('P2400'); END $$;
-- Publication is atomic and may only follow trusted server read-back plus strict Decode.
CREATE FUNCTION banking.publish_import_preview(actor uuid,e uuid,b uuid,id uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; x banking.import_batch; s banking.source_account; n banking.source_namespace; previous banking.import_preview; p banking.import_preview;
 r jsonb; old_fields jsonb; rows_value jsonb; diagnostics_value jsonb:='[]'::jsonb; errors_count integer:=0; truncated_value boolean:=false; new_count integer:=0; old_count integer:=0; next_revision bigint:=1;
BEGIN
 v:=banking.import_json(raw,2097152);
 IF v?'rows' THEN PERFORM ledger.keys(v,ARRAY['profile','sha256','byte_count','rows']); ELSE PERFORM ledger.keys(v,ARRAY['profile','sha256','byte_count','diagnostics','total','truncated']); END IF;
 IF ledger.text_value(v->'profile')<>'jayburd-synthetic-statement-csv/1' OR ledger.text_value(v->'sha256') COLLATE "C" !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(v->'byte_count')<>'number' OR v->>'byte_count' !~ '^[1-9][0-9]{0,6}$' OR (v->>'byte_count')::numeric>1048576 THEN PERFORM banking.import_error('P2400'); END IF;
 SELECT * INTO x FROM banking.import_batch WHERE entity_id=e AND book_id=b AND batch_id=id;
 s:=banking.import_authorize(actor,e,b,x.source_account_id,true);
 PERFORM 1 FROM banking.import_batch WHERE entity_id=e AND book_id=b AND batch_id=id FOR UPDATE;
 SELECT * INTO n FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=x.namespace_id;
 IF x.publication IS NOT NULL AND x.publication<>v THEN PERFORM banking.import_error('P2402'); END IF;
 IF x.state='cancelled' THEN PERFORM banking.import_error('P2402'); END IF;
 IF x.byte_sha256 IS NOT NULL AND ROW(x.byte_sha256,x.byte_count) IS DISTINCT FROM ROW(v->>'sha256',(v->>'byte_count')::integer) THEN PERFORM banking.import_error('P2402'); END IF;
 IF v?'rows' THEN
  rows_value:=v->'rows'; PERFORM banking.validate_import_rows(rows_value,x,s,n);
 ELSE
  IF jsonb_typeof(v->'diagnostics') IS DISTINCT FROM 'array' OR jsonb_typeof(v->'total')<>'number' OR v->>'total' !~ '^[1-9][0-9]{0,4}$' OR (v->>'total')::integer>10000 OR jsonb_typeof(v->'truncated')<>'boolean' THEN PERFORM banking.import_error('P2400'); END IF;
  diagnostics_value:=v->'diagnostics'; errors_count:=(v->>'total')::integer; truncated_value:=(v->>'truncated')::boolean;
  IF jsonb_array_length(diagnostics_value)<>least(errors_count,100) OR truncated_value<>(errors_count>100) THEN PERFORM banking.import_error('P2400'); END IF;
  FOR r IN SELECT value FROM jsonb_array_elements(diagnostics_value) LOOP
   PERFORM ledger.keys(r,ARRAY['code','record','field']);
   IF ledger.text_value(r->'code') NOT IN ('invalid_expectations','read_failed','file_too_large','invalid_line_ending','invalid_encoding','invalid_header','no_records','too_many_records','record_too_large','blank_record','invalid_csv','account_mismatch','invalid_entry_id','duplicate_entry_id','invalid_date','outside_interval','currency_not_valid','currency_mismatch','invalid_direction','invalid_amount','invalid_description')
    OR ledger.text_value(r->'field') NOT IN ('','source_account_ref','entry_id','booked_on','currency','direction','amount','description') OR jsonb_typeof(r->'record')<>'number' OR r->>'record' !~ '^(0|[1-9][0-9]{0,6})$' OR (r->>'record')::integer>1048576 THEN PERFORM banking.import_error('P2400'); END IF;
  END LOOP;
  rows_value:='[]'::jsonb;
 END IF;
 SELECT * INTO previous FROM banking.import_preview WHERE entity_id=e AND book_id=b AND batch_id=id ORDER BY preview_revision DESC LIMIT 1;
 -- A published byte identity cannot be reinterpreted by submitting different decoded fields.
 IF FOUND THEN
  IF previous.error_count=0 AND (NOT(v?'rows') OR previous.rows<>rows_value) THEN PERFORM banking.import_error('P2402'); END IF;
  IF previous.error_count>0 THEN
   IF x.state<>'rejected' THEN PERFORM banking.import_error('P2402'); END IF;
   RETURN banking.render_import(x);
  END IF;
  IF x.state='imported' OR previous.source_revision=s.source_revision AND previous.binding_revision=s.binding_revision THEN RETURN banking.render_import(x); END IF;
  IF previous.preview_revision=9223372036854775807 THEN PERFORM banking.import_error('P2404'); END IF;
  next_revision:=previous.preview_revision+1;
 END IF;
 IF x.state NOT IN ('received','preview_ready') OR clock_timestamp()>=x.expires_at OR x.raw_deleted_at IS NOT NULL THEN PERFORM banking.import_error('P2402'); END IF;
 IF s.state<>'active' OR n.state<>'active' OR s.binding_revision<>x.binding_revision THEN PERFORM banking.import_error('P2403'); END IF;
 IF errors_count=0 THEN
  FOR r IN SELECT value FROM jsonb_array_elements(rows_value) LOOP
   SELECT q.canonical_fields INTO old_fields FROM banking.source_observation o JOIN banking.observation_revision q USING(entity_id,book_id,observation_id)
    WHERE o.entity_id=e AND o.book_id=b AND o.source_account_id=x.source_account_id AND o.namespace_id=x.namespace_id AND o.external_entry_id=r->>'entry_id' AND q.revision=o.current_revision;
   IF FOUND THEN
    IF old_fields<>banking.import_canonical(r) THEN errors_count:=errors_count+1; IF errors_count<=100 THEN diagnostics_value:=diagnostics_value||jsonb_build_array(jsonb_build_object('code','source_correction_required','record',(r->>'record')::integer,'field','entry_id')); END IF;
    ELSE old_count:=old_count+1; END IF;
   ELSE new_count:=new_count+1; END IF;
  END LOOP;
  IF errors_count>0 THEN rows_value:='[]'::jsonb; new_count:=0; old_count:=0; truncated_value:=errors_count>100; END IF;
 END IF;
 INSERT INTO banking.import_preview(entity_id,book_id,batch_id,preview_revision,source_revision,binding_revision,rows,diagnostics,error_count,truncated,created_count,existing_count)
  VALUES(e,b,id,next_revision,s.source_revision,s.binding_revision,rows_value,diagnostics_value,errors_count,truncated_value,new_count,old_count) RETURNING * INTO p;
 UPDATE banking.import_batch SET publication=v,byte_sha256=v->>'sha256',byte_count=(v->>'byte_count')::integer,state=CASE WHEN errors_count=0 THEN 'preview_ready' ELSE 'rejected' END WHERE entity_id=e AND book_id=b AND batch_id=id RETURNING * INTO x;
 RETURN banking.render_import(x);
EXCEPTION WHEN SQLSTATE 'P2100' OR numeric_value_out_of_range THEN PERFORM banking.import_error('P2400'); END $$;
-- Only the interactive command owner invokes this helper; no direct runtime EXECUTE grant.
CREATE FUNCTION banking.execute_import(actor uuid,e uuid,b uuid,body jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE x banking.import_batch; s banking.source_account; p banking.import_preview; n banking.source_namespace; r jsonb; oid uuid; rid uuid; did uuid; old_fields jsonb; new_count integer:=0; old_count integer:=0; result_value jsonb;
BEGIN
 SELECT * INTO x FROM banking.import_batch WHERE entity_id=e AND book_id=b AND batch_id=(body->>'batch_id')::uuid;
 s:=banking.import_authorize(actor,e,b,x.source_account_id,true);
 PERFORM 1 FROM banking.import_batch WHERE entity_id=e AND book_id=b AND batch_id=x.batch_id FOR UPDATE;
 -- A second command may recover the same immutable batch result without duplicate effects.
 IF x.state='imported' THEN IF x.imported_body<>body THEN PERFORM banking.import_error('P2402'); END IF; RETURN x.imported_result; END IF;
 SELECT * INTO p FROM banking.import_preview WHERE entity_id=e AND book_id=b AND batch_id=x.batch_id ORDER BY preview_revision DESC LIMIT 1;
 IF p.preview_id IS NULL OR x.state<>'preview_ready' OR p.error_count<>0 OR clock_timestamp()>=x.expires_at OR x.raw_deleted_at IS NOT NULL THEN PERFORM banking.import_error('P2402'); END IF;
 SELECT * INTO n FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=x.namespace_id;
 IF s.state<>'active' OR n.state<>'active' OR s.binding_revision<>p.binding_revision OR s.source_revision<>p.source_revision
  OR ROW(p.preview_id,p.preview_revision,p.binding_revision,p.source_revision,x.profile) IS DISTINCT FROM ROW((body->>'preview_id')::uuid,(body->>'preview_revision')::bigint,(body->>'binding_revision')::bigint,(body->>'source_revision')::bigint,body->>'profile') THEN PERFORM banking.import_error('P2403'); END IF;
 IF s.source_revision=9223372036854775807 THEN PERFORM banking.import_error('P2404'); END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(p.rows) LOOP
  SELECT o.observation_id,q.canonical_fields INTO oid,old_fields FROM banking.source_observation o JOIN banking.observation_revision q USING(entity_id,book_id,observation_id)
   WHERE o.entity_id=e AND o.book_id=b AND o.source_account_id=x.source_account_id AND o.namespace_id=x.namespace_id AND o.external_entry_id=r->>'entry_id' AND q.revision=o.current_revision;
  IF FOUND THEN
   IF old_fields<>banking.import_canonical(r) THEN PERFORM banking.import_error('P2403'); END IF; old_count:=old_count+1;
  ELSE
   INSERT INTO banking.source_observation(entity_id,book_id,source_account_id,namespace_id,external_entry_id) VALUES(e,b,x.source_account_id,x.namespace_id,r->>'entry_id') RETURNING observation_id INTO oid;
   INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields)
    VALUES(e,b,oid,1,'booked',(r->>'booked_on')::date,(r->>'currency_version_id')::uuid,r->>'direction',r->>'amount',(r->>'canonical_amount')::numeric,r->>'description',banking.import_canonical(r));
   INSERT INTO banking.review_item(entity_id,book_id,source_account_id,state) VALUES(e,b,x.source_account_id,CASE WHEN n.overlap_review_required THEN 'overlap_hold' ELSE 'needs_review' END) RETURNING review_id INTO rid;
   INSERT INTO banking.review_decision(entity_id,book_id,review_id,expected_review_revision,kind,actor_id,reason,body) VALUES(e,b,rid,0,'initialize',actor,'synthetic_import',jsonb_build_object('batch_id',x.batch_id,'observation_id',oid,'revision','1')) RETURNING decision_id INTO did;
   INSERT INTO banking.review_evidence VALUES(e,b,did,oid,rid,1);
   INSERT INTO banking.review_assignment VALUES(e,b,oid,rid,did);
   new_count:=new_count+1;
  END IF;
  INSERT INTO banking.import_row VALUES(e,b,x.batch_id,(r->>'record')::integer,oid,1,r);
 END LOOP;
 IF ROW(new_count,old_count) IS DISTINCT FROM ROW(p.created_count,p.existing_count) THEN PERFORM banking.import_error('P2403'); END IF;
 UPDATE banking.source_account SET source_revision=source_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=x.source_account_id RETURNING * INTO s;
 result_value:=jsonb_build_object('entity_id',e,'book_id',b,'batch_id',x.batch_id,'created_count',new_count::text,'existing_count',old_count::text,'source_revision',s.source_revision::text);
 UPDATE banking.import_batch SET state='imported',imported_body=body,imported_result=result_value WHERE entity_id=e AND book_id=b AND batch_id=x.batch_id;
 RETURN result_value;
END $$;
-- Scoped review evidence is readable but cannot yet be categorized, drafted or posted.
CREATE FUNCTION banking.get_review_item(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE r banking.review_item; s banking.source_account; result_value jsonb;
BEGIN
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=id;
 s:=banking.import_authorize(actor,e,b,r.source_account_id,false);
 SELECT jsonb_build_object('profile','source-review-v1','review_id',id,'source_account_id',r.source_account_id,'state',r.state,'review_revision',r.review_revision::text,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,
  'draft_id',NULL,'journal_id',NULL,'blockers',CASE WHEN r.state='overlap_hold' THEN jsonb_build_array('overlap_review_required') ELSE '[]'::jsonb END,'created_at',r.created_at,
  'evidence',jsonb_agg(jsonb_build_object('observation_id',o.observation_id,'observation_revision',q.revision::text,'namespace_id',o.namespace_id,'external_entry_id',o.external_entry_id,'source_state',q.source_state,'booked_date',to_char(q.booked_date,'YYYY-MM-DD'),'currency_version_id',q.currency_version_id,'direction',q.direction,'amount',trim_scale(q.amount)::text,'description',q.description) ORDER BY o.observation_id)) INTO result_value
 FROM banking.review_assignment a JOIN banking.source_observation o USING(entity_id,book_id,observation_id) JOIN banking.observation_revision q ON q.entity_id=o.entity_id AND q.book_id=o.book_id AND q.observation_id=o.observation_id AND q.revision=o.current_revision
 WHERE a.entity_id=e AND a.book_id=b AND a.review_id=id;
 RETURN result_value;
END $$;
-- One principal recovery gate now covers setup, post and import.
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_operation_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_operation_check CHECK(operation IN ('setup','post','import'));
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_check CHECK((operation='setup' AND entity_id IS NULL AND book_id IS NULL) OR (operation IN ('post','import') AND entity_id IS NOT NULL AND book_id IS NOT NULL));
CREATE OR REPLACE FUNCTION ledger.validate_command_body(op text,v jsonb) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE a jsonb; code_value text; basis_value text; first_day date; last_day date;
BEGIN
 IF op='setup' THEN
  PERFORM ledger.keys(v,ARRAY['entity_name','entity_kind','book_name','starts_on','ends_before','functional_currency_version_id','accounts']);
  PERFORM ledger.setup_text(v->'entity_name',200); PERFORM ledger.setup_text(v->'book_name',200);
  IF ledger.text_value(v->'entity_kind') NOT IN ('legal','personal') OR jsonb_typeof(v->'accounts') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(v->'accounts') NOT BETWEEN 2 AND 100 THEN PERFORM ledger.fail('P2100'); END IF;
  first_day:=ledger.date_value(ledger.text_value(v->'starts_on')); last_day:=ledger.date_value(ledger.text_value(v->'ends_before'));
  IF first_day>=last_day THEN PERFORM ledger.fail('P2100'); END IF;
  PERFORM ledger.uuid_value(v->'functional_currency_version_id');
  FOR a IN SELECT value FROM jsonb_array_elements(v->'accounts') LOOP
   PERFORM ledger.keys(a,ARRAY['code','name','category','normal_side','monetary']);
   PERFORM ledger.setup_text(a->'code',64); PERFORM ledger.setup_text(a->'name',200);
   IF ledger.text_value(a->'category') NOT IN ('asset','liability','equity','revenue','expense') OR ledger.text_value(a->'normal_side') NOT IN ('debit','credit') OR jsonb_typeof(a->'monetary') IS DISTINCT FROM 'boolean' THEN PERFORM ledger.fail('P2100'); END IF;
  END LOOP;
  IF (SELECT count(DISTINCT value->>'code') FROM jsonb_array_elements(v->'accounts'))<>jsonb_array_length(v->'accounts') THEN PERFORM ledger.fail('P2100'); END IF;
 ELSIF op='post' THEN
  PERFORM ledger.keys(v,ARRAY['expected_revision','policy_id','accounting_date','description','lines']);
  code_value:=ledger.text_value(v->'expected_revision');
  IF code_value !~ '^(0|[1-9][0-9]{0,18})$' OR code_value::numeric>9223372036854775807 THEN PERFORM ledger.fail('P2100'); END IF;
  PERFORM ledger.uuid_value(v->'policy_id'); PERFORM ledger.date_value(ledger.text_value(v->'accounting_date'));
  code_value:=ledger.text_value(v->'description'); IF length(code_value) NOT BETWEEN 1 AND 500 OR jsonb_typeof(v->'lines') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(v->'lines') NOT BETWEEN 2 AND 1000 THEN PERFORM ledger.fail('P2108'); END IF;
  FOR a IN SELECT value FROM jsonb_array_elements(v->'lines') LOOP
   PERFORM ledger.keys(a,ARRAY['account_id','side','transaction_currency','transaction_amount','functional_amount','valuation_basis'],ARRAY['rate_observation_id']);
   PERFORM ledger.uuid_value(a->'account_id'); PERFORM ledger.money_value(a->'transaction_amount'); PERFORM ledger.money_value(a->'functional_amount');
   basis_value:=ledger.text_value(a->'valuation_basis');
   IF ledger.text_value(a->'side') NOT IN ('debit','credit') OR basis_value NOT IN ('identity','spot') OR ledger.text_value(a->'transaction_currency') !~ '^[A-Z]{3}$' THEN PERFORM ledger.fail('P2108'); END IF;
   IF (basis_value='identity' AND a?'rate_observation_id') OR (basis_value='spot' AND NOT a?'rate_observation_id') THEN PERFORM ledger.fail('P2108'); END IF;
   IF basis_value='spot' THEN PERFORM ledger.uuid_value(a->'rate_observation_id'); END IF;
  END LOOP;
 ELSIF op='import' THEN
  PERFORM ledger.keys(v,ARRAY['batch_id','preview_id','preview_revision','binding_revision','source_revision','profile']);
  PERFORM ledger.uuid_value(v->'batch_id'); PERFORM ledger.uuid_value(v->'preview_id');
  IF banking.import_revision(v->'preview_revision')<1 OR banking.import_revision(v->'binding_revision')<1 OR banking.import_revision(v->'source_revision')<0
   OR ledger.text_value(v->'profile')<>'jayburd-synthetic-statement-csv/1' THEN PERFORM ledger.fail('P2100'); END IF;
 ELSE PERFORM ledger.fail('P2100'); END IF;
END $$;
CREATE OR REPLACE FUNCTION ledger.command_authorize(actor uuid,c ledger.interactive_command) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE e uuid; b uuid; x banking.import_batch;
BEGIN
 IF c.principal_id IS DISTINCT FROM actor THEN PERFORM ledger.fail('P2101'); END IF;
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
CREATE OR REPLACE FUNCTION ledger.register_command(actor uuid,k text,request uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; op text; e uuid; b uuid; c ledger.interactive_command;
BEGIN
 IF raw IS NULL OR k IS NULL OR request IS NULL THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF length(k) NOT BETWEEN 1 AND 128 OR k !~ '^[A-Za-z0-9._:-]+$' OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN v:=raw::jsonb; EXCEPTION WHEN untranslatable_character OR numeric_value_out_of_range THEN PERFORM ledger.fail('P2100'); END;
 -- Envelope targets deliberately permit JSON null; ordinary ledger.keys forbids it.
 IF NOT(v ?& ARRAY['operation','entity_id','book_id','body']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['operation','entity_id','book_id','body'])) THEN PERFORM ledger.fail('P2100'); END IF;
 op:=ledger.text_value(v->'operation');
 IF op='setup' THEN IF v->'entity_id'<>'null'::jsonb OR v->'book_id'<>'null'::jsonb THEN PERFORM ledger.fail('P2100'); END IF;
 ELSIF op IN ('post','import') THEN e:=ledger.uuid_value(v->'entity_id'); b:=ledger.uuid_value(v->'book_id');
 ELSE PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.validate_command_body(op,v->'body');
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE principal_id=actor AND registration_key=k;
 IF FOUND THEN
  PERFORM ledger.command_authorize(actor,c);
  IF ROW(c.operation,c.entity_id,c.book_id,c.request_body) IS DISTINCT FROM ROW(op,e,b,v->'body') THEN PERFORM ledger.fail('P2104'); END IF;
  RETURN ledger.render_command(c);
 END IF;
 -- Inaccessible pending work still blocks new intent, without exposing its payload.
 IF EXISTS(SELECT FROM ledger.interactive_command WHERE principal_id=actor AND acknowledged_at IS NULL) THEN PERFORM ledger.fail('P2112'); END IF;
 c.principal_id:=actor; c.operation:=op; c.entity_id:=e; c.book_id:=b; c.request_body:=v->'body';
 PERFORM ledger.command_authorize(actor,c);
 INSERT INTO ledger.interactive_command(command_id,principal_id,registration_key,operation,entity_id,book_id,request_body)
 VALUES(gen_random_uuid(),actor,k,op,e,b,v->'body') RETURNING * INTO c;
 INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(c.command_id,actor,'register',request);
 RETURN ledger.render_command(c);
EXCEPTION WHEN unique_violation THEN
 -- A stale serializable snapshot after waiting on the unchanged principal can
 -- discover the competing insert at its unique index; retry with a fresh snapshot.
 RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_command_registration';
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
   IF c.operation='import' THEN value:=banking.execute_import(actor,c.entity_id,c.book_id,c.request_body);
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
-- The batch identity and published evidence cannot be replaced or deleted.
CREATE FUNCTION banking.import_batch_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM banking.import_error('P2402'); END IF;
 IF (to_jsonb(NEW)-ARRAY['publication','byte_sha256','byte_count','state','imported_result','imported_body']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['publication','byte_sha256','byte_count','state','imported_result','imported_body'])
 OR OLD.publication IS NOT NULL AND ROW(NEW.publication,NEW.byte_sha256,NEW.byte_count) IS DISTINCT FROM ROW(OLD.publication,OLD.byte_sha256,OLD.byte_count)
 OR OLD.state IN ('rejected','imported','cancelled')
 OR NEW.state NOT IN ('preview_ready','rejected','imported','cancelled')
 OR NEW.state='imported' AND OLD.state<>'preview_ready' THEN PERFORM banking.import_error('P2402'); END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_batch BEFORE UPDATE OR DELETE ON banking.import_batch FOR EACH ROW EXECUTE FUNCTION banking.import_batch_guard();
REVOKE ALL ON banking.import_batch,banking.import_preview,banking.source_observation,banking.observation_revision,banking.import_row,banking.review_item,banking.review_decision,banking.review_evidence,banking.review_assignment,integration.import_registration FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION banking.import_error(text),banking.import_json(text,integer),banking.import_revision(jsonb),banking.set_source_import_permission(uuid,uuid,uuid,uuid,text,text),banking.import_authorize(uuid,uuid,uuid,uuid,boolean),banking.render_import(banking.import_batch),banking.import_canonical(jsonb),banking.validate_import_rows(jsonb,banking.import_batch,banking.source_account,banking.source_namespace),banking.execute_import(uuid,uuid,uuid,jsonb),banking.import_batch_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION banking.register_import_batch(uuid,uuid,uuid,text,text),banking.import_context(uuid,uuid,uuid,uuid),banking.publish_import_preview(uuid,uuid,uuid,uuid,text),banking.get_import_batch(uuid,uuid,uuid,uuid),banking.get_review_item(uuid,uuid,uuid,uuid) TO bookkeeping_api;
COMMIT;
