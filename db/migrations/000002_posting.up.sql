-- Restricted SQL authority for ADR 0005; Go never supplies a trusted total/hash.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Strict text primitives reject coercion, nonfinite input, and unknown shapes.
CREATE FUNCTION ledger.fail(code text) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN RAISE EXCEPTION USING ERRCODE=code,MESSAGE='ledger_operation_rejected'; END $$;
-- Reject unknown fields and nulls before any semantic interpretation.
CREATE FUNCTION ledger.keys(v jsonb, required text[], optional text[] DEFAULT ARRAY[]::text[]) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF jsonb_typeof(v) IS DISTINCT FROM 'object' OR NOT v ?& required THEN PERFORM ledger.fail('P2100'); END IF;
 IF EXISTS(SELECT FROM jsonb_object_keys(v) k WHERE NOT k=ANY(required||optional)) OR EXISTS(SELECT FROM jsonb_each(v) kv WHERE kv.value='null'::jsonb) THEN PERFORM ledger.fail('P2100'); END IF;
END $$;
-- Require JSON string types so numeric coercion cannot hide bad transport input.
CREATE FUNCTION ledger.text_value(v jsonb) RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
BEGIN IF jsonb_typeof(v) IS DISTINCT FROM 'string' THEN PERFORM ledger.fail('P2100'); END IF; RETURN v#>>'{}'; END $$;
-- Normalize only standard hyphenated UUID spellings.
CREATE FUNCTION ledger.uuid_value(v jsonb) RETURNS uuid LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE s text:=ledger.text_value(v);
BEGIN IF s !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN PERFORM ledger.fail('P2100'); END IF; RETURN s::uuid; END $$;
-- Accept real bounded Gregorian dates, not permissive SQL date syntax.
CREATE FUNCTION ledger.date_value(v text) RETURNS date LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE d date;
BEGIN
 IF v IS NULL OR v !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR left(v,4)='0000' THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN d:=v::date; EXCEPTION WHEN datetime_field_overflow OR invalid_datetime_format THEN PERFORM ledger.fail('P2100'); END;
 RETURN d;
END $$;
-- Validate positive decimal text before bounded storage can round it.
CREATE FUNCTION ledger.money_value(v jsonb) RETURNS numeric LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE s text:=ledger.text_value(v); n numeric;
BEGIN
 IF s !~ '^(0|[1-9][0-9]{0,28})(\.[0-9]{1,9})?$' THEN PERFORM ledger.fail('P2107'); END IF;
 n:=s::numeric; IF n<=0 THEN PERFORM ledger.fail('P2107'); END IF; RETURN n;
END $$;
-- Round exact positive conversion products once using the selected tie rule.
CREATE FUNCTION ledger.round_even(v numeric,s integer) RETURNS numeric LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE u numeric:=v*power(10::numeric,s); n numeric:=floor(u); f numeric:=u-n;
BEGIN IF f>0.5 OR (f=0.5 AND mod(n,2)=1) THEN n:=n+1; END IF; RETURN n/power(10::numeric,s); END $$;

-- Permission/disable checks lock the same rows administrative revocations update.
CREATE FUNCTION ledger.authorize(actor uuid,e uuid,b uuid,permission_name text,writing boolean) RETURNS bigint LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE rev bigint;
BEGIN
 IF actor IS NULL OR e IS NULL OR b IS NULL THEN PERFORM ledger.fail('P2100'); END IF;
 IF (writing AND current_setting('transaction_isolation')<>'serializable') OR
 (NOT writing AND current_setting('transaction_isolation') NOT IN ('serializable','repeatable read')) THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 PERFORM 1 FROM ledger.entity WHERE entity_id=e AND disabled_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 PERFORM 1 FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND permission=permission_name AND revoked_at IS NULL FOR SHARE;
 IF NOT FOUND THEN
  IF EXISTS(SELECT FROM ledger.membership WHERE entity_id=e AND principal_id=actor AND revoked_at IS NULL) THEN PERFORM ledger.fail('P2102'); END IF;
  PERFORM ledger.fail('P2101');
 END IF;
 IF writing THEN SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b AND disabled_at IS NULL FOR UPDATE;
 ELSE SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b AND disabled_at IS NULL FOR SHARE; END IF;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 RETURN rev;
END $$;

