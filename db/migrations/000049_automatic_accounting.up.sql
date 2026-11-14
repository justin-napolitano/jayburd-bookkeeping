-- #334 reviewed bounded source-identity-usd-v1 automatic accounting; no runtime activation.
BEGIN
  ;

SET LOCAL ROLE bookkeeping_owner;

CREATE TABLE banking.automatic_accounting_policy(
  entity_id uuid NOT NULL,book_id uuid NOT NULL,revision bigint NOT NULL CHECK(revision>0),
  PRIMARY KEY(entity_id,book_id),FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
  );

CREATE TABLE banking.automatic_accounting_edition(
  entity_id uuid NOT NULL,book_id uuid NOT NULL,revision bigint NOT NULL CHECK(revision>0),actor_id uuid NOT NULL REFERENCES ledger.principal,
  enabled boolean NOT NULL,configuration jsonb NOT NULL CHECK(octet_length(configuration::text)<=131072),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY(entity_id,book_id,revision),FOREIGN KEY(entity_id,book_id) REFERENCES banking.automatic_accounting_policy DEFERRABLE INITIALLY DEFERRED
  );

ALTER TABLE banking.automatic_accounting_policy ADD FOREIGN KEY(entity_id,book_id,revision)REFERENCES banking.automatic_accounting_edition DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE banking.automatic_accounting_setting(
  entity_id uuid NOT NULL,book_id uuid NOT NULL,actor_id uuid NOT NULL REFERENCES ledger.principal,idempotency_key text NOT NULL,
  normalized_request jsonb NOT NULL,result jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY(entity_id,book_id,actor_id,idempotency_key),FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
  );

CREATE TABLE banking.automatic_accounting_operation(
  entity_id uuid NOT NULL,book_id uuid NOT NULL,operation_id uuid NOT NULL DEFAULT gen_random_uuid(),actor_id uuid NOT NULL REFERENCES ledger.principal,
  policy_revision bigint NOT NULL,review_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,observation_revision bigint NOT NULL,
  input_context jsonb NOT NULL CHECK(octet_length(input_context::text)<=16384),input_hash text NOT NULL CHECK(input_hash COLLATE "C" ~ '^[0-9a-f]{64}$'),
  rule_id uuid,state text NOT NULL CHECK(state IN('prepared','committed','held','stale')),reason text,result jsonb,
  created_at timestamptz NOT NULL DEFAULT clock_timestamp(),terminal_at timestamptz,
  PRIMARY KEY(entity_id,book_id,operation_id),FOREIGN KEY(entity_id,book_id,policy_revision) REFERENCES banking.automatic_accounting_edition,
  FOREIGN KEY(entity_id,book_id,review_id) REFERENCES banking.review_item,
  FOREIGN KEY(entity_id,book_id,source_account_id) REFERENCES banking.source_account,
  FOREIGN KEY(entity_id,book_id,observation_id,observation_revision) REFERENCES banking.observation_revision,
  UNIQUE(entity_id,book_id,review_id,input_hash),
  CHECK((state='prepared' AND reason IS NULL AND terminal_at IS NULL)OR(state='committed' AND reason IS NULL AND result IS NOT NULL AND terminal_at IS NOT NULL)OR(state IN('held','stale')AND reason IS NOT NULL AND terminal_at IS NOT NULL))
  );

CREATE UNIQUE INDEX automatic_accounting_one_prepared ON banking.automatic_accounting_operation(entity_id,book_id)
  WHERE state='prepared';

ALTER TABLE banking.review_draft ADD COLUMN automatic_operation_id uuid;

ALTER TABLE banking.review_draft ADD UNIQUE(entity_id,book_id,automatic_operation_id);

ALTER TABLE banking.review_draft ADD FOREIGN KEY(entity_id,book_id,automatic_operation_id) REFERENCES banking.automatic_accounting_operation;

ALTER TABLE banking.review_posting ALTER COLUMN command_id DROP NOT NULL;

ALTER TABLE banking.review_posting ADD COLUMN automatic_operation_id uuid;

ALTER TABLE banking.review_posting ADD CONSTRAINT review_posting_origin CHECK(num_nonnulls(command_id,automatic_operation_id)=1);

ALTER TABLE banking.review_posting ADD UNIQUE(entity_id,book_id,automatic_operation_id);

ALTER TABLE banking.review_posting ADD FOREIGN KEY(entity_id,book_id,automatic_operation_id) REFERENCES banking.automatic_accounting_operation;

ALTER TABLE banking.review_decision DROP CONSTRAINT review_decision_reason_check;

ALTER TABLE banking.review_decision ADD CONSTRAINT review_decision_reason_check CHECK(reason IN('synthetic_import','explicit_category','provider_sync','automatic_policy'));

CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.automatic_accounting_edition FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();

CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.automatic_accounting_setting FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();

CREATE TABLE banking.automatic_accounting_cursor(entity_id uuid NOT NULL,book_id uuid NOT NULL,after_review_id uuid,last_work_at timestamptz NOT NULL DEFAULT '-infinity',PRIMARY KEY(entity_id,book_id),FOREIGN KEY(entity_id,book_id)REFERENCES ledger.book);

CREATE FUNCTION banking.automatic_accounting_error(code text)
  RETURNS void
  LANGUAGE plpgsql
  SET search_path=pg_catalog,pg_temp AS $$BEGIN
  RAISE
  EXCEPTION  USING ERRCODE=code,MESSAGE='automatic_accounting_rejected';

END $$;

CREATE FUNCTION banking.automatic_accounting_guard()
  RETURNS trigger
  LANGUAGE plpgsql
  SET search_path=pg_catalog,pg_temp AS $$BEGIN

  IF TG_OP='DELETE' OR OLD.state<>'prepared' OR NEW.state NOT IN('committed','stale','held') OR (to_jsonb(NEW)-ARRAY['state','reason','result','terminal_at'])IS DISTINCT
  FROM(to_jsonb(OLD)-ARRAY['state','reason','result','terminal_at']) THEN
  PERFORM banking.automatic_accounting_error('P2A02');

END IF;
  RETURN NEW;

END $$;

CREATE TRIGGER preserve_automatic_operation BEFORE UPDATE OR DELETE ON banking.automatic_accounting_operation FOR EACH ROW EXECUTE FUNCTION banking.automatic_accounting_guard();

-- Strong principal and membership locks precede existing book/source helpers; no worker grants are inherited.
CREATE FUNCTION banking.automatic_accounting_authorize(actor uuid,e uuid,b uuid,writing boolean)
  RETURNS void
  LANGUAGE plpgsql
  SET search_path=pg_catalog,pg_temp AS $$BEGIN

  IF actor IS NULL OR e IS NULL OR b IS NULL OR current_setting('transaction_isolation')<>'serializable' THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

  IF writing THEN
  PERFORM 1
  FROM ledger.principal
  WHERE principal_id=actor AND disabled_at IS NULL FOR UPDATE;

  ELSE  PERFORM 1
  FROM ledger.principal
  WHERE principal_id=actor AND disabled_at IS NULL FOR SHARE;

