#!/usr/bin/env bash
# Open a short-lived IAM-authenticated dev Cloud SQL proxy for the read-only binding lookup.
set -euo pipefail

if [[ $# != 2 ]]; then
  echo 'Usage: run-worker-binding-cloud-shell.sh ENTITY_UUID BOOK_UUID' >&2
  exit 2
fi
for command in gcloud cloud-sql-proxy psql; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo 'Run this in the authorized Google Cloud Shell, which has gcloud, cloud-sql-proxy and psql.' >&2
    exit 1
  fi
done
operator=$(gcloud config get-value account 2>/dev/null) || {
  echo 'Authorize Google Cloud Shell with the development database operator account first.' >&2
  exit 1
}
if [[ -z "$operator" || "$operator" == '(unset)' ]]; then
  echo 'Authorize Google Cloud Shell with the development database operator account first.' >&2
  exit 1
fi
if [[ "$operator" != 'personal.jnapolitano@gmail.com' ]]; then
  echo 'Cloud Shell is not using the expected development database operator account.' >&2
  exit 1
fi

umask 077
socket_dir=$(mktemp -d /tmp/jb-binding-sql.XXXXXX)
proxy_pid=''
cleanup() {
  if [[ -n "$proxy_pid" ]]; then
    kill "$proxy_pid" 2>/dev/null || true
    wait "$proxy_pid" 2>/dev/null || true
  fi
  rm -rf "$socket_dir"
}
trap cleanup EXIT
trap 'exit 130' INT TERM
instance='jayburd-bookkeeping-dev:us-central1:jayburd-dev'
cloud-sql-proxy --auto-iam-authn --unix-socket="$socket_dir" \
  --quota-project=jayburd-bookkeeping-dev "$instance" >"$socket_dir/proxy.log" 2>&1 &
proxy_pid=$!
for ((i=0; i<30; i++)); do
  if [[ -S "$socket_dir/$instance/.s.PGSQL.5432" ]]; then break; fi
  if ! kill -0 "$proxy_pid" 2>/dev/null; then break; fi
  sleep 1
done
if [[ ! -S "$socket_dir/$instance/.s.PGSQL.5432" ]]; then
  echo 'Cloud SQL proxy did not become ready. Check Cloud Shell authorization and project access.' >&2
  exit 1
fi

export PGHOST="$socket_dir/$instance" PGPORT=5432 PGUSER="$operator" PGPASSFILE=/dev/null PGCONNECT_TIMEOUT=5
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
"$script_dir/show-worker-binding.sh" "$1" "$2"
