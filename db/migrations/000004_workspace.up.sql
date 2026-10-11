-- Internal discovery, not an ISO financial exchange. No journal/balance payloads.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
-- Actor-first visibility avoids scanning unrelated entity memberships.
CREATE INDEX membership_read_discovery ON ledger.membership(principal_id,entity_id) WHERE permission='read' AND revoked_at IS NULL;
-- Lock current authority before constructing one bounded repeatable-read snapshot.
CREATE FUNCTION reporting.workspace(actor uuid) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE e record; b record; entities jsonb:='[]'; books jsonb; permissions jsonb;
 policies jsonb; periods jsonb; accounts jsonb; currencies jsonb; result jsonb;
 entity_count integer:=0; book_count integer:=0;
BEGIN
 IF actor IS NULL OR current_setting('transaction_isolation') NOT IN ('repeatable read','serializable') THEN PERFORM ledger.fail('P2100'); END IF;
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR SHARE;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 -- Read grant determines visibility; post alone never exposes configuration.
 FOR e IN SELECT x.* FROM ledger.membership m JOIN ledger.entity x ON x.entity_id=m.entity_id
  WHERE m.principal_id=actor AND m.permission='read' AND m.revoked_at IS NULL AND x.disabled_at IS NULL
  ORDER BY x.entity_id LIMIT 101 FOR SHARE OF x LOOP
  entity_count:=entity_count+1; IF entity_count>100 THEN PERFORM ledger.fail('P2111'); END IF;
  PERFORM 1 FROM ledger.membership m WHERE m.entity_id=e.entity_id AND m.principal_id=actor AND m.revoked_at IS NULL ORDER BY m.permission FOR SHARE;
  SELECT jsonb_agg(m.permission ORDER BY m.permission) INTO permissions FROM ledger.membership m WHERE m.entity_id=e.entity_id AND m.principal_id=actor AND m.revoked_at IS NULL;
  books:='[]';
  FOR b IN SELECT * FROM ledger.book WHERE entity_id=e.entity_id AND disabled_at IS NULL ORDER BY book_id LIMIT 201 FOR SHARE LOOP
   book_count:=book_count+1; IF book_count>200 THEN PERFORM ledger.fail('P2111'); END IF;
   -- Exact scoped columns avoid inadvertently exposing future table additions.
   SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.effective_from),'[]') INTO policies FROM
    (SELECT p.policy_id,c.alphabetic_code AS functional_currency,p.functional_currency_version_id,p.effective_from,p.effective_to
     FROM ledger.book_policy p JOIN reference.currency c ON c.currency_version_id=p.functional_currency_version_id
     WHERE p.entity_id=e.entity_id AND p.book_id=b.book_id ORDER BY p.effective_from LIMIT 101) x;
   SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.starts_on),'[]') INTO periods FROM
    (SELECT period_id,starts_on,ends_before,state FROM ledger.period WHERE entity_id=e.entity_id AND book_id=b.book_id ORDER BY starts_on LIMIT 241) x;
   SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.code),'[]') INTO accounts FROM
    (SELECT account_id,code,name,category,normal_side,monetary,active_from,active_to FROM ledger.account WHERE entity_id=e.entity_id AND book_id=b.book_id ORDER BY code LIMIT 201) x;
   IF jsonb_array_length(policies)>100 OR jsonb_array_length(periods)>240 OR jsonb_array_length(accounts)>200 THEN PERFORM ledger.fail('P2111'); END IF;
   books:=books||jsonb_build_array(jsonb_build_object('book_id',b.book_id,'name',b.name,'book_revision',b.revision::text,'policies',policies,'periods',periods,'accounts',accounts));
   IF octet_length(books::text)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
  END LOOP;
  entities:=entities||jsonb_build_array(jsonb_build_object('entity_id',e.entity_id,'name',e.name,'kind',e.kind,'permissions',permissions,'books',books));
  IF octet_length(entities::text)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 END LOOP;
 SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.alphabetic_code,x.valid_from),'[]') INTO currencies FROM
  (SELECT currency_version_id,alphabetic_code,minor_unit,valid_from,valid_to FROM reference.currency ORDER BY alphabetic_code,valid_from LIMIT 1001) x;
 IF jsonb_array_length(currencies)>1000 THEN PERFORM ledger.fail('P2111'); END IF;
 result:=jsonb_build_object('entities',entities,'currencies',currencies,'development_policy','dev-accrual-usd-v1');
 IF octet_length(result::text)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 RETURN result;
END $$;
REVOKE ALL ON FUNCTION reporting.workspace(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION reporting.workspace(uuid) TO bookkeeping_api;
COMMIT;
