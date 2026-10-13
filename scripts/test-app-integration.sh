#!/usr/bin/env bash
# Real browser/Next/Go/PostgreSQL proof, with synthetic identity confined to test binaries.
set -euo pipefail
demo=false
demo_test=false
case "${1:-}" in --demo) demo=true ;; --demo-test) demo=true; demo_test=true ;; '') ;; *) echo 'Usage: test-app-integration.sh [--demo|--demo-test]' >&2; exit 1 ;; esac
proof_timeout=600s
if [[ "$demo" == true ]]; then proof_timeout=0; fi
cd "$(dirname "$0")/.."
POSTGRES_IMAGE='postgres:17.11-alpine3.24@sha256:b0f9560a2de083e2cc7382e75f808c7381a32852a7ec49117deedb300e552b24'
MIGRATE_IMAGE='migrate/migrate:v4.20.1@sha256:76cc2074cb6642631f34a898ced71e6aeaa6b1a4d78c4daa743275a22e0c5be7'
proof_id="jayburd-app-$(date +%s)-$$"
network_created=0 volume_created=0 database_created=0 api_created=0 next_pid='' bridge_pid=''
proof_dir=$(mktemp -d /tmp/jayburd-app-proof.XXXXXX)
# Resources belong only to this invocation; never inspect another database or credential.
cleanup() {
 if [[ -n "$next_pid" ]]; then kill "$next_pid" 2>/dev/null || true; wait "$next_pid" 2>/dev/null || true; fi
 if [[ -n "$bridge_pid" ]]; then kill "$bridge_pid" 2>/dev/null || true; wait "$bridge_pid" 2>/dev/null || true; fi
 if [[ "$api_created" == 1 ]]; then docker rm -fv "$proof_id-api" >/dev/null; fi
 if [[ "$database_created" == 1 ]]; then docker rm -fv "$proof_id" >/dev/null; fi
 if [[ "$volume_created" == 1 ]]; then docker volume rm "$proof_id" >/dev/null; fi
 if [[ "$network_created" == 1 ]]; then docker network rm "$proof_id" >/dev/null; fi
 rm -rf "$proof_dir"
}
trap cleanup EXIT
trap 'exit 130' INT TERM
# Next loads dotenv automatically: refuse file presence without opening any contents.
for file in web/.env web/.env.*; do
 if [[ -e "$file" ]]; then echo 'BLOCKED: browser proof requires a credential-free checkout' >&2; exit 1; fi
done
# Hold all listeners together during the collision check; readiness also checks owned PIDs/marker.
node -e 'const net=require("node:net"); const servers=[]; let count=0; for(const port of [39081,39082,39083]) { const server=net.createServer(); servers.push(server); server.on("error",()=>{console.error("Synthetic proof port unavailable"); process.exit(1)}); server.listen(port,"127.0.0.1",()=>{if(++count===3)for(const s of servers)s.close()}); }'
bash scripts/migrate.sh check
go test -tags integration ./internal/server -run '^TestDemoCentsOracle$' -count=1
docker pull "$POSTGRES_IMAGE" >/dev/null
docker pull "$MIGRATE_IMAGE" >/dev/null
docker network create --internal "$proof_id" >/dev/null; network_created=1
docker volume create "$proof_id" >/dev/null; volume_created=1
docker run -d --name "$proof_id" --network "$proof_id" --network-alias database \
 --mount "type=volume,source=$proof_id,target=/proof" --tmpfs /var/lib/postgresql/data \
 --env POSTGRES_HOST_AUTH_METHOD=trust "$POSTGRES_IMAGE" >/dev/null
database_created=1
docker cp db/migrations "$proof_id:/proof/migrations"
ready=0
for ((i=0;i<60;i++)); do
 if docker exec "$proof_id" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then ready=1; break; fi
 sleep 1
done
[[ "$ready" == 1 ]] || { echo 'Synthetic database not ready' >&2; exit 1; }
docker exec "$proof_id" createdb -U postgres app_proof
docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" \
 "$MIGRATE_IMAGE" -path /proof/migrations -database 'postgres://postgres@database:5432/app_proof?sslmode=disable' up
case "$(docker exec "$proof_id" uname -m)" in
 aarch64) proof_arch=arm64 ;; x86_64) proof_arch=amd64 ;; *) exit 1 ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH="$proof_arch" go test -c -tags integration -o "$proof_dir/app.test" ./internal/server
docker cp "$proof_dir/app.test" "$proof_id:/proof/app.test"
# Separate disposable database proves generated imports before serving the browser fixture.
docker run --rm --network "$proof_id" --mount "type=volume,source=$proof_id,target=/proof,readonly" --entrypoint /proof/app.test "$POSTGRES_IMAGE" -demo-acquisition-proof -test.run='^TestDemoAcquisitionPostgres$' -test.v -test.timeout=180s
docker run -d --name "$proof_id-api" --network "$proof_id" \
 --mount "type=volume,source=$proof_id,target=/proof,readonly" --entrypoint /proof/app.test \
 "$POSTGRES_IMAGE" -app-proof -app-proof-demo="$demo" -app-proof-marker="$proof_id" -test.run='^TestAppIntegrationServer$' -test.timeout="$proof_timeout" >/dev/null
