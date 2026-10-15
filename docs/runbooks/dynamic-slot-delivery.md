<!-- stage-runbook: true -->
# Dynamic Sandbox credential delivery

## Status and boundary

Issue #131 implements explicit delivery for the eight fixed development slots.
No cloud/Infisical mutation or hosted token qualification occurred in this branch.
The [delivery contract](../contracts/delivery-job.md#fixed-dynamic-slot-extension-131)
owns the command/IAM boundary. #11 remains open.

## Stage ledger

- Reviewed #131, provider-vault/scoped-delivery contracts and existing source review.
- Added fixed slot command selection and value-disabled metadata lookup; retained
  immutable one-shot delivery, exact versions, fixed paths and separate identity.
- Added per-slot add-only Terraform mapping without owning destination containers.
- Added all-eight-slot command, metadata/mismatch, replay/uncertainty and IAM tests.

## Current evidence

PASS: `make check test build`, including full Go race tests. The first sandboxed
run could not bind loopback test listeners; rerun with local-listener permission
passed. PASS: 24 mocked delivery Terraform runs from a clean synthetic copy,
using the pinned Google 8.6.0 provider mirror and no backend.
No default tests contact Infisical, Google or Plaid; transports are synthetic.

## Remaining work

Owner setup requires public metadata only:

- Separate Infisical identity, suggested name `jayburd-dev-api-writer`, using GCP
  **ID Token Auth**, allowed account
  `jayburd-dev-api@jayburd-bookkeeping-dev.iam.gserviceaccount.com`, TTL/max TTL 900s,
  and no Universal Auth. Supply identity UUID and approved project UUID.
- Create `/bank-sync` in `dev`, leave slots initially absent, and grant writer
  create/read only. Existing delivery identity receives read-only access to this
  selected namespace; retain its separate GCP account. Free-plan role limitations
  are an owner/configuration blocker, not permission to add Admin.
- Provision worker-owned slot containers and approved API registry access, then
  configure connections with the dynamic project/writer metadata and fixed static
  GSM version pins. This branch does not create an API writer identity.

After an authorized fictional Hosted Link exchange creates its source slot, choose
that exact key/version in lookup mode; retain metadata receipt. Review its source
ID, switch execute mode with that ID/version and invoke once. Return to lookup to
remove write authority. The API recovery path must independently verify exact GSM
contents and commit SQL credential activation; delivery acknowledgement alone does
not prove that. No blind retry, slot recycling or second token exchange.

## PR handoff

No SQL migration or API/browser contract change. Hosted acceptance belongs to #133 and remains BLOCKED
on writer permissions/metadata, worker-owned containers and approved deployment.
Implementation completion of #131 does not close #11 or establish hosted readiness.
Rotation/revocation/freshness and broader #11 qualification remain separate work.
