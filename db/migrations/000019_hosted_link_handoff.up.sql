-- Retain a short-lived encrypted Hosted Link token so completion survives browser reloads.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;

ALTER TABLE integration.connection_attempt ALTER COLUMN expires_at SET DEFAULT clock_timestamp()+interval '7 hours';

CREATE TABLE integration.link_handoff (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, attempt_id uuid NOT NULL,
 envelope jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,attempt_id),
 FOREIGN KEY(entity_id,book_id,attempt_id) REFERENCES integration.connection_attempt,
 CHECK(octet_length(envelope::text)<=32768)
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.link_handoff
 FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
REVOKE ALL ON integration.link_handoff FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;

-- The existing attempt lock rechecks actor, book, worker generation and current grants.
CREATE FUNCTION integration.get_link_handoff(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.mode<>'new' OR a.link_token_sha256 IS NULL THEN PERFORM integration.connection_error('P2701');END IF;
 SELECT envelope INTO v FROM integration.link_handoff WHERE entity_id=e AND book_id=b AND attempt_id=id;
 RETURN jsonb_build_object('envelope',v);
END $$;

-- Concurrent writes return the immutable winner; Go decrypts and compares before handoff.
CREATE FUNCTION integration.put_link_handoff(actor uuid,e uuid,b uuid,id uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;winner jsonb;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);
 IF a.mode<>'new' OR a.state<>'link_ready' OR a.claim_id IS NOT NULL OR a.link_token_sha256 IS NULL OR clock_timestamp()>=a.expires_at THEN PERFORM integration.connection_error('P2704');END IF;
 v:=integration.connection_json(raw,32768);
 PERFORM ledger.keys(v,ARRAY['profile','key_version','wrapped_key','nonce','ciphertext']);
 IF v->>'profile' IS DISTINCT FROM 'jayburd-hosted-link-aead-v1'
 OR jsonb_typeof(v->'key_version') IS DISTINCT FROM 'string'
 OR length(v->>'key_version') NOT BETWEEN 1 AND 512
 OR (v->>'key_version') !~ '^projects/[0-9]+/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$'
 OR jsonb_typeof(v->'wrapped_key') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v->'nonce') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v->'ciphertext') IS DISTINCT FROM 'string'
 OR octet_length(decode(v->>'wrapped_key','base64')) NOT BETWEEN 1 AND 8192
 OR octet_length(decode(v->>'nonce','base64'))<>12
 OR octet_length(decode(v->>'ciphertext','base64')) NOT BETWEEN 17 AND 4112
 THEN PERFORM integration.connection_error('P2700');END IF;
 INSERT INTO integration.link_handoff(entity_id,book_id,attempt_id,envelope) VALUES(e,b,id,v)
 ON CONFLICT(entity_id,book_id,attempt_id) DO NOTHING;
 SELECT envelope INTO winner FROM integration.link_handoff WHERE entity_id=e AND book_id=b AND attempt_id=id;
 RETURN jsonb_build_object('envelope',winner);
EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');END $$;

-- Plaid keeps completed Hosted Link results for up to six hours after session end.
-- The browser handoff still uses Plaid's URL expiry; the attempt stays claimable for recovery.
CREATE OR REPLACE FUNCTION integration.record_link_session(actor uuid,e uuid,b uuid,id uuid,nonce_value uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;expiry timestamptz;hash_value text;
BEGIN
 a:=integration.connection_lock(actor,e,b,id);v:=integration.connection_json(raw,2048);PERFORM ledger.keys(v,ARRAY['link_token_sha256','expires_at']);hash_value:=ledger.text_value(v->'link_token_sha256');
 IF hash_value !~ '^[0-9a-f]{64}$' OR ledger.text_value(v->'expires_at') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$' THEN PERFORM integration.connection_error('P2700');END IF;expiry:=integration.connection_expiry(v->'expires_at');
 IF a.nonce IS DISTINCT FROM nonce_value OR a.mode NOT IN ('new','reconnect') OR clock_timestamp()>=a.expires_at OR expiry<=clock_timestamp() THEN PERFORM integration.connection_error('P2703');END IF;
 IF a.link_token_sha256 IS NOT NULL THEN IF a.link_token_sha256<>hash_value THEN PERFORM integration.connection_error('P2702');END IF;RETURN integration.render_connection(a,true);END IF;
 IF a.state<>'link_pending' THEN PERFORM integration.connection_error('P2704');END IF;
 UPDATE integration.connection_attempt SET link_token_sha256=hash_value,expires_at=least(expires_at,expiry+interval '6 hours'),state='link_ready' WHERE entity_id=e AND book_id=b AND attempt_id=id RETURNING * INTO a;
 INSERT INTO integration.connection_event(entity_id,book_id,attempt_id,kind,body) VALUES(e,b,id,'link_ready',v);RETURN integration.render_connection(a,true);
EXCEPTION WHEN SQLSTATE 'P2100' OR invalid_datetime_format OR datetime_field_overflow THEN PERFORM integration.connection_error('P2700');END $$;

REVOKE ALL ON FUNCTION integration.get_link_handoff(uuid,uuid,uuid,uuid),integration.put_link_handoff(uuid,uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION integration.get_link_handoff(uuid,uuid,uuid,uuid),integration.put_link_handoff(uuid,uuid,uuid,uuid,text) TO bookkeeping_api;
COMMIT;
