# Project status — 2026-10-13

## Decision

Continue small Go/PostgreSQL delivery slices. PR #106 is merged (`6c6b546`) with synthetic enrollment/acquisition evidence.
This delivery adds the supported #104 connections workspace. Implemented local behavior is distinct
from hosted proof and approval to keep official books.

| Area | Evidence and remaining boundary |
| --- | --- |
| Ledger | Atomic double-entry posting, immutable journals, explicit reversals and exact valuation: [posting proof](runbooks/posting-proof.md). Advanced FX and accounting-policy cutover remain separate work. |
| Interactive bookkeeping | Book-first navigation adds visible balances, account activity, reports and manual entry: [workspace evidence](runbooks/book-workspace.md). Recoverable commands: [recovery evidence](runbooks/command-recovery.md). |
| Financial review | Trial balance, account general ledger, balance sheet, P&L, comparisons, journal detail and internal CSV: [review tools](runbooks/report-review-tools.md). |
| Synthetic demonstrations | Real browser/Go/PostgreSQL posting; 1,000-posting proof passed. Full 5,000/10,000 runs remain unproved: [demo evidence](runbooks/synthetic-demo.md). |
| Background work | Restricted observer SQL and recovery tests are merged: [foundation evidence](runbooks/ingestion-foundations.md). Merged #105 adds private HTTP and injected Cloud Tasks transport with real PostgreSQL replay tests: [transport evidence](runbooks/observer-transport.md). Hosted executable, queue and OIDC qualification remain pending. |
| Ingestion | Generated acquisition and category review/source-linked posting are merged: [populated demo](runbooks/populated-demo.md). Merged #105 adds fenced provider publication and exact Plaid decoding. Merged #106 adds scoped enrollment and a restartable acquisition runner: [delivery evidence](runbooks/provider-acquisition.md). Hosted composition, connection UI, webhook ingress and real provider qualification remain open. |
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

The merged workspace supplies runtime navigation. Data remains synthetic; connected-bank
feeds and consolidated multi-entity reports are not implied. Review the [workspace
evidence](runbooks/book-workspace.md) for named tests and current limitations.

## Connections workspace

The **Connections** tab lists scoped accounts, opens their transaction review and
confirms account-only disconnect. An injected Link adapter enables enrollment and
explicit verified account binding. Default startup has no adapter and shows setup
unavailable. [Evidence and remaining contracts](runbooks/connection-workspace.md)
distinguish this UI from working real bank connectivity.

## Next delivery

Follow the [product and collection milestones](plans/product-and-ingestion.md).
The merged #96–#98 slice supplies generated acquisition and explicit source-linked
posting without owner-prepared files. #100–#104 plan connected collection. The [enrollment and acquisition delivery](runbooks/provider-acquisition.md) builds
on the merged [sync foundations](runbooks/sync-foundations.md). Next: connection
UI (#104), verified webhook ingress (#103), hosted credential/provider composition
and separately authorized Sandbox qualification (#88). [Quality gates](plans/sync-quality.md)
remain open for reconciliation, provider qualification and measured scaling.
