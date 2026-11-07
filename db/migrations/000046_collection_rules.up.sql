-- #414: private analytic rule editions and explicit selected application; no financial posting.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE banking.collection_workspace,banking.collection_label_state IN SHARE ROW EXCLUSIVE MODE;
CREATE TABLE banking.collection_rule_edition(
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,ruleset_revision bigint NOT NULL CHECK(ruleset_revision>=0),
 rules jsonb NOT NULL CHECK(jsonb_typeof(rules)='array'AND jsonb_array_length(rules)<=1000 AND octet_length(rules::text)<=262144),
 principal_id uuid REFERENCES ledger.principal,created_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(created_at)),PRIMARY KEY(workspace_id,ruleset_revision)
);
INSERT INTO banking.collection_rule_edition(workspace_id,ruleset_revision,rules)SELECT workspace_id,0,'[]'FROM banking.collection_workspace;
ALTER TABLE banking.collection_label_state ADD COLUMN ruleset_revision bigint NOT NULL DEFAULT 0 CHECK(ruleset_revision>=0);
ALTER TABLE banking.collection_label_state ADD CONSTRAINT collection_rule_current_fk FOREIGN KEY(workspace_id,ruleset_revision)REFERENCES banking.collection_rule_edition DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE banking.collection_label_preview DROP CONSTRAINT collection_label_preview_wire_version_check;
ALTER TABLE banking.collection_label_preview ADD CONSTRAINT collection_label_preview_wire_version_check CHECK(wire_version IN(1,2,3));
CREATE TABLE banking.collection_rule_preview_result(
 workspace_id uuid NOT NULL,preview_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,observation_revision bigint NOT NULL CHECK(observation_revision>0),
 body jsonb NOT NULL CHECK(jsonb_typeof(body)='object'AND octet_length(body::text)<=131072),
 PRIMARY KEY(workspace_id,preview_id,source_account_id,observation_id),FOREIGN KEY(workspace_id,preview_id)REFERENCES banking.collection_label_preview,FOREIGN KEY(workspace_id,source_account_id)REFERENCES banking.collection_source
);
CREATE TABLE banking.collection_rule_decision(
 workspace_id uuid NOT NULL,decision_revision bigint NOT NULL CHECK(decision_revision>0),source_account_id uuid NOT NULL,observation_id uuid NOT NULL,observation_revision bigint NOT NULL CHECK(observation_revision>0),
 ruleset_revision bigint NOT NULL,taxonomy_revision bigint NOT NULL,tag_revision bigint NOT NULL,command_id uuid NOT NULL,
 output jsonb NOT NULL CHECK(jsonb_typeof(output)='object'),rules jsonb NOT NULL CHECK(jsonb_typeof(rules)='array'AND jsonb_array_length(rules)BETWEEN 1 AND 100),snapshot jsonb NOT NULL,
 principal_id uuid NOT NULL REFERENCES ledger.principal,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),
 PRIMARY KEY(workspace_id,decision_revision,source_account_id,observation_id),FOREIGN KEY(workspace_id,source_account_id)REFERENCES banking.collection_source,
 FOREIGN KEY(workspace_id,ruleset_revision)REFERENCES banking.collection_rule_edition,FOREIGN KEY(workspace_id,taxonomy_revision)REFERENCES banking.collection_taxonomy_edition,FOREIGN KEY(workspace_id,tag_revision)REFERENCES banking.collection_tag_edition,FOREIGN KEY(workspace_id,command_id)REFERENCES banking.collection_label_command,
 CHECK(octet_length((output::text||rules::text||snapshot::text))<=131072)
);
CREATE INDEX collection_rule_latest ON banking.collection_rule_decision(workspace_id,source_account_id,observation_id,decision_revision DESC);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_rule_edition FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_rule_preview_result FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_rule_decision FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE FUNCTION banking.collection_rule_initialize()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 INSERT INTO banking.collection_rule_edition(workspace_id,ruleset_revision,rules)VALUES(NEW.workspace_id,0,'[]');RETURN NEW;
END $$;
CREATE TRIGGER collection_rule_initialize AFTER INSERT ON banking.collection_workspace FOR EACH ROW EXECUTE FUNCTION banking.collection_rule_initialize();
-- A rule edition advances the existing projection fence too; ordinary manual/source writers stay unchanged.
CREATE FUNCTION banking.collection_rule_state_guard()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF NEW.ruleset_revision<>OLD.ruleset_revision AND NOT(NEW.ruleset_revision=OLD.ruleset_revision+1 AND NEW.projection_revision=OLD.projection_revision+1 AND ROW(NEW.workspace_id,NEW.taxonomy_revision,NEW.tag_revision,NEW.decision_revision,NEW.command_revision)=ROW(OLD.workspace_id,OLD.taxonomy_revision,OLD.tag_revision,OLD.decision_revision,OLD.command_revision))THEN PERFORM banking.collection_error('P2802');END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_collection_rules BEFORE UPDATE ON banking.collection_label_state FOR EACH ROW EXECUTE FUNCTION banking.collection_rule_state_guard();
ALTER FUNCTION banking.collection_label_snapshot(uuid)RENAME TO collection_rule_legacy_snapshot_base;
CREATE FUNCTION banking.collection_label_snapshot(w uuid)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT CASE WHEN ruleset_revision=0 THEN banking.collection_rule_legacy_snapshot_base(w)ELSE jsonb_set(banking.collection_rule_legacy_snapshot_base(w),'{source_snapshot}',to_jsonb(encode(sha256(convert_to(jsonb_build_array(banking.collection_rule_legacy_snapshot_base(w)->>'source_snapshot',ruleset_revision::text)::text,'UTF8')),'hex')))END FROM banking.collection_label_state WHERE workspace_id=w
$$;
CREATE OR REPLACE FUNCTION banking.collection_query_snapshot(w uuid)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT banking.collection_tag_snapshot(w)||jsonb_build_object('ruleset_revision',ruleset_revision::text)FROM banking.collection_label_state WHERE workspace_id=w
$$;
CREATE OR REPLACE FUNCTION banking.collection_query_pins(v jsonb,current_value jsonb)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 PERFORM banking.import_revision(v->'ruleset_revision');PERFORM banking.collection_tag_pins(jsonb_set(v,'{ruleset_revision}','"0"'),jsonb_set(current_value,'{ruleset_revision}','"0"'));
 IF v<>current_value THEN PERFORM banking.collection_error('P2803');END IF;
