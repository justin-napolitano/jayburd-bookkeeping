-- #450/#451. Separate analytic tags and payload-free private command closure.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE ledger.principal,ledger.interactive_command_reservation,banking.collection_workspace,banking.collection_label_state IN SHARE ROW EXCLUSIVE MODE;
CREATE TABLE banking.collection_tag_edition(
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,tag_revision bigint NOT NULL CHECK(tag_revision>0),tags jsonb NOT NULL CHECK(jsonb_typeof(tags)='array'AND jsonb_array_length(tags)<=1000 AND octet_length(tags::text)<=524288),
 principal_id uuid REFERENCES ledger.principal,created_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(created_at)),PRIMARY KEY(workspace_id,tag_revision)
);
INSERT INTO banking.collection_tag_edition(workspace_id,tag_revision,tags)SELECT workspace_id,1,'[]'FROM banking.collection_workspace;
ALTER TABLE banking.collection_label_state ADD COLUMN tag_revision bigint NOT NULL DEFAULT 1 CHECK(tag_revision>0);
ALTER TABLE banking.collection_label_state ADD CONSTRAINT collection_label_tag_edition_fk FOREIGN KEY(workspace_id,tag_revision)REFERENCES banking.collection_tag_edition DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE banking.collection_label_preview ADD COLUMN wire_version smallint NOT NULL DEFAULT 1 CHECK(wire_version IN(1,2));
CREATE FUNCTION banking.collection_tag_sorted(ids uuid[])RETURNS uuid[]LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $$SELECT coalesce(array_agg(DISTINCT id ORDER BY id),'{}'::uuid[])FROM unnest(ids)id$$;
CREATE TABLE banking.collection_tag_decision(
 workspace_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,observation_revision bigint NOT NULL CHECK(observation_revision>0),decision_revision bigint NOT NULL CHECK(decision_revision>0),tag_revision bigint NOT NULL,command_id uuid NOT NULL,
 operation text NOT NULL CHECK(operation IN('add','remove','replace')),tag_ids uuid[]NOT NULL CHECK(cardinality(tag_ids)<=25 AND array_position(tag_ids,NULL)IS NULL AND tag_ids=banking.collection_tag_sorted(tag_ids)),
 principal_id uuid NOT NULL REFERENCES ledger.principal,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),PRIMARY KEY(workspace_id,decision_revision,source_account_id,observation_id),
 FOREIGN KEY(workspace_id,source_account_id)REFERENCES banking.collection_source,FOREIGN KEY(workspace_id,tag_revision)REFERENCES banking.collection_tag_edition,FOREIGN KEY(workspace_id,command_id)REFERENCES banking.collection_label_command
);
CREATE INDEX collection_tag_latest ON banking.collection_tag_decision(workspace_id,source_account_id,observation_id,decision_revision DESC);
CREATE TABLE banking.collection_tag_preview_result(
 workspace_id uuid NOT NULL,preview_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,observation_revision bigint NOT NULL CHECK(observation_revision>0),tag_ids uuid[]NOT NULL CHECK(cardinality(tag_ids)<=25 AND array_position(tag_ids,NULL)IS NULL AND tag_ids=banking.collection_tag_sorted(tag_ids)),tags jsonb NOT NULL CHECK(jsonb_typeof(tags)='array'AND jsonb_array_length(tags)<=25),
 PRIMARY KEY(workspace_id,preview_id,source_account_id,observation_id),FOREIGN KEY(workspace_id,preview_id)REFERENCES banking.collection_label_preview,FOREIGN KEY(workspace_id,source_account_id)REFERENCES banking.collection_source
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_tag_edition FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_tag_decision FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_tag_preview_result FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE OR REPLACE FUNCTION banking.collection_label_initialize()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics)VALUES(NEW.workspace_id,1,'[]');
 INSERT INTO banking.collection_label_state(workspace_id)VALUES(NEW.workspace_id);INSERT INTO banking.collection_tag_edition(workspace_id,tag_revision,tags)VALUES(NEW.workspace_id,1,'[]');RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION banking.collection_label_guard()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF TG_OP='DELETE'THEN PERFORM banking.collection_error('P2802');END IF;
 IF TG_TABLE_NAME='collection_label_state'THEN
  IF NEW.workspace_id<>OLD.workspace_id OR NOT((NEW.tag_revision=OLD.tag_revision AND(
   (NEW.command_revision=OLD.command_revision AND((NEW.taxonomy_revision=OLD.taxonomy_revision+1 AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision)OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision+1 AND NEW.projection_revision=OLD.projection_revision)OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision+1)))
   OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision AND NEW.command_revision=OLD.command_revision+1)))OR(NEW.tag_revision=OLD.tag_revision+1 AND NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision AND NEW.command_revision=OLD.command_revision))THEN PERFORM banking.collection_error('P2802');END IF;
 ELSE
  IF(to_jsonb(NEW)-ARRAY['state','result','terminal_at','acknowledged_at'])IS DISTINCT FROM(to_jsonb(OLD)-ARRAY['state','result','terminal_at','acknowledged_at'])THEN PERFORM banking.collection_error('P2802');END IF;
  IF OLD.state='prepared'THEN
   IF NEW.state NOT IN('committed','cancelled')OR NEW.terminal_at IS NULL OR NEW.result IS NULL OR NEW.acknowledged_at IS NOT NULL THEN PERFORM banking.collection_error('P2802');END IF;
  ELSE
   IF ROW(NEW.state,NEW.result,NEW.terminal_at)IS DISTINCT FROM ROW(OLD.state,OLD.result,OLD.terminal_at)OR OLD.acknowledged_at IS NOT NULL OR NEW.acknowledged_at IS NULL THEN PERFORM banking.collection_error('P2802');END IF;
  END IF;
 END IF;RETURN NEW;
