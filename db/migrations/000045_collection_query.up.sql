-- #242; additive safe query/detail only. Existing commands, source facts and journals are unchanged.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;

-- Only v3 reads use these aliases; later additive rule pins cannot alter v1/v2 wire shapes.
CREATE FUNCTION banking.collection_query_snapshot(w uuid)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$SELECT banking.collection_tag_snapshot(w)$$;
CREATE FUNCTION banking.collection_query_pins(v jsonb,current_value jsonb)RETURNS void LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$SELECT banking.collection_tag_pins(v,current_value)$$;

-- One safe row remains one source observation. Missing provenance is never inferred from book custody.
CREATE FUNCTION banking.collection_query_rows(actor uuid,w uuid)
RETURNS TABLE(source_id uuid,observation_id uuid,observation_revision bigint,booked_date date,body jsonb)
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT r.source_id,r.observation_id,r.observation_revision,r.booked_date,
 r.body||jsonb_build_object('institution',jsonb_build_object('state','unavailable'),'legal_owner',jsonb_build_object('state','unknown'))
 FROM banking.collection_tag_rows(actor,w)r
$$;

-- Options are complete over currently visible sources, not an unrelated membership inventory.
CREATE FUNCTION banking.get_collection_taxonomy_v3(actor uuid,w uuid)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE res jsonb;books_value jsonb;currencies_value jsonb;BEGIN
 res:=banking.get_collection_taxonomy_v2(actor,w);
 SELECT coalesce(jsonb_agg(jsonb_build_object('entity_id',e.entity_id,'book_id',b.book_id,'name',left(b.name,200),'entity_name',left(e.name,200))ORDER BY e.entity_id,b.book_id),'[]'::jsonb)
 INTO books_value FROM ledger.book b JOIN ledger.entity e USING(entity_id)
 WHERE EXISTS(SELECT FROM banking.collection_visible_sources(w)s WHERE s.origin='legacy'AND s.legacy_entity_id=b.entity_id AND s.legacy_book_id=b.book_id)
 AND banking.collection_label_book_read(actor,b.entity_id,b.book_id);
 WITH ids AS MATERIALIZED(SELECT DISTINCT (r.body->'money'->>'currency_version_id')::uuid id FROM banking.collection_query_rows(actor,w)r WHERE r.body->'money'->>'state'='available')
 SELECT coalesce(jsonb_agg(jsonb_build_object('currency_version_id',c.currency_version_id,'code',c.alphabetic_code,'numeric_code',c.numeric_code,'minor_unit',c.minor_unit,'valid_from',to_char(c.valid_from,'YYYY-MM-DD'),'valid_to',to_char(c.valid_to,'YYYY-MM-DD'))ORDER BY c.alphabetic_code COLLATE "C",c.valid_from,c.currency_version_id),'[]'::jsonb)
 INTO currencies_value FROM ids JOIN reference.currency c ON c.currency_version_id=ids.id;
 IF jsonb_array_length(books_value)>100 OR jsonb_array_length(currencies_value)>1000 THEN PERFORM banking.collection_error('P2805');END IF;
 res:=res||jsonb_build_object('profile','private-collection-catalog-v3','books',books_value,'currencies',currencies_value,'institution',jsonb_build_object('state','unavailable'),'legal_owner',jsonb_build_object('state','unknown'));
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;

-- Decimal bounds are original-unit magnitudes; reject malformed/overprecise values before casting.
CREATE FUNCTION banking.collection_query_amount(v jsonb,units integer)RETURNS numeric
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $$DECLARE txt text;n numeric;BEGIN
 IF v='null'::jsonb THEN RETURN NULL;END IF;txt:=ledger.text_value(v);
 IF units IS NULL OR units NOT BETWEEN 0 AND 9 OR length(txt)>39 OR txt COLLATE "C" !~ '^(0|[1-9][0-9]{0,28})([.][0-9]{1,9})?$'THEN PERFORM banking.collection_error('P2800');END IF;
 n:=txt::numeric;IF n>=1e29 OR scale(trim_scale(n))>units THEN PERFORM banking.collection_error('P2800');END IF;RETURN n;
END $$;

