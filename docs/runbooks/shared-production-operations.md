<!-- stage-runbook: true -->
# Shared Production SQL operations — #316/#337

## Status and boundary

SHARED316-OPS; ops/316-shared-sql from f69f3f6. Owner authorized reuse of
the existing instance, migrations, and the specific browser-mediated access
grants. Operations target bookkeeping_prod on
jayburd-bookkeeping-dev:us-central1:jayburd-dev. No bank connection, provider
payload, application secret or financial record was accessed.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source | PR357 merged f69f3f6 after all five checks passed |
| Operator | Cloud Shell browser terminal and pinned official proxy2.26.0; catalog-only IAM operator CREATEROLE/CREATEDB confirmed |
| Database | Created with ALLOW_CONNECTIONS false; PUBLIC rights revoked before enabling connections |
| Identities | Separate jayburd-prod-api and jayburd-prod-migrator service accounts and IAM SQL users; cloudsql.client and cloudsql.instanceUser only added |
| Bootstrap | Merged wrapper checksum verified; transaction committed and replay committed |
| Isolation | Actual catalog grants deny all five existing jayburd-dev logins Production CONNECT/CREATE; both Production logins lack Sandbox CONNECT |
| Build | Existing reviewed Build images run40806816655 at exact merged SHA; migrator job/runtime/scan PASS; separate API scan failed, tracked in #358 |
| Repository checks | make check/test/build PASS with installed Go1.26.5 and external temporary modfile; tracked Go1.26.10 unchanged |
| Additional synthetic SQL proof | Existing isolated two-database bootstrap test extended temporarily to run all35 up.sql files as the Production migrator login; PASS, including real cross-DB login denials |
| Managed migration | jayburd-prod-migrate-rs5z5 completed successfully at2026-10-29T02:56:44Z, one task, under the restricted Production workload identity |
| Managed verification | jayburd-prod-migrate-qncsb completed successfully at2026-10-29T02:59:55Z; exact-artifact Status checks latest version35, clean flag and all recorded checksums |

## Current evidence

Bootstrap SHA256:
047f3bef21ce78bba33c932c6e81e1430b6d146cf9f87b0a96accceab1da7a21.
Production API has CONNECT but no database CREATE; migrator has CONNECT/CREATE.
Both new SQL users have no superuser, CREATEDB, CREATEROLE, BYPASSRLS,
REPLICATION or cloudsqlsuperuser membership. No unexpected Production sessions
or foreign servers were present. Bootstrap's capability-role CONNECT guards
passed. These are administrator catalog readbacks, not independent API login
or attempted cross-database connection proofs.
Project IAM readback confirms the only bindings for the two new identities are
cloudsql.client and cloudsql.instanceUser. Managed database listing confirms
bookkeeping_prod alongside the original bookkeeping and postgres databases.

Migrator digest: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/migrate@sha256:7ca37ddda3936998fec11967519aa467bb29e1c2ecf261c7f69a538f1a752cdb.
The successful component receipt matches the exact commit, tested image ID and
published digest; scan contains zero HIGH/CRITICAL vulnerabilities. This existing
builder artifact is accepted only for the explicitly authorized migration;
the full application run/release is not accepted. The API scan reports
go-jose/v4 v4.1.4/CVE-2026-116587/HIGH, with fixed version4.1.6.

Dedicated job jayburd-prod-migrate uses the Production migrator identity, one
task/parallelism1, retries0, timeout660s, CPU1/memory512Mi, exact shared target,
environment=production, database=bookkeeping_prod and deadline2026-10-29T04:00:00Z.
No secrets or application runtime configuration are mounted. Read-only initial
status execution h4qbl failed; independent operator metadata confirms both
deployment.schema_migrations and deployment.migration_artifact were absent.
This expected fresh-schema rejection alone does not establish runtime login.
The subsequent successful up/status executions establish the restricted
migrator's actual IAM SQL connection and clean complete schema at version35.
The operator's direct migration-table query was denied; no extra privileges
were granted. Verification therefore uses the reviewed Status implementation
under the actual migrator identity, not an administrator bypass.
Post-migration catalog readback again found zero Sandbox/shared capability roles
with Production CONNECT and both Production logins denied Sandbox CONNECT.
The operator proxy was stopped. The unscheduled job is left in read-only status
mode with its original finite deadline; no live application release occurred.

The first SQL user creation request incorrectly used the full service-account
email and was rejected before mutation for PostgreSQL identifier length. The
correct Cloud SQL PostgreSQL names omit .gserviceaccount.com; both then succeeded.
SSH fails before handshake with a tunnel connection reset; browser Cloud Shell
is the working operator path. No credentials or private logs were inspected.

## Remaining work

Migrations1–35 are applied and verified. Independent API runtime login/denial
proof, shared-instance runtime support, infrastructure reconciliation, API
dependency fix #358 and full pilot release gates remain outstanding. This database
setup does not enable live bank registration or establish the GCP cost target.

## PR handoff

Refs #316 and #337; keep both open. No new or modified product SQL migration.
Existing Sandbox database/resources were preserved. No financial rows or bank
credentials were read or copied. Neither issue is completed by database setup.
