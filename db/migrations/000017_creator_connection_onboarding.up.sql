-- Creator-authorized book preparation; no cloud IAM, role grants or provider calls.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Creator proof stays scoped without scanning all books created by a large operator.
CREATE INDEX setup_receipt_creator_scope ON ledger.setup_receipt(principal_id,entity_id,book_id);

-- The private helper checks eligibility on reads and mutates only for an explicit Begin(new).
CREATE FUNCTION integration.creator_connection_setup(actor uuid,e uuid,b uuid,worker name,publisher name,apply_value boolean) RETURNS boolean
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE w integration.connection_worker;d integration.sync_delivery_binding;s uuid;
BEGIN
 IF current_user<>'bookkeeping_owner' OR current_setting('transaction_isolation')<>'serializable'
 OR actor IS NULL OR e IS NULL OR b IS NULL OR apply_value IS NULL THEN PERFORM integration.connection_error('P2700');END IF;
 IF worker IS NULL OR publisher IS NULL OR worker=publisher OR NOT integration.sync_login(worker)
 OR NOT integration.sync_publisher_login(publisher) THEN PERFORM integration.connection_error('P2701');END IF;
 -- Serialize creator provisioning with setup/command changes before taking book locks.
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN PERFORM integration.connection_error('P2701');END IF;
 PERFORM ledger.authorize(actor,e,b,'read',true);
 SELECT * INTO w FROM integration.connection_worker WHERE entity_id=e AND book_id=b;
 SELECT * INTO d FROM integration.sync_delivery_binding WHERE entity_id=e AND book_id=b;
 IF w.book_id IS NOT NULL THEN
  PERFORM integration.connection_authorize(actor,e,b);
  IF d.book_id IS NULL OR NOT d.enabled OR w.worker_login<>worker
  OR ROW(d.service_principal,d.worker_login,d.publisher_login) IS DISTINCT FROM ROW(w.service_principal,worker,publisher)
  THEN PERFORM integration.connection_error('P2701');END IF;
  RETURN false;
 END IF;
 IF d.book_id IS NOT NULL THEN PERFORM integration.connection_error('P2701');END IF;
 PERFORM ledger.authorize(actor,e,b,'post',true);
 IF NOT EXISTS(SELECT FROM ledger.setup_receipt WHERE principal_id=actor AND entity_id=e AND book_id=b)
 THEN PERFORM integration.connection_error('P2701');END IF;
 SELECT principal_id INTO s FROM ledger.principal WHERE issuer='urn:jayburd:service:book-acquisition' AND subject='acquisition:'||b;
 IF s IS NOT NULL THEN
  PERFORM 1 FROM ledger.principal WHERE principal_id=s AND disabled_at IS NULL FOR UPDATE;
  IF NOT FOUND OR s=actor THEN PERFORM integration.connection_error('P2701');END IF;
 END IF;
 IF EXISTS(SELECT FROM ledger.membership WHERE entity_id=e AND revoked_at IS NOT NULL
 AND ((principal_id=actor AND permission IN ('import','connect')) OR (principal_id=s AND permission IN ('read','import'))))
 THEN PERFORM integration.connection_error('P2701');END IF;
 -- Discovery proves eligibility but neither creates identities nor grants access.
 IF NOT apply_value THEN RETURN true;END IF;
 IF s IS NULL THEN
  INSERT INTO ledger.principal(principal_id,issuer,subject)
  VALUES(gen_random_uuid(),'urn:jayburd:service:book-acquisition','acquisition:'||b)
  ON CONFLICT(issuer,subject) DO NOTHING;
  SELECT principal_id INTO s FROM ledger.principal WHERE issuer='urn:jayburd:service:book-acquisition' AND subject='acquisition:'||b AND disabled_at IS NULL FOR UPDATE;
  IF s IS NULL OR s=actor THEN PERFORM integration.connection_error('P2701');END IF;
 END IF;
 INSERT INTO ledger.membership(entity_id,principal_id,permission,revoked_at)
 VALUES(e,actor,'import',NULL),(e,s,'read',NULL),(e,s,'import',NULL)
 ON CONFLICT(entity_id,principal_id,permission) DO NOTHING;
 PERFORM integration.set_connect_permission(actor,e,b,actor,'creator-connect-v1',true);
 PERFORM integration.set_connection_worker(actor,e,b,'creator-worker-v1',jsonb_build_object('service_principal_id',s,'worker_login',worker,'enabled',true)::text);
 PERFORM integration.enroll_sync_delivery(actor,e,b,'creator-delivery-v1',jsonb_build_object('service_principal_id',s,'worker_login',worker,'publisher_login',publisher,'enabled',true)::text);
 RETURN true;
END $$;

-- Reads expose the existing wire shape and never provision a book or contact Plaid.
CREATE FUNCTION integration.creator_list_connections(actor uuid,e uuid,b uuid,worker name,publisher name) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF integration.creator_connection_setup(actor,e,b,worker,publisher,false) THEN RETURN jsonb_build_object('items','[]'::jsonb);END IF;
 RETURN integration.list_connections(actor,e,b);
END $$;
CREATE FUNCTION integration.creator_list_connection_attempts(actor uuid,e uuid,b uuid,maximum integer,after_id uuid,worker name,publisher name) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
BEGIN
 IF maximum IS NULL OR maximum<1 OR maximum>100 THEN PERFORM integration.connection_error('P2700');END IF;
 IF integration.creator_connection_setup(actor,e,b,worker,publisher,false) THEN
  RETURN jsonb_build_object('items','[]'::jsonb,'unfinished_count','0','next_cursor',NULL);
 END IF;
 RETURN integration.list_connection_attempts(actor,e,b,maximum,after_id);
END $$;

-- Preparation and attempt creation commit together; an invalid Begin rolls back all grants.
CREATE FUNCTION integration.creator_begin_connection(actor uuid,e uuid,b uuid,k text,raw text,worker name,publisher name) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;
BEGIN
 v:=integration.connection_json(raw,32768);
 IF v->>'mode'='new' THEN PERFORM integration.creator_connection_setup(actor,e,b,worker,publisher,true);END IF;
 RETURN integration.begin_connection(actor,e,b,k,raw);
END $$;
REVOKE ALL ON FUNCTION integration.creator_connection_setup(uuid,uuid,uuid,name,name,boolean) FROM PUBLIC,bookkeeping_api;
REVOKE ALL ON FUNCTION integration.creator_list_connections(uuid,uuid,uuid,name,name),integration.creator_list_connection_attempts(uuid,uuid,uuid,integer,uuid,name,name),integration.creator_begin_connection(uuid,uuid,uuid,text,text,name,name) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION integration.creator_list_connections(uuid,uuid,uuid,name,name),integration.creator_list_connection_attempts(uuid,uuid,uuid,integer,uuid,name,name),integration.creator_begin_connection(uuid,uuid,uuid,text,text,name,name) TO bookkeeping_api;
COMMIT;
