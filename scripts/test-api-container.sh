#!/usr/bin/env bash
# Build only the allowlisted source, then prove inert defaults on an internal network.
set -euo pipefail
cd "$(dirname "$0")/.."
probe_image='golang:1.26.9-alpine@sha256:3082400e369fa24d5fc60bca20edab3f6d604e0c5a690ec66b295eff4dd87ade'
proof_id="jayburd-api-proof-$$-$(date +%s)"
proof_image="$proof_id:local"
proof_dir=""
owned_image=1
cleanup() {
 docker rm -f "$proof_id" >/dev/null 2>&1 || true
 docker network rm "$proof_id" >/dev/null 2>&1 || true
 if [[ "$owned_image" == 1 ]]; then docker image rm "$proof_image" >/dev/null 2>&1 || true; fi
 if [[ -n "$proof_dir" ]]; then rm -rf "$proof_dir"; fi
}
trap cleanup EXIT INT TERM
# No credentials, environment forwarding, bind mounts or published ports.
case "${1:-}" in
 "") docker build --file Dockerfile.api --tag "$proof_image" . ;;
 --image)
  [[ $# == 2 && "$2" == sha256:* ]] || exit 1
  proof_image="$2"; owned_image=0 ;;
 --local-build)
  # Explicit fallback for a Docker VM lacking its outbound proxy CA. Never disable TLS.
  proof_dir="$(mktemp -d "${TMPDIR:-/tmp}/jayburd-api-proof.XXXXXX")"
  proof_arch="$(docker version --format '{{.Server.Arch}}')"
  case "$proof_arch" in arm64|amd64) ;; *) echo 'Unsupported Docker architecture' >&2; exit 1 ;; esac
  CGO_ENABLED=0 GOOS=linux GOARCH="$proof_arch" go build -trimpath -ldflags='-s -w' -o "$proof_dir/api" ./cmd/api
  CGO_ENABLED=0 GOOS=linux GOARCH="$proof_arch" go build -trimpath -ldflags='-s -w' -o "$proof_dir/database-probe" ./cmd/database-probe
  cat > "$proof_dir/Dockerfile" <<DOCKERFILE
FROM $probe_image AS certificates
FROM scratch
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY api /api
COPY database-probe /database-probe
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/api"]
CMD ["-mode=health", "-listen=0.0.0.0:8080"]
DOCKERFILE
  docker build --file "$proof_dir/Dockerfile" --tag "$proof_image" "$proof_dir"
  echo 'PARTIAL: explicit host build proves runtime only; canonical source-builder remains a CI gate'
  ;;
 *) echo 'Usage: scripts/test-api-container.sh [--local-build | --image sha256:ID]' >&2; exit 1 ;;
esac
[[ "$(docker image inspect --format '{{.Config.User}}' "$proof_image")" == '65532:65532' ]]
# The additional proof command is inert by default and rejects invalid actions offline.
docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges --entrypoint /database-probe "$proof_image"
if docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges --entrypoint /database-probe "$proof_image" -action=up; then
 echo 'FAIL: invalid database probe action accepted' >&2; exit 1
fi
docker network create --internal "$proof_id" >/dev/null
docker run -d --name "$proof_id" --network "$proof_id" --network-alias api \
 --read-only --user 65532:65532 --cap-drop ALL --security-opt no-new-privileges \
 --memory 256m --cpus 1 --pids-limit 64 "$proof_image" >/dev/null
[[ "$(docker inspect --format '{{.Config.User}} {{.HostConfig.ReadonlyRootfs}}' "$proof_id")" == '65532:65532 true' ]]
# The pinned builder includes BusyBox wget; probe only the synthetic health container.
docker run --rm --network "$proof_id" --read-only --cap-drop ALL \
 --security-opt no-new-privileges --user 65532:65532 --entrypoint /bin/sh "$probe_image" -ec '
 for attempt in $(seq 1 40); do
  if wget -q -O /dev/null http://api:8080/healthz; then break; fi
  sleep 0.25
 done
 for spec in "/healthz 200" "/readyz 503" "/v1/workspace 404" "/v1/setup 404"; do
  set -- $spec
  result=$(wget -S -O /dev/null "http://api:8080$1" 2>&1 || true)
  if ! printf "%s\n" "$result" | grep -q "HTTP/1.1 $2"; then
   echo "FAIL: health-only route contract" >&2; exit 1
  fi
 done
'
docker stop --time 20 "$proof_id" >/dev/null
[[ "$(docker inspect --format '{{.State.ExitCode}}' "$proof_id")" == '0' ]]
echo 'PASS: health-only API image; non-root/read-only; isolated network; absent ledger routes; graceful stop'
