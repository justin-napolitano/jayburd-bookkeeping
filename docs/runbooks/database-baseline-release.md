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
`make check test build release-test`, and staged runbook validation. Actual status
has not run. Latest images embed migrations1–15; using them to check the old
baseline would fail the exact-version contract. Only the older migrator executes;
its API/web image references satisfy same-build provenance and are never deployed.

## Remaining work

Owner reviews and merges this exact candidate. Dispatch `deploy-dev.yml` on main
with its SHA256. Require successful exact-image/action readback and the redacted
migration-verification receipt. Any dirty, missing, changed or unexpected version
blocks further migration; never force or down-migrate.

Then prepare the forward upgrade to migrations1–15 and post-upgrade verification.
Application promotion still requires actual API IAM denials, ownership review,
restore/load acceptance (#26) and outstanding isolation/rotation/revocation/recovery
proof (#11), or an explicitly reviewed change to development release scope. Do not
close those issues merely to unblock the workflow. Identity and worker enrollment
also need hosted acceptance before an interactive bank-to-review claim.

## PR handoff

This is the concrete next deployment prerequisite, not a frontend deployment.
No SQL or application implementation change. Source/build compatibility evidence:
https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6181599981.
Existing local demo remains running. No real bank activation is authorized.

Reviewed candidate SHA256:
`3b1962af44c61bf1b26f09ab0f19e3e895c5885b0beaebc0eb037f649903e25d`.
