#!/usr/bin/env bash
# Author and verify SQL only; this entrypoint never accepts a database URL.
set -euo pipefail
cd "$(dirname "$0")/.."
MIGRATE_IMAGE='migrate/migrate:v4.20.1@sha256:76cc2074cb6642631f34a898ced71e6aeaa6b1a4d78c4daa743275a22e0c5be7'
export MIGRATE_IMAGE
shopt -s nullglob

# Reject ambiguous versions, missing pairs and unfinished templates before use.
validate_files() {
  local file stem version previous='' direction
  local files=(db/migrations/*.sql)
  for file in "${files[@]}"; do
    [[ ! -L "$file" && "$file" =~ /([0-9]{6})_([a-z][a-z0-9_]*)\.(up|down)\.sql$ ]] || {
      echo 'FAIL: invalid migration filename or symlink.' >&2; return 1;
    }
    version=${BASH_REMATCH[1]}; direction=${BASH_REMATCH[3]}
    [[ "$version" != 000000 ]] || return 1
    stem=${file%.$direction.sql}
    [[ -s "$stem.up.sql" && -s "$stem.down.sql" ]] || {
      echo 'FAIL: migrations need nonempty up/down pairs.' >&2; return 1;
    }
    if [[ "$direction" == up ]]; then
      [[ "$version" != "$previous" ]] || { echo 'FAIL: duplicate version.' >&2; return 1; }
      previous=$version
    fi
    if grep -q 'MIGRATION_NOT_READY' "$file"; then
      echo 'FAIL: finish the migration and contract before sealing.' >&2; return 1
    fi
  done
}

# A manifest detects content drift; release-base comparison prevents resealing old SQL.
manifest() {
  local files=(db/migrations/*.sql)
  if ((${#files[@]})); then shasum -a 256 "${files[@]}"; fi
}

case "${1:-help}" in
  create)
    [[ $# == 2 && "$2" =~ ^[a-z][a-z0-9_]*$ ]] || { echo 'Use: migrate.sh create snake_case_name' >&2; exit 2; }
    docker run --rm --network none --user "$(id -u):$(id -g)" \
      --mount "type=bind,source=$PWD/db/migrations,target=/migrations" \
      "$MIGRATE_IMAGE" create -ext sql -dir /migrations -seq -digits 6 "$2"
    for file in db/migrations/*_"$2".*.sql; do
      # Newly generated empty SQL must fail visibly until a reviewed design is implemented.
      [[ -s "$file" ]] && continue
      cat > "$file" <<'SQL'
-- MIGRATION_NOT_READY: link the reviewed contract and implement this direction.
BEGIN;
DO $$ BEGIN RAISE EXCEPTION 'MIGRATION_NOT_READY'; END $$;
COMMIT;
SQL
    done
    ;;
  seal)
    validate_files
    manifest > db/migrations/SHA256SUMS
    echo 'Manifest updated. Review the diff; this does not authorize editing released SQL.'
    ;;
  check)
    validate_files
    [[ -f db/migrations/SHA256SUMS && ! -L db/migrations/SHA256SUMS ]] || exit 1
    diff -u db/migrations/SHA256SUMS <(manifest)
    if [[ $# == 2 ]]; then
      git rev-parse --verify "$2^{commit}" >/dev/null
      changed=$(git diff --name-only --diff-filter=DMRT --no-renames "$2" -- 'db/migrations/*.sql')
      [[ -z "$changed" ]] || { echo 'FAIL: existing migration SQL changed; add a forward migration.' >&2; exit 1; }
      # migrate would skip an added version older than the database's current version.
      latest=$(git ls-tree -r --name-only "$2" -- db/migrations/ | sed -nE 's@.*/([0-9]{6})_.*\.up\.sql$@\1@p' | sort | tail -1)
      if [[ -n "$latest" ]]; then
        for file in db/migrations/*.up.sql; do
          if ! git cat-file -e "$2:$file" 2>/dev/null; then
            version=${file##*/}; version=${version%%_*}
            [[ "$version" > "$latest" ]] || { echo 'FAIL: new migration must follow the base version.' >&2; exit 1; }
          fi
        done
      fi
    fi
    echo 'PASS: migration pairs and checksums (no SQL execution).'
    ;;
  test)
    bash scripts/migrate.sh check
    exec bash scripts/test-migrations.sh
    ;;
  *) echo 'Usage: bash scripts/migrate.sh {create NAME|seal|check [BASE_COMMIT]|test}'; exit 2 ;;
esac
