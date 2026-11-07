-- Empty-only44 rollback; no retained tag or closure history is destroyed.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE ledger.principal,ledger.interactive_command_reservation,banking.collection_workspace,banking.collection_label_state IN ACCESS EXCLUSIVE MODE;
DO $$BEGIN
 IF EXISTS(SELECT FROM banking.collection_tag_edition)OR EXISTS(SELECT FROM banking.collection_tag_decision)OR EXISTS(SELECT FROM banking.collection_tag_preview_result)OR EXISTS(SELECT FROM ledger.collection_recovery_state)OR EXISTS(SELECT FROM ledger.collection_recovery_receipt)OR EXISTS(SELECT FROM banking.collection_label_preview WHERE wire_version=2)THEN RAISE EXCEPTION USING ERRCODE='P2802',MESSAGE='tags_recovery_retained_records';END IF;
END $$;
DROP TRIGGER collection_recovery_initialize ON ledger.principal;
DROP TRIGGER collection_recovery_fence ON ledger.interactive_command_reservation;
DROP TRIGGER preserve_recovery_state ON ledger.collection_recovery_state;
DROP TRIGGER immutable ON ledger.collection_recovery_receipt;
DROP TRIGGER immutable ON banking.collection_tag_edition;
DROP TRIGGER immutable ON banking.collection_tag_decision;
DROP TRIGGER immutable ON banking.collection_tag_preview_result;
DROP FUNCTION banking.get_collection_taxonomy_v2(uuid,uuid),banking.query_collection_review_v2(uuid,uuid,text),banking.preview_collection_labels_v2(uuid,uuid,text),banking.prepare_collection_label_command_v2(uuid,uuid,text),banking.pending_collection_label_command_v2(uuid,uuid),banking.transition_collection_label_command_v2(uuid,uuid,uuid,text,text),banking.collection_recovery_status(uuid),banking.clear_collection_recovery(uuid,text);
DROP FUNCTION banking.prepare_collection_label_command(uuid,uuid,text),banking.pending_collection_label_command(uuid,uuid),banking.transition_collection_label_command(uuid,uuid,uuid,text,text);
ALTER FUNCTION banking.collection_tag_prepare_v1_base(uuid,uuid,text)RENAME TO prepare_collection_label_command;
ALTER FUNCTION banking.collection_tag_pending_v1_base(uuid,uuid)RENAME TO pending_collection_label_command;
ALTER FUNCTION banking.collection_tag_transition_v1_base(uuid,uuid,uuid,text,text)RENAME TO transition_collection_label_command;
DROP FUNCTION banking.collection_tag_reauthorize(uuid,uuid,banking.collection_label_preview),banking.collection_tag_render_command(banking.collection_label_command),banking.collection_tag_render_preview(banking.collection_label_preview),banking.collection_tag_result(uuid,uuid,uuid,bigint,jsonb),banking.collection_tag_action(uuid,jsonb),banking.collection_tag_rows(uuid,uuid),banking.collection_tag_names(uuid,uuid[],bigint),banking.collection_tag_pins(jsonb,jsonb),banking.collection_tag_snapshot(uuid);
DROP FUNCTION banking.collection_label_snapshot(uuid);
ALTER FUNCTION banking.collection_tag_snapshot_v1_base(uuid)RENAME TO collection_label_snapshot;
DROP FUNCTION banking.collection_recovery_available(uuid,banking.collection_label_command),banking.collection_recovery_lock(uuid),banking.collection_recovery_principal(uuid),banking.collection_recovery_fence(),banking.collection_recovery_guard(),banking.collection_recovery_initialize();
CREATE OR REPLACE FUNCTION banking.collection_label_initialize()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics)VALUES(NEW.workspace_id,1,'[]');
 INSERT INTO banking.collection_label_state(workspace_id)VALUES(NEW.workspace_id);RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION banking.collection_label_guard()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF TG_OP='DELETE'THEN PERFORM banking.collection_error('P2802');END IF;
 IF TG_TABLE_NAME='collection_label_state'THEN
  IF NEW.workspace_id<>OLD.workspace_id OR NOT(
   (NEW.command_revision=OLD.command_revision AND((NEW.taxonomy_revision=OLD.taxonomy_revision+1 AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision)OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision+1 AND NEW.projection_revision=OLD.projection_revision)OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision+1)))
   OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision AND NEW.command_revision=OLD.command_revision+1))THEN PERFORM banking.collection_error('P2802');END IF;
 ELSE
  IF(to_jsonb(NEW)-ARRAY['state','result','terminal_at','acknowledged_at'])IS DISTINCT FROM(to_jsonb(OLD)-ARRAY['state','result','terminal_at','acknowledged_at'])THEN PERFORM banking.collection_error('P2802');END IF;
  IF OLD.state='prepared'THEN
   IF NEW.state NOT IN('committed','cancelled')OR NEW.terminal_at IS NULL OR NEW.result IS NULL OR NEW.acknowledged_at IS NOT NULL THEN PERFORM banking.collection_error('P2802');END IF;
  ELSE
   IF ROW(NEW.state,NEW.result,NEW.terminal_at)IS DISTINCT FROM ROW(OLD.state,OLD.result,OLD.terminal_at)OR OLD.acknowledged_at IS NOT NULL OR NEW.acknowledged_at IS NULL THEN PERFORM banking.collection_error('P2802');END IF;
  END IF;
 END IF;RETURN NEW;
END $$;
ALTER TABLE banking.collection_label_state DROP CONSTRAINT collection_label_tag_edition_fk;
ALTER TABLE banking.collection_label_state DROP COLUMN tag_revision;
ALTER TABLE banking.collection_label_preview DROP COLUMN wire_version;
ALTER TABLE ledger.interactive_command_reservation DROP CONSTRAINT interactive_command_recovery_id_key;
ALTER TABLE ledger.interactive_command_reservation DROP COLUMN recovery_id;
DROP TABLE banking.collection_tag_preview_result,banking.collection_tag_decision;
DROP TABLE banking.collection_tag_edition,ledger.collection_recovery_receipt,ledger.collection_recovery_state;
DROP FUNCTION banking.collection_tag_sorted(uuid[]);
GRANT EXECUTE ON FUNCTION banking.prepare_collection_label_command(uuid,uuid,text),banking.pending_collection_label_command(uuid,uuid),banking.transition_collection_label_command(uuid,uuid,uuid,text,text)TO bookkeeping_api;
COMMIT;
