# PFC normalization source review —413

Public official sections reviewed2026-11-06; no provider endpoint, private payload,
bank statement, secret or retained historical evidence was accessed. This source
record supplements CR04/07, without claiming full provider documentation review.

| Source / exact selected sections | Established fact | Jayburd inference / limit |
| --- | --- | --- |
| [Transactions API](https://plaid.com/docs/api/products/transactions/), sync request options.personal_finance_category_version and response personal_finance_category primary/detailed/confidence_level/version | PFC is nullable; response version distinguishes v1/v2. Confidence is vendor metadata. | Preserve exact bounded attribution; never infer version from API2020-09-14 or turn confidence into a Jayburd probability. |
| [PFC migration](https://plaid.com/docs/transactions/pfc-migration/), migration opt-in/default and historical backfill sections | Taxonomy defaults depend on enablement timing; explicit options select v2. Historical taxonomy retrieval does not backfill prior confidence/accuracy improvements. | No automatic request change, historical version inference or refetch. Unknowns abstain under an explicit map edition. |

Actual source inspection: account-filtered HTTP options; DecodePage and Transaction
JSON; runner's pageWire marshal; immutable SQL25/29 closed normalized key sets and
provider-field equality. The sidecar keeps that legacy wire unchanged. The source
sections do not decide Jayburd topic identity, owner grants, revision equality or
accounting policy. Those are independently reviewed project contracts.
