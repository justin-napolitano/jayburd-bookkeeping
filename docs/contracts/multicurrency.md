# Money and multicurrency contract

## Status and source boundary

Representation and operation requirements for JB-002. The selected synthetic
development profile is [ADR 0005](../decisions/0005-development-accounting-policy.md),
with exact rounding and replay details in [posting boundary](posting-boundary.md). ISO 4217 identifies
currencies; it does not select the accounting framework. See
[contract sources](../research/contract-sources.md) C04 and bibliography R03/R06/R07.
The FX examples are conditional accounting illustrations; US GAAP paragraph-level
review remains outstanding. This contract does not assert GAAP/IFRS compliance.

## Representations

Keep transaction currency, book functional currency and presentation currency
separate. Functional currency belongs to an approved book policy revision. Changing
it requires a dedicated transition contract; never reinterpret historical journals.

Selected storage bounds for the first posting slice: finite `numeric(38,9)`, maximum 29 integer digits; rates:
finite positive `numeric(38,18)`, maximum 20 integer digits. Validate lexical scale
and range BEFORE casting to these types, since PostgreSQL can round on coercion.
These bounds are engineering choices, not currency-standard requirements.
HTTP money/rates are unsigned plain-decimal strings, no exponent or grouping.
Reject negative values, NaN, infinity and excess precision. Side conveys sign.

Currency versions record ISO code, numeric code when assigned, minor-unit metadata,
validity interval and source release. Non-currency assets need a separate future
unit contract. Currency validity and posting precision are distinct policies;
never assume all currencies use two decimals. Proposed functional posting scale
is the selected currency's minor unit, bounded by storage precision, with half-even
rounding. ADR 0005 selects this for the synthetic profile; real-entity exceptions still
require explicit policy.

## Line semantics

`transaction_amount` is the original currency magnitude; `functional_amount` is
its booked magnitude. Ordinary lines require both positive. A valuation line may
have zero transaction amount and positive functional amount, keeping the original
foreign exposure currency. Small real exposures whose valuation rounds to zero
require a reviewed precision exception; do not silently drop them in this slice.

Every line has a valuation basis: `spot`, `carrying`, `functional_adjustment` or
`identity`. Identity requires matching currencies and amounts. Spot requires an
immutable rate observation and recomputation at approved rounding. Carrying needs
a source position/allocation, never an arbitrary client exchange rate. Adjustments
require a reviewed valuation/correction reference and privileged operation kind.
All journal functional debits equal functional credits exactly. Raw currency
buckets are never summed together to test balance. Ordinary API posting initially
accepts only identity/spot lines; specialized operations own carrying/adjustment
lines, and those operations remain blocked until their full contracts are accepted.

Rate observations record base/quote currencies, quote-units per base-unit, rate,
effective time, observation time, provider/reference, purpose and policy revision.
For a transaction-to-functional conversion, base is transaction currency and quote
is functional currency. No guessed inversion, fallback to 1 or replacement of
historical observations. Missing/stale/nonexchangeable rates block automatic posting
and require an explicit reviewed valuation method. Store selected provenance on
posted lines; later provider corrections create new observations.

## Measurement and settlement requirements

Under the illustrative IAS-21-derived policy, monetary exposure uses period-end
rates; historical-cost nonmonetary items retain transaction-date valuation; fair
value items use the relevant valuation-date rate. Ordinary monetary exchange
differences affect earnings, with exceptions requiring separate policy. See C04.

Project operations must retain foreign quantity and functional carrying amount
per identified position. Revaluation posts only the difference from current
carrying value, not repeatedly from initial cost. Settlement allocations remove
foreign quantity and its associated carrying value; consideration and fees are
separate, with the balancing FX difference explicit. Partial allocation rounds
under one policy; the final allocation consumes the exact residual. A rerun of a
valuation/settlement operation cannot duplicate its effect. Concurrent allocations
must not exceed the open position. See [FX-01–FX-06](accounting-scenarios.md).

Reversing a posted journal swaps sides while preserving original amounts/rates;
it is a correction, not a new conversion at today's rate. Position-affecting
reversals also need inverse allocation effects and are outside the first manual
posting slice. FX profit/loss and presentation-translation reserves remain distinct.
Explicit rounding lines use an approved account and bounded tolerance; never
silently alter a user line or manufacture a balancing amount for arbitrary errors.

## Acceptance and remaining gates

`TestMoneyInputBounds` covers zero/2/3-decimal policies, half-way rounding, overflow,
scale, nonfinite input and identity rules. `TestFXLifecycle` covers FX-01–FX-06,
rate direction/staleness, period changes and reruns. `TestSettlementAllocationRace`
covers concurrent partial settlements and final residuals. Manual bounds/spot tests now execute against PostgreSQL; full FX lifecycle
and settlement allocation tests remain future work. See the
[posting proof](../runbooks/posting-proof.md).

Before automated settlement/revaluation, specify position/allocation physical
columns, operation permissions, valuation batches and reversal semantics. For the first manual posting implementation, use only ADR 0005
and its same-day synthetic rate policy; live rate-provider policy remains open. This deliberate boundary prevents generic journal creation from
pretending to implement receivable subledgers or the full currency lifecycle.
