#!/usr/bin/env bash
# Only disposable synthetic databases; no inherited database/service credentials.
set -euo pipefail
cd "$(dirname "$0")/.."
POSTGRES_IMAGE='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
MIGRATE_IMAGE='migrate/migrate:v4.20.1@sha256:76cc2074cb6642631f34a898ced71e6aeaa6b1a4d78c4daa743275a22e0c5be7'
proof_id="jayburd-production-webhook-bootstrap-$(date +%s)-$$"
network_created=0
container_created=0
volume_created=0
# Tear down only this invocation's synthetic container, volume and private network.
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
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -c 'CREATE DATABASE bookkeeping_prod' >/dev/null
docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" \
 "$MIGRATE_IMAGE" -path /proof/migrations -database 'postgres://postgres@database:5432/bookkeeping_prod?sslmode=disable' up
# The platform provisions these fictional roles; bootstrap never creates IAM users or passwords.
docker exec -i "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d bookkeeping_prod <<'SQL' >/dev/null
CREATE ROLE synthetic_owner LOGIN CREATEROLE;
CREATE ROLE "jayburd-prod-webhook@jayburd-bookkeeping-dev.iam" LOGIN;
GRANT bookkeeping_owner TO synthetic_owner WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT bookkeeping_webhook TO synthetic_owner WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;
GRANT "jayburd-prod-webhook@jayburd-bookkeeping-dev.iam" TO synthetic_owner WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;
REVOKE ALL ON DATABASE bookkeeping_prod FROM PUBLIC;
GRANT CONNECT ON DATABASE bookkeeping_prod TO synthetic_owner;
ALTER DATABASE bookkeeping_prod OWNER TO synthetic_owner;
SQL
docker exec "$proof_id" createdb -U postgres bookkeeping
docker exec "$proof_id" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping_prod -c 'REVOKE ALL ON DATABASE bookkeeping FROM PUBLIC' >/dev/null
docker cp db/bootstrap/production-webhook-login.sql "$proof_id:/proof/production-webhook-login.sql"
# Execute the reviewed bootstrap only in the fixture; overrides exercise target guards.
bootstrap() { docker exec "$proof_id" psql -X -w -v ON_ERROR_STOP=1 -U synthetic_owner -d bookkeeping_prod -v project_id="${3:-jayburd-bookkeeping-dev}" -v project_number="${4:-9088794002}" -v operation_key="$1" -v enabled="$2" -f /proof/production-webhook-login.sql; }
# Fixture-only catalog setup/assertions never accept an inherited database destination.
query() { docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U postgres -d bookkeeping_prod -c "$1" >/dev/null; }
# A denial must come from the intended guard or existing ingress domain boundary.
reject() { local result; if result=$(bootstrap "$@" 2>&1); then echo 'FAIL: unsafe webhook bootstrap accepted' >&2;exit 1;fi; [[ "$result" == *webhook_bootstrap_rejected* || "$result" == *connection_operation_rejected* ]] || { echo 'FAIL: unexpected synthetic rejection' >&2;exit 1; }; }
reject fixture true wrong-project
reject wrong-number true jayburd-bookkeeping-dev 9088794003
query 'GRANT CONNECT ON DATABASE bookkeeping TO "jayburd-prod-webhook@jayburd-bookkeeping-dev.iam"'
reject cross-database true
query 'REVOKE CONNECT ON DATABASE bookkeeping FROM "jayburd-prod-webhook@jayburd-bookkeeping-dev.iam"'
query 'GRANT SELECT ON ledger.book TO bookkeeping_webhook'
reject unsafe-capability true
query 'REVOKE SELECT ON ledger.book FROM bookkeeping_webhook'
query "DO \$\$ BEGIN IF EXISTS(SELECT FROM integration.webhook_ingress) OR pg_has_role('jayburd-prod-webhook@jayburd-bookkeeping-dev.iam','bookkeeping_webhook','MEMBER') THEN RAISE EXCEPTION 'partial unsafe bootstrap';END IF;END \$\$;"
bootstrap enable-1 true >/dev/null
# Simulate Google's managed marker only in this disposable database; replay must preserve it.
query 'CREATE ROLE cloudsqliamserviceaccount NOLOGIN;GRANT cloudsqliamserviceaccount TO "jayburd-prod-webhook@jayburd-bookkeeping-dev.iam" WITH ADMIN FALSE, INHERIT TRUE, SET TRUE'
bootstrap enable-1 true >/dev/null
reject enable-1 false
bootstrap disable-1 false >/dev/null
bootstrap disable-1 false >/dev/null
reject enable-1 true
# Only scoped synthetic aggregate assertions are read in this disposable database.
query "DO \$\$ BEGIN IF (SELECT revision FROM integration.webhook_ingress)<>2 OR (SELECT enabled FROM integration.webhook_ingress) OR (SELECT count(*) FROM integration.webhook_operator_receipt)<>2 THEN RAISE EXCEPTION 'bad replay/revocation';END IF; END \$\$;"
bootstrap enable-2 true >/dev/null
docker exec -i "$proof_id" psql -X -w -v ON_ERROR_STOP=1 -U 'jayburd-prod-webhook@jayburd-bookkeeping-dev.iam' -d bookkeeping_prod <<'SQL' >/dev/null
DO $$ BEGIN
 IF current_user<>session_user OR NOT pg_has_role(current_user,'bookkeeping_webhook','USAGE') OR pg_has_role(current_user,'bookkeeping_webhook','SET') OR current_setting('search_path')<>'pg_catalog' OR (SELECT rolconnlimit FROM pg_roles WHERE rolname=current_user)<>4 THEN RAISE EXCEPTION 'unsafe runtime grants'; END IF;
 IF NOT has_function_privilege(current_user,'integration.accept_verified_link_session(text,text,text)','EXECUTE') OR has_function_privilege(current_user,'integration.connection_link_proof(uuid,uuid,uuid,uuid)','EXECUTE') THEN RAISE EXCEPTION 'unsafe Link receipt scope';END IF;
 BEGIN EXECUTE 'CREATE TABLE public.forbidden(x int)';RAISE EXCEPTION 'DDL allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN EXECUTE 'SELECT * FROM ledger.book LIMIT 0';RAISE EXCEPTION 'table read allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN EXECUTE 'SET ROLE bookkeeping_owner';RAISE EXCEPTION 'owner allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN EXECUTE 'SELECT integration.set_webhook_ingress(''x'',''production'',''runtime'',false)';RAISE EXCEPTION 'owner function allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $$;
