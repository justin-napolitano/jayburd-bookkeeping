-- Owner-invoked finite activation only; no enrollment, role grants or adoption.
\set ON_ERROR_STOP on
\set VERBOSITY terse
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='10s';
SET LOCAL lock_timeout='2s';
SELECT set_config('jayburd.activation.project', :'project', true),
 set_config('jayburd.activation.subject', :'subject', true),
 set_config('jayburd.activation.principal', :'principal_id', true),
 set_config('jayburd.activation.key', :'idempotency_key', true),
 set_config('jayburd.activation.body', :'body', true),
 set_config('jayburd.activation.expiry', :'expires_at', true) \gset
-- Input conversion is caught so operator diagnostics cannot expose the subject.
DO $$ DECLARE expiry timestamptz; BEGIN
 BEGIN
  expiry:=current_setting('jayburd.activation.expiry')::timestamptz;
  IF NOT isfinite(expiry) OR expiry<=clock_timestamp() OR expiry>clock_timestamp()+interval '1 hour'
   OR current_database()<>'bookkeeping' OR current_setting('jayburd.activation.project')<>'jayburd-bookkeeping-dev'
   OR session_user LIKE 'jayburd-dev-%' OR NOT pg_has_role(session_user,'bookkeeping_owner','SET')
   OR EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_owner' AND(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
   THEN RAISE EXCEPTION 'denied'; END IF;
  IF current_setting('jayburd.activation.expiry') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
   OR length(current_setting('jayburd.activation.subject')) NOT BETWEEN 1 AND 128
   OR octet_length(current_setting('jayburd.activation.body'))>4096 THEN RAISE EXCEPTION 'denied'; END IF;
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'sandbox_activation_denied'; END;
END $$;
SET LOCAL ROLE bookkeeping_owner;
DO $$ DECLARE matched uuid; BEGIN
 BEGIN
  SELECT principal_id INTO matched FROM ledger.principal
   WHERE issuer='https://securetoken.google.com/jayburd-bookkeeping-dev'
   AND subject=current_setting('jayburd.activation.subject') AND disabled_at IS NULL FOR UPDATE;
  IF matched IS NULL OR matched<>current_setting('jayburd.activation.principal')::uuid THEN RAISE EXCEPTION 'denied'; END IF;
  PERFORM set_config('jayburd.activation.result',banking.provision_collection_owner(matched,
   current_setting('jayburd.activation.key'),current_setting('jayburd.activation.body'))::text,true);
 EXCEPTION WHEN OTHERS THEN RAISE EXCEPTION 'sandbox_activation_denied'; END;
END $$;
-- An approval that expires while awaiting locks cannot commit a new capability.
DO $$ BEGIN
 IF clock_timestamp()>=current_setting('jayburd.activation.expiry')::timestamptz THEN RAISE EXCEPTION 'sandbox_activation_denied'; END IF;
END $$;
SELECT current_setting('jayburd.activation.result') AS activation_result \gset
COMMIT;
SELECT 'PASS' AS status, :'activation_result'::jsonb AS result;
