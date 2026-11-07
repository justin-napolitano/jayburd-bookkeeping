-- #304: private taxonomy, analytic decisions and explicit command recovery; no journals.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE banking.collection_workspace IN SHARE ROW EXCLUSIVE MODE;
CREATE TABLE banking.collection_taxonomy_edition(
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,taxonomy_revision bigint NOT NULL CHECK(taxonomy_revision>0),
 topics jsonb NOT NULL CHECK(jsonb_typeof(topics)='array' AND jsonb_array_length(topics)<=1000 AND octet_length(topics::text)<=524288),
 principal_id uuid REFERENCES ledger.principal,created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)),PRIMARY KEY(workspace_id,taxonomy_revision)
);
CREATE TABLE banking.collection_label_state(
 workspace_id uuid PRIMARY KEY REFERENCES banking.collection_workspace,taxonomy_revision bigint NOT NULL DEFAULT 1 CHECK(taxonomy_revision>0),
 decision_revision bigint NOT NULL DEFAULT 0 CHECK(decision_revision>=0),projection_revision bigint NOT NULL DEFAULT 0 CHECK(projection_revision>=0),command_revision bigint NOT NULL DEFAULT 0 CHECK(command_revision>=0),
 FOREIGN KEY(workspace_id,taxonomy_revision)REFERENCES banking.collection_taxonomy_edition DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE banking.collection_label_preview(
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,preview_id uuid NOT NULL DEFAULT gen_random_uuid(),principal_id uuid NOT NULL REFERENCES ledger.principal,
 snapshot jsonb NOT NULL,selection jsonb NOT NULL CHECK(jsonb_typeof(selection)='array' AND jsonb_array_length(selection)<=100),action jsonb NOT NULL,
 review_rows jsonb NOT NULL CHECK(jsonb_typeof(review_rows)='array' AND jsonb_array_length(review_rows)<=100),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)),expires_at timestamptz NOT NULL CHECK(isfinite(expires_at) AND expires_at>created_at),
 PRIMARY KEY(workspace_id,preview_id),UNIQUE(workspace_id,preview_id,principal_id),CHECK(octet_length((snapshot::text||selection::text||action::text||review_rows::text))<=1040000)
);
CREATE TABLE banking.collection_label_command(
 workspace_id uuid NOT NULL,command_id uuid NOT NULL DEFAULT gen_random_uuid(),principal_id uuid NOT NULL REFERENCES ledger.principal,preview_id uuid NOT NULL,
 prepare_key text NOT NULL CHECK(prepare_key COLLATE "C" ~ '^[A-Za-z0-9._:-]{1,128}$'),state text NOT NULL DEFAULT 'prepared' CHECK(state IN('prepared','committed','cancelled')),
 result jsonb,created_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(created_at)),terminal_at timestamptz CHECK(isfinite(terminal_at)),acknowledged_at timestamptz CHECK(isfinite(acknowledged_at)),
 PRIMARY KEY(workspace_id,command_id),UNIQUE(workspace_id,principal_id,prepare_key),FOREIGN KEY(workspace_id,preview_id,principal_id)REFERENCES banking.collection_label_preview(workspace_id,preview_id,principal_id),
 CHECK((state='prepared' AND terminal_at IS NULL AND result IS NULL AND acknowledged_at IS NULL)OR(state IN('committed','cancelled')AND terminal_at IS NOT NULL AND result IS NOT NULL)),CHECK(result IS NULL OR octet_length(result::text)<=4096)
);
CREATE UNIQUE INDEX collection_label_one_pending ON banking.collection_label_command(workspace_id)WHERE acknowledged_at IS NULL;
CREATE TABLE banking.collection_label_decision(
 workspace_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,observation_revision bigint NOT NULL CHECK(observation_revision>0),
 decision_revision bigint NOT NULL CHECK(decision_revision>0),taxonomy_revision bigint NOT NULL,command_id uuid NOT NULL,kind text NOT NULL CHECK(kind IN('topic','uncategorized')),topic_id uuid,
 principal_id uuid NOT NULL REFERENCES ledger.principal,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK(isfinite(recorded_at)),
 PRIMARY KEY(workspace_id,decision_revision,source_account_id,observation_id),FOREIGN KEY(workspace_id,source_account_id)REFERENCES banking.collection_source,
 FOREIGN KEY(workspace_id,taxonomy_revision)REFERENCES banking.collection_taxonomy_edition,FOREIGN KEY(workspace_id,command_id)REFERENCES banking.collection_label_command,
 CHECK((kind='topic' AND topic_id IS NOT NULL)OR(kind='uncategorized'AND topic_id IS NULL))
);
CREATE INDEX collection_label_latest ON banking.collection_label_decision(workspace_id,source_account_id,observation_id,decision_revision DESC);
CREATE TABLE audit.collection_label_action(
 action_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,principal_id uuid NOT NULL REFERENCES ledger.principal,
 operation text NOT NULL CHECK(operation IN('preview','prepare','execute','cancel','acknowledge')),preview_id uuid NOT NULL,command_id uuid,
 database_actor name NOT NULL DEFAULT session_user,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),
 FOREIGN KEY(workspace_id,preview_id)REFERENCES banking.collection_label_preview,FOREIGN KEY(workspace_id,command_id)REFERENCES banking.collection_label_command
);
-- Precreated state is a real current-row fence even for an empty new workspace.
CREATE FUNCTION banking.collection_label_initialize()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics)VALUES(NEW.workspace_id,1,'[]');
 INSERT INTO banking.collection_label_state(workspace_id)VALUES(NEW.workspace_id);RETURN NEW;