END $$;
ALTER FUNCTION banking.get_collection_taxonomy_v3(uuid,uuid)RENAME TO collection_rule_catalog_base;
CREATE FUNCTION banking.get_collection_taxonomy_v3(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE res jsonb;BEGIN
 res:=banking.collection_rule_catalog_base(actor,w);
 SELECT res||jsonb_build_object('ruleset_revision',s.ruleset_revision::text,'rules',e.rules)INTO res FROM banking.collection_label_state s JOIN banking.collection_rule_edition e USING(workspace_id,ruleset_revision)WHERE s.workspace_id=w;
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;

-- Stored accepted output overlays only fields with no explicit manual history; query filters see this projection.
CREATE OR REPLACE FUNCTION banking.collection_query_rows(actor uuid,w uuid)
RETURNS TABLE(source_id uuid,observation_id uuid,observation_revision bigint,booked_date date,body jsonb)
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT r.source_id,r.observation_id,r.observation_revision,r.booked_date,r.body||jsonb_build_object(
 'institution',jsonb_build_object('state','unavailable'),'legal_owner',jsonb_build_object('state','unknown'),
 'owner_topic',CASE WHEN manual_topic.present OR d.decision_revision IS NULL THEN r.body->'owner_topic'WHEN d.observation_revision<>r.observation_revision THEN jsonb_build_object('state','stale')WHEN d.output->>'kind'='uncategorized'THEN jsonb_build_object('state','uncategorized')ELSE jsonb_build_object('state','topic','topic_id',d.output->>'topic_id','name',topic.value->>'name')END,
 'owner_tags',CASE WHEN manual_tags.present OR d.decision_revision IS NULL THEN r.body->'owner_tags'ELSE jsonb_build_object('state',CASE WHEN d.observation_revision<>r.observation_revision THEN 'stale'ELSE 'current'END,'tags',banking.collection_tag_names(w,ARRAY(SELECT value::uuid FROM jsonb_array_elements_text(d.output->'tag_ids'))))END,
 'rule_application',CASE WHEN d.decision_revision IS NULL THEN jsonb_build_object('state','none')ELSE jsonb_build_object('state',CASE WHEN manual_topic.present AND manual_tags.present THEN 'superseded'WHEN d.observation_revision<>r.observation_revision OR d.taxonomy_revision<>s.taxonomy_revision OR d.tag_revision<>s.tag_revision OR d.ruleset_revision<>s.ruleset_revision THEN 'stale'ELSE 'current'END,'ruleset_revision',d.ruleset_revision::text,'rules',d.rules,'fields',jsonb_build_object('topic',NOT manual_topic.present,'tags',NOT manual_tags.present))END)
 FROM banking.collection_tag_rows(actor,w)r JOIN banking.collection_label_state s ON s.workspace_id=w
 LEFT JOIN LATERAL(SELECT x.* FROM banking.collection_rule_decision x WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id ORDER BY decision_revision DESC LIMIT 1)d ON true
 CROSS JOIN LATERAL(SELECT EXISTS(SELECT FROM banking.collection_label_decision x WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id)present)manual_topic
 CROSS JOIN LATERAL(SELECT EXISTS(SELECT FROM banking.collection_tag_decision x WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id)present)manual_tags
 LEFT JOIN LATERAL(SELECT value FROM banking.collection_taxonomy_edition e,jsonb_array_elements(e.topics)t(value)WHERE e.workspace_id=w AND e.taxonomy_revision=s.taxonomy_revision AND value->>'topic_id'=d.output->>'topic_id')topic ON true
$$;

-- SQL validates the closed rule language/authority; only the bounded Go evaluator compiles regex and ranks.
CREATE FUNCTION banking.collection_rule_action(w uuid,v jsonb)RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE kind_value text;r jsonb;c jsonb;o jsonb;id uuid;account_value uuid;ids uuid[];x uuid;name_value text;status_value text;field_value text;operator_value text;condition_value text;rules_value jsonb:='[]';old_rules jsonb;BEGIN
 kind_value:=ledger.text_value(v->'kind');
 IF kind_value='rules_apply'THEN PERFORM banking.collection_keys(v,ARRAY['kind']);RETURN v;END IF;
 IF kind_value<>'rule_catalog'THEN RETURN banking.collection_tag_action(w,v);END IF;
 PERFORM banking.collection_keys(v,ARRAY['kind','rules']);IF jsonb_typeof(v->'rules')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_array_length(v->'rules')>1000 OR octet_length((v->'rules')::text)>262144 THEN PERFORM banking.collection_error('P2805');END IF;
 FOR r IN SELECT value FROM jsonb_array_elements(v->'rules')LOOP
  PERFORM banking.collection_keys(r,ARRAY['rule_id','name','status','source_account_id','priority','conditions','output']);id:=ledger.uuid_value(r->'rule_id');name_value:=ledger.text_value(r->'name');status_value:=ledger.text_value(r->'status');
  IF length(name_value)NOT BETWEEN 1 AND 80 OR btrim(name_value)=''OR name_value~'[[:cntrl:]]'OR status_value<>ALL(ARRAY['active','retired'])THEN PERFORM banking.collection_error('P2800');END IF;
  IF EXISTS(SELECT FROM jsonb_array_elements(rules_value)rule_row WHERE rule_row->>'rule_id'=id::text)THEN PERFORM banking.collection_error('P2800');END IF;
  IF jsonb_typeof(r->'priority')IS DISTINCT FROM 'number'OR r->>'priority'COLLATE "C" !~ '^-?(0|[1-9][0-9]{0,6})$'OR(r->>'priority')::integer NOT BETWEEN -1000000 AND 1000000 THEN PERFORM banking.collection_error('P2800');END IF;
  account_value:=NULL;IF r->'source_account_id'<>'null'::jsonb THEN account_value:=ledger.uuid_value(r->'source_account_id');IF NOT EXISTS(SELECT FROM banking.collection_source WHERE workspace_id=w AND source_account_id=account_value)OR(status_value='active'AND NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id=account_value))THEN PERFORM banking.collection_error('P2801');END IF;END IF;
  IF jsonb_typeof(r->'conditions')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;IF jsonb_array_length(r->'conditions')NOT BETWEEN 1 AND 8 THEN PERFORM banking.collection_error('P2805');END IF;
  FOR c IN SELECT value FROM jsonb_array_elements(r->'conditions')LOOP
   PERFORM banking.collection_keys(c,ARRAY['field','operator','value']);field_value:=ledger.text_value(c->'field');operator_value:=ledger.text_value(c->'operator');condition_value:=ledger.text_value(c->'value');
   IF octet_length(condition_value)NOT BETWEEN 1 AND 128 THEN PERFORM banking.collection_error('P2805');END IF;
   IF field_value IN('merchant','description')THEN IF operator_value<>ALL(ARRAY['exact','contains','regex'])THEN PERFORM banking.collection_error('P2800');END IF;
   ELSIF field_value='account'THEN x:=ledger.uuid_value(c->'value');IF operator_value<>'exact'THEN PERFORM banking.collection_error('P2800');END IF;IF NOT EXISTS(SELECT FROM banking.collection_source WHERE workspace_id=w AND source_account_id=x)THEN PERFORM banking.collection_error('P2801');END IF;
   ELSIF field_value='currency_version'THEN x:=ledger.uuid_value(c->'value');IF operator_value<>'exact'THEN PERFORM banking.collection_error('P2800');END IF;IF NOT EXISTS(SELECT FROM reference.currency WHERE currency_version_id=x)THEN PERFORM banking.collection_error('P2801');END IF;
   ELSIF field_value='direction'THEN IF operator_value<>'exact'OR condition_value<>ALL(ARRAY['inflow','outflow'])THEN PERFORM banking.collection_error('P2800');END IF;
   ELSIF field_value='date'THEN IF operator_value<>ALL(ARRAY['on_or_after','on_or_before'])THEN PERFORM banking.collection_error('P2800');END IF;PERFORM ledger.date_value(condition_value);
   ELSE PERFORM banking.collection_error('P2800');END IF;
  END LOOP;
  o:=r->'output';kind_value:=ledger.text_value(o->'kind');
  IF kind_value='topic'THEN PERFORM banking.collection_keys(o,ARRAY['kind','topic_id','tag_ids']);x:=ledger.uuid_value(o->'topic_id');
   IF NOT EXISTS(SELECT FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'topic_id'=x::text AND(status_value='retired'OR t->>'status'='active'))THEN PERFORM banking.collection_error('P2803');END IF;
  ELSIF kind_value='uncategorized'THEN PERFORM banking.collection_keys(o,ARRAY['kind','tag_ids']);ELSE PERFORM banking.collection_error('P2800');END IF;
  IF jsonb_typeof(o->'tag_ids')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;IF jsonb_array_length(o->'tag_ids')>20 THEN PERFORM banking.collection_error('P2805');END IF;
  ids:='{}';FOR c IN SELECT value FROM jsonb_array_elements(o->'tag_ids')LOOP
   x:=ledger.uuid_value(c);IF x=ANY(ids)THEN PERFORM banking.collection_error('P2800');END IF;ids:=array_append(ids,x);
   IF NOT EXISTS(SELECT FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision),jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND t->>'tag_id'=x::text AND(status_value='retired'OR t->>'status'='active'))THEN PERFORM banking.collection_error('P2803');END IF;
  END LOOP;
  rules_value:=rules_value||jsonb_build_array(r||jsonb_build_object('rule_id',id,'source_account_id',account_value,'output',o||jsonb_build_object('tag_ids',to_jsonb(banking.collection_tag_sorted(ids)))));
 END LOOP;
 IF(SELECT count(*)FROM jsonb_array_elements(rules_value)rule_row WHERE rule_row->>'status'='active')>100 THEN PERFORM banking.collection_error('P2805');END IF;
 IF EXISTS(SELECT FROM jsonb_array_elements(rules_value)rule_row WHERE rule_row->>'status'='active'GROUP BY rule_row->>'name'HAVING count(*)>1)THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT e.rules INTO old_rules FROM banking.collection_rule_edition e JOIN banking.collection_label_state s USING(workspace_id,ruleset_revision)WHERE e.workspace_id=w;
 IF EXISTS(SELECT FROM jsonb_array_elements(old_rules)rule_row WHERE NOT EXISTS(SELECT FROM jsonb_array_elements(rules_value)y WHERE y->>'rule_id'=rule_row->>'rule_id')OR(rule_row->>'status'='retired'AND EXISTS(SELECT FROM jsonb_array_elements(rules_value)y WHERE y->>'rule_id'=rule_row->>'rule_id'AND y->>'status'<>'retired')))THEN PERFORM banking.collection_error('P2802');END IF;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'rule_id'),'[]')INTO rules_value FROM jsonb_array_elements(rules_value);
 IF octet_length(rules_value::text)>262144 THEN PERFORM banking.collection_error('P2805');END IF;RETURN jsonb_build_object('kind','rule_catalog','rules',rules_value);
