-- Explicitly release only owner-held commands whose current scope is inaccessible.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;

-- This operation reveals no saved payload or scope and never executes accounting work.
CREATE FUNCTION ledger.clear_inaccessible_command(actor uuid,request uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE c ledger.interactive_command; authorized boolean:=false;
BEGIN
 IF actor IS NULL OR request IS NULL OR raw IS NULL OR current_setting('transaction_isolation')<>'serializable'
 OR octet_length(raw)>1024 OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 -- Cast only after grammar validation so malformed JSON maps to the owned error contract.
 IF raw::jsonb<>'{}'::jsonb THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE principal_id=actor AND acknowledged_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN RETURN jsonb_build_object('cleared',false); END IF;
 -- Only the existing scoped authorization failures qualify. Unexpected failures remain fatal.
 BEGIN
  PERFORM ledger.command_authorize(actor,c);
  authorized:=true;
 EXCEPTION WHEN SQLSTATE 'P2101' OR SQLSTATE 'P2102' THEN
  authorized:=false;
 END;
 IF authorized THEN PERFORM ledger.fail('P2113'); END IF;
 IF c.state='prepared' THEN
  UPDATE ledger.interactive_command SET state='cancelled',terminal_at=clock_timestamp() WHERE command_id=c.command_id RETURNING * INTO c;
  INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(c.command_id,actor,'cancel',request);
 END IF;
 UPDATE ledger.interactive_command SET acknowledged_at=clock_timestamp() WHERE command_id=c.command_id RETURNING * INTO c;
 INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(c.command_id,actor,'acknowledge',request);
 RETURN jsonb_build_object('cleared',true);
END $$;

REVOKE ALL ON FUNCTION ledger.clear_inaccessible_command(uuid,uuid,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION ledger.clear_inaccessible_command(uuid,uuid,text) TO bookkeeping_api;
COMMIT;