END IF;

  IF NOT FOUND THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END IF;

  PERFORM 1
  FROM ledger.entity
  WHERE entity_id=e AND disabled_at IS NULL FOR SHARE;
  IF NOT FOUND THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END IF;

  PERFORM 1
  FROM ledger.membership
  WHERE entity_id=e AND principal_id=actor AND permission IN('read','prepare','post') AND revoked_at IS NULL
  ORDER BY permission FOR SHARE;

  IF (SELECT count(*)
  FROM ledger.membership
  WHERE entity_id=e AND principal_id=actor AND permission IN('read','prepare','post') AND revoked_at IS NULL)<>3 THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END IF;

  PERFORM ledger.authorize(actor,e,b,'read',writing);

  EXCEPTION  WHEN SQLSTATE 'P2100' THEN
  PERFORM banking.automatic_accounting_error('P2A00');
  WHEN SQLSTATE 'P2101' OR SQLSTATE 'P2102' THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END $$;

CREATE FUNCTION banking.automatic_accounting_source(actor uuid,e uuid,b uuid,sid uuid)
  RETURNS banking.source_account
  LANGUAGE plpgsql
  SET search_path=pg_catalog,pg_temp AS $$DECLARE s banking.source_account;
BEGIN

  s:=banking.import_authorize(actor,e,b,sid,false);

  PERFORM 1
  FROM banking.source_permission
  WHERE entity_id=e AND book_id=b AND source_account_id=sid AND principal_id=actor AND capability='review' AND revoked_at IS NULL FOR SHARE;

  IF NOT FOUND THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END IF;
  RETURN s;

  EXCEPTION  WHEN SQLSTATE 'P2401' OR SQLSTATE 'P2501' THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END $$;

CREATE FUNCTION banking.automatic_accounting_policy_wire(e uuid,b uuid)
  RETURNS jsonb
  LANGUAGE plpgsql
  SET search_path=pg_catalog,pg_temp AS $$DECLARE p banking.automatic_accounting_edition;
BEGIN

  SELECT x.* INTO p
  FROM banking.automatic_accounting_policy h JOIN banking.automatic_accounting_edition x USING(entity_id,book_id,revision)
  WHERE h.entity_id=e AND h.book_id=b;

  RETURN jsonb_build_object('profile','automatic-accounting-policy-v1','entity_id',e,'book_id',b,'revision',coalesce(p.revision,0)::text,'enabled',coalesce(p.enabled,false),'policy',p.configuration,'supported_profile','source-identity-usd-v1');

END $$;

CREATE FUNCTION banking.get_automatic_accounting(actor uuid,e uuid,b uuid)
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$BEGIN

  PERFORM banking.automatic_accounting_authorize(actor,e,b,false);
  RETURN banking.automatic_accounting_policy_wire(e,b);

END $$;

-- Configuration is explicit current-authorized owner work; it cannot accept source facts or posting lines.
CREATE FUNCTION banking.set_automatic_accounting(actor uuid,e uuid,b uuid,raw text)
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$
  DECLARE v jsonb;
  config jsonb;
  r jsonb;
  rr jsonb;
  sid uuid;
  aid uuid;
  pid uuid;
  first_day date;
  last_day date;
  expected bigint;
  revision_value bigint;
  k text;
  p ledger.book_policy;
  s banking.source_account;
  receipt banking.automatic_accounting_setting;
  result_value jsonb;
  ids uuid[]:='{}';
  names text[]:='{}';
  ruleid uuid;
  rule_name text;
  kind text;
  direction_value text;
  account_value ledger.account;

BEGIN

  PERFORM banking.automatic_accounting_authorize(actor,e,b,true);
  v:=banking.import_json(raw,131072);

  IF NOT(v?&ARRAY['key','expected_revision','enabled','book_policy_id','effective_from','effective_to','rules'])OR EXISTS(SELECT
  FROM jsonb_object_keys(v)x
  WHERE x<>ALL(ARRAY['key','expected_revision','enabled','book_policy_id','effective_from','effective_to','rules']))THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

  k:=ledger.text_value(v->'key');
  expected:=banking.import_revision(v->'expected_revision');
  pid:=ledger.uuid_value(v->'book_policy_id');
  first_day:=ledger.date_value(ledger.text_value(v->'effective_from'));

  IF v->'effective_to'<>'null'::jsonb THEN
  last_day:=ledger.date_value(ledger.text_value(v->'effective_to'));
  IF last_day<=first_day THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

END IF;

  IF k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' OR jsonb_typeof(v->'enabled')<>'boolean' OR jsonb_typeof(v->'rules')<>'array' OR jsonb_array_length(v->'rules')>100 THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

  SELECT * INTO p
  FROM ledger.book_policy
  WHERE entity_id=e AND book_id=b AND policy_id=pid;

  IF NOT FOUND OR first_day<p.effective_from OR(p.effective_to IS NOT NULL AND(last_day IS NULL OR last_day>p.effective_to))THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

-- Source scopes are locked in UUID order before account inspection, independent of caller array order.
  FOR sid IN SELECT DISTINCT ledger.uuid_value(x->'source_account_id')
  FROM jsonb_array_elements(v->'rules')x
  ORDER BY 1 LOOP
  s:=banking.automatic_accounting_source(actor,e,b,sid);

END LOOP
  ;

  FOR r IN SELECT value
  FROM jsonb_array_elements(v->'rules')LOOP

  PERFORM ledger.keys(r,ARRAY['rule_id','name','source_account_id','direction','description','offset_account_id','treatment']);

  ruleid:=ledger.uuid_value(r->'rule_id');
  rule_name:=ledger.text_value(r->'name');
  sid:=ledger.uuid_value(r->'source_account_id');
  aid:=ledger.uuid_value(r->'offset_account_id');
  direction_value:=ledger.text_value(r->'direction');
  kind:=ledger.text_value(r->'treatment');

  IF ruleid=ANY(ids)OR rule_name=ANY(names)OR length(rule_name)NOT BETWEEN 1 AND 80 OR btrim(rule_name)='' OR rule_name~'[[:cntrl:]]' OR length(ledger.text_value(r->'description'))NOT BETWEEN 1 AND 500 OR direction_value NOT IN('inflow','outflow')OR kind NOT IN('expense','income','transfer','card_payment','loan')THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;
  ids:=array_append(ids,ruleid);
  names:=array_append(names,rule_name);

  SELECT * INTO s
  FROM banking.source_account
  WHERE entity_id=e AND book_id=b AND source_account_id=sid;

  SELECT * INTO account_value
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id=aid FOR SHARE;

  IF NOT FOUND OR aid=s.ledger_account_id THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

  IF(kind='expense'AND(direction_value<>'outflow'OR account_value.category<>'expense'OR account_value.normal_side<>'debit'))OR(kind='income'AND(direction_value<>'inflow'OR account_value.category<>'revenue'OR account_value.normal_side<>'credit'))THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

END LOOP
  ;

  config:=v-ARRAY['key','expected_revision','enabled'];

  SELECT * INTO receipt
  FROM banking.automatic_accounting_setting
  WHERE entity_id=e AND book_id=b AND actor_id=actor AND idempotency_key=k;

  IF FOUND THEN
  IF receipt.normalized_request<>v THEN
  PERFORM banking.automatic_accounting_error('P2A02');