END $$;
-- Preserve original v1 pins when the newly introduced tag edition is still empty1.
ALTER FUNCTION banking.collection_label_snapshot(uuid)RENAME TO collection_tag_snapshot_v1_base;
CREATE FUNCTION banking.collection_label_snapshot(w uuid)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT CASE WHEN tag_revision=1 THEN banking.collection_tag_snapshot_v1_base(w)ELSE jsonb_set(banking.collection_tag_snapshot_v1_base(w),'{source_snapshot}',to_jsonb(encode(sha256(convert_to(jsonb_build_array(banking.collection_tag_snapshot_v1_base(w)->>'source_snapshot',tag_revision::text)::text,'UTF8')),'hex')))END FROM banking.collection_label_state WHERE workspace_id=w
$$;
CREATE FUNCTION banking.collection_tag_snapshot(w uuid)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$SELECT banking.collection_label_snapshot(w)||jsonb_build_object('tag_revision',tag_revision::text)FROM banking.collection_label_state WHERE workspace_id=w$$;
CREATE FUNCTION banking.collection_tag_pins(v jsonb,current_value jsonb)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 PERFORM banking.collection_keys(v,ARRAY['authority_revision','source_snapshot','taxonomy_revision','ruleset_revision','map_revision','decision_revision','tag_revision']);PERFORM banking.import_revision(v->'tag_revision');PERFORM banking.collection_label_pins(v-'tag_revision',current_value-'tag_revision');IF v<>current_value THEN PERFORM banking.collection_error('P2803');END IF;
END $$;
-- Full current tag set projection remains one row per observation.
CREATE FUNCTION banking.collection_tag_names(w uuid,ids uuid[],version bigint DEFAULT NULL)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT coalesce(jsonb_agg(t ORDER BY t->>'tag_id'),'[]')FROM banking.collection_tag_edition e,jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND e.tag_revision=coalesce(version,(SELECT tag_revision FROM banking.collection_label_state WHERE workspace_id=w))AND(t->>'tag_id')::uuid=ANY(ids)
$$;
CREATE FUNCTION banking.collection_tag_rows(actor uuid,w uuid)RETURNS TABLE(source_id uuid,observation_id uuid,observation_revision bigint,booked_date date,body jsonb)LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT r.source_id,r.observation_id,r.observation_revision,r.booked_date,r.body||jsonb_build_object('owner_tags',jsonb_build_object('state',CASE WHEN d.decision_revision IS NULL THEN 'unclassified'WHEN d.observation_revision<>r.observation_revision THEN 'stale'ELSE 'current'END,'tags',banking.collection_tag_names(w,coalesce(d.tag_ids,'{}'))))
 FROM banking.collection_label_rows(actor,w)r LEFT JOIN LATERAL(SELECT x.* FROM banking.collection_tag_decision x WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id ORDER BY decision_revision DESC LIMIT 1)d ON true
