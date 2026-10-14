<!-- stage-runbook: true -->
# Allowlisted delivery evidence

## Status and boundary

Issue #11. Adapter implemented for synthetic review; no hosted delivery job or real
credential transfer. [Contract](../contracts/scoped-delivery.md) is authoritative.
The user authorized resolving the native-sync permission blocker. Existing native
connection remains disabled for sync; no permissions were broadened in this branch.

## Stage ledger

- Reviewed upstream native GCP sync and official REST sources before implementation.
- Added fixed source mapping, exact-version validation, durable one-shot intent,
  CRC32C addVersion payload, numeric receipt validation and replay recovery.
- Added offline synthetic canary, concurrent/rejected/mismatched/unknown-ack tests.
- Recorded prior metadata-only bootstrap and Terraform adoption requirements below.

## Current evidence

PASS: focused providersecrets race tests and `make check test build`.
The first full run could not bind loopback ports inside the sandbox; rerunning
with local-listener permission passed. Seven new delivery test groups cover
canary/replay, failures, concurrency, allowlist, registry acknowledgement loss,
receipt tampering and dynamic-slot reuse (failure subcases run separately).
Tests inject all identity and HTTP dependencies; no credentials, external network,
Plaid requests or financial payloads. Receipt acknowledgement is not live acceptance.

Cloud bootstrap from the preceding authorized setup: two empty containers
`jayburd-dev-bank-sync-client-id` and `jayburd-dev-bank-sync-client-secret`, plus
`jayburdProviderSecretSync` with secrets.get, versions.add and versions.list.
Bindings on those containers only to
`jayburd-dev-98389fdb-6ac3@jayburd-bookkeeping-dev.iam.gserviceaccount.com`.
Metadata listings found zero versions. No state or secret payload retrieved.
[Full receipt and import IDs](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/11#issuecomment-6173954332).

These gcloud-created resources are not yet Terraform-managed. Before workers apply,
import both `google_secret_manager_secret.provider` instances, the
`google_project_iam_custom_role.secret_sync` role and both
`google_secret_manager_secret_iam_member.infisical_sync` instances into the correct
workers backend. Existing labels express intended ownership only. The external
connection SA is supplied as a variable and is not owned by that root.

## Remaining work

- Controlled job composition and workload identity distinct from the existing native
  connection identity; exact-key Infisical read scope and isolated registry prefix.
- Terraform adoption, source IDs/versions and owner-injected synthetic canary.
- Hosted allow/deny, revocation, rotation, freshness and unknown-outcome recovery.
- Independent runtime payload verification and explicit numeric release pins.
- Unknown intent without receipt stays blocked; no automatic retry/reset tool exists.

## PR handoff

No SQL, API or frontend change. No deployment or hosted compatibility claim.
Focused and full regression gates passed. Hosted execution remains pending; #11 stays open.
