#!/usr/bin/env bash
# Exercise reviewed login wiring against migrated disposable SQL; never a live DSN or provider.
set -euo pipefail
cd "$(dirname "$0")/.."
postgres='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
migrator='migrate/migrate:v4.20.1@sha256:76cc2074cb6642631f34a898ced71e6aeaa6b1a4d78c4daa743275a22e0c5be7'
proof="jayburd-verifier-bootstrap-$(date +%s)-$$"
test_dir=$(mktemp -d)
network_created=0
volume_created=0
container_created=0
cleanup() {
 if [[ "$container_created" == 1 ]]; then docker rm -fv "$proof" >/dev/null; fi
 if [[ "$volume_created" == 1 ]]; then docker volume rm "$proof" >/dev/null; fi
 if [[ "$network_created" == 1 ]]; then docker network rm "$proof" >/dev/null; fi
 rm -rf "$test_dir"
}
trap cleanup EXIT
trap 'exit 130' INT TERM
docker network create --internal "$proof" >/dev/null
network_created=1
docker volume create "$proof" >/dev/null
volume_created=1
docker run -d --name "$proof" --network "$proof" --network-alias database --mount "type=volume,source=$proof,target=/proof" --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$postgres" >/dev/null
container_created=1
docker cp db/migrations "$proof:/proof/migrations"
docker cp db/bootstrap/verifier-login.sql "$proof:/proof/verifier-login.sql"
ready=0
for ((i=0;i<60;i++)); do
 if docker exec "$proof" pg_isready -U postgres >/dev/null 2>&1; then ready=1; break; fi
 sleep 1
done
[[ "$ready" == 1 ]]
docker exec "$proof" createdb -U postgres bookkeeping
docker run --rm --network "$proof" --mount "type=volume,source=$proof,target=/proof,readonly" "$migrator" -path /proof/migrations -database 'postgres://postgres@database:5432/bookkeeping?sslmode=disable' up >/dev/null 2>&1
# Operator has role administration, not inherited business capabilities or runtime impersonation.
admin() { docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -c "$1" >/dev/null; }
admin 'REVOKE ALL ON DATABASE bookkeeping FROM PUBLIC; CREATE ROLE synthetic_verifier_operator LOGIN CREATEROLE NOINHERIT; CREATE ROLE "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" LOGIN; GRANT CONNECT ON DATABASE bookkeeping TO synthetic_verifier_operator; GRANT bookkeeping_owner TO synthetic_verifier_operator WITH INHERIT FALSE, SET TRUE; GRANT bookkeeping_verifier TO synthetic_verifier_operator WITH ADMIN TRUE, INHERIT FALSE, SET FALSE; GRANT "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" TO synthetic_verifier_operator WITH ADMIN TRUE, INHERIT FALSE, SET FALSE;'
# Model the Google-managed authentication marker absent from plain local PostgreSQL.
admin 'CREATE ROLE cloudsqliamserviceaccount NOLOGIN; GRANT cloudsqliamserviceaccount TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" WITH ADMIN FALSE, INHERIT TRUE, SET TRUE'
bootstrap() { docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U synthetic_verifier_operator -d bookkeeping -v "project_id=${1:-jayburd-bookkeeping-dev}" -f /proof/verifier-login.sql; }
# Fixed sentinel and SQLSTATE establish rejection by the reviewed predicate, not connection failure.
expect_bootstrap_denied() {
 if bootstrap "${1:-jayburd-bookkeeping-dev}" >"$test_dir/out" 2>"$test_dir/err"; then echo 'FAIL: unsafe verifier bootstrap accepted' >&2; exit 1; fi
 grep -Eq 'P0001: verifier_bootstrap_rejected' "$test_dir/err"
}
# Lacking CONNECT grant authority must reject and roll back, never report a wired but unusable login.
expect_bootstrap_denied
admin "DO \$\$ BEGIN IF pg_has_role('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam','bookkeeping_verifier','MEMBER') OR has_database_privilege('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam','bookkeeping','CONNECT') THEN RAISE EXCEPTION 'failed bootstrap retained authority'; END IF; END \$\$;"
admin 'GRANT CONNECT ON DATABASE bookkeeping TO synthetic_verifier_operator WITH GRANT OPTION'
bootstrap >/dev/null
bootstrap >/dev/null
# Predicate replacement preserves owner/ACL/OID, and down restores rejection of the same marker.
predicate_metadata() { docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -c "SELECT oid::text||':'||proowner::text||':'||proacl::text FROM pg_proc WHERE oid='integration.verifier_login(name)'::regprocedure"; }
before_metadata=$(predicate_metadata)
docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -f /proof/migrations/000031_verifier_iam_compatibility.down.sql >/dev/null
expect_bootstrap_denied
[[ "$(predicate_metadata)" == "$before_metadata" ]]
docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -f /proof/migrations/000031_verifier_iam_compatibility.up.sql >/dev/null
[[ "$(predicate_metadata)" == "$before_metadata" ]]
bootstrap >/dev/null
actual_login=$(docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U 'jayburd-dev-verifier@jayburd-bookkeeping-dev.iam' -d bookkeeping -c 'SELECT current_user')
[[ "$actual_login" == 'jayburd-dev-verifier@jayburd-bookkeeping-dev.iam' ]]
expect_bootstrap_denied other-project
docker exec -i "$proof" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping >/dev/null <<'SQL'
DO $$ BEGIN
 IF (SELECT rolconnlimit FROM pg_roles WHERE rolname='jayburd-dev-verifier@jayburd-bookkeeping-dev.iam')<>1
 OR NOT pg_has_role('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam','bookkeeping_verifier','USAGE')
 OR pg_has_role('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam','bookkeeping_verifier','SET')
 THEN RAISE EXCEPTION 'wrong verifier wiring'; END IF;
