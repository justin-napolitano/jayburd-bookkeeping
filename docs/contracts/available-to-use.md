# Available-to-use planning contract (proposal)

## Decision record

**Status:** proposed for #251, 2026-10-21. Owner/accountant acceptance pending.
**Question:** How can one book show a dated estimate of cash left after named
commitments without presenting a balance, forecast or authorization as fact?
**Scope:** read-only Plan tool for one authorized entity/book, one original
currency bucket at a time. No cross-entity pooling, payment, reservation,
posting, provider write, or service grant. #248 must define Position and #1
must settle real policy before a live result is accepted. #56 must define any
future service role. [Source review](../research/available-to-use-sources.md)
separates sourced accounting facts from these product choices.

### Alternatives and provisional selection

| Model | Benefit | Failure mode / disposition |
| --- | --- | --- |
| Cash less one reserve number | Easy to explain and enter | Hides dates, double counts bills already paid, and implies a single universal spending limit. Retain as a clearly named *owner reserve* input, not the entire model. |
| Dated cash-flow scenario with explicit obligations and reserves | Answers “for what and by when?” and exposes timing/uncertainty | Needs reviewed due-date, source status and overlap rules. **Proposed** model, subject to owner/accountant review and fixed-case validation. |

### Proposed meaning and arithmetic

The displayed heading is **“Scenario headroom through DATE”**, never “cash
balance”, “spendable cash” or “approved to spend”. A context line identifies
book, scenario, as-of accounting date, generated time, horizon, currency and
source freshness. The posted cash figure is a ledger-derived input, not the
result. Exclude unposted provider observations from that figure.

For one selected cash account set and **one currency** `C`, define `B_C` as
the posted cash position at book revision `R` and accounting date `D`, subject
to #248's account membership/sign rules. Each future cash movement has signed
exact amount `m_i`, effective date `t_i`, source identity and status. For every
day `d` in `[D,H]`, the proposed curve is:

`P_C(d) = B_C + sum(m_i where D < t_i <= d) - F_C(d)`

where `F_C(d)` is the owner-entered minimum cash floor valid on `d`. The
proposed headroom is `min(P_C(d) for D <= d <= H)`, with the date of the minimum
shown. A one-day horizon is permitted. Treat due-date uncertainty as a range of
separate scenarios, never silently choose the optimistic date. Missing amounts,
ambiguous overlaps, unreviewed policy, unsupported FX, stale/unavailable Position
or authorization failure return **unknown/unavailable** with reasons rather
than zero. A negative result is shown signed and labeled **shortfall in this
scenario**; it never clamps to zero.

Only cash effects belong in `m_i`. A posted payable and its later planned cash
settlement may both be visible, but the payable itself is not subtracted from
cash until its anticipated payment date. A posted payment already reflected in
`B_C` must be excluded from future movements. Proposed stable movement keys and
explicit supersedes/settles links prevent overlapping import, owner assumption,
and later service forecasts from silently counting twice. If the overlap cannot
be decided, block the estimate. This is a design rule, not an implemented match.

No implicit exchange or functional-currency roll-up. Each original currency
quantity keeps its ISO 4217 version and units. Functional booked amounts and
valuation provenance can be displayed beside posted evidence, but a future
foreign-currency obligation requires an explicit dated rate *scenario* and
separate policy review before any combined functional estimate. Never reuse a
posted historic rate as a future forecast or add unlike units.

### Inputs, provenance and uncertainty

| Input | Origin | Required trace and freshness | Treatment |
| --- | --- | --- | --- |
| `B_C` posted position | Authorized ledger/reporting read under #248 | entity/book, cash account IDs, accounting date, book revision, policy/currency versions, journal trace, generated time | Exact posted amount for that revision; completeness of source acquisition is separate. |
| Observed source item | Bank/provider or statement staging | source ID/version, observed/effective timestamps, status, account binding and ingestion state | Context only until posted or explicitly promoted to a reviewed scenario movement; never alters posted cash. |
| Obligation or expected inflow | Owner entry, posted payable schedule, or future scoped service | origin ID, amount/currency, due date or range, creation/review timestamps, owner, confidence and source link | Scenario assumption. Posted liability is evidence, not an automatic cash deduction. |
| Reserve floor | Owner entry | purpose, amount/currency, active dates, reviewer, version | Scenario constraint, not a ledger entry or held funds. |
| Adjustment | Owner entry | signed amount, reason, date, source/reviewer and expiry | Explicit scenario only; cannot correct ledger state. |

Confidence is displayed as **posted evidence / observed / owner assumption /
service proposal**, plus separate **confirmed / estimated / disputed / unknown**
review status. These labels do not express a statistical probability. A newer
source observation does not make an old book revision current. A refreshed book
revision invalidates a pinned scenario result until it is recalculated with
current read permission. Staleness threshold and escalation language require
owner review; v1 should show timestamps and a conservative unavailable state
rather than invent a universal threshold. Source errors and denied reads must
not reveal financial values or existence across books.