END $$;
INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics)SELECT workspace_id,1,'[]'FROM banking.collection_workspace;
INSERT INTO banking.collection_label_state(workspace_id)SELECT workspace_id FROM banking.collection_workspace;
CREATE TRIGGER collection_label_initialize AFTER INSERT ON banking.collection_workspace FOR EACH ROW EXECUTE FUNCTION banking.collection_label_initialize();
-- Monotonic state and command lifecycle; no deletion or mutable reviewed intent.
CREATE FUNCTION banking.collection_label_guard()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF TG_OP='DELETE'THEN PERFORM banking.collection_error('P2802');END IF;
 IF TG_TABLE_NAME='collection_label_state'THEN
  IF NEW.workspace_id<>OLD.workspace_id OR NOT(
   (NEW.command_revision=OLD.command_revision AND((NEW.taxonomy_revision=OLD.taxonomy_revision+1 AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision)OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision+1 AND NEW.projection_revision=OLD.projection_revision)OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision+1)))
   OR(NEW.taxonomy_revision=OLD.taxonomy_revision AND NEW.decision_revision=OLD.decision_revision AND NEW.projection_revision=OLD.projection_revision AND NEW.command_revision=OLD.command_revision+1))THEN PERFORM banking.collection_error('P2802');END IF;
 ELSE
  IF(to_jsonb(NEW)-ARRAY['state','result','terminal_at','acknowledged_at'])IS DISTINCT FROM(to_jsonb(OLD)-ARRAY['state','result','terminal_at','acknowledged_at'])THEN PERFORM banking.collection_error('P2802');END IF;
  IF OLD.state='prepared'THEN
   IF NEW.state NOT IN('committed','cancelled')OR NEW.terminal_at IS NULL OR NEW.result IS NULL OR NEW.acknowledged_at IS NOT NULL THEN PERFORM banking.collection_error('P2802');END IF;
  ELSE
   IF ROW(NEW.state,NEW.result,NEW.terminal_at)IS DISTINCT FROM ROW(OLD.state,OLD.result,OLD.terminal_at)OR OLD.acknowledged_at IS NOT NULL OR NEW.acknowledged_at IS NULL THEN PERFORM banking.collection_error('P2802');END IF;
  END IF;
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_collection_label BEFORE UPDATE OR DELETE ON banking.collection_label_state FOR EACH ROW EXECUTE FUNCTION banking.collection_label_guard();
CREATE TRIGGER preserve_collection_label BEFORE UPDATE OR DELETE ON banking.collection_label_command FOR EACH ROW EXECUTE FUNCTION banking.collection_label_guard();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_taxonomy_edition FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_label_preview FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_label_decision FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON audit.collection_label_action FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Source inventory and legacy authority/projection insertions cannot hide in a primed RR snapshot.
CREATE FUNCTION banking.collection_label_fence()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE r jsonb;target uuid;eid uuid;bid uuid;pid uuid;BEGIN
 r:=CASE WHEN TG_OP='DELETE'THEN to_jsonb(OLD)ELSE to_jsonb(NEW)END;
 IF TG_TABLE_NAME='collection_label_command'THEN
  UPDATE banking.collection_label_state SET command_revision=command_revision+1 WHERE workspace_id=(r->>'workspace_id')::uuid;
 ELSIF TG_TABLE_NAME IN('collection_source','collection_item')THEN
  target:=(r->>'workspace_id')::uuid;
  UPDATE banking.collection_label_state SET projection_revision=projection_revision+1 WHERE workspace_id=target;
 ELSE
  eid:=(r->>'entity_id')::uuid;bid:=(r->>'book_id')::uuid;pid:=(r->>'principal_id')::uuid;
  FOR target IN SELECT DISTINCT s.workspace_id FROM banking.collection_source s JOIN banking.collection_workspace c USING(workspace_id)
   WHERE s.origin='legacy' AND s.legacy_entity_id=eid AND(bid IS NULL OR s.legacy_book_id=bid)AND(TG_TABLE_NAME<>'membership'OR c.owner_principal_id=pid)ORDER BY s.workspace_id LOOP
   UPDATE banking.collection_label_state SET projection_revision=projection_revision+1 WHERE workspace_id=target;
  END LOOP;
 END IF;RETURN NULL;
