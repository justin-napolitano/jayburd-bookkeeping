-- Owner-only Production name guard; actual server identity must be independently verified.
-- Required psql variables: project_id, api_user, migrator_user. Never passwords.
\set ON_ERROR_STOP on
SELECT set_config('jayburd.production_project', :'project_id', false),
       set_config('jayburd.production_api', :'api_user', false),
       set_config('jayburd.production_migrator', :'migrator_user', false);
DO $$
DECLARE project_name text := current_setting('jayburd.production_project');
BEGIN
 IF project_name !~ '^[a-z][a-z0-9-]{4,28}[a-z0-9]$' OR project_name='jayburd-bookkeeping-dev'
 OR current_setting('jayburd.production_api') <> 'jayburd-prod-api@'||project_name||'.iam'
 OR current_setting('jayburd.production_migrator') <> 'jayburd-prod-migrator@'||project_name||'.iam' THEN
  RAISE EXCEPTION 'Production bootstrap identity rejected';
 END IF;
END $$;
\ir cloud-sql.sql
