#!/usr/bin/env bash
# Exercise actual orchestration with disposable fake executables; never access a cloud or backend.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(pwd)
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/release" "$scratch/infra/gcp/migration" "$scratch/infra/gcp/runtime" "$scratch/scripts"
cp scripts/migration-receipt.mjs "$scratch/scripts/"
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
if [[ "$*" == *'jobs execute'* ]]; then printf 'jayburd-dev-migrate-abc12\n'; fi
if [[ "$*" == *'jobs executions describe'* ]]; then
  [[ "$FAIL_STEP" != readback ]] || exit 1
  cat "$TEST_EXECUTION"
fi
MOCK
chmod +x "$scratch/bin/terraform" "$scratch/bin/gcloud"
export TEST_LOG="$scratch/calls" FAIL_STEP=''
export TEST_EXECUTION="$scratch/execution.json"
export GITHUB_ACTIONS=true GITHUB_EVENT_NAME=workflow_dispatch GITHUB_REF=refs/heads/main
export PATH="$scratch/bin:$PATH"
node --input-type=module - "$scratch" <<'JS'
import {writeFileSync} from 'node:fs';
const root=process.argv[2], image='us-central1-docker.pkg.dev/jayburd-synthetic-test/jayburd-dev/migrate@sha256:'+'a'.repeat(64);
writeFileSync(root+'/release/approved.json',JSON.stringify({project:'jayburd-synthetic-test',operation:'promote',images:{migrate:image}}));
writeFileSync(root+'/execution.json',JSON.stringify({metadata:{name:'jayburd-dev-migrate-abc12'},spec:{taskCount:1,template:{spec:{serviceAccountName:'jayburd-dev-migrator@jayburd-synthetic-test.iam.gserviceaccount.com',maxRetries:0,timeoutSeconds:'660',containers:[{image,args:['-action=up','-instance=jayburd-synthetic-test:us-central1:jayburd-dev']}]}}},status:{conditions:[{type:'Completed',status:'True'}],succeededCount:1,completionTime:'2026-10-12T00:00:00Z'}}));
JS
cd "$scratch"
bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null
node --input-type=module - "$TEST_LOG" <<'JS'
import {readFileSync} from 'node:fs';import assert from 'node:assert/strict';
const calls=readFileSync(process.argv[2],'utf8').trim().split('\n');assert.equal(calls.length,10);
assert.match(calls[2],/migration apply/);assert.match(calls[3],/jobs execute.*--wait/);assert.match(calls[4],/jobs executions describe/);assert.match(calls[5],/runtime init/);
// The real CLI omits repeated fields without []; lock the allowlisted readback contract to prevent recurrence.
for (const field of ['containers[].image','containers[].command','containers[].args','conditions[].type','conditions[].status']) assert.ok(calls[4].includes(field));
assert.ok(!calls[4].includes('.env') && !calls[4].includes('.message'));
for (const call of calls.slice(8)) assert.ok(call.includes('containers[].image'));
assert.ok(calls.every(line=>!line.includes('destroy')&&!line.includes('-target')));
JS
for FAIL_STEP in 'migration plan' 'migration apply' job readback; do
 export FAIL_STEP
 : > "$TEST_LOG"
 if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then exit 1; fi
 if grep -Eq 'chdir=infra/gcp/runtime' "$TEST_LOG"; then echo 'FAIL: runtime changed after migration failure'; exit 1; fi
done
export FAIL_STEP='runtime apply'
if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then exit 1; fi
export FAIL_STEP=''
# A real migration-only execution must return before either runtime Terraform or service readback.
node -e 'const fs=require("fs"),p=process.argv[1],m=JSON.parse(fs.readFileSync(p));m.operation="migrate";fs.writeFileSync(p,JSON.stringify(m));' "$scratch/release/approved.json"
: > "$TEST_LOG"
bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null
if grep -Eq 'chdir=infra/gcp/runtime|run services' "$TEST_LOG"; then echo 'FAIL: migration-only touched services'; exit 1; fi
node --input-type=module - "$TEST_LOG" "$scratch/release/migration.json" <<'JS'
import {readFileSync} from 'node:fs';import assert from 'node:assert/strict';
assert.equal(readFileSync(process.argv[2],'utf8').trim().split('\n').length,5);
assert.equal(JSON.parse(readFileSync(process.argv[3])).status,'migration-execution-passed');
JS
# Read-only verification uses the same isolated path but rejects an up execution.
node -e 'const fs=require("fs"),p=process.argv[1],m=JSON.parse(fs.readFileSync(p));m.operation="verify";fs.writeFileSync(p,JSON.stringify(m));' "$scratch/release/approved.json"
if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then echo 'FAIL: verification accepted an up execution'; exit 1; fi
node -e 'const fs=require("fs"),p=process.argv[1],m=JSON.parse(fs.readFileSync(p));m.spec.template.spec.containers[0].args[0]="-action=status";fs.writeFileSync(p,JSON.stringify(m));' "$TEST_EXECUTION"
: > "$TEST_LOG"
bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null
if grep -Eq 'chdir=infra/gcp/runtime|run services' "$TEST_LOG"; then echo 'FAIL: verification touched services'; exit 1; fi
node -e 'const fs=require("fs"),assert=require("assert/strict");assert.equal(JSON.parse(fs.readFileSync(process.argv[1])).status,"migration-verification-passed");' "$scratch/release/migration.json"
# Completed-but-wrong execution must fail acceptance, not emit a new success receipt.
node -e 'const fs=require("fs"),p=process.argv[1],m=JSON.parse(fs.readFileSync(p));m.spec.template.spec.containers[0].image="wrong";fs.writeFileSync(p,JSON.stringify(m));' "$TEST_EXECUTION"
if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then echo 'FAIL: foreign image accepted'; exit 1; fi
printf '%s\n' '{"project":"jayburd-synthetic-test","operation":"rollback"}' > "$scratch/release/approved.json"
: > "$TEST_LOG"
bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null
if grep -Eq 'migration|jobs execute' "$TEST_LOG"; then echo 'FAIL: rollback touched migration'; exit 1; fi
export GITHUB_EVENT_NAME=pull_request
if bash "$root/scripts/deploy-dev.sh" "$scratch/release" >/dev/null 2>&1; then exit 1; fi
echo 'PASS: migration-only isolation, exact execution, stop on failures, rollback without SQL, non-dispatch rejection.'