END $$;
CREATE TRIGGER collection_label_fence AFTER INSERT OR UPDATE ON banking.collection_label_command FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER INSERT OR UPDATE ON banking.collection_source FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER INSERT OR UPDATE ON integration.collection_item FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER INSERT OR UPDATE OR DELETE ON ledger.membership FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER UPDATE ON ledger.entity FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER INSERT OR UPDATE OR DELETE ON ledger.book FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER INSERT OR UPDATE ON banking.review_item FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER INSERT OR UPDATE ON banking.review_assignment FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
CREATE TRIGGER collection_label_fence AFTER UPDATE ON banking.source_account FOR EACH ROW EXECUTE FUNCTION banking.collection_label_fence();
-- Every legacy metadata authority lock precedes workspace/source locks, matching adoption/posting.
CREATE FUNCTION banking.collection_label_authorize(actor uuid,w uuid,writing boolean)RETURNS banking.collection_workspace LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE r record;c banking.collection_workspace;BEGIN
 IF NOT integration.provider_api_login(session_user)OR actor IS NULL OR w IS NULL THEN PERFORM banking.collection_error('P2801');END IF;
 IF(writing AND current_setting('transaction_isolation')<>'serializable')OR(NOT writing AND current_setting('transaction_isolation')NOT IN('repeatable read','serializable'))THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM set_config('lock_timeout','1s',true);
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor FOR SHARE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND owner_principal_id=actor)THEN PERFORM banking.collection_error('P2801');END IF;
 IF(SELECT count(*)FROM banking.collection_source WHERE workspace_id=w)>100 THEN PERFORM banking.collection_error('P2805');END IF;
 FOR r IN SELECT DISTINCT legacy_entity_id e,legacy_book_id b FROM banking.collection_source WHERE workspace_id=w AND origin='legacy'ORDER BY e,b LOOP
  PERFORM 1 FROM ledger.entity WHERE entity_id=r.e FOR SHARE;
  PERFORM 1 FROM ledger.membership WHERE entity_id=r.e AND principal_id=actor AND permission='read' FOR SHARE;
  PERFORM 1 FROM ledger.book WHERE entity_id=r.e AND book_id=r.b FOR SHARE;
 END LOOP;
 c:=banking.collection_authorize(actor,w,CASE WHEN writing THEN 'classify'ELSE 'read'END,writing);
 PERFORM banking.collection_read_locks(w);
 -- The final fence follows all trigger source rows, including denied metadata scopes.
 IF writing THEN PERFORM 1 FROM banking.collection_label_state WHERE workspace_id=w FOR UPDATE;
 ELSE PERFORM 1 FROM banking.collection_label_state WHERE workspace_id=w FOR SHARE;END IF;
 IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;RETURN c;
END $$;
CREATE FUNCTION banking.collection_label_book_read(actor uuid,e uuid,b uuid)RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM ledger.entity x JOIN ledger.book y USING(entity_id)JOIN ledger.membership m USING(entity_id)WHERE x.entity_id=e AND y.book_id=b AND m.principal_id=actor AND m.permission='read'AND x.disabled_at IS NULL AND y.disabled_at IS NULL AND m.revoked_at IS NULL)
$$;
CREATE FUNCTION banking.collection_label_snapshot(w uuid)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT jsonb_build_object('authority_revision',c.revision::text,'source_snapshot',encode(sha256(convert_to(jsonb_build_array(banking.collection_snapshot(w)->>'source_snapshot',s.projection_revision::text)::text,'UTF8')),'hex'),'taxonomy_revision',s.taxonomy_revision::text,'ruleset_revision','0','map_revision','0','decision_revision',s.decision_revision::text)FROM banking.collection_workspace c JOIN banking.collection_label_state s USING(workspace_id)WHERE c.workspace_id=w
$$;
CREATE FUNCTION banking.collection_label_pins(v jsonb,current_value jsonb)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$DECLARE x text;BEGIN
 PERFORM banking.collection_keys(v,ARRAY['authority_revision','source_snapshot','taxonomy_revision','ruleset_revision','map_revision','decision_revision']);
 FOREACH x IN ARRAY ARRAY['authority_revision','taxonomy_revision','ruleset_revision','map_revision','decision_revision']LOOP PERFORM banking.import_revision(v->x);END LOOP;
 IF ledger.text_value(v->'source_snapshot')COLLATE "C" !~ '^[0-9a-f]{64}$'THEN PERFORM banking.collection_error('P2800');END IF;
 IF v<>current_value THEN PERFORM banking.collection_error('P2803');END IF;
