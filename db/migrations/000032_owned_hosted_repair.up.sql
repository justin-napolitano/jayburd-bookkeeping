-- Bind signed Hosted repair proof to an owned attempt; public ingress never resumes an Item.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;

ALTER TABLE integration.connection_attempt ADD COLUMN repair_scope_sha256 text
 CHECK(repair_scope_sha256 IS NULL OR repair_scope_sha256~'^[0-9a-f]{64}$');
-- A fresh server-authorized begin may pin scope once; later commands cannot rescope the same intent.
CREATE FUNCTION integration.owned_repair_attempt_guard() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF OLD.repair_scope_sha256 IS NOT NULL AND NEW.repair_scope_sha256 IS DISTINCT FROM OLD.repair_scope_sha256
 THEN PERFORM integration.connection_error('P2702');END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_repair_scope BEFORE UPDATE ON integration.connection_attempt
 FOR EACH ROW EXECUTE FUNCTION integration.owned_repair_attempt_guard();

CREATE TABLE integration.link_session_receipt (
 environment text NOT NULL CHECK(environment='sandbox'),
 receipt_key text NOT NULL CHECK(receipt_key~'^[0-9a-f]{64}$'),
 entity_id uuid NOT NULL,book_id uuid NOT NULL,attempt_id uuid NOT NULL,
 link_token_sha256 text NOT NULL CHECK(link_token_sha256~'^[0-9a-f]{64}$'),
 link_session_sha256 text NOT NULL CHECK(link_session_sha256~'^[0-9a-f]{64}$'),
 body_sha256 text NOT NULL CHECK(body_sha256~'^[0-9a-f]{64}$'),
 status text NOT NULL CHECK(status IN ('success','exited')),
 issued_at bigint NOT NULL CHECK(issued_at>0),recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(environment,receipt_key),
 UNIQUE(entity_id,book_id,attempt_id,environment,receipt_key),
 FOREIGN KEY(entity_id,book_id,attempt_id) REFERENCES integration.connection_attempt
);
CREATE TABLE integration.link_repair_claim (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,attempt_id uuid NOT NULL,
 claim_id uuid NOT NULL UNIQUE,environment text NOT NULL,receipt_key text NOT NULL,
 credential_ref uuid NOT NULL,credential_version bigint NOT NULL CHECK(credential_version>0),
 consent_ceiling timestamptz NOT NULL,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,attempt_id),
 FOREIGN KEY(entity_id,book_id,attempt_id) REFERENCES integration.connection_attempt,
 FOREIGN KEY(entity_id,book_id,attempt_id,environment,receipt_key)
 REFERENCES integration.link_session_receipt(entity_id,book_id,attempt_id,environment,receipt_key)
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.link_session_receipt
 FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.link_repair_claim
 FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
REVOKE ALL ON integration.link_session_receipt,integration.link_repair_claim
 FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
CREATE INDEX connection_attempt_link_hash ON integration.connection_attempt(link_token_sha256)
 WHERE link_token_sha256 IS NOT NULL;

-- The immutable begin event, rather than an invented creation column, bounds old and new repairs.
CREATE FUNCTION integration.owned_repair_deadline(a integration.connection_attempt)
RETURNS timestamptz LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT least(a.expires_at,CASE WHEN count(*)=1 THEN min(recorded_at)+interval '15 minutes'
 ELSE '1970-01-01T00:00:00Z'::timestamptz END)
 FROM integration.connection_event WHERE entity_id=a.entity_id AND book_id=a.book_id
 AND attempt_id=a.attempt_id AND kind='begin'
$$;

