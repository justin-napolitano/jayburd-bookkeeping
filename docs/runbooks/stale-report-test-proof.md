<!-- stage-runbook: true -->
# Deterministic stale report rejection proof

## Status and boundary

#278, discovered during PR #277 validation under #88; coordinator wave lease
`PLAID-88-WAVE2-20261022T1930Z`. Base a2a277c. Tests only: no runtime, SQL,
financial contract, API, migration, cloud or provider changes.

## Stage ledger

- Reviewed report export/drill contracts, reporting delivery evidence and bibliography.
- CI39854143741 failed a raw `1300` substring assertion. The response body was not
  retained, so its exact triggering UUID cannot be reconstructed. Source inspection
  establishes that a valid random diagnostic request ID can contain that sentinel.
- Replaced the substring check with a closed five-field stale-problem validator:
  exact type/title/status/code, canonical generated UUID, problem media type and
  absent download header. Status409 and successful golden export values remain.
- Added deterministic evidence using the actual problem helper and a UUID containing
  `1300`; additional totals/rows/journal fields, CSV, duplicate keys and altered
  status/download/media metadata are rejected. No production error behavior changed.

## Current evidence

PASS: focused `TestStaleReportProblemDiagnosticID`; full `make ledger-test`,
including actual PostgreSQL `TestReportReviewHTTPPostgres`, the deterministic
regression, restore/replay and runtime readiness checks. `make check test build`
also passed, including race tests and all command builds.
Logs: `/private/tmp/jayburd-278-ledger.log`, `/private/tmp/jayburd-278-core.log`.
Initial sandbox runs could not access Docker or bind local test sockets; required
escalation was approved and the same commands rerun without source changes.
These environment-denied runs were not counted as passing product tests.

## Remaining work

The original CI body is unavailable; passing tests
must not be represented as retrospective proof of its specific request ID.

## PR handoff

Refs #278 and PR #277. Only report integration test and this runbook are in scope.
Existing RFC9457 transport/ISO applicability from the reporting contracts remains;
no financial-message profile or accounting policy changes. Coordinator integrates
and reviews final evidence before merge.
