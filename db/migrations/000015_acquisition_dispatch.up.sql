-- Scoped delivery sidecars; acquisition generations and publication remain in sync tables.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE integration.sync_delivery_binding (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,binding_id uuid NOT NULL DEFAULT gen_random_uuid(),service_principal uuid NOT NULL REFERENCES ledger.principal,publisher_login name NOT NULL,worker_login name NOT NULL,binding_revision bigint NOT NULL DEFAULT 1 CHECK(binding_revision>0),enabled boolean NOT NULL,
 PRIMARY KEY(binding_id),UNIQUE(entity_id,book_id),UNIQUE(entity_id,book_id,binding_id),FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,CHECK(publisher_login<>worker_login)
);
CREATE TABLE integration.sync_delivery_operation (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,operation_id uuid NOT NULL DEFAULT gen_random_uuid(),binding_id uuid NOT NULL,stream_id uuid NOT NULL,target_generation bigint NOT NULL CHECK(target_generation>0),claim_id uuid NOT NULL DEFAULT gen_random_uuid(),claim_round bigint NOT NULL DEFAULT 1 CHECK(claim_round>0),
 state text NOT NULL DEFAULT 'ready' CHECK(state IN ('ready','running','retry_wait','succeeded','quarantined')),failed_count integer NOT NULL DEFAULT 0 CHECK(failed_count BETWEEN 0 AND 8),dispatch_generation bigint NOT NULL DEFAULT 1 CHECK(dispatch_generation>0),run_fence bigint NOT NULL DEFAULT 0 CHECK(run_fence>=0),run_holder uuid,run_lease_until timestamptz,claim_binding_revision bigint,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),next_eligible_at timestamptz NOT NULL DEFAULT clock_timestamp(),first_claimed_at timestamptz,deadline_at timestamptz,terminal_code text,finished_at timestamptz,
 PRIMARY KEY(operation_id),UNIQUE(entity_id,book_id,operation_id),FOREIGN KEY(entity_id,book_id,binding_id) REFERENCES integration.sync_delivery_binding(entity_id,book_id,binding_id),FOREIGN KEY(entity_id,book_id,stream_id) REFERENCES integration.sync_stream,
 CHECK((state='running')=(run_holder IS NOT NULL AND run_lease_until IS NOT NULL AND claim_binding_revision IS NOT NULL)),CHECK((first_claimed_at IS NULL AND deadline_at IS NULL) OR deadline_at=first_claimed_at+interval '24 hours'),
 CHECK((state IN ('succeeded','quarantined'))=(finished_at IS NOT NULL AND terminal_code IS NOT NULL)),CHECK(terminal_code IS NULL OR terminal_code IN ('completed','retry_exhausted','age_exhausted','permanent_failure','outcome_unknown'))
);
CREATE UNIQUE INDEX sync_delivery_current ON integration.sync_delivery_operation(entity_id,book_id,stream_id) WHERE state<>'succeeded';
CREATE INDEX sync_delivery_due ON integration.sync_delivery_operation(binding_id,next_eligible_at,operation_id) WHERE state IN ('ready','retry_wait','running');
CREATE TABLE integration.sync_delivery_dispatch (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,operation_id uuid NOT NULL,generation bigint NOT NULL CHECK(generation>0),task_name_hex text NOT NULL CHECK(task_name_hex~'^[0-9a-f]{64}$'),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),publisher_fence bigint NOT NULL DEFAULT 0 CHECK(publisher_fence>=0),holder uuid,lease_until timestamptz,outcome text CHECK(outcome IN ('accepted','already_exists','timeout','transient','rejected')),
 PRIMARY KEY(operation_id,generation),UNIQUE(task_name_hex),FOREIGN KEY(entity_id,book_id,operation_id) REFERENCES integration.sync_delivery_operation(entity_id,book_id,operation_id),CHECK((holder IS NULL)=(lease_until IS NULL))
);
CREATE TABLE integration.sync_delivery_enqueue_receipt (
 operation_id uuid NOT NULL,generation bigint NOT NULL,publisher_fence bigint NOT NULL,holder uuid NOT NULL,outcome text NOT NULL CHECK(outcome IN ('accepted','already_exists','timeout','transient','rejected')),recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(operation_id,generation,publisher_fence),FOREIGN KEY(operation_id,generation) REFERENCES integration.sync_delivery_dispatch
);
CREATE TABLE integration.sync_delivery_run_receipt (
 operation_id uuid NOT NULL REFERENCES integration.sync_delivery_operation,run_fence bigint NOT NULL,holder uuid NOT NULL,outcome text NOT NULL CHECK(outcome IN ('completed','retryable','permanent','unknown')),retry_after_seconds integer NOT NULL CHECK(retry_after_seconds BETWEEN 0 AND 86400),result jsonb NOT NULL,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(operation_id,run_fence)
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.sync_delivery_enqueue_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.sync_delivery_run_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Fixed errors contain no provider data or database diagnostics.
CREATE FUNCTION integration.sync_delivery_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='acquisition_dispatch_rejected';END $$;
-- Publisher identity cannot inherit any consumer, application or privileged role.
CREATE FUNCTION integration.sync_publisher_login(login_value name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolcanlogin AND NOT(rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication)) AND pg_has_role(login_value,'bookkeeping_publisher','MEMBER')
 AND NOT EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolreplication OR rolname IN ('bookkeeping_owner','bookkeeping_api','bookkeeping_identity_operator','bookkeeping_migrator','bookkeeping_sync','bookkeeping_webhook','bookkeeping_observer','cloudsqlsuperuser')) AND pg_has_role(login_value,oid,'MEMBER'))