$$;
CREATE FUNCTION banking.get_collection_taxonomy_v2(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE v jsonb;BEGIN
 v:=banking.get_collection_taxonomy(actor,w);v:=v||jsonb_build_object('profile','private-collection-catalog-v2','tag_revision',(SELECT tag_revision::text FROM banking.collection_label_state WHERE workspace_id=w),'tags',(SELECT e.tags FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision)WHERE e.workspace_id=w));
 IF octet_length(v::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN v;
END $$;
CREATE FUNCTION banking.query_collection_review_v2(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;sel jsonb;snap jsonb;sid uuid;first_day date;last_day date;search_value text;assignment_value text;sort_value text;topic_kind text;tid uuid;tag_kind text;tagid uuid;cur jsonb;txt text;page_offset bigint:=0;res jsonb;total bigint;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);v:=banking.import_json(raw,8192);
 PERFORM banking.collection_keys(v,ARRAY['profile','source_account_id','assignment','starts_on','through_date','search','topic','tag','sort','snapshot','cursor','limit']);
 IF ledger.text_value(v->'profile')<>'private-collection-review-v2'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_typeof(v->'limit')IS DISTINCT FROM 'number'OR v->>'limit'!~ '^[1-9][0-9]{0,2}$'OR(v->>'limit')::integer>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'source_account_id'<>'null'::jsonb THEN sid:=ledger.uuid_value(v->'source_account_id');IF NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id=sid)THEN PERFORM banking.collection_error('P2801');END IF;END IF;
 assignment_value:=ledger.text_value(v->'assignment');sort_value:=ledger.text_value(v->'sort');search_value:=ledger.text_value(v->'search');
 IF assignment_value<>ALL(ARRAY['all','assigned','unassigned'])OR sort_value<>ALL(ARRAY['date_desc','date_asc'])OR length(search_value)>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'starts_on'<>'null'::jsonb THEN first_day:=ledger.date_value(ledger.text_value(v->'starts_on'));END IF;IF v->'through_date'<>'null'::jsonb THEN last_day:=ledger.date_value(ledger.text_value(v->'through_date'));END IF;IF first_day>last_day THEN PERFORM banking.collection_error('P2800');END IF;
 topic_kind:=ledger.text_value(v->'topic'->'kind');
 IF topic_kind='topic'THEN PERFORM banking.collection_keys(v->'topic',ARRAY['kind','topic_id']);tid:=ledger.uuid_value(v->'topic'->'topic_id');
  IF NOT EXISTS(SELECT FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'topic_id'=tid::text)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'topic',ARRAY['kind']);IF topic_kind<>ALL(ARRAY['all','unclassified','uncategorized'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 tag_kind:=ledger.text_value(v->'tag'->'kind');
 IF tag_kind='tag'THEN PERFORM banking.collection_keys(v->'tag',ARRAY['kind','tag_id']);tagid:=ledger.uuid_value(v->'tag'->'tag_id');IF NOT EXISTS(SELECT FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision),jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND t->>'tag_id'=tagid::text)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'tag',ARRAY['kind']);IF tag_kind<>ALL(ARRAY['all','untagged'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 snap:=banking.collection_tag_snapshot(w);IF v->'snapshot'<>'null'::jsonb THEN PERFORM banking.collection_tag_pins(v->'snapshot',snap);END IF;
 sel:=v-ARRAY['profile','snapshot','cursor','limit'];
 IF v->'cursor'<>'null'::jsonb THEN
  txt:=ledger.text_value(v->'cursor');IF length(txt)NOT BETWEEN 2 AND 4096 OR txt COLLATE "C" !~ '^([0-9a-f]{2})+$'THEN PERFORM banking.collection_error('P2800');END IF;
  cur:=banking.import_json(convert_from(decode(txt,'hex'),'UTF8'),2048);PERFORM banking.collection_keys(cur,ARRAY['workspace_id','snapshot','selector','offset']);
  IF ledger.uuid_value(cur->'workspace_id')<>w OR cur->'selector'<>sel THEN PERFORM banking.collection_error('P2800');END IF;
  PERFORM banking.collection_tag_pins(cur->'snapshot',snap);page_offset:=banking.import_revision(cur->'offset');IF page_offset=0 THEN PERFORM banking.collection_error('P2800');END IF;
 END IF;
 WITH matched AS MATERIALIZED(SELECT f.* FROM banking.collection_tag_rows(actor,w)f WHERE(sid IS NULL OR f.source_id=sid)AND(first_day IS NULL OR f.booked_date>=first_day)AND(last_day IS NULL OR f.booked_date<=last_day)AND strpos(lower(f.body->>'description'COLLATE "C"),lower(search_value COLLATE "C"))>0
 AND(assignment_value='all'OR f.body->'assignment'->>'state'=assignment_value)
 AND(topic_kind='all'OR topic_kind='unclassified'AND f.body->'owner_topic'->>'state'IN('unclassified','stale')OR topic_kind='uncategorized'AND f.body->'owner_topic'->>'state'='uncategorized'OR topic_kind='topic'AND f.body->'owner_topic'->>'state'='topic'AND f.body->'owner_topic'->>'topic_id'=tid::text)
 AND(tag_kind='all'OR tag_kind='untagged'AND(f.body->'owner_tags'->>'state'<>'current'OR jsonb_array_length(f.body->'owner_tags'->'tags')=0)OR tag_kind='tag'AND f.body->'owner_tags'->>'state'='current'AND EXISTS(SELECT FROM jsonb_array_elements(f.body->'owner_tags'->'tags')t WHERE t->>'tag_id'=tagid::text))),
 numbered AS MATERIALIZED(SELECT f.*,row_number()OVER(ORDER BY CASE WHEN sort_value='date_desc'THEN booked_date END DESC,CASE WHEN sort_value='date_asc'THEN booked_date END ASC,source_id,observation_id)rn FROM matched f),
 picked AS MATERIALIZED(SELECT * FROM numbered WHERE rn>page_offset ORDER BY rn LIMIT(v->>'limit')::integer)
 SELECT jsonb_build_object('profile','private-collection-review-v2','workspace_id',w,'selector',sel,'snapshot',snap,'matching_count',(SELECT count(*)::text FROM matched),'start_index',CASE WHEN EXISTS(SELECT FROM picked)THEN(page_offset+1)::text ELSE '0'END,'items',coalesce((SELECT jsonb_agg(body ORDER BY rn)FROM picked),'[]'),
 'next_cursor',CASE WHEN(SELECT max(rn)FROM picked)<(SELECT count(*)FROM matched)THEN encode(convert_to(jsonb_build_object('workspace_id',w,'snapshot',snap,'selector',sel,'offset',(SELECT max(rn)::text FROM picked))::text,'UTF8'),'hex')END,'partial',EXISTS(SELECT FROM banking.collection_visible_sources(w)coverage WHERE(sid IS NULL OR coverage.source_account_id=sid)AND(coverage.origin='legacy'OR coverage.readiness<>'HISTORICAL_UPDATE_COMPLETE'))),(SELECT count(*)FROM matched)INTO res,total;
 IF page_offset>0 AND page_offset>=total THEN PERFORM banking.collection_error('P2800');END IF;
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.collection_error('P2800');END $$;
-- Closed actions normalize stable catalog identities and explicit bounded tag sets.
CREATE FUNCTION banking.collection_tag_action(w uuid,v jsonb)RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE kind_value text;z jsonb;id uuid;name_value text;status_value text;tags_value jsonb:='[]';oldtags jsonb;ids uuid[]:='{}';operation_value text;BEGIN
 kind_value:=ledger.text_value(v->'kind');IF kind_value IN('topic','uncategorized','taxonomy')THEN RETURN banking.collection_label_action(w,v);END IF;
 SELECT e.tags INTO oldtags FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision)WHERE e.workspace_id=w;
 IF kind_value='tags'THEN
  PERFORM banking.collection_keys(v,ARRAY['kind','operation','tag_ids']);operation_value:=ledger.text_value(v->'operation');IF operation_value<>ALL(ARRAY['add','remove','replace'])OR jsonb_typeof(v->'tag_ids')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;
  IF jsonb_array_length(v->'tag_ids')>25 OR operation_value<>'replace'AND jsonb_array_length(v->'tag_ids')=0 THEN PERFORM banking.collection_error('P2805');END IF;
  FOR z IN SELECT value FROM jsonb_array_elements(v->'tag_ids')LOOP
   id:=ledger.uuid_value(z);IF id=ANY(ids)THEN PERFORM banking.collection_error('P2800');END IF;
   IF NOT EXISTS(SELECT FROM jsonb_array_elements(oldtags)t WHERE t->>'tag_id'=id::text)THEN PERFORM banking.collection_error('P2801');END IF;
   IF operation_value<>'remove'AND NOT EXISTS(SELECT FROM jsonb_array_elements(oldtags)t WHERE t->>'tag_id'=id::text AND t->>'status'='active')THEN PERFORM banking.collection_error('P2803');END IF;ids:=array_append(ids,id);
  END LOOP;RETURN jsonb_build_object('kind','tags','operation',operation_value,'tag_ids',to_jsonb(banking.collection_tag_sorted(ids)));
 ELSIF kind_value<>'tag_catalog'THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_keys(v,ARRAY['kind','tags']);IF jsonb_typeof(v->'tags')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;IF jsonb_array_length(v->'tags')>1000 THEN PERFORM banking.collection_error('P2805');END IF;
 FOR z IN SELECT value FROM jsonb_array_elements(v->'tags')LOOP
  PERFORM banking.collection_keys(z,ARRAY['tag_id','name','status']);id:=ledger.uuid_value(z->'tag_id');name_value:=ledger.text_value(z->'name');status_value:=ledger.text_value(z->'status');
  IF length(name_value)NOT BETWEEN 1 AND 80 OR btrim(name_value)=''OR name_value~'[[:cntrl:]]'OR status_value<>ALL(ARRAY['active','retired'])THEN PERFORM banking.collection_error('P2800');END IF;
  tags_value:=tags_value||jsonb_build_array(jsonb_build_object('tag_id',id,'name',name_value,'status',status_value));
 END LOOP;
 IF(SELECT count(*)FROM jsonb_array_elements(tags_value)t WHERE t->>'status'='active')>500 THEN PERFORM banking.collection_error('P2805');END IF;
 IF EXISTS(SELECT t->>'tag_id'FROM jsonb_array_elements(tags_value)t GROUP BY t->>'tag_id'HAVING count(*)>1)OR EXISTS(SELECT t->>'name'COLLATE "C" FROM jsonb_array_elements(tags_value)t GROUP BY t->>'name'COLLATE "C" HAVING count(*)>1)THEN PERFORM banking.collection_error('P2800');END IF;
 IF EXISTS(SELECT FROM jsonb_array_elements(oldtags)o WHERE NOT EXISTS(SELECT FROM jsonb_array_elements(tags_value)t WHERE t->>'tag_id'=o->>'tag_id'AND(o->>'status'<>'retired'OR t->>'status'='retired')))THEN PERFORM banking.collection_error('P2802');END IF;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'tag_id'),'[]')INTO tags_value FROM jsonb_array_elements(tags_value);RETURN jsonb_build_object('kind','tag_catalog','tags',tags_value);
