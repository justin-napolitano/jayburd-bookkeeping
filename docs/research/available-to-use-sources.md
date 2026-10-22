# Available-to-use source record

Reviewed 2026-10-21 for proposed #251. This record uses the existing annotated
[bibliography](bibliography.md), [decision-evidence requirements](decision-evidence.md)
and linked canonical records. It records selected-section review, not a new
normative conformance review. Project rules and proposals are labeled inference.

| ID | Source and reviewed portion | Fact supported; limits |
| --- | --- | --- |
| ATU-01 | OpenStax *Financial Accounting*, accounting equation/journals and cash/accrual discussion, as recorded at [S01](database-sources.md#s01--accounting-foundations); selected sections in existing register | Ledger balances and accrued obligations are different concepts; does not define discretionary cash or this Plan formula. |
| ATU-02 | OpenStax *Financial Accounting*, bank reconciliation section, [S02](database-sources.md#s02--reconciliation); selected teaching section | Bank observations and book balances require explicit reconciliation; no claim that a provider balance equals spendable cash. |
| ATU-03 | [Financial reporting contract](../contracts/financial-reporting.md), scope/dates/snapshot and amount semantics; repository design, not external authority | Existing proposal supplies book revision, date, exact decimal and authorization boundaries. #248 may refine Position; reporting design does not approve this Plan metric. |
| ATU-04 | [ISO 20022 FAQ](https://www.iso20022.org/faq), About and Compliance, and [message catalogue](https://www.iso20022.org/catalogue-messages), overview/version navigation; [R01](bibliography.md#r01--iso-20022-faq)–[R02](bibliography.md#r02--iso-20022-message-catalogue) | Registered financial-message use needs a selected definition and profile. No Plan message was selected; normative eight-part text and payment profile not reviewed. |
| ATU-05 | [ISO 4217 overview](https://www.iso.org/iso-4217-currency-codes.html), public overview, [R03](bibliography.md#r03--iso-4217-currency-codes); [multicurrency contract](../contracts/multicurrency.md) representation | Currency codes and minor units are reference metadata, not future FX rates or a cash policy. Release-specific production mapping still requires verification. |
| ATU-06 | [ISO 21378:2019](https://www.iso.org/standard/70823.html), abstract/edition, [R04](bibliography.md#r04--iso-213782019-audit-data-collection); [ISO 21926:2026 preview](https://www.iso.org/obp/ui?_escaped_fragment_=iso%3Astd%3Aiso%3A21926%3Aed-1%3Av1%3Aen), contents/introduction/§1, [R05](bibliography.md#r05--iso-219262026-semantic-data-model-for-audit-data-services) | Audit data and semantic mapping are relevant to future exports. No field-level clauses or full text reviewed, so no Plan export mapping or conformance claim. |
| ATU-07 | [Identity/setup contract](../contracts/identity-setup.md), authentication and authorization sections; repository design | Verified issuer/subject maps to a principal; email and caller IDs cannot confer authority. Future service identity/execution is a separate decision under #56. |

## Interpretation and alternatives

Accounting and reconciliation sources support keeping posted balances separate
from source observations and projections. They do **not** endorse a universal
“available to use” figure. The dated minimum-curve model, overlap rule, reserve
floor and uncertainty behavior are project design proposals. The simpler
cash-minus-reserve model hides due-date shortfalls; the dated model costs more
input review. This tradeoff is for owner/accountant decision, not source fact.

## Gaps and acceptance

No real-entity accounting, legal restriction, debt/tax schedule, forecast
accuracy, data-freshness SLA, service authority, or payment usage profile was
reviewed. Do not infer these from the synthetic development policy. Acceptance
requires owner/accountant review of the contract's questions, fixed independent
worked cases, #248 Position, #1 policy, #56 permissions, and the named tests in
the contract before implementation. A future export or payment workflow must
refresh the applicable standards source review and pin versions/profiles.
