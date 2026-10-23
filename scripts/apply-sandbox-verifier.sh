#!/usr/bin/env bash
# Explicit owner-only provisioning; no source-provider calls or job execution, and default is inert.
set -euo pipefail
if [[ $# == 0 ]]; then printf '{"offline":true}\n'; exit 0; fi
[[ $# == 4 && "$1" == --apply && "$3" =~ ^[a-f0-9]{64}$ && "$4" =~ ^[a-f0-9]{40}$ ]] || { echo 'BLOCKED: verifier apply invocation rejected' >&2; exit 1; }
cd "$(dirname "$0")/.."
case "$2" in deploy/verifier-bootstrap.json|deploy/verifier-runtime.json) ;; *) echo 'BLOCKED: verifier metadata path rejected' >&2;exit 1;;esac
# A reviewed commit must be current main, with clean tracked source, before any platform mutation.
[[ "$(git rev-parse HEAD)" == "$4" ]] && git diff --quiet HEAD -- || { echo 'BLOCKED: verifier source differs from review' >&2;exit 1; }
remote_sha=$(git -c credential.helper= ls-remote --exit-code https://github.com/justin-napolitano/jayburd-bookkeeping.git refs/heads/main | cut -f1)
[[ "$remote_sha" == "$4" ]] || { echo 'BLOCKED: verifier source is not current main' >&2;exit 1; }
umask 077
mkdir -p .local
private_dir=$(mktemp -d "$PWD/.local/verifier-apply.XXXXXX")
node scripts/verifier-platform.mjs "$2" "$3" "$private_dir"
mkdir "$private_dir/root"
for name in versions.tf variables.tf main.tf outputs.tf .terraform.lock.hcl; do cp "infra/gcp/sandbox-verifier/$name" "$private_dir/root/$name"; done
cp infra/gcp/sandbox-verifier/backend.tf.example "$private_dir/root/backend.tf"
# Provider diagnostics, plans and state stay private; never inspect or upload their contents.
quiet() {
 if ! "$@" >"$private_dir/provider-private.log" 2>&1; then echo 'BLOCKED: verifier platform operation failed; private owner diagnosis required' >&2;return 1;fi
}
quiet terraform -chdir="$private_dir/root" init -input=false -lockfile=readonly -backend-config=bucket=jayburd-bookkeeping-dev-jayburd-dev-runtime-state -backend-config=prefix=sandbox-verifier
quiet terraform -chdir="$private_dir/root" plan -input=false -lock-timeout=5m -var-file="$private_dir/reviewed.tfvars.json" -out="$private_dir/reviewed.tfplan"
quiet terraform -chdir="$private_dir/root" apply -input=false -lock-timeout=5m "$private_dir/reviewed.tfplan"
echo 'PASS: reviewed verifier platform applied; no comparison executed'
