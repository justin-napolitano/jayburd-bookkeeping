<!-- stage-runbook: true -->
# Reporting HTTP delivery

## Status and boundary

Bounded JSON route delivery toward #53, lease REPORT-HTTP-20261013T155400Z.
No SQL migration, CSV, cloud or live data. Root owns integration; UI agent owns
web files and generated client. Contract reviewed before code by coordinator.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Review | Root approved wire; fixed BS layout, layout-bound cursor, malformed URL rejection and canonical no-padding base64. |
| Implementation | Four authenticated routes, bounded query/cursor/response mapping, typed OpenAPI. |
| Independent review | Root and posting-gate agent reviewed identity, scope, cursor strictness, response bounds and unchanged SQL authority. Fixed response version guard and added Unicode/size negatives. |
| Validation | PASS: make check test build; make ledger-test using disposable PostgreSQL. Final Unicode and response-bound refinements passed make check test build afterward. |

## Current evidence

PASS: TestReportingHTTPContract covers verified actor propagation, old routes,
strict query/cursor parsing, HEAD/body rejection, snapshot error redaction,
incompatible SQL envelopes, Unicode escapes and exact decimal preservation.
PASS: TestReportingHTTPResponseBounds rejects oversized SQL output and wire
expansion with 422 and no report rows.
PASS: TestReportingHTTPPostgres exercises all four actual reports through HTTP:
TB 1300, BS 1145, P&L 70, GL closing 1200; cursor continuation changes rows,
revision change returns 409 and revoked read permission returns 403.
The full ledger gate also passed existing SQL, immutable ledger and restore tests.
Synthetic identity adapter is test-only; this is not hosted authentication proof.

## Remaining work

Coordinator integrates frontend and runs combined application/browser gates. CSV,
full policy matrix, large-history optimization and hosted proof remain #51/#53.

## PR handoff

Required: make check test build ledger-test; OpenAPI generated client validation
and web-check web-test web-build web-browser plus app-integration in integrated
branch. No migration files changed. Update evidence before handoff.
