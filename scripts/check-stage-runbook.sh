#!/usr/bin/env bash
# Verify reviewed evidence structure, including the staged version before commit.
set -euo pipefail
cd "$(dirname "$0")/.."
mode=${1:---validate}
found=0
validate() {
  local content=$1
  local heading
  for heading in '<!-- stage-runbook: true -->' '## Status and boundary' \
    '## Stage ledger' '## Current evidence' '## Remaining work' '## PR handoff'; do
    if ! printf '%s\n' "$content" | grep -Fx "$heading" >/dev/null; then
      echo 'BLOCKED: stage runbook lacks a required marker or heading.' >&2
      return 1
    fi
  done
}
case "$mode" in
  --validate)
    for path in docs/runbooks/*.md; do
      [[ "$path" =~ ^docs/runbooks/[a-z0-9-]+\.md$ ]] || continue
      [[ ! -L "$path" && -f "$path" ]] || exit 1
      validate "$(cat "$path")"
      found=1
    done
    ;;
  --staged)
    while IFS= read -r path; do
      [[ "$path" =~ ^docs/runbooks/[a-z0-9-]+\.md$ ]] || continue
      validate "$(git show ":$path")"
      found=1
    done < <(git diff --cached --name-only --diff-filter=ACM -- docs/runbooks/)
    ;;
  *) echo 'Usage: bash scripts/check-stage-runbook.sh [--validate|--staged]' >&2; exit 2 ;;
esac
[[ "$found" == 1 ]] || { echo 'BLOCKED: no updated living stage runbook.' >&2; exit 1; }
echo 'PASS: stage runbook structure.'
