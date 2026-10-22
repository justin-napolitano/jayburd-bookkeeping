#!/usr/bin/env bash
# Only disposable synthetic databases; no inherited database/service credentials.
set -euo pipefail
cd "$(dirname "$0")/.."
POSTGRES_IMAGE='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
MIGRATE_IMAGE='migrate/migrate:v4.20.1@sha256:76cc2074cb6642631f34a898ced71e6aeaa6b1a4d78c4daa743275a22e0c5be7'
proof_id="jayburd-webhook-bootstrap-$(date +%s)-$$"
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
 if docker exec "$proof_id" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then ready=1; break; fi
 sleep 1
done
[[ "$ready" == 1 ]] || exit 1
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE DATABASE bookkeeping' >/dev/null
docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" \
 "$MIGRATE_IMAGE" -path /proof/migrations -database 'postgres://postgres@database:5432/bookkeeping?sslmode=disable' up
# The platform provisions these fictional roles; bootstrap never creates IAM users or passwords.
docker exec -i "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d bookkeeping <<'SQL' >/dev/null
CREATE ROLE synthetic_owner LOGIN CREATEROLE;
CREATE ROLE "jayburd-dev-webhook@jayburd-bookkeeping-dev.iam" LOGIN;
GRANT bookkeeping_owner TO synthetic_owner WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT bookkeeping_webhook TO synthetic_owner WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;
GRANT "jayburd-dev-webhook@jayburd-bookkeeping-dev.iam" TO synthetic_owner WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;
REVOKE ALL ON DATABASE bookkeeping FROM PUBLIC;
GRANT CONNECT ON DATABASE bookkeeping TO synthetic_owner;
ALTER DATABASE bookkeeping OWNER TO synthetic_owner;
SQL
docker cp db/bootstrap/webhook-login.sql "$proof_id:/proof/webhook-login.sql"
bootstrap() { docker exec "$proof_id" psql -X -w -v ON_ERROR_STOP=1 -U synthetic_owner -d bookkeeping -v project_id="${3:-jayburd-bookkeeping-dev}" -v operation_key="$1" -v enabled="$2" -f /proof/webhook-login.sql; }
query() { docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -c "$1" >/dev/null; }
reject() { if bootstrap "$@" >/dev/null 2>&1; then echo 'FAIL: unsafe webhook bootstrap accepted' >&2;exit 1;fi; }
reject fixture true wrong-project
query 'GRANT SELECT ON ledger.book TO bookkeeping_webhook'
reject unsafe-capability true
query 'REVOKE SELECT ON ledger.book FROM bookkeeping_webhook'
query "DO \$\$ BEGIN IF EXISTS(SELECT FROM integration.webhook_ingress) OR pg_has_role('jayburd-dev-webhook@jayburd-bookkeeping-dev.iam','bookkeeping_webhook','MEMBER') THEN RAISE EXCEPTION 'partial unsafe bootstrap';END IF;END \$\$;"
bootstrap enable-1 true >/dev/null
bootstrap enable-1 true >/dev/null
reject enable-1 false
bootstrap disable-1 false >/dev/null
bootstrap disable-1 false >/dev/null
reject enable-1 true
# Only scoped synthetic aggregate assertions are read in this disposable database.
query "DO \$\$ BEGIN IF (SELECT revision FROM integration.webhook_ingress)<>2 OR (SELECT enabled FROM integration.webhook_ingress) OR (SELECT count(*) FROM integration.webhook_operator_receipt)<>2 THEN RAISE EXCEPTION 'bad replay/revocation';END IF; END \$\$;"
bootstrap enable-2 true >/dev/null
docker exec -i "$proof_id" psql -X -w -v ON_ERROR_STOP=1 -U 'jayburd-dev-webhook@jayburd-bookkeeping-dev.iam' -d bookkeeping <<'SQL' >/dev/null
DO $$ BEGIN
 IF current_user<>session_user OR NOT pg_has_role(current_user,'bookkeeping_webhook','USAGE') OR pg_has_role(current_user,'bookkeeping_webhook','SET') OR current_setting('search_path')<>'pg_catalog' OR (SELECT rolconnlimit FROM pg_roles WHERE rolname=current_user)<>4 THEN RAISE EXCEPTION 'unsafe runtime grants'; END IF;
 BEGIN EXECUTE 'CREATE TABLE public.forbidden(x int)';RAISE EXCEPTION 'DDL allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN EXECUTE 'SELECT * FROM ledger.book LIMIT 0';RAISE EXCEPTION 'table read allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN EXECUTE 'SET ROLE bookkeeping_owner';RAISE EXCEPTION 'owner allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN EXECUTE 'SELECT integration.set_webhook_ingress(''x'',''sandbox'',''runtime'',false)';RAISE EXCEPTION 'owner function allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $$;
