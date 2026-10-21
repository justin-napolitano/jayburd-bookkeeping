# Position read model — proposed for #248

## Decision boundary

**Status:** proposed for owner review, 2026-10-21. This contract defines a
single-book, explicit-date Position view over posted journals. It does not approve
live-entity policy, change SQL/HTTP, classify an account as cash, or calculate
spendable funds. The existing [financial reporting](financial-reporting.md),
[reporting SQL](reporting-sql.md), [HTTP](reporting-http.md), and
[pinned journal](report-drill.md) contracts remain authoritative for implemented
report arithmetic and access. [Evidence](../research/position-sources.md) records
review depth, inference and alternatives.

The proposed first-screen vocabulary is **Position**, **Cash**, **Other assets**,
**Total assets**, **Liabilities**, **Direct equity**, **Unclosed earnings**, and
**Total equity**, followed by **Change since [date]** and **Needs attention**.
The owner must approve these labels and which accounts qualify as cash. A line
whose policy is unresolved displays `Not configured`, never an invented zero.

## Selection, exact arithmetic and trace

Request exactly one verified, currently readable entity/book, an inclusive
`through_date`, and an explicit earlier comparison `through_date` if movement is
shown. Dates are accounting dates in the book's supported UTC policy, independent
of browser time and journal `posted_at`. A Position line is a point balance,
not a period flow. `Change since` is **current point balance minus earlier point
balance**, preserving a signed result. It does not claim cash flow, profitability,
or the cause of change. The balance-sheet comparison uses one authorized book
revision for both dates. Do not fill a failed or missing comparison with zero.

For eligible posted lines through the selected date, `N(account)=Σ(functional
debit−functional credit)` as an exact decimal. Preserve unexpected signs and
contra balances. The existing unclassified balance-sheet categories define:

| Position line | Exact functional-currency expression | Existing evidence |
| --- | --- | --- |
| Total assets | `Σ N(asset)` | Balance-sheet `totals.assets`, account rows and selectors |
| Liabilities | `−Σ N(liability)` | `totals.liabilities` |
| Direct equity | `−Σ N(equity)` | `totals.direct_equity` |
| Unclosed earnings | `−Σ [N(revenue)+N(expense)]` over eligible history | `totals.unclosed_earnings`; derived, never a journal |
| Total equity | Direct equity + unclosed earnings | `totals.total_equity` |
| Cash | `Σ N(a)` for a reviewed, versioned set of asset account IDs `C` | **Extension required**; no existing cash classification |
| Other assets | Total assets − Cash | **Extension required** after `C` is approved |

`Total assets = Liabilities + Total equity` is an invariant, not an adjustment.
Never take absolute values to force a pleasant presentation. Account code/name,
provider type and source account links cannot substitute for reviewed cash
membership. A credit-balance cash account stays negative. Inactive historical
accounts remain in totals. Empty valid books return exact zero statement totals,
with a visible `No posted entries` status; this is not an unavailable read.

The server must own cash membership and calculation, not browser-selected rows or
client-side arithmetic. A proposed `position-v1` read extension can return a
versioned `cash_account_ids` policy reference, `cash`, `other_assets`, and their
comparisons, computed with the existing balance-sheet authority and same
revision/transaction. It should reject missing, foreign, non-asset, ambiguous or
unsupported classifications and never silently omit an account. Its eventual
policy storage and update workflow require a reviewed decision before DDL.
Until then, the UI may show existing balance-sheet totals and `Cash: Not
configured`; it must not relabel total assets as cash. Initial Position can show
current and prior point totals from the existing comparison response; a single
server response is required for consistent movement.

All amounts remain canonical signed decimal **strings**, at most 29 integer and
9 fractional digits; zero is `"0"`. No float or JavaScript financial arithmetic.
Functional currency ISO 4217 code, pinned version ID and minor units accompany
each total. UI padding is display only. A book's original amounts/currency versions
and immutable rate IDs survive in the supporting GL and journal; original units
are never summed across currencies. No cross-book/entity consolidation. A policy
or functional-currency transition without approved mapping returns
`unsupported_policy`, never a converted total. The first implemented report
profile is synthetic accrual/UTC/USD functional currency; future functional
currencies require their own approved policy and tests.

Each account/derived line uses its server-generated selector to show contributing
accounts. A selected account opens scoped GL with the same `snapshot_revision`,
date bounds and exact amounts; a selected GL row opens the existing pinned
journal endpoint in the same revision. The current journal endpoint proves one
selected GL-line inclusion, **not** arbitrary subtotal-to-journal membership.
For subtotal evidence, the UI must expose the contributing account list and then
GL; a direct subtotal journal list needs a bounded server contract and tests.
For point balances, GL must cover opening history through `through_date`, not
only a convenient recent window. An empty account selection shows an explicit
zero and no journals. Original and reversal journals remain immutable on their
own accounting dates; the view never erases the original when corrected.