END $$;
-- Whole current safe row projection; owner labels are independent of accounting state.
CREATE FUNCTION banking.collection_label_rows(actor uuid,w uuid)RETURNS TABLE(source_id uuid,observation_id uuid,observation_revision bigint,booked_date date,body jsonb)
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT f.source_account_id,f.observation_id,f.observation_revision,f.booked_date,
 jsonb_build_object('workspace_id',w,'source_id',f.source_account_id,'observation_id',f.observation_id,'observation_revision',f.observation_revision::text,'account_name',s.display_name,'booked_date',to_char(f.booked_date,'YYYY-MM-DD'),'description',f.description,'merchant',f.merchant_name,'source_state',CASE WHEN hold.reason IS NOT NULL AND f.source_state<>'removed'THEN 'held'ELSE f.source_state END,
 'money',CASE WHEN hold.reason IS NOT NULL THEN jsonb_build_object('state','held','reason',hold.reason,'amount',NULL,'direction',NULL,'currency',NULL,'currency_version_id',NULL,'numeric_code',NULL,'minor_unit',NULL,'valid_from',NULL,'valid_to',NULL)ELSE jsonb_build_object('state','available','amount',trim_scale(f.amount)::text,'direction',f.direction,'currency',cur.alphabetic_code,'currency_version_id',cur.currency_version_id,'numeric_code',cur.numeric_code,'minor_unit',cur.minor_unit,'valid_from',to_char(cur.valid_from,'YYYY-MM-DD'),'valid_to',to_char(cur.valid_to,'YYYY-MM-DD'))END,
 'assignment',CASE WHEN s.origin='native'THEN jsonb_build_object('state','unassigned')WHEN banking.collection_label_book_read(actor,s.legacy_entity_id,s.legacy_book_id)THEN jsonb_build_object('state','assigned','books',jsonb_build_array(jsonb_build_object('book_id',s.legacy_book_id,'name',left(b.name,200))))ELSE jsonb_build_object('state','unavailable')END,
 'journal_state',CASE WHEN s.origin='native'THEN 'unposted'WHEN NOT banking.collection_label_book_read(actor,s.legacy_entity_id,s.legacy_book_id)THEN 'unavailable'WHEN ri.state='posted'AND posted.journal_id IS NOT NULL THEN CASE WHEN ev.linked_revision IS DISTINCT FROM f.observation_revision THEN 'correction_required'ELSE 'posted'END ELSE 'unposted'END,
 'owner_topic',CASE WHEN d.decision_revision IS NULL THEN jsonb_build_object('state','unclassified')WHEN d.observation_revision<>f.observation_revision THEN jsonb_build_object('state','stale')WHEN d.kind='uncategorized'THEN jsonb_build_object('state','uncategorized')ELSE jsonb_build_object('state','topic','topic_id',d.topic_id,'name',topic.value->>'name')END,
 'provider_suggestion',jsonb_build_object('state','unavailable'))
 FROM banking.collection_facts(w)f JOIN banking.collection_source s ON s.workspace_id=w AND s.source_account_id=f.source_account_id
 JOIN banking.collection_label_state ls ON ls.workspace_id=w JOIN banking.collection_taxonomy_edition te ON te.workspace_id=w AND te.taxonomy_revision=ls.taxonomy_revision
 LEFT JOIN reference.currency cur ON cur.currency_version_id=f.currency_version_id
 LEFT JOIN banking.observation_revision legacy_fact ON s.origin='legacy'AND legacy_fact.entity_id=s.legacy_entity_id AND legacy_fact.book_id=s.legacy_book_id AND legacy_fact.observation_id=f.observation_id AND legacy_fact.revision=f.observation_revision
 CROSS JOIN LATERAL(SELECT CASE WHEN f.hold_reason IS NOT NULL THEN f.hold_reason WHEN s.origin='legacy'AND(cur.currency_version_id IS NULL OR cur.minor_unit IS NULL OR f.booked_date<cur.valid_from OR cur.valid_to IS NOT NULL AND f.booked_date>=cur.valid_to)THEN 'unsupported_currency'WHEN s.origin='legacy'AND scale(trim_scale(legacy_fact.amount_lexeme::numeric))>cur.minor_unit THEN 'invalid_precision'END reason)hold
 LEFT JOIN ledger.book b ON b.entity_id=s.legacy_entity_id AND b.book_id=s.legacy_book_id
 LEFT JOIN banking.review_assignment a ON a.entity_id=s.legacy_entity_id AND a.book_id=s.legacy_book_id AND a.observation_id=f.observation_id
 LEFT JOIN banking.review_item ri ON ri.entity_id=a.entity_id AND ri.book_id=a.book_id AND ri.review_id=a.review_id
 LEFT JOIN banking.review_posting rp ON rp.entity_id=ri.entity_id AND rp.book_id=ri.book_id AND rp.review_id=ri.review_id
 LEFT JOIN ledger.journal_entry posted ON posted.entity_id=rp.entity_id AND posted.book_id=rp.book_id AND posted.journal_id=rp.journal_id
 LEFT JOIN banking.review_evidence ev ON ev.entity_id=a.entity_id AND ev.book_id=a.book_id AND ev.decision_id=a.decision_id AND ev.observation_id=a.observation_id
 LEFT JOIN LATERAL(SELECT x.* FROM banking.collection_label_decision x WHERE x.workspace_id=w AND x.source_account_id=f.source_account_id AND x.observation_id=f.observation_id ORDER BY x.decision_revision DESC LIMIT 1)d ON true
 LEFT JOIN LATERAL(SELECT value FROM jsonb_array_elements(te.topics)WHERE value->>'topic_id'=d.topic_id::text)topic ON true