SQL
login='"jayburd-dev-webhook@jayburd-bookkeeping-dev.iam"'
query "GRANT bookkeeping_api TO $login WITH INHERIT FALSE, SET FALSE"
reject mixed true
query "REVOKE bookkeeping_api FROM $login;GRANT pg_execute_server_program TO $login WITH INHERIT TRUE, SET FALSE"
reject predefined-server-program true
query "REVOKE pg_execute_server_program FROM $login"
query "DO \$\$ BEGIN IF (SELECT rolconnlimit FROM pg_roles WHERE rolname='jayburd-dev-webhook@jayburd-bookkeeping-dev.iam')<>4 OR (SELECT revision FROM integration.webhook_ingress)<>3 THEN RAISE EXCEPTION 'rejection changed bounds/mandate';END IF;END \$\$;"
query "ALTER ROLE $login CREATEDB"
reject privileged true
query "ALTER ROLE $login NOCREATEDB;GRANT SELECT ON ledger.book TO $login"
reject table true
query "REVOKE SELECT ON ledger.book FROM $login;CREATE SEQUENCE integration.synthetic_bootstrap_sequence;GRANT USAGE ON SEQUENCE integration.synthetic_bootstrap_sequence TO $login"
reject sequence true
query "REVOKE USAGE ON SEQUENCE integration.synthetic_bootstrap_sequence FROM $login;GRANT EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) TO $login"
reject direct-function true
query "REVOKE EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) FROM $login;GRANT EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) TO bookkeeping_webhook WITH GRANT OPTION"
reject inherited-function-delegation true
query "REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) FROM bookkeeping_webhook;GRANT CREATE ON SCHEMA integration TO $login"
reject schema true
query "REVOKE CREATE ON SCHEMA integration FROM $login;GRANT USAGE ON SCHEMA integration TO $login WITH GRANT OPTION"
reject delegation true
query "REVOKE USAGE ON SCHEMA integration FROM $login;CREATE ROLE synthetic_grantor LOGIN;GRANT CONNECT ON DATABASE bookkeeping TO synthetic_grantor;GRANT bookkeeping_webhook TO synthetic_grantor WITH ADMIN TRUE, INHERIT FALSE, SET FALSE"
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U synthetic_grantor -d bookkeeping -c "GRANT bookkeeping_webhook TO $login WITH ADMIN TRUE, INHERIT TRUE, SET TRUE" >/dev/null
reject alternate-grantor true
query "DO \$\$ BEGIN IF (SELECT revision FROM integration.webhook_ingress)<>3 OR (SELECT count(*) FROM integration.webhook_operator_receipt)<>3 THEN RAISE EXCEPTION 'rejected operation changed mandate';END IF;END \$\$;"
# Restore only the synthetic alternate grant, then prove the real wrapper's sanitized apply path.
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U synthetic_grantor -d bookkeeping -c "REVOKE bookkeeping_webhook FROM $login" >/dev/null
docker exec "$proof_id" mkdir -p /proof/scripts /proof/db/bootstrap
docker cp scripts/bootstrap-webhook.sh "$proof_id:/proof/scripts/bootstrap-webhook.sh"
docker cp db/bootstrap/webhook-login.sql "$proof_id:/proof/db/bootstrap/webhook-login.sql"
for round in 1 2; do
 result=$(docker exec --env PGSERVICE=untrusted-synthetic-service "$proof_id" bash /proof/scripts/bootstrap-webhook.sh --apply --project jayburd-bookkeeping-dev --operator-user synthetic_owner --socket-dir /var/run/postgresql --operation-key wrapper-enable --enabled true)
 [[ "$result" == 'PASS: webhook bootstrap operation committed.' ]]
done
query "DO \$\$ BEGIN IF (SELECT revision FROM integration.webhook_ingress)<>4 THEN RAISE EXCEPTION 'wrapper replay';END IF;END \$\$;"
# Wrapper defaults and invalid metadata cannot invoke psql; --check is metadata-only.
if bash scripts/bootstrap-webhook.sh >/dev/null 2>&1;then exit 1;fi
bash scripts/bootstrap-webhook.sh --check --project jayburd-bookkeeping-dev --operator-user synthetic_owner --socket-dir /synthetic/socket --operation-key enable-1 --enabled true >/dev/null
if bash scripts/bootstrap-webhook.sh --check --project wrong-project --operator-user synthetic_owner --socket-dir /synthetic/socket --operation-key x --enabled true >/dev/null 2>&1;then exit 1;fi
if bash scripts/bootstrap-webhook.sh --check --project jayburd-bookkeeping-dev --operator-user synthetic_owner --socket-dir /synthetic/socket,tcp.invalid --operation-key x --enabled true >/dev/null 2>&1;then exit 1;fi
echo 'PASS: webhook bootstrap replay/revocation, least privilege, role/grant/ACL denials and inert metadata validation.'
