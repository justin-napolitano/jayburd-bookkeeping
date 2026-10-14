-- Durable synthetic connection lifecycle. Tokens and remote side effects stay outside SQL.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
ALTER TABLE ledger.membership DROP CONSTRAINT membership_permission_check;
ALTER TABLE ledger.membership ADD CONSTRAINT membership_permission_check CHECK(permission IN ('read','prepare','post','close','import','connect'));
ALTER TABLE integration.sync_item DROP CONSTRAINT sync_item_state_check;
ALTER TABLE integration.sync_item ADD CONSTRAINT sync_item_state_check CHECK(state IN ('active','paused','disconnected','credential_recovery_required','disconnect_pending','reauthorization_required'));
ALTER TABLE integration.sync_stream ADD COLUMN state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','disconnected'));
CREATE TABLE integration.connection_worker (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,service_principal uuid NOT NULL REFERENCES ledger.principal,worker_login name NOT NULL,enabled boolean NOT NULL,revision bigint NOT NULL CHECK(revision>0),
 PRIMARY KEY(entity_id,book_id),FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
CREATE TABLE integration.connection_attempt (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,attempt_id uuid NOT NULL DEFAULT gen_random_uuid(),actor_id uuid NOT NULL REFERENCES ledger.principal,idempotency_key text NOT NULL,
 mode text NOT NULL CHECK(mode IN ('new','reconnect','rotate','remove')),state text NOT NULL CHECK(state IN ('link_pending','link_ready','exchange_claimed','credential_pending','binding_review','active','credential_recovery_required','disconnect_pending','disconnected')),
 body jsonb NOT NULL,worker_revision bigint NOT NULL,nonce uuid NOT NULL DEFAULT gen_random_uuid(),expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '15 minutes',
 sync_item_id uuid,item_generation bigint,expected_item_id text,claim_id uuid,credential_ref uuid,credential_version bigint,previous_credential_ref uuid,previous_credential_version bigint,
 link_token_sha256 text CHECK(link_token_sha256 ~ '^[0-9a-f]{64}$'),public_token_sha256 text CHECK(public_token_sha256 ~ '^[0-9a-f]{64}$'),manifest jsonb,reason text,bind_body jsonb,streams jsonb,
 PRIMARY KEY(entity_id,book_id,attempt_id),UNIQUE(entity_id,book_id,actor_id,idempotency_key),UNIQUE(public_token_sha256),
 FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,FOREIGN KEY(entity_id,book_id,sync_item_id) REFERENCES integration.sync_item,
 CHECK((claim_id IS NULL)=(credential_ref IS NULL)),CHECK(credential_version IS NULL OR credential_version>0),CHECK(octet_length(body::text)<=16384 AND (manifest IS NULL OR octet_length(manifest::text)<=32768))
);
CREATE TABLE integration.connection_receipt (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,actor_id uuid NOT NULL REFERENCES ledger.principal,operation text NOT NULL,idempotency_key text NOT NULL,body jsonb NOT NULL,result jsonb NOT NULL,database_actor name NOT NULL DEFAULT session_user,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,actor_id,operation,idempotency_key),FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
CREATE TABLE integration.connection_event (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,attempt_id uuid NOT NULL,event_id uuid NOT NULL DEFAULT gen_random_uuid(),kind text NOT NULL,body jsonb NOT NULL,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,event_id),FOREIGN KEY(entity_id,book_id,attempt_id) REFERENCES integration.connection_attempt
);
CREATE TABLE integration.credential_publication (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,sync_item_id uuid NOT NULL,credential_version bigint NOT NULL,credential_ref uuid NOT NULL,attempt_id uuid NOT NULL,published_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,sync_item_id,credential_version),UNIQUE(credential_ref),FOREIGN KEY(entity_id,book_id,sync_item_id) REFERENCES integration.sync_item,FOREIGN KEY(entity_id,book_id,attempt_id) REFERENCES integration.connection_attempt
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.connection_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.connection_event FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.credential_publication FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Fixed diagnostics never echo provider/account metadata.
CREATE FUNCTION integration.connection_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='connection_rejected'; END $$;
CREATE FUNCTION integration.connection_json(raw text,max_bytes integer) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN RETURN banking.import_json(raw,max_bytes); EXCEPTION WHEN SQLSTATE 'P2400' THEN PERFORM integration.connection_error('P2700'); END $$;
CREATE FUNCTION integration.connection_key(k text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN IF k IS NULL OR k !~ '^[A-Za-z0-9._:-]{1,128}$' THEN PERFORM integration.connection_error('P2700'); END IF; END $$;
-- Exact UTC expiry syntax avoids permissive PostgreSQL timestamp normalization.
CREATE FUNCTION integration.connection_expiry(v jsonb) RETURNS timestamptz LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE x text;
BEGIN
 x:=ledger.text_value(v);
 IF x !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$' OR substring(x,12,2)>'23' OR substring(x,15,2)>'59' OR substring(x,18,2)>'59' THEN PERFORM integration.connection_error('P2700');END IF;
 PERFORM ledger.date_value(left(x,10));RETURN x::timestamptz;
EXCEPTION WHEN SQLSTATE 'P2100' OR invalid_datetime_format OR datetime_field_overflow THEN PERFORM integration.connection_error('P2700');END $$;
-- Explicit setup grants are distinct from runtime connection consent.
CREATE FUNCTION integration.set_connect_permission(actor uuid,e uuid,b uuid,subject uuid,k text,enabled_value boolean) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE rec integration.connection_receipt;v jsonb;result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='connection_rejected'; END IF;
 PERFORM integration.connection_key(k);IF enabled_value IS NULL THEN PERFORM integration.connection_error('P2700'); END IF;
 PERFORM banking.source_operator_authorize(actor,subject,e,b,enabled_value);v:=jsonb_build_object('subject',subject,'enabled',enabled_value);
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='grant_connect' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>v THEN PERFORM integration.connection_error('P2702'); END IF;RETURN rec.result;END IF;
 INSERT INTO ledger.membership VALUES(e,subject,'connect',CASE WHEN enabled_value THEN NULL ELSE clock_timestamp() END) ON CONFLICT(entity_id,principal_id,permission) DO UPDATE SET revoked_at=EXCLUDED.revoked_at;
 result_value:=jsonb_build_object('principal_id',subject,'enabled',enabled_value);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'grant_connect',k,v,result_value);RETURN result_value;
