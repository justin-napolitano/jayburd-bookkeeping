#!/usr/bin/env bash
# Exercise the finite operator stage in a disposable synthetic, networkless DB.
set -euo pipefail
cd "$(dirname "$0")/.."
image='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
proof="jayburd-activation453-$(date +%s)-$$"
created=0
cleanup() { if [[ "$created" == 1 ]]; then docker rm -fv "$proof" >/dev/null; fi; }
trap cleanup EXIT
trap 'exit 130' INT TERM
docker run -d --name "$proof" --network none --tmpfs /var/lib/postgresql/data --env POSTGRES_HOST_AUTH_METHOD=trust "$image" >/dev/null
created=1
for ((i=0;i<60;i++)); do
 if docker exec "$proof" pg_isready -U postgres >/dev/null 2>&1; then break; fi
 sleep 1
done
docker exec "$proof" createdb -U postgres bookkeeping
docker cp db/migrations "$proof:/tmp/migrations"
docker cp db/bootstrap/sandbox-private-owner.sql "$proof:/tmp/activation.sql"
for migration in db/migrations/*.up.sql; do
 docker exec "$proof" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -f "/tmp/migrations/$(basename "$migration")" >/dev/null
done
sql() { docker exec "$proof" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d bookkeeping -c "$1"; }
sql 'CREATE ROLE synthetic_operator LOGIN NOINHERIT; GRANT bookkeeping_owner TO synthetic_operator WITH INHERIT FALSE, SET TRUE; CREATE ROLE synthetic_api LOGIN IN ROLE bookkeeping_api' >/dev/null
actor='00000000-0000-4000-8000-000000000001'
other='00000000-0000-4000-8000-000000000002'
evidence='00000000-0000-4000-8000-000000000003'
sql "INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('$actor','https://securetoken.google.com/jayburd-bookkeeping-dev','fictional-owner'),('$other','https://securetoken.google.com/foreign','fictional-foreign')" >/dev/null
expiry=$(sql "SELECT to_char(clock_timestamp()+interval '30 minutes','YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"')")
body="{\"expected_revision\":null,\"capabilities\":[\"read\",\"acquire\",\"classify\"],\"evidence_ref\":\"$evidence\"}"
activate() {
 docker exec "$proof" psql -X -qAt -U synthetic_operator -d bookkeeping \
 -v project=jayburd-bookkeeping-dev -v subject=fictional-owner -v principal_id="$actor" \
 -v idempotency_key=reviewed-activation -v body="$body" -v expires_at="$expiry" "$@" -f /tmp/activation.sql
}
reject() {
 local output
 if output=$(activate "$@" 2>&1); then echo 'FAIL: activation denial succeeded' >&2; exit 1; fi
 case "$output" in *fictional-owner*|*fictional-foreign*) echo 'FAIL: unsafe activation diagnostic' >&2; exit 1;; esac
 case "$output" in *sandbox_activation_denied*) :;; *) echo 'FAIL: unexpected denial' >&2; exit 1;; esac
}
first=$(activate)
[ "$(activate)" = "$first" ]
case "$first" in *fictional-owner*) exit 1;; esac
[ "$(sql 'SELECT count(*) FROM banking.collection_workspace')" = 1 ]
[ "$(sql "SELECT count(*) FROM integration.collection_receipt WHERE operation='provision'")" = 1 ]
echo 'PASS TestActivationOperatorInitialReplay'
reject -v project=production
reject -v principal_id="$other"
reject -v subject=not-enrolled
reject -v subject=fictional-foreign -v principal_id="$other"
reject -v expires_at=2000-01-01T00:00:00Z
reject -v expires_at=2099-01-01T00:00:00Z
reject -U synthetic_api
reject -v body="{\"expected_revision\":\"1\",\"capabilities\":[\"read\"],\"evidence_ref\":\"$evidence\"}"
reject -v body='{"expected_revision":null,"capabilities":[],"evidence_ref":"invalid"}'
sql "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id='$actor'" >/dev/null
reject
sql "UPDATE ledger.principal SET disabled_at=NULL WHERE principal_id='$actor'" >/dev/null
[ "$(sql 'SELECT count(*) FROM banking.collection_workspace')" = 1 ]
[ "$(sql "SELECT count(*) FROM integration.collection_receipt WHERE operation='provision'")" = 1 ]
echo 'PASS TestActivationOperatorIdentityScopeAndAtomicDeny'
# This delay belongs only to this disposable fixture. It lets approval expire
# after the provision function has written, proving precommit expiry rolls back.
fresh='00000000-0000-4000-8000-000000000004'
sql "INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES('$fresh','https://securetoken.google.com/jayburd-bookkeeping-dev','fictional-expiry')" >/dev/null
sql 'CREATE FUNCTION public.synthetic_activation_delay()RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(3.2);RETURN NEW;END $$; CREATE TRIGGER synthetic_activation_delay BEFORE INSERT ON banking.collection_workspace FOR EACH ROW EXECUTE FUNCTION public.synthetic_activation_delay()' >/dev/null
short=$(sql "SELECT to_char(clock_timestamp()+interval '3 seconds','YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"')")
reject -v subject=fictional-expiry -v principal_id="$fresh" -v expires_at="$short"
[ "$(sql "SELECT count(*) FROM banking.collection_workspace WHERE owner_principal_id='$fresh'")" = 0 ]
echo 'PASS TestActivationOperatorPrecommitExpiryRollback'
[ "$(sql "SELECT has_function_privilege('synthetic_api','banking.provision_collection_owner(uuid,text,text)','EXECUTE')")" = f ]
echo 'PASS TestActivationOperatorRuntimeProvisionDenied'
