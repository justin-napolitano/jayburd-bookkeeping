# Annotated bibliography and reading guide

## Purpose and reading order

Shared resource index for the owner and agents, maintained with the contracts.
Updated 2026-10-11. Sources explain or constrain decisions; listing a source
does not approve a design or establish compliance.

Start with these six reading steps:

1. **Learn the books:** OpenStax accounting foundations and reconciliation (S01–S02).
2. **Understand financial interchange:** ISO 20022 FAQ and catalogue (R01–R02).
3. **Separate currency codes from FX accounting:** ISO 4217, then the applicable
   accounting framework's currency rules (R03, R06–R07).
4. **Design the data:** BCcampus relational design (S03), then audit-data
   definitions and semantic modelling (R04–R05).
5. **Protect posting and closing:** PostgreSQL constraints/isolation and the SSI
   paper (S06–S12).
6. **Handle asynchronous work:** transactional outbox and actual Cloud Tasks
   semantics (S13–S15), then migration/access tooling (S18–S20).

## How to read the review labels

- **Selected sections reviewed:** only the sections identified in the detailed
  source record were read; never infer a cover-to-cover review.
- **Overview / abstract / preview only:** useful for discovery and scope;
  insufficient evidence for unseen detailed requirements.
- **Pending:** a reading destination, not supporting evidence for a design claim.

The [database source register](database-sources.md) is the authoritative home for
S01–S22 citation metadata, exact sections, findings and limitations. The table
below is a navigation guide, not a second copy of those evidence records.

## Accounting, database and engineering resources

