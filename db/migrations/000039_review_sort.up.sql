-- #406; docs/contracts/transaction-review-sort.md. No financial state changes.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Current review lookup uses review_id; the original assignment key is observation_id.
CREATE INDEX review_assignment_scope_review ON banking.review_assignment(entity_id,book_id,review_id);
-- Rank the complete currently authorized source snapshot without exposing private cursor keys.
CREATE FUNCTION banking.sorted_review_page(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;cur jsonb;sid uuid;s banking.source_account;filter_value text;page_limit integer;last_id uuid;cursor_text text;selected_item banking.review_item;row_value record;rows_value jsonb:='[]';n integer:=0;next_value text;sort_value text;order_value text;total_value bigint;start_value bigint:=0;
BEGIN
 v:=banking.import_json(raw,4096);
 IF NOT(v ?& ARRAY['source_account_id','state','limit','cursor','sort','order']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['source_account_id','state','limit','cursor','sort','order'])) THEN PERFORM banking.review_error('P2500');END IF;
 sid:=ledger.uuid_value(v->'source_account_id');sort_value:=ledger.text_value(v->'sort');order_value:=ledger.text_value(v->'order');
 IF sort_value NOT IN ('date','description','direction','amount','status') OR order_value NOT IN ('asc','desc') THEN PERFORM banking.review_error('P2500');END IF;
 IF v->'state'<>'null'::jsonb THEN filter_value:=ledger.text_value(v->'state');IF filter_value NOT IN ('needs_review','overlap_hold','draft_ready','posted','correction_hold') THEN PERFORM banking.review_error('P2500');END IF;END IF;
 IF jsonb_typeof(v->'limit')<>'number' OR v->>'limit' !~ '^[1-9][0-9]{0,2}$' OR (v->>'limit')::integer>100 THEN PERFORM banking.review_error('P2500');END IF;page_limit:=(v->>'limit')::integer;
 s:=banking.import_authorize(actor,e,b,sid,false);
 IF v->'cursor'<>'null'::jsonb THEN
  cursor_text:=ledger.text_value(v->'cursor');
  IF length(cursor_text) NOT BETWEEN 2 AND 2048 OR cursor_text COLLATE "C" !~ '^([0-9a-f]{2})+$' THEN PERFORM banking.review_error('P2500');END IF;
  cur:=banking.import_json(convert_from(decode(cursor_text,'hex'),'UTF8'),1024);
  IF NOT(cur ?& ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id','sort','order']) OR EXISTS(SELECT FROM jsonb_object_keys(cur) x WHERE x<>ALL(ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id','sort','order'])) THEN PERFORM banking.review_error('P2500');END IF;
  IF ROW(ledger.uuid_value(cur->'entity_id'),ledger.uuid_value(cur->'book_id'),ledger.uuid_value(cur->'source_account_id'),cur->'state',cur->'sort',cur->'order') IS DISTINCT FROM ROW(e,b,sid,v->'state',v->'sort',v->'order') THEN PERFORM banking.review_error('P2500');END IF;
  IF banking.import_revision(cur->'source_revision')<>s.source_revision THEN PERFORM banking.review_error('P2503');END IF;
  last_id:=ledger.uuid_value(cur->'last_id');
  IF NOT EXISTS(SELECT FROM banking.review_item WHERE entity_id=e AND book_id=b AND source_account_id=sid AND review_id=last_id AND (filter_value IS NULL OR state=filter_value)) THEN PERFORM banking.review_error('P2500');END IF;
 END IF;
 SELECT count(*) INTO total_value FROM banking.review_item WHERE entity_id=e AND book_id=b AND source_account_id=sid AND (filter_value IS NULL OR state=filter_value);
 -- Preserve review_fact's one-current-observation invariant before ranking with set-based joins.
 IF EXISTS(
  SELECT r.review_id FROM banking.review_item r
  LEFT JOIN LATERAL (SELECT aa.observation_id FROM banking.review_assignment aa WHERE aa.entity_id=r.entity_id AND aa.book_id=r.book_id AND aa.review_id=r.review_id LIMIT 2) a ON true
  WHERE r.entity_id=e AND r.book_id=b AND r.source_account_id=sid AND (filter_value IS NULL OR r.state=filter_value)
  GROUP BY r.review_id HAVING count(a.observation_id)<>1
 ) THEN PERFORM banking.review_error('P2502');END IF;
 FOR row_value IN
  WITH ordered AS MATERIALIZED (
   SELECT r AS item,row_number() OVER(ORDER BY CASE WHEN sort_value='date' AND order_value='asc' THEN q.booked_date END ASC,
   CASE WHEN sort_value='date' AND order_value='desc' THEN q.booked_date END DESC,
   CASE WHEN sort_value='description' AND order_value='asc' THEN lower(q.description) COLLATE "C" END ASC,
   CASE WHEN sort_value='description' AND order_value='desc' THEN lower(q.description) COLLATE "C" END DESC,
   CASE WHEN sort_value='direction' AND order_value='asc' THEN q.direction COLLATE "C" END ASC,
   CASE WHEN sort_value='direction' AND order_value='desc' THEN q.direction COLLATE "C" END DESC,
   CASE WHEN sort_value='amount' AND order_value='asc' THEN c.alphabetic_code COLLATE "C" END ASC,
   CASE WHEN sort_value='amount' AND order_value='desc' THEN c.alphabetic_code COLLATE "C" END DESC,
   CASE WHEN sort_value='amount' AND order_value='asc' THEN q.amount END ASC,
   CASE WHEN sort_value='amount' AND order_value='desc' THEN q.amount END DESC,
   CASE WHEN sort_value='status' AND order_value='asc' THEN r.state COLLATE "C" END ASC,
   CASE WHEN sort_value='status' AND order_value='desc' THEN r.state COLLATE "C" END DESC,r.review_id ASC) AS position
   FROM banking.review_item r
   JOIN LATERAL (SELECT aa.observation_id FROM banking.review_assignment aa WHERE aa.entity_id=r.entity_id AND aa.book_id=r.book_id AND aa.review_id=r.review_id LIMIT 2) a ON true
   JOIN LATERAL (SELECT oo.* FROM banking.source_observation oo WHERE oo.entity_id=r.entity_id AND oo.book_id=r.book_id AND oo.observation_id=a.observation_id LIMIT 1) o ON true
   JOIN LATERAL (SELECT qq.* FROM banking.observation_revision qq WHERE qq.entity_id=r.entity_id AND qq.book_id=r.book_id AND qq.observation_id=o.observation_id AND qq.revision=o.current_revision LIMIT 1) q ON true
   JOIN reference.currency c ON c.currency_version_id=q.currency_version_id
   WHERE r.entity_id=e AND r.book_id=b AND r.source_account_id=sid AND (filter_value IS NULL OR r.state=filter_value)
  ) SELECT item,position FROM ordered WHERE last_id IS NULL OR position>(SELECT position FROM ordered WHERE (item).review_id=last_id) ORDER BY position LIMIT page_limit+1
 LOOP
  n:=n+1;
  IF n>page_limit THEN next_value:=encode(convert_to(jsonb_build_object('entity_id',e,'book_id',b,'source_account_id',sid,'source_revision',s.source_revision::text,'state',filter_value,'last_id',last_id,'sort',sort_value,'order',order_value)::text,'UTF8'),'hex');EXIT;END IF;
  selected_item:=row_value.item;IF n=1 THEN start_value:=row_value.position;END IF;
  rows_value:=rows_value||jsonb_build_array(banking.render_review_summary(selected_item));last_id:=selected_item.review_id;
 END LOOP;
 RETURN jsonb_build_object('profile','source-review-page-v1','source_account_id',sid,'source_revision',s.source_revision::text,'items',rows_value,'next_cursor',next_value,'sort',sort_value,'order',order_value,'matching_count',total_value::text,'start_index',start_value::text);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.review_error('P2500');WHEN SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501');END $$;
