# Reviewed synthetic demo sample

The integration-only seeder uses four balanced journals from T04's seed-42 smoke
baseline at commit `b0f0e34ade2b18995252837e3011f3a89eb1ca52`:
`JE000000001` through `JE000000004`, lines `JL000000001`–`JL000000010`.
See the [annotated source register](../../../../docs/research/testing-sources.md#t04--synthetic-accounting-data-generator).
The original 115-journal dataset has 15 journals with one-cent imbalances.
Those remain rejected, never silently repaired or plugged. This demo is a
reviewed four-journal subset, not a complete import or the generator's claimed
validation result. No Python tool or external download runs at demo startup.

Original EUR quantities and account codes are preserved. Names identify a
fictional review LLC. The source January 2023 dates move to January 2026 for
the development book. Controlled setup currently permits USD functional books,
so explicit synthetic spot observations use **1 USD per EUR** on each date.
That is a fictional test assumption, not a historical rate or currency relabeling.
VAT labels preserve source semantics; they do not approve any LLC tax treatment.

Independent fixed debit-positive balances are bank `5432.29`, receivables and
payables `0`, input VAT `50.61`, output VAT `-917.95`, revenue `-4831.31`, and
materials expense `266.36`. EUR original-unit and USD functional balances are
both verified against these values after SQL posting.

A separate original Cents Lab LLC generates USD deposits of `0.10` and `0.20`
and a `0.01` fee. Fixed expectations are bank `0.29`, capital `-0.30`, fees
`0.01`. All values remain decimal strings; validation uses exact rational
arithmetic. `TestDemoCentsOracle` rejects one-cent, fractional-cent and expected-
balance corruption. Seed failure prevents serving the demo.

The subset is derived from an MIT-licensed synthetic generator. Its license
notice is retained in [LICENSE.generator](LICENSE.generator).
