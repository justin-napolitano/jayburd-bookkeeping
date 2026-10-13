# Reporting, ingestion and worker foundation delivery

<!-- stage-runbook: true -->

## Status and boundary

Owner directed the next build after merged #71 (`e6a41f1`). Three bounded slices:
#72 reporting SQL/Go service, #73 pure synthetic CSV decoding, #74 physical worker
SQL contract. Parent #53/#8/#7 acceptance remains open. No public routes, UI changes,
cloud mutation, provider calls or live data in this increment.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Dispatch | Fresh issues/native parents and isolated worktrees; leases in #65 | Review physical/parser contracts before implementation |
| Integration preparation | Reporting alone owns migration 000006/checksums; root reserves two current-version test files | Integrate stopped-agent handoffs |
| Contract review | #72 SQL request/response reviewed before DDL; #73 bounded decoder API reviewed before implementation | Test exact arithmetic, scope, malformed inputs and commit boundaries |
| Parser integration | Agent `b9c6c61` integrated; independent digest and exact-value/negative fixtures reviewed | Run combined checks |
| Worker design integration | Agent `6ccd9f7` integrated; root reviewed typed functions, privilege boundary and recovery cases | Keep proposed worker DDL separate from reporting migration |
| Reporting review | Independent review found scope-free cursor hash; root required bounded materialization and nullable currency metadata rejection | Require negative fixtures and actual posted-volume pagination evidence |
| Reporting integration | Agent `b860beb` integrated as `d5dda82`; cursor binds scope/version and negative fixtures cover identified defects | Review PR |
| Combined validation | `make check test build migration-test ledger-test cloud-sql-test` PASS on 2026-10-13; disposable synthetic PostgreSQL only | Review and merge |

## Current evidence

Existing source review and contracts remain the accounting and standards authority.
Root identified version-five fixture assumptions in the ledger/schema-status tests;
the additive reporting migration requires version-six expectations and preserved
older/newer rejection checks. No merged SQL is modified.

`TestReportThousandPostings` committed 1,024 one-cent journals and checked nine
GL pages without missing/duplicate rows, exact running balances and closing 10.24.
Measured posting/report work: 10.145 seconds locally, not a hosted SLA.
Golden-month, reversal provenance, revocation race, cursor-scope collision,
aggregate overflow and materialization/policy bounds passed. Restricted migration
status, contention, dirty failure and artifact-integrity tests passed.
The first sandbox Docker attempt failed at its credential helper; rerunning the
disposable database gates outside the sandbox succeeded. No real credential was read.

Parser tests cannot prove durable import,
cross-upload deduplication, binding authorization or posted bookkeeping effects.
Reporting service tests cannot establish unimplemented HTTP/browser behavior.
The worker physical design is not executable worker delivery.

## Remaining work

Next: review/merge, then resolve parent gates before HTTP/UI reporting, durable
import or worker implementation. Reports currently materialize at most 4,096
selected GL rows/accounts and 1,024 policy revisions, with a 2 MiB response cap.
Large-history SQL optimization remains work before raising these limits.

## PR handoff

Integrated and validated; PR review pending. Forward reporting migration requires compatible
runtime artifact/schema expectations; do not apply or downgrade a live database.
No automatic closure of the broader parent issues.