-- Only mutable unclaimed deadlines change; append the policy transition without rewriting receipts.
WITH capped AS (
 SELECT a.entity_id,a.book_id,a.attempt_id,a.expires_at AS previous_expiry,
 least(a.expires_at,CASE WHEN count(e.event_id)=1 THEN min(e.recorded_at)+interval '15 minutes'
 ELSE clock_timestamp() END) AS deadline
 FROM integration.connection_attempt a LEFT JOIN integration.connection_event e
 ON e.entity_id=a.entity_id AND e.book_id=a.book_id AND e.attempt_id=a.attempt_id AND e.kind='begin'
 WHERE a.mode='reconnect' AND a.claim_id IS NULL AND a.state IN ('link_pending','link_ready')
 GROUP BY a.entity_id,a.book_id,a.attempt_id,a.expires_at
), changed AS (
 UPDATE integration.connection_attempt a SET expires_at=c.deadline FROM capped c
 WHERE a.entity_id=c.entity_id AND a.book_id=c.book_id AND a.attempt_id=c.attempt_id
 AND a.expires_at>c.deadline
 RETURNING a.entity_id,a.book_id,a.attempt_id,c.previous_expiry,c.deadline
)
INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body)
 SELECT entity_id,book_id,attempt_id,'repair_window_capped',
 jsonb_build_object('profile','jayburd-owned-hosted-repair/1','previous_expiry',previous_expiry,'expires_at',deadline)
 FROM changed;

-- Prelock the entire authority graph before any shared scope/source/Item lock.
CREATE FUNCTION integration.owned_repair_principals(a integration.connection_attempt)
RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE p uuid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' THEN PERFORM integration.connection_error('P2700');END IF;
 PERFORM set_config('lock_timeout','1s',true);
 FOR p IN SELECT DISTINCT x FROM (
  SELECT a.actor_id x
  UNION ALL SELECT service_principal FROM integration.connection_worker WHERE entity_id=a.entity_id AND book_id=a.book_id
  UNION ALL SELECT consent_principal FROM integration.sync_item WHERE entity_id=a.entity_id AND book_id=a.book_id AND sync_item_id=a.sync_item_id
  UNION ALL SELECT service_principal FROM integration.sync_stream WHERE entity_id=a.entity_id AND book_id=a.book_id AND sync_item_id=a.sync_item_id
 ) q ORDER BY x LOOP
  PERFORM 1 FROM ledger.principal WHERE principal_id=p AND disabled_at IS NULL FOR SHARE;
  IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 END LOOP;
END $$;

