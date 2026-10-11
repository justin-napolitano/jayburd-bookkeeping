#!/usr/bin/env bash
# Exercise migrate against a private disposable PostgreSQL; never use inherited DSNs.
set -euo pipefail
cd "$(dirname "$0")/.."
[[ -n "${MIGRATE_IMAGE:-}" ]] || { echo 'Run make migration-test.' >&2; exit 2; }
POSTGRES_IMAGE='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
test_id="jayburd-migrations-$(date +%s)-$$"
network_created=0
container_created=0
volume_created=0

# Remove only resources successfully created by this invocation, including on failure.
cleanup() {
  if [[ "$container_created" == 1 ]]; then docker rm -f -v "$test_id" >/dev/null; fi
  if [[ "$volume_created" == 1 ]]; then docker volume rm "$test_id" >/dev/null; fi
  if [[ "$network_created" == 1 ]]; then docker network rm "$test_id" >/dev/null; fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM
docker pull "$POSTGRES_IMAGE" >/dev/null
docker pull "$MIGRATE_IMAGE" >/dev/null
docker network create --internal "$test_id" >/dev/null
network_created=1
docker volume create "$test_id" >/dev/null
volume_created=1
docker run -d --name "$test_id" --network "$test_id" --network-alias database \
  --mount "type=volume,source=$test_id,target=/proof" --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust \
  "$POSTGRES_IMAGE" >/dev/null
container_created=1
# Only migration artifacts are copied, never the whole checkout or credentials.
docker exec "$test_id" mkdir -p /proof/db /proof/tests
docker cp db/migrations "$test_id:/proof/db/migrations"
docker cp tests/migrations "$test_id:/proof/tests/migrations"
ready=0
for ((attempt=0; attempt<60; attempt++)); do
  if docker exec "$test_id" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then ready=1; break; fi
  sleep 1
done
[[ "$ready" == 1 ]] || { echo 'FAIL: disposable database readiness timeout.' >&2; exit 1; }

# SQL and migration connections are restricted to the container created above.
sql() { docker exec "$test_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d "$1" -Atc "$2"; }
run_migrate() {
  local folder=$1 database=$2
  shift 2
  docker run --rm --network "$test_id" \
    --mount "type=volume,source=$test_id,target=/proof,readonly" \
    "$MIGRATE_IMAGE" -path "/proof/$folder" -database \
    "postgres://postgres@database:5432/$database?sslmode=disable&connect_timeout=5&x-statement-timeout=10000" \
    -lock-timeout 3 "$@"
}
sql postgres 'CREATE DATABASE migration_good' >/dev/null
sql postgres 'CREATE DATABASE migration_failure' >/dev/null
run_migrate tests/migrations/good migration_good up
run_migrate tests/migrations/good migration_good up
[[ "$(sql migration_good 'SELECT count(*) FROM migration_probe')" == 1 ]]
[[ "$(sql migration_good 'SELECT version::text || chr(58) || dirty::text FROM schema_migrations')" == '1:false' ]]
echo 'PASS: fresh migration and replay.'
run_migrate tests/migrations/good migration_good down 1
[[ "$(sql migration_good "SELECT to_regclass('migration_probe') IS NULL")" == t ]]
run_migrate tests/migrations/good migration_good up
echo 'PASS: synthetic down/up roundtrip.'
if run_migrate tests/migrations/failure migration_failure up; then echo 'FAIL: expected SQL failure.' >&2; exit 1; fi
[[ "$(sql migration_failure "SELECT to_regclass('must_rollback') IS NULL")" == t ]]
[[ "$(sql migration_failure 'SELECT dirty FROM schema_migrations')" == t ]]
if run_migrate tests/migrations/failure migration_failure up; then echo 'FAIL: dirty state did not block replay.' >&2; exit 1; fi
echo 'PASS: failed DDL rolls back; dirty state blocks retry.'
shopt -s nullglob
migrations=(db/migrations/*.up.sql)
if ((${#migrations[@]})); then
  sql postgres 'CREATE DATABASE migration_product' >/dev/null
  run_migrate db/migrations migration_product up
  run_migrate db/migrations migration_product up
  echo 'PASS: product migration fresh install and replay.'
else
  echo 'SKIP: no approved product schema yet; this proves tooling only.'
fi
