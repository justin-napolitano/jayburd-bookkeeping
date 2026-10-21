-- #156: login-scoped durable keyset selection for the shared Sandbox publisher.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE INDEX sync_delivery_publisher_page ON integration.sync_delivery_binding(publisher_login,binding_id) WHERE enabled;
CREATE TABLE integration.sync_publisher_cursor (
 publisher_login name PRIMARY KEY,
 last_binding_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
 updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE FUNCTION integration.sync_delivery_bindings(maximum integer) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE current_id uuid;ids uuid[]:=ARRAY[]::uuid[];next_id uuid;count_value integer:=0;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR maximum IS NULL OR maximum NOT BETWEEN 1 AND 5 THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 IF NOT integration.sync_publisher_login(session_user) THEN PERFORM integration.sync_delivery_error('P2901');END IF;
 INSERT INTO integration.sync_publisher_cursor(publisher_login) VALUES(session_user::name) ON CONFLICT DO NOTHING;
 SELECT last_binding_id INTO current_id FROM integration.sync_publisher_cursor WHERE publisher_login=session_user::name FOR UPDATE;
 FOR next_id IN SELECT binding_id FROM integration.sync_delivery_binding WHERE publisher_login=session_user::name AND enabled AND binding_id>current_id ORDER BY binding_id LIMIT maximum LOOP
  ids:=array_append(ids,next_id);count_value:=count_value+1;
 END LOOP;
 IF count_value<maximum THEN
  FOR next_id IN SELECT binding_id FROM integration.sync_delivery_binding WHERE publisher_login=session_user::name AND enabled AND binding_id<=current_id ORDER BY binding_id LIMIT (maximum-count_value) LOOP
   ids:=array_append(ids,next_id);count_value:=count_value+1;
  END LOOP;
 END IF;
 IF count_value>0 THEN UPDATE integration.sync_publisher_cursor SET last_binding_id=ids[count_value],updated_at=clock_timestamp() WHERE publisher_login=session_user::name;END IF;
 RETURN jsonb_build_object('binding_ids',ids);
END $$;
REVOKE ALL ON integration.sync_publisher_cursor FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION integration.sync_delivery_bindings(integer) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.sync_delivery_bindings(integer) TO bookkeeping_publisher;
COMMIT;