-- Canonical operational metadata excludes changing cursors/demands and preserves every approved binding.
CREATE FUNCTION integration.owned_repair_scope_digest(a integration.connection_attempt)
RETURNS text LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT encode(sha256(convert_to(jsonb_build_object(
  'profile','jayburd-owned-hosted-repair/1','actor',a.actor_id,'entity',a.entity_id,'book',a.book_id,'item',a.sync_item_id,
  'consent_principal',i.consent_principal,'consent_evidence',i.consent_evidence_ref,
  'consent_expires_at',to_char(i.consent_expires_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'worker_revision',w.revision,'worker_login',w.worker_login,'worker_principal',w.service_principal,
  'streams',coalesce((SELECT jsonb_agg(jsonb_build_object('stream',t.stream_id,'state',t.state,'revision',t.stream_revision,
   'account',t.account_id,'service',t.service_principal,'login',t.worker_login,
   'source',s.source_account_id,'source_state',s.state,'binding_revision',s.binding_revision,
   'ledger_account',s.ledger_account_id,'currency',s.currency_version_id,'namespace',n.namespace_id,'namespace_state',n.state)
   ORDER BY s.source_account_id,t.stream_id)
   FROM integration.sync_stream t JOIN banking.source_account s USING(entity_id,book_id,source_account_id)
   JOIN banking.source_namespace n ON n.entity_id=t.entity_id AND n.book_id=t.book_id AND n.namespace_id=t.namespace_id
   WHERE t.entity_id=a.entity_id AND t.book_id=a.book_id AND t.sync_item_id=a.sync_item_id),'[]'::jsonb)
 )::text,'UTF8')),'hex')
 FROM integration.sync_item i JOIN integration.connection_worker w USING(entity_id,book_id)
 WHERE i.entity_id=a.entity_id AND i.book_id=a.book_id AND i.sync_item_id=a.sync_item_id
$$;

-- Shared post-lock checks deny expired approval, stale generations and every lost sibling capability.
CREATE FUNCTION integration.owned_repair_scope(a integration.connection_attempt)
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

-- API locking retains the existing identity boundary after prelocking additional consent principals.
CREATE FUNCTION integration.owned_repair_lock(actor uuid,e uuid,b uuid,id uuid)
RETURNS integration.connection_attempt LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF FOUND AND a.mode='reconnect' THEN PERFORM integration.owned_repair_principals(a);END IF;
 a:=integration.connection_lock(actor,e,b,id);
 IF a.mode<>'reconnect' THEN PERFORM integration.connection_error('P2704');END IF;
 RETURN a;
END $$;

-- Dedicated ingress cannot borrow ledger.authorize: that function intentionally denies its session role.
-- The stored token association supplies the actor; no notification or caller chooses one.
CREATE FUNCTION integration.owned_repair_receipt_lock(candidate integration.connection_attempt)
RETURNS integration.connection_attempt LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;w integration.connection_worker;p text;s uuid;
BEGIN
 PERFORM integration.owned_repair_principals(candidate);
 PERFORM 1 FROM ledger.entity WHERE entity_id=candidate.entity_id AND disabled_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 FOREACH p IN ARRAY ARRAY['read','connect','import'] LOOP
  PERFORM 1 FROM ledger.membership WHERE entity_id=candidate.entity_id AND principal_id=candidate.actor_id
  AND permission=p AND revoked_at IS NULL FOR SHARE;
  IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 END LOOP;
 PERFORM 1 FROM ledger.book WHERE entity_id=candidate.entity_id AND book_id=candidate.book_id AND disabled_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 SELECT * INTO w FROM integration.connection_worker WHERE entity_id=candidate.entity_id AND book_id=candidate.book_id FOR SHARE;
 IF NOT FOUND OR NOT w.enabled OR NOT integration.sync_login(w.worker_login) OR w.revision<>candidate.worker_revision
 THEN PERFORM integration.connection_error('P2701');END IF;
 FOREACH p IN ARRAY ARRAY['read','import'] LOOP
  PERFORM 1 FROM ledger.membership WHERE entity_id=candidate.entity_id AND principal_id=w.service_principal
  AND permission=p AND revoked_at IS NULL FOR SHARE;
  IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 END LOOP;
 FOR s IN SELECT source_account_id FROM integration.sync_stream WHERE entity_id=candidate.entity_id
 AND book_id=candidate.book_id AND sync_item_id=candidate.sync_item_id ORDER BY source_account_id LOOP
  PERFORM integration.connection_source(candidate.actor_id,candidate.entity_id,candidate.book_id,s);
 END LOOP;
 PERFORM 1 FROM integration.sync_item WHERE entity_id=candidate.entity_id AND book_id=candidate.book_id AND sync_item_id=candidate.sync_item_id FOR UPDATE;
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=candidate.entity_id AND book_id=candidate.book_id
 AND attempt_id=candidate.attempt_id AND actor_id=candidate.actor_id FOR UPDATE;
 IF NOT FOUND OR a.mode<>'reconnect' OR a.worker_revision<>w.revision THEN PERFORM integration.connection_error('P2701');END IF;
 RETURN a;
END $$;

-- Proof is derived from distinct sessions/outcomes, never from the first arriving event.
CREATE FUNCTION integration.owned_repair_proof(a integration.connection_attempt)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE sessions bigint;outcomes bigint;outcome text;k text;
BEGIN
 SELECT count(DISTINCT link_session_sha256),count(DISTINCT status),min(status),min(receipt_key)
 INTO sessions,outcomes,outcome,k FROM integration.link_session_receipt
 WHERE entity_id=a.entity_id AND book_id=a.book_id AND attempt_id=a.attempt_id;
 IF sessions=0 THEN RETURN jsonb_build_object('status','pending');END IF;
 IF sessions<>1 OR outcomes<>1 THEN RETURN jsonb_build_object('status','ambiguous');END IF;
 RETURN jsonb_build_object('status',outcome,'receipt_key',k);
END $$;

-- Only this grant admits the public receiver; signature verification remains mandatory in its Go host.
CREATE FUNCTION integration.accept_verified_link_session(env text,k text,raw text)
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

-- Public shape stays unchanged; only reconnect's private service view gains its existing consent ceiling.
ALTER FUNCTION integration.render_connection(integration.connection_attempt,boolean) RENAME TO render_connection_before_owned_repair32;
CREATE FUNCTION integration.render_connection(a integration.connection_attempt,private_value boolean)
RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;ceiling timestamptz;
BEGIN
 v:=integration.render_connection_before_owned_repair32(a,private_value);
 IF a.mode='reconnect' THEN
  v:=v||jsonb_build_object('expires_at',integration.owned_repair_deadline(a));
  IF private_value THEN
   SELECT consent_expires_at INTO ceiling FROM integration.sync_item WHERE entity_id=a.entity_id AND book_id=a.book_id AND sync_item_id=a.sync_item_id;
   v:=v||jsonb_build_object('item_consent_expires_at',ceiling);
  END IF;
 END IF;
 RETURN v;
END $$;

-- Repair creation checks old consent before any service can read its credential or mint a provider Link.
ALTER FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text) RENAME TO begin_connection_before_owned_repair32;
CREATE FUNCTION integration.begin_connection(actor uuid,e uuid,b uuid,k text,raw text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;r jsonb;a integration.connection_attempt;
BEGIN
 v:=integration.connection_json(raw,16384);
 IF v->>'mode'='reconnect' THEN
  a.actor_id:=actor;a.entity_id:=e;a.book_id:=b;a.sync_item_id:=ledger.uuid_value(v->'sync_item_id');
  PERFORM integration.owned_repair_principals(a);
 END IF;
 r:=integration.begin_connection_before_owned_repair32(actor,e,b,k,raw);
 IF r->>'mode'='reconnect' THEN
  a:=integration.owned_repair_lock(actor,e,b,(r->>'attempt_id')::uuid);
  IF a.state NOT IN ('active','abandoned') THEN
   IF a.claim_id IS NULL THEN
    IF r->'execute_link'='true'::jsonb THEN
     UPDATE integration.connection_attempt SET repair_scope_sha256=integration.owned_repair_scope_digest(a)
     WHERE entity_id=e AND book_id=b AND attempt_id=a.attempt_id RETURNING * INTO a;
    END IF;
    PERFORM integration.owned_repair_scope(a);
    UPDATE integration.connection_attempt SET expires_at=integration.owned_repair_deadline(a)
    WHERE entity_id=e AND book_id=b AND attempt_id=a.attempt_id RETURNING * INTO a;
    r:=r||integration.render_connection(a,true)||jsonb_build_object('execute_link',r->'execute_link');
   ELSE
    PERFORM integration.owned_repair_scope(a);
   END IF;
  END IF;
 END IF;
 RETURN r;
EXCEPTION WHEN SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');END $$;

-- The old six-hour record extension cannot lengthen reconnect's separately pinned deadline.
ALTER FUNCTION integration.record_link_session(uuid,uuid,uuid,uuid,uuid,text) RENAME TO record_link_session_before_owned_repair32;
CREATE FUNCTION integration.record_link_session(actor uuid,e uuid,b uuid,id uuid,nonce_value uuid,raw text)
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

-- Proof discovery reauthorizes without requiring a bank secret, manifest read or effect reservation.
CREATE FUNCTION integration.connection_link_proof(actor uuid,e uuid,b uuid,id uuid)
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

-- The old directly callable claim cannot bypass signed proof; the internal helper loses runtime grants.
ALTER FUNCTION integration.claim_connection(uuid,uuid,uuid,uuid,uuid,text) RENAME TO claim_connection_before_owned_repair32;
CREATE FUNCTION integration.claim_connection(actor uuid,e uuid,b uuid,id uuid,nonce_value uuid,raw text)
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

-- All approved accounts, including paused siblings, must retain their compatible currency/type identity.
ALTER FUNCTION integration.record_connection_manifest(uuid,uuid,uuid,uuid,uuid,text) RENAME TO record_connection_manifest_before_owned_repair32;
CREATE FUNCTION integration.record_connection_manifest(actor uuid,e uuid,b uuid,id uuid,claim_value uuid,raw text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;r jsonb;c integration.link_repair_claim;v jsonb;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.record_connection_manifest_before_owned_repair32(actor,e,b,id,claim_value,raw);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);i:=integration.owned_repair_scope(a);
 SELECT * INTO c FROM integration.link_repair_claim WHERE entity_id=e AND book_id=b AND attempt_id=id AND claim_id=a.claim_id;
 IF NOT FOUND OR c.claim_id IS DISTINCT FROM claim_value THEN PERFORM integration.connection_error('P2704');END IF;
 r:=integration.record_connection_manifest_before_owned_repair32(actor,e,b,id,claim_value,raw);
 v:=r->'manifest';
 IF (v->>'consent_expires_at')::timestamptz>least(i.consent_expires_at,c.consent_ceiling)
 THEN PERFORM integration.connection_error('P2703');END IF;
 IF EXISTS(SELECT FROM integration.sync_stream t
 JOIN banking.source_account s USING(entity_id,book_id,source_account_id)
 JOIN reference.currency currency ON currency.currency_version_id=s.currency_version_id
 JOIN ledger.account la ON la.entity_id=s.entity_id AND la.book_id=s.book_id AND la.account_id=s.ledger_account_id
 WHERE t.entity_id=e AND t.book_id=b AND t.sync_item_id=a.sync_item_id AND NOT EXISTS(
 SELECT FROM jsonb_array_elements(v->'accounts') z WHERE z.value->>'account_id'=t.account_id
 AND z.value->>'iso_currency_code'=currency.alphabetic_code
 AND ((z.value->>'type'='depository' AND la.category='asset' AND la.normal_side='debit')
 OR (z.value->>'type'='credit' AND z.value->>'subtype'='credit card' AND la.category='liability' AND la.normal_side='credit')
 OR (z.value->>'type'='loan' AND z.value->>'subtype' IN ('student','mortgage','auto','consumer','line of credit') AND la.category='liability' AND la.normal_side='credit'))))
 THEN PERFORM integration.connection_error('P2704');END IF;
 RETURN r;
END $$;

-- Same-claim confirmation atomically publishes one stable repair cause per active approved stream.
ALTER FUNCTION integration.confirm_connection_credential(uuid,uuid,uuid,uuid,uuid) RENAME TO confirm_connection_credential_before_owned_repair32;
CREATE FUNCTION integration.confirm_connection_credential(actor uuid,e uuid,b uuid,id uuid,claim_value uuid)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;c integration.link_repair_claim;t integration.sync_stream;r jsonb;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.confirm_connection_credential_before_owned_repair32(actor,e,b,id,claim_value);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);
 SELECT * INTO c FROM integration.link_repair_claim WHERE entity_id=e AND book_id=b AND attempt_id=id AND claim_id=a.claim_id;
 IF NOT FOUND OR c.claim_id IS DISTINCT FROM claim_value OR c.credential_ref IS DISTINCT FROM a.credential_ref
 OR c.credential_version IS DISTINCT FROM a.credential_version THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.state='active' THEN RETURN integration.confirm_connection_credential_before_owned_repair32(actor,e,b,id,claim_value);END IF;
 i:=integration.owned_repair_scope(a);
 IF a.manifest IS NULL OR (a.manifest->>'consent_expires_at')::timestamptz>least(i.consent_expires_at,c.consent_ceiling)
 THEN PERFORM integration.connection_error('P2703');END IF;
 r:=integration.confirm_connection_credential_before_owned_repair32(actor,e,b,id,claim_value);
 FOR t IN SELECT * FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=a.sync_item_id
 AND state='active' ORDER BY source_account_id,stream_id LOOP
  t:=integration.signal_authorize(e,b,t.stream_id);
  PERFORM integration.signal_sync(t,'repair:'||id::text,'resume');
 END LOOP;
 RETURN r;
