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
| Build | Existing reviewed Build images run40806816655 at exact merged SHA; pending completion |
| Repository checks | make check/test/build PASS with installed Go1.26.5 and external temporary modfile; tracked Go1.26.10 unchanged |

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

The first SQL user creation request incorrectly used the full service-account
email and was rejected before mutation for PostgreSQL identifier length. The
correct Cloud SQL PostgreSQL names omit .gserviceaccount.com; both then succeeded.
SSH fails before handshake with a tunnel connection reset; browser Cloud Shell
is the working operator path. No credentials or private logs were inspected.

## Remaining work

Accept the immutable migrator build receipt, execute a dedicated restricted
Production migration job with exact target and finite deadline, and verify
clean version35/checksum receipts. No product migration has run yet. Independent
runtime login/denial proof, shared-instance runtime support, infrastructure
reconciliation and full pilot release gates remain outstanding. This database
setup does not enable live bank registration or establish the GCP cost target.

## PR handoff

Refs #316 and #337; keep both open. No new or modified product SQL migration.
Preserve existing Sandbox resources and data. Report actual job results here
before claiming managed migrations applied.
