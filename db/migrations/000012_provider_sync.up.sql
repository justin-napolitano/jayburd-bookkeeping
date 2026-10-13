-- Synthetic provider sync state; no network, credentials, scheduler or posting authority.
BEGIN;
DO $$ BEGIN
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_sync') THEN CREATE ROLE bookkeeping_sync NOLOGIN; END IF;
 IF EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_sync' AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolcanlogin))
 OR pg_has_role('bookkeeping_sync','bookkeeping_owner','MEMBER') OR pg_has_role('bookkeeping_sync','bookkeeping_api','MEMBER') OR pg_has_role('bookkeeping_sync','bookkeeping_observer','MEMBER') OR pg_has_role('bookkeeping_sync','bookkeeping_publisher','MEMBER') THEN RAISE EXCEPTION 'unsafe sync role'; END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
ALTER TABLE banking.source_namespace ALTER COLUMN synthetic_dataset_id DROP NOT NULL;
ALTER TABLE banking.source_namespace ALTER COLUMN source_account_ref DROP NOT NULL;
ALTER TABLE banking.source_namespace DROP CONSTRAINT source_namespace_profile_check;
ALTER TABLE banking.source_namespace ADD COLUMN provider text, ADD COLUMN environment text, ADD COLUMN item_id text COLLATE "C", ADD COLUMN provider_account_id text COLLATE "C";
ALTER TABLE banking.source_namespace ADD CONSTRAINT source_namespace_profile_check CHECK(
 (profile='jayburd-synthetic-statement-csv/1' AND synthetic_dataset_id IS NOT NULL AND source_account_ref IS NOT NULL AND provider IS NULL AND environment IS NULL AND item_id IS NULL AND provider_account_id IS NULL)
 OR (profile='jayburd-plaid-transactions/1' AND synthetic_dataset_id IS NULL AND source_account_ref IS NULL AND provider IS NOT NULL AND environment IS NOT NULL AND provider='plaid' AND environment='sandbox' AND item_id IS NOT NULL AND provider_account_id IS NOT NULL));
CREATE UNIQUE INDEX source_namespace_provider_identity ON banking.source_namespace(provider,environment,item_id,provider_account_id) WHERE provider IS NOT NULL;
ALTER TABLE banking.source_observation DROP CONSTRAINT source_observation_current_revision_check;
ALTER TABLE banking.source_observation ADD CONSTRAINT source_observation_current_revision_check CHECK(current_revision>0);
ALTER TABLE banking.source_observation DROP CONSTRAINT source_observation_external_entry_id_check;
ALTER TABLE banking.source_observation ADD CONSTRAINT source_observation_external_entry_id_check CHECK(length(external_entry_id) BETWEEN 1 AND 256 AND external_entry_id COLLATE "C" ~ '^[!-~]+$');
ALTER TABLE banking.observation_revision DROP CONSTRAINT observation_revision_revision_check;
ALTER TABLE banking.observation_revision ADD CONSTRAINT observation_revision_revision_check CHECK(revision>0);
ALTER TABLE banking.observation_revision DROP CONSTRAINT observation_revision_source_state_check;
ALTER TABLE banking.observation_revision ADD CONSTRAINT observation_revision_source_state_check CHECK(source_state IN ('booked','pending','removed'));
ALTER TABLE banking.observation_revision DROP CONSTRAINT observation_revision_direction_check;
ALTER TABLE banking.observation_revision ADD CONSTRAINT observation_revision_direction_check CHECK(direction IN ('inflow','outflow','zero'));
ALTER TABLE banking.observation_revision DROP CONSTRAINT observation_revision_amount_check;
ALTER TABLE banking.observation_revision ADD COLUMN previous_revision bigint, ADD COLUMN adapter_profile text NOT NULL DEFAULT 'jayburd-synthetic-statement-csv/1', ADD COLUMN provider_amount_lexeme text, ADD COLUMN provider_fields jsonb;
ALTER TABLE banking.observation_revision ADD FOREIGN KEY(entity_id,book_id,observation_id,previous_revision) REFERENCES banking.observation_revision(entity_id,book_id,observation_id,revision);
ALTER TABLE banking.observation_revision ADD CONSTRAINT observation_profile CHECK(
 (adapter_profile='jayburd-synthetic-statement-csv/1' AND revision=1 AND previous_revision IS NULL AND source_state='booked' AND amount>0 AND direction<>'zero' AND provider_amount_lexeme IS NULL AND provider_fields IS NULL)
 OR (adapter_profile='jayburd-plaid-transactions/1' AND (revision=1 AND previous_revision IS NULL OR previous_revision=revision-1) AND provider_amount_lexeme IS NOT NULL AND length(provider_amount_lexeme)<=128 AND provider_fields IS NOT NULL AND octet_length(provider_fields::text)<=8192 AND ((amount=0)=(direction='zero'))));
-- Provider publication alone advances current pointers; identities and all historical rows remain immutable.
CREATE FUNCTION banking.observation_pointer_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM ledger.fail('P2108'); END IF;
 IF (to_jsonb(NEW)-'current_revision') IS DISTINCT FROM (to_jsonb(OLD)-'current_revision') OR NEW.current_revision::numeric<>OLD.current_revision::numeric+1 THEN PERFORM ledger.fail('P2108'); END IF;
 RETURN NEW;
END $$;
DROP TRIGGER immutable ON banking.source_observation;
CREATE TRIGGER preserve_observation BEFORE UPDATE OR DELETE ON banking.source_observation FOR EACH ROW EXECUTE FUNCTION banking.observation_pointer_guard();
ALTER TABLE banking.review_item DROP CONSTRAINT review_item_state_check;
ALTER TABLE banking.review_item ADD CONSTRAINT review_item_state_check CHECK(state IN ('needs_review','overlap_hold','draft_ready','posted','correction_hold'));
ALTER TABLE banking.review_decision DROP CONSTRAINT review_decision_kind_check;
ALTER TABLE banking.review_decision ADD CONSTRAINT review_decision_kind_check CHECK(kind IN ('initialize','draft','source_update'));
ALTER TABLE banking.review_decision DROP CONSTRAINT review_decision_reason_check;
ALTER TABLE banking.review_decision ADD CONSTRAINT review_decision_reason_check CHECK(reason IN ('synthetic_import','explicit_category','provider_sync'));
CREATE OR REPLACE FUNCTION banking.review_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM banking.review_error('P2502'); END IF;
 IF (to_jsonb(NEW)-ARRAY['state','review_revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','review_revision']) OR NEW.review_revision::numeric<>OLD.review_revision::numeric+1
 OR NOT (OLD.state IN ('needs_review','draft_ready') AND NEW.state='draft_ready' OR OLD.state='draft_ready' AND NEW.state='posted'
 OR OLD.state<>'posted' AND NEW.state='correction_hold' OR OLD.state='posted' AND NEW.state='posted') THEN PERFORM banking.review_error('P2502'); END IF;
 RETURN NEW;
