-- Owner-operated shared-instance Production bootstrap; exact database/logins only.
-- psql variables api_user/migrator_user are identifiers supplied separately by owner.
\set ON_ERROR_STOP on
BEGIN;
-- Shared-instance preconditions precede any grants or capability creation.
DO $$
DECLARE owner_name text;
BEGIN
 IF current_database() <> 'bookkeeping_prod' THEN RAISE EXCEPTION 'shared Production database rejected'; END IF;
 SELECT r.rolname INTO owner_name FROM pg_database d JOIN pg_roles r ON r.oid=d.datdba WHERE d.datname=current_database();
 IF owner_name LIKE 'bookkeeping_%' OR owner_name LIKE 'jayburd-%' THEN RAISE EXCEPTION 'shared Production owner rejected'; END IF;
END $$;
SELECT set_config('jayburd.api_user', :'api_user', true), set_config('jayburd.migrator_user', :'migrator_user', true);
DO $$
DECLARE api_name text := current_setting('jayburd.api_user');
        migrator_name text := current_setting('jayburd.migrator_user');
        role_name text;
BEGIN
 IF api_name <> 'jayburd-prod-api@jayburd-bookkeeping-dev.iam' OR migrator_name <> 'jayburd-prod-migrator@jayburd-bookkeeping-dev.iam' THEN
  RAISE EXCEPTION 'shared Production login rejected';
 END IF;
 IF api_name=migrator_name OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname=api_name AND rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolbypassrls AND NOT rolreplication)
 OR NOT EXISTS(SELECT FROM pg_roles WHERE rolname=migrator_name AND rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolbypassrls AND NOT rolreplication) THEN
  RAISE EXCEPTION 'unsafe IAM login roles';
 END IF;
 IF EXISTS(SELECT FROM pg_roles WHERE (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication OR rolname='cloudsqlsuperuser') AND (pg_has_role(api_name,oid,'MEMBER') OR pg_has_role(migrator_name,oid,'MEMBER'))) THEN RAISE EXCEPTION 'unsafe inherited authority'; END IF;
 -- Capability roles exist before restricted migrations; no worker login or binding is provisioned.
 FOREACH role_name IN ARRAY ARRAY['bookkeeping_owner','bookkeeping_api','bookkeeping_identity_operator','bookkeeping_publisher','bookkeeping_observer','bookkeeping_sync','bookkeeping_webhook','bookkeeping_verifier'] LOOP
  IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname=role_name) THEN
   EXECUTE format('CREATE ROLE %I NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS',role_name);
  END IF;
  IF EXISTS(SELECT FROM pg_roles WHERE rolname=role_name AND (rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)) THEN RAISE EXCEPTION 'unsafe bookkeeping role'; END IF;
 END LOOP;
 -- Reject existing privilege inheritance instead of silently downgrading an unsafe account.
 IF pg_has_role(api_name,'bookkeeping_owner','MEMBER') OR pg_has_role(api_name,'bookkeeping_identity_operator','MEMBER')
 OR pg_has_role(api_name,'bookkeeping_publisher','MEMBER') OR pg_has_role(api_name,'bookkeeping_observer','MEMBER') OR pg_has_role(api_name,'bookkeeping_sync','MEMBER') OR pg_has_role(api_name,'bookkeeping_webhook','MEMBER') OR pg_has_role(api_name,'bookkeeping_verifier','MEMBER') THEN RAISE EXCEPTION 'unsafe runtime membership'; END IF;
 EXECUTE format('GRANT bookkeeping_owner TO %I WITH INHERIT FALSE, SET TRUE',migrator_name);
 EXECUTE format('GRANT bookkeeping_api TO %I WITH INHERIT TRUE, SET FALSE',api_name);
 EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC',current_database());
 EXECUTE format('GRANT CONNECT, CREATE ON DATABASE %I TO %I',current_database(),migrator_name);
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I',current_database(),api_name);
 REVOKE CREATE ON SCHEMA public FROM PUBLIC;
 CREATE SCHEMA IF NOT EXISTS deployment;
 REVOKE ALL ON SCHEMA deployment FROM PUBLIC;
 EXECUTE format('GRANT USAGE, CREATE ON SCHEMA deployment TO %I',migrator_name);
 EXECUTE format('ALTER ROLE %I SET search_path = pg_catalog',api_name);
 EXECUTE format('ALTER ROLE %I SET search_path = deployment, pg_catalog',migrator_name);
 EXECUTE format('ALTER ROLE %I CONNECTION LIMIT 20',api_name);
 EXECUTE format('ALTER ROLE %I CONNECTION LIMIT 2',migrator_name);
 -- Database boundaries are separate from cluster-wide capability-role membership.
 IF EXISTS(SELECT FROM pg_roles WHERE (rolname LIKE 'jayburd-dev-%' OR rolname LIKE 'bookkeeping_%') AND has_database_privilege(oid,current_database(),'CONNECT')) THEN
  RAISE EXCEPTION 'shared Production CONNECT isolation rejected';
 END IF;
 IF has_database_privilege(api_name,'bookkeeping','CONNECT') OR has_database_privilege(migrator_name,'bookkeeping','CONNECT') THEN
  RAISE EXCEPTION 'Production login can connect to Sandbox';
 END IF;
END $$;
COMMIT;