END $$;

-- Internal application context is authorized before raw-body parsing and never accepts caller facts/results.
CREATE FUNCTION banking.collection_rule_context(actor uuid,w uuid,raw text,writing boolean)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;snap jsonb;action_value jsonb;z jsonb;sid uuid;oid uuid;rev bigint;selected jsonb:='[]';row_cache jsonb:='[]';rows_value jsonb:='[]';facts_value jsonb:='[]';fact jsonb;topic_value jsonb;tags_value jsonb;rules_value jsonb;res jsonb;catalog_value jsonb;BEGIN
 IF writing IS NULL THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_label_authorize(actor,w,writing);v:=banking.import_json(raw,524288);PERFORM banking.collection_keys(v,ARRAY['snapshot','selection','action']);snap:=banking.collection_query_snapshot(w);PERFORM banking.collection_query_pins(v->'snapshot',snap);
 IF NOT writing AND v->'action'<>jsonb_build_object('kind','rules_apply')THEN PERFORM banking.collection_error('P2800');END IF;action_value:=banking.collection_rule_action(w,v->'action');
 IF jsonb_typeof(v->'selection')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_array_length(v->'selection')>100 OR(writing AND((action_value->>'kind'IN('taxonomy','tag_catalog','rule_catalog')AND jsonb_array_length(v->'selection')<>0)OR(action_value->>'kind'NOT IN('taxonomy','tag_catalog','rule_catalog')AND jsonb_array_length(v->'selection')=0)))THEN PERFORM banking.collection_error('P2805');END IF;
 SELECT coalesce(jsonb_agg(r.body),'[]')INTO row_cache FROM banking.collection_query_rows(actor,w)r WHERE EXISTS(SELECT FROM jsonb_array_elements(v->'selection')x WHERE(x->>'source_id')::uuid=r.source_id AND(x->>'observation_id')::uuid=r.observation_id);
 FOR z IN SELECT value FROM jsonb_array_elements(v->'selection')LOOP
  PERFORM banking.collection_keys(z,ARRAY['workspace_id','source_id','observation_id','observation_revision']);IF ledger.uuid_value(z->'workspace_id')<>w THEN PERFORM banking.collection_error('P2801');END IF;
  sid:=ledger.uuid_value(z->'source_id');oid:=ledger.uuid_value(z->'observation_id');rev:=banking.import_revision(z->'observation_revision');IF rev=0 THEN PERFORM banking.collection_error('P2800');END IF;
  IF EXISTS(SELECT FROM jsonb_array_elements(selected)x WHERE x->>'source_id'=sid::text AND x->>'observation_id'=oid::text)THEN PERFORM banking.collection_error('P2800');END IF;
  SELECT x INTO fact FROM jsonb_array_elements(row_cache)x WHERE x->>'source_id'=sid::text AND x->>'observation_id'=oid::text;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;IF fact->>'observation_revision'<>rev::text THEN PERFORM banking.collection_error('P2803');END IF;
  selected:=selected||jsonb_build_array(jsonb_build_object('workspace_id',w,'source_id',sid,'observation_id',oid,'observation_revision',rev::text));rows_value:=rows_value||jsonb_build_array(fact-ARRAY['institution','legal_owner','rule_application']);
  SELECT jsonb_build_object('observation_revision',d.observation_revision::text,'taxonomy_revision',d.taxonomy_revision::text,'kind',d.kind,'topic_id',d.topic_id)INTO topic_value FROM banking.collection_label_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid ORDER BY decision_revision DESC LIMIT 1;
  SELECT jsonb_build_object('observation_revision',d.observation_revision::text,'tag_revision',d.tag_revision::text,'tag_ids',to_jsonb(d.tag_ids))INTO tags_value FROM banking.collection_tag_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid ORDER BY decision_revision DESC LIMIT 1;
  facts_value:=facts_value||jsonb_build_array(jsonb_build_object('source_id',sid,'observation_id',oid,'observation_revision',rev::text,'booked_date',fact->'booked_date','merchant',fact->'merchant','description',fact->'description','currency_version_id',fact->'money'->'currency_version_id','direction',fact->'money'->'direction','source_state',fact->'source_state','money_state',fact->'money'->'state','manual_topic',topic_value,'manual_tags',tags_value));
 END LOOP;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO selected FROM jsonb_array_elements(selected);
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO rows_value FROM jsonb_array_elements(rows_value);
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO facts_value FROM jsonb_array_elements(facts_value);
 IF action_value->>'kind'='rule_catalog'THEN rules_value:=action_value->'rules';
  catalog_value:=banking.get_collection_taxonomy_v3(actor,w)||jsonb_build_object('rules',rules_value,'ruleset_revision',((snap->>'ruleset_revision')::bigint+1)::text);
  IF octet_length(catalog_value::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;
 ELSE SELECT e.rules INTO rules_value FROM banking.collection_rule_edition e JOIN banking.collection_label_state s USING(workspace_id,ruleset_revision)WHERE e.workspace_id=w;END IF;
 res:=jsonb_build_object('profile','private-rule-context-v1','principal_id',actor,'workspace_id',w,'snapshot',snap,'action',action_value,'selection',selected,'review_rows',rows_value,'rules',rules_value,'facts',facts_value,
 'topic_ids',(SELECT coalesce(jsonb_agg(t->'topic_id'ORDER BY t->>'topic_id'),'[]')FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'status'='active'),
 'tag_ids',(SELECT coalesce(jsonb_agg(t->'tag_id'ORDER BY t->>'tag_id'),'[]')FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision),jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND t->>'status'='active'));
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;