$$;
-- A catalog is bounded, current and private; it creates no seed or implicit capability.
CREATE FUNCTION banking.get_collection_taxonomy(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE c banking.collection_workspace;s banking.collection_label_state;res jsonb;BEGIN
 c:=banking.collection_label_authorize(actor,w,false);SELECT * INTO s FROM banking.collection_label_state WHERE workspace_id=w;
 SELECT jsonb_build_object('profile','private-collection-catalog-v1','workspace_id',w,'authority_revision',c.revision::text,'taxonomy_revision',s.taxonomy_revision::text,'ruleset_revision','0','map_revision','0','topics',e.topics,'sources',coalesce((SELECT jsonb_agg(jsonb_build_object('source_id',v.source_account_id,'name',v.display_name)ORDER BY v.source_account_id)FROM banking.collection_visible_sources(w)v),'[]'))INTO res FROM banking.collection_taxonomy_edition e WHERE e.workspace_id=w AND e.taxonomy_revision=s.taxonomy_revision;
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;
-- Date sorting/count/filtering applies to the full matching set, never only a fetched page.
CREATE FUNCTION banking.query_collection_review(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;sel jsonb;snap jsonb;sid uuid;first_day date;last_day date;search_value text;assignment_value text;sort_value text;topic_kind text;tid uuid;cur jsonb;txt text;page_offset bigint:=0;res jsonb;total bigint;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);v:=banking.import_json(raw,8192);
 PERFORM banking.collection_keys(v,ARRAY['profile','source_account_id','assignment','starts_on','through_date','search','topic','sort','snapshot','cursor','limit']);
 IF ledger.text_value(v->'profile')<>'private-collection-review-v1'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_typeof(v->'limit')IS DISTINCT FROM 'number'OR v->>'limit'!~ '^[1-9][0-9]{0,2}$'OR(v->>'limit')::integer>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'source_account_id'<>'null'::jsonb THEN sid:=ledger.uuid_value(v->'source_account_id');IF NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id=sid)THEN PERFORM banking.collection_error('P2801');END IF;END IF;
 assignment_value:=ledger.text_value(v->'assignment');sort_value:=ledger.text_value(v->'sort');search_value:=ledger.text_value(v->'search');
 IF assignment_value<>ALL(ARRAY['all','assigned','unassigned'])OR sort_value<>ALL(ARRAY['date_desc','date_asc'])OR length(search_value)>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'starts_on'<>'null'::jsonb THEN first_day:=ledger.date_value(ledger.text_value(v->'starts_on'));END IF;IF v->'through_date'<>'null'::jsonb THEN last_day:=ledger.date_value(ledger.text_value(v->'through_date'));END IF;IF first_day>last_day THEN PERFORM banking.collection_error('P2800');END IF;
 topic_kind:=ledger.text_value(v->'topic'->'kind');
 IF topic_kind='topic'THEN PERFORM banking.collection_keys(v->'topic',ARRAY['kind','topic_id']);tid:=ledger.uuid_value(v->'topic'->'topic_id');
  IF NOT EXISTS(SELECT FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'topic_id'=tid::text)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'topic',ARRAY['kind']);IF topic_kind<>ALL(ARRAY['all','unclassified','uncategorized'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 snap:=banking.collection_label_snapshot(w);IF v->'snapshot'<>'null'::jsonb THEN PERFORM banking.collection_label_pins(v->'snapshot',snap);END IF;
 sel:=v-ARRAY['profile','snapshot','cursor','limit'];
 IF v->'cursor'<>'null'::jsonb THEN
  txt:=ledger.text_value(v->'cursor');IF length(txt)NOT BETWEEN 2 AND 4096 OR txt COLLATE "C" !~ '^([0-9a-f]{2})+$'THEN PERFORM banking.collection_error('P2800');END IF;
  cur:=banking.import_json(convert_from(decode(txt,'hex'),'UTF8'),2048);PERFORM banking.collection_keys(cur,ARRAY['workspace_id','snapshot','selector','offset']);
  IF ledger.uuid_value(cur->'workspace_id')<>w OR cur->'selector'<>sel THEN PERFORM banking.collection_error('P2800');END IF;
  PERFORM banking.collection_label_pins(cur->'snapshot',snap);page_offset:=banking.import_revision(cur->'offset');IF page_offset=0 THEN PERFORM banking.collection_error('P2800');END IF;
 END IF;
 WITH matched AS MATERIALIZED(SELECT f.* FROM banking.collection_label_rows(actor,w)f WHERE(sid IS NULL OR f.source_id=sid)AND(first_day IS NULL OR f.booked_date>=first_day)AND(last_day IS NULL OR f.booked_date<=last_day)AND strpos(lower(f.body->>'description'COLLATE "C"),lower(search_value COLLATE "C"))>0
 AND(assignment_value='all'OR f.body->'assignment'->>'state'=assignment_value)
 AND(topic_kind='all'OR topic_kind='unclassified'AND f.body->'owner_topic'->>'state'IN('unclassified','stale')OR topic_kind='uncategorized'AND f.body->'owner_topic'->>'state'='uncategorized'OR topic_kind='topic'AND f.body->'owner_topic'->>'state'='topic'AND f.body->'owner_topic'->>'topic_id'=tid::text)),
 numbered AS MATERIALIZED(SELECT f.*,row_number()OVER(ORDER BY CASE WHEN sort_value='date_desc'THEN booked_date END DESC,CASE WHEN sort_value='date_asc'THEN booked_date END ASC,source_id,observation_id)rn FROM matched f),
 picked AS MATERIALIZED(SELECT * FROM numbered WHERE rn>page_offset ORDER BY rn LIMIT(v->>'limit')::integer)
 SELECT jsonb_build_object('profile','private-collection-review-v1','workspace_id',w,'selector',sel,'snapshot',snap,'matching_count',(SELECT count(*)::text FROM matched),'start_index',CASE WHEN EXISTS(SELECT FROM picked)THEN(page_offset+1)::text ELSE '0'END,'items',coalesce((SELECT jsonb_agg(body ORDER BY rn)FROM picked),'[]'),
 'next_cursor',CASE WHEN(SELECT max(rn)FROM picked)<(SELECT count(*)FROM matched)THEN encode(convert_to(jsonb_build_object('workspace_id',w,'snapshot',snap,'selector',sel,'offset',(SELECT max(rn)::text FROM picked))::text,'UTF8'),'hex')END,'partial',EXISTS(SELECT FROM banking.collection_visible_sources(w)coverage WHERE(sid IS NULL OR coverage.source_account_id=sid)AND(coverage.origin='legacy'OR coverage.readiness<>'HISTORICAL_UPDATE_COMPLETE'))),(SELECT count(*)FROM matched)INTO res,total;
 IF page_offset>0 AND page_offset>=total THEN PERFORM banking.collection_error('P2800');END IF;
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.collection_error('P2800');END $$;
-- Full editions retain every stable identity, including terminal retired topics.
CREATE FUNCTION banking.collection_label_action(w uuid,v jsonb)RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE kind_value text;z jsonb;id uuid;parent uuid;name_value text;status_value text;topics_value jsonb:='[]';oldtopics jsonb;n integer;BEGIN
 kind_value:=ledger.text_value(v->'kind');
 SELECT e.topics INTO oldtopics FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision)WHERE e.workspace_id=w;
 IF kind_value='uncategorized'THEN PERFORM banking.collection_keys(v,ARRAY['kind']);RETURN v;
 ELSIF kind_value='topic'THEN PERFORM banking.collection_keys(v,ARRAY['kind','topic_id']);id:=ledger.uuid_value(v->'topic_id');
  IF NOT EXISTS(SELECT FROM jsonb_array_elements(oldtopics)t WHERE t->>'topic_id'=id::text AND t->>'status'='active')THEN PERFORM banking.collection_error('P2803');END IF;
  RETURN jsonb_build_object('kind','topic','topic_id',id);
 ELSIF kind_value<>'taxonomy'THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_keys(v,ARRAY['kind','topics']);IF jsonb_typeof(v->'topics')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_array_length(v->'topics')>1000 THEN PERFORM banking.collection_error('P2805');END IF;
 FOR z IN SELECT value FROM jsonb_array_elements(v->'topics')LOOP
  PERFORM banking.collection_keys(z,ARRAY['topic_id','name','parent_id','status']);id:=ledger.uuid_value(z->'topic_id');name_value:=ledger.text_value(z->'name');status_value:=ledger.text_value(z->'status');parent:=NULL;
  IF z->'parent_id'<>'null'::jsonb THEN parent:=ledger.uuid_value(z->'parent_id');END IF;
  IF length(name_value)NOT BETWEEN 1 AND 80 OR btrim(name_value)=''OR name_value~'[[:cntrl:]]'OR status_value<>ALL(ARRAY['active','retired'])THEN PERFORM banking.collection_error('P2800');END IF;
  topics_value:=topics_value||jsonb_build_array(jsonb_build_object('topic_id',id,'name',name_value,'parent_id',parent,'status',status_value));
 END LOOP;
 IF(SELECT count(*)FROM jsonb_array_elements(topics_value)t WHERE t->>'status'='active')>500 THEN PERFORM banking.collection_error('P2805');END IF;
 IF EXISTS(SELECT t->>'topic_id'FROM jsonb_array_elements(topics_value)t GROUP BY t->>'topic_id'HAVING count(*)>1)OR EXISTS(SELECT t->>'parent_id',t->>'name'COLLATE "C" FROM jsonb_array_elements(topics_value)t GROUP BY t->>'parent_id',t->>'name'COLLATE "C" HAVING count(*)>1)THEN PERFORM banking.collection_error('P2800');END IF;
 IF EXISTS(SELECT FROM jsonb_array_elements(topics_value)t WHERE t->'parent_id'<>'null'::jsonb AND NOT EXISTS(SELECT FROM jsonb_array_elements(topics_value)p WHERE p->>'topic_id'=t->>'parent_id'AND(t->>'status'<>'active'OR p->>'status'='active')))THEN PERFORM banking.collection_error('P2800');END IF;
 WITH RECURSIVE walk AS(SELECT t->>'topic_id'id,1 depth FROM jsonb_array_elements(topics_value)t WHERE t->'parent_id'='null'::jsonb UNION ALL SELECT t->>'topic_id',p.depth+1 FROM walk p,jsonb_array_elements(topics_value)t WHERE t->>'parent_id'=p.id AND p.depth<9)SELECT count(*)INTO n FROM walk WHERE depth<=8;
 IF n<>jsonb_array_length(topics_value)THEN PERFORM banking.collection_error('P2800');END IF;
 IF EXISTS(SELECT FROM jsonb_array_elements(oldtopics)o WHERE NOT EXISTS(SELECT FROM jsonb_array_elements(topics_value)t WHERE t->>'topic_id'=o->>'topic_id'AND(o->>'status'<>'retired'OR t->>'status'='retired')))THEN PERFORM banking.collection_error('P2802');END IF;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'topic_id'),'[]')INTO topics_value FROM jsonb_array_elements(topics_value);
 RETURN jsonb_build_object('kind','taxonomy','topics',topics_value);
