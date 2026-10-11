#!/usr/bin/env bash
# Only disposable synthetic databases; no inherited database/service credentials.
set -euo pipefail
cd "$(dirname "$0")/.."
POSTGRES_IMAGE='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
MIGRATE_IMAGE='migrate/migrate:v4.20.1@sha256:76cc2074cb6642631f34a898ced71e6aeaa6b1a4d78c4daa743275a22e0c5be7'
proof_id="jayburd-ledger-$(date +%s)-$$"
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
# Copy explicit synthetic inputs; the daemon need not share the checkout path.
docker cp db/migrations "$proof_id:/proof/migrations"
ready=0
for ((i=0;i<60;i++)); do
 if docker exec "$proof_id" pg_isready -U postgres >/dev/null 2>&1; then ready=1; break; fi
 sleep 1
done
[[ "$ready" == 1 ]] || exit 1
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE DATABASE ledger_template' >/dev/null
docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" \
 "$MIGRATE_IMAGE" -path /proof/migrations -database 'postgres://postgres@database:5432/ledger_template?sslmode=disable' up
# Prove a forward upgrade preserves existing synthetic rows between versions.
docker exec "$proof_id" createdb -U postgres ledger_upgrade
run_upgrade() {
 docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" \
  "$MIGRATE_IMAGE" -path /proof/migrations -database 'postgres://postgres@database:5432/ledger_upgrade?sslmode=disable' "$@"
}
run_upgrade up 1
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d ledger_upgrade -c "INSERT INTO ledger.entity(entity_id,name,kind) VALUES('00000000-0000-4000-8000-000000000099','Synthetic upgrade sentinel','legal')" >/dev/null
run_upgrade up
[[ "$(docker exec "$proof_id" psql -X -At -U postgres -d ledger_upgrade -c 'SELECT count(*) FROM ledger.entity')" == 1 ]]
echo 'PASS: forward migration retained synthetic state'
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE ROLE synthetic_runtime LOGIN IN ROLE bookkeeping_api; CREATE ROLE synthetic_identity_operator LOGIN IN ROLE bookkeeping_identity_operator' >/dev/null
mkdir -p .local
case "$(docker exec "$proof_id" uname -m)" in
 aarch64) proof_arch=arm64 ;;
 x86_64) proof_arch=amd64 ;;
 *) echo 'Unsupported disposable test architecture' >&2; exit 1 ;;
esac
# Static test executable runs on the internal network; no published database port.
CGO_ENABLED=0 GOOS=linux GOARCH="$proof_arch" go test -c -tags integration -o .local/ledger.test ./internal/ledger
docker cp .local/ledger.test "$proof_id:/proof/ledger.test"
docker cp tests/corpus "$proof_id:/proof/corpus"
docker exec "$proof_id" mkdir -p /proof/work/tests
docker exec "$proof_id" ln -s /proof/corpus /proof/work/tests/corpus
run_proof() {
 docker run --rm --network "$proof_id" --workdir /proof/work \
  --mount "type=volume,source=$proof_id,target=/proof,readonly" \
  --entrypoint /proof/ledger.test "$POSTGRES_IMAGE" -ledger-port=5432 "$@"
}
run_proof -test.v -test.timeout=120s
# Stream only this invocation's synthetic proof database into a fresh restore.
docker exec "$proof_id" createdb -U postgres ledger_restored
docker exec "$proof_id" pg_dump -U postgres -Fc ledger_restore | \
 docker exec -i "$proof_id" pg_restore -U postgres -d ledger_restored
run_proof -test.v -test.run=TestLedgerRestoreReplay -ledger-restored
# Prove managed composition rejects privileged runtime roles on the same isolated DB.
CGO_ENABLED=0 GOOS=linux GOARCH="$proof_arch" go test -c -tags integration -o .local/server.test ./internal/server
docker cp .local/server.test "$proof_id:/proof/server.test"
docker run --rm --network "$proof_id" --workdir /proof/work \
 --mount "type=volume,source=$proof_id,target=/proof,readonly" \
 --entrypoint /proof/server.test "$POSTGRES_IMAGE" \
 -server-proof -test.run=TestRuntimeReadiness -test.v -test.timeout=60s
