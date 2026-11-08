-- #460: explicit authenticated own-workspace setup and audited legacy adoption.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE ledger.interactive_command_reservation,banking.collection_workspace IN SHARE ROW EXCLUSIVE MODE;
CREATE TABLE banking.collection_onboarding_preview(
 preview_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),principal_id uuid NOT NULL REFERENCES ledger.principal,
 operation text NOT NULL CHECK(operation IN('setup','adopt')),workspace_id uuid REFERENCES banking.collection_workspace,
 snapshot text NOT NULL CHECK(snapshot COLLATE "C" ~ '^[0-9a-f]{64}$'),review jsonb NOT NULL CHECK(jsonb_typeof(review)='object'AND octet_length(review::text)<=1030000),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(created_at)),expires_at timestamptz NOT NULL CHECK(isfinite(expires_at)AND expires_at>created_at),
 UNIQUE(principal_id,preview_id),CHECK((operation='setup'AND workspace_id IS NULL)OR(operation='adopt'AND workspace_id IS NOT NULL))
);
CREATE TABLE banking.collection_onboarding_command(
 command_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),recovery_id uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),principal_id uuid NOT NULL REFERENCES ledger.principal,preview_id uuid NOT NULL,
 prepare_key text NOT NULL CHECK(prepare_key COLLATE "C" ~ '^[A-Za-z0-9._:-]{1,128}$'),state text NOT NULL DEFAULT 'prepared'CHECK(state IN('prepared','committed','cancelled')),
 result jsonb,created_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(created_at)),terminal_at timestamptz CHECK(isfinite(terminal_at)),acknowledged_at timestamptz CHECK(isfinite(acknowledged_at)),
 UNIQUE(principal_id,prepare_key),UNIQUE(principal_id,recovery_id,command_id),FOREIGN KEY(principal_id,preview_id)REFERENCES banking.collection_onboarding_preview(principal_id,preview_id),
 CHECK((state='prepared'AND result IS NULL AND terminal_at IS NULL AND acknowledged_at IS NULL)OR(state IN('committed','cancelled')AND result IS NOT NULL AND jsonb_typeof(result)='object'AND terminal_at IS NOT NULL)),CHECK(result IS NULL OR octet_length(result::text)<=8192)
);
CREATE TABLE audit.collection_onboarding_action(
 action_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),principal_id uuid NOT NULL REFERENCES ledger.principal,
 operation text NOT NULL CHECK(operation IN('preview','prepare','execute','cancel','acknowledge')),preview_id uuid NOT NULL REFERENCES banking.collection_onboarding_preview,command_id uuid REFERENCES banking.collection_onboarding_command,
 database_actor name NOT NULL DEFAULT session_user,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at))
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_onboarding_preview FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON audit.collection_onboarding_action FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE FUNCTION banking.collection_onboarding_guard()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF TG_OP='DELETE'OR(to_jsonb(NEW)-ARRAY['state','result','terminal_at','acknowledged_at'])IS DISTINCT FROM(to_jsonb(OLD)-ARRAY['state','result','terminal_at','acknowledged_at'])THEN PERFORM banking.collection_error('P2802');END IF;
 IF OLD.state='prepared'THEN IF NEW.state NOT IN('committed','cancelled')OR NEW.acknowledged_at IS NOT NULL THEN PERFORM banking.collection_error('P2802');END IF;
 ELSE IF ROW(NEW.state,NEW.result,NEW.terminal_at)IS DISTINCT FROM ROW(OLD.state,OLD.result,OLD.terminal_at)OR OLD.acknowledged_at IS NOT NULL OR NEW.acknowledged_at IS NULL THEN PERFORM banking.collection_error('P2802');END IF;END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_onboarding BEFORE UPDATE OR DELETE ON banking.collection_onboarding_command FOR EACH ROW EXECUTE FUNCTION banking.collection_onboarding_guard();
