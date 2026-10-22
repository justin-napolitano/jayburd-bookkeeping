# Focused financial tools — #249

Owner-approved visual/scope, 2026-10-22. The [approved reference](../prototypes/position-approved.png)
and [Position contract](../contracts/position-read-model.md) drive this slice.
Read the [source record](../research/position-sources.md) and
[delivery evidence](../runbooks/position-workspace.md) before extending it.

## Screen and navigation

Position opens with all currently authorized books, grouped by entity. A scope
selector narrows it to an entity or book. Each row shows functional currency,
assets, liabilities, book equity and signed equity change from an explicit earlier
date. The SQL report supplies every amount and difference; the browser only
formats exact strings. There is no cross-book total, consolidation or FX conversion.
Every book has its own report revision and generation timestamp. A failure in one
row is unavailable, not zero, while other authorized rows remain usable.

Position, Ledger and Reports are always visible in the top navigation. Ledger
uses the existing account-scoped report and journal reader. Reports retains the
existing statement/comparison/export tools. Those tools require a book; opening
them from All books presents explicit book choices rather than silently selecting
one. A Position book link selects that book and hands its revision to Ledger's
first read. A changed revision yields the existing conflict and explicit refresh.
Dates remain in memory across tools. Review, journal, accounts, connections,
refresh and sign-out remain available through Tools. Setup is a separate disclosure.
An unacknowledged command stays visible and blocks unsafe new work and reads.

## Visual language

One clean system sans-serif family, readable dark text, modest headings, aligned
tabular numerals and white background. No typewriter effect or decorative hero.
Book rows alternate faint neutral gray (`#f3f4f4`) and unshaded white; entity
headings remain white. No colored column backgrounds. Positive equity changes
use restrained green (`#286044`), negative changes muted red (`#9b3e38`), and exact
zero stays neutral. Signs remain visible independently of color. These are signed
movements, not claims of profitability or funds available to spend. Other asset,
liability and credit values remain neutral; an increase is not universally favorable.

At narrow widths, each book becomes a labeled two-column reading inside the same
semantic table, preserving every amount and its label without page overflow or
sideways scrolling. Desktop remains the approved six-column statement. Native
date/select controls, focus outlines, heading focus and accessible snapshot
descriptions support keyboard and assistive use.

## Boundary and acceptance

No API/OpenAPI, SQL, migration, accounting policy, authorization or durable
command storage changes. Standards applicability and versioned currency mapping
are inherited from the reviewed Position/reporting contracts; no new financial
message or ISO designation is introduced. The current server's synthetic
functional-policy limits remain explicit; original multicurrency units remain
available in Ledger/journal evidence. Cash classification, available-to-use
planning, investigation status ownership and group consolidation retain their
own reviewed contracts/issues.

Acceptance evidence: Position unit cases for exact large decimals, versioned
currencies, old-response invalidation, zero/unavailable, denial and bounded reads;
browser cases for all-book/entity/book scope, dates, partial failure, keyboard
tool navigation, recovery lock and desktop/mobile layout; existing saved-command
journeys and real browser/Next/Go/PostgreSQL integration. Visual QA compares the
approved reference with actual screenshots. Test-only fixtures never enter app routes.