END $$;
CREATE TABLE integration.sync_item (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, sync_item_id uuid NOT NULL DEFAULT gen_random_uuid(), provider text NOT NULL CHECK(provider='plaid'), environment text NOT NULL CHECK(environment='sandbox'), item_id text COLLATE "C" NOT NULL CHECK(length(item_id) BETWEEN 1 AND 256 AND item_id COLLATE "C" ~ '^[!-~]+$'),
 credential_ref uuid NOT NULL, credential_version bigint NOT NULL CHECK(credential_version>0), consent_principal uuid NOT NULL REFERENCES ledger.principal, consent_evidence_ref uuid NOT NULL, consent_expires_at timestamptz NOT NULL CHECK(isfinite(consent_expires_at)),
 generation bigint NOT NULL DEFAULT 1 CHECK(generation>0), state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','paused','disconnected')), synthetic boolean NOT NULL CHECK(synthetic),
 PRIMARY KEY(entity_id,book_id,sync_item_id), UNIQUE(provider,environment,item_id), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
CREATE TABLE integration.sync_stream (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, stream_id uuid NOT NULL DEFAULT gen_random_uuid(), sync_item_id uuid NOT NULL, source_account_id uuid NOT NULL, namespace_id uuid NOT NULL,
 account_id text COLLATE "C" NOT NULL CHECK(length(account_id) BETWEEN 1 AND 256 AND account_id COLLATE "C" ~ '^[!-~]+$'), worker_login name NOT NULL, service_principal uuid NOT NULL REFERENCES ledger.principal,
 requested_generation bigint NOT NULL DEFAULT 1 CHECK(requested_generation>0), completed_generation bigint NOT NULL DEFAULT 0 CHECK(completed_generation>=0 AND completed_generation<=requested_generation),
 cursor_value text CHECK(length(cursor_value)<=256), fence bigint NOT NULL DEFAULT 0 CHECK(fence>=0), lease_until timestamptz, current_set uuid, last_completed_at timestamptz, readiness text NOT NULL DEFAULT 'NOT_READY' CHECK(readiness IN ('TRANSACTIONS_UPDATE_STATUS_UNKNOWN','NOT_READY','INITIAL_UPDATE_COMPLETE','HISTORICAL_UPDATE_COMPLETE')),
 PRIMARY KEY(entity_id,book_id,stream_id), UNIQUE(entity_id,book_id,sync_item_id,account_id), UNIQUE(entity_id,book_id,namespace_id),
 FOREIGN KEY(entity_id,book_id,sync_item_id) REFERENCES integration.sync_item, FOREIGN KEY(entity_id,book_id,source_account_id,namespace_id) REFERENCES banking.source_namespace(entity_id,book_id,source_account_id,namespace_id), CHECK((lease_until IS NULL)=(current_set IS NULL))
);
CREATE TABLE integration.sync_set (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, set_id uuid NOT NULL DEFAULT gen_random_uuid(), stream_id uuid NOT NULL, claim_key uuid NOT NULL,
 item_generation bigint NOT NULL CHECK(item_generation>0), binding_revision bigint NOT NULL CHECK(binding_revision>0), target_generation bigint NOT NULL CHECK(target_generation>0), fence bigint NOT NULL CHECK(fence>0), starting_cursor text, continuation_cursor text,
 state text NOT NULL DEFAULT 'staging' CHECK(state IN ('staging','completed','abandoned','quarantined')), reason text,
 page_count integer NOT NULL DEFAULT 0 CHECK(page_count BETWEEN 0 AND 10), change_count integer NOT NULL DEFAULT 0 CHECK(change_count BETWEEN 0 AND 1000), byte_count integer NOT NULL DEFAULT 0 CHECK(byte_count BETWEEN 0 AND 2097152), final_page boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '24 hours',
 PRIMARY KEY(entity_id,book_id,set_id), UNIQUE(entity_id,book_id,stream_id,claim_key), UNIQUE(entity_id,book_id,stream_id,set_id), FOREIGN KEY(entity_id,book_id,stream_id) REFERENCES integration.sync_stream
);
ALTER TABLE integration.sync_stream ADD FOREIGN KEY(entity_id,book_id,stream_id,current_set) REFERENCES integration.sync_set(entity_id,book_id,stream_id,set_id) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE integration.sync_page (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, set_id uuid NOT NULL, page_index integer NOT NULL CHECK(page_index BETWEEN 0 AND 9), body jsonb NOT NULL CHECK(octet_length(body::text)<=2097152),
 PRIMARY KEY(entity_id,book_id,set_id,page_index), FOREIGN KEY(entity_id,book_id,set_id) REFERENCES integration.sync_set
);
CREATE TABLE integration.sync_completion (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, set_id uuid NOT NULL, fence bigint NOT NULL, result jsonb NOT NULL,
 PRIMARY KEY(entity_id,book_id,set_id), FOREIGN KEY(entity_id,book_id,set_id) REFERENCES integration.sync_set
);
CREATE TABLE integration.sync_signal (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, stream_id uuid NOT NULL, idempotency_key text NOT NULL, requested_generation bigint NOT NULL,
 PRIMARY KEY(entity_id,book_id,stream_id,idempotency_key), FOREIGN KEY(entity_id,book_id,stream_id) REFERENCES integration.sync_stream
);
CREATE TABLE integration.sync_operator_receipt (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, operator_login name NOT NULL, operation text NOT NULL CHECK(operation IN ('enroll','item_state')), idempotency_key text NOT NULL,
 approved_by uuid NOT NULL REFERENCES ledger.principal, body jsonb NOT NULL, result jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,operator_login,operation,idempotency_key), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
CREATE TABLE banking.provider_revision_origin (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, observation_id uuid NOT NULL, revision bigint NOT NULL, set_id uuid NOT NULL, page_index integer NOT NULL, operation text NOT NULL CHECK(operation IN ('added','modified','removed')),
 PRIMARY KEY(entity_id,book_id,observation_id,revision), FOREIGN KEY(entity_id,book_id,observation_id,revision) REFERENCES banking.observation_revision, FOREIGN KEY(entity_id,book_id,set_id,page_index) REFERENCES integration.sync_page
);
CREATE TABLE banking.provider_pending_link (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, pending_observation_id uuid NOT NULL, booked_observation_id uuid NOT NULL, set_id uuid NOT NULL,
 PRIMARY KEY(entity_id,book_id,pending_observation_id), UNIQUE(entity_id,book_id,booked_observation_id), FOREIGN KEY(entity_id,book_id,pending_observation_id) REFERENCES banking.source_observation, FOREIGN KEY(entity_id,book_id,booked_observation_id) REFERENCES banking.source_observation, FOREIGN KEY(entity_id,book_id,set_id) REFERENCES integration.sync_set, CHECK(pending_observation_id<>booked_observation_id)
);
-- Immutable direct reference keeps original posted evidence stable across any number of corrections.
CREATE TABLE banking.provider_posted_anchor (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, review_id uuid NOT NULL, observation_id uuid NOT NULL, revision bigint NOT NULL, journal_id uuid NOT NULL,
 PRIMARY KEY(entity_id,book_id,review_id), FOREIGN KEY(entity_id,book_id,review_id) REFERENCES banking.review_posting, FOREIGN KEY(entity_id,book_id,observation_id,revision) REFERENCES banking.observation_revision, FOREIGN KEY(entity_id,book_id,journal_id) REFERENCES ledger.journal_entry
);
CREATE TABLE banking.provider_review_exception (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, exception_id uuid NOT NULL DEFAULT gen_random_uuid(), review_id uuid NOT NULL, observation_id uuid NOT NULL, previous_revision bigint NOT NULL, current_revision bigint NOT NULL, set_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('modified','removed')), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,exception_id), UNIQUE(entity_id,book_id,observation_id,current_revision), FOREIGN KEY(entity_id,book_id,review_id) REFERENCES banking.provider_posted_anchor,
 FOREIGN KEY(entity_id,book_id,observation_id,previous_revision) REFERENCES banking.observation_revision, FOREIGN KEY(entity_id,book_id,observation_id,current_revision) REFERENCES banking.observation_revision, FOREIGN KEY(entity_id,book_id,set_id) REFERENCES integration.sync_set
);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['sync_page','sync_completion','sync_signal','sync_operator_receipt'] LOOP EXECUTE format('CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.%I FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row()',t); END LOOP;
 FOREACH t IN ARRAY ARRAY['provider_revision_origin','provider_pending_link','provider_posted_anchor','provider_review_exception'] LOOP EXECUTE format('CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.%I FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row()',t); END LOOP;
