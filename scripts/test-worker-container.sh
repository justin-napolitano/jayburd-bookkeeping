#!/usr/bin/env bash
# Build the allowlisted candidate or validate an exact existing image; runtime checks have no network.
set -euo pipefail
cd "$(dirname "$0")/.."
owned_image=0
candidate=""
case "${1:-}" in
  "") [[ $# == 0 ]] || exit 2; candidate="jayburd-worker-proof-$$-$(date +%s):local"; owned_image=1 ;;
  --image) [[ $# == 2 && "$2" =~ ^[a-zA-Z0-9][a-zA-Z0-9._/:@-]+$ ]] || exit 2; candidate="$2" ;;
  *) echo 'usage: test-worker-container.sh [--image IMAGE]' >&2; exit 2 ;;
esac
work_dir="$(mktemp -d)"
cleanup() {
  for cid_file in "$work_dir"/*.cid; do
    [[ -f "$cid_file" ]] || continue
    docker rm -f "$(cat "$cid_file")" >/dev/null 2>&1 || true
  done
  if [[ "$owned_image" == 1 ]]; then docker image rm "$candidate" >/dev/null 2>&1 || true; fi
  rm -rf "$work_dir"
}
trap cleanup EXIT
trap 'exit 130' INT TERM
if [[ "$owned_image" == 1 ]]; then docker build --file Dockerfile.worker --tag "$candidate" .; fi
candidate_id="$(docker image inspect --format '{{.Id}}' "$candidate")"
[[ "$candidate_id" =~ ^sha256:[a-f0-9]{64}$ ]] || exit 1
# Each command must stop itself offline with exact output; only verifier default succeeds inertly.
check_rejected() {
  local case_name="$1"
  shift
  local result=0
  local entrypoint=()
  if [[ "$case_name" == delivery-* ]]; then entrypoint=(--entrypoint /deliver); fi
  if [[ "$case_name" == probe-* ]]; then entrypoint=(--entrypoint /sandbox-probe); fi
  if [[ "$case_name" == webhook-* ]]; then entrypoint=(--entrypoint /webhook); fi
  if [[ "$case_name" == verifier-* ]]; then entrypoint=(--entrypoint /sandbox-verify); fi
  docker run --cidfile "$work_dir/$case_name.cid" --rm --pull=never --network=none --read-only --cap-drop=ALL \
    --security-opt=no-new-privileges:true --user=65532:65532 --pids-limit=64 \
    --memory=128m --cpus=1 "${entrypoint[@]}" "$candidate_id" "$@" >"$work_dir/$case_name" 2>&1 &
  local run_pid=$!
  for ((i=0;i<40;i++)); do
    kill -0 "$run_pid" 2>/dev/null || break
    sleep 0.25
  done
  if kill -0 "$run_pid" 2>/dev/null; then
    echo 'FAIL: worker startup check exceeded ten seconds' >&2
    exit 1
  fi
  wait "$run_pid" || result=$?
  local expected_status=1
  if [[ "$case_name" == probe-* ]]; then expected_status=2; fi
  if [[ "$case_name" == verifier-default ]]; then expected_status=0; fi
  [[ "$result" == "$expected_status" ]] || { echo 'FAIL: worker did not reject configuration' >&2; exit 1; }
  if [[ "$case_name" == probe-* ]]; then
    node -e 'const fs=require("fs");const r=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));if(r.status!=="blocked"||r.checks[0].reason!=="invalid_invocation")process.exit(1)' "$work_dir/$case_name"
    return
  fi
  if [[ "$case_name" == verifier-default ]]; then
    printf '{"offline":true}\n' >"$work_dir/expected"
  elif [[ "$case_name" == verifier-* ]]; then
    printf 'BLOCKED: Sandbox verification\n' >"$work_dir/expected"
  elif [[ "$case_name" == delivery-* ]]; then
    printf 'BLOCKED: delivery configuration\n' >"$work_dir/expected"
  elif [[ "$case_name" == webhook-* ]]; then
    printf 'webhook initialization or service failed\n' >"$work_dir/expected"
  else
    printf 'worker initialization or service failed\n' >"$work_dir/expected"
  fi
  cmp -s "$work_dir/expected" "$work_dir/$case_name" || { echo 'FAIL: worker emitted unexpected diagnostics' >&2; exit 1; }
}
check_rejected default
check_rejected invalid-mode -mode=invalid
check_rejected missing-consumer -mode=consumer
check_rejected secret-flag -password=synthetic-rejected-value
check_rejected delivery-default
check_rejected delivery-secret-flag -secret=synthetic-rejected-value
check_rejected probe-default
check_rejected probe-no-pins -allow-sandbox
check_rejected webhook-default
check_rejected webhook-secret-flag -secret=synthetic-rejected-value
check_rejected webhook-mutable-secret -cloud-sql-instance=synthetic-project:us-central1:jayburd-dev -project-number=123456789012 -plaid-client-id-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1 -plaid-client-secret-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/latest
check_rejected verifier-default
check_rejected verifier-missing-manifest -action=compare
check_rejected verifier-dsn-flag -dsn=synthetic-rejected-value
check_rejected verifier-secret-flag -secret=synthetic-rejected-value
printf 'PASS: worker image commands reject fourteen invalid configurations and verifier defaults inertly offline with fixed diagnostics.\n'
