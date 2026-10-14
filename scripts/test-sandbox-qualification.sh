#!/usr/bin/env bash
# Offline guard proof; never accepts credentials, descriptors or a provider mode.
set -euo pipefail
cd "$(dirname "$0")/.."
[[ $# == 0 ]] || { echo 'BLOCKED: offline proof accepts no arguments.' >&2; exit 2; }
go test -race ./internal/sandboxqualification ./cmd/sandbox-qualify
proof_dir=$(mktemp -d "${TMPDIR:-/tmp}/jayburd-sandbox-offline.XXXXXX")
trap 'rm -rf "$proof_dir"' EXIT
go build -trimpath -o "$proof_dir/qualify" ./cmd/sandbox-qualify
set +e
receipt=$("$proof_dir/qualify")
status=$?
set -e
[[ "$status" == 2 && "$receipt" == *'"profile":"plaid-sandbox-diagnostic-v1"'* && "$receipt" == *'"status":"blocked"'* && "$receipt" == *'"reason":"offline"'* ]] || { echo 'FAIL: default offline guard.' >&2; exit 1; }
echo 'PASS: synthetic tests and default offline guard only; Sandbox remains unqualified.'
