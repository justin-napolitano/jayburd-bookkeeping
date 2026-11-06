-- #239; docs/contracts/private-registry.md. Source custody only, no journal authority.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Reservation backfill and legacy trigger installation share one admission lock.
LOCK TABLE integration.sync_item IN SHARE ROW EXCLUSIVE MODE;
CREATE TABLE banking.collection_workspace (
 workspace_id uuid PRIMARY KEY DEFAULT gen_random_uuid(), owner_principal_id uuid NOT NULL UNIQUE REFERENCES ledger.principal,
 revision bigint NOT NULL DEFAULT 1 CHECK(revision>0), capabilities text[] NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)),
 CHECK(capabilities<@ARRAY['read','acquire','classify']::text[] AND cardinality(capabilities)<=3 AND (cardinality(capabilities)=0 OR 'read'=ANY(capabilities)))
);
CREATE TABLE integration.collection_item (
 collection_item_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),provider text NOT NULL CHECK(provider='plaid'),environment text NOT NULL CHECK(environment IN ('sandbox','production')),
 item_id text COLLATE "C" NOT NULL CHECK(length(item_id) BETWEEN 1 AND 256 AND item_id COLLATE "C" ~ '^[!-~]+$'),
 workspace_id uuid REFERENCES banking.collection_workspace,
 legacy_entity_id uuid,legacy_book_id uuid,legacy_sync_item_id uuid,
 credential_ref uuid,credential_version bigint,consent_evidence_ref uuid,consent_expires_at timestamptz,
 state text NOT NULL DEFAULT 'paused' CHECK(state IN ('active','paused','revoked')),generation bigint NOT NULL DEFAULT 1 CHECK(generation>0),
 UNIQUE(provider,environment,item_id),UNIQUE(legacy_entity_id,legacy_book_id,legacy_sync_item_id),UNIQUE(workspace_id,collection_item_id),
 FOREIGN KEY(legacy_entity_id,legacy_book_id,legacy_sync_item_id) REFERENCES integration.sync_item,
 CHECK((legacy_entity_id IS NOT NULL AND legacy_book_id IS NOT NULL AND legacy_sync_item_id IS NOT NULL AND credential_ref IS NULL AND credential_version IS NULL AND consent_evidence_ref IS NULL AND consent_expires_at IS NULL)
 OR(legacy_entity_id IS NULL AND legacy_book_id IS NULL AND legacy_sync_item_id IS NULL AND workspace_id IS NOT NULL AND credential_ref IS NOT NULL AND credential_version IS NOT NULL AND credential_version>0 AND consent_evidence_ref IS NOT NULL AND consent_expires_at IS NOT NULL AND isfinite(consent_expires_at)))
);
CREATE TABLE banking.collection_source (
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,source_account_id uuid NOT NULL DEFAULT gen_random_uuid(),collection_item_id uuid NOT NULL,
 account_id text COLLATE "C" NOT NULL CHECK(length(account_id) BETWEEN 1 AND 256 AND account_id COLLATE "C" ~ '^[!-~]+$'),
 display_name text NOT NULL CHECK(length(display_name) BETWEEN 1 AND 200 AND btrim(display_name)<>'' AND display_name !~ '[[:cntrl:]]'),
 origin text NOT NULL CHECK(origin IN ('native','legacy')),legacy_entity_id uuid,legacy_book_id uuid,legacy_source_account_id uuid,
 state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','revoked')),revision bigint NOT NULL DEFAULT 1 CHECK(revision>0),source_revision bigint NOT NULL DEFAULT 0 CHECK(source_revision>=0),
 cursor_value text CHECK(length(cursor_value)<=256),readiness text NOT NULL DEFAULT 'NOT_READY' CHECK(readiness IN ('NOT_READY','TRANSACTIONS_UPDATE_STATUS_UNKNOWN','INITIAL_UPDATE_COMPLETE','HISTORICAL_UPDATE_COMPLETE')),last_completed_at timestamptz,
 PRIMARY KEY(workspace_id,source_account_id),UNIQUE(collection_item_id,account_id),UNIQUE(legacy_entity_id,legacy_book_id,legacy_source_account_id),
 FOREIGN KEY(workspace_id,collection_item_id) REFERENCES integration.collection_item(workspace_id,collection_item_id),
 FOREIGN KEY(legacy_entity_id,legacy_book_id,legacy_source_account_id) REFERENCES banking.source_account,
 CHECK((origin='native' AND legacy_entity_id IS NULL AND legacy_book_id IS NULL AND legacy_source_account_id IS NULL) OR(origin='legacy' AND legacy_entity_id IS NOT NULL AND legacy_book_id IS NOT NULL AND legacy_source_account_id IS NOT NULL)),
 CHECK(last_completed_at IS NULL OR isfinite(last_completed_at))
);
CREATE TABLE banking.collection_observation (
 workspace_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL DEFAULT gen_random_uuid(),external_entry_id text COLLATE "C" NOT NULL CHECK(length(external_entry_id) BETWEEN 1 AND 256 AND external_entry_id COLLATE "C" ~ '^[!-~]+$'),
 current_revision bigint NOT NULL DEFAULT 1 CHECK(current_revision>0),created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)),
 PRIMARY KEY(workspace_id,source_account_id,observation_id),UNIQUE(workspace_id,source_account_id,external_entry_id),FOREIGN KEY(workspace_id,source_account_id) REFERENCES banking.collection_source
);
CREATE TABLE banking.collection_revision (
 workspace_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,revision bigint NOT NULL CHECK(revision>0),previous_revision bigint,
 booked_date date NOT NULL CHECK(booked_date BETWEEN DATE '0001-01-01' AND DATE '9999-12-31'),description text NOT NULL CHECK(length(description)<=500 AND octet_length(description)<=2000),merchant_name text CHECK(length(merchant_name)<=500 AND octet_length(merchant_name)<=2000),
 source_state text NOT NULL CHECK(source_state IN ('booked','pending','removed','held')),currency_code text CHECK(length(currency_code)<=10),currency_version_id uuid REFERENCES reference.currency,
 amount_lexeme text NOT NULL CHECK(length(amount_lexeme) BETWEEN 1 AND 128),amount ledger.amount,direction text CHECK(direction IN ('inflow','outflow','zero')),
 hold_reason text CHECK(hold_reason IN ('unsupported_currency','invalid_precision')),normalized_fact jsonb NOT NULL CHECK(octet_length(normalized_fact::text)<=8192),
 PRIMARY KEY(workspace_id,source_account_id,observation_id,revision),FOREIGN KEY(workspace_id,source_account_id,observation_id) REFERENCES banking.collection_observation,
 FOREIGN KEY(workspace_id,source_account_id,observation_id,previous_revision) REFERENCES banking.collection_revision,
 CHECK((revision=1 AND previous_revision IS NULL) OR(revision>1 AND previous_revision IS NOT NULL AND previous_revision=revision-1)),
 CHECK((hold_reason IS NULL AND source_state<>'held' AND currency_version_id IS NOT NULL AND amount IS NOT NULL AND direction IS NOT NULL AND ((amount=0)=(direction='zero')))
 OR(hold_reason IS NOT NULL AND source_state IN('held','removed')AND amount IS NULL AND direction IS NULL AND currency_version_id IS NULL))
);
ALTER TABLE banking.collection_observation ADD CONSTRAINT collection_observation_current_fk FOREIGN KEY(workspace_id,source_account_id,observation_id,current_revision)REFERENCES banking.collection_revision(workspace_id,source_account_id,observation_id,revision)DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE integration.collection_receipt (
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,principal_id uuid NOT NULL REFERENCES ledger.principal,
 operation text NOT NULL CHECK(operation IN ('provision','register','publish','state','adopt')),idempotency_key text NOT NULL CHECK(idempotency_key COLLATE "C" ~ '^[A-Za-z0-9._:-]{1,128}$'),
 request jsonb NOT NULL CHECK(octet_length(request::text)<=2097152),result jsonb NOT NULL CHECK(octet_length(result::text)<=2097152),recorded_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(recorded_at)),
 PRIMARY KEY(workspace_id,principal_id,operation,idempotency_key)
);
CREATE TABLE audit.collection_action (
 action_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,principal_id uuid NOT NULL REFERENCES ledger.principal,
 operation text NOT NULL,target_id uuid NOT NULL,evidence_ref uuid,database_actor name NOT NULL DEFAULT session_user,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(recorded_at))
);
CREATE INDEX collection_source_workspace_idx ON banking.collection_source(workspace_id,source_account_id);
CREATE INDEX collection_revision_date_idx ON banking.collection_revision(workspace_id,booked_date DESC,source_account_id,observation_id);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_revision FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.collection_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON audit.collection_action FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Uniform diagnostics never include source values or provider identifiers.
CREATE FUNCTION banking.collection_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='collection_rejected';END $$;
-- Closed objects permit explicitly nullable fields only after each caller validates their type.
CREATE FUNCTION banking.collection_keys(v jsonb,names text[]) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN
 IF jsonb_typeof(v) IS DISTINCT FROM 'object' OR NOT(v ?& names) OR EXISTS(SELECT FROM jsonb_object_keys(v) k WHERE k<>ALL(names)) THEN PERFORM banking.collection_error('P2800');END IF;
