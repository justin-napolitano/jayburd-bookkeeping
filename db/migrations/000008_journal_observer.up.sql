-- Receipt-only journal observation. No financial data or external execution authority.
BEGIN;
DO $$ BEGIN
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='bookkeeping_publisher') THEN CREATE ROLE bookkeeping_publisher NOLOGIN; END IF;
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='bookkeeping_observer') THEN CREATE ROLE bookkeeping_observer NOLOGIN; END IF;
 IF EXISTS(SELECT FROM pg_roles WHERE rolname IN ('bookkeeping_publisher','bookkeeping_observer') AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolcanlogin OR rolreplication))
 OR EXISTS(SELECT FROM pg_roles r CROSS JOIN pg_roles p WHERE r.rolname IN ('bookkeeping_publisher','bookkeeping_observer') AND p.rolname IN ('bookkeeping_api','bookkeeping_owner','bookkeeping_migrator','bookkeeping_identity_operator','bookkeeping_publisher','bookkeeping_observer') AND r.oid<>p.oid AND (pg_has_role(r.oid,p.oid,'MEMBER') OR pg_has_role(p.oid,r.oid,'MEMBER'))) THEN
 RAISE EXCEPTION USING ERRCODE='P2205',MESSAGE='worker_configuration_rejected'; END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE integration.observer_binding (
 binding_id uuid PRIMARY KEY DEFAULT gen_random_uuid(), entity_id uuid NOT NULL, book_id uuid NOT NULL,
 principal_id uuid NOT NULL REFERENCES ledger.principal, publisher_login name NOT NULL, consumer_login name NOT NULL,
 consumer_id text NOT NULL DEFAULT 'journal.observe' CHECK(consumer_id='journal.observe'), handler_version integer NOT NULL DEFAULT 1 CHECK(handler_version=1),
 binding_revision bigint NOT NULL DEFAULT 1 CHECK(binding_revision>0), enabled boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)), updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(updated_at)),
 CHECK(publisher_login<>consumer_login), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,
 UNIQUE(entity_id,book_id,consumer_id,handler_version), UNIQUE(entity_id,book_id,binding_id)
);
CREATE TABLE integration.work_operation (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, operation_id uuid NOT NULL DEFAULT gen_random_uuid(), binding_id uuid NOT NULL,
 message_id uuid NOT NULL, consumer_id text NOT NULL DEFAULT 'journal.observe' CHECK(consumer_id='journal.observe'), handler_version integer NOT NULL DEFAULT 1 CHECK(handler_version=1),
 request_digest bytea NOT NULL CHECK(octet_length(request_digest)=32), created_at timestamptz NOT NULL CHECK(isfinite(created_at)),
 state text NOT NULL CHECK(state IN ('ready','running','retry_wait','succeeded','blocked','quarantined')),
 attempt_count bigint NOT NULL DEFAULT 0 CHECK(attempt_count>=0), failed_attempt_count bigint NOT NULL DEFAULT 0 CHECK(failed_attempt_count BETWEEN 0 AND attempt_count),
 execution_generation bigint NOT NULL DEFAULT 0 CHECK(execution_generation=attempt_count), fence_token bigint NOT NULL DEFAULT 0 CHECK(fence_token>=0),
 dispatch_generation bigint NOT NULL DEFAULT 1 CHECK(dispatch_generation>0), next_eligible_at timestamptz NOT NULL CHECK(isfinite(next_eligible_at)),
 first_claimed_at timestamptz CHECK(isfinite(first_claimed_at)), deadline_at timestamptz CHECK(isfinite(deadline_at)),
 lease_holder uuid, lease_until timestamptz CHECK(isfinite(lease_until)), claim_binding_revision bigint CHECK(claim_binding_revision>0),
 terminal_code text, finished_at timestamptz CHECK(isfinite(finished_at)),
 PRIMARY KEY(entity_id,book_id,operation_id), UNIQUE(operation_id), UNIQUE(entity_id,book_id,message_id,consumer_id,handler_version),
 UNIQUE(entity_id,book_id,message_id,consumer_id,handler_version,operation_id),
 FOREIGN KEY(entity_id,book_id,binding_id) REFERENCES integration.observer_binding(entity_id,book_id,binding_id),
 FOREIGN KEY(entity_id,book_id,message_id) REFERENCES integration.outbox_message,
 CHECK((first_claimed_at IS NULL AND deadline_at IS NULL AND attempt_count=0) OR (first_claimed_at IS NOT NULL AND deadline_at IS NOT NULL AND deadline_at=first_claimed_at+interval '24 hours' AND attempt_count>0)),
 CHECK((state='running' AND lease_holder IS NOT NULL AND lease_until IS NOT NULL AND claim_binding_revision IS NOT NULL) OR (state<>'running' AND lease_holder IS NULL AND lease_until IS NULL AND claim_binding_revision IS NULL)),
 CHECK((state IN ('ready','running','retry_wait') AND terminal_code IS NULL AND finished_at IS NULL) OR
 (finished_at IS NOT NULL AND terminal_code IS NOT NULL AND ((state='succeeded' AND terminal_code='observed') OR (state='blocked' AND terminal_code='authority_revoked') OR (state='quarantined' AND terminal_code IN ('event_mismatch','retry_exhausted','age_exhausted')))))
);
CREATE INDEX work_due ON integration.work_operation(next_eligible_at,operation_id) WHERE state IN ('ready','retry_wait');
CREATE INDEX work_expired ON integration.work_operation(lease_until,operation_id) WHERE state='running';
CREATE INDEX work_binding ON integration.work_operation(binding_id,state,operation_id);
CREATE TABLE integration.outbox_delivery (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, message_id uuid NOT NULL, consumer_id text NOT NULL CHECK(consumer_id='journal.observe'), handler_version integer NOT NULL CHECK(handler_version=1), operation_id uuid NOT NULL,
 created_at timestamptz NOT NULL CHECK(isfinite(created_at)), PRIMARY KEY(entity_id,book_id,message_id,consumer_id,handler_version), UNIQUE(entity_id,book_id,operation_id),
 FOREIGN KEY(entity_id,book_id,message_id) REFERENCES integration.outbox_message,
 FOREIGN KEY(entity_id,book_id,message_id,consumer_id,handler_version,operation_id) REFERENCES integration.work_operation(entity_id,book_id,message_id,consumer_id,handler_version,operation_id)
);
CREATE TABLE integration.work_dispatch (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, operation_id uuid NOT NULL, dispatch_generation bigint NOT NULL CHECK(dispatch_generation>0), target_version integer NOT NULL DEFAULT 1 CHECK(target_version=1),
 task_name_digest bytea NOT NULL UNIQUE CHECK(octet_length(task_name_digest)=32), state text NOT NULL CHECK(state IN ('pending','publishing','enqueued','uncertain','retired')),
 enqueue_attempt_count bigint NOT NULL DEFAULT 0 CHECK(enqueue_attempt_count>=0), publisher_fence bigint NOT NULL DEFAULT 0 CHECK(publisher_fence>=0), lease_holder uuid, lease_until timestamptz CHECK(isfinite(lease_until)),
 created_at timestamptz NOT NULL CHECK(isfinite(created_at)), last_attempt_at timestamptz CHECK(isfinite(last_attempt_at)), last_result_at timestamptz CHECK(isfinite(last_result_at)),
 result_code text CHECK(result_code IN ('accepted','already_exists','timeout','transient','rejected')),
 PRIMARY KEY(entity_id,book_id,operation_id,dispatch_generation), FOREIGN KEY(entity_id,book_id,operation_id) REFERENCES integration.work_operation,
 CHECK((state='publishing' AND lease_holder IS NOT NULL AND lease_until IS NOT NULL) OR (state<>'publishing' AND lease_holder IS NULL AND lease_until IS NULL))
);
CREATE INDEX dispatch_due ON integration.work_dispatch(created_at,operation_id,dispatch_generation) WHERE state IN ('pending','uncertain');
CREATE INDEX dispatch_expired ON integration.work_dispatch(lease_until,operation_id,dispatch_generation) WHERE state='publishing';
CREATE TABLE integration.work_attempt (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, operation_id uuid NOT NULL, execution_generation bigint NOT NULL CHECK(execution_generation>0), attempt_id uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
 fence_token bigint NOT NULL CHECK(fence_token>0), holder_id uuid NOT NULL, binding_revision bigint NOT NULL CHECK(binding_revision>0), started_at timestamptz NOT NULL CHECK(isfinite(started_at)),
 PRIMARY KEY(entity_id,book_id,operation_id,execution_generation), FOREIGN KEY(entity_id,book_id,operation_id) REFERENCES integration.work_operation,
 UNIQUE(entity_id,book_id,operation_id,execution_generation,fence_token)
);
CREATE TABLE integration.work_receipt (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, operation_id uuid NOT NULL, execution_generation bigint NOT NULL CHECK(execution_generation>0), fence_token bigint NOT NULL CHECK(fence_token>0),
 outcome text NOT NULL CHECK(outcome IN ('observed','retryable','abandoned','authority_revoked','event_mismatch','retry_exhausted','age_exhausted')),
 request_digest bytea NOT NULL CHECK(octet_length(request_digest)=32), binding_revision bigint NOT NULL CHECK(binding_revision>0), recorded_at timestamptz NOT NULL CHECK(isfinite(recorded_at)),
 PRIMARY KEY(entity_id,book_id,operation_id,execution_generation),
 FOREIGN KEY(entity_id,book_id,operation_id,execution_generation,fence_token) REFERENCES integration.work_attempt(entity_id,book_id,operation_id,execution_generation,fence_token)
);
CREATE UNIQUE INDEX one_observation ON integration.work_receipt(entity_id,book_id,operation_id) WHERE outcome='observed';
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.outbox_delivery FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.work_attempt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.work_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();

