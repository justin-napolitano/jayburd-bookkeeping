-- Fence a reviewed, unbound Item before provider-backed cancellation.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;

-- Abandoned attempts remain readable even when their retained Item is disconnected.
CREATE OR REPLACE FUNCTION integration.connection_context(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.claim_id IS NULL AND clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.sync_item_id IS NOT NULL AND a.state NOT IN ('active','disconnected','abandoned') THEN PERFORM integration.connection_item_pin(a);END IF;
 RETURN integration.render_connection(a,true);
END $$;

CREATE FUNCTION integration.begin_binding_abandon(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.state='abandoned' THEN RETURN integration.render_connection(a,true);END IF;
 IF a.mode<>'new' OR a.claim_id IS NULL OR a.sync_item_id IS NULL OR a.manifest IS NULL OR a.credential_ref IS NULL OR a.credential_version IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 IF a.state='binding_review' THEN
  i:=integration.connection_item_pin(a);
  IF i.state<>'paused'
   OR EXISTS(SELECT FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id)
   OR NOT EXISTS(SELECT FROM integration.credential_publication WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id AND credential_ref=a.credential_ref AND credential_version=a.credential_version AND attempt_id=id)
  THEN PERFORM integration.connection_error('P2704');END IF;
  UPDATE integration.sync_item SET state='disconnect_pending',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
  UPDATE integration.connection_attempt SET state='disconnect_pending',item_generation=i.generation,reason='owner_cancelled_binding_review' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
  INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'binding_cancel_requested',jsonb_build_object('item_generation',i.generation::text));
 ELSIF a.state='disconnect_pending' THEN
  i:=integration.connection_item_pin(a);
  IF i.state<>'disconnect_pending' OR EXISTS(SELECT FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id) THEN PERFORM integration.connection_error('P2704');END IF;
 ELSE PERFORM integration.connection_error('P2704');END IF;
 RETURN integration.render_connection(a,true)||jsonb_build_object('execute_provider',true);
END $$;

CREATE FUNCTION integration.confirm_binding_abandoned(actor uuid,e uuid,b uuid,id uuid,claim_value uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;i integration.sync_item;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.state='abandoned' THEN RETURN integration.render_connection(a,false);END IF;
 IF a.mode<>'new' OR a.state<>'disconnect_pending' OR claim_value IS NULL OR a.claim_id IS DISTINCT FROM claim_value OR a.sync_item_id IS NULL THEN PERFORM integration.connection_error('P2703');END IF;
 i:=integration.connection_item_pin(a);
 IF i.state<>'disconnect_pending'
  OR EXISTS(SELECT FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id)
  OR NOT EXISTS(SELECT FROM integration.credential_publication WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id AND credential_ref=a.credential_ref AND credential_version=a.credential_version AND attempt_id=id)
 THEN PERFORM integration.connection_error('P2704');END IF;
 UPDATE integration.sync_item SET state='disconnected',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=i.sync_item_id RETURNING * INTO i;
 UPDATE integration.connection_attempt SET state='abandoned',item_generation=i.generation WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'abandoned',jsonb_build_object('reason','owner_cancelled_binding_review','provider_item_removed',true));
 RETURN integration.render_connection(a,false);
END $$;

REVOKE ALL ON FUNCTION integration.begin_binding_abandon(uuid,uuid,uuid,uuid),integration.confirm_binding_abandoned(uuid,uuid,uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.begin_binding_abandon(uuid,uuid,uuid,uuid),integration.confirm_binding_abandoned(uuid,uuid,uuid,uuid,uuid) TO bookkeeping_api;
COMMIT;
