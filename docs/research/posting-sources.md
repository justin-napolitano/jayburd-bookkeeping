# Posting policy and boundary sources

## P08 — Owner artifact policy and existing atomic source linkage (2026-11-08)

Owner decision473 is the source of the no-manual-entry product requirement;
it is not attributed to an ISO or accounting-standard paragraph. Reviewed actual
repository contracts source-review-sql.md/command-recovery.md and implementation
000011_source_review.up.sql, banking.source_post_authorize/execute_source_post:
the owned retained review/draft/revision tuple is checked before accounting, and
journal+source link+projection+receipt share one transaction. This establishes
an implemented mechanism to test, not independent correctness evidence.

Reviewed [ISO21378:2019 public abstract/metadata](https://www.iso.org/standard/70823.html):
audit-data collection/accounting data-element semantics are relevant to later
artifact/export mapping. No paid normative text or field-level conformity is
claimed. Reviewed [ISO20022 FAQ, About/Compliance](https://www.iso20022.org/faq):
message adoption depends on a specific usage profile. Current internal command
restriction introduces no new registered financial exchange message; existing
ISO4217 original-unit handling is unchanged. Document ingestion has its own
unresolved mapping/access/retention gate and is not advertised as implemented.

Chosen boundary: public application Handler always requires evidence-aware
registration/execution; retired direct manual/reversal routes return a fixed
authenticated rejection. Prepared legacy manual execution checks current pending
inside the same serializable transaction. Hiding only the button is inadequate.
Low-level accounting primitives remain isolated proof/library operations; this
does not claim database-wide privileged-operator constraints without a migration.
Historical command receipt read/replay and cancel/ack safety remain explicit;
manual registration replay is retired. No API profile can select a legacy mode.

Threat/acceptance review: forged manual requests and legacy command IDs must not
reach posting; source_post must retain current SQL authorization, exact immutable
draft and unique original-source link; malformed/unknown policy output fails
closed; unknown commit never retries or claims success. Named HTTP/transaction,
unit/browser and actual source-linked integration evidence is ART01–ART05 in the
[artifact contract](../contracts/artifact-backed-accounting.md). Documents and
economic-event matching/corrections remain476 until end-to-end proof exists.

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

## P05 — pgx transaction adapter

[pgx v5.11.0 API documentation](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0#Conn.BeginTx),
MIT licensed, reviewed BeginTx, Tx Commit/Rollback and PostgreSQL error handling.
The caller must explicitly manage transaction completion; cancellation does not
replace rollback cleanup. We pin the driver and module checksums, use explicit
isolation and parameter binding, and retain SQL as accounting authority. Bounded
whole-transaction retry and receipt resolution after uncertain commit are our
protocol; the driver alone does not establish exactly-once business effects.
The service accepts an owned connection/pool interface and never fetches secrets.

## P06 — Reference version serialization

PostgreSQL 17 [§13.3.5 Advisory Locks](https://www.postgresql.org/docs/17/explicit-locking.html#ADVISORY-LOCKS),
reviewed transaction-scoped lock lifetime. Our controlled currency setup uses a
per-code transaction advisory lock plus overlap checks; repeatable-read setup is
rejected because its old snapshot can miss a committed version after waiting.
Read-committed setup sees the subsequent statement snapshot; serializable setup
must retry a serialization failure. Hash collisions only serialize extra codes.
This is a setup concurrency choice, not a financial standard or a runtime grant.

## P07 — Provider revisions and linked journal corrections (2026-10-25)

[Plaid Transaction states](https://plaid.com/docs/transactions/transactions-data/),
Pending and posted transactions: full selected section reviewed. Pending may be
removed/replaced with a distinct settled identity; settled records can also change.
API2020-09-14 sync added/modified/removed facts must preserve provider consistency.
These are feed mechanics, not authorization for accounting recognition or policy.
The existing proprietary mapping and ISO4217 units remain unchanged.

Refreshed PostgreSQL17 P03 selected full Serializable Isolation and Writing
SECURITY DEFINER Functions Safely sections2026-10-25: retries cover the complete
transaction, trusted search path/privilege setup protects restricted functions.
These mechanisms support one atomic correction but do not establish our lock
order, current actor/source permissions or exactly-once effects by themselves.

[Correction decision](../contracts/source-journal-corrections.md) compares linked
reversal/replacement with in-place edits, bank-table regeneration and delta
adjustments. SQL retains exact money/current authority and original historical
valuation; no new live policy is selected. USD50->55->60 and EUR examples are
independently fixed proposed acceptance, not generated implementation results.
Full Plaid manuals/paid accounting or ISO normative text were not reviewed here.