-- Render immutable facts; only authorized wrappers or posting routines may call.
CREATE FUNCTION ledger.render_journal(e uuid,b uuid,j uuid) RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog AS $$
 SELECT jsonb_build_object('entity_id',h.entity_id,'book_id',h.book_id,'journal_id',h.journal_id,
 'period_id',h.period_id,'policy_id',h.policy_id,'book_revision',h.book_revision::text,
 'functional_currency',c.alphabetic_code,'functional_currency_version_id',c.currency_version_id,
 'accounting_date',to_char(h.accounting_date,'YYYY-MM-DD'),'description',h.description,'kind',h.kind,
 'posted_at',to_char(h.posted_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
 'posted_by',h.posted_by,'reverses_journal_id',h.reverses_journal_id,'reason',h.reason,
 'lines',(SELECT jsonb_agg(jsonb_strip_nulls(jsonb_build_object(
  'line_number',l.line_number,'account_id',l.account_id,'side',l.side,'transaction_currency',lc.alphabetic_code,
  'transaction_currency_version_id',lc.currency_version_id,'transaction_amount',trim_scale(l.transaction_amount)::text,
  'functional_amount',trim_scale(l.functional_amount)::text,'valuation_basis',l.valuation_basis,'rate_observation_id',l.rate_observation_id)) ORDER BY l.line_number)
 FROM ledger.journal_line l JOIN reference.currency lc ON lc.currency_version_id=l.transaction_currency_version_id
 WHERE l.entity_id=e AND l.book_id=b AND l.journal_id=j))
 FROM ledger.journal_entry h JOIN ledger.book_policy p USING(entity_id,book_id,policy_id)
 JOIN reference.currency c ON c.currency_version_id=p.functional_currency_version_id
 WHERE h.entity_id=e AND h.book_id=b AND h.journal_id=j
$$;

-- Common mutation authority: validated replay is resolved before mutable policy checks.
CREATE FUNCTION ledger.mutate(actor uuid,e uuid,b uuid,original uuid,k text,request uuid,raw text,op text)
RETURNS TABLE(http_status smallint,body jsonb,replayed boolean) LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE v jsonb; norm jsonb; l jsonb; ls jsonb:='[]'; stored_lines jsonb:='[]'; result jsonb;
 rev bigint; expected bigint; d date; description_value text; reason_value text; j uuid:=gen_random_uuid();
 policy ledger.book_policy; current_policy ledger.book_policy; period_row ledger.period; old ledger.journal_entry;
 receipt integration.operation_receipt; curr reference.currency; fc reference.currency; rate_row reference.rate_observation;
 aid uuid; cid uuid; rid uuid; amount_value numeric; functional_value numeric; total numeric:=0; n integer:=0;
 policy_uuid uuid; basis_value text; side_value text; code_value text; now_value timestamptz:=clock_timestamp();
BEGIN
 IF raw IS NULL OR k IS NULL OR request IS NULL OR op NOT IN ('post','reverse') OR (op='reverse' AND original IS NULL) THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF length(k) NOT BETWEEN 1 AND 128 OR k !~ '^[A-Za-z0-9._:-]+$' OR NOT (raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN v:=raw::jsonb; EXCEPTION WHEN untranslatable_character THEN PERFORM ledger.fail('P2100'); END;
 IF op='post' THEN PERFORM ledger.keys(v,ARRAY['expected_revision','policy_id','accounting_date','description','lines']);
 ELSE PERFORM ledger.keys(v,ARRAY['expected_revision','accounting_date','reason']); END IF;
 code_value:=ledger.text_value(v->'expected_revision');
 IF code_value !~ '^(0|[1-9][0-9]{0,18})$' OR code_value::numeric>9223372036854775807 THEN PERFORM ledger.fail('P2100'); END IF;
 expected:=code_value::bigint; d:=ledger.date_value(ledger.text_value(v->'accounting_date'));
 norm:=v;
 IF op='post' THEN
  policy_uuid:=ledger.uuid_value(v->'policy_id'); norm:=jsonb_set(norm,'{policy_id}',to_jsonb(policy_uuid));
  description_value:=ledger.text_value(v->'description');
  IF length(description_value) NOT BETWEEN 1 AND 500 OR jsonb_typeof(v->'lines') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
  IF jsonb_array_length(v->'lines') NOT BETWEEN 2 AND 1000 THEN PERFORM ledger.fail('P2108'); END IF;
  FOR l IN SELECT value FROM jsonb_array_elements(v->'lines') LOOP
   PERFORM ledger.keys(l,ARRAY['account_id','side','transaction_currency','transaction_amount','functional_amount','valuation_basis'],ARRAY['rate_observation_id']);
   aid:=ledger.uuid_value(l->'account_id'); side_value:=ledger.text_value(l->'side'); basis_value:=ledger.text_value(l->'valuation_basis'); code_value:=ledger.text_value(l->'transaction_currency');
   IF side_value NOT IN ('debit','credit') OR basis_value NOT IN ('identity','spot') OR code_value !~ '^[A-Z]{3}$' THEN PERFORM ledger.fail('P2108'); END IF;
   IF (basis_value='identity' AND l?'rate_observation_id') OR (basis_value='spot' AND NOT l?'rate_observation_id') THEN PERFORM ledger.fail('P2108'); END IF;
   amount_value:=ledger.money_value(l->'transaction_amount'); functional_value:=ledger.money_value(l->'functional_amount');
   l:=l||jsonb_build_object('account_id',aid,'transaction_amount',trim_scale(amount_value)::text,'functional_amount',trim_scale(functional_value)::text);
   IF basis_value='spot' THEN l:=l||jsonb_build_object('rate_observation_id',ledger.uuid_value(l->'rate_observation_id')); END IF;
   ls:=ls||jsonb_build_array(l);
  END LOOP;
  norm:=jsonb_set(norm,'{lines}',ls);
 ELSE
  reason_value:=ledger.text_value(v->'reason'); IF length(reason_value) NOT BETWEEN 1 AND 500 THEN PERFORM ledger.fail('P2100'); END IF;
 END IF;
 norm:=jsonb_build_object('operation',op,'entity_id',e,'book_id',b,'original',original,'body',norm);
 rev:=ledger.authorize(actor,e,b,'post',true);
 SELECT * INTO receipt FROM integration.operation_receipt WHERE entity_id=e AND book_id=b AND operation=op AND idempotency_key=k;
 IF FOUND THEN
  IF receipt.principal_id<>actor OR receipt.canonical_version<>1 OR receipt.normalized_request<>norm THEN PERFORM ledger.fail('P2104'); END IF;
  RETURN QUERY SELECT receipt.http_status,receipt.result,true; RETURN;
 END IF;
 IF expected<>rev THEN PERFORM ledger.fail('P2103'); END IF;
 IF rev=9223372036854775807 THEN PERFORM ledger.fail('P2110'); END IF;
 BEGIN
  SELECT * INTO STRICT period_row FROM ledger.period WHERE entity_id=e AND book_id=b AND starts_on<=d AND ends_before>d FOR UPDATE;
 EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM ledger.fail('P2108'); END;
 IF period_row.state<>'open' THEN PERFORM ledger.fail('P2105'); END IF;
 BEGIN
  SELECT * INTO STRICT current_policy FROM ledger.book_policy WHERE entity_id=e AND book_id=b AND effective_from<=d AND (effective_to IS NULL OR effective_to>d);
 EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM ledger.fail('P2109'); END;
 policy:=current_policy;
 IF op='reverse' THEN
  SELECT * INTO old FROM ledger.journal_entry WHERE entity_id=e AND book_id=b AND journal_id=original;
  IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
  IF old.kind<>'manual' OR EXISTS(SELECT FROM ledger.journal_entry WHERE entity_id=e AND book_id=b AND reverses_journal_id=original) THEN PERFORM ledger.fail('P2106'); END IF;
  SELECT * INTO policy FROM ledger.book_policy WHERE entity_id=e AND book_id=b AND policy_id=old.policy_id;
  IF policy.functional_currency_version_id<>current_policy.functional_currency_version_id THEN PERFORM ledger.fail('P2109'); END IF;
  policy_uuid:=policy.policy_id; description_value:='Reversal';
  SELECT jsonb_agg(jsonb_build_object('account_id',x.account_id,'side',CASE x.side WHEN 'debit' THEN 'credit' ELSE 'debit' END,
   'transaction_currency_version_id',x.transaction_currency_version_id,'transaction_amount',x.transaction_amount::text,
   'functional_amount',x.functional_amount::text,'valuation_basis',x.valuation_basis,'rate_observation_id',x.rate_observation_id) ORDER BY line_number)
  INTO ls FROM ledger.journal_line x WHERE entity_id=e AND book_id=b AND journal_id=original;
 ELSE IF policy_uuid<>policy.policy_id THEN PERFORM ledger.fail('P2109'); END IF; END IF;
 SELECT * INTO fc FROM reference.currency WHERE currency_version_id=policy.functional_currency_version_id;
 IF policy.basis<>'accrual' OR policy.timezone<>'UTC' OR policy.posting_scale<>2 OR policy.rounding_mode<>'half_even'
 OR policy.rate_policy_code<>'synthetic-reviewed-spot-v1' OR fc.alphabetic_code<>'USD' OR fc.minor_unit IS DISTINCT FROM 2
 OR d<fc.valid_from OR (fc.valid_to IS NOT NULL AND d>=fc.valid_to) THEN PERFORM ledger.fail('P2109'); END IF;
 -- Lock every existing account in stable order before inspecting original history.
 PERFORM 1 FROM ledger.account a WHERE a.entity_id=e AND a.book_id=b AND a.account_id IN (SELECT (value->>'account_id')::uuid FROM jsonb_array_elements(ls)) ORDER BY a.account_id FOR SHARE;
 IF original IS NOT NULL THEN PERFORM 1 FROM ledger.journal_entry WHERE entity_id=e AND book_id=b AND journal_id=original FOR SHARE; END IF;
 FOR l IN SELECT value FROM jsonb_array_elements(ls) LOOP
  aid:=(l->>'account_id')::uuid;
  IF NOT EXISTS(SELECT FROM ledger.account WHERE entity_id=e AND book_id=b AND account_id=aid AND active_from<=d AND (active_to IS NULL OR active_to>d)) THEN PERFORM ledger.fail('P2108'); END IF;
  amount_value:=(l->>'transaction_amount')::numeric; functional_value:=(l->>'functional_amount')::numeric; rid:=NULL;
  IF op='post' THEN
   BEGIN SELECT * INTO STRICT curr FROM reference.currency WHERE alphabetic_code=l->>'transaction_currency' AND valid_from<=d AND (valid_to IS NULL OR valid_to>d);
   EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM ledger.fail('P2108'); END;
   cid:=curr.currency_version_id;
   IF curr.minor_unit IS NULL OR amount_value<>trunc(amount_value,curr.minor_unit) OR functional_value<>trunc(functional_value,policy.posting_scale) THEN PERFORM ledger.fail('P2107'); END IF;
   IF l->>'valuation_basis'='identity' THEN
    IF cid<>fc.currency_version_id OR amount_value<>functional_value THEN PERFORM ledger.fail('P2108'); END IF;
   ELSE
    rid:=(l->>'rate_observation_id')::uuid; SELECT * INTO rate_row FROM reference.rate_observation WHERE rate_observation_id=rid;
    IF NOT FOUND THEN PERFORM ledger.fail('P2108'); END IF;
    IF rate_row.base_currency_version_id<>cid OR rate_row.quote_currency_version_id<>fc.currency_version_id OR rate_row.purpose<>'spot'
    OR rate_row.policy_code<>policy.rate_policy_code OR rate_row.source<>'synthetic' OR (rate_row.effective_at AT TIME ZONE 'UTC')::date<>d
    OR ledger.round_even(amount_value*rate_row.quote_per_base,policy.posting_scale)<>functional_value THEN PERFORM ledger.fail('P2108'); END IF;
   END IF;
  ELSE cid:=(l->>'transaction_currency_version_id')::uuid; rid:=(l->>'rate_observation_id')::uuid; END IF;
  total:=total+CASE l->>'side' WHEN 'debit' THEN functional_value ELSE -functional_value END;
  stored_lines:=stored_lines||jsonb_build_array(l||jsonb_build_object('transaction_currency_version_id',cid,'rate_observation_id',rid));
 END LOOP;
 IF total<>0 THEN PERFORM ledger.fail('P2108'); END IF;
 UPDATE ledger.book SET revision=revision+1 WHERE entity_id=e AND book_id=b;
 INSERT INTO ledger.journal_entry VALUES(e,b,j,period_row.period_id,policy_uuid,d,CASE op WHEN 'post' THEN 'manual' ELSE 'reversal' END,description_value,now_value,actor,rev+1,original,reason_value);
 FOR l IN SELECT value FROM jsonb_array_elements(stored_lines) LOOP
  n:=n+1;
  INSERT INTO ledger.journal_line VALUES(e,b,j,n,(l->>'account_id')::uuid,l->>'side',(l->>'transaction_currency_version_id')::uuid,
   (l->>'transaction_amount')::numeric,(l->>'functional_amount')::numeric,l->>'valuation_basis',(l->>'rate_observation_id')::uuid);
 END LOOP;
 INSERT INTO audit.action VALUES(e,b,gen_random_uuid(),actor,op,j,rev+1,request,now_value,reason_value);
 result:=ledger.render_journal(e,b,j);
 INSERT INTO integration.operation_receipt VALUES(e,b,op,k,actor,sha256(convert_to(norm::text,'UTF8')),norm,1,result,201,now_value);
 INSERT INTO integration.outbox_message VALUES(e,b,gen_random_uuid(),'journal.posted',1,j,rev+1,now_value);
 RETURN QUERY SELECT 201::smallint,result,false;
END $$;

-- Only these wrappers are executable by the runtime role; helpers stay private.
CREATE FUNCTION ledger.post_journal(actor uuid,e uuid,b uuid,k text,request uuid,raw text)
RETURNS TABLE(http_status smallint,body jsonb,replayed boolean) LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,ledger,reference,integration,audit,pg_temp AS $$
 SELECT * FROM ledger.mutate(actor,e,b,NULL,k,request,raw,'post')
$$;
-- Reuse the atomic authority with an explicit original journal target.
CREATE FUNCTION ledger.reverse_journal(actor uuid,e uuid,b uuid,original uuid,k text,request uuid,raw text)
RETURNS TABLE(http_status smallint,body jsonb,replayed boolean) LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,ledger,reference,integration,audit,pg_temp AS $$
 SELECT * FROM ledger.mutate(actor,e,b,original,k,request,raw,'reverse')
$$;
-- Authorize the read snapshot before exposing immutable journal facts.
CREATE FUNCTION reporting.get_journal(actor uuid,e uuid,b uuid,j uuid) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,ledger,reference,integration,audit,pg_temp AS $$
DECLARE result jsonb;
BEGIN
 PERFORM ledger.authorize(actor,e,b,'read',false);
 IF j IS NULL THEN PERFORM ledger.fail('P2100'); END IF;
 result:=ledger.render_journal(e,b,j); IF result IS NULL THEN PERFORM ledger.fail('P2101'); END IF; RETURN result;
END $$;
-- Aggregate only this scoped snapshot, counting originals and reversals.
CREATE FUNCTION reporting.trial_balance(actor uuid,e uuid,b uuid,through_date text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,ledger,reference,integration,audit,pg_temp AS $$
DECLARE rev bigint; d date:=ledger.date_value(through_date); rows jsonb; dr numeric; cr numeric; curr text;
BEGIN
 rev:=ledger.authorize(actor,e,b,'read',false);
 BEGIN SELECT c.alphabetic_code INTO STRICT curr FROM ledger.book_policy p JOIN reference.currency c ON c.currency_version_id=p.functional_currency_version_id
 WHERE p.entity_id=e AND p.book_id=b AND p.effective_from<=d AND (p.effective_to IS NULL OR p.effective_to>d);
 EXCEPTION WHEN no_data_found OR too_many_rows THEN PERFORM ledger.fail('P2109'); END;
 IF EXISTS(SELECT FROM ledger.journal_entry h JOIN ledger.book_policy p USING(entity_id,book_id,policy_id) JOIN reference.currency c ON c.currency_version_id=p.functional_currency_version_id WHERE h.entity_id=e AND h.book_id=b AND h.accounting_date<=d AND c.alphabetic_code<>curr) THEN PERFORM ledger.fail('P2109'); END IF;
 WITH totals AS (
 SELECT a.account_id,a.code,coalesce(sum(CASE l.side WHEN 'debit' THEN l.functional_amount ELSE -l.functional_amount END) FILTER(WHERE h.journal_id IS NOT NULL),0) AS net
 FROM ledger.account a LEFT JOIN ledger.journal_line l USING(entity_id,book_id,account_id)
 LEFT JOIN ledger.journal_entry h ON h.entity_id=l.entity_id AND h.book_id=l.book_id AND h.journal_id=l.journal_id AND h.accounting_date<=d
 WHERE a.entity_id=e AND a.book_id=b GROUP BY a.account_id,a.code)
 SELECT jsonb_agg(jsonb_build_object('account_id',account_id,'account_code',code,'debit_balance',trim_scale(greatest(net,0))::text,'credit_balance',trim_scale(greatest(-net,0))::text) ORDER BY code,account_id),coalesce(sum(greatest(net,0)),0),coalesce(sum(greatest(-net,0)),0)
 INTO rows,dr,cr FROM totals;
 IF dr>=1e29 OR cr>=1e29 THEN PERFORM ledger.fail('P2110'); END IF;
 RETURN jsonb_build_object('entity_id',e,'book_id',b,'through_date',through_date,'snapshot_revision',rev::text,'functional_currency',curr,
 'total_debits',trim_scale(dr)::text,'total_credits',trim_scale(cr)::text,'accounts',coalesce(rows,'[]'::jsonb));
END $$;
GRANT EXECUTE ON FUNCTION ledger.post_journal(uuid,uuid,uuid,text,uuid,text),ledger.reverse_journal(uuid,uuid,uuid,uuid,text,uuid,text),reporting.get_journal(uuid,uuid,uuid,uuid),reporting.trial_balance(uuid,uuid,uuid,text) TO bookkeeping_api;
COMMIT;
