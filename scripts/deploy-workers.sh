#!/usr/bin/env bash
# Owner-approved worker runtime only. Bootstrap, SQL and provider calls are never executed here.
set -euo pipefail
[[ "${GITHUB_ACTIONS:-}" == true && "${GITHUB_EVENT_NAME:-}" == workflow_dispatch && "${GITHUB_REF:-}" == refs/heads/main ]]
[[ $# == 1 && -d "$1" ]]
release_dir="$1"
project=$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).project' "$release_dir/approved.json")
[[ "$project" =~ ^[a-z][a-z0-9-]{4,28}[a-z0-9]$ ]]
umask 077
# Suppress provider diagnostics; only allowlisted readback files become public receipts.
quiet() {
  if ! "$@" > "$release_dir/provider-private.log" 2>&1; then
    echo 'BLOCKED: worker deployment failed; owner diagnosis required.' >&2
    return 1
  fi
}
# Refuse rollout while owner-controlled scheduling is active; this does not drain existing tasks.
read_scheduler() {
  if ! gcloud scheduler jobs describe jayburd-dev-acquisition-publisher --project "$project" --location us-central1 \
    --format='json(name,state,httpTarget.uri,httpTarget.httpMethod,httpTarget.oidcToken,httpTarget.oauthToken,pubsubTarget,appEngineHttpTarget)' \
    > "$release_dir/scheduler.json" 2> "$release_dir/provider-private.log"; then
    echo 'BLOCKED: worker scheduling readback failed' >&2; return 1
  fi
  node --input-type=module - "$release_dir/scheduler.json" "$project" <<'JS'
import {readFileSync} from 'node:fs';
try {
  const s=JSON.parse(readFileSync(process.argv[2]));
  if(s.name!==`projects/${process.argv[3]}/locations/us-central1/jobs/jayburd-dev-acquisition-publisher` || s.state!=='PAUSED') throw new Error();
} catch {console.error('BLOCKED: worker scheduler must already be paused');process.exitCode=1;}
JS
}
read_scheduler
cp infra/gcp/worker-runtime/backend.tf.example infra/gcp/worker-runtime/backend.tf
quiet terraform -chdir=infra/gcp/worker-runtime init -input=false -lockfile=readonly \
  -backend-config="bucket=$project-jayburd-dev-worker-runtime-state" -backend-config=prefix=worker-runtime
quiet terraform -chdir=infra/gcp/worker-runtime plan -input=false -lock-timeout=5m \
  -var-file="$release_dir/worker-runtime.tfvars.json" -out="$release_dir/worker-runtime.tfplan"
read_scheduler
quiet terraform -chdir=infra/gcp/worker-runtime apply -input=false -lock-timeout=5m "$release_dir/worker-runtime.tfplan"
for component in publisher consumer; do
  suffix="$component"
  [[ "$component" != consumer ]] || suffix=sync
  if ! gcloud run services describe "jayburd-dev-acquisition-$suffix" --project "$project" --region us-central1 \
    --format='json(metadata.name,metadata.generation,status.observedGeneration,status.conditions[].type,status.conditions[].status,status.latestCreatedRevisionName,status.latestReadyRevisionName,status.traffic,spec.template.metadata.name,spec.template.spec.serviceAccountName,spec.template.spec.containers[].image)' \
    > "$release_dir/$component-revision.json" 2> "$release_dir/provider-private.log"; then
    echo 'BLOCKED: worker revision readback failed' >&2; exit 1
  fi
done
read_scheduler
echo 'PASS: worker runtime applied; observed receipt and hosted qualification remain separate.'
