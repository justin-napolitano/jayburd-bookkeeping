-- Preserve imported evidence, receipts and source identity; corrections move forward.
DO $$ BEGIN RAISE EXCEPTION 'financial migrations are forward-only'; END $$;