ALTER TABLE ledger.interactive_command_reservation ADD COLUMN onboarding_command_id uuid UNIQUE REFERENCES banking.collection_onboarding_command DEFERRABLE INITIALLY DEFERRED;
-- Preserve engine ownership on closure proof, so another clear endpoint cannot
-- misclassify an acknowledged onboarding command as its own historical work.
ALTER TABLE ledger.collection_recovery_receipt ADD COLUMN onboarding_command_id uuid REFERENCES banking.collection_onboarding_command;
ALTER TABLE ledger.interactive_command_reservation ADD CONSTRAINT onboarding_claim_identity FOREIGN KEY(principal_id,recovery_id,onboarding_command_id)REFERENCES banking.collection_onboarding_command(principal_id,recovery_id,command_id)DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE ledger.collection_recovery_receipt ADD CONSTRAINT onboarding_receipt_identity FOREIGN KEY(principal_id,recovery_id,onboarding_command_id)REFERENCES banking.collection_onboarding_command(principal_id,recovery_id,command_id);
ALTER TABLE ledger.interactive_command_reservation DROP CONSTRAINT interactive_command_reservation_check;
ALTER TABLE ledger.interactive_command_reservation ADD CONSTRAINT interactive_command_reservation_check CHECK(
 (legacy_command_id IS NOT NULL AND workspace_id IS NULL AND label_command_id IS NULL AND onboarding_command_id IS NULL)OR
 (legacy_command_id IS NULL AND workspace_id IS NOT NULL AND label_command_id IS NOT NULL AND onboarding_command_id IS NULL)OR
 (legacy_command_id IS NULL AND workspace_id IS NULL AND label_command_id IS NULL AND onboarding_command_id IS NOT NULL));
-- Dispatch exact typed engines; claim last and release only an explicitly acknowledged exact command.
CREATE OR REPLACE FUNCTION banking.collection_label_reserve()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE old_id uuid;private_id uuid;onboard_id uuid;work uuid;accepted uuid;claim_id uuid:=gen_random_uuid();legacy boolean:=TG_TABLE_SCHEMA='ledger';BEGIN
 IF TG_TABLE_NAME='collection_onboarding_command'THEN onboard_id:=NEW.command_id;claim_id:=NEW.recovery_id;ELSIF legacy THEN old_id:=NEW.command_id;ELSE private_id:=NEW.command_id;work:=NEW.workspace_id;END IF;
 IF TG_OP='INSERT'THEN
  IF NEW.acknowledged_at IS NOT NULL THEN RETURN NEW;END IF;
  INSERT INTO ledger.interactive_command_reservation(principal_id,recovery_id,legacy_command_id,workspace_id,label_command_id,onboarding_command_id)VALUES(NEW.principal_id,claim_id,old_id,work,private_id,onboard_id)ON CONFLICT(principal_id)DO NOTHING RETURNING principal_id INTO accepted;
  IF accepted IS NULL THEN IF legacy THEN PERFORM ledger.fail('P2112');ELSE PERFORM banking.collection_error('P2802');END IF;END IF;
 ELSIF OLD.acknowledged_at IS NULL AND NEW.acknowledged_at IS NOT NULL THEN
  DELETE FROM ledger.interactive_command_reservation WHERE principal_id=NEW.principal_id AND legacy_command_id IS NOT DISTINCT FROM old_id AND workspace_id IS NOT DISTINCT FROM work AND label_command_id IS NOT DISTINCT FROM private_id AND onboarding_command_id IS NOT DISTINCT FROM onboard_id RETURNING principal_id INTO accepted;
  IF accepted IS NULL THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_command_acknowledgement';END IF;
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER onboarding_reserve BEFORE INSERT ON banking.collection_onboarding_command FOR EACH ROW EXECUTE FUNCTION banking.collection_label_reserve();
CREATE TRIGGER onboarding_release AFTER UPDATE ON banking.collection_onboarding_command FOR EACH ROW EXECUTE FUNCTION banking.collection_label_reserve();
CREATE OR REPLACE FUNCTION banking.collection_recovery_fence()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE subject uuid;observed uuid;BEGIN
 subject:=CASE WHEN TG_OP='DELETE'THEN OLD.principal_id ELSE NEW.principal_id END;
 IF TG_OP='DELETE'AND OLD.legacy_command_id IS NULL THEN
  IF OLD.onboarding_command_id IS NOT NULL THEN
   IF NOT EXISTS(SELECT FROM banking.collection_onboarding_command WHERE command_id=OLD.onboarding_command_id AND principal_id=OLD.principal_id AND acknowledged_at IS NOT NULL)THEN PERFORM banking.collection_error('P2802');END IF;
  ELSIF NOT EXISTS(SELECT FROM banking.collection_label_command WHERE workspace_id=OLD.workspace_id AND command_id=OLD.label_command_id AND principal_id=OLD.principal_id AND acknowledged_at IS NOT NULL)THEN PERFORM banking.collection_error('P2802');END IF;
  INSERT INTO ledger.collection_recovery_receipt(principal_id,recovery_id,onboarding_command_id)VALUES(OLD.principal_id,OLD.recovery_id,OLD.onboarding_command_id)ON CONFLICT DO NOTHING;
 END IF;
 UPDATE ledger.collection_recovery_state SET revision=revision+1 WHERE principal_id=subject RETURNING principal_id INTO observed;
 IF observed IS NULL THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_recovery';END IF;RETURN NULL;
