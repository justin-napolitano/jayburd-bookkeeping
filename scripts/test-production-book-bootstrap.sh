#!/usr/bin/env bash
# Exercise metadata-only operator stages against disposable PostgreSQL with no network.
set -euo pipefail
cd "$(dirname "$0")/.."
image='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
proof="jayburd-production-book-$(date +%s)-$$"
created=0
# Remove only the disposable fixture container owned by this invocation.
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
docker exec "$proof" createdb -U postgres bookkeeping_prod
docker cp db/migrations "$proof:/tmp/migrations"
docker cp db/bootstrap "$proof:/tmp/bootstrap"
for migration in db/migrations/*.up.sql; do
 docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping_prod -f "/tmp/migrations/$(basename "$migration")" >/dev/null
done
# Test helpers use only this container and fixed test database; no inherited DSN.
sql() { docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d bookkeeping_prod -c "$1"; }
# Managed names are synthetic fixture metadata only; no cloud credentials or DSN.
sql 'CREATE ROLE synthetic_operator LOGIN; GRANT bookkeeping_owner TO synthetic_operator WITH INHERIT FALSE, SET TRUE;' >/dev/null
docker exec "$proof" createdb -U postgres bookkeeping
sql 'REVOKE CONNECT ON DATABASE bookkeeping FROM PUBLIC; GRANT CONNECT ON DATABASE bookkeeping TO synthetic_operator; CREATE ROLE "jayburd-prod-api@jayburd-bookkeeping-dev.iam" LOGIN; GRANT bookkeeping_api TO "jayburd-prod-api@jayburd-bookkeeping-dev.iam" WITH INHERIT TRUE, SET FALSE; CREATE ROLE "jayburd-prod-sync@jayburd-bookkeeping-dev.iam" LOGIN; CREATE ROLE "jayburd-prod-publisher@jayburd-bookkeeping-dev.iam" LOGIN;' >/dev/null
docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d bookkeeping_prod -v publisher_user=jayburd-prod-publisher@jayburd-bookkeeping-dev.iam -v sync_user=jayburd-prod-sync@jayburd-bookkeeping-dev.iam -f /tmp/bootstrap/worker-logins.sql >/dev/null
# Fail for the intended invariant, never an unrelated syntax/connection error.
reject() {
 local expected="$1" output
 shift
 if output=$("$@" 2>&1); then echo 'FAIL: denied Production operation succeeded' >&2; exit 1; fi
 [[ "$output" != *fictional-private* ]] || { echo 'FAIL: unsafe synthetic diagnostic' >&2; exit 1; }
 [[ "$output" == *"$expected"* ]] || { printf 'FAIL: expected %s; synthetic diagnostic: %s\n' "$expected" "$output" >&2; exit 1; }
}
currency='00000000-0000-4000-8000-000000000002'
source_release='SIX-list-one:2026-09-17:sha256:33139b438657d1cee116ba737807ea71d19d6de4b90f799a09c56f0cc6a1b0ff'
# Invoke the reference stage with fixture overrides for explicit denial cases.
reference_stage() {
 docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U "${test_user:-synthetic_operator}" -d "${test_db:-bookkeeping_prod}" -v project=jayburd-bookkeeping-dev -v project_number=9088794002 -v currency_id="$currency" -v source_release="$source_release" -v valid_from=2026-09-17 -v valid_to=2027-01-01 "$@" -f /tmp/bootstrap/production-reference.sql
}
test_db=bookkeeping reject production_bootstrap_denied reference_stage
test_user=jayburd-prod-api@jayburd-bookkeeping-dev.iam reject production_bootstrap_denied reference_stage
reject production_bootstrap_denied reference_stage -v project_number=9088794003
reject production_bootstrap_denied reference_stage -v project=fictional-private
reject production_reference_rejected reference_stage -v source_release=fictional-private
reject production_reference_rejected reference_stage -v valid_from=2001-01-01
reject production_reference_rejected reference_stage -v currency_id=00000000-0000-0000-0000-000000000000
[[ "$(sql "SELECT count(*) FROM reference.currency WHERE valid_from=DATE '2026-09-17'")" == 0 ]]
first=$(reference_stage)
[[ "$first" == 'PASS: exact Production reference metadata installed' && "$(reference_stage)" == "$first" ]]
[[ "$(sql "SELECT alphabetic_code||'|'||numeric_code||'|'||minor_unit||'|'||valid_from||'|'||valid_to||'|'||source_release FROM reference.currency WHERE currency_version_id='$currency'")" == "USD|840|2|2026-09-17|2027-01-01|$source_release" ]]
reject 'overlapping currency versions' reference_stage -v currency_id=00000000-0000-4000-8000-000000000099
sql "SET ROLE bookkeeping_owner; INSERT INTO reference.currency VALUES('00000000-0000-4000-8000-000000000098','USD','840',2,'2027-01-01','2027-01-02','fictional-reference');" >/dev/null
reject production_reference_conflict reference_stage -v currency_id=00000000-0000-4000-8000-000000000098
echo 'PASS TestProductionReferenceExactReplayAndDenials'
actor=$(sql "SET ROLE bookkeeping_identity_operator; SELECT ledger.enroll_principal('https://securetoken.google.com/fictional-project','fictional-owner');")
# Independently authored Q4 fixture uses ordinary setup and its creator receipt.
create_book() {
 sql "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL ROLE bookkeeping_owner; SELECT body::text FROM ledger.setup_book('$actor','$1',gen_random_uuid(),'{\"entity_name\":\"Fictional Personal\",\"entity_kind\":\"personal\",\"book_name\":\"Personal\",\"starts_on\":\"2026-10-01\",\"ends_before\":\"2027-01-01\",\"functional_currency_version_id\":\"$currency\",\"accounts\":[{\"code\":\"1000\",\"name\":\"Checking\",\"category\":\"asset\",\"normal_side\":\"debit\",\"monetary\":true},{\"code\":\"3000\",\"name\":\"Owner equity\",\"category\":\"equity\",\"normal_side\":\"credit\",\"monetary\":false}]}'); COMMIT;" >/dev/null
 entity=$(sql "SELECT entity_id FROM ledger.setup_receipt WHERE principal_id='$actor' AND idempotency_key='$1'")
 book=$(sql "SELECT book_id FROM ledger.setup_receipt WHERE principal_id='$actor' AND idempotency_key='$1'")
}
# Invoke only the synthetic creator's exact target; overrides exercise guard failures.
book_stage() {
 docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U "${test_user:-synthetic_operator}" -d "${test_db:-bookkeeping_prod}" -v project=jayburd-bookkeeping-dev -v project_number=9088794002 -v actor_id="$actor" -v entity_id="$entity" -v book_id="$book" -v worker_login=jayburd-prod-sync@jayburd-bookkeeping-dev.iam -v publisher_login=jayburd-prod-publisher@jayburd-bookkeeping-dev.iam -v scheduler_paused=true "$@" -f /tmp/bootstrap/production-book.sql
}
create_book fictional-personal
first_entity=$entity; first_book=$book
reject production_bootstrap_denied book_stage -v scheduler_paused=false
reject production_bootstrap_denied book_stage -v worker_login=jayburd-dev-sync@jayburd-bookkeeping-dev.iam
reject production_bootstrap_denied book_stage -v project_number=9088794003
test_db=bookkeeping reject production_bootstrap_denied book_stage
test_user=jayburd-prod-api@jayburd-bookkeeping-dev.iam reject production_bootstrap_denied book_stage
reject production_creator_denied book_stage -v book_id=00000000-0000-4000-8000-000000000099
sql 'GRANT CONNECT ON DATABASE bookkeeping TO "jayburd-prod-sync@jayburd-bookkeeping-dev.iam";' >/dev/null
reject production_worker_denied book_stage
sql 'REVOKE CONNECT ON DATABASE bookkeeping FROM "jayburd-prod-sync@jayburd-bookkeeping-dev.iam";' >/dev/null
[[ "$(sql "SELECT count(*) FROM ledger.principal WHERE issuer LIKE 'urn:jayburd:%'")" == 0 ]]
first=$(book_stage)
[[ "$first" == "$(book_stage)" && "$first" == *PASS* ]]
[[ "$(sql "SELECT environment||'|'||enabled||'|'||revision||'|'||api_login||'|'||worker_login FROM integration.provider_scope WHERE book_id='$book'")" == 'production|true|1|jayburd-prod-api@jayburd-bookkeeping-dev.iam|jayburd-prod-sync@jayburd-bookkeeping-dev.iam' ]]
[[ "$(sql "SELECT bank_key||'|'||link_key FROM integration.provider_scope WHERE book_id='$book'")" == 'projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/bank-credentials|projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/hosted-link' ]]
[[ "$(sql "SELECT count(*) FROM integration.sync_delivery_binding WHERE book_id='$book' AND enabled")" == 1 ]]
[[ "$(sql "SELECT count(*) FROM integration.connection_receipt WHERE book_id='$book' AND operation='provider_scope'")" == 1 ]]
echo 'PASS TestProductionCreatorScopeAndDeliveryExactReplay'
# Only synthetic fixture teardown/restoration changes these states; product replay must reject them.
service=$(sql "SELECT service_principal FROM integration.connection_worker WHERE book_id='$book'")
sql "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id='$service';" >/dev/null
reject production_service_denied book_stage
sql "UPDATE ledger.principal SET disabled_at=NULL WHERE principal_id='$service';" >/dev/null
sql 'REVOKE bookkeeping_sync FROM "jayburd-prod-sync@jayburd-bookkeeping-dev.iam";' >/dev/null
reject production_worker_denied book_stage
sql 'GRANT bookkeeping_sync TO "jayburd-prod-sync@jayburd-bookkeeping-dev.iam" WITH INHERIT TRUE, SET FALSE;' >/dev/null
for binding_table in connection_worker sync_delivery_binding; do
 revision_change=''; [[ "$binding_table" != sync_delivery_binding ]] || revision_change=',binding_revision=binding_revision+1'
 sql "UPDATE integration.$binding_table SET enabled=false$revision_change WHERE book_id='$book';" >/dev/null
 reject production_binding_conflict book_stage
 [[ "$(sql "SELECT enabled FROM integration.$binding_table WHERE book_id='$book'")" == f ]]
 sql "UPDATE integration.$binding_table SET enabled=true$revision_change WHERE book_id='$book';" >/dev/null
done
sql "UPDATE integration.provider_scope SET enabled=false,revision=revision+1 WHERE book_id='$book';" >/dev/null
reject production_scope_disabled book_stage
[[ "$(sql "SELECT enabled FROM integration.provider_scope WHERE book_id='$book'")" == f ]]
sql "UPDATE integration.provider_scope SET enabled=true,revision=revision+1 WHERE book_id='$book';" >/dev/null
echo 'PASS TestProductionDisabledScopeServiceAndBindingReplayDenied'
# Force the last mutation to fail: preceding scope, grants and receipts must roll back.
create_book fictional-late-failure
sql "CREATE FUNCTION integration.synthetic_late_failure() RETURNS trigger LANGUAGE plpgsql AS \$\$ BEGIN IF NEW.book_id='$book' THEN RAISE EXCEPTION 'synthetic late failure'; END IF; RETURN NEW; END \$\$; CREATE TRIGGER synthetic_late_failure BEFORE INSERT ON integration.sync_delivery_binding FOR EACH ROW EXECUTE FUNCTION integration.synthetic_late_failure();" >/dev/null
reject 'synthetic late failure' book_stage
[[ "$(sql "SELECT (SELECT count(*) FROM integration.provider_scope WHERE book_id='$book')+(SELECT count(*) FROM integration.connection_worker WHERE book_id='$book')+(SELECT count(*) FROM integration.connection_receipt WHERE book_id='$book')+(SELECT count(*) FROM ledger.principal WHERE subject='acquisition:$book')+(SELECT count(*) FROM ledger.membership WHERE entity_id='$entity' AND permission='import')")" == 0 ]]
sql 'DROP TRIGGER synthetic_late_failure ON integration.sync_delivery_binding; DROP FUNCTION integration.synthetic_late_failure();' >/dev/null
echo 'PASS TestProductionLateScopeAndBindingRollback'
sql "BEGIN ISOLATION LEVEL SERIALIZABLE; SET LOCAL ROLE bookkeeping_owner; SELECT integration.configure_provider_scope('$actor','$entity','$book','fictional-wrong-key','{\"environment\":\"production\",\"api_login\":\"jayburd-prod-api@jayburd-bookkeeping-dev.iam\",\"worker_login\":\"jayburd-prod-sync@jayburd-bookkeeping-dev.iam\",\"bank_key\":\"projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/other-bank-key\",\"link_key\":\"projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/hosted-link\",\"enabled\":true}'); COMMIT;" >/dev/null
reject connection_rejected book_stage
[[ "$(sql "SELECT count(*) FROM ledger.principal WHERE subject='acquisition:$book'")" == 0 ]]
echo 'PASS TestProductionImmutableKeyConflictRollback'
entity=$first_entity; book=$first_book
sql "UPDATE ledger.membership SET revoked_at=clock_timestamp() WHERE entity_id='$entity' AND principal_id='$actor' AND permission='import';" >/dev/null
reject production_revoked_grant book_stage
[[ "$(sql "SELECT count(*) FROM ledger.membership WHERE entity_id='$entity' AND principal_id='$actor' AND permission='import' AND revoked_at IS NOT NULL")" == 1 ]]
sql "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id='$actor';" >/dev/null
reject production_actor_denied book_stage
[[ "$(sql "SELECT (SELECT count(*) FROM integration.sync_item)+(SELECT count(*) FROM integration.connection_attempt)+(SELECT count(*) FROM ledger.journal_entry)")" == 0 ]]
[[ "$(sql "SELECT has_function_privilege('jayburd-prod-api@jayburd-bookkeeping-dev.iam','integration.configure_provider_scope(uuid,uuid,uuid,text,text)','EXECUTE') OR has_function_privilege('jayburd-prod-sync@jayburd-bookkeeping-dev.iam','integration.configure_provider_scope(uuid,uuid,uuid,text,text)','EXECUTE')")" == f ]]
echo 'PASS TestProductionRevocationNoProviderOrJournalEffects'