END $$;
-- Identifiers remain opaque printable provider text, never authority or display markup.
CREATE FUNCTION banking.collection_text(v jsonb,n integer) RETURNS text LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ DECLARE s text;BEGIN
 s:=ledger.text_value(v);IF length(s) NOT BETWEEN 1 AND n OR s COLLATE "C" !~ '^[!-~]+$' THEN PERFORM banking.collection_error('P2800');END IF;RETURN s;
END $$;
-- Evidence pointer/identity fields are immutable; current source pointers advance monotonically.
CREATE FUNCTION banking.collection_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN
 IF TG_OP='DELETE' THEN PERFORM banking.collection_error('P2802');END IF;
 IF TG_TABLE_NAME='collection_workspace' THEN
  IF (to_jsonb(NEW)-ARRAY['capabilities','revision']) IS DISTINCT FROM(to_jsonb(OLD)-ARRAY['capabilities','revision']) OR NEW.revision::numeric<>OLD.revision::numeric+1 THEN PERFORM banking.collection_error('P2802');END IF;
 ELSIF TG_TABLE_NAME='collection_source' THEN
  IF (to_jsonb(NEW)-ARRAY['state','revision','source_revision','cursor_value','readiness','last_completed_at']) IS DISTINCT FROM(to_jsonb(OLD)-ARRAY['state','revision','source_revision','cursor_value','readiness','last_completed_at']) OR NEW.revision::numeric<>OLD.revision::numeric+1 OR NEW.source_revision<OLD.source_revision OR OLD.state='revoked' THEN PERFORM banking.collection_error('P2802');END IF;
 ELSIF TG_TABLE_NAME='collection_observation' THEN
  IF(to_jsonb(NEW)-'current_revision') IS DISTINCT FROM(to_jsonb(OLD)-'current_revision') OR NEW.current_revision::numeric<>OLD.current_revision::numeric+1 THEN PERFORM banking.collection_error('P2802');END IF;
 ELSE
  IF(to_jsonb(NEW)-ARRAY['workspace_id','state','generation']) IS DISTINCT FROM(to_jsonb(OLD)-ARRAY['workspace_id','state','generation']) OR NEW.generation::numeric<>OLD.generation::numeric+1 OR OLD.state='revoked' OR OLD.workspace_id IS NOT NULL AND NEW.workspace_id IS DISTINCT FROM OLD.workspace_id THEN PERFORM banking.collection_error('P2802');END IF;
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_collection BEFORE UPDATE OR DELETE ON banking.collection_workspace FOR EACH ROW EXECUTE FUNCTION banking.collection_guard();
CREATE TRIGGER preserve_collection BEFORE UPDATE OR DELETE ON banking.collection_source FOR EACH ROW EXECUTE FUNCTION banking.collection_guard();
CREATE TRIGGER preserve_collection BEFORE UPDATE OR DELETE ON banking.collection_observation FOR EACH ROW EXECUTE FUNCTION banking.collection_guard();
CREATE TRIGGER preserve_collection BEFORE UPDATE OR DELETE ON integration.collection_item FOR EACH ROW EXECUTE FUNCTION banking.collection_guard();
-- Backfill reservations only: no private grant, cursor, credential or source history changes.
DO $$ BEGIN
 IF EXISTS(SELECT FROM integration.sync_item GROUP BY provider,environment,item_id HAVING count(*)>1)THEN PERFORM banking.collection_error('P2802');END IF;
END $$;
INSERT INTO integration.collection_item(provider,environment,item_id,legacy_entity_id,legacy_book_id,legacy_sync_item_id)
 SELECT provider,environment,item_id,entity_id,book_id,sync_item_id FROM integration.sync_item;
-- Both legacy enrollment and native registration use the same unique reservation key.
CREATE FUNCTION integration.collection_legacy_identity() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ DECLARE i integration.collection_item;BEGIN
 IF TG_TABLE_NAME='sync_stream' THEN
  IF EXISTS(SELECT FROM integration.collection_item WHERE legacy_entity_id=NEW.entity_id AND legacy_book_id=NEW.book_id AND legacy_sync_item_id=NEW.sync_item_id AND workspace_id IS NOT NULL) THEN PERFORM banking.collection_error('P2804');END IF;RETURN NEW;
 END IF;
 IF TG_OP='UPDATE' THEN
  IF EXISTS(SELECT FROM integration.collection_item WHERE legacy_entity_id=OLD.entity_id AND legacy_book_id=OLD.book_id AND legacy_sync_item_id=OLD.sync_item_id AND workspace_id IS NOT NULL) THEN PERFORM banking.collection_error('P2804');END IF;RETURN NEW;
 END IF;
 INSERT INTO integration.collection_item(provider,environment,item_id,legacy_entity_id,legacy_book_id,legacy_sync_item_id) VALUES(NEW.provider,NEW.environment,NEW.item_id,NEW.entity_id,NEW.book_id,NEW.sync_item_id) ON CONFLICT(provider,environment,item_id) DO NOTHING;
 SELECT * INTO i FROM integration.collection_item WHERE provider=NEW.provider AND environment=NEW.environment AND item_id=NEW.item_id;
 IF ROW(i.legacy_entity_id,i.legacy_book_id,i.legacy_sync_item_id) IS DISTINCT FROM ROW(NEW.entity_id,NEW.book_id,NEW.sync_item_id) OR i.workspace_id IS NOT NULL THEN PERFORM banking.collection_error('P2802');END IF;RETURN NEW;
