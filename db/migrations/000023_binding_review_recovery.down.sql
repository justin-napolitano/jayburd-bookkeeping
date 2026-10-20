BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DROP FUNCTION integration.confirm_binding_abandoned(uuid,uuid,uuid,uuid,uuid);
DROP FUNCTION integration.begin_binding_abandon(uuid,uuid,uuid,uuid);
CREATE OR REPLACE FUNCTION integration.connection_context(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.claim_id IS NULL AND clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.sync_item_id IS NOT NULL AND a.state NOT IN ('active','disconnected') THEN PERFORM integration.connection_item_pin(a);END IF;
 RETURN integration.render_connection(a,true);
END $$;
COMMIT;
