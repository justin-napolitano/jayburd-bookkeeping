-- #429; docs/contracts/source-activity-api.md. Exact private reads, no financial mutation.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- An equality-only UUID probe avoids competing scope-prefix plans; scoped predicates remain mandatory.
CREATE INDEX review_assignment_activity_observation ON banking.review_assignment USING hash(observation_id);
-- Owner-only minimal projection. Bounded lateral lookups retain key probes under cold statistics.
CREATE FUNCTION banking.activity_facts(e uuid,b uuid,sid uuid)
RETURNS TABLE(observation_id uuid,observation_revision bigint,booked_date date,description text,source_state text,
 currency_version_id uuid,currency text,numeric_code text,minor_unit integer,amount numeric,direction text,
 review_id uuid,review_state text,merchant_status text,merchant_label text,bucket text,attention boolean,ambiguous boolean)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT o.observation_id,q.revision,q.booked_date,q.description,q.source_state,q.currency_version_id,c.alphabetic_code::text,c.numeric_code::text,c.minor_unit::integer,q.amount::numeric,q.direction,
 r.review_id,r.state,
 CASE WHEN q.adapter_profile<>'jayburd-plaid-transactions/1' THEN 'unavailable' WHEN q.provider_fields->'merchant_name'='null'::jsonb OR btrim(q.provider_fields->>'merchant_name',E' \t\n\r\f\013')='' THEN 'not_provided' ELSE 'attributed' END,
 CASE WHEN q.adapter_profile='jayburd-plaid-transactions/1' AND q.provider_fields->'merchant_name'<>'null'::jsonb AND btrim(q.provider_fields->>'merchant_name',E' \t\n\r\f\013')<>'' THEN q.provider_fields->>'merchant_name' END,
 CASE WHEN q.source_state='removed' THEN 'removed' WHEN q.source_state='pending' THEN 'pending' WHEN q.amount=0 THEN 'zero'
 WHEN c.minor_unit IS NULL OR q.amount<>trunc(q.amount,c.minor_unit::integer) OR q.booked_date<c.valid_from OR c.valid_to IS NOT NULL AND q.booked_date>=c.valid_to OR n.overlap_review_required OR r.state='overlap_hold' THEN 'quality_hold' ELSE 'eligible' END,
 (r.state='correction_hold' AND q.source_state='booked' AND q.amount>0 AND c.minor_unit IS NOT NULL AND q.amount=trunc(q.amount,c.minor_unit::integer) AND NOT n.overlap_review_required)
 OR EXISTS(SELECT FROM banking.provider_review_exception x WHERE x.entity_id=e AND x.book_id=b AND x.observation_id=o.observation_id AND x.current_revision=o.current_revision),
 (q.observation_id IS NULL OR c.currency_version_id IS NULL OR n.namespace_id IS NULL OR r.review_id IS NOT NULL AND r.source_account_id<>sid
 OR q.source_state='booked' AND q.amount>0 AND r.review_id IS NULL
 OR q.adapter_profile='jayburd-plaid-transactions/1' AND (NOT(q.provider_fields ? 'merchant_name') OR jsonb_typeof(q.provider_fields->'merchant_name') NOT IN ('string','null') OR length(q.provider_fields->>'merchant_name')>500))
 FROM banking.source_observation o
 LEFT JOIN LATERAL (SELECT qq.* FROM banking.observation_revision qq WHERE qq.entity_id=e AND qq.book_id=b AND qq.observation_id=o.observation_id AND qq.revision=o.current_revision LIMIT 1) q ON true
 LEFT JOIN LATERAL (SELECT nn.namespace_id,nn.overlap_review_required FROM banking.source_namespace nn WHERE nn.entity_id=e AND nn.book_id=b AND nn.source_account_id=sid AND nn.namespace_id=o.namespace_id LIMIT 1) n ON true
 LEFT JOIN LATERAL (SELECT aa.review_id FROM banking.review_assignment aa WHERE aa.observation_id=o.observation_id AND ROW(aa.entity_id,aa.book_id) IS NOT DISTINCT FROM ROW(e,b) LIMIT 1) a ON true
 LEFT JOIN LATERAL (SELECT rr.review_id,rr.source_account_id,rr.state FROM banking.review_item rr WHERE rr.entity_id=e AND rr.book_id=b AND rr.review_id=a.review_id LIMIT 1) r ON true
 LEFT JOIN reference.currency c ON c.currency_version_id=q.currency_version_id
 WHERE o.entity_id=e AND o.book_id=b AND o.source_account_id=sid
