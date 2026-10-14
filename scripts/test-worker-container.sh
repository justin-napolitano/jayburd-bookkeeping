#!/usr/bin/env bash
# Validate an explicitly supplied local candidate without networking, mounts or credentials.
set -euo pipefail
if [[ $# != 2 || "$1" != '--image' || ! "$2" =~ ^[a-zA-Z0-9][a-zA-Z0-9._/:@-]+$ ]]; then
  echo 'usage: test-worker-container.sh --image IMAGE' >&2
  exit 2
fi
candidate="$2"
candidate_id="$(docker image inspect --format '{{.Id}}' "$candidate")"
[[ "$candidate_id" =~ ^sha256:[a-f0-9]{64}$ ]] || exit 1
work_dir="$(mktemp -d)"
cleanup() {
  for cid_file in "$work_dir"/*.cid; do
    [[ -f "$cid_file" ]] || continue
    docker rm -f "$(cat "$cid_file")" >/dev/null 2>&1 || true
  done
  rm -rf "$work_dir"
}
trap cleanup EXIT
# Each rejected startup must stop itself, emit only the fixed diagnostic and access no network.
check_rejected() {
  local case_name="$1"
  shift
  local result=0
  docker run --cidfile "$work_dir/$case_name.cid" --rm --pull=never --network=none --read-only --cap-drop=ALL \
    --security-opt=no-new-privileges:true --user=65532:65532 --pids-limit=64 \
    --memory=128m --cpus=1 "$candidate_id" "$@" >"$work_dir/$case_name" 2>&1 &
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
  printf 'worker initialization or service failed\n' >"$work_dir/expected"
  cmp -s "$work_dir/expected" "$work_dir/$case_name" || { echo 'FAIL: worker emitted unexpected diagnostics' >&2; exit 1; }
}
check_rejected default
check_rejected invalid-mode -mode=invalid
check_rejected missing-consumer -mode=consumer
check_rejected secret-flag -password=synthetic-rejected-value
printf 'PASS: worker candidate rejects four invalid configurations offline with fixed diagnostics.\n'
