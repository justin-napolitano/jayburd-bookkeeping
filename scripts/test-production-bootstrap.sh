#!/usr/bin/env bash
# Prove the owner-only name guard and restricted grants in a network-isolated disposable database.
set -euo pipefail
cd "$(dirname "$0")/.."
proof_image='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
proof_id="jayburd-prod-bootstrap-$(date +%s)-$$"
created=0
proof_dir=$(mktemp -d)
cleanup() {
 if [[ "$created" == 1 ]]; then docker rm -fv "$proof_id" >/dev/null; fi
 rm -rf "$proof_dir"
}
trap cleanup EXIT
trap 'exit 130' INT TERM
docker run -d --name "$proof_id" --network none --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$proof_image" >/dev/null
created=1
ready=0
for ((i=0;i<60;i++)); do
 if docker exec "$proof_id" pg_isready -U postgres >/dev/null 2>&1; then ready=1;break;fi
 sleep 1
done
[[ "$ready" == 1 ]]
docker exec "$proof_id" mkdir /proof
docker cp db/bootstrap/production-cloud-sql.sql "$proof_id:/proof/production-cloud-sql.sql"
docker cp db/bootstrap/cloud-sql.sql "$proof_id:/proof/cloud-sql.sql"
docker exec "$proof_id" createdb -U postgres bookkeeping
# No host DSN, credentials, state or environment is consumed by the SQL client.
psql_proof() { docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d bookkeeping "$@"; }
bootstrap() { psql_proof -v project_id="$1" -v api_user="$2" -v migrator_user="$3" -f /proof/production-cloud-sql.sql; }
for bad in dev-project dev-api foreign-migrator; do
 project=fictional-project
 api='jayburd-prod-api@fictional-project.iam'
 migrator='jayburd-prod-migrator@fictional-project.iam'
 case "$bad" in
  dev-project) project=jayburd-bookkeeping-dev ;;
  dev-api) api='jayburd-dev-api@fictional-project.iam' ;;
  foreign-migrator) migrator='jayburd-prod-migrator@other-project.iam' ;;
 esac
 if bootstrap "$project" "$api" "$migrator" >"$proof_dir/rejected" 2>&1;then exit 1;fi
 grep -Fq 'Production bootstrap identity rejected' "$proof_dir/rejected"
 [[ "$(psql_proof -Atc "SELECT count(*) FROM pg_roles WHERE rolname='bookkeeping_owner'")" == 0 ]]
 [[ "$(psql_proof -Atc "SELECT count(*) FROM pg_namespace WHERE nspname='deployment'")" == 0 ]]
done
psql_proof -c 'CREATE ROLE "jayburd-prod-api@fictional-project.iam" LOGIN; CREATE ROLE "jayburd-prod-migrator@fictional-project.iam" LOGIN;' >/dev/null
for ((i=0;i<2;i++));do bootstrap fictional-project 'jayburd-prod-api@fictional-project.iam' 'jayburd-prod-migrator@fictional-project.iam' >"$proof_dir/accepted" 2>&1;done
[[ "$(psql_proof -Atc "SELECT pg_has_role('jayburd-prod-migrator@fictional-project.iam','bookkeeping_owner','SET') AND NOT pg_has_role('jayburd-prod-api@fictional-project.iam','bookkeeping_owner','MEMBER') AND pg_has_role('jayburd-prod-api@fictional-project.iam','bookkeeping_api','MEMBER') AND NOT has_database_privilege('jayburd-prod-api@fictional-project.iam','bookkeeping','CREATE')")" == t ]]
[[ "$(psql_proof -Atc "SELECT count(*) FROM pg_roles WHERE rolname LIKE 'bookkeeping_%' AND (rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls)")" == 0 ]]
psql_proof -c 'ALTER ROLE "jayburd-prod-api@fictional-project.iam" CREATEROLE' >/dev/null
if bootstrap fictional-project 'jayburd-prod-api@fictional-project.iam' 'jayburd-prod-migrator@fictional-project.iam' >"$proof_dir/unsafe" 2>&1;then exit 1;fi
grep -Fq 'unsafe IAM login roles' "$proof_dir/unsafe"
printf 'PASS: Production bootstrap rejects mixed names before mutation; safe grants replay and unsafe role denial verified.\n'