| ID and resource | What it is and why read it | Review status |
| --- | --- | --- |
| [S01 — OpenStax financial accounting](database-sources.md#s01--accounting-foundations) | Free textbook introduction to the accounting equation, journals and double-entry. Start here for the business vocabulary. | Selected sections reviewed |
| [S02 — OpenStax bank reconciliation](database-sources.md#s02--reconciliation) | Worked teaching material for comparing bank statements with book balances and identifying adjustments. | Selected sections reviewed |
| [S03 — Watt & Eng, Database Design, 2nd ed.](database-sources.md#s03--relational-design-textbook) | Open textbook on ER models, normalization and the design process; useful before drawing our schema. | Selected chapters reviewed |
| [S04 — Codd, relational model (1970)](database-sources.md#s04--relational-independence) | Foundational research explaining logical data independence; background for separating business models from storage choices. | Opening sections reviewed |
| [S05 — Ellerman, double-entry mathematics (2014)](database-sources.md#s05--double-entry-mathematics) | Mathematical account of debit/credit pairs and signed representations; helps reason about signs and reversals. | Pages 1–7 reviewed; not accounting-policy authority |
| [S06 — Ports & Grittner, PostgreSQL SSI (2012)](database-sources.md#s06--serializable-snapshot-isolation) | Research paper on serializable isolation, including a close/report anomaly relevant to period locking. | Overview and §§2–3 reviewed; older PostgreSQL implementation |
| [S07 — PostgreSQL constraints](database-sources.md#s07--database-constraints) | Official reference for CHECK, keys and references; identifies limits of row-local enforcement. | Selected PostgreSQL 17 sections reviewed |
| [S08 — PostgreSQL isolation](database-sources.md#s08--isolation-semantics) | Official behavior of snapshots, isolation levels and transaction retries. | Selected PostgreSQL 17 sections reviewed |
| [S09 — PostgreSQL row security](database-sources.md#s09--row-level-security) | Official RLS rules and bypass caveats; relevant to separate LLC books. | Selected PostgreSQL 17 sections reviewed |
| [S10 — PostgreSQL numeric types](database-sources.md#s10--exact-amounts) | Exact numeric storage, precision and special values; useful when specifying money validation. | Selected PostgreSQL 17 sections reviewed |
| [S11 — PostgreSQL triggers](database-sources.md#s11--deferred-enforcement) | Mechanism reference for a candidate deferred balance check. | Selected PostgreSQL 17 sections reviewed |
| [S12 — PostgreSQL function security](database-sources.md#s12--restricted-posting-functions) | Privilege and search-path rules for a candidate restricted posting function. | Selected PostgreSQL 17 sections reviewed |
| [S13 — Helland, Data on the Outside versus Data on the Inside (2005)](database-sources.md#s13--internal-state-versus-messages) | Paper about service data and transaction boundaries; useful before splitting services. | Introduction/§1 reviewed |
| [S14 — Richardson, transactional outbox](database-sources.md#s14--transactional-outbox) | Pattern description for committing business changes and message intent together. | Pattern sections reviewed |
| [S15 — Google Cloud Tasks pitfalls](database-sources.md#s15--actual-queue-semantics) | Provider documentation about duplicates and ordering; grounds retry assumptions in the planned transport. | Relevant sections reviewed; recheck at implementation |
| [S16 — Square Books (2019)](database-sources.md#s16--production-ledger-case-study) | Engineering case study of an immutable double-entry ledger; example rather than a universal blueprint. | Accounting/model sections reviewed |
| [S17 — Fowler, Event Sourcing (2005)](database-sources.md#s17--event-sourcing-distinction) | Pattern explanation of state reconstruction and replay costs; helps distinguish ledger history from full event sourcing. | Selected sections reviewed |
| [S18 — Sadalage & Fowler, Evolutionary Database Design](database-sources.md#s18--evolving-schemas) | Design article about incremental schema changes and coordination with application releases. | Selected sections reviewed |
| [S19 — sqlc documentation](database-sources.md#s19--sql-generated-access) | Tool reference for generating Go access from SQL and binding queries to transactions. | Overview and versioned transaction guide reviewed |
| [S20 — golang-migrate documentation](database-sources.md#s20--migration-tool-guarantees) | Owner-selected migration tool's format and failure-handling reference. | v4.20.1 format/FAQ sections reviewed |
| [S21 — Kleppmann & Riccomini, Designing Data-Intensive Applications, 2nd ed.](database-sources.md#s21--additional-book-preview-only) | Broader systems textbook for further study of distributed-data tradeoffs. | Publisher introduction/contents preview only |
| [S22 — GORM documentation](database-sources.md#s22--orm-counterexample) | Alternative-tool reference separating explicit transactions from automatic schema migration. | Transaction/migration pages reviewed |

## Standards and multicurrency resources

These R entries are the canonical bibliographic records for this additional
reading set. Access/review date: 2026-10-11. Full normative texts and applicable
amendments must be reviewed before claiming detailed conformance.

### R01 — ISO 20022 FAQ

**Publisher/link:** ISO 20022 Registration Authority,
[Frequently asked questions](https://www.iso20022.org/faq).
**What it is:** official orientation to the financial-message methodology,
repository and implementation context. **Use:** understand what an ISO 20022
claim means before designing integrations. **Reviewed:** About and Compliance
sections of the public FAQ. **Limit:** not a review of the eight normative parts
or a particular bank's usage requirements.

### R02 — ISO 20022 message catalogue

**Publisher/link:** ISO 20022 Registration Authority,
[Catalogue of messages](https://www.iso20022.org/catalogue-messages).
**What it is:** entry point to registered definitions, schemas, Message Definition
Reports and available usage guides. **Use:** select exact versions and obtain
the artifacts required by an integration contract. **Reviewed:** catalogue
documentation/version overview. **Pending:** individual messages and bank profiles;
no implementation version has been selected.

### R03 — ISO 4217 currency codes

**Publisher/link:** ISO, [Currency codes](https://www.iso.org/iso-4217-currency-codes.html).
**What it is:** official overview of currency identifiers, minor units and their
maintenance. **Use:** currency reference-data design and a route to the designated
maintenance agency. **Reviewed:** public overview. **Limit:** neither an exchange-rate
feed nor a source of recognition, remeasurement or rounding policy; the actual
data release used by software must be recorded separately.

### R04 — ISO 21378:2019, Audit data collection

**Publisher/link:** ISO, [standard page](https://www.iso.org/standard/70823.html).
**What it is:** common accounting data definitions for audit extraction across
ledger and related business areas. **Use:** assess accountant/auditor export
mappings. **Reviewed:** public abstract and edition metadata; the page records
confirmation in 2025. **Limit:** full standard not reviewed; field-level requirements
and mappings remain pending.

### R05 — ISO 21926:2026, Semantic data model for audit data services

**Publisher/link:** ISO, [official browsing preview](https://www.iso.org/obp/ui?_escaped_fragment_=iso%3Astd%3Aiso%3A21926%3Aed-1%3Av1%3Aen).
**What it is:** a framework for modelling accounting/audit semantics and binding
them to exchange formats, building on audit-data definitions. **Use:** evaluate
our shared glossary, semantic contracts and export mappings. **Reviewed:** contents,
introduction and §1 Scope. **Limit:** detailed modelling clauses and annexes remain
unreviewed; it does not supply a complete physical ledger schema or implementation.

### R06 — FASB Accounting Standards Codification, including Topic 830

**Publisher/link:** Financial Accounting Standards Board,
[Accounting Standards Codification](https://asc.fasb.org/).
**What it is:** the US GAAP codification; Topic 830 is the foreign-currency research
target. **Use:** resolve US GAAP policy and exact paragraph-level requirements if
that reporting basis is selected. **Status: pending normative review.** The site
returned no readable body in this research tool. Do not cite unseen paragraphs
or substitute an accounting firm's interpretation for the authoritative text.
Applicable version, amendments and effective dates still need recording.

### R07 — IAS 21, The Effects of Changes in Foreign Exchange Rates

**Publisher/link:** IFRS Foundation,
[IAS 21 official overview](https://www.ifrs.org/issued-standards/list-of-standards/ias-21-the-effects-of-changes-in-foreign-exchange-rates/).
**What it is:** IFRS guidance concerning foreign-currency transactions, functional
currency and financial-statement translation. **Use:** research the IFRS currency
model and distinguish transaction, functional and presentation currencies.
**Reviewed:** About and standard-history overview only. **Limit:** full operative
paragraphs and effective amendments remain pending; IFRS must not silently become
the policy for a US GAAP book.

## Testing resources

The [synthetic accounting resource register](testing-sources.md) describes T01
Beancount, T02 hledger, T03 ISO 20022 artifacts and T04 the MIT synthetic
accounting generator, with pins, review depth, rights and observed limitations.

The [posting policy source notes](posting-sources.md) add P01 accrual mechanics
and the US GAAP authority gap, P02 numeric rounding, P03 restricted functions and
P04 strict JSON/replay handling, P05 pinned pgx transactions and P06 reference
version serialization. Read these before the first posting implementation.

The [identity source notes](identity-sources.md) add I01–I05: managed ID-token
verification, revocation, TOTP MFA the pinned Firebase Admin Go API and dependency maintenance evidence. These
support the identity/setup contract; live deployment evidence remains separate.

## Cloud delivery resources

The [GCP source register](gcp-sources.md) describes managed request billing,
Cloud Tasks pricing and duplicate delivery, Cloud SQL connectivity and IAM
authentication, Terraform mock tests, state bootstrap and GitHub federation. Read its section-level review limits before implementing cloud
delivery; provider documentation and budget assumptions are not deployment proof.

## Agent maintenance rules

The first application adds three source registers: [server composition](server-sources.md)
for bounded HTTP, workload identity and PostgreSQL pooling;
[browser authentication](browser-auth-sources.md) for memory-only sign-in and
TOTP enrollment/challenge; and [frontend behavior](frontend-sources.md) for
Next.js, React and synthetic browser testing. The [integration source notes](app-integration-sources.md)
record generated OpenAPI types, Node LTS and CI choices. Read their stated review
depth and limits; offline tests do not verify live provider configuration.

The [contract-design source notes](contract-sources.md) add C01 (PostgreSQL
schemas/locking), C02 (OpenAPI 3.1.1), C03 (RFC 9457 HTTP problems) and C04
(public AASB 121 currency guidance). Read them alongside the SQL/API contracts.
C04 is a limited paragraph review of an Australian source incorporating IAS 21,
not a substitute for pending US GAAP research.

Use stable S/R identifiers when linking a source to an ADR or contract. Read the
underlying section before relying on it, and update its canonical record with
title, publisher, version, section/pages, access date, relevance and limitations.
Keep discovery separate from reviewed evidence. Cite direct authoritative links
in decisions; this index is a navigation aid, not a substitute for the source.
Do not mirror copyrighted texts or infer approval from a bibliography entry.

Apply the [decision evidence standard](decision-evidence.md) and
[standards applicability contract](../contracts/standards.md). Maintain these
records in the same change as a source-dependent decision. Unresolved FX policy,
message-profile selection and export mappings remain research work; this guide
does not declare them complete.
