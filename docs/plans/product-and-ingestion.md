# Product and data-collection delivery

## Decision

Make Jayburd useful for daily multi-LLC bookkeeping: visible balances, traceable
transactions and explicit review. Competing with established financial products
is an ambition, not a benchmark result. Go/PostgreSQL/Next.js and reviewed
accounting contracts remain authoritative. GitHub owns current task/lease state.

## Work

| Milestone | Delivery and acceptance |
| --- | --- |
| Usable books | #80 under #52/#50: compact book context; Overview, Accounts, Account activity, Reports and New journal. Owner can find a balance, inspect its postings and return with context. Keyboard/mobile and recovery locking must pass. |
| Durable source foundations | #81 stable account/namespace identity and scoped grants; #82 immutable local synthetic staging. Contracts precede code. Neither alone is a working importer. |
| Complete CSV workflow | #8: register batch, publish validated preview, explicitly confirm recoverable import, persist observations/provenance, review/classify, then atomically post and link a journal. Upload/preview/import/draft each create zero journals. |
| Connected acquisition | #7 private HTTP/OIDC/Cloud Tasks service composition; #70 Plaid enrollment, sync pagination, verified webhook triggers and scheduled catch-up. All feed the same source/review core. Provider acquisition does not imply bookkeeping completion. |
| Reconcile and operate | #10 statement reconciliation and close; #58 recovery/retention/load/cost evidence. #27 hosted application and #13/#1 source qualification/official-books cutover remain separate gates. |

Parallel work has separate worktrees and bounded leases in #65. One SQL writer
owns migration9; coordinator owns shared API/CI/version integration. Reuse stopped
threads; no duplicate agents or shared-file writes. Keep merged migrations immutable.

## Done when

The [sync quality gates](sync-quality.md) connect #84–#90 to these milestones:
independent reconciliation, provider recovery, fuzz/isolation, webhook security,
Sandbox, measured scaling and a separately authorized live pilot.

Named tests prove each journey with independent synthetic results, exact currency
units, replay/conflict/failure and current scope checks. Distinguish source evidence,
drafts and posted journals. Every balance links to ledger evidence; unavailable
or stale data is never displayed as zero. Record measured fixture sizes and
actual latency before making scale claims. Owner reviews the completed journey.

## Blockers

No active provider or hosted frontend is implied. Private cloud storage, raw
retention/deletion, connection identity/secrets and hosted queue acceptance remain
required. Initial staging/enrollment are synthetic only. Advanced FX and group
consolidation remain explicitly deferred under #59.
