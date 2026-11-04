# Compact transaction reading —415

Accepted owner direction412; bounded existing authorized book/source reads only.
Stacked on407/PR421, whose review/merge remains independent, not assumed complete.
One route heading and compact named book/currency context replace duplicate
headings and ledger date controls in the feed. Transaction date filtering waits
for408's server contract; hiding unrelated ledger inputs does not change dates.

Single-source selection is a visible default only after current sources confirm
exactly one active source. An explicit connection handoff must itself be present
in the current source list. Multiple/paused/missing sources stay explicit; never
fallback from an invalid explicit handoff to a different account. Remember an
owner's manual source choice in this mounted book's memory for ordinary navigation;
connection handoffs do not overwrite that preference. Scope/revision/lock unmounts
still discard snapshots, and no financial state/tokens enter browser persistence.

Desktop retains the existing sorted semantic table and server cursor. At650px
and below, the same table rows reflow into merchant/amount, date/status and
direction/action lines; sort controls remain keyboard-visible. No second row
dataset, local sort, currency sum or fabricated category. Original exact amounts,
dates and currencies unchanged. Category labels wait for413/242, not inferred.

Detail is a focused page within the same route; hide the list while retaining its
snapshot/selection. Back restores the selected row's focus and visible position.
An uncertain draft request disables Back and keeps its unchanged-key retry;
never erase that recovery intent. After a successful draft invalidates the old
page, Back refreshes instead of resurfacing stale rows. Failed detail reads leave
the list available; lost authority clears all detail/page state as before.

Sources: inherited406 PG17 ordering/limit and W3C APG sortable-table selected
sections plus current identity/command-recovery/source-review contracts. This
slice changes no financial exchange/API/SQL; no new ISO20022 message or audit
export is applicable. Existing ISO4217/exact original units mapping unchanged.
Read-only UI projection is not an ISO or accessibility conformance claim.

STRIDE scope: no new trust boundary/grants/integration. Retain current server
authorization and deny cleanup; ignore superseded responses; bound existing
pages; no private persistence; do not discard uncertain mutation intent. Named
evidence: TestSingleSourceSelection/InvalidHandoff, TestSourceContextReturn,
TestReviewDetailFocusReturn, TestTransactionPrimaryViewport390, revoked/scope/
stale/late-source and uncertain-draft regressions, locked web gates, core and
actual synthetic app integration. Screen-reader semantics and zoom checked
separately from visual screenshots; no full WCAG assertion.