END $$;
CREATE TRIGGER collection_identity AFTER INSERT ON integration.sync_item FOR EACH ROW EXECUTE FUNCTION integration.collection_legacy_identity();
CREATE TRIGGER collection_fence BEFORE UPDATE ON integration.sync_item FOR EACH ROW EXECUTE FUNCTION integration.collection_legacy_identity();
CREATE TRIGGER collection_fence BEFORE INSERT ON integration.sync_stream FOR EACH ROW EXECUTE FUNCTION integration.collection_legacy_identity();
-- Current private capability is independent of all entity/book memberships.
CREATE FUNCTION banking.collection_authorize(actor uuid,w uuid,cap text,writing boolean) RETURNS banking.collection_workspace LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ DECLARE c banking.collection_workspace;BEGIN
 IF actor IS NULL OR (writing AND current_setting('transaction_isolation')<>'serializable') OR(NOT writing AND current_setting('transaction_isolation') NOT IN ('repeatable read','serializable')) THEN PERFORM banking.collection_error('P2800');END IF;
 IF NOT integration.provider_api_login(session_user) THEN PERFORM banking.collection_error('P2801');END IF;
 PERFORM set_config('lock_timeout','1s',true);
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 IF writing THEN SELECT * INTO c FROM banking.collection_workspace WHERE owner_principal_id=actor AND(w IS NULL OR workspace_id=w) FOR UPDATE;
 ELSE SELECT * INTO c FROM banking.collection_workspace WHERE owner_principal_id=actor AND(w IS NULL OR workspace_id=w) FOR SHARE;END IF;
 IF NOT FOUND OR NOT cap=ANY(c.capabilities) THEN PERFORM banking.collection_error('P2801');END IF;RETURN c;
