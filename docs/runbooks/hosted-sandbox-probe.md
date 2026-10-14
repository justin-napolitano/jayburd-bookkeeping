<!-- stage-runbook: true -->
# Hosted Sandbox smoke test

## Status and boundary

Issues #11 and #88. Owner requested actual Sandbox execution and the missing
credential/runner wiring. Implemented a disposable API-created Item probe; hosted
execution completed with authentication/cleanup PASS and initial sync BLOCKED at the decoder. This does not
qualify interactive Link, SQL import, webhooks or live books.

## Stage ledger

- Reviewed [probe contract](../contracts/hosted-sandbox-probe.md) and official Plaid
  Sandbox/Items endpoints; existing exact-decoding diagnostic remains authoritative.
- Expanded delivery command to three fixed root keys; exact-ID/version delivery
  still uses durable intent and receipt fencing. Metadata lookup never reads values.
- Added managed-identity probe, fixed Sandbox API sequence, bounded cleanup and
  count-only diagnostic output. Initial not-ready reads get at most five attempts,
  two seconds apart; Item creation and exchange are never retried.
- Added optional Terraform probe job with separate identity, two read-only secret
  grants, exact version pins and no task retries. Delivery gets only its selected
  key write grant. Existing manual workflow selects one of two fixed job names.

## Current evidence

PASS: `make check test build`, focused Go tests, workflow lint and all infrastructure
mocks, including twelve delivery/probe runs. Eight invalid startup cases passed
in a network-disabled scratch image built from current Linux binaries. The full
worker Dockerfile build/publication remains the main-branch image workflow gate. Credentials have not been retrieved by agents. Actual hosted API evidence is recorded below.

## Remaining work

1. Owner selects source project access. Existing machine identity currently has
   access only to the isolated canary project. Original root credentials stay in
   `jayburd-bookkeeping` unless owner chooses separate Sandbox isolation. Viewer on
   the free plan is whole-project access; our job requests only the two fixed keys.
2. Merge and publish worker image containing `/sandbox-probe`; retain tested digest.
3. For each root key (`PLAID_CLIENT_ID`, `PLAID_SECRET`), configure `canary.key`,
   selected source project/identity and exact source version in lookup mode; inspect
   ID/version metadata, then switch to execute with that ID. Confirm numeric GCP
   version metadata. Do not retrieve values or retry ambiguous delivery.
4. Set `sandbox_probe` to reviewed image and the two GCP version numbers. Apply
   Terraform; dispatch Run development delivery canary with `job=sandbox-probe`.
5. Read only receipt checks/counts. Overall partial is expected: this probe cannot
   qualify application behavior. Cleanup failure or exchange ambiguity requires
   owner review in Plaid Sandbox Dashboard before another creation attempt.

## PR handoff

No database migration or accounting behavior change. Runtime secrets stay in
memory. Probe has no Infisical or database authority; delivery cannot read GCP
payloads. Real bank connections remain outside this scope.

## Hosted execution — 2026-10-14

Owner confirmed Viewer membership on the original source project and authorized
actual Sandbox execution. Build `38628435259` passed for main `a036982`; worker
image digest `95600d84701773f6aa6f5501206a3b24004eacbb49ac00245a03109e9c6ec7e7`
passed runtime/vulnerability gates and was deployed by Terraform.

- Client-ID metadata lookup `jayburd-dev-delivery-canary-w4q4v` passed: source ID
  `0ed3537c-3d54-4bf7-b401-7bad5ad38e9d`, version `1`.
- Client-ID delivery `jayburd-dev-delivery-canary-6x6bw` passed; destination version
  `1` enabled. Switching back to lookup removed its write grant.
- Secret metadata lookup `jayburd-dev-delivery-canary-xpwl4` passed: source ID
  `3f29c2eb-4f71-49d3-96a8-ff0da6564347`, version `1`.
- Secret delivery `jayburd-dev-delivery-canary-fdvwq` passed; destination version
  `1` enabled. Delivery switched back to lookup, removing write permission.
- Probe identity/job deployed with both exact destination version pins `1` and
  read-only access. Workflow `38632226850` dispatched the first actual Sandbox test;
  execution `jayburd-dev-sandbox-probe-k6cd9` completed with exit `2` (one failed
  task), as expected for a blocked diagnostic. No secret value or raw provider
  payload was retrieved into agent context.
- PASS: `sandbox_api_auth / fictional_item_exchanged`. Existing client-ID/secret
  pair was accepted by Plaid Sandbox; no key replacement indicated by this test.
- BLOCKED: `initial_sync / decoder_rejected`, zero decoded pages/changes. These
  counts mean decoding stopped, not an empty account. The generic reason does not
  identify the rejected field or invariant; root cause remains unproven.
- PASS: `sandbox_item_cleanup / removal_acknowledged`. The disposable Item was
  removed. Link, persisted source review, pending-to-posted and signed webhooks
  remain unqualified. No automatic probe rerun was performed.

Next work: expose only a fixed allowlisted decoder error classification in the
redacted receipt, reproduce it with a synthetic regression, and correct the
reviewed profile before re-running. Do not fetch or log raw provider payloads to
diagnose this failure. Required repository checks passed for this evidence update.