### Synthetic, independently worked illustrations

These decimal examples are fixed design expectations, not produced by code and
not approved live policy. Assume a fictional USD book, `D=2026-01-01`,
`H=2026-01-31`, exact opening posted cash `1000.00`, no other movements unless
specified, and a constant `100.00` reserve floor.

| Case | Inputs beyond base | Expected minimum and meaning |
| --- | --- | --- |
| Sufficient | None | `1000.00 - 100.00 = 900.00` on D; scenario headroom `900.00`. |
| Committed obligation | `-600.00` due Jan 15, linked to a posted payable; payable itself has no second cash effect | Jan 15 projection `1000.00 - 600.00 - 100.00 = 300.00`; headroom `300.00`. |
| Uncertain item | Another `-250.00` with unknown amount or due date | `unknown`, with uncertain obligation ID; do not publish `300.00` as complete headroom. A separate explicitly incomplete scenario may still show the known `300.00` subtotal. |
| Multiple currencies | USD base above and distinct EUR `200.00` posted cash, EUR `50.00` floor | USD `900.00` and EUR `150.00` shown separately; no `1050.00` mixed-unit figure. |
| Stale/unavailable Position | Posted snapshot cannot be refreshed or authorized for selected revision | `unavailable`; previously cached `900.00` is not shown as a current result. |
| Negative | `-1100.00` due Jan 15 | Jan 15 `1000.00 - 1100.00 - 100.00 = -200.00`; display `-200.00` shortfall. |
| Timing | `-800.00` Jan 10; `+900.00` Jan 20 | Jan 10 `100.00`, Jan 20 `1000.00`; headroom `100.00`, not end-date `1000.00`. |

The first case's cash and reserve are synthetic assumptions. It does not prove
the book is reconciled, funds are legally available, or the plan is executable.

### Future service boundary (proposal only)

Later services may receive a separately approved, revocable **purpose, entity,
book, currency, account and amount/time bounded** capability. A service may
first submit a proposal that the owner reviews; reading a Plan estimate grants
no write capability. A future execution contract must define trusted workload
identity, enrollment, current permission check at execution, explicit approval,
idempotency, outbox/worker recovery, revocation and audit evidence. It must not
derive authority from a scenario ID, browser state, provider token or caller
actor ID. Revocation must prevent queued work from executing a new effect.
No capability schema or API is approved here.

### Standards applicability and acceptance gates

The Plan calculation and prospective internal JSON are local analytical
semantics: **ISO 20022 not applicable as a registered message** because there is
no financial institution exchange or selected registered business message.
Any later payment initiation must choose its actual ISO 20022 message version,
namespace and counterpart usage profile. Currency identifiers and minor-unit
metadata: **ISO 4217 applicable**, pin the release and validity interval via
the existing reference contract. **ISO 21378:2019 and ISO 21926:2026:** relevant
to eventual audit export/semantic mapping, unresolved at field level because
only public abstracts/preview were reviewed; no audit-conformance claim. An
internal Plan export, if proposed, needs a separate applicability and loss review.
No ISO designation is invented for this calculation. See the source register
for edition, review depth and gaps. The accounting framework and jurisdiction
remain governed by #1, not by ISO naming.

Before implementation, owner/accountant must approve eligible cash accounts,
whether available means general or purpose-limited funds, obligations and
reserve categories, timing horizon, uncertainty policy, stale thresholds,
entity restrictions, and treatment of foreign cash. #248 must settle Position
semantics; #56 must settle service authorization. Follow-on work must name
`TestPlanExactTimeline`, `TestPlanOverlapAndPaidExclusion`,
`TestPlanUnknownAndStale`, `TestPlanCurrencyIsolation`,
`TestPlanRevisionAndRevocation`, and fixed corpus cases above. Tests need a
separate oracle and synthetic fixtures. No endpoint, SQL object, UI component or
service permission exists because of this document.

### Open owner decisions

1. Which obligations count: posted payables with expected payment dates,
   manually entered commitments, debt minimums, tax estimates, or another
   reviewed set? How are paid/duplicated items resolved?
2. Are reserves hard purpose restrictions, owner planning floors, or both, and
   who may change each reserve?
3. What horizon and date convention answers the owner's first planning question,
   and what uncertainty or source age makes the estimate unavailable?
4. Is the view per book, per eligible cash account, or per restricted purpose?
   Which funds can never be pooled, even within one book?

These decisions gate Plan acceptance only. Position and Ledger may proceed under
their own approved contracts without waiting for this proposed metric.
