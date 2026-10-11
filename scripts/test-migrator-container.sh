#!/usr/bin/env bash
# Build the allowlisted migration image and prove its default action needs no network or identity.
set -euo pipefail
cd "$(dirname "$0")/.."
tag="jayburd-migrate-proof:$(date +%s)-$$"
built=0
proof_dir=""
cleanup() {
 if [[ "$built" == 1 ]]; then docker image rm "$tag" >/dev/null; fi
 if [[ -n "$proof_dir" ]]; then rm -rf "$proof_dir"; fi
}
trap cleanup EXIT
trap 'exit 130' INT TERM
case "${1:-}" in
 "") docker build -f Dockerfile.migrate -t "$tag" . ;;
 --local-build)
  # Explicit runtime-only fallback when the Docker VM lacks its outbound proxy CA.
  proof_dir="$(mktemp -d "${TMPDIR:-/tmp}/jayburd-migrate-proof.XXXXXX")"
  proof_arch="$(docker version --format '{{.Server.Arch}}')"
  case "$proof_arch" in arm64|amd64) ;; *) echo 'Unsupported Docker architecture' >&2; exit 1 ;; esac
  CGO_ENABLED=0 GOOS=linux GOARCH="$proof_arch" go build -trimpath -ldflags='-s -w' -o "$proof_dir/migrate" ./cmd/migrate
  cat > "$proof_dir/Dockerfile" <<'DOCKERFILE'
FROM golang:1.26.9-alpine@sha256:3082400e369fa24d5fc60bca20edab3f6d604e0c5a690ec66b295eff4dd87ade AS certificates
FROM scratch
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY migrate /migrate
USER 65532:65532
ENTRYPOINT ["/migrate"]
CMD ["-action=verify"]
DOCKERFILE
  docker build -f "$proof_dir/Dockerfile" -t "$tag" "$proof_dir"
  echo 'PARTIAL: host build proves runtime only; canonical source-builder remains a CI gate'
  ;;
 *) echo 'Usage: scripts/test-migrator-container.sh [--local-build]' >&2; exit 1 ;;
esac
built=1
[[ "$(docker image inspect --format '{{.Config.User}}' "$tag")" == '65532:65532' ]]
docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges "$tag"
if docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges "$tag" -action=down; then
 echo 'FAIL: down action accepted' >&2; exit 1
fi
echo 'PASS: non-root migration image verifies offline and rejects down.'
