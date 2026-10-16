<!-- stage-runbook: true -->
# Encrypted bank credential delivery

## Status and boundary

Issues #162 (SQL contract), #163 (crypto/runtime), #164 (IAM/rollout), under #160.
Owner explicitly selected encrypted DB persistence instead of per-account vault
slots. Static app secrets remain managed secrets. Contract:
[encrypted bank credentials](../contracts/bank-token-vault.md).

## Stage ledger

Reviewed current writer/readers and official Plaid/KMS selected sections. Created
issues before implementation. PR #165 incorporates the recovery UI and regression tests originally in #161,
so one PR carries the encrypted-storage and recovery behavior.
No real token retrieval, exchange retry, attempt reset or legacy-secret deletion.

## Current evidence

PASS: migration fresh/replay/failure proof, connection SQL authorization and
abandonment tests, encrypted provider-flow recovery, Cloud SQL status and complete
ledger suites. PASS: Go vet/race/build before final packaging; final checks rerun
after preflight integration. PASS: web type/unit/build and 17 browser tests.
PASS: real application recovery plus real browser/Next/API/PostgreSQL enrollment,
encrypted persistence, scoped acquisition, review and exact posting. External
Plaid and KMS remain synthetic in these proofs. The provider-flow test injects a
lost committed storage acknowledgement and recovers without another exchange.
Terraform validation/mock tests pass across all nine roots, including narrow
API encrypt/decrypt and sync decrypt-only grants. No live infrastructure applied.

## Remaining work

Implemented promotion/rollback ancestry gates for #165 and closure of #162/#163.
Remaining: owner-reviewed KMS provisioning,
forward migration 18, matching API/consumer rollout and hosted owner acceptance
under #164. No old-image rollback after encrypted credentials are active. The
lost-token attempt requires affirmative abandonment only after confirming absent
storage; no claim reset, forced credential reconstruction or silent cleanup.

## PR handoff

Target main. Additive migration 18 must precede new application and worker images.
This runbook records checks and remaining limits at every implementation commit.


## Final packaging

PR #165 contains the encrypted implementation. Runtime application/worker gates
require this PR's merge to be ancestral to the chosen build, blocking incompatible
legacy rollbacks before cloud authentication. Migration/status gates are unchanged.
Final local Go gates, 266 web unit tests, 17 component-browser tests and the
production encrypted browser/API/SQL journey passed after preflight changes.
Release gate and deployment isolation tests are rerun for the ancestry change.

PASS: 26 release-manifest/worker/Sandbox gate tests plus application and worker
deployment-isolation/failure tests after the ancestry barrier. No provider or cloud
mutation performed by these synthetic checks.


## Consolidated recovery UI

Integrated the prior #161 changes: an Item-list failure cannot erase independent
attempt discovery, and completion 409 inspects the exact current attempt without
repeating exchange. Kept all recovery and abandonment tests. Revalidating the
combined browser flow before updating the PR; #161 is superseded after publication.


PASS after consolidation: 268 web unit tests and all 17 component-browser tests;
both real browser/API/PostgreSQL recovery and encrypted connection journeys;
final repository/context/migration/generated-contract/vet checks. Implementation
is ready for review; no live deployment, KMS apply or owner acceptance claimed.