END IF;
  RETURN receipt.result;

END IF;

  SELECT revision INTO revision_value
  FROM banking.automatic_accounting_policy
  WHERE entity_id=e AND book_id=b;

  revision_value:=coalesce(revision_value,0);
  IF revision_value<>expected THEN
  PERFORM banking.automatic_accounting_error('P2A03');

END IF;
  IF revision_value=9223372036854775807 THEN
  PERFORM banking.automatic_accounting_error('P2A04');

END IF;

  revision_value:=revision_value+1;

  INSERT INTO banking.automatic_accounting_policy
  VALUES(e,b,revision_value)ON CONFLICT(entity_id,book_id)DO UPDATE SET revision=EXCLUDED.revision;

  INSERT INTO banking.automatic_accounting_edition(entity_id,book_id,revision,actor_id,enabled,configuration)
  VALUES(e,b,revision_value,actor,(v->>'enabled')::boolean,config);

  result_value:=banking.automatic_accounting_policy_wire(e,b);

  INSERT INTO banking.automatic_accounting_setting(entity_id,book_id,actor_id,idempotency_key,normalized_request,result)
  VALUES(e,b,actor,k,v,result_value);
  RETURN result_value;

  EXCEPTION  WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR numeric_value_out_of_range THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END $$;

-- One shared financial builder retains manual semantics and records a distinct automatic origin.
CREATE FUNCTION banking.create_source_draft(actor uuid,e uuid,b uuid,id uuid,k text,raw text,automatic uuid)
  RETURNS jsonb
  LANGUAGE plpgsql
  SET search_path=pg_catalog AS $$
  DECLARE v jsonb;
  r banking.review_item;
  s banking.source_account;
  q banking.observation_revision;
  d banking.review_draft;
  p ledger.book_policy;
  a ledger.account;
  rev bigint;
  category_id uuid;
  day_value date;
  description_value text;
  code text;
  amount_value text;
  body jsonb;
  norm jsonb;

BEGIN

  v:=banking.import_json(raw,8192);
  PERFORM ledger.keys(v,ARRAY['expected_review_revision','source_revision','binding_revision','expected_book_revision','category_account_id','policy_id','accounting_date','description']);

  IF k IS NULL OR k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$' THEN
  PERFORM banking.review_error('P2500');

END IF;

  PERFORM banking.import_revision(v->'expected_review_revision');
  PERFORM banking.import_revision(v->'source_revision');
  PERFORM banking.import_revision(v->'binding_revision');
  PERFORM banking.import_revision(v->'expected_book_revision');

  category_id:=ledger.uuid_value(v->'category_account_id');
  PERFORM ledger.uuid_value(v->'policy_id');
  day_value:=ledger.date_value(ledger.text_value(v->'accounting_date'));
  description_value:=ledger.text_value(v->'description');

  IF length(description_value) NOT BETWEEN 1 AND 500 THEN
  PERFORM banking.review_error('P2500');

END IF;

  s:=banking.review_authorize(actor,e,b,id,false);

  norm:=jsonb_build_object('review_id',id,'body',v);

  SELECT * INTO d
  FROM banking.review_draft
  WHERE entity_id=e AND book_id=b AND created_by=actor AND idempotency_key=k;

  IF FOUND THEN
  IF d.automatic_operation_id IS DISTINCT
  FROM automatic OR d.normalized_request<>norm THEN
  PERFORM banking.review_error('P2502');

END IF;
  RETURN banking.render_review_draft(d);

END IF;

  SELECT * INTO r
  FROM banking.review_item
  WHERE entity_id=e AND book_id=b AND review_id=id FOR UPDATE;

  SELECT revision INTO rev
  FROM ledger.book
  WHERE entity_id=e AND book_id=b;

  IF r.state NOT IN ('needs_review','draft_ready') OR s.state<>'active' THEN
  PERFORM banking.review_error('P2502');

END IF;

  IF ROW(r.review_revision,s.source_revision,s.binding_revision,rev) IS DISTINCT
  FROM ROW((v->>'expected_review_revision')::bigint,(v->>'source_revision')::bigint,(v->>'binding_revision')::bigint,(v->>'expected_book_revision')::bigint) THEN
  PERFORM banking.review_error('P2503');

END IF;

  IF s.source_revision=9223372036854775807 OR r.review_revision=9223372036854775807 THEN
  PERFORM banking.review_error('P2505');

END IF;

  q:=banking.review_fact(e,b,id);

  IF q.booked_date<>day_value OR q.source_state<>'booked' THEN
  PERFORM banking.review_error('P2500');

END IF;

BEGIN
  SELECT * INTO STRICT p
  FROM ledger.book_policy
  WHERE entity_id=e AND book_id=b AND effective_from<=day_value AND (effective_to IS NULL OR effective_to>day_value);

  EXCEPTION  WHEN no_data_found OR too_many_rows THEN
  PERFORM ledger.fail('P2109');

END;

  IF p.policy_id<>(v->>'policy_id')::uuid THEN
  PERFORM banking.review_error('P2503');

END IF;

  IF q.currency_version_id<>p.functional_currency_version_id THEN
  PERFORM banking.review_error('P2504');

END IF;

  SELECT alphabetic_code INTO code
  FROM reference.currency
  WHERE currency_version_id=q.currency_version_id;

-- Match the existing reviewed posting policy; do not claim a new accounting framework.
  IF p.basis<>'accrual' OR p.timezone<>'UTC' OR p.posting_scale<>2 OR p.rounding_mode<>'half_even' OR p.rate_policy_code<>'synthetic-reviewed-spot-v1' OR code<>'USD' THEN
  PERFORM ledger.fail('P2109');

END IF;

  PERFORM 1
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id IN (s.ledger_account_id,category_id)
  ORDER BY account_id FOR SHARE;

  IF category_id=s.ledger_account_id THEN
  PERFORM banking.review_error('P2500');

END IF;

  SELECT * INTO a
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id=category_id AND active_from<=day_value AND (active_to IS NULL OR active_to>day_value);

  IF NOT FOUND OR NOT EXISTS(SELECT
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id=s.ledger_account_id AND active_from<=day_value AND (active_to IS NULL OR active_to>day_value)) THEN
  PERFORM banking.review_error('P2500');

