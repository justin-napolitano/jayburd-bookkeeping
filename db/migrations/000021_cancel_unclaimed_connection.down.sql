-- Restore the prior abandonment policy; terminal history stays immutable.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE OR REPLACE FUNCTION integration.abandon_connection(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.state='abandoned' THEN RETURN integration.render_connection(a,false);END IF;
 IF a.mode<>'new' OR a.sync_item_id IS NOT NULL OR a.state<>'credential_recovery_required' OR a.claim_id IS NULL
 OR EXISTS(SELECT FROM integration.bank_credential WHERE entity_id=e AND book_id=b AND credential_ref=a.credential_ref AND credential_version=a.credential_version)
 OR EXISTS(SELECT FROM integration.credential_publication WHERE entity_id=e AND book_id=b AND attempt_id=id)
 THEN PERFORM integration.connection_error('P2704');END IF;
 UPDATE integration.connection_attempt SET state='abandoned' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'abandoned',jsonb_build_object('reason','owner_acknowledged_unavailable_credential','orphan_item_cleanup_required',a.manifest IS NOT NULL));
 RETURN integration.render_connection(a,false);
END $$;
COMMIT;