END $$;
CREATE FUNCTION banking.collection_label_render_preview(p banking.collection_label_preview)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT jsonb_build_object('profile','private-collection-preview-v1','workspace_id',p.workspace_id,'preview_id',p.preview_id,'expires_at',to_char(p.expires_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'snapshot',p.snapshot,'selection',p.selection,'action',p.action,'selected_count',jsonb_array_length(p.selection)::text,'review_rows',p.review_rows)
$$;
-- Exact source identities/revisions are canonicalized and stored before any durable command.
CREATE FUNCTION banking.preview_collection_labels(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;snap jsonb;action_value jsonb;z jsonb;sid uuid;oid uuid;rev bigint;selection_value jsonb:='[]';rows_value jsonb:='[]';fact jsonb;p banking.collection_label_preview;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,524288);PERFORM banking.collection_keys(v,ARRAY['snapshot','selection','action']);snap:=banking.collection_label_snapshot(w);PERFORM banking.collection_label_pins(v->'snapshot',snap);action_value:=banking.collection_label_action(w,v->'action');
 IF jsonb_typeof(v->'selection')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;
 IF(action_value->>'kind'='taxonomy'AND jsonb_array_length(v->'selection')<>0)OR(action_value->>'kind'<>'taxonomy'AND jsonb_array_length(v->'selection')NOT BETWEEN 1 AND 100)THEN PERFORM banking.collection_error('P2805');END IF;
 FOR z IN SELECT value FROM jsonb_array_elements(v->'selection')LOOP
  PERFORM banking.collection_keys(z,ARRAY['workspace_id','source_id','observation_id','observation_revision']);IF ledger.uuid_value(z->'workspace_id')<>w THEN PERFORM banking.collection_error('P2801');END IF;
  sid:=ledger.uuid_value(z->'source_id');oid:=ledger.uuid_value(z->'observation_id');rev:=banking.import_revision(z->'observation_revision');IF rev=0 THEN PERFORM banking.collection_error('P2800');END IF;
  IF EXISTS(SELECT FROM jsonb_array_elements(selection_value)x WHERE x->>'source_id'=sid::text AND x->>'observation_id'=oid::text)THEN PERFORM banking.collection_error('P2800');END IF;
  SELECT f.body INTO fact FROM banking.collection_label_rows(actor,w)f WHERE f.source_id=sid AND f.observation_id=oid;
  IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;IF fact->>'observation_revision'<>rev::text THEN PERFORM banking.collection_error('P2803');END IF;
  selection_value:=selection_value||jsonb_build_array(jsonb_build_object('workspace_id',w,'source_id',sid,'observation_id',oid,'observation_revision',rev::text));rows_value:=rows_value||jsonb_build_array(fact);
 END LOOP;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO selection_value FROM jsonb_array_elements(selection_value);
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO rows_value FROM jsonb_array_elements(rows_value);
 INSERT INTO banking.collection_label_preview(workspace_id,principal_id,snapshot,selection,action,review_rows,expires_at)VALUES(w,actor,snap,selection_value,action_value,rows_value,clock_timestamp()+interval '15 minutes')RETURNING * INTO p;
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id)VALUES(w,actor,'preview',p.preview_id);
 res:=banking.collection_label_render_preview(p);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
