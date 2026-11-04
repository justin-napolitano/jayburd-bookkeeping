#!/usr/bin/env bash
# Install the existing exact scanner from its official checksum-pinned release before cloud auth.
set -euo pipefail
if [[ "${GITHUB_ACTIONS:-}" != true || -z "${RUNNER_TEMP:-}" || -z "${GITHUB_PATH:-}" || ! -d "$RUNNER_TEMP" || ! -f "$GITHUB_PATH" ]]; then
  echo 'BLOCKED: scanner installation requires explicit hosted-runner context' >&2
  exit 1
fi
scanner_dir=$(mktemp -d "$RUNNER_TEMP/jayburd-scanner.XXXXXX")
archive="$scanner_dir/trivy.tar.gz"
expected='c6e65abddb348e25f10549df887045629cf28cc72453cd1c63acb717316b3f3f'
curl --fail --location --silent --show-error --retry 3 --max-time 120 \
  'https://github.com/aquasecurity/trivy/releases/download/v0.75.0/trivy_0.75.0_Linux-64bit.tar.gz' \
  --output "$archive"
# No archive extraction, binary execution or PATH publication precedes exact integrity validation.
printf '%s  %s\n' "$expected" "$archive" | sha256sum --check --status
mkdir "$scanner_dir/bin"
tar -xzf "$archive" -C "$scanner_dir/bin" trivy
chmod 0700 "$scanner_dir/bin/trivy"
"$scanner_dir/bin/trivy" --version | grep -qx 'Version: 0.75.0'
printf '%s\n' "$scanner_dir/bin" >> "$GITHUB_PATH"
echo 'PASS: exact Trivy0.75.0 release checksum and version verified'
