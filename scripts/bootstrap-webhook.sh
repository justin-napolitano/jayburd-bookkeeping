#!/usr/bin/env bash
# Explicit owner operation over an already authorized Unix socket; no cloud or secret discovery.
set -euo pipefail
usage() { echo 'Usage: bootstrap-webhook.sh --check|--apply --project jayburd-bookkeeping-dev --operator-user SQL_LOGIN --socket-dir ABSOLUTE_PATH --operation-key STABLE_KEY --enabled true|false'; }
fail() { echo 'BLOCKED: webhook bootstrap metadata or execution rejected.' >&2; exit 1; }
mode=''; project=''; operator=''; socket=''; key=''; enabled=''
while [[ $# -gt 0 ]]; do
 case "$1" in
  --help) usage; exit 0 ;;
  --check|--apply) [[ -z "$mode" ]] || fail; mode="$1"; shift ;;
  --project|--operator-user|--socket-dir|--operation-key|--enabled)
   [[ $# -ge 2 ]] || fail
   case "$1" in
    --project) [[ -z "$project" ]] || fail; project="$2" ;;
    --operator-user) [[ -z "$operator" ]] || fail; operator="$2" ;;
    --socket-dir) [[ -z "$socket" ]] || fail; socket="$2" ;;
    --operation-key) [[ -z "$key" ]] || fail; key="$2" ;;
    --enabled) [[ -z "$enabled" ]] || fail; enabled="$2" ;;
   esac
   shift 2 ;;
  *) fail ;;
 esac
done
[[ "$mode" == --check || "$mode" == --apply ]] || { usage; exit 2; }
[[ "$project" == jayburd-bookkeeping-dev && "$operator" =~ ^[A-Za-z0-9][A-Za-z0-9_.@-]{0,62}$ && "$socket" == /* && "$socket" != *,* && "$socket" != *$'\n'* && "$socket" != *$'\r'* && "$key" =~ ^[A-Za-z0-9._:-]{1,128}$ && ( "$enabled" == true || "$enabled" == false ) ]] || fail
[[ "$operator" != jayburd-dev-webhook@jayburd-bookkeeping-dev.iam ]] || fail
if [[ "$mode" == --check ]]; then echo 'PASS: webhook bootstrap metadata only; no execution.'; exit 0; fi
[[ -S "$socket/.s.PGSQL.5432" ]] || fail
psql_bin=$(command -v psql) || fail
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# Clear libpq overrides; suppress raw SQL, catalog results and server diagnostics on every outcome.
if ! env -i PGCONNECT_TIMEOUT=5 PGAPPNAME=jayburd-webhook-bootstrap PGPASSFILE=/dev/null PGSERVICEFILE=/dev/null \
 "$psql_bin" -X -w --host="$socket" --port=5432 --username="$operator" --dbname=bookkeeping \
 --set=ON_ERROR_STOP=on --set=project_id="$project" --set=operation_key="$key" --set=enabled="$enabled" \
 --file="$script_dir/../db/bootstrap/webhook-login.sql" >/dev/null 2>&1; then fail; fi
echo 'PASS: webhook bootstrap operation committed.'
