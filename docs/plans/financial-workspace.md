# Financial workspace design

## Decision

Issue #80 implements a book-first workspace under #52/#50 using the existing
Next.js identity, durable-command and ledger-read boundaries. The static
[prototype](../prototypes/financial-workspace.html) remains design history, not
an application route or runtime data source. No combined balances across LLCs.

## Implemented work

Select an authorized entity/book, then use **Overview, Accounts, Account activity,
Reports or New journal**. The compact header retains functional currency and
explicit dates. Workspace metadata revision is labelled separately from freshly
read report snapshots. Overview and Accounts read the existing trial-balance API;
amounts remain exact strings. Account activity uses the existing scoped general
ledger, followed by revision-pinned journal detail. There is no book-wide
transaction endpoint or connected bank feed.

Account balance → account ledger → journal → Overview preserves date context.
Changing book remounts scope-owned state; dates invalidate superseded responses.
Hidden report views cannot repopulate from late requests. Denied reads clear the
book selection. Loading/error/denied is never rendered as zero. Report comparison,
CSV review download and source-provenance behavior retain their owned contracts.

Recovery stays mounted. Idle status and discovery action remain visible in a
compact row; registration uncertainty, pending work, errors and unacknowledged
results retain the full recovery panel. Scope/navigation/editing remain locked
until authoritative recovery permits new work. New journal moves behind explicit
navigation without bypassing review or terminal acknowledgement.

## Evidence and remaining gates

[Living workspace evidence](../runbooks/book-workspace.md) owns named tests and
review captures. Native controls, visible focus, semantic tables and mobile layout
reuse [FW01–FW02](../research/financial-workspace-sources.md). Amount/read correctness
still comes from SQL contracts, not screenshots or synthetic browser mocks.

Owner visual approval, full accessibility audit and hosted evidence remain separate.
Review inbox, reconciliation, bank-provider ingestion and live policy approval are
future work; #80 does not close #52/#50. No arbitrary functional FX or audit-export
conformance is introduced by this interface.
