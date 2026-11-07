#!/usr/bin/env bash
# Only disposable synthetic databases on an internal network; never an inherited DSN.
set -euo pipefail
cd "$(dirname "$0")/.."
POSTGRES_IMAGE='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
MIGRATE_IMAGE='migrate/migrate:v4.20.1@sha256:76cc2074cb6642631f34a898ced71e6aeaa6b1a4d78c4daa743275a22e0c5be7'
proof_id="jayburd-private-$(date +%s)-$$"
network_created=0
container_created=0
volume_created=0
cleanup() {
 if [[ "$container_created" == 1 ]]; then docker rm -fv "$proof_id" >/dev/null; fi
 if [[ "$volume_created" == 1 ]]; then docker volume rm "$proof_id" >/dev/null; fi
 if [[ "$network_created" == 1 ]]; then docker network rm "$proof_id" >/dev/null; fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM
bash scripts/migrate.sh check
docker pull "$POSTGRES_IMAGE" >/dev/null
docker pull "$MIGRATE_IMAGE" >/dev/null
docker network create --internal "$proof_id" >/dev/null
network_created=1
docker volume create "$proof_id" >/dev/null
volume_created=1
docker run -d --name "$proof_id" --network "$proof_id" --network-alias database \
 --mount "type=volume,source=$proof_id,target=/proof" --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$POSTGRES_IMAGE" >/dev/null
container_created=1
docker cp db/migrations "$proof_id:/proof/migrations"
ready=0
for ((i=0;i<60;i++)); do
 if docker exec "$proof_id" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then ready=1; break; fi
 sleep 1
done
[[ "$ready" == 1 ]] || exit 1
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE DATABASE private_template' >/dev/null
docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" \
 "$MIGRATE_IMAGE" -path /proof/migrations -database 'postgres://postgres@database:5432/private_template?sslmode=disable' up
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE ROLE synthetic_operator LOGIN NOINHERIT IN ROLE bookkeeping_owner; CREATE ROLE synthetic_runtime LOGIN IN ROLE bookkeeping_api; CREATE ROLE synthetic_sync LOGIN IN ROLE bookkeeping_sync; CREATE ROLE synthetic_other LOGIN; CREATE ROLE synthetic_mixed LOGIN IN ROLE bookkeeping_api,bookkeeping_sync' >/dev/null
mkdir -p .local
case "$(docker exec "$proof_id" uname -m)" in
 aarch64) proof_arch=arm64 ;;
 x86_64) proof_arch=amd64 ;;
 *) exit 1 ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH="$proof_arch" go test -c -tags integration -o .local/private.test ./internal/privateregistry
docker cp .local/private.test "$proof_id:/proof/private.test"
docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" --entrypoint /proof/private.test "$POSTGRES_IMAGE" -private-registry-proof -collection-read-proof -test.v -test.timeout=240s
# Exercise42 itself even when later empty-safe migrations are installed.
docker exec "$proof_id" createdb -U postgres private_empty_down -T private_template
run_empty_migration() {
 docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" \
  "$MIGRATE_IMAGE" -path /proof/migrations -database 'postgres://postgres@database:5432/private_empty_down?sslmode=disable' "$@"
}
run_empty_migration goto 42
run_empty_migration down 1
run_empty_migration up 1
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d private_empty_down -c "INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('00000000-0000-4000-8000-000000000999','synthetic','retained-owner');INSERT INTO banking.collection_workspace(owner_principal_id,capabilities)VALUES('00000000-0000-4000-8000-000000000999',ARRAY['read']);" >/dev/null
if run_empty_migration down 1 >".local/$proof_id-down.log" 2>&1; then
 echo 'FAIL: retained custody allowed destructive downgrade.' >&2
 exit 1
fi
grep -q 'private_registry_retained_records' ".local/$proof_id-down.log"
[[ "$(docker exec "$proof_id" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d private_empty_down -c 'SELECT count(*) FROM banking.collection_workspace')" == 1 ]]
rm -f ".local/$proof_id-down.log"
echo 'PASS: empty-only rollback/reapply and retained custody refusal.'
