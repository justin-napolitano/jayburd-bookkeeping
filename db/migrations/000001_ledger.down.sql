-- Posted history has no destructive rollback path. Use a reviewed forward repair.
BEGIN;
DO $$ BEGIN RAISE EXCEPTION 'forward-only ledger migration'; END $$;
COMMIT;
