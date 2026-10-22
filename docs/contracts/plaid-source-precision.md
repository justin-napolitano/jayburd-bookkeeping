# Source precision hold — issue #258

## Evidence and decision (2026-10-22)

Hosted Sandbox generation 2 failed with the fixed code
`currency_precision_exceeded`, before any checking publication. No amount or raw
provider payload was inspected. This establishes the validation branch, not the
institution's intended value or a floating point artifact. The owner requested a
shared repair across account types.

The [Transactions API](https://plaid.com/docs/api/products/transactions/), selected
`/transactions/sync` transaction amount/currency sections, describes a numeric
settled amount and ISO currency identifier; its `double` transport description
does not authorize rounding imported evidence. Selected public sections reviewed
on 2026-10-22; no complete schema or bank guarantee is claimed. PI01 in the
[source register](../research/plaid-ingestion-sources.md) is the canonical source.

Decision: retain exact source amounts within existing numeric(38,9) bounds even
when they exceed the dated currency's minor-unit precision. Such rows have
`eligible_for_review=false`, enter `correction_hold` and expose
`source_precision_review_required`. Other rows publish in the same atomic set.
Original signed number lexemes and exact unsigned values remain unchanged.
Pending and zero remain held. Over-nine-decimal or magnitude overflow still
quarantine the complete page; this slice does not claim arbitrary precision.

Alternatives: whole-page rejection blocks unrelated transactions (current
defect); rounding loses evidence; dropping the row loses completeness; arbitrary
precision requires a separate storage contract. No account-specific exception.

## Applicability and mapping

This changes proprietary Plaid JSON to internal observation/review semantics,
profile `jayburd-plaid-transactions/1`, API pin `2020-09-14`. ISO 20022 is not a
counterparty exchange here; the existing documented adapter exception applies.
ISO 4217 identifiers, historical versions and minor-unit metadata remain
applicable. Minor units govern posting eligibility, not lossy evidence storage.
ISO 21378:2019 remains relevant to future audit exports; no export or conformance
claim is added and full normative text has not been reviewed. See
[standards](standards.md) for authoritative sources/access limits.

Amount/direction/date/currency/identity mappings are unchanged. SQL independently
checks exact lexeme equality, dated currency and eligibility; it derives the hold
from exact quantity and currency precision. No new transport field, nullable
money, float, new currency reference or ledger rounding policy. No auto-posting.

## Acceptance and compatibility

Named evidence: seven-account, three-currency decoder matrix; independent
rational fuzz oracle; SQL mixed valid/held publication, exact replay, forged
eligibility rejection, draft refusal and zero journal assertions; review browser
test showing all digits and disabled drafting for the held row. Migration 29
precedes the new worker. Old worker inputs remain valid. Down migration refuses
retained sub-minor-unit provider evidence; no deletion or rounding. Existing
posted journals and ledger valuation constraints remain unchanged.
