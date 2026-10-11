-- Durable financial intent/history is forward-only, like its accounting receipts.
DO $$ BEGIN RAISE EXCEPTION 'financial migrations are forward-only'; END $$;
