# Development Cloud SQL provisioning

<!-- stage-runbook: true -->

## Status and boundary

Owner authorized the next Cloud SQL slice after merging #41 on 2026-10-12.
Claimed #26 on `issue/26-cloud-sql-bootstrap`, base `5f99dd3`. Use merged Terraform
source; no real financial data, credential retrieval or plan/state inspection.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Review | Existing database/role contract and bootstrap reviewed; 19 Terraform mocked runs pass | Apply scoped infrastructure |
| Foundation dependency | Apply succeeded; private drift check exit 0; deployment provider read back disabled | Retain separate runtime state and disabled deployment federation |
| Database provisioning | Apply succeeded; drift check exit 0; API readback RUNNABLE with expected settings and two IAM logins | Separately validate SQL bootstrap |

## Current evidence

- PASS: owner-operated foundation and application applies from merged `5f99dd3`,
  Terraform 1.15.7 / Google provider 8.6.0. Both private drift checks returned 0.
- PASS: Cloud SQL API selected-field readback: `jayburd-dev`, `RUNNABLE`,
  PostgreSQL 17 Enterprise, `us-central1`, zonal `db-g1-small`, 10 GiB SSD with
  20 GiB growth limit, deletion protection, connector enforcement `REQUIRED`,
  encrypted-only transport, no authorized networks, IAM authentication and
  `max_connections=50`. Seven retained daily backups and seven-day PITR are
  configured; successful backup/restore is not yet demonstrated.
- PASS: database `bookkeeping` and separate API/migrator IAM service-account
  SQL logins read back by name/type only. SQL permissions and login are untested.
- PASS: deployment provider `owner-deploy` read back disabled. Foundation moved
  deployer object access to the runtime state bucket; database state stays separate.
- PASS: `make infra-test` (19 mocked runs) and `make check test build`
  (Go 1.26.9, vet/race tests, document links, immutable migration checks, binaries).
- No SQL bootstrap, migration execution, enrollment, image publication or Cloud Run
  deployment was performed. Terraform used existing ADC internally; credentials,
  logs, state and plan contents were not inspected.

## Remaining work

Follow-up: [IAM operator implementation and connectivity evidence](iam-bootstrap.md)
records the next attempt and its local transport blocker.

1. Establish the administrative SQL bootstrap path. The current Terraform root
   provisions only API/migrator logins; neither is an administrator. G24 in the
   [source register](../research/gcp-sources.md) documents a possible separate IAM
   operator path. It is a proposal, not an applied grant or proven compatibility.
2. Prove the administrator has the database/schema ownership, CREATEROLE and
   authority over both IAM logins required by `db/bootstrap/cloud-sql.sql`.
   Execute the reviewed bootstrap and demonstrate API DDL/table denial and
   restricted migrator readiness. Do not weaken guards to accommodate Cloud SQL.
3. Run the pinned migration artifact with its dedicated identity; prove replay,
   checksums and clean version state. Record image/job provenance under #27.
4. Complete shared-core connection/load recovery, separate-instance restore/PITR,
   identity mapping and hosted finance acceptance under #26/#6/#27.

Use secret-free configuration and redacted outcomes. Never retrieve a password or
print an access token. If the IAM operator cannot satisfy the administrator gate,
the owner must inject any required credential privately; this is not permission
to give the API or migrator cloud administrator membership.

## Operational resume

Private operational copies: `.local/cloud-sql/foundation` and
`.local/cloud-sql/application` in this worktree. Logs, plans and state remain
protected and excluded from Git. GCS backends use
`jayburd-bookkeeping-dev-jayburd-dev-foundation-state` / prefix `foundation` and
`jayburd-bookkeeping-dev-jayburd-dev-application-state` / prefix `application`.
Initialize from the merged source if the temporary worktree is unavailable.

Both roots require `project_id=jayburd-bookkeeping-dev`. Foundation also uses
`project_number=9088794002`, `billing_account_id=012EA6-AB37B6-77BD42`,
`enable_github_build_identity=true`, `enable_github_deploy_identity=false`.
With existing user ADC, supply `USER_PROJECT_OVERRIDE=true` and
`GOOGLE_BILLING_PROJECT=jayburd-bookkeeping-dev` (source G17).
Keep command output private; a `plan -detailed-exitcode` result of 0 means no drift,
1 failure, 2 changes. Do not inspect or publish its contents.

## PR handoff

Keep #26 open until its named cloud acceptance checks pass. This runbook records
actual results, not readiness inferred from a Terraform apply.

## Provisioning estimate

Rechecked [Google's Iowa list prices](https://cloud.google.com/sql/pricing) on
2026-10-12; the full page exceeds the research tool's response limit, but indexed
regional compute/storage tables were available. This arithmetic estimate replaces
an interactive calculator export for this authorized development apply:

| Assumption | Monthly USD at 730 hours |
| --- | ---: |
| db-g1-small × 0.035/hour | 25.55 |
| Initial 10 GiB SSD × 0.000232877/GiB-hour | 1.70 |
| Assumed 10 GiB used backups × 0.000109589/GiB-hour | 0.80 |
| Baseline subtotal | 28.05 |

At the 20 GiB disk cap and assumed 20 GiB used backups, the subtotal is USD30.55.
Backup usage is not capped by that assumption. Retain the existing USD5
backup/PITR/restore allocation; a four-hour same-tier restore adds roughly USD0.15
compute/storage before backup/log/egress charges. Other services, log/PITR growth,
network egress, taxes and actual restore duration are additional; the USD50
project alert is not a hard limit. Observed spend remains #12 evidence.
