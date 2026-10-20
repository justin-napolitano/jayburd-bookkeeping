<!-- stage-runbook: true -->
# Plaid history currency release

## Status and boundary

Owner authorized development Sandbox deployment on 2026-10-20. Issue #225,
implementation PR #227; branch release/plaid-history-currency. Existing enrollment
and explicit application retry are retained. No live-account scope.

## Stage ledger

1. **Implementation validation — passed.** PR #227 merged as
   `6c2db0076c7d21f42d9dde78547eef427daa0066` after Seed run
   `39476121675` passed all checks. Local gates are recorded in the linked
   delivery ledger.
2. **Candidate — passed.** Main image run `39478271802` passed validation and
   all four image builds, runtime proofs and scans. Retained candidate receipts
   bind api, migrate, web and worker digests to the merged source commit.
3. **Release review — prepared.** The migration-only and paused-worker manifests
   bind the same successful build, finite policy and [compatibility evidence](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/225#issuecomment-6298637942).
   Both pass their exact build-receipt validators. Manifest SHA256 values are
   `2722d1f76b69de59e61b8642d19664fb4aa3532bc91f0a296c98aa31ce329d7f`
   (migration) and
   `f44a556f4a4f5954b4e7194fa3ef2813015b4d1e5f54f3716f2af6e362effe22`
   (worker).
4. **Migration 25 — pending.** Append reference history without rewriting current
   references, bindings or observations; do not execute down migrations.
5. **Worker — pending.** Scheduler is observed paused. Deploy matching worker
   after migration succeeds, before explicit queued retry dispatch.
6. **Hosted acquisition/review — pending.** Owner-session request, controlled
   worker execution and completed publication/review are independent acceptance.

## Current evidence

Local implementation proof is recorded in the linked
[delivery ledger](plaid-history-currency.md). Operator metadata confirms Scheduler
PAUSED before promotion. No private provider records, credentials or Terraform
state/plans are inspected or retained in this delivery record.

## Remaining work

Merge the exact release manifests and deploy migration then paused workers.
Existing app runtime does not require a new image for this
internal worker context change. HTTP 204 is not proof of acquisition completion.

## PR handoff

Refs #225/#226/#227. Preserve the exact binding/client versions and existing
finite connection authorization. No automatic enrollment, journal or retry reset.
