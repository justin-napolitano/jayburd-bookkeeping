-- Disposable rollback only: never use a down migration as live credential/session recovery.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DROP FUNCTION integration.connection_link_proof(uuid,uuid,uuid,uuid);
DROP FUNCTION integration.accept_verified_link_session(text,text,text);
DROP FUNCTION integration.abandon_connection(uuid,uuid,uuid,uuid);
DROP FUNCTION integration.put_link_handoff(uuid,uuid,uuid,uuid,text);
DROP FUNCTION integration.get_link_handoff(uuid,uuid,uuid,uuid);
DROP FUNCTION integration.get_repair_bank_credential(uuid,uuid,uuid,uuid);
DROP FUNCTION integration.connection_context(uuid,uuid,uuid,uuid);
DROP FUNCTION integration.confirm_connection_credential(uuid,uuid,uuid,uuid,uuid);
DROP FUNCTION integration.record_connection_manifest(uuid,uuid,uuid,uuid,uuid,text);
DROP FUNCTION integration.claim_connection(uuid,uuid,uuid,uuid,uuid,text);
DROP FUNCTION integration.record_link_session(uuid,uuid,uuid,uuid,uuid,text);
DROP FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text);
DROP FUNCTION integration.render_connection(integration.connection_attempt,boolean);
ALTER FUNCTION integration.render_connection_before_owned_repair32(integration.connection_attempt,boolean) RENAME TO render_connection;
ALTER FUNCTION integration.begin_connection_before_owned_repair32(uuid,uuid,uuid,text,text) RENAME TO begin_connection;
ALTER FUNCTION integration.record_link_session_before_owned_repair32(uuid,uuid,uuid,uuid,uuid,text) RENAME TO record_link_session;
ALTER FUNCTION integration.claim_connection_before_owned_repair32(uuid,uuid,uuid,uuid,uuid,text) RENAME TO claim_connection;
ALTER FUNCTION integration.record_connection_manifest_before_owned_repair32(uuid,uuid,uuid,uuid,uuid,text) RENAME TO record_connection_manifest;
ALTER FUNCTION integration.confirm_connection_credential_before_owned_repair32(uuid,uuid,uuid,uuid,uuid) RENAME TO confirm_connection_credential;
ALTER FUNCTION integration.connection_context_before_owned_repair32(uuid,uuid,uuid,uuid) RENAME TO connection_context;
ALTER FUNCTION integration.get_link_handoff_before_owned_repair32(uuid,uuid,uuid,uuid) RENAME TO get_link_handoff;
ALTER FUNCTION integration.put_link_handoff_before_owned_repair32(uuid,uuid,uuid,uuid,text) RENAME TO put_link_handoff;
ALTER FUNCTION integration.abandon_connection_before_owned_repair32(uuid,uuid,uuid,uuid) RENAME TO abandon_connection;
GRANT EXECUTE ON FUNCTION integration.begin_connection(uuid,uuid,uuid,text,text),
 integration.record_link_session(uuid,uuid,uuid,uuid,uuid,text),integration.claim_connection(uuid,uuid,uuid,uuid,uuid,text),
 integration.record_connection_manifest(uuid,uuid,uuid,uuid,uuid,text),integration.confirm_connection_credential(uuid,uuid,uuid,uuid,uuid),
 integration.connection_context(uuid,uuid,uuid,uuid),integration.get_link_handoff(uuid,uuid,uuid,uuid),
 integration.put_link_handoff(uuid,uuid,uuid,uuid,text),integration.abandon_connection(uuid,uuid,uuid,uuid) TO bookkeeping_api;
DROP FUNCTION integration.owned_repair_proof(integration.connection_attempt);
DROP FUNCTION integration.owned_repair_receipt_lock(integration.connection_attempt);
DROP FUNCTION integration.owned_repair_lock(uuid,uuid,uuid,uuid);
DROP FUNCTION integration.owned_repair_scope(integration.connection_attempt);
DROP FUNCTION integration.owned_repair_scope_digest(integration.connection_attempt);
DROP FUNCTION integration.owned_repair_principals(integration.connection_attempt);
DROP FUNCTION integration.owned_repair_deadline(integration.connection_attempt);
DROP TRIGGER preserve_repair_scope ON integration.connection_attempt;
DROP FUNCTION integration.owned_repair_attempt_guard();
ALTER TABLE integration.connection_attempt DROP COLUMN repair_scope_sha256;
DROP INDEX integration.connection_attempt_link_hash;
DROP TABLE integration.link_repair_claim,integration.link_session_receipt;
-- Immutable policy-change events and already shortened mutable deadlines are not erased or extended.
COMMIT;