-- Fixed diagnostics do not disclose existence, account data or caller-controlled text.
CREATE FUNCTION integration.observer_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='worker_operation_rejected'; END $$;
-- Revisioned enablement is the only binding mutation; identity handoff is not implicit.
CREATE FUNCTION integration.observer_binding_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.observer_error('P2205'); END IF;
 IF (to_jsonb(NEW)-ARRAY['enabled','binding_revision','updated_at']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['enabled','binding_revision','updated_at']) OR NEW.enabled=OLD.enabled OR OLD.binding_revision=9223372036854775807 OR NEW.binding_revision<>OLD.binding_revision+1 THEN PERFORM integration.observer_error('P2205'); END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER binding_guard BEFORE UPDATE OR DELETE ON integration.observer_binding FOR EACH ROW EXECUTE FUNCTION integration.observer_binding_guard();
-- Immutable operation identity and terminal records cannot be reset by ordinary owner DML.
CREATE FUNCTION integration.observer_operation_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.observer_error('P2205'); END IF;
 IF ROW(NEW.entity_id,NEW.book_id,NEW.operation_id,NEW.binding_id,NEW.message_id,NEW.consumer_id,NEW.handler_version,NEW.request_digest,NEW.created_at) IS DISTINCT FROM ROW(OLD.entity_id,OLD.book_id,OLD.operation_id,OLD.binding_id,OLD.message_id,OLD.consumer_id,OLD.handler_version,OLD.request_digest,OLD.created_at)
 OR OLD.state IN ('succeeded','blocked','quarantined') OR NEW.attempt_count<OLD.attempt_count OR NEW.failed_attempt_count<OLD.failed_attempt_count OR NEW.fence_token<OLD.fence_token OR NEW.dispatch_generation<OLD.dispatch_generation OR (OLD.first_claimed_at IS NOT NULL AND NEW.first_claimed_at IS DISTINCT FROM OLD.first_claimed_at) THEN PERFORM integration.observer_error('P2205'); END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER operation_guard BEFORE UPDATE OR DELETE ON integration.work_operation FOR EACH ROW EXECUTE FUNCTION integration.observer_operation_guard();
