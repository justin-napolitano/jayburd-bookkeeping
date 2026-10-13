-- Bind an actual GL line and immutable journal to one authorized report snapshot.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE FUNCTION reporting.report_journal(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,ledger,reporting,pg_temp AS $$
DECLARE v jsonb; rev bigint; expected text; aid uuid; jid uuid; lo date; hi date; ln integer; result jsonb;
BEGIN
 rev:=ledger.authorize(actor,e,b,'read',false);
 IF raw IS NULL OR octet_length(raw)>8192 OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN v:=raw::jsonb; EXCEPTION WHEN untranslatable_character OR invalid_text_representation THEN PERFORM ledger.fail('P2100'); END;
 PERFORM ledger.keys(v,ARRAY['snapshot_revision','account_id','journal_id','starts_on','ends_before','line_number']);
 expected:=ledger.text_value(v->'snapshot_revision');
 IF expected!~'^(0|[1-9][0-9]{0,18})$' OR expected::numeric>9223372036854775807 THEN PERFORM ledger.fail('P2100'); END IF;
 aid:=ledger.uuid_value(v->'account_id'); jid:=ledger.uuid_value(v->'journal_id');
 lo:=ledger.date_value(ledger.text_value(v->'starts_on')); hi:=ledger.date_value(ledger.text_value(v->'ends_before'));
 IF hi<=lo OR hi-lo>366 THEN PERFORM ledger.fail('P2100'); END IF;
 IF jsonb_typeof(v->'line_number')<>'number' OR (v->>'line_number')!~'^[1-9][0-9]{0,3}$' THEN PERFORM ledger.fail('P2100'); END IF;
 ln:=(v->>'line_number')::integer; IF ln>1000 THEN PERFORM ledger.fail('P2100'); END IF;
 IF expected::bigint<>rev THEN PERFORM ledger.fail('P2114'); END IF;
 IF NOT EXISTS(SELECT FROM ledger.journal_line l JOIN ledger.journal_entry h USING(entity_id,book_id,journal_id)
  WHERE l.entity_id=e AND l.book_id=b AND l.account_id=aid AND l.journal_id=jid AND l.line_number=ln
  AND h.accounting_date>=lo AND h.accounting_date<hi) THEN PERFORM ledger.fail('P2101'); END IF;
 result:=jsonb_build_object('contract_version','report-journal-v1','entity_id',e,'book_id',b,'snapshot_revision',rev::text,
  'selector',jsonb_build_object('account_id',aid,'journal_id',jid,'starts_on',lo::text,'ends_before',hi::text,'line_number',ln),
  'journal',reporting.get_journal(actor,e,b,jid));
 IF octet_length(result::text)>2097152 THEN PERFORM ledger.fail('P2110'); END IF;
 RETURN result;
END $$;
REVOKE ALL ON FUNCTION reporting.report_journal(uuid,uuid,uuid,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION reporting.report_journal(uuid,uuid,uuid,text) TO bookkeeping_api;
COMMIT;
