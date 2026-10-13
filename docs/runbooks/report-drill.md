<!-- stage-runbook: true -->
# Report journal drill

## Status and boundary

Bounded #53 SQL/service delivery; coordinator owns HTTP/OpenAPI integration.
Lease DRILL-REVIEW-20261013T170800Z. No cloud/provider/live data or policy approval.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source and design review | Read existing report/journal contracts and PostgreSQL 17 snapshot/security sections; coordinator approved strict pinned-line contract before DDL. |
| Implementation | Restricted migration 7 wrapper, exact line/account/range/revision validation, existing authorized journal rendering and Go commit acknowledgement boundary. |
| Initial validation | make check test build passed; initial drill PostgreSQL tests passed, full gate required root-owned version 7 fixture correction. |
| Final validation | PASS: make check test build migration-test ledger-test with version 7 fixtures; actual PostgreSQL reversal and revocation-race proofs passed. |

## Current evidence

Contract: [report-drill](../contracts/report-drill.md). Actual PostgreSQL proof covers
GL inclusion and unchanged durable state, stale revision, wrong entity/book/account/line,
malformed and duplicate JSON, absent PUBLIC/private-helper grants and revoked read.
`TestReportJournalReversal` passed with EUR 10 / USD 11 original-rate provenance;
`TestReportJournalRevocationRace` passed actual lock-ordered revocation.
`TestReportJournalAdapter` passed repeatable-read/retry/lost-commit checks.

## Remaining work

Coordinator integrates HTTP/UI and browser evidence.
No historical report storage or category/subtotal drill support in this slice.

## PR handoff

Required: make check test build migration-test ledger-test and staged runbook guard.
Root integrates separately owned migration-version fixture updates and HTTP/UI.