-- Every deferred check reads final rows, so operation/receipt insertion order is irrelevant.
CREATE FUNCTION integration.observer_consistency() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; a integration.work_attempt; n bigint; failures bigint; observed bigint;
BEGIN
 SELECT * INTO o FROM integration.work_operation WHERE operation_id=NEW.operation_id;
 SELECT count(*),count(*) FILTER(WHERE outcome NOT IN ('observed','authority_revoked')),count(*) FILTER(WHERE outcome='observed') INTO n,failures,observed FROM integration.work_receipt WHERE operation_id=o.operation_id;
 IF o.attempt_count<>(SELECT count(*) FROM integration.work_attempt WHERE operation_id=o.operation_id) OR failures<>o.failed_attempt_count OR (o.state='running' AND n<>o.attempt_count-1) OR (o.state<>'running' AND n<>o.attempt_count) OR (o.state='succeeded')<>(observed=1)
 OR NOT EXISTS(SELECT FROM integration.outbox_delivery WHERE operation_id=o.operation_id) OR NOT EXISTS(SELECT FROM integration.work_dispatch WHERE operation_id=o.operation_id AND dispatch_generation=o.dispatch_generation AND state<>'retired')
 OR EXISTS(SELECT FROM integration.work_receipt r JOIN integration.work_attempt t USING(entity_id,book_id,operation_id,execution_generation) WHERE r.operation_id=o.operation_id AND (r.binding_revision<>t.binding_revision OR r.request_digest<>o.request_digest)) THEN PERFORM integration.observer_error('P2205'); END IF;
 IF o.state='running' THEN
 SELECT * INTO a FROM integration.work_attempt WHERE operation_id=o.operation_id AND execution_generation=o.execution_generation;
 IF ROW(o.fence_token,o.lease_holder,o.claim_binding_revision) IS DISTINCT FROM ROW(a.fence_token,a.holder_id,a.binding_revision) OR o.lease_until>a.started_at+interval '90 seconds' THEN PERFORM integration.observer_error('P2205'); END IF;
 END IF;
 RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER operation_consistency AFTER INSERT OR UPDATE ON integration.work_operation DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION integration.observer_consistency();
