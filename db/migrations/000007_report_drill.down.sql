-- Preserve reviewed financial migration history; corrections move forward.
DO $$ BEGIN RAISE EXCEPTION 'financial migrations are forward-only'; END $$;
