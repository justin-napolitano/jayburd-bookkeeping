<!-- stage-runbook: true -->
# Encrypted bank credential delivery

## Status and boundary

Issues #162 (SQL contract), #163 (crypto/runtime), #164 (IAM/rollout), under #160.
Owner explicitly selected encrypted DB persistence instead of per-account vault
slots. Static app secrets remain managed secrets. Contract:
[encrypted bank credentials](../contracts/bank-token-vault.md).

## Stage ledger

Reviewed current writer/readers and official Plaid/KMS selected sections. Created
issues before implementation. Existing #161 remains independent recovery UI work.
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

Finish immutable build-compatibility gates, then owner-reviewed KMS provisioning,
forward migration 18, matching API/consumer rollout and hosted owner acceptance
under #164. No old-image rollback after encrypted credentials are active. The
lost-token attempt requires affirmative abandonment only after confirming absent
storage; no claim reset, forced credential reconstruction or silent cleanup.

## PR handoff

Target main. Additive migration 18 must precede new application and worker images.
This runbook records checks and remaining limits at every implementation commit.
