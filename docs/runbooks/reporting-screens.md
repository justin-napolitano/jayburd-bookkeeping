<!-- stage-runbook: true -->
# Financial reporting screens

## Status and boundary

First authenticated reporting screen increment toward #52/#53. Four reports use
[the reviewed HTTP contract](../../api/openapi.json); SQL owns all
calculations. No CSV export, comparison controls, source-document drillthrough,
reconciliation, policy approval or consolidated currencies.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Design | Scope-owned report panel preserves journals/setup. FW01/FW02 semantic table/focus guidance and existing exact decimal formatter reused. Root approved bounded first screen slice. |
| Implementation | Generated HTTP types, authenticated GET bridge, TB/GL/BS/P&L tables, explicit dates/account, exact signed values, snapshot pagination and refresh. |
| Review corrections | Invalidate on scope/revision/command lock/client changes; reject mismatched wire version/layout; preserve report-specific 2MiB response bound and existing 1MiB other routes. |
| Validation | make check test build web-check web-test web-build passed; 130 web unit tests. Browser suite passes all eight Chromium journeys. |
| Final schema | Regenerated types against HTTP commit d45e9d6 after its OpenAPI ordering normalization; web-check passes. |
| Visual review | Coordinator inspected the synthetic desktop GL screenshot: readable controls/tables, original EUR separate from functional USD, no clipping. This is not accessibility conformance or live-data approval. |

## Current evidence

`Reports.test.tsx` covers signed amounts beyond JS safe integers, exact page
replacement/totals, stale409 refresh without old cursor, denied/error hiding,
filter/unmount races, direct scope/client/lock changes, scope and wire mismatches.
`reports.spec.tsx` exercises all four reports with independent fixed values,
original EUR units, stale pagination and late-response scope/logout boundaries.
`bridge.test.ts` covers route/query allowlisting, no-store, redacted error status,
and the separate report response size bound. Cursor is at most 2,048 characters;
whole URI remains bounded to 4,096. No browser persistence or financial arithmetic.

Existing `tests/integration/app.spec.tsx` now reads all four real reports after its
single committed 125.50 synthetic journal, preserving existing final database
counts. Root must run `make app-integration` on the combined HTTP/UI branch;
component mocks do not establish backend correctness. Generated API types were
validated against the HTTP agent's reviewed OpenAPI; that file is owned there.

## Remaining work

Comparison controls, exports, statement imports, report-to-journal navigation and
full workspace redesign remain separate increments; this does not close #52/#53.
Mobile/keyboard/screen-reader review and full application integration remain
explicit gates. Screens inherit the SQL foundation's materialization/policy bounds.

## PR handoff

No migrations, live providers, cloud or secrets. Dedicated lease
SCREEN-20261013T155400Z; root integrates and records final cross-boundary evidence.
Required staged runbook guard passes before commit. Screenshot is a temporary
synthetic review artifact, not a tracked product fixture.
