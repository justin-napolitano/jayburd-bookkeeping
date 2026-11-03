-- Read ordering only: restore the exact migration12 legacy list; no stored state removed.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE OR REPLACE FUNCTION banking.list_review_items(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; cur jsonb; sid uuid; s banking.source_account; filter_value text; page_limit integer; last_id uuid; cursor_text text; r banking.review_item; rows_value jsonb:='[]'; n integer:=0; next_value text;
BEGIN
 v:=banking.import_json(raw,4096);
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
DROP FUNCTION banking.sorted_review_page(uuid,uuid,uuid,text);
DROP INDEX banking.review_assignment_scope_review;
COMMIT;
