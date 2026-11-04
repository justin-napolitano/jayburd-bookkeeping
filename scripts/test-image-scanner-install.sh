#!/usr/bin/env bash
# Synthetic negatives prove default inertness and rejection before executing unverified downloads.
set -euo pipefail
command -v sha256sum >/dev/null
cd "$(dirname "$0")/.."
test_dir=$(mktemp -d)
trap 'rm -rf "$test_dir"' EXIT
mkdir "$test_dir/bin" "$test_dir/runner"
: > "$test_dir/path"
cat > "$test_dir/bin/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$@" > "$SCANNER_TEST_CALLS"
while [[ $# -gt 0 ]]; do
  if [[ "$1" == --output ]]; then printf 'synthetic corrupt download\n' > "$2"; exit 0; fi
  shift
done
exit 1
SH
cat > "$test_dir/bin/tar" <<'SH'
#!/usr/bin/env bash
: > "$SCANNER_TEST_EXTRACTED"
exit 99
SH
chmod +x "$test_dir/bin/curl" "$test_dir/bin/tar"
# A normal developer invocation fails before any downloader even sees an argument.
if env -i PATH="$test_dir/bin:$PATH" SCANNER_TEST_CALLS="$test_dir/calls" bash scripts/install-image-scanner.sh > "$test_dir/output" 2>&1; then exit 1; fi
[[ ! -e "$test_dir/calls" ]]
# Even in a hosted context, wrong bytes cannot be extracted, run or added to PATH.
if env -i PATH="$test_dir/bin:$PATH" GITHUB_ACTIONS=true RUNNER_TEMP="$test_dir/runner" GITHUB_PATH="$test_dir/path" \
  SCANNER_TEST_CALLS="$test_dir/calls" SCANNER_TEST_EXTRACTED="$test_dir/extracted" \
  bash scripts/install-image-scanner.sh > "$test_dir/output" 2>&1; then exit 1; fi
grep -qx 'https://github.com/aquasecurity/trivy/releases/download/v0.75.0/trivy_0.75.0_Linux-64bit.tar.gz' "$test_dir/calls"
downloads=("$test_dir"/runner/jayburd-scanner.*/trivy.tar.gz)
[[ ${#downloads[@]} == 1 && -s "${downloads[0]}" ]]
[[ ! -e "$test_dir/extracted" && ! -s "$test_dir/path" ]]
echo 'PASS: default-local inertness and corrupt scanner rejection before extraction/execution/PATH'
