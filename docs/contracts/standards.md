# Standards applicability contract

## Decision

Owner requirement: every financial message, data contract and practice must use
applicable ISO standards. This is an accepted design requirement, not evidence
of implemented conformance. It covers imports, exports, APIs, internal events,
currency metadata and audit data. Accounting recognition and measurement still
require a selected accounting framework and applicable jurisdictional rules.

## Reviewed sources and applicability

| Source | Application and review boundary |
| --- | --- |
| [ISO 20022 FAQ](https://www.iso20022.org/faq), sections About and Compliance | Financial messaging methodology and repository. Implementation depends on community requirements; there is no official ISO 20022 certification authority. Public FAQ reviewed, not the purchased standard's eight parts. |
| [Official message catalogue](https://www.iso20022.org/catalogue-messages), documentation and versions | Select registered definitions, schemas, Message Definition Reports and applicable Message Usage Guides. Catalogue overview reviewed; individual integration definitions remain to be selected and read. |
| [ISO 4217 currency codes](https://www.iso.org/iso-4217-currency-codes.html) | Use maintained currency identifiers and minor-unit metadata, including historical validity. Public overview reviewed. Codes do not determine exchange rates, functional currency or accounting rounding policy. |
| [ISO 21378:2019](https://www.iso.org/standard/70823.html), abstract | Audit-data collection and accounting data elements are relevant to audit exports. Abstract/metadata reviewed; full normative text not reviewed. Field-level conformance remains unverified. |

This list is a starting register, not an exhaustive declaration that other ISO
standards are inapplicable. Each slice must search authoritative sources for
additional applicable standards, including identifiers and audit semantics.
Textbooks and arXiv explain mechanics; they do not override accounting standards.

## Required record before implementation

Each message or practice contract must name:

1. Business purpose, direction, counterparties and responsible component.
2. Applicable standard, edition/effective date, exact message identifier and
   version, namespace, code-list release and bank/community usage profile.
3. Reviewed source sections and access limits. A paywalled abstract cannot
   substantiate detailed normative requirements.
4. A field/meaning mapping to internal contracts, including amounts, currencies,
   debit/credit perspective, dates, identifiers, statuses and reversal semantics.
5. Required fields, validation rules, supported versions, extension handling,
   information-loss policy and migration/compatibility policy.
6. Named synthetic fixtures, expected outcomes and the review evidence needed
   to assert the precisely scoped conformance claim.

Record `applicable`, `not applicable` with justification, or `unresolved` for
each assessed standard. An unresolved required mapping blocks implementation
of that slice. Deviations require a documented reviewed decision; silence is
not an exception. Prefer current definitions where counterparties support them;
pin supported versions instead of silently upgrading to a moving latest version.

## Project design consequences

Financial exchange adapters must validate the selected schema and its business
rules/profile. Schema validity alone cannot establish bookkeeping correctness.
Unsupported versions and unmapped financially material fields must be rejected
or held for explicit review before posting; never silently discard them.

Non-ISO bank feeds need explicit, traceable mappings into the internal financial
contract. Custom JSON does not become a registered ISO message by sharing field
names. Internal notifications may use a local schema when no ISO definition
applies, with that assessment recorded. Transport encoding is selected by the
applicable definition/profile, not a blanket XML requirement for every job.

Preserve original message identity and mapping/version provenance under the
data-access and retention policy. Retries, ordering, duplicates, corrections and
authorization remain governed by the accounting and background-work contracts.
These controls are project requirements, not claims that ISO alone specifies
the implementation.

## Required acceptance evidence

| Evidence identifier | Required result before a supported integration ships |
| --- | --- |
| STD-01 applicability review | Every message/practice in the slice has a sourced assessment and resolved required mappings. |
| STD-02 schema/profile fixtures | Valid examples pass; malformed, wrong-version and profile-invalid examples fail. |
| STD-03 semantic mapping fixtures | Currency, decimal precision, debit/credit perspective, dates, status and correction meaning survive mapping; prohibited loss fails. |
| STD-04 replay/correction fixtures | Duplicate input has one authorized effect; correction retains provenance and cannot rewrite posted books. |
| STD-05 compatibility review | Supported versions, code-list updates and any deviations have explicit evidence and review. |

These are planned evidence items, not existing tests. No runtime financial
messages or ISO conformance have been implemented on this research branch.
