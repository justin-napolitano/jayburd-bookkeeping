-- Durable status/recovery and acquisition signals; no provider calls or financial effects.
BEGIN;
DO $$ BEGIN
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_webhook') THEN CREATE ROLE bookkeeping_webhook NOLOGIN; END IF;
 IF EXISTS(SELECT FROM pg_roles r WHERE r.rolname='bookkeeping_webhook' AND (r.rolcanlogin OR r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication))
 OR EXISTS(SELECT FROM pg_roles r WHERE r.rolname<> 'bookkeeping_webhook' AND pg_has_role('bookkeeping_webhook',r.oid,'MEMBER')) THEN RAISE EXCEPTION 'unsafe webhook role'; END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
ALTER TABLE integration.sync_stream ADD COLUMN stream_revision bigint NOT NULL DEFAULT 1 CHECK(stream_revision>0);
ALTER TABLE integration.sync_stream DROP CONSTRAINT sync_stream_state_check;
ALTER TABLE integration.sync_stream ADD CONSTRAINT sync_stream_state_check CHECK(state IN ('active','paused','disconnected'));
CREATE INDEX connection_attempt_discovery ON integration.connection_attempt(entity_id,book_id,actor_id,attempt_id);
CREATE INDEX sync_set_status ON integration.sync_set(entity_id,book_id,stream_id,created_at DESC,set_id DESC);
-- This is an outbox of immutable generation facts, not a second worker state machine.
CREATE TABLE integration.sync_wakeup (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,stream_id uuid NOT NULL,requested_generation bigint NOT NULL CHECK(requested_generation>0),message_id uuid NOT NULL DEFAULT gen_random_uuid(),
 reason text NOT NULL CHECK(reason IN ('worker','manual','resume','webhook','catchup')),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,stream_id,requested_generation),UNIQUE(message_id),FOREIGN KEY(entity_id,book_id,stream_id) REFERENCES integration.sync_stream
);
CREATE TABLE integration.webhook_ingress (
 environment text PRIMARY KEY CHECK(environment='sandbox'),login_name name NOT NULL,enabled boolean NOT NULL,revision bigint NOT NULL CHECK(revision>0)
);
CREATE TABLE integration.webhook_operator_receipt (
 database_actor name NOT NULL, idempotency_key text NOT NULL,body jsonb NOT NULL,result jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(database_actor,idempotency_key)
);
CREATE TABLE integration.webhook_receipt (
 environment text NOT NULL CHECK(environment='sandbox'),receipt_key text NOT NULL CHECK(receipt_key~'^[0-9a-f]{64}$'),receipt_id uuid NOT NULL DEFAULT gen_random_uuid(),
 entity_id uuid NOT NULL,book_id uuid NOT NULL,sync_item_id uuid NOT NULL,body jsonb NOT NULL,stream_ids uuid[] NOT NULL CHECK(cardinality(stream_ids) BETWEEN 1 AND 25),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(environment,receipt_key),UNIQUE(receipt_id),FOREIGN KEY(entity_id,book_id,sync_item_id) REFERENCES integration.sync_item
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.sync_wakeup FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.webhook_operator_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.webhook_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Fixed error vocabulary contains no raw webhook, token or database diagnostic.
CREATE FUNCTION integration.connection_ops_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='connection_operation_rejected'; END $$;
-- Dedicated authenticated login may not inherit any other runtime or privileged capability.
CREATE FUNCTION integration.webhook_login(login_value name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolcanlogin AND NOT(rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication))
 AND pg_has_role(login_value,'bookkeeping_webhook','MEMBER')
 AND NOT EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_identity_operator','bookkeeping_migrator','bookkeeping_sync','bookkeeping_publisher','bookkeeping_observer','cloudsqlsuperuser')) AND pg_has_role(login_value,oid,'MEMBER'))
$$;
-- Owner configuration binds ingress to the actual login; no caller can choose an actor or scope.
CREATE FUNCTION integration.set_webhook_ingress(login_value name,env text,k text,enabled_value boolean) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
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
-- Caller already holds current authority and stream lock; exact signal key is one increment and one intent.
CREATE FUNCTION integration.signal_sync(t integration.sync_stream,k text,why text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE n bigint;
BEGIN
 IF k IS NULL OR k!~'^[A-Za-z0-9._:-]{1,128}$' OR why NOT IN ('worker','manual','resume','webhook','catchup') THEN PERFORM integration.connection_ops_error('P2800');END IF;
 SELECT requested_generation INTO n FROM integration.sync_signal WHERE entity_id=t.entity_id AND book_id=t.book_id AND stream_id=t.stream_id AND idempotency_key=k;
 IF NOT FOUND THEN
  IF t.requested_generation=9223372036854775807 THEN PERFORM integration.connection_ops_error('P2805');END IF;
  UPDATE integration.sync_stream SET requested_generation=requested_generation+1 WHERE entity_id=t.entity_id AND book_id=t.book_id AND stream_id=t.stream_id RETURNING requested_generation INTO n;
  INSERT INTO integration.sync_signal VALUES(t.entity_id,t.book_id,t.stream_id,k,n);
  INSERT INTO integration.sync_wakeup(entity_id,book_id,stream_id,requested_generation,reason) VALUES(t.entity_id,t.book_id,t.stream_id,n,why);
 END IF;
 RETURN jsonb_build_object('stream_id',t.stream_id,'requested_generation',n::text);
END $$;
CREATE OR REPLACE FUNCTION integration.request_sync(e uuid,b uuid,id uuid,k text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;
BEGIN t:=integration.sync_authorize(e,b,id);RETURN integration.signal_sync(t,k,'worker');
EXCEPTION WHEN SQLSTATE 'P2800' THEN PERFORM integration.sync_error('P2600');WHEN SQLSTATE 'P2805' THEN PERFORM integration.sync_error('P2605');END $$;
-- A blocked attempt is a committed unresolved operation, not a browser-memory latch.
CREATE FUNCTION integration.connection_unfinished(a integration.connection_attempt) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT a.state NOT IN ('active','disconnected') AND NOT (a.state='binding_review' AND a.expires_at<=statement_timestamp()) AND (a.claim_id IS NOT NULL OR a.expires_at>statement_timestamp())
$$;
-- Discover only this authenticated actor's attempts, without private context or credentials.
CREATE FUNCTION integration.list_connection_attempts(actor uuid,e uuid,b uuid,maximum integer,after_id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;items jsonb:='[]';n bigint;last_id uuid;more boolean;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);
 IF maximum IS NULL OR maximum NOT BETWEEN 1 AND 100 THEN PERFORM integration.connection_ops_error('P2800');END IF;
 SELECT count(*) INTO n FROM integration.connection_attempt x WHERE x.entity_id=e AND x.book_id=b AND x.actor_id=actor AND integration.connection_unfinished(x);
 FOR a IN SELECT x.* FROM integration.connection_attempt x WHERE x.entity_id=e AND x.book_id=b AND x.actor_id=actor AND (after_id IS NULL OR x.attempt_id>after_id) ORDER BY x.attempt_id LIMIT maximum LOOP
  PERFORM integration.connection_lock(actor,e,b,a.attempt_id);
  items:=items||jsonb_build_array(integration.render_connection(a,false)||jsonb_build_object('expired_unclaimed',a.claim_id IS NULL AND a.expires_at<=clock_timestamp(),'binding_expired',a.state='binding_review' AND a.expires_at<=clock_timestamp()));last_id:=a.attempt_id;
 END LOOP;
 SELECT EXISTS(SELECT FROM integration.connection_attempt x WHERE x.entity_id=e AND x.book_id=b AND x.actor_id=actor AND x.attempt_id>last_id) INTO more;
 RETURN jsonb_build_object('items',items,'unfinished_count',n::text,'next_cursor',CASE WHEN more THEN last_id ELSE NULL END);
END $$;
-- Preserve exact existing-key recovery, while the canonical book lock serializes fresh enrollment checks.
ALTER FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) RENAME TO begin_connection_v13;
REVOKE ALL ON FUNCTION integration.begin_connection_v13(uuid,uuid,uuid,text,text) FROM bookkeeping_api;
CREATE FUNCTION integration.begin_connection(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);v:=integration.connection_json(raw,16384);
 IF v->>'mode'='new' AND NOT EXISTS(SELECT FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND actor_id=actor AND idempotency_key=k)
 AND EXISTS(SELECT FROM integration.connection_attempt a WHERE a.entity_id=e AND a.book_id=b AND a.actor_id=actor AND integration.connection_unfinished(a)) THEN PERFORM integration.connection_ops_error('P2804');END IF;
 RETURN integration.begin_connection_v13(actor,e,b,k,raw);
END $$;
-- Projection names distinguish complete local publication from unknown provider update time.
CREATE FUNCTION integration.render_stream_status(t integration.sync_stream) RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE i integration.sync_item;s banking.source_account;reason_value text;
BEGIN
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=t.entity_id AND book_id=t.book_id AND sync_item_id=t.sync_item_id;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=t.entity_id AND book_id=t.book_id AND source_account_id=t.source_account_id;
 SELECT CASE WHEN x.state='abandoned' AND x.reason IN ('pagination_mutation','provider_failure','decoder_rejected','cancelled','item_changed','lease_expired','account_disconnected','account_paused') THEN x.reason ELSE NULL END INTO reason_value FROM integration.sync_set x WHERE x.entity_id=t.entity_id AND x.book_id=t.book_id AND x.stream_id=t.stream_id ORDER BY x.created_at DESC,x.set_id DESC LIMIT 1;
 RETURN jsonb_build_object('stream_id',t.stream_id,'source_account_id',t.source_account_id,'sync_item_id',t.sync_item_id,'state',t.state,'stream_revision',t.stream_revision::text,'item_state',i.state,'item_generation',i.generation::text,'binding_revision',s.binding_revision::text,'requested_generation',t.requested_generation::text,'completed_generation',t.completed_generation::text,'queued',t.requested_generation>t.completed_generation,'lease_active',t.current_set IS NOT NULL AND t.lease_until>statement_timestamp(),'last_completed_at',t.last_completed_at,'readiness',t.readiness,'last_error',reason_value,'provider_updated_at',NULL);
END $$;
-- Public source-specific reads keep current management/source capability checks even for paused history.
CREATE FUNCTION integration.connection_stream_status(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id;IF NOT FOUND THEN PERFORM integration.connection_ops_error('P2801');END IF;
 PERFORM integration.connection_source(actor,e,b,t.source_account_id);
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id FOR UPDATE;
 RETURN integration.render_stream_status(t);
END $$;
-- Eligibility is only candidate selection; mutation repeats every authority check under canonical locks.
CREATE FUNCTION integration.signal_eligible(t integration.sync_stream) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT t.state='active' AND integration.sync_login(t.worker_login) AND EXISTS(
 SELECT FROM integration.sync_item i JOIN ledger.entity e ON e.entity_id=i.entity_id JOIN ledger.book b ON b.entity_id=i.entity_id AND b.book_id=i.book_id
 JOIN banking.source_account s ON s.entity_id=t.entity_id AND s.book_id=t.book_id AND s.source_account_id=t.source_account_id
 JOIN banking.source_namespace n ON n.entity_id=t.entity_id AND n.book_id=t.book_id AND n.namespace_id=t.namespace_id
 WHERE i.entity_id=t.entity_id AND i.book_id=t.book_id AND i.sync_item_id=t.sync_item_id AND i.state='active' AND i.consent_expires_at>statement_timestamp() AND e.disabled_at IS NULL AND b.disabled_at IS NULL AND s.state='active' AND n.state='active'
 AND NOT EXISTS(SELECT FROM unnest(ARRAY[i.consent_principal,t.service_principal]) p WHERE NOT EXISTS(SELECT FROM ledger.principal x WHERE x.principal_id=p AND x.disabled_at IS NULL)
 OR EXISTS(SELECT FROM unnest(ARRAY['read','import']) cap WHERE NOT EXISTS(SELECT FROM ledger.membership m WHERE m.entity_id=t.entity_id AND m.principal_id=p AND m.permission=cap AND m.revoked_at IS NULL)
 OR NOT EXISTS(SELECT FROM banking.source_permission q WHERE q.entity_id=t.entity_id AND q.book_id=t.book_id AND q.source_account_id=t.source_account_id AND q.principal_id=p AND q.capability=cap AND q.revoked_at IS NULL))))
$$;
-- Trusted ingress/manual callers share acquisition authorization without impersonating a worker session.
CREATE FUNCTION integration.signal_authorize(e uuid,b uuid,id uuid) RETURNS integration.sync_stream LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;i integration.sync_item;
BEGIN
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id;
 IF NOT FOUND OR t.state<>'active' OR NOT integration.sync_login(t.worker_login) THEN PERFORM integration.connection_ops_error('P2801');END IF;
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id;
 PERFORM integration.sync_authority(e,b,t.source_account_id,i.consent_principal,t.service_principal);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id FOR UPDATE;
 IF i.state<>'active' OR clock_timestamp()>=i.consent_expires_at THEN PERFORM integration.connection_ops_error('P2801');END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id FOR UPDATE;
 PERFORM 1 FROM banking.source_namespace WHERE entity_id=e AND book_id=b AND namespace_id=t.namespace_id AND state='active';IF NOT FOUND THEN PERFORM integration.connection_ops_error('P2801');END IF;
 RETURN t;
EXCEPTION WHEN SQLSTATE 'P2601' THEN PERFORM integration.connection_ops_error('P2801');END $$;
-- Per-account pause fences held work and advances its own revision; source approval pins remain unchanged.
CREATE FUNCTION integration.mutate_connection_stream(actor uuid,e uuid,b uuid,id uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;i integration.sync_item;s banking.source_account;v jsonb;rec integration.connection_receipt;result_value jsonb;act text;why text;norm jsonb;principal_value uuid;
BEGIN
 -- Preserve principal-before-book ordering when an authorized manager differs from Item consent.
 PERFORM set_config('lock_timeout','1s',true);
 FOR principal_value IN SELECT DISTINCT p FROM (
 SELECT actor p UNION ALL SELECT w.service_principal FROM integration.connection_worker w WHERE w.entity_id=e AND w.book_id=b
 UNION ALL SELECT x.service_principal FROM integration.sync_stream x WHERE x.entity_id=e AND x.book_id=b AND x.stream_id=id
 UNION ALL SELECT item_row.consent_principal FROM integration.sync_stream x JOIN integration.sync_item item_row USING(entity_id,book_id,sync_item_id) WHERE x.entity_id=e AND x.book_id=b AND x.stream_id=id) q ORDER BY p LOOP PERFORM 1 FROM ledger.principal WHERE principal_id=principal_value FOR SHARE;END LOOP;
 PERFORM integration.connection_authorize(actor,e,b);PERFORM integration.connection_key(k);v:=integration.connection_json(raw,2048);PERFORM ledger.keys(v,ARRAY['action','expected_item_generation','expected_binding_revision','expected_stream_revision']);
 act:=ledger.text_value(v->'action');IF act NOT IN ('pause','resume','request') THEN PERFORM integration.connection_ops_error('P2800');END IF;
 PERFORM banking.import_revision(v->'expected_item_generation');PERFORM banking.import_revision(v->'expected_binding_revision');PERFORM banking.import_revision(v->'expected_stream_revision');
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id;IF NOT FOUND THEN PERFORM integration.connection_ops_error('P2801');END IF;
 s:=integration.connection_source(actor,e,b,t.source_account_id);norm:=jsonb_build_object('stream_id',id,'body',v);
 SELECT * INTO rec FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='stream_operation' AND idempotency_key=k;
 IF FOUND THEN IF rec.body<>norm THEN PERFORM integration.connection_ops_error('P2802');END IF;RETURN rec.result;END IF;
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=t.sync_item_id FOR UPDATE;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id FOR UPDATE;
 IF t.stream_revision<>banking.import_revision(v->'expected_stream_revision') OR i.generation<>banking.import_revision(v->'expected_item_generation') OR s.binding_revision<>banking.import_revision(v->'expected_binding_revision') THEN PERFORM integration.connection_ops_error('P2803');END IF;
 IF t.stream_revision=9223372036854775807 OR t.fence=9223372036854775807 THEN PERFORM integration.connection_ops_error('P2805');END IF;
 IF t.state='disconnected' OR i.state NOT IN ('active','paused','reauthorization_required') THEN PERFORM integration.connection_ops_error('P2804');END IF;
 IF act='pause' THEN
  IF t.state<>'active' THEN PERFORM integration.connection_ops_error('P2804');END IF;
  UPDATE integration.sync_set SET state='abandoned',reason='account_paused' WHERE entity_id=e AND book_id=b AND set_id=t.current_set AND state='staging';
  UPDATE integration.sync_stream SET state='paused',stream_revision=stream_revision+1,fence=fence+1,current_set=NULL,lease_until=NULL WHERE entity_id=e AND book_id=b AND stream_id=id RETURNING * INTO t;
 ELSE
  IF act='resume' THEN
   IF t.state<>'paused' THEN PERFORM integration.connection_ops_error('P2804');END IF;
   UPDATE integration.sync_stream SET state='active',stream_revision=stream_revision+1 WHERE entity_id=e AND book_id=b AND stream_id=id RETURNING * INTO t;
   ELSIF t.state<>'active' THEN PERFORM integration.connection_ops_error('P2804');END IF;
  t:=integration.signal_authorize(e,b,id);why:=CASE WHEN act='resume' THEN 'resume' ELSE 'manual' END;
  PERFORM integration.signal_sync(t,'api:'||encode(sha256(convert_to(actor::text||':'||k,'UTF8')),'hex'),why);
 END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND stream_id=id;
 result_value:=integration.render_stream_status(t);INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'stream_operation',k,norm,result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR SQLSTATE 'P2700' THEN PERFORM integration.connection_ops_error('P2800');END $$;
-- Only a verified-service login can map provider Item identity into approved SQL scope.
CREATE FUNCTION integration.accept_verified_webhook(env text,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
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
-- A scheduled tick reuses existing worker authority and the same durable generation mechanism.
CREATE FUNCTION integration.catchup_sync(e uuid,b uuid,tick uuid,maximum integer,after_id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE t integration.sync_stream;ids uuid[];principal_value uuid;items jsonb:='[]';last_id uuid;more boolean;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR tick IS NULL OR tick='00000000-0000-0000-0000-000000000000' OR maximum IS NULL OR maximum NOT BETWEEN 1 AND 100 OR e IS NULL OR b IS NULL THEN PERFORM integration.connection_ops_error('P2800');END IF;
 IF NOT integration.sync_login(session_user) THEN PERFORM integration.connection_ops_error('P2801');END IF;
 SELECT array_agg(q.stream_id ORDER BY q.stream_id) INTO ids FROM (SELECT x.stream_id FROM integration.sync_stream x WHERE x.entity_id=e AND x.book_id=b AND x.worker_login=session_user AND (after_id IS NULL OR x.stream_id>after_id) AND integration.signal_eligible(x) ORDER BY x.stream_id LIMIT maximum) q;
 FOR principal_value IN SELECT DISTINCT p FROM (SELECT i.consent_principal p FROM integration.sync_stream x JOIN integration.sync_item i USING(entity_id,book_id,sync_item_id) WHERE x.entity_id=e AND x.book_id=b AND x.stream_id=ANY(ids) UNION ALL SELECT x.service_principal FROM integration.sync_stream x WHERE x.entity_id=e AND x.book_id=b AND x.stream_id=ANY(ids)) q ORDER BY p LOOP PERFORM 1 FROM ledger.principal WHERE principal_id=principal_value AND disabled_at IS NULL FOR SHARE;END LOOP;
 FOREACH last_id IN ARRAY coalesce(ids,ARRAY[]::uuid[]) LOOP
  BEGIN
   t:=integration.sync_authorize(e,b,last_id);items:=items||jsonb_build_array(integration.signal_sync(t,'catchup:'||tick::text,'catchup'));
  EXCEPTION WHEN SQLSTATE 'P2601' THEN NULL;END;
 END LOOP;
 SELECT EXISTS(SELECT FROM integration.sync_stream x WHERE x.entity_id=e AND x.book_id=b AND x.worker_login=session_user AND x.stream_id>last_id AND integration.signal_eligible(x)) INTO more;
 RETURN jsonb_build_object('items',items,'next_cursor',CASE WHEN more THEN last_id ELSE NULL END);
END $$;

-- Forward cross-role denial preserves the existing authority boundary.
CREATE OR REPLACE FUNCTION ledger.authorize(actor uuid,e uuid,b uuid,permission_name text,writing boolean) RETURNS bigint LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE rev bigint;
BEGIN
 IF pg_has_role(session_user,'bookkeeping_sync','MEMBER') OR pg_has_role(session_user,'bookkeeping_webhook','MEMBER') THEN PERFORM ledger.fail('P2101'); END IF;
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

-- Forward cross-role denial preserves the existing authority boundary.
CREATE OR REPLACE FUNCTION integration.observer_binding_lock(id uuid,publisher boolean) RETURNS integration.observer_binding LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE b integration.observer_binding;
BEGIN
 IF pg_has_role(session_user,'bookkeeping_sync','MEMBER') OR pg_has_role(session_user,'bookkeeping_webhook','MEMBER') THEN PERFORM integration.observer_error('P2201'); END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR id IS NULL THEN PERFORM integration.observer_error('P2200'); END IF;
 PERFORM set_config('lock_timeout','1s',true);
 SELECT * INTO b FROM integration.observer_binding WHERE binding_id=id FOR SHARE;
 IF NOT FOUND OR session_user<>(CASE WHEN publisher THEN b.publisher_login ELSE b.consumer_login END) THEN PERFORM integration.observer_error('P2201'); END IF;
 RETURN b;
END $$;

-- Forward cross-role denial preserves the existing authority boundary.
CREATE OR REPLACE FUNCTION integration.sync_login(login_name name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_name AND rolcanlogin AND NOT(rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication))
 AND NOT EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolname='cloudsqlsuperuser') AND pg_has_role(login_name,oid,'MEMBER'))
 AND pg_has_role(login_name,'bookkeeping_sync','MEMBER') AND NOT EXISTS(SELECT FROM pg_roles WHERE rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_observer','bookkeeping_publisher','bookkeeping_identity_operator','bookkeeping_migrator','bookkeeping_webhook') AND pg_has_role(login_name,oid,'MEMBER'))
$$;

-- Stream revision protects operator intent independently of source approval.
CREATE OR REPLACE FUNCTION integration.sync_state_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.sync_error('P2602'); END IF;
 IF TG_TABLE_NAME='sync_item' THEN
  IF (to_jsonb(NEW)-ARRAY['generation','state','credential_ref','credential_version','consent_principal','consent_evidence_ref','consent_expires_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['generation','state','credential_ref','credential_version','consent_principal','consent_evidence_ref','consent_expires_at']) OR NEW.generation::numeric<>OLD.generation::numeric+1 OR OLD.state='disconnected' OR NEW.credential_version<OLD.credential_version OR NEW.credential_version>OLD.credential_version+1 OR NEW.credential_ref<>OLD.credential_ref AND NEW.credential_version<>OLD.credential_version+1 THEN PERFORM integration.sync_error('P2602'); END IF;
 ELSIF TG_TABLE_NAME='sync_stream' THEN
  IF (to_jsonb(NEW)-ARRAY['requested_generation','completed_generation','cursor_value','fence','lease_until','current_set','last_completed_at','readiness','state','stream_revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['requested_generation','completed_generation','cursor_value','fence','lease_until','current_set','last_completed_at','readiness','state','stream_revision']) OR (NEW.stream_revision IS DISTINCT FROM OLD.stream_revision AND NOT ((OLD.state='active' AND NEW.state='paused' OR OLD.state='paused' AND NEW.state='active') AND NEW.stream_revision::numeric=OLD.stream_revision::numeric+1)) OR NEW.stream_revision<OLD.stream_revision OR NEW.stream_revision::numeric>OLD.stream_revision::numeric+1 OR (NEW.state IS DISTINCT FROM OLD.state AND NEW.state<>'disconnected' AND NEW.stream_revision::numeric<>OLD.stream_revision::numeric+1) OR NEW.requested_generation<OLD.requested_generation OR NEW.completed_generation<OLD.completed_generation OR NEW.fence<OLD.fence OR OLD.state='disconnected' AND NEW.state<>'disconnected' THEN PERFORM integration.sync_error('P2602'); END IF;
 ELSE
  IF (to_jsonb(NEW)-ARRAY['state','reason','page_count','change_count','byte_count','final_page','continuation_cursor']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','reason','page_count','change_count','byte_count','final_page','continuation_cursor']) OR OLD.state<>'staging' OR NEW.page_count<OLD.page_count OR NEW.change_count<OLD.change_count OR NEW.byte_count<OLD.byte_count THEN PERFORM integration.sync_error('P2602'); END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE ALL ON ALL TABLES IN SCHEMA integration FROM bookkeeping_webhook;
GRANT USAGE ON SCHEMA integration TO bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.connection_ops_error(text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.webhook_login(name) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.set_webhook_ingress(name,text,text,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.signal_sync(integration.sync_stream,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.connection_unfinished(integration.connection_attempt) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.list_connection_attempts(uuid,uuid,uuid,integer,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.render_stream_status(integration.sync_stream) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.connection_stream_status(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.signal_eligible(integration.sync_stream) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.signal_authorize(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.mutate_connection_stream(uuid,uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.accept_verified_webhook(text,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
REVOKE ALL ON FUNCTION integration.catchup_sync(uuid,uuid,uuid,integer,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher,bookkeeping_webhook;
GRANT EXECUTE ON FUNCTION integration.set_webhook_ingress(name,text,text,boolean) TO bookkeeping_owner;
GRANT EXECUTE ON FUNCTION integration.list_connection_attempts(uuid,uuid,uuid,integer,uuid),integration.begin_connection(uuid,uuid,uuid,text,text),integration.connection_stream_status(uuid,uuid,uuid,uuid),integration.mutate_connection_stream(uuid,uuid,uuid,uuid,text,text) TO bookkeeping_api;
GRANT EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) TO bookkeeping_webhook;
GRANT EXECUTE ON FUNCTION integration.catchup_sync(uuid,uuid,uuid,integer,uuid) TO bookkeeping_sync;
COMMIT;