END $$;
-- Stale evidence requires an explicit whole-set replacement, never implicit inherited approval.
CREATE FUNCTION banking.collection_tag_result(w uuid,sid uuid,oid uuid,rev bigint,action jsonb)RETURNS uuid[]LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE old banking.collection_tag_decision;requested uuid[];result_ids uuid[];BEGIN
 SELECT * INTO old FROM banking.collection_tag_decision WHERE workspace_id=w AND source_account_id=sid AND observation_id=oid ORDER BY decision_revision DESC LIMIT 1;
 IF FOUND AND old.observation_revision<>rev AND action->>'operation'<>'replace'THEN PERFORM banking.collection_error('P2803');END IF;
 SELECT coalesce(array_agg(value::uuid),'{}')INTO requested FROM jsonb_array_elements_text(action->'tag_ids');
 IF action->>'operation'='replace'THEN result_ids:=requested;
 ELSIF action->>'operation'='add'THEN result_ids:=coalesce(old.tag_ids,'{}')||requested;
 ELSE SELECT coalesce(array_agg(x),'{}')INTO result_ids FROM unnest(coalesce(old.tag_ids,'{}'))x WHERE NOT x=ANY(requested);END IF;
 result_ids:=banking.collection_tag_sorted(result_ids);IF cardinality(result_ids)>25 THEN PERFORM banking.collection_error('P2805');END IF;RETURN result_ids;
