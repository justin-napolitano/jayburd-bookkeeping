-- Issue72: restricted exact report reads; reviewed contract docs/contracts/reporting-sql.md.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Convert computed amounts only after bounding every result and preserving exact scale.
CREATE FUNCTION reporting.report_amounts(v jsonb) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,ledger,pg_temp AS $$
DECLARE result jsonb:='{}'; r record; n numeric;
BEGIN
 FOR r IN SELECT * FROM jsonb_each_text(v) LOOP
  n:=r.value::numeric;
  IF n IS NULL OR abs(n)>=1e29 OR scale(trim_scale(n))>9 THEN PERFORM ledger.fail('P2110'); END IF;
  result:=result||jsonb_build_object(r.key,trim_scale(n)::text);
 END LOOP;
 RETURN result;
END $$;
-- Compare exact amounts with identical keys; no percentages or floating arithmetic.
CREATE FUNCTION reporting.report_difference(a jsonb,b jsonb) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,reporting,pg_temp AS $$
DECLARE result jsonb:='{}'; r record;
BEGIN
 FOR r IN SELECT * FROM jsonb_each_text(a) LOOP
  result:=result||jsonb_build_object(r.key,r.value::numeric-(b->>r.key)::numeric);
 END LOOP;
 RETURN reporting.report_amounts(result);
END $$;
-- Owner-only aggregation helper: the wrapper holds authorization and book locks.
CREATE FUNCTION reporting.report_data(e uuid,b uuid,k text,lo date,hi date,aid uuid,rev bigint) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,ledger,reference,reporting,pg_temp AS $$
DECLARE rows jsonb:='[]'; totals jsonb; r record; amounts jsonb; selector jsonb;
 dr numeric:=0; cr numeric:=0; assets numeric:=0; liabilities numeric:=0; equity numeric:=0;
 revenue numeric:=0; expense numeric:=0; opening numeric:=0; running numeric:=0; net numeric;
