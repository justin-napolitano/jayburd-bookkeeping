-- Retain durable observer receipts; reviewed correction migrations move forward.
DO $$ BEGIN RAISE EXCEPTION 'financial migrations are forward-only'; END $$;
