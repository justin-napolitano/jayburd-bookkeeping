<!-- stage-runbook: true -->
# Shared-instance Production database — #316/#337

## Status and boundary

SHARED316-SQL; build/316-shared-sql-pilot from53351a7. Owner requested separate
Production database on existing Cloud SQL and migrations. Preparation source only;
no database/resource creation, live SQL, credentials or financial data accessed.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Metadata readback | jayburd-dev PostgreSQL17/us-central1/db-g1-small/10GiB ZONAL; databases bookkeeping/postgres only |
| Owner choice |337 comment6485227284 records shared instance; separate-instance proposal superseded for pilot |
| Target routing | Explicit bookkeeping_prod migration-only exception; exact instance/environment/IAM login and finite deadline |
| Isolation | Two disposable DBs: both profiles denied cross-DB connection; replay safe; bad grant rolls back |
| Validation | Focused tests, full core check/test/build, migration-test and release-test PASS |

## Current evidence

Metadata-only IAM/SQL user listings found no jayburd-prod service accounts or
SQL logins. Local checks use installed Go1.26.5/external temporary modfile; tracked
1.26.10 unchanged. No managed database/logins were created.
Default migrations still target bookkeeping. Production runtime guards remain
unchanged; only explicit migrator selection admits shared instance. Bootstrap
requires administrator-owned bookkeeping_prod and rejects Production CONNECT to
Sandbox or Sandbox/capability CONNECT to Production. Shared roles are not separate
physical/admin isolation. Existing generic bootstrap and merged SQL unchanged.

## Remaining work

Human review of new shared-instance boundary before managed use. Create database
initially closed to connections; revoke PUBLIC and set exact owner/login grants
before enabling access. Verify real catalog privileges, no existing cross-database
sessions/bridges and intended server identity. Provision restricted Production
IAM logins and reviewed workload-identity migrator job/image, then run status/up/
status and confirm migration35 clean with exact checksums. No administrator-run
product migrations. Runtime/release validators still require dedicated follow-up.
Existing instance sizing is verified; total25USD GCP target is not established.

## PR handoff

Refs316/337. No live migration applied. USD25 target excludes Plaid. Synthetic
proof does not establish real IAM acceptance, actual role grants or hosted sync.
