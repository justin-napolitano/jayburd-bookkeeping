-- Provider and accounting evidence is forward-only.
DO $$ BEGIN RAISE EXCEPTION 'financial migrations are forward-only'; END $$;
