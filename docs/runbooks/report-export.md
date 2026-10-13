<!-- stage-runbook: true -->
# Report review export

## Status and boundary

Implemented bounded CSV service for #51/#53. No SQL migrations, route/UI ownership,
provider or cloud actions. Internal review only; accountant/audit exchange deferred.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source and wire review | RFC4180 §2 and OWASP CSV Injection reread; coordinator approved exact long measure schema before implementation. |
| Implementation | All selection pages, fixed 44-column wire, exact signed magnitudes and lineage, shared deadline, buffered limits, final authorization/revision check. |
| Independent review | Fixed optional UUID wrong-type handling and line-number upper bound; strict envelope validation rejects missing/null structure. |
| Fixture correction | Account immutability rejected hostile-label UPDATE; test now inserts a separate hostile-labelled synthetic account without changing existing accounts. |
| Final validation | PASS make check test build and complete make ledger-test; disposable PostgreSQL only. |

## Current evidence

[Contract](../contracts/report-export.md) owns the exact wire and source assessment.
Observed automated proofs:

- `TestReportCSVGoldenSQL`: actual SQL TB 1,300, P&L 70, BS 1,145; comparison
  measures and empty GL retains 1,045 opening/closing; hostile labels omitted;
  reporting leaves durable ledger state unchanged.
- `TestReportCSVAllPagesSQL`: 101 actual committed one-cent journals, complete
  509-record export, distinct journal IDs, exact 1.01 closing. Local combined
  posting/export test took 0.97 seconds; no hosted throughput claim.
- `TestReportCSVAuthoritySQL`: repeated and mid-export committed revocation denied;
  committed posting during export produces stale-snapshot error, no partial body.
- `TestReportCSVFXReversalSQL`: EUR 10/USD 11 original units, rate UUID and reversal
  linkage survive export; original/reversal closing zero.
- `TestReportCSVReviewExport`, `TestReportCSVPagesAndFailures`,
  `TestReportCSVEmptySelection`: independently fixed header/amount expectations,
  all pages, final authorization, metadata drift, duplicates, cancellation and
  all-or-nothing failures.
- `TestReportCSVSignGrammar`, `TestReportCSVPostingGrammar`: signed decimal,
  exact lineage types and calendar/enum validation, no float/spreadsheet formulas.
- `TestReportCSVRealLimits`, `TestReportCSVByteCap`: actual production 10,000
  physical-record and 10-MiB serialized-byte boundaries reject without partial data.

## Remaining work

HTTP/OpenAPI and browser downloads are integrated; combined validation passed
in [report review tools](report-review-tools.md). SQL materialization limits remain; no arbitrary functional
FX, large-ledger performance approval or accountant/audit exchange mapping.

## PR handoff

Lease EXPORT-REVIEW-20261013T170800Z; five export-owned new files only. No migrations
or financial arithmetic added. Required default gates and ledger gate passed with
Go 1.26.9/PostgreSQL 17.11; staged runbook guard required before commit.