-- Managed marker SET preserves login authentication but grants no application authority.
SET ROLE cloudsqliamserviceaccount;
DO $$ BEGIN
 BEGIN EXECUTE 'SELECT * FROM ledger.book LIMIT 0';RAISE EXCEPTION 'marker table read allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
 BEGIN EXECUTE 'SELECT integration.accept_verified_webhook(''x'',''x'',''x'')';RAISE EXCEPTION 'marker ingress allowed';EXCEPTION WHEN insufficient_privilege THEN NULL;END;
END $$;
RESET ROLE;
SQL
login='"jayburd-prod-webhook@jayburd-bookkeeping-dev.iam"'
# Each unsafe marker shape must fail without advancing the ingress receipt.
query "GRANT cloudsqliamserviceaccount TO $login WITH ADMIN TRUE, INHERIT TRUE, SET TRUE"
reject marker-admin true
query "GRANT cloudsqliamserviceaccount TO $login WITH ADMIN FALSE, INHERIT FALSE, SET TRUE"
reject marker-noinherit true
query "GRANT cloudsqliamserviceaccount TO $login WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;ALTER ROLE cloudsqliamserviceaccount LOGIN"
reject marker-login true
query 'ALTER ROLE cloudsqliamserviceaccount NOLOGIN'
# Role flags are independent privilege paths even without any financial ACL.
for marker_flag in SUPERUSER CREATEDB CREATEROLE REPLICATION BYPASSRLS; do
 query "ALTER ROLE cloudsqliamserviceaccount $marker_flag"
 reject "marker-$marker_flag" true
 query "ALTER ROLE cloudsqliamserviceaccount NO$marker_flag"
