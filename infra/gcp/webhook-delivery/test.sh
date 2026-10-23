#!/usr/bin/env bash
# Isolated mocked plan only; no owner backend, variables, credentials or cloud provider calls.
set -euo pipefail
repo_root=$(cd "$(dirname "$0")/../../.." && pwd)
terraform_bin=$(command -v terraform)
proof=$(mktemp -d)
trap 'rm -rf "$proof"' EXIT
mkdir -p "$proof/tests"
for name in versions.tf variables.tf main.tf outputs.tf .terraform.lock.hcl; do cp "$repo_root/infra/gcp/webhook-delivery/$name" "$proof/$name"; done
cp "$repo_root/infra/gcp/webhook-delivery/tests/webhook-delivery.tftest.hcl" "$proof/tests/"
: > "$proof/terraform.rc"
run() { env -i PATH="$PATH" TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1 TF_CLI_CONFIG_FILE="$proof/terraform.rc" "$terraform_bin" -chdir="$proof" "$@"; }
run fmt -check -recursive
run init -backend=false -input=false -lockfile=readonly
run validate -no-color
run test -no-color -filter=tests/webhook-delivery.tftest.hcl