END $$;
-- Fixed diagnostics and validated JSON keep source data out of database errors.
CREATE FUNCTION integration.sync_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='sync_rejected'; END $$;
CREATE FUNCTION integration.sync_text(v jsonb,max_length integer) RETURNS text LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE x text; BEGIN x:=ledger.text_value(v); IF length(x) NOT BETWEEN 1 AND max_length OR x COLLATE "C" !~ '^[!-~]+$' THEN PERFORM integration.sync_error('P2600'); END IF; RETURN x; EXCEPTION WHEN SQLSTATE 'P2100' THEN PERFORM integration.sync_error('P2600'); END $$;
-- Lock all authority principals first; missing/revoked consent and service grants fail uniformly.
CREATE FUNCTION integration.sync_authority(e uuid,b uuid,sid uuid,consent uuid,service uuid) RETURNS banking.source_account LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE id uuid; permission_name text; s banking.source_account;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.sync_error('P2600'); END IF;
 PERFORM set_config('lock_timeout','1s',true);
 FOR id IN SELECT DISTINCT x FROM unnest(ARRAY[consent,service]) x ORDER BY x LOOP
  PERFORM 1 FROM ledger.principal WHERE principal_id=id AND disabled_at IS NULL FOR SHARE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
 END LOOP;
 PERFORM 1 FROM ledger.entity WHERE entity_id=e AND disabled_at IS NULL FOR SHARE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
 FOR id IN SELECT DISTINCT x FROM unnest(ARRAY[consent,service]) x ORDER BY x LOOP
  FOREACH permission_name IN ARRAY ARRAY['read','import'] LOOP
   PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=id AND permission=permission_name AND revoked_at IS NULL FOR SHARE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
  END LOOP;
 END LOOP;
 PERFORM 1 FROM ledger.book WHERE entity_id=e AND book_id=b AND disabled_at IS NULL FOR UPDATE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR UPDATE; IF NOT FOUND OR s.state<>'active' THEN PERFORM integration.sync_error('P2601'); END IF;
 FOR id IN SELECT DISTINCT x FROM unnest(ARRAY[consent,service]) x ORDER BY x LOOP
  FOREACH permission_name IN ARRAY ARRAY['read','import'] LOOP
   PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=id AND capability=permission_name AND revoked_at IS NULL FOR SHARE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
  END LOOP;
 END LOOP;
 RETURN s;
END $$;
-- Session identity is fixed by enrollment, never supplied as an actor parameter by a worker.
CREATE FUNCTION integration.sync_authorize(e uuid,b uuid,id uuid) RETURNS integration.sync_stream LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream; i integration.sync_item;
BEGIN
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id;
 IF NOT FOUND OR t.worker_login<>session_user OR NOT integration.sync_login(session_user) THEN PERFORM integration.sync_error('P2601'); END IF;
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 PERFORM integration.sync_authority(e,b,t.source_account_id,i.consent_principal,t.service_principal);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id FOR UPDATE;
 IF i.state<>'active' OR clock_timestamp()>=i.consent_expires_at THEN PERFORM integration.sync_error('P2601'); END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id FOR UPDATE;
 PERFORM 1 FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=t.namespace_id AND state='active'; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
 RETURN t;
