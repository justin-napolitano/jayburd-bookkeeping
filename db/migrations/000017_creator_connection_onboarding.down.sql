-- Provisioned permissions and audit history must be repaired forward, never silently removed.
BEGIN;
DO $$ BEGIN RAISE EXCEPTION 'creator_connection_onboarding_requires_forward_repair';END $$;
COMMIT;