BEGIN
 -- This foundation explicitly caps complete materialization; larger scopes fail
 -- instead of pretending a page limit bounds the aggregate's intermediate rows.
 IF k='general-ledger' THEN
  IF (SELECT count(*) FROM (SELECT 1 FROM ledger.journal_line l JOIN ledger.journal_entry h USING(entity_id,book_id,journal_id)
   WHERE l.entity_id=e AND l.book_id=b AND l.account_id=aid AND h.accounting_date BETWEEN lo AND hi LIMIT 4097) q)>4096 THEN PERFORM ledger.fail('P2110'); END IF;
 ELSE
  IF (SELECT count(*) FROM (SELECT 1 FROM ledger.account a WHERE a.entity_id=e AND a.book_id=b LIMIT 4097) q)>4096 THEN PERFORM ledger.fail('P2110'); END IF;
  IF EXISTS(SELECT FROM ledger.account a WHERE a.entity_id=e AND a.book_id=b AND (octet_length(a.code)>1024 OR octet_length(a.name)>1024)) THEN PERFORM ledger.fail('P2110'); END IF;
 END IF;
 selector:=jsonb_build_object('entity_id',e,'book_id',b,'snapshot_revision',rev::text);
 IF k IN ('trial-balance','balance-sheet') THEN selector:=selector||jsonb_build_object('through_date',hi::text);
 ELSE selector:=selector||jsonb_build_object('starts_on',lo::text,'ends_before',(hi+1)::text); END IF;
 IF k='general-ledger' THEN
  IF NOT EXISTS(SELECT FROM ledger.account a WHERE a.entity_id=e AND a.book_id=b AND a.account_id=aid) THEN PERFORM ledger.fail('P2101'); END IF;
  SELECT coalesce(sum(CASE l.side WHEN 'debit' THEN l.functional_amount ELSE -l.functional_amount END),0) INTO opening
  FROM ledger.journal_line l JOIN ledger.journal_entry h USING(entity_id,book_id,journal_id)
  WHERE l.entity_id=e AND l.book_id=b AND l.account_id=aid AND h.accounting_date<lo;
  running:=opening;
  FOR r IN SELECT h.*,l.line_number,l.side,l.transaction_currency_version_id,l.transaction_amount,l.functional_amount,l.valuation_basis,l.rate_observation_id,c.alphabetic_code
   FROM ledger.journal_line l JOIN ledger.journal_entry h USING(entity_id,book_id,journal_id)
   JOIN reference.currency c ON c.currency_version_id=l.transaction_currency_version_id
   WHERE l.entity_id=e AND l.book_id=b AND l.account_id=aid AND h.accounting_date BETWEEN lo AND hi
   ORDER BY h.accounting_date,h.journal_id,l.line_number LOOP
   IF r.side='debit' THEN dr:=dr+r.functional_amount; running:=running+r.functional_amount;
   ELSE cr:=cr+r.functional_amount; running:=running-r.functional_amount; END IF;
   amounts:=reporting.report_amounts(jsonb_build_object('debit',CASE r.side WHEN 'debit' THEN r.functional_amount ELSE 0 END,'credit',CASE r.side WHEN 'credit' THEN r.functional_amount ELSE 0 END,'running_balance',running));
   rows:=rows||jsonb_build_array(jsonb_build_object('id','line:'||r.journal_id::text||':'||r.line_number::text,
    '_key',jsonb_build_array(r.accounting_date::text,r.journal_id::text,lpad(r.line_number::text,4,'0')),
    'amounts',amounts,'selector',selector||jsonb_build_object('account_id',aid),
    'journal_id',r.journal_id,'line_number',r.line_number,'account_id',aid,'accounting_date',r.accounting_date::text,
    'posted_at',to_char(r.posted_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'kind',r.kind,'reverses_journal_id',r.reverses_journal_id,'policy_id',r.policy_id,'side',r.side,
    'transaction_currency',r.alphabetic_code,'transaction_currency_version_id',r.transaction_currency_version_id,
    'transaction_amount',trim_scale(r.transaction_amount)::text,'functional_amount',trim_scale(r.functional_amount)::text,
    'valuation_basis',r.valuation_basis,'rate_observation_id',r.rate_observation_id));
  END LOOP;
  totals:=reporting.report_amounts(jsonb_build_object('opening',opening,'period_debits',dr,'period_credits',cr,'closing',running));
 ELSE
  FOR r IN SELECT a.account_id,a.code,a.name,a.category,a.normal_side,
    coalesce(sum(CASE l.side WHEN 'debit' THEN l.functional_amount ELSE -l.functional_amount END) FILTER(WHERE h.journal_id IS NOT NULL),0) n
   FROM ledger.account a LEFT JOIN ledger.journal_line l USING(entity_id,book_id,account_id)
   LEFT JOIN ledger.journal_entry h ON h.entity_id=l.entity_id AND h.book_id=l.book_id AND h.journal_id=l.journal_id
    AND h.accounting_date<=hi AND (k<>'profit-and-loss' OR h.accounting_date>=lo)
   WHERE a.entity_id=e AND a.book_id=b GROUP BY a.account_id,a.code,a.name,a.category,a.normal_side ORDER BY a.code COLLATE "C",a.account_id LOOP
   net:=r.n; dr:=dr+greatest(net,0); cr:=cr+greatest(-net,0);
   CASE r.category WHEN 'asset' THEN assets:=assets+net; WHEN 'liability' THEN liabilities:=liabilities-net;
    WHEN 'equity' THEN equity:=equity-net; WHEN 'revenue' THEN revenue:=revenue-net; WHEN 'expense' THEN expense:=expense+net; END CASE;
   IF k='trial-balance' THEN amounts:=reporting.report_amounts(jsonb_build_object('net',net,'debit_balance',greatest(net,0),'credit_balance',greatest(-net,0)));
   ELSIF (k='profit-and-loss' AND r.category IN ('revenue','expense')) OR (k='balance-sheet' AND r.category IN ('asset','liability','equity')) THEN
    amounts:=reporting.report_amounts(jsonb_build_object('balance',CASE WHEN r.category IN ('liability','equity','revenue') THEN -net ELSE net END));
   ELSE CONTINUE; END IF;
   rows:=rows||jsonb_build_array(jsonb_build_object('id','account:'||r.account_id::text,'_key',jsonb_build_array(r.code,r.account_id::text),
    'account_id',r.account_id,'account_code',r.code,'account_name',r.name,'category',r.category,'normal_side',r.normal_side,
    'amounts',amounts,'selector',selector||jsonb_build_object('account_id',r.account_id)));
  END LOOP;
  IF k='trial-balance' THEN
   IF dr<>cr THEN PERFORM ledger.fail('P2108'); END IF;
   totals:=reporting.report_amounts(jsonb_build_object('total_debits',dr,'total_credits',cr));
  ELSIF k='profit-and-loss' THEN totals:=reporting.report_amounts(jsonb_build_object('revenue',revenue,'expense',expense,'net_income',revenue-expense));
  ELSE
   IF assets<>liabilities+equity+revenue-expense THEN PERFORM ledger.fail('P2108'); END IF;
   totals:=reporting.report_amounts(jsonb_build_object('assets',assets,'liabilities',liabilities,'direct_equity',equity,'unclosed_earnings',revenue-expense,'total_equity',equity+revenue-expense));
   rows:=rows||jsonb_build_array(jsonb_build_object('id','unclosed_earnings','_key',jsonb_build_array('~unclosed_earnings',''),
    'amounts',reporting.report_amounts(jsonb_build_object('balance',revenue-expense)),
    'selector',selector||jsonb_build_object('categories',jsonb_build_array('revenue','expense'))));
  END IF;
 END IF;
 RETURN jsonb_build_object('rows',rows,'totals',totals);
END $$;
-- Public restricted report boundary validates raw JSON, current scope and pagination.
CREATE FUNCTION reporting.financial_report(actor uuid,e uuid,b uuid,raw text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,ledger,reference,reporting,pg_temp AS $$
DECLARE v jsonb; k text; rev bigint; expected text; lo date; hi date; clo date; chi date; aid uuid;
 allowed text[]; required text[]; lim integer:=100; filters jsonb; fh text; cursor_value jsonb; last_key text[];
 data jsonb; prior jsonb; row_value jsonb; prior_row jsonb; page jsonb:='[]'; next_value jsonb:='null'; count_rows integer:=0;
 item record; result jsonb; coverage_lo date; coverage_hi date; covered date; coverage_done boolean:=false;
 p record; curr reference.currency; curr_id uuid; policies jsonb:='[]'; cmp jsonb:='null';
BEGIN
 rev:=ledger.authorize(actor,e,b,'read',false);
 IF raw IS NULL OR octet_length(raw)>8192 OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN v:=raw::jsonb; EXCEPTION WHEN untranslatable_character OR invalid_text_representation THEN PERFORM ledger.fail('P2100'); END;
 k:=ledger.text_value(v->'report');
 required:=ARRAY['report']; allowed:=ARRAY['snapshot_revision','limit','cursor'];
 IF k IN ('trial-balance','balance-sheet') THEN
  required:=required||ARRAY['through_date']; allowed:=allowed||ARRAY['compare_through_date'];
  hi:=ledger.date_value(ledger.text_value(v->'through_date')); lo:=hi;
  IF v?'compare_through_date' THEN chi:=ledger.date_value(ledger.text_value(v->'compare_through_date')); clo:=chi; END IF;
 ELSIF k IN ('general-ledger','profit-and-loss') THEN
  required:=required||ARRAY['starts_on','ends_before'];
  lo:=ledger.date_value(ledger.text_value(v->'starts_on')); hi:=ledger.date_value(ledger.text_value(v->'ends_before'))-1;
  IF hi<lo OR hi-lo>=366 THEN PERFORM ledger.fail('P2100'); END IF;
  IF k='general-ledger' THEN required:=required||ARRAY['account_id']; aid:=ledger.uuid_value(v->'account_id'); v:=jsonb_set(v,'{account_id}',to_jsonb(aid::text));
  ELSE
   allowed:=allowed||ARRAY['compare_starts_on','compare_ends_before'];
   IF (v?'compare_starts_on')<>(v?'compare_ends_before') THEN PERFORM ledger.fail('P2100'); END IF;
   IF v?'compare_starts_on' THEN clo:=ledger.date_value(ledger.text_value(v->'compare_starts_on')); chi:=ledger.date_value(ledger.text_value(v->'compare_ends_before'))-1;
    IF chi<clo OR chi-clo>=366 THEN PERFORM ledger.fail('P2100'); END IF;
   END IF;
  END IF;
 ELSE PERFORM ledger.fail('P2100'); END IF;
 PERFORM ledger.keys(v,required,allowed);
 IF v?'snapshot_revision' THEN expected:=ledger.text_value(v->'snapshot_revision');
  IF expected!~'^(0|[1-9][0-9]{0,18})$' OR expected::numeric>9223372036854775807 THEN PERFORM ledger.fail('P2100'); END IF;
  IF expected::bigint<>rev THEN PERFORM ledger.fail('P2114'); END IF;
 END IF;
 IF v?'limit' THEN
  IF jsonb_typeof(v->'limit')<>'number' OR (v->>'limit')!~'^[1-9][0-9]{0,2}$' THEN PERFORM ledger.fail('P2100'); END IF;
  lim:=(v->>'limit')::integer; IF lim>500 THEN PERFORM ledger.fail('P2100'); END IF;
 END IF;
 filters:=v-ARRAY['cursor','limit','snapshot_revision']; fh:=encode(sha256(convert_to(jsonb_build_object('contract_version','financial-reporting-sql-v1','entity_id',e,'book_id',b,'filters',filters)::text,'UTF8')),'hex');
 IF v?'cursor' THEN
  IF NOT(v?'snapshot_revision') THEN PERFORM ledger.fail('P2100'); END IF;
  cursor_value:=v->'cursor'; PERFORM ledger.keys(cursor_value,ARRAY['revision','filter_hash','last_key']);
  IF ledger.text_value(cursor_value->'revision')<>rev::text OR ledger.text_value(cursor_value->'filter_hash')<>fh OR jsonb_typeof(cursor_value->'last_key')<>'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(cursor_value->'last_key')<>(CASE k WHEN 'general-ledger' THEN 3 ELSE 2 END) THEN PERFORM ledger.fail('P2100'); END IF;
  IF EXISTS(SELECT FROM jsonb_array_elements(cursor_value->'last_key') x WHERE jsonb_typeof(x)<>'string') THEN PERFORM ledger.fail('P2100'); END IF;
  last_key:=ARRAY(SELECT jsonb_array_elements_text(cursor_value->'last_key'));
 END IF;
 coverage_hi:=greatest(hi,chi); coverage_lo:=least(lo,clo);
 SELECT least(coverage_lo,min(h.accounting_date)) INTO coverage_lo FROM ledger.journal_entry h WHERE h.entity_id=e AND h.book_id=b AND h.accounting_date<=coverage_hi;
 IF (SELECT count(*) FROM (SELECT 1 FROM ledger.book_policy bp WHERE bp.entity_id=e AND bp.book_id=b AND bp.effective_from<=coverage_hi AND (bp.effective_to IS NULL OR bp.effective_to>coverage_lo) LIMIT 1025) q)>1024 THEN PERFORM ledger.fail('P2110'); END IF;
 covered:=coverage_lo;
 FOR p IN SELECT * FROM ledger.book_policy bp WHERE bp.entity_id=e AND bp.book_id=b AND bp.effective_from<=coverage_hi AND (bp.effective_to IS NULL OR bp.effective_to>coverage_lo) ORDER BY bp.effective_from LOOP
  IF p.effective_from>covered OR p.basis<>'accrual' OR p.timezone<>'UTC' OR p.posting_scale<>2 OR p.rounding_mode<>'half_even' OR p.rate_policy_code<>'synthetic-reviewed-spot-v1' THEN PERFORM ledger.fail('P2109'); END IF;
  IF curr_id IS NOT NULL AND curr_id<>p.functional_currency_version_id THEN PERFORM ledger.fail('P2109'); END IF;
  curr_id:=p.functional_currency_version_id; policies:=policies||jsonb_build_array(p.policy_id);
  IF p.effective_to IS NULL OR p.effective_to>coverage_hi THEN coverage_done:=true; EXIT; END IF;
  covered:=p.effective_to;
 END LOOP;
 IF NOT coverage_done THEN PERFORM ledger.fail('P2109'); END IF;
 SELECT * INTO curr FROM reference.currency WHERE currency_version_id=curr_id;
 IF curr.alphabetic_code IS DISTINCT FROM 'USD' OR curr.minor_unit IS DISTINCT FROM 2 OR curr.valid_from>coverage_lo OR (curr.valid_to IS NOT NULL AND curr.valid_to<=coverage_hi) THEN PERFORM ledger.fail('P2109'); END IF;
 data:=reporting.report_data(e,b,k,lo,hi,aid,rev);
 IF chi IS NOT NULL THEN
  prior:=reporting.report_data(e,b,k,clo,chi,aid,rev);
  cmp:=jsonb_build_object('totals',prior->'totals','difference',reporting.report_difference(data->'totals',prior->'totals'));
 END IF;
 IF last_key IS NOT NULL AND NOT EXISTS(SELECT FROM jsonb_array_elements(data->'rows') x WHERE ARRAY(SELECT jsonb_array_elements_text(x->'_key'))=last_key) THEN PERFORM ledger.fail('P2100'); END IF;
 FOR item IN SELECT x,ARRAY(SELECT jsonb_array_elements_text(x->'_key')) sort_key FROM jsonb_array_elements(data->'rows') x
  ORDER BY ARRAY(SELECT jsonb_array_elements_text(x->'_key')) COLLATE "C" LOOP
  IF last_key IS NOT NULL AND item.sort_key COLLATE "C"<=last_key COLLATE "C" THEN CONTINUE; END IF;
  IF count_rows=lim THEN
   next_value:=jsonb_build_object('revision',rev::text,'filter_hash',fh,'last_key',page->(count_rows-1)->'_key'); EXIT;
  END IF;
  row_value:=item.x;
  IF prior IS NOT NULL THEN
   SELECT x INTO prior_row FROM jsonb_array_elements(prior->'rows') x WHERE x->>'id'=row_value->>'id';
   row_value:=row_value||jsonb_build_object('comparison',jsonb_build_object('amounts',prior_row->'amounts','difference',reporting.report_difference(row_value->'amounts',prior_row->'amounts')));
  END IF;
  page:=page||jsonb_build_array(row_value); count_rows:=count_rows+1;
 END LOOP;
 SELECT coalesce(jsonb_agg(x-'_key'),'[]'::jsonb) INTO page FROM jsonb_array_elements(page) x;
 result:=jsonb_build_object('contract_version','financial-reporting-sql-v1','report',k,'entity_id',e,'book_id',b,'snapshot_revision',rev::text,
  'generated_at',to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'basis','accrual','timezone','UTC',
  'functional_currency',curr.alphabetic_code,'functional_currency_version_id',curr_id,'minor_units',curr.minor_unit,'policy_ids',policies,
  'filters',filters,'totals',data->'totals','comparison',cmp,'rows',page,'next_cursor',next_value);
 IF octet_length(result::text)>2097152 THEN PERFORM ledger.fail('P2110'); END IF;
 RETURN result;
END $$;
REVOKE ALL ON FUNCTION reporting.report_amounts(jsonb),reporting.report_difference(jsonb,jsonb),reporting.report_data(uuid,uuid,text,date,date,uuid,bigint),reporting.financial_report(uuid,uuid,uuid,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION reporting.financial_report(uuid,uuid,uuid,text) TO bookkeeping_api;
COMMIT;
