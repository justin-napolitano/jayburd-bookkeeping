-- Empty-only rollback; retained native/legacy authority, evidence and audit forbid destruction.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DO $$ BEGIN
 IF EXISTS(SELECT FROM banking.collection_workspace)OR EXISTS(SELECT FROM integration.collection_item)
 OR EXISTS(SELECT FROM banking.collection_source)OR EXISTS(SELECT FROM banking.collection_observation)
 OR EXISTS(SELECT FROM banking.collection_revision)OR EXISTS(SELECT FROM integration.collection_receipt)
 OR EXISTS(SELECT FROM audit.collection_action)THEN RAISE EXCEPTION 'private_registry_retained_records';END IF;
END $$;
DROP TRIGGER collection_identity ON integration.sync_item;
DROP TRIGGER collection_fence ON integration.sync_item;
DROP TRIGGER collection_fence ON integration.sync_stream;
DROP TRIGGER preserve_collection ON banking.collection_workspace;
DROP TRIGGER preserve_collection ON banking.collection_source;
DROP TRIGGER preserve_collection ON banking.collection_observation;
DROP TRIGGER preserve_collection ON integration.collection_item;
DROP TRIGGER immutable ON banking.collection_revision;
DROP TRIGGER immutable ON integration.collection_receipt;
DROP TRIGGER immutable ON audit.collection_action;
-- Explicit names only; no CASCADE can remove another slice's objects.
DROP FUNCTION banking.get_collection_workspace(uuid);
DROP FUNCTION banking.list_collection_sources(uuid,uuid,text);
DROP FUNCTION banking.query_collection_observations(uuid,uuid,text);
DROP FUNCTION banking.register_collection_item(uuid,uuid,text,text);
DROP FUNCTION banking.publish_collection_batch(uuid,uuid,uuid,text,text);
DROP FUNCTION banking.set_collection_state(uuid,uuid,text,text);
DROP FUNCTION banking.adopt_collection_item(uuid,uuid,uuid,uuid,uuid,text,text);
DROP FUNCTION banking.provision_collection_owner(uuid,text,text);
DROP FUNCTION banking.collection_facts(uuid);
DROP FUNCTION banking.collection_snapshot(uuid);
DROP FUNCTION banking.collection_visible_sources(uuid);
DROP FUNCTION banking.collection_page(jsonb,uuid,jsonb,jsonb);
DROP FUNCTION banking.collection_authorize(uuid,uuid,text,boolean);
DROP FUNCTION banking.collection_read_locks(uuid);
DROP FUNCTION banking.collection_normalize_fact(jsonb);
DROP FUNCTION banking.collection_guard();
DROP FUNCTION integration.collection_legacy_identity();
DROP FUNCTION banking.collection_text(jsonb,integer);
DROP FUNCTION banking.collection_keys(jsonb,text[]);
DROP FUNCTION banking.collection_error(text);
ALTER TABLE banking.collection_observation DROP CONSTRAINT collection_observation_current_fk;
DROP TABLE banking.collection_revision;
DROP TABLE banking.collection_observation;
DROP TABLE banking.collection_source;
DROP TABLE integration.collection_receipt;
DROP TABLE audit.collection_action;
DROP TABLE integration.collection_item;
DROP TABLE banking.collection_workspace;
COMMIT;
