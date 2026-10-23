-- Owner-operated platform login wiring only; no business mandate, financial read or provider operation.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='10s';
SET LOCAL lock_timeout='2s';
SET LOCAL idle_in_transaction_session_timeout='15s';
SELECT set_config('jayburd.verifier_project', :'project_id', true);
DO $$
DECLARE login_value name:='jayburd-dev-verifier@jayburd-bookkeeping-dev.iam';
BEGIN
 IF current_setting('jayburd.verifier_project')<>'jayburd-bookkeeping-dev'
 OR current_database()<>'bookkeeping' OR session_user<>current_user
 OR NOT pg_has_role(session_user,'bookkeeping_owner','SET')
 OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolcanlogin AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_verifier' AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 THEN RAISE EXCEPTION 'verifier_bootstrap_rejected'; END IF;
 -- Never repair an unsafe existing grant/delegation silently, including another grantor's grant.
 IF EXISTS(SELECT FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.member
 WHERE r.rolname IN (login_value,'bookkeeping_verifier') AND (m.admin_option OR m.set_option))
 THEN RAISE EXCEPTION 'verifier_bootstrap_rejected'; END IF;
 EXECUTE format('GRANT bookkeeping_verifier TO %I WITH ADMIN FALSE, INHERIT TRUE, SET FALSE',login_value);
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I',current_database(),login_value);
 EXECUTE format('ALTER ROLE %I SET search_path=pg_catalog',login_value);
 EXECUTE format('ALTER ROLE %I SET statement_timeout=''3s''',login_value);
 EXECUTE format('ALTER ROLE %I SET lock_timeout=''1s''',login_value);
 EXECUTE format('ALTER ROLE %I SET idle_in_transaction_session_timeout=''5s''',login_value);
 EXECUTE format('ALTER ROLE %I CONNECTION LIMIT 1',login_value);
END $$;
SET LOCAL ROLE bookkeeping_owner;
-- Reuse the reviewed complete effective-ACL/member safety predicate; any failure rolls back every grant.
DO $$ BEGIN
 IF NOT has_database_privilege('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam',current_database(),'CONNECT')
 OR NOT integration.verifier_login('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam') THEN
 RAISE EXCEPTION 'verifier_bootstrap_rejected'; END IF;
END $$;
COMMIT;
