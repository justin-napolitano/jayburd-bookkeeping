-- Empty-only downgrade; retained onboarding review/command/audit is never erased.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE banking.collection_onboarding_preview,banking.collection_onboarding_command,audit.collection_onboarding_action IN ACCESS EXCLUSIVE MODE;
DO $$BEGIN IF EXISTS(SELECT FROM banking.collection_onboarding_preview)OR EXISTS(SELECT FROM banking.collection_onboarding_command)OR EXISTS(SELECT FROM audit.collection_onboarding_action)OR EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE onboarding_command_id IS NOT NULL)THEN RAISE EXCEPTION 'collection_onboarding_retained_records';END IF;END $$;
DROP TRIGGER onboarding_workspace_fence ON banking.collection_workspace;
DROP TRIGGER onboarding_reserve ON banking.collection_onboarding_command;
DROP TRIGGER onboarding_release ON banking.collection_onboarding_command;
DROP TRIGGER preserve_onboarding ON banking.collection_onboarding_command;
DROP FUNCTION banking.get_collection_onboarding(uuid),banking.preview_collection_onboarding(uuid,text),banking.prepare_collection_onboarding(uuid,text),banking.pending_collection_onboarding(uuid),banking.transition_collection_onboarding(uuid,uuid,text,text),banking.clear_collection_onboarding(uuid,text);
DROP FUNCTION banking.collection_onboarding_command_wire(banking.collection_onboarding_command),banking.collection_onboarding_preview_wire(banking.collection_onboarding_preview),banking.collection_onboarding_authorize(uuid,banking.collection_onboarding_preview),banking.collection_onboarding_candidate(uuid,uuid,boolean),banking.collection_onboarding_workspace_fence(),banking.collection_onboarding_guard();
DROP FUNCTION ledger.pending_command(uuid),ledger.clear_inaccessible_command(uuid,uuid,text),banking.collection_recovery_status(uuid),banking.clear_collection_recovery(uuid,text);
ALTER FUNCTION ledger.onboarding_pending_base(uuid)RENAME TO pending_command;
ALTER FUNCTION ledger.onboarding_clear_base(uuid,uuid,text)RENAME TO clear_inaccessible_command;
ALTER FUNCTION banking.collection_onboarding_recovery_status_base(uuid)RENAME TO collection_recovery_status;
ALTER FUNCTION banking.collection_onboarding_recovery_clear_base(uuid,text)RENAME TO clear_collection_recovery;
DO $$DECLARE n text;ver text;BEGIN FOR ver IN SELECT unnest(ARRAY['','_v2','_v3'])LOOP
 n:='pending_collection_label_command'||ver;EXECUTE format('DROP FUNCTION banking.%I(uuid,uuid)',n);
 EXECUTE format('ALTER FUNCTION banking.%I(uuid,uuid)RENAME TO %I','collection_onboarding_pending'||coalesce(nullif(ver,''),'_v1')||'_base',n);
 END LOOP;END $$;
GRANT EXECUTE ON FUNCTION ledger.pending_command(uuid),ledger.clear_inaccessible_command(uuid,uuid,text),banking.collection_recovery_status(uuid),banking.clear_collection_recovery(uuid,text),banking.pending_collection_label_command(uuid,uuid),banking.pending_collection_label_command_v2(uuid,uuid),banking.pending_collection_label_command_v3(uuid,uuid)TO bookkeeping_api;
ALTER TABLE ledger.interactive_command_reservation DROP CONSTRAINT interactive_command_reservation_check;
ALTER TABLE ledger.interactive_command_reservation DROP COLUMN onboarding_command_id;
ALTER TABLE ledger.collection_recovery_receipt DROP COLUMN onboarding_command_id;
ALTER TABLE ledger.interactive_command_reservation ADD CONSTRAINT interactive_command_reservation_check CHECK((legacy_command_id IS NOT NULL AND workspace_id IS NULL AND label_command_id IS NULL)OR(legacy_command_id IS NULL AND workspace_id IS NOT NULL AND label_command_id IS NOT NULL));
CREATE OR REPLACE FUNCTION banking.collection_label_reserve()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE old_id uuid;private_id uuid;work uuid;accepted uuid;legacy boolean:=TG_TABLE_SCHEMA='ledger';BEGIN
 IF TG_OP='INSERT'THEN
  IF NEW.acknowledged_at IS NOT NULL THEN RETURN NEW;END IF;
  IF legacy THEN old_id:=NEW.command_id;ELSE private_id:=NEW.command_id;work:=NEW.workspace_id;END IF;
  INSERT INTO ledger.interactive_command_reservation(principal_id,legacy_command_id,workspace_id,label_command_id)
   VALUES(NEW.principal_id,old_id,work,private_id)ON CONFLICT(principal_id)DO NOTHING RETURNING principal_id INTO accepted;
  IF accepted IS NULL THEN
   IF legacy THEN PERFORM ledger.fail('P2112');ELSE PERFORM banking.collection_error('P2802');END IF;
  END IF;
 ELSIF OLD.acknowledged_at IS NULL AND NEW.acknowledged_at IS NOT NULL THEN
  accepted:=NULL;
  IF legacy THEN DELETE FROM ledger.interactive_command_reservation WHERE principal_id=NEW.principal_id AND legacy_command_id=NEW.command_id RETURNING principal_id INTO accepted;
  ELSE DELETE FROM ledger.interactive_command_reservation WHERE principal_id=NEW.principal_id AND workspace_id=NEW.workspace_id AND label_command_id=NEW.command_id RETURNING principal_id INTO accepted;END IF;
  -- A pre43 SERIALIZABLE snapshot may not see the newly backfilled reservation.
  -- Never commit acknowledgement while silently leaving its exact current claim behind.
  IF accepted IS NULL THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_command_acknowledgement';END IF;
 END IF;RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION banking.collection_recovery_fence()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE subject uuid;observed uuid;BEGIN
 subject:=CASE WHEN TG_OP='DELETE'THEN OLD.principal_id ELSE NEW.principal_id END;
 IF TG_OP='DELETE'AND OLD.legacy_command_id IS NULL THEN
  IF NOT EXISTS(SELECT FROM banking.collection_label_command WHERE workspace_id=OLD.workspace_id AND command_id=OLD.label_command_id AND principal_id=OLD.principal_id AND acknowledged_at IS NOT NULL)THEN PERFORM banking.collection_error('P2802');END IF;
  INSERT INTO ledger.collection_recovery_receipt(principal_id,recovery_id)VALUES(OLD.principal_id,OLD.recovery_id)ON CONFLICT DO NOTHING;
 END IF;
 UPDATE ledger.collection_recovery_state SET revision=revision+1 WHERE principal_id=subject RETURNING principal_id INTO observed;
 IF observed IS NULL THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_recovery';END IF;RETURN NULL;
END $$;
DROP TABLE audit.collection_onboarding_action,banking.collection_onboarding_command,banking.collection_onboarding_preview;
COMMIT;