END $$;
-- Historic v1 previews remain byte-meaning compatible inside an explicit v2 union.
CREATE FUNCTION banking.collection_tag_render_preview(p banking.collection_label_preview)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT CASE WHEN p.wire_version=1 THEN banking.collection_label_render_preview(p)ELSE banking.collection_label_render_preview(p)||jsonb_build_object('profile','private-collection-preview-v2','tag_results',coalesce((SELECT jsonb_agg(jsonb_build_object('workspace_id',r.workspace_id,'source_id',r.source_account_id,'observation_id',r.observation_id,'observation_revision',r.observation_revision::text,'tags',r.tags)ORDER BY r.source_account_id,r.observation_id)FROM banking.collection_tag_preview_result r WHERE r.workspace_id=p.workspace_id AND r.preview_id=p.preview_id),'[]'))END
$$;
CREATE FUNCTION banking.collection_tag_render_command(c banking.collection_label_command)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT jsonb_build_object('profile','private-collection-command-v2','workspace_id',c.workspace_id,'command',jsonb_set(banking.collection_label_render_command(c)->'command','{preview}',banking.collection_tag_render_preview(p)))FROM banking.collection_label_preview p WHERE p.workspace_id=c.workspace_id AND p.preview_id=c.preview_id
$$;
CREATE FUNCTION banking.preview_collection_labels_v2(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;snap jsonb;action_value jsonb;z jsonb;sid uuid;oid uuid;rev bigint;selection_value jsonb:='[]';rows_value jsonb:='[]';fact jsonb;p banking.collection_label_preview;res jsonb;result_ids uuid[];result_rows jsonb:='[]';result_row jsonb;row_cache jsonb:='[]';BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,524288);PERFORM banking.collection_keys(v,ARRAY['snapshot','selection','action']);snap:=banking.collection_tag_snapshot(w);PERFORM banking.collection_tag_pins(v->'snapshot',snap);action_value:=banking.collection_tag_action(w,v->'action');
 IF jsonb_typeof(v->'selection')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;
 IF(action_value->>'kind'IN('taxonomy','tag_catalog')AND jsonb_array_length(v->'selection')<>0)OR(action_value->>'kind'NOT IN('taxonomy','tag_catalog')AND jsonb_array_length(v->'selection')NOT BETWEEN 1 AND 100)THEN PERFORM banking.collection_error('P2805');END IF;
 -- Render the authorized selected rows once, not once per selected identity.
 IF jsonb_array_length(v->'selection')>0 THEN
  SELECT coalesce(jsonb_agg(f.body),'[]')INTO row_cache FROM banking.collection_tag_rows(actor,w)f WHERE EXISTS(SELECT FROM jsonb_array_elements(v->'selection')x WHERE (x->>'source_id')::uuid=f.source_id AND(x->>'observation_id')::uuid=f.observation_id);
 END IF;
 FOR z IN SELECT value FROM jsonb_array_elements(v->'selection')LOOP
  PERFORM banking.collection_keys(z,ARRAY['workspace_id','source_id','observation_id','observation_revision']);IF ledger.uuid_value(z->'workspace_id')<>w THEN PERFORM banking.collection_error('P2801');END IF;
  sid:=ledger.uuid_value(z->'source_id');oid:=ledger.uuid_value(z->'observation_id');rev:=banking.import_revision(z->'observation_revision');IF rev=0 THEN PERFORM banking.collection_error('P2800');END IF;
  IF EXISTS(SELECT FROM jsonb_array_elements(selection_value)x WHERE x->>'source_id'=sid::text AND x->>'observation_id'=oid::text)THEN PERFORM banking.collection_error('P2800');END IF;
  SELECT x INTO fact FROM jsonb_array_elements(row_cache)x WHERE x->>'source_id'=sid::text AND x->>'observation_id'=oid::text;
  IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;IF fact->>'observation_revision'<>rev::text THEN PERFORM banking.collection_error('P2803');END IF;
  IF action_value->>'kind'='tags'THEN result_ids:=banking.collection_tag_result(w,sid,oid,rev,action_value);result_rows:=result_rows||jsonb_build_array(jsonb_build_object('source_id',sid,'observation_id',oid,'observation_revision',rev::text,'tag_ids',to_jsonb(result_ids),'tags',banking.collection_tag_names(w,result_ids)));END IF;
  selection_value:=selection_value||jsonb_build_array(jsonb_build_object('workspace_id',w,'source_id',sid,'observation_id',oid,'observation_revision',rev::text));rows_value:=rows_value||jsonb_build_array(fact);
 END LOOP;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO selection_value FROM jsonb_array_elements(selection_value);
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO rows_value FROM jsonb_array_elements(rows_value);
 INSERT INTO banking.collection_label_preview(workspace_id,principal_id,snapshot,selection,action,review_rows,expires_at,wire_version)VALUES(w,actor,snap,selection_value,action_value,rows_value,clock_timestamp()+interval '15 minutes',2)RETURNING * INTO p;
 FOR result_row IN SELECT value FROM jsonb_array_elements(result_rows)LOOP
 INSERT INTO banking.collection_tag_preview_result(workspace_id,preview_id,source_account_id,observation_id,observation_revision,tag_ids,tags)VALUES(w,p.preview_id,(result_row->>'source_id')::uuid,(result_row->>'observation_id')::uuid,(result_row->>'observation_revision')::bigint,ARRAY(SELECT value::uuid FROM jsonb_array_elements_text(result_row->'tag_ids')),result_row->'tags');END LOOP;
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id)VALUES(w,actor,'preview',p.preview_id);
 res:=banking.collection_tag_render_preview(p);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;
-- Current disclosure uses one fact inventory for the entire immutable reviewed selection.
CREATE FUNCTION banking.collection_tag_reauthorize(actor uuid,w uuid,p banking.collection_label_preview)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF p.workspace_id IS DISTINCT FROM w OR p.principal_id IS DISTINCT FROM actor THEN PERFORM banking.collection_error('P2801');END IF;
 IF jsonb_array_length(p.review_rows)>0 AND EXISTS(
  WITH facts AS MATERIALIZED(SELECT source_account_id,observation_id FROM banking.collection_facts(w))
  SELECT FROM jsonb_array_elements(p.review_rows)z LEFT JOIN banking.collection_source s ON s.workspace_id=w AND s.source_account_id=(z->>'source_id')::uuid
  WHERE NOT EXISTS(SELECT FROM facts f WHERE f.source_account_id=(z->>'source_id')::uuid AND f.observation_id=(z->>'observation_id')::uuid)
  OR(z->'assignment'->>'state'='assigned'AND NOT banking.collection_label_book_read(actor,s.legacy_entity_id,s.legacy_book_id)))THEN PERFORM banking.collection_error('P2801');END IF;
