#!/usr/bin/env bash
# Exercise the real worker rollout orchestrator with disposable cloud/backend substitutes.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd)
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/release" "$scratch/infra/gcp/worker-runtime"
cp infra/gcp/worker-runtime/backend.tf.example "$scratch/infra/gcp/worker-runtime/"
cat > "$scratch/bin/terraform" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "terraform $*" >> "$TEST_LOG"
if [[ "$*" == *" $FAIL_STEP "* && -n "$FAIL_STEP" ]]; then
  echo 'synthetic-private-provider-diagnostic' >&2
  exit 1
fi
MOCK
cat > "$scratch/bin/gcloud" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "gcloud $*" >> "$TEST_LOG"
if [[ "$*" == 'scheduler jobs describe '* ]]; then
  count=$(cat "$SCHEDULER_COUNT")
  count=$((count + 1))
  printf '%s\n' "$count" > "$SCHEDULER_COUNT"
  if [[ "$FAIL_STEP" == missing ]]; then
    echo 'synthetic-private-provider-diagnostic' >&2
    exit 1
  fi
  state=PAUSED
  if [[ "$FAIL_STEP" == active || "$FAIL_STEP" == "active-$count" ]]; then state=ENABLED; fi
  name=projects/jayburd-synthetic-test/locations/us-central1/jobs/jayburd-dev-acquisition-publisher
  [[ "$FAIL_STEP" != foreign ]] || name=projects/foreign-project/locations/us-central1/jobs/jayburd-dev-acquisition-publisher
  printf '{"name":"%s","state":"%s"}\n' "$name" "$state"
elif [[ "$*" == 'run services describe '* ]]; then
  if [[ "$FAIL_STEP" == readback || ( "$FAIL_STEP" == consumer-readback && "$*" == *acquisition-sync* ) ]]; then
    echo 'synthetic-private-provider-diagnostic' >&2
    exit 1
  fi
  printf '{}\n'
else
  echo 'Unexpected cloud mutation' >&2
  exit 99
fi
MOCK
chmod +x "$scratch/bin/terraform" "$scratch/bin/gcloud"
export TEST_LOG="$scratch/calls" SCHEDULER_COUNT="$scratch/scheduler-count" FAIL_STEP=''
export PATH="$scratch/bin:$PATH"
export GITHUB_ACTIONS=true GITHUB_EVENT_NAME=workflow_dispatch GITHUB_REF=refs/heads/main
cd "$scratch"
# Reset only synthetic outputs so each failure case has independent call-order evidence.
reset_case() {
  : > "$TEST_LOG"
  printf '0\n' > "$SCHEDULER_COUNT"
}
# Assert the public diagnostic is fixed and cannot echo private provider output.
expect_failure() {
  reset_case
  if bash "$root/scripts/deploy-workers.sh" "$scratch/release" > "$scratch/output" 2>&1; then
    echo "FAIL: accepted $FAIL_STEP"; exit 1
  fi
  if grep -Eq 'synthetic-private-provider-diagnostic|PASS:' "$scratch/output"; then
    echo 'FAIL: private diagnostics or success escaped a failed rollout'; exit 1
  fi
  [[ $(cat "$scratch/output") == "$1" ]]
  [[ $(wc -l < "$TEST_LOG" | tr -d ' ') == "$2" ]]
}
for operation in promote rollback sandbox-promote sandbox-rollback; do
  printf '{"project":"jayburd-synthetic-test","operation":"%s"}\n' "$operation" > "$scratch/release/approved.json"
  reset_case
  bash "$root/scripts/deploy-workers.sh" "$scratch/release" > "$scratch/output"
  node --input-type=module - "$TEST_LOG" <<'JS'
import {readFileSync} from 'node:fs';
import assert from 'node:assert/strict';
const calls=readFileSync(process.argv[2],'utf8').trim().split('\n');
assert.equal(calls.length,8);
assert.match(calls[0],/^gcloud scheduler jobs describe /);
assert.match(calls[1],/^terraform -chdir=infra\/gcp\/worker-runtime init /);
assert.match(calls[1],/-backend-config=bucket=jayburd-synthetic-test-jayburd-dev-worker-runtime-state/);
assert.match(calls[2],/^terraform -chdir=infra\/gcp\/worker-runtime plan /);
assert.match(calls[3],/^gcloud scheduler jobs describe /);
assert.match(calls[4],/^terraform -chdir=infra\/gcp\/worker-runtime apply /);
assert.match(calls[5],/^gcloud run services describe jayburd-dev-acquisition-publisher /);
assert.match(calls[6],/^gcloud run services describe jayburd-dev-acquisition-sync /);
assert.match(calls[7],/^gcloud scheduler jobs describe /);
for(const call of calls) {
  assert.doesNotMatch(call,/migration|gcloud sql|jobs execute|destroy|-target|scheduler jobs (create|update|pause|resume|run)/);
  if(call.startsWith('terraform')) assert.ok(call.includes('-chdir=infra/gcp/worker-runtime '));
}
for(const call of calls.slice(5,7)) {
  for(const field of ['containers[].image','conditions[].type','conditions[].status','status.latestReadyRevisionName','spec.template.spec.serviceAccountName']) assert.ok(call.includes(field));
  assert.doesNotMatch(call,/\.env|\.message/);
}
JS
  grep -Eq '^PASS: worker runtime applied;' "$scratch/output"
done
for FAIL_STEP in active foreign; do
  export FAIL_STEP
  expect_failure 'BLOCKED: worker scheduler must already be paused' 1
done
export FAIL_STEP=missing
expect_failure 'BLOCKED: worker scheduling readback failed' 1
export FAIL_STEP=init
expect_failure 'BLOCKED: worker deployment failed; owner diagnosis required.' 2
export FAIL_STEP=plan
expect_failure 'BLOCKED: worker deployment failed; owner diagnosis required.' 3
export FAIL_STEP=active-2
expect_failure 'BLOCKED: worker scheduler must already be paused' 4
export FAIL_STEP=apply
expect_failure 'BLOCKED: worker deployment failed; owner diagnosis required.' 5
export FAIL_STEP=readback
expect_failure 'BLOCKED: worker revision readback failed' 6
export FAIL_STEP=consumer-readback
expect_failure 'BLOCKED: worker revision readback failed' 7
export FAIL_STEP=active-3
expect_failure 'BLOCKED: worker scheduler must already be paused' 8
export FAIL_STEP=''
export GITHUB_EVENT_NAME=pull_request
expect_failure '' 0
export GITHUB_EVENT_NAME=workflow_dispatch GITHUB_REF=refs/heads/unreviewed
expect_failure '' 0
export GITHUB_REF=refs/heads/main GITHUB_ACTIONS=false
expect_failure '' 0
printf '%s\n' 'PASS: worker promote/rollback isolation, paused scheduling gates, fixed failure diagnostics, exact readback and dispatch-only execution.'