$$;
REVOKE ALL ON FUNCTION banking.activity_facts(uuid,uuid,uuid) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync,bookkeeping_webhook,bookkeeping_verifier;
-- One authorized transaction owns summary, exact selector, complete totals and pinned drill pages.
CREATE FUNCTION banking.query_source_activity(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb;s banking.source_account;sid uuid;unit_id uuid;from_day date;through_day date;search_value text;kind_value text;page_limit integer;
 snap jsonb;selector_value text;cur jsonb;anchor text;cursor_text text;merchant_status_value text;merchant_label_value text;result_value jsonb;bad_anchor boolean;unknown_unit boolean;unit_count bigint;
BEGIN
 v:=banking.import_json(raw,8192);
 IF NOT(v ?& ARRAY['profile','source_account_id','starts_on','through_date','search','kind','currency_version_id','merchant','snapshot','cursor','limit']) OR EXISTS(SELECT FROM jsonb_object_keys(v) k WHERE k<>ALL(ARRAY['profile','source_account_id','starts_on','through_date','search','kind','currency_version_id','merchant','snapshot','cursor','limit'])) THEN PERFORM banking.review_error('P2500');END IF;
 IF ledger.text_value(v->'profile')<>'source-activity-v1' THEN PERFORM banking.review_error('P2500');END IF;
 sid:=ledger.uuid_value(v->'source_account_id');
 IF sid='00000000-0000-0000-0000-000000000000' THEN PERFORM banking.review_error('P2500');END IF;
 from_day:=ledger.date_value(ledger.text_value(v->'starts_on'));through_day:=ledger.date_value(ledger.text_value(v->'through_date'));
 IF from_day>through_day OR jsonb_typeof(v->'search') IS DISTINCT FROM 'string' OR length(v->>'search')>100 THEN PERFORM banking.review_error('P2500');END IF;
 search_value:=v->>'search';kind_value:=ledger.text_value(v->'kind');
 IF kind_value NOT IN ('summary','observations','merchants') OR jsonb_typeof(v->'limit') IS DISTINCT FROM 'number' OR v->>'limit' !~ '^[1-9][0-9]{0,2}$' THEN PERFORM banking.review_error('P2500');END IF;
 page_limit:=(v->>'limit')::integer;
 IF page_limit>(CASE WHEN kind_value='observations' THEN 100 ELSE 20 END) OR kind_value='summary' AND page_limit<>20 THEN PERFORM banking.review_error('P2500');END IF;
 IF kind_value='summary' THEN
  IF v->'currency_version_id'<>'null'::jsonb OR v->'merchant'<>'null'::jsonb OR v->'cursor'<>'null'::jsonb THEN PERFORM banking.review_error('P2500');END IF;
 ELSE unit_id:=ledger.uuid_value(v->'currency_version_id');IF unit_id='00000000-0000-0000-0000-000000000000' OR v->'snapshot'='null'::jsonb THEN PERFORM banking.review_error('P2500');END IF;END IF;
 IF v->'merchant'<>'null'::jsonb THEN
  IF kind_value<>'observations' THEN PERFORM banking.review_error('P2500');END IF;
  IF jsonb_typeof(v->'merchant') IS DISTINCT FROM 'object' OR NOT(v->'merchant' ?& ARRAY['status','label']) THEN PERFORM banking.review_error('P2500');END IF;
  IF EXISTS(SELECT FROM jsonb_object_keys(v->'merchant') k WHERE k<>ALL(ARRAY['status','label'])) THEN PERFORM banking.review_error('P2500');END IF;
  merchant_status_value:=ledger.text_value(v->'merchant'->'status');
  IF merchant_status_value NOT IN ('attributed','not_provided','unavailable') THEN PERFORM banking.review_error('P2500');END IF;
  IF merchant_status_value='attributed' THEN
   merchant_label_value:=ledger.text_value(v->'merchant'->'label');
   IF length(merchant_label_value)>500 OR btrim(merchant_label_value,E' \t\n\r\f\013')='' THEN PERFORM banking.review_error('P2500');END IF;
  ELSIF v->'merchant'->'label'<>'null'::jsonb THEN PERFORM banking.review_error('P2500');END IF;
 END IF;
 -- Current principal/entity/source grants precede all snapshots, labels, counts and cursor diagnostics.
 s:=banking.import_authorize(actor,e,b,sid,false);
 -- Check shared-review ambiguity once, rather than repeating an aggregate for every observation.
 IF EXISTS(
  WITH counts AS MATERIALIZED (SELECT review_id FROM banking.review_assignment WHERE entity_id=e AND book_id=b GROUP BY review_id HAVING count(*)>1)
  SELECT FROM counts x WHERE EXISTS(SELECT FROM banking.review_item r WHERE r.entity_id=e AND r.book_id=b AND r.review_id=x.review_id AND r.source_account_id=sid)
 ) THEN PERFORM banking.review_error('P2502');END IF;
 IF EXISTS(SELECT FROM banking.activity_facts(e,b,sid) WHERE ambiguous) THEN PERFORM banking.review_error('P2502');END IF;
 SELECT jsonb_build_object('source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'review_revision_sum',coalesce(sum(r.review_revision::numeric),0)::text,'filter_digest',md5(jsonb_build_object('profile','source-activity-v1','entity_id',e,'book_id',b,'source_account_id',sid,'starts_on',v->'starts_on','through_date',v->'through_date','search',v->'search')::text))
 INTO snap FROM banking.review_item r WHERE r.entity_id=e AND r.book_id=b AND r.source_account_id=sid;
 IF v->'snapshot'<>'null'::jsonb THEN
  PERFORM ledger.keys(v->'snapshot',ARRAY['source_revision','binding_revision','review_revision_sum','filter_digest']);
  PERFORM banking.import_revision(v->'snapshot'->'source_revision');
  IF banking.import_revision(v->'snapshot'->'binding_revision')<1 OR jsonb_typeof(v->'snapshot'->'review_revision_sum') IS DISTINCT FROM 'string' OR v->'snapshot'->>'review_revision_sum' !~ '^(0|[1-9][0-9]{0,38})$' THEN PERFORM banking.review_error('P2500');END IF;
  IF ledger.text_value(v->'snapshot'->'filter_digest') COLLATE "C" !~ '^[0-9a-f]{32}$' THEN PERFORM banking.review_error('P2500');END IF;
  IF v->'snapshot' IS DISTINCT FROM snap THEN PERFORM banking.review_error('P2503');END IF;
 END IF;
 selector_value:=md5((v-ARRAY['snapshot','cursor','limit'])::text);
 IF v->'cursor'<>'null'::jsonb THEN
  cursor_text:=ledger.text_value(v->'cursor');
  IF length(cursor_text) NOT BETWEEN 2 AND 2048 OR cursor_text COLLATE "C" !~ '^([0-9a-f]{2})+$' THEN PERFORM banking.review_error('P2500');END IF;
  cur:=banking.import_json(convert_from(decode(cursor_text,'hex'),'UTF8'),1024);
  PERFORM ledger.keys(cur,ARRAY['entity_id','book_id','source_account_id','snapshot','selector','kind','anchor']);
  IF ROW(cur->>'entity_id',cur->>'book_id',cur->>'source_account_id',cur->'snapshot',cur->>'selector',cur->>'kind') IS DISTINCT FROM ROW(e::text,b::text,sid::text,snap,selector_value,kind_value) THEN PERFORM banking.review_error('P2500');END IF;
  anchor:=ledger.text_value(cur->'anchor');
 END IF;
 WITH facts AS MATERIALIZED (SELECT * FROM banking.activity_facts(e,b,sid)),
 matched AS MATERIALIZED (SELECT * FROM facts WHERE booked_date BETWEEN from_day AND through_day AND strpos(lower(description COLLATE "C"),lower(search_value COLLATE "C"))>0),
 eligible AS MATERIALIZED (SELECT * FROM matched WHERE bucket='eligible'),
 units AS MATERIALIZED (
  SELECT currency_version_id,currency,numeric_code,minor_unit,count(*)::text AS count,
   coalesce(sum(amount) FILTER(WHERE direction='inflow'),0) AS inflow,coalesce(sum(amount) FILTER(WHERE direction='outflow'),0) AS outflow
  FROM eligible GROUP BY currency_version_id,currency,numeric_code,minor_unit
 ), selected AS MATERIALIZED (
  SELECT * FROM eligible WHERE currency_version_id=unit_id AND (merchant_status_value IS NULL OR merchant_status=merchant_status_value AND merchant_label COLLATE "C" IS NOT DISTINCT FROM merchant_label_value COLLATE "C")
 ), groups AS MATERIALIZED (
  SELECT merchant_status COLLATE "C" AS merchant_status,merchant_label COLLATE "C" AS merchant_label,count(*)::text AS count,coalesce(sum(amount) FILTER(WHERE direction='inflow'),0) AS inflow,coalesce(sum(amount) FILTER(WHERE direction='outflow'),0) AS outflow
  FROM selected GROUP BY merchant_status COLLATE "C",merchant_label COLLATE "C"
 ), ranked AS MATERIALIZED (
  SELECT *,row_number() OVER(ORDER BY outflow DESC,merchant_status COLLATE "C",merchant_label COLLATE "C" NULLS FIRST) AS position FROM groups
 ), rendered AS MATERIALIZED (
  SELECT observation_id::text AS key,row_number() OVER(ORDER BY booked_date,observation_id) AS position,
   jsonb_build_object('observation_id',observation_id,'observation_revision',observation_revision::text,'booked_date',to_char(booked_date,'YYYY-MM-DD'),'description',description,'source_state',source_state,'direction',direction,'amount',trim_scale(amount)::text,'currency_version_id',currency_version_id,'currency',currency,'numeric_code',numeric_code,'minor_unit',minor_unit,'review_id',review_id,'review_state',review_state,'merchant',jsonb_build_object('status',merchant_status,'label',merchant_label,'profile','provider-merchant-exact-v1')) AS body
  FROM selected WHERE kind_value='observations'
  UNION ALL
  SELECT position::text,position,jsonb_build_object('status',merchant_status,'label',merchant_label,'profile','provider-merchant-exact-v1','count',count,'inflow',trim_scale(inflow)::text,'outflow',trim_scale(outflow)::text,'net_outflow',trim_scale(outflow-inflow)::text)
  FROM ranked WHERE kind_value='merchants'
 ), picked AS MATERIALIZED (
  SELECT * FROM rendered WHERE anchor IS NULL OR position>(SELECT position FROM rendered WHERE key=anchor) ORDER BY position LIMIT page_limit
 )
 SELECT jsonb_build_object('profile','source-activity-v1','entity_id',e,'book_id',b,'source_account_id',sid,'source_state',s.state,'snapshot',snap,'selector',v-ARRAY['snapshot','cursor','limit'],
  'generated_at',to_char(statement_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'bank_balance',NULL,'functional_total',NULL,'categorized_spending',NULL,'last_successful_sync',NULL,'unavailable_reasons',jsonb_build_object('bank_balance','not_projected','functional_total','no_valuation','categorized_spending','not_implemented','last_successful_sync','not_projected'),'history_completeness','unknown',
  'available_from',(SELECT to_char(min(booked_date),'YYYY-MM-DD') FROM facts),'available_through',(SELECT to_char(max(booked_date),'YYYY-MM-DD') FROM facts),
  'inventory',jsonb_build_object('source_count',(SELECT count(*)::text FROM facts),'matching_count',(SELECT count(*)::text FROM matched),'eligible_count',(SELECT count(*)::text FROM matched WHERE bucket='eligible'),'pending_count',(SELECT count(*)::text FROM matched WHERE bucket='pending'),'removed_count',(SELECT count(*)::text FROM matched WHERE bucket='removed'),'zero_count',(SELECT count(*)::text FROM matched WHERE bucket='zero'),'quality_hold_count',(SELECT count(*)::text FROM matched WHERE bucket='quality_hold'),'correction_attention_count',(SELECT count(*)::text FROM matched WHERE attention),'partial',EXISTS(SELECT FROM matched WHERE bucket='quality_hold')),
  'units',CASE WHEN kind_value='summary' THEN coalesce((SELECT jsonb_agg(jsonb_build_object('currency_version_id',currency_version_id,'currency',currency,'numeric_code',numeric_code,'minor_unit',minor_unit,'count',count,'inflow',trim_scale(inflow)::text,'outflow',trim_scale(outflow)::text,'net_inflow',trim_scale(inflow-outflow)::text) ORDER BY currency_version_id) FROM units),'[]'::jsonb) ELSE '[]'::jsonb END,
  'totals',CASE WHEN kind_value<>'summary' THEN (SELECT jsonb_build_object('count',count(*)::text,'inflow',trim_scale(coalesce(sum(amount) FILTER(WHERE direction='inflow'),0))::text,'outflow',trim_scale(coalesce(sum(amount) FILTER(WHERE direction='outflow'),0))::text,'net_inflow',trim_scale(coalesce(sum(CASE WHEN direction='inflow' THEN amount ELSE -amount END),0))::text) FROM selected) END,
  'items',CASE WHEN kind_value='observations' THEN coalesce((SELECT jsonb_agg(body ORDER BY position) FROM picked),'[]'::jsonb) ELSE '[]'::jsonb END,
  'merchant_groups',CASE WHEN kind_value='merchants' THEN coalesce((SELECT jsonb_agg(body ORDER BY position) FROM picked),'[]'::jsonb) ELSE '[]'::jsonb END,
  'group_count',CASE WHEN kind_value='merchants' THEN (SELECT count(*)::text FROM groups) END,
  'remaining_group_count',CASE WHEN kind_value='merchants' THEN (SELECT count(*)::text FROM rendered WHERE position>coalesce((SELECT max(position) FROM picked),(SELECT position FROM rendered WHERE key=anchor),0)) END,
  'next_cursor',CASE WHEN EXISTS(SELECT FROM rendered WHERE position>(SELECT max(position) FROM picked)) THEN encode(convert_to(jsonb_build_object('entity_id',e,'book_id',b,'source_account_id',sid,'snapshot',snap,'selector',selector_value,'kind',kind_value,'anchor',(SELECT key FROM picked ORDER BY position DESC LIMIT 1))::text,'UTF8'),'hex') END),
  anchor IS NOT NULL AND NOT EXISTS(SELECT FROM rendered WHERE key=anchor),(SELECT count(*) FROM units),kind_value<>'summary' AND NOT EXISTS(SELECT FROM units WHERE currency_version_id=unit_id)
 INTO result_value,bad_anchor,unit_count,unknown_unit;
 IF bad_anchor THEN PERFORM banking.review_error('P2500');END IF;
 IF unknown_unit OR unit_count>100 OR octet_length(result_value::text)>1048576 THEN PERFORM banking.review_error('P2505');END IF;
 RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.review_error('P2500');WHEN SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501');END $$;
REVOKE ALL ON FUNCTION banking.query_source_activity(uuid,uuid,uuid,text) FROM PUBLIC,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync,bookkeeping_webhook,bookkeeping_verifier;
GRANT EXECUTE ON FUNCTION banking.query_source_activity(uuid,uuid,uuid,text) TO bookkeeping_api;
COMMIT;
