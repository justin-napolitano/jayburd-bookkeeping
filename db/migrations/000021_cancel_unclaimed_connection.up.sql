-- Serialize explicit cancellation with the exchange claim; never discard a claimed exchange.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE OR REPLACE FUNCTION integration.abandon_connection(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;unclaimed boolean;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.state='abandoned' THEN RETURN integration.render_connection(a,false);END IF;
 unclaimed:=a.state IN ('link_pending','link_ready') AND a.claim_id IS NULL;
 IF a.mode<>'new' OR a.sync_item_id IS NOT NULL
 OR NOT (unclaimed OR (a.state='credential_recovery_required' AND a.claim_id IS NOT NULL))
 OR EXISTS(SELECT FROM integration.bank_credential WHERE entity_id=e AND book_id=b AND credential_ref=a.credential_ref AND credential_version=a.credential_version)
 OR EXISTS(SELECT FROM integration.credential_publication WHERE entity_id=e AND book_id=b AND attempt_id=id)
 THEN PERFORM integration.connection_error('P2704');END IF;
 UPDATE integration.connection_attempt SET state='abandoned' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'abandoned',jsonb_build_object('reason',CASE WHEN unclaimed THEN 'owner_cancelled_unclaimed_setup' ELSE 'owner_acknowledged_unavailable_credential' END,'orphan_item_cleanup_required',a.manifest IS NOT NULL));
 RETURN integration.render_connection(a,false);
END $$;
COMMIT;