REVOKE ALL ON FUNCTION banking.sorted_review_page(uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync,bookkeeping_webhook,bookkeeping_verifier;
-- Preserve the prior exact legacy request/projection when no sort pair is supplied.
CREATE OR REPLACE FUNCTION banking.list_review_items(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; cur jsonb; sid uuid; s banking.source_account; filter_value text; page_limit integer; last_id uuid; cursor_text text; r banking.review_item; rows_value jsonb:='[]'; n integer:=0; next_value text;
BEGIN
 v:=banking.import_json(raw,4096);
 IF v ? 'sort' OR v ? 'order' THEN RETURN banking.sorted_review_page(actor,e,b,raw);END IF;
 IF NOT(v ?& ARRAY['source_account_id','state','limit','cursor']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['source_account_id','state','limit','cursor'])) THEN PERFORM banking.review_error('P2500'); END IF;
 sid:=ledger.uuid_value(v->'source_account_id');
 IF v->'state'<>'null'::jsonb THEN filter_value:=ledger.text_value(v->'state'); IF filter_value NOT IN ('needs_review','overlap_hold','draft_ready','posted','correction_hold') THEN PERFORM banking.review_error('P2500'); END IF; END IF;
 IF jsonb_typeof(v->'limit')<>'number' OR v->>'limit' !~ '^[1-9][0-9]{0,2}$' OR (v->>'limit')::integer>100 THEN PERFORM banking.review_error('P2500'); END IF; page_limit:=(v->>'limit')::integer;
 s:=banking.import_authorize(actor,e,b,sid,false);
 IF v->'cursor'<>'null'::jsonb THEN
  cursor_text:=ledger.text_value(v->'cursor');
  IF length(cursor_text) NOT BETWEEN 2 AND 2048 OR cursor_text COLLATE "C" !~ '^([0-9a-f]{2})+$' THEN PERFORM banking.review_error('P2500'); END IF;
  cur:=banking.import_json(convert_from(decode(cursor_text,'hex'),'UTF8'),1024);
  IF NOT(cur ?& ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id']) OR EXISTS(SELECT FROM jsonb_object_keys(cur) x WHERE x<>ALL(ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id'])) THEN PERFORM banking.review_error('P2500'); END IF;
  IF ROW(ledger.uuid_value(cur->'entity_id'),ledger.uuid_value(cur->'book_id'),ledger.uuid_value(cur->'source_account_id'),cur->'state') IS DISTINCT FROM ROW(e,b,sid,v->'state') THEN PERFORM banking.review_error('P2500'); END IF;
  IF banking.import_revision(cur->'source_revision')<>s.source_revision THEN PERFORM banking.review_error('P2503'); END IF;
  last_id:=ledger.uuid_value(cur->'last_id');
 END IF;
 FOR r IN SELECT * FROM banking.review_item WHERE entity_id=e AND book_id=b AND source_account_id=sid AND (filter_value IS NULL OR state=filter_value) AND (last_id IS NULL OR review_id>last_id) ORDER BY review_id LIMIT page_limit+1 LOOP
  n:=n+1;
  IF n>page_limit THEN next_value:=encode(convert_to(jsonb_build_object('entity_id',e,'book_id',b,'source_account_id',sid,'source_revision',s.source_revision::text,'state',filter_value,'last_id',last_id)::text,'UTF8'),'hex'); EXIT; END IF;
  rows_value:=rows_value||jsonb_build_array(banking.render_review_summary(r)); last_id:=r.review_id;
 END LOOP;
 RETURN jsonb_build_object('profile','source-review-page-v1','source_account_id',sid,'source_revision',s.source_revision::text,'items',rows_value,'next_cursor',next_value);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.review_error('P2500'); WHEN SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501'); END $$;
COMMIT;