END $$;
-- Reject accidental mixed-role provisioning at the worker boundary.
CREATE FUNCTION integration.sync_login(login_name name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_name AND rolcanlogin AND NOT(rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication))
 AND NOT EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolname='cloudsqlsuperuser') AND pg_has_role(login_name,oid,'MEMBER'))
 AND pg_has_role(login_name,'bookkeeping_sync','MEMBER') AND NOT EXISTS(SELECT FROM pg_roles WHERE rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_observer','bookkeeping_publisher','bookkeeping_identity_operator','bookkeeping_migrator') AND pg_has_role(login_name,oid,'MEMBER'))
$$;
CREATE FUNCTION integration.enroll_sync_stream(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb; s banking.source_account; i integration.sync_item; t integration.sync_stream; rec integration.sync_operator_receipt; sid uuid; consent uuid; service uuid; item_value text; account_value text; login_value text; ns uuid; result_value jsonb; norm jsonb; id uuid; expiry timestamptz;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='sync_rejected'; END IF;
 v:=banking.import_json(raw,8192); PERFORM ledger.keys(v,ARRAY['source_account_id','expected_binding_revision','provider','environment','item_id','account_id','credential_ref','credential_version','consent_principal_id','consent_evidence_ref','consent_expires_at','service_principal_id','worker_login','synthetic']);
 IF k IS NULL OR k !~ '^[A-Za-z0-9._:-]{1,128}$' OR v->'provider' IS DISTINCT FROM '"plaid"'::jsonb OR v->'environment' IS DISTINCT FROM '"sandbox"'::jsonb OR v->'synthetic' IS DISTINCT FROM 'true'::jsonb THEN PERFORM integration.sync_error('P2600'); END IF;
 sid:=ledger.uuid_value(v->'source_account_id'); consent:=ledger.uuid_value(v->'consent_principal_id'); service:=ledger.uuid_value(v->'service_principal_id'); item_value:=integration.sync_text(v->'item_id',256); account_value:=integration.sync_text(v->'account_id',256); login_value:=integration.sync_text(v->'worker_login',63);
 IF NOT integration.sync_login(login_value::name) THEN PERFORM integration.sync_error('P2601'); END IF;
 PERFORM ledger.uuid_value(v->'credential_ref'); PERFORM ledger.uuid_value(v->'consent_evidence_ref');
 IF banking.import_revision(v->'credential_version')=0 OR v->>'consent_expires_at' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' THEN PERFORM integration.sync_error('P2600'); END IF;
 expiry:=(v->>'consent_expires_at')::timestamptz; IF expiry<=clock_timestamp() OR NOT isfinite(expiry) THEN PERFORM integration.sync_error('P2601'); END IF;
 -- All principal locks precede the shared book lock, including the approving human.
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.sync_error('P2600'); END IF;
 FOR id IN SELECT DISTINCT x FROM unnest(ARRAY[actor,consent,service]) x ORDER BY x LOOP PERFORM 1 FROM ledger.principal WHERE principal_id=id AND disabled_at IS NULL FOR SHARE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF; END LOOP;
 PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='read' AND revoked_at IS NULL FOR SHARE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
 s:=integration.sync_authority(e,b,sid,consent,service);
 norm:=jsonb_build_object('approved_by',actor,'body',v);
 SELECT * INTO rec FROM integration.sync_operator_receipt WHERE entity_id=e AND book_id=b AND operator_login=session_user AND operation='enroll' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>norm THEN PERFORM integration.sync_error('P2602'); END IF; RETURN rec.result; END IF;
 IF s.binding_revision<>banking.import_revision(v->'expected_binding_revision') OR s.binding_revision=9223372036854775807 THEN PERFORM integration.sync_error('P2603'); END IF;
 SELECT * INTO i FROM integration.sync_item WHERE provider='plaid' AND environment='sandbox' AND item_id=item_value FOR UPDATE;
 IF FOUND THEN
  IF ROW(i.entity_id,i.book_id,i.credential_ref,i.credential_version,i.consent_principal,i.consent_evidence_ref,i.consent_expires_at,i.state) IS DISTINCT FROM ROW(e,b,ledger.uuid_value(v->'credential_ref'),banking.import_revision(v->'credential_version'),consent,ledger.uuid_value(v->'consent_evidence_ref'),expiry,'active'::text) THEN PERFORM integration.sync_error('P2602'); END IF;
 ELSE INSERT INTO integration.sync_item(entity_id,book_id,provider,environment,item_id,credential_ref,credential_version,consent_principal,consent_evidence_ref,consent_expires_at,synthetic) VALUES(e,b,'plaid','sandbox',item_value,ledger.uuid_value(v->'credential_ref'),banking.import_revision(v->'credential_version'),consent,ledger.uuid_value(v->'consent_evidence_ref'),expiry,true) RETURNING * INTO i; END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id AND account_id=account_value;
 IF FOUND THEN PERFORM integration.sync_error('P2602'); END IF;
 IF (SELECT count(*) FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND source_account_id=sid)>=25 THEN PERFORM integration.sync_error('P2605'); END IF;
 INSERT INTO banking.source_namespace(entity_id,book_id,source_account_id,profile,provider,environment,item_id,provider_account_id,overlap_review_required,created_by) VALUES(e,b,sid,'jayburd-plaid-transactions/1','plaid','sandbox',item_value,account_value,true,actor) RETURNING namespace_id INTO ns;
 UPDATE banking.source_account SET binding_revision=binding_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=sid RETURNING * INTO s;
 INSERT INTO integration.sync_stream(entity_id,book_id,sync_item_id,source_account_id,namespace_id,account_id,worker_login,service_principal) VALUES(e,b,i.sync_item_id,sid,ns,account_value,login_value::name,service) RETURNING * INTO t;
 result_value:=jsonb_build_object('stream_id',t.stream_id,'sync_item_id',i.sync_item_id,'namespace_id',ns,'source_account_id',sid,'binding_revision',s.binding_revision::text,'item_generation',i.generation::text);
 INSERT INTO integration.sync_operator_receipt VALUES(e,b,session_user,'enroll',k,actor,norm,result_value,clock_timestamp()); RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range THEN PERFORM integration.sync_error('P2600'); WHEN unique_violation THEN PERFORM integration.sync_error('P2602'); END $$;
-- State transitions fence all sibling account streams; disconnect is terminal in this profile.
CREATE FUNCTION integration.set_sync_item_state(actor uuid,e uuid,b uuid,id uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb; i integration.sync_item; p uuid; rec integration.sync_operator_receipt; result_value jsonb; norm jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='sync_rejected'; END IF;
 v:=banking.import_json(raw,2048); PERFORM ledger.keys(v,ARRAY['expected_generation','state']);
 IF k IS NULL OR k !~ '^[A-Za-z0-9._:-]{1,128}$' OR v->>'state' NOT IN ('active','paused','disconnected') THEN PERFORM integration.sync_error('P2600'); END IF;
 -- Revocation must work even after consent/service principals are disabled.
 PERFORM banking.source_operator_authorize(actor,actor,e,b,true);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=id FOR UPDATE; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
 norm:=jsonb_build_object('approved_by',actor,'sync_item_id',id,'body',v);
 SELECT * INTO rec FROM integration.sync_operator_receipt WHERE entity_id=e AND book_id=b AND operator_login=session_user AND operation='item_state' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>norm THEN PERFORM integration.sync_error('P2602'); END IF; RETURN rec.result; END IF;
 IF i.generation<>banking.import_revision(v->'expected_generation') OR i.generation=9223372036854775807 OR i.state='disconnected' THEN PERFORM integration.sync_error('P2603'); END IF;
 UPDATE integration.sync_item SET state=v->>'state',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=id RETURNING * INTO i;
 UPDATE integration.sync_set x SET state='abandoned',reason='item_changed' FROM integration.sync_stream t WHERE t.entity_id=e AND t.book_id=b AND t.sync_item_id=id AND x.entity_id=e AND x.book_id=b AND x.set_id=t.current_set AND x.state='staging';
 UPDATE integration.sync_stream SET current_set=NULL,lease_until=NULL WHERE entity_id=e AND book_id=b AND sync_item_id=id;
 result_value:=jsonb_build_object('sync_item_id',id,'item_generation',i.generation::text,'state',i.state);
 INSERT INTO integration.sync_operator_receipt VALUES(e,b,session_user,'item_state',k,actor,norm,result_value,clock_timestamp()); RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM integration.sync_error('P2600'); END $$;
-- Complete transaction caller retries serialization; a signal key always names one increment.
CREATE FUNCTION integration.request_sync(e uuid,b uuid,id uuid,k text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream; n bigint;
BEGIN
 t:=integration.sync_authorize(e,b,id);
 IF k IS NULL OR k !~ '^[A-Za-z0-9._:-]{1,128}$' THEN PERFORM integration.sync_error('P2600'); END IF;
 SELECT requested_generation INTO n FROM integration.sync_signal WHERE entity_id=e AND book_id=b AND stream_id=id AND idempotency_key=k;
 IF NOT FOUND THEN
  IF t.requested_generation=9223372036854775807 THEN PERFORM integration.sync_error('P2605'); END IF;
  UPDATE integration.sync_stream SET requested_generation=requested_generation+1 WHERE entity_id=e AND book_id=b AND stream_id=id RETURNING requested_generation INTO n;
  INSERT INTO integration.sync_signal VALUES(e,b,id,k,n);
 END IF;
 RETURN jsonb_build_object('stream_id',id,'requested_generation',n::text);
END $$;
CREATE FUNCTION integration.render_sync_claim(t integration.sync_stream,x integration.sync_set) RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT jsonb_build_object('stream_id',t.stream_id,'set_id',x.set_id,'fence',x.fence::text,'starting_cursor',x.starting_cursor,'next_page_index',x.page_count,'request_cursor',x.continuation_cursor,'target_generation',x.target_generation::text,'binding_revision',x.binding_revision::text,'lease_until',t.lease_until)
$$;
-- No page adoption: an expired claim restarts from the last fully committed cursor.
CREATE FUNCTION integration.claim_sync(e uuid,b uuid,id uuid,k uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream; x integration.sync_set; g bigint; br bigint;
BEGIN
 t:=integration.sync_authorize(e,b,id); IF k IS NULL OR k='00000000-0000-0000-0000-000000000000' THEN PERFORM integration.sync_error('P2600'); END IF;
 SELECT * INTO x FROM integration.sync_set WHERE entity_id=e AND book_id=b AND stream_id=id AND claim_key=k;
 IF FOUND THEN
  IF t.current_set=x.set_id AND t.lease_until>clock_timestamp() AND x.state='staging' THEN x:=integration.sync_attempt(t,x.set_id,x.fence); RETURN integration.render_sync_claim(t,x); END IF; PERFORM integration.sync_error('P2603');
 END IF;
 IF t.current_set IS NOT NULL AND t.lease_until>clock_timestamp() THEN PERFORM integration.sync_error('P2603'); END IF;
 IF t.fence=9223372036854775807 THEN PERFORM integration.sync_error('P2605'); END IF;
 UPDATE integration.sync_set SET state='abandoned',reason='lease_expired' WHERE entity_id=e AND book_id=b AND set_id=t.current_set AND state='staging';
 SELECT generation INTO g FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 SELECT binding_revision INTO br FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id;
 INSERT INTO integration.sync_set(entity_id,book_id,stream_id,claim_key,item_generation,binding_revision,target_generation,fence,starting_cursor,continuation_cursor) VALUES(e,b,id,k,g,br,t.requested_generation,t.fence+1,t.cursor_value,t.cursor_value) RETURNING * INTO x;
 UPDATE integration.sync_stream SET fence=x.fence,current_set=x.set_id,lease_until=clock_timestamp()+interval '90 seconds' WHERE entity_id=e AND book_id=b AND stream_id=id RETURNING * INTO t;
 RETURN integration.render_sync_claim(t,x);
END $$;
-- Stage/publication share one fence and binding validator after current authority locks.
CREATE FUNCTION integration.sync_attempt(t integration.sync_stream,id uuid,f bigint) RETURNS integration.sync_set LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE x integration.sync_set; br bigint; g bigint;
BEGIN
 SELECT * INTO x FROM integration.sync_set WHERE entity_id=t.entity_id AND book_id=t.book_id AND stream_id=t.stream_id AND set_id=id FOR UPDATE;
 SELECT binding_revision INTO br FROM banking.source_account WHERE entity_id=t.entity_id AND book_id=t.book_id AND source_account_id=t.source_account_id;
 SELECT generation INTO g FROM integration.sync_item WHERE entity_id=t.entity_id AND book_id=t.book_id AND sync_item_id=t.sync_item_id;
 IF x.set_id IS NULL OR t.current_set IS DISTINCT FROM id OR f IS DISTINCT FROM t.fence OR x.fence<>f OR x.state<>'staging' OR t.lease_until<=clock_timestamp() OR x.expires_at<=clock_timestamp() OR x.item_generation<>g OR x.binding_revision<>br OR x.starting_cursor IS DISTINCT FROM t.cursor_value THEN PERFORM integration.sync_error('P2603'); END IF;
 RETURN x;
END $$;
-- Validate each trusted-decoder row independently before accepting private staged evidence.
CREATE FUNCTION integration.sync_row(v jsonb,account_value text,currency_id uuid,removed boolean) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE c reference.currency; d date; a numeric; signed_value numeric; lex text; field_name text; text_value text; pending_value boolean; exp_value numeric; coeff text;
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
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=currency_id;
 IF ledger.uuid_value(v->'currency_version_id')<>currency_id OR ledger.text_value(v->'iso_currency_code')<>c.alphabetic_code OR c.minor_unit IS NULL OR d<c.valid_from OR c.valid_to IS NOT NULL AND d>=c.valid_to THEN PERFORM integration.sync_error('P2604'); END IF;
 text_value:=ledger.text_value(v->'amount'); IF length(text_value)>39 OR text_value !~ '^(0|[1-9][0-9]{0,28})(\.[0-9]{0,8}[1-9])?$' THEN PERFORM integration.sync_error('P2600'); END IF;
 a:=text_value::numeric; IF scale(a)>c.minor_unit THEN PERFORM integration.sync_error('P2604'); END IF;
 lex:=ledger.text_value(v->'amount_lexeme'); IF length(lex)>128 OR lex !~ '^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$' THEN PERFORM integration.sync_error('P2600'); END IF;
 coeff:=regexp_replace(split_part(lower(lex),'e',1),'[-.]','','g'); IF length(trim(both '0' FROM coeff))>38 THEN PERFORM integration.sync_error('P2600'); END IF;
 IF lower(lex) LIKE '%e%' THEN exp_value:=split_part(lower(lex),'e',2)::numeric; IF exp_value NOT BETWEEN -100 AND 100 THEN PERFORM integration.sync_error('P2600'); END IF; END IF;
 signed_value:=lex::numeric;
 IF abs(signed_value)<>a OR ledger.text_value(v->'direction')<>(CASE WHEN signed_value<0 THEN 'inflow' WHEN signed_value>0 THEN 'outflow' ELSE 'zero' END) OR (v->>'eligible_for_review')::boolean<>(NOT pending_value AND a>0) THEN PERFORM integration.sync_error('P2604'); END IF;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR datetime_field_overflow OR numeric_value_out_of_range THEN PERFORM integration.sync_error('P2600'); END $$;
-- Pages are private until an entire contiguous cursor chain commits atomically.
CREATE FUNCTION integration.stage_sync_page(e uuid,b uuid,id uuid,set_value uuid,f bigint,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream; x integration.sync_set; v jsonb; old jsonb; row_value jsonb; field_name text; idx integer; count_value integer:=0; bytes_value integer; currency_id uuid;
BEGIN
 t:=integration.sync_authorize(e,b,id); x:=integration.sync_attempt(t,set_value,f); v:=banking.import_json(raw,2097152);
 IF NOT(v ?& ARRAY['page_index','request_cursor','next_cursor','has_more','transactions_update_status','request_id','sha256','added','modified','removed']) OR EXISTS(SELECT FROM jsonb_object_keys(v) z WHERE z<>ALL(ARRAY['page_index','request_cursor','next_cursor','has_more','transactions_update_status','request_id','sha256','added','modified','removed'])) THEN PERFORM integration.sync_error('P2600'); END IF;
 IF jsonb_typeof(v->'page_index')<>'number' OR v->>'page_index' !~ '^[0-9]$' OR jsonb_typeof(v->'has_more')<>'boolean' OR jsonb_typeof(v->'transactions_update_status') IS DISTINCT FROM 'string' OR v->>'transactions_update_status' NOT IN ('TRANSACTIONS_UPDATE_STATUS_UNKNOWN','NOT_READY','INITIAL_UPDATE_COMPLETE','HISTORICAL_UPDATE_COMPLETE') THEN PERFORM integration.sync_error('P2600'); END IF;
 IF ledger.text_value(v->'next_cursor') !~ '^[A-Za-z0-9+/=_-]*$' OR v->'request_cursor'<>'null'::jsonb AND ledger.text_value(v->'request_cursor') !~ '^[A-Za-z0-9+/=_-]+$' THEN PERFORM integration.sync_error('P2600'); END IF;
 idx:=(v->>'page_index')::integer; IF v->'next_cursor'='""'::jsonb THEN IF idx<>0 OR (v->>'has_more')::boolean OR jsonb_typeof(v->'transactions_update_status') IS DISTINCT FROM 'string' OR v->>'transactions_update_status' NOT IN ('TRANSACTIONS_UPDATE_STATUS_UNKNOWN','NOT_READY') OR v->'added'<>'[]'::jsonb OR v->'modified'<>'[]'::jsonb OR v->'removed'<>'[]'::jsonb OR x.starting_cursor IS NOT NULL THEN PERFORM integration.sync_error('P2604'); END IF; ELSE PERFORM integration.sync_text(v->'next_cursor',256); END IF; PERFORM integration.sync_text(v->'request_id',256);
 IF v->'request_cursor'<>'null'::jsonb THEN PERFORM integration.sync_text(v->'request_cursor',256); END IF;
 IF ledger.text_value(v->'sha256') !~ '^[0-9a-f]{64}$' THEN PERFORM integration.sync_error('P2600'); END IF;
 SELECT body INTO old FROM integration.sync_page WHERE entity_id=e AND book_id=b AND set_id=set_value AND page_index=idx;
 IF FOUND THEN IF old<>v THEN PERFORM integration.sync_error('P2602'); END IF; RETURN jsonb_build_object('set_id',set_value,'page_count',x.page_count,'change_count',x.change_count,'final_page',x.final_page); END IF;
 IF idx<>x.page_count OR x.final_page OR (v->>'request_cursor') IS DISTINCT FROM x.continuation_cursor THEN PERFORM integration.sync_error('P2604'); END IF;
 -- A has_more page cannot loop its cursor or return an already-seen continuation.
 IF (v->>'has_more')::boolean AND (v->>'next_cursor' IS NOT DISTINCT FROM x.continuation_cursor OR EXISTS(SELECT FROM integration.sync_page WHERE entity_id=e AND book_id=b AND set_id=set_value AND body->>'next_cursor'=v->>'next_cursor')) THEN PERFORM integration.sync_error('P2604'); END IF;
 SELECT currency_version_id INTO currency_id FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id;
 FOREACH field_name IN ARRAY ARRAY['added','modified','removed'] LOOP
  IF jsonb_typeof(v->field_name)<>'array' THEN PERFORM integration.sync_error('P2600'); END IF; count_value:=count_value+jsonb_array_length(v->field_name);
  FOR row_value IN SELECT value FROM jsonb_array_elements(v->field_name) LOOP PERFORM integration.sync_row(row_value,t.account_id,currency_id,field_name='removed'); END LOOP;
 END LOOP;
 bytes_value:=octet_length(v::text);
 IF count_value>500 OR x.page_count>=10 OR x.change_count+count_value>1000 OR x.byte_count+bytes_value>2097152 THEN PERFORM integration.sync_error('P2605'); END IF;
 INSERT INTO integration.sync_page VALUES(e,b,set_value,idx,v);
 UPDATE integration.sync_set SET page_count=page_count+1,change_count=change_count+count_value,byte_count=byte_count+bytes_value,continuation_cursor=nullif(v->>'next_cursor',''),final_page=NOT(v->>'has_more')::boolean WHERE entity_id=e AND book_id=b AND set_id=set_value RETURNING * INTO x;
 RETURN jsonb_build_object('set_id',set_value,'page_count',x.page_count,'change_count',x.change_count,'final_page',x.final_page);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR numeric_value_out_of_range THEN PERFORM integration.sync_error('P2600'); END $$;
CREATE FUNCTION integration.abandon_sync(e uuid,b uuid,id uuid,set_value uuid,f bigint,reason_value text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream; x integration.sync_set;
BEGIN
 t:=integration.sync_authorize(e,b,id); x:=integration.sync_attempt(t,set_value,f);
 IF reason_value IS NULL OR reason_value NOT IN ('pagination_mutation','provider_failure','decoder_rejected','cancelled') THEN PERFORM integration.sync_error('P2600'); END IF;
 UPDATE integration.sync_set SET state='abandoned',reason=reason_value WHERE entity_id=e AND book_id=b AND set_id=set_value;
 UPDATE integration.sync_stream SET current_set=NULL,lease_until=NULL WHERE entity_id=e AND book_id=b AND stream_id=id;
 RETURN jsonb_build_object('set_id',set_value,'state','abandoned');
END $$;
-- Append evidence/review history with no ledger mutation or inferred financial matching.
CREATE FUNCTION integration.publish_sync_row(t integration.sync_stream,set_value uuid,page_value integer,op text,v jsonb) RETURNS boolean LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE o banking.source_observation; old banking.observation_revision; q banking.observation_revision; r banking.review_item; ns banking.source_namespace; did uuid; next_rev bigint; state_value text; currency_id uuid; material jsonb; jid uuid;
BEGIN
 SELECT * INTO o FROM banking.source_observation WHERE entity_id=t.entity_id AND book_id=t.book_id AND namespace_id=t.namespace_id AND external_entry_id=v->>'transaction_id';
 IF FOUND THEN
  SELECT * INTO old FROM banking.observation_revision WHERE entity_id=t.entity_id AND book_id=t.book_id AND observation_id=o.observation_id AND revision=o.current_revision;
  IF op='added' THEN IF old.provider_fields=v AND old.source_state<>'removed' THEN RETURN false; END IF; PERFORM integration.sync_error('P2602'); END IF;
  IF op='modified' AND old.provider_fields=v AND old.source_state<>'removed' OR op='removed' AND old.source_state='removed' THEN RETURN false; END IF;
  IF old.adapter_profile<>'jayburd-plaid-transactions/1' OR o.current_revision=9223372036854775807 THEN PERFORM integration.sync_error('P2604'); END IF;
  next_rev:=o.current_revision+1;
 ELSE
  IF op<>'added' THEN PERFORM integration.sync_error('P2604'); END IF;
  INSERT INTO banking.source_observation(entity_id,book_id,source_account_id,namespace_id,external_entry_id) VALUES(t.entity_id,t.book_id,t.source_account_id,t.namespace_id,v->>'transaction_id') RETURNING * INTO o;
  next_rev:=1;
 END IF;
 IF op='removed' THEN
  INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,previous_revision,adapter_profile,provider_amount_lexeme,provider_fields)
  VALUES(t.entity_id,t.book_id,o.observation_id,next_rev,'removed',old.booked_date,old.currency_version_id,old.direction,old.amount_lexeme,old.amount,old.description,old.canonical_fields,old.revision,old.adapter_profile,old.provider_amount_lexeme,old.provider_fields) RETURNING * INTO q;
 ELSE
  state_value:=CASE WHEN (v->>'pending')::boolean THEN 'pending' ELSE 'booked' END;
  material:=jsonb_build_object('date',v->'date','currency_version_id',v->'currency_version_id','direction',v->'direction','amount',v->'amount','description',v->'name');
  INSERT INTO banking.observation_revision(entity_id,book_id,observation_id,revision,source_state,booked_date,currency_version_id,direction,amount_lexeme,amount,description,canonical_fields,previous_revision,adapter_profile,provider_amount_lexeme,provider_fields)
  VALUES(t.entity_id,t.book_id,o.observation_id,next_rev,state_value,(v->>'date')::date,(v->>'currency_version_id')::uuid,v->>'direction',v->>'amount',(v->>'amount')::numeric,v->>'name',material,CASE WHEN next_rev>1 THEN next_rev-1 END,'jayburd-plaid-transactions/1',v->>'amount_lexeme',v) RETURNING * INTO q;
 END IF;
 INSERT INTO banking.provider_revision_origin VALUES(t.entity_id,t.book_id,o.observation_id,next_rev,set_value,page_value,op);
 IF next_rev>1 THEN
  SELECT x.* INTO STRICT r FROM banking.review_item x JOIN banking.review_assignment a USING(entity_id,book_id,review_id) WHERE a.entity_id=t.entity_id AND a.book_id=t.book_id AND a.observation_id=o.observation_id;
  IF r.state='posted' THEN
   SELECT journal_id INTO STRICT jid FROM banking.review_posting WHERE entity_id=t.entity_id AND book_id=t.book_id AND review_id=r.review_id;
   INSERT INTO banking.provider_posted_anchor VALUES(t.entity_id,t.book_id,r.review_id,o.observation_id,old.revision,jid) ON CONFLICT DO NOTHING;
   INSERT INTO banking.provider_review_exception(entity_id,book_id,review_id,observation_id,previous_revision,current_revision,set_id,kind) VALUES(t.entity_id,t.book_id,r.review_id,o.observation_id,old.revision,next_rev,set_value,op);
  END IF;
  UPDATE banking.review_item SET state=CASE WHEN state='posted' THEN 'posted' ELSE 'correction_hold' END,review_revision=review_revision+1 WHERE entity_id=t.entity_id AND book_id=t.book_id AND review_id=r.review_id;
  UPDATE banking.source_observation SET current_revision=next_rev WHERE entity_id=t.entity_id AND book_id=t.book_id AND observation_id=o.observation_id;
 ELSE
  SELECT * INTO ns FROM banking.source_namespace WHERE entity_id=t.entity_id AND book_id=t.book_id AND namespace_id=t.namespace_id;
  INSERT INTO banking.review_item(entity_id,book_id,source_account_id,state) VALUES(t.entity_id,t.book_id,t.source_account_id,CASE WHEN q.source_state<>'booked' OR q.amount=0 THEN 'correction_hold' WHEN ns.overlap_review_required THEN 'overlap_hold' ELSE 'needs_review' END) RETURNING * INTO r;
 END IF;
 INSERT INTO banking.review_decision(entity_id,book_id,review_id,expected_review_revision,kind,actor_id,reason,body) VALUES(t.entity_id,t.book_id,r.review_id,CASE WHEN next_rev=1 THEN 0 ELSE r.review_revision END,CASE WHEN next_rev=1 THEN 'initialize' ELSE 'source_update' END,t.service_principal,'provider_sync',jsonb_build_object('set_id',set_value,'observation_id',o.observation_id,'revision',next_rev::text,'operation',op)) RETURNING decision_id INTO did;
 INSERT INTO banking.review_evidence VALUES(t.entity_id,t.book_id,did,o.observation_id,r.review_id,next_rev);
 IF next_rev=1 THEN INSERT INTO banking.review_assignment VALUES(t.entity_id,t.book_id,o.observation_id,r.review_id,did); END IF;
 RETURN true;
END $$;
-- Receipt, all immutable revisions, review projections and C1 share one transaction.
CREATE FUNCTION integration.complete_sync(e uuid,b uuid,id uuid,set_value uuid,f bigint) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream; x integration.sync_set; result_value jsonb; row_value record; changed integer:=0; sr bigint; readiness_value text; pending_id uuid; child_id uuid; old_link uuid;
BEGIN
 t:=integration.sync_authorize(e,b,id);
 SELECT c.result INTO result_value FROM integration.sync_completion c JOIN integration.sync_set s USING(entity_id,book_id,set_id) WHERE c.entity_id=e AND c.book_id=b AND c.set_id=set_value AND s.stream_id=id AND c.fence=f;
 IF FOUND THEN RETURN result_value; END IF;
 x:=integration.sync_attempt(t,set_value,f); IF NOT x.final_page OR x.page_count=0 THEN PERFORM integration.sync_error('P2604'); END IF;
 IF EXISTS(SELECT FROM integration.sync_page p CROSS JOIN LATERAL (SELECT z.value FROM jsonb_array_elements(p.body->'added') z UNION ALL SELECT z.value FROM jsonb_array_elements(p.body->'modified') z UNION ALL SELECT z.value FROM jsonb_array_elements(p.body->'removed') z) rows WHERE p.entity_id=e AND p.book_id=b AND p.set_id=set_value GROUP BY rows.value->>'transaction_id' HAVING count(*)>1) THEN PERFORM integration.sync_error('P2604'); END IF;
 FOR row_value IN SELECT p.page_index,changes.op,changes.v FROM integration.sync_page p CROSS JOIN LATERAL (SELECT 'added'::text op,z.value v FROM jsonb_array_elements(p.body->'added') z UNION ALL SELECT 'modified',z.value FROM jsonb_array_elements(p.body->'modified') z UNION ALL SELECT 'removed',z.value FROM jsonb_array_elements(p.body->'removed') z) changes WHERE p.entity_id=e AND p.book_id=b AND p.set_id=set_value ORDER BY p.page_index,changes.op,changes.v->>'transaction_id' LOOP
  IF integration.publish_sync_row(t,set_value,row_value.page_index,row_value.op,row_value.v) THEN changed:=changed+1; END IF;
 END LOOP;
 -- Resolve lineage only after all pages are published inside this uncommitted transaction.
 FOR row_value IN SELECT z.value v FROM integration.sync_page p CROSS JOIN LATERAL jsonb_array_elements((p.body->'added')||(p.body->'modified')) z WHERE p.entity_id=e AND p.book_id=b AND p.set_id=set_value AND z.value->'pending_transaction_id'<>'null'::jsonb LOOP
  SELECT observation_id INTO pending_id FROM banking.source_observation WHERE entity_id=e AND book_id=b AND namespace_id=t.namespace_id AND external_entry_id=row_value.v->>'pending_transaction_id';
  SELECT observation_id INTO child_id FROM banking.source_observation WHERE entity_id=e AND book_id=b AND namespace_id=t.namespace_id AND external_entry_id=row_value.v->>'transaction_id';
  IF pending_id IS NULL OR pending_id=child_id OR NOT EXISTS(SELECT FROM banking.observation_revision WHERE entity_id=e AND book_id=b AND observation_id=pending_id AND source_state='pending') THEN PERFORM integration.sync_error('P2604'); END IF;
  SELECT booked_observation_id INTO old_link FROM banking.provider_pending_link WHERE entity_id=e AND book_id=b AND pending_observation_id=pending_id;
  IF FOUND THEN IF old_link<>child_id THEN PERFORM integration.sync_error('P2604'); END IF;
  ELSE
   IF EXISTS(SELECT FROM banking.provider_pending_link WHERE entity_id=e AND book_id=b AND booked_observation_id=child_id) THEN PERFORM integration.sync_error('P2604'); END IF;
   INSERT INTO banking.provider_pending_link VALUES(e,b,pending_id,child_id,set_value);
  END IF;
 END LOOP;
 SELECT source_revision INTO sr FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id;
 IF changed>0 THEN IF sr=9223372036854775807 THEN PERFORM integration.sync_error('P2605'); END IF; UPDATE banking.source_account SET source_revision=source_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id RETURNING source_revision INTO sr; END IF;
 SELECT body->>'transactions_update_status' INTO readiness_value FROM integration.sync_page WHERE entity_id=e AND book_id=b AND set_id=set_value ORDER BY page_index DESC LIMIT 1;
 UPDATE integration.sync_set SET state='completed' WHERE entity_id=e AND book_id=b AND set_id=set_value;
 UPDATE integration.sync_stream SET cursor_value=x.continuation_cursor,completed_generation=x.target_generation,current_set=NULL,lease_until=NULL,last_completed_at=clock_timestamp(),readiness=readiness_value WHERE entity_id=e AND book_id=b AND stream_id=id;
 result_value:=jsonb_build_object('stream_id',id,'set_id',set_value,'cursor',x.continuation_cursor,'completed_generation',x.target_generation::text,'source_revision',sr::text,'changed_count',changed::text,'readiness',readiness_value);
 INSERT INTO integration.sync_completion VALUES(e,b,set_value,f,result_value); RETURN result_value;
END $$;

-- Read posted evidence from its immutable anchor, never the latest correction.
CREATE OR REPLACE FUNCTION banking.review_fact(e uuid,b uuid,id uuid) RETURNS banking.observation_revision LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE q banking.observation_revision;
BEGIN
 BEGIN SELECT v.* INTO STRICT q FROM banking.review_assignment a JOIN banking.source_observation o USING(entity_id,book_id,observation_id) LEFT JOIN banking.provider_posted_anchor h ON h.entity_id=a.entity_id AND h.book_id=a.book_id AND h.review_id=a.review_id JOIN banking.observation_revision v ON v.entity_id=o.entity_id AND v.book_id=o.book_id AND v.observation_id=o.observation_id AND v.revision=coalesce(h.revision,o.current_revision) WHERE a.entity_id=e AND a.book_id=b AND a.review_id=id;
 EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM banking.review_error('P2502'); END;
 RETURN q;
END $$;
CREATE OR REPLACE FUNCTION banking.list_review_items(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; cur jsonb; sid uuid; s banking.source_account; filter_value text; page_limit integer; last_id uuid; cursor_text text; r banking.review_item; rows_value jsonb:='[]'; n integer:=0; next_value text;
BEGIN
 v:=banking.import_json(raw,4096);
 IF NOT(v ?& ARRAY['source_account_id','state','limit','cursor']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['source_account_id','state','limit','cursor'])) THEN PERFORM banking.review_error('P2500'); END IF;
 sid:=ledger.uuid_value(v->'source_account_id');
 IF v->'state'<>'null'::jsonb THEN filter_value:=ledger.text_value(v->'state'); IF filter_value NOT IN ('needs_review','overlap_hold','draft_ready','posted','correction_hold') THEN PERFORM banking.review_error('P2500'); END IF; END IF;
 IF jsonb_typeof(v->'limit')<>'number' OR v->>'limit' !~ '^[1-9][0-9]{0,2}$' OR (v->>'limit')::integer>100 THEN PERFORM banking.review_error('P2500'); END IF; page_limit:=(v->>'limit')::integer;
 s:=banking.import_authorize(actor,e,b,sid,false);
 IF v->'cursor'<>'null'::jsonb THEN
  cursor_text:=ledger.text_value(v->'cursor');
  IF length(cursor_text) NOT BETWEEN 2 AND 2048 OR cursor_text COLLATE "C" !~ '^([0-9a-f]{2})+$' THEN PERFORM banking.review_error('P2500'); END IF;
  cur:=banking.import_json(convert_from(decode(cursor_text,'hex'),'UTF8'),1024);
  IF NOT(cur ?& ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id']) OR EXISTS(SELECT FROM jsonb_object_keys(cur) x WHERE x<>ALL(ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id'])) THEN PERFORM banking.review_error('P2500'); END IF;
  IF ROW(ledger.uuid_value(cur->'entity_id'),ledger.uuid_value(cur->'book_id'),ledger.uuid_value(cur->'source_account_id'),cur->'state') IS DISTINCT FROM ROW(e,b,sid,v->'state') THEN PERFORM banking.review_error('P2500'); END IF;
  IF banking.import_revision(cur->'source_revision')<>s.source_revision THEN PERFORM banking.review_error('P2503'); END IF;
  last_id:=ledger.uuid_value(cur->'last_id');
 END IF;
 FOR r IN SELECT * FROM banking.review_item WHERE entity_id=e AND book_id=b AND source_account_id=sid AND (filter_value IS NULL OR state=filter_value) AND (last_id IS NULL OR review_id>last_id) ORDER BY review_id LIMIT page_limit+1 LOOP
  n:=n+1;
  IF n>page_limit THEN next_value:=encode(convert_to(jsonb_build_object('entity_id',e,'book_id',b,'source_account_id',sid,'source_revision',s.source_revision::text,'state',filter_value,'last_id',last_id)::text,'UTF8'),'hex'); EXIT; END IF;
  rows_value:=rows_value||jsonb_build_array(banking.render_review_summary(r)); last_id:=r.review_id;
 END LOOP;
 RETURN jsonb_build_object('profile','source-review-page-v1','source_account_id',sid,'source_revision',s.source_revision::text,'items',rows_value,'next_cursor',next_value);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.review_error('P2500'); WHEN SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501'); END $$;
-- Exceptions are bounded, ordered and separately identified from original posted evidence.
CREATE FUNCTION banking.render_provider_exceptions(e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT CASE WHEN n=0 THEN '{}'::jsonb ELSE jsonb_build_object('provider_exceptions',jsonb_build_object('total_count',n::text,'truncated',n>100,'items',coalesce((SELECT jsonb_agg(v ORDER BY rev DESC,eid) FROM (SELECT current_revision rev,exception_id eid,jsonb_build_object('exception_id',exception_id,'kind',kind,'previous_revision',previous_revision::text,'current_revision',current_revision::text,'observation_id',observation_id,'created_at',created_at) v FROM banking.provider_review_exception WHERE entity_id=e AND book_id=b AND review_id=id ORDER BY current_revision DESC,exception_id LIMIT 100) rows),'[]'::jsonb))) END FROM (SELECT count(*) n FROM banking.provider_review_exception WHERE entity_id=e AND book_id=b AND review_id=id) totals
$$;
CREATE OR REPLACE FUNCTION banking.get_review_item(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE r banking.review_item; s banking.source_account; q banking.observation_revision; d banking.review_draft; o banking.source_observation; rev bigint; blockers jsonb:='[]';
BEGIN
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=id;
 s:=banking.import_authorize(actor,e,b,r.source_account_id,false); q:=banking.review_fact(e,b,id);
 SELECT * INTO o FROM banking.source_observation WHERE entity_id=e AND book_id=b AND observation_id=q.observation_id;
 SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b;
 SELECT * INTO d FROM banking.review_draft WHERE entity_id=e AND book_id=b AND review_id=id ORDER BY review_revision DESC LIMIT 1;
 IF r.state='correction_hold' THEN blockers:=jsonb_build_array('provider_correction_required'); END IF;
 IF r.state='overlap_hold' THEN blockers:=jsonb_build_array('overlap_review_required'); END IF;
 IF r.state='draft_ready' AND ROW(d.source_revision,d.binding_revision,d.book_revision,d.review_revision) IS DISTINCT FROM ROW(s.source_revision,s.binding_revision,rev,r.review_revision) THEN blockers:=blockers||jsonb_build_array('draft_stale'); END IF;
 RETURN banking.render_review_summary(r)||banking.render_provider_exceptions(e,b,id)||jsonb_build_object('profile','source-review-v1','source_account_id',r.source_account_id,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'book_revision',rev::text,'ledger_account_id',s.ledger_account_id,'created_at',r.created_at,'blockers',blockers,'draft',banking.render_review_draft(d),
  'evidence',jsonb_build_array(jsonb_build_object('observation_id',o.observation_id,'observation_revision',q.revision::text,'namespace_id',o.namespace_id,'external_entry_id',o.external_entry_id,'source_state',q.source_state,'booked_date',to_char(q.booked_date,'YYYY-MM-DD'),'currency_version_id',q.currency_version_id,'direction',q.direction,'amount',trim_scale(q.amount)::text,'description',q.description)));
END $$;
-- Reject mixed sync membership on existing interactive and observer authorization paths.
CREATE OR REPLACE FUNCTION ledger.authorize(actor uuid,e uuid,b uuid,permission_name text,writing boolean) RETURNS bigint LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE rev bigint;
BEGIN
 IF pg_has_role(session_user,'bookkeeping_sync','MEMBER') THEN PERFORM ledger.fail('P2101'); END IF;
 IF actor IS NULL OR e IS NULL OR b IS NULL THEN PERFORM ledger.fail('P2100'); END IF;
 IF (writing AND current_setting('transaction_isolation')<>'serializable') OR
 (NOT writing AND current_setting('transaction_isolation') NOT IN ('serializable','repeatable read')) THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 PERFORM 1 FROM ledger.entity WHERE entity_id=e AND disabled_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission=permission_name AND revoked_at IS NULL FOR SHARE;
 IF NOT FOUND THEN
  IF EXISTS(SELECT FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND revoked_at IS NULL) THEN PERFORM ledger.fail('P2102'); END IF;
  PERFORM ledger.fail('P2101');
 END IF;
 IF writing THEN SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b AND disabled_at IS NULL FOR UPDATE;
 ELSE SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b AND disabled_at IS NULL FOR SHARE; END IF;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 RETURN rev;
END $$;
CREATE OR REPLACE FUNCTION integration.observer_binding_lock(id uuid,publisher boolean) RETURNS integration.observer_binding LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE b integration.observer_binding;
BEGIN
 IF pg_has_role(session_user,'bookkeeping_sync','MEMBER') THEN PERFORM integration.observer_error('P2201'); END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR id IS NULL THEN PERFORM integration.observer_error('P2200'); END IF;
 PERFORM set_config('lock_timeout','1s',true);
 SELECT * INTO b FROM integration.observer_binding WHERE binding_id=id FOR SHARE;
 IF NOT FOUND OR session_user<>(CASE WHEN publisher THEN b.publisher_login ELSE b.consumer_login END) THEN PERFORM integration.observer_error('P2201'); END IF;
 RETURN b;
END $$;
-- Existing source reads retain their exact CSV branch and expose a tagged provider identity.
CREATE OR REPLACE FUNCTION banking.get_source(actor uuid,e uuid,b uuid,sid uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
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
 'namespaces',(SELECT coalesce(jsonb_agg(CASE WHEN n.profile='jayburd-plaid-transactions/1' THEN jsonb_build_object('namespace_id',n.namespace_id,'profile',n.profile,'provider',n.provider,'environment',n.environment,'item_id',n.item_id,'provider_account_id',n.provider_account_id,'state',n.state,'overlap_review_required',n.overlap_review_required) ELSE jsonb_build_object('namespace_id',n.namespace_id,'profile',n.profile,'synthetic_dataset_id',n.synthetic_dataset_id,'source_account_ref',n.source_account_ref,'state',n.state,'overlap_review_required',n.overlap_review_required) END ORDER BY n.namespace_id),'[]'::jsonb) FROM banking.source_namespace n WHERE n.entity_id=e AND n.book_id=b AND n.source_account_id=sid)) INTO result_value FROM reference.currency c WHERE c.currency_version_id=s.currency_version_id;
 IF (SELECT count(*) FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND source_account_id=sid)>25 OR octet_length(result_value::text)>65536 THEN PERFORM banking.source_error('P2304'); END IF;
 RETURN result_value;
END $$;
-- State can advance; enrolled identities and completed/private history cannot be rewritten.
CREATE FUNCTION integration.sync_state_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.sync_error('P2602'); END IF;
 IF TG_TABLE_NAME='sync_item' THEN
  IF (to_jsonb(NEW)-ARRAY['generation','state']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['generation','state']) OR NEW.generation::numeric<>OLD.generation::numeric+1 OR OLD.state='disconnected' THEN PERFORM integration.sync_error('P2602'); END IF;
 ELSIF TG_TABLE_NAME='sync_stream' THEN
  IF (to_jsonb(NEW)-ARRAY['requested_generation','completed_generation','cursor_value','fence','lease_until','current_set','last_completed_at','readiness']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['requested_generation','completed_generation','cursor_value','fence','lease_until','current_set','last_completed_at','readiness']) OR NEW.requested_generation<OLD.requested_generation OR NEW.completed_generation<OLD.completed_generation OR NEW.fence<OLD.fence THEN PERFORM integration.sync_error('P2602'); END IF;
 ELSE
  IF (to_jsonb(NEW)-ARRAY['state','reason','page_count','change_count','byte_count','final_page','continuation_cursor']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','reason','page_count','change_count','byte_count','final_page','continuation_cursor']) OR OLD.state<>'staging' OR NEW.page_count<OLD.page_count OR NEW.change_count<OLD.change_count OR NEW.byte_count<OLD.byte_count THEN PERFORM integration.sync_error('P2602'); END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_sync BEFORE UPDATE OR DELETE ON integration.sync_item FOR EACH ROW EXECUTE FUNCTION integration.sync_state_guard();
CREATE TRIGGER preserve_sync BEFORE UPDATE OR DELETE ON integration.sync_stream FOR EACH ROW EXECUTE FUNCTION integration.sync_state_guard();
CREATE TRIGGER preserve_sync BEFORE UPDATE OR DELETE ON integration.sync_set FOR EACH ROW EXECUTE FUNCTION integration.sync_state_guard();
REVOKE ALL ON FUNCTION integration.sync_state_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
-- ACLs are explicit: owner provisioners remain invoker-only; sync gets six fixed operations.
REVOKE ALL ON ALL TABLES IN SCHEMA integration FROM bookkeeping_sync;
REVOKE ALL ON ALL TABLES IN SCHEMA banking FROM bookkeeping_sync;
GRANT USAGE ON SCHEMA integration TO bookkeeping_sync;
REVOKE ALL ON FUNCTION banking.observation_pointer_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.sync_error(text) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.sync_text(jsonb,integer) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.sync_authority(uuid,uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.sync_authorize(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.sync_login(name) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.enroll_sync_stream(uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.set_sync_item_state(uuid,uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.request_sync(uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.render_sync_claim(integration.sync_stream,integration.sync_set) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.claim_sync(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.sync_attempt(integration.sync_stream,uuid,bigint) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.sync_row(jsonb,text,uuid,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.stage_sync_page(uuid,uuid,uuid,uuid,bigint,text) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.abandon_sync(uuid,uuid,uuid,uuid,bigint,text) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.publish_sync_row(integration.sync_stream,uuid,integer,text,jsonb) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.complete_sync(uuid,uuid,uuid,uuid,bigint) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
GRANT EXECUTE ON FUNCTION integration.request_sync(uuid,uuid,uuid,text),integration.claim_sync(uuid,uuid,uuid,uuid),integration.stage_sync_page(uuid,uuid,uuid,uuid,bigint,text),integration.complete_sync(uuid,uuid,uuid,uuid,bigint),integration.abandon_sync(uuid,uuid,uuid,uuid,bigint,text) TO bookkeeping_sync;
REVOKE ALL ON FUNCTION banking.render_provider_exceptions(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;
COMMIT;