CREATE CONSTRAINT TRIGGER attempt_consistency AFTER INSERT ON integration.work_attempt DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION integration.observer_consistency();
CREATE CONSTRAINT TRIGGER receipt_consistency AFTER INSERT ON integration.work_receipt DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION integration.observer_consistency();
CREATE CONSTRAINT TRIGGER dispatch_consistency AFTER INSERT OR UPDATE ON integration.work_dispatch DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION integration.observer_consistency();

-- Validate transaction mode before work; login identity is checked even on denied scope.
CREATE FUNCTION integration.observer_binding_lock(id uuid,publisher boolean) RETURNS integration.observer_binding LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE b integration.observer_binding;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR id IS NULL THEN PERFORM integration.observer_error('P2200'); END IF;
 PERFORM set_config('lock_timeout','1s',true);
 SELECT * INTO b FROM integration.observer_binding WHERE binding_id=id FOR SHARE;
 IF NOT FOUND OR session_user<>(CASE WHEN publisher THEN b.publisher_login ELSE b.consumer_login END) THEN PERFORM integration.observer_error('P2201'); END IF;
 RETURN b;
END $$;
-- Retain locks even when scope is revoked: never catch authorize inside a subtransaction.
CREATE FUNCTION integration.observer_authorized(b integration.observer_binding) RETURNS boolean LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE ok boolean:=b.enabled; active boolean;
BEGIN
 SELECT disabled_at IS NULL INTO active FROM ledger.principal WHERE principal_id=b.principal_id FOR SHARE; ok:=ok AND coalesce(active,false);
 SELECT disabled_at IS NULL INTO active FROM ledger.entity WHERE entity_id=b.entity_id FOR SHARE; ok:=ok AND coalesce(active,false);
 SELECT revoked_at IS NULL INTO active FROM ledger.membership WHERE entity_id=b.entity_id AND principal_id=b.principal_id AND permission='read' FOR SHARE; ok:=ok AND coalesce(active,false);
 SELECT disabled_at IS NULL INTO active FROM ledger.book WHERE entity_id=b.entity_id AND book_id=b.book_id FOR SHARE; ok:=ok AND coalesce(active,false);
 IF ok THEN PERFORM ledger.authorize(b.principal_id,b.entity_id,b.book_id,'read',false); END IF;
 RETURN ok;
END $$;
-- Resolve scope without disclosing it; auth locks precede the mutable operation lock.
CREATE FUNCTION integration.observer_lock(id uuid,publisher boolean,OUT operation integration.work_operation,OUT binding integration.observer_binding,OUT authorized boolean) RETURNS record LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE bid uuid;
BEGIN
 IF id IS NULL THEN PERFORM integration.observer_error('P2200'); END IF;
 SELECT binding_id INTO bid FROM integration.work_operation WHERE operation_id=id;
 IF bid IS NULL THEN PERFORM integration.observer_error('P2201'); END IF;
 binding:=integration.observer_binding_lock(bid,publisher); authorized:=integration.observer_authorized(binding);
 SELECT * INTO operation FROM integration.work_operation WHERE operation_id=id FOR UPDATE;
END $$;
-- Source digest deliberately excludes transient binding and dispatch state.
CREATE FUNCTION integration.observer_digest(m integration.outbox_message) RETURNS bytea LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT sha256(convert_to('journal.observe|1|'||m.entity_id::text||'|'||m.book_id::text||'|'||m.message_id::text||'|'||m.target_id::text||'|'||m.book_revision::text,'UTF8'))
$$;
-- Validate the existing immutable event/header without copying financial amounts.
CREATE FUNCTION integration.observer_event(o integration.work_operation) RETURNS boolean LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM integration.outbox_message m JOIN ledger.journal_entry j ON ROW(j.entity_id,j.book_id,j.journal_id)=ROW(m.entity_id,m.book_id,m.target_id)
 WHERE ROW(m.entity_id,m.book_id,m.message_id)=ROW(o.entity_id,o.book_id,o.message_id) AND m.event_type='journal.posted' AND m.event_version=1 AND m.book_revision=j.book_revision AND integration.observer_digest(m)=o.request_digest)
$$;
-- Each dispatch identity is deterministic and contains no source data.
CREATE FUNCTION integration.observer_new_dispatch(o integration.work_operation,t timestamptz) RETURNS void LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 INSERT INTO integration.work_dispatch(entity_id,book_id,operation_id,dispatch_generation,task_name_digest,state,created_at)
 VALUES(o.entity_id,o.book_id,o.operation_id,o.dispatch_generation,sha256(convert_to(o.operation_id::text||'|'||o.dispatch_generation::text||'|1','UTF8')),'pending',t)
$$;
-- Close a held execution at most once; caller already owns the operation lock.
CREATE FUNCTION integration.observer_end(o integration.work_operation,outcome_value text,t timestamptz) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF o.state='running' THEN
 INSERT INTO integration.work_receipt VALUES(o.entity_id,o.book_id,o.operation_id,o.execution_generation,o.fence_token,outcome_value,o.request_digest,o.claim_binding_revision,t);
 END IF;