END IF;

  amount_value:=trim_scale(q.amount)::text;

  body:=jsonb_build_object('expected_revision',rev::text,'policy_id',p.policy_id,'accounting_date',to_char(day_value,'YYYY-MM-DD'),'description',description_value,'lines',jsonb_build_array(
  jsonb_build_object('account_id',s.ledger_account_id,'side',CASE q.direction WHEN 'inflow' THEN
  'debit'
  ELSE  'credit'
END,'transaction_currency',code,'transaction_amount',amount_value,'functional_amount',amount_value,'valuation_basis','identity'),
  jsonb_build_object('account_id',category_id,'side',CASE q.direction WHEN 'inflow' THEN
  'credit'
  ELSE  'debit'
END,'transaction_currency',code,'transaction_amount',amount_value,'functional_amount',amount_value,'valuation_basis','identity')));

  PERFORM ledger.validate_command_body('post',body);

  UPDATE banking.source_account SET source_revision=source_revision+1
  WHERE entity_id=e AND book_id=b AND source_account_id=s.source_account_id RETURNING * INTO s;

  UPDATE banking.review_item SET state='draft_ready',review_revision=review_revision+1
  WHERE entity_id=e AND book_id=b AND review_id=id RETURNING * INTO r;

  INSERT INTO banking.review_draft(entity_id,book_id,review_id,review_revision,source_revision,binding_revision,book_revision,posting_body,normalized_request,created_by,idempotency_key,automatic_operation_id)

  VALUES(e,b,id,r.review_revision,s.source_revision,s.binding_revision,rev,body,norm,actor,k,automatic) RETURNING * INTO d;

  INSERT INTO banking.review_decision(entity_id,book_id,review_id,expected_review_revision,kind,actor_id,reason,body)
  VALUES(e,b,id,r.review_revision-1,'draft',actor,CASE WHEN automatic IS NULL THEN
  'explicit_category'
  ELSE  'automatic_policy'
END,jsonb_strip_nulls(jsonb_build_object('draft_id',d.draft_id,'category_account_id',category_id,'source_revision',s.source_revision::text,'automatic_operation_id',automatic)));

  RETURN banking.render_review_draft(d);

  EXCEPTION  WHEN SQLSTATE 'P2100' OR SQLSTATE 'P2400' OR numeric_value_out_of_range THEN
  PERFORM banking.review_error('P2500');
  WHEN unique_violation THEN
  RAISE
  EXCEPTION  USING ERRCODE='40001',MESSAGE='retry_review_draft';

END $$;

CREATE OR REPLACE FUNCTION banking.create_review_draft(actor uuid,e uuid,b uuid,id uuid,k text,raw text)
  RETURNS jsonb
  LANGUAGE sql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$ SELECT banking.create_source_draft(actor,e,b,id,k,raw,NULL) $$;

CREATE FUNCTION banking.authorize_source_post_core(actor uuid,e uuid,b uuid,body jsonb,automatic uuid)
  RETURNS banking.review_draft
  LANGUAGE plpgsql
  SET search_path=pg_catalog AS $$
  DECLARE d banking.review_draft;

BEGIN

  PERFORM banking.review_authorize(actor,e,b,(body->>'review_id')::uuid,true);

  SELECT * INTO d
  FROM banking.review_draft
  WHERE entity_id=e AND book_id=b AND review_id=(body->>'review_id')::uuid AND draft_id=(body->>'draft_id')::uuid;

  IF NOT FOUND THEN
  PERFORM banking.review_error('P2501');

END IF;

  IF d.automatic_operation_id IS DISTINCT
  FROM automatic THEN
  PERFORM banking.review_error('P2502');

END IF;

  IF ROW(d.review_revision,d.source_revision,d.binding_revision,d.book_revision,d.posting_body) IS DISTINCT
  FROM ROW((body->>'review_revision')::bigint,(body->>'source_revision')::bigint,(body->>'binding_revision')::bigint,(body->>'expected_book_revision')::bigint,body->'posting_body') THEN
  PERFORM banking.review_error('P2502');

END IF;

  RETURN d;

END $$;

CREATE OR REPLACE FUNCTION banking.source_post_authorize(actor uuid,e uuid,b uuid,body jsonb)
  RETURNS banking.review_draft
  LANGUAGE sql
  SET search_path=pg_catalog,pg_temp AS $$ SELECT banking.authorize_source_post_core(actor,e,b,body,NULL) $$;

-- Journal/link/projection mutation remains a single shared source-backed transaction.
CREATE FUNCTION banking.commit_source_post(actor uuid,e uuid,b uuid,command uuid,automatic uuid,request uuid,body jsonb)
  RETURNS jsonb
  LANGUAGE plpgsql
  SET search_path=pg_catalog AS $$
  DECLARE d banking.review_draft;
  r banking.review_item;
  s banking.source_account;
  link banking.review_posting;
  rev bigint;
  posted record;
  result_value jsonb;

BEGIN

  IF num_nonnulls(command,automatic)<>1 THEN
  PERFORM banking.review_error('P2500');

END IF;

  d:=banking.authorize_source_post_core(actor,e,b,body,automatic);

  IF d.automatic_operation_id IS DISTINCT
  FROM automatic THEN
  PERFORM banking.review_error('P2502');

END IF;

  SELECT * INTO link
  FROM banking.review_posting
  WHERE entity_id=e AND book_id=b AND review_id=d.review_id;

  IF FOUND THEN
  IF link.draft_id<>d.draft_id THEN
  PERFORM banking.review_error('P2502');

END IF;
  RETURN link.result;

END IF;

  SELECT * INTO r
  FROM banking.review_item
  WHERE entity_id=e AND book_id=b AND review_id=d.review_id FOR UPDATE;

  SELECT * INTO s
  FROM banking.source_account
  WHERE entity_id=e AND book_id=b AND source_account_id=r.source_account_id;

  SELECT revision INTO rev
  FROM ledger.book
  WHERE entity_id=e AND book_id=b;

  IF r.state<>'draft_ready' OR s.state<>'active' THEN
  PERFORM banking.review_error('P2502');

END IF;

  IF ROW(r.review_revision,s.source_revision,s.binding_revision,rev) IS DISTINCT
  FROM ROW(d.review_revision,d.source_revision,d.binding_revision,d.book_revision) THEN
  PERFORM banking.review_error('P2503');

END IF;

  IF r.review_revision=9223372036854775807 OR s.source_revision=9223372036854775807 THEN
  PERFORM banking.review_error('P2505');

END IF;

  SELECT * INTO posted
  FROM ledger.post_journal(actor,e,b,CASE WHEN automatic IS NULL THEN
  'source-command:'||command
  ELSE  'source-automatic:'||automatic
END,request,d.posting_body::text);

  UPDATE banking.source_account SET source_revision=source_revision+1
  WHERE entity_id=e AND book_id=b AND source_account_id=s.source_account_id RETURNING * INTO s;

  UPDATE banking.review_item SET state='posted',review_revision=review_revision+1
  WHERE entity_id=e AND book_id=b AND review_id=r.review_id RETURNING * INTO r;

  result_value:=jsonb_build_object('entity_id',e,'book_id',b,'review_id',r.review_id,'draft_id',d.draft_id,'journal_id',posted.body->>'journal_id','book_revision',posted.body->>'book_revision','journal',posted.body,'review_revision',r.review_revision::text,'source_revision',s.source_revision::text);

  INSERT INTO banking.review_posting(entity_id,book_id,review_id,draft_id,command_id,automatic_operation_id,journal_id,result)
  VALUES(e,b,r.review_id,d.draft_id,command,automatic,(posted.body->>'journal_id')::uuid,result_value);

  RETURN result_value;

END $$;

CREATE OR REPLACE FUNCTION banking.execute_source_post(actor uuid,e uuid,b uuid,command uuid,request uuid,body jsonb)
  RETURNS jsonb
  LANGUAGE sql
  SET search_path=pg_catalog,pg_temp AS $$ SELECT banking.commit_source_post(actor,e,b,command,NULL,request,body) $$;

