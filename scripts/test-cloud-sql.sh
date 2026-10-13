#!/usr/bin/env bash
# Execute cloud bootstrap/migrator under restricted roles on an isolated disposable PostgreSQL.
set -euo pipefail
cd "$(dirname "$0")/.."
image='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
proof="jayburd-cloud-sql-$(date +%s)-$$"
network_created=0
container_created=0
cleanup() {
 if [[ "$container_created" == 1 ]]; then docker rm -fv "$proof" >/dev/null; fi
 if [[ "$network_created" == 1 ]]; then docker network rm "$proof" >/dev/null; fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM
bash scripts/migrate.sh check
docker pull "$image" >/dev/null
docker network create --internal "$proof" >/dev/null
network_created=1
docker run -d --name "$proof" --network "$proof" --network-alias database \
 --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$image" >/dev/null
container_created=1
# TCP waits for the final server; the entrypoint temporary initialization server is socket-only.
ready=0
for ((i=0;i<60;i++)); do
 if docker exec "$proof" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then ready=1;break;fi
 sleep 1
done
[[ "$ready" == 1 ]] || exit 1
# Superuser is used only to model platform creation of logins and an administrator.
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE ROLE synthetic_admin LOGIN CREATEDB CREATEROLE; CREATE ROLE synthetic_api LOGIN; CREATE ROLE synthetic_migrator LOGIN; GRANT synthetic_api, synthetic_migrator TO synthetic_admin WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;' >/dev/null
docker exec "$proof" createdb -U synthetic_admin bookkeeping
docker cp db/bootstrap/cloud-sql.sql "$proof:/tmp/bootstrap.sql"
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_admin -d bookkeeping -v api_user=synthetic_api -v migrator_user=synthetic_migrator -f /tmp/bootstrap.sql >/dev/null
# Replay bootstrap must be safe; no product SQL is rewritten.
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_admin -d bookkeeping -v api_user=synthetic_api -v migrator_user=synthetic_migrator -f /tmp/bootstrap.sql >/dev/null
docker exec "$proof" createdb -U synthetic_admin bookkeeping_failure
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_admin -d bookkeeping_failure -v api_user=synthetic_api -v migrator_user=synthetic_migrator -f /tmp/bootstrap.sql >/dev/null
docker exec "$proof" createdb -U synthetic_admin bookkeeping_status
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_admin -d bookkeeping_status -v api_user=synthetic_api -v migrator_user=synthetic_migrator -f /tmp/bootstrap.sql >/dev/null
case "$(docker exec "$proof" uname -m)" in
 aarch64) arch=arm64 ;;
 x86_64) arch=amd64 ;;
 *) exit 1 ;;
esac
mkdir -p .local
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go test -c -tags integration -o .local/cloud-sql.test ./internal/migration
docker cp .local/cloud-sql.test "$proof:/tmp/proof.test"
docker exec "$proof" /tmp/proof.test -cloud-sql-proof -test.v -test.timeout=120s