END $$;
SQL
expect_runtime_denied() {
 if docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U 'jayburd-dev-verifier@jayburd-bookkeeping-dev.iam' -d bookkeeping -c "$1" >"$test_dir/out" 2>"$test_dir/err"; then echo 'FAIL: runtime authority widened' >&2; exit 1; fi
 grep -Eq '42501:' "$test_dir/err"
}
expect_runtime_denied 'SET ROLE bookkeeping_verifier'
expect_runtime_denied 'CREATE TABLE public.denied(x int)'
expect_runtime_denied 'SELECT name FROM ledger.entity'
# The marker exception is exact; test the SQL predicate independently of bootstrap prechecks.
expect_marker_denied() {
 expect_bootstrap_denied
 admin "DO \$\$ BEGIN IF integration.verifier_login('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam') THEN RAISE EXCEPTION 'unsafe marker predicate accepted'; END IF; END \$\$;"
}
admin 'GRANT cloudsqliamserviceaccount TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" WITH ADMIN TRUE, INHERIT TRUE, SET TRUE'
expect_marker_denied
admin 'GRANT cloudsqliamserviceaccount TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" WITH ADMIN FALSE, INHERIT FALSE, SET TRUE'
expect_marker_denied
admin 'GRANT cloudsqliamserviceaccount TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" WITH ADMIN FALSE, INHERIT TRUE, SET TRUE; ALTER ROLE cloudsqliamserviceaccount CREATEDB'
expect_marker_denied
admin 'ALTER ROLE cloudsqliamserviceaccount NOCREATEDB LOGIN'
expect_marker_denied
admin 'ALTER ROLE cloudsqliamserviceaccount NOLOGIN; CREATE ROLE synthetic_marker_parent; GRANT synthetic_marker_parent TO cloudsqliamserviceaccount WITH INHERIT FALSE, SET FALSE'
expect_marker_denied
admin 'REVOKE synthetic_marker_parent FROM cloudsqliamserviceaccount; GRANT SELECT ON ledger.entity TO cloudsqliamserviceaccount'
expect_marker_denied
admin 'REVOKE SELECT ON ledger.entity FROM cloudsqliamserviceaccount; GRANT SELECT(name) ON ledger.entity TO cloudsqliamserviceaccount'
expect_marker_denied
admin 'REVOKE SELECT(name) ON ledger.entity FROM cloudsqliamserviceaccount; GRANT EXECUTE ON FUNCTION integration.verifier_login(name) TO cloudsqliamserviceaccount'
expect_marker_denied
admin 'REVOKE EXECUTE ON FUNCTION integration.verifier_login(name) FROM cloudsqliamserviceaccount; CREATE ROLE synthetic_unrelated_set; GRANT synthetic_unrelated_set TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" WITH ADMIN FALSE, INHERIT TRUE, SET TRUE'
expect_marker_denied
admin 'REVOKE synthetic_unrelated_set FROM "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"'
bootstrap >/dev/null
# SET marker drops the verifier capability and never grants table/DDL access.
expect_runtime_denied 'SET ROLE cloudsqliamserviceaccount; SELECT name FROM ledger.entity'
expect_runtime_denied 'SET ROLE cloudsqliamserviceaccount; CREATE TABLE public.denied(x int)'
expect_runtime_denied "SET ROLE cloudsqliamserviceaccount; SELECT integration.verify_sandbox_fixture('00000000-0000-4000-8000-000000000272',1,'{}','{}')"
docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U synthetic_verifier_operator -d bookkeeping -c 'REVOKE bookkeeping_verifier FROM "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"' >/dev/null
admin 'GRANT SELECT ON ledger.entity TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"'
expect_bootstrap_denied
admin "DO \$\$ BEGIN IF pg_has_role('jayburd-dev-verifier@jayburd-bookkeeping-dev.iam','bookkeeping_verifier','MEMBER') THEN RAISE EXCEPTION 'failed bootstrap retained new capability'; END IF; END \$\$;"
admin 'REVOKE SELECT ON ledger.entity FROM "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"'
bootstrap >/dev/null
admin 'ALTER ROLE "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" CREATEDB'
expect_bootstrap_denied
admin 'ALTER ROLE "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" NOCREATEDB; GRANT bookkeeping_api TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" WITH INHERIT FALSE, SET FALSE'
expect_bootstrap_denied
admin 'REVOKE bookkeeping_api FROM "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"; GRANT SELECT ON ledger.entity TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"'
expect_bootstrap_denied
admin 'REVOKE SELECT ON ledger.entity FROM "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"; GRANT SELECT(name) ON ledger.entity TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"'
expect_bootstrap_denied
admin 'REVOKE SELECT(name) ON ledger.entity FROM "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam"; GRANT bookkeeping_verifier TO "jayburd-dev-verifier@jayburd-bookkeeping-dev.iam" WITH ADMIN TRUE, INHERIT TRUE, SET TRUE'
expect_bootstrap_denied
echo 'PASS: real verifier bootstrap/replay with managed IAM marker; exact SET exception, privilege/parent/ADMIN/NOINHERIT rejection and runtime SET/DDL/financial denials.'
