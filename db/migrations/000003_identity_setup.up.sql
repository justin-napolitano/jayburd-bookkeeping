-- Controlled identity enrollment and atomic development-book setup (#6).
BEGIN;
DO $$ BEGIN
 IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_identity_operator') THEN CREATE ROLE bookkeeping_identity_operator NOLOGIN; END IF;
 IF EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_identity_operator' AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolcanlogin))
 OR pg_has_role('bookkeeping_api','bookkeeping_identity_operator','MEMBER')
 OR pg_has_role('bookkeeping_identity_operator','bookkeeping_owner','MEMBER')
 OR pg_has_role('bookkeeping_identity_operator','bookkeeping_api','MEMBER') THEN RAISE EXCEPTION 'unsafe identity roles'; END IF;
END $$;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE audit.identity_action (
 action_id uuid PRIMARY KEY, principal_id uuid NOT NULL REFERENCES ledger.principal,
 operation text NOT NULL CHECK(operation='enroll'), database_actor text NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE ledger.setup_receipt (
 principal_id uuid NOT NULL REFERENCES ledger.principal,
 idempotency_key text NOT NULL, normalized_request jsonb NOT NULL,
 result jsonb NOT NULL, entity_id uuid NOT NULL, book_id uuid NOT NULL,
 request_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(principal_id,idempotency_key), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON audit.identity_action FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON ledger.setup_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();

-- Only a separately granted operator can enroll; retry never reactivates a user.
CREATE FUNCTION ledger.enroll_principal(issuer_value text,subject_value text) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE p ledger.principal; new_id uuid:=gen_random_uuid();
BEGIN
 IF issuer_value IS NULL OR issuer_value !~ '^https://securetoken.google.com/[a-z][a-z0-9-]{4,28}[a-z0-9]$'
 OR subject_value IS NULL OR length(subject_value) NOT BETWEEN 1 AND 128 OR subject_value ~ '[[:cntrl:]]' THEN PERFORM ledger.fail('P2100'); END IF;
 INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES(new_id,issuer_value,subject_value) ON CONFLICT(issuer,subject) DO NOTHING;
 SELECT * INTO p FROM ledger.principal WHERE issuer=issuer_value AND subject=subject_value FOR UPDATE;
 IF NOT FOUND OR p.disabled_at IS NOT NULL THEN PERFORM ledger.fail('P2101'); END IF;
 IF p.principal_id=new_id THEN
  INSERT INTO audit.identity_action(action_id,principal_id,operation,database_actor) VALUES(gen_random_uuid(),new_id,'enroll',session_user);
 END IF;
 RETURN p.principal_id;
END $$;

-- Resolve verified identity only; no email matching, enrollment or permissions.
CREATE FUNCTION ledger.resolve_principal(issuer_value text,subject_value text) RETURNS uuid
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $$
 SELECT principal_id FROM ledger.principal WHERE issuer=issuer_value AND subject=subject_value AND disabled_at IS NULL
$$;

-- Bound setup text without trimming or changing replay semantics.
CREATE FUNCTION ledger.setup_text(v jsonb,max_length integer) RETURNS text
LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $$
DECLARE s text:=ledger.text_value(v);
BEGIN
 IF length(s) NOT BETWEEN 1 AND max_length OR btrim(s)='' OR s ~ '[[:cntrl:]]' THEN PERFORM ledger.fail('P2100'); END IF;
 RETURN s;
END $$;

-- One new entity/book bundle, with no claims on existing entities or opening balances.
CREATE FUNCTION ledger.setup_book(actor uuid,k text,request uuid,raw text)
RETURNS TABLE(http_status smallint,body jsonb,replayed boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE v jsonb; a jsonb; norm jsonb; receipt ledger.setup_receipt; fc reference.currency;
 e uuid:=gen_random_uuid(); b uuid:=gen_random_uuid(); p uuid:=gen_random_uuid(); period_id_value uuid:=gen_random_uuid();
 account_id_value uuid; account_result jsonb:='[]'; result jsonb; first_day date; last_day date;
 currency_id uuid; entity_name_value text; kind_value text; book_name_value text; code_value text; name_value text;
 category_value text; side_value text; n integer; rev bigint;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR actor IS NULL OR request IS NULL OR k IS NULL OR raw IS NULL THEN PERFORM ledger.fail('P2100'); END IF;
 IF octet_length(raw)>1048576 THEN PERFORM ledger.fail('P2111'); END IF;
 IF length(k) NOT BETWEEN 1 AND 128 OR k !~ '^[A-Za-z0-9._:-]+$' OR NOT(raw IS JSON OBJECT WITH UNIQUE KEYS) THEN PERFORM ledger.fail('P2100'); END IF;
 BEGIN v:=raw::jsonb; EXCEPTION WHEN untranslatable_character THEN PERFORM ledger.fail('P2100'); END;
 PERFORM ledger.keys(v,ARRAY['entity_name','entity_kind','book_name','starts_on','ends_before','functional_currency_version_id','accounts']);
 entity_name_value:=ledger.setup_text(v->'entity_name',200); kind_value:=ledger.text_value(v->'entity_kind'); book_name_value:=ledger.setup_text(v->'book_name',200);
 IF kind_value NOT IN ('legal','personal') OR jsonb_typeof(v->'accounts') IS DISTINCT FROM 'array' THEN PERFORM ledger.fail('P2100'); END IF;
 n:=jsonb_array_length(v->'accounts'); IF n NOT BETWEEN 2 AND 100 THEN PERFORM ledger.fail('P2100'); END IF;
 first_day:=ledger.date_value(ledger.text_value(v->'starts_on')); last_day:=ledger.date_value(ledger.text_value(v->'ends_before'));
 IF first_day>=last_day THEN PERFORM ledger.fail('P2100'); END IF;
 currency_id:=ledger.uuid_value(v->'functional_currency_version_id'); norm:=jsonb_set(v,'{functional_currency_version_id}',to_jsonb(currency_id));
 -- Per-principal lock serializes setup replay and local disabling.
 PERFORM 1 FROM ledger.principal WHERE principal_id=actor AND disabled_at IS NULL FOR UPDATE;
 IF NOT FOUND THEN PERFORM ledger.fail('P2101'); END IF;
 SELECT * INTO receipt FROM ledger.setup_receipt WHERE principal_id=actor AND idempotency_key=k;
 IF FOUND THEN
  PERFORM ledger.authorize(actor,receipt.entity_id,receipt.book_id,'read',true);
  PERFORM ledger.authorize(actor,receipt.entity_id,receipt.book_id,'post',true);
  IF receipt.normalized_request<>norm THEN PERFORM ledger.fail('P2104'); END IF;
  RETURN QUERY SELECT 201::smallint,receipt.result,true; RETURN;
 END IF;
 SELECT * INTO fc FROM reference.currency WHERE currency_version_id=currency_id;
 IF NOT FOUND OR fc.alphabetic_code<>'USD' OR fc.minor_unit IS DISTINCT FROM 2 OR fc.valid_from>first_day OR (fc.valid_to IS NOT NULL AND fc.valid_to<last_day) THEN PERFORM ledger.fail('P2109'); END IF;
 INSERT INTO ledger.entity(entity_id,name,kind) VALUES(e,entity_name_value,kind_value);
 INSERT INTO ledger.membership(entity_id,principal_id,permission) VALUES(e,actor,'read'),(e,actor,'post');
 INSERT INTO ledger.book(entity_id,book_id,name) VALUES(e,b,book_name_value);
 INSERT INTO ledger.book_policy VALUES(e,b,p,first_day,last_day,currency_id,'accrual','UTC',2,'half_even','synthetic-reviewed-spot-v1',actor,clock_timestamp());
 INSERT INTO ledger.period(entity_id,book_id,period_id,starts_on,ends_before) VALUES(e,b,period_id_value,first_day,last_day);
 FOR a IN SELECT value FROM jsonb_array_elements(v->'accounts') LOOP
  PERFORM ledger.keys(a,ARRAY['code','name','category','normal_side','monetary']);
  code_value:=ledger.setup_text(a->'code',64); name_value:=ledger.setup_text(a->'name',200);
  category_value:=ledger.text_value(a->'category'); side_value:=ledger.text_value(a->'normal_side');
  IF category_value NOT IN ('asset','liability','equity','revenue','expense') OR side_value NOT IN ('debit','credit') OR jsonb_typeof(a->'monetary') IS DISTINCT FROM 'boolean' THEN PERFORM ledger.fail('P2100'); END IF;
  account_id_value:=gen_random_uuid();
  INSERT INTO ledger.account VALUES(e,b,account_id_value,code_value,name_value,category_value,side_value,(a->>'monetary')::boolean,first_day,NULL);
  account_result:=account_result||jsonb_build_array(jsonb_build_object('account_id',account_id_value,'code',code_value));
 END LOOP;
 SELECT revision INTO rev FROM ledger.book WHERE entity_id=e AND book_id=b;
 result:=jsonb_build_object('entity_id',e,'book_id',b,'policy_id',p,'period_id',period_id_value,'book_revision',rev::text,'accounts',account_result);
 INSERT INTO ledger.setup_receipt(principal_id,idempotency_key,normalized_request,result,entity_id,book_id,request_id) VALUES(actor,k,norm,result,e,b,request);
 RETURN QUERY SELECT 201::smallint,result,false;
EXCEPTION WHEN unique_violation THEN PERFORM ledger.fail('P2100');
END $$;
GRANT USAGE ON SCHEMA ledger TO bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION ledger.enroll_principal(text,text) TO bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION ledger.resolve_principal(text,text),ledger.setup_book(uuid,text,uuid,text) TO bookkeeping_api;
COMMIT;