END $$;

-- Expired or revoked reconnect cannot reach a bank-token lookup through the older context entrypoint.
ALTER FUNCTION integration.connection_context(uuid,uuid,uuid,uuid) RENAME TO connection_context_before_owned_repair32;
CREATE FUNCTION integration.connection_context(actor uuid,e uuid,b uuid,id uuid)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;c integration.link_repair_claim;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.connection_context_before_owned_repair32(actor,e,b,id);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);
 IF a.state NOT IN ('active','abandoned') THEN
  PERFORM integration.owned_repair_scope(a);
  IF a.claim_id IS NOT NULL THEN
   SELECT * INTO c FROM integration.link_repair_claim WHERE entity_id=e AND book_id=b AND attempt_id=id AND claim_id=a.claim_id;
   IF NOT FOUND THEN PERFORM integration.connection_error('P2704');END IF;
  END IF;
 END IF;
 RETURN integration.render_connection(a,true);
END $$;

-- Read the existing ciphertext by exact repair intent; never choose an unrelated older attempt by expiry.
CREATE FUNCTION integration.get_repair_bank_credential(actor uuid,e uuid,b uuid,id uuid)
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

-- The retained update token is readable only before a live unclaimed repair reservation.
ALTER FUNCTION integration.get_link_handoff(uuid,uuid,uuid,uuid) RENAME TO get_link_handoff_before_owned_repair32;
CREATE FUNCTION integration.get_link_handoff(actor uuid,e uuid,b uuid,id uuid)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.get_link_handoff_before_owned_repair32(actor,e,b,id);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);PERFORM integration.owned_repair_scope(a);
 IF a.claim_id IS NOT NULL OR a.state<>'link_ready' OR a.link_token_sha256 IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 SELECT envelope INTO v FROM integration.link_handoff WHERE entity_id=e AND book_id=b AND attempt_id=id;
 RETURN jsonb_build_object('envelope',v);