END $$;
-- Apply terminal and retry rules once, never fabricating an execution-zero receipt.
CREATE FUNCTION integration.observer_close(o integration.work_operation,reason text) RETURNS text LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE t timestamptz:=clock_timestamp(); failed bigint:=o.failed_attempt_count; s text; code text; delay_seconds numeric;
BEGIN
 IF o.fence_token=9223372036854775807 THEN PERFORM integration.observer_error('P2204'); END IF;
 IF o.state='running' AND reason<>'authority_revoked' THEN
 IF failed=9223372036854775807 THEN PERFORM integration.observer_error('P2204'); END IF; failed:=failed+1;
 END IF;
 IF reason='authority_revoked' THEN s:='blocked'; code:=reason;
 ELSIF reason='observed' THEN s:='succeeded'; code:=reason; failed:=o.failed_attempt_count;
 ELSIF reason='event_mismatch' THEN s:='quarantined'; code:=reason;
 ELSIF o.deadline_at IS NOT NULL AND t>=o.deadline_at THEN s:='quarantined'; code:='age_exhausted';
 ELSIF failed>=8 THEN s:='quarantined'; code:='retry_exhausted';
 ELSE s:='retry_wait'; END IF;
 PERFORM integration.observer_end(o,CASE WHEN code='age_exhausted' THEN code ELSE reason END,t);
 delay_seconds:=least(900::numeric,5*power(2::numeric,greatest(0,least(failed-1,8))));
 UPDATE integration.work_operation SET state=s,failed_attempt_count=failed,fence_token=fence_token+1,lease_holder=NULL,lease_until=NULL,claim_binding_revision=NULL,
 terminal_code=code,finished_at=CASE WHEN code IS NOT NULL THEN t END,
 next_eligible_at=CASE WHEN s='retry_wait' THEN t+(delay_seconds/2+random()*delay_seconds/2)*interval '1 second' ELSE next_eligible_at END WHERE operation_id=o.operation_id;
 RETURN s;
END $$;
-- Stale holders cannot manufacture failure/success after deadline, expiry or takeover.
CREATE FUNCTION integration.observer_live(o integration.work_operation,b integration.observer_binding,h uuid,g bigint,f bigint) RETURNS boolean LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT coalesce(o.state='running' AND o.lease_holder=h AND o.execution_generation=g AND o.fence_token=f AND o.claim_binding_revision=b.binding_revision AND clock_timestamp()<o.lease_until AND clock_timestamp()<o.deadline_at,false)
$$;

-- Publisher scans one explicitly configured book, committing sidecar and intent together.
CREATE FUNCTION integration.observe_discover(id uuid,maximum integer) RETURNS integer LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE b integration.observer_binding; m integration.outbox_message; o integration.work_operation; n integer:=0; t timestamptz;
BEGIN
 b:=integration.observer_binding_lock(id,true);
 IF NOT integration.observer_authorized(b) THEN PERFORM integration.observer_error('P2201'); END IF;
 IF maximum IS NULL OR maximum NOT BETWEEN 1 AND 25 THEN PERFORM integration.observer_error('P2200'); END IF;
 FOR m IN SELECT x.* FROM integration.outbox_message x WHERE x.entity_id=b.entity_id AND x.book_id=b.book_id AND NOT EXISTS(SELECT FROM integration.outbox_delivery d WHERE ROW(d.entity_id,d.book_id,d.message_id)=ROW(x.entity_id,x.book_id,x.message_id)) ORDER BY x.created_at,x.message_id LIMIT maximum LOOP
 t:=clock_timestamp();
 INSERT INTO integration.work_operation(entity_id,book_id,binding_id,message_id,request_digest,created_at,state,next_eligible_at) VALUES(m.entity_id,m.book_id,b.binding_id,m.message_id,integration.observer_digest(m),t,'ready',t) ON CONFLICT DO NOTHING RETURNING * INTO o;
 IF NOT FOUND THEN
 SELECT * INTO o FROM integration.work_operation WHERE entity_id=m.entity_id AND book_id=m.book_id AND message_id=m.message_id AND consumer_id='journal.observe' AND handler_version=1;
 IF o.request_digest IS DISTINCT FROM integration.observer_digest(m) OR o.binding_id IS DISTINCT FROM b.binding_id THEN PERFORM integration.observer_error('P2202'); END IF;
 CONTINUE;
 END IF;
 INSERT INTO integration.outbox_delivery VALUES(m.entity_id,m.book_id,m.message_id,'journal.observe',1,o.operation_id,t);
 PERFORM integration.observer_new_dispatch(o,t);
 IF NOT integration.observer_event(o) THEN PERFORM integration.observer_close(o,'event_mismatch'); END IF;
 n:=n+1;
 END LOOP;
 RETURN n;
