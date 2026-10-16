-- Encrypted credential history is immutable. Repair forward; never delete stored credentials.
DO $$ BEGIN RAISE EXCEPTION 'forward repair required for encrypted bank credentials'; END $$;
