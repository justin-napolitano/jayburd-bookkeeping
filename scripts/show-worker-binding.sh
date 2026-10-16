#!/usr/bin/env bash
# Show only the active worker binding ID for one book via the owner's existing psql connection.
set -euo pipefail

usage='Usage: scripts/show-worker-binding.sh ENTITY_UUID BOOK_UUID (from an owner psql connection to the development bookkeeping database)'
if [[ $# != 2 ]]; then
  echo "$usage" >&2
  exit 2
fi
uuid='^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12}$'
if [[ ! $1 =~ $uuid || ! $2 =~ $uuid ]]; then
  echo 'Entity and book IDs must both be UUIDs.' >&2
  exit 2
fi
if ! command -v psql >/dev/null 2>&1; then
  echo 'psql is required; run this where the Cloud SQL proxy and owner IAM login are configured.' >&2
  exit 1
fi

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# Never request a password interactively or echo connection diagnostics/financial rows.
if ! binding=$(psql -X -qAt -w -v ON_ERROR_STOP=1 -v "entity_id=$1" -v "book_id=$2" \
  -d bookkeeping -f "$script_dir/../db/bootstrap/worker-binding-id.sql" 2>/dev/null); then
  echo 'Lookup failed. Verify the dev Cloud SQL proxy, owner IAM login, and bookkeeping database.' >&2
  exit 1
fi
if [[ ! $binding =~ $uuid ]]; then
  echo 'No active worker binding was found for that entity and book.' >&2
  exit 1
fi
printf '%s\n' "$binding"