END $$;
-- Candidate enumeration is bounded and nonauthoritative; every later call locks anew.
CREATE FUNCTION integration.observe_candidates(id uuid,maximum integer) RETURNS TABLE(operation_id uuid) LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE b integration.observer_binding;
BEGIN
 b:=integration.observer_binding_lock(id,true);
 IF NOT integration.observer_authorized(b) THEN PERFORM integration.observer_error('P2201'); END IF;
 IF maximum IS NULL OR maximum NOT BETWEEN 1 AND 25 THEN PERFORM integration.observer_error('P2200'); END IF;
 RETURN QUERY SELECT o.operation_id FROM integration.work_operation o WHERE o.binding_id=id AND ((o.state IN ('ready','retry_wait') AND o.next_eligible_at<=clock_timestamp()) OR (o.state='running' AND (o.lease_until<=clock_timestamp() OR o.deadline_at<=clock_timestamp()))) ORDER BY o.next_eligible_at,o.operation_id LIMIT maximum;
END $$;
-- Only enqueue ownership changes here; consumer state never follows transport acknowledgement.
CREATE FUNCTION integration.observe_dispatch_claim(id uuid,h uuid) RETURNS TABLE(status text,dispatch_generation bigint,target_version integer,task_name_hex text,publisher_fence bigint,lease_until timestamptz) LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; b integration.observer_binding; authorized boolean; locked record; d integration.work_dispatch; t timestamptz;
BEGIN
 SELECT * INTO locked FROM integration.observer_lock(id,true); o:=locked.operation; b:=locked.binding; authorized:=locked.authorized;
 IF NOT authorized THEN PERFORM integration.observer_error('P2201'); END IF;
 IF h IS NULL THEN PERFORM integration.observer_error('P2200'); END IF;
 SELECT x.* INTO d FROM integration.work_dispatch x WHERE x.operation_id=id AND x.dispatch_generation=o.dispatch_generation FOR UPDATE;
 t:=clock_timestamp();
 IF o.state NOT IN ('ready','retry_wait') OR o.next_eligible_at>t OR (o.deadline_at IS NOT NULL AND t>=o.deadline_at) OR NOT(d.state IN ('pending','uncertain') OR (d.state='publishing' AND d.lease_until<=t)) THEN RETURN QUERY SELECT 'not_due'::text,NULL::bigint,NULL::integer,NULL::text,NULL::bigint,NULL::timestamptz; RETURN; END IF;
 IF d.publisher_fence=9223372036854775807 OR d.enqueue_attempt_count=9223372036854775807 THEN PERFORM integration.observer_error('P2204'); END IF;
 UPDATE integration.work_dispatch x SET state='publishing',publisher_fence=x.publisher_fence+1,enqueue_attempt_count=x.enqueue_attempt_count+1,lease_holder=h,lease_until=t+interval '30 seconds',last_attempt_at=t WHERE x.operation_id=id AND x.dispatch_generation=d.dispatch_generation RETURNING x.* INTO d;
 RETURN QUERY SELECT 'claimed'::text,d.dispatch_generation,d.target_version,encode(d.task_name_digest,'hex'),d.publisher_fence,d.lease_until;
END $$;
-- Acknowledgement changes dispatch metadata only; it never implies consumer completion.
CREATE FUNCTION integration.observe_dispatch_result(id uuid,g bigint,h uuid,f bigint,r text) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; b integration.observer_binding; authorized boolean; locked record; d integration.work_dispatch; t timestamptz;
BEGIN
 SELECT * INTO locked FROM integration.observer_lock(id,true); o:=locked.operation; b:=locked.binding; authorized:=locked.authorized;
 IF NOT authorized THEN PERFORM integration.observer_error('P2201'); END IF;
 IF h IS NULL OR g IS NULL OR g<1 OR f IS NULL OR f<1 OR r IS NULL OR r NOT IN ('accepted','already_exists','timeout','transient','rejected') THEN PERFORM integration.observer_error('P2200'); END IF;
 SELECT * INTO d FROM integration.work_dispatch WHERE operation_id=id AND dispatch_generation=g FOR UPDATE;
 t:=clock_timestamp();
 IF NOT FOUND OR d.state<>'publishing' OR d.lease_holder<>h OR d.publisher_fence<>f OR t>=d.lease_until THEN RETURN false; END IF;
 UPDATE integration.work_dispatch SET state=CASE WHEN r='accepted' THEN 'enqueued' ELSE 'uncertain' END,lease_holder=NULL,lease_until=NULL,last_result_at=t,result_code=r WHERE operation_id=id AND dispatch_generation=g;
 RETURN true;
