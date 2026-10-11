#!/usr/bin/env bash
# Fault injection runs in a synthetic copy; never modifies actual migration history.
set -euo pipefail
cd "$(dirname "$0")/.."
test_root=$(mktemp -d "${TMPDIR:-/tmp}/jayburd-migration-files.XXXXXX")
trap 'rm -rf "$test_root"' EXIT
mkdir -p "$test_root/scripts" "$test_root/db/migrations"
cp scripts/migrate.sh "$test_root/scripts/"
cp tests/migrations/good/*.sql "$test_root/db/migrations/"
cd "$test_root"

# Assert a named invalid bundle fails closed, suppressing expected diagnostics.
reject() {
  if bash scripts/migrate.sh "$@" >/dev/null 2>&1; then
    echo "FAIL: expected rejection from $*" >&2; exit 1
  fi
}
bash scripts/migrate.sh seal >/dev/null
bash scripts/migrate.sh check >/dev/null
printf '\n-- changed\n' >> db/migrations/000001_probe.up.sql
reject check
bash scripts/migrate.sh seal >/dev/null
mv db/migrations/000001_probe.down.sql db/migrations/missing.txt
reject check
mv db/migrations/missing.txt db/migrations/000001_probe.down.sql
cp db/migrations/000001_probe.up.sql db/migrations/000001_duplicate.up.sql
cp db/migrations/000001_probe.down.sql db/migrations/000001_duplicate.down.sql
reject seal
rm db/migrations/000001_duplicate.*.sql
printf '\n-- MIGRATION_NOT_READY\n' >> db/migrations/000001_probe.up.sql
reject seal
sed '/MIGRATION_NOT_READY/d' db/migrations/000001_probe.up.sql > db/migrations/fixed.txt
mv db/migrations/fixed.txt db/migrations/000001_probe.up.sql
bash scripts/migrate.sh seal >/dev/null
mv db/migrations/000001_probe.up.sql db/migrations/000002_probe.up.sql
mv db/migrations/000001_probe.down.sql db/migrations/000002_probe.down.sql
bash scripts/migrate.sh seal >/dev/null
git init -q
git add db scripts
git -c user.name=Fixture -c user.email=fixture@example.invalid commit -qm baseline
printf '\n-- changed after release\n' >> db/migrations/000002_probe.up.sql
bash scripts/migrate.sh seal >/dev/null
reject check HEAD
git show HEAD:db/migrations/000002_probe.up.sql > db/migrations/000002_probe.up.sql
cp db/migrations/000002_probe.up.sql db/migrations/000001_older.up.sql
cp db/migrations/000002_probe.down.sql db/migrations/000001_older.down.sql
bash scripts/migrate.sh seal >/dev/null
reject check HEAD
echo 'PASS: checksum drift, missing pair, duplicate version, unfinished template, resealed history and out-of-order addition rejected.'
