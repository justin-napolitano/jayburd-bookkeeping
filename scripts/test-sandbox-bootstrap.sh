#!/usr/bin/env bash
# Exercise metadata-only operator stages against disposable PostgreSQL with no network.
set -euo pipefail
cd "$(dirname "$0")/.."
image='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
proof="jayburd-sandbox-bootstrap-$(date +%s)-$$"
created=0
cleanup() { if [[ "$created" == 1 ]]; then docker rm -fv "$proof" >/dev/null; fi; }
trap cleanup EXIT
trap 'exit 130' INT TERM
docker run -d --name "$proof" --network none --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$image" >/dev/null
created=1
for ((i=0;i<60;i++)); do
 if docker exec "$proof" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then break; fi
 sleep 1
done
docker exec "$proof" pg_isready -h 127.0.0.1 -U postgres >/dev/null
docker exec "$proof" createdb -U postgres bookkeeping
docker cp db/migrations "$proof:/tmp/migrations"
docker cp db/bootstrap "$proof:/tmp/bootstrap"
for migration in db/migrations/*.up.sql; do
 docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -f "/tmp/migrations/$(basename "$migration")" >/dev/null
done
# Test helpers use only this container and fixed test database; no inherited DSN.
sql() { docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -c "$1"; }
sql 'CREATE ROLE synthetic_operator LOGIN; GRANT bookkeeping_owner,bookkeeping_identity_operator TO synthetic_operator WITH INHERIT FALSE, SET TRUE; CREATE ROLE synthetic_api LOGIN; GRANT bookkeeping_api TO synthetic_api WITH INHERIT TRUE, SET FALSE;' >/dev/null
sql 'CREATE ROLE "jayburd-dev-sync@jayburd-bookkeeping-dev.iam" LOGIN; CREATE ROLE "jayburd-dev-publisher@jayburd-bookkeeping-dev.iam" LOGIN; GRANT bookkeeping_sync TO "jayburd-dev-sync@jayburd-bookkeeping-dev.iam" WITH INHERIT TRUE, SET FALSE; GRANT bookkeeping_publisher TO "jayburd-dev-publisher@jayburd-bookkeeping-dev.iam" WITH INHERIT TRUE, SET FALSE;' >/dev/null
# A denied operation must fail for the intended boundary, never an unrelated connection/syntax error.
reject() {
 local expected="$1" output
 shift
 if output=$("$@" 2>&1); then echo 'FAIL: denied operation succeeded' >&2; exit 1; fi
 [[ "$output" == *"$expected"* ]] || { echo 'FAIL: denial reason mismatch' >&2; exit 1; }
 [[ "$output" != *fictional-owner* && "$output" != *fictional-private-subject* ]] || { echo 'FAIL: subject in diagnostic' >&2; exit 1; }
}
source_release='SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff'
currency='00000000-0000-4000-8000-000000000002'
identity() {
 docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U "${test_user:-synthetic_operator}" -d bookkeeping \
 -v project=jayburd-bookkeeping-dev -v subject=fictional-owner -v currency_id="$currency" -v source_release="$source_release" \
 -v valid_from=2026-09-17 -v valid_to=2027-01-01 "$@" -f /tmp/bootstrap/sandbox-identity.sql
}
first=$(identity)
[[ "$(identity)" == "$first" && "$first" != *fictional-owner* ]]
if rejection=$(identity -v source_release=wrong -v subject=fictional-private-subject 2>&1); then exit 1; fi
[[ "$rejection" != *fictional-private-subject* ]]
actor=$(sql "SELECT principal_id FROM ledger.principal WHERE subject='fictional-owner'")
[[ "$(sql 'SELECT count(*) FROM audit.identity_action')" == 1 ]]
echo 'PASS TestIdentityInitialReplay'
[[ "$(sql "SELECT count(*) FROM reference.currency WHERE currency_version_id='3a184659-cf9b-4b80-b140-3f7a7796cbbd' AND alphabetic_code='USD' AND numeric_code='840' AND minor_unit=2 AND valid_from=DATE '2001-01-01' AND valid_to=DATE '2026-09-17'")" == 1 ]]
echo 'PASS TestBootstrapHistoricalCurrencyReplay'
# Rejected source/version and alternate-ID overlap cannot leave partial enrollment.
reject sandbox_reference_rejected identity -v source_release=synthetic-fixture
reject "overlapping currency versions" identity -v currency_id=00000000-0000-4000-8000-000000000099 -v subject=not-enrolled
[[ "$(sql "SELECT count(*) FROM ledger.principal WHERE subject='not-enrolled'")" == 0 ]]
echo 'PASS TestReferenceConflictRollback'
test_user=synthetic_api reject sandbox_bootstrap_denied identity
sql "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id='$actor'" >/dev/null
reject ledger_operation_rejected identity
sql "UPDATE ledger.principal SET disabled_at=NULL WHERE principal_id='$actor'" >/dev/null
# This restore is fixture-only; product scripts never reactivate principals.
echo 'PASS TestIdentityDisabledAndRuntimeDenied'
# The actual setup function creates the book, accounts and creator receipt.
setup=$(sql "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL ROLE bookkeeping_owner; SELECT body::text FROM ledger.setup_book('$actor','fictional-setup',gen_random_uuid(),'{\"entity_name\":\"Fictional Test LLC\",\"entity_kind\":\"legal\",\"book_name\":\"Sandbox\",\"starts_on\":\"2026-09-17\",\"ends_before\":\"2027-01-01\",\"functional_currency_version_id\":\"$currency\",\"accounts\":[{\"code\":\"1000\",\"name\":\"Cash\",\"category\":\"asset\",\"normal_side\":\"debit\",\"monetary\":true},{\"code\":\"3000\",\"name\":\"Capital\",\"category\":\"equity\",\"normal_side\":\"credit\",\"monetary\":false}]}'); COMMIT;")
entity=$(sql "SELECT entity_id FROM ledger.setup_receipt WHERE principal_id='$actor'")
book=$(sql "SELECT book_id FROM ledger.setup_receipt WHERE principal_id='$actor'")
book_stage() {
 docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U "${test_user:-synthetic_operator}" -d bookkeeping \
 -v project=jayburd-bookkeeping-dev -v actor_id="$actor" -v entity_id="$entity" -v book_id="$book" \
 -v worker_login=jayburd-dev-sync@jayburd-bookkeeping-dev.iam -v publisher_login=jayburd-dev-publisher@jayburd-bookkeeping-dev.iam \
 -v scheduler_paused=true "$@" -f /tmp/bootstrap/sandbox-book.sql
}
# Missing prerequisite, cross-entity and wrong identity fail before any service membership.
reject sandbox_creator_denied book_stage -v entity_id=00000000-0000-4000-8000-000000000099
reject sandbox_bootstrap_denied book_stage -v worker_login=synthetic_api
reject sandbox_bootstrap_denied book_stage -v scheduler_paused=false
sql 'REVOKE bookkeeping_sync FROM "jayburd-dev-sync@jayburd-bookkeeping-dev.iam"' >/dev/null
reject sandbox_worker_denied book_stage
sql 'GRANT bookkeeping_sync TO "jayburd-dev-sync@jayburd-bookkeeping-dev.iam" WITH INHERIT TRUE, SET FALSE' >/dev/null
[[ "$(sql "SELECT count(*) FROM ledger.principal WHERE issuer LIKE 'urn:jayburd:%'")" == 0 ]]
echo 'PASS TestBookScopeWorkerPrerequisites'
# Force a late transaction failure after principal/grants creation; none may survive.
sql "CREATE FUNCTION public.reject_delivery() RETURNS trigger LANGUAGE plpgsql AS \$\$ BEGIN RAISE EXCEPTION 'fixture failure'; END \$\$; CREATE TRIGGER fixture_failure BEFORE INSERT ON integration.sync_delivery_binding FOR EACH ROW EXECUTE FUNCTION public.reject_delivery();" >/dev/null
reject "fixture failure" book_stage
[[ "$(sql 'SELECT count(*) FROM integration.connection_worker')" == 0 ]]
[[ "$(sql "SELECT count(*) FROM ledger.membership WHERE permission IN ('connect','import')")" == 0 ]]
[[ "$(sql "SELECT count(*) FROM ledger.principal WHERE issuer LIKE 'urn:jayburd:%'")" == 0 ]]
sql 'DROP TRIGGER fixture_failure ON integration.sync_delivery_binding; DROP FUNCTION public.reject_delivery()' >/dev/null
echo 'PASS TestLateFailureAtomicRollback'
receipt=$(book_stage)
[[ "$(book_stage)" == "$receipt" ]]
[[ "$(sql 'SELECT count(*) FROM integration.connection_receipt')" == 3 ]]
[[ "$(sql 'SELECT binding_revision FROM integration.sync_delivery_binding')" == 1 ]]
echo 'PASS TestBookInitialReplay'
# The owner lookup returns exactly the active binding for this book, without exposing another scope.
lookup_binding() {
 docker exec "$proof" psql -X -qAt -w -v ON_ERROR_STOP=1 -U synthetic_operator -d bookkeeping \
  -v entity_id="$1" -v book_id="$2" -f /tmp/bootstrap/worker-binding-id.sql
}
[[ "$(lookup_binding "$entity" "$book")" == "$(sql 'SELECT binding_id FROM integration.sync_delivery_binding')" ]]
[[ -z "$(lookup_binding "$entity" '00000000-0000-4000-8000-000000000099')" ]]
[[ -z "$(lookup_binding '00000000-0000-4000-8000-000000000099' "$book")" ]]
echo 'PASS TestScopedWorkerBindingLookup'
service=$(sql "SELECT principal_id FROM ledger.principal WHERE issuer LIKE 'urn:jayburd:%'")
for principal in "$actor" "$service"; do
 sql "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id='$principal'" >/dev/null
 if [[ "$principal" == "$actor" ]]; then reject sandbox_actor_denied book_stage; else reject sandbox_service_denied book_stage; fi
 sql "UPDATE ledger.principal SET disabled_at=NULL WHERE principal_id='$principal'" >/dev/null
done
sql "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE principal_id='$service' AND permission='import'" >/dev/null
reject sandbox_revoked_grant book_stage
[[ "$(sql "SELECT count(*) FROM ledger.membership WHERE principal_id='$service' AND permission='import' AND revoked_at IS NOT NULL")" == 1 ]]
test_user=synthetic_api reject sandbox_bootstrap_denied book_stage
reject 'permission denied' docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U synthetic_api -d bookkeeping -c 'CREATE TABLE public.forbidden(id int)'
reject "permission denied" docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U synthetic_api -d bookkeeping -c "SELECT ledger.enroll_principal('https://securetoken.google.com/jayburd-bookkeeping-dev','not-authorized')"
echo 'PASS TestBookDisabledRevokedAndRuntimeDenied'

# Fixture restoration only: independently prove both disabled binding states cannot be reactivated.
sql "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id='$service' AND permission='import'" >/dev/null
for binding in connection_worker sync_delivery_binding; do
 for enabled in false true; do
  if [[ "$binding" == connection_worker ]]; then
   operation=set_connection_worker
   payload="{\"service_principal_id\":\"$service\",\"worker_login\":\"jayburd-dev-sync@jayburd-bookkeeping-dev.iam\",\"enabled\":$enabled}"
  else
   operation=enroll_sync_delivery
   payload="{\"service_principal_id\":\"$service\",\"worker_login\":\"jayburd-dev-sync@jayburd-bookkeeping-dev.iam\",\"publisher_login\":\"jayburd-dev-publisher@jayburd-bookkeeping-dev.iam\",\"enabled\":$enabled}"
  fi
  sql "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL ROLE bookkeeping_owner; SELECT integration.$operation('$actor','$entity','$book','fixture-$binding-$enabled','$payload'); COMMIT;" >/dev/null
  if [[ "$enabled" == false ]]; then
   reject sandbox_binding_conflict book_stage
   [[ "$(sql "SELECT count(*) FROM integration.$binding WHERE NOT enabled")" == 1 ]]
  fi
 done
done
echo 'PASS TestDisabledBindingReplayDenied'
