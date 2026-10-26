-- Disposable/quiesced rollback only; never remove Production receipt or repair provenance.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN
 IF EXISTS(SELECT FROM integration.webhook_login_profile WHERE environment='production')
 OR EXISTS(SELECT FROM integration.webhook_ingress WHERE environment='production')
 OR EXISTS(SELECT FROM integration.webhook_receipt WHERE environment='production')
 OR EXISTS(SELECT FROM integration.link_session_receipt WHERE environment='production')
 OR EXISTS(SELECT FROM integration.link_repair_claim WHERE environment='production')
 OR EXISTS(SELECT FROM integration.connection_attempt WHERE environment='production' AND mode='reconnect')
 THEN RAISE EXCEPTION USING ERRCODE='P2802',MESSAGE='production_webhook_history_preserved';END IF;
END $$;
CREATE OR REPLACE FUNCTION integration.webhook_login(login_value name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolcanlogin AND NOT(rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication))
 AND pg_has_role(login_value,'bookkeeping_webhook','MEMBER')
 AND NOT EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_identity_operator','bookkeeping_migrator','bookkeeping_sync','bookkeeping_publisher','bookkeeping_observer','cloudsqlsuperuser')) AND pg_has_role(login_value,oid,'MEMBER'))
$$;

CREATE OR REPLACE FUNCTION integration.set_webhook_ingress(login_value name,env text,k text,enabled_value boolean) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;r integration.webhook_operator_receipt;g bigint;result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='connection_operation_rejected';END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR env IS DISTINCT FROM 'sandbox' OR enabled_value IS NULL OR k IS NULL OR k!~'^[A-Za-z0-9._:-]{1,128}$' OR login_value IS NULL THEN PERFORM integration.connection_ops_error('P2800');END IF;
 IF enabled_value AND NOT integration.webhook_login(login_value) THEN PERFORM integration.connection_ops_error('P2801');END IF;
 PERFORM pg_advisory_xact_lock(28014,1);
 v:=jsonb_build_object('login',login_value,'environment',env,'enabled',enabled_value);
 SELECT * INTO r FROM integration.webhook_operator_receipt WHERE database_actor=session_user AND idempotency_key=k;
 IF FOUND THEN IF r.body<>v THEN PERFORM integration.connection_ops_error('P2802');END IF;RETURN r.result;END IF;
 INSERT INTO integration.webhook_ingress VALUES(env,login_value,enabled_value,1) ON CONFLICT(environment) DO UPDATE SET login_name=EXCLUDED.login_name,enabled=EXCLUDED.enabled,revision=integration.webhook_ingress.revision+1 RETURNING revision INTO g;
 result_value:=jsonb_build_object('revision',g::text);INSERT INTO integration.webhook_operator_receipt VALUES(session_user,k,v,result_value,clock_timestamp());RETURN result_value;
END $$;

