#!/usr/bin/env bash
# Run only after owner-dispatch manifest/evidence checks; Terraform owns revisions, gcloud only executes the job.
set -euo pipefail
[[ "${GITHUB_ACTIONS:-}" == true && "${GITHUB_EVENT_NAME:-}" == workflow_dispatch && "${GITHUB_REF:-}" == refs/heads/main ]]
[[ $# == 1 && -d "$1" ]]
release_dir="$1"
project=$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).project' "$release_dir/approved.json")
operation=$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).operation' "$release_dir/approved.json")
[[ "$project" =~ ^[a-z][a-z0-9-]{4,28}[a-z0-9]$ && "$operation" =~ ^(promote|rollback)$ ]]
umask 077
# Provider diagnostics/plans stay runner-private and are never emitted or uploaded. Fixed outcomes only.
quiet() {
  if ! "$@" > "$release_dir/provider-private.log" 2>&1; then
    echo 'BLOCKED: deployment operation failed; owner diagnosis required. No automatic rollback or down migration.' >&2
    return 1
  fi
}
# Apply one saved plan under the root's own backend lock; never target individual resources.
apply_root() {
  local root="$1"
  cp "infra/gcp/$root/backend.tf.example" "infra/gcp/$root/backend.tf"
  quiet terraform -chdir="infra/gcp/$root" init -input=false -lockfile=readonly \
    -backend-config="bucket=$project-jayburd-dev-runtime-state" -backend-config="prefix=$root"
  quiet terraform -chdir="infra/gcp/$root" plan -input=false -lock-timeout=5m \
    -var-file="$release_dir/$root.tfvars.json" -out="$release_dir/$root.tfplan"
  quiet terraform -chdir="infra/gcp/$root" apply -input=false -lock-timeout=5m "$release_dir/$root.tfplan"
}
if [[ "$operation" == promote ]]; then
  apply_root migration
  quiet gcloud run jobs execute jayburd-dev-migrate --project "$project" --region us-central1 --wait --quiet
  echo 'PASS: migration execution completed before runtime changes.'
fi
# Rollback deliberately never touches the job or database; reviewed compatibility is required upstream.
apply_root runtime
# Read only selected image/revision/traffic metadata; never export environment or state.
for component in api web; do
  if ! gcloud run services describe "jayburd-dev-$component" --project "$project" --region us-central1 \
    --format='json(status.latestReadyRevisionName,status.traffic,spec.template.spec.containers.image)' > "$release_dir/$component-revision.json" 2> "$release_dir/provider-private.log"; then
    echo 'BLOCKED: revision readback failed' >&2; exit 1
  fi
done
echo 'PASS: Terraform runtime update completed; transport and real identity evidence are separate gates.'
