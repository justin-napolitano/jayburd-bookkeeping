<!-- stage-runbook: true -->
# Connection status and recovery UI

## Status and boundary

Issue #104 followup, lease STATUS-UI-104. Component/tests and new status contract
only; coordinator owns HTTP/OpenAPI and SQL author owns forward migration14.

## Stage ledger

Reviewed merged #107 UI and connection/provider state contracts. Sent required
recovery/status fields before implementation. Root froze single-stream operations,
independent stream revisions and authenticated paginated attempt discovery.
Implemented first-page discovery before Begin, paginated inspection, honest status,
explicit stream operation confirmation and exact same-key uncertain-result recovery.
Historical operation receipts are followed by a fresh status read; numeric generations
remain strings with exact BigInt consistency checks, never floating point.

## Current evidence

`make check test build` PASS. Web type check, all 231 unit tests across 16 files,
production build and 13 existing browser regressions PASS. Connections has 22 tests,
including 10 new cases for reload discovery, incomplete/malformed discovery, expired
unclaimed pagination, exact generations/unknown timestamps, uncertain manual request,
late scoped replies, pause/resume revision pins, stale rejection and revocation.

Coordinator-generated ConnectionAttemptList/ConnectionStreamStatus/
ConnectionStreamOperation aliases compile; generated file is borrowed locally and
not committed by this lane. Focused 22-test rerun after alias integration PASS.
Logs: `/private/tmp/jayburd-status-ui-default.log` and
`/private/tmp/jayburd-status-ui-browser.log`. New full browser/API/SQL status proof
belongs to coordinator; existing browser regressions are not that new evidence.

## Remaining work

Coordinator integration, new end-to-end status/discovery proof and peer review.
No real provider, secrets, cloud writes or concrete reconnect qualification.

## PR handoff

Coordinator integrates shared contracts and the full browser/API/SQL evidence.
