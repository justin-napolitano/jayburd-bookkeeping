# Source activity, spending and merchant metrics —417/428

## Decision, owner and delivery gates

Proposed2026-11-03 for maintainer review under428, child of417. This defines
financial meaning and independent examples before SQL/API implementation. It
does not install an endpoint, authorize a bank read or approve accounting policy.
[Source review](../research/spending-metrics-sources.md) distinguishes published
facts, repository evidence and our interpretation. The fixed
[oracle](../../tests/corpus/spending/README.md) is separate from runtime code.

Phase A delivers exact **source activity** for one currently authorized book and
source: eligible inflow/outflow, attention counts, observed date range and
attributed merchant-label groups where stored normalized metadata supports them.
These are source observations, not income, expenses, bank balance, cash available
or reconciled ledger values. Credit/loan source activity is not automatically cash
movement. Source readiness does not depend on whether that date is postable.

Phase B adds **categorized spending** only after reviewed private taxonomy,
classification and allocation contracts (#238/#242/#243/#413 as used). No missing
classification becomes an expense or an Uncategorized count by default.418's
Overview/Spending consumes only implemented profiles and their exact drills.419's
opportunities wait for that evidence; recurrence alone proves no unused service.
Cross-book/entity aggregation retains238→239→304 dependencies and explicit scope.

## Grain, current facts and scope

One source account in one entity/book, trusted principal, explicit original-date
period and one source revision. Reauthorize entity and source before counts,
cursor/revision checks or disclosure; grants are not inferred from labels or
provider metadata. A foreign/denied scope fails without zero-filled financial data.
All filters/counts/totals/ranks apply to the complete matching set before paging.

Use one immutable current observation revision per source identity. Existing
review-item pages are not the inventory of all observations: pending/zero rows
may lack a review item. Raw old revisions and posted journals remain immutable.
Missing/ambiguous assignment, provenance or current-fact joins fail closed; no
join multiplicity may increase a metric. Input changes invalidate the snapshot.

Phase A eligibility: current booked, positive nonzero magnitude, installed dated
currency/precision valid and no overlap/identity ambiguity or precision hold. Needs-review,
draft-ready and ledger-posted review states can qualify; a book-policy date gate
alone cannot exclude otherwise valid imported history. Pending, removed, zero
and quality-held current facts have separate exact counts, not positive activity totals.
An accounting correction hold alone does not exclude a valid current source fact:
the oracle includes the corrected15 while the original posted100 stays unchanged.
Posting-correction attention can overlap eligible activity counts. If any quality-held
facts are excluded, eligible totals are explicitly partial. Never
round extra precision into eligibility or interpret a removed item as a refund.

Pending-to-booked links preserve lineage; pending amounts are never added to
booked totals. A modified observation contributes only its current eligible
revision. A removed or valid booked correction can change the source projection without
changing an already posted journal; expose attention and retain the original
journal/source revision. A read never creates a correction or clears a hold.

## Exact quantities and period

Money is a bounded exact decimal string at every boundary. PostgreSQL owns
aggregate arithmetic; Go transports strings; the UI formats original units using
the referenced currency metadata. Never use float, parseFloat or Number for money.
Group by `currency_version_id`, original ISO currency and minor-unit definition;
no implicit FX or folding separate reference versions into one apparent total.
This conservative first profile avoids assuming historical unit equivalence.
No grand total across currencies, books, principals or unreviewed source groups.

For eligible set E and original unit version U:

- `inflow(U)` = sum of positive magnitudes for direction=inflow.
- `outflow(U)` = sum for direction=outflow.
- `net_inflow(U)` = inflow minus outflow, allowed negative.
- Counts and revisions are decimal strings. A successful empty eligible set has
  zero quantities and count0 for its selected unit, with excluded/coverage state
  still explicit. Unknown balance/classification/freshness remain null/unavailable.

Amounts are quantities in source direction, not account normal-side debits or
credits. A card repayment can be inflow on a card and outflow on checking; neither
becomes a second purchase. No ledger sign/balance is derived from this metric.

Dates use immutable source booked_date. Bounds are inclusive `starts_on` and
`through_date`, real Gregorian dates with start≤through. Do not silently substitute
authorized date or shift a date through a timezone. Book timezone is display/
report context, not an invented source timestamp. Query generation time is a
separate RFC3339 snapshot timestamp. No host-clock-dependent period defaults.

Comparison supplies both intervals explicitly. Equal-period presets use equal
calendar-day counts; unequal full months display both lengths and are not labeled
equivalent. Delta is current minus prior in the same unit/version. Percentage is
null with `no_positive_baseline` when prior≤0; no Infinity or invented growth.
Positive-baseline percentages use exact ratios, explicitly half-even to2 decimal
places for display; this is analytical display, not currency valuation/rounding.

## Spending recognition and contribution drills —Phase B gate

This is an analytical purchase/spending profile, not GAAP/tax P&L or a statutory
cash-flow statement. Economic annotations below are explicit versioned reviewed
inputs, not interpretations inferred from a merchant string or transaction sign.

| Reviewed treatment | Spending contribution | Source quantity retained |
| --- | --- | --- |
| Purchase, fee or allocated interest | Positive allocated amount | Full original transaction and unit |
| Linked purchase refund | Negative refund on its own booked date, including prior-period purchases | Original inflow and reviewed purchase link |
| Internal transfer | Excluded from spending only with reviewed disposition; each source leg remains visible | Both legs, separate source scope |
| Credit-card purchase/repayment | Purchase contributes once; repayment contributes0 | Original card/checking records remain distinct |
| Loan principal/interest split | Principal0; explicit interest allocation contributes | Full payment plus exact reconciled allocations |
| Unsplit loan, cash withdrawal, unknown purpose | Unallocated/unknown; no guessed expense or all-clear total | Whole original quantity remains visible |

`gross_spending` sums positive reviewed contributions; `refunds` sums reviewed
refund magnitudes; `net_spending=gross_spending-refunds` may be negative.
Unallocated outgoing amounts/counts remain separate and mark incomplete spending
coverage. Categories, suggestions and posted ledger accounts retain distinct
provenance. Retagging never changes a journal or source observation.

Every total drills to exact **contributions**, not merely whole source amounts.
The220 payment in the oracle has20 interest and200 principal: an interest drill
shows20 contribution beside220 original, same source ID/revision and allocation
revision. Adding whole transaction amounts would fail equality. Phase A contributes
the whole eligible source magnitude; Phase B needs a reviewed allocation wire.
The existing408 contains-search endpoint alone cannot provide exact merchant or
partial-allocation drill equality; runtime work must add the reviewed exact/pinned
drill profile before418 links those metrics. No client-side financial filtering.

## Merchant provenance and unknowns

The current normalized Plaid transaction and stored provider_fields have nullable
merchant_name. PFC has no dedicated normalized field; legacy/synthetic profiles
may have no merchant metadata. Do not inspect raw payloads or refetch history for
this contract. A future approved projection selects only the named normalized
field, never the whole provider_fields object.

First merchant grouping profile: `provider-merchant-exact-v1`. Preserve original
label and profile/version; exact case-sensitive labels remain separate. A supported
profile's null/empty/whitespace-only label is `not_provided`; unsupported or absent
metadata profile is `unavailable`, not an empty successful merchant list. Never
infer a merchant from description/name, load its URL/image or treat its label as
merchant entity identity, legal owner or category. Alias normalization requires
its own reviewed mapping revision and current-source evidence.

Rank only within the same source/unit/version and selected contribution basis.
Stable ties use the exact group key; page bounded groups and return complete-set
totals plus remainder/count context, never silently drop lower-ranked groups.
An exact merchant-label drill cannot be implemented as substring description
search. The oracle deliberately keeps `Fictional Cafe` and `FICTIONAL CAFE`
separate and repeats a label in EUR to forbid cross-currency merging.

## Snapshots, transport, privacy and bounds

The implementation follow-up must choose a closed versioned POST read profile
with no private filters, merchant labels or cursors in URLs, logs or browser
persistence. Fresh bearer/current source authority and no-store apply. No caller
actor, credential, arbitrary SQL, idempotency mutation key or automatic retries.
No command registration/execution/acknowledgement or financial outbox effects.

First summary may request the current snapshot; response pins entity/book/source,
source revision, period/filter profile and currency versions. Every continuation/
metric drill sends the pinned revision and complete metric selector. A changed
source returns stale/refresh-required; never silently read a newer drill under an
older headline. Phase B additionally pins taxonomy/rule/annotation/allocation and
merchant-mapping revisions. These are consistency pins, never access grants.

Proposed engineering bounds:8KiB UTF8 JSON,100-codepoint literal search,500-codepoint
exact merchant label,100 source rows or20 merchant groups per page,100 unit/version
groups and existing server statement deadline. Reject overflow, malformed/duplicate/
unknown fields, ambiguous anchors and foreign/stale/filter-changed cursors. Exact
filters can be bound by a canonical selector digest so large private labels need
not be copied into cursors; digest is not authorization or secret authentication.
No full-table download, raised timeout or new search/database/model service.

STRIDE: restricted-data disclosure/escalation/high→current principal/entity/source
authorization before aggregate/drill, no raw provider object/PII logs; tamper/high→
closed versioned selectors, exact arithmetic and stale pins; DoS/medium→bounded
input/output/deadlines and independent3000-row benchmark; repudiation/high→
preserved source/annotation provenance, no implicit mutation or posting. Human
security/source review before sensitive runtime merge/apply; no conformance claim.

## Freshness, coverage and standards

Keep source state, last successful import commit, provider-supplied update time,
available current source-date range, provider readiness, completeness and value
reconciliation separate. SQL has `sync_stream.last_completed_at`; current source
list does not expose an approved source-to-stream operational projection. First
summary may return unknown freshness until that mapping/grant review is complete.
Never use query generation, token expiry or a successful GET as last bank sync.
An age threshold must be an explicit policy/version, not invented per component.
Missing balance/classification/metadata is null with a reason, not ledger zero.

| Standard/profile | Applicability and mapping |
| --- | --- |
| ISO4217:2015 public overview; installed dated SIX currency references | Applicable: retain original code, version, minor unit and validity per fact. No FX, unit replacement or implicit historical equivalence; full normative text unread |
| ISO8601-1:2019 and Amd1:2022 public metadata; RFC3339 (July2002) §5.6 | Applicable: original booked day/inclusive periods use YYYY-MM-DD; separately named snapshot timestamps use the existing RFC3339 UTC profile. No inferred timezone conversion, expanded/week/ordinal dates or full normative ISO conformance claim; SM06 records access limits |
| ISO20022 | No registered exchange/message/version selected for this internal analytical read. Existing proprietary Plaid API2020-09-14 adapter mapping remains; do not label custom JSON camt or a certified statement |
| ISO21378:2019 | Relevant to future audit exports, not a new export here. Public abstract/metadata only; field-level normative conformance unresolved and unclaimed |

STD01 is this assessment; STD02 closed profiles/unknowns; STD03 independent exact
unit/sign/date/contribution examples; STD04 correction/pin/drill invariants;
STD05 future API/migration compatibility and explicit version review. No new
normative ISO text or regulated accounting/tax treatment is approved by fixtures.

## Named acceptance and alternatives

Runtime acceptance remains **unimplemented** until actual SQL/API/browser evidence:
TestSpendingRefundsTransfersCards, TestSpendingCurrenciesSeparate,
TestWholeResultNotPage, TestMerchantNormalizationVersion,
TestAggregateDrillEquality, TestFreshnessUnknownStale,
TestAggregatePrincipalDeny, TestSourceRevisionSnapshot, TestReadNeverPosts.
Use the independent oracle, duplicate dates,3000-row complete-set/drill equality,
revocation/foreign/stale/filter mutations and zero journals/outbox/provider calls.
Core/migration/ledger/API/locked web/app-integration gates follow the changed boundary.

Rejected alternatives: treat all outgoing as spending (repayments/principal/transfers
inflate it); net all credits against expenses (payroll/refunds differ); aggregate
the visible page (misses facts); merge currencies or reference versions (assumes
valuation/equivalence); guess purpose from merchant (insufficient evidence); combine
posted/source values (mixes measures); inspect raw payloads/refetch (unreviewed data
access); add a model/service before deterministic provenance. Revisit only through
reviewed417/242/243 decisions and named counterexamples. No parent completion
from a contract merge, fixture arithmetic or a successful deployment.