END $$;

-- Same closed AEAD profile as19; only the new reconnect eligibility branch is added.
ALTER FUNCTION integration.put_link_handoff(uuid,uuid,uuid,uuid,text) RENAME TO put_link_handoff_before_owned_repair32;
CREATE FUNCTION integration.put_link_handoff(actor uuid,e uuid,b uuid,id uuid,raw text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
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
 OR (v->>'key_version') !~ '^projects/[0-9]+/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$'
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

-- Explicit local cancellation never removes the Item or reactivates the old credential.
ALTER FUNCTION integration.abandon_connection(uuid,uuid,uuid,uuid) RENAME TO abandon_connection_before_owned_repair32;
CREATE FUNCTION integration.abandon_connection(actor uuid,e uuid,b uuid,id uuid)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.abandon_connection_before_owned_repair32(actor,e,b,id);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);
 IF a.state='abandoned' THEN RETURN integration.render_connection(a,false);END IF;
 IF a.claim_id IS NOT NULL OR a.state NOT IN ('link_pending','link_ready') THEN PERFORM integration.connection_error('P2704');END IF;
 UPDATE integration.connection_attempt SET state='abandoned' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body)
 VALUES(e,b,id,'abandoned',jsonb_build_object('reason','owner_cancelled_unclaimed_repair','provider_item_removed',false));
 RETURN integration.render_connection(a,false);