CREATE OR REPLACE FUNCTION banking.render_review_draft(d banking.review_draft)
  RETURNS jsonb
  LANGUAGE sql STABLE
  SET search_path=pg_catalog AS $$
  SELECT CASE WHEN d.draft_id IS NULL THEN
  NULL
  ELSE  jsonb_build_object('draft_id',d.draft_id,'review_id',d.review_id,'review_revision',d.review_revision::text,'source_revision',d.source_revision::text,'binding_revision',d.binding_revision::text,'book_revision',d.book_revision::text,'posting_body',d.posting_body)||CASE WHEN d.automatic_operation_id IS NULL THEN
  '{}'::jsonb
  ELSE  jsonb_build_object('profile','automatic-source-draft-v1','origin','automatic','automatic_operation_id',d.automatic_operation_id)
END
END
  $$;

-- Stable set-based input identities support complete counts before bounded selection/pagination.
CREATE FUNCTION banking.automatic_accounting_inputs(actor uuid,e uuid,b uuid,policy bigint)
  RETURNS TABLE(review_id uuid,source_account_id uuid,observation_id uuid,observation_revision bigint,input_context jsonb)
  LANGUAGE sql STABLE
  SET search_path=pg_catalog,pg_temp AS $$
  SELECT r.review_id,s.source_account_id,o.observation_id,o.current_revision,
  jsonb_build_object('policy_revision',policy::text,'review_revision',r.review_revision::text,'review_state',r.state,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'source_state',s.state,'book_revision',bk.revision::text,'observation_id',o.observation_id,'observation_revision',o.current_revision::text)

  FROM banking.review_item r JOIN banking.source_account s USING(entity_id,book_id,source_account_id)JOIN ledger.book bk USING(entity_id,book_id)
  JOIN LATERAL(SELECT a.observation_id
  FROM banking.review_assignment a
  WHERE a.entity_id=e AND a.book_id=b AND a.review_id=r.review_id
  ORDER BY a.observation_id LIMIT 1)a ON true
  JOIN banking.source_observation o ON o.entity_id=e AND o.book_id=b AND o.observation_id=a.observation_id
  JOIN banking.automatic_accounting_edition p ON p.entity_id=e AND p.book_id=b AND p.revision=policy

  WHERE r.entity_id=e AND r.book_id=b AND EXISTS(SELECT
  FROM jsonb_array_elements(p.configuration->'rules')rule
  WHERE(rule->>'source_account_id')::uuid=s.source_account_id)
  AND(r.state<>'posted' OR EXISTS(SELECT
  FROM banking.provider_review_exception x
  WHERE x.entity_id=e AND x.book_id=b AND x.review_id=r.review_id))
  AND EXISTS(SELECT
  FROM banking.source_permission sp
  WHERE sp.entity_id=e AND sp.book_id=b AND sp.source_account_id=s.source_account_id AND sp.principal_id=actor AND sp.capability='read'AND sp.revoked_at IS NULL)
  AND EXISTS(SELECT
  FROM banking.source_permission sp
  WHERE sp.entity_id=e AND sp.book_id=b AND sp.source_account_id=s.source_account_id AND sp.principal_id=actor AND sp.capability='review'AND sp.revoked_at IS NULL)
  $$;

CREATE FUNCTION banking.render_automatic_accounting_operation(o banking.automatic_accounting_operation)
  RETURNS jsonb
  LANGUAGE sql STABLE
  SET search_path=pg_catalog,pg_temp AS $$
  SELECT CASE WHEN o.operation_id IS NULL THEN
  NULL
  ELSE  jsonb_build_object('operation_id',o.operation_id,'state',o.state,'review_id',o.review_id,'journal_id',o.result->>'journal_id','reason',o.reason)
END
  $$;

CREATE FUNCTION banking.automatic_accounting_sources(actor uuid,e uuid,b uuid,configuration jsonb)
  RETURNS void
  LANGUAGE plpgsql
  SET search_path=pg_catalog,pg_temp AS $$DECLARE sid uuid;
  s banking.source_account;
BEGIN

  FOR sid IN SELECT DISTINCT(x->>'source_account_id')::uuid
  FROM jsonb_array_elements(configuration->'rules')x
  ORDER BY 1 LOOP
  s:=banking.automatic_accounting_source(actor,e,b,sid);

END LOOP
  ;

END $$;

CREATE OR REPLACE FUNCTION banking.get_automatic_accounting(actor uuid,e uuid,b uuid)
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$DECLARE res jsonb;
BEGIN

  PERFORM banking.automatic_accounting_authorize(actor,e,b,false);
  res:=banking.automatic_accounting_policy_wire(e,b);

  IF res->'policy'<>'null'::jsonb THEN
  PERFORM banking.automatic_accounting_sources(actor,e,b,res->'policy');

END IF;
  RETURN res;

END $$;

-- Historical operation metadata remains scoped to its own current actor/source permission.
-- The caller already holds the book lock; supported source-grant/state mutations
-- require that same book FOR UPDATE. Avoid acquiring historical sources out of UUID order.
CREATE FUNCTION banking.automatic_accounting_visible(actor uuid,e uuid,b uuid,op banking.automatic_accounting_operation)
  RETURNS boolean
  LANGUAGE sql STABLE
  SET search_path=pg_catalog,pg_temp AS $$
  SELECT coalesce(op.operation_id IS NOT NULL AND op.actor_id=actor AND op.entity_id=e AND op.book_id=b AND EXISTS(
  SELECT
  FROM banking.review_item r JOIN banking.source_account s USING(entity_id,book_id,source_account_id)

  WHERE r.entity_id=e AND r.book_id=b AND r.review_id=op.review_id AND s.source_account_id=op.source_account_id
  AND EXISTS(SELECT
  FROM banking.source_permission p
  WHERE p.entity_id=e AND p.book_id=b AND p.source_account_id=s.source_account_id AND p.principal_id=actor AND p.capability='read'AND p.revoked_at IS NULL)
  AND EXISTS(SELECT
  FROM banking.source_permission p
  WHERE p.entity_id=e AND p.book_id=b AND p.source_account_id=s.source_account_id AND p.principal_id=actor AND p.capability='review'AND p.revoked_at IS NULL)),false)
  $$;

CREATE FUNCTION banking.automatic_accounting_status(actor uuid,e uuid,b uuid)
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$
  DECLARE p banking.automatic_accounting_edition;
  last_op banking.automatic_accounting_operation;
  pending_value bigint:=0;
  attention_value bigint:=0;
  holds jsonb:='[]';
  blocked text;
BEGIN

  PERFORM banking.automatic_accounting_authorize(actor,e,b,false);

  SELECT x.* INTO p
  FROM banking.automatic_accounting_policy h JOIN banking.automatic_accounting_edition x USING(entity_id,book_id,revision)
  WHERE h.entity_id=e AND h.book_id=b;

  IF p.revision IS NOT NULL THEN

  PERFORM banking.automatic_accounting_sources(actor,e,b,p.configuration);

  IF EXISTS(SELECT
  FROM ledger.interactive_command_reservation
  WHERE principal_id=actor)THEN
  blocked:='saved_work';

  ELSIF  EXISTS(SELECT
  FROM banking.source_account s
  WHERE s.entity_id=e AND s.book_id=b AND s.state<>'active'AND EXISTS(SELECT
  FROM jsonb_array_elements(p.configuration->'rules')r
  WHERE(r->>'source_account_id')::uuid=s.source_account_id))THEN
  blocked:='source_unavailable';

