-- Restore numeric-only validation only before canonical envelopes exist.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;

-- Do not restore numeric-only writers after canonical envelopes have been stored.
DO $$ BEGIN
 IF EXISTS(SELECT FROM integration.bank_credential WHERE envelope->>'key_version' !~ '^projects/[0-9]+/')
 OR EXISTS(SELECT FROM integration.link_handoff WHERE envelope->>'key_version' !~ '^projects/[0-9]+/')
 THEN RAISE EXCEPTION 'canonical KMS envelopes require migration 20';END IF;
END $$;

CREATE OR REPLACE FUNCTION integration.put_bank_credential(actor uuid,e uuid,b uuid,r uuid,v bigint,item text,env text,raw text) RETURNS jsonb
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

CREATE OR REPLACE FUNCTION integration.put_link_handoff(actor uuid,e uuid,b uuid,id uuid,raw text) RETURNS jsonb
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
COMMIT;
