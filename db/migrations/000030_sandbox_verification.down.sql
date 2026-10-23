-- Only an unused capability can be removed; retained authorization history requires forward repair.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN
 IF EXISTS(SELECT FROM integration.verification_mandate) OR EXISTS(SELECT FROM integration.verification_operator_receipt) THEN RAISE EXCEPTION 'retained Sandbox verification evidence prevents downgrade';END IF;
END $$;
REVOKE ALL ON FUNCTION integration.verify_sandbox_fixture(uuid,bigint,text,text) FROM bookkeeping_verifier;
DROP FUNCTION integration.verify_sandbox_fixture(uuid,bigint,text,text);
DROP FUNCTION integration.verification_authorize(uuid,bigint);
DROP FUNCTION integration.enroll_verification(uuid,uuid,uuid,text,text);
DROP FUNCTION integration.revoke_verification(uuid,bigint,text);
DROP FUNCTION integration.verification_scope(integration.verification_mandate,boolean);
DROP FUNCTION integration.verifier_login(name);
DROP TABLE integration.verification_operator_receipt;
DROP TABLE integration.verification_mandate;
DROP FUNCTION integration.verification_guard();
REVOKE USAGE ON SCHEMA integration FROM bookkeeping_verifier;
-- The safe NOLOGIN capability is bootstrap-owned and is deliberately retained.
COMMIT;
