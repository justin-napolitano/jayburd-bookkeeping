-- Retained connection/work receipts cannot be discarded by downgrade.
DO $$ BEGIN RAISE EXCEPTION 'connection operations migration is forward-only'; END $$;
