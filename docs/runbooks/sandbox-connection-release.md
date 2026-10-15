<!-- stage-runbook: true -->
# First hosted Sandbox connection release

## Status and boundary

Issue #133. Prepare the application release after completed database gate #130.
Provider remains Sandbox, journals require explicit review, acquisition scheduling
remains paused. No application deployment or hosted writer test has occurred.

## Stage ledger

- API IAM probe, exact-image replay and post-replay clean15/hash verification passed;
  #130 closed with three independently validated hosted receipts.
- Owner supplied and confirmed separate Infisical writer/project IDs, exact GCP
  service account, ID Token Auth, 900-second TTLs and Development `/bank-sync`.
- Recorded the explicit isolated-project Member-role exception in the vault contract.
- Selected canonical build 38797942308, commit
  `025963e0cf152df365682e201ef3347d27dbf67e`; all four image receipts passed
  validation, runtime checks, vulnerability scans and immutable publication.
- PR #143 added the reviewed Sandbox starter-book command so a signed-in owner
  can prepare the required entity, book and four-account chart without manual
  accounting setup. The user must still review and submit the saved command.

## Current evidence

[Writer confirmation](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/133#issuecomment-6197028300)
is owner-attested configuration, not hosted access proof.
[Database acceptance](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/130#issuecomment-6196651622)
is complete. Static Sandbox client-id/client-secret numeric version1 pins retain
prior delivery scope; no credential values are read. Owner supplied the public Firebase browser identifier; the complete manifest
uses `sandbox-promote`, immutable build receipts, static version1 pins and the
Hosted Link profile. Release authorization expires 2026-10-18T17:00:00Z; connection
consent ends 2026-10-22T17:00:00Z and is never automatically extended.

PASS: build 38797942308 ran the complete seed suite, application integration,
frontend proof and all four image publication jobs from merged `main`. Exact
manifest/build validation passed. Manifest SHA256:
`f211c3143cf7a9e63f3ac9c4ed83f9eecfdfa1ccb7e7ba1e4e89fd3aa611007d`.

Read-only bucket-name listing confirms credential-registry and acquisition-evidence
are absent as of 2026-10-15. Existing delivery-registry is a different resource.
[Issue #141](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/141)
tracks reviewed bootstrap and metadata-only acceptance before connection testing.

## Remaining work

Review/merge the exact `sandbox-promote` manifest; resolve #141 infrastructure
preflight before dispatching this connection-enabled release. Runtime status must pass before service rollout. Confirm actual
writer and separate delivery access in #133; saved roles alone are insufficient.
Run the reviewed first-principal/book bootstrap with owner-supplied UID and real
sign-in/MFA before Hosted Link and manual acquisition. No automatic journal posting.

## PR handoff

The release replaces the prior status-only manifest with Sandbox application
promotion. It verifies database status before deploying API/frontend; no migrations
are applied. The Firebase identifier is public browser configuration. No broadened
GCP IAM, SQL change or runtime permission bypass is introduced. Before Hosted Link,
verify the worker bootstrap owns the credential registry, eight slot containers
and exact API grants; application rollout does not create these prerequisites.