END $$;
CREATE FUNCTION banking.prepare_collection_label_command_v2(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;id uuid;k text;p banking.collection_label_preview;c banking.collection_label_command;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);id:=ledger.uuid_value(v->'preview_id');k:=ledger.text_value(v->'prepare_key');IF k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=id AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;PERFORM banking.collection_tag_reauthorize(actor,w,p);
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND prepare_key=k;
 IF FOUND THEN IF c.preview_id<>id THEN PERFORM banking.collection_error('P2802');END IF;RETURN banking.collection_tag_render_command(c);END IF;
 IF EXISTS(SELECT FROM banking.collection_label_command WHERE workspace_id=w AND acknowledged_at IS NULL)THEN PERFORM banking.collection_error('P2802');END IF;
 IF p.wire_version=2 THEN PERFORM banking.collection_tag_pins(p.snapshot,banking.collection_tag_snapshot(w));ELSE PERFORM banking.collection_label_pins(p.snapshot,banking.collection_label_snapshot(w));END IF;IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;
 INSERT INTO banking.collection_label_command(workspace_id,principal_id,preview_id,prepare_key)VALUES(w,actor,id,k)RETURNING * INTO c;
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,'prepare',id,c.command_id);
 res:=banking.collection_tag_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.pending_collection_label_command_v2(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_label_command;p banking.collection_label_preview;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND acknowledged_at IS NULL;
 -- Read-only owners may establish that no work is pending; saved intent still requires classify.
 IF NOT FOUND THEN RETURN jsonb_build_object('profile','private-collection-command-v2','workspace_id',w,'command',NULL);END IF;
 IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND 'classify'=ANY(capabilities))THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_tag_reauthorize(actor,w,p);
 res:=banking.collection_tag_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;
CREATE FUNCTION banking.transition_collection_label_command_v2(actor uuid,w uuid,id uuid,operation text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;c banking.collection_label_command;p banking.collection_label_preview;ls banking.collection_label_state;action_value jsonb;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY[]::text[]);
 IF operation IS NULL OR operation<>ALL(ARRAY['execute','cancel','acknowledge'])THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND command_id=id AND principal_id=actor FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_tag_reauthorize(actor,w,p);
 IF operation='acknowledge'THEN
  IF c.state='prepared'THEN PERFORM banking.collection_error('P2802');END IF;
  IF c.acknowledged_at IS NULL THEN UPDATE banking.collection_label_command SET acknowledged_at=clock_timestamp()WHERE workspace_id=w AND command_id=id RETURNING * INTO c;INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,'acknowledge',p.preview_id,id);END IF;
 ELSIF c.state='prepared'THEN
  IF operation='execute'THEN
   IF p.wire_version=2 THEN PERFORM banking.collection_tag_pins(p.snapshot,banking.collection_tag_snapshot(w));ELSE PERFORM banking.collection_label_pins(p.snapshot,banking.collection_label_snapshot(w));END IF;IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;
   action_value:=banking.collection_tag_action(w,p.action);SELECT * INTO ls FROM banking.collection_label_state WHERE workspace_id=w;
   -- Validate the complete exact selection before any durable effect, with one current fact scan.
   IF jsonb_array_length(p.selection)>0 AND EXISTS(
    WITH facts AS MATERIALIZED(SELECT source_account_id,observation_id,observation_revision FROM banking.collection_facts(w))
    SELECT FROM jsonb_array_elements(p.selection)z WHERE NOT EXISTS(SELECT FROM facts f WHERE f.source_account_id=(z->>'source_id')::uuid AND f.observation_id=(z->>'observation_id')::uuid AND f.observation_revision=(z->>'observation_revision')::bigint))THEN PERFORM banking.collection_error('P2803');END IF;
   IF action_value->>'kind'='tag_catalog'THEN
    INSERT INTO banking.collection_tag_edition(workspace_id,tag_revision,tags,principal_id)VALUES(w,ls.tag_revision+1,action_value->'tags',actor);
    UPDATE banking.collection_label_state SET tag_revision=tag_revision+1 WHERE workspace_id=w;res:=jsonb_build_object('tag_revision',(ls.tag_revision+1)::text);
   ELSIF action_value->>'kind'='tags'THEN
    IF(SELECT count(*)FROM banking.collection_tag_preview_result WHERE workspace_id=w AND preview_id=p.preview_id)<>jsonb_array_length(p.selection)THEN PERFORM banking.collection_error('P2802');END IF;
    INSERT INTO banking.collection_tag_decision(workspace_id,source_account_id,observation_id,observation_revision,decision_revision,tag_revision,command_id,operation,tag_ids,principal_id)
     SELECT w,r.source_account_id,r.observation_id,r.observation_revision,ls.decision_revision+1,ls.tag_revision,id,action_value->>'operation',r.tag_ids,actor FROM banking.collection_tag_preview_result r WHERE r.workspace_id=w AND r.preview_id=p.preview_id;
    UPDATE banking.collection_label_state SET decision_revision=decision_revision+1 WHERE workspace_id=w;res:=jsonb_build_object('decision_revision',(ls.decision_revision+1)::text,'selected_count',jsonb_array_length(p.selection)::text);
   ELSIF action_value->>'kind'='taxonomy'THEN
    INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics,principal_id)VALUES(w,ls.taxonomy_revision+1,action_value->'topics',actor);
    UPDATE banking.collection_label_state SET taxonomy_revision=taxonomy_revision+1 WHERE workspace_id=w;
    res:=jsonb_build_object('taxonomy_revision',(ls.taxonomy_revision+1)::text);
   ELSE
    INSERT INTO banking.collection_label_decision(workspace_id,source_account_id,observation_id,observation_revision,decision_revision,taxonomy_revision,command_id,kind,topic_id,principal_id)
    SELECT w,(x->>'source_id')::uuid,(x->>'observation_id')::uuid,(x->>'observation_revision')::bigint,ls.decision_revision+1,ls.taxonomy_revision,id,action_value->>'kind',(action_value->>'topic_id')::uuid,actor FROM jsonb_array_elements(p.selection)x;
    UPDATE banking.collection_label_state SET decision_revision=decision_revision+1 WHERE workspace_id=w;
    res:=jsonb_build_object('decision_revision',(ls.decision_revision+1)::text,'selected_count',jsonb_array_length(p.selection)::text);
   END IF;
   UPDATE banking.collection_label_command SET state='committed',result=res,terminal_at=clock_timestamp()WHERE workspace_id=w AND command_id=id RETURNING * INTO c;
  ELSE UPDATE banking.collection_label_command SET state='cancelled',result=jsonb_build_object('cancelled',true),terminal_at=clock_timestamp()WHERE workspace_id=w AND command_id=id RETURNING * INTO c;END IF;
  INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,operation,p.preview_id,id);
 END IF;
 res:=banking.collection_tag_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
