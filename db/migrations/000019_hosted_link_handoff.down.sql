-- Encrypted handoff history is immutable; repair forward without erasing evidence.
DO $$ BEGIN RAISE EXCEPTION 'forward repair required for hosted Link handoffs'; END $$;
