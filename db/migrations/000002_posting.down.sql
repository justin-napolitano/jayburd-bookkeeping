-- Keep posting and historical receipt interpretation compatible; repair forward.
BEGIN;
DO $$ BEGIN RAISE EXCEPTION 'forward-only posting migration'; END $$;
COMMIT;
