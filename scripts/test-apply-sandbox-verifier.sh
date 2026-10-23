#!/usr/bin/env bash
# Real orchestration, fake executables and private scratch source; never a cloud/backend/provider.
set -euo pipefail
cd "$(dirname "$0")/.."
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/scripts" "$scratch/deploy" "$scratch/infra/gcp/sandbox-verifier"
cp scripts/apply-sandbox-verifier.sh scripts/verifier-platform.mjs "$scratch/scripts/"
cp deploy/verifier-bootstrap.json deploy/verifier-runtime.json "$scratch/deploy/"
for name in versions.tf variables.tf main.tf outputs.tf .terraform.lock.hcl backend.tf.example; do cp "infra/gcp/sandbox-verifier/$name" "$scratch/infra/gcp/sandbox-verifier/$name"; done
cat >"$scratch/bin/git" <<'MOCK'
#!/usr/bin/env bash
printf 'git %s\n' "$*" >>"$TEST_LOG"
if [[ "$*" == 'rev-parse HEAD' ]]; then printf '%s\n' "$TEST_SOURCE";exit 0;fi
if [[ "$*" == 'diff --quiet HEAD --' ]]; then exit "$TEST_DIRTY";fi
if [[ "$*" == '-c credential.helper= ls-remote --exit-code https://github.com/justin-napolitano/jayburd-bookkeeping.git refs/heads/main' ]]; then printf '%s\trefs/heads/main\n' "$TEST_MAIN";exit 0;fi
exit 1
MOCK
cat >"$scratch/bin/terraform" <<'MOCK'
#!/usr/bin/env bash
printf 'terraform %s\n' "$*" >>"$TEST_LOG"
if [[ -n "$FAIL_STEP" && "$*" == *" $FAIL_STEP "* ]]; then exit 1;fi
MOCK
chmod +x "$scratch/bin/git" "$scratch/bin/terraform"
export PATH="$scratch/bin:$PATH" TEST_LOG="$scratch/calls"
export TEST_SOURCE=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa TEST_MAIN=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa TEST_DIRTY=0 FAIL_STEP=''
script="$scratch/scripts/apply-sandbox-verifier.sh"
: >"$TEST_LOG"
[[ "$(bash "$script")" == '{"offline":true}' && ! -s "$TEST_LOG" ]]
hash=$(shasum -a 256 "$scratch/deploy/verifier-bootstrap.json" | cut -d' ' -f1)
apply() { bash "$script" --apply deploy/verifier-bootstrap.json "$hash" "$TEST_SOURCE"; }
apply >/dev/null
[[ $(grep -Ec '^terraform ' "$TEST_LOG") == 3 ]]
grep -Eq 'prefix=sandbox-verifier' "$TEST_LOG"
if grep -Eq -- 'gcloud|jobs execute| -target|destroy|chdir=infra/gcp/(runtime|workers|application)' "$TEST_LOG"; then echo 'FAIL: unrelated mutation/execution selected' >&2;exit 1;fi
for FAIL_STEP in init plan apply; do
 export FAIL_STEP
 : >"$TEST_LOG"
 if apply >"$scratch/out" 2>"$scratch/err"; then echo 'FAIL: failed apply stage ignored' >&2;exit 1;fi
 grep -Eq '^BLOCKED: verifier platform operation failed; private owner diagnosis required$' "$scratch/err"
 if [[ "$FAIL_STEP" == init ]]; then [[ $(grep -Ec '^terraform ' "$TEST_LOG") == 1 ]];fi
 if [[ "$FAIL_STEP" == plan ]]; then [[ $(grep -Ec '^terraform ' "$TEST_LOG") == 2 ]];fi
done
export FAIL_STEP=''
for mode in stale dirty hash; do
 : >"$TEST_LOG"
 export TEST_MAIN="$TEST_SOURCE" TEST_DIRTY=0
 if [[ "$mode" == stale ]];then export TEST_MAIN=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb;fi
 if [[ "$mode" == dirty ]];then export TEST_DIRTY=1;fi
 if [[ "$mode" == hash ]];then hash=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb;fi
 if apply >/dev/null 2>&1;then echo 'FAIL: unreviewed metadata/source accepted' >&2;exit 1;fi
 if grep -Eq '^terraform ' "$TEST_LOG";then echo 'FAIL: cloud operation before review guards' >&2;exit 1;fi
done
echo 'PASS: inert default, source/hash guards, isolated verifier backend, staged failure stop and no implicit execution.'
