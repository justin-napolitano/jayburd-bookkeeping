-- Owner-operated setup after existing Cloud SQL IAM login/capability provisioning.
-- Identifier-only publisher_user/sync_user inputs; no password or enrollment side effects.
\set ON_ERROR_STOP on
BEGIN;
SELECT set_config('jayburd.publisher_user', :'publisher_user', true),
       set_config('jayburd.sync_user', :'sync_user', true);
DO $$
DECLARE publisher_name text := current_setting('jayburd.publisher_user');
        sync_name text := current_setting('jayburd.sync_user');
        login_name text;
        capability text;
BEGIN
 IF publisher_name=sync_name THEN RAISE EXCEPTION 'unsafe worker login'; END IF;
 -- Reject unsafe roles before granting either capability so failure is atomic.
 FOREACH login_name IN ARRAY ARRAY[publisher_name,sync_name] LOOP
  capability:=CASE WHEN login_name=publisher_name THEN 'bookkeeping_publisher' ELSE 'bookkeeping_sync' END;
  IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname=login_name AND rolcanlogin AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication))
   OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname=capability AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication))
   OR EXISTS(SELECT FROM pg_auth_members m JOIN pg_roles r ON r.oid=m.roleid JOIN pg_roles u ON u.oid=m.member WHERE u.rolname=login_name AND (m.admin_option OR (r.rolname=capability AND m.set_option)))
   OR EXISTS(SELECT FROM pg_roles WHERE rolname<>capability AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication OR rolname LIKE 'bookkeeping_%' OR rolname='cloudsqlsuperuser') AND pg_has_role(login_name,oid,'MEMBER'))
   OR EXISTS(SELECT FROM pg_roles WHERE rolname<>capability AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication OR rolname LIKE 'bookkeeping_%' OR rolname='cloudsqlsuperuser') AND pg_has_role(capability,oid,'MEMBER')) THEN
   RAISE EXCEPTION 'unsafe worker authority';
  END IF;
 END LOOP;
 FOREACH login_name IN ARRAY ARRAY[publisher_name,sync_name] LOOP
  capability:=CASE WHEN login_name=publisher_name THEN 'bookkeeping_publisher' ELSE 'bookkeeping_sync' END;
  EXECUTE format('GRANT %I TO %I WITH ADMIN FALSE, INHERIT TRUE, SET FALSE',capability,login_name);
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I',current_database(),login_name);
  EXECUTE format('ALTER ROLE %I SET search_path = pg_catalog',login_name);
  -- Two connections per instance with at most old/new revision overlap.
  EXECUTE format('ALTER ROLE %I CONNECTION LIMIT 4',login_name);
 END LOOP;
END $$;
COMMIT;