END $$;
-- Workspace existence/capability changes fence global discovery even before any command exists.
CREATE FUNCTION banking.collection_onboarding_workspace_fence()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 UPDATE ledger.collection_recovery_state SET revision=revision+1 WHERE principal_id=NEW.owner_principal_id;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_onboarding';END IF;RETURN NULL;
END $$;
CREATE TRIGGER onboarding_workspace_fence AFTER INSERT OR UPDATE ON banking.collection_workspace FOR EACH ROW EXECUTE FUNCTION banking.collection_onboarding_workspace_fence();

-- Safe complete source metadata under original book/source/consent authority; never provider identifiers.
CREATE FUNCTION banking.collection_onboarding_candidate(actor uuid,id uuid,eligible boolean)RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE ci integration.collection_item;i integration.sync_item;c banking.collection_workspace;s banking.source_account;t record;b ledger.book;e ledger.entity;sources jsonb:='[]';pins jsonb:='[]';res jsonb;digest text;BEGIN
 SELECT * INTO ci FROM integration.collection_item WHERE collection_item_id=id AND legacy_sync_item_id IS NOT NULL;IF NOT FOUND OR ci.state='revoked'THEN PERFORM banking.collection_error('P2801');END IF;
 PERFORM ledger.authorize(actor,ci.legacy_entity_id,ci.legacy_book_id,'read',true);
 PERFORM 1 FROM ledger.membership WHERE entity_id=ci.legacy_entity_id AND principal_id=actor AND permission='connect'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 PERFORM 1 FROM ledger.membership WHERE entity_id=ci.legacy_entity_id AND principal_id=actor AND permission='import'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 c:=banking.collection_authorize(actor,NULL,'acquire',true);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=ci.legacy_entity_id AND book_id=ci.legacy_book_id AND sync_item_id=ci.legacy_sync_item_id FOR UPDATE;
 IF NOT FOUND OR i.consent_principal<>actor OR(ci.workspace_id IS NOT NULL AND ci.workspace_id<>c.workspace_id)THEN PERFORM banking.collection_error('P2801');END IF;
 PERFORM 1 FROM integration.collection_item WHERE collection_item_id=id FOR UPDATE;
 IF eligible AND(ci.workspace_id IS NOT NULL OR i.state NOT IN('active','paused')OR EXISTS(SELECT FROM integration.sync_set ss JOIN integration.sync_stream st USING(entity_id,book_id,stream_id)WHERE st.entity_id=i.entity_id AND st.book_id=i.book_id AND st.sync_item_id=i.sync_item_id AND ss.state='staging')OR EXISTS(SELECT FROM integration.connection_attempt a WHERE a.entity_id=i.entity_id AND a.book_id=i.book_id AND a.sync_item_id=i.sync_item_id AND a.state NOT IN('active','disconnected','abandoned')))THEN PERFORM banking.collection_error('P2804');END IF;
 SELECT * INTO b FROM ledger.book WHERE entity_id=i.entity_id AND book_id=i.book_id;SELECT * INTO e FROM ledger.entity WHERE entity_id=i.entity_id;
 FOR t IN SELECT st.*,a.name display_name FROM integration.sync_stream st JOIN banking.source_account sa USING(entity_id,book_id,source_account_id)JOIN ledger.account a ON a.entity_id=sa.entity_id AND a.book_id=sa.book_id AND a.account_id=sa.ledger_account_id WHERE st.entity_id=i.entity_id AND st.book_id=i.book_id AND st.sync_item_id=i.sync_item_id AND st.state='active'ORDER BY st.source_account_id LOOP
  s:=banking.import_authorize(actor,i.entity_id,i.book_id,t.source_account_id,false);
  PERFORM 1 FROM banking.source_permission WHERE entity_id=i.entity_id AND book_id=i.book_id AND source_account_id=t.source_account_id AND principal_id=actor AND capability='import'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
  IF ci.workspace_id IS NOT NULL AND NOT EXISTS(SELECT FROM banking.collection_source WHERE workspace_id=c.workspace_id AND collection_item_id=id AND legacy_source_account_id=t.source_account_id AND state='active')THEN PERFORM banking.collection_error('P2801');END IF;
  sources:=sources||jsonb_build_array(jsonb_build_object('source_id',t.source_account_id,'display_name',left(t.display_name,200)));
  pins:=pins||jsonb_build_array(jsonb_build_array(t.stream_id,t.source_account_id,s.binding_revision::text));
 END LOOP;
 IF jsonb_array_length(sources)=0 THEN PERFORM banking.collection_error('P2804');END IF;IF jsonb_array_length(sources)>25 THEN PERFORM banking.collection_error('P2805');END IF;
 IF eligible AND(SELECT count(*)FROM banking.collection_source WHERE workspace_id=c.workspace_id)+jsonb_array_length(sources)>100 THEN PERFORM banking.collection_error('P2805');END IF;
 res:=jsonb_build_object('candidate_id',id,'entity_id',i.entity_id,'book_id',i.book_id,'entity_name',left(e.name,200),'book_name',left(b.name,200),'sources',sources,'expected_generation',i.generation::text,'workspace_id',c.workspace_id);
 digest:=encode(sha256(convert_to(jsonb_build_array(res,c.revision::text,b.revision::text,i.state,ci.state,ci.workspace_id,pins)::text,'UTF8')),'hex');RETURN res||jsonb_build_object('snapshot',digest);
