<!-- stage-runbook: true -->
# Database baseline verification release

## Status and boundary

Owner requested hosted API/frontend deployment after #126. Application images
built successfully in [38654097056](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38654097056)
at `5c10e944678e15caf5f832fc87b5c95aa6f86456`; all validation and four images PASS.
Application promotion remains blocked by open #26/#11. No services are deployed.

This release requests only `verify`: no schema upgrade, replay or application
rollout. Contract: [read-only status](../contracts/migration-status.md).

## Stage ledger

- Read cloud resource metadata: PostgreSQL17 instance RUNNABLE, existing migration
  job present; no regional Cloud Run services. No credentials or data inspected.
- Retained successful build38327676280 at `0376b8793c882311f2d9cf0fbb2e99ecfc0bf540`
  supplies status-capable images with migrations1–5, matching the last recorded
  execution. Those ten artifact hashes remain unchanged in current main.
- Downloaded matching api/migrate/web receipts; all retained scans have zero
  HIGH/CRITICAL findings. Historical scans are not fresh vulnerability assessments.
- Prepared exact `deploy/dev-release.json` with operation=verify and fresh expiry.

## Current evidence

PASS: exact manifest/expiry/build validator, bootstrap ancestry,
`make check test build release-test`, and staged runbook validation. Hosted
verification now PASS in workflow38663999729, execution jayburd-dev-migrate-sp7xd,
completed 2026-10-15T01:11:09.256743Z. The retained receipt and independently
validated allowlisted metadata match the exact image, identity, status action,
one task, zero retries and 660-second timeout. This establishes clean version5
and exact baseline artifact hashes, without an upgrade.

PASS: fresh local `make migration-test` and `make cloud-sql-test check test build`,
including reviewed role-bootstrap replay and restricted migrator/API checks.
These local results do not substitute for the pending hosted role bootstrap.

Latest images embed migrations1–15; using them to check the old
baseline would fail the exact-version contract. Only the older migrator executes;
its API/web image references satisfy same-build provenance and are never deployed.

## Remaining work

#127 is merged and its exact-hash verification succeeded. Before the forward
upgrade, rerun the reviewed `db/bootstrap/cloud-sql.sql` through the retained
operator to precreate safe NOLOGIN publisher/observer/sync/webhook roles. The
original hosted bootstrap predates these additions; migrator cannot CREATE ROLE.
Do not assume the roles exist or grant CREATEROLE to the migrator.

Cloud Shell command access PASS, but its project authorization check is BLOCKED.
Owner must reopen Cloud Shell and run:
`gcloud projects describe jayburd-bookkeeping-dev --format='value(projectNumber)'`,
authorize the session, and confirm public project number9088794002. No credential
transfer or `--authorize-session` command was used by the agent. After restoration,
check operator capabilities, execute hash-verified bootstrap, and read back safe
role attributes/memberships before preparing a dispatchable upgrade manifest.

Then prepare the forward upgrade to migrations1–15 and post-upgrade verification.
Application promotion still requires actual API IAM denials, ownership review,
restore/load acceptance (#26) and outstanding isolation/rotation/revocation/recovery
proof (#11), or an explicitly reviewed change to development release scope. Do not
close those issues merely to unblock the workflow. Identity and worker enrollment
also need hosted acceptance before an interactive bank-to-review claim.

## PR handoff

The read-only prerequisite succeeded; forward upgrade remains blocked on operator
authorization and role bootstrap. No frontend deployment.
No SQL or application implementation change. Source/build compatibility evidence:
https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6181599981.
Existing local demo remains running. No real bank activation is authorized.

Reviewed candidate SHA256:
`3b1962af44c61bf1b26f09ab0f19e3e895c5885b0beaebc0eb037f649903e25d`.

Hosted result: https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6182078948.
