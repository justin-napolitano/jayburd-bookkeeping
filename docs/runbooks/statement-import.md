# Statement-import design evidence

<!-- stage-runbook: true -->

## Status and boundary

Contract-only proposal for [#61](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/61),
under [#8](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/8).
No importer, migration, provider call, live-data review or conformance claim.
Lease INGEST-61-20261013T134933Z; coordinator owns integration and shared bibliography.

## Stage ledger

| Stage | Evidence | Result |
| --- | --- | --- |
| Source review | [SI01–SI06](../research/statement-import-sources.md), source sections and access limits | Relevant public sections reviewed; official camt profile/XSD not verified |
| Contract proposal | [Format, binding, identity, drafts, bounds and fixtures](../contracts/statement-import.md) | Ready for owner review; internal-template exception remains proposed |
| Repository checks | Exact commands below | PASS: links, vet, race tests, migration-file checks and build |

## Current evidence

The three-row original worked example independently expects three distinct drafts,
USD100 inflow/USD25 outflow/net75; this is specification arithmetic, not a running
importer test. Every named importer test is future evidence.
Integration review distinguishes import audit/receipt writes from unchanged
accounting journals and `journal.posted` outbox intent.

PASS: `PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache make check test build`.
PASS: `bash scripts/check-stage-runbook.sh --staged` (three documentation files staged).

## Remaining work

Accept format/standards exception, permission, binding, source-ID and retention
choices. Implement only after separate SQL/API/storage and relevant #7 review.
[#17](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/17) stays open
for official ISO artifacts/profile; no source download here resolves it.

## PR handoff

Three documentation files only. Coordinator links source register in bibliography
and combines independent contracts in one PR. No GitHub mutation/push from this lane.
Next delivery is contract review, not real statement upload or automatic posting.
