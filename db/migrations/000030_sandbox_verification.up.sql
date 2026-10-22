-- #272 reviewed scoped synthetic equality capability; no runtime table reads or posting authority.
BEGIN;
DO $$ BEGIN
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_verifier') THEN CREATE ROLE bookkeeping_verifier NOLOGIN;END IF;
 IF EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_verifier' AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)) THEN RAISE EXCEPTION 'verification_rejected';END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE integration.verification_mandate (
 mandate_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),entity_id uuid NOT NULL,book_id uuid NOT NULL,source_account_id uuid NOT NULL,namespace_id uuid NOT NULL,stream_id uuid NOT NULL,sync_item_id uuid NOT NULL,
 verifier_login name NOT NULL CHECK(verifier_login='jayburd-dev-verifier@jayburd-bookkeeping-dev.iam'),approving_principal uuid NOT NULL REFERENCES ledger.principal,
 binding_revision bigint NOT NULL CHECK(binding_revision>0),item_generation bigint NOT NULL CHECK(item_generation>0),target_generation bigint NOT NULL CHECK(target_generation>0),
 fixture_id text NOT NULL CHECK(fixture_id~'^[A-Za-z0-9._:-]{1,64}$'),fixture_sha256 bytea NOT NULL CHECK(octet_length(fixture_sha256)=32),identity_map_sha256 bytea NOT NULL CHECK(octet_length(identity_map_sha256)=32),
 enabled boolean NOT NULL DEFAULT true,mandate_revision bigint NOT NULL DEFAULT 1 CHECK(mandate_revision>0),created_by_login name NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),expires_at timestamptz NOT NULL,
 FOREIGN KEY(entity_id,book_id,source_account_id,namespace_id) REFERENCES banking.source_namespace(entity_id,book_id,source_account_id,namespace_id),
 FOREIGN KEY(entity_id,book_id,stream_id) REFERENCES integration.sync_stream,FOREIGN KEY(entity_id,book_id,sync_item_id) REFERENCES integration.sync_item,
 CHECK(isfinite(created_at) AND isfinite(expires_at) AND expires_at>created_at AND expires_at<=created_at+interval '60 minutes')
);
CREATE TABLE integration.verification_operator_receipt (
 operator_login name NOT NULL,idempotency_key text NOT NULL CHECK(idempotency_key~'^[A-Za-z0-9._:-]{1,128}$'),body jsonb NOT NULL CHECK(octet_length(body::text)<=8192),result jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(operator_login,idempotency_key)
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.verification_operator_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Only explicit revocation may alter a mandate; expiry/scope/digests cannot be renewed in place.
CREATE FUNCTION integration.verification_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN
 IF TG_OP='DELETE' OR (to_jsonb(NEW)-ARRAY['enabled','mandate_revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['enabled','mandate_revision']) OR NOT OLD.enabled OR NEW.enabled OR NEW.mandate_revision<>OLD.mandate_revision+1 THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_mandate BEFORE UPDATE OR DELETE ON integration.verification_mandate FOR EACH ROW EXECUTE FUNCTION integration.verification_guard();
-- Membership checks cover predefined capabilities even when SUPERUSER/BYPASSRLS attributes are absent.
CREATE FUNCTION integration.verifier_login(login_value name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolname='jayburd-dev-verifier@jayburd-bookkeeping-dev.iam' AND rolcanlogin AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 AND EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_verifier' AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 AND pg_has_role(login_value,'bookkeeping_verifier','USAGE') AND NOT pg_has_role(login_value,'bookkeeping_verifier','SET')
 AND NOT EXISTS(SELECT FROM pg_roles WHERE rolname<>'bookkeeping_verifier' AND (rolname LIKE 'bookkeeping_%' OR rolname LIKE 'pg_%' OR rolname='cloudsqlsuperuser' OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND (pg_has_role(login_value,oid,'MEMBER') OR pg_has_role('bookkeeping_verifier',oid,'MEMBER')))
 AND NOT EXISTS(SELECT FROM pg_auth_members a JOIN pg_roles r ON r.oid=a.member WHERE r.rolname IN (login_value,'bookkeeping_verifier') AND (a.admin_option OR a.set_option))
 AND NOT has_database_privilege(login_value,current_database(),'CREATE,TEMP,CONNECT WITH GRANT OPTION')
 AND NOT EXISTS(SELECT FROM pg_namespace n WHERE has_schema_privilege(login_value,n.oid,'CREATE,USAGE WITH GRANT OPTION'))
 AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment') AND c.relkind IN ('r','p','v','m','f') AND (has_table_privilege(login_value,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege(login_value,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment') AND CASE WHEN c.relkind='S' THEN has_sequence_privilege(login_value,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)
 AND NOT EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment') AND (has_function_privilege(login_value,p.oid,'EXECUTE WITH GRANT OPTION') OR p.oid<>to_regprocedure('integration.verify_sandbox_fixture(uuid,bigint,text,text)') AND has_function_privilege(login_value,p.oid,'EXECUTE')))
 AND NOT EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace CROSS JOIN LATERAL aclexplode(p.proacl) a JOIN pg_roles r ON r.oid=a.grantee WHERE r.rolname=login_value AND n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment'))
$$;
-- Lock current authority before the common book/source/Item/stream order; no external work is performed.
CREATE FUNCTION integration.verification_scope(m integration.verification_mandate,ready boolean) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;i integration.sync_item;s banking.source_account;p uuid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=m.entity_id AND book_id=m.book_id AND stream_id=m.stream_id;
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=m.entity_id AND book_id=m.book_id AND sync_item_id=m.sync_item_id;
 IF t.stream_id IS NULL OR i.sync_item_id IS NULL OR t.sync_item_id<>m.sync_item_id OR t.source_account_id<>m.source_account_id OR t.namespace_id<>m.namespace_id THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 FOR p IN SELECT DISTINCT x FROM unnest(ARRAY[m.approving_principal,i.consent_principal,t.service_principal]) x ORDER BY x LOOP
  PERFORM 1 FROM ledger.principal WHERE principal_id=p AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 END LOOP;
 PERFORM 1 FROM ledger.entity WHERE entity_id=m.entity_id AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 FOR p IN SELECT DISTINCT x FROM unnest(ARRAY[m.approving_principal,i.consent_principal,t.service_principal]) x ORDER BY x LOOP
  PERFORM 1 FROM ledger.membership WHERE entity_id=m.entity_id AND principal_id=p AND permission='read' AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 END LOOP;
 PERFORM 1 FROM ledger.book WHERE entity_id=m.entity_id AND book_id=m.book_id AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=m.entity_id AND book_id=m.book_id AND source_account_id=m.source_account_id FOR SHARE;
 IF NOT FOUND OR s.state<>'active' OR s.binding_revision<>m.binding_revision OR (SELECT count(*) FROM banking.source_account WHERE entity_id=m.entity_id AND book_id=m.book_id)<>1 THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 FOR p IN SELECT DISTINCT x FROM unnest(ARRAY[m.approving_principal,i.consent_principal,t.service_principal]) x ORDER BY x LOOP
  PERFORM 1 FROM banking.source_permission WHERE entity_id=m.entity_id AND book_id=m.book_id AND source_account_id=m.source_account_id AND principal_id=p AND capability='read' AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 END LOOP;
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=m.entity_id AND book_id=m.book_id AND sync_item_id=m.sync_item_id FOR SHARE;
 IF i.provider<>'plaid' OR i.environment<>'sandbox' OR NOT i.synthetic OR i.state<>'active' OR i.generation<>m.item_generation OR i.consent_expires_at<=clock_timestamp() OR m.expires_at>i.consent_expires_at THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 PERFORM 1 FROM banking.source_namespace WHERE entity_id=m.entity_id AND book_id=m.book_id AND source_account_id=m.source_account_id AND namespace_id=m.namespace_id AND state='active' AND profile='jayburd-plaid-transactions/1' FOR SHARE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=m.entity_id AND book_id=m.book_id AND stream_id=m.stream_id FOR SHARE;
 IF t.state<>'active' THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 IF ready AND (t.current_set IS NOT NULL OR t.lease_until IS NOT NULL OR t.completed_generation<>m.target_generation OR t.requested_generation<>m.target_generation OR t.readiness<>'HISTORICAL_UPDATE_COMPLETE') THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
END $$;
-- Owner supplies only reviewed metadata/hashes; this attestation is distinct from runtime authentication.
CREATE FUNCTION integration.enroll_verification(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;m integration.verification_mandate;r integration.verification_operator_receipt;result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='verification_rejected';END IF;
 v:=banking.import_json(raw,8192);PERFORM ledger.keys(v,ARRAY['mandate_id','source_account_id','namespace_id','stream_id','sync_item_id','verifier_login','binding_revision','item_generation','target_generation','fixture_id','fixture_sha256','identity_map_sha256','expires_at']);
 IF k IS NULL OR k!~'^[A-Za-z0-9._:-]{1,128}$' OR actor IS NULL OR e IS NULL OR b IS NULL THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 m.mandate_id:=ledger.uuid_value(v->'mandate_id');m.entity_id:=e;m.book_id:=b;m.source_account_id:=ledger.uuid_value(v->'source_account_id');m.namespace_id:=ledger.uuid_value(v->'namespace_id');m.stream_id:=ledger.uuid_value(v->'stream_id');m.sync_item_id:=ledger.uuid_value(v->'sync_item_id');m.verifier_login:=ledger.text_value(v->'verifier_login');m.approving_principal:=actor;
 m.binding_revision:=banking.import_revision(v->'binding_revision');m.item_generation:=banking.import_revision(v->'item_generation');m.target_generation:=banking.import_revision(v->'target_generation');m.fixture_id:=ledger.text_value(v->'fixture_id');
 IF m.binding_revision=0 OR m.item_generation=0 OR m.target_generation=0 OR m.fixture_id!~'^[A-Za-z0-9._:-]{1,64}$' OR ledger.text_value(v->'fixture_sha256')!~'^[0-9a-f]{64}$' OR ledger.text_value(v->'identity_map_sha256')!~'^[0-9a-f]{64}$' OR NOT integration.verifier_login(m.verifier_login) THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 m.fixture_sha256:=decode(v->>'fixture_sha256','hex');m.identity_map_sha256:=decode(v->>'identity_map_sha256','hex');
 IF ledger.text_value(v->'expires_at')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 m.expires_at:=(v->>'expires_at')::timestamptz;m.created_at:=clock_timestamp();m.created_by_login:=session_user;m.enabled:=true;m.mandate_revision:=1;
 IF NOT isfinite(m.expires_at) OR m.expires_at<=clock_timestamp() OR m.expires_at>clock_timestamp()+interval '60 minutes' THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 PERFORM integration.verification_scope(m,false);
 IF EXISTS(SELECT FROM ledger.journal_entry WHERE entity_id=e AND book_id=b) OR EXISTS(SELECT FROM integration.outbox_message WHERE entity_id=e AND book_id=b AND event_type='journal.posted') THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 PERFORM pg_advisory_xact_lock(31030,hashtext(session_user||':'||k));
 SELECT * INTO r FROM integration.verification_operator_receipt WHERE operator_login=session_user AND idempotency_key=k;
 v:=jsonb_build_object('operation','enroll','actor',actor,'entity',e,'book',b,'body',v);
 IF FOUND THEN IF r.body<>v THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;RETURN r.result;END IF;
 INSERT INTO integration.verification_mandate SELECT m.*;
 result_value:=jsonb_build_object('mandate_id',m.mandate_id,'mandate_revision','1');
 INSERT INTO integration.verification_operator_receipt VALUES(session_user,k,v,result_value,clock_timestamp());RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_datetime_format OR datetime_field_overflow OR numeric_value_out_of_range THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';
END $$;
-- Revocation works even after source/principal disable; only current owner identity authorizes it.
CREATE FUNCTION integration.revoke_verification(id uuid,expected bigint,k text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE m integration.verification_mandate;r integration.verification_operator_receipt;body_value jsonb;result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='verification_rejected';END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR id IS NULL OR expected IS NULL OR expected<1 OR k IS NULL OR k!~'^[A-Za-z0-9._:-]{1,128}$' THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 PERFORM pg_advisory_xact_lock(31030,hashtext(session_user||':'||k));body_value:=jsonb_build_object('operation','revoke','mandate_id',id,'expected_revision',expected::text);
 SELECT * INTO r FROM integration.verification_operator_receipt WHERE operator_login=session_user AND idempotency_key=k;
 IF FOUND THEN IF r.body<>body_value THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;RETURN r.result;END IF;
 SELECT * INTO m FROM integration.verification_mandate WHERE mandate_id=id FOR UPDATE;
 IF NOT FOUND OR NOT m.enabled OR m.mandate_revision<>expected THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 UPDATE integration.verification_mandate SET enabled=false,mandate_revision=mandate_revision+1 WHERE mandate_id=id;
 result_value:=jsonb_build_object('revoked',true,'mandate_revision',(expected+1)::text);INSERT INTO integration.verification_operator_receipt VALUES(session_user,k,body_value,result_value,clock_timestamp());RETURN result_value;
END $$;
-- Resolve mandate metadata privately, then serialize its revocation after current scope locks.
CREATE FUNCTION integration.verification_authorize(id uuid,expected bigint) RETURNS integration.verification_mandate LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE m integration.verification_mandate;
BEGIN
 IF current_setting('role')<>'none' OR NOT integration.verifier_login(session_user) THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 SELECT * INTO m FROM integration.verification_mandate WHERE mandate_id=id;
 IF NOT FOUND OR m.verifier_login<>session_user THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 PERFORM integration.verification_scope(m,true);
 SELECT * INTO m FROM integration.verification_mandate WHERE mandate_id=id FOR SHARE;
 IF NOT m.enabled OR m.mandate_revision IS DISTINCT FROM expected OR m.expires_at<=clock_timestamp() THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;RETURN m;
END $$;
-- One authorized complete comparison; raw expectations/maps stay ephemeral and output is count/Boolean only.
CREATE FUNCTION integration.verify_sandbox_fixture(id uuid,expected bigint,fixture_raw text,map_raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE m integration.verification_mandate;f jsonb;im jsonb;v jsonb;er jsonb:='[]';mapped text;logical text;rv bigint;prev bigint;missing_count bigint;extra_count bigint;field_count bigint;review_count bigint:=0;lineage_count bigint;journals bigint;outboxes bigint;rid uuid;observation_value uuid;detail jsonb;ev jsonb;
BEGIN
 m:=integration.verification_authorize(id,expected);
 IF (SELECT count(*) FROM (SELECT 1 FROM banking.source_observation o JOIN banking.observation_revision q USING(entity_id,book_id,observation_id) WHERE o.entity_id=m.entity_id AND o.book_id=m.book_id AND o.source_account_id=m.source_account_id LIMIT 1001) bounded)>1000 OR (SELECT count(*) FROM (SELECT 1 FROM banking.review_item WHERE entity_id=m.entity_id AND book_id=m.book_id AND source_account_id=m.source_account_id LIMIT 1001) bounded)>1000 THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 IF fixture_raw IS NULL OR map_raw IS NULL OR octet_length(fixture_raw)+octet_length(map_raw)>2097152 OR sha256(convert_to(fixture_raw,'UTF8'))<>m.fixture_sha256 OR sha256(convert_to(map_raw,'UTF8'))<>m.identity_map_sha256 THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 f:=banking.import_json(fixture_raw,2097152);im:=banking.import_json(map_raw,2097152);
 PERFORM ledger.keys(f,ARRAY['profile','fixture_id','rows','pending_links']);PERFORM ledger.keys(im,ARRAY['profile','fixture_id','items']);
 IF f->>'profile'<>'jayburd-sandbox-verification/1' OR im->>'profile'<>'jayburd-sandbox-identity-map/1' OR f->>'fixture_id'<>m.fixture_id OR im->>'fixture_id'<>m.fixture_id OR jsonb_typeof(f->'rows')<>'array' OR jsonb_typeof(f->'pending_links')<>'array' OR jsonb_typeof(im->'items')<>'array' THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 IF jsonb_array_length(f->'rows') NOT BETWEEN 1 AND 1000 OR jsonb_array_length(im->'items') NOT BETWEEN 1 AND 1000 OR jsonb_array_length(f->'pending_links')>1000 THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 FOR v IN SELECT value FROM jsonb_array_elements(im->'items') LOOP
  PERFORM ledger.keys(v,ARRAY['logical_id','transaction_id']);logical:=ledger.text_value(v->'logical_id');PERFORM integration.sync_text(v->'transaction_id',256);
  IF logical!~'^[A-Za-z0-9._:-]{1,64}$' THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 END LOOP;
 IF EXISTS(SELECT FROM jsonb_array_elements(im->'items') x GROUP BY x->>'logical_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(im->'items') x GROUP BY x->>'transaction_id' HAVING count(*)>1) THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 FOR v IN SELECT value FROM jsonb_array_elements(f->'rows') LOOP
  PERFORM ledger.keys(v-'previous_revision'-'review_state',ARRAY['logical_id','revision','current','source_state','booked_date','currency_code','currency_version_id','direction','amount','provider_amount_lexeme','description']);
  IF NOT (v ?& ARRAY['previous_revision','review_state']) THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
  logical:=ledger.text_value(v->'logical_id');rv:=banking.import_revision(v->'revision');prev:=CASE WHEN v->'previous_revision'='null'::jsonb THEN NULL ELSE banking.import_revision(v->'previous_revision') END;
  IF rv=0 OR (rv=1 AND prev IS NOT NULL) OR (rv>1 AND prev IS DISTINCT FROM rv-1) OR jsonb_typeof(v->'current')<>'boolean' OR ledger.text_value(v->'source_state') NOT IN ('pending','booked','removed') OR ledger.text_value(v->'direction') NOT IN ('inflow','outflow','zero') OR ledger.text_value(v->'currency_code')!~'^[A-Z]{3}$' THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
  PERFORM ledger.date_value(ledger.text_value(v->'booked_date'));PERFORM ledger.uuid_value(v->'currency_version_id');
  IF ledger.text_value(v->'amount')!~'^(0|[1-9][0-9]{0,28})(\.[0-9]{0,8}[1-9])?$' OR length(v->>'amount')>39 OR ledger.text_value(v->'provider_amount_lexeme')!~'^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]{1,3})?$' OR length(v->>'provider_amount_lexeme')>128 OR length(ledger.text_value(v->'description')) NOT BETWEEN 1 AND 500 THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
  IF (v->>'current')::boolean THEN
   IF ledger.text_value(v->'review_state') NOT IN ('needs_review','overlap_hold','draft_ready','posted','correction_hold') THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
  ELSIF v->'review_state'<>'null'::jsonb THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
  SELECT x->>'transaction_id' INTO mapped FROM jsonb_array_elements(im->'items') x WHERE x->>'logical_id'=logical;
  IF mapped IS NULL THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
  er:=er||jsonb_build_array(v||jsonb_build_object('transaction_id',mapped));
 END LOOP;
 IF EXISTS(SELECT FROM jsonb_array_elements(er) x GROUP BY x->>'logical_id',x->>'revision' HAVING count(*)>1)
 OR EXISTS(SELECT FROM jsonb_array_elements(er) x GROUP BY x->>'logical_id' HAVING sum(CASE WHEN (x->>'current')::boolean THEN 1 ELSE 0 END)<>1 OR max((x->>'revision')::bigint)<>count(*) OR max(CASE WHEN (x->>'current')::boolean THEN (x->>'revision')::bigint ELSE 0 END)<>max((x->>'revision')::bigint))
 OR EXISTS(SELECT FROM jsonb_array_elements(im->'items') x WHERE NOT EXISTS(SELECT FROM jsonb_array_elements(er) y WHERE y->>'logical_id'=x->>'logical_id')) THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 FOR v IN SELECT value FROM jsonb_array_elements(f->'pending_links') LOOP
  PERFORM ledger.keys(v,ARRAY['pending_id','booked_id']);PERFORM ledger.text_value(v->'pending_id');PERFORM ledger.text_value(v->'booked_id');
  IF v->>'pending_id'=v->>'booked_id' OR NOT EXISTS(SELECT FROM jsonb_array_elements(im->'items') x WHERE x->>'logical_id'=v->>'pending_id') OR NOT EXISTS(SELECT FROM jsonb_array_elements(im->'items') x WHERE x->>'logical_id'=v->>'booked_id') THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 END LOOP;
 IF EXISTS(SELECT FROM jsonb_array_elements(f->'pending_links') x GROUP BY x->>'pending_id' HAVING count(*)>1) OR EXISTS(SELECT FROM jsonb_array_elements(f->'pending_links') x GROUP BY x->>'booked_id' HAVING count(*)>1) THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';END IF;
 WITH expectations AS(SELECT value x FROM jsonb_array_elements(er)), actuals AS(
  SELECT o.external_entry_id,o.current_revision,q.*,c.alphabetic_code FROM banking.source_observation o JOIN banking.observation_revision q USING(entity_id,book_id,observation_id) JOIN reference.currency c USING(currency_version_id) WHERE o.entity_id=m.entity_id AND o.book_id=m.book_id AND o.source_account_id=m.source_account_id AND o.namespace_id=m.namespace_id)
 SELECT (SELECT count(*) FROM expectations e WHERE NOT EXISTS(SELECT FROM actuals a WHERE a.external_entry_id=e.x->>'transaction_id' AND a.revision=(e.x->>'revision')::bigint)),
 (SELECT count(*) FROM actuals a WHERE NOT EXISTS(SELECT FROM expectations e WHERE a.external_entry_id=e.x->>'transaction_id' AND a.revision=(e.x->>'revision')::bigint)),
 (SELECT count(*) FROM expectations e JOIN actuals a ON a.external_entry_id=e.x->>'transaction_id' AND a.revision=(e.x->>'revision')::bigint WHERE ROW(a.previous_revision,(a.current_revision=a.revision),a.source_state,a.booked_date::text,a.alphabetic_code,a.currency_version_id::text,a.direction,trim_scale(a.amount)::text,a.provider_amount_lexeme,a.description) IS DISTINCT FROM ROW(CASE WHEN e.x->'previous_revision'='null'::jsonb THEN NULL ELSE (e.x->>'previous_revision')::bigint END,(e.x->>'current')::boolean,e.x->>'source_state',e.x->>'booked_date',e.x->>'currency_code',e.x->>'currency_version_id',e.x->>'direction',e.x->>'amount',e.x->>'provider_amount_lexeme',e.x->>'description'))
 INTO missing_count,extra_count,field_count;
 SELECT extra_count+count(*) INTO extra_count FROM banking.source_observation o JOIN banking.observation_revision q USING(entity_id,book_id,observation_id) WHERE o.entity_id=m.entity_id AND o.book_id=m.book_id AND o.source_account_id=m.source_account_id AND o.namespace_id<>m.namespace_id;
 FOR v IN SELECT value FROM jsonb_array_elements(er) WHERE (value->>'current')::boolean LOOP
  SELECT a.review_id,o.observation_id INTO rid,observation_value FROM banking.source_observation o JOIN banking.review_assignment a USING(entity_id,book_id,observation_id) WHERE o.entity_id=m.entity_id AND o.book_id=m.book_id AND o.namespace_id=m.namespace_id AND o.external_entry_id=v->>'transaction_id';
  IF rid IS NULL OR (SELECT count(*) FROM banking.review_assignment WHERE entity_id=m.entity_id AND book_id=m.book_id AND review_id=rid)<>1 THEN review_count:=review_count+1;CONTINUE;END IF;
  detail:=banking.get_review_item(m.approving_principal,m.entity_id,m.book_id,rid);
  IF jsonb_typeof(detail->'evidence') IS DISTINCT FROM 'array' OR jsonb_array_length(detail->'evidence')<>1 THEN review_count:=review_count+1;CONTINUE;END IF;ev:=detail->'evidence'->0;
  IF ROW(ev->>'observation_id',ev->>'namespace_id',ev->>'external_entry_id') IS DISTINCT FROM ROW(observation_value::text,m.namespace_id::text,v->>'transaction_id') THEN review_count:=review_count+1;CONTINUE;END IF;
  IF ROW(detail->>'state',ev->>'observation_revision',ev->>'source_state',ev->>'booked_date',ev->>'currency_version_id',ev->>'direction',ev->>'amount',ev->>'description') IS DISTINCT FROM ROW(v->>'review_state',v->>'revision',v->>'source_state',v->>'booked_date',v->>'currency_version_id',v->>'direction',v->>'amount',v->>'description') THEN review_count:=review_count+1;END IF;
 END LOOP;
 SELECT review_count+count(*) INTO review_count FROM banking.review_item r WHERE r.entity_id=m.entity_id AND r.book_id=m.book_id AND r.source_account_id=m.source_account_id AND NOT EXISTS(SELECT FROM banking.review_assignment a JOIN banking.source_observation o USING(entity_id,book_id,observation_id) WHERE a.entity_id=r.entity_id AND a.book_id=r.book_id AND a.review_id=r.review_id AND o.namespace_id=m.namespace_id);
 WITH el AS(SELECT p->>'transaction_id' pending_id,b->>'transaction_id' booked_id FROM jsonb_array_elements(f->'pending_links') l JOIN jsonb_array_elements(im->'items') p ON p->>'logical_id'=l->>'pending_id' JOIN jsonb_array_elements(im->'items') b ON b->>'logical_id'=l->>'booked_id'), al AS(SELECT p.external_entry_id pending_id,b.external_entry_id booked_id FROM banking.provider_pending_link l JOIN banking.source_observation p ON (p.entity_id,p.book_id,p.observation_id)=(l.entity_id,l.book_id,l.pending_observation_id) JOIN banking.source_observation b ON (b.entity_id,b.book_id,b.observation_id)=(l.entity_id,l.book_id,l.booked_observation_id) WHERE l.entity_id=m.entity_id AND l.book_id=m.book_id)
 SELECT (SELECT count(*) FROM el WHERE NOT EXISTS(SELECT FROM al WHERE (al.pending_id,al.booked_id)=(el.pending_id,el.booked_id)))+(SELECT count(*) FROM al WHERE NOT EXISTS(SELECT FROM el WHERE (al.pending_id,al.booked_id)=(el.pending_id,el.booked_id))) INTO lineage_count;
 SELECT count(*) INTO journals FROM (SELECT 1 FROM ledger.journal_entry WHERE entity_id=m.entity_id AND book_id=m.book_id LIMIT 1001) bounded;
 SELECT count(*) INTO outboxes FROM (SELECT 1 FROM integration.outbox_message WHERE entity_id=m.entity_id AND book_id=m.book_id AND event_type='journal.posted' LIMIT 1001) bounded;
 IF m.expires_at<=clock_timestamp() THEN RAISE EXCEPTION USING ERRCODE='P3101',MESSAGE='verification_rejected';END IF;
 RETURN jsonb_build_object('equal',missing_count+extra_count+field_count+review_count+lineage_count=0 AND journals=0 AND outboxes=0,'missing_count',least(missing_count,1001)::text,'extra_count',least(extra_count,1001)::text,'field_mismatch_count',least(field_count,1001)::text,'review_mismatch_count',least(review_count,1001)::text,'lineage_mismatch_count',least(lineage_count,1001)::text,'journal_count',journals::text,'journal_outbox_count',outboxes::text,'no_automatic_journals',journals=0 AND outboxes=0);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2600' OR data_exception THEN RAISE EXCEPTION USING ERRCODE='P3100',MESSAGE='verification_rejected';
END $$;
REVOKE ALL ON ALL TABLES IN SCHEMA integration FROM bookkeeping_verifier;
GRANT USAGE ON SCHEMA integration TO bookkeeping_verifier;
REVOKE ALL ON FUNCTION integration.verification_guard(),integration.verifier_login(name),integration.verification_scope(integration.verification_mandate,boolean),integration.enroll_verification(uuid,uuid,uuid,text,text),integration.revoke_verification(uuid,bigint,text),integration.verification_authorize(uuid,bigint),integration.verify_sandbox_fixture(uuid,bigint,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_publisher,bookkeeping_observer,bookkeeping_verifier;
GRANT EXECUTE ON FUNCTION integration.verify_sandbox_fixture(uuid,bigint,text,text) TO bookkeeping_verifier;
GRANT EXECUTE ON FUNCTION integration.enroll_verification(uuid,uuid,uuid,text,text),integration.revoke_verification(uuid,bigint,text) TO bookkeeping_owner;
COMMIT;
