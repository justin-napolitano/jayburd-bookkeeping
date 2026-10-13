<!-- stage-runbook: true -->
# Reporting foundation

## Status and boundary

Issue #72 implements scoped SQL/Go report reads only. No HTTP route, generated
OpenAPI/browser/CSV/cloud release or live accounting-policy approval. Lease
BUILD-72-20261013T152500Z owns migration 6 and reporting-only contract/service/tests.
Root owns the two shared integration fixtures updated for migration version 6.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Contract review | Coordinator reviewed reporting-sql.md before DDL on 2026-10-13; exact amount fields, policy coverage and deferred HTTP cursor encoding recorded. |
| Implementation | New restricted SQL wrapper/private helpers; exact decimal aggregation, current authorization and scoped cursor revision guard; Go read transaction/commit boundary. |
| Review corrections | Corrected migration CASE syntax and currency field name; bound cursor hash to version/entity/book; added NULL/validity currency rejection and explicit materialization/policy caps. |
| Initial validation | make check test build, migration-test and full ledger-test passed, including restore/replay and runtime privilege checks. |
| Final validation | PASS: make check test build migration-test ledger-test after final policy-cap and canonical account-filter changes; no failures. |

## Current evidence

[SQL contract](../contracts/reporting-sql.md) owns the implemented request/response
shape; [financial reporting](../contracts/financial-reporting.md) retains the broader
product proposal. Actual PostgreSQL tests:

- `TestFinancialReportsGoldenMonth`: TB 1,300/1,300, P&L profit 70, BS 1,145,
  comparisons, GL opening 1,000/closing 1,200, February opening/closing 1,045,
  and complete durable-state equality before/after reporting.
- `TestReportBoundaryAndRevision`, `TestReportCursorScopeCollision`: raw duplicate
  and malformed JSON, foreign scope/account denial, unchanged revision zero,
  revoked membership, stale revision and equal-ID/equal-revision book isolation.
- `TestReportReversalProvenance`: EUR 10/USD 11 and original rate/reversal link
  retained; January original and February correction stay on their dates.
- `TestReportThousandPostings`: 1,024 actual committed one-cent journals, nine
  pages with no missing/duplicate lines, every running balance and final 10.24
  exact. Final local run took 10.23 seconds including posting and all pages;
  this is synthetic evidence, not a hosted throughput/SLA claim.
- `TestReportMaterializationBound`, `TestReportPolicyAndUnicode`: 4,097 account
  rows fail even with page limit 1; unknown currency scale and unsafe Unicode fail.
- `TestReportAggregateOverflow`, `TestReportRevocationRace`: one-cent aggregate
  overflow emits no partial result; a waiting reader cannot outrun committed revocation.
- `TestReportPolicyCountBound`: 1,025 historical policy rows reject before provenance
  array construction; final full gate passed.
- `TestFinancialReportAdapter`: repeatable-read, unchanged raw request, bounded
  retries, snapshot mapping and no result after uncertain commit.

## Remaining work

HTTP contracts/routes, CSV wire schema, browser integration, full comparison/FX
policy corpus and managed performance evidence remain #51/#53 gates. SQL uses
explicitly bounded complete materialization (4,096 selected GL rows or book accounts,
1,024 policy revisions), not streaming pagination. Larger selections fail rather
than truncate; measured query redesign precedes raising caps. No indexes added.

## PR handoff

Migration 1–5 remain unchanged; migration 6 is forward-only. Repository checks use
Go 1.26.9 and disposable PostgreSQL 17.11; tests need no live database/credentials.
Required gates: make check test build migration-test ledger-test, then the staged
runbook guard. Root will integrate separately owned migration-version fixture fixes.