$$;
-- Owner provisioned identities are immutable; toggling a mandate is separately revisioned.
CREATE FUNCTION integration.enroll_sync_delivery(actor uuid,e uuid,b uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;w integration.connection_worker;d integration.sync_delivery_binding;r integration.connection_receipt;result_value jsonb;p uuid;pub name;worker name;enabled_value boolean;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='acquisition_dispatch_rejected';END IF;
 v:=integration.connection_json(raw,4096);PERFORM ledger.keys(v,ARRAY['publisher_login','worker_login','service_principal_id','enabled']);PERFORM integration.connection_key(k);
 p:=ledger.uuid_value(v->'service_principal_id');pub:=integration.sync_text(v->'publisher_login',63)::name;worker:=integration.sync_text(v->'worker_login',63)::name;
 IF jsonb_typeof(v->'enabled') IS DISTINCT FROM 'boolean' THEN PERFORM integration.sync_delivery_error('P2900');END IF;enabled_value:=(v->>'enabled')::boolean;
 PERFORM banking.source_operator_authorize(actor,p,e,b,enabled_value);
 SELECT * INTO w FROM integration.connection_worker WHERE entity_id=e AND book_id=b FOR SHARE;
 IF NOT FOUND OR w.service_principal<>p OR w.worker_login<>worker THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 IF enabled_value THEN
  IF NOT w.enabled OR NOT integration.sync_login(worker) OR NOT integration.sync_publisher_login(pub) THEN PERFORM integration.sync_delivery_error('P2901');END IF;
  PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=p AND permission='import' AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 END IF;
 SELECT * INTO r FROM integration.connection_receipt WHERE entity_id=e AND book_id=b AND actor_id=actor AND operation='sync_delivery_enroll' AND idempotency_key=k;
 IF FOUND THEN IF r.body<>v THEN PERFORM integration.sync_delivery_error('P2902');END IF;RETURN r.result;END IF;
 SELECT * INTO d FROM integration.sync_delivery_binding WHERE entity_id=e AND book_id=b FOR UPDATE;
 IF FOUND THEN
  IF ROW(d.service_principal,d.publisher_login,d.worker_login) IS DISTINCT FROM ROW(p,pub,worker) THEN PERFORM integration.sync_delivery_error('P2902');END IF;
  UPDATE integration.sync_delivery_binding SET enabled=enabled_value,binding_revision=binding_revision+1 WHERE binding_id=d.binding_id RETURNING * INTO d;
 ELSE INSERT INTO integration.sync_delivery_binding(entity_id,book_id,service_principal,publisher_login,worker_login,enabled) VALUES(e,b,p,pub,worker,enabled_value) RETURNING * INTO d;END IF;
 result_value:=jsonb_build_object('binding_id',d.binding_id,'binding_revision',d.binding_revision::text);
 INSERT INTO integration.connection_receipt(entity_id,book_id,actor_id,operation,idempotency_key,body,result) VALUES(e,b,actor,'sync_delivery_enroll',k,v,result_value);RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2700' OR SQLSTATE 'P2600' THEN PERFORM integration.sync_delivery_error('P2900');END $$;
-- Bind actual login and current service mandate before returning even historical delivery metadata.
CREATE FUNCTION integration.sync_delivery_authorize(id uuid,publisher boolean,sid uuid DEFAULT NULL) RETURNS integration.sync_delivery_binding LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE d integration.sync_delivery_binding;w integration.connection_worker;p uuid;i integration.sync_item;t integration.sync_stream;cap text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR id IS NULL THEN PERFORM integration.sync_delivery_error('P2900');END IF;PERFORM set_config('lock_timeout','1s',true);
 SELECT * INTO d FROM integration.sync_delivery_binding WHERE binding_id=id;
 IF NOT FOUND OR NOT d.enabled OR (publisher AND (session_user<>d.publisher_login OR NOT integration.sync_publisher_login(session_user))) OR (NOT publisher AND (session_user<>d.worker_login OR NOT integration.sync_login(session_user))) THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 IF sid IS NOT NULL THEN
  SELECT * INTO t FROM integration.sync_stream WHERE entity_id=d.entity_id AND book_id=d.book_id AND stream_id=sid;
  IF NOT FOUND OR t.worker_login<>d.worker_login OR t.service_principal<>d.service_principal THEN PERFORM integration.sync_delivery_error('P2901');END IF;
  SELECT * INTO i FROM integration.sync_item WHERE entity_id=d.entity_id AND book_id=d.book_id AND sync_item_id=t.sync_item_id;
 END IF;
 FOR p IN SELECT DISTINCT x FROM unnest(ARRAY[d.service_principal,i.consent_principal]) x WHERE x IS NOT NULL ORDER BY x LOOP PERFORM 1 FROM ledger.principal WHERE principal_id=p AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.sync_delivery_error('P2901');END IF;END LOOP;
 PERFORM 1 FROM ledger.entity WHERE entity_id=d.entity_id AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 FOREACH cap IN ARRAY ARRAY['read','import'] LOOP PERFORM 1 FROM ledger.membership WHERE entity_id=d.entity_id AND principal_id=d.service_principal AND permission=cap AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM integration.sync_delivery_error('P2901');END IF;END LOOP;
 PERFORM 1 FROM ledger.book WHERE entity_id=d.entity_id AND book_id=d.book_id AND disabled_at IS NULL FOR UPDATE;IF NOT FOUND THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 SELECT * INTO w FROM integration.connection_worker WHERE entity_id=d.entity_id AND book_id=d.book_id FOR SHARE;
 IF NOT FOUND OR NOT w.enabled OR ROW(w.service_principal,w.worker_login) IS DISTINCT FROM ROW(d.service_principal,d.worker_login) OR NOT integration.sync_login(d.worker_login) THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 IF sid IS NOT NULL THEN PERFORM integration.signal_authorize(d.entity_id,d.book_id,sid);END IF;
 SELECT * INTO d FROM integration.sync_delivery_binding WHERE binding_id=id FOR SHARE;RETURN d;
EXCEPTION WHEN SQLSTATE 'P2601' OR SQLSTATE 'P2801' THEN PERFORM integration.sync_delivery_error('P2901');END $$;
-- Resolve scoped identity before acquiring operation lock; no caller actor or provider filter exists.
CREATE FUNCTION integration.sync_delivery_lock(id uuid,publisher boolean) RETURNS integration.sync_delivery_operation LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.sync_delivery_operation;
BEGIN
 SELECT * INTO o FROM integration.sync_delivery_operation WHERE operation_id=id;IF NOT FOUND THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 PERFORM integration.sync_delivery_authorize(o.binding_id,publisher,o.stream_id);SELECT * INTO o FROM integration.sync_delivery_operation WHERE operation_id=id FOR UPDATE;RETURN o;
END $$;
-- Exact stableID parity with acquisition/evidence.go, including UUID version/variant masking.
CREATE FUNCTION integration.sync_restart_claim(e uuid,b uuid,s uuid,k uuid) RETURNS uuid LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE x text;v text:='';d bytea;h text;
BEGIN
 FOREACH x IN ARRAY ARRAY['acquisition-mutation-restart-v1',e::text,b::text,s::text,k::text] LOOP v:=v||octet_length(x)::text||':'||x;END LOOP;
 d:=substring(sha256(convert_to(v,'UTF8')),1,16);d:=set_byte(d,6,(get_byte(d,6)&15)|128);d:=set_byte(d,8,(get_byte(d,8)&63)|128);h:=encode(d,'hex');RETURN (substr(h,1,8)||'-'||substr(h,9,4)||'-'||substr(h,13,4)||'-'||substr(h,17,4)||'-'||substr(h,21,12))::uuid;
END $$;
-- Queue generations never select or reset an acquisition business claim.
CREATE FUNCTION integration.sync_delivery_new_dispatch(o integration.sync_delivery_operation) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 INSERT INTO integration.sync_delivery_dispatch(entity_id,book_id,operation_id,generation,task_name_hex) VALUES(o.entity_id,o.book_id,o.operation_id,o.dispatch_generation,encode(sha256(convert_to(o.operation_id::text||'|'||o.dispatch_generation::text||'|2','UTF8')),'hex'));
END $$;
-- Both Runner claim identities must be checked before taking over an expired delivery lease.
CREATE FUNCTION integration.sync_delivery_acquisition(o integration.sync_delivery_operation) RETURNS text LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE original integration.sync_set;restarted integration.sync_set;t integration.sync_stream;restart uuid;
BEGIN
 restart:=integration.sync_restart_claim(o.entity_id,o.book_id,o.stream_id,o.claim_id);
 SELECT * INTO original FROM integration.sync_set WHERE entity_id=o.entity_id AND book_id=o.book_id AND stream_id=o.stream_id AND claim_key=o.claim_id;
 SELECT * INTO restarted FROM integration.sync_set WHERE entity_id=o.entity_id AND book_id=o.book_id AND stream_id=o.stream_id AND claim_key=restart;
 IF EXISTS(SELECT FROM integration.sync_completion c JOIN integration.sync_set x USING(entity_id,book_id,set_id) WHERE x.entity_id=o.entity_id AND x.book_id=o.book_id AND x.stream_id=o.stream_id AND x.claim_key IN (o.claim_id,restart) AND x.state='completed' AND (c.result->>'completed_generation')::bigint>=o.target_generation) THEN RETURN 'completed';END IF;
 SELECT * INTO t FROM integration.sync_stream WHERE entity_id=o.entity_id AND book_id=o.book_id AND stream_id=o.stream_id;
 IF t.current_set IS NOT NULL AND t.lease_until>clock_timestamp() THEN RETURN 'live';END IF;
 IF original.state='staging' OR restarted.state='staging' THEN RETURN 'expired';END IF;
 IF restarted.state='abandoned' THEN RETURN 'abandoned';END IF;
 IF original.state='abandoned' AND original.reason='pagination_mutation' AND restarted.set_id IS NULL THEN RETURN 'restart';END IF;
 IF original.state='abandoned' OR restarted.state='quarantined' OR original.state='quarantined' THEN RETURN 'abandoned';END IF;
 RETURN 'absent';
END $$;
-- Fence only this operation's original/derived acquisition set, never another stream owner's work.
CREATE FUNCTION integration.sync_delivery_stop_sets(o integration.sync_delivery_operation,reason_value text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE ids uuid[];
BEGIN
 SELECT array_agg(set_id) INTO ids FROM integration.sync_set WHERE entity_id=o.entity_id AND book_id=o.book_id AND stream_id=o.stream_id AND claim_key IN (o.claim_id,integration.sync_restart_claim(o.entity_id,o.book_id,o.stream_id,o.claim_id)) AND state='staging';
 UPDATE integration.sync_set SET state='abandoned',reason=reason_value WHERE entity_id=o.entity_id AND book_id=o.book_id AND set_id=ANY(ids);
 UPDATE integration.sync_stream SET current_set=NULL,lease_until=NULL,fence=fence+1 WHERE entity_id=o.entity_id AND book_id=o.book_id AND stream_id=o.stream_id AND current_set=ANY(ids);
END $$;
-- Terminal holds keep the unique stream slot occupied until a future reviewed operator recovery.
CREATE FUNCTION integration.sync_delivery_terminal(o integration.sync_delivery_operation,code text) RETURNS integration.sync_delivery_operation LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF code<>'completed' THEN PERFORM integration.sync_delivery_stop_sets(o,'dispatch_quarantined');END IF;
 UPDATE integration.sync_delivery_operation SET state=CASE WHEN code='completed' THEN 'succeeded' ELSE 'quarantined' END,terminal_code=code,finished_at=clock_timestamp(),run_holder=NULL,run_lease_until=NULL,claim_binding_revision=NULL WHERE operation_id=o.operation_id RETURNING * INTO o;RETURN o;
END $$;
-- A new business claim requires known failed/expired SQL state, not an uncertain queue or response.
CREATE FUNCTION integration.sync_delivery_retry(o integration.sync_delivery_operation,delay_seconds integer,new_claim boolean) RETURNS integration.sync_delivery_operation LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE next_time timestamptz:=clock_timestamp()+make_interval(secs=>greatest(delay_seconds,5));
BEGIN
 IF new_claim AND o.failed_count>=7 THEN UPDATE integration.sync_delivery_operation SET failed_count=8 WHERE operation_id=o.operation_id RETURNING * INTO o;RETURN integration.sync_delivery_terminal(o,'retry_exhausted');END IF;
 IF o.deadline_at IS NOT NULL AND next_time>=o.deadline_at THEN RETURN integration.sync_delivery_terminal(o,'age_exhausted');END IF;
 UPDATE integration.sync_delivery_operation SET state='retry_wait',run_holder=NULL,run_lease_until=NULL,claim_binding_revision=NULL,next_eligible_at=next_time,failed_count=failed_count+CASE WHEN new_claim THEN 1 ELSE 0 END,claim_round=claim_round+CASE WHEN new_claim THEN 1 ELSE 0 END,claim_id=CASE WHEN new_claim THEN gen_random_uuid() ELSE claim_id END,dispatch_generation=dispatch_generation+1 WHERE operation_id=o.operation_id RETURNING * INTO o;PERFORM integration.sync_delivery_new_dispatch(o);RETURN o;
END $$;
-- Scan authoritative demand, including initial generation without an outbox row.
CREATE FUNCTION integration.sync_delivery_discover(id uuid,maximum integer) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE d integration.sync_delivery_binding;t integration.sync_stream;o integration.sync_delivery_operation;n integer:=0;
BEGIN
 d:=integration.sync_delivery_authorize(id,true);IF maximum IS NULL OR maximum NOT BETWEEN 1 AND 25 THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 FOR t IN SELECT x.* FROM integration.sync_stream x WHERE x.entity_id=d.entity_id AND x.book_id=d.book_id AND x.worker_login=d.worker_login AND x.service_principal=d.service_principal AND x.requested_generation>x.completed_generation AND integration.signal_eligible(x) AND NOT EXISTS(SELECT FROM integration.sync_delivery_operation a WHERE a.entity_id=x.entity_id AND a.book_id=x.book_id AND a.stream_id=x.stream_id AND a.state<>'succeeded') ORDER BY x.stream_id LIMIT maximum LOOP
  PERFORM integration.sync_delivery_authorize(id,true,t.stream_id);
  INSERT INTO integration.sync_delivery_operation(entity_id,book_id,binding_id,stream_id,target_generation) VALUES(t.entity_id,t.book_id,id,t.stream_id,t.requested_generation) RETURNING * INTO o;
  PERFORM integration.sync_delivery_new_dispatch(o);n:=n+1;
 END LOOP;RETURN jsonb_build_object('created_count',n);
END $$;
-- Bounded IDs are hints only; every operation checks current authority again.
CREATE FUNCTION integration.sync_delivery_candidates(id uuid,maximum integer) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE d integration.sync_delivery_binding;ids uuid[];
BEGIN
 d:=integration.sync_delivery_authorize(id,true);IF maximum IS NULL OR maximum NOT BETWEEN 1 AND 25 THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 SELECT array_agg(q.operation_id ORDER BY q.operation_id) INTO ids FROM (SELECT o.operation_id FROM integration.sync_delivery_operation o JOIN integration.sync_stream t USING(entity_id,book_id,stream_id) WHERE o.binding_id=id AND o.state IN ('ready','retry_wait','running') AND integration.signal_eligible(t) AND (o.deadline_at<=clock_timestamp() OR (o.state='running' AND o.run_lease_until<=clock_timestamp()) OR (o.state IN ('ready','retry_wait') AND o.next_eligible_at<=clock_timestamp())) AND (o.state='running' OR o.deadline_at<=clock_timestamp() OR EXISTS(SELECT FROM integration.sync_delivery_dispatch due WHERE due.operation_id=o.operation_id AND due.generation=o.dispatch_generation AND (due.lease_until IS NULL OR due.lease_until<=clock_timestamp()) AND (coalesce(due.outcome,'') NOT IN ('accepted','already_exists') OR due.created_at<=clock_timestamp()-interval '15 minutes'))) AND NOT EXISTS(SELECT FROM integration.sync_set live_set WHERE live_set.set_id=t.current_set AND live_set.state='staging' AND t.lease_until>clock_timestamp()) ORDER BY o.next_eligible_at,o.operation_id LIMIT maximum) q;
 RETURN jsonb_build_object('operation_ids',coalesce(ids,ARRAY[]::uuid[]));
END $$;
-- Queue loss replaces task identity, never a still-live acquisition claim or lost Retry-After.
CREATE FUNCTION integration.sync_delivery_recover(id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.sync_delivery_operation;d integration.sync_delivery_dispatch;proof text;status_value text:='no_change';
BEGIN
 o:=integration.sync_delivery_lock(id,true);
 IF o.state='succeeded' THEN RETURN jsonb_build_object('status','done');END IF;IF o.state='quarantined' THEN RETURN jsonb_build_object('status','quarantined');END IF;
 proof:=integration.sync_delivery_acquisition(o);
 IF proof='completed' THEN o:=integration.sync_delivery_terminal(o,'completed');RETURN jsonb_build_object('status','done');END IF;
 IF o.deadline_at IS NOT NULL AND clock_timestamp()>=o.deadline_at THEN o:=integration.sync_delivery_terminal(o,'age_exhausted');RETURN jsonb_build_object('status','quarantined');END IF;
 IF o.state='running' AND o.run_lease_until>clock_timestamp() OR proof='live' THEN RETURN jsonb_build_object('status','no_change');END IF;
 IF proof='expired' THEN
  PERFORM integration.sync_delivery_stop_sets(o,'dispatch_lease_expired');o:=integration.sync_delivery_retry(o,5,true);RETURN jsonb_build_object('status',CASE WHEN o.state='quarantined' THEN 'quarantined' ELSE 'retry_wait' END);
 ELSIF proof='abandoned' THEN
  o:=integration.sync_delivery_terminal(o,'outcome_unknown');RETURN jsonb_build_object('status','quarantined');
 ELSIF o.state='running' THEN o:=integration.sync_delivery_retry(o,5,false);RETURN jsonb_build_object('status',CASE WHEN o.state='quarantined' THEN 'quarantined' ELSE 'retry_wait' END);END IF;
 IF o.next_eligible_at>clock_timestamp() THEN RETURN jsonb_build_object('status','no_change');END IF;
 SELECT * INTO d FROM integration.sync_delivery_dispatch WHERE operation_id=id AND generation=o.dispatch_generation FOR UPDATE;
 IF d.lease_until>clock_timestamp() THEN RETURN jsonb_build_object('status','no_change');END IF;
 IF d.outcome='rejected' OR d.created_at<=clock_timestamp()-interval '15 minutes' THEN
  UPDATE integration.sync_delivery_operation SET dispatch_generation=dispatch_generation+1 WHERE operation_id=id RETURNING * INTO o;PERFORM integration.sync_delivery_new_dispatch(o);status_value:='redispatched';
 END IF;RETURN jsonb_build_object('status',status_value);
END $$;
-- The committed enqueue lease owns one deterministic task create attempt.
CREATE FUNCTION integration.sync_delivery_dispatch_claim(id uuid,holder_value uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.sync_delivery_operation;d integration.sync_delivery_dispatch;s text;
BEGIN
 o:=integration.sync_delivery_lock(id,true);IF holder_value IS NULL OR holder_value='00000000-0000-0000-0000-000000000000' THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 SELECT * INTO d FROM integration.sync_delivery_dispatch WHERE operation_id=id AND generation=o.dispatch_generation FOR UPDATE;
 s:=CASE WHEN o.state='succeeded' THEN 'done' WHEN o.state='quarantined' THEN 'quarantined' ELSE 'not_due' END;
 IF o.state IN ('ready','retry_wait') AND o.next_eligible_at<=clock_timestamp() AND (o.deadline_at IS NULL OR o.deadline_at>clock_timestamp()) AND coalesce(d.outcome,'') NOT IN ('accepted','already_exists','rejected') THEN
  IF d.holder=holder_value AND d.lease_until>clock_timestamp() THEN s:='claimed';
  ELSIF d.lease_until IS NULL OR d.lease_until<=clock_timestamp() THEN UPDATE integration.sync_delivery_dispatch SET holder=holder_value,lease_until=clock_timestamp()+interval '30 seconds',publisher_fence=publisher_fence+1 WHERE operation_id=id AND generation=o.dispatch_generation RETURNING * INTO d;s:='claimed';END IF;
 END IF;
 RETURN jsonb_build_object('status',s,'operation_id',id,'entity_id',o.entity_id,'book_id',o.book_id,'dispatch_generation',o.dispatch_generation::text,'target_version',2,'task_name_hex',d.task_name_hex,'dispatch_fence',CASE WHEN s='claimed' THEN d.publisher_fence::text ELSE NULL END,'lease_until',CASE WHEN s='claimed' THEN d.lease_until ELSE NULL END);
END $$;
-- Enqueue success/ALREADY_EXISTS is transport evidence only; late reports cannot complete work.
CREATE FUNCTION integration.sync_delivery_dispatch_result(id uuid,g bigint,holder_value uuid,f bigint,result_value text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.sync_delivery_operation;d integration.sync_delivery_dispatch;r integration.sync_delivery_enqueue_receipt;
BEGIN
 o:=integration.sync_delivery_lock(id,true);
 IF g IS NULL OR f IS NULL OR holder_value IS NULL OR result_value IS NULL OR result_value NOT IN ('accepted','already_exists','timeout','transient','rejected') THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 SELECT * INTO r FROM integration.sync_delivery_enqueue_receipt WHERE operation_id=id AND generation=g AND publisher_fence=f;
 IF FOUND THEN IF r.holder<>holder_value OR r.outcome<>result_value THEN PERFORM integration.sync_delivery_error('P2902');END IF;RETURN jsonb_build_object('recorded',true);END IF;
 SELECT * INTO d FROM integration.sync_delivery_dispatch WHERE operation_id=id AND generation=g FOR UPDATE;
 IF NOT FOUND OR d.holder IS DISTINCT FROM holder_value OR d.publisher_fence<>f OR d.lease_until<=clock_timestamp() OR g<>o.dispatch_generation THEN RETURN jsonb_build_object('recorded',false);END IF;
 INSERT INTO integration.sync_delivery_enqueue_receipt(operation_id,generation,publisher_fence,holder,outcome) VALUES(id,g,f,holder_value,result_value);
 UPDATE integration.sync_delivery_dispatch SET holder=NULL,lease_until=NULL,outcome=result_value WHERE operation_id=id AND generation=g;RETURN jsonb_build_object('recorded',true);
END $$;
-- Current source authorization plus both inner claims precede any consumer takeover.
CREATE FUNCTION integration.sync_delivery_claim(id uuid,g bigint,holder_value uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.sync_delivery_operation;d integration.sync_delivery_binding;proof text;s text:='not_due';claimed_at timestamptz:=clock_timestamp();
BEGIN
 o:=integration.sync_delivery_lock(id,false);SELECT * INTO d FROM integration.sync_delivery_binding WHERE binding_id=o.binding_id;
 IF g IS NULL OR g<=0 OR holder_value IS NULL OR holder_value='00000000-0000-0000-0000-000000000000' THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 IF o.state='succeeded' THEN s:='done';ELSIF o.state='quarantined' THEN s:='quarantined';ELSE
  proof:=integration.sync_delivery_acquisition(o);
  IF proof='completed' THEN o:=integration.sync_delivery_terminal(o,'completed');s:='done';
  ELSIF o.deadline_at IS NOT NULL AND clock_timestamp()>=o.deadline_at THEN o:=integration.sync_delivery_terminal(o,'age_exhausted');s:='quarantined';
  ELSIF g<>o.dispatch_generation OR o.next_eligible_at>clock_timestamp() THEN s:='not_due';
  ELSIF o.state='running' AND o.run_lease_until>clock_timestamp() THEN s:=CASE WHEN o.run_holder=holder_value AND o.claim_binding_revision=d.binding_revision THEN 'claimed' ELSE 'busy' END;
  ELSIF proof='live' THEN s:='busy';
  ELSIF proof='expired' THEN PERFORM integration.sync_delivery_stop_sets(o,'dispatch_lease_expired');o:=integration.sync_delivery_retry(o,5,true);s:=CASE WHEN o.state='quarantined' THEN 'quarantined' ELSE 'not_due' END;
  ELSIF proof='abandoned' THEN o:=integration.sync_delivery_terminal(o,'outcome_unknown');s:='quarantined';
  ELSE
   UPDATE integration.sync_delivery_operation SET state='running',run_fence=run_fence+1,run_holder=holder_value,run_lease_until=least(clock_timestamp()+interval '120 seconds',coalesce(deadline_at,clock_timestamp()+interval '24 hours')),claim_binding_revision=d.binding_revision,first_claimed_at=coalesce(first_claimed_at,claimed_at),deadline_at=coalesce(deadline_at,claimed_at+interval '24 hours') WHERE operation_id=id RETURNING * INTO o;s:='claimed';
  END IF;
 END IF;
 RETURN jsonb_build_object('status',s,'operation_id',id,'entity_id',o.entity_id,'book_id',o.book_id,'stream_id',o.stream_id,'claim_id',CASE WHEN s='claimed' THEN o.claim_id ELSE NULL END,'fence',CASE WHEN s='claimed' THEN o.run_fence::text ELSE NULL END,'lease_until',CASE WHEN s='claimed' THEN o.run_lease_until ELSE NULL END,'deadline_at',o.deadline_at,'target_version',2);
END $$;
-- Runner outcome is a hint; only committed SQL publication proves successful acquisition.
CREATE FUNCTION integration.sync_delivery_settle(id uuid,holder_value uuid,f bigint,outcome_value text,delay_seconds integer) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.sync_delivery_operation;d integration.sync_delivery_binding;r integration.sync_delivery_run_receipt;proof text;result_value jsonb;s text;
BEGIN
 o:=integration.sync_delivery_lock(id,false);SELECT * INTO d FROM integration.sync_delivery_binding WHERE binding_id=o.binding_id;
 IF holder_value IS NULL OR f IS NULL OR outcome_value IS NULL OR outcome_value NOT IN ('completed','retryable','permanent','unknown') OR delay_seconds IS NULL OR delay_seconds NOT BETWEEN 0 AND 86400 THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 SELECT * INTO r FROM integration.sync_delivery_run_receipt WHERE operation_id=id AND run_fence=f;
 IF FOUND THEN IF r.holder<>holder_value OR r.outcome<>outcome_value OR r.retry_after_seconds<>delay_seconds THEN PERFORM integration.sync_delivery_error('P2902');END IF;RETURN r.result;END IF;
 IF o.state<>'running' OR o.run_holder IS DISTINCT FROM holder_value OR o.run_fence<>f OR o.claim_binding_revision<>d.binding_revision OR o.run_lease_until<=clock_timestamp() THEN RETURN jsonb_build_object('status','stale');END IF;
 proof:=integration.sync_delivery_acquisition(o);
 IF proof='completed' THEN o:=integration.sync_delivery_terminal(o,'completed');s:='done';
 ELSIF o.deadline_at<=clock_timestamp() THEN o:=integration.sync_delivery_terminal(o,'age_exhausted');s:='quarantined';
 ELSIF outcome_value='permanent' THEN o:=integration.sync_delivery_terminal(o,'permanent_failure');s:='quarantined';
 ELSIF proof='abandoned' AND outcome_value='retryable' THEN o:=integration.sync_delivery_retry(o,delay_seconds,true);s:=CASE WHEN o.state='quarantined' THEN 'quarantined' ELSE 'retry_wait' END;
 ELSIF proof='abandoned' THEN o:=integration.sync_delivery_terminal(o,'outcome_unknown');s:='quarantined';
 ELSIF proof='expired' THEN PERFORM integration.sync_delivery_stop_sets(o,'dispatch_lease_expired');o:=integration.sync_delivery_retry(o,delay_seconds,true);s:=CASE WHEN o.state='quarantined' THEN 'quarantined' ELSE 'retry_wait' END;
 ELSE o:=integration.sync_delivery_retry(o,delay_seconds,false);s:=CASE WHEN o.state='quarantined' THEN 'quarantined' ELSE 'retry_wait' END;END IF;
 result_value:=jsonb_build_object('status',s);INSERT INTO integration.sync_delivery_run_receipt(operation_id,run_fence,holder,outcome,retry_after_seconds,result) VALUES(id,f,holder_value,outcome_value,delay_seconds,result_value);RETURN result_value;
END $$;
-- Runtime functions alone own transitions; ordinary owner DML cannot rewrite identity or terminal facts.
CREATE FUNCTION integration.sync_delivery_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM integration.sync_delivery_error('P2902');END IF;
 IF TG_TABLE_NAME='sync_delivery_binding' THEN
  IF (to_jsonb(NEW)-ARRAY['enabled','binding_revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['enabled','binding_revision']) OR NEW.binding_revision::numeric<>OLD.binding_revision::numeric+1 THEN PERFORM integration.sync_delivery_error('P2902');END IF;
 ELSIF TG_TABLE_NAME='sync_delivery_operation' THEN
  IF ROW(NEW.entity_id,NEW.book_id,NEW.operation_id,NEW.binding_id,NEW.stream_id,NEW.target_generation,NEW.created_at) IS DISTINCT FROM ROW(OLD.entity_id,OLD.book_id,OLD.operation_id,OLD.binding_id,OLD.stream_id,OLD.target_generation,OLD.created_at) OR OLD.state IN ('succeeded','quarantined') OR NEW.failed_count<OLD.failed_count OR NEW.claim_round<OLD.claim_round OR NEW.run_fence<OLD.run_fence OR NEW.dispatch_generation<OLD.dispatch_generation OR OLD.first_claimed_at IS NOT NULL AND ROW(NEW.first_claimed_at,NEW.deadline_at) IS DISTINCT FROM ROW(OLD.first_claimed_at,OLD.deadline_at) OR NEW.claim_id<>OLD.claim_id AND NOT (NEW.claim_round=OLD.claim_round+1 AND NEW.failed_count=OLD.failed_count+1) THEN PERFORM integration.sync_delivery_error('P2902');END IF;
 ELSE
  IF ROW(NEW.entity_id,NEW.book_id,NEW.operation_id,NEW.generation,NEW.task_name_hex,NEW.created_at) IS DISTINCT FROM ROW(OLD.entity_id,OLD.book_id,OLD.operation_id,OLD.generation,OLD.task_name_hex,OLD.created_at) OR NEW.publisher_fence<OLD.publisher_fence THEN PERFORM integration.sync_delivery_error('P2902');END IF;
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_delivery BEFORE UPDATE OR DELETE ON integration.sync_delivery_binding FOR EACH ROW EXECUTE FUNCTION integration.sync_delivery_guard();
CREATE TRIGGER preserve_delivery BEFORE UPDATE OR DELETE ON integration.sync_delivery_operation FOR EACH ROW EXECUTE FUNCTION integration.sync_delivery_guard();
CREATE TRIGGER preserve_delivery BEFORE UPDATE OR DELETE ON integration.sync_delivery_dispatch FOR EACH ROW EXECUTE FUNCTION integration.sync_delivery_guard();
REVOKE ALL ON FUNCTION integration.sync_delivery_error(text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_publisher_login(name) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.enroll_sync_delivery(uuid,uuid,uuid,text,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_authorize(uuid,boolean,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_lock(uuid,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_restart_claim(uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_new_dispatch(integration.sync_delivery_operation) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_acquisition(integration.sync_delivery_operation) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_stop_sets(integration.sync_delivery_operation,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_terminal(integration.sync_delivery_operation,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_retry(integration.sync_delivery_operation,integer,boolean) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_discover(uuid,integer) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_candidates(uuid,integer) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_recover(uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_dispatch_claim(uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_dispatch_result(uuid,bigint,uuid,bigint,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_claim(uuid,bigint,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_settle(uuid,uuid,bigint,text,integer) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_guard() FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.enroll_sync_delivery(uuid,uuid,uuid,text,text) TO bookkeeping_owner;
GRANT EXECUTE ON FUNCTION integration.sync_delivery_discover(uuid,integer),integration.sync_delivery_candidates(uuid,integer),integration.sync_delivery_recover(uuid),integration.sync_delivery_dispatch_claim(uuid,uuid),integration.sync_delivery_dispatch_result(uuid,bigint,uuid,bigint,text) TO bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.sync_delivery_claim(uuid,bigint,uuid),integration.sync_delivery_settle(uuid,uuid,bigint,text,integer) TO bookkeeping_sync;
COMMIT;