-- The matcher result is application-generated, but SQL still protects exact identities/manual fields/pins.
CREATE FUNCTION banking.collection_rule_result(w uuid,context_value jsonb,v jsonb)RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE sid uuid;oid uuid;rev bigint;fact jsonb;status_value text;preserve_topic boolean;preserve_tags boolean;stale boolean;o jsonb;p jsonb;rule_value jsonb;rules_value jsonb:='[]';id uuid;ids uuid[]:='{}';x jsonb;kind_value text;will_apply boolean:=false;snap jsonb:=context_value->'snapshot';BEGIN
 PERFORM banking.collection_keys(v,ARRAY['workspace_id','source_id','observation_id','observation_revision','status','output','rules','preserved','will_apply']);
 IF ledger.uuid_value(v->'workspace_id')<>w THEN PERFORM banking.collection_error('P2802');END IF;sid:=ledger.uuid_value(v->'source_id');oid:=ledger.uuid_value(v->'observation_id');rev:=banking.import_revision(v->'observation_revision');
 SELECT f INTO fact FROM jsonb_array_elements(context_value->'facts')f WHERE f->>'source_id'=sid::text AND f->>'observation_id'=oid::text AND f->>'observation_revision'=rev::text;
 IF NOT FOUND THEN PERFORM banking.collection_error('P2802');END IF;
 status_value:=ledger.text_value(v->'status');IF status_value<>ALL(ARRAY['rule','override','conflict','unclassified','stale_override','unavailable'])THEN PERFORM banking.collection_error('P2800');END IF;
 preserve_topic:=fact->'manual_topic'<>'null'::jsonb;preserve_tags:=fact->'manual_tags'<>'null'::jsonb;
 IF v->'preserved'<>jsonb_build_object('topic',preserve_topic,'tags',preserve_tags)OR jsonb_typeof(v->'will_apply')IS DISTINCT FROM 'boolean'THEN PERFORM banking.collection_error('P2802');END IF;
 stale:=(preserve_topic AND(fact->'manual_topic'->>'observation_revision'<>rev::text OR fact->'manual_topic'->>'taxonomy_revision'<>snap->>'taxonomy_revision'))OR(preserve_tags AND(fact->'manual_tags'->>'observation_revision'<>rev::text OR fact->'manual_tags'->>'tag_revision'<>snap->>'tag_revision'));
 IF(stale AND status_value<>'stale_override')OR(NOT stale AND preserve_topic AND preserve_tags AND status_value<>'override')OR(NOT stale AND NOT(preserve_topic AND preserve_tags)AND status_value IN('override','stale_override'))THEN PERFORM banking.collection_error('P2802');END IF;
 IF NOT stale AND NOT(preserve_topic AND preserve_tags)AND(fact->>'source_state'<>'booked'OR fact->>'money_state'<>'available'OR coalesce(fact->>'direction','')NOT IN('inflow','outflow'))AND status_value<>'unavailable'THEN PERFORM banking.collection_error('P2802');END IF;
 IF jsonb_typeof(v->'rules')IS DISTINCT FROM 'array'OR jsonb_array_length(v->'rules')>100 THEN PERFORM banking.collection_error('P2800');END IF;
 FOR p IN SELECT value FROM jsonb_array_elements(v->'rules')LOOP
  PERFORM banking.collection_keys(p,ARRAY['rule_id','rule_revision']);id:=ledger.uuid_value(p->'rule_id');IF banking.import_revision(p->'rule_revision')<>(snap->>'ruleset_revision')::bigint THEN PERFORM banking.collection_error('P2802');END IF;
  SELECT r INTO rule_value FROM jsonb_array_elements(context_value->'rules')r WHERE r->>'rule_id'=id::text AND r->>'status'='active'AND(r->'source_account_id'='null'::jsonb OR r->>'source_account_id'=sid::text);
  IF NOT FOUND OR EXISTS(SELECT FROM jsonb_array_elements(rules_value)duplicate_rule WHERE duplicate_rule->>'rule_id'=id::text)THEN PERFORM banking.collection_error('P2802');END IF;
  rules_value:=rules_value||jsonb_build_array(jsonb_build_object('rule_id',id,'rule_revision',snap->>'ruleset_revision'));
 END LOOP;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'rule_id'),'[]')INTO rules_value FROM jsonb_array_elements(rules_value);
 IF(status_value='rule'AND jsonb_array_length(rules_value)=0)OR(status_value='conflict'AND jsonb_array_length(rules_value)<2)OR(status_value NOT IN('rule','conflict')AND rules_value<>'[]'::jsonb)THEN PERFORM banking.collection_error('P2802');END IF;
 o:=v->'output';
 IF status_value IN('rule','override')THEN
  kind_value:=ledger.text_value(o->'kind');
  IF kind_value='topic'THEN PERFORM banking.collection_keys(o,ARRAY['kind','topic_id','tag_ids']);id:=ledger.uuid_value(o->'topic_id');
   IF NOT EXISTS(SELECT FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'topic_id'=id::text AND(preserve_topic OR t->>'status'='active'))THEN PERFORM banking.collection_error('P2803');END IF;
  ELSIF kind_value='uncategorized'THEN PERFORM banking.collection_keys(o,ARRAY['kind','tag_ids']);ELSE PERFORM banking.collection_error('P2800');END IF;
  IF preserve_topic AND(ROW(kind_value,o->>'topic_id')IS DISTINCT FROM ROW(fact->'manual_topic'->>'kind',fact->'manual_topic'->>'topic_id'))THEN PERFORM banking.collection_error('P2802');END IF;
  IF jsonb_typeof(o->'tag_ids')IS DISTINCT FROM 'array'OR jsonb_array_length(o->'tag_ids')>(CASE WHEN preserve_tags THEN 25 ELSE 20 END)THEN PERFORM banking.collection_error('P2805');END IF;
  FOR x IN SELECT value FROM jsonb_array_elements(o->'tag_ids')LOOP
   id:=ledger.uuid_value(x);IF id=ANY(ids)THEN PERFORM banking.collection_error('P2800');END IF;ids:=array_append(ids,id);
   IF NOT EXISTS(SELECT FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision),jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND t->>'tag_id'=id::text AND(preserve_tags OR t->>'status'='active'))THEN PERFORM banking.collection_error('P2803');END IF;
  END LOOP;
  o:=o||jsonb_build_object('tag_ids',to_jsonb(banking.collection_tag_sorted(ids)));
  IF preserve_tags AND o->'tag_ids'<>fact->'manual_tags'->'tag_ids'THEN PERFORM banking.collection_error('P2802');END IF;
  IF status_value='rule'AND EXISTS(SELECT FROM jsonb_array_elements(context_value->'rules')r WHERE EXISTS(SELECT FROM jsonb_array_elements(rules_value)winner WHERE winner->>'rule_id'=r->>'rule_id')AND((NOT preserve_topic AND((r->'output')-'tag_ids')<>(o-'tag_ids'))OR(NOT preserve_tags AND r->'output'->'tag_ids'<>o->'tag_ids')))THEN PERFORM banking.collection_error('P2802');END IF;
 ELSE IF o<>'null'::jsonb THEN PERFORM banking.collection_error('P2802');END IF;END IF;
 IF status_value='rule'THEN
  will_apply:=NOT EXISTS(SELECT FROM banking.collection_rule_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid AND d.decision_revision=(SELECT max(x.decision_revision)FROM banking.collection_rule_decision x WHERE x.workspace_id=w AND x.source_account_id=sid AND x.observation_id=oid)AND d.observation_revision=rev AND d.ruleset_revision=(snap->>'ruleset_revision')::bigint AND d.taxonomy_revision=(snap->>'taxonomy_revision')::bigint AND d.tag_revision=(snap->>'tag_revision')::bigint AND(preserve_topic OR d.output-'tag_ids'=o-'tag_ids')AND(preserve_tags OR d.output->'tag_ids'=o->'tag_ids')AND d.rules=rules_value);
 END IF;
 RETURN jsonb_build_object('workspace_id',w,'source_id',sid,'observation_id',oid,'observation_revision',rev::text,'status',status_value,'output',o,'rules',rules_value,'preserved',jsonb_build_object('topic',preserve_topic,'tags',preserve_tags),'will_apply',will_apply);