done
query 'CREATE ROLE synthetic_marker_parent;GRANT synthetic_marker_parent TO cloudsqliamserviceaccount WITH INHERIT FALSE, SET FALSE'
reject marker-parent true
query 'REVOKE synthetic_marker_parent FROM cloudsqliamserviceaccount;GRANT SELECT ON ledger.book TO cloudsqliamserviceaccount'
reject marker-table true
query 'REVOKE SELECT ON ledger.book FROM cloudsqliamserviceaccount;GRANT SELECT(book_id) ON ledger.book TO cloudsqliamserviceaccount'
reject marker-column true
query 'REVOKE SELECT(book_id) ON ledger.book FROM cloudsqliamserviceaccount;GRANT EXECUTE ON FUNCTION integration.connection_link_proof(uuid,uuid,uuid,uuid) TO cloudsqliamserviceaccount'
reject marker-function true
query 'REVOKE EXECUTE ON FUNCTION integration.connection_link_proof(uuid,uuid,uuid,uuid) FROM cloudsqliamserviceaccount;CREATE ROLE synthetic_marker_unrelated'
query "GRANT synthetic_marker_unrelated TO $login WITH INHERIT TRUE, SET TRUE"
reject unrelated-set true
query "REVOKE synthetic_marker_unrelated FROM $login;GRANT bookkeeping_webhook TO $login WITH ADMIN FALSE, INHERIT TRUE, SET TRUE"
reject capability-set true
query "GRANT bookkeeping_webhook TO $login WITH ADMIN FALSE, INHERIT TRUE, SET FALSE"
bootstrap enable-2 true >/dev/null
query "GRANT bookkeeping_api TO $login WITH INHERIT FALSE, SET FALSE"
reject mixed true
query "REVOKE bookkeeping_api FROM $login;GRANT pg_execute_server_program TO $login WITH INHERIT TRUE, SET FALSE"
reject predefined-server-program true
query "REVOKE pg_execute_server_program FROM $login"
query "DO \$\$ BEGIN IF (SELECT rolconnlimit FROM pg_roles WHERE rolname='jayburd-prod-webhook@jayburd-bookkeeping-dev.iam')<>4 OR (SELECT revision FROM integration.webhook_ingress)<>3 THEN RAISE EXCEPTION 'rejection changed bounds/mandate';END IF;END \$\$;"
query "ALTER ROLE $login CREATEDB"
reject privileged true
query "ALTER ROLE $login NOCREATEDB;GRANT SELECT ON ledger.book TO $login"
reject table true
query "REVOKE SELECT ON ledger.book FROM $login;CREATE SEQUENCE integration.synthetic_bootstrap_sequence;GRANT USAGE ON SEQUENCE integration.synthetic_bootstrap_sequence TO $login"
reject sequence true
query "REVOKE USAGE ON SEQUENCE integration.synthetic_bootstrap_sequence FROM $login;GRANT EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) TO $login"
reject direct-function true
query "REVOKE EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) FROM $login;GRANT EXECUTE ON FUNCTION integration.accept_verified_link_session(text,text,text) TO $login"
reject direct-link-function true
query "REVOKE EXECUTE ON FUNCTION integration.accept_verified_link_session(text,text,text) FROM $login;GRANT EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) TO $login"
query "REVOKE EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) FROM $login;GRANT EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) TO bookkeeping_webhook WITH GRANT OPTION"
reject inherited-function-delegation true
query "REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION integration.accept_verified_webhook(text,text,text) FROM bookkeeping_webhook;GRANT CREATE ON SCHEMA integration TO $login"
reject schema true
query "REVOKE CREATE ON SCHEMA integration FROM $login;GRANT USAGE ON SCHEMA integration TO $login WITH GRANT OPTION"
reject delegation true
query "REVOKE USAGE ON SCHEMA integration FROM $login;CREATE ROLE synthetic_grantor LOGIN;GRANT CONNECT ON DATABASE bookkeeping_prod TO synthetic_grantor;GRANT bookkeeping_webhook TO synthetic_grantor WITH ADMIN TRUE, INHERIT FALSE, SET FALSE"
docker exec "$proof_id" psql -X -v ON_ERROR_STOP=1 -U synthetic_grantor -d bookkeeping_prod -c "GRANT bookkeeping_webhook TO $login WITH ADMIN TRUE, INHERIT TRUE, SET TRUE" >/dev/null
reject alternate-grantor true
query "DO \$\$ BEGIN IF (SELECT revision FROM integration.webhook_ingress)<>3 OR (SELECT count(*) FROM integration.webhook_operator_receipt)<>3 THEN RAISE EXCEPTION 'rejected operation changed mandate';END IF;END \$\$;"
# Real synthetic login must also fail at PostgreSQL's Sandbox CONNECT boundary.
if denied=$(docker exec "$proof_id" psql -X -w -v ON_ERROR_STOP=1 -U 'jayburd-prod-webhook@jayburd-bookkeeping-dev.iam' -d bookkeeping -c 'SELECT 1' 2>&1); then exit 1; fi
[[ "$denied" == *'permission denied for database'* ]]
query "DO \$\$ BEGIN IF NOT EXISTS(SELECT FROM integration.webhook_login_profile WHERE environment='production') OR EXISTS(SELECT FROM integration.webhook_ingress WHERE environment='sandbox') THEN RAISE EXCEPTION 'wrong immutable profile'; END IF; END \$\$;"
echo 'PASS: Production webhook exact target/profile/replay/revocation/ACL/grant denials and actual Sandbox login denial.'