-- All closed selectors apply before count/rank/limit; no page-local filter or mixed-currency valuation.
CREATE FUNCTION banking.query_collection_review_v3(actor uuid,w uuid,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;sel jsonb;snap jsonb;sid uuid;first_day date;last_day date;search_value text;assignment_value text;
 topic_kind text;tid uuid;tag_kind text;tagid uuid;merchant_kind text;merchant_value text;institution_kind text;owner_kind text;
 amount_currency uuid;amount_min numeric;amount_max numeric;amount_direction text;units integer;
 book_kind text;eid uuid;bid uuid;source_status text;journal_status text;sort_field text;sort_order text;
 cur jsonb;txt text;page_offset bigint:=0;res jsonb;total bigint;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);v:=banking.import_json(raw,8192);
 PERFORM banking.collection_keys(v,ARRAY['profile','source_account_id','assignment','starts_on','through_date','search','topic','tag','merchant','amount','institution','legal_owner','book','status','sort','snapshot','cursor','limit']);
 IF ledger.text_value(v->'profile')<>'private-collection-review-v3'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_typeof(v->'limit')IS DISTINCT FROM 'number'OR v->>'limit'!~ '^[1-9][0-9]{0,2}$'OR(v->>'limit')::integer>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'source_account_id'<>'null'::jsonb THEN sid:=ledger.uuid_value(v->'source_account_id');IF NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id=sid)THEN PERFORM banking.collection_error('P2801');END IF;END IF;
 assignment_value:=ledger.text_value(v->'assignment');search_value:=ledger.text_value(v->'search');
 IF assignment_value<>ALL(ARRAY['all','assigned','unassigned'])OR length(search_value)>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'starts_on'<>'null'::jsonb THEN first_day:=ledger.date_value(ledger.text_value(v->'starts_on'));END IF;
 IF v->'through_date'<>'null'::jsonb THEN last_day:=ledger.date_value(ledger.text_value(v->'through_date'));END IF;
 IF first_day>last_day THEN PERFORM banking.collection_error('P2800');END IF;
 topic_kind:=ledger.text_value(v->'topic'->'kind');
 IF topic_kind='topic'THEN PERFORM banking.collection_keys(v->'topic',ARRAY['kind','topic_id']);tid:=ledger.uuid_value(v->'topic'->'topic_id');
  IF NOT EXISTS(SELECT FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'topic_id'=tid::text)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'topic',ARRAY['kind']);IF topic_kind<>ALL(ARRAY['all','unclassified','uncategorized'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 tag_kind:=ledger.text_value(v->'tag'->'kind');
 IF tag_kind='tag'THEN PERFORM banking.collection_keys(v->'tag',ARRAY['kind','tag_id']);tagid:=ledger.uuid_value(v->'tag'->'tag_id');
  IF NOT EXISTS(SELECT FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision),jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND t->>'tag_id'=tagid::text)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'tag',ARRAY['kind']);IF tag_kind<>ALL(ARRAY['all','untagged'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 merchant_kind:=ledger.text_value(v->'merchant'->'kind');
 IF merchant_kind IN('exact','contains')THEN PERFORM banking.collection_keys(v->'merchant',ARRAY['kind','value']);merchant_value:=ledger.text_value(v->'merchant'->'value');IF length(merchant_value)NOT BETWEEN 1 AND 100 THEN PERFORM banking.collection_error('P2800');END IF;
 ELSE PERFORM banking.collection_keys(v->'merchant',ARRAY['kind']);IF merchant_kind<>ALL(ARRAY['all','missing'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 PERFORM banking.collection_keys(v->'institution',ARRAY['kind']);institution_kind:=ledger.text_value(v->'institution'->'kind');IF institution_kind<>ALL(ARRAY['all','unavailable'])THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_keys(v->'legal_owner',ARRAY['kind']);owner_kind:=ledger.text_value(v->'legal_owner'->'kind');IF owner_kind<>ALL(ARRAY['all','unknown'])THEN PERFORM banking.collection_error('P2800');END IF;
 book_kind:=ledger.text_value(v->'book'->'kind');
 IF book_kind='book'THEN PERFORM banking.collection_keys(v->'book',ARRAY['kind','entity_id','book_id']);eid:=ledger.uuid_value(v->'book'->'entity_id');bid:=ledger.uuid_value(v->'book'->'book_id');
  IF NOT banking.collection_label_book_read(actor,eid,bid)OR NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)s WHERE s.origin='legacy'AND s.legacy_entity_id=eid AND s.legacy_book_id=bid)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'book',ARRAY['kind']);IF book_kind<>ALL(ARRAY['all','unassigned','unavailable'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 PERFORM banking.collection_keys(v->'status',ARRAY['source','journal']);source_status:=ledger.text_value(v->'status'->'source');journal_status:=ledger.text_value(v->'status'->'journal');
 IF source_status<>ALL(ARRAY['all','booked','pending','removed','held'])OR journal_status<>ALL(ARRAY['all','unposted','posted','correction_required','unavailable'])THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'amount'<>'null'::jsonb THEN
  PERFORM banking.collection_keys(v->'amount',ARRAY['currency_version_id','minimum','maximum','direction']);amount_currency:=ledger.uuid_value(v->'amount'->'currency_version_id');amount_direction:=ledger.text_value(v->'amount'->'direction');
  IF amount_direction<>ALL(ARRAY['all','inflow','outflow','zero'])THEN PERFORM banking.collection_error('P2800');END IF;
  SELECT c.minor_unit INTO units FROM reference.currency c WHERE c.currency_version_id=amount_currency;
  IF NOT FOUND OR units IS NULL THEN PERFORM banking.collection_error('P2801');END IF;
  amount_min:=banking.collection_query_amount(v->'amount'->'minimum',units);amount_max:=banking.collection_query_amount(v->'amount'->'maximum',units);
  IF amount_min>amount_max THEN PERFORM banking.collection_error('P2800');END IF;
 END IF;
 PERFORM banking.collection_keys(v->'sort',ARRAY['field','order']);sort_field:=ledger.text_value(v->'sort'->'field');sort_order:=ledger.text_value(v->'sort'->'order');
 IF sort_field<>ALL(ARRAY['date','account','description','merchant','amount','currency','book','source_state','journal_state','topic','tags','institution','legal_owner'])OR sort_order<>ALL(ARRAY['asc','desc'])THEN PERFORM banking.collection_error('P2800');END IF;
 snap:=banking.collection_query_snapshot(w);IF v->'snapshot'<>'null'::jsonb THEN PERFORM banking.collection_query_pins(v->'snapshot',snap);END IF;sel:=v-ARRAY['profile','snapshot','cursor','limit'];
 IF v->'cursor'<>'null'::jsonb THEN
  txt:=ledger.text_value(v->'cursor');IF length(txt)NOT BETWEEN 2 AND 4096 OR txt COLLATE "C" !~ '^([0-9a-f]{2})+$'THEN PERFORM banking.collection_error('P2800');END IF;
  cur:=banking.import_json(convert_from(decode(txt,'hex'),'UTF8'),2048);PERFORM banking.collection_keys(cur,ARRAY['workspace_id','snapshot','selector','offset']);
  IF ledger.uuid_value(cur->'workspace_id')<>w OR cur->'selector'<>sel THEN PERFORM banking.collection_error('P2800');END IF;
  PERFORM banking.collection_query_pins(cur->'snapshot',snap);page_offset:=banking.import_revision(cur->'offset');IF page_offset=0 THEN PERFORM banking.collection_error('P2800');END IF;
 END IF;
 WITH facts AS MATERIALIZED(SELECT r.*,s.origin,s.legacy_entity_id,s.legacy_book_id FROM banking.collection_query_rows(actor,w)r JOIN banking.collection_source s ON s.workspace_id=w AND s.source_account_id=r.source_id),
 matched AS MATERIALIZED(SELECT f.* FROM facts f WHERE(sid IS NULL OR f.source_id=sid)AND(first_day IS NULL OR f.booked_date>=first_day)AND(last_day IS NULL OR f.booked_date<=last_day)
 AND strpos(lower(f.body->>'description'COLLATE "C"),lower(search_value COLLATE "C"))>0
 AND(assignment_value='all'OR f.body->'assignment'->>'state'=assignment_value)
 AND(topic_kind='all'OR topic_kind='unclassified'AND f.body->'owner_topic'->>'state'IN('unclassified','stale')OR topic_kind='uncategorized'AND f.body->'owner_topic'->>'state'='uncategorized'OR topic_kind='topic'AND f.body->'owner_topic'->>'state'='topic'AND f.body->'owner_topic'->>'topic_id'=tid::text)
 AND(tag_kind='all'OR tag_kind='untagged'AND(f.body->'owner_tags'->>'state'<>'current'OR jsonb_array_length(f.body->'owner_tags'->'tags')=0)OR tag_kind='tag'AND f.body->'owner_tags'->>'state'='current'AND EXISTS(SELECT FROM jsonb_array_elements(f.body->'owner_tags'->'tags')t WHERE t->>'tag_id'=tagid::text))
 AND(merchant_kind='all'OR merchant_kind='missing'AND(f.body->>'merchant'IS NULL OR f.body->>'merchant'='')OR merchant_kind='exact'AND lower(f.body->>'merchant'COLLATE "C")=lower(merchant_value COLLATE "C")OR merchant_kind='contains'AND strpos(lower(f.body->>'merchant'COLLATE "C"),lower(merchant_value COLLATE "C"))>0)
 AND(book_kind='all'OR book_kind='unassigned'AND f.body->'assignment'->>'state'='unassigned'OR book_kind='unavailable'AND f.body->'assignment'->>'state'='unavailable'OR book_kind='book'AND f.body->'assignment'->>'state'='assigned'AND f.legacy_entity_id=eid AND f.legacy_book_id=bid)
 AND(source_status='all'OR f.body->>'source_state'=source_status)AND(journal_status='all'OR f.body->>'journal_state'=journal_status)
 AND(amount_currency IS NULL OR f.body->'money'->>'state'='available'AND f.body->'money'->>'currency_version_id'=amount_currency::text AND(amount_direction='all'OR f.body->'money'->>'direction'=amount_direction)AND(amount_min IS NULL OR(f.body->'money'->>'amount')::numeric>=amount_min)AND(amount_max IS NULL OR(f.body->'money'->>'amount')::numeric<=amount_max))),
 keyed AS MATERIALIZED(SELECT f.*,
 CASE sort_field WHEN 'date'THEN to_char(booked_date,'YYYY-MM-DD')WHEN 'account'THEN body->>'account_name'WHEN 'description'THEN body->>'description'WHEN 'merchant'THEN nullif(body->>'merchant','')WHEN 'amount'THEN body->'money'->>'currency'WHEN 'currency'THEN body->'money'->>'currency'WHEN 'book'THEN body->'assignment'->'books'->0->>'name'WHEN 'source_state'THEN body->>'source_state'WHEN 'journal_state'THEN body->>'journal_state'WHEN 'topic'THEN CASE WHEN body->'owner_topic'->>'state'='topic'THEN body->'owner_topic'->>'name'WHEN body->'owner_topic'->>'state'='uncategorized'THEN 'Uncategorized'END WHEN 'tags'THEN(SELECT string_agg(t->>'name',E'\n'ORDER BY(t->>'name')COLLATE "C")FROM jsonb_array_elements(body->'owner_tags'->'tags')t)END text_key,
 CASE WHEN sort_field IN('amount','currency')THEN body->'money'->>'currency_version_id'END currency_key,
 CASE WHEN sort_field='amount'AND body->'money'->>'state'='available'THEN(body->'money'->>'amount')::numeric END amount_key FROM matched f),
 numbered AS MATERIALIZED(SELECT f.*,row_number()OVER(ORDER BY CASE WHEN sort_order='asc'THEN text_key END COLLATE "C" ASC NULLS LAST,CASE WHEN sort_order='desc'THEN text_key END COLLATE "C" DESC NULLS LAST,CASE WHEN sort_order='asc'THEN currency_key END ASC NULLS LAST,CASE WHEN sort_order='desc'THEN currency_key END DESC NULLS LAST,CASE WHEN sort_order='asc'THEN amount_key END ASC NULLS LAST,CASE WHEN sort_order='desc'THEN amount_key END DESC NULLS LAST,source_id,observation_id)rn FROM keyed f),
 picked AS MATERIALIZED(SELECT * FROM numbered WHERE rn>page_offset ORDER BY rn LIMIT(v->>'limit')::integer)
 SELECT jsonb_build_object('profile','private-collection-review-v3','workspace_id',w,'selector',sel,'snapshot',snap,'matching_count',(SELECT count(*)::text FROM matched),'start_index',CASE WHEN EXISTS(SELECT FROM picked)THEN(page_offset+1)::text ELSE '0'END,'items',coalesce((SELECT jsonb_agg(body ORDER BY rn)FROM picked),'[]'),
 'next_cursor',CASE WHEN(SELECT max(rn)FROM picked)<(SELECT count(*)FROM matched)THEN encode(convert_to(jsonb_build_object('workspace_id',w,'snapshot',snap,'selector',sel,'offset',(SELECT max(rn)::text FROM picked))::text,'UTF8'),'hex')END,'partial',EXISTS(SELECT FROM banking.collection_visible_sources(w)coverage WHERE(sid IS NULL OR coverage.source_account_id=sid)AND(coverage.origin='legacy'OR coverage.readiness<>'HISTORICAL_UPDATE_COMPLETE'))),(SELECT count(*)FROM matched)INTO res,total;
 IF page_offset>0 AND page_offset>=total THEN PERFORM banking.collection_error('P2800');END IF;
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.collection_error('P2800');END $$;

-- Return only current source detail and independently authorized existing journal anchors.
CREATE FUNCTION banking.get_collection_detail(actor uuid,w uuid,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;snap jsonb;sid uuid;oid uuid;rev bigint;s banking.collection_source;row_value jsonb;links_value jsonb;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);v:=banking.import_json(raw,8192);
 PERFORM banking.collection_keys(v,ARRAY['profile','snapshot','source_id','observation_id','observation_revision']);IF ledger.text_value(v->'profile')<>'private-collection-detail-v1'THEN PERFORM banking.collection_error('P2800');END IF;
 sid:=ledger.uuid_value(v->'source_id');oid:=ledger.uuid_value(v->'observation_id');rev:=banking.import_revision(v->'observation_revision');IF rev=0 THEN PERFORM banking.collection_error('P2800');END IF;
 snap:=banking.collection_query_snapshot(w);PERFORM banking.collection_query_pins(v->'snapshot',snap);
 SELECT * INTO s FROM banking.collection_visible_sources(w)WHERE source_account_id=sid;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT r.body INTO row_value FROM banking.collection_query_rows(actor,w)r WHERE r.source_id=sid AND r.observation_id=oid;
 IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;IF(row_value->>'observation_revision')::bigint<>rev THEN PERFORM banking.collection_error('P2803');END IF;
 IF s.origin='native'THEN links_value:=jsonb_build_object('state','unassigned','journals','[]'::jsonb);
 ELSIF NOT banking.collection_label_book_read(actor,s.legacy_entity_id,s.legacy_book_id)THEN links_value:=jsonb_build_object('state','unavailable','journals','[]'::jsonb);
 ELSE
  SELECT jsonb_build_object('state','available','journals',coalesce(jsonb_agg(jsonb_build_object('entity_id',p.entity_id,'book_id',p.book_id,'journal_id',p.journal_id,'source_revision',ev.linked_revision::text)ORDER BY p.entity_id,p.book_id,p.journal_id),'[]'::jsonb))INTO links_value
  FROM banking.review_assignment a JOIN banking.review_item i ON i.entity_id=a.entity_id AND i.book_id=a.book_id AND i.review_id=a.review_id
  JOIN banking.review_posting p ON p.entity_id=i.entity_id AND p.book_id=i.book_id AND p.review_id=i.review_id
  JOIN ledger.journal_entry j ON j.entity_id=p.entity_id AND j.book_id=p.book_id AND j.journal_id=p.journal_id
  JOIN banking.review_evidence ev ON ev.entity_id=a.entity_id AND ev.book_id=a.book_id AND ev.decision_id=a.decision_id AND ev.observation_id=a.observation_id
  WHERE a.entity_id=s.legacy_entity_id AND a.book_id=s.legacy_book_id AND a.observation_id=oid AND i.state='posted';
 END IF;
 IF jsonb_array_length(links_value->'journals')>100 THEN PERFORM banking.collection_error('P2805');END IF;
 res:=jsonb_build_object('profile','private-collection-detail-v1','workspace_id',w,'snapshot',snap,'row',row_value,'links',links_value);
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.collection_error('P2800');END $$;

REVOKE ALL ON FUNCTION banking.collection_query_snapshot(uuid),banking.collection_query_pins(jsonb,jsonb),banking.collection_query_rows(uuid,uuid),banking.collection_query_amount(jsonb,integer),banking.get_collection_taxonomy_v3(uuid,uuid),banking.query_collection_review_v3(uuid,uuid,text),banking.get_collection_detail(uuid,uuid,text)FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION banking.get_collection_taxonomy_v3(uuid,uuid),banking.query_collection_review_v3(uuid,uuid,text),banking.get_collection_detail(uuid,uuid,text)TO bookkeeping_api;
COMMIT;
