-- Per-account encrypted tokens; only ciphertext crosses the SQL boundary.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE integration.bank_credential (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, credential_ref uuid NOT NULL,
 credential_version bigint NOT NULL CHECK(credential_version>0), item_id text NOT NULL,
 environment text NOT NULL CHECK(environment='sandbox'), envelope jsonb NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,credential_ref,credential_version),
 FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,
 CHECK(length(item_id) BETWEEN 1 AND 256), CHECK(octet_length(envelope::text)<=32768)
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.bank_credential
 FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
REVOKE ALL ON integration.bank_credential FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
-- Recheck current initiating-actor authority; the reference must belong to a scoped attempt.
CREATE FUNCTION integration.bank_credential_authorize(actor uuid,e uuid,b uuid,r uuid,v bigint,item text,env text,write_value boolean) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;
BEGIN
 PERFORM integration.connection_authorize(actor,e,b);
 IF v IS NULL OR v<1 OR env IS DISTINCT FROM 'sandbox' OR item IS NULL THEN PERFORM integration.connection_error('P2700');END IF;
 SELECT * INTO a FROM integration.connection_attempt x WHERE x.entity_id=e AND x.book_id=b AND x.actor_id=actor
 AND ((x.credential_ref=r AND x.credential_version=v AND x.manifest->>'item_id'=item)
 OR (NOT write_value AND x.previous_credential_ref=r AND x.previous_credential_version=v AND x.expected_item_id=item))
 ORDER BY x.expires_at DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND OR a.claim_id IS NULL THEN PERFORM integration.connection_error('P2701');END IF;
 a:=integration.connection_lock(actor,e,b,a.attempt_id);
 IF a.sync_item_id IS NOT NULL THEN PERFORM integration.connection_item_pin(a);END IF;
 IF write_value AND (a.state NOT IN ('credential_pending','credential_recovery_required') OR a.manifest IS NULL) THEN PERFORM integration.connection_error('P2704');END IF;
