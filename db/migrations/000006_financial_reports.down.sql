-- Reviewed reporting releases preserve immutable migration history; correct forward.
DO $$ BEGIN RAISE EXCEPTION 'financial migrations are forward-only'; END $$;
