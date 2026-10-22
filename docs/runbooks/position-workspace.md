<!-- stage-runbook: true -->
# Position workspace delivery — #249

## Status and boundary

Implementation on `issue/249-focused-position`, stacked on open #253/#248.
Owner approved the all-books zebra-row reference and requested build. This is
frontend/report-consumer work; no API, SQL, migrations, live financial records,
provider calls, cloud mutation or deployment. Existing synthetic reporting policy
is not real-book accounting approval. [Design](../plans/position-workspace.md),
[read model](../contracts/position-read-model.md), and issue #247 define scope.

## Stage ledger

| Stage | Evidence | Status |
| --- | --- | --- |
| Live authority | Rechecked #249/#248/#253 and #65 leases; retained an isolated worktree and open dependency state. | Complete |
| Contract update | Integrated #248 all-books read-model update `9270869` as `56e7556`; per-book snapshots, max four reads, no mixed-currency sum. | Complete |
| Position | Server comparison totals, exact display, group rows, scope/date invalidation, independent errors and snapshot metadata. | Complete |
| Shell | Direct Position/Ledger/Reports, scoped tool handoff, existing tasks and recovery safety retained. | Complete |
| Responsive review | Approved reference compared with desktop/table captures; mobile labeled rows remove sideways scrolling. [QA](../../design-qa.md) passed. | Complete |
| Verification | Core checks/race/build, 296 web unit cases, 22 browser journeys and real browser/Next/Go/SQL posting/Position proof passed, with environment qualifications below. | Complete |

## Current evidence

- Locked `npm ci` succeeded, zero reported advisories. Initial web API/type checks
  and 293 unit cases passed; Position focused browser cases passed.
- First component browser pass: 20/21 passed. A legacy GL fixture intercepted
  the newly added Position read; narrowed the fixture to its owned endpoint.
- Real browser → production Next bridge → Go → disposable PostgreSQL proof
  passed, including explicit setup and recovery of one committed journal after
  lost delivery; SQL assertions verified exactly one posting and explicit command
  acknowledgement. Later UI refinements require the final rerun below.
- Default demo/integration setup failed pulling Docker Hub (EOF). Both exact
  pinned PostgreSQL/migrate image digests were already cached and inspected. A
  temporary copy of the proof script substitutes only those two pulls with exact
  image inspections; product scripts/images remain unchanged.
- Local pinned Go 1.26.10 download is unavailable. Supplemental core/integration
  uses installed Go 1.26.5 with a temporary modfile/cache outside the repository;
  tracked manifests remain unchanged. Hosted pinned-toolchain CI remains required.
- Final `make web-check web-test web-build web-browser` passed: checked OpenAPI
  output, TypeScript, 296 unit cases, production Next build and 22 browser cases.
  After the final typography adjustment, the 15 affected Position/book browser
  cases passed again; source/desktop/table/mobile captures were compared together.
- Final temporary-script integration passed with explicit Position `0.00` before
  posting and `125.50` after one recovered journal. No API/SQL contract changed.
- The separate synthetic connection/enrollment/acquisition/review/post/recovery
  integration also passed after the tool navigation change; no provider was called.
- `make check test build` passed under the documented temporary Go1.26.5 modfile,
  including vet, race tests, migration integrity and executable builds.
- Static review found two navigation issues (reused consumed snapshot and retained
  connection-specific Review preference); both were fixed and regression-tested.
  A max-four scheduling race during date replacement was fixed and its test passes.

## Remaining work

The separate synthetic demo serves the real production components against a
disposable SQL/API fixture. It is local only and does not connect accounts.
Visual fixture screenshots separately prove layout with fixed fictional values;
they do not prove accounting semantics or production identity.

Remaining gates: staged runbook check, hosted pinned-toolchain CI and draft PR review.
Migration impact: none. #248 remains open for its own review; #250's broader
Ledger work, #251 planning and #255 Investigate are not marked complete here.

## PR handoff

Draft implementation stacks on #253 while #248 is open. Review the
[screenshots](../plans/position-workspace.md), [QA report](../../design-qa.md), and
environment qualifications above. No migration or deployment impact; no real-book
readiness claim. Browser-only fixtures remain separate from app routes.
Branch ancestry is aligned with #253; its contract commits are not treated as merged.
