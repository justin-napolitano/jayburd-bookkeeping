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

## Post-merge CI correction

The merged PR #165 sync-state CI failed because its legacy privilege-count
assertion still expected ten worker functions. Migration 18 grants the worker
one additional lease-scoped encrypted-credential reader. Issue #166 updates the
assertion to eleven and explicitly checks that reader; the existing denied
table and API-function assertions remain. Re-run the synthetic sync proof and
full CI before publishing images.

## Development rollout

PR #167 merged the CI correction. Its synthetic sync proof, Go checks and all
hosted PR checks passed. The owner-operated workers Terraform root enabled
Cloud KMS and created the `jayburd-dev/bank-credentials` key in `us-central1`.
After API propagation, key IAM readback confirmed only the development API
service account has encrypt/decrypt and the sync service account has decrypt.
No account token, plaintext key, Terraform state or plan entered this evidence.
Build 38843645129 passed complete validation and published all four candidate
image receipts from merged commit 74823ef. The migration-only manifest pins
its matching API/migrator/web digests for forward migration 18. Next: run the
hash-approved migration workflow and check its retained receipt, then promote
matching API and web images. The new worker image needs its own paused release
and binding review before acquisition. Hosted Sandbox enrollment remains the
acceptance gate.

## Matching application promotion

This second, dependent release restores the reviewed Sandbox connection policy
and public browser identity metadata while pinning API, migrator and web to the
same successful build 38843645129. Dispatch only after migration-only release
reports `migration-execution-passed` at schema 18. The runtime workflow checks
the current schema before changing Cloud Run services. The separate paused
worker image, account binding and actual hosted Sandbox completion remain to
be verified after application promotion.

## First paused worker candidate

The owner reported binding `b095afc3-d9e4-4273-9c2f-bc7c766adce3` from the
scoped read-only lookup. The same successful build 38843645129 contains an
encrypted-reader worker candidate, pinned by digest in
`deploy/worker-release.json`. A [metadata-only preflight](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/164#issuecomment-6219312071)
read back paused Scheduler, an existing queue, enabled static credential version
metadata, narrow KMS key IAM, and an absent worker service pair. The dedicated
worker deployment identity provider was disabled at preflight. This established
the initial release boundary; it is superseded by the paused deployment below.
No provider credential, Terraform state/plan, or financial row was read.

## Paused worker deployment

PR #172 merged the pinned candidate to main after all five checks passed. On
2026-10-16 the owner authorized enabling the isolated worker WIF pool and OIDC
provider directly in GCP. Readback confirmed the provider still restricts the
numeric owner, actor and repository IDs, `workflow_dispatch`, main and the exact
`deploy-workers.yml` workflow; its service-account impersonation binding is
scoped to that repository ID. The private worker runtime state bucket exists
with versioning. The direct enablement creates Terraform drift: preserve the
already-enabled build and application deployment flags, and set
`enable_github_worker_deploy_identity=true` in the owner-operated foundation
inputs before the next foundation apply. No foundation state or plan was read.
The checked-in `deployed-dev-identities.tfvars` overlay keeps all three GitHub
identity flags true when passed after private `owner.tfvars`. Foundation input
validation rejects a plan for this deployed project if any is false, including
when the overlay is omitted. Owner review and apply against private state remain
required to reconcile the direct GCP enablement.

[Worker release run 38946843411](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38946843411)
passed exact manifest/build evidence, infrastructure tests, workload identity
authentication, isolated runtime apply and receipt validation. The public
receipt reports `paused-workers-observed`, publisher revision
`jayburd-dev-acquisition-publisher-00001-rs4`, sync revision
`jayburd-dev-acquisition-sync-00001-ktn`, and Scheduler `PAUSED`; independent
Cloud Run and Scheduler metadata readback agrees. This is control-plane evidence
only. It does not prove hosted ciphertext persistence, authorized worker read,
provider acquisition or ledger acceptance. Do not activate Scheduler from this
release alone.

The old Hosted Link attempt remains a one-shot recovery case. Its binding ID
does not demonstrate encrypted credential persistence or a completed Item.
Issue #160 remains open until the owner uses the updated UI to classify the
attempt and demonstrates a fresh Sandbox completion through binding review.
Issue #164 remains open through hosted API ciphertext and worker read acceptance.
