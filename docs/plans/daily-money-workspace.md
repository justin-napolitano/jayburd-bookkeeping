# Daily money workspace —412

## Decision and sequence

Owner approved the2026-11-03 Sandbox audit and Plaid-first, deterministic-rule
categorization approach. GitHub owns task state; this is the design/dependency
map. Estimates are relative; no delivery dates or closed predecessors inferred.
Existing source-scoped improvements ship before private cross-book capabilities.

| Issue | Bounded outcome | Dependencies |
| --- | --- | --- |
| #407 | Primary Transactions entry and posted/imported handoff | merged406/409; current scoped reads only |
| #408 | Search/date/coverage and whole-result pagination | reviewed query/transport contract;226/404 semantics |
| #415 | Compact feed/context, mobile rows, focused detail |407; coordinate408/305 |
| #416 | Reproduce/fix unavailable categorization and recovery | investigate first;291/294/404 as cause requires |
| #290 / #104 | Recognizable labels, compact status, secondary disconnect | reviewed safe source metadata; preserve identity |
| #291 | Read unavailability versus mutation/recovery uncertainty | current command boundary |
| #413 | Plaid PFC primary/detailed/version/confidence projection |238/242 authority/taxonomy for implementation |
| #414 / #243 | Private deterministic rules, preview, attribution/override |238/242 and reviewed provider/taxonomy wire |
| #417 | Exact spending/merchant metric definitions/read models | reviewed financial contract; labels as supported |
| #418 | Overview/Spending, category and merchant drill-down |417/408; classification wire as used |
| #419 | Evidence-backed opportunities and owner scenarios |417/418 and attributed classification |
| #420 | Report checkbox/initial state, headings and conditional diagnostics |407;291/250 coordination |

Primary daily experience evolves toward Overview, Transactions, Spending and
Accounts; Ledger, journals and formal reports remain accessible in Bookkeeping.
#247 retains its posted Position/evidence outcome; its earlier shell preference
does not override this newer owner-approved daily navigation direction.
#238 -> #239 -> #304 -> #305 remain genuine cross-book/taxonomy gates under242/237.
None is treated as done because a mock, unrelated deployment or plan exists.

## Categories and rules

Accepted product direction; exact financial/SQL/API contract remains414's gate:
explicit transaction override, approved specific owner rules, broader rules,
attributed Plaid fallback, then Uncategorized. Equal-priority disagreement must
be explained/abstained; no silent arbitrary tie winner. Show source/rule version
and why it matched. Offer a reusable future rule after correction with a match
preview; prior-label changes require explicit reviewed scope and audit. Keep
provider category, private topic, suggestion, accounting category and posted state
distinct. Labels never rewrite source observations or posted journals.

[Research](../research/categorization-rules-sources.md) distinguishes reviewed
implementation documentation from abstract-only research and unverified accuracy.
Actual is a source/pattern candidate, not an added runtime dependency. No new
Python/service/model/vector database. Review actual copied-file licenses and
retain notices before reuse; hledger's regex/account semantics are not imported
blindly. Provider version/mapping is pinned and attributed rather than invented.

## Overview, merchants and saving opportunities

Every summary uses the complete authorized matching set and drills into matching
rows at a known source/taxonomy/rule snapshot. Preserve exact original amounts/
currencies and separate unrelated entities/currencies. Explicitly define refunds,
reversals, pending/removed revisions, transfers, card repayments and loan splits.
Money-in is not automatically income, and a card repayment is not a second
purchase. ISO4217/reference validity remains required;417 owns full standards/
recognition/date/time-zone mapping before implementation.

Account connection state, last successful import, provider update time, available
history range and proven completeness are different facts. Show unknown/stale
states and never substitute a report-generation timestamp. Bank-reported balance,
imported movement and posted ledger balance are separately named. No unsupported
bank-balance read is invented for a dashboard.

Savings candidates are traceable fees, recurrence and comparable-period spending
changes; recurrence does not establish an unused subscription. Owner-selected
reductions are explicit scenarios, not promised savings. No external data/model
upload, financial transaction, automatic cancellation or new provider service.

## Acceptance and delivery ownership

Each issue names independent fixed examples, permissions/revisions and applicable
core/web/API/SQL gates.390px first-row visibility, keyboard/focus, screen-reader,
contrast and zoom/reflow checks are separate evidence. The audit is not WCAG or
accounting conformance. No sensitive browser persistence or values in URLs/logs.
Live #65 leases control execution: sole SQL writer; shared navigation/styles/
OpenAPI/types coordinator-owned; disjoint lanes may run only after explicit
assignment and reviewed contracts. This plan does not dispatch agents.

Initial407 retains the existing palette/type and financial tools, adds a primary
Transactions button and truthful Position handoff, and preserves scope/locks.
The larger shell/Overview redesign is418/420; it is not claimed shipped by407.
No cloud, provider, consent, registration-cap or Production operation follows
from these issue updates.
