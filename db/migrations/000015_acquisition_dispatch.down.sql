-- Delivery and acquisition retry evidence are retained.
DO $$ BEGIN RAISE EXCEPTION 'acquisition dispatch migration is forward-only';END $$;
