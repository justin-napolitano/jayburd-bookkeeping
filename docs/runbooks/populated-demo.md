# Populated demo and source review

<!-- stage-runbook: true -->

## Status and boundary

Build #96–#98 from 116d84c, merged import work on still-open #83. User requests
one-click generated transactions, not CSV preparation. No live/provider/cloud work.

## Stage ledger

- Agreed SQL/API contracts before migration11: scoped review pages, same-currency
  category drafts and atomic source_post with exact immutable posting-body equality.
- Three exclusive agent worktrees and leases recorded in #65; root owns shared
  API/recovery/composition/CI. Migrations1–10 remain immutable.
- Demo acquisition advances bounded durable stages per click-driven request;
  refresh recovers SQL state. Loading source evidence creates no journals.

## Current evidence

- PASS `make check test build`; generated API types match `npm run api:check`.
- PASS `make migration-test ledger-test cloud-sql-test observer-test
  source-registry-test import-service-test`: disposable PostgreSQL, migration11,
  current permissions, scope, stale revisions, rollback, replay and restricted roles.
- PASS `make web-test web-build`: 181 unit tests. PASS `make web-browser`:
  12 component browser journeys. PASS `make app-integration`: production Next
  bridge to Go/PostgreSQL with synthetic identity.
- PASS `make demo-test`: four browser journeys in 3.6 minutes. Acquisition imports
  3,000 observations across three LLCs, then explicitly posts one linked journal.
  A lost successful execution response survives reload and acknowledgement;
  independently expected cents match the balance and linked journal. The existing
  volume journey commits 1,000 deposits and verifies exact ledger totals.
- `TestDemoAcquisitionPostgres` passes against restricted runtime roles in 1.04s:
  3,000 imports, independent counts/net amounts, zero journals from import/draft,
  reconstruction after lost response and unrelated pending-command protection.
- Browser testing found a missing synthetic entity `prepare` permission. Commit
  848c83f fixes the owner-only fixture and adds a real-role draft/replay regression;
  product authorization remains unchanged. Earlier selector and live-reload test
  interference failures were corrected before the final complete passing run.
- Visually reviewed the final posted-item screenshot, including original source
  amount and immutable journal lines. Agent SQL, acquisition and UI commits are
  integrated; independent cross-lane reviews found no remaining blockers.

Proof used isolated loopback ports 39581–39583 to preserve other running demos;
temporary port changes were restored before commit. Synthetic fixtures only.

## Remaining work

First category draft supports identity valuation only; foreign source amounts are
preserved and require an explicit future FX draft workflow, never guessed rates.
Provider acquisition, splits/transfer matching, reconciliation, hosted storage and
measured scaling remain separate gates. This is not live bookkeeping approval.

## PR handoff

Ready for review of #96–#98. Stack on still-open #83
(`build/product-workspace-batch`); merge #83 first, then retarget to main.
Keep #8/#9 and quality gates open until their full acceptance is met.
