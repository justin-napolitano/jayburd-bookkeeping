<!-- stage-runbook: true -->
# Strategic tags and private recovery integration

## Status and boundary

Source implementation from main1a1491a under GitHub450/451/65 claim
TAGREC-20261107T0340Z. Synthetic data only; no hosted/provider/live activation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume and claim | Verified449 merged and main1a1491a; clean dedicated root/SQL/UI/proof worktrees; one SQL44 writer, three independent agents |
| Profile freeze | collection-tags-recovery.md fixes v2 tags, exact result sets and payload-free private recovery before DDL/dependent implementation |
| Source review | Revisited PostgreSQL17 isolation/locking and RFC9457 fixed diagnostics; current verified owner, receipt replay and primed snapshot threats recorded |

| API and transport | Fixed v2 SQL calls and exact opaque recovery envelopes; active actor separate from raw JSON; SERIALIZABLE recovery including GET, no retry after unknown commit. Focused Go adapter/server tests pass. Exact bridge/version/cap/same-origin checks extended; old schemas preserved while v2 OpenAPI/types added |

| Compatibility and gating | Every original OpenAPI schema/route is semantically identical to merged449.106 focused browser bridge/API tests pass. Go server feature/origin and adapter tests pass; matching44 readiness now requires all eight new grants. Existing43 runner pinned explicitly43; separate44 proof runner/CI preserves all old proofs |

## Current evidence

Implementation in progress. Parent449 all five checks passed; that is baseline,
not acceptance of this new source. GitHub450/451/65 own actual progress/results.

## Remaining work

Implement SQL/API/UI and independent proofs; inspect synthetic UI; run required
checks and full canonical browser/API/PostgreSQL journeys before source merge.

## PR handoff

Refs450/451/242/65. New migration44 only; merged1–43 immutable. No new dependency,
provider call or live record. Security-sensitive data/API review remains explicit.
