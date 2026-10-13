# Financial workspace source notes

## Decision and review boundary

Support the proposed [workspace design](../plans/financial-workspace.md), issue
#52. Reviewed 2026-10-13; selected public sections only. These sources constrain
presentation, not accounting policy, authorization implementation or accessibility
conformance. Existing accounting and ISO sources remain authoritative in the
[bibliography](bibliography.md); no new financial interchange is introduced.

## FW01 — Keyboard focus

Publisher: W3C Web Accessibility Initiative. Resource:
[Understanding WCAG 2.2 SC 2.4.7: Focus Visible](https://www.w3.org/WAI/WCAG22/Understanding/focus-visible.html).
Version: WCAG 2.2 explanatory guidance; page updated September 6, 2026.
Reviewed: Success Criterion, Intent and Benefits. The source explains the need
for a visible keyboard focus indicator. Interpretation: use native buttons and
selects, persistent focus outlines, a skip link and explicit focus after changing
detail views. This is informative guidance, not a full normative WCAG review;
keyboard/screen-reader/contrast audits remain required before acceptance.

## FW02 — Financial tables

Publisher: W3C Web Accessibility Initiative. Resource:
[Tables with One Header](https://www.w3.org/WAI/tutorials/tables/one-header/).
Version: living tutorial, accessed October 13, 2026. Reviewed: Overview and
Table with header cells in the top row only. The source distinguishes header
cells and warns about ambiguous relationships. Interpretation: the prototype
uses table captions and explicit column header scope. A card-only financial
layout was considered but obscures account comparisons and journal sides;
summary cards accompany semantic tables instead. No screen-reader acceptance
is inferred from markup alone.

## Data evidence

The embedded fixture is an exact decoded copy of
[`golden-month.json`](../../tests/corpus/golden-month.json)'s books array.
Fixed report expectations come from that file and the corpus guide: checking
1,045 USD, savings 100, assets 1,145, capital 1,100, distributions 25, revenue
200, expenses 130 and net income 70. Book B remains separate at 70 USD assets.
Money formatting operates on strings; there is no browser FX conversion.
These figures support a synthetic design review, not a runtime reporting claim.