END $$;
-- Operator attestation is separate from user SQL authority; runtime cannot self-provision.
CREATE FUNCTION banking.provision_collection_owner(subject_value uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;c banking.collection_workspace;r integration.collection_receipt;caps text[];res jsonb;norm jsonb;expected bigint;evidence uuid;BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='collection_rejected';END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR subject_value IS NULL OR k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' THEN PERFORM banking.collection_error('P2800');END IF;
 v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['expected_revision','capabilities','evidence_ref']);evidence:=ledger.uuid_value(v->'evidence_ref');
 IF jsonb_typeof(v->'capabilities') IS DISTINCT FROM 'array' OR jsonb_array_length(v->'capabilities')>3 OR EXISTS(SELECT FROM jsonb_array_elements(v->'capabilities') z WHERE jsonb_typeof(z)<>'string' OR z#>>'{}'<>ALL(ARRAY['read','acquire','classify'])) THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT coalesce(array_agg(x ORDER BY x),'{}') INTO caps FROM jsonb_array_elements_text(v->'capabilities') x;
 IF cardinality(caps)<>(SELECT count(DISTINCT x) FROM unnest(caps)x) OR cardinality(caps)>0 AND NOT 'read'=ANY(caps) THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'expected_revision'<>'null'::jsonb THEN expected:=banking.import_revision(v->'expected_revision');END IF;
 PERFORM 1 FROM ledger.principal WHERE principal_id=subject_value AND disabled_at IS NULL FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO c FROM banking.collection_workspace WHERE owner_principal_id=subject_value FOR UPDATE;
 norm:=jsonb_build_object('database_actor',session_user,'body',v);
 IF FOUND THEN
  SELECT * INTO r FROM integration.collection_receipt WHERE workspace_id=c.workspace_id AND principal_id=subject_value AND operation='provision' AND idempotency_key=k;
  IF FOUND THEN IF r.request<>norm THEN PERFORM banking.collection_error('P2802');END IF;IF c.revision::text<>r.result->>'workspace_revision' THEN PERFORM banking.collection_error('P2803');END IF;RETURN r.result;END IF;
  IF expected IS DISTINCT FROM c.revision THEN PERFORM banking.collection_error('P2803');END IF;
  UPDATE banking.collection_workspace SET revision=revision+1,capabilities=caps WHERE workspace_id=c.workspace_id RETURNING * INTO c;
 ELSE
  IF expected IS NOT NULL THEN PERFORM banking.collection_error('P2803');END IF;
  INSERT INTO banking.collection_workspace(owner_principal_id,capabilities)VALUES(subject_value,caps)RETURNING * INTO c;
 END IF;
 res:=jsonb_build_object('workspace_id',c.workspace_id,'workspace_revision',c.revision::text,'capabilities',to_jsonb(c.capabilities));
 INSERT INTO integration.collection_receipt(workspace_id,principal_id,operation,idempotency_key,request,result)VALUES(c.workspace_id,subject_value,'provision',k,norm,res);
 INSERT INTO audit.collection_action(workspace_id,principal_id,operation,target_id,evidence_ref)VALUES(c.workspace_id,subject_value,'provision',c.workspace_id,evidence);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM banking.collection_error('P2800');END $$;
-- This view function is owner-only; authorized wrappers perform all principal checks.
CREATE FUNCTION banking.collection_visible_sources(w uuid) RETURNS SETOF banking.collection_source LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT s.* FROM banking.collection_source s JOIN integration.collection_item i USING(collection_item_id)
 LEFT JOIN banking.source_account legacy ON legacy.entity_id=s.legacy_entity_id AND legacy.book_id=s.legacy_book_id AND legacy.source_account_id=s.legacy_source_account_id
 WHERE s.workspace_id=w AND i.workspace_id=w AND s.state='active' AND i.state<>'revoked' AND(s.origin='native' OR legacy.state='active')
$$;
-- Exact ordered revisions detect legacy importer changes without leaking legacy scope tuples.
CREATE FUNCTION banking.collection_snapshot(w uuid) RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT jsonb_build_object('workspace_revision',c.revision::text,'source_snapshot',encode(sha256(convert_to(coalesce((
 SELECT jsonb_agg(jsonb_build_array(s.source_account_id,s.revision::text,s.source_revision::text,i.generation::text,i.state,old.binding_revision::text,old.source_revision::text,li.generation::text,li.state)ORDER BY s.source_account_id)::text
 FROM banking.collection_visible_sources(w)s JOIN integration.collection_item i USING(collection_item_id)
 LEFT JOIN banking.source_account old ON old.entity_id=s.legacy_entity_id AND old.book_id=s.legacy_book_id AND old.source_account_id=s.legacy_source_account_id
 LEFT JOIN integration.sync_item li ON li.entity_id=i.legacy_entity_id AND li.book_id=i.legacy_book_id AND li.sync_item_id=i.legacy_sync_item_id
 ),'[]'),'UTF8')),'hex'))FROM banking.collection_workspace c WHERE c.workspace_id=w
$$;
-- Snapshot and cursor strings are equality pins only, never capabilities.
CREATE FUNCTION banking.collection_page(v jsonb,w uuid,snapshot_value jsonb,selector_value jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ DECLARE cur jsonb;txt text;BEGIN
 IF jsonb_typeof(v->'limit') IS DISTINCT FROM 'number' OR v->>'limit' !~ '^[1-9][0-9]{0,2}$' OR(v->>'limit')::integer>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'snapshot'<>'null'::jsonb THEN
  PERFORM banking.collection_keys(v->'snapshot',ARRAY['workspace_revision','source_snapshot']);PERFORM banking.import_revision(v->'snapshot'->'workspace_revision');
  IF ledger.text_value(v->'snapshot'->'source_snapshot') COLLATE "C" !~ '^[0-9a-f]{64}$' THEN PERFORM banking.collection_error('P2800');END IF;
  IF v->'snapshot'<>snapshot_value THEN PERFORM banking.collection_error('P2803');END IF;
 END IF;
 IF v->'cursor'='null'::jsonb THEN RETURN NULL;END IF;
 txt:=ledger.text_value(v->'cursor');IF length(txt) NOT BETWEEN 2 AND 4096 OR txt COLLATE "C" !~ '^([0-9a-f]{2})+$' THEN PERFORM banking.collection_error('P2800');END IF;
 cur:=banking.import_json(convert_from(decode(txt,'hex'),'UTF8'),2048);PERFORM banking.collection_keys(cur,ARRAY['workspace_id','snapshot','selector','anchor']);
 IF ledger.uuid_value(cur->'workspace_id')<>w OR cur->'selector'<>selector_value THEN PERFORM banking.collection_error('P2800');END IF;
 IF cur->'snapshot'<>snapshot_value THEN PERFORM banking.collection_error('P2803');END IF;RETURN cur->'anchor';
EXCEPTION WHEN character_not_in_repertoire OR untranslatable_character OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;
-- No read creates a command, audit entry, provider operation or journal.
-- Current row locks reject primed RR snapshots after native or legacy lifecycle changes.
CREATE FUNCTION banking.collection_read_locks(w uuid)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$ BEGIN
 IF(SELECT count(*)FROM integration.collection_item WHERE workspace_id=w)>100 OR(SELECT count(*)FROM banking.collection_source WHERE workspace_id=w)>100 THEN PERFORM banking.collection_error('P2805');END IF;
 PERFORM 1 FROM integration.collection_item i WHERE i.workspace_id=w ORDER BY i.collection_item_id FOR SHARE OF i;
 PERFORM 1 FROM banking.collection_source s WHERE s.workspace_id=w ORDER BY s.source_account_id FOR SHARE OF s;
 -- No state predicate: a concurrent pause/revoke must not disappear before its current-row check.
 -- Raw private reads never lock a book or infer a book grant from this provenance lookup.
 PERFORM 1 FROM banking.source_account legacy JOIN banking.collection_source s ON s.legacy_entity_id=legacy.entity_id AND s.legacy_book_id=legacy.book_id AND s.legacy_source_account_id=legacy.source_account_id
 WHERE s.workspace_id=w AND s.origin='legacy'ORDER BY legacy.entity_id,legacy.book_id,legacy.source_account_id FOR SHARE OF legacy;
END $$;
CREATE FUNCTION banking.get_collection_workspace(actor uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$ DECLARE c banking.collection_workspace;BEGIN
 c:=banking.collection_authorize(actor,NULL,'read',false);RETURN jsonb_build_object('profile','private-source-workspace-v1','workspace_id',c.workspace_id,'workspace_revision',c.revision::text,'capabilities',to_jsonb(c.capabilities));
END $$;
-- Safe source aliases expose no provider IDs, credential references or book/journal metadata.
CREATE FUNCTION banking.list_collection_sources(actor uuid,w uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_workspace;v jsonb;snap jsonb;sel jsonb:='{}';anchor jsonb;aid uuid;res jsonb;BEGIN
 c:=banking.collection_authorize(actor,w,'read',false);PERFORM banking.collection_read_locks(w);v:=banking.import_json(raw,8192);PERFORM banking.collection_keys(v,ARRAY['limit','cursor','snapshot']);
 IF(SELECT count(*)FROM banking.collection_visible_sources(w))>100 THEN PERFORM banking.collection_error('P2805');END IF;
 snap:=banking.collection_snapshot(w);anchor:=banking.collection_page(v,w,snap,sel);IF anchor IS NOT NULL THEN aid:=ledger.uuid_value(anchor);IF NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id=aid)THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 WITH picked AS MATERIALIZED(SELECT * FROM banking.collection_visible_sources(w)WHERE aid IS NULL OR source_account_id>aid ORDER BY source_account_id LIMIT(v->>'limit')::integer)
 SELECT jsonb_build_object('profile','private-source-page-v1','workspace_id',w,'workspace_revision',c.revision::text,'snapshot',snap,
 'items',coalesce((SELECT jsonb_agg(jsonb_build_object('source_account_id',source_account_id,'display_name',display_name,'state',state,'source_revision',CASE WHEN origin='legacy'THEN(SELECT l.source_revision::text FROM banking.source_account l WHERE l.entity_id=picked.legacy_entity_id AND l.book_id=picked.legacy_book_id AND l.source_account_id=picked.legacy_source_account_id)ELSE source_revision::text END,'origin',origin)ORDER BY source_account_id)FROM picked),'[]'::jsonb),
 'next_cursor',CASE WHEN EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id>(SELECT max(source_account_id::text)::uuid FROM picked))THEN encode(convert_to(jsonb_build_object('workspace_id',w,'snapshot',snap,'selector',sel,'anchor',(SELECT source_account_id FROM picked ORDER BY source_account_id DESC LIMIT 1))::text,'UTF8'),'hex')END)INTO res;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM banking.collection_error('P2800');END $$;
-- Union reads preserve native and legacy original evidence in place and hide legacy book scope.
CREATE FUNCTION banking.collection_facts(w uuid)RETURNS TABLE(source_account_id uuid,observation_id uuid,observation_revision bigint,source_state text,booked_date date,description text,merchant_name text,amount numeric,direction text,currency_version_id uuid,hold_reason text)
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT s.source_account_id,o.observation_id,q.revision,q.source_state,q.booked_date,q.description,q.merchant_name,q.amount::numeric,q.direction,q.currency_version_id,q.hold_reason
 FROM banking.collection_visible_sources(w)s JOIN banking.collection_observation o ON o.workspace_id=w AND o.source_account_id=s.source_account_id JOIN banking.collection_revision q ON q.workspace_id=w AND q.source_account_id=s.source_account_id AND q.observation_id=o.observation_id AND q.revision=o.current_revision WHERE s.origin='native'
 UNION ALL
 SELECT s.source_account_id,o.observation_id,q.revision,q.source_state,q.booked_date,q.description,CASE WHEN q.adapter_profile='jayburd-plaid-transactions/1' THEN q.provider_fields->>'merchant_name' END,q.amount::numeric,q.direction,q.currency_version_id,NULL::text
 FROM banking.collection_visible_sources(w)s JOIN banking.source_observation o ON o.entity_id=s.legacy_entity_id AND o.book_id=s.legacy_book_id AND o.source_account_id=s.legacy_source_account_id JOIN banking.observation_revision q ON q.entity_id=o.entity_id AND q.book_id=o.book_id AND q.observation_id=o.observation_id AND q.revision=o.current_revision WHERE s.origin='legacy'
$$;
-- Whole matching counts and pages share the same exact original-unit snapshot.
CREATE FUNCTION banking.query_collection_observations(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE cw banking.collection_workspace;v jsonb;snap jsonb;sel jsonb;anchor jsonb;sid uuid;from_day date;through_day date;state_value text;search_value text;ad date;asource uuid;ao uuid;res jsonb;BEGIN
 cw:=banking.collection_authorize(actor,w,'read',false);PERFORM banking.collection_read_locks(w);v:=banking.import_json(raw,8192);PERFORM banking.collection_keys(v,ARRAY['source_account_id','starts_on','through_date','state','search','limit','cursor','snapshot']);
 IF v->'source_account_id'<>'null'::jsonb THEN sid:=ledger.uuid_value(v->'source_account_id');IF NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id=sid)THEN PERFORM banking.collection_error('P2801');END IF;END IF;
 IF v->'starts_on'<>'null'::jsonb THEN from_day:=ledger.date_value(ledger.text_value(v->'starts_on'));END IF;IF v->'through_date'<>'null'::jsonb THEN through_day:=ledger.date_value(ledger.text_value(v->'through_date'));END IF;IF from_day>through_day THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'state'<>'null'::jsonb THEN state_value:=ledger.text_value(v->'state');IF state_value<>ALL(ARRAY['booked','pending','removed','held'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 search_value:=ledger.text_value(v->'search');IF length(search_value)>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF(SELECT count(*)FROM banking.collection_visible_sources(w))>100 THEN PERFORM banking.collection_error('P2805');END IF;
 snap:=banking.collection_snapshot(w);sel:=v-ARRAY['limit','cursor','snapshot'];anchor:=banking.collection_page(v,w,snap,sel);
 IF anchor IS NOT NULL THEN PERFORM banking.collection_keys(anchor,ARRAY['booked_date','source_account_id','observation_id']);ad:=ledger.date_value(ledger.text_value(anchor->'booked_date'));asource:=ledger.uuid_value(anchor->'source_account_id');ao:=ledger.uuid_value(anchor->'observation_id');END IF;
 WITH matched AS MATERIALIZED(SELECT * FROM banking.collection_facts(w)f WHERE(sid IS NULL OR f.source_account_id=sid)AND(from_day IS NULL OR f.booked_date>=from_day)AND(through_day IS NULL OR f.booked_date<=through_day)AND(state_value IS NULL OR f.source_state=state_value)AND strpos(lower(f.description COLLATE "C"),lower(search_value COLLATE "C"))>0),
 picked AS MATERIALIZED(SELECT * FROM matched f WHERE anchor IS NULL OR f.booked_date<ad OR f.booked_date=ad AND ROW(f.source_account_id,f.observation_id)>ROW(asource,ao)ORDER BY f.booked_date DESC,f.source_account_id,f.observation_id LIMIT(v->>'limit')::integer),
 rendered AS MATERIALIZED(SELECT p.*,jsonb_build_object('source_account_id',p.source_account_id,'observation_id',p.observation_id,'observation_revision',p.observation_revision::text,'source_state',p.source_state,'booked_date',to_char(p.booked_date,'YYYY-MM-DD'),'description',p.description,'merchant_name',p.merchant_name,'amount',trim_scale(p.amount)::text,'direction',p.direction,'hold_reason',p.hold_reason,'currency',CASE WHEN c.currency_version_id IS NOT NULL THEN jsonb_build_object('version_id',c.currency_version_id,'code',c.alphabetic_code,'numeric_code',c.numeric_code,'minor_unit',c.minor_unit,'valid_from',to_char(c.valid_from,'YYYY-MM-DD'),'valid_to',to_char(c.valid_to,'YYYY-MM-DD'))END)AS body FROM picked p LEFT JOIN reference.currency c ON c.currency_version_id=p.currency_version_id),
 last_row AS(SELECT * FROM picked ORDER BY booked_date ASC,source_account_id DESC,observation_id DESC LIMIT 1)
 SELECT jsonb_build_object('profile','private-source-query-v1','workspace_id',w,'workspace_revision',cw.revision::text,'snapshot',snap,'selector',sel,'matching_count',(SELECT count(*)::text FROM matched),
 'items',coalesce((SELECT jsonb_agg(body ORDER BY booked_date DESC,source_account_id,observation_id)FROM rendered),'[]'::jsonb),
 'next_cursor',CASE WHEN EXISTS(SELECT FROM matched f,last_row x WHERE f.booked_date<x.booked_date OR f.booked_date=x.booked_date AND ROW(f.source_account_id,f.observation_id)>ROW(x.source_account_id,x.observation_id))THEN encode(convert_to(jsonb_build_object('workspace_id',w,'snapshot',snap,'selector',sel,'anchor',(SELECT jsonb_build_object('booked_date',to_char(booked_date,'YYYY-MM-DD'),'source_account_id',source_account_id,'observation_id',observation_id)FROM last_row))::text,'UTF8'),'hex')END)INTO res;
 IF anchor IS NOT NULL AND NOT EXISTS(SELECT FROM banking.collection_facts(w)f WHERE f.source_account_id=asource AND f.observation_id=ao AND f.booked_date=ad AND(sid IS NULL OR f.source_account_id=sid)AND(from_day IS NULL OR f.booked_date>=from_day)AND(through_day IS NULL OR f.booked_date<=through_day)AND(state_value IS NULL OR f.source_state=state_value)AND strpos(lower(f.description COLLATE "C"),lower(search_value COLLATE "C"))>0)THEN PERFORM banking.collection_error('P2800');END IF;
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM banking.collection_error('P2800');END $$;
-- Grants and remaining write functions are appended below before transaction commit.
-- Trusted verified-manifest storage only; no browser route or provider token enters this boundary.
CREATE FUNCTION banking.register_collection_item(actor uuid,w uuid,k text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE cw banking.collection_workspace;v jsonb;r integration.collection_receipt;i integration.collection_item;a jsonb;aid text;label text;expiry timestamptz;ref uuid;ev uuid;ver bigint;res jsonb;items jsonb:='[]';sid uuid;BEGIN
 cw:=banking.collection_authorize(actor,w,'acquire',true);IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 v:=banking.import_json(raw,32768);PERFORM banking.collection_keys(v,ARRAY['provider','environment','item_id','credential_ref','credential_version','consent_evidence_ref','consent_expires_at','accounts']);
 IF v->'provider'IS DISTINCT FROM '"plaid"'::jsonb OR v->'environment'IS DISTINCT FROM '"sandbox"'::jsonb OR jsonb_typeof(v->'accounts')IS DISTINCT FROM 'array' OR jsonb_array_length(v->'accounts')NOT BETWEEN 1 AND 25 THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_text(v->'item_id',256);ref:=ledger.uuid_value(v->'credential_ref');ev:=ledger.uuid_value(v->'consent_evidence_ref');ver:=banking.import_revision(v->'credential_version');IF ver<1 THEN PERFORM banking.collection_error('P2800');END IF;
 IF ledger.text_value(v->'consent_expires_at') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'THEN PERFORM banking.collection_error('P2800');END IF;expiry:=(v->>'consent_expires_at')::timestamptz;
 IF NOT isfinite(expiry)OR expiry<=clock_timestamp()THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO r FROM integration.collection_receipt WHERE workspace_id=w AND principal_id=actor AND operation='register'AND idempotency_key=k;
 IF FOUND THEN IF r.request<>v THEN PERFORM banking.collection_error('P2802');END IF;
  IF NOT EXISTS(SELECT FROM integration.collection_item WHERE workspace_id=w AND collection_item_id=(r.result->>'collection_item_id')::uuid AND state<>'revoked') OR EXISTS(SELECT FROM banking.collection_source WHERE workspace_id=w AND collection_item_id=(r.result->>'collection_item_id')::uuid AND state<>'active')THEN PERFORM banking.collection_error('P2801');END IF;RETURN r.result;
 END IF;
 IF(SELECT count(*)FROM banking.collection_source WHERE workspace_id=w)+jsonb_array_length(v->'accounts')>100 THEN PERFORM banking.collection_error('P2805');END IF;
 IF EXISTS(SELECT x->>'account_id'FROM jsonb_array_elements(v->'accounts')x GROUP BY x->>'account_id'HAVING count(*)>1)THEN PERFORM banking.collection_error('P2802');END IF;
 INSERT INTO integration.collection_item(provider,environment,item_id,workspace_id,credential_ref,credential_version,consent_evidence_ref,consent_expires_at,state)VALUES('plaid','sandbox',v->>'item_id',w,ref,ver,ev,expiry,'active')RETURNING * INTO i;
 FOR a IN SELECT * FROM jsonb_array_elements(v->'accounts')LOOP
  PERFORM banking.collection_keys(a,ARRAY['account_id','display_name']);aid:=banking.collection_text(a->'account_id',256);label:=ledger.setup_text(a->'display_name',200);
  INSERT INTO banking.collection_source(workspace_id,collection_item_id,account_id,display_name,origin)VALUES(w,i.collection_item_id,aid,label,'native')RETURNING source_account_id INTO sid;
  items:=items||jsonb_build_array(jsonb_build_object('source_account_id',sid,'source_revision','0','revision','1'));
 END LOOP;
 res:=jsonb_build_object('collection_item_id',i.collection_item_id,'item_generation',i.generation::text,'sources',items);
 INSERT INTO integration.collection_receipt(workspace_id,principal_id,operation,idempotency_key,request,result)VALUES(w,actor,'register',k,v,res);
 INSERT INTO audit.collection_action(workspace_id,principal_id,operation,target_id,evidence_ref)VALUES(w,actor,'register',i.collection_item_id,ev);RETURN res;
EXCEPTION WHEN unique_violation THEN PERFORM banking.collection_error('P2802');WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR datetime_field_overflow OR invalid_datetime_format THEN PERFORM banking.collection_error('P2800');END $$;
-- Validate exact normalized financial text before any numeric domain could round it.
CREATE FUNCTION banking.collection_normalize_fact(v jsonb)RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE d date;code text;lexeme text;amount_value numeric;dir text;cv reference.currency;hold text;description_value text;merchant text;BEGIN
 PERFORM banking.collection_keys(v,ARRAY['booked_date','description','merchant_name','source_state','currency_code','amount','direction']);
 d:=ledger.date_value(ledger.text_value(v->'booked_date'));description_value:=ledger.text_value(v->'description');IF length(description_value)>500 OR octet_length(description_value)>2000 THEN PERFORM banking.collection_error('P2805');END IF;
 IF v->'merchant_name'<>'null'::jsonb THEN merchant:=ledger.text_value(v->'merchant_name');IF length(merchant)>500 OR octet_length(merchant)>2000 THEN PERFORM banking.collection_error('P2805');END IF;END IF;
 IF v->'currency_code'<>'null'::jsonb THEN code:=ledger.text_value(v->'currency_code');IF length(code)>10 THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 IF ledger.text_value(v->'source_state')<>ALL(ARRAY['booked','pending'])THEN PERFORM banking.collection_error('P2800');END IF;
 lexeme:=ledger.text_value(v->'amount');dir:=ledger.text_value(v->'direction');
 IF length(lexeme)NOT BETWEEN 1 AND 128 OR lexeme COLLATE "C" !~ '^(0|[1-9][0-9]*)(\.[0-9]+)?$'OR dir<>ALL(ARRAY['inflow','outflow','zero'])THEN PERFORM banking.collection_error('P2800');END IF;
 amount_value:=lexeme::numeric;IF(amount_value=0)<>(dir='zero')THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO cv FROM reference.currency WHERE alphabetic_code=code AND d>=valid_from AND(valid_to IS NULL OR d<valid_to);
 IF NOT FOUND OR cv.minor_unit IS NULL THEN hold:='unsupported_currency';
 ELSIF amount_value>=1e29 OR scale(amount_value)>9 OR scale(amount_value)>cv.minor_unit THEN hold:='invalid_precision';END IF;
 RETURN v||jsonb_build_object('currency_version_id',CASE WHEN hold IS NULL THEN cv.currency_version_id END,'validated_amount',CASE WHEN hold IS NULL THEN lexeme END,'validated_direction',CASE WHEN hold IS NULL THEN dir END,'hold_reason',hold,'effective_state',CASE WHEN hold IS NULL THEN v->>'source_state'ELSE 'held'END);
END $$;
-- Complete normalized batches advance one account cursor and all evidence atomically.
CREATE FUNCTION banking.publish_collection_batch(actor uuid,w uuid,sid uuid,k text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE cw banking.collection_workspace;s banking.collection_source;i integration.collection_item;r integration.collection_receipt;v jsonb;z jsonb;f jsonb;norm jsonb;o banking.collection_observation;old banking.collection_revision;oid uuid;rev bigint;op text;ext text;changed integer:=0;next_value text;start_value text;ready text;res jsonb;BEGIN
 cw:=banking.collection_authorize(actor,w,'acquire',true);IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO s FROM banking.collection_source WHERE workspace_id=w AND source_account_id=sid;IF NOT FOUND OR s.state<>'active'OR s.origin<>'native'THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO i FROM integration.collection_item WHERE collection_item_id=s.collection_item_id AND workspace_id=w FOR UPDATE;IF NOT FOUND OR i.state<>'active'OR i.consent_expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO s FROM banking.collection_source WHERE workspace_id=w AND source_account_id=sid FOR UPDATE;
 v:=banking.import_json(raw,2097152);PERFORM banking.collection_keys(v,ARRAY['expected_source_revision','expected_item_generation','starting_cursor','next_cursor','readiness','changes']);
 PERFORM banking.import_revision(v->'expected_source_revision');PERFORM banking.import_revision(v->'expected_item_generation');IF v->'starting_cursor'<>'null'::jsonb THEN start_value:=banking.collection_text(v->'starting_cursor',256);END IF;IF v->'next_cursor'<>'null'::jsonb THEN next_value:=banking.collection_text(v->'next_cursor',256);END IF;
 ready:=ledger.text_value(v->'readiness');IF ready<>ALL(ARRAY['NOT_READY','TRANSACTIONS_UPDATE_STATUS_UNKNOWN','INITIAL_UPDATE_COMPLETE','HISTORICAL_UPDATE_COMPLETE'])OR jsonb_typeof(v->'changes')IS DISTINCT FROM 'array'OR jsonb_array_length(v->'changes')>1000 THEN PERFORM banking.collection_error('P2800');END IF;
 norm:=jsonb_build_object('source_account_id',sid,'body',v);
 SELECT * INTO r FROM integration.collection_receipt WHERE workspace_id=w AND principal_id=actor AND operation='publish'AND idempotency_key=k;
 IF FOUND THEN IF r.request<>norm THEN PERFORM banking.collection_error('P2802');END IF;RETURN r.result;END IF;
 IF i.generation<>banking.import_revision(v->'expected_item_generation')OR s.source_revision<>banking.import_revision(v->'expected_source_revision')OR s.cursor_value IS DISTINCT FROM start_value THEN PERFORM banking.collection_error('P2803');END IF;
 IF EXISTS(SELECT x->>'transaction_id'FROM jsonb_array_elements(v->'changes')x GROUP BY x->>'transaction_id'HAVING count(*)>1)THEN PERFORM banking.collection_error('P2802');END IF;
 IF jsonb_array_length(v->'changes')>0 AND next_value IS NULL THEN PERFORM banking.collection_error('P2804');END IF;
 FOR z IN SELECT * FROM jsonb_array_elements(v->'changes')LOOP
  old:=NULL;o:=NULL;
  op:=ledger.text_value(z->'operation');ext:=banking.collection_text(z->'transaction_id',256);
  SELECT * INTO o FROM banking.collection_observation WHERE workspace_id=w AND source_account_id=sid AND external_entry_id=ext FOR UPDATE;
  IF NOT FOUND THEN
   IF op<>'added'THEN PERFORM banking.collection_error('P2804');END IF;
   INSERT INTO banking.collection_observation(workspace_id,source_account_id,external_entry_id)VALUES(w,sid,ext)RETURNING * INTO o;
  ELSE
   SELECT * INTO old FROM banking.collection_revision WHERE workspace_id=w AND source_account_id=sid AND observation_id=o.observation_id AND revision=o.current_revision;
  END IF;
  IF op='removed'THEN
   PERFORM banking.collection_keys(z,ARRAY['operation','transaction_id']);IF old.observation_id IS NULL THEN PERFORM banking.collection_error('P2804');END IF;
   IF old.source_state='removed'THEN CONTINUE;END IF;
   f:=old.normalized_fact||jsonb_build_object('source_state','removed','effective_state','removed');
  ELSIF op IN('added','modified')THEN
   PERFORM banking.collection_keys(z,ARRAY['operation','transaction_id','booked_date','description','merchant_name','source_state','currency_code','amount','direction']);f:=banking.collection_normalize_fact(z-ARRAY['operation','transaction_id']);
   IF old.observation_id IS NOT NULL THEN
    IF old.normalized_fact=f THEN CONTINUE;END IF;
    IF op='added'THEN PERFORM banking.collection_error('P2802');END IF;
   END IF;
  ELSE PERFORM banking.collection_error('P2800');END IF;
  rev:=CASE WHEN old.observation_id IS NULL THEN 1 ELSE old.revision+1 END;
  IF rev>1 THEN UPDATE banking.collection_observation SET current_revision=rev WHERE workspace_id=w AND source_account_id=sid AND observation_id=o.observation_id;END IF;
  INSERT INTO banking.collection_revision(workspace_id,source_account_id,observation_id,revision,previous_revision,booked_date,description,merchant_name,source_state,currency_code,currency_version_id,amount_lexeme,amount,direction,hold_reason,normalized_fact)
  VALUES(w,sid,o.observation_id,rev,CASE WHEN rev>1 THEN rev-1 END,(f->>'booked_date')::date,f->>'description',f->>'merchant_name',f->>'effective_state',f->>'currency_code',(f->>'currency_version_id')::uuid,f->>'amount',(f->>'validated_amount')::numeric,f->>'validated_direction',f->>'hold_reason',f);
  changed:=changed+1;old:=NULL;
 END LOOP;
 UPDATE banking.collection_source SET revision=revision+1,source_revision=source_revision+CASE WHEN changed>0 THEN 1 ELSE 0 END,cursor_value=next_value,readiness=ready,last_completed_at=clock_timestamp()WHERE workspace_id=w AND source_account_id=sid RETURNING * INTO s;
 res:=jsonb_build_object('source_account_id',sid,'source_revision',s.source_revision::text,'changed_count',changed::text,'readiness',s.readiness);
 INSERT INTO integration.collection_receipt(workspace_id,principal_id,operation,idempotency_key,request,result)VALUES(w,actor,'publish',k,norm,res);
 INSERT INTO audit.collection_action(workspace_id,principal_id,operation,target_id)VALUES(w,actor,'publish',sid);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
-- Local lifecycle fencing never asserts provider-side removal or deletes retained evidence.
CREATE FUNCTION banking.set_collection_state(actor uuid,w uuid,k text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE cw banking.collection_workspace;v jsonb;r integration.collection_receipt;id uuid;kind text;state_value text;expected bigint;i integration.collection_item;s banking.collection_source;res jsonb;BEGIN
 cw:=banking.collection_authorize(actor,w,'acquire',true);IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['kind','target_id','expected_revision','state']);id:=ledger.uuid_value(v->'target_id');kind:=ledger.text_value(v->'kind');state_value:=ledger.text_value(v->'state');expected:=banking.import_revision(v->'expected_revision');
 IF kind<>ALL(ARRAY['item','source'])OR state_value<>ALL(ARRAY['active','paused','revoked'])THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO r FROM integration.collection_receipt WHERE workspace_id=w AND principal_id=actor AND operation='state'AND idempotency_key=k;
 IF FOUND THEN
  IF r.request<>v THEN PERFORM banking.collection_error('P2802');END IF;
  IF kind='item'THEN SELECT * INTO i FROM integration.collection_item WHERE workspace_id=w AND collection_item_id=id FOR UPDATE;
   IF NOT FOUND OR i.state='revoked'AND state_value<>'revoked'THEN PERFORM banking.collection_error('P2801');END IF;
   IF i.state<>state_value OR i.generation::text<>r.result->>'revision'THEN PERFORM banking.collection_error('P2803');END IF;
  ELSE SELECT * INTO s FROM banking.collection_source WHERE workspace_id=w AND source_account_id=id FOR UPDATE;
   IF NOT FOUND OR s.state='revoked'AND state_value<>'revoked'THEN PERFORM banking.collection_error('P2801');END IF;
   IF s.state<>state_value OR s.revision::text<>r.result->>'revision'THEN PERFORM banking.collection_error('P2803');END IF;
  END IF;RETURN r.result;
 END IF;
 IF kind='item'THEN
  SELECT * INTO i FROM integration.collection_item WHERE workspace_id=w AND collection_item_id=id FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
  IF expected<>i.generation THEN PERFORM banking.collection_error('P2803');END IF;IF i.state='revoked'OR i.legacy_sync_item_id IS NOT NULL AND state_value='active'THEN PERFORM banking.collection_error('P2804');END IF;
  UPDATE integration.collection_item SET state=state_value,generation=generation+1 WHERE collection_item_id=id;
 ELSE
  SELECT * INTO s FROM banking.collection_source WHERE workspace_id=w AND source_account_id=id FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
  IF expected<>s.revision THEN PERFORM banking.collection_error('P2803');END IF;IF s.state<>'active'OR state_value<>'revoked'THEN PERFORM banking.collection_error('P2804');END IF;
  UPDATE banking.collection_source SET state='revoked',revision=revision+1 WHERE workspace_id=w AND source_account_id=id;
 END IF;
 res:=jsonb_build_object('target_id',id,'revision',(expected+1)::text,'state',state_value);INSERT INTO integration.collection_receipt(workspace_id,principal_id,operation,idempotency_key,request,result)VALUES(w,actor,'state',k,v,res);
 INSERT INTO audit.collection_action(workspace_id,principal_id,operation,target_id)VALUES(w,actor,'state',id);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
-- Explicit audited adoption references old tuples and fences old acquisition without history copies.
CREATE FUNCTION banking.adopt_collection_item(actor uuid,w uuid,e uuid,b uuid,item uuid,k text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE cw banking.collection_workspace;v jsonb;norm jsonb;r integration.collection_receipt;i integration.sync_item;ci integration.collection_item;t integration.sync_stream;s banking.source_account;z jsonb;label text;sid uuid;ev uuid;res jsonb;items jsonb:='[]';BEGIN
 IF NOT integration.provider_api_login(session_user)OR current_setting('transaction_isolation')<>'serializable'THEN PERFORM banking.collection_error('P2801');END IF;
 -- The legacy book lock precedes workspace because no private raw read takes that book lock.
 PERFORM ledger.authorize(actor,e,b,'read',true);
 PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='connect'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='import'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 cw:=banking.collection_authorize(actor,w,'acquire',true);IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 v:=banking.import_json(raw,16384);PERFORM banking.collection_keys(v,ARRAY['expected_generation','sources','evidence_ref']);PERFORM banking.import_revision(v->'expected_generation');ev:=ledger.uuid_value(v->'evidence_ref');
 IF jsonb_typeof(v->'sources')IS DISTINCT FROM 'array'OR jsonb_array_length(v->'sources')NOT BETWEEN 1 AND 25 THEN PERFORM banking.collection_error('P2800');END IF;
 norm:=jsonb_build_object('entity_id',e,'book_id',b,'sync_item_id',item,'body',v);
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=e AND book_id=b AND sync_item_id=item FOR UPDATE;IF NOT FOUND OR i.consent_principal<>actor THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO ci FROM integration.collection_item WHERE legacy_entity_id=e AND legacy_book_id=b AND legacy_sync_item_id=item FOR UPDATE;
 SELECT * INTO r FROM integration.collection_receipt WHERE workspace_id=w AND principal_id=actor AND operation='adopt'AND idempotency_key=k;
 IF FOUND THEN IF r.request<>norm THEN PERFORM banking.collection_error('P2802');END IF;IF ci.workspace_id IS DISTINCT FROM w OR ci.state='revoked'THEN PERFORM banking.collection_error('P2801');END IF;
  FOR z IN SELECT * FROM jsonb_array_elements(v->'sources')LOOP
   sid:=ledger.uuid_value(z->'source_account_id');s:=banking.import_authorize(actor,e,b,sid,false);
   PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=actor AND capability='import'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
   IF NOT EXISTS(SELECT FROM banking.collection_source WHERE workspace_id=w AND collection_item_id=ci.collection_item_id AND legacy_entity_id=e AND legacy_book_id=b AND legacy_source_account_id=sid AND state='active')THEN PERFORM banking.collection_error('P2801');END IF;
  END LOOP;RETURN r.result;END IF;
 IF ci.workspace_id IS NOT NULL THEN PERFORM banking.collection_error('P2802');END IF;IF i.generation<>banking.import_revision(v->'expected_generation')THEN PERFORM banking.collection_error('P2803');END IF;
 IF i.state NOT IN('active','paused')OR EXISTS(SELECT FROM integration.sync_set ss WHERE ss.entity_id=e AND ss.book_id=b AND ss.state='staging'AND EXISTS(SELECT FROM integration.sync_stream st WHERE st.entity_id=e AND st.book_id=b AND st.stream_id=ss.stream_id AND st.sync_item_id=item))OR EXISTS(SELECT FROM integration.connection_attempt a WHERE a.entity_id=e AND a.book_id=b AND a.sync_item_id=item AND a.state NOT IN('active','disconnected','abandoned'))THEN PERFORM banking.collection_error('P2804');END IF;
 IF jsonb_array_length(v->'sources')<>(SELECT count(*)FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=item AND state='active')OR EXISTS(SELECT x->>'source_account_id'FROM jsonb_array_elements(v->'sources')x GROUP BY x->>'source_account_id'HAVING count(*)>1)THEN PERFORM banking.collection_error('P2804');END IF;
 IF(SELECT count(*)FROM banking.collection_source WHERE workspace_id=w)+jsonb_array_length(v->'sources')>100 THEN PERFORM banking.collection_error('P2805');END IF;
 FOR z IN SELECT * FROM jsonb_array_elements(v->'sources')ORDER BY value->>'source_account_id'LOOP
  PERFORM banking.collection_keys(z,ARRAY['source_account_id','display_name']);sid:=ledger.uuid_value(z->'source_account_id');label:=ledger.setup_text(z->'display_name',200);
  s:=banking.import_authorize(actor,e,b,sid,false);
  PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=actor AND capability='import'AND revoked_at IS NULL FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
  SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=item AND source_account_id=sid AND state='active'FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2804');END IF;
 END LOOP;
 -- Fence old Item before assigning private authority; the immutable credential AAD stays in place.
 UPDATE integration.sync_item SET state='paused',generation=generation+1 WHERE entity_id=e AND book_id=b AND sync_item_id=item;
 UPDATE integration.collection_item SET workspace_id=w,state='paused',generation=generation+1 WHERE collection_item_id=ci.collection_item_id;
 FOR z IN SELECT * FROM jsonb_array_elements(v->'sources')LOOP
  sid:=ledger.uuid_value(z->'source_account_id');SELECT * INTO t FROM integration.sync_stream WHERE entity_id=e AND book_id=b AND sync_item_id=item AND source_account_id=sid AND state='active';
  INSERT INTO banking.collection_source(workspace_id,collection_item_id,account_id,display_name,origin,legacy_entity_id,legacy_book_id,legacy_source_account_id)
  VALUES(w,ci.collection_item_id,t.account_id,z->>'display_name','legacy',e,b,sid)RETURNING source_account_id INTO sid;items:=items||jsonb_build_array(jsonb_build_object('source_account_id',sid,'source_revision','0','revision','1'));
 END LOOP;
 res:=jsonb_build_object('collection_item_id',ci.collection_item_id,'item_generation',(ci.generation+1)::text,'sources',items);
 INSERT INTO integration.collection_receipt(workspace_id,principal_id,operation,idempotency_key,request,result)VALUES(w,actor,'adopt',k,norm,res);INSERT INTO audit.collection_action(workspace_id,principal_id,operation,target_id,evidence_ref)VALUES(w,actor,'adopt',ci.collection_item_id,ev);RETURN res;
EXCEPTION WHEN unique_violation THEN PERFORM banking.collection_error('P2802');WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2101'OR SQLSTATE 'P2102'OR SQLSTATE 'P2401'THEN PERFORM banking.collection_error('P2801');WHEN SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
-- Helpers and state remain inaccessible even when runtime receives only the closed entrypoints.
DO $$ DECLARE f regprocedure;BEGIN
 FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE(n.nspname IN('banking','integration'))AND(p.proname LIKE '%collection%')LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator',f);
 END LOOP;
END $$;
REVOKE ALL ON banking.collection_workspace,banking.collection_source,banking.collection_observation,banking.collection_revision,integration.collection_item,integration.collection_receipt,audit.collection_action FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION banking.get_collection_workspace(uuid),banking.list_collection_sources(uuid,uuid,text),banking.query_collection_observations(uuid,uuid,text),banking.register_collection_item(uuid,uuid,text,text),banking.publish_collection_batch(uuid,uuid,uuid,text,text),banking.set_collection_state(uuid,uuid,text,text),banking.adopt_collection_item(uuid,uuid,uuid,uuid,uuid,text,text)TO bookkeeping_api;
COMMIT;
