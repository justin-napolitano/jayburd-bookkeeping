-- Explicit owner-operated setup after platform provisioning; identifiers only, no credential discovery.
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL statement_timeout='10s';
SET LOCAL lock_timeout='2s';
SET LOCAL idle_in_transaction_session_timeout='15s';
SELECT set_config('jayburd.webhook_project', :'project_id', true),
       set_config('jayburd.webhook_key', :'operation_key', true),
       set_config('jayburd.webhook_enabled', :'enabled', true);
DO $$
DECLARE project_value text:=current_setting('jayburd.webhook_project');
        login_value text;
        probe_role text;
        capability text:='bookkeeping_webhook';
BEGIN
 IF project_value<>'jayburd-bookkeeping-dev' OR current_database()<>'bookkeeping'
 OR current_setting('jayburd.webhook_key')!~'^[A-Za-z0-9._:-]{1,128}$'
 OR current_setting('jayburd.webhook_enabled') NOT IN ('true','false')
 OR session_user<>current_user OR NOT pg_has_role(session_user,'bookkeeping_owner','SET') THEN
  RAISE EXCEPTION 'webhook_bootstrap_rejected';
 END IF;
 login_value:='jayburd-dev-webhook@'||project_value||'.iam';
 IF login_value=session_user
 OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname=login_value AND rolcanlogin AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication))
 OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname=capability AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication)) THEN
  RAISE EXCEPTION 'webhook_bootstrap_rejected';
 END IF;
 -- Check every grantor and transitive membership; never silently repair delegated/mixed authority.
 IF EXISTS(SELECT FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.roleid JOIN pg_roles u ON u.oid=m.member WHERE u.rolname IN (login_value,capability) AND (m.admin_option OR m.set_option))
 OR EXISTS(SELECT FROM pg_roles WHERE rolname<>capability AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication OR rolname LIKE 'bookkeeping_%' OR rolname LIKE 'pg_%' OR rolname='cloudsqlsuperuser') AND (pg_has_role(login_value,oid,'MEMBER') OR pg_has_role(capability,oid,'MEMBER')))
 THEN RAISE EXCEPTION 'webhook_bootstrap_rejected';END IF;
 FOREACH probe_role IN ARRAY ARRAY[login_value,capability] LOOP
 IF has_database_privilege(probe_role,current_database(),'CREATE,TEMP,CONNECT WITH GRANT OPTION')
 OR EXISTS(SELECT FROM pg_namespace n WHERE n.nspname !~ '^pg_temp_' AND has_schema_privilege(probe_role,n.oid,'CREATE,USAGE WITH GRANT OPTION'))
 OR EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','reporting','deployment') AND c.relkind IN ('r','p','v','m','f') AND has_table_privilege(probe_role,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN'))
 OR EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','reporting','deployment') AND CASE WHEN c.relkind='S' THEN has_sequence_privilege(probe_role,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)
 OR EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname IN ('ledger','reference','audit','integration','banking','reporting','deployment') AND (has_function_privilege(probe_role,p.oid,'EXECUTE WITH GRANT OPTION') OR NOT (n.nspname='integration' AND p.proname='accept_verified_webhook' AND p.proargtypes='25 25 25'::oidvector) AND has_function_privilege(probe_role,p.oid,'EXECUTE'))) THEN
  RAISE EXCEPTION 'webhook_bootstrap_rejected';
 END IF;
 END LOOP;
 -- Recheck direct ACLs too: only intended inherited function/schema capability is permitted.
 IF EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace CROSS JOIN LATERAL aclexplode(p.proacl) a JOIN pg_roles r ON r.oid=a.grantee WHERE r.rolname=login_value AND n.nspname IN ('ledger','reference','audit','integration','banking','reporting','deployment'))
 OR EXISTS(SELECT FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a JOIN pg_roles r ON r.oid=a.grantee WHERE r.rolname=login_value AND a.is_grantable)
 OR EXISTS(SELECT FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a JOIN pg_roles r ON r.oid=a.grantee WHERE r.rolname=login_value AND a.is_grantable) THEN
  RAISE EXCEPTION 'webhook_bootstrap_rejected';
 END IF;
 EXECUTE format('GRANT %I TO %I WITH ADMIN FALSE, INHERIT TRUE, SET FALSE',capability,login_value);
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I',current_database(),login_value);
 EXECUTE format('ALTER ROLE %I SET search_path=pg_catalog',login_value);
 -- Maximum old/new revisions: each one instance, each pool capped at two.
 EXECUTE format('ALTER ROLE %I CONNECTION LIMIT 4',login_value);
END $$;
SET LOCAL ROLE bookkeeping_owner;
SELECT integration.set_webhook_ingress(('jayburd-dev-webhook@'||current_setting('jayburd.webhook_project')||'.iam')::name,'sandbox',current_setting('jayburd.webhook_key'),current_setting('jayburd.webhook_enabled')::boolean);
-- Historical receipt replay must not claim that a later changed ingress still has this desired state.
DO $$ BEGIN
 PERFORM 1 FROM integration.webhook_ingress WHERE environment='sandbox'
 AND login_name=('jayburd-dev-webhook@'||current_setting('jayburd.webhook_project')||'.iam')::name
 AND enabled=current_setting('jayburd.webhook_enabled')::boolean FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'webhook_bootstrap_rejected';END IF;
END $$;
COMMIT;