-- Old v1 callers retain closed responses and never receive unsupported v2 intent.
ALTER FUNCTION banking.prepare_collection_label_command(uuid,uuid,text)RENAME TO collection_tag_prepare_v1_base;
ALTER FUNCTION banking.pending_collection_label_command(uuid,uuid)RENAME TO collection_tag_pending_v1_base;
ALTER FUNCTION banking.transition_collection_label_command(uuid,uuid,uuid,text,text)RENAME TO collection_tag_transition_v1_base;
CREATE FUNCTION banking.prepare_collection_label_command(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;p banking.collection_label_preview;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=ledger.uuid_value(v->'preview_id')AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;PERFORM banking.collection_label_reauthorize(actor,w,p);IF p.wire_version=2 THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_tag_prepare_v1_base(actor,w,raw);
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.pending_collection_label_command(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_label_command;p banking.collection_label_preview;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND acknowledged_at IS NULL;
 IF FOUND THEN
  IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND 'classify'=ANY(capabilities))THEN PERFORM banking.collection_error('P2801');END IF;
  SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_label_reauthorize(actor,w,p);IF p.wire_version=2 THEN PERFORM banking.collection_error('P2804');END IF;
 END IF;RETURN banking.collection_tag_pending_v1_base(actor,w);
END $$;
CREATE FUNCTION banking.transition_collection_label_command(actor uuid,w uuid,id uuid,operation text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_label_command;p banking.collection_label_preview;v jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY[]::text[]);IF operation IS NULL OR operation<>ALL(ARRAY['execute','cancel','acknowledge'])THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND command_id=id AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_label_reauthorize(actor,w,p);IF p.wire_version=2 THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_tag_transition_v1_base(actor,w,id,operation,raw);
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
-- A current-row fence exists before any claim, including for a principal without a collection.
CREATE TABLE ledger.collection_recovery_state(principal_id uuid PRIMARY KEY REFERENCES ledger.principal,revision bigint NOT NULL DEFAULT 0 CHECK(revision>=0));
INSERT INTO ledger.collection_recovery_state(principal_id)SELECT principal_id FROM ledger.principal;
CREATE FUNCTION banking.collection_recovery_initialize()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN INSERT INTO ledger.collection_recovery_state(principal_id)VALUES(NEW.principal_id);RETURN NEW;END $$;
CREATE TRIGGER collection_recovery_initialize AFTER INSERT ON ledger.principal FOR EACH ROW EXECUTE FUNCTION banking.collection_recovery_initialize();
ALTER TABLE ledger.interactive_command_reservation ADD COLUMN recovery_id uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE ledger.interactive_command_reservation ADD CONSTRAINT interactive_command_recovery_id_key UNIQUE(recovery_id);
CREATE TABLE ledger.collection_recovery_receipt(principal_id uuid NOT NULL REFERENCES ledger.principal,recovery_id uuid NOT NULL,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),PRIMARY KEY(principal_id,recovery_id));
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON ledger.collection_recovery_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE FUNCTION banking.collection_recovery_guard()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF TG_OP='DELETE'OR NEW.principal_id<>OLD.principal_id OR NEW.revision<>OLD.revision+1 THEN PERFORM banking.collection_error('P2802');END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_recovery_state BEFORE UPDATE OR DELETE ON ledger.collection_recovery_state FOR EACH ROW EXECUTE FUNCTION banking.collection_recovery_guard();
-- Only a real explicit private acknowledgement can write closure proof; absence is never proof.
CREATE FUNCTION banking.collection_recovery_fence()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE subject uuid;observed uuid;BEGIN
 subject:=CASE WHEN TG_OP='DELETE'THEN OLD.principal_id ELSE NEW.principal_id END;
 IF TG_OP='DELETE'AND OLD.legacy_command_id IS NULL THEN
  IF NOT EXISTS(SELECT FROM banking.collection_label_command WHERE workspace_id=OLD.workspace_id AND command_id=OLD.label_command_id AND principal_id=OLD.principal_id AND acknowledged_at IS NOT NULL)THEN PERFORM banking.collection_error('P2802');END IF;
  INSERT INTO ledger.collection_recovery_receipt(principal_id,recovery_id)VALUES(OLD.principal_id,OLD.recovery_id)ON CONFLICT DO NOTHING;
 END IF;
 UPDATE ledger.collection_recovery_state SET revision=revision+1 WHERE principal_id=subject RETURNING principal_id INTO observed;
 IF observed IS NULL THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_recovery';END IF;RETURN NULL;