END $$;

CREATE FUNCTION banking.collection_rule_render_preview(p banking.collection_label_preview)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT CASE WHEN p.wire_version<>3 THEN banking.collection_tag_render_preview(p)ELSE banking.collection_tag_render_preview(p)||jsonb_build_object('profile','private-collection-preview-v3','rule_results',coalesce((SELECT jsonb_agg(r.body ORDER BY r.source_account_id,r.observation_id)FROM banking.collection_rule_preview_result r WHERE r.workspace_id=p.workspace_id AND r.preview_id=p.preview_id),'[]'),'apply_count',(SELECT count(*)::text FROM banking.collection_rule_preview_result r WHERE r.workspace_id=p.workspace_id AND r.preview_id=p.preview_id AND r.body->>'will_apply'='true'),'abstain_count',(SELECT count(*)::text FROM banking.collection_rule_preview_result r WHERE r.workspace_id=p.workspace_id AND r.preview_id=p.preview_id AND r.body->>'will_apply'='false'))END
$$;
CREATE FUNCTION banking.collection_rule_render_command(c banking.collection_label_command)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT jsonb_build_object('profile','private-collection-command-v3','workspace_id',c.workspace_id,'command',jsonb_set(banking.collection_label_render_command(c)->'command','{preview}',banking.collection_rule_render_preview(p)))FROM banking.collection_label_preview p WHERE p.workspace_id=c.workspace_id AND p.preview_id=c.preview_id
$$;

