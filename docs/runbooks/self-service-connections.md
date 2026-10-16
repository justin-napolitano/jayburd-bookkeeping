<!-- stage-runbook: true -->
# Self-service connection delivery

## Status and boundary

Issue #155 removes the operator step between creating a book and opening Plaid.
[Contract](../contracts/self-service-connections.md) owns authority and consent.

## Stage ledger

Add bounded SQL discovery/Begin wrappers, enable them in managed Sandbox
composition and explain the existing single-button behavior in the UI. Keep
operator scripts for explicit administration and existing deployment gates.

## Current evidence

PASS: migration 17 installs and replays on disposable PostgreSQL. Web checks,
263 unit tests, production build and all 17 browser journeys pass. Exact cached
digest-pinned database images were used after registry refresh failed; no image
tag substitution. The local Go 1.26.9 archive matched the official SHA256.
PASS: `make check test build` includes vet/race tests and compiled binaries.
`make connection-service-test ledger-test` passes all new creator cases and the
full ledger suite. `scripts/test-app-integration.sh --connections` passes the real
browser/Next/Go/PostgreSQL journey with operator connection grants removed from
the test fixture. It covers enrollment, acquisition, exact posting and lost
response recovery. This is synthetic integration evidence, not hosted acceptance.
PASS: the ordinary `make app-integration` recovery journey also passes through
the production bridge, with exactly one posting after a lost committed response.
CI caught the separate cloud migration-status fixture still treating version 17
as invalid. Its older/newer/restore metadata cases now require migration 17;
the focused restricted Cloud SQL suite is rerun before release.

## Remaining work

Fresh creator-owned books reach Link through the same browser/API/SQL path.
Migration, denied/replay/race tests, Go/web and application integration checks
pass. Runtime retains no owner membership or direct table authority.

## PR handoff

Hosted release requires a forward migration and matching application promotion.
The publisher currently has one fixed binding; shared multi-book acquisition
is tracked in [#156](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/156)
with fairness, current-grant and deployment-policy gates. Real bank activation remains outside
this Sandbox implementation.

## Hosted forward migration

PR #157 merged as `eaa016a711308427f52fc8be5c90e025fcffcc32`.
[Build 38829829288](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38829829288)
passed all validation, frontend, real application integration and four image
publication jobs. The migration-only manifest selects its matching immutable
receipts and applies migration 17 before application promotion. Runtime services
remain on the previous release during this operation. Hosted authenticated Link
acceptance remains pending the owner's UI test.
