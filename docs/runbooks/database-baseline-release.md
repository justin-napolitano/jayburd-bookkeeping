<!-- stage-runbook: true -->
# Database baseline verification and forward release

## Status and boundary

Owner requested hosted API/frontend deployment after #126. Application images
built successfully in [38654097056](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38654097056)
at `5c10e944678e15caf5f832fc87b5c95aa6f86456`; all validation and four images PASS.
Application promotion remains blocked by open #26/#11. No services are deployed.

Baseline `verify` succeeded. The updated candidate requests migration5→15 only;
no application rollout. Contracts: [read-only status](../contracts/migration-status.md)
and [migration-only release](../contracts/migration-release.md).

## Stage ledger

- Read cloud resource metadata: PostgreSQL17 instance RUNNABLE, existing migration
  job present; no regional Cloud Run services. No credentials or data inspected.
- Retained successful build38327676280 at `0376b8793c882311f2d9cf0fbb2e99ecfc0bf540`
  supplies status-capable images with migrations1–5, matching the last recorded
  execution. Those ten artifact hashes remain unchanged in current main.
- Downloaded matching api/migrate/web receipts; all retained scans have zero
  HIGH/CRITICAL findings. Historical scans are not fresh vulnerability assessments.
- Executed the approved baseline verify manifest; prepared its successor with
  operation=migrate, current successful build38654097056 and fresh expiry.

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
These local results do not substitute for hosted migration acceptance.

PASS: restored Cloud Shell authorization and retained operator capability check.
Hash-verified merged bootstrap from build commit5c10e944 ran and replayed with
ON_ERROR_STOP. SHA256:
`c19921224c635ad7105f5a46d1a6d6e66012a0699a1d3f3077280ea273cf15dd`.
Catalog-only readback confirms all seven capability roles are safe NOLOGIN roles;
API lacks DDL and privileged/worker membership; migrator retains explicit owner
SET without inheritance, ADMIN or CREATEROLE. Connection limits remain20/2.
Pinned proxy2.26.0 stopped on exit; private diagnostics stayed private.
The operator remains retained for existing ownership/grantor dependencies.

PASS: forward manifest exact-hash validation and all three receipts against the
successful owner-dispatched build; `make check test build release-test` rerun.
Forward candidate SHA256:
`1a23467c0d44ff044e2702591edc1a4d6144eea1a77c113207cfa0d10ebabc32`.

Latest images embed migrations1–15; using them to check the old baseline would
fail the exact-version contract. The completed baseline check used the older
migrator. The forward candidate selects the newer migrator; API/web references
satisfy same-build provenance and are never deployed by either operation.

## Remaining work

#127 is merged and its exact-hash verification succeeded. Role prerequisites are
now established. Owner reviews the exact updated manifest in #128 before merge
and dispatch. The restricted migrator applies pending versions6–15, then a separate
reviewed verify release checks clean version15 and hashes using the new image.
No credential transfer or `--authorize-session` command was used.

Do not replay automatically after a failed execution or force dirty state. Review
the receipt first. API/web image references provide same-build provenance only;
neither service is deployed by this release.
Application promotion still requires actual API IAM denials, ownership review,
restore/load acceptance (#26) and outstanding isolation/rotation/revocation/recovery
proof (#11), or an explicitly reviewed change to development release scope. Do not
close those issues merely to unblock the workflow. Identity and worker enrollment
also need hosted acceptance before an interactive bank-to-review claim.

## PR handoff

The baseline and hosted role prerequisites succeeded; forward upgrade awaits owner
review, merge and exact-hash dispatch. No frontend deployment.
No SQL or application implementation change. Source/build compatibility evidence:
https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6181599981.
Existing local demo remains running. No real bank activation is authorized.

Executed baseline SHA256:
`3b1962af44c61bf1b26f09ab0f19e3e895c5885b0beaebc0eb037f649903e25d`.

The hash above identifies the executed baseline verification, not the new forward
candidate. Compute the forward hash from `deploy/dev-release.json` after review.

Hosted bootstrap: https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6182206357.
Forward compatibility: https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6182206763.

Hosted result: https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6182078948.