END $$;
-- Claim is an authorization and fencing boundary, not a queue-delivery counter.
CREATE FUNCTION integration.observe_claim(id uuid,g bigint,h uuid) RETURNS TABLE(status text,execution_generation bigint,fence_token bigint,lease_until timestamptz) LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; b integration.observer_binding; authorized boolean; locked record; t timestamptz; closed_state text;
BEGIN
 SELECT * INTO locked FROM integration.observer_lock(id,false); o:=locked.operation; b:=locked.binding; authorized:=locked.authorized;
 IF h IS NULL OR g IS NULL OR g<1 OR NOT EXISTS(SELECT FROM integration.work_dispatch d WHERE d.operation_id=id AND d.dispatch_generation=g) THEN PERFORM integration.observer_error('P2200'); END IF;
 IF o.state IN ('succeeded','blocked','quarantined') THEN status:='done';
 ELSIF NOT authorized THEN PERFORM integration.observer_close(o,'authority_revoked'); status:='done';
 ELSE
 t:=clock_timestamp();
 IF o.deadline_at IS NOT NULL AND t>=o.deadline_at THEN PERFORM integration.observer_close(o,'age_exhausted'); status:='done';
 ELSIF NOT integration.observer_event(o) THEN PERFORM integration.observer_close(o,'event_mismatch'); status:='done';
 ELSIF o.state='running' THEN
 IF t<o.lease_until THEN status:='busy'; ELSE closed_state:=integration.observer_close(o,'abandoned'); status:=CASE WHEN closed_state='quarantined' THEN 'done' ELSE 'not_due' END; END IF;
 ELSIF o.next_eligible_at>t THEN status:='not_due';
 ELSE
 IF o.attempt_count=9223372036854775807 OR o.fence_token=9223372036854775807 THEN PERFORM integration.observer_error('P2204'); END IF;
 t:=clock_timestamp();
 INSERT INTO integration.work_attempt(entity_id,book_id,operation_id,execution_generation,fence_token,holder_id,binding_revision,started_at) VALUES(o.entity_id,o.book_id,id,o.execution_generation+1,o.fence_token+1,h,b.binding_revision,t);
 UPDATE integration.work_operation x SET state='running',attempt_count=x.attempt_count+1,execution_generation=x.execution_generation+1,fence_token=x.fence_token+1,first_claimed_at=coalesce(x.first_claimed_at,t),deadline_at=coalesce(x.deadline_at,t+interval '24 hours'),lease_holder=h,lease_until=t+interval '90 seconds',claim_binding_revision=b.binding_revision WHERE x.operation_id=id RETURNING x.* INTO o;
 status:='claimed'; execution_generation:=o.execution_generation; fence_token:=o.fence_token; lease_until:=o.lease_until;
 END IF;
 END IF;
 RETURN NEXT;
END $$;
-- Heartbeat cannot extend the original ninety-second attempt or revive expiry.
CREATE FUNCTION integration.observe_heartbeat(id uuid,h uuid,g bigint,f bigint) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; b integration.observer_binding; authorized boolean; locked record; t timestamptz;
BEGIN
 SELECT * INTO locked FROM integration.observer_lock(id,false); o:=locked.operation; b:=locked.binding; authorized:=locked.authorized;
 IF NOT authorized OR NOT integration.observer_live(o,b,h,g,f) THEN RETURN false; END IF;
 t:=clock_timestamp();
 UPDATE integration.work_operation SET lease_until=least(t+interval '90 seconds',(SELECT started_at+interval '90 seconds' FROM integration.work_attempt WHERE operation_id=id AND execution_generation=g)) WHERE operation_id=id;
 RETURN true;
END $$;
-- Finish binds one receipt to the current unexpired execution and immutable journal fact.
CREATE FUNCTION integration.observe_finish(id uuid,h uuid,g bigint,f bigint) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; b integration.observer_binding; authorized boolean; locked record;
BEGIN
 SELECT * INTO locked FROM integration.observer_lock(id,false); o:=locked.operation; b:=locked.binding; authorized:=locked.authorized;
 IF o.state IN ('succeeded','blocked','quarantined') THEN RETURN 'done'; END IF;
 IF NOT authorized THEN PERFORM integration.observer_close(o,'authority_revoked'); RETURN 'done'; END IF;
 IF o.deadline_at IS NOT NULL AND clock_timestamp()>=o.deadline_at THEN PERFORM integration.observer_close(o,'age_exhausted'); RETURN 'done'; END IF;
 IF NOT integration.observer_live(o,b,h,g,f) THEN PERFORM integration.observer_error('P2203'); END IF;
 IF NOT integration.observer_event(o) THEN PERFORM integration.observer_close(o,'event_mismatch'); RETURN 'done'; END IF;
 IF NOT integration.observer_live(o,b,h,g,f) THEN PERFORM integration.observer_error('P2203'); END IF;
 PERFORM integration.observer_close(o,'observed'); RETURN 'observed';