END IF;

  SELECT * INTO last_op
  FROM banking.automatic_accounting_operation
  WHERE entity_id=e AND book_id=b
  ORDER BY created_at DESC,operation_id DESC LIMIT 1;

  IF NOT banking.automatic_accounting_visible(actor,e,b,last_op)THEN
  last_op:=NULL;

END IF;

  IF p.enabled THEN

  SELECT count(*) INTO pending_value
  FROM banking.automatic_accounting_operation o
  WHERE o.entity_id=e AND o.book_id=b AND o.state='prepared'AND o.policy_revision=p.revision AND banking.automatic_accounting_visible(actor,e,b,o);

  SELECT pending_value+count(*) INTO pending_value
  FROM banking.automatic_accounting_inputs(actor,e,b,p.revision)i
  WHERE NOT EXISTS(SELECT
  FROM banking.automatic_accounting_operation o
  WHERE o.entity_id=e AND o.book_id=b AND o.review_id=i.review_id AND(o.state='prepared'OR o.input_context=i.input_context));

  SELECT count(*) INTO attention_value
  FROM banking.automatic_accounting_inputs(actor,e,b,p.revision)i JOIN banking.automatic_accounting_operation o ON o.entity_id=e AND o.book_id=b AND o.review_id=i.review_id AND o.input_context=i.input_context AND o.actor_id=actor
  WHERE o.state='held';

  SELECT coalesce(jsonb_agg(jsonb_build_object('review_id',x.review_id,'source_account_id',x.source_account_id,'reason',x.reason,'operation_id',x.operation_id)
  ORDER BY x.review_id),'[]'::jsonb)INTO holds
  FROM(SELECT o.*
  FROM banking.automatic_accounting_inputs(actor,e,b,p.revision)i JOIN banking.automatic_accounting_operation o ON o.entity_id=e AND o.book_id=b AND o.review_id=i.review_id AND o.input_context=i.input_context AND o.actor_id=actor
  WHERE o.state='held'
  ORDER BY o.review_id LIMIT 100)x;

END IF;

END IF;

  RETURN jsonb_build_object('profile','automatic-accounting-status-v1','entity_id',e,'book_id',b,'enabled',coalesce(p.enabled,false),'pending_count',pending_value::text,'attention_count',attention_value::text,'blocked_reason',blocked,'last_operation',banking.render_automatic_accounting_operation(last_op),'holds',holds,'holds_truncated',attention_value>jsonb_array_length(holds));

END $$;

-- Work inventory rotates bounded book scopes in its own transaction, never grants posting authority.
CREATE FUNCTION banking.automatic_accounting_work()
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$DECLARE x record;
  items jsonb:='[]';
BEGIN

  IF current_setting('transaction_isolation')<>'serializable' THEN
  PERFORM banking.automatic_accounting_error('P2A00');

END IF;

  FOR x IN SELECT p.actor_id,p.entity_id,p.book_id
  FROM banking.automatic_accounting_policy h JOIN banking.automatic_accounting_edition p USING(entity_id,book_id,revision)LEFT JOIN banking.automatic_accounting_cursor c USING(entity_id,book_id)JOIN ledger.principal a ON a.principal_id=p.actor_id AND a.disabled_at IS NULL
  WHERE p.enabled
  ORDER BY coalesce(c.last_work_at,'-infinity'),p.entity_id,p.book_id LIMIT 10 LOOP

  INSERT INTO banking.automatic_accounting_cursor(entity_id,book_id,last_work_at)
  VALUES(x.entity_id,x.book_id,clock_timestamp())ON CONFLICT(entity_id,book_id)DO UPDATE SET last_work_at=EXCLUDED.last_work_at;

  items:=items||jsonb_build_array(jsonb_build_object('actor_id',x.actor_id,'entity_id',x.entity_id,'book_id',x.book_id));

END LOOP
  ;
  RETURN jsonb_build_object('profile','automatic-accounting-work-v1','items',items);

END $$;

-- Preparation selects current retained evidence; exact source matching is approved policy, not category inference.
CREATE FUNCTION banking.prepare_automatic_accounting(actor uuid,e uuid,b uuid)
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$
  DECLARE p banking.automatic_accounting_edition;
  op banking.automatic_accounting_operation;
  last_op banking.automatic_accounting_operation;
  x record;
  r banking.review_item;
  s banking.source_account;
  q banking.observation_revision;
  bp ledger.book_policy;
  a ledger.account;
  matches jsonb;
  rule jsonb;
  reason_value text;
  cursor_value uuid;
  draft jsonb;
  rev bigint;
  n integer;
  blocked text;

BEGIN

  PERFORM banking.automatic_accounting_authorize(actor,e,b,true);

  SELECT a.* INTO p
  FROM banking.automatic_accounting_policy h JOIN banking.automatic_accounting_edition a USING(entity_id,book_id,revision)
  WHERE h.entity_id=e AND h.book_id=b;

  IF p.revision IS NULL OR NOT p.enabled THEN
  RETURN jsonb_build_object('profile','automatic-accounting-next-v1','entity_id',e,'book_id',b,'state','idle','operation',NULL,'last_operation',NULL,'blocked_reason',NULL);

END IF;

  IF p.actor_id<>actor THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END IF;

  PERFORM banking.automatic_accounting_sources(actor,e,b,p.configuration);

  SELECT * INTO last_op
  FROM banking.automatic_accounting_operation
  WHERE entity_id=e AND book_id=b
  ORDER BY created_at DESC,operation_id DESC LIMIT 1;

  IF NOT banking.automatic_accounting_visible(actor,e,b,last_op)THEN
  last_op:=NULL;

END IF;

  IF EXISTS(SELECT
  FROM ledger.interactive_command_reservation
  WHERE principal_id=actor)THEN
  RETURN jsonb_build_object('profile','automatic-accounting-next-v1','entity_id',e,'book_id',b,'state','blocked','operation',NULL,'last_operation',banking.render_automatic_accounting_operation(last_op),'blocked_reason','saved_work');

END IF;

  SELECT * INTO op
  FROM banking.automatic_accounting_operation
  WHERE entity_id=e AND book_id=b AND state='prepared' FOR UPDATE;

  IF FOUND THEN

  IF op.policy_revision<>p.revision OR op.actor_id<>actor THEN
  UPDATE banking.automatic_accounting_operation SET state='stale',reason='stale_input',terminal_at=clock_timestamp()
  WHERE entity_id=e AND book_id=b AND operation_id=op.operation_id RETURNING * INTO op;

END IF;

  IF NOT banking.automatic_accounting_visible(actor,e,b,op)THEN
  RETURN jsonb_build_object('profile','automatic-accounting-next-v1','entity_id',e,'book_id',b,'state','idle','operation',NULL,'last_operation',banking.render_automatic_accounting_operation(last_op),'blocked_reason',NULL);

