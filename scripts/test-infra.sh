#!/usr/bin/env bash
# Validate only a clean synthetic copy; never load owner backend, variable or credential files.
set -euo pipefail
cd "$(dirname "$0")/.."
terraform_bin=$(command -v terraform)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT
for target in foundation application migration runtime identity; do
mkdir -p "$test_root/$target/tests"
for name in versions.tf variables.tf main.tf outputs.tf .terraform.lock.hcl; do
  cp "infra/gcp/$target/$name" "$test_root/$target/$name"
done
cp "infra/gcp/$target/tests/$target.tftest.hcl" "$test_root/$target/tests/"
# Empty CLI configuration and clean environment prevent ambient credentials/config injection.
: > "$test_root/terraform.rc"
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
