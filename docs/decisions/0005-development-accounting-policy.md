# ADR 0005: First development accounting policy

## Decision, owner and scope

Owner requested the proposed accrual/USD posting sequence on 2026-10-11. Adopt
`dev-accrual-usd-v1` as the concrete **synthetic development profile**, subject to
this PR's review. It is not a real LLC's approved reporting policy or tax election.
US GAAP is the intended research target; do not label these books GAAP-compliant.
ASC topic review and real-entity facts remain required before official use.

First workflow: a fictional service LLC receives owner capital, earns service
income, pays expenses, uses a card, transfers cash and records manual accruals and
corrections. Use AC-01–10 and a separate second LLC for isolation. The model and
posting interface retain foreign currency and support reviewed spot conversion;
full foreign receivable settlement/remeasurement follows its own contracts.

## Profile

| Decision | Selected development rule |
| --- | --- |
| Basis | Accrual; transaction recognition is a reviewed manual decision, not inferred from bank-feed dates |
| Functional currency | USD; source currency metadata is versioned, with EUR fixtures for foreign transactions |
| Calendar | UTC, explicit accounting date, Gregorian monthly half-open periods; January `[2026-01-01, 2026-02-01)` in the proof |
| Opening | Original AC-01 synthetic capital journal; no inferred live opening equity or imported personal data |
| Chart | Asset/liability/equity/revenue/expense; explicit normal side and monetary flag; original fixture accounts; no automated subledger controls yet |
| Precision | Storage amounts 38 digits/9 fractional; rates 38/18; USD functional posting scale 2; original amount must be an exact multiple of selected currency minor unit, no silent rounding |
| Rounding | Exact product, round once to functional scale using ties-to-even; SQL is authoritative; client functional amount is an assertion |
| Rate policy | `synthetic-reviewed-spot-v1`: immutable fixture observation, base transaction/quote USD, purpose spot, UTC effective date equals accounting date; source explicitly synthetic; no live provider/fallback/inversion |
| Corrections | Exact linked reversal into an open period; original values/rates preserved; no deletion or reversal-of-reversal |
| Closing | Closed periods reject new work; no user-facing close operation until close-evidence contract exists; test closed state through controlled fixture setup |
| Permissions | Separate read/prepare/post/close memberships; post does not imply read, prepare cannot post; setup belongs to controlled migration/test identity |

Amounts that round to zero, overflow, invalid minor-unit precision and any
functional mismatch fail. A journal must balance exactly after per-line rounding.
No automatic rounding plug or tolerance is enabled. USD 1.005 rounds to 1.00;
1.015 rounds to 1.02; 2.675 rounds to 2.68. These are conversion-result vectors,
not permission to post three-decimal USD original amounts. A genuine rounding
adjustment needs a separately reviewed operation; the generator's one-cent errors
must continue to fail.

## Evidence and alternatives

[P01–P04](../research/posting-sources.md) review accrual mechanics, PostgreSQL
rounding, restricted functions and JSON validation. These sources establish
mechanisms, not this policy's approval. Half-even, the same-day synthetic rate
restriction and the bounds are project choices. C04's IAS-21-derived examples
remain illustrations, not a substitute for ASC 830.

Cash basis is simpler for some workflows but would not exercise AC-10's accrual
and settlement separation. Integer minor units are viable but add currency-scale
handling to every conversion; bounded exact numeric keeps the selected model
explicit. Half-up is viable, but half-even makes tie behavior explicit and avoids
always rounding positive ties upward. Broader rate-provider rules are deferred
until provider coverage, timestamps and exception policies can be reviewed.

## Done when and follow-up

The [posting boundary and acceptance matrix](../contracts/posting-boundary.md)
define the next implementation gate. Tests must prove rounding, balance, replay,
isolation and authorization; a document does not implement them.

Keep #1 open for a named real entity, accounting timezone, complete opening
trial balance, accountant export/review and cutover acceptance. Tax basis,
revenue-recognition automation, annual closing, currency transitions and foreign
position accounting are not selected by this development profile. Profile changes
require a new policy revision; never reinterpret posted history.
