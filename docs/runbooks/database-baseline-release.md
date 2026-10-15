<!-- stage-runbook: true -->
# Hosted database verification

## Status and boundary

Forward migration execution passed after #128. The current candidate selects
`operation=verify` with the same immutable build38654097056 images to establish
clean version15 and exact registered migration hashes. Verification has not run.
It updates the existing job action to status; no SQL mutation or API/frontend
rollout. Contracts: [read-only status](../contracts/migration-status.md) and
[migration-only release](../contracts/migration-release.md).

## Stage ledger

- Baseline verification38663999729, execution jayburd-dev-migrate-sp7xd, passed
  at clean version5 with exact hashes using build38327676280.
- Owner restored Cloud Shell authorization. Merged role bootstrap, replay and
  catalog-only privilege readback passed through the retained operator. All seven
  capability roles are safe NOLOGIN roles; API has no DDL or privileged/worker
  membership; migrator has explicit owner SET without inheritance, ADMIN or
  CREATEROLE. Connection limits remain20/2. Pinned proxy2.26.0 stopped on exit.
- Bootstrap SHA256:
  `c19921224c635ad7105f5a46d1a6d6e66012a0699a1d3f3077280ea273cf15dd`.
- Forward run38743082827, execution jayburd-dev-migrate-g4pj4, completed successfully
  at 2026-10-15T13:31:59.567105Z. Retained receipt and independently validated
  allowlisted metadata match the exact image, identity, up arguments, one task,
  zero retries and 660-second timeout. This proves execution, not status or replay.
- Prepared verification candidate retaining build38654097056 attempt1 at
  `5c10e944678e15caf5f832fc87b5c95aa6f86456`, all three image digests, and bootstrap
  evidence. Updated only operation, compatibility evidence and expiry.

## Current evidence

Prior local `make migration-test` and `make cloud-sql-test check test build` passed,
including fresh install/replay and restricted roles. They do not replace hosted
acceptance. PASS: exact candidate/build validation and a fresh
`make check test build release-test`. The initial documentation check rejected a
renamed required heading; restoring `Current evidence` resolved it before rerun.
Reviewed candidate SHA256:
`b22eae58a60e3d2f0aa6304a19cab35bc9954f0841d275122ec9dd38e431a004`.

No SQL, application implementation or new infrastructure resources changed.
Historical build scans are retained evidence, not fresh vulnerability assessments.

## Remaining work

Owner reviews and merges the exact verify manifest, then dispatches deploy-dev.yml
on main with its SHA256. Successful status must prove clean version15 and all
embedded artifact hashes. Stop on a failed check; never force dirty state or
rerun up as an inspection shortcut. Replay requires separate reviewed execution.

Application promotion still requires actual API IAM denials, ownership/grantor
review, restore/load acceptance (#26) and outstanding secrets isolation/rotation/
revocation/recovery proof (#11), or an explicitly reviewed development scope change.
The retained operator has ownership dependencies; do not remove it blindly.
No frontend deployment or real-bank activation is authorized by this release.

## PR handoff

Keep #26/#11 open. Compute the new hash from `deploy/dev-release.json`; the previous
migration dispatch hash must not be reused. No credential transfer, raw logs,
provider payloads or business rows entered this evidence.

- [Baseline result](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6182078948).
- [Hosted bootstrap](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6182206357).
- [Forward compatibility](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6182206763).
- [Forward execution and status candidate](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6193592881).
