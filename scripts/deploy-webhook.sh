#!/usr/bin/env bash
# Owner-reviewed existing host only; first creation, public IAM, SQL and secrets remain separate owner operations.
set -euo pipefail
[[ $# == 3 && -d "$1" && -d "$2" && "$3" =~ ^[a-f0-9]{64}$ ]] || { echo 'BLOCKED: webhook rollout invocation rejected.' >&2; exit 1; }
release_dir="$1"; evidence="$2"; hash="$3"
umask 077
[[ "${GITHUB_ACTIONS:-}" == true && "${GITHUB_EVENT_NAME:-}" == workflow_dispatch && "${GITHUB_REF:-}" == refs/heads/main && "${GITHUB_ACTOR_ID:-}" == 55423093 && "${GITHUB_REPOSITORY_ID:-}" == 1414515318 && "${GITHUB_TRIGGERING_ACTOR:-}" == justin-napolitano ]] || { echo 'BLOCKED: webhook rollout context rejected.' >&2; exit 1; }
[[ -n "${GOOGLE_OAUTH_ACCESS_TOKEN:-}" && -n "${CLOUDSDK_AUTH_ACCESS_TOKEN:-}" ]] || { echo 'BLOCKED: explicit webhook federation token required.' >&2; exit 1; }
quiet() {
  if ! "$@" >"$release_dir/operator-private.log" 2>&1; then
    echo 'BLOCKED: webhook rollout failed; owner review required.' >&2
    return 1
  fi
}
recheck() {
  if ! CURRENT_MAIN_SHA=$(gh api repos/justin-napolitano/jayburd-bookkeeping/commits/main --jq .sha 2>"$release_dir/operator-private.log"); then
    echo 'BLOCKED: webhook main readback rejected.' >&2; return 1
  fi
  export CURRENT_MAIN_SHA
  node scripts/prepare-webhook-rollout.mjs deploy/webhook-release.json "$hash" "$release_dir" "$evidence"
}
# Never trust caller-prepared Terraform variables: rederive from exact tracked bytes/provenance/current main.
recheck
node scripts/collect-webhook-release.mjs deploy/webhook-release.json "$hash" "$evidence" preflight unused
cp infra/gcp/webhook-runtime/backend.tf.example infra/gcp/webhook-runtime/backend.tf
quiet terraform -chdir=infra/gcp/webhook-runtime init -input=false -lockfile=readonly \
  -backend-config=bucket=jayburd-bookkeeping-dev-jayburd-dev-webhook-runtime-state -backend-config=prefix=webhook-runtime
# Owner evidence must attest the existing isolated state handoff; tooling never reads state/plan contents.
# Creation is forbidden by both this preflight and cloud IAM. Missing/imported-wrong state cannot authorize create/delete.
quiet terraform -chdir=infra/gcp/webhook-runtime plan -input=false -lock-timeout=5m \
  -var-file="$release_dir/webhook-runtime.tfvars.json" -out="$release_dir/webhook-runtime.tfplan"
node scripts/collect-webhook-release.mjs deploy/webhook-release.json "$hash" "$evidence" preflight unused
recheck
quiet terraform -chdir=infra/gcp/webhook-runtime apply -input=false -lock-timeout=5m "$release_dir/webhook-runtime.tfplan"
mkdir -p "$release_dir/receipt"
node scripts/collect-webhook-release.mjs deploy/webhook-release.json "$hash" "$evidence" receipt "$release_dir/receipt/webhook-deployment.json"
echo 'PASS: webhook runtime updated; hosted qualification remains separate.'