END IF;

  RETURN jsonb_build_object('profile','automatic-accounting-next-v1','entity_id',e,'book_id',b,'state',op.state,'operation',banking.render_automatic_accounting_operation(op),'last_operation',banking.render_automatic_accounting_operation(last_op),'blocked_reason',NULL);

END IF;

  SELECT after_review_id INTO cursor_value
  FROM banking.automatic_accounting_cursor
  WHERE entity_id=e AND book_id=b;

  FOR x IN SELECT i.*
  FROM banking.automatic_accounting_inputs(actor,e,b,p.revision)i
  ORDER BY CASE WHEN cursor_value IS NULL OR i.review_id>cursor_value THEN
  0
  ELSE  1
END,i.review_id LIMIT 100 LOOP

  INSERT INTO banking.automatic_accounting_cursor(entity_id,book_id,after_review_id)
  VALUES(e,b,x.review_id)ON CONFLICT(entity_id,book_id)DO UPDATE SET after_review_id=EXCLUDED.after_review_id;

  SELECT * INTO op
  FROM banking.automatic_accounting_operation
  WHERE entity_id=e AND book_id=b AND review_id=x.review_id AND input_hash=encode(sha256(convert_to(x.input_context::text,'UTF8')),'hex');

  IF FOUND THEN
  IF op.input_context<>x.input_context THEN
  PERFORM banking.automatic_accounting_error('P2A02');

END IF;
  CONTINUE;

END IF;

  SELECT * INTO r
  FROM banking.review_item
  WHERE entity_id=e AND book_id=b AND review_id=x.review_id FOR UPDATE;

  SELECT * INTO s
  FROM banking.source_account
  WHERE entity_id=e AND book_id=b AND source_account_id=x.source_account_id;

  SELECT * INTO q
  FROM banking.observation_revision
  WHERE entity_id=e AND book_id=b AND observation_id=x.observation_id AND revision=x.observation_revision;

  reason_value:=NULL;
  rule:=NULL;

  IF s.state<>'active' THEN
  reason_value:='source_unavailable';

  ELSIF  r.state IN('posted','correction_hold')OR EXISTS(SELECT
  FROM banking.provider_review_exception ex
  WHERE ex.entity_id=e AND ex.book_id=b AND ex.review_id=r.review_id)THEN
  reason_value:='source_correction_required';

  ELSIF  r.state='overlap_hold'OR(SELECT count(*)
  FROM banking.review_assignment ra
  WHERE ra.entity_id=e AND ra.book_id=b AND ra.review_id=r.review_id)<>1 THEN
  reason_value:='overlap_review_required';

  ELSIF  r.state='draft_ready'AND EXISTS(SELECT
  FROM banking.review_draft d
  WHERE d.entity_id=e AND d.book_id=b AND d.review_id=r.review_id AND d.review_revision=r.review_revision AND d.automatic_operation_id IS NULL)THEN
  reason_value:='manual_draft';

  ELSIF  q.source_state='held'OR q.booked_date IS NULL OR q.amount IS NULL OR q.currency_version_id IS NULL THEN
  reason_value:='source_held';

  ELSIF  q.source_state<>'booked'THEN
  reason_value:='source_not_booked';

  ELSIF  q.amount<=0 OR q.direction='zero'THEN
  reason_value:='zero_amount';

  ELSIF  q.booked_date<(p.configuration->>'effective_from')::date OR(p.configuration->'effective_to'<>'null'::jsonb AND q.booked_date>=(p.configuration->>'effective_to')::date)THEN
  reason_value:='outside_policy_interval';

  ELSE
  SELECT coalesce(jsonb_agg(z),'[]'::jsonb)INTO matches
  FROM jsonb_array_elements(p.configuration->'rules')z
  WHERE(z->>'source_account_id')::uuid=s.source_account_id AND z->>'direction'=q.direction AND z->>'description'=q.description;

  IF jsonb_array_length(matches)=0 THEN
  reason_value:='unmapped_rule';

  ELSIF  jsonb_array_length(matches)>1 THEN
  reason_value:='ambiguous_rule';

  ELSE
  rule:=matches->0;

  IF rule->>'treatment'IN('transfer','card_payment')THEN
  reason_value:='paired_transfer_required';

  ELSIF  rule->>'treatment'='loan'THEN
  reason_value:='split_treatment_required';

  ELSE
  SELECT * INTO bp
  FROM ledger.book_policy
  WHERE entity_id=e AND book_id=b AND policy_id=(p.configuration->>'book_policy_id')::uuid;

  IF NOT FOUND OR q.booked_date<bp.effective_from OR(bp.effective_to IS NOT NULL AND q.booked_date>=bp.effective_to)THEN
  reason_value:='unsupported_policy';

  ELSIF  q.currency_version_id<>bp.functional_currency_version_id THEN
  reason_value:='fx_required';

  ELSIF  q.currency_version_id<>s.currency_version_id OR(SELECT alphabetic_code
  FROM reference.currency
  WHERE currency_version_id=q.currency_version_id)<>'USD'THEN
  reason_value:='unsupported_currency';

  ELSIF  bp.basis<>'accrual'OR bp.timezone<>'UTC'OR bp.posting_scale<>2 OR bp.rounding_mode<>'half_even'OR bp.rate_policy_code<>'synthetic-reviewed-spot-v1'THEN
  reason_value:='unsupported_policy';

  ELSIF  NOT EXISTS(SELECT
  FROM ledger.period
  WHERE entity_id=e AND book_id=b AND starts_on<=q.booked_date AND ends_before>q.booked_date AND state='open')THEN
  reason_value:='closed_period';

  ELSE
  SELECT * INTO a
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id=(rule->>'offset_account_id')::uuid AND active_from<=q.booked_date AND(active_to IS NULL OR active_to>q.booked_date);

  IF NOT FOUND OR NOT EXISTS(SELECT
  FROM ledger.account
  WHERE entity_id=e AND book_id=b AND account_id=s.ledger_account_id AND active_from<=q.booked_date AND(active_to IS NULL OR active_to>q.booked_date))THEN
  reason_value:='inactive_account';

END IF;

END IF;

END IF;

END IF;

END IF;

  INSERT INTO banking.automatic_accounting_operation(entity_id,book_id,actor_id,policy_revision,review_id,source_account_id,observation_id,observation_revision,input_context,input_hash,rule_id,state,reason,terminal_at)

  VALUES(e,b,actor,p.revision,r.review_id,s.source_account_id,q.observation_id,q.revision,x.input_context,encode(sha256(convert_to(x.input_context::text,'UTF8')),'hex'),(rule->>'rule_id')::uuid,CASE WHEN reason_value IS NULL THEN
  'prepared'
  ELSE  'held'
END,reason_value,CASE WHEN reason_value IS NOT NULL THEN
  clock_timestamp()
  ELSE  NULL
END)RETURNING * INTO op;

  IF reason_value IS NULL THEN

  SELECT revision INTO rev
  FROM ledger.book
  WHERE entity_id=e AND book_id=b;

  draft:=banking.create_source_draft(actor,e,b,r.review_id,'automatic:'||op.operation_id,jsonb_build_object('expected_review_revision',r.review_revision::text,'source_revision',s.source_revision::text,'binding_revision',s.binding_revision::text,'expected_book_revision',rev::text,'category_account_id',rule->>'offset_account_id','policy_id',p.configuration->>'book_policy_id','accounting_date',to_char(q.booked_date,'YYYY-MM-DD'),'description',q.description)::text,op.operation_id);

