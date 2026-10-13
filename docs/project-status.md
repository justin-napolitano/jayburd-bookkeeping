# Project status — 2026-10-13

## Decision

Continue small Go/PostgreSQL delivery slices. Merged PR #78 is the current
baseline (`38b9451`). Implemented local behavior is distinct from hosted proof
and approval to keep official books.

| Area | Evidence and remaining boundary |
| --- | --- |
| Ledger | Atomic double-entry posting, immutable journals, explicit reversals and exact valuation: [posting proof](runbooks/posting-proof.md). Advanced FX and accounting-policy cutover remain separate work. |
| Interactive bookkeeping | Entity/book workspace and recoverable manual commands: [recovery evidence](runbooks/command-recovery.md). |
| Financial review | Trial balance, account general ledger, balance sheet, P&L, comparisons, journal detail and internal CSV: [review tools](runbooks/report-review-tools.md). |
| Synthetic demonstrations | Real browser/Go/PostgreSQL posting; 1,000-posting proof passed. Full 5,000/10,000 runs remain unproved: [demo evidence](runbooks/synthetic-demo.md). |
| Background work | Restricted observer SQL and recovery tests merged: [foundation evidence](runbooks/ingestion-foundations.md). Local Go adapter and injected-transport proof are the current delivery: [runtime evidence](runbooks/observer-runtime-delivery.md). Hosted queue/OIDC proof remains pending. |
| Ingestion | Pure CSV decoder implemented; durable import/review and Plaid sync/webhooks remain contract/prototype work: [import contract](contracts/durable-import.md). |
| Hosted application | Read-only Cloud Run service listing across regions returned no services in `jayburd-bookkeeping-dev` on 2026-10-13. No hosted frontend URL is available from that check. |

## Review the frontend

Start with `make demo` using the [demo prerequisites](runbooks/synthetic-demo.md).
Open the printed loopback URL. Data is fictional, disposable and backed by real
ledger validation.

1. Choose **Manual books**; scroll past **Saved command** to **Workspace**.
2. Select **Entity and book** (Cents Lab or Generator Trading).
3. Find **Financial reports**, choose **Report type**, and **Refresh report**.
4. Trial balance shows account balances. General ledger requires an account and
   date range; use 2026-01-01 through 2026-12-31 for demo activity. Select
   **Review journal line** for journal detail. Balance sheet and P&L summarize
   the selected book.

The layout currently places reports below manual-entry controls. A visible book
summary with Accounts, Transactions and Reports navigation is still a frontend
usability improvement, not implemented behavior. No live bank connection or
consolidated multi-entity report is implied.