END $$;
CREATE TRIGGER collection_recovery_fence AFTER INSERT OR DELETE ON ledger.interactive_command_reservation FOR EACH ROW EXECUTE FUNCTION banking.collection_recovery_fence();
CREATE FUNCTION banking.collection_recovery_principal(actor uuid)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF actor IS NULL OR NOT integration.provider_api_login(session_user)THEN PERFORM banking.collection_error('P2801');END IF;IF current_setting('transaction_isolation')<>'serializable'THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM set_config('lock_timeout','1s',true);PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
END $$;
CREATE FUNCTION banking.collection_recovery_lock(actor uuid)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 PERFORM 1 FROM ledger.collection_recovery_state WHERE principal_id=actor FOR UPDATE;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_recovery';END IF;
END $$;
CREATE FUNCTION banking.collection_recovery_available(actor uuid,c banking.collection_label_command)RETURNS boolean LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE p banking.collection_label_preview;BEGIN
 PERFORM banking.collection_label_authorize(actor,c.workspace_id,false);
 IF c.principal_id IS DISTINCT FROM actor OR NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=c.workspace_id AND 'classify'=ANY(capabilities))THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=c.workspace_id AND preview_id=c.preview_id;PERFORM banking.collection_tag_reauthorize(actor,c.workspace_id,p);RETURN true;
EXCEPTION WHEN SQLSTATE 'P2801'THEN RETURN false;END $$;
CREATE FUNCTION banking.collection_recovery_status(actor uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE r ledger.interactive_command_reservation;c banking.collection_label_command;available boolean;BEGIN
 PERFORM banking.collection_recovery_principal(actor);SELECT * INTO r FROM ledger.interactive_command_reservation WHERE principal_id=actor;
 IF NOT FOUND OR r.legacy_command_id IS NOT NULL THEN PERFORM banking.collection_recovery_lock(actor);RETURN jsonb_build_object('profile','private-collection-recovery-v1','state','none','recovery_id',NULL);END IF;
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=r.workspace_id AND command_id=r.label_command_id AND principal_id=actor;IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_recovery';END IF;
 available:=banking.collection_recovery_available(actor,c);PERFORM banking.collection_recovery_lock(actor);RETURN jsonb_build_object('profile','private-collection-recovery-v1','state',CASE WHEN available THEN 'available'ELSE 'inaccessible'END,'recovery_id',r.recovery_id);
END $$;
CREATE FUNCTION banking.clear_collection_recovery(actor uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;id uuid;r ledger.interactive_command_reservation;c banking.collection_label_command;available boolean;BEGIN
 PERFORM banking.collection_recovery_principal(actor);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY['recovery_id']);id:=ledger.uuid_value(v->'recovery_id');
 IF EXISTS(SELECT FROM ledger.collection_recovery_receipt WHERE principal_id=actor AND recovery_id=id)THEN PERFORM banking.collection_recovery_lock(actor);RETURN jsonb_build_object('profile','private-collection-recovery-clear-v1','recovery_id',id,'cleared',true);END IF;
 SELECT * INTO r FROM ledger.interactive_command_reservation WHERE principal_id=actor AND recovery_id=id AND legacy_command_id IS NULL;
 IF NOT FOUND THEN PERFORM banking.collection_recovery_lock(actor);PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=r.workspace_id AND command_id=r.label_command_id AND principal_id=actor;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_recovery';END IF;
 available:=banking.collection_recovery_available(actor,c);PERFORM banking.collection_recovery_lock(actor);IF available THEN PERFORM banking.collection_error('P2802');END IF;
 PERFORM 1 FROM banking.collection_label_command WHERE workspace_id=c.workspace_id AND command_id=c.command_id FOR UPDATE;
 IF c.state='prepared'THEN
  UPDATE banking.collection_label_command SET state='cancelled',result=jsonb_build_object('cancelled',true),terminal_at=clock_timestamp()WHERE workspace_id=c.workspace_id AND command_id=c.command_id RETURNING * INTO c;
  INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(c.workspace_id,actor,'cancel',c.preview_id,c.command_id);
 END IF;
 UPDATE banking.collection_label_command SET acknowledged_at=clock_timestamp()WHERE workspace_id=c.workspace_id AND command_id=c.command_id RETURNING * INTO c;
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(c.workspace_id,actor,'acknowledge',c.preview_id,c.command_id);
 IF NOT EXISTS(SELECT FROM ledger.collection_recovery_receipt WHERE principal_id=actor AND recovery_id=id)THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_collection_recovery';END IF;
 RETURN jsonb_build_object('profile','private-collection-recovery-clear-v1','recovery_id',id,'cleared',true);
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
DO $$DECLARE f regprocedure;BEGIN
 FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='banking'AND(p.proname LIKE 'collection_tag_%'OR p.proname LIKE 'collection_recovery_%'OR p.proname IN('clear_collection_recovery','get_collection_taxonomy_v2','query_collection_review_v2','preview_collection_labels_v2','prepare_collection_label_command_v2','pending_collection_label_command_v2','transition_collection_label_command_v2'))LOOP
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator',f);END LOOP;
END $$;
REVOKE ALL ON banking.collection_tag_edition,banking.collection_tag_decision,banking.collection_tag_preview_result,ledger.collection_recovery_state,ledger.collection_recovery_receipt FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
REVOKE ALL ON FUNCTION banking.prepare_collection_label_command(uuid,uuid,text),banking.pending_collection_label_command(uuid,uuid),banking.transition_collection_label_command(uuid,uuid,uuid,text,text),banking.collection_label_snapshot(uuid)FROM PUBLIC;
GRANT EXECUTE ON FUNCTION banking.prepare_collection_label_command(uuid,uuid,text),banking.pending_collection_label_command(uuid,uuid),banking.transition_collection_label_command(uuid,uuid,uuid,text,text),banking.get_collection_taxonomy_v2(uuid,uuid),banking.query_collection_review_v2(uuid,uuid,text),banking.preview_collection_labels_v2(uuid,uuid,text),banking.prepare_collection_label_command_v2(uuid,uuid,text),banking.pending_collection_label_command_v2(uuid,uuid),banking.transition_collection_label_command_v2(uuid,uuid,uuid,text,text),banking.collection_recovery_status(uuid),banking.clear_collection_recovery(uuid,text)TO bookkeeping_api;
COMMIT;
