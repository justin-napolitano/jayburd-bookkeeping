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
# Each rejected startup must stop itself, emit only the fixed diagnostic and access no network.
check_rejected() {
  local case_name="$1"
  shift
  local result=0
  local entrypoint=()
  if [[ "$case_name" == delivery-* ]]; then entrypoint=(--entrypoint /deliver); fi
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
  [[ "$result" == 1 ]] || { echo 'FAIL: worker did not reject configuration' >&2; exit 1; }
  if [[ "$case_name" == delivery-* ]]; then
    printf 'BLOCKED: delivery configuration\n' >"$work_dir/expected"
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
printf 'PASS: worker and delivery commands reject six invalid configurations offline with fixed diagnostics.\n'
