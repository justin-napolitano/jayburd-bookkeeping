# Financial reporting sources

## Scope and review

Reviewed 2026-10-13 for #51, beginning with the shared
[bibliography](bibliography.md) and existing policy/standards contracts. These
selected sections support the [proposed contract](../contracts/financial-reporting.md),
not live accounting approval. No textbook reproduction or normative certification.

## FR01 — Introductory statement relationships

OpenStax / Rice University, *Principles of Accounting, Volume 1: Financial
Accounting*, §2.3 [Prepare an Income Statement, Statement of Owner's Equity,
and Balance Sheet](https://openstax.org/books/principles-financial-accounting/pages/2-3-prepare-an-income-statement-statement-of-owners-equity-and-balance-sheet).
Reviewed the income statement, owner's equity, balance sheet and connecting
statements subsections. The worked example distinguishes a period's performance
from a dated financial position and connects earnings/distributions to equity.
**Inference:** our review application should expose these relationships and
separate owner movements from expenses. Five-category unclassified statements
and `unclosed_earnings` are project choices, not prescribed classifications.
**Limit:** introductory sole-proprietor illustration; not US GAAP paragraph-level
research or a complete LLC reporting framework. Existing corpus constants are
original fixtures, not copied textbook amounts. Follow-up: #1/#10 real policy.

## FR02 — Snapshot, locking and query mode

PostgreSQL Global Development Group, PostgreSQL **17**,
[§13.2.2 Repeatable Read Isolation Level](https://www.postgresql.org/docs/17/transaction-iso.html#XACT-REPEATABLE-READ)
and [SET TRANSACTION, Description](https://www.postgresql.org/docs/17/sql-set-transaction.html).
Also reviewed the PostgreSQL 17 maintenance branch's
[`execMain.c`](https://raw.githubusercontent.com/postgres/postgres/REL_17_STABLE/src/backend/executor/execMain.c),
`standard_ExecutorStart` row-mark handling and `ExecCheckXactReadOnly` permission
check, for the row-lock/read-only incompatibility (moving branch, not a deployed
server binary attestation).
Reviewed repeatable-read visibility, lock conflict/retry and read-only command
restrictions. A transaction snapshot persists within that transaction; a later
HTTP request does not inherit it. Read-only mode prohibits row-locking SELECT.
**Inference:** preserve existing authorization `FOR SHARE` locks in bounded
repeatable-read transactions, and reject changed revision on later requests.
That rejection protocol is our design, not a PostgreSQL feature. Alternative:
long-lived snapshots/materialized reports, deferred pending measurements.
**Limit:** documentation does not prove our role grants, cursor protocol or
performance; named synthetic authorization/concurrency tests remain required.

## FR03 — CSV formula hazards

OWASP Foundation community, [CSV Injection](https://community.owasp.org/attacks/CSV_Injection),
current page, reviewed hazard characters, escaping, Excel save/reopen caveats and
cross-application limitations. Formula execution depends on spreadsheet behavior;
quoting alone is insufficient and transformations can alter exported data.
**Inference:** initial review CSV omits arbitrary text and validates generated
field grammars; signed amounts become magnitude plus explicit sign. Exact JSON retains
rich text and lossless decimal meaning. **Alternative:** descriptive CSV with
spreadsheet-specific escaping, deferred until target compatibility is tested.
**Limit:** no universal safety guarantee for downstream edits or spreadsheet
numeric precision; OWASP is security guidance, not an accounting export standard.

## FR04 — Standards scope and limitations

ISO 20022 Registration Authority, [FAQ](https://www.iso20022.org/faq), About and
Compliance sections, and [Catalogue of messages](https://www.iso20022.org/catalogue-messages),
introductory catalogue/version description reviewed. Community usage requirements
matter; a custom report API is not a registered message by naming its fields.
**Inference:** application-owned query responses have no selected bank exchange
profile; reassess when an actual exchange/counterparty exists. Individual report
message definitions were not exhaustively reviewed; no universal non-applicability
claim. No purchased eight-part text reviewed.

ISO, [ISO 4217 currency codes overview](https://www.iso.org/iso-4217-currency-codes.html),
public code/minor-unit overview reviewed; reuse the existing pinned currency
reference model. This overview supplies no FX valuation policy or new code-list
release approval. See bibliography R03 for canonical currency source context.

ISO, [ISO 21378:2019, Audit data collection](https://www.iso.org/standard/70823.html),
first edition metadata and abstract reviewed; confirmed in 2025. Accounting data
extraction is relevant to future accountant/audit exports. **Limit:** normative
field definitions not accessed, so field-level mapping remains unresolved.
Bibliography R05's ISO 21926:2026 preview remains the semantic-model research
entry; no additional normative clauses reviewed here. Promotion to audit export
requires those mappings and scoped validation, not just a CSV header.

## FR05 — Transport dates and CSV structure

IETF, [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339.html), §5.6 Internet
Date/Time Format and §5.7 Restrictions; [RFC 4180](https://www.rfc-editor.org/rfc/rfc4180.html),
§2 Definition of CSV Format. Selected sections reviewed. Use valid full dates,
UTC timestamps, fixed field counts and escaped quoted CSV fields. Project choices
further restrict date range, timestamp generation and allowed field grammar.
**Limit:** IETF transport specifications do not define accounting dates, recognition,
authorization or spreadsheet formula safety. No ISO financial message designation
follows from RFC-compatible strings.
