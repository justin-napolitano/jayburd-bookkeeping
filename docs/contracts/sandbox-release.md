# Limited hosted Sandbox release

## Decision and scope

For #132, add explicit `sandbox-promote` and `sandbox-rollback` operations to the
existing application and paused-worker release tools. Owner requested the integrated
fictional account journey on2026-10-15. This contract requires owner review/merge
before dispatch; it does not authorize production or complete #11/#26.

Only project `jayburd-bookkeeping-dev`, number `9088794002`, is eligible. Existing
provider composition fixes Plaid to Sandbox; manifests cannot select a provider URL
or environment. Exact image/build, expiry, owner/main, WIF, source ancestry and
Terraform ownership remain unchanged. This is internal release metadata, not an
ISO financial message; existing [standards](standards.md) profiles remain applicable.

## Prerequisites and phases

- App shell: #5/#6/#130 closed on evidence. #130 includes hosted database replay,
  status and actual API IAM denials. Connections may be explicitly null.
- Connection qualification UI and paused workers: additionally #131 implementation
  reviewed/merged. Source writer identity, scoped delivery identity, exact static
  GSM versions and consent expiry are reviewed in the manifest's evidence comments.
- Hosted dynamic credential activation and one manual acquisition remain #133.
  Existing SQL state, exact-version verification and explicit account bindings
  enforce activation; a deployed shell is not successful enrollment.
- Scheduler remains paused; automatic acquisition is outside this release.

The original full promote/rollback gates remain #6/#11/#26 (and #5 for workers).
They cannot be satisfied by closing implementation subtasks. #11 retains hosted
isolation/rotation/revocation/recovery; #26 retains restore/load/cost and operator
disposition. Declaring these complete to unblock deployment is prohibited.

## Execution and recovery

Sandbox app promotion executes read-only schema status using its exact build image
before runtime apply, never `up`. A schema mismatch blocks deployment. Sandbox
rollback skips SQL and requires a prior Sandbox receipt with the identical approved
connection policy. Full and Sandbox receipts cannot cross the rollback boundary.
Worker rollout retains the existing paused checks and fixed binding/version policy.
No API IAM bypass, test identity adapter, secret retrieval, SQL grant or new provider
endpoint is introduced. Runtime receipts explicitly identify Sandbox scope.

## Evidence and alternatives

Sources: existing [Cloud Run rollout](cloud-run-rollout.md),
[status](migration-status.md), [worker release](worker-release.md) and
[connection policy](connection-release.md) contracts and their reviewed GCP sources.
Inference: a separate operation permits qualification without falsely claiming the
broader readiness criteria. Rejected alternatives are silently removing closure
gates, closing incomplete epics, or deploying a local fake-identity demo.

Named `sandbox-release.test.mjs` tests cover profile/project restrictions, unchanged
full gates, missing/expired connection policy, status-before-runtime, scoped rollback
and receipt labels. Existing release/fake-command tests continue to apply. Hosted
UI/MFA/Link/activation/acquisition evidence remains separate; no readiness claim
follows from synthetic tests.

Selected official sections rechecked2026-10-15:
[Cloud Run Execute jobs](https://docs.cloud.google.com/run/docs/execute/jobs),
“What is job execution?”, required roles and gcloud execution: job configuration
and completed execution are distinct; `--wait` waits for completion. This supports
our execution receipt, not SQL correctness by itself.
[Plaid Sandbox overview](https://plaid.com/docs/sandbox/), “Differences between
Sandbox and Production”: Sandbox has institution/data-consistency limitations.
Therefore this milestone proves a fictional integration, not bank reconciliation
or production compatibility. Review depth: selected sections only; no account data.
