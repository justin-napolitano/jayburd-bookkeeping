<!-- stage-runbook: true -->
# Investigate contract stage ledger

## Status and boundary

Issue #255, branch `issue/255-investigate-contract`, dedicated worktree.
Design/status inventory only. No SQL migration, API, UI, provider call, real
financial data, cloud mutation or automatic posting.

## Stage ledger

| Stage | Evidence / next |
| --- | --- |
| Scope | Read live #255, #104 and #242. #104 and #242 remain open; no predecessor is treated as complete. Read AGENTS, index, bibliography, decision and standards requirements. |
| Inventory | Reviewed source review, command recovery, connection/stream status, reporting contracts and their route definitions. Recorded owned facts, permissions, freshness, actions and gaps in [proposal](../contracts/investigate-handoff.md). |
| Synthetic examples | Fixed expectations below are independent design examples, not API fixtures or runtime results. |
| Handoff | Owner/coordinator reviews task vocabulary and whether a future server-owned index is desired. Implementation requires its own scoped issue/evidence and current dependency check. |

## Current evidence

The proposed source and handoff contract is written and has no implementation
effect. The following fixed examples define expected later evidence.

### Fixed synthetic examples

All IDs, dates and amounts below are fictional. The selected book is B1, the
selected source S1, and no example performs an action automatically.

| Case | Inputs / evidence | Expected presentation and action |
| --- | --- | --- |
| Actionable review | Authorized S1 read returns one `needs_review` row R1 at source revision `7`, `USD 12.34`; detail and categories confirm current pins. | `Review source: 1 visible item` for S1 only. Open R1's existing review flow; draft creation makes no journal. Explicit command confirmation remains separate. |
| Empty selected source | Authorized S1 source list and complete `needs_review` page return `items:[]`, `next_cursor:null`, revision `7`. | `No known items for S1` under that filter. No book-wide All clear or zero across unseen sources. |
| Incomplete page | S1 first page returns a cursor and the next page fails or returns `review_snapshot_changed`. | `Status unavailable` or `Changed since you opened it`; do not call S1 empty or keep stale action pins. |
| Optional runtime absent | Connection reads are unavailable because the connection service is not composed. | `Connection status unavailable`; no zero attempts, no Begin/retry inferred, no global All clear. |
| Denied book | Current read grant for B1 is revoked between handoff and fetch. | `Permission denied`; clear B1 financial/status detail and discard late prior responses. |
| Stale status | Stream action was shown with generation `10` and revision `3`; fresh status is generation `11`/revision `4`. | Explain changed status and require review of the fresh pins before any confirmation. A queued request is not completed acquisition. |
| Other-book command | Principal pending command targets B2 while B1 is selected. | Do not disclose B2 body in B1. State only a generic principal gate blocker if authorized by the recovery contract; navigate to B2 only through verified current scope. |
| Handoff/back | Ledger passes B1, date `2026-10-20`, account A1 and revision `42` in memory. Investigate has no account-wide status index. | Retain B1/date context, ignore unsupported A1 for queue filtering, read a selected owned source; return to Ledger with B1/date/A1 and let Ledger refresh revision `42`. |
| Transient report error | B1 report returns 409 for revision `42`; no durable issue ID exists. | Keep the error in Reports/Ledger and offer refresh. Do not create an Investigate queue item or count. |

## Remaining work

`make check test build` passed with local Go 1.26.5, a temporary copied module
file changing only the Go directive, a temporary build cache, and execution
outside the sandbox for synthetic loopback listeners. The pinned Go 1.26.10
download failed with a disconnected network socket both inside and outside the
sandbox; no repository module file changed. `make check` verified local links,
migration metadata and stage-runbook structure; `make test` ran the race suite
and migration-file checks; `make build` built the named binaries. This is a
compatibility validation, not a pinned-toolchain result. These examples need implementation tests for API current
authority, pagination, denial, stale pins, one-item and empty states, and browser
keyboard/390px behavior. No source list currently proves a complete book-wide
status. Color/font and top-level placement require the separate design review.

## PR handoff

Link #255, record validation, and identify this as design only with no migration
impact. Leave the issue open for reviewed implementation and browser/API proofs.
