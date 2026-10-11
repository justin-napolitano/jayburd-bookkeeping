# Posting policy and boundary sources

Reviewed 2026-10-11. These extend the [bibliography](bibliography.md),
[database sources](database-sources.md) and [contract sources](contract-sources.md).
The normative project choices are in [ADR 0005](../decisions/0005-development-accounting-policy.md)
and the [posting boundary](../contracts/posting-boundary.md).

## P01 — Accrual mechanics and authority boundary

OpenStax, *Principles of Accounting, Volume 1: Financial Accounting*,
[§4.1, Accrual Accounting](https://openstax.org/books/principles-financial-accounting/pages/4-1-explain-the-concepts-and-guidelines-affecting-adjusting-entries).
Reviewed the accrual-versus-cash discussion: recognition timing differs from cash
movement. This supports using a separately recorded accrual and settlement in
AC-10. It is introductory textbook guidance, not a complete revenue-recognition,
LLC tax or foreign-currency policy. Our service-LLC development workflow is an
engineering scope choice.

FASB [standards portal](https://fasb.org/standards?locale=en-US) identifies the
Codification as the authoritative nongovernmental US GAAP source in public search
metadata; direct browser extraction was effectively empty. **Discovery only**:
no ASC operative paragraphs were reviewed in this session. US GAAP remains a
research target; do not upgrade C04's AASB/IAS examples into US GAAP authority.

## P02 — Exact amounts and rounding

PostgreSQL 17 [§8.1.2 Arbitrary Precision Numbers](https://www.postgresql.org/docs/17/datatype-numeric.html#DATATYPE-NUMERIC-DECIMAL).
Reviewed exact numeric behavior, scale coercion, special values and tie rounding.
Numeric typmod casts can round before constraints inspect the original input;
built-in numeric rounding breaks ties away from zero. Hence our inference: validate
original text first and implement the selected half-even rule explicitly in the
owned database operation. The precision bounds and half-even choice are ours,
not prescribed by PostgreSQL, ISO 4217 or this accounting textbook.

## P03 — Restricted functions and transactions

PostgreSQL 17 [CREATE FUNCTION, Writing SECURITY DEFINER Functions Safely](https://www.postgresql.org/docs/17/sql-createfunction.html#SQL-CREATEFUNCTION-SECURITY),
and [§13.2.3 Serializable Isolation](https://www.postgresql.org/docs/17/transaction-iso.html#XACT-SERIALIZABLE).
Reviewed secure search path, public execution defaults, atomic privilege setup
and whole-transaction retry requirements. These support the restricted operation
boundary and retries; they do not prove our particular lock order, identity trust
or concurrency schedules. Go-with-triggers remains an alternative but would need
a separately demonstrated complete mutation protocol. Our choice favors one
reviewable database authority for the initial journal invariant.

## P04 — JSON input and replay equality

PostgreSQL 17 [§9.16.1, Table 9.48](https://www.postgresql.org/docs/17/functions-json.html)
(`IS JSON ... WITH UNIQUE KEYS`) and jsonb comparison operators; reviewed the
predicate's nested-key behavior and object/array facilities. Validate unique keys
on text before jsonb, then reject unsupported fields and normalize only defined
values. Our canonical request version is a local contract, not RFC 8785 or an ISO
financial message. Persisting normalized input and comparing it exactly is our
choice; no claim that arbitrary jsonb serialization is an interchange standard.

Also reviewed PostgreSQL 17 [§9.5 `sha256(bytea)`](https://www.postgresql.org/docs/17/functions-binarystring.html):
core digest computation needs no extension. The digest is diagnostic; stored
normalized-request equality owns replay. Upgrade tests must preserve historical
receipt interpretation even when serializer output changes.

## Applicability and limitations

Manual posting/reversal/read APIs and identifier-only commit notifications retain
the [existing ISO assessment](../contracts/api.md#operations-and-permissions).
Currency versions use ISO 4217 identifiers. No new bank message or external audit
export is introduced here. ISO schema provenance work in #17 neither approves nor
blocks this internal synthetic operation. Real financial exchange requires its
own completed mapping/profile gate. No source above certifies accounting software.
