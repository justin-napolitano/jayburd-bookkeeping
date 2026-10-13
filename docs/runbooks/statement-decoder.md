# Synthetic statement decoder delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #73, lease BUILD-73-20261013T152500Z. Pure Go decoder only; synthetic
fixtures, no persistence, provider calls, imports or posting authority.
[Exact contract](../contracts/statement-decoder.md) precedes implementation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Contract | Profile and SI01–SI06 source register reviewed; coordinator approved exact API before code. |
| Implementation | Pure bounded decoder with exact decimal strings; no dependencies or schema changes. Coordinator code review found no blocker. |
| Verification | `make check test build` PASS; race tests include decoder. Independent literal SHA-256 expectation added after review; focused race rerun PASS. |

## Current evidence

All local credential-free gates PASS on 2026-10-13. `TestStatementProfile` covers
original fields, independent identical purchases, quoted comma/quote and byte
provenance; `TestStatementMoney` covers 0/2/3-decimal exactness and limits;
`TestStatementSyntax`, `TestStatementFields`, `TestStatementExpectations`,
`TestStatementBounds` and `TestStatementIOFailure` cover malformed input,
account/date/metadata checks, bounded atomic failures and redacted reader errors.
SHA-256 fixture expectation was independently calculated with Python hashlib;
tests themselves are Go-only and offline. `make check` includes doc links,
migration file integrity, runbook structure, go vet and diff whitespace.
No SQL was executed and no provider was contacted.

## Remaining work

#8 binding authorization, durable
source identity, draft/recovery/storage and live-use gates remain unchanged.

## PR handoff

Coordinator owns shared index and PR. No schema or API changes. The local parser
does not authenticate bindings or prove metadata provenance; callers must supply
reviewed values and enforce Reader deadlines. Staged runbook guard PASS before
commit. Review scope: internal/ingestion and the two statement-decoder documents.
