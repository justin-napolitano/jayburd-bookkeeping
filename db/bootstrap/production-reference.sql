-- Owner-operated exact reference metadata only; no identity, book, role or provider effects.
\set ON_ERROR_STOP on
\set VERBOSITY terse
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='10s';
SET LOCAL lock_timeout='2s';
SELECT set_config('jayburd.bootstrap.project', :'project', true),
 set_config('jayburd.bootstrap.number', :'project_number', true),
 set_config('jayburd.bootstrap.currency', :'currency_id', true),
 set_config('jayburd.bootstrap.source', :'source_release', true),
 set_config('jayburd.bootstrap.from', :'valid_from', true),
 set_config('jayburd.bootstrap.to', :'valid_to', true) \gset
DO $$ BEGIN
 IF current_database()<>'bookkeeping_prod'
 OR current_setting('jayburd.bootstrap.project')<>'jayburd-bookkeeping-dev'
 OR current_setting('jayburd.bootstrap.number')<>'9088794002'
 OR session_user LIKE 'jayburd-%' OR NOT pg_has_role(session_user,'bookkeeping_owner','SET')
 OR EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_owner' AND (rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls OR rolreplication))
 THEN RAISE EXCEPTION 'production_bootstrap_denied'; END IF;
 IF current_setting('jayburd.bootstrap.source')<>'SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff'
 OR current_setting('jayburd.bootstrap.from')<>'2026-09-17'
 OR current_setting('jayburd.bootstrap.to')<>'2027-01-01'
 OR current_setting('jayburd.bootstrap.currency') !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
 OR current_setting('jayburd.bootstrap.currency')='00000000-0000-0000-0000-000000000000'
 THEN RAISE EXCEPTION 'production_reference_rejected'; END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
-- Exact replay preserves all fields; the existing currency constraint owns overlap rejection.
DO $$ DECLARE c reference.currency; id uuid:=current_setting('jayburd.bootstrap.currency')::uuid;
 history_id constant uuid:='3a184659-cf9b-4b80-b140-3f7a7796cbbd';
 history_source constant text:='ISO4217-MA115:2002-05-13+SIX-list-one:2026-09-17;local-coverage-v1'; BEGIN
 PERFORM pg_advisory_xact_lock(77421,hashtext('USD'));
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=id;
 IF FOUND THEN
  IF ROW(c.alphabetic_code,c.numeric_code,c.minor_unit,c.valid_from,c.valid_to,c.source_release) IS DISTINCT FROM
   ROW('USD'::text,'840'::text,2::smallint,'2026-09-17'::date,'2027-01-01'::date,current_setting('jayburd.bootstrap.source'))
  THEN RAISE EXCEPTION 'production_reference_conflict'; END IF;
 ELSE
  INSERT INTO reference.currency(currency_version_id,alphabetic_code,numeric_code,minor_unit,valid_from,valid_to,source_release)
  VALUES(id,'USD','840',2,'2026-09-17','2027-01-01',current_setting('jayburd.bootstrap.source'));
 END IF;
 -- Acquisition resolves original dates; install the separately reviewed local history without rewriting current provenance.
 SELECT * INTO c FROM reference.currency WHERE currency_version_id=history_id;
 IF FOUND THEN
  IF ROW(c.alphabetic_code,c.numeric_code,c.minor_unit,c.valid_from,c.valid_to,c.source_release) IS DISTINCT FROM
   ROW('USD'::text,'840'::text,2::smallint,DATE '2001-01-01',DATE '2026-09-17',history_source)
  THEN RAISE EXCEPTION 'production_history_conflict'; END IF;
 ELSE
  INSERT INTO reference.currency(currency_version_id,alphabetic_code,numeric_code,minor_unit,valid_from,valid_to,source_release)
  VALUES(history_id,'USD','840',2,'2001-01-01','2026-09-17',history_source);
 END IF;
END $$;
COMMIT;
SELECT 'PASS: exact Production reference metadata installed' AS status;
