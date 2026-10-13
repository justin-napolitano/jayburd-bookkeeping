# Project status — 2026-10-13

## Decision

Continue small Go/PostgreSQL delivery slices. Merged PR #79 is the current
baseline (`5490f6d`). Implemented local behavior is distinct from hosted proof
and approval to keep official books.

| Area | Evidence and remaining boundary |
| --- | --- |
| Ledger | Atomic double-entry posting, immutable journals, explicit reversals and exact valuation: [posting proof](runbooks/posting-proof.md). Advanced FX and accounting-policy cutover remain separate work. |
| Interactive bookkeeping | Book-first navigation adds visible balances, account activity, reports and manual entry: [workspace evidence](runbooks/book-workspace.md). Recoverable commands: [recovery evidence](runbooks/command-recovery.md). |
| Financial review | Trial balance, account general ledger, balance sheet, P&L, comparisons, journal detail and internal CSV: [review tools](runbooks/report-review-tools.md). |
| Synthetic demonstrations | Real browser/Go/PostgreSQL posting; 1,000-posting proof passed. Full 5,000/10,000 runs remain unproved: [demo evidence](runbooks/synthetic-demo.md). |
| Background work | Restricted observer SQL and recovery tests merged: [foundation evidence](runbooks/ingestion-foundations.md). Local Go adapter and injected-transport proof are merged: [runtime evidence](runbooks/observer-runtime-delivery.md). Hosted queue/OIDC proof remains pending. |
| Ingestion | Pure CSV decoder, synthetic source registry and private local staging implemented: [delivery evidence](runbooks/product-workspace-delivery.md). Upload-to-review, source posting and Plaid remain future work: [import contract](contracts/durable-import.md). |
| Hosted application | Read-only Cloud Run service listing across regions returned no services in `jayburd-bookkeeping-dev` on 2026-10-13. No hosted frontend URL is available from that check. |

## Review the frontend

Start with `make demo` using the [demo prerequisites](runbooks/synthetic-demo.md).
Open the printed loopback URL. Data is fictional, disposable and backed by real
ledger validation.

1. Choose **Manual books** and select **Entity and book** at the top.
2. **Overview** loads account balances for the visible through date; **Accounts**
   keeps the same ledger-backed balance view.
3. Select an account to open **Account activity**, then **Review journal line**
   to inspect its journal. The activity date range is explicit and revision-pinned.
4. **Reports** provides trial balance, balance sheet, P&L, comparisons and internal
   CSV. **New journal** contains manual entry; recovery remains visible and blocks
   new work until required acknowledgement.

This delivery adds runtime navigation. Data remains synthetic; connected-bank
feeds and consolidated multi-entity reports are not implied. Review the [workspace
evidence](runbooks/book-workspace.md) for named tests and current limitations.

## Next delivery

Follow the [product and collection milestones](plans/product-and-ingestion.md).
#80 improves runtime navigation; #81 establishes stable scoped source identities;
#82 adds immutable synthetic staging. End-to-end import, source-linked posting,
Plaid acquisition and reconciliation remain subsequent acceptance slices.
