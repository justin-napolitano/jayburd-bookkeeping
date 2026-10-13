# Parallel financial application contract review

<!-- stage-runbook: true -->

## Status and boundary

Coordinate reporting #51, workspace design/prototype #52 and statement-import
contract #61 from merged main `0f732bae04579f5495a54e01a4bd3a20a0cad248`.
Each agent has an isolated worktree and disjoint file scope under
[lease register #65](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/65).
The coordinator alone owns shared bibliography/index updates and integration.

The deliverable is a reviewed proposal package and static synthetic UI prototype.
No product endpoints, accounting schema, cloud resources, provider ingestion or
live financial data are authorized by this stage. Merged test proof #68 does not
complete cloud acceptance or add general functional-currency support.

## Stage ledger

| Stage | Evidence / result | Next action |
| --- | --- | --- |
| Reconcile | #68 merged; #51/#52/#61 open, no native blockers/competing claims or open PRs | Grant three bounded leases |
| Dispatch | REPORTS, UI and INGEST completed independently; exact grants linked in #65 and task comments | Writers stopped at handoff |
| Integration | UI `c5cba75`, import `1361b22`, reporting `443146a` integrated; shared source/index links added | Review proposal package |
| Semantic review | Corrected revision-zero acceptance, earnings labels and import-audit distinction; CSV wire-schema gap explicit | Preserve implementation gates |

## Current evidence

Execution results belong to the individual living runbooks and this final
integration review. Named future tests in a contract are requirements, not passing
tests. Authoritative source sections and access limits must accompany new design
decisions; a proposed policy is not owner/accountant approval.

Coordinator review checklist:

- Entity/book/period/currency scope remains explicit; never aggregate unlike
  currencies or mislabel an LLC comparison as consolidated statements.
- Reports read posted journals; drafts/source observations do not inflate ledger
  totals. Comparison periods, openings, cumulative equity/earnings and reversals
  must agree with independent fictional examples.
- Current authorization applies before each report page/export/import action;
  revision checks cannot substitute for revocation checks. Stale UI scope clears.
- Reporting consistency and cursor expiry/refresh are explicit; no unsupported
  claim of replaying old mutable metadata snapshots.
- Import identity distinguishes repeated delivery from legitimate identical
  purchases; errors and retention rules preserve provenance without live data.
- UI fixtures trace to reviewed synthetic examples. Static navigation and demo
  states cannot be represented as implemented backend, reconciliation or close.
- Contract/source references stay discoverable through the shared index and
  bibliography; unsupported ISO mappings and owner decisions remain visible.

Reviewed all checklist items against the three proposals and current contracts.
The import format is a proposed synthetic internal CSV exception, not an accepted
ISO bank adapter. Reporting CSV safety/scope is specified but its ordered columns
and row-kind/nullability mapping remain a gate. The static UI deliberately shows
February as unavailable fixture evidence, while #51 specifies how a verified empty
February report would carry openings; those are different evidence states.

Each agent ran `make check test build` and its staged-runbook gate. Coordinator
reran FW-DOM from the UI runbook using the existing locked jsdom at
`/private/tmp/jayburd-bookkeeping/web/node_modules/jsdom`: PASS after integration.
It proves fixture equality and DOM state/navigation only. Browser setup returned
`Browser is not available: iab`; visual, native keyboard/mobile and owner review
remain pending. No frontend dependency, application route or auth adapter changed.

## Remaining work

Owner reviews the reporting proposal, prototype and synthetic import exception.
Before implementing exports, finish the reporting CSV wire schema. Before import
implementation, accept the scope/standards decision and SQL/API/storage contracts.
Dependent product implementation and runtime/browser/hosted tests remain subsequent
gates; named new report/import tests have not run because they are not implemented.

## PR handoff

Coordinator validation: `make check test build` PASS on the integrated branch
(Go vet/race tests/build, local links, migration file checks and runbook structure).
FW-DOM PASS; no product browser or database integration suite was required or
claimed for this proposal/static-prototype-only package.

Preserve #51/#52/#61 acceptance boundaries;
do not close #52 on a static prototype or call proposed endpoint tests executed.
No migrations or runtime rollback are needed for this design package.