END $$;
-- No request may choose its own sync identity; the operator configures one reviewed binding.
CREATE FUNCTION integration.set_connection_worker(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;subject uuid;login_value name;w integration.connection_worker;rec integration.connection_receipt;result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='connection_rejected'; END IF;
 v:=integration.connection_json(raw,4096);PERFORM ledger.keys(v,ARRAY['service_principal_id','worker_login','enabled']);PERFORM integration.connection_key(k);
 subject:=ledger.uuid_value(v->'service_principal_id');login_value:=integration.sync_text(v->'worker_login',63)::name;
 IF jsonb_typeof(v->'enabled')<>'boolean' OR NOT integration.sync_login(login_value) THEN PERFORM integration.connection_error('P2700'); END IF;
 PERFORM banking.source_operator_authorize(actor,subject,e,b,(v->>'enabled')::boolean);
 IF (v->>'enabled')::boolean THEN PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=subject AND permission='import' AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;END IF;
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='worker' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>v THEN PERFORM integration.connection_error('P2702');END IF;RETURN rec.result;END IF;
 INSERT INTO integration.connection_worker VALUES(e,b,subject,login_value,(v->>'enabled')::boolean,1) ON CONFLICT(entity_id,book_id) DO UPDATE SET service_principal=EXCLUDED.service_principal,worker_login=EXCLUDED.worker_login,enabled=EXCLUDED.enabled,revision=integration.connection_worker.revision+1 RETURNING * INTO w;
 result_value:=jsonb_build_object('revision',w.revision::text);INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'worker',k,v,result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_error('P2700');END $$;
-- Existing-source management is separately approved and revocable even for dormant subjects.
CREATE FUNCTION integration.set_source_bind_permission(actor uuid,e uuid,b uuid,sid uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;subject uuid;s banking.source_account;rec integration.connection_receipt;result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='connection_rejected'; END IF;
 v:=integration.connection_json(raw,4096);PERFORM ledger.keys(v,ARRAY['expected_binding_revision','principal_id','enabled']);PERFORM integration.connection_key(k);subject:=ledger.uuid_value(v->'principal_id');
 IF jsonb_typeof(v->'enabled')<>'boolean' THEN PERFORM integration.connection_error('P2700');END IF;
 PERFORM banking.source_operator_authorize(actor,subject,e,b,(v->>'enabled')::boolean);
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='source_bind' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>jsonb_build_object('source_account_id',sid,'body',v) THEN PERFORM integration.connection_error('P2702');END IF;RETURN rec.result;END IF;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR UPDATE;
 IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 IF s.binding_revision<>banking.import_revision(v->'expected_binding_revision') THEN PERFORM integration.connection_error('P2703');END IF;
 INSERT INTO banking.source_permission VALUES(e,b,sid,subject,'bind',CASE WHEN (v->>'enabled')::boolean THEN NULL ELSE clock_timestamp() END) ON CONFLICT(entity_id,book_id,source_account_id,principal_id,capability) DO UPDATE SET revoked_at=EXCLUDED.revoked_at;
 UPDATE banking.source_account SET binding_revision=binding_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=sid RETURNING * INTO s;
 result_value:=jsonb_build_object('binding_revision',s.binding_revision::text);INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'source_bind',k,jsonb_build_object('source_account_id',sid,'body',v),result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM integration.connection_error('P2700');END $$;
-- Lock principals before the shared book/source/item order used by acquisition.
CREATE FUNCTION integration.connection_authorize(actor uuid,e uuid,b uuid) RETURNS integration.connection_worker LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;id uuid;permission_value text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.connection_error('P2700');END IF;
 PERFORM set_config('lock_timeout','1s',true);
 SELECT * INTO w FROM integration.connection_worker WHERE entity_id=e AND book_id=b;
 IF NOT FOUND OR NOT w.enabled OR NOT integration.sync_login(w.worker_login) THEN PERFORM integration.connection_error('P2701');END IF;
 FOR id IN SELECT DISTINCT x FROM unnest(ARRAY[actor,w.service_principal]) x ORDER BY x LOOP PERFORM 1 FROM ledger.principal WHERE principal_id=id AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;END LOOP;
 PERFORM ledger.authorize(actor,e,b,'read',true);PERFORM ledger.authorize(actor,e,b,'connect',true);PERFORM ledger.authorize(actor,e,b,'import',true);
 FOREACH permission_value IN ARRAY ARRAY['read','import'] LOOP PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=w.service_principal AND permission=permission_value AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;END LOOP;
 SELECT * INTO w FROM integration.connection_worker WHERE entity_id=e AND book_id=b FOR SHARE;RETURN w;
EXCEPTION WHEN SQLSTATE 'P2101' OR SQLSTATE 'P2102' THEN PERFORM integration.connection_error('P2701');END $$;
-- Recheck every required source capability under its canonical scoped lock.
CREATE FUNCTION integration.connection_source(actor uuid,e uuid,b uuid,sid uuid) RETURNS banking.source_account LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE s banking.source_account;p text;
BEGIN
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR UPDATE;IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 FOREACH p IN ARRAY ARRAY['read','import','bind'] LOOP PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=actor AND capability=p AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;END LOOP;
 RETURN s;
END $$;
-- Resolve only the initiating actor's attempt after current scoped authority checks.
CREATE FUNCTION integration.connection_lock(actor uuid,e uuid,b uuid,id uuid) RETURNS integration.connection_attempt LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;a integration.connection_attempt;s uuid;
BEGIN
 w:=integration.connection_authorize(actor,e,b);
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 IF a.worker_revision<>w.revision THEN PERFORM integration.connection_error('P2703');END IF;
 FOR s IN SELECT source_account_id FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=a.sync_item_id ORDER BY source_account_id LOOP PERFORM integration.connection_source(actor,e,b,s);END LOOP;
 IF a.sync_item_id IS NOT NULL THEN PERFORM 1 FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=a.sync_item_id FOR UPDATE;END IF;
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id FOR UPDATE;RETURN a;
END $$;
-- Private projection is used only by the service and never returned directly by its HTTP handlers.
CREATE FUNCTION integration.render_connection(a integration.connection_attempt,private_value boolean) RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;
BEGIN
 v:=jsonb_build_object('attempt_id',a.attempt_id,'entity_id',a.entity_id,'book_id',a.book_id,'mode',a.mode,'state',a.state,'expires_at',a.expires_at,'sync_item_id',a.sync_item_id,'manifest',a.manifest);
 IF NOT private_value THEN IF a.streams IS NOT NULL THEN v:=v||jsonb_build_object('streams',a.streams);END IF;RETURN v;END IF;
 RETURN v||jsonb_build_object('nonce',a.nonce,'item_generation',a.item_generation::text,'expected_item_id',a.expected_item_id,'claim_id',a.claim_id,'credential_ref',a.credential_ref,'credential_version',a.credential_version::text,'previous_credential_ref',a.previous_credential_ref,'previous_credential_version',a.previous_credential_version::text,'link_token_sha256',a.link_token_sha256,'public_token_sha256',a.public_token_sha256,'execute_link',false,'execute_provider',false);
END $$;
-- Read paths reauthorize instead of exposing stale attempt metadata after revocation.
CREATE FUNCTION integration.connection_context(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$ BEGIN RETURN integration.render_connection(integration.connection_lock(actor,e,b,id),true);END $$;
CREATE FUNCTION integration.get_connection_attempt(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$ BEGIN RETURN integration.render_connection(integration.connection_lock(actor,e,b,id),false);END $$;
-- Bind user intent to actor/scope before any Link session is created.
CREATE FUNCTION integration.begin_connection(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;a integration.connection_attempt;v jsonb;i integration.sync_item;sid uuid;approved jsonb;mode_value text;
BEGIN
 w:=integration.connection_authorize(actor,e,b);PERFORM integration.connection_key(k);v:=integration.connection_json(raw,16384);
 IF NOT(v ?& ARRAY['mode','sync_item_id','expected_generation','approved_stream_ids']) OR EXISTS(SELECT FROM jsonb_object_keys(v) z WHERE z<>ALL(ARRAY['mode','sync_item_id','expected_generation','approved_stream_ids'])) THEN PERFORM integration.connection_error('P2700');END IF;
 mode_value:=ledger.text_value(v->'mode');IF mode_value NOT IN ('new','reconnect','rotate','remove') OR jsonb_typeof(v->'approved_stream_ids')<>'array' OR (CASE WHEN jsonb_typeof(v->'approved_stream_ids')='array' THEN jsonb_array_length(v->'approved_stream_ids') ELSE -1 END)>25 THEN PERFORM integration.connection_error('P2700');END IF;
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND actor_id=actor AND idempotency_key=k;
 IF FOUND THEN a:=integration.connection_lock(actor,e,b,a.attempt_id);IF a.body<>v THEN PERFORM integration.connection_error('P2702');END IF;RETURN integration.render_connection(a,true);END IF;
 IF mode_value='new' THEN IF v->'sync_item_id'<>'null'::jsonb OR v->'expected_generation'<>'null'::jsonb OR v->'approved_stream_ids'<>'[]'::jsonb THEN PERFORM integration.connection_error('P2700');END IF;
 ELSE
  SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=ledger.uuid_value(v->'sync_item_id');IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
  FOR sid IN SELECT source_account_id FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id ORDER BY source_account_id LOOP PERFORM integration.connection_source(actor,e,b,sid);END LOOP;
  SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id FOR UPDATE;
  IF i.generation<>banking.import_revision(v->'expected_generation') OR i.state='disconnected' THEN PERFORM integration.connection_error('P2703');END IF;
  SELECT coalesce(jsonb_agg(stream_id ORDER BY stream_id),'[]'::jsonb) INTO approved FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id;
  IF approved<>(SELECT coalesce(jsonb_agg(ledger.uuid_value(value) ORDER BY ledger.uuid_value(value)),'[]'::jsonb) FROM jsonb_array_elements(v->'approved_stream_ids')) THEN PERFORM integration.connection_error('P2703');END IF;
 END IF;
 INSERT INTO integration.connection_attempt(entity_id,book_id,actor_id,idempotency_key,mode,state,body,worker_revision,sync_item_id,item_generation,expected_item_id,previous_credential_ref,previous_credential_version) VALUES(e,b,actor,k,mode_value,'link_pending',v,w.revision,i.sync_item_id,i.generation,i.item_id,i.credential_ref,i.credential_version) RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,a.attempt_id,'begin',v);
 RETURN integration.render_connection(a,true)||jsonb_build_object('execute_link',mode_value IN ('new','reconnect'));
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM integration.connection_error('P2700');END $$;
-- The server records only a Link-session fingerprint, never the Link token itself.
CREATE FUNCTION integration.record_link_session(actor uuid,e uuid,b uuid,id uuid,nonce_value uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;expiry timestamptz;hash_value text;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);v:=integration.connection_json(raw,2048);PERFORM ledger.keys(v,ARRAY['link_token_sha256','expires_at']);hash_value:=ledger.text_value(v->'link_token_sha256');
 IF hash_value !~ '^[0-9a-f]{64}$' OR ledger.text_value(v->'expires_at') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$' THEN PERFORM integration.connection_error('P2700');END IF;expiry:=integration.connection_expiry(v->'expires_at');
 IF a.nonce IS DISTINCT FROM nonce_value OR a.mode NOT IN ('new','reconnect') OR clock_timestamp()>=a.expires_at OR expiry<=clock_timestamp() THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.link_token_sha256 IS NOT NULL THEN IF a.link_token_sha256<>hash_value THEN PERFORM integration.connection_error('P2702');END IF;RETURN integration.render_connection(a,true);END IF;
 IF a.state<>'link_pending' THEN PERFORM integration.connection_error('P2704');END IF;
 UPDATE integration.connection_attempt SET link_token_sha256=hash_value,expires_at=least(expires_at,expiry),state='link_ready' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'link_ready',v);RETURN integration.render_connection(a,true);
EXCEPTION WHEN SQLSTATE 'P2100' OR invalid_datetime_format OR datetime_field_overflow THEN PERFORM integration.connection_error('P2700');END $$;
-- Fence all sibling acquisitions before an operation that can invalidate a shared credential.
CREATE FUNCTION integration.connection_fence(e uuid,b uuid,id uuid,state_value text) RETURNS integration.sync_item LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE i integration.sync_item;
BEGIN
 UPDATE integration.sync_item SET state=state_value,generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=id RETURNING * INTO i;
 UPDATE integration.sync_set x SET state='abandoned',reason='item_changed' FROM integration.sync_stream t WHERE t.entity_id=e AND t.book_id=b AND t.sync_item_id=id AND x.entity_id=e AND x.book_id=b AND x.set_id=t.current_set AND x.state='staging';
 UPDATE integration.sync_stream SET current_set=NULL,lease_until=NULL WHERE entity_id=e AND book_id=b AND sync_item_id=id;RETURN i;
END $$;
-- execute_provider is a one-response permission: replay never renews a remote side effect.
CREATE FUNCTION integration.claim_connection(actor uuid,e uuid,b uuid,id uuid,nonce_value uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;i integration.sync_item;link_hash text;public_hash text;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);v:=integration.connection_json(raw,2048);
 IF NOT(v ?& ARRAY['link_token_sha256','public_token_sha256']) OR EXISTS(SELECT FROM jsonb_object_keys(v) z WHERE z<>ALL(ARRAY['link_token_sha256','public_token_sha256'])) THEN PERFORM integration.connection_error('P2700');END IF;
 IF v->'link_token_sha256'<>'null'::jsonb THEN link_hash:=ledger.text_value(v->'link_token_sha256');IF link_hash !~ '^[0-9a-f]{64}$' THEN PERFORM integration.connection_error('P2700');END IF;END IF;
 IF v->'public_token_sha256'<>'null'::jsonb THEN public_hash:=ledger.text_value(v->'public_token_sha256');IF public_hash !~ '^[0-9a-f]{64}$' THEN PERFORM integration.connection_error('P2700');END IF;END IF;
 IF a.nonce IS DISTINCT FROM nonce_value THEN PERFORM integration.connection_error('P2701');END IF;
 IF a.claim_id IS NOT NULL THEN IF a.link_token_sha256 IS DISTINCT FROM link_hash OR a.public_token_sha256 IS DISTINCT FROM public_hash THEN PERFORM integration.connection_error('P2702');END IF;RETURN integration.render_connection(a,true);END IF;
 IF clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.mode IN ('new','reconnect') THEN IF a.state<>'link_ready' OR link_hash IS NULL OR link_hash<>a.link_token_sha256 OR (a.mode='new')<>(public_hash IS NOT NULL) THEN PERFORM integration.connection_error('P2704');END IF;
 ELSE IF a.state<>'link_pending' OR link_hash IS NOT NULL OR public_hash IS NOT NULL THEN PERFORM integration.connection_error('P2704');END IF;END IF;
 IF a.sync_item_id IS NOT NULL THEN
  i:=integration.connection_item_pin(a);
  IF i.generation<>a.item_generation OR i.state='disconnected' THEN PERFORM integration.connection_error('P2703');END IF;
  i:=integration.connection_fence(e,b,i.sync_item_id,CASE WHEN a.mode='remove' THEN 'disconnect_pending' ELSE 'credential_recovery_required' END);
 END IF;
 UPDATE integration.connection_attempt SET claim_id=gen_random_uuid(),credential_ref=CASE WHEN mode IN ('reconnect','remove') THEN previous_credential_ref ELSE gen_random_uuid() END,credential_version=CASE WHEN mode='new' THEN 1 WHEN mode='rotate' THEN previous_credential_version+1 ELSE previous_credential_version END,public_token_sha256=public_hash,item_generation=i.generation,state=CASE WHEN mode='remove' THEN 'disconnect_pending' ELSE 'exchange_claimed' END WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'claim',jsonb_build_object('claim_id',a.claim_id));RETURN integration.render_connection(a,true)||jsonb_build_object('execute_provider',true);
EXCEPTION WHEN SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');WHEN unique_violation THEN PERFORM integration.connection_error('P2702');END $$;
-- Manifest is trusted server verification evidence, not browser-supplied account metadata.
CREATE FUNCTION integration.record_connection_manifest(actor uuid,e uuid,b uuid,id uuid,claim_value uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;r jsonb;expiry timestamptz;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);v:=integration.connection_json(raw,32768);PERFORM ledger.keys(v,ARRAY['item_id','accounts','consent_expires_at']);PERFORM integration.sync_text(v->'item_id',256);
 IF jsonb_typeof(v->'accounts')<>'array' OR (CASE WHEN jsonb_typeof(v->'accounts')='array' THEN jsonb_array_length(v->'accounts') ELSE -1 END) NOT BETWEEN 1 AND 25 OR ledger.text_value(v->'consent_expires_at') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$' THEN PERFORM integration.connection_error('P2700');END IF;
 expiry:=integration.connection_expiry(v->'consent_expires_at');IF expiry<=clock_timestamp() THEN PERFORM integration.connection_error('P2703');END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(v->'accounts') LOOP
  IF jsonb_typeof(r) IS DISTINCT FROM 'object' THEN PERFORM integration.connection_error('P2700');END IF;
  IF NOT(r ?& ARRAY['account_id','type','iso_currency_code']) OR EXISTS(SELECT FROM jsonb_object_keys(r) z WHERE z<>ALL(ARRAY['account_id','type','iso_currency_code'])) THEN PERFORM integration.connection_error('P2700');END IF;
  PERFORM integration.sync_text(r->'account_id',256);IF ledger.text_value(r->'type') NOT IN ('depository','credit','loan','investment','other') OR r->'iso_currency_code'<>'null'::jsonb AND ledger.text_value(r->'iso_currency_code') !~ '^[A-Z]{3}$' THEN PERFORM integration.connection_error('P2700');END IF;
 END LOOP;
 IF EXISTS(SELECT FROM jsonb_array_elements(v->'accounts') z GROUP BY value->>'account_id' HAVING count(*)>1) THEN PERFORM integration.connection_error('P2700');END IF;
 IF a.claim_id IS DISTINCT FROM claim_value OR claim_value IS NULL OR a.mode='remove' THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.manifest IS NOT NULL THEN IF a.manifest<>v THEN PERFORM integration.connection_error('P2702');END IF;RETURN integration.render_connection(a,true);END IF;
 IF a.state NOT IN ('exchange_claimed','credential_recovery_required') THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.expected_item_id IS NOT NULL AND a.expected_item_id<>v->>'item_id' THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.sync_item_id IS NOT NULL THEN
  PERFORM integration.connection_item_pin(a);
  IF EXISTS(SELECT FROM integration.sync_stream t JOIN banking.source_account s USING(entity_id,book_id,source_account_id) JOIN reference.currency c ON c.currency_version_id=s.currency_version_id WHERE t.entity_id=e AND t.book_id=b AND t.sync_item_id=a.sync_item_id AND t.state='active' AND NOT EXISTS(SELECT FROM jsonb_array_elements(v->'accounts') z WHERE z.value->>'account_id'=t.account_id AND z.value->>'type'='depository' AND z.value->>'iso_currency_code'=c.alphabetic_code)) THEN PERFORM integration.connection_error('P2704');END IF;
 END IF;
 UPDATE integration.connection_attempt SET manifest=v,state='credential_pending' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'manifest',v);RETURN integration.render_connection(a,true);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2600' OR invalid_datetime_format OR datetime_field_overflow THEN PERFORM integration.connection_error('P2700');END $$;
-- A recorded failure cannot reactivate an old credential or grant a second exchange.
CREATE FUNCTION integration.connection_failure(actor uuid,e uuid,b uuid,id uuid,claim_value uuid,reason_value text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF reason_value IS NULL OR reason_value NOT IN ('provider_unknown','secret_unknown','provider_rejected','manifest_rejected','sql_unknown') THEN PERFORM integration.connection_error('P2700');END IF;
 IF claim_value IS NULL OR a.claim_id IS DISTINCT FROM claim_value OR a.state IN ('active','disconnected','binding_review') THEN PERFORM integration.connection_error('P2703');END IF;
 UPDATE integration.connection_attempt SET state=CASE WHEN mode='remove' THEN 'disconnect_pending' ELSE 'credential_recovery_required' END,reason=reason_value WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'uncertainty',jsonb_build_object('reason',reason_value));RETURN integration.render_connection(a,true);
END $$;
-- Recheck the exact approved sibling set as well as generation before remote-result publication.
CREATE FUNCTION integration.connection_item_pin(a integration.connection_attempt) RETURNS integration.sync_item LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE i integration.sync_item;ids jsonb;
BEGIN
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=a.entity_id AND book_id=a.book_id AND sync_item_id=a.sync_item_id;
 IF NOT FOUND OR i.generation IS DISTINCT FROM a.item_generation OR i.state='disconnected' THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.mode<>'new' THEN
  SELECT coalesce(jsonb_agg(stream_id ORDER BY stream_id),'[]'::jsonb) INTO ids FROM integration.sync_stream WHERE entity_id=a.entity_id AND book_id=a.book_id AND sync_item_id=a.sync_item_id;
  IF ids<>(SELECT coalesce(jsonb_agg(ledger.uuid_value(value) ORDER BY ledger.uuid_value(value)),'[]'::jsonb) FROM jsonb_array_elements(a.body->'approved_stream_ids')) THEN PERFORM integration.connection_error('P2703');END IF;
 END IF;RETURN i;
END $$;
-- Credential acknowledgement publishes only the SQL-pinned reference and immutable manifest.
CREATE FUNCTION integration.confirm_connection_credential(actor uuid,e uuid,b uuid,id uuid,claim_value uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
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
-- Explicit bindings are the only route from verified account metadata into acquisition authority.
CREATE FUNCTION integration.bind_connection(actor uuid,e uuid,b uuid,id uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;w integration.connection_worker;i integration.sync_item;v jsonb;r jsonb;provider_row jsonb;s banking.source_account;la ledger.account;c reference.currency;ns uuid;stream_value uuid;d date;cid uuid;lid uuid;sid uuid;new_source boolean;result_value jsonb:='[]';p uuid;cap text;rec integration.connection_receipt;
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
  IF NOT(r ?& ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref']) OR EXISTS(SELECT FROM jsonb_object_keys(r) z WHERE z<>ALL(ARRAY['account_id','ledger_account_id','currency_version_id','source_account_id','expected_binding_revision','binding_date','binding_evidence_ref'])) THEN PERFORM integration.connection_error('P2700');END IF;
  PERFORM integration.sync_text(r->'account_id',256);cid:=ledger.uuid_value(r->'currency_version_id');lid:=ledger.uuid_value(r->'ledger_account_id');d:=ledger.date_value(ledger.text_value(r->'binding_date'));IF ledger.uuid_value(r->'binding_evidence_ref')='00000000-0000-0000-0000-000000000000' THEN PERFORM integration.connection_error('P2700');END IF;
  SELECT value INTO provider_row FROM jsonb_array_elements(a.manifest->'accounts') WHERE value->>'account_id'=r->>'account_id';
  IF NOT FOUND OR provider_row->>'type'<>'depository' OR provider_row->'iso_currency_code'='null'::jsonb THEN PERFORM integration.connection_error('P2704');END IF;
  SELECT * INTO c FROM reference.currency WHERE currency_version_id=cid;
  IF NOT FOUND OR c.alphabetic_code<>provider_row->>'iso_currency_code' OR c.minor_unit IS NULL OR d<c.valid_from OR c.valid_to IS NOT NULL AND d>=c.valid_to THEN PERFORM integration.connection_error('P2704');END IF;
  SELECT * INTO la FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id=lid FOR SHARE;
  IF NOT FOUND OR la.category<>'asset' OR la.normal_side<>'debit' OR NOT la.monetary OR d<la.active_from OR la.active_to IS NOT NULL AND d>=la.active_to THEN PERFORM integration.connection_error('P2704');END IF;
  new_source:=r->'source_account_id'='null'::jsonb;
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
-- Removing one account never touches provider Item access or sibling cursor/lease state.
CREATE FUNCTION integration.disconnect_stream(actor uuid,e uuid,b uuid,id uuid,k text,expected bigint) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;t integration.sync_stream;s banking.source_account;rec integration.connection_receipt;v jsonb;result_value jsonb;
BEGIN
 w:=integration.connection_authorize(actor,e,b);PERFORM integration.connection_key(k);
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id;IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;s:=integration.connection_source(actor,e,b,t.source_account_id);
 v:=jsonb_build_object('stream_id',id,'expected_binding_revision',expected::text);
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='disconnect' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>v THEN PERFORM integration.connection_error('P2702');END IF;RETURN rec.result;END IF;
 IF expected IS NULL OR s.binding_revision<>expected THEN PERFORM integration.connection_error('P2703');END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id FOR UPDATE;
 UPDATE integration.sync_set SET state='abandoned',reason='account_disconnected' WHERE entity_id=e AND book_id=b AND set_id=t.current_set AND state='staging';
 UPDATE integration.sync_stream SET state='disconnected',current_set=NULL,lease_until=NULL WHERE entity_id=e AND book_id=b AND stream_id=id;
 result_value:=jsonb_build_object('stream_id',id,'state','disconnected');INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'disconnect',k,v,result_value);RETURN result_value;
END $$;
-- Remote removal becomes confirmed only through the exact already-claimed operation.
CREATE FUNCTION integration.confirm_item_removed(actor uuid,e uuid,b uuid,id uuid,claim_value uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);IF a.mode<>'remove' OR claim_value IS NULL OR a.claim_id IS DISTINCT FROM claim_value THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.state='disconnected' THEN RETURN integration.render_connection(a,false);END IF;
 IF a.state<>'disconnect_pending' THEN PERFORM integration.connection_error('P2704');END IF;i:=integration.connection_item_pin(a);
 UPDATE integration.sync_item SET state='disconnected',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
 UPDATE integration.sync_stream SET state='disconnected' WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id;
 UPDATE integration.connection_attempt SET state='disconnected',item_generation=i.generation WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'item_removed','{}');RETURN integration.render_connection(a,false);
END $$;
-- Connection management listing requires all exposed account bindings to remain authorized.
CREATE FUNCTION integration.list_connections(actor uuid,e uuid,b uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;i integration.sync_item;t integration.sync_stream;s banking.source_account;items jsonb:='[]';accounts jsonb;n integer:=0;
BEGIN
 w:=integration.connection_authorize(actor,e,b);
 FOR i IN SELECT * FROM integration.sync_item WHERE entity_id=e AND book_id=b ORDER BY sync_item_id LIMIT 101 LOOP
  n:=n+1;IF n>100 THEN PERFORM integration.connection_error('P2705');END IF;accounts:='[]';
  FOR t IN SELECT * FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id ORDER BY stream_id LOOP
   s:=integration.connection_source(actor,e,b,t.source_account_id);accounts:=accounts||jsonb_build_array(jsonb_build_object('stream_id',t.stream_id,'account_id',t.account_id,'source_account_id',t.source_account_id,'state',t.state,'binding_revision',s.binding_revision::text));
  END LOOP;
  IF jsonb_array_length(accounts)>25 THEN PERFORM integration.connection_error('P2705');END IF;
  items:=items||jsonb_build_array(jsonb_build_object('sync_item_id',i.sync_item_id,'item_id',i.item_id,'state',i.state,'item_generation',i.generation::text,'accounts',accounts));
 END LOOP;RETURN jsonb_build_object('items',items);
END $$;
-- Worker metadata is scoped by the current live claim immediately before external I/O.
CREATE FUNCTION integration.sync_context(e uuid,b uuid,id uuid,set_value uuid,f bigint) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;x integration.sync_set;i integration.sync_item;s banking.source_account;c reference.currency;
BEGIN
 t:=integration.sync_authorize(e,b,id);x:=integration.sync_attempt(t,set_value,f);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id;SELECT * INTO c FROM reference.currency WHERE currency_version_id=s.currency_version_id;
 RETURN jsonb_build_object('stream_id',id,'set_id',set_value,'fence',x.fence::text,'provider',i.provider,'environment',i.environment,'item_id',i.item_id,'account_id',t.account_id,'item_generation',i.generation::text,'credential_ref',i.credential_ref,'credential_version',i.credential_version::text,'binding_revision',x.binding_revision::text,'starting_cursor',x.starting_cursor,'request_cursor',x.continuation_cursor,'next_page_index',x.page_count,'page_count',x.page_count,'byte_count',x.byte_count,'change_count',x.change_count,'final_page',x.final_page,'lease_until',t.lease_until,'currency',jsonb_build_object('code',c.alphabetic_code,'version_id',c.currency_version_id,'minor_unit',c.minor_unit,'valid_from',to_char(c.valid_from,'YYYY-MM-DD'),'valid_to',to_char(c.valid_to,'YYYY-MM-DD')),'limits',jsonb_build_object('max_pages',10,'max_changes',1000,'max_bytes',2097152,'max_page_changes',500));
END $$;
-- Stable claim lookup resolves ambiguous commits without granting another provider operation.
CREATE FUNCTION integration.sync_resume(e uuid,b uuid,id uuid,k uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;x integration.sync_set;result_value jsonb;
BEGIN
 t:=integration.sync_authorize(e,b,id);IF k IS NULL THEN PERFORM integration.sync_error('P2600');END IF;
 SELECT * INTO x FROM integration.sync_set WHERE entity_id=e AND book_id=b AND stream_id=id AND claim_key=k;
 IF NOT FOUND THEN RETURN jsonb_build_object('state','absent','set_id',NULL,'fence',NULL,'reason',NULL,'completion',NULL);END IF;
 SELECT result INTO result_value FROM integration.sync_completion WHERE entity_id=e AND book_id=b AND set_id=x.set_id;
 RETURN jsonb_build_object('state',x.state,'set_id',x.set_id,'fence',x.fence::text,'reason',x.reason,'completion',result_value);
END $$;
-- Guard stored attempt identity/reference evidence; an uncertain claim is never reset.
CREATE FUNCTION integration.connection_attempt_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.connection_error('P2702');END IF;
 IF ROW(NEW.entity_id,NEW.book_id,NEW.attempt_id,NEW.actor_id,NEW.idempotency_key,NEW.mode,NEW.body,NEW.worker_revision,NEW.nonce,NEW.expected_item_id,NEW.previous_credential_ref,NEW.previous_credential_version) IS DISTINCT FROM ROW(OLD.entity_id,OLD.book_id,OLD.attempt_id,OLD.actor_id,OLD.idempotency_key,OLD.mode,OLD.body,OLD.worker_revision,OLD.nonce,OLD.expected_item_id,OLD.previous_credential_ref,OLD.previous_credential_version)
 OR NEW.expires_at>OLD.expires_at OR OLD.claim_id IS NOT NULL AND ROW(NEW.claim_id,NEW.credential_ref,NEW.credential_version,NEW.public_token_sha256) IS DISTINCT FROM ROW(OLD.claim_id,OLD.credential_ref,OLD.credential_version,OLD.public_token_sha256)
 OR OLD.sync_item_id IS NOT NULL AND NEW.sync_item_id IS DISTINCT FROM OLD.sync_item_id OR OLD.manifest IS NOT NULL AND NEW.manifest IS DISTINCT FROM OLD.manifest OR OLD.link_token_sha256 IS NOT NULL AND NEW.link_token_sha256 IS DISTINCT FROM OLD.link_token_sha256 OR OLD.state IN ('active','disconnected') THEN PERFORM integration.connection_error('P2702');END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_connection BEFORE UPDATE OR DELETE ON integration.connection_attempt FOR EACH ROW EXECUTE FUNCTION integration.connection_attempt_guard();

-- Forward lifecycle extension preserves provider identity while credential versions advance.
CREATE OR REPLACE FUNCTION integration.sync_state_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.sync_error('P2602'); END IF;
 IF TG_TABLE_NAME='sync_item' THEN
  IF (to_jsonb(NEW)-ARRAY['generation','state','credential_ref','credential_version','consent_principal','consent_evidence_ref','consent_expires_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['generation','state','credential_ref','credential_version','consent_principal','consent_evidence_ref','consent_expires_at']) OR NEW.generation::numeric<>OLD.generation::numeric+1 OR OLD.state='disconnected' OR NEW.credential_version<OLD.credential_version OR NEW.credential_version>OLD.credential_version+1 OR NEW.credential_ref<>OLD.credential_ref AND NEW.credential_version<>OLD.credential_version+1 THEN PERFORM integration.sync_error('P2602'); END IF;
 ELSIF TG_TABLE_NAME='sync_stream' THEN
  IF (to_jsonb(NEW)-ARRAY['requested_generation','completed_generation','cursor_value','fence','lease_until','current_set','last_completed_at','readiness','state']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['requested_generation','completed_generation','cursor_value','fence','lease_until','current_set','last_completed_at','readiness','state']) OR NEW.requested_generation<OLD.requested_generation OR NEW.completed_generation<OLD.completed_generation OR NEW.fence<OLD.fence OR OLD.state='disconnected' AND NEW.state<>'disconnected' THEN PERFORM integration.sync_error('P2602'); END IF;
 ELSE
  IF (to_jsonb(NEW)-ARRAY['state','reason','page_count','change_count','byte_count','final_page','continuation_cursor']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','reason','page_count','change_count','byte_count','final_page','continuation_cursor']) OR OLD.state<>'staging' OR NEW.page_count<OLD.page_count OR NEW.change_count<OLD.change_count OR NEW.byte_count<OLD.byte_count THEN PERFORM integration.sync_error('P2602'); END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION integration.sync_authorize(e uuid,b uuid,id uuid) RETURNS integration.sync_stream LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream; i integration.sync_item;
BEGIN
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id;
 IF NOT FOUND OR t.state<>'active' OR t.worker_login<>session_user OR NOT integration.sync_login(session_user) THEN PERFORM integration.sync_error('P2601'); END IF;
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 PERFORM integration.sync_authority(e,b,t.source_account_id,i.consent_principal,t.service_principal);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id FOR UPDATE;
 IF i.state<>'active' OR clock_timestamp()>=i.consent_expires_at THEN PERFORM integration.sync_error('P2601'); END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id FOR UPDATE;
 PERFORM 1 FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=t.namespace_id AND state='active'; IF NOT FOUND THEN PERFORM integration.sync_error('P2601'); END IF;
 RETURN t;
END $$;
REVOKE ALL ON FUNCTION integration.connection_error(text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_json(text,integer) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_key(text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.set_connect_permission(uuid,uuid,uuid,uuid,text,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.set_connection_worker(uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.set_source_bind_permission(uuid,uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_authorize(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_source(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_lock(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.render_connection(integration.connection_attempt,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_context(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.get_connection_attempt(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.record_link_session(uuid,uuid,uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_fence(uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.claim_connection(uuid,uuid,uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.record_connection_manifest(uuid,uuid,uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_failure(uuid,uuid,uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_item_pin(integration.connection_attempt) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.confirm_connection_credential(uuid,uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.bind_connection(uuid,uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.disconnect_stream(uuid,uuid,uuid,uuid,text,bigint) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.confirm_item_removed(uuid,uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.list_connections(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_context(uuid,uuid,uuid,uuid,bigint) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_resume(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.connection_attempt_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON integration.connection_worker,integration.connection_attempt,integration.connection_receipt,integration.connection_event,integration.credential_publication FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
GRANT USAGE ON SCHEMA integration TO bookkeeping_api;
GRANT EXECUTE ON FUNCTION integration.connection_context(uuid,uuid,uuid,uuid),integration.get_connection_attempt(uuid,uuid,uuid,uuid),integration.begin_connection(uuid,uuid,uuid,text,text),integration.record_link_session(uuid,uuid,uuid,uuid,uuid,text),integration.claim_connection(uuid,uuid,uuid,uuid,uuid,text),integration.record_connection_manifest(uuid,uuid,uuid,uuid,uuid,text),integration.connection_failure(uuid,uuid,uuid,uuid,uuid,text),integration.confirm_connection_credential(uuid,uuid,uuid,uuid,uuid),integration.bind_connection(uuid,uuid,uuid,uuid,text,text),integration.disconnect_stream(uuid,uuid,uuid,uuid,text,bigint),integration.confirm_item_removed(uuid,uuid,uuid,uuid,uuid),integration.list_connections(uuid,uuid,uuid) TO bookkeeping_api;
GRANT EXECUTE ON FUNCTION integration.sync_context(uuid,uuid,uuid,uuid,bigint),integration.sync_resume(uuid,uuid,uuid,uuid) TO bookkeeping_sync;
REVOKE ALL ON FUNCTION integration.connection_expiry(jsonb) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
COMMIT;
