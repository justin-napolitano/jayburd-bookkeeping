# Position design QA — #249

final result: passed

## Evidence and state

- Source visual truth: [approved all-books zebra reference](docs/prototypes/position-approved.png),
  1586 × 992 pixels, selected by the owner before implementation.
- Implementation: [desktop](docs/prototypes/position-desktop.png), actual Chromium
  render of production `Bookkeeping`/`Position` with five fixed fictional books;
  [mobile](docs/prototypes/position-mobile.png), 390 CSS px wide, same fixture.
- Desktop viewport 1440 × 900 CSS px, device scale factor 1. The full-page
  screenshot includes command/setup controls below Position; its top 900 px were
  compared with the reference normalized to 1440 × 900. No browser chrome included.
- [Full comparison](docs/prototypes/position-comparison.png): source on left,
  implementation on right in one image. [Focused table comparison](docs/prototypes/position-table-comparison.png)
  examines grouping, numeric alignment, typography, row shading and signed colors.
- Same all-books/as-of-2026-10-21/compare-2026-09-30 state. The reference's EUR
  functional example is illustrative; the browser fixture uses supported synthetic
  USD for all five books. Multicurrency rendering is separately unit-tested; the
  real SQL demo retains EUR original-unit journal evidence without changing its
  USD functional policy.

## Comparison history and findings

1. Initial desktop/mobile capture found a P2 mobile usability issue: the wide table
   put monetary columns off-screen. Replaced the narrow layout with labeled cells
   per book while retaining entity grouping, exact amounts, zebra rows and table
   semantics. Post-fix mobile capture shows every amount without horizontal page
   or table scrolling; the browser assertion passes.
2. Initial combined desktop comparison found P2 typography/column rhythm drift:
   table text was smaller and the book/currency columns pushed amounts away from
   the source alignment. Raised desktop table text to 1.15rem and set consistent
   column proportions. The final combined full/table captures above show readable
   aligned amounts and the approved calm visual hierarchy. Long real book names
   truncate visually while retaining their accessible full label.
3. Mobile top-bar review found an extra Tools wrap. Allowed the scope selector to
   shrink in the shared row. Final mobile capture shows brand/scope/Tools together
   and Position/Ledger/Reports directly below, without control overflow.

No actionable P0/P1/P2 visual findings remain. Expected differences: native date
controls use the platform's localized format; Tools, saved-command recovery and
book setup retain required application operations absent from the mock. The
mobile layout has no supplied visual target and was reviewed for readable scope,
labels, values, touch controls and overflow rather than invented pixel fidelity.

## Required fidelity surfaces

- **Fonts/typography:** clean system sans throughout; no serif/typewriter effect.
  Dark readable labels, tabular amounts, modest heading, bold entity/equity. The
  final focused capture was opened at full resolution to inspect numeric columns.
- **Spacing/layout:** primary tools at top, scope at right, grouped rows and thin
  rules. Zebra bands span each full book row. Narrow labeled rows preserve data.
- **Colors/tokens:** white, near-black `#202124`, neutral stripe `#f3f4f4`, positive
  equity change/active tool `#286044`, negative change `#9b3e38`. Signs carry meaning
  without color. No column tints, blue or amber accents.
- **Assets/image quality:** this screen contains editable UI text and native
  controls, with no pictorial assets to recreate. Mock screenshots are evidence,
  never used as application backgrounds or a replacement for functional controls.
- **Copy/content:** all-book/entity/book scope, explicit cutoff/comparison, currency
  per row, exact signed changes, unavailable/denied/locked states. No fake cash,
  attention count, mixed-currency total or consolidation claim.

## Interaction and verification

The five Position browser cases cover all-book/entity/book filtering, date
invalidation, partial denial, direct Ledger/Reports navigation, source-snapshot
conflict and explicit refresh, recovery lock, desktop/mobile layout and no
uncaught page errors in the primary flow. Existing report, posting/recovery and
review journeys also pass. Unit coverage includes exact large decimals, versioned
currencies, zero/unavailable, old-client/date responses and max-four reads across
superseded date generations. This is semantic/browser verification, not a complete
assistive-technology audit.

The live Edge demo additionally showed real per-book SQL totals and scoped Ledger
entries with EUR original units. The production Next/Go/PostgreSQL synthetic
integration asserts Position zero before posting and `125.50` equity after one
atomic recovered posting. No personal financial records were used.

## Follow-up polish

- P3: native date presentation varies by platform. Keep its keyboard/date behavior
  unless a future review justifies a custom control.
- P3: consider a compact visible per-book snapshot disclosure if owner review
  needs more than the current accessible revision/generation description.

## Implementation checklist

- [x] Match approved layout, zebra rows, readable type and restrained signed colors.
- [x] Verify desktop, mobile, scope/date changes, unavailable and recovery states.
- [x] Open normalized combined and focused comparisons after visual fixes.
- [x] Record expected product constraints and residual P3 items.
