<!-- stage-runbook: true -->
# Hosted Sandbox smoke test

## Status and boundary

Issues #11 and #88. Owner requested actual Sandbox execution and the missing
credential/runner wiring. Implemented a disposable API-created Item probe; hosted
execution remains pending image publication and credential access. This does not
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
worker Dockerfile build/publication remains the main-branch image workflow gate. Credentials have not been retrieved by agents and no actual Plaid API
call has been made by this change.

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