api_created=1
# Raw HTTP bridge makes daemon location irrelevant; no database socket is exposed.
go test -c -tags integration -o "$proof_dir/bridge.test" ./internal/server
"$proof_dir/bridge.test" -app-bridge="$proof_id-api" -test.run='^TestAppIntegrationBridge$' -test.timeout="$proof_timeout" >"$proof_dir/bridge.log" 2>&1 &
bridge_pid=$!
ready=0
for ((i=0;i<60;i++)); do
 if ! kill -0 "$bridge_pid" 2>/dev/null; then break; fi
 if [[ "$(docker inspect --format '{{.State.Running}}' "$proof_id-api")" != true ]]; then break; fi
 if curl --fail --silent --max-time 5 -D "$proof_dir/api.headers" http://127.0.0.1:39081/readyz >/dev/null && grep -qi "^x-jayburd-proof: $proof_id" "$proof_dir/api.headers"; then ready=1; break; fi
 sleep 1
done
[[ "$ready" == 1 ]] || { cat "$proof_dir/bridge.log"; docker logs "$proof_id-api"; exit 1; }
# Explicit local review mode reuses the real SQL/API fixture with a separate UI bundle.
if [[ "$demo" == true ]]; then
 (cd web && exec env -i PATH="$PATH" node node_modules/vite/bin/vite.js --config demo/vite.config.ts) &
 next_pid=$!
 ready=0
 for ((i=0;i<30;i++)); do
  if ! kill -0 "$next_pid" 2>/dev/null; then break; fi
  status=$(curl --silent --max-time 2 -D "$proof_dir/demo.headers" -o /dev/null -w '%{http_code}' http://127.0.0.1:39083/v1/workspace || true)
  if [[ "$status" == 401 ]] && grep -qi "^x-jayburd-proof: $proof_id" "$proof_dir/demo.headers"; then ready=1; break; fi
  sleep 1
 done
 [[ "$ready" == 1 ]] || { echo 'Synthetic review UI not ready' >&2; exit 1; }
 if [[ "$demo_test" == true ]]; then
  (cd web && env -i PATH="$PATH" HOME="$HOME" node node_modules/playwright/cli.js test --config playwright-demo.config.ts)
  [[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c "SELECT count(*) FROM ledger.journal_entry j JOIN ledger.entity e USING(entity_id) WHERE e.name LIKE 'Volume Lab %'")" == 1000 ]]
  [[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c 'SELECT count(*) FROM banking.source_observation')" == 3000 ]]
  [[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c 'SELECT count(*) FROM banking.review_posting')" == 1 ]]
  [[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c "SELECT count(*) FROM ledger.journal_entry j JOIN ledger.entity e USING(entity_id) WHERE e.name='Fictional Harbor Design LLC'")" == 1 ]]
  echo 'PASS: 3000 generated imports, one atomic source-linked posting, exact cents and recovery'
  exit 0
 fi
 echo 'Synthetic demo: http://localhost:39083 — Ctrl+C to stop; make demo to reset.'
 wait "$next_pid"
 exit 0
fi
# Build an unconfigured production UI. Only explicit public transport settings reach Next.
(cd web && env -i PATH="$PATH" NEXT_TELEMETRY_DISABLED=1 BOOKKEEPING_API_MODE=loopback BOOKKEEPING_PUBLIC_ORIGIN=http://localhost:39083 BOOKKEEPING_API_ORIGIN=http://127.0.0.1:39081 node node_modules/next/dist/bin/next build --webpack)
(cd web && exec env -i PATH="$PATH" NEXT_TELEMETRY_DISABLED=1 BOOKKEEPING_API_MODE=loopback BOOKKEEPING_PUBLIC_ORIGIN=http://localhost:39083 BOOKKEEPING_API_ORIGIN=http://127.0.0.1:39081 node node_modules/next/dist/bin/next start --hostname 127.0.0.1 --port 39082) >"$proof_dir/next.log" 2>&1 &
next_pid=$!
ready=0
for ((i=0;i<60;i++)); do
 if ! kill -0 "$next_pid" 2>/dev/null; then break; fi
 status=$(curl --silent --max-time 5 -D "$proof_dir/next.headers" -o /dev/null -w '%{http_code}' http://127.0.0.1:39082/v1/workspace || true)
 if [[ "$status" == 401 ]]; then ready=1; break; fi
 sleep 1
done
[[ "$ready" == 1 ]] || { cat "$proof_dir/next.log"; exit 1; }
# The ordinary auth provider remains unconfigured; only the test component injects a fictional token.
(cd web && env -i PATH="$PATH" HOME="$HOME" node node_modules/playwright/cli.js test --config playwright-integration.config.ts)
[[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c 'SELECT count(*) FROM ledger.journal_entry')" == 1 ]]
[[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c 'SELECT count(*) FROM integration.operation_receipt')" == 1 ]]
[[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c "SELECT count(*) FROM ledger.interactive_command WHERE state='committed' AND acknowledged_at IS NOT NULL")" == 2 ]]
[[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c "SELECT count(*) FROM audit.command_action WHERE action='execute'")" == 2 ]]
[[ "$(docker exec "$proof_id" psql -X -At -U postgres -d app_proof -c 'SELECT count(*) FROM integration.outbox_message')" == 1 ]]
echo 'PASS: browser → production Next API bridge → Go HTTP → real synthetic PostgreSQL'
echo 'LIMIT: identity is a test-only fixture; no provider, MFA, cloud or live-use approval proven'
