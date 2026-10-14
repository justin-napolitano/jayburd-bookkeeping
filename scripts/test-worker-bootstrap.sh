#!/usr/bin/env bash
# Prove owner SQL role wiring on a disposable synthetic database; no cloud/provider calls.
set -euo pipefail
cd "$(dirname "$0")/.."
image='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
proof="jayburd-worker-bootstrap-$(date +%s)-$$"
created=0
cleanup() { if [[ "$created" == 1 ]]; then docker rm -fv "$proof" >/dev/null; fi; }
trap cleanup EXIT
trap 'exit 130' INT TERM
docker run -d --name "$proof" --network none --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$image" >/dev/null
created=1
ready=0
for ((i=0;i<60;i++)); do
 if docker exec "$proof" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then ready=1;break;fi
 sleep 1
done
[[ "$ready" == 1 ]]
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE ROLE synthetic_owner LOGIN CREATEDB CREATEROLE; CREATE ROLE synthetic_publisher LOGIN; CREATE ROLE synthetic_sync LOGIN; GRANT synthetic_publisher,synthetic_sync TO synthetic_owner WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;' >/dev/null
docker exec "$proof" createdb -U synthetic_owner bookkeeping
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_owner -d bookkeeping -c 'CREATE ROLE bookkeeping_publisher NOLOGIN; CREATE ROLE bookkeeping_sync NOLOGIN; CREATE ROLE bookkeeping_api NOLOGIN;' >/dev/null
docker cp db/bootstrap/worker-logins.sql "$proof:/tmp/workers.sql"
bootstrap() { docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_owner -d bookkeeping -v publisher_user=synthetic_publisher -v sync_user=synthetic_sync -f /tmp/workers.sql; }
bootstrap >/dev/null
bootstrap >/dev/null
# Exact inherited capability; SET ROLE, cross-worker authority and DDL stay denied.
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_publisher -d bookkeeping -c "DO \$\$ BEGIN IF NOT pg_has_role(current_user,'bookkeeping_publisher','USAGE') OR pg_has_role(current_user,'bookkeeping_publisher','SET') OR pg_has_role(current_user,'bookkeeping_sync','MEMBER') THEN RAISE EXCEPTION 'wrong publisher capability'; END IF; BEGIN EXECUTE 'CREATE TABLE public.forbidden(x int)'; RAISE EXCEPTION 'DDL allowed'; EXCEPTION WHEN insufficient_privilege THEN NULL; END; END \$\$;" >/dev/null
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_sync -d bookkeeping -c "DO \$\$ BEGIN IF NOT pg_has_role(current_user,'bookkeeping_sync','USAGE') OR pg_has_role(current_user,'bookkeeping_sync','SET') OR pg_has_role(current_user,'bookkeeping_publisher','MEMBER') THEN RAISE EXCEPTION 'wrong sync capability'; END IF; END \$\$;" >/dev/null
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_owner -d bookkeeping -c 'GRANT bookkeeping_api TO synthetic_sync WITH INHERIT FALSE, SET FALSE;' >/dev/null
if bootstrap >/dev/null 2>&1; then echo 'FAIL: mixed capability accepted' >&2;exit 1;fi
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_owner -d bookkeeping -c 'REVOKE bookkeeping_api FROM synthetic_sync; ALTER ROLE synthetic_sync CREATEDB;' >/dev/null
if bootstrap >/dev/null 2>&1; then echo 'FAIL: privileged login accepted' >&2;exit 1;fi
# Grants are grantor-specific: a different grantor's delegation must also block bootstrap.
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -c 'ALTER ROLE synthetic_sync NOCREATEDB; CREATE ROLE synthetic_grantor LOGIN; GRANT bookkeeping_sync TO synthetic_grantor WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;' >/dev/null
docker exec "$proof" psql -X -v ON_ERROR_STOP=1 -U synthetic_grantor -d bookkeeping -c 'GRANT bookkeeping_sync TO synthetic_sync WITH ADMIN TRUE, INHERIT TRUE, SET TRUE;' >/dev/null
if bootstrap >/dev/null 2>&1; then echo 'FAIL: alternate grantor delegation accepted' >&2;exit 1;fi
echo 'PASS: worker role wiring, replay, inherited capability, SET/DDL denial and unsafe-role rejection.'