END $$;
-- Trusted internal failures have one closed code and consume exactly one execution budget.
CREATE FUNCTION integration.observe_fail(id uuid,h uuid,g bigint,f bigint) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; b integration.observer_binding; authorized boolean; locked record;
BEGIN
 SELECT * INTO locked FROM integration.observer_lock(id,false); o:=locked.operation; b:=locked.binding; authorized:=locked.authorized;
 IF o.state IN ('succeeded','blocked','quarantined') THEN RETURN 'done'; END IF;
 IF NOT authorized THEN PERFORM integration.observer_close(o,'authority_revoked'); RETURN 'done'; END IF;
 IF o.deadline_at IS NOT NULL AND clock_timestamp()>=o.deadline_at THEN RETURN integration.observer_close(o,'age_exhausted'); END IF;
 IF NOT integration.observer_live(o,b,h,g,f) THEN PERFORM integration.observer_error('P2203'); END IF;
 RETURN integration.observer_close(o,'retryable');
END $$;
-- Recovery owns abandonment and missing-delivery redispatch, never resets terminal work.
CREATE FUNCTION integration.observe_recover(id uuid) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.work_operation; b integration.observer_binding; authorized boolean; locked record; d integration.work_dispatch; t timestamptz; result text;
BEGIN
 SELECT * INTO locked FROM integration.observer_lock(id,true); o:=locked.operation; b:=locked.binding; authorized:=locked.authorized;
 IF o.state IN ('succeeded','blocked','quarantined') THEN RETURN 'done'; END IF;
 IF NOT authorized THEN RETURN integration.observer_close(o,'authority_revoked'); END IF;
 SELECT * INTO d FROM integration.work_dispatch WHERE operation_id=id AND dispatch_generation=o.dispatch_generation FOR UPDATE;
 t:=clock_timestamp();
 IF o.deadline_at IS NOT NULL AND t>=o.deadline_at THEN RETURN integration.observer_close(o,'age_exhausted'); END IF;
 IF o.state='running' THEN
 IF t<o.lease_until THEN RETURN 'no_change'; END IF;
 RETURN integration.observer_close(o,'abandoned');
 END IF;
 IF o.next_eligible_at>t THEN RETURN 'no_change'; END IF;
 -- An expired publisher claim first retries its stable task identity, even if old.
 IF d.state='publishing' THEN
 IF t<d.lease_until THEN RETURN 'no_change'; END IF;
 IF d.enqueue_attempt_count<=1 OR d.created_at>t-interval '15 minutes' THEN
 UPDATE integration.work_dispatch SET state='uncertain',lease_holder=NULL,lease_until=NULL,last_result_at=t,result_code='timeout' WHERE operation_id=id AND dispatch_generation=d.dispatch_generation;
 RETURN 'no_change';
 END IF;
 END IF;
 IF d.result_code IS DISTINCT FROM 'rejected' AND d.created_at>t-interval '15 minutes' THEN RETURN 'no_change'; END IF;
 IF o.dispatch_generation=9223372036854775807 THEN PERFORM integration.observer_error('P2204'); END IF;
 UPDATE integration.work_dispatch SET state='retired',lease_holder=NULL,lease_until=NULL WHERE operation_id=id AND dispatch_generation=d.dispatch_generation;
 UPDATE integration.work_operation SET dispatch_generation=dispatch_generation+1 WHERE operation_id=id RETURNING * INTO o;
 PERFORM integration.observer_new_dispatch(o,t); RETURN 'redispatched';
END $$;
-- Only the small published surface is executable; tables and helper functions remain owner-only.
REVOKE ALL ON integration.observer_binding,integration.work_operation,integration.outbox_delivery,integration.work_dispatch,integration.work_attempt,integration.work_receipt FROM PUBLIC,bookkeeping_api,bookkeeping_publisher,bookkeeping_observer;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA integration FROM PUBLIC,bookkeeping_publisher,bookkeeping_observer;
GRANT USAGE ON SCHEMA integration TO bookkeeping_publisher,bookkeeping_observer;
GRANT EXECUTE ON FUNCTION integration.observe_discover(uuid,integer),integration.observe_candidates(uuid,integer),integration.observe_dispatch_claim(uuid,uuid),integration.observe_dispatch_result(uuid,bigint,uuid,bigint,text),integration.observe_recover(uuid) TO bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.observe_claim(uuid,bigint,uuid),integration.observe_heartbeat(uuid,uuid,bigint,bigint),integration.observe_finish(uuid,uuid,bigint,bigint),integration.observe_fail(uuid,uuid,bigint,bigint) TO bookkeeping_observer;
COMMIT;
