#!/usr/bin/env bash
# Two synthetic databases on one isolated PostgreSQL instance prove CONNECT isolation and bootstrap replay.
set -euo pipefail
cd "$(dirname "$0")/.."
proof_image='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
proof_id="jayburd-shared-sql-$(date +%s)-$$"
created=0
proof_dir=$(mktemp -d)
cleanup(){ if [[ "$created" == 1 ]];then docker rm -fv "$proof_id" >/dev/null;fi; rm -rf "$proof_dir"; }
trap cleanup EXIT
trap 'exit 130' INT TERM
docker run -d --name "$proof_id" --network none --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$proof_image" >/dev/null
created=1
ready=0
for ((i=0;i<60;i++));do if docker exec "$proof_id" pg_isready -U postgres >/dev/null 2>&1;then ready=1;break;fi;sleep 1;done
[[ "$ready" == 1 ]]
docker exec "$proof_id" mkdir /proof
for file in cloud-sql.sql shared-production-cloud-sql.sql;do docker cp "db/bootstrap/$file" "$proof_id:/proof/$file";done
for db in bookkeeping bookkeeping_prod;do docker exec "$proof_id" createdb -U postgres "$db";done
sql(){ docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d "$1" "${@:2}"; }
for env in dev prod;do for component in api migrator;do sql bookkeeping -c "CREATE ROLE \"jayburd-$env-$component@jayburd-bookkeeping-dev.iam\" LOGIN" >/dev/null;done;done
sql bookkeeping -v api_user='jayburd-dev-api@jayburd-bookkeeping-dev.iam' -v migrator_user='jayburd-dev-migrator@jayburd-bookkeeping-dev.iam' -f /proof/cloud-sql.sql >/dev/null
bootstrap(){ sql "$1" -v api_user='jayburd-prod-api@jayburd-bookkeeping-dev.iam' -v migrator_user='jayburd-prod-migrator@jayburd-bookkeeping-dev.iam' -f /proof/shared-production-cloud-sql.sql; }
if bootstrap bookkeeping >"$proof_dir/wrong-db" 2>&1;then exit 1;fi
grep -Fq 'shared Production database rejected' "$proof_dir/wrong-db"
# A pre-existing cross-database grant must abort atomically instead of silently blessing isolation.
sql bookkeeping_prod -c 'GRANT CONNECT ON DATABASE bookkeeping_prod TO bookkeeping_api' >/dev/null
if bootstrap bookkeeping_prod >"$proof_dir/grant" 2>&1;then exit 1;fi
grep -Fq 'shared Production CONNECT isolation rejected' "$proof_dir/grant"
[[ "$(sql bookkeeping_prod -Atc "SELECT count(*) FROM pg_namespace WHERE nspname='deployment'")" == 0 ]]
sql bookkeeping_prod -c 'REVOKE CONNECT ON DATABASE bookkeeping_prod FROM bookkeeping_api' >/dev/null
for ((i=0;i<2;i++));do bootstrap bookkeeping_prod >"$proof_dir/bootstrap" 2>&1;done
for env in dev prod;do
 if [[ "$env" == dev ]];then own=bookkeeping;other=bookkeeping_prod;else own=bookkeeping_prod;other=bookkeeping;fi
 for component in api migrator;do
  login="jayburd-$env-$component@jayburd-bookkeeping-dev.iam"
  docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U "$login" -d "$own" -Atc 'SELECT current_database()' >"$proof_dir/own"
  [[ "$(cat "$proof_dir/own")" == "$own" ]]
  if docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U "$login" -d "$other" -Atc 'SELECT 1' >"$proof_dir/denied" 2>&1;then exit 1;fi
  grep -Fq 'permission denied for database' "$proof_dir/denied"
 done
done
printf 'PASS: shared-instance bootstrap replay, atomic rollback and real cross-database connection denial for both profiles.\n'