END $$;

-- Renamed privileged implementations are owner-only; runtime callers cannot bypass the proof wrappers.
REVOKE ALL ON FUNCTION
 integration.render_connection_before_owned_repair32(integration.connection_attempt,boolean),
 integration.begin_connection_before_owned_repair32(uuid,uuid,uuid,text,text),
 integration.record_link_session_before_owned_repair32(uuid,uuid,uuid,uuid,uuid,text),
 integration.claim_connection_before_owned_repair32(uuid,uuid,uuid,uuid,uuid,text),
 integration.record_connection_manifest_before_owned_repair32(uuid,uuid,uuid,uuid,uuid,text),
 integration.confirm_connection_credential_before_owned_repair32(uuid,uuid,uuid,uuid,uuid),
 integration.connection_context_before_owned_repair32(uuid,uuid,uuid,uuid),
 integration.get_link_handoff_before_owned_repair32(uuid,uuid,uuid,uuid),
 integration.put_link_handoff_before_owned_repair32(uuid,uuid,uuid,uuid,text),
 integration.abandon_connection_before_owned_repair32(uuid,uuid,uuid,uuid)
 FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION
 integration.owned_repair_attempt_guard(),
 integration.owned_repair_deadline(integration.connection_attempt),
 integration.owned_repair_principals(integration.connection_attempt),
 integration.owned_repair_scope_digest(integration.connection_attempt),
 integration.owned_repair_scope(integration.connection_attempt),
 integration.owned_repair_lock(uuid,uuid,uuid,uuid),
 integration.owned_repair_receipt_lock(integration.connection_attempt),
 integration.owned_repair_proof(integration.connection_attempt),
 integration.render_connection(integration.connection_attempt,boolean),
 integration.accept_verified_link_session(text,text,text),
 integration.connection_link_proof(uuid,uuid,uuid,uuid),
 integration.begin_connection(uuid,uuid,uuid,text,text),
 integration.record_link_session(uuid,uuid,uuid,uuid,uuid,text),
 integration.claim_connection(uuid,uuid,uuid,uuid,uuid,text),
 integration.record_connection_manifest(uuid,uuid,uuid,uuid,uuid,text),
 integration.confirm_connection_credential(uuid,uuid,uuid,uuid,uuid),
 integration.connection_context(uuid,uuid,uuid,uuid),
 integration.get_repair_bank_credential(uuid,uuid,uuid,uuid),
 integration.get_link_handoff(uuid,uuid,uuid,uuid),
 integration.put_link_handoff(uuid,uuid,uuid,uuid,text),
 integration.abandon_connection(uuid,uuid,uuid,uuid)
 FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION
 integration.connection_link_proof(uuid,uuid,uuid,uuid),
 integration.begin_connection(uuid,uuid,uuid,text,text),
 integration.record_link_session(uuid,uuid,uuid,uuid,uuid,text),
 integration.claim_connection(uuid,uuid,uuid,uuid,uuid,text),
 integration.record_connection_manifest(uuid,uuid,uuid,uuid,uuid,text),
 integration.confirm_connection_credential(uuid,uuid,uuid,uuid,uuid),
 integration.connection_context(uuid,uuid,uuid,uuid),
 integration.get_repair_bank_credential(uuid,uuid,uuid,uuid),
 integration.get_link_handoff(uuid,uuid,uuid,uuid),
 integration.put_link_handoff(uuid,uuid,uuid,uuid,text),
 integration.abandon_connection(uuid,uuid,uuid,uuid)
 TO bookkeeping_api;
GRANT EXECUTE ON FUNCTION integration.accept_verified_link_session(text,text,text) TO bookkeeping_webhook;
COMMIT;