END IF;

  RETURN jsonb_build_object('profile','automatic-accounting-next-v1','entity_id',e,'book_id',b,'state',op.state,'operation',banking.render_automatic_accounting_operation(op),'last_operation',banking.render_automatic_accounting_operation(last_op),'blocked_reason',NULL);

END LOOP
  ;

  RETURN jsonb_build_object('profile','automatic-accounting-next-v1','entity_id',e,'book_id',b,'state','idle','operation',NULL,'last_operation',banking.render_automatic_accounting_operation(last_op),'blocked_reason',NULL);

END $$;

-- Exact durable identity owns recovery; a successful financial effect and its terminal receipt are atomic.
CREATE FUNCTION banking.execute_automatic_accounting(actor uuid,e uuid,b uuid,id uuid)
  RETURNS jsonb
  LANGUAGE plpgsql SECURITY DEFINER
  SET search_path=pg_catalog,pg_temp AS $$
  DECLARE op banking.automatic_accounting_operation;
  p banking.automatic_accounting_edition;
  d banking.review_draft;
  r banking.review_item;
  s banking.source_account;
  rev bigint;
  current_observation bigint;
  value jsonb;
  body jsonb;
  blocked text;
  reason_value text;

BEGIN

  PERFORM banking.automatic_accounting_authorize(actor,e,b,true);

  SELECT * INTO op
  FROM banking.automatic_accounting_operation
  WHERE entity_id=e AND book_id=b AND operation_id=id AND actor_id=actor FOR UPDATE;

  IF NOT FOUND THEN
  PERFORM banking.automatic_accounting_error('P2A01');

END IF;

  s:=banking.automatic_accounting_source(actor,e,b,op.source_account_id);

  IF op.state='prepared'THEN

  SELECT x.* INTO p
  FROM banking.automatic_accounting_policy h JOIN banking.automatic_accounting_edition x USING(entity_id,book_id,revision)
  WHERE h.entity_id=e AND h.book_id=b;

  IF EXISTS(SELECT
  FROM ledger.interactive_command_reservation
  WHERE principal_id=actor)THEN
  blocked:='saved_work';

  ELSE
  SELECT * INTO d
  FROM banking.review_draft
  WHERE entity_id=e AND book_id=b AND automatic_operation_id=id;

  SELECT * INTO r
  FROM banking.review_item
  WHERE entity_id=e AND book_id=b AND review_id=op.review_id;

  SELECT revision INTO rev
  FROM ledger.book
  WHERE entity_id=e AND book_id=b;

  SELECT current_revision INTO current_observation
  FROM banking.source_observation
  WHERE entity_id=e AND book_id=b AND observation_id=op.observation_id;

  IF p.revision IS DISTINCT
  FROM op.policy_revision OR NOT coalesce(p.enabled,false)OR p.actor_id<>actor OR d.draft_id IS NULL OR s.state<>'active'OR current_observation<>op.observation_revision OR ROW(r.review_revision,s.source_revision,s.binding_revision,rev)IS DISTINCT
  FROM ROW(d.review_revision,d.source_revision,d.binding_revision,d.book_revision)OR r.state<>'draft_ready'OR EXISTS(SELECT
  FROM banking.review_posting
  WHERE entity_id=e AND book_id=b AND review_id=r.review_id)THEN

  UPDATE banking.automatic_accounting_operation SET state='stale',reason='stale_input',terminal_at=clock_timestamp()
  WHERE entity_id=e AND book_id=b AND operation_id=id RETURNING * INTO op;

  ELSE
  body:=jsonb_build_object('review_id',d.review_id,'draft_id',d.draft_id,'review_revision',d.review_revision::text,'source_revision',d.source_revision::text,'binding_revision',d.binding_revision::text,'expected_book_revision',d.book_revision::text,'posting_body',d.posting_body);

BEGIN

  value:=banking.commit_source_post(actor,e,b,NULL,id,id,body);

  UPDATE banking.automatic_accounting_operation SET state='committed',result=value,terminal_at=clock_timestamp()
  WHERE entity_id=e AND book_id=b AND operation_id=id RETURNING * INTO op;

  EXCEPTION
  WHEN SQLSTATE 'P2105' THEN
  reason_value:='closed_period';

  WHEN SQLSTATE 'P2109' THEN
  reason_value:='unsupported_policy';

  WHEN SQLSTATE 'P2503' THEN
  reason_value:='stale_input';

  WHEN SQLSTATE 'P2504' THEN
  reason_value:='fx_required';

  WHEN SQLSTATE 'P2107' OR SQLSTATE 'P2108' THEN
  reason_value:='unsupported_currency';

END;

  IF reason_value IS NOT NULL THEN
  UPDATE banking.automatic_accounting_operation SET state=CASE WHEN reason_value='stale_input'THEN
  'stale'
  ELSE  'held'
END,reason=reason_value,terminal_at=clock_timestamp()
  WHERE entity_id=e AND book_id=b AND operation_id=id RETURNING * INTO op;

END IF;

END IF;

END IF;

END IF;

  RETURN jsonb_build_object('profile','automatic-accounting-execution-v1','entity_id',e,'book_id',b,'operation',banking.render_automatic_accounting_operation(op),'blocked_reason',blocked);

END $$;

-- Runtime roles receive only closed operations; financial internals retain owner-only execution.
REVOKE ALL ON banking.automatic_accounting_policy,banking.automatic_accounting_edition,banking.automatic_accounting_setting,banking.automatic_accounting_operation,banking.automatic_accounting_cursor
  FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;

REVOKE ALL ON FUNCTION banking.automatic_accounting_error(text),banking.automatic_accounting_guard(),banking.automatic_accounting_authorize(uuid,uuid,uuid,boolean),banking.automatic_accounting_source(uuid,uuid,uuid,uuid),banking.automatic_accounting_sources(uuid,uuid,uuid,jsonb),banking.automatic_accounting_policy_wire(uuid,uuid),banking.create_source_draft(uuid,uuid,uuid,uuid,text,text,uuid),banking.commit_source_post(uuid,uuid,uuid,uuid,uuid,uuid,jsonb),banking.authorize_source_post_core(uuid,uuid,uuid,jsonb,uuid),banking.automatic_accounting_inputs(uuid,uuid,uuid,bigint),banking.automatic_accounting_visible(uuid,uuid,uuid,banking.automatic_accounting_operation),banking.render_automatic_accounting_operation(banking.automatic_accounting_operation)
  FROM PUBLIC,bookkeeping_api,bookkeeping_observer,bookkeeping_publisher,bookkeeping_sync;

GRANT EXECUTE ON FUNCTION banking.get_automatic_accounting(uuid,uuid,uuid),banking.set_automatic_accounting(uuid,uuid,uuid,text),banking.automatic_accounting_status(uuid,uuid,uuid),banking.automatic_accounting_work(),banking.prepare_automatic_accounting(uuid,uuid,uuid),banking.execute_automatic_accounting(uuid,uuid,uuid,uuid)TO bookkeeping_api;

COMMIT;