-- Recovery rechecks current source/book authority without confusing stale intent with authority.
CREATE FUNCTION banking.collection_label_reauthorize(actor uuid,w uuid,p banking.collection_label_preview)RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE z jsonb;s banking.collection_source;BEGIN
 IF p.workspace_id IS DISTINCT FROM w OR p.principal_id IS DISTINCT FROM actor THEN PERFORM banking.collection_error('P2801');END IF;
 FOR z IN SELECT value FROM jsonb_array_elements(p.review_rows)LOOP
  SELECT * INTO s FROM banking.collection_visible_sources(w)WHERE source_account_id=(z->>'source_id')::uuid;
  IF NOT FOUND OR NOT EXISTS(SELECT FROM banking.collection_facts(w)f WHERE f.source_account_id=s.source_account_id AND f.observation_id=(z->>'observation_id')::uuid)THEN PERFORM banking.collection_error('P2801');END IF;
  IF z->'assignment'->>'state'='assigned'AND NOT banking.collection_label_book_read(actor,s.legacy_entity_id,s.legacy_book_id)THEN PERFORM banking.collection_error('P2801');END IF;
 END LOOP;
END $$;
CREATE FUNCTION banking.collection_label_render_command(c banking.collection_label_command)RETURNS jsonb LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$
 SELECT jsonb_build_object('profile','private-collection-command-v1','workspace_id',c.workspace_id,'command',jsonb_build_object('workspace_id',c.workspace_id,'command_id',c.command_id,'prepare_key',c.prepare_key,'state',c.state,'acknowledged',c.acknowledged_at IS NOT NULL,'preview',banking.collection_label_render_preview(p)))FROM banking.collection_label_preview p WHERE p.workspace_id=c.workspace_id AND p.preview_id=c.preview_id
