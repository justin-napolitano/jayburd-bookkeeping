-- Internal, synchronous command recovery; no ISO exchange, worker or new accounting policy.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE ledger.interactive_command (
 command_id uuid PRIMARY KEY, principal_id uuid NOT NULL REFERENCES ledger.principal,
 registration_key text NOT NULL CHECK(length(registration_key) BETWEEN 1 AND 128 AND registration_key ~ '^[A-Za-z0-9._:-]+$'),
 operation text NOT NULL CHECK(operation IN ('setup','post')), entity_id uuid, book_id uuid,
 request_body jsonb NOT NULL CHECK(jsonb_typeof(request_body)='object'),
 state text NOT NULL DEFAULT 'prepared' CHECK(state IN ('prepared','committed','cancelled')),
 result jsonb, result_entity_id uuid, result_book_id uuid,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), terminal_at timestamptz, acknowledged_at timestamptz,
 UNIQUE(principal_id,registration_key), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book MATCH FULL,
 FOREIGN KEY(result_entity_id,result_book_id) REFERENCES ledger.book MATCH FULL,
 CHECK((operation='setup' AND entity_id IS NULL AND book_id IS NULL) OR (operation='post' AND entity_id IS NOT NULL AND book_id IS NOT NULL)),
 CHECK((state='prepared' AND terminal_at IS NULL AND acknowledged_at IS NULL) OR (state IN ('committed','cancelled') AND terminal_at IS NOT NULL)),
 CHECK((state='committed' AND result IS NOT NULL AND result_entity_id IS NOT NULL AND result_book_id IS NOT NULL) OR
       (state<>'committed' AND result IS NULL AND result_entity_id IS NULL AND result_book_id IS NULL))
);
CREATE UNIQUE INDEX interactive_command_pending ON ledger.interactive_command(principal_id) WHERE acknowledged_at IS NULL;
CREATE TABLE audit.command_action (
 action_id uuid PRIMARY KEY DEFAULT gen_random_uuid(), command_id uuid NOT NULL REFERENCES ledger.interactive_command,
 principal_id uuid NOT NULL REFERENCES ledger.principal, action text NOT NULL CHECK(action IN ('register','execute','cancel','acknowledge')),
 request_id uuid NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TRIGGER immutable_command_audit BEFORE UPDATE OR DELETE ON audit.command_action FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Preserve every reviewed payload and terminal result while permitting only owned state transitions.
CREATE FUNCTION ledger.command_change() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM ledger.fail('P2113'); END IF;
 IF ROW(NEW.command_id,NEW.principal_id,NEW.registration_key,NEW.operation,NEW.entity_id,NEW.book_id,NEW.request_body,NEW.created_at)
 IS DISTINCT FROM ROW(OLD.command_id,OLD.principal_id,OLD.registration_key,OLD.operation,OLD.entity_id,OLD.book_id,OLD.request_body,OLD.created_at)
 THEN PERFORM ledger.fail('P2113'); END IF;
 IF OLD.state='prepared' THEN
  IF NEW.state NOT IN ('committed','cancelled') OR NEW.acknowledged_at IS NOT NULL THEN PERFORM ledger.fail('P2113'); END IF;
 ELSE
  IF ROW(NEW.state,NEW.result,NEW.result_entity_id,NEW.result_book_id,NEW.terminal_at)
  IS DISTINCT FROM ROW(OLD.state,OLD.result,OLD.result_entity_id,OLD.result_book_id,OLD.terminal_at)
  OR OLD.acknowledged_at IS NOT NULL OR NEW.acknowledged_at IS NULL THEN PERFORM ledger.fail('P2113'); END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER preserve_command BEFORE UPDATE OR DELETE ON ledger.interactive_command FOR EACH ROW EXECUTE FUNCTION ledger.command_change();
-- Validate transport shape without dry-run posting or copying accounting calculations.
-- Current references, policy, revision, valuation and balancing remain execute-time authority.
CREATE FUNCTION ledger.validate_command_body(op text,v jsonb) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE a jsonb; code_value text; basis_value text; first_day date; last_day date;
BEGIN
 IF op='setup' THEN
  PERFORM ledger.keys(v,ARRAY['entity_name','entity_kind','book_name','starts_on','ends_before','functional_currency_version_id','accounts']);
  PERFORM ledger.setup_text(v->'entity_name',200); PERFORM ledger.setup_text(v->'book_name',200);
  IF ledger.text_value(v->'entity_kind') NOT IN ('legal','personal') OR jsonb_typeof(v->'accounts') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(v->'accounts') NOT BETWEEN 2 AND 100 THEN PERFORM ledger.fail('P2100'); END IF;
  first_day:=ledger.date_value(ledger.text_value(v->'starts_on')); last_day:=ledger.date_value(ledger.text_value(v->'ends_before'));
  IF first_day>=last_day THEN PERFORM ledger.fail('P2100'); END IF;
  PERFORM ledger.uuid_value(v->'functional_currency_version_id');
  FOR a IN SELECT value FROM jsonb_array_elements(v->'accounts') LOOP
   PERFORM ledger.keys(a,ARRAY['code','name','category','normal_side','monetary']);
   PERFORM ledger.setup_text(a->'code',64); PERFORM ledger.setup_text(a->'name',200);
   IF ledger.text_value(a->'category') NOT IN ('asset','liability','equity','revenue','expense') OR ledger.text_value(a->'normal_side') NOT IN ('debit','credit') OR jsonb_typeof(a->'monetary') IS DISTINCT FROM 'boolean' THEN PERFORM ledger.fail('P2100'); END IF;
  END LOOP;
  IF (SELECT count(DISTINCT value->>'code') FROM jsonb_array_elements(v->'accounts'))<>jsonb_array_length(v->'accounts') THEN PERFORM ledger.fail('P2100'); END IF;
 ELSIF op='post' THEN
  PERFORM ledger.keys(v,ARRAY['expected_revision','policy_id','accounting_date','description','lines']);
  code_value:=ledger.text_value(v->'expected_revision');
  IF code_value !~ '^(0|[1-9][0-9]{0,18})$' OR code_value::numeric>9223372036854775807 THEN PERFORM ledger.fail('P2100'); END IF;
  PERFORM ledger.uuid_value(v->'policy_id'); PERFORM ledger.date_value(ledger.text_value(v->'accounting_date'));
  code_value:=ledger.text_value(v->'description'); IF length(code_value) NOT BETWEEN 1 AND 500 OR jsonb_typeof(v->'lines') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(v->'lines') NOT BETWEEN 2 AND 1000 THEN PERFORM ledger.fail('P2108'); END IF;
  FOR a IN SELECT value FROM jsonb_array_elements(v->'lines') LOOP
   PERFORM ledger.keys(a,ARRAY['account_id','side','transaction_currency','transaction_amount','functional_amount','valuation_basis'],ARRAY['rate_observation_id']);
   PERFORM ledger.uuid_value(a->'account_id'); PERFORM ledger.money_value(a->'transaction_amount'); PERFORM ledger.money_value(a->'functional_amount');
   basis_value:=ledger.text_value(a->'valuation_basis');
   IF ledger.text_value(a->'side') NOT IN ('debit','credit') OR basis_value NOT IN ('identity','spot') OR ledger.text_value(a->'transaction_currency') !~ '^[A-Z]{3}$' THEN PERFORM ledger.fail('P2108'); END IF;
   IF (basis_value='identity' AND a?'rate_observation_id') OR (basis_value='spot' AND NOT a?'rate_observation_id') THEN PERFORM ledger.fail('P2108'); END IF;
   IF basis_value='spot' THEN PERFORM ledger.uuid_value(a->'rate_observation_id'); END IF;
  END LOOP;
 ELSE PERFORM ledger.fail('P2100'); END IF;
END $$;
-- All recovery operations take the strongest principal lock first, matching setup.
CREATE FUNCTION ledger.command_principal(actor uuid) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF actor IS NULL OR current_setting('transaction_isolation')<>'serializable' THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
END $$;
-- Disclosure requires both current read and post grants, including committed setup results.
CREATE FUNCTION ledger.command_authorize(actor uuid,c ledger.interactive_command) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE e uuid; b uuid;
BEGIN
 IF c.principal_id IS DISTINCT FROM actor THEN PERFORM ledger.fail('P2101'); END IF;
 IF c.operation='post' THEN e:=c.entity_id; b:=c.book_id;
 ELSIF c.state='committed' THEN e:=c.result_entity_id; b:=c.result_book_id; END IF;
 IF e IS NOT NULL THEN
  PERFORM ledger.authorize(actor,e,b,'read',true);
  PERFORM ledger.authorize(actor,e,b,'post',true);
 END IF;
END $$;
-- Return only contract fields; internal registration keys and audit identities stay private.
CREATE FUNCTION ledger.render_command(c ledger.interactive_command) RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog AS $$
 SELECT jsonb_build_object('command_id',c.command_id,'operation',c.operation,'entity_id',c.entity_id,'book_id',c.book_id,
 'state',c.state,'body',c.request_body,'result',c.result,'created_at',c.created_at,'acknowledged',c.acknowledged_at IS NOT NULL)
$$;
-- Registration commits only reviewed intent; no accounting authority runs here.
CREATE FUNCTION ledger.register_command(actor uuid,k text,request uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; op text; e uuid; b uuid; c ledger.interactive_command;
BEGIN
 IF raw IS NULL OR k IS NULL OR request IS NULL THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF length(k) NOT BETWEEN 1 AND 128 OR k !~ '^[A-Za-z0-9._:-]+$' OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN v:=raw::jsonb; EXCEPTION WHEN untranslatable_character THEN PERFORM ledger.fail('P2100'); END;
 -- Envelope targets deliberately permit JSON null; ordinary ledger.keys forbids it.
 IF NOT(v ?& ARRAY['operation','entity_id','book_id','body']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['operation','entity_id','book_id','body'])) THEN PERFORM ledger.fail('P2100'); END IF;
 op:=ledger.text_value(v->'operation');
 IF op='setup' THEN IF v->'entity_id'<>'null'::jsonb OR v->'book_id'<>'null'::jsonb THEN PERFORM ledger.fail('P2100'); END IF;
 ELSIF op='post' THEN e:=ledger.uuid_value(v->'entity_id'); b:=ledger.uuid_value(v->'book_id');
 ELSE PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.validate_command_body(op,v->'body');
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE principal_id=actor AND registration_key=k;
 IF FOUND THEN
  PERFORM ledger.command_authorize(actor,c);
  IF ROW(c.operation,c.entity_id,c.book_id,c.request_body) IS DISTINCT FROM ROW(op,e,b,v->'body') THEN PERFORM ledger.fail('P2104'); END IF;
  RETURN ledger.render_command(c);
 END IF;
 -- Inaccessible pending work still blocks new intent, without exposing its payload.
 IF EXISTS(SELECT FROM ledger.interactive_command WHERE principal_id=actor AND acknowledged_at IS NULL) THEN PERFORM ledger.fail('P2112'); END IF;
 c.principal_id:=actor; c.operation:=op; c.entity_id:=e; c.book_id:=b;
 PERFORM ledger.command_authorize(actor,c);
 INSERT INTO ledger.interactive_command(command_id,principal_id,registration_key,operation,entity_id,book_id,request_body)
 VALUES(gen_random_uuid(),actor,k,op,e,b,v->'body') RETURNING * INTO c;
 INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(c.command_id,actor,'register',request);
 RETURN ledger.render_command(c);
EXCEPTION WHEN unique_violation THEN
 -- A stale serializable snapshot after waiting on the unchanged principal can
 -- discover the competing insert at its unique index; retry with a fresh snapshot.
 RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_command_registration';
END $$;
-- Discovery locks and reauthorizes but never executes or acknowledges intent.
CREATE FUNCTION ledger.pending_command(actor uuid) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE c ledger.interactive_command;
BEGIN
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE principal_id=actor AND acknowledged_at IS NULL;
 IF NOT FOUND THEN RETURN jsonb_build_object('command',NULL); END IF;
 PERFORM ledger.command_authorize(actor,c);
 RETURN jsonb_build_object('command',ledger.render_command(c));
END $$;
-- Execution, cancellation and acknowledgement use one principal/scope/command order.
-- Existing accounting receipt + journal + audit + outbox commit with command completion.
CREATE FUNCTION ledger.transition_command(actor uuid,id uuid,request uuid,action_name text,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE c ledger.interactive_command; posted record; value jsonb;
BEGIN
 IF id IS NULL OR request IS NULL OR raw IS NULL OR action_name IS NULL OR action_name NOT IN ('execute','cancel','acknowledge') THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN value:=raw::jsonb; EXCEPTION WHEN untranslatable_character THEN PERFORM ledger.fail('P2100'); END;
 IF value<>'{}'::jsonb THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE command_id=id AND principal_id=actor;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 PERFORM ledger.command_authorize(actor,c);
 PERFORM 1 FROM ledger.interactive_command WHERE command_id=id FOR UPDATE;
 IF action_name='execute' THEN
  IF c.state='cancelled' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.state='prepared' THEN
   IF c.operation='setup' THEN SELECT * INTO posted FROM ledger.setup_book(actor,'command:'||c.command_id,request,c.request_body::text);
   ELSE SELECT * INTO posted FROM ledger.post_journal(actor,c.entity_id,c.book_id,'command:'||c.command_id,request,c.request_body::text); END IF;
   value:=posted.body;
   UPDATE ledger.interactive_command SET state='committed',result=value,result_entity_id=(value->>'entity_id')::uuid,result_book_id=(value->>'book_id')::uuid,terminal_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'execute',request);
  END IF;
 ELSIF action_name='cancel' THEN
  IF c.state='committed' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.state='prepared' THEN
   UPDATE ledger.interactive_command SET state='cancelled',terminal_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'cancel',request);
  END IF;
 ELSE
  IF c.state='prepared' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.acknowledged_at IS NULL THEN
   UPDATE ledger.interactive_command SET acknowledged_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'acknowledge',request);
  END IF;
 END IF;
 RETURN ledger.render_command(c);
END $$;
GRANT EXECUTE ON FUNCTION ledger.register_command(uuid,text,uuid,text),ledger.pending_command(uuid),ledger.transition_command(uuid,uuid,uuid,text,text) TO bookkeeping_api;
COMMIT;
