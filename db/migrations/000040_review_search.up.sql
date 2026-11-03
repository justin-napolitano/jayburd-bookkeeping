-- #408; docs/contracts/transaction-search.md. Scoped read only, no source or journal mutation.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- One authorized snapshot owns filtering, ordering, pagination and observed coverage.
CREATE FUNCTION banking.search_review_items(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb;cur jsonb;sid uuid;s banking.source_account;filter_value text;page_limit integer;last_id uuid;cursor_text text;sort_value text;order_value text;search_value text;from_value date;through_value date;result_value jsonb;anchor_missing boolean;
BEGIN
 v:=banking.import_json(raw,8192);
 IF NOT(v ?& ARRAY['source_account_id','state','limit','cursor','sort','order','search','starts_on','through_date']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['source_account_id','state','limit','cursor','sort','order','search','starts_on','through_date'])) THEN PERFORM banking.review_error('P2500');END IF;
 sid:=ledger.uuid_value(v->'source_account_id');sort_value:=ledger.text_value(v->'sort');order_value:=ledger.text_value(v->'order');
 IF sort_value NOT IN ('date','description','direction','amount','status') OR order_value NOT IN ('asc','desc') THEN PERFORM banking.review_error('P2500');END IF;
 IF jsonb_typeof(v->'search') IS DISTINCT FROM 'string' OR length(v->>'search')>100 THEN PERFORM banking.review_error('P2500');END IF;search_value:=v->>'search';
 IF v->'starts_on'<>'null'::jsonb THEN from_value:=ledger.date_value(ledger.text_value(v->'starts_on'));END IF;
 IF v->'through_date'<>'null'::jsonb THEN through_value:=ledger.date_value(ledger.text_value(v->'through_date'));END IF;
 IF from_value>through_value THEN PERFORM banking.review_error('P2500');END IF;
 IF v->'state'<>'null'::jsonb THEN filter_value:=ledger.text_value(v->'state');IF filter_value NOT IN ('needs_review','overlap_hold','draft_ready','posted','correction_hold') THEN PERFORM banking.review_error('P2500');END IF;END IF;
 IF jsonb_typeof(v->'limit')<>'number' OR v->>'limit' !~ '^[1-9][0-9]{0,2}$' OR (v->>'limit')::integer>100 THEN PERFORM banking.review_error('P2500');END IF;page_limit:=(v->>'limit')::integer;
 s:=banking.import_authorize(actor,e,b,sid,false);
 IF v->'cursor'<>'null'::jsonb THEN
  cursor_text:=ledger.text_value(v->'cursor');
  IF length(cursor_text) NOT BETWEEN 2 AND 2048 OR cursor_text COLLATE "C" !~ '^([0-9a-f]{2})+$' THEN PERFORM banking.review_error('P2500');END IF;
  cur:=banking.import_json(convert_from(decode(cursor_text,'hex'),'UTF8'),1024);
  IF NOT(cur ?& ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id','sort','order','search','starts_on','through_date']) OR EXISTS(SELECT FROM jsonb_object_keys(cur) x WHERE x<>ALL(ARRAY['entity_id','book_id','source_account_id','source_revision','state','last_id','sort','order','search','starts_on','through_date'])) THEN PERFORM banking.review_error('P2500');END IF;
  IF ROW(ledger.uuid_value(cur->'entity_id'),ledger.uuid_value(cur->'book_id'),ledger.uuid_value(cur->'source_account_id'),cur->'state',cur->'sort',cur->'order',cur->'search',cur->'starts_on',cur->'through_date') IS DISTINCT FROM ROW(e,b,sid,v->'state',v->'sort',v->'order',v->'search',v->'starts_on',v->'through_date') THEN PERFORM banking.review_error('P2500');END IF;
  IF banking.import_revision(cur->'source_revision')<>s.source_revision THEN PERFORM banking.review_error('P2503');END IF;
  last_id:=ledger.uuid_value(cur->'last_id');
 END IF;
 -- Coverage uses all source facts, so ambiguity anywhere in that source rejects the read.
 IF EXISTS(
  SELECT r.review_id FROM banking.review_item r
  LEFT JOIN LATERAL (SELECT aa.observation_id FROM banking.review_assignment aa WHERE aa.entity_id=r.entity_id AND aa.book_id=r.book_id AND aa.review_id=r.review_id LIMIT 2) a ON true
  WHERE r.entity_id=e AND r.book_id=b AND r.source_account_id=sid
  GROUP BY r.review_id HAVING count(a.observation_id)<>1
 ) THEN PERFORM banking.review_error('P2502');END IF;
 WITH facts AS MATERIALIZED (
  SELECT r AS item,q.booked_date,q.description,q.direction,q.amount,c.alphabetic_code,r.state,r.review_id
  FROM banking.review_item r
  JOIN LATERAL (SELECT aa.observation_id FROM banking.review_assignment aa WHERE aa.entity_id=r.entity_id AND aa.book_id=r.book_id AND aa.review_id=r.review_id LIMIT 2) a ON true
  JOIN LATERAL (SELECT oo.* FROM banking.source_observation oo WHERE oo.entity_id=r.entity_id AND oo.book_id=r.book_id AND oo.observation_id=a.observation_id LIMIT 1) o ON true
  JOIN LATERAL (SELECT qq.* FROM banking.observation_revision qq WHERE qq.entity_id=r.entity_id AND qq.book_id=r.book_id AND qq.observation_id=o.observation_id AND qq.revision=o.current_revision LIMIT 1) q ON true
  JOIN reference.currency c ON c.currency_version_id=q.currency_version_id
  WHERE r.entity_id=e AND r.book_id=b AND r.source_account_id=sid
 ), matched AS MATERIALIZED (
  SELECT * FROM facts WHERE (filter_value IS NULL OR state=filter_value)
  AND (from_value IS NULL OR booked_date>=from_value) AND (through_value IS NULL OR booked_date<=through_value)
  AND strpos(lower(description COLLATE "C"),lower(search_value COLLATE "C"))>0
 ), ordered AS MATERIALIZED (
  SELECT *,row_number() OVER(ORDER BY
   CASE WHEN sort_value='date' AND order_value='asc' THEN booked_date END ASC,
   CASE WHEN sort_value='date' AND order_value='desc' THEN booked_date END DESC,
   CASE WHEN sort_value='description' AND order_value='asc' THEN lower(description) COLLATE "C" END ASC,
   CASE WHEN sort_value='description' AND order_value='desc' THEN lower(description) COLLATE "C" END DESC,
   CASE WHEN sort_value='direction' AND order_value='asc' THEN direction COLLATE "C" END ASC,
   CASE WHEN sort_value='direction' AND order_value='desc' THEN direction COLLATE "C" END DESC,
   CASE WHEN sort_value='amount' AND order_value='asc' THEN alphabetic_code COLLATE "C" END ASC,
   CASE WHEN sort_value='amount' AND order_value='desc' THEN alphabetic_code COLLATE "C" END DESC,
   CASE WHEN sort_value='amount' AND order_value='asc' THEN amount END ASC,
   CASE WHEN sort_value='amount' AND order_value='desc' THEN amount END DESC,
   CASE WHEN sort_value='status' AND order_value='asc' THEN state COLLATE "C" END ASC,
   CASE WHEN sort_value='status' AND order_value='desc' THEN state COLLATE "C" END DESC,review_id ASC) AS position
  FROM matched
 ), picked AS MATERIALIZED (
  SELECT * FROM ordered WHERE last_id IS NULL OR position>(SELECT position FROM ordered WHERE review_id=last_id) ORDER BY position LIMIT page_limit
 )
 SELECT jsonb_build_object('profile','source-review-query-v1','source_account_id',sid,'source_revision',s.source_revision::text,
  'items',coalesce((SELECT jsonb_agg(banking.render_review_summary(item) ORDER BY position) FROM picked),'[]'::jsonb),
  'sort',sort_value,'order',order_value,'matching_count',(SELECT count(*)::text FROM matched),'start_index',coalesce((SELECT min(position)::text FROM picked),'0'),
  'source_count',(SELECT count(*)::text FROM facts),'available_from',(SELECT to_char(min(booked_date),'YYYY-MM-DD') FROM facts),'available_through',(SELECT to_char(max(booked_date),'YYYY-MM-DD') FROM facts),
  'next_cursor',CASE WHEN EXISTS(SELECT FROM ordered WHERE position>(SELECT max(position) FROM picked)) THEN encode(convert_to(jsonb_build_object('entity_id',e,'book_id',b,'source_account_id',sid,'source_revision',s.source_revision::text,'state',filter_value,'last_id',(SELECT review_id FROM picked ORDER BY position DESC LIMIT 1),'sort',sort_value,'order',order_value,'search',search_value,'starts_on',v->'starts_on','through_date',v->'through_date')::text,'UTF8'),'hex') END),
  last_id IS NOT NULL AND NOT EXISTS(SELECT FROM matched WHERE review_id=last_id)
 INTO result_value,anchor_missing;
 IF anchor_missing THEN PERFORM banking.review_error('P2500');END IF;
 RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.review_error('P2500');WHEN SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501');END $$;
REVOKE ALL ON FUNCTION banking.search_review_items(uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync,bookkeeping_webhook,bookkeeping_verifier;
GRANT EXECUTE ON FUNCTION banking.search_review_items(uuid,uuid,uuid,text) TO bookkeeping_api;
COMMIT;
