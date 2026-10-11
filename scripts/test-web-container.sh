#!/usr/bin/env bash
# Test the exact candidate image without identity, host mounts, published ports or external networking.
set -euo pipefail
cd "$(dirname "$0")/.."
proof_id="jayburd-web-proof-$$-$(date +%s)"
proof_image="$proof_id:local"
owned_image=1
cleanup() {
 docker rm -f "$proof_id" >/dev/null 2>&1 || true
 docker network rm "$proof_id" >/dev/null 2>&1 || true
 if [[ "$owned_image" == 1 ]]; then docker image rm "$proof_image" >/dev/null 2>&1 || true; fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM
case "${1:-}" in
 "") docker build --file Dockerfile.web --tag "$proof_image" . ;;
 --image)
  [[ $# == 2 && "$2" == sha256:* ]] || exit 1
  proof_image="$2"; owned_image=0 ;;
 *) echo 'Usage: scripts/test-web-container.sh [--image sha256:ID]' >&2; exit 1 ;;
esac
[[ "$(docker image inspect --format '{{.Config.User}}' "$proof_image")" == '65532:65532' ]]
docker network create --internal "$proof_id" >/dev/null
docker run -d --name "$proof_id" --network "$proof_id" --network-alias web \
 --read-only --cap-drop ALL --security-opt no-new-privileges \
 --memory 512m --cpus 1 --pids-limit 64 "$proof_image" >/dev/null
# The image's own Node fetch checks the shell and a referenced static asset, not just a TCP port.
docker run --rm -i --network "$proof_id" --read-only --cap-drop ALL \
 --security-opt no-new-privileges --entrypoint node "$proof_image" --input-type=module <<'JS'
let response;
for (let attempt = 0; attempt < 40; attempt++) {
 try { response = await fetch('http://web:8080/', {signal: AbortSignal.timeout(1000)}); if (response.ok) break; } catch {}
 await new Promise(resolve => setTimeout(resolve, 250));
}
if (!response?.ok) throw new Error('shell unavailable');
const body = await response.text();
if (!body.includes('Sign-in is not configured')) throw new Error('unconfigured sign-in boundary missing');
if (response.headers.get('x-frame-options') !== 'DENY') throw new Error('frame boundary missing');
const asset = body.match(/src="([^" ]+\/_next\/static\/[^" ]+\.js)"/)?.[1] ?? body.match(/src="(\/_next\/static\/[^" ]+\.js)"/)?.[1];
if (!asset || !(await fetch(new URL(asset, 'http://web:8080'), {signal: AbortSignal.timeout(3000)})).ok) throw new Error('static asset missing');
for (const path of ['/demo', '/v1/workspace']) {
 const denied = await fetch('http://web:8080' + path, {signal: AbortSignal.timeout(5000)});
 if (denied.ok) throw new Error('unexpected finance or demo route');
}
JS
docker stop --time 20 "$proof_id" >/dev/null
# Node may report SIGTERM as 143; SIGKILL/timeout (137) remains a failure.
case "$(docker inspect --format '{{.State.ExitCode}}' "$proof_id")" in 0|143) ;; *) exit 1 ;; esac
echo 'PASS: non-root/read-only frontend serves unconfigured sign-in and static assets; no demo or finance access.'
