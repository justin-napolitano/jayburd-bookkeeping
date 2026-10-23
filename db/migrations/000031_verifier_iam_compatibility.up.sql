-- Exact managed IAM marker exception; all effective authority checks remain intact.
-- Contract: docs/contracts/verifier-iam-compatibility.md
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE OR REPLACE FUNCTION integration.verifier_login(login_value name) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolname='jayburd-dev-verifier@jayburd-bookkeeping-dev.iam' AND rolcanlogin AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 AND EXISTS(SELECT FROM pg_roles WHERE rolname='bookkeeping_verifier' AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 AND pg_has_role(login_value,'bookkeeping_verifier','USAGE') AND NOT pg_has_role(login_value,'bookkeeping_verifier','SET')
 AND NOT EXISTS(SELECT FROM pg_roles WHERE rolname<>'bookkeeping_verifier' AND (rolname LIKE 'bookkeeping_%' OR rolname LIKE 'pg_%' OR rolname='cloudsqlsuperuser' OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND (pg_has_role(login_value,oid,'MEMBER') OR pg_has_role('bookkeeping_verifier',oid,'MEMBER')))
 AND NOT EXISTS(SELECT FROM pg_auth_members a JOIN pg_roles r ON r.oid=a.member WHERE r.rolname IN (login_value,'bookkeeping_verifier') AND (a.admin_option OR (a.set_option AND NOT (
   r.rolname=login_value AND a.inherit_option
   AND a.roleid IN (SELECT oid FROM pg_roles WHERE rolname='cloudsqliamserviceaccount'
     AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
   AND NOT EXISTS(SELECT FROM pg_auth_members parent WHERE parent.member=a.roleid)
  ))))
 AND NOT has_database_privilege(login_value,current_database(),'CREATE,TEMP,CONNECT WITH GRANT OPTION')
 AND NOT EXISTS(SELECT FROM pg_namespace n WHERE has_schema_privilege(login_value,n.oid,'CREATE,USAGE WITH GRANT OPTION'))
 AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment') AND c.relkind IN ('r','p','v','m','f') AND (has_table_privilege(login_value,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege(login_value,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 AND NOT EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment') AND CASE WHEN c.relkind='S' THEN has_sequence_privilege(login_value,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)
 AND NOT EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment') AND (has_function_privilege(login_value,p.oid,'EXECUTE WITH GRANT OPTION') OR p.oid<>to_regprocedure('integration.verify_sandbox_fixture(uuid,bigint,text,text)') AND has_function_privilege(login_value,p.oid,'EXECUTE')))
 AND NOT EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace CROSS JOIN LATERAL aclexplode(p.proacl) a JOIN pg_roles r ON r.oid=a.grantee WHERE r.rolname=login_value AND n.nspname IN ('ledger','reference','audit','banking','integration','reporting','deployment'))
$$;
COMMIT;
