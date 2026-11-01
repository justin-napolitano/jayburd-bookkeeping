-- Owner-invoked first-book permissions; never schedules work or accepts provider payloads.
\set ON_ERROR_STOP on
\set VERBOSITY terse
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='10s';
SET LOCAL lock_timeout='2s';
SELECT set_config('jayburd.bootstrap.project', :'project', true),
 set_config('jayburd.bootstrap.number', :'project_number', true),
 set_config('jayburd.bootstrap.actor', :'actor_id', true),
 set_config('jayburd.bootstrap.entity', :'entity_id', true),
 set_config('jayburd.bootstrap.book', :'book_id', true),
 set_config('jayburd.bootstrap.worker', :'worker_login', true),
 set_config('jayburd.bootstrap.publisher', :'publisher_login', true),
 set_config('jayburd.bootstrap.paused', :'scheduler_paused', true) \gset
DO $$ BEGIN
 IF current_database()<>'bookkeeping_prod' OR current_setting('jayburd.bootstrap.project')<>'jayburd-bookkeeping-dev'
 OR current_setting('jayburd.bootstrap.number')<>'9088794002'
 OR session_user LIKE 'jayburd-%' OR NOT pg_has_role(session_user,'bookkeeping_owner','SET')
 OR current_setting('jayburd.bootstrap.paused')<>'true'
 OR current_setting('jayburd.bootstrap.worker')<>'jayburd-prod-sync@jayburd-bookkeeping-dev.iam'
 OR current_setting('jayburd.bootstrap.publisher')<>'jayburd-prod-publisher@jayburd-bookkeeping-dev.iam'
 THEN RAISE EXCEPTION 'production_bootstrap_denied'; END IF;
 IF EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_owner' AND (rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls OR rolreplication)) THEN RAISE EXCEPTION 'production_bootstrap_denied'; END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
-- Creator proof is stricter than ordinary book read access. Revoked grants are never restored.
DO $$ DECLARE a uuid:=current_setting('jayburd.bootstrap.actor')::uuid;
 e uuid:=current_setting('jayburd.bootstrap.entity')::uuid;b uuid:=current_setting('jayburd.bootstrap.book')::uuid;
 s uuid; r record; result_value jsonb;
BEGIN
 PERFORM 1 FROM ledger.principal WHERE principal_id=a AND disabled_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'production_actor_denied'; END IF;
 IF NOT EXISTS(SELECT FROM ledger.setup_receipt WHERE principal_id=a AND entity_id=e AND book_id=b) THEN RAISE EXCEPTION 'production_creator_denied'; END IF;
 PERFORM ledger.authorize(a,e,b,'read',true);PERFORM ledger.authorize(a,e,b,'post',true);
 IF NOT integration.provider_api_login('jayburd-prod-api@jayburd-bookkeeping-dev.iam'::name)
 OR has_database_privilege('jayburd-prod-api@jayburd-bookkeeping-dev.iam','bookkeeping','CONNECT')
 OR has_database_privilege(current_setting('jayburd.bootstrap.worker'),'bookkeeping','CONNECT')
 OR has_database_privilege(current_setting('jayburd.bootstrap.publisher'),'bookkeeping','CONNECT')
 OR NOT integration.sync_login(current_setting('jayburd.bootstrap.worker')::name)
 OR NOT integration.sync_publisher_login(current_setting('jayburd.bootstrap.publisher')::name) THEN RAISE EXCEPTION 'production_worker_denied'; END IF;
 IF EXISTS(SELECT FROM integration.provider_scope WHERE entity_id=e AND book_id=b AND NOT enabled) THEN RAISE EXCEPTION 'production_scope_disabled'; END IF;
 INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES(gen_random_uuid(),'urn:jayburd:service:jayburd-bookkeeping-dev:production','acquisition:'||b) ON CONFLICT(issuer,subject) DO NOTHING;
 SELECT principal_id INTO s FROM ledger.principal WHERE issuer='urn:jayburd:service:jayburd-bookkeeping-dev:production' AND subject='acquisition:'||b AND disabled_at IS NULL FOR UPDATE;
 IF NOT FOUND OR s=a THEN RAISE EXCEPTION 'production_service_denied'; END IF;
 FOR r IN SELECT * FROM (VALUES(a,'import'),(a,'connect'),(s,'read'),(s,'import')) AS grants(principal_id,permission) LOOP
  IF EXISTS(SELECT FROM ledger.membership m WHERE m.entity_id=e AND m.principal_id=r.principal_id AND m.permission=r.permission AND m.revoked_at IS NOT NULL) THEN RAISE EXCEPTION 'production_revoked_grant'; END IF;
 END LOOP;
 IF EXISTS(SELECT FROM integration.connection_worker WHERE entity_id=e AND book_id=b AND (NOT enabled OR service_principal<>s OR worker_login<>current_setting('jayburd.bootstrap.worker')::name))
 OR EXISTS(SELECT FROM integration.sync_delivery_binding WHERE entity_id=e AND book_id=b AND (NOT enabled OR service_principal<>s OR worker_login<>current_setting('jayburd.bootstrap.worker')::name OR publisher_login<>current_setting('jayburd.bootstrap.publisher')::name)) THEN RAISE EXCEPTION 'production_binding_conflict'; END IF;
 INSERT INTO ledger.membership(entity_id,principal_id,permission,revoked_at) VALUES(e,a,'import',NULL),(e,s,'read',NULL),(e,s,'import',NULL) ON CONFLICT(entity_id,principal_id,permission) DO NOTHING;
 -- Configure the immutable Production profile before connection/delivery enrollment; existing SQL owns empty-book and replay checks.
 PERFORM integration.configure_provider_scope(a,e,b,'production-bootstrap-scope-v1',jsonb_build_object(
  'environment','production','api_login','jayburd-prod-api@jayburd-bookkeeping-dev.iam',
  'worker_login',current_setting('jayburd.bootstrap.worker'),
  'bank_key','projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/bank-credentials',
  'link_key','projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/hosted-link',
  'enabled',true)::text);
 PERFORM integration.set_connect_permission(a,e,b,a,'production-bootstrap-connect-v1',true);
 PERFORM integration.set_connection_worker(a,e,b,'production-bootstrap-worker-v1',jsonb_build_object('service_principal_id',s,'worker_login',current_setting('jayburd.bootstrap.worker'),'enabled',true)::text);
 result_value:=integration.enroll_sync_delivery(a,e,b,'production-bootstrap-delivery-v1',jsonb_build_object('service_principal_id',s,'worker_login',current_setting('jayburd.bootstrap.worker'),'publisher_login',current_setting('jayburd.bootstrap.publisher'),'enabled',true)::text);
 PERFORM set_config('jayburd.bootstrap.result',jsonb_build_object('status','PASS','service_principal_id',s,'binding_id',result_value->>'binding_id','binding_revision',result_value->>'binding_revision')::text,true);
END $$;
SELECT current_setting('jayburd.bootstrap.result') AS receipt \gset
COMMIT;
SELECT :'receipt'::jsonb AS receipt;
