-- Read only the active delivery binding for one reviewed development book.
\set ON_ERROR_STOP on
\set VERBOSITY terse
BEGIN READ ONLY;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN
  IF current_database() <> 'bookkeeping' THEN
    RAISE EXCEPTION 'worker_binding_lookup_denied';
  END IF;
END $$;
SELECT binding_id::text
FROM integration.sync_delivery_binding
WHERE entity_id = :'entity_id'::uuid
  AND book_id = :'book_id'::uuid
  AND enabled
  AND worker_login = 'jayburd-dev-sync@jayburd-bookkeeping-dev.iam'::name
  AND publisher_login = 'jayburd-dev-publisher@jayburd-bookkeeping-dev.iam'::name;
COMMIT;