-- Persist only a complete bounded inert preview; SQL recomputes protected fields/eligibility before any insert.
CREATE FUNCTION banking.publish_collection_rule_preview(actor uuid,w uuid,raw text,results text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE context_value jsonb;incoming jsonb;normalized jsonb:='[]';z jsonb;r jsonb;p banking.collection_label_preview;res jsonb;BEGIN
 context_value:=banking.collection_rule_context(actor,w,raw,true);IF context_value->'action'->>'kind'NOT IN('rule_catalog','rules_apply')THEN PERFORM banking.collection_error('P2800');END IF;
 incoming:=banking.import_json(results,1048576);PERFORM banking.collection_keys(incoming,ARRAY['results']);incoming:=incoming->'results';IF jsonb_typeof(incoming)IS DISTINCT FROM 'array'OR jsonb_array_length(incoming)<>jsonb_array_length(context_value->'selection')THEN PERFORM banking.collection_error('P2802');END IF;
 FOR z IN SELECT value FROM jsonb_array_elements(incoming)LOOP
  r:=banking.collection_rule_result(w,context_value,z);IF EXISTS(SELECT FROM jsonb_array_elements(normalized)x WHERE x->>'source_id'=r->>'source_id'AND x->>'observation_id'=r->>'observation_id')THEN PERFORM banking.collection_error('P2802');END IF;normalized:=normalized||jsonb_build_array(r);
 END LOOP;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO normalized FROM jsonb_array_elements(normalized);
 p.workspace_id:=w;p.preview_id:=gen_random_uuid();p.principal_id:=actor;p.snapshot:=context_value->'snapshot';p.selection:=context_value->'selection';p.action:=context_value->'action';p.review_rows:=context_value->'review_rows';p.created_at:=clock_timestamp();p.expires_at:=p.created_at+interval '15 minutes';p.wire_version:=3;
 res:=banking.collection_label_render_preview(p)||jsonb_build_object('profile','private-collection-preview-v3','tag_results','[]'::jsonb,'rule_results',normalized,'apply_count',(SELECT count(*)::text FROM jsonb_array_elements(normalized)x WHERE x->>'will_apply'='true'),'abstain_count',(SELECT count(*)::text FROM jsonb_array_elements(normalized)x WHERE x->>'will_apply'='false'));
 -- Reserve envelope room for the complete durable command and its bounded terminal receipt.
 IF octet_length(res::text)>1040384 THEN PERFORM banking.collection_error('P2805');END IF;
 INSERT INTO banking.collection_label_preview VALUES(p.*);
 FOR r IN SELECT value FROM jsonb_array_elements(normalized)LOOP INSERT INTO banking.collection_rule_preview_result VALUES(w,p.preview_id,(r->>'source_id')::uuid,(r->>'observation_id')::uuid,(r->>'observation_revision')::bigint,r);END LOOP;
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id)VALUES(w,actor,'preview',p.preview_id);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;

-- Manual v3 tags start from the effective accepted set, preserving already-applied rule labels in the review.
CREATE FUNCTION banking.preview_collection_labels_v3(actor uuid,w uuid,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c jsonb;z jsonb;row_value jsonb;p banking.collection_label_preview;ids uuid[];requested uuid[];tag_results jsonb:='[]';res jsonb;BEGIN
 c:=banking.collection_rule_context(actor,w,raw,true);IF c->'action'->>'kind'IN('rule_catalog','rules_apply')THEN PERFORM banking.collection_error('P2804');END IF;
 IF c->'action'->>'kind'='tags'THEN
  requested:=ARRAY(SELECT value::uuid FROM jsonb_array_elements_text(c->'action'->'tag_ids'));
  FOR row_value IN SELECT value FROM jsonb_array_elements(c->'review_rows')LOOP
   IF c->'action'->>'operation'<>'replace'AND row_value->'owner_tags'->>'state'='stale'THEN PERFORM banking.collection_error('P2803');END IF;
   ids:=ARRAY(SELECT(t->>'tag_id')::uuid FROM jsonb_array_elements(row_value->'owner_tags'->'tags')t);
   IF c->'action'->>'operation'='replace'THEN ids:=requested;
   ELSIF c->'action'->>'operation'='add'THEN ids:=banking.collection_tag_sorted(ids||requested);
   ELSE ids:=ARRAY(SELECT x FROM unnest(ids)x WHERE NOT x=ANY(requested));END IF;
   ids:=banking.collection_tag_sorted(ids);IF cardinality(ids)>25 THEN PERFORM banking.collection_error('P2805');END IF;
   tag_results:=tag_results||jsonb_build_array(jsonb_build_object('workspace_id',w,'source_id',row_value->'source_id','observation_id',row_value->'observation_id','observation_revision',row_value->'observation_revision','tags',banking.collection_tag_names(w,ids),'tag_ids',to_jsonb(ids)));
  END LOOP;
 END IF;
 p.workspace_id:=w;p.preview_id:=gen_random_uuid();p.principal_id:=actor;p.snapshot:=c->'snapshot';p.selection:=c->'selection';p.action:=c->'action';p.review_rows:=c->'review_rows';p.created_at:=clock_timestamp();p.expires_at:=p.created_at+interval '15 minutes';p.wire_version:=3;
 res:=banking.collection_label_render_preview(p)||jsonb_build_object('profile','private-collection-preview-v3','tag_results',coalesce((SELECT jsonb_agg(wire_row-'tag_ids'ORDER BY wire_row->>'source_id',wire_row->>'observation_id')FROM jsonb_array_elements(tag_results)wire_row),'[]'),'rule_results','[]'::jsonb,'apply_count','0','abstain_count','0');
 IF octet_length(res::text)>1040384 THEN PERFORM banking.collection_error('P2805');END IF;
 INSERT INTO banking.collection_label_preview VALUES(p.*);
 FOR z IN SELECT value FROM jsonb_array_elements(tag_results)LOOP INSERT INTO banking.collection_tag_preview_result VALUES(w,p.preview_id,(z->>'source_id')::uuid,(z->>'observation_id')::uuid,(z->>'observation_revision')::bigint,ARRAY(SELECT value::uuid FROM jsonb_array_elements_text(z->'tag_ids')),z->'tags');END LOOP;
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id)VALUES(w,actor,'preview',p.preview_id);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;