EXCEPTION WHEN SQLSTATE 'P2101'OR SQLSTATE 'P2102'OR SQLSTATE 'P2401'THEN PERFORM banking.collection_error('P2801');WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.collection_onboarding_authorize(actor uuid,p banking.collection_onboarding_preview)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$DECLARE c banking.collection_workspace;ci integration.collection_item;i integration.sync_item;a banking.collection_source;z jsonb;sid uuid;e uuid;b uuid;BEGIN
 IF p.principal_id IS DISTINCT FROM actor THEN PERFORM banking.collection_error('P2801');END IF;
 IF p.operation='setup'THEN
  SELECT * INTO c FROM banking.collection_workspace WHERE owner_principal_id=actor FOR SHARE;
  IF FOUND AND NOT ARRAY['read','acquire','classify']::text[]<@c.capabilities THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE
  e:=(p.review->>'entity_id')::uuid;b:=(p.review->>'book_id')::uuid;
  PERFORM ledger.authorize(actor,e,b,'read',true);
  PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='connect'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
  PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='import'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
  c:=banking.collection_authorize(actor,p.workspace_id,'acquire',true);
  SELECT * INTO ci FROM integration.collection_item WHERE collection_item_id=(p.review->>'candidate_id')::uuid;
  IF NOT FOUND OR ci.legacy_entity_id<>e OR ci.legacy_book_id<>b THEN PERFORM banking.collection_error('P2801');END IF;
  -- Current authority must survive a primed snapshot, including when generation
  -- intentionally advanced at execute. Preserve legacy-Item then canonical order.
  SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=ci.legacy_sync_item_id FOR SHARE;
  IF NOT FOUND OR i.consent_principal<>actor THEN PERFORM banking.collection_error('P2801');END IF;
  SELECT * INTO ci FROM integration.collection_item WHERE collection_item_id=ci.collection_item_id FOR SHARE;
  IF NOT FOUND OR ci.state='revoked'OR(ci.workspace_id IS NOT NULL AND ci.workspace_id<>p.workspace_id)THEN PERFORM banking.collection_error('P2801');END IF;
  -- Disclosure pins the ORIGINAL reviewed scopes, not a newly changed active set.
  -- Changed generation/stream sets may stale execution but cannot strand cancel/ack.
  FOR z IN SELECT value FROM jsonb_array_elements(p.review->'sources')LOOP
   sid:=(z->>'source_id')::uuid;PERFORM banking.import_authorize(actor,e,b,sid,false);
   PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=actor AND capability='import'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
   IF ci.workspace_id IS NOT NULL THEN
    SELECT * INTO a FROM banking.collection_source WHERE workspace_id=p.workspace_id AND collection_item_id=ci.collection_item_id AND legacy_source_account_id=sid FOR SHARE;
    IF NOT FOUND OR a.state<>'active'THEN PERFORM banking.collection_error('P2801');END IF;
   END IF;
  END LOOP;
 END IF;