END $$;
-- Read a ciphertext-only record; a missing row is explicit and cannot become plaintext fallback.
CREATE FUNCTION integration.get_bank_credential(actor uuid,e uuid,b uuid,r uuid,v bigint,item text,env text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE value jsonb;
BEGIN
 PERFORM integration.bank_credential_authorize(actor,e,b,r,v,item,env,false);
 SELECT envelope INTO value FROM integration.bank_credential WHERE entity_id=e AND book_id=b AND credential_ref=r AND credential_version=v AND item_id=item AND environment=env;
 RETURN jsonb_build_object('envelope',value);
END $$;
-- Concurrent inserts return the immutable winner; the caller decrypts and compares before confirmation.
CREATE FUNCTION integration.put_bank_credential(actor uuid,e uuid,b uuid,r uuid,v bigint,item text,env text,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE value jsonb;result_value jsonb;
BEGIN
 PERFORM integration.bank_credential_authorize(actor,e,b,r,v,item,env,true);
 value:=integration.connection_json(raw,32768);
 PERFORM ledger.keys(value,ARRAY['profile','key_version','wrapped_key','nonce','ciphertext']);
 IF value->>'profile' IS DISTINCT FROM 'jayburd-bank-aead-v1'
 OR jsonb_typeof(value->'key_version') IS DISTINCT FROM 'string'
 OR length(value->>'key_version') NOT BETWEEN 1 AND 512
 OR (value->>'key_version') !~ '^projects/[0-9]+/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$'
 OR jsonb_typeof(value->'wrapped_key') IS DISTINCT FROM 'string'
 OR jsonb_typeof(value->'nonce') IS DISTINCT FROM 'string'
 OR jsonb_typeof(value->'ciphertext') IS DISTINCT FROM 'string'
 THEN PERFORM integration.connection_error('P2700');END IF;
 IF octet_length(decode(value->>'wrapped_key','base64')) NOT BETWEEN 1 AND 8192
 OR octet_length(decode(value->>'nonce','base64'))<>12
 OR octet_length(decode(value->>'ciphertext','base64')) NOT BETWEEN 17 AND 4112
 THEN PERFORM integration.connection_error('P2700');END IF;
 INSERT INTO integration.bank_credential(entity_id,book_id,credential_ref,credential_version,item_id,environment,envelope)
 VALUES(e,b,r,v,item,env,value) ON CONFLICT(entity_id,book_id,credential_ref,credential_version) DO NOTHING;
 SELECT envelope INTO result_value FROM integration.bank_credential WHERE entity_id=e AND book_id=b AND credential_ref=r AND credential_version=v AND item_id=item AND environment=env;
 IF result_value IS NULL THEN PERFORM integration.connection_error('P2702');END IF;
 RETURN jsonb_build_object('envelope',result_value);
EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');END $$;
-- Worker lookup derives its reference from current lease/fence authority, not caller metadata.
CREATE FUNCTION integration.worker_bank_credential(e uuid,b uuid,stream uuid,set_value uuid,f bigint) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c jsonb;value jsonb;
BEGIN
 c:=integration.sync_context(e,b,stream,set_value,f);
 SELECT envelope INTO value FROM integration.bank_credential WHERE entity_id=e AND book_id=b
 AND credential_ref=(c->>'credential_ref')::uuid AND credential_version=(c->>'credential_version')::bigint
 AND item_id=c->>'item_id' AND environment=c->>'environment';
 IF value IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 RETURN jsonb_build_object('envelope',value,'credential_ref',c->'credential_ref','credential_version',c->'credential_version','item_id',c->'item_id','environment',c->'environment');
END $$;
REVOKE ALL ON FUNCTION integration.bank_credential_authorize(uuid,uuid,uuid,uuid,bigint,text,text,boolean),integration.get_bank_credential(uuid,uuid,uuid,uuid,bigint,text,text),integration.put_bank_credential(uuid,uuid,uuid,uuid,bigint,text,text,text),integration.worker_bank_credential(uuid,uuid,uuid,uuid,bigint) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.get_bank_credential(uuid,uuid,uuid,uuid,bigint,text,text),integration.put_bank_credential(uuid,uuid,uuid,uuid,bigint,text,text,text) TO bookkeeping_api;
GRANT EXECUTE ON FUNCTION integration.worker_bank_credential(uuid,uuid,uuid,uuid,bigint) TO bookkeeping_sync;
-- Explicit Sandbox abandonment preserves the failed claim and orphan Item audit trail.
ALTER TABLE integration.connection_attempt DROP CONSTRAINT connection_attempt_state_check;
ALTER TABLE integration.connection_attempt ADD CONSTRAINT connection_attempt_state_check CHECK(state IN ('link_pending','link_ready','exchange_claimed','credential_pending','binding_review','active','credential_recovery_required','disconnect_pending','disconnected','abandoned'));
CREATE OR REPLACE FUNCTION integration.connection_unfinished(a integration.connection_attempt) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT a.state NOT IN ('active','disconnected','abandoned') AND NOT (a.state='binding_review' AND a.expires_at<=statement_timestamp()) AND (a.claim_id IS NOT NULL OR a.expires_at>statement_timestamp())
$$;
CREATE FUNCTION integration.abandon_connection(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
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
CREATE FUNCTION integration.abandoned_connection_immutable() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN PERFORM integration.connection_error('P2702');RETURN NULL;END $$;
CREATE TRIGGER abandoned_immutable BEFORE UPDATE OR DELETE ON integration.connection_attempt FOR EACH ROW WHEN(OLD.state='abandoned') EXECUTE FUNCTION integration.abandoned_connection_immutable();
REVOKE ALL ON FUNCTION integration.abandon_connection(uuid,uuid,uuid,uuid),integration.abandoned_connection_immutable() FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.abandon_connection(uuid,uuid,uuid,uuid) TO bookkeeping_api;
COMMIT;
