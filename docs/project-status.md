# Project status — 2026-10-13

## Decision

Continue small Go/PostgreSQL delivery slices. Merged PR #79 is the current
main baseline (`5490f6d`) at this build dispatch. Planning PR #91 and durable import PR #95 merged into
the still-open #83 branch; this delivery stacks on #83. Implemented local behavior is distinct from hosted proof
and approval to keep official books.

| Area | Evidence and remaining boundary |
| --- | --- |
| Ledger | Atomic double-entry posting, immutable journals, explicit reversals and exact valuation: [posting proof](runbooks/posting-proof.md). Advanced FX and accounting-policy cutover remain separate work. |
| Interactive bookkeeping | Book-first navigation adds visible balances, account activity, reports and manual entry: [workspace evidence](runbooks/book-workspace.md). Recoverable commands: [recovery evidence](runbooks/command-recovery.md). |
| Financial review | Trial balance, account general ledger, balance sheet, P&L, comparisons, journal detail and internal CSV: [review tools](runbooks/report-review-tools.md). |
| Synthetic demonstrations | Real browser/Go/PostgreSQL posting; 1,000-posting proof passed. Full 5,000/10,000 runs remain unproved: [demo evidence](runbooks/synthetic-demo.md). |
| Background work | Restricted observer SQL and recovery tests merged: [foundation evidence](runbooks/ingestion-foundations.md). Local Go adapter and injected-transport proof are merged: [runtime evidence](runbooks/observer-runtime-delivery.md). Hosted queue/OIDC proof remains pending. |
| Ingestion | SQL/Go foundation connects private synthetic staging to validated preview and explicit recoverable import into immutable observations/review items: [delivery evidence](runbooks/durable-import-delivery.md). Generated acquisition and category review/source-linked posting are the current slice: [populated demo](runbooks/populated-demo.md). Plaid remains future work. |
| Hosted application | Read-only Cloud Run service listing across regions returned no services in `jayburd-bookkeeping-dev` on 2026-10-13. No hosted frontend URL is available from that check. |

## Review the frontend

Start with `make demo` using the [demo prerequisites](runbooks/synthetic-demo.md).
Open the printed loopback URL. Data is fictional, disposable and backed by real
ledger validation.

1. Click **Load demo books** to import 3,000 generated transactions across three
   fictional LLCs. Select an LLC using **Entity and book**. Loading does not post.
2. **Overview** loads account balances for the visible through date; **Accounts**
   keeps the same ledger-backed balance view.
3. Select an account to open **Account activity**, then **Review journal line**
   to inspect its journal. The activity date range is explicit and revision-pinned.
4. **Reports** provides trial balance, balance sheet, P&L, comparisons and internal
   CSV. **New journal** contains manual entry; recovery remains visible and blocks
   new work until required acknowledgement.
5. **Review transactions** opens source evidence. Choose a category, create a
   draft, review its posting command and explicitly post. Acknowledge the result,
   then inspect the updated balance or the posted item's linked journal.

This delivery adds runtime navigation. Data remains synthetic; connected-bank
feeds and consolidated multi-entity reports are not implied. Review the [workspace
evidence](runbooks/book-workspace.md) for named tests and current limitations.

## Next delivery

Follow the [product and collection milestones](plans/product-and-ingestion.md).
#92/#93 provide the durable SQL/Go import foundation; #94 adds independent decoder
properties. The current #96–#98 slice adds generated acquisition, review/category drafts and
explicit source-linked posting without asking the owner to prepare files. [Quality gates](plans/sync-quality.md)
remain open for complete reconciliation, provider integration and measured scaling.