EXCEPTION WHEN SQLSTATE 'P2101'OR SQLSTATE 'P2102'OR SQLSTATE 'P2401'THEN PERFORM banking.collection_error('P2801');WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.collection_onboarding_preview_wire(p banking.collection_onboarding_preview)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$SELECT jsonb_build_object('profile','private-onboarding-preview-v1','preview_id',p.preview_id,'operation',p.operation,'expires_at',to_char(p.expires_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'workspace_id',p.workspace_id,'snapshot',p.snapshot,'review',p.review)$$;
CREATE FUNCTION banking.collection_onboarding_command_wire(c banking.collection_onboarding_command)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$SELECT jsonb_build_object('profile','private-onboarding-command-v1','command_id',c.command_id,'recovery_id',c.recovery_id,'prepare_key',c.prepare_key,'state',c.state,'preview',banking.collection_onboarding_preview_wire(p),'result',c.result,'acknowledged',c.acknowledged_at IS NOT NULL)FROM banking.collection_onboarding_preview p WHERE p.preview_id=c.preview_id$$;

CREATE FUNCTION banking.get_collection_onboarding(actor uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_workspace;ws jsonb;items jsonb:='[]';z record;candidate jsonb;can_setup boolean:=false;can_adopt boolean:=false;res jsonb;BEGIN
 PERFORM banking.collection_recovery_principal(actor);SELECT * INTO c FROM banking.collection_workspace WHERE owner_principal_id=actor;
 IF NOT FOUND THEN ws:=jsonb_build_object('state','absent');can_setup:=true;
 ELSIF NOT 'read'=ANY(c.capabilities)THEN ws:=jsonb_build_object('state','unavailable');
 ELSE ws:=jsonb_build_object('state','ready','workspace_id',c.workspace_id,'authority_revision',c.revision::text,'capabilities',to_jsonb(c.capabilities));
  IF 'acquire'=ANY(c.capabilities)THEN
   FOR z IN SELECT ci.collection_item_id FROM integration.collection_item ci JOIN integration.sync_item i ON i.entity_id=ci.legacy_entity_id AND i.book_id=ci.legacy_book_id AND i.sync_item_id=ci.legacy_sync_item_id WHERE i.consent_principal=actor AND ci.workspace_id IS NULL AND ci.state<>'revoked'ORDER BY i.entity_id,i.book_id,ci.collection_item_id LOOP
    BEGIN candidate:=banking.collection_onboarding_candidate(actor,z.collection_item_id,true);items:=items||jsonb_build_array(candidate-ARRAY['workspace_id','expected_generation']);EXCEPTION WHEN SQLSTATE 'P2801'OR SQLSTATE 'P2804'THEN NULL;END;
    IF jsonb_array_length(items)>100 THEN PERFORM banking.collection_error('P2805');END IF;
   END LOOP;
   can_adopt:=jsonb_array_length(items)>0;
  END IF;
 END IF;
 PERFORM banking.collection_recovery_lock(actor);res:=jsonb_build_object('profile','private-onboarding-catalog-v1','workspace',ws,'can_setup',can_setup,'can_adopt',can_adopt,'candidates',items);
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;
CREATE FUNCTION banking.preview_collection_onboarding(actor uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;p banking.collection_onboarding_preview;x jsonb;res jsonb;BEGIN
 PERFORM banking.collection_recovery_principal(actor);v:=banking.import_json(raw,8192);p.principal_id:=actor;p.operation:=ledger.text_value(v->'operation');
 IF p.operation='setup'THEN
  PERFORM banking.collection_keys(v,ARRAY['operation']);IF EXISTS(SELECT FROM banking.collection_workspace WHERE owner_principal_id=actor)THEN PERFORM banking.collection_error('P2802');END IF;
  p.snapshot:=encode(sha256(convert_to(jsonb_build_array(actor,'absent')::text,'UTF8')),'hex');p.review:=jsonb_build_object('capabilities',jsonb_build_array('read','acquire','classify'));
 ELSIF p.operation='adopt'THEN
  PERFORM banking.collection_keys(v,ARRAY['operation','candidate_id','snapshot']);x:=banking.collection_onboarding_candidate(actor,ledger.uuid_value(v->'candidate_id'),true);
  IF ledger.text_value(v->'snapshot')COLLATE "C" !~ '^[0-9a-f]{64}$'THEN PERFORM banking.collection_error('P2800');END IF;IF v->>'snapshot'<>x->>'snapshot'THEN PERFORM banking.collection_error('P2803');END IF;
  p.workspace_id:=(x->>'workspace_id')::uuid;p.snapshot:=x->>'snapshot';p.review:=(x-'workspace_id')||jsonb_build_object('impact','pause_legacy_acquisition');
 ELSE PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_recovery_lock(actor);IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor)THEN PERFORM banking.collection_error('P2802');END IF;
 p.preview_id:=gen_random_uuid();p.created_at:=clock_timestamp();p.expires_at:=p.created_at+interval '15 minutes';res:=banking.collection_onboarding_preview_wire(p);IF octet_length(res::text)>1040384 THEN PERFORM banking.collection_error('P2805');END IF;
 INSERT INTO banking.collection_onboarding_preview VALUES(p.*);INSERT INTO audit.collection_onboarding_action(principal_id,operation,preview_id)VALUES(actor,'preview',p.preview_id);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.prepare_collection_onboarding(actor uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;id uuid;k text;p banking.collection_onboarding_preview;c banking.collection_onboarding_command;x jsonb;res jsonb;BEGIN
 PERFORM banking.collection_recovery_principal(actor);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);id:=ledger.uuid_value(v->'preview_id');k:=ledger.text_value(v->'prepare_key');IF k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO p FROM banking.collection_onboarding_preview WHERE preview_id=id AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;PERFORM banking.collection_onboarding_authorize(actor,p);
 SELECT * INTO c FROM banking.collection_onboarding_command WHERE principal_id=actor AND prepare_key=k FOR SHARE;
 IF FOUND THEN IF c.preview_id<>id THEN PERFORM banking.collection_error('P2802');END IF;PERFORM banking.collection_recovery_lock(actor);RETURN banking.collection_onboarding_command_wire(c);END IF;
 IF p.operation='setup'THEN IF EXISTS(SELECT FROM banking.collection_workspace WHERE owner_principal_id=actor)THEN PERFORM banking.collection_error('P2803');END IF;
 ELSE x:=banking.collection_onboarding_candidate(actor,(p.review->>'candidate_id')::uuid,true);IF x->>'snapshot'<>p.snapshot THEN PERFORM banking.collection_error('P2803');END IF;END IF;
 IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;PERFORM banking.collection_recovery_lock(actor);
 c.command_id:=gen_random_uuid();c.recovery_id:=gen_random_uuid();c.principal_id:=actor;c.preview_id:=id;c.prepare_key:=k;c.state:='prepared';c.created_at:=clock_timestamp();res:=banking.collection_onboarding_command_wire(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;
 INSERT INTO banking.collection_onboarding_command(command_id,recovery_id,principal_id,preview_id,prepare_key,created_at)VALUES(c.command_id,c.recovery_id,actor,id,k,c.created_at);INSERT INTO audit.collection_onboarding_action(principal_id,operation,preview_id,command_id)VALUES(actor,'prepare',id,c.command_id);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.pending_collection_onboarding(actor uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE r ledger.interactive_command_reservation;c banking.collection_onboarding_command;p banking.collection_onboarding_preview;available boolean:=true;res jsonb;BEGIN
 PERFORM banking.collection_recovery_principal(actor);SELECT * INTO r FROM ledger.interactive_command_reservation WHERE principal_id=actor;
 IF NOT FOUND THEN PERFORM banking.collection_recovery_lock(actor);RETURN jsonb_build_object('profile','private-onboarding-pending-v1','state','none','command',NULL,'recovery_id',NULL);END IF;
 IF r.onboarding_command_id IS NULL THEN PERFORM banking.collection_recovery_lock(actor);RETURN jsonb_build_object('profile','private-onboarding-pending-v1','state','blocked','command',NULL,'recovery_id',NULL);END IF;
 SELECT * INTO c FROM banking.collection_onboarding_command WHERE command_id=r.onboarding_command_id AND principal_id=actor FOR SHARE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_onboarding';END IF;
 SELECT * INTO p FROM banking.collection_onboarding_preview WHERE preview_id=c.preview_id;
 BEGIN PERFORM banking.collection_onboarding_authorize(actor,p);EXCEPTION WHEN SQLSTATE 'P2801'THEN available:=false;END;
 PERFORM banking.collection_recovery_lock(actor);res:=jsonb_build_object('profile','private-onboarding-pending-v1','state',CASE WHEN available THEN 'available'ELSE 'inaccessible'END,'command',CASE WHEN available THEN banking.collection_onboarding_command_wire(c)END,'recovery_id',r.recovery_id);
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;
CREATE FUNCTION banking.transition_collection_onboarding(actor uuid,id uuid,operation text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;c banking.collection_onboarding_command;p banking.collection_onboarding_preview;x jsonb;body jsonb;res jsonb;BEGIN
 PERFORM banking.collection_recovery_principal(actor);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY[]::text[]);IF operation IS NULL OR operation<>ALL(ARRAY['execute','cancel','acknowledge'])THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO c FROM banking.collection_onboarding_command WHERE command_id=id AND principal_id=actor FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;SELECT * INTO p FROM banking.collection_onboarding_preview WHERE preview_id=c.preview_id;PERFORM banking.collection_onboarding_authorize(actor,p);
 IF operation='acknowledge'THEN
  IF c.state='prepared'THEN PERFORM banking.collection_error('P2802');END IF;
  IF c.acknowledged_at IS NULL THEN UPDATE banking.collection_onboarding_command SET acknowledged_at=clock_timestamp()WHERE command_id=id RETURNING * INTO c;INSERT INTO audit.collection_onboarding_action(principal_id,operation,preview_id,command_id)VALUES(actor,'acknowledge',p.preview_id,id);END IF;
 ELSIF c.state='prepared'THEN
  IF operation='execute'THEN
   IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;
   IF p.operation='setup'THEN
    IF EXISTS(SELECT FROM banking.collection_workspace WHERE owner_principal_id=actor)THEN PERFORM banking.collection_error('P2803');END IF;
    res:=banking.provision_collection_owner(actor,'onboarding:'||id::text,jsonb_build_object('expected_revision',NULL,'capabilities',jsonb_build_array('read','acquire','classify'),'evidence_ref',p.preview_id)::text);
   ELSE
    x:=banking.collection_onboarding_candidate(actor,(p.review->>'candidate_id')::uuid,true);IF x->>'snapshot'<>p.snapshot THEN PERFORM banking.collection_error('P2803');END IF;
    body:=jsonb_build_object('expected_generation',p.review->'expected_generation','sources',(SELECT jsonb_agg(jsonb_build_object('source_account_id',s->'source_id','display_name',s->'display_name')ORDER BY s->>'source_id')FROM jsonb_array_elements(p.review->'sources')s),'evidence_ref',p.preview_id);
    res:=banking.adopt_collection_item(actor,p.workspace_id,(p.review->>'entity_id')::uuid,(p.review->>'book_id')::uuid,(SELECT legacy_sync_item_id FROM integration.collection_item WHERE collection_item_id=(p.review->>'candidate_id')::uuid),'onboarding:'||id::text,body::text);
   END IF;
   UPDATE banking.collection_onboarding_command SET state='committed',result=res,terminal_at=clock_timestamp()WHERE command_id=id RETURNING * INTO c;
  ELSE UPDATE banking.collection_onboarding_command SET state='cancelled',result=jsonb_build_object('cancelled',true),terminal_at=clock_timestamp()WHERE command_id=id RETURNING * INTO c;END IF;
  INSERT INTO audit.collection_onboarding_action(principal_id,operation,preview_id,command_id)VALUES(actor,operation,p.preview_id,id);
 END IF;
 PERFORM banking.collection_recovery_lock(actor);res:=banking.collection_onboarding_command_wire(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.clear_collection_onboarding(actor uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;id uuid;r ledger.interactive_command_reservation;c banking.collection_onboarding_command;p banking.collection_onboarding_preview;available boolean:=true;BEGIN
 PERFORM banking.collection_recovery_principal(actor);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY['recovery_id']);id:=ledger.uuid_value(v->'recovery_id');
 IF EXISTS(SELECT FROM ledger.collection_recovery_receipt WHERE principal_id=actor AND recovery_id=id AND onboarding_command_id IS NOT NULL)THEN PERFORM banking.collection_recovery_lock(actor);RETURN jsonb_build_object('profile','private-onboarding-clear-v1','recovery_id',id,'cleared',true);END IF;
 SELECT * INTO r FROM ledger.interactive_command_reservation WHERE principal_id=actor AND recovery_id=id AND onboarding_command_id IS NOT NULL;IF NOT FOUND THEN PERFORM banking.collection_recovery_lock(actor);PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO c FROM banking.collection_onboarding_command WHERE command_id=r.onboarding_command_id AND principal_id=actor FOR UPDATE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_onboarding';END IF;SELECT * INTO p FROM banking.collection_onboarding_preview WHERE preview_id=c.preview_id;
 BEGIN PERFORM banking.collection_onboarding_authorize(actor,p);EXCEPTION WHEN SQLSTATE 'P2801'THEN available:=false;END;IF available THEN PERFORM banking.collection_error('P2802');END IF;
 IF c.state='prepared'THEN UPDATE banking.collection_onboarding_command SET state='cancelled',result=jsonb_build_object('cancelled',true),terminal_at=clock_timestamp()WHERE command_id=c.command_id RETURNING * INTO c;INSERT INTO audit.collection_onboarding_action(principal_id,operation,preview_id,command_id)VALUES(actor,'cancel',p.preview_id,c.command_id);END IF;
 UPDATE banking.collection_onboarding_command SET acknowledged_at=clock_timestamp()WHERE command_id=c.command_id RETURNING * INTO c;INSERT INTO audit.collection_onboarding_action(principal_id,operation,preview_id,command_id)VALUES(actor,'acknowledge',p.preview_id,c.command_id);
 PERFORM banking.collection_recovery_lock(actor);IF NOT EXISTS(SELECT FROM ledger.collection_recovery_receipt WHERE principal_id=actor AND recovery_id=id AND onboarding_command_id=c.command_id)THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_onboarding';END IF;
 RETURN jsonb_build_object('profile','private-onboarding-clear-v1','recovery_id',id,'cleared',true);
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;

-- Old readers must not report false-none or parse this engine's payload.
ALTER FUNCTION ledger.pending_command(uuid)RENAME TO onboarding_pending_base;
CREATE FUNCTION ledger.pending_command(actor uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE res jsonb;BEGIN
 res:=ledger.onboarding_pending_base(actor);IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor AND onboarding_command_id IS NOT NULL)THEN PERFORM ledger.fail('P2112');END IF;PERFORM banking.collection_recovery_lock(actor);RETURN res;END $$;
ALTER FUNCTION ledger.clear_inaccessible_command(uuid,uuid,text)RENAME TO onboarding_clear_base;
CREATE FUNCTION ledger.clear_inaccessible_command(actor uuid,request uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE res jsonb;BEGIN
 PERFORM banking.collection_recovery_principal(actor);IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor AND onboarding_command_id IS NOT NULL)THEN PERFORM ledger.fail('P2112');END IF;res:=ledger.onboarding_clear_base(actor,request,raw);PERFORM banking.collection_recovery_lock(actor);RETURN res;END $$;
ALTER FUNCTION banking.collection_recovery_status(uuid)RENAME TO collection_onboarding_recovery_status_base;
CREATE FUNCTION banking.collection_recovery_status(actor uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$BEGIN
 PERFORM banking.collection_recovery_principal(actor);IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor AND onboarding_command_id IS NOT NULL)THEN PERFORM banking.collection_error('P2802');END IF;RETURN banking.collection_onboarding_recovery_status_base(actor);END $$;
ALTER FUNCTION banking.clear_collection_recovery(uuid,text)RENAME TO collection_onboarding_recovery_clear_base;
CREATE FUNCTION banking.clear_collection_recovery(actor uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE v jsonb;BEGIN
 PERFORM banking.collection_recovery_principal(actor);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY['recovery_id']);IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor AND recovery_id=ledger.uuid_value(v->'recovery_id')AND onboarding_command_id IS NOT NULL)OR EXISTS(SELECT FROM ledger.collection_recovery_receipt WHERE principal_id=actor AND recovery_id=ledger.uuid_value(v->'recovery_id')AND onboarding_command_id IS NOT NULL)THEN PERFORM banking.collection_error('P2801');END IF;RETURN banking.collection_onboarding_recovery_clear_base(actor,raw);END $$;
DO $$DECLARE n text;ver text;BEGIN FOR ver IN SELECT unnest(ARRAY['','_v2','_v3'])LOOP
 n:='pending_collection_label_command'||ver;EXECUTE format('ALTER FUNCTION banking.%I(uuid,uuid) RENAME TO %I',n,'collection_onboarding_pending'||coalesce(nullif(ver,''),'_v1')||'_base');
 EXECUTE format('CREATE FUNCTION banking.%I(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $body$ DECLARE res jsonb;BEGIN res:=banking.%I(actor,w);IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor AND onboarding_command_id IS NOT NULL)THEN PERFORM banking.collection_error(''P2802'');END IF;PERFORM banking.collection_recovery_lock(actor);RETURN res;END $body$',n,'collection_onboarding_pending'||coalesce(nullif(ver,''),'_v1')||'_base');
 END LOOP;END $$;
DO $$DECLARE f regprocedure;BEGIN FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE(n.nspname='banking'AND(p.proname LIKE '%collection_onboarding%'OR p.proname IN('collection_recovery_status','clear_collection_recovery','pending_collection_label_command','pending_collection_label_command_v2','pending_collection_label_command_v3')))OR(n.nspname='ledger'AND p.proname IN('onboarding_pending_base','onboarding_clear_base','pending_command','clear_inaccessible_command'))LOOP
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator',f);END LOOP;END $$;
REVOKE ALL ON banking.collection_onboarding_preview,banking.collection_onboarding_command,audit.collection_onboarding_action FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION banking.get_collection_onboarding(uuid),banking.preview_collection_onboarding(uuid,text),banking.prepare_collection_onboarding(uuid,text),banking.pending_collection_onboarding(uuid),banking.transition_collection_onboarding(uuid,uuid,text,text),banking.clear_collection_onboarding(uuid,text),banking.collection_recovery_status(uuid),banking.clear_collection_recovery(uuid,text),banking.pending_collection_label_command(uuid,uuid),banking.pending_collection_label_command_v2(uuid,uuid),banking.pending_collection_label_command_v3(uuid,uuid),ledger.pending_command(uuid),ledger.clear_inaccessible_command(uuid,uuid,text)TO bookkeeping_api;
COMMIT;
