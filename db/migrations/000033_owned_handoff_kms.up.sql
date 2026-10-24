-- Restore20's canonical KMS project grammar for32's owned repair handoff.
-- All ownership, scope, consent, deadline, immutable-winner and ACL semantics stay32.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE OR REPLACE FUNCTION integration.put_link_handoff(actor uuid,e uuid,b uuid,id uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a integration.connection_attempt;v jsonb;winner jsonb;
BEGIN
 SELECT * INTO a FROM integration.connection_attempt WHERE entity_id=e AND book_id=b AND attempt_id=id AND actor_id=actor;
 IF a.mode IS DISTINCT FROM 'reconnect' THEN RETURN integration.put_link_handoff_before_owned_repair32(actor,e,b,id,raw);END IF;
 a:=integration.owned_repair_lock(actor,e,b,id);PERFORM integration.owned_repair_scope(a);
 IF a.claim_id IS NOT NULL OR a.state<>'link_ready' OR a.link_token_sha256 IS NULL THEN PERFORM integration.connection_error('P2704');END IF;
 v:=integration.connection_json(raw,32768);
 PERFORM ledger.keys(v,ARRAY['profile','key_version','wrapped_key','nonce','ciphertext']);
 IF v->>'profile' IS DISTINCT FROM 'jayburd-hosted-link-aead-v1'
 OR jsonb_typeof(v->'key_version') IS DISTINCT FROM 'string' OR length(v->>'key_version') NOT BETWEEN 1 AND 512
 OR (v->>'key_version') !~ '^projects/([1-9][0-9]*|[a-z][a-z0-9-]{4,28}[a-z0-9])/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/[1-9][0-9]*$'
 OR jsonb_typeof(v->'wrapped_key') IS DISTINCT FROM 'string' OR jsonb_typeof(v->'nonce') IS DISTINCT FROM 'string'
 OR jsonb_typeof(v->'ciphertext') IS DISTINCT FROM 'string'
 OR octet_length(decode(v->>'wrapped_key','base64')) NOT BETWEEN 1 AND 8192
 OR octet_length(decode(v->>'nonce','base64'))<>12
 OR octet_length(decode(v->>'ciphertext','base64')) NOT BETWEEN 17 AND 4112
 THEN PERFORM integration.connection_error('P2700');END IF;
 INSERT INTO integration.link_handoff(entity_id,book_id,attempt_id,envelope) VALUES(e,b,id,v) ON CONFLICT(entity_id,book_id,attempt_id) DO NOTHING;
 SELECT envelope INTO winner FROM integration.link_handoff WHERE entity_id=e AND book_id=b AND attempt_id=id;
 RETURN jsonb_build_object('envelope',winner);
EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE 'P2100' THEN PERFORM integration.connection_error('P2700');END $$;
COMMIT;