CREATE FUNCTION banking.prepare_collection_label_command_v3(actor uuid,w uuid,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;id uuid;k text;p banking.collection_label_preview;c banking.collection_label_command;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);id:=ledger.uuid_value(v->'preview_id');k:=ledger.text_value(v->'prepare_key');IF k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=id AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;PERFORM banking.collection_tag_reauthorize(actor,w,p);
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND prepare_key=k;
 IF FOUND THEN IF c.preview_id<>id THEN PERFORM banking.collection_error('P2802');END IF;RETURN banking.collection_rule_render_command(c);END IF;
 IF EXISTS(SELECT FROM banking.collection_label_command WHERE workspace_id=w AND acknowledged_at IS NULL)THEN PERFORM banking.collection_error('P2802');END IF;
 IF p.wire_version=3 THEN PERFORM banking.collection_query_pins(p.snapshot,banking.collection_query_snapshot(w));
 ELSIF p.wire_version=2 THEN PERFORM banking.collection_tag_pins(p.snapshot,banking.collection_tag_snapshot(w));ELSE PERFORM banking.collection_label_pins(p.snapshot,banking.collection_label_snapshot(w));END IF;
 IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;
 c.workspace_id:=w;c.command_id:=gen_random_uuid();c.principal_id:=actor;c.preview_id:=id;c.prepare_key:=k;c.state:='prepared';c.created_at:=clock_timestamp();
 res:=banking.collection_rule_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;
 INSERT INTO banking.collection_label_command(workspace_id,command_id,principal_id,preview_id,prepare_key,created_at)VALUES(w,c.command_id,actor,id,k,c.created_at);
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,'prepare',id,c.command_id);RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;

CREATE FUNCTION banking.pending_collection_label_command_v3(actor uuid,w uuid)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE c banking.collection_label_command;p banking.collection_label_preview;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND acknowledged_at IS NULL;
 IF NOT FOUND THEN RETURN jsonb_build_object('profile','private-collection-command-v3','workspace_id',w,'command',NULL);END IF;
 IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND 'classify'=ANY(capabilities))THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_tag_reauthorize(actor,w,p);
 res:=banking.collection_rule_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;

CREATE FUNCTION banking.transition_collection_label_command_v3(actor uuid,w uuid,id uuid,operation text,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;c banking.collection_label_command;p banking.collection_label_preview;ls banking.collection_label_state;action_value jsonb;res jsonb;n bigint;catalog_value jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY[]::text[]);
 IF operation IS NULL OR operation<>ALL(ARRAY['execute','cancel','acknowledge'])THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND command_id=id AND principal_id=actor FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_tag_reauthorize(actor,w,p);
 IF p.wire_version<>3 THEN
  PERFORM banking.collection_rule_transition_v2_base(actor,w,id,operation,raw);SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND command_id=id;RETURN banking.collection_rule_render_command(c);
 END IF;
 IF operation='acknowledge'THEN
  IF c.state='prepared'THEN PERFORM banking.collection_error('P2802');END IF;
  IF c.acknowledged_at IS NULL THEN UPDATE banking.collection_label_command SET acknowledged_at=clock_timestamp()WHERE workspace_id=w AND command_id=id RETURNING * INTO c;INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,'acknowledge',p.preview_id,id);END IF;
 ELSIF c.state='prepared'THEN
  IF operation='execute'THEN
   PERFORM banking.collection_query_pins(p.snapshot,banking.collection_query_snapshot(w));IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;
   action_value:=banking.collection_rule_action(w,p.action);SELECT * INTO ls FROM banking.collection_label_state WHERE workspace_id=w;
   IF jsonb_array_length(p.selection)>0 AND EXISTS(
    WITH facts AS MATERIALIZED(SELECT source_account_id,observation_id,observation_revision FROM banking.collection_facts(w))
    SELECT FROM jsonb_array_elements(p.selection)z WHERE NOT EXISTS(SELECT FROM facts f WHERE f.source_account_id=(z->>'source_id')::uuid AND f.observation_id=(z->>'observation_id')::uuid AND f.observation_revision=(z->>'observation_revision')::bigint))THEN PERFORM banking.collection_error('P2803');END IF;
   IF action_value->>'kind'='rule_catalog'THEN
    catalog_value:=banking.get_collection_taxonomy_v3(actor,w)||jsonb_build_object('rules',action_value->'rules','ruleset_revision',(ls.ruleset_revision+1)::text);
    IF octet_length(catalog_value::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;
    INSERT INTO banking.collection_rule_edition(workspace_id,ruleset_revision,rules,principal_id)VALUES(w,ls.ruleset_revision+1,action_value->'rules',actor);
    UPDATE banking.collection_label_state SET ruleset_revision=ruleset_revision+1,projection_revision=projection_revision+1 WHERE workspace_id=w;
    res:=jsonb_build_object('ruleset_revision',(ls.ruleset_revision+1)::text);
   ELSIF action_value->>'kind'='rules_apply'THEN
    IF(SELECT count(*)FROM banking.collection_rule_preview_result WHERE workspace_id=w AND preview_id=p.preview_id)<>jsonb_array_length(p.selection)THEN PERFORM banking.collection_error('P2802');END IF;
    SELECT count(*)INTO n FROM banking.collection_rule_preview_result WHERE workspace_id=w AND preview_id=p.preview_id AND body->>'will_apply'='true';
    IF n>0 THEN
     INSERT INTO banking.collection_rule_decision(workspace_id,decision_revision,source_account_id,observation_id,observation_revision,ruleset_revision,taxonomy_revision,tag_revision,command_id,output,rules,snapshot,principal_id)
      SELECT w,ls.decision_revision+1,r.source_account_id,r.observation_id,r.observation_revision,ls.ruleset_revision,ls.taxonomy_revision,ls.tag_revision,id,r.body->'output',r.body->'rules',p.snapshot,actor FROM banking.collection_rule_preview_result r WHERE r.workspace_id=w AND r.preview_id=p.preview_id AND r.body->>'will_apply'='true';
     UPDATE banking.collection_label_state SET decision_revision=decision_revision+1 WHERE workspace_id=w;
    END IF;
    res:=jsonb_build_object('selected_count',jsonb_array_length(p.selection)::text,'apply_count',n::text,'abstain_count',(jsonb_array_length(p.selection)-n)::text,'ruleset_revision',ls.ruleset_revision::text,'decision_revision',(ls.decision_revision+CASE WHEN n>0 THEN 1 ELSE 0 END)::text);
   ELSIF action_value->>'kind'='tag_catalog'THEN
    INSERT INTO banking.collection_tag_edition(workspace_id,tag_revision,tags,principal_id)VALUES(w,ls.tag_revision+1,action_value->'tags',actor);UPDATE banking.collection_label_state SET tag_revision=tag_revision+1 WHERE workspace_id=w;res:=jsonb_build_object('tag_revision',(ls.tag_revision+1)::text);
   ELSIF action_value->>'kind'='tags'THEN
    IF(SELECT count(*)FROM banking.collection_tag_preview_result WHERE workspace_id=w AND preview_id=p.preview_id)<>jsonb_array_length(p.selection)THEN PERFORM banking.collection_error('P2802');END IF;
    INSERT INTO banking.collection_tag_decision(workspace_id,source_account_id,observation_id,observation_revision,decision_revision,tag_revision,command_id,operation,tag_ids,principal_id)
     SELECT w,r.source_account_id,r.observation_id,r.observation_revision,ls.decision_revision+1,ls.tag_revision,id,action_value->>'operation',r.tag_ids,actor FROM banking.collection_tag_preview_result r WHERE r.workspace_id=w AND r.preview_id=p.preview_id;
    UPDATE banking.collection_label_state SET decision_revision=decision_revision+1 WHERE workspace_id=w;res:=jsonb_build_object('decision_revision',(ls.decision_revision+1)::text,'selected_count',jsonb_array_length(p.selection)::text);
   ELSIF action_value->>'kind'='taxonomy'THEN
    INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics,principal_id)VALUES(w,ls.taxonomy_revision+1,action_value->'topics',actor);UPDATE banking.collection_label_state SET taxonomy_revision=taxonomy_revision+1 WHERE workspace_id=w;res:=jsonb_build_object('taxonomy_revision',(ls.taxonomy_revision+1)::text);
   ELSE
    INSERT INTO banking.collection_label_decision(workspace_id,source_account_id,observation_id,observation_revision,decision_revision,taxonomy_revision,command_id,kind,topic_id,principal_id)
     SELECT w,(x->>'source_id')::uuid,(x->>'observation_id')::uuid,(x->>'observation_revision')::bigint,ls.decision_revision+1,ls.taxonomy_revision,id,action_value->>'kind',(action_value->>'topic_id')::uuid,actor FROM jsonb_array_elements(p.selection)x;
    UPDATE banking.collection_label_state SET decision_revision=decision_revision+1 WHERE workspace_id=w;res:=jsonb_build_object('decision_revision',(ls.decision_revision+1)::text,'selected_count',jsonb_array_length(p.selection)::text);
   END IF;
   UPDATE banking.collection_label_command SET state='committed',result=res,terminal_at=clock_timestamp()WHERE workspace_id=w AND command_id=id RETURNING * INTO c;
  ELSE UPDATE banking.collection_label_command SET state='cancelled',result=jsonb_build_object('cancelled',true),terminal_at=clock_timestamp()WHERE workspace_id=w AND command_id=id RETURNING * INTO c;END IF;
  INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,operation,p.preview_id,id);
 END IF;
 res:=banking.collection_rule_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;

