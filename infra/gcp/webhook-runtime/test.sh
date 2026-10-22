#!/usr/bin/env bash
# Copy only reviewed public source/locks/tests; never initialize owner backends or load runtime variable files.
set -euo pipefail
repo_root=$(cd "$(dirname "$0")/../../.." && pwd)
terraform_bin=$(command -v terraform)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
: > "$test_root/terraform.rc"
for target in webhook-bootstrap webhook-runtime; do
  mkdir -p "$test_root/$target/tests"
  for name in versions.tf variables.tf main.tf outputs.tf .terraform.lock.hcl; do
    cp "$repo_root/infra/gcp/$target/$name" "$test_root/$target/$name"
  done
  cp "$repo_root/infra/gcp/$target/tests/$target.tftest.hcl" "$test_root/$target/tests/"
  run_terraform() {
    env -i PATH="$PATH" TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1 \
      TF_CLI_CONFIG_FILE="$test_root/terraform.rc" \
      "$terraform_bin" -chdir="$test_root/$target" "$@"
  }
  run_terraform fmt -check -recursive
  run_terraform init -backend=false -input=false -lockfile=readonly
  run_terraform validate -no-color
  run_terraform test -no-color -filter="tests/$target.tftest.hcl"
done