## Truthful state and attention

Position has three independent clocks: (1) the **accounting cutoff**
`through_date`, (2) the **book snapshot** `snapshot_revision` and
`generated_at`, and (3) **source freshness** from separately authorized
acquisition/status reads. A source being current does not prove its observations
were reviewed or posted. A posted balance does not prove reconciliation,
completeness or availability to spend. `generated_at` is computation time,
not a posting cutoff. Pending/failed commands may need attention but are not
included in balances until posted.

The short `Needs attention` list is a separate, typed, server-supported feed of
actionable items with scope, source/status ID, observed timestamp, state and
direct task target. Candidate kinds: unreviewed source observations, failed or
stale acquisition, and a pending/failed command requiring terminal
acknowledgement. Each kind needs its owning contract, authorization and a
reviewed freshness threshold. Until such a feed exists, render `Attention
status unavailable` or omit the list, not `All clear`. Do not infer a reconciled
state from an empty list. Source-only observations and drafts never enter the
posted Position. The attention feed may fail independently; its failure must not
change a valid posted balance into zero or imply a complete review.

| Condition | Required result |
| --- | --- |
| No book selected / no posted journals | Selection prompt / authorized exact-zero Position with `No posted entries` |
| Unknown or inaccessible scope | 404 with no totals or existence hint |
| Missing credentials / read grant denied | 401 / 403 with no financial body |
| Revision changed between views | 409 `report_snapshot_changed`; discard old evidence and offer explicit refresh |
| Unsupported policy, overflow or timeout | 422 / 503, no partial totals or substitute zero |
| Stale source status | Keep posted Position labelled by revision/date; show separately sourced freshness warning only |

Every report request reauthorizes current read permission before revision
comparison. Responses are `no-store`; no financial rows, tokens or requests in
browser persistence, analytics URLs or shared caches. A later page or drill must
not silently switch revisions. The existing report bounds (4,096 selected rows,
2 MiB response, bounded execution) may make Position unavailable for a larger
book; scale work needs measured evidence, not client truncation.

## Standards and decisions

This is an internal book-query response, not a selected bank-to-bank exchange.
ISO 20022 FAQ and message catalogue R01–R02 were reviewed at public overview
depth; no applicable registered message/version or counterparty usage profile
has been identified for this internal read. The extension therefore uses a
versioned local contract without an ISO designation. Reassess for an actual
financial exchange. ISO 4217 R03 applies to currency identity and versioned
metadata, not FX valuation or account classification. The existing reporting
currency mapping is retained. ISO 21378:2019 R04 and ISO 21926:2026 R05 are
relevant to future audit extraction/semantics; public abstract/preview only was
reviewed, so no field-level mapping or conformance is claimed. The proposed
Position response is not an audit export. No purchased normative text, external
usage profile or live accounting framework was reviewed for this slice.

| Open decision | Owner and condition |
| --- | --- |
| Cash account membership and change control, including restricted cash and cash equivalents | Owner/accountant under #1; approve policy/version and tests before a Cash total ships |
| Real entity basis, opening balances, FX/close semantics | Owner/accountant #1/#10/#59 before live use |
| Attention sources, thresholds and priority | Owning acquisition/command contracts plus owner workflow review |
| Position labels and first-screen order | Owner visual/content review before #249 acceptance |

**Alternative considered:** derive Cash from account codes or linked bank
accounts. It is easy to display but misclassifies noncash or restricted balances
and is not stable under renamed accounts. **Alternative:** aggregate all account
rows in the browser. It duplicates SQL sign/currency/policy rules and can mix
pages or revisions. The proposed server-owned classified subset preserves the
existing ledger/report boundary, at the cost of a policy workflow and extension.

## Named acceptance evidence

The fixed independent values in [Position sources](../research/position-sources.md)
are review fixtures, not generated expectations. Before implementation, add
`TestPositionFixedExample` (all lines, account IDs, signs, inclusive dates,
original/functional units and direct journal trace), `TestPositionCashPolicy`
(missing/foreign/reclassified/restricted IDs and version change),
`TestPositionRevisionAndAuthorization` (single snapshot comparison, stale/denied
read, no leaks), `TestPositionSourceExclusion` (source-only and pending commands),
and `TestPositionBounds` (overflow, unsupported policy and response caps). Browser
acceptance in #249 must prove the labels, explicit date, empty/unavailable states,
direct evidence path and no client financial arithmetic. Owner review is a
separate required acceptance item, not inferred from passing tests.
