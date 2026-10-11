-- Identity/setup history is durable; repair forward instead of deleting it.
DO $$ BEGIN RAISE EXCEPTION 'forward-only financial migration'; END $$;
