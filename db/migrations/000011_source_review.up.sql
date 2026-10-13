-- Synthetic category review and explicit source-linked posting; no provider/FX automation.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
ALTER TABLE banking.review_item DROP CONSTRAINT review_item_review_revision_check;
ALTER TABLE banking.review_item ADD CONSTRAINT review_item_review_revision_check CHECK(review_revision>0);
ALTER TABLE banking.review_item DROP CONSTRAINT review_item_state_check;
ALTER TABLE banking.review_item ADD CONSTRAINT review_item_state_check CHECK(state IN ('needs_review','overlap_hold','draft_ready','posted'));
ALTER TABLE banking.review_decision DROP CONSTRAINT review_decision_expected_review_revision_check;
ALTER TABLE banking.review_decision ADD CONSTRAINT review_decision_expected_review_revision_check CHECK(expected_review_revision>=0);
ALTER TABLE banking.review_decision DROP CONSTRAINT review_decision_kind_check;
ALTER TABLE banking.review_decision ADD CONSTRAINT review_decision_kind_check CHECK(kind IN ('initialize','draft'));
ALTER TABLE banking.review_decision DROP CONSTRAINT review_decision_reason_check;
ALTER TABLE banking.review_decision ADD CONSTRAINT review_decision_reason_check CHECK(reason IN ('synthetic_import','explicit_category'));
ALTER TABLE integration.source_registry_receipt DROP CONSTRAINT source_registry_receipt_operation_check;
ALTER TABLE integration.source_registry_receipt ADD CONSTRAINT source_registry_receipt_operation_check CHECK(operation IN ('register','namespace','state','permission','import_permission','review_permission'));
CREATE TABLE banking.review_draft (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, draft_id uuid NOT NULL DEFAULT gen_random_uuid(), review_id uuid NOT NULL,
 review_revision bigint NOT NULL CHECK(review_revision>0), source_revision bigint NOT NULL CHECK(source_revision>=0), binding_revision bigint NOT NULL CHECK(binding_revision>0), book_revision bigint NOT NULL CHECK(book_revision>=0),
 posting_body jsonb NOT NULL CHECK(octet_length(posting_body::text)<=1048576), normalized_request jsonb NOT NULL CHECK(octet_length(normalized_request::text)<=8192),
 created_by uuid NOT NULL REFERENCES ledger.principal, idempotency_key text NOT NULL CHECK(idempotency_key COLLATE "C" ~ '^[A-Za-z0-9._:-]{1,128}$'), created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,draft_id), UNIQUE(entity_id,book_id,review_id,review_revision), UNIQUE(entity_id,book_id,created_by,idempotency_key), UNIQUE(entity_id,book_id,review_id,draft_id), FOREIGN KEY(entity_id,book_id,review_id) REFERENCES banking.review_item
);
CREATE TABLE banking.review_posting (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, review_id uuid NOT NULL, draft_id uuid NOT NULL, command_id uuid NOT NULL REFERENCES ledger.interactive_command,
 journal_id uuid NOT NULL, result jsonb NOT NULL CHECK(octet_length(result::text)<=1048576), posted_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,review_id), UNIQUE(entity_id,book_id,draft_id), UNIQUE(command_id), UNIQUE(entity_id,book_id,journal_id),
 FOREIGN KEY(entity_id,book_id,review_id,draft_id) REFERENCES banking.review_draft(entity_id,book_id,review_id,draft_id), FOREIGN KEY(entity_id,book_id,journal_id) REFERENCES ledger.journal_entry
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.review_draft FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.review_posting FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Only monotonic unposted-to-draft-to-posted projection changes; evidence is unchanged.
CREATE FUNCTION banking.review_error(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$ BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='review_rejected'; END $$;
CREATE FUNCTION banking.review_guard() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF TG_OP='DELETE' THEN PERFORM banking.review_error('P2502'); END IF;
 IF (to_jsonb(NEW)-ARRAY['state','review_revision']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','review_revision']) OR NEW.review_revision::numeric<>OLD.review_revision::numeric+1
  OR NOT (OLD.state IN ('needs_review','draft_ready') AND NEW.state='draft_ready' OR OLD.state='draft_ready' AND NEW.state='posted') THEN PERFORM banking.review_error('P2502'); END IF;
 RETURN NEW;
END $$;
DROP TRIGGER immutable ON banking.review_item;
CREATE TRIGGER preserve_review BEFORE UPDATE OR DELETE ON banking.review_item FOR EACH ROW EXECUTE FUNCTION banking.review_guard();
-- Explicit operator attestation grants only source review, never entity prepare/post.
CREATE FUNCTION banking.set_source_review_permission(actor uuid,e uuid,b uuid,sid uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE v jsonb; subject uuid; expected bigint; enabled boolean; s banking.source_account; receipt integration.source_registry_receipt; norm jsonb; result_value jsonb;
BEGIN
 IF current_user<>'bookkeeping_owner' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='review_rejected'; END IF;
 v:=banking.import_json(raw,8192); PERFORM ledger.keys(v,ARRAY['expected_binding_revision','principal_id','enabled']); subject:=ledger.uuid_value(v->'principal_id'); expected:=banking.import_revision(v->'expected_binding_revision');
 IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' OR jsonb_typeof(v->'enabled')<>'boolean' THEN PERFORM banking.review_error('P2500'); END IF; enabled:=(v->>'enabled')::boolean;
 PERFORM banking.source_operator_authorize(actor,subject,e,b,enabled);
 norm:=jsonb_build_object('approved_by',actor,'source_account_id',sid,'body',v);
 SELECT * INTO receipt FROM integration.source_registry_receipt WHERE entity_id=e AND book_id=b AND operator_login=session_user AND operation='review_permission' AND idempotency_key=k;
 IF FOUND THEN IF receipt.normalized_request<>norm THEN PERFORM banking.review_error('P2502'); END IF; RETURN receipt.result; END IF;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=sid FOR UPDATE;
 IF NOT FOUND THEN PERFORM banking.review_error('P2501'); END IF;
 IF s.binding_revision<>expected OR expected=9223372036854775807 THEN PERFORM banking.review_error('P2503'); END IF;
 INSERT INTO banking.source_permission VALUES(e,b,sid,subject,'review',CASE WHEN enabled THEN NULL ELSE clock_timestamp() END) ON CONFLICT(entity_id,book_id,source_account_id,principal_id,capability) DO UPDATE SET revoked_at=EXCLUDED.revoked_at;
 UPDATE banking.source_account SET binding_revision=binding_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=sid RETURNING * INTO s;
 result_value:=jsonb_build_object('source_account_id',sid,'binding_revision',s.binding_revision::text);
 INSERT INTO integration.source_registry_receipt(entity_id,book_id,operator_login,operation,idempotency_key,approved_by,normalized_request,result) VALUES(e,b,session_user,'review_permission',k,actor,norm,result_value);
 RETURN result_value;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM banking.review_error('P2500'); END $$;
-- Current read/prepare/review and, for source_post, post permissions precede any replay.
CREATE FUNCTION banking.review_authorize(actor uuid,e uuid,b uuid,id uuid,posting boolean) RETURNS banking.source_account LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE r banking.review_item; s banking.source_account;
BEGIN
 PERFORM ledger.authorize(actor,e,b,'read',true);
 PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='prepare' AND revoked_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM banking.review_error('P2501'); END IF;
 IF posting THEN PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission='post' AND revoked_at IS NULL FOR SHARE; IF NOT FOUND THEN PERFORM banking.review_error('P2501'); END IF; END IF;
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=id;
 s:=banking.import_authorize(actor,e,b,r.source_account_id,false);
 PERFORM 1 FROM banking.source_permission WHERE entity_id=e AND book_id=b AND source_account_id=s.source_account_id AND principal_id=actor AND capability='review' AND revoked_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM banking.review_error('P2501'); END IF;
 RETURN s;
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' THEN PERFORM banking.review_error('P2500'); WHEN SQLSTATE 'P2101' OR SQLSTATE 'P2102' OR SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501'); END $$;
-- First-profile groups contain exactly one immutable observation; future matching needs a new contract.
CREATE FUNCTION banking.review_fact(e uuid,b uuid,id uuid) RETURNS banking.observation_revision LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE q banking.observation_revision;
BEGIN
 BEGIN SELECT v.* INTO STRICT q FROM banking.review_assignment a JOIN banking.source_observation o USING(entity_id,book_id,observation_id) JOIN banking.observation_revision v ON v.entity_id=o.entity_id AND v.book_id=o.book_id AND v.observation_id=o.observation_id AND v.revision=o.current_revision WHERE a.entity_id=e AND a.book_id=b AND a.review_id=id;
 EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM banking.review_error('P2502'); END;
 RETURN q;
END $$;
-- Server renders immutable draft pins and exact human-reviewable posting lines.
CREATE FUNCTION banking.render_review_draft(d banking.review_draft) RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT CASE WHEN d.draft_id IS NULL THEN NULL ELSE jsonb_build_object('draft_id',d.draft_id,'review_id',d.review_id,'review_revision',d.review_revision::text,'source_revision',d.source_revision::text,'binding_revision',d.binding_revision::text,'book_revision',d.book_revision::text,'posting_body',d.posting_body) END
$$;
CREATE FUNCTION banking.render_review_summary(r banking.review_item) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE q banking.observation_revision; code text; did uuid; jid uuid;
BEGIN
 q:=banking.review_fact(r.entity_id,r.book_id,r.review_id); SELECT alphabetic_code INTO code FROM reference.currency WHERE currency_version_id=q.currency_version_id;
 SELECT draft_id INTO did FROM banking.review_draft WHERE entity_id=r.entity_id AND book_id=r.book_id AND review_id=r.review_id ORDER BY review_revision DESC LIMIT 1;
 SELECT journal_id INTO jid FROM banking.review_posting WHERE entity_id=r.entity_id AND book_id=r.book_id AND review_id=r.review_id;
 RETURN jsonb_build_object('review_id',r.review_id,'state',r.state,'review_revision',r.review_revision::text,'booked_date',to_char(q.booked_date,'YYYY-MM-DD'),'currency',code,'currency_version_id',q.currency_version_id,'direction',q.direction,'amount',trim_scale(q.amount)::text,'description',q.description,'draft_id',did,'journal_id',jid);
END $$;
-- Source selector discloses only explicitly readable sources in the selected book.
CREATE FUNCTION banking.list_sources(actor uuid,e uuid,b uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE id uuid; s banking.source_account; rows_value jsonb:='[]'; row_value jsonb; n integer:=0;
BEGIN
 PERFORM ledger.authorize(actor,e,b,'read',false);
 FOR id IN SELECT x.source_account_id FROM banking.source_account x JOIN banking.source_permission p USING(entity_id,book_id,source_account_id) WHERE x.entity_id=e AND x.book_id=b AND p.principal_id=actor AND p.capability='read' AND p.revoked_at IS NULL ORDER BY x.source_account_id LIMIT 101 LOOP
  n:=n+1; IF n>100 THEN PERFORM banking.review_error('P2505'); END IF;
  s:=banking.import_authorize(actor,e,b,id,false);
  SELECT jsonb_build_object('source_account_id',id,'display_name',a.name,'currency',c.alphabetic_code,'currency_version_id',c.currency_version_id,'ledger_account_id',a.account_id,'state',s.state,'binding_revision',s.binding_revision::text,'source_revision',s.source_revision::text) INTO row_value FROM ledger.account a JOIN reference.currency c ON c.currency_version_id=s.currency_version_id WHERE a.entity_id=e AND a.book_id=b AND a.account_id=s.ledger_account_id;
  rows_value:=rows_value||jsonb_build_array(row_value);
 END LOOP;
 RETURN jsonb_build_object('profile','source-list-v1','items',rows_value);
EXCEPTION WHEN SQLSTATE 'P2101' OR SQLSTATE 'P2102' OR SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501'); END $$;
-- Cursor binds full scope, filter and source snapshot; edits invalidate subsequent pages.
CREATE FUNCTION banking.list_review_items(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; cur jsonb; sid uuid; s banking.source_account; filter_value text; page_limit integer; last_id uuid; cursor_text text; r banking.review_item; rows_value jsonb:='[]'; n integer:=0; next_value text;
BEGIN
 v:=banking.import_json(raw,4096);
 IF NOT(v ?& ARRAY['source_account_id','state','limit','cursor']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['source_account_id','state','limit','cursor'])) THEN PERFORM banking.review_error('P2500'); END IF;
 sid:=ledger.uuid_value(v->'source_account_id');
 IF v->'state'<>'null'::jsonb THEN filter_value:=ledger.text_value(v->'state'); IF filter_value NOT IN ('needs_review','overlap_hold','draft_ready','posted') THEN PERFORM banking.review_error('P2500'); END IF; END IF;
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
-- Detail exposes current pins plus immutable original source evidence and latest draft.
CREATE OR REPLACE FUNCTION banking.get_review_item(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE r banking.review_item; s banking.source_account; q banking.observation_revision; d banking.review_draft; o banking.source_observation; rev bigint; blockers jsonb:='[]';
BEGIN
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=id;
 s:=banking.import_authorize(actor,e,b,r.source_account_id,false); q:=banking.review_fact(e,b,id);
 SELECT * INTO o FROM banking.source_observation WHERE entity_id=e AND book_id=b AND observation_id=q.observation_id;
 SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b;
 SELECT * INTO d FROM banking.review_draft WHERE entity_id=e AND book_id=b AND review_id=id ORDER BY review_revision DESC LIMIT 1;
 IF r.state='overlap_hold' THEN blockers:=jsonb_build_array('overlap_review_required'); END IF;
 IF r.state='draft_ready' AND ROW(d.source_revision,d.binding_revision,d.book_revision,d.review_revision) IS DISTINCT FROM ROW(s.source_revision,s.binding_revision,rev,r.review_revision) THEN blockers:=blockers||jsonb_build_array('draft_stale'); END IF;
 RETURN banking.render_review_summary(r)||jsonb_build_object('profile','source-review-v1','source_account_id',r.source_account_id,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'book_revision',rev::text,'ledger_account_id',s.ledger_account_id,'created_at',r.created_at,'blockers',blockers,'draft',banking.render_review_draft(d),
  'evidence',jsonb_build_array(jsonb_build_object('observation_id',o.observation_id,'observation_revision',q.revision::text,'namespace_id',o.namespace_id,'external_entry_id',o.external_entry_id,'source_state',q.source_state,'booked_date',to_char(q.booked_date,'YYYY-MM-DD'),'currency_version_id',q.currency_version_id,'direction',q.direction,'amount',trim_scale(q.amount)::text,'description',q.description)));
END $$;
-- Category candidates are scoped active ledger accounts; FX is an explicit unmet gate.
CREATE FUNCTION banking.review_categories(actor uuid,e uuid,b uuid,id uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE r banking.review_item; s banking.source_account; q banking.observation_revision; p ledger.book_policy; rev bigint; source_code text; functional_code text; rows_value jsonb;
BEGIN
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=id;
 s:=banking.import_authorize(actor,e,b,r.source_account_id,false); q:=banking.review_fact(e,b,id);
 BEGIN SELECT * INTO STRICT p FROM ledger.book_policy WHERE entity_id=e AND book_id=b AND effective_from<=q.booked_date AND (effective_to IS NULL OR effective_to>q.booked_date); EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM ledger.fail('P2109'); END;
 SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b;
 SELECT alphabetic_code INTO source_code FROM reference.currency WHERE currency_version_id=q.currency_version_id;
 SELECT alphabetic_code INTO functional_code FROM reference.currency WHERE currency_version_id=p.functional_currency_version_id;
 IF (SELECT count(*) FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id<>s.ledger_account_id AND active_from<=q.booked_date AND (active_to IS NULL OR active_to>q.booked_date))>100 THEN PERFORM banking.review_error('P2505'); END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('account_id',account_id,'code',code,'name',name,'category',category,'normal_side',normal_side,'monetary',monetary) ORDER BY code,account_id),'[]'::jsonb) INTO rows_value FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id<>s.ledger_account_id AND active_from<=q.booked_date AND (active_to IS NULL OR active_to>q.booked_date);
 RETURN jsonb_build_object('profile','source-review-categories-v1','review_id',id,'source_account_id',s.source_account_id,'book_revision',rev::text,'policy_id',p.policy_id,'accounting_date',to_char(q.booked_date,'YYYY-MM-DD'),'source_currency',source_code,'functional_currency',functional_code,'fx_required',q.currency_version_id<>p.functional_currency_version_id,'accounts',rows_value);
EXCEPTION WHEN SQLSTATE 'P2401' THEN PERFORM banking.review_error('P2501'); END $$;
-- Categorization stores an immutable two-line identity draft; this function cannot post.
CREATE FUNCTION banking.create_review_draft(actor uuid,e uuid,b uuid,id uuid,k text,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; r banking.review_item; s banking.source_account; q banking.observation_revision; d banking.review_draft; p ledger.book_policy; a ledger.account; rev bigint; category_id uuid; day_value date; description_value text; code text; amount_value text; body jsonb; norm jsonb;
BEGIN
 v:=banking.import_json(raw,8192); PERFORM ledger.keys(v,ARRAY['expected_review_revision','source_revision','binding_revision','expected_book_revision','category_account_id','policy_id','accounting_date','description']);
 IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' THEN PERFORM banking.review_error('P2500'); END IF;
 PERFORM banking.import_revision(v->'expected_review_revision'); PERFORM banking.import_revision(v->'source_revision'); PERFORM banking.import_revision(v->'binding_revision'); PERFORM banking.import_revision(v->'expected_book_revision');
 category_id:=ledger.uuid_value(v->'category_account_id'); PERFORM ledger.uuid_value(v->'policy_id'); day_value:=ledger.date_value(ledger.text_value(v->'accounting_date')); description_value:=ledger.text_value(v->'description');
 IF length(description_value) NOT BETWEEN 1 AND 500 THEN PERFORM banking.review_error('P2500'); END IF;
 s:=banking.review_authorize(actor,e,b,id,false);
 norm:=jsonb_build_object('review_id',id,'body',v);
 SELECT * INTO d FROM banking.review_draft WHERE entity_id=e AND book_id=b AND created_by=actor AND idempotency_key=k;
 IF FOUND THEN IF d.normalized_request<>norm THEN PERFORM banking.review_error('P2502'); END IF; RETURN banking.render_review_draft(d); END IF;
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=id FOR UPDATE;
 SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b;
 IF r.state NOT IN ('needs_review','draft_ready') OR s.state<>'active' THEN PERFORM banking.review_error('P2502'); END IF;
 IF ROW(r.review_revision,s.source_revision,s.binding_revision,rev) IS DISTINCT FROM ROW((v->>'expected_review_revision')::bigint,(v->>'source_revision')::bigint,(v->>'binding_revision')::bigint,(v->>'expected_book_revision')::bigint) THEN PERFORM banking.review_error('P2503'); END IF;
 IF s.source_revision=9223372036854775807 OR r.review_revision=9223372036854775807 THEN PERFORM banking.review_error('P2505'); END IF;
 q:=banking.review_fact(e,b,id);
 IF q.booked_date<>day_value OR q.source_state<>'booked' THEN PERFORM banking.review_error('P2500'); END IF;
 BEGIN SELECT * INTO STRICT p FROM ledger.book_policy WHERE entity_id=e AND book_id=b AND effective_from<=day_value AND (effective_to IS NULL OR effective_to>day_value); EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM ledger.fail('P2109'); END;
 IF p.policy_id<>(v->>'policy_id')::uuid THEN PERFORM banking.review_error('P2503'); END IF;
 IF q.currency_version_id<>p.functional_currency_version_id THEN PERFORM banking.review_error('P2504'); END IF;
 SELECT alphabetic_code INTO code FROM reference.currency WHERE currency_version_id=q.currency_version_id;
 -- Match the existing reviewed posting policy; do not claim a new accounting framework.
 IF p.basis<>'accrual' OR p.timezone<>'UTC' OR p.posting_scale<>2 OR p.rounding_mode<>'half_even' OR p.rate_policy_code<>'synthetic-reviewed-spot-v1' OR code<>'USD' THEN PERFORM ledger.fail('P2109'); END IF;
 PERFORM 1 FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id IN (s.ledger_account_id,category_id) ORDER BY account_id FOR SHARE;
 IF category_id=s.ledger_account_id THEN PERFORM banking.review_error('P2500'); END IF;
 SELECT * INTO a FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id=category_id AND active_from<=day_value AND (active_to IS NULL OR active_to>day_value);
 IF NOT FOUND OR NOT EXISTS(SELECT FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id=s.ledger_account_id AND active_from<=day_value AND (active_to IS NULL OR active_to>day_value)) THEN PERFORM banking.review_error('P2500'); END IF;
 amount_value:=trim_scale(q.amount)::text;
 body:=jsonb_build_object('expected_revision',rev::text,'policy_id',p.policy_id,'accounting_date',to_char(day_value,'YYYY-MM-DD'),'description',description_value,'lines',jsonb_build_array(
  jsonb_build_object('account_id',s.ledger_account_id,'side',CASE q.direction WHEN 'inflow' THEN 'debit' ELSE 'credit' END,'transaction_currency',code,'transaction_amount',amount_value,'functional_amount',amount_value,'valuation_basis','identity'),
  jsonb_build_object('account_id',category_id,'side',CASE q.direction WHEN 'inflow' THEN 'credit' ELSE 'debit' END,'transaction_currency',code,'transaction_amount',amount_value,'functional_amount',amount_value,'valuation_basis','identity')));
 PERFORM ledger.validate_command_body('post',body);
 UPDATE banking.source_account SET source_revision=source_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=s.source_account_id RETURNING * INTO s;
 UPDATE banking.review_item SET state='draft_ready',review_revision=review_revision+1 WHERE entity_id=e AND book_id=b AND review_id=id RETURNING * INTO r;
 INSERT INTO banking.review_draft(entity_id,book_id,review_id,review_revision,source_revision,binding_revision,book_revision,posting_body,normalized_request,created_by,idempotency_key)
  VALUES(e,b,id,r.review_revision,s.source_revision,s.binding_revision,rev,body,norm,actor,k) RETURNING * INTO d;
 INSERT INTO banking.review_decision(entity_id,book_id,review_id,expected_review_revision,kind,actor_id,reason,body) VALUES(e,b,id,r.review_revision-1,'draft',actor,'explicit_category',jsonb_build_object('draft_id',d.draft_id,'category_account_id',category_id,'source_revision',s.source_revision::text));
 RETURN banking.render_review_draft(d);
EXCEPTION WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR numeric_value_out_of_range THEN PERFORM banking.review_error('P2500'); WHEN unique_violation THEN RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_review_draft'; END $$;
-- Recoverable intent must reproduce the immutable reviewed draft; never trust replacement lines.
CREATE FUNCTION banking.source_post_authorize(actor uuid,e uuid,b uuid,body jsonb) RETURNS banking.review_draft LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE d banking.review_draft;
BEGIN
 PERFORM banking.review_authorize(actor,e,b,(body->>'review_id')::uuid,true);
 SELECT * INTO d FROM banking.review_draft WHERE entity_id=e AND book_id=b AND review_id=(body->>'review_id')::uuid AND draft_id=(body->>'draft_id')::uuid;
 IF NOT FOUND THEN PERFORM banking.review_error('P2501'); END IF;
 IF ROW(d.review_revision,d.source_revision,d.binding_revision,d.book_revision,d.posting_body) IS DISTINCT FROM ROW((body->>'review_revision')::bigint,(body->>'source_revision')::bigint,(body->>'binding_revision')::bigint,(body->>'expected_book_revision')::bigint,body->'posting_body') THEN PERFORM banking.review_error('P2502'); END IF;
 RETURN d;
END $$;
-- The journal, link, source projection and interactive receipt commit in one transaction.
CREATE FUNCTION banking.execute_source_post(actor uuid,e uuid,b uuid,command uuid,request uuid,body jsonb) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE d banking.review_draft; r banking.review_item; s banking.source_account; link banking.review_posting; rev bigint; posted record; result_value jsonb;
BEGIN
 d:=banking.source_post_authorize(actor,e,b,body);
 SELECT * INTO link FROM banking.review_posting WHERE entity_id=e AND book_id=b AND review_id=d.review_id;
 IF FOUND THEN IF link.draft_id<>d.draft_id THEN PERFORM banking.review_error('P2502'); END IF; RETURN link.result; END IF;
 SELECT * INTO r FROM banking.review_item WHERE entity_id=e AND book_id=b AND review_id=d.review_id FOR UPDATE;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=e AND book_id=b AND source_account_id=r.source_account_id;
 SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b;
 IF r.state<>'draft_ready' OR s.state<>'active' THEN PERFORM banking.review_error('P2502'); END IF;
 IF ROW(r.review_revision,s.source_revision,s.binding_revision,rev) IS DISTINCT FROM ROW(d.review_revision,d.source_revision,d.binding_revision,d.book_revision) THEN PERFORM banking.review_error('P2503'); END IF;
 IF r.review_revision=9223372036854775807 OR s.source_revision=9223372036854775807 THEN PERFORM banking.review_error('P2505'); END IF;
 SELECT * INTO posted FROM ledger.post_journal(actor,e,b,'source-command:'||command,request,d.posting_body::text);
 UPDATE banking.source_account SET source_revision=source_revision+1 WHERE entity_id=e AND book_id=b AND source_account_id=s.source_account_id RETURNING * INTO s;
 UPDATE banking.review_item SET state='posted',review_revision=review_revision+1 WHERE entity_id=e AND book_id=b AND review_id=r.review_id RETURNING * INTO r;
 result_value:=jsonb_build_object('entity_id',e,'book_id',b,'review_id',r.review_id,'draft_id',d.draft_id,'journal_id',posted.body->>'journal_id','book_revision',posted.body->>'book_revision','journal',posted.body,'review_revision',r.review_revision::text,'source_revision',s.source_revision::text);
 INSERT INTO banking.review_posting(entity_id,book_id,review_id,draft_id,command_id,journal_id,result) VALUES(e,b,r.review_id,d.draft_id,command,(posted.body->>'journal_id')::uuid,result_value);
 RETURN result_value;
END $$;
-- Existing setup/post/import recoverability is preserved; source_post shares the same principal gate.
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_operation_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_operation_check CHECK(operation IN ('setup','post','import','source_post'));
ALTER TABLE ledger.interactive_command DROP CONSTRAINT interactive_command_check;
ALTER TABLE ledger.interactive_command ADD CONSTRAINT interactive_command_check CHECK((operation='setup' AND entity_id IS NULL AND book_id IS NULL) OR (operation IN ('post','import','source_post') AND entity_id IS NOT NULL AND book_id IS NOT NULL));
CREATE OR REPLACE FUNCTION ledger.validate_command_body(op text,v jsonb) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE a jsonb; code_value text; basis_value text; first_day date; last_day date;
BEGIN
 IF op='setup' THEN
  PERFORM ledger.keys(v,ARRAY['entity_name','entity_kind','book_name','starts_on','ends_before','functional_currency_version_id','accounts']);
  PERFORM ledger.setup_text(v->'entity_name',200); PERFORM ledger.setup_text(v->'book_name',200);
  IF ledger.text_value(v->'entity_kind') NOT IN ('legal','personal') OR jsonb_typeof(v->'accounts') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(v->'accounts') NOT BETWEEN 2 AND 100 THEN PERFORM ledger.fail('P2100'); END IF;
  first_day:=ledger.date_value(ledger.text_value(v->'starts_on')); last_day:=ledger.date_value(ledger.text_value(v->'ends_before'));
  IF first_day>=last_day THEN PERFORM ledger.fail('P2100'); END IF;
  PERFORM ledger.uuid_value(v->'functional_currency_version_id');
  FOR a IN SELECT value FROM jsonb_array_elements(v->'accounts') LOOP
   PERFORM ledger.keys(a,ARRAY['code','name','category','normal_side','monetary']);
   PERFORM ledger.setup_text(a->'code',64); PERFORM ledger.setup_text(a->'name',200);
   IF ledger.text_value(a->'category') NOT IN ('asset','liability','equity','revenue','expense') OR ledger.text_value(a->'normal_side') NOT IN ('debit','credit') OR jsonb_typeof(a->'monetary') IS DISTINCT FROM 'boolean' THEN PERFORM ledger.fail('P2100'); END IF;
  END LOOP;
  IF (SELECT count(DISTINCT value->>'code') FROM jsonb_array_elements(v->'accounts'))<>jsonb_array_length(v->'accounts') THEN PERFORM ledger.fail('P2100'); END IF;
 ELSIF op='post' THEN
  PERFORM ledger.keys(v,ARRAY['expected_revision','policy_id','accounting_date','description','lines']);
  code_value:=ledger.text_value(v->'expected_revision');
  IF code_value !~ '^(0|[1-9][0-9]{0,18})$' OR code_value::numeric>9223372036854775807 THEN PERFORM ledger.fail('P2100'); END IF;
  PERFORM ledger.uuid_value(v->'policy_id'); PERFORM ledger.date_value(ledger.text_value(v->'accounting_date'));
  code_value:=ledger.text_value(v->'description'); IF length(code_value) NOT BETWEEN 1 AND 500 OR jsonb_typeof(v->'lines') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(v->'lines') NOT BETWEEN 2 AND 1000 THEN PERFORM ledger.fail('P2108'); END IF;
  FOR a IN SELECT value FROM jsonb_array_elements(v->'lines') LOOP
   PERFORM ledger.keys(a,ARRAY['account_id','side','transaction_currency','transaction_amount','functional_amount','valuation_basis'],ARRAY['rate_observation_id']);
   PERFORM ledger.uuid_value(a->'account_id'); PERFORM ledger.money_value(a->'transaction_amount'); PERFORM ledger.money_value(a->'functional_amount');
   basis_value:=ledger.text_value(a->'valuation_basis');
   IF ledger.text_value(a->'side') NOT IN ('debit','credit') OR basis_value NOT IN ('identity','spot') OR ledger.text_value(a->'transaction_currency') !~ '^[A-Z]{3}$' THEN PERFORM ledger.fail('P2108'); END IF;
   IF (basis_value='identity' AND a?'rate_observation_id') OR (basis_value='spot' AND NOT a?'rate_observation_id') THEN PERFORM ledger.fail('P2108'); END IF;
   IF basis_value='spot' THEN PERFORM ledger.uuid_value(a->'rate_observation_id'); END IF;
  END LOOP;
 ELSIF op='import' THEN
  PERFORM ledger.keys(v,ARRAY['batch_id','preview_id','preview_revision','binding_revision','source_revision','profile']);
  PERFORM ledger.uuid_value(v->'batch_id'); PERFORM ledger.uuid_value(v->'preview_id');
  IF banking.import_revision(v->'preview_revision')<1 OR banking.import_revision(v->'binding_revision')<1 OR banking.import_revision(v->'source_revision')<0
   OR ledger.text_value(v->'profile')<>'jayburd-synthetic-statement-csv/1' THEN PERFORM ledger.fail('P2100'); END IF;
 ELSIF op='source_post' THEN
  PERFORM ledger.keys(v,ARRAY['review_id','draft_id','review_revision','source_revision','binding_revision','expected_book_revision','posting_body']);
  PERFORM ledger.uuid_value(v->'review_id'); PERFORM ledger.uuid_value(v->'draft_id');
  IF banking.import_revision(v->'review_revision')<1 OR banking.import_revision(v->'binding_revision')<1 THEN PERFORM ledger.fail('P2100'); END IF;
  PERFORM banking.import_revision(v->'source_revision'); PERFORM banking.import_revision(v->'expected_book_revision'); PERFORM ledger.validate_command_body('post',v->'posting_body');
 ELSE PERFORM ledger.fail('P2100'); END IF;
END $$;
CREATE OR REPLACE FUNCTION ledger.command_authorize(actor uuid,c ledger.interactive_command) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE e uuid; b uuid; x banking.import_batch;
BEGIN
 IF c.principal_id IS DISTINCT FROM actor THEN PERFORM ledger.fail('P2101'); END IF;
 IF c.operation='source_post' THEN PERFORM banking.source_post_authorize(actor,c.entity_id,c.book_id,c.request_body); RETURN; END IF;
 IF c.operation='import' THEN
  SELECT * INTO x FROM banking.import_batch WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid;
  PERFORM banking.import_authorize(actor,c.entity_id,c.book_id,x.source_account_id,true);
  RETURN;
 END IF;
 IF c.operation='post' THEN e:=c.entity_id; b:=c.book_id;
 ELSIF c.state='committed' THEN e:=c.result_entity_id; b:=c.result_book_id; END IF;
 IF e IS NOT NULL THEN
  PERFORM ledger.authorize(actor,e,b,'read',true);
  PERFORM ledger.authorize(actor,e,b,'post',true);
 END IF;
END $$;
CREATE OR REPLACE FUNCTION ledger.register_command(actor uuid,k text,request uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; op text; e uuid; b uuid; c ledger.interactive_command;
BEGIN
 IF raw IS NULL OR k IS NULL OR request IS NULL THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF length(k) NOT BETWEEN 1 AND 128 OR k !~ '^[A-Za-z0-9._:-]+$' OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN v:=raw::jsonb; EXCEPTION WHEN untranslatable_character OR numeric_value_out_of_range THEN PERFORM ledger.fail('P2100'); END;
 -- Envelope targets deliberately permit JSON null; ordinary ledger.keys forbids it.
 IF NOT(v ?& ARRAY['operation','entity_id','book_id','body']) OR EXISTS(SELECT FROM jsonb_object_keys(v) x WHERE x<>ALL(ARRAY['operation','entity_id','book_id','body'])) THEN PERFORM ledger.fail('P2100'); END IF;
 op:=ledger.text_value(v->'operation');
 IF op='setup' THEN IF v->'entity_id'<>'null'::jsonb OR v->'book_id'<>'null'::jsonb THEN PERFORM ledger.fail('P2100'); END IF;
 ELSIF op IN ('post','import','source_post') THEN e:=ledger.uuid_value(v->'entity_id'); b:=ledger.uuid_value(v->'book_id');
 ELSE PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.validate_command_body(op,v->'body');
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE principal_id=actor AND registration_key=k;
 IF FOUND THEN
  PERFORM ledger.command_authorize(actor,c);
  IF ROW(c.operation,c.entity_id,c.book_id,c.request_body) IS DISTINCT FROM ROW(op,e,b,v->'body') THEN PERFORM ledger.fail('P2104'); END IF;
  RETURN ledger.render_command(c);
 END IF;
 -- Inaccessible pending work still blocks new intent, without exposing its payload.
 IF EXISTS(SELECT FROM ledger.interactive_command WHERE principal_id=actor AND acknowledged_at IS NULL) THEN PERFORM ledger.fail('P2112'); END IF;
 c.principal_id:=actor; c.operation:=op; c.entity_id:=e; c.book_id:=b; c.request_body:=v->'body';
 PERFORM ledger.command_authorize(actor,c);
 INSERT INTO ledger.interactive_command(command_id,principal_id,registration_key,operation,entity_id,book_id,request_body)
 VALUES(gen_random_uuid(),actor,k,op,e,b,v->'body') RETURNING * INTO c;
 INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(c.command_id,actor,'register',request);
 RETURN ledger.render_command(c);
EXCEPTION WHEN unique_violation THEN
 -- A stale serializable snapshot after waiting on the unchanged principal can
 -- discover the competing insert at its unique index; retry with a fresh snapshot.
 RAISE EXCEPTION USING ERRCODE='40001',MESSAGE='retry_command_registration';
END $$;
CREATE OR REPLACE FUNCTION ledger.transition_command(actor uuid,id uuid,request uuid,action_name text,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE c ledger.interactive_command; posted record; value jsonb;
BEGIN
 IF id IS NULL OR request IS NULL OR raw IS NULL OR action_name IS NULL OR action_name NOT IN ('execute','cancel','acknowledge') THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN value:=raw::jsonb; EXCEPTION WHEN untranslatable_character THEN PERFORM ledger.fail('P2100'); END;
 IF value<>'{}'::jsonb THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.command_principal(actor);
 SELECT * INTO c FROM ledger.interactive_command WHERE command_id=id AND principal_id=actor;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 PERFORM ledger.command_authorize(actor,c);
 PERFORM 1 FROM ledger.interactive_command WHERE command_id=id FOR UPDATE;
 IF action_name='execute' THEN
  IF c.state='cancelled' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.state='prepared' THEN
   IF c.operation='source_post' THEN value:=banking.execute_source_post(actor,c.entity_id,c.book_id,c.command_id,request,c.request_body);
   ELSIF c.operation='import' THEN value:=banking.execute_import(actor,c.entity_id,c.book_id,c.request_body);
   ELSE
   IF c.operation='setup' THEN SELECT * INTO posted FROM ledger.setup_book(actor,'command:'||c.command_id,request,c.request_body::text);
   ELSE SELECT * INTO posted FROM ledger.post_journal(actor,c.entity_id,c.book_id,'command:'||c.command_id,request,c.request_body::text); END IF;
   value:=posted.body;
   END IF;
   UPDATE ledger.interactive_command SET state='committed',result=value,result_entity_id=(value->>'entity_id')::uuid,result_book_id=(value->>'book_id')::uuid,terminal_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'execute',request);
  END IF;
 ELSIF action_name='cancel' THEN
  IF c.state='committed' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.state='prepared' THEN
   IF c.operation='import' THEN
    UPDATE banking.import_batch SET state='cancelled' WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid AND state IN ('received','preview_ready') AND (SELECT preview_id FROM banking.import_preview WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid ORDER BY preview_revision DESC LIMIT 1)=(c.request_body->>'preview_id')::uuid;
    IF EXISTS(SELECT FROM banking.import_batch WHERE entity_id=c.entity_id AND book_id=c.book_id AND batch_id=(c.request_body->>'batch_id')::uuid AND state='imported') THEN PERFORM ledger.fail('P2113'); END IF;
   END IF;
   UPDATE ledger.interactive_command SET state='cancelled',terminal_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'cancel',request);
  END IF;
 ELSE
  IF c.state='prepared' THEN PERFORM ledger.fail('P2113'); END IF;
  IF c.acknowledged_at IS NULL THEN
   UPDATE ledger.interactive_command SET acknowledged_at=clock_timestamp() WHERE command_id=id RETURNING * INTO c;
   INSERT INTO audit.command_action(command_id,principal_id,action,request_id) VALUES(id,actor,'acknowledge',request);
  END IF;
 END IF;
 RETURN ledger.render_command(c);
END $$;
REVOKE ALL ON banking.review_draft,banking.review_posting FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher;
REVOKE ALL ON FUNCTION banking.review_error(text),banking.review_guard(),banking.set_source_review_permission(uuid,uuid,uuid,uuid,text,text),banking.review_authorize(uuid,uuid,uuid,uuid,boolean),banking.review_fact(uuid,uuid,uuid),banking.render_review_draft(banking.review_draft),banking.render_review_summary(banking.review_item),banking.source_post_authorize(uuid,uuid,uuid,jsonb),banking.execute_source_post(uuid,uuid,uuid,uuid,uuid,jsonb) FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher;
GRANT EXECUTE ON FUNCTION banking.list_sources(uuid,uuid,uuid),banking.list_review_items(uuid,uuid,uuid,text),banking.review_categories(uuid,uuid,uuid,uuid),banking.create_review_draft(uuid,uuid,uuid,uuid,text,text) TO bookkeeping_api;
COMMIT;
