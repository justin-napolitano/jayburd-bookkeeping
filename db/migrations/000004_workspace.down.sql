-- History-bearing releases only move forward; corrective changes need a new migration.
DO $$ BEGIN RAISE EXCEPTION 'financial migrations are forward-only'; END $$;
