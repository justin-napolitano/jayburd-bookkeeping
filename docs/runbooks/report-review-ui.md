<!-- stage-runbook: true -->
# Report review UI

## Status and boundary

Implement comparisons, scoped navigation and internal review CSV download for #53.
SQL owns exact amounts and membership; the browser keeps responses in memory only.
Root reviewed comparison bounds and CSV/journal wire contracts before their
implementation. Existing financial-reporting and reporting-http contracts own semantics.

## Stage ledger

- 2026-10-13: reviewed report contracts and existing browser lifecycle. Added optional
  point comparison or paired P&L range controls, server amounts/differences and
  filter invalidation. GL rejects/hides comparisons. No browser arithmetic.
- Added revision-pinned account navigation with explicit date range, authorized
  journal detail, authenticated CSV download and bounded same-origin bridge.
  Download errors preserve fixed safe text; stale/denied access clears all report
  state. Scope/client/filter changes discard late actions and revoke download URLs.
- Generated web types from coordinator-owned OpenAPI. The copied API file is a
  validation input only and is not included in the UI commit.

## Current evidence

PASS: `make check test build`; `make web-check web-test web-build`; `make web-browser`.
148 web unit tests and nine Chromium journeys passed. Named `comparison review`
cases prove beyond-safe-integer amounts, negative one-cent differences,
comparison-preserving pagination and clearing on edit. `scoped report actions`
proves pinned navigation, CSV revision/filter retention, URL cleanup, late response
suppression and denied/stale clearing. Client/bridge cases reject wrong media,
oversized streaming bodies, absent revisions and page filters on exports.
Browser journey `comparison, account drill, pinned journal and download remain
scoped` proves those interactions using synthetic HTTP fixtures.

Extended `web/tests/integration/app.spec.tsx` to prove real comparison125.50/0,
pinned journal read, CSV download and account drill. This requires combined backend
changes; coordinator must run `make app-integration` before PR readiness.

## Remaining work

Combined real browser/API/SQL test remains pending integration. Category/derived
subtotal navigation is deferred; only account and GL journal paths are implemented.
Point account review uses an explicit <=366-day GL range and does not claim that
those period postings alone explain the full historical balance. No live data,
deployed claims, full FX or audit export.

## PR handoff

Coordinator owns integration, issue leases and PR. UI agent owns web changes and
this living evidence document. No SQL migrations in this workstream.