CREATE OR REPLACE FUNCTION integration.accept_verified_webhook(env text,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;i integration.sync_item;t integration.sync_stream;rec integration.webhook_receipt;ids uuid[];principal_value uuid;issue_time numeric;replay boolean;rid uuid:=gen_random_uuid();
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.connection_ops_error('P2800');END IF;
 IF NOT integration.webhook_login(session_user) THEN PERFORM integration.connection_ops_error('P2801');END IF;
 PERFORM 1 FROM integration.webhook_ingress WHERE environment=env AND login_name=session_user AND enabled FOR SHARE;IF NOT FOUND THEN PERFORM integration.connection_ops_error('P2801');END IF;
 IF env IS DISTINCT FROM 'sandbox' OR k IS NULL OR k!~'^[0-9a-f]{64}$' THEN PERFORM integration.connection_ops_error('P2800');END IF;
 v:=integration.connection_json(raw,2048);PERFORM ledger.keys(v,ARRAY['item_id','body_sha256','issued_at','webhook_type','webhook_code']);PERFORM integration.sync_text(v->'item_id',256);
 IF jsonb_typeof(v->'body_sha256') IS DISTINCT FROM 'string' OR v->>'body_sha256'!~'^[0-9a-f]{64}$' OR jsonb_typeof(v->'issued_at') IS DISTINCT FROM 'number' OR v->>'issued_at'!~'^[0-9]{1,12}$'
 OR v->'webhook_type' IS DISTINCT FROM '"TRANSACTIONS"'::jsonb OR v->'webhook_code' IS DISTINCT FROM '"SYNC_UPDATES_AVAILABLE"'::jsonb THEN PERFORM integration.connection_ops_error('P2800');END IF;
 SELECT * INTO i FROM integration.sync_item WHERE provider='plaid' AND environment=env AND item_id=v->>'item_id';IF NOT FOUND THEN PERFORM integration.connection_ops_error('P2801');END IF;
 SELECT * INTO rec FROM integration.webhook_receipt WHERE environment=env AND receipt_key=k;replay:=FOUND;
 IF replay THEN
  IF rec.body<>v OR ROW(rec.entity_id,rec.book_id,rec.sync_item_id) IS DISTINCT FROM ROW(i.entity_id,i.book_id,i.sync_item_id) THEN PERFORM integration.connection_ops_error('P2802');END IF;
  ids:=rec.stream_ids;rid:=rec.receipt_id;
 ELSE
  issue_time:=(v->>'issued_at')::numeric;
  IF issue_time<extract(epoch FROM clock_timestamp())-300 OR issue_time>extract(epoch FROM clock_timestamp())+60 THEN PERFORM integration.connection_ops_error('P2803');END IF;
  SELECT array_agg(q.stream_id ORDER BY q.stream_id) INTO ids FROM (SELECT x.stream_id FROM integration.sync_stream x WHERE x.entity_id=i.entity_id AND x.book_id=i.book_id AND x.sync_item_id=i.sync_item_id AND integration.signal_eligible(x) ORDER BY x.stream_id LIMIT 26) q;
 END IF;
 IF ids IS NULL OR cardinality(ids)=0 THEN PERFORM integration.connection_ops_error('P2801');END IF;IF cardinality(ids)>25 THEN PERFORM integration.connection_ops_error('P2805');END IF;
 -- Prelock all principals before any scope lock, even when siblings have distinct service principals.
 FOR principal_value IN SELECT DISTINCT p FROM (SELECT i.consent_principal p UNION ALL SELECT x.service_principal FROM integration.sync_stream x WHERE x.entity_id=i.entity_id AND x.book_id=i.book_id AND x.stream_id=ANY(ids)) q ORDER BY p LOOP PERFORM 1 FROM ledger.principal WHERE principal_id=principal_value AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.connection_ops_error('P2801');END IF;END LOOP;
 FOR t IN SELECT x.* FROM integration.sync_stream x WHERE x.entity_id=i.entity_id AND x.book_id=i.book_id AND x.stream_id=ANY(ids) ORDER BY x.source_account_id,x.stream_id LOOP
  t:=integration.signal_authorize(i.entity_id,i.book_id,t.stream_id);
  IF NOT replay THEN PERFORM integration.signal_sync(t,'webhook:'||k,'webhook');END IF;
 END LOOP;
 IF NOT replay THEN INSERT INTO integration.webhook_receipt VALUES(env,k,rid,i.entity_id,i.book_id,i.sync_item_id,v,ids,clock_timestamp());END IF;
 RETURN jsonb_build_object('state',CASE WHEN replay THEN 'duplicate' ELSE 'accepted' END,'receipt_id',rid,'stream_count',cardinality(ids));
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2700' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_ops_error('P2800');END $$;

CREATE OR REPLACE FUNCTION integration.owned_repair_scope(a integration.connection_attempt)
RETURNS integration.sync_item LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE i integration.sync_item;t integration.sync_stream;w integration.connection_worker;
BEGIN
 IF a.mode<>'reconnect' OR (a.claim_id IS NULL AND clock_timestamp()>=integration.owned_repair_deadline(a)) THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.claim_id IS NOT NULL AND NOT EXISTS(SELECT FROM integration.link_repair_claim
 WHERE entity_id=a.entity_id AND book_id=a.book_id AND attempt_id=a.attempt_id AND claim_id=a.claim_id
 AND credential_ref=a.previous_credential_ref AND credential_version=a.previous_credential_version)
 THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.manifest IS NOT NULL AND (a.manifest->>'consent_expires_at')::timestamptz<=clock_timestamp()
 THEN PERFORM integration.connection_error('P2703');END IF;
 i:=integration.connection_item_pin(a);
 IF i.provider<>'plaid' OR i.environment<>'sandbox' OR i.item_id IS DISTINCT FROM a.expected_item_id
 OR i.credential_ref IS DISTINCT FROM a.previous_credential_ref OR i.credential_version IS DISTINCT FROM a.previous_credential_version
 OR clock_timestamp()>=i.consent_expires_at THEN PERFORM integration.connection_error('P2703');END IF;
 IF i.consent_principal IS DISTINCT FROM a.actor_id THEN PERFORM integration.connection_error('P2701');END IF;
 IF a.repair_scope_sha256 IS NULL OR a.repair_scope_sha256 IS DISTINCT FROM integration.owned_repair_scope_digest(a)
 OR (SELECT count(*) FROM integration.sync_stream WHERE entity_id=a.entity_id AND book_id=a.book_id AND sync_item_id=a.sync_item_id) NOT BETWEEN 1 AND 25
 THEN PERFORM integration.connection_error('P2703');END IF;
 SELECT * INTO w FROM integration.connection_worker WHERE entity_id=a.entity_id AND book_id=a.book_id;
 FOR t IN SELECT * FROM integration.sync_stream WHERE entity_id=a.entity_id AND book_id=a.book_id
 AND sync_item_id=a.sync_item_id ORDER BY source_account_id,stream_id LOOP
  IF t.state='disconnected' OR t.worker_login<>w.worker_login OR t.service_principal<>w.service_principal
  THEN PERFORM integration.connection_error('P2703');END IF;
  PERFORM integration.sync_authority(a.entity_id,a.book_id,t.source_account_id,a.actor_id,w.service_principal);
  PERFORM integration.sync_authority(a.entity_id,a.book_id,t.source_account_id,i.consent_principal,t.service_principal);
 END LOOP;
 RETURN i;
EXCEPTION WHEN SQLSTATE 'P2601' THEN PERFORM integration.connection_error('P2701');END $$;

CREATE OR REPLACE FUNCTION integration.accept_verified_link_session(env text,k text,raw text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;a integration.connection_attempt;r integration.link_session_receipt;n bigint;h text;s text;body_hash text;outcome text;issued bigint;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.connection_ops_error('P2800');END IF;
 IF NOT integration.webhook_login(session_user) THEN PERFORM integration.connection_ops_error('P2801');END IF;
 IF env IS DISTINCT FROM 'sandbox' OR k IS NULL OR k!~'^[0-9a-f]{64}$' THEN PERFORM integration.connection_ops_error('P2800');END IF;
 PERFORM 1 FROM integration.webhook_ingress WHERE environment=env AND login_name=session_user AND enabled FOR SHARE;
 IF NOT FOUND THEN PERFORM integration.connection_ops_error('P2801');END IF;
 v:=integration.connection_json(raw,2048);
 PERFORM ledger.keys(v,ARRAY['link_token_sha256','link_session_sha256','body_sha256','status','issued_at']);
 h:=ledger.text_value(v->'link_token_sha256');s:=ledger.text_value(v->'link_session_sha256');body_hash:=ledger.text_value(v->'body_sha256');outcome:=ledger.text_value(v->'status');
 IF h!~'^[0-9a-f]{64}$' OR s!~'^[0-9a-f]{64}$' OR body_hash!~'^[0-9a-f]{64}$' OR outcome NOT IN ('success','exited')
 OR jsonb_typeof(v->'issued_at') IS DISTINCT FROM 'number' OR v->>'issued_at'!~'^[1-9][0-9]{0,9}$'
 THEN PERFORM integration.connection_ops_error('P2800');END IF;
 issued:=(v->>'issued_at')::bigint;
 IF issued<extract(epoch FROM clock_timestamp())-300 OR issued>extract(epoch FROM clock_timestamp())+60
 THEN PERFORM integration.connection_ops_error('P2803');END IF;
 SELECT * INTO r FROM integration.link_session_receipt WHERE environment=env AND receipt_key=k;
 IF FOUND AND ROW(r.link_token_sha256,r.link_session_sha256,r.body_sha256,r.status,r.issued_at)
 IS DISTINCT FROM ROW(h,s,body_hash,outcome,issued) THEN PERFORM integration.connection_ops_error('P2802');END IF;
 SELECT count(*) INTO n FROM integration.connection_attempt WHERE link_token_sha256=h;
 IF n<>1 THEN RETURN jsonb_build_object('state','ignored','receipt_count',0);END IF;
 SELECT * INTO a FROM integration.connection_attempt WHERE link_token_sha256=h;
 IF a.mode<>'reconnect' THEN RETURN jsonb_build_object('state','ignored','receipt_count',0);END IF;
 a:=integration.owned_repair_receipt_lock(a);
 IF a.link_token_sha256 IS DISTINCT FROM h THEN PERFORM integration.connection_ops_error('P2803');END IF;
 -- A committed claim is immutable proof selection; late events cannot replace or undo it.
 IF a.claim_id IS NOT NULL OR a.state<>'link_ready' THEN RETURN jsonb_build_object('state','ignored','receipt_count',0);END IF;
 PERFORM integration.owned_repair_scope(a);
 IF issued<(SELECT extract(epoch FROM min(recorded_at))-60 FROM integration.connection_event
 WHERE entity_id=a.entity_id AND book_id=a.book_id AND attempt_id=a.attempt_id AND kind='begin')
 THEN PERFORM integration.connection_ops_error('P2803');END IF;
 SELECT * INTO r FROM integration.link_session_receipt WHERE environment=env AND receipt_key=k;
 IF FOUND THEN
  IF ROW(r.entity_id,r.book_id,r.attempt_id,r.link_token_sha256,r.link_session_sha256,r.body_sha256,r.status,r.issued_at)
  IS DISTINCT FROM ROW(a.entity_id,a.book_id,a.attempt_id,h,s,body_hash,outcome,issued)
  THEN PERFORM integration.connection_ops_error('P2802');END IF;
  RETURN jsonb_build_object('state','duplicate','receipt_count',1);
 END IF;
 INSERT INTO integration.link_session_receipt(environment,receipt_key,entity_id,book_id,attempt_id,link_token_sha256,link_session_sha256,body_sha256,status,issued_at)
 VALUES(env,k,a.entity_id,a.book_id,a.attempt_id,h,s,body_hash,outcome,issued);
 RETURN jsonb_build_object('state','accepted','receipt_count',1);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2700' OR SQLSTATE 'P2600' THEN PERFORM integration.connection_ops_error('P2800');END $$;

CREATE OR REPLACE FUNCTION integration.record_link_session(actor uuid,e uuid,b uuid,id uuid,nonce_value uuid,raw text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;r jsonb;n bigint;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode='reconnect' THEN
  a:=integration.owned_repair_lock(actor,e,b,id);PERFORM integration.owned_repair_scope(a);
  UPDATE integration.connection_attempt SET expires_at=integration.owned_repair_deadline(a)
  WHERE entity_id=e AND book_id=b AND attempt_id=id;
 END IF;
 r:=integration.record_link_session_before_owned_repair32(actor,e,b,id,nonce_value,raw);
 IF a.mode='reconnect' THEN
  SELECT count(*) INTO n FROM integration.connection_attempt WHERE link_token_sha256=r->>'link_token_sha256';
  IF n<>1 THEN PERFORM integration.connection_error('P2702');END IF;
 END IF;
 RETURN r;
END $$;

CREATE OR REPLACE FUNCTION integration.connection_link_proof(actor uuid,e uuid,b uuid,id uuid)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;r integration.link_repair_claim;
BEGIN
 a:=integration.owned_repair_lock(actor,e,b,id);
 IF a.claim_id IS NOT NULL THEN
  SELECT * INTO r FROM integration.link_repair_claim WHERE entity_id=e AND book_id=b AND attempt_id=id AND claim_id=a.claim_id;
  IF NOT FOUND THEN PERFORM integration.connection_error('P2704');END IF;
  IF a.state<>'active' THEN PERFORM integration.owned_repair_scope(a);END IF;
  RETURN jsonb_build_object('status','success','receipt_key',r.receipt_key);
 END IF;
 PERFORM integration.owned_repair_scope(a);
 IF (SELECT count(*) FROM integration.connection_attempt WHERE link_token_sha256=a.link_token_sha256)<>1
 THEN PERFORM integration.connection_error('P2704');END IF;
 RETURN integration.owned_repair_proof(a);
END $$;

CREATE OR REPLACE FUNCTION integration.claim_connection(actor uuid,e uuid,b uuid,id uuid,nonce_value uuid,raw text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;p jsonb;r jsonb;c integration.link_repair_claim;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.claim_connection_before_owned_repair32(actor,e,b,id,nonce_value,raw);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);
 IF a.claim_id IS NOT NULL THEN
  SELECT * INTO c FROM integration.link_repair_claim WHERE entity_id=e AND book_id=b AND attempt_id=id AND claim_id=a.claim_id;
  IF NOT FOUND THEN PERFORM integration.connection_error('P2704');END IF;
  IF a.state<>'active' THEN PERFORM integration.owned_repair_scope(a);END IF;
  RETURN integration.claim_connection_before_owned_repair32(actor,e,b,id,nonce_value,raw);
 END IF;
 i:=integration.owned_repair_scope(a);
 IF (SELECT count(*) FROM integration.connection_attempt WHERE link_token_sha256=a.link_token_sha256)<>1
 THEN PERFORM integration.connection_error('P2704');END IF;
 p:=integration.owned_repair_proof(a);
 IF p->>'status'<>'success' THEN PERFORM integration.connection_error('P2704');END IF;
 r:=integration.claim_connection_before_owned_repair32(actor,e,b,id,nonce_value,raw);
 INSERT INTO integration.link_repair_claim(entity_id,book_id,attempt_id,claim_id,environment,receipt_key,credential_ref,credential_version,consent_ceiling)
 VALUES(e,b,id,(r->>'claim_id')::uuid,'sandbox',p->>'receipt_key',i.credential_ref,i.credential_version,i.consent_expires_at);
 RETURN r;
END $$;

CREATE OR REPLACE FUNCTION integration.get_repair_bank_credential(actor uuid,e uuid,b uuid,id uuid)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;c integration.link_repair_claim;envelope_value jsonb;
BEGIN
 a:=integration.owned_repair_lock(actor,e,b,id);i:=integration.owned_repair_scope(a);
 IF a.claim_id IS NULL THEN
  IF a.state<>'link_pending' THEN PERFORM integration.connection_error('P2704');END IF;
 ELSE
  SELECT * INTO c FROM integration.link_repair_claim WHERE entity_id=e AND book_id=b AND attempt_id=id AND claim_id=a.claim_id;
  IF NOT FOUND OR a.state NOT IN ('exchange_claimed','credential_pending','credential_recovery_required')
  OR c.credential_ref IS DISTINCT FROM i.credential_ref OR c.credential_version IS DISTINCT FROM i.credential_version
  THEN PERFORM integration.connection_error('P2704');END IF;
 END IF;
 SELECT envelope INTO envelope_value FROM integration.bank_credential WHERE entity_id=e AND book_id=b
 AND credential_ref=i.credential_ref AND credential_version=i.credential_version AND item_id=i.item_id AND environment='sandbox';
 IF envelope_value IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 RETURN jsonb_build_object('envelope',envelope_value,'credential_ref',i.credential_ref,'credential_version',i.credential_version::text,
 'item_id',i.item_id,'environment','sandbox');
END $$;

CREATE OR REPLACE FUNCTION integration.put_link_handoff(actor uuid,e uuid,b uuid,id uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;winner jsonb;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.put_link_handoff_before_owned_repair32(actor,e,b,id,raw);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);PERFORM integration.owned_repair_scope(a);
 IF a.claim_id IS NOT NULL OR a.state<>'link_ready' OR a.link_token_sha256 IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 v:=integration.connection_json(raw,32768);
 PERFORM ledger.keys(v,ARRAY['profile','key_version','wrapped_key','nonce','ciphertext']);
 IF v->>'profile' IS DISTINCT FROM 'jayburd-hosted-link-aead-v1'
 OR jsonb_typeof(v->'key_version') IS DISTINCT FROM 'string' OR length(v->>'key_version') NOT BETWEEN 1 AND 512
 OR (v->>'key_version') !~ '^projects/([1-9][0-9]*|[a-z][a-z0-9-]{4,28}[a-z0-9])/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$'
 OR jsonb_typeof(v->'wrapped_key') IS DISTINCT FROM 'string' OR jsonb_typeof(v->'nonce') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v->'ciphertext') IS DISTINCT FROM 'string'
 OR octet_length(decode(v->>'wrapped_key','base64')) NOT BETWEEN 1 AND 8192
 OR octet_length(decode(v->>'nonce','base64'))<>12
 OR octet_length(decode(v->>'ciphertext','base64')) NOT BETWEEN 17 AND 4112
 THEN PERFORM integration.connection_error('P2700');END IF;
 INSERT INTO integration.link_handoff(entity_id,book_id,attempt_id,envelope) VALUES(e,b,id,v) ON CONFLICT(entity_id,book_id,attempt_id) DO NOTHING;
 SELECT envelope INTO winner FROM integration.link_handoff WHERE entity_id=e AND book_id=b AND attempt_id=id;
 RETURN jsonb_build_object('envelope',winner);
EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');END $$;
DROP FUNCTION integration.webhook_profile_authorize(text);
DROP TABLE integration.webhook_login_profile;
ALTER TABLE integration.webhook_ingress DROP CONSTRAINT webhook_ingress_environment_check;
ALTER TABLE integration.webhook_ingress ADD CONSTRAINT webhook_ingress_environment_check CHECK(environment='sandbox');
ALTER TABLE integration.webhook_receipt DROP CONSTRAINT webhook_receipt_environment_check;
ALTER TABLE integration.webhook_receipt ADD CONSTRAINT webhook_receipt_environment_check CHECK(environment='sandbox');
ALTER TABLE integration.link_session_receipt DROP CONSTRAINT link_session_receipt_environment_check;
ALTER TABLE integration.link_session_receipt ADD CONSTRAINT link_session_receipt_environment_check CHECK(environment='sandbox');
COMMIT;
