-- Owner-invoked metadata provisioning; no credentials, role grants or cloud calls.
\set ON_ERROR_STOP on
\set VERBOSITY terse
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='10s';
SET LOCAL lock_timeout='2s';
SELECT set_config('jayburd.bootstrap.project', :'project', true),
 set_config('jayburd.bootstrap.subject', :'subject', true),
 set_config('jayburd.bootstrap.currency', :'currency_id', true),
 set_config('jayburd.bootstrap.source', :'source_release', true),
 set_config('jayburd.bootstrap.from', :'valid_from', true),
 set_config('jayburd.bootstrap.to', :'valid_to', true) \gset
-- Fail before role changes; operator authority must be separately reviewed.
DO $$ BEGIN
 IF current_database()<>'bookkeeping' OR current_setting('jayburd.bootstrap.project')<>'jayburd-bookkeeping-dev'
 OR session_user LIKE 'jayburd-dev-%' OR NOT pg_has_role(session_user,'bookkeeping_owner','SET')
 OR NOT pg_has_role(session_user,'bookkeeping_identity_operator','SET') THEN RAISE EXCEPTION 'sandbox_bootstrap_denied'; END IF;
 IF EXISTS(SELECT FROM pg_roles WHERE rolname IN ('bookkeeping_owner','bookkeeping_identity_operator') AND (rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls OR rolreplication)) THEN RAISE EXCEPTION 'sandbox_bootstrap_denied'; END IF;
 IF current_setting('jayburd.bootstrap.source')<>'SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff'
 OR current_setting('jayburd.bootstrap.from')<>'2026-09-17' OR current_setting('jayburd.bootstrap.to')<>'2027-01-01'
 OR current_setting('jayburd.bootstrap.currency')::uuid='00000000-0000-0000-0000-000000000000' THEN RAISE EXCEPTION 'sandbox_reference_rejected'; END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
-- Serialize the exact local reference and reject changed replay instead of overwriting history.
DO $$ DECLARE c reference.currency; id uuid:=current_setting('jayburd.bootstrap.currency')::uuid; BEGIN
 PERFORM pg_advisory_xact_lock(77421,hashtext('USD'));
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=id;
 IF FOUND THEN
  IF ROW(c.alphabetic_code,c.numeric_code,c.minor_unit,c.valid_from,c.valid_to,c.source_release) IS DISTINCT FROM
   ROW('USD'::text,'840'::text,2::smallint,'2026-09-17'::date,'2027-01-01'::date,current_setting('jayburd.bootstrap.source')) THEN RAISE EXCEPTION 'sandbox_reference_conflict'; END IF;
 ELSE INSERT INTO reference.currency(currency_version_id,alphabetic_code,numeric_code,minor_unit,valid_from,valid_to,source_release) VALUES(id,'USD','840',2,'2026-09-17','2027-01-01',current_setting('jayburd.bootstrap.source')); END IF;
END $$;
SET LOCAL ROLE bookkeeping_identity_operator;
SELECT ledger.enroll_principal('https://securetoken.google.com/jayburd-bookkeeping-dev',current_setting('jayburd.bootstrap.subject')) AS principal_id \gset
COMMIT;
SELECT 'PASS' AS status, :'principal_id' AS principal_id, :'currency_id' AS currency_version_id;
