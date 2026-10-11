# Contract design source notes

Accessed 2026-10-11. These records extend the [bibliography](bibliography.md).
Proposals and test requirements are project inferences, not source requirements.

## C01 — PostgreSQL namespaces and privileges

PostgreSQL 17, [§5.10 Schemas](https://www.postgresql.org/docs/17/ddl-schemas.html).
Reviewed §§5.10.1–6: qualification, privileges and search-path trust. Supports
separate names/grants; does not prescribe our six schemas or tenant layout.
[ADR 0004](../decisions/0004-schema-ownership.md) records the chosen convention.
Also reviewed [SELECT locking clause](https://www.postgresql.org/docs/17/sql-select.html)
and retained S08/S12 isolation/function-security evidence. Row-locking reads need
careful privilege/transaction design; names alone provide no security proof.

## C02 — OpenAPI Specification 3.1.1

OpenAPI Initiative, [specification](https://spec.openapis.org/oas/v3.1.1.html).
Reviewed Schema Object §4.8.24 and Reference Object §4.8.23; consulted operation,
response and security object definitions for the draft interface. Describes HTTP
contracts with JSON Schema-compatible shapes. Our operation names, decimal-string
encoding, revision checks and endpoint scope are application choices. OpenAPI is
not an ISO financial-message schema or an authorization implementation.

## C03 — RFC 9457, Problem Details for HTTP APIs

IETF, [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457.html), 2023.
Reviewed §3 and §3.1 fields and §5 security considerations. Supports structured
HTTP errors with safe detail and extension fields. Our code vocabulary, request
IDs and HTTP status mapping are explicit project choices; avoid leaking internals.

## C04 — Public AASB 121 compilation incorporating IAS 21

Australian Accounting Standards Board,
[AASB 121 October 2023 compilation](https://standards.aasb.gov.au/aasb-121-oct-2023).
Reviewed comparison with IAS 21, operative-period metadata and paragraphs
21–23, 27–29. This compilation applies to periods beginning in 2025–2026.
An attempted older PDF URL was inaccessible; the official HTML was readable.

Evidence: initial transaction-date valuation; different subsequent treatment for
monetary, historical-cost and fair-value items; ordinary monetary exchange
differences in earnings with exceptions. These motivate the conditional FX
examples and separation of foreign quantity from functional carrying amount.
Limit: Australian normative source incorporating IAS 21, not US GAAP authority;
Australian-specific clauses and unreviewed exceptions are not adopted. Full
ASC 830 review is still needed for a US GAAP policy. No claim that reading this
subset establishes general IFRS/AASB compliance or covers consolidation/hedging.

## Review decisions and gaps

Numeric limits, rounding, function boundary and book-lock protocol are proposals
with named tests, not decisions dictated by literature. The owner accepted domain
schemas. Reporting basis, functional-currency policy and automated FX exceptions
remain unapproved. See [policy register](../contracts/accounting-scenarios.md).
