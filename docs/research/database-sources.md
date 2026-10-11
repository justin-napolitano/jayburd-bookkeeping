# Database and bookkeeping source register

For reading order, topic descriptions and the standards/multicurrency resources,
see the [annotated bibliography](bibliography.md). This register retains the
detailed S01–S22 evidence records.

## Scope and method

Targeted public-web research, accessed 2026-10-11. Queries covered relational
design, accounting, PostgreSQL isolation, ledger case studies, outbox delivery,
ORM alternatives and migrations. Followed publisher/author links and read the
sections identified below. This is a focused review, not an exhaustive literature
review or a claim to have read every book/paper cover to cover. No books or papers
are mirrored here; notes are short original paraphrases.

Prefer accounting texts for accounting concepts; versioned PostgreSQL manuals
for PostgreSQL behavior; original papers for mechanisms; vendor cases for examples.
An arXiv listing is not itself peer-review evidence. Search-result timestamps were
not treated as publication dates. Recommendations live in the [decision brief](database-foundations.md).

## S01 — Accounting foundations

OpenStax, *Principles of Accounting, Volume 1: Financial Accounting*,
[section 3.1](https://openstax.org/books/principles-financial-accounting/pages/3-1-describe-principles-assumptions-and-concepts-of-accounting-and-their-relationship-to-financial-statements).
Read: accounting equation, double-entry, journals and account balances.
Evidence: transactions affect accounts through equal debit/credit totals.
Use: journal/header/line model and balance invariants. Limit: educational accounting
text, not a database schema or a decision about this owner's accounting basis.

## S02 — Reconciliation

OpenStax, same book,
[section 8.6](https://openstax.org/books/principles-financial-accounting/pages/8-6-define-the-purpose-of-a-bank-reconciliation-and-prepare-a-bank-reconciliation-and-its-associated-journal-entries).
Read: bank-versus-book adjustments and worked reconciliation.
Evidence: timing differences, bank errors and unrecorded book items are different
causes of disagreement. Use: keep imported observations separate from posted books.
Limit: importing bank rows is not a complete bookkeeping method.

## S03 — Relational design textbook

Adrienne Watt and Nelson Eng (2014), *Database Design, 2nd Edition*, BCcampus.
Read selected material in [ER modelling, chapter 8](https://opentextbc.ca/dbdesign01/chapter/chapter-8-entity-relationship-model/),
[normalization, chapter 12](https://opentextbc.ca/dbdesign01/chapter/chapter-12-normalization/),
and [development process, chapter 13](https://opentextbc.ca/dbdesign01/chapter/chapter-13-database-development-process/).
Evidence: requirements, entities/relationships, dependencies and physical design
are distinct work. Use: a data dictionary and dependency review before DDL.
Limit: introductory simplifications and a waterfall presentation; a surrogate
primary key alone does not establish normalization across all candidate keys.
Use the stages iteratively, not as a requirement to freeze the entire future schema.

## S04 — Relational independence

E. F. Codd (1970), *A Relational Model of Data for Large Shared Data Banks*,
Communications of the ACM 13(6), 377–387, DOI 10.1145/362384.362685.
[Paper, university-hosted copy](https://cs.uwaterloo.ca/~david/cs848s14/codd-relational.pdf).
Read: opening sections on ordering, indexing and access-path dependence.
Evidence: logical representation should not depend on physical access paths.
Use: separate business identity/relationships from indexing and deployment choices.
Limit: foundational model, not a modern PostgreSQL implementation recipe.

## S05 — Double-entry mathematics

David Ellerman (2014), *On Double-Entry Bookkeeping: The Mathematical Treatment*,
[arXiv:1407.1898](https://arxiv.org/abs/1407.1898),
[paper](https://arxiv.org/pdf/1407.1898).
Read: pp. 1–7, T-account algebra and signed representations.
Evidence: debit/credit pairs admit precise algebra; signed storage requires an
explicit interpretation. Use: test sign conventions and reversals, not an assumption
that double-entry means exactly two SQL rows. Limit: mathematical exposition;
not authority for tax policy, concurrency, or a mandatory storage encoding.
The PDF's extracted metadata title differed; its title page matches the citation.

## S06 — Serializable snapshot isolation

Dan R. K. Ports and Kevin Grittner (2012), *Serializable Snapshot Isolation in
PostgreSQL*, PVLDB 5(12), 1850–1861,
[arXiv:1208.4179](https://arxiv.org/abs/1208.4179),
[paper](https://arxiv.org/pdf/1208.4179).
Read: overview and sections 2–3, especially the receipt/close/report anomaly.
Evidence: snapshot isolation permits write skew and some read-only anomalies;
SSI detects dangerous dependencies and aborts transactions.
Use: adversarial posting/close/report schedules and whole-transaction retry design.
Limit: implementation/benchmarks concern PostgreSQL 9.1; use S08 for version 17
semantics. The paper's performance results are not our capacity estimate.

## S07 — Database constraints

PostgreSQL 17 manual, [5.5 Constraints](https://www.postgresql.org/docs/17/ddl-constraints.html).
Read: CHECK limitations, null handling, unique keys and composite foreign keys.
Evidence: row CHECKs cannot reliably maintain conditions over other rows.
Use: explicit NOT NULL, entity-scoped references and a separate balance-enforcement
mechanism. Limit: naming an invariant does not implement its enforcement.

## S08 — Isolation semantics

PostgreSQL 17 manual, [13.2 Transaction Isolation](https://www.postgresql.org/docs/17/transaction-iso.html).
Read: Read Committed, Repeatable Read and Serializable sections.
Evidence: statement snapshots differ under Read Committed; serializable failures
require retrying the transaction. Use: specify isolation per operation and retry
from original inputs. Limit: serializability preserves correctly written operations;
it does not invent accounting rules or automatically order external side effects.

## S09 — Row-level security

PostgreSQL 17 manual, [5.9 Row Security Policies](https://www.postgresql.org/docs/17/ddl-rowsecurity.html).
Read: defaults, owner/BYPASSRLS behavior, whole-table operations and policy caveats.
Evidence: enabled RLS defaults to denial without policies, but privileged roles
and some operations bypass it. Use: nonowner runtime roles and role-specific tests.
Limit: RLS does not authenticate caller-supplied entity IDs or replace write grants.

## S10 — Exact amounts

PostgreSQL 17 manual, [8.1 Numeric Types](https://www.postgresql.org/docs/17/datatype-numeric.html).
Read: numeric precision/scale, rounding, special values and floating-point behavior.
Evidence: numeric is exact but can round to declared scale; NaN exists and compares
above finite values. Use: finite-value checks and explicit input/rounding policy.
Limit: documentation does not choose the correct scale, currency rules or minor-unit
encoding for our product. A positive-value check alone is insufficient for NaN.

## S11 — Deferred enforcement

PostgreSQL 17 manual, [CREATE TRIGGER](https://www.postgresql.org/docs/17/sql-createtrigger.html).
Read: constraint-trigger timing and limitations.
Evidence: constraint triggers are AFTER ROW and can defer to transaction end.
Use: candidate final-journal validation. Limit: a trigger on lines alone cannot
notice an empty journal; concurrency, header transitions and all mutation paths
still need an explicit design and tests.

## S12 — Restricted posting functions

PostgreSQL 17 manual,
[CREATE FUNCTION: security-definer safety](https://www.postgresql.org/docs/17/sql-createfunction.html#SQL-CREATEFUNCTION-SECURITY).
Read: ownership, trusted search_path and PUBLIC execute privileges.
Evidence: functions can execute with owner privileges and require careful grants.
Use: evaluate a narrow posting function with revoked direct DML. Limit: this is
an elevated trust boundary, not a shortcut around authorization.

## S13 — Internal state versus messages

Pat Helland (2005), *Data on the Outside versus Data on the Inside*, CIDR,
[paper](https://www.cidrdb.org/cidr2005/papers/P12.pdf).
Read: introduction, encapsulation and transaction-boundary discussion (§1).
Evidence: autonomous services exchange data across different trust/transaction
boundaries. Use: keep ledger changes in one authority and treat messages as external
contracts. Limit: the paper assumes a particular service boundary; it does not
prove that each Go executable should be a microservice or own a database.

## S14 — Transactional outbox

Chris Richardson, [Transactional Outbox](https://microservices.io/patterns/data/transactional-outbox.html).
Read: problem, forces, solution and result context.
Evidence: durable message intent shares the business transaction; the relay can
publish again after a crash. Use: transactional intent plus idempotent consumers.
Limit: delivery progress requires a functioning relay; ordering depends on actual
relay/transport design, not merely the presence of an outbox table.

## S15 — Actual queue semantics

Google Cloud, [Cloud Tasks issues and limitations](https://docs.cloud.google.com/tasks/docs/common-pitfalls).
Read: execution order, delay and duplicate execution.
Evidence: duplicate execution occurs and tasks have no general execution-order
guarantee. Use: operation identities, durable outcomes and stale/version checks.
Limit: the generic outbox ordering goal in S14 is not automatically provided by
this transport. We must design any required per-account sequencing ourselves.

## S16 — Production ledger case study

Square Engineering (2019),
[Books, an immutable double-entry accounting database service](https://developer.squareup.com/blog/books-an-immutable-double-entry-accounting-database-service/).
Read: accounting model, SQL representation and immutable entries.
Evidence: an operational ledger uses journals and entries with derived/cached balances.
Use: corroborates an append-only posting model. Limit: Square chose Spanner for its
workload and simplified sign conventions. Neither its database choice nor its
payment-ledger model establishes the right general-ledger/accountant model for us.

## S17 — Event sourcing distinction

Martin Fowler (2005), [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html).
Read: definition, rebuild/replay facilities and external-query complications.
Evidence: event sourcing rebuilds application state from events; external historical
inputs require special treatment. Use: distinguish immutable financial postings
from sourcing every application state transition. Limit: replay facilities have
costs; the article does not require them for every bookkeeping application.

## S18 — Evolving schemas

Pramod Sadalage and Martin Fowler,
[Evolutionary Database Design](https://martinfowler.com/articles/evodb.html).
Read: versioned migrations, application/data coordination, CI and sequence collisions.
Evidence: ordered, tracked schema/data changes can evolve alongside code and tests.
Use: small reviewed migrations after each contract slice. Limit: examples use other
tools/databases; specific locking, transactional DDL and recovery require our driver.

## S19 — SQL-generated access

sqlc, [overview](https://docs.sqlc.dev/) and
[using transactions, v1.31.1](https://docs.sqlc.dev/en/v1.31.1/howto/transactions.html).
Read: SQL-to-Go generation and WithTx transaction binding.
Evidence: generated query methods can run within an explicitly supplied transaction.
Use: typed access without schema generation from application models. Limit: it
does not establish authorization, isolation, retries or accounting correctness.

## S20 — Migration-tool guarantees

golang-migrate v4.20.1,
[migration format](https://github.com/golang-migrate/migrate/blob/v4.20.1/MIGRATIONS.md) and
[FAQ](https://github.com/golang-migrate/migrate/blob/v4.20.1/FAQ.md).
Read: version ordering, paired files, failure/dirty handling and PostgreSQL locking.
Evidence: migrations are ordered SQL with explicit failed-state handling.
Use: retain the owner-selected tool. Limit: a successful migration run does not
prove ledger invariants or data recovery. Project manifest/history checks and
deployed-release receipts are separate controls.

## S21 — Additional book, preview only

Martin Kleppmann and Chris Riccomini, *Designing Data-Intensive Applications*,
2nd edition, [publisher chapter 8 preview](https://www.oreilly.com/library/view/designing-data-intensive-applications/9781098119058/ch08.html).
Read: visible introduction and table of contents only; full chapter requires access.
Use: further reading on faults and transaction tradeoffs. No detailed design claim
in this review relies on unseen book content. No unofficial book download used.

## S22 — ORM counterexample

GORM, [transactions](https://gorm.io/docs/transactions.html) and
[migration](https://gorm.io/docs/migration.html).
Read: explicit transaction callback/manual control and AutoMigrate behavior.
Evidence: an ORM can support multi-operation transactions; automatic schema changes
are a distinct feature. Use: correct any claim that ORM usage inherently prevents
correct bookkeeping. Limit: convenience features still require our contracts.
SQL-first is a scoped maintainability choice, not a theorem or universal standard.
