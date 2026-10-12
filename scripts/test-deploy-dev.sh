#!/usr/bin/env bash
# Exercise actual orchestration with disposable fake executables; never access a cloud or backend.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd)
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/release" "$scratch/infra/gcp/migration" "$scratch/infra/gcp/runtime"
cp infra/gcp/migration/backend.tf.example "$scratch/infra/gcp/migration/"
cp infra/gcp/runtime/backend.tf.example "$scratch/infra/gcp/runtime/"
cat > "$scratch/bin/terraform" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "terraform $*" >> "$TEST_LOG"
if [[ "$*" == *"$FAIL_STEP"* && -n "$FAIL_STEP" ]]; then exit 1; fi
MOCK
cat > "$scratch/bin/gcloud" <<'MOCK'
#!/usr/bin/env bash
printf '%s\n' "gcloud $*" >> "$TEST_LOG"
if [[ "$FAIL_STEP" == job ]]; then exit 1; fi
MOCK
chmod +x "$scratch/bin/terraform" "$scratch/bin/gcloud"
export TEST_LOG="$scratch/calls" FAIL_STEP=''
export GITHUB_ACTIONS=true GITHUB_EVENT_NAME=workflow_dispatch GITHUB_REF=refs/heads/main
export PATH="$scratch/bin:$PATH"
printf '%s\n' '{"project":"jayburd-synthetic-test","operation":"promote"}' > "$scratch/release/approved.json"
cd "$scratch"
bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null
node --input-type=module - "$TEST_LOG" <<'JS'
import {readFileSync} from 'node:fs';import assert from 'node:assert/strict';
const calls=readFileSync(process.argv[2],'utf8').trim().split('\n');assert.equal(calls.length,9);
assert.match(calls[2],/migration apply/);assert.match(calls[3],/jobs execute.*--wait/);assert.match(calls[4],/runtime init/);
assert.ok(calls.every(line=>!line.includes('destroy')&&!line.includes('-target')));
JS
for FAIL_STEP in 'migration plan' 'migration apply' job; do
 export FAIL_STEP
 : > "$TEST_LOG"
 if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then exit 1; fi
 if rg -q 'chdir=infra/gcp/runtime' "$TEST_LOG"; then echo 'FAIL: runtime changed after migration failure'; exit 1; fi
done
export FAIL_STEP='runtime apply'
if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then exit 1; fi
export FAIL_STEP=''
printf '%s\n' '{"project":"jayburd-synthetic-test","operation":"rollback"}' > "$scratch/release/approved.json"
: > "$TEST_LOG"
bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null
if rg -q 'migration|jobs execute' "$TEST_LOG"; then echo 'FAIL: rollback touched migration'; exit 1; fi
export GITHUB_EVENT_NAME=pull_request
if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then exit 1; fi
echo 'PASS: migration before services, stop on failures, rollback without SQL, non-dispatch rejection.'