CREATE OR REPLACE FUNCTION banking.prepare_collection_label_command(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;p banking.collection_label_preview;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=ledger.uuid_value(v->'preview_id')AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;PERFORM banking.collection_label_reauthorize(actor,w,p);IF p.wire_version<>1 THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_tag_prepare_v1_base(actor,w,raw);
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE OR REPLACE FUNCTION banking.pending_collection_label_command(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_label_command;p banking.collection_label_preview;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND acknowledged_at IS NULL;
 IF FOUND THEN
  IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND 'classify'=ANY(capabilities))THEN PERFORM banking.collection_error('P2801');END IF;
  SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_label_reauthorize(actor,w,p);IF p.wire_version<>1 THEN PERFORM banking.collection_error('P2804');END IF;
 END IF;RETURN banking.collection_tag_pending_v1_base(actor,w);
END $$;

-- Legacy callers fail closed for a v3 saved intent instead of returning a coerced partial preview.
ALTER FUNCTION banking.prepare_collection_label_command_v2(uuid,uuid,text)RENAME TO collection_rule_prepare_v2_base;
ALTER FUNCTION banking.pending_collection_label_command_v2(uuid,uuid)RENAME TO collection_rule_pending_v2_base;
ALTER FUNCTION banking.transition_collection_label_command_v2(uuid,uuid,uuid,text,text)RENAME TO collection_rule_transition_v2_base;
CREATE FUNCTION banking.prepare_collection_label_command_v2(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE v jsonb;id uuid;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);id:=ledger.uuid_value(v->'preview_id');
 IF EXISTS(SELECT FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=id AND principal_id=actor AND wire_version=3)THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_rule_prepare_v2_base(actor,w,raw);
END $$;
CREATE FUNCTION banking.pending_collection_label_command_v2(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);
 IF EXISTS(SELECT FROM banking.collection_label_command c JOIN banking.collection_label_preview p USING(workspace_id,preview_id)WHERE c.workspace_id=w AND c.principal_id=actor AND c.acknowledged_at IS NULL AND p.wire_version=3)THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_rule_pending_v2_base(actor,w);
END $$;
CREATE FUNCTION banking.transition_collection_label_command_v2(actor uuid,w uuid,id uuid,operation text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);
 IF EXISTS(SELECT FROM banking.collection_label_command c JOIN banking.collection_label_preview p USING(workspace_id,preview_id)WHERE c.workspace_id=w AND c.command_id=id AND c.principal_id=actor AND p.wire_version=3)THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_rule_transition_v2_base(actor,w,id,operation,raw);
END $$;
CREATE OR REPLACE FUNCTION banking.transition_collection_label_command(actor uuid,w uuid,id uuid,operation text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE c banking.collection_label_command;p banking.collection_label_preview;v jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY[]::text[]);
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND command_id=id AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_tag_reauthorize(actor,w,p);IF p.wire_version<>1 THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_tag_transition_v1_base(actor,w,id,operation,raw);
END $$;

DO $$DECLARE f regprocedure;BEGIN
 FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='banking'AND(p.proname LIKE 'collection_rule_%'OR p.proname IN('get_collection_taxonomy_v3','preview_collection_labels_v3','prepare_collection_label_command_v3','pending_collection_label_command_v3','transition_collection_label_command_v3','publish_collection_rule_preview','prepare_collection_label_command_v2','pending_collection_label_command_v2','transition_collection_label_command_v2'))LOOP
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator',f);END LOOP;
END $$;
REVOKE ALL ON banking.collection_rule_edition,banking.collection_rule_decision,banking.collection_rule_preview_result FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION banking.get_collection_taxonomy_v3(uuid,uuid),banking.collection_rule_context(uuid,uuid,text,boolean),banking.publish_collection_rule_preview(uuid,uuid,text,text),banking.preview_collection_labels_v3(uuid,uuid,text),banking.prepare_collection_label_command_v3(uuid,uuid,text),banking.pending_collection_label_command_v3(uuid,uuid),banking.transition_collection_label_command_v3(uuid,uuid,uuid,text,text),banking.prepare_collection_label_command_v2(uuid,uuid,text),banking.pending_collection_label_command_v2(uuid,uuid),banking.transition_collection_label_command_v2(uuid,uuid,uuid,text,text)TO bookkeeping_api;
COMMIT;
