# Accounting proof hardening

<!-- stage-runbook: true -->

## Status and boundary

Issue [#67](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/67),
following merged audit #66 at `92044b0`. Strengthen synthetic evidence for the
existing [posting contract](../contracts/posting-boundary.md), exact historical
reversal and restricted writes. No accounting-policy, SQL migration, runtime,
API or cloud changes. General FX lifecycle and non-USD functional books remain
outside this proof.

## Stage ledger

| Stage | Result | Next action |
| --- | --- | --- |
| Claim | #65 ACCT lease; isolated branch `issue/67-accounting-proof-hardening` | Preserve exact file scope |
| Implement tests | Historical spot reversal, reversal-of-reversal denial, exact SQLSTATE and complete-row comparisons | Review expected values |
| Validate | Default gates and full isolated PostgreSQL ledger suite passed | PR review and hosted checks |

## Current evidence

- `TestSpotReversalHistoricalProvenance`: independently specified EUR 100 / USD
  110 original, later eligible 1.25 rate proved by a separate USD 125 posting;
  reversal must retain USD 110 and original currency/rate IDs, account order,
  opposite sides, policy, scope and linkage. Original header/lines stay unchanged;
  replay leaves all captured durable rows unchanged.
- `TestReversalOfReversalDenied`: expected `already_reversed` semantic error,
  unchanged journal/line/audit/receipt/outbox/book/reference rows and subsequent
  authorized read. This does not test a provider or physical principal deletion.
- `TestSchemaPrivileges`: seed real history before mutation probes, require
  SQLSTATE 42501 and unchanged rows, then prove an authorized posting/read still
  works despite the temporary-table/search-path probe.
- `TestJournalCorrectionHistory`: immutable-row trigger must return P2108;
  compare complete durable rows after each denied update/delete, then read the
  trial balance successfully.

PASS on 2026-10-13 with Go 1.26.9: `make check test build` (vet, race tests,
context/doc links, migration integrity and binaries) and `make ledger-test`
(pinned PostgreSQL 17.11, all integration tests including the four above,
forward-upgrade sentinel, restore/replay and runtime-readiness denials).
The initial Docker credential-helper failure inside the sandbox was resolved by
an approved outside-sandbox rerun; no credential values were inspected. The
integration executable uses the existing static internal-network harness;
the ordinary Go tests use `-race`. No hosted identity or scale claim follows.

Source authority is the existing [SQL dictionary](../contracts/ledger-sql.md),
[development policy](../decisions/0005-development-accounting-policy.md) and
[audit findings](core-acceptance-audit.md). No new normative interpretation or
claim of accounting/ISO conformance.

## Remaining work

Hosted PR checks and owner review remain pending. Cloud acceptance stays #26/#27/#12; real policy,
import provenance and close/reconciliation remain their existing issue gates.

## PR handoff

Only integration tests and this runbook change. No migration or application
rollback is needed. Review expected values and row comparisons independently of
the posting implementation. Keep #1–4/#17 open; #67 acceptance is this bounded
test improvement, not broad bookkeeping readiness.