$$;
CREATE FUNCTION banking.prepare_collection_label_command(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;id uuid;k text;p banking.collection_label_preview;c banking.collection_label_command;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);id:=ledger.uuid_value(v->'preview_id');k:=ledger.text_value(v->'prepare_key');IF k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=id AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;PERFORM banking.collection_label_reauthorize(actor,w,p);
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND prepare_key=k;
 IF FOUND THEN IF c.preview_id<>id THEN PERFORM banking.collection_error('P2802');END IF;RETURN banking.collection_label_render_command(c);END IF;
 IF EXISTS(SELECT FROM banking.collection_label_command WHERE workspace_id=w AND acknowledged_at IS NULL)THEN PERFORM banking.collection_error('P2802');END IF;
 PERFORM banking.collection_label_pins(p.snapshot,banking.collection_label_snapshot(w));IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;
 INSERT INTO banking.collection_label_command(workspace_id,principal_id,preview_id,prepare_key)VALUES(w,actor,id,k)RETURNING * INTO c;
 INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,'prepare',id,c.command_id);
 res:=banking.collection_label_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE FUNCTION banking.pending_collection_label_command(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_label_command;p banking.collection_label_preview;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND acknowledged_at IS NULL;
 -- Read-only owners may establish that no work is pending; saved intent still requires classify.
 IF NOT FOUND THEN RETURN jsonb_build_object('profile','private-collection-command-v1','workspace_id',w,'command',NULL);END IF;
 IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND 'classify'=ANY(capabilities))THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_label_reauthorize(actor,w,p);
 res:=banking.collection_label_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;
-- Analytic decisions and terminal receipt are one atomic transaction; execute never posts.
CREATE FUNCTION banking.transition_collection_label_command(actor uuid,w uuid,id uuid,operation text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;c banking.collection_label_command;p banking.collection_label_preview;ls banking.collection_label_state;z jsonb;action_value jsonb;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY[]::text[]);
 IF operation IS NULL OR operation<>ALL(ARRAY['execute','cancel','acknowledge'])THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND command_id=id AND principal_id=actor FOR UPDATE;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_label_reauthorize(actor,w,p);
 IF operation='acknowledge'THEN
  IF c.state='prepared'THEN PERFORM banking.collection_error('P2802');END IF;
  IF c.acknowledged_at IS NULL THEN UPDATE banking.collection_label_command SET acknowledged_at=clock_timestamp()WHERE workspace_id=w AND command_id=id RETURNING * INTO c;INSERT INTO audit.collection_label_action(workspace_id,principal_id,operation,preview_id,command_id)VALUES(w,actor,'acknowledge',p.preview_id,id);END IF;
 ELSIF c.state='prepared'THEN
  IF operation='execute'THEN
   PERFORM banking.collection_label_pins(p.snapshot,banking.collection_label_snapshot(w));IF p.expires_at<=clock_timestamp()THEN PERFORM banking.collection_error('P2803');END IF;
   action_value:=banking.collection_label_action(w,p.action);SELECT * INTO ls FROM banking.collection_label_state WHERE workspace_id=w;
   -- Validate every explicit source revision before any insert; a failing member rolls back all.
   FOR z IN SELECT value FROM jsonb_array_elements(p.selection)LOOP
    IF NOT EXISTS(SELECT FROM banking.collection_facts(w)f WHERE f.source_account_id=(z->>'source_id')::uuid AND f.observation_id=(z->>'observation_id')::uuid AND f.observation_revision=(z->>'observation_revision')::bigint)THEN PERFORM banking.collection_error('P2803');END IF;
   END LOOP;
   IF action_value->>'kind'='taxonomy'THEN
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
 res:=banking.collection_label_render_command(c);IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
-- Only six reviewed exact entrypoints; no helper, table or owner provisioning authority.
DO $$DECLARE f regprocedure;BEGIN
 FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='banking'AND(p.proname LIKE 'collection_label_%'OR p.proname IN('get_collection_taxonomy','query_collection_review','preview_collection_labels','prepare_collection_label_command','pending_collection_label_command','transition_collection_label_command'))LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator',f);
 END LOOP;
END $$;
REVOKE ALL ON banking.collection_taxonomy_edition,banking.collection_label_state,banking.collection_label_preview,banking.collection_label_command,banking.collection_label_decision,audit.collection_label_action FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION banking.get_collection_taxonomy(uuid,uuid),banking.query_collection_review(uuid,uuid,text),banking.preview_collection_labels(uuid,uuid,text),banking.prepare_collection_label_command(uuid,uuid,text),banking.pending_collection_label_command(uuid,uuid),banking.transition_collection_label_command(uuid,uuid,uuid,text,text)TO bookkeeping_api;
COMMIT;
