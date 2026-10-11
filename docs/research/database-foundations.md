# Bookkeeping database research: first decision review

## Recommendation

Keep a normalized PostgreSQL ledger with atomic, immutable posted journals.
Separate source observations, editable drafts and posted accounting. Keep one
ledger authority with Go entrypoints; use asynchronous work outside the posting
transaction. Retain golang-migrate and prefer explicit SQL with later pgx/sqlc.
These are project recommendations, not a universally agreed software stack.

Research-only review, 2026-10-11. No schema or application implementation is
authorized by these findings. [Source register](database-sources.md) records
22 source entries, including one publisher-preview-only book. Findings distinguish
documented behavior from our proposed application of it.

## Findings that affect our design

| Question | Source-backed finding | Project recommendation or unresolved choice |
| --- | --- | --- |
| What is the accounting core? | Equal debit/credit totals and traceable journals underpin double-entry [S01](database-sources.md#s01--accounting-foundations), [S05](database-sources.md#s05--double-entry-mathematics). | One journal with multiple lines, never a fixed two-row model. Determine normal balances and sign presentation explicitly. |
| Are bank transactions the books? | Reconciliation distinguishes timing differences, errors and book adjustments [S02](database-sources.md#s02--reconciliation). | Import observations and matches separately; classification creates drafts before posting. |
| How should we model data? | ER modelling and dependency analysis precede physical details [S03](database-sources.md#s03--relational-design-textbook), [S04](database-sources.md#s04--relational-independence). | Review a normalized operational model, initially targeting 3NF; retain justified immutable snapshots and derive reporting views. No early warehouse schema. |
| Does a transaction guarantee correctness? | Isolation levels have materially different behaviors [S06](database-sources.md#s06--serializable-snapshot-isolation), [S08](database-sources.md#s08--isolation-semantics). | Prototype SERIALIZABLE for ledger mutations; compare explicit locking only after defining the same invariants. Retry the whole transaction without external side effects. |
| Can CHECK enforce balance? | Cross-row CHECK guarantees are unsupported [S07](database-sources.md#s07--database-constraints). | Review an owned posting function and/or deferred constraint triggers; include empty journals and alternate mutation paths. |
| Does entity_id secure multiple LLCs? | Foreign keys and RLS enforce different boundaries [S07](database-sources.md#s07--database-constraints), [S09](database-sources.md#s09--row-level-security). | Entity-scoped references plus trusted authorization; test RLS with actual runtime roles if adopted. Shared tables versus separate databases remains a deployment choice. |
| Which money type? | Exact numeric still has scale coercion and special values [S10](database-sources.md#s10--exact-amounts). | Choose bounded numeric or integer minor units after currency/precision policy. Reject nonfinite values; define rounding at the boundary. |
| Does ledger history require event sourcing? | Full event sourcing adds reconstruction/replay obligations [S17](database-sources.md#s17--event-sourcing-distinction); immutable ledgers also exist as ordinary SQL services [S16](database-sources.md#s16--production-ledger-case-study). | Append-only posted facts and linked correction entries; do not source every application state transition as an event. |
| Are small services inherently better? | Messages cross trust/transaction boundaries [S13](database-sources.md#s13--internal-state-versus-messages). | Separate executables where useful, while keeping accounting writes under one contract. Source evidence does not justify a database per job. |
| What does an outbox buy? | Durable intent and repeatable delivery; queues may duplicate/reorder [S14](database-sources.md#s14--transactional-outbox), [S15](database-sources.md#s15--actual-queue-semantics). | Durable idempotency and stale-work checks; define per-source sequencing if needed. No exactly-once or FIFO claim. |
| Is an ORM wrong? | Both sqlc and GORM can participate in explicit transactions [S19](database-sources.md#s19--sql-generated-access), [S22](database-sources.md#s22--orm-counterexample). | Prefer SQL-first for this ledger's reviewability. No claim of inherent ORM unsafety or demonstrated performance superiority. |
| Must we finish all database design first? | Reviewed migrations support incremental evolution [S18](database-sources.md#s18--evolving-schemas), [S20](database-sources.md#s20--migration-tool-guarantees). | Complete the contract for one slice, then implement/test it; avoid freezing hypothetical future providers and features. |

## Unresolved enforcement alternatives

This table is a proposal for JB-002, not approval to build either option.

| Alternative | Benefit | Obligation before acceptance |
| --- | --- | --- |
| Narrow PostgreSQL posting function; revoke application direct ledger DML | One database entrypoint can validate the complete journal | Prove caller authority, secure function ownership/search_path/EXECUTE grants, and correct audit/outbox/idempotency participation [S12](database-sources.md#s12--restricted-posting-functions) |
| Go transaction plus database constraint triggers | Explicit Go orchestration with final database validation | Cover header and lines, empty journals, correction paths, concurrent edits and role bypasses; deferred triggers alone do not establish the protocol [S11](database-sources.md#s11--deferred-enforcement) |

Current preference: evaluate a small database posting boundary for structural
ledger invariants, with Go handling workflows and user-facing policy. Assign each
rule one owner; do not maintain competing copies in Go and SQL. Decide after the
role matrix and transaction schedules are reviewed.

## Concrete counterexample to review

The SSI paper's batch example includes a writer reading an old batch, a closer
advancing the batch, and a report missing the late write (section 2.1.2, pp. 2–3).
Our analogous risk is a report that claims a closed month while a previously
started posting can still affect it. This is an application of the paper, not
a discovered bug in our unimplemented software. [S06](database-sources.md#s06--serializable-snapshot-isolation)

Require a documented serialization point for close, post and report publication.
Review whole-transaction retries, close-time policy checks and report snapshot
semantics together. Serializable execution alone does not define business rules
for when a month may close. The desired test asserts no successfully closed report
can silently change through a later accepted posting.

## Research-to-design path

| Step | Deliverable before implementation | Owner / issue |
| --- | --- | --- |
| 1. Accounting scenarios | Fictional month; basis, currency, opening balances, corrections, bank/card/owner/intercompany examples | Owner, JB-001 / #1 |
| 2. Logical design | Glossary, row grain, candidate keys, relationships, functional dependencies and provenance | Schema review, JB-002 / #2 |
| 3. Physical/transaction contract | Column dictionary, constraints, role matrix, write entrypoints, isolation/lock order, idempotency and report semantics | Schema review, JB-002 / #2 |
| 4. Adversarial test specification | Named failure, race, privilege, replay, reconciliation and restoration cases | JB-002 with JB-003 / #3 |
| 5. Review decision | Accepted ADR with exact source sections, assumptions, alternatives, rejection reasons and owner approval | Owner + implementer |
| 6. First implementation slice | Reviewed migration and PostgreSQL contract tests, then typed queries and application code | After steps 1–5 |

## Boundaries and deferred claims

No source establishes our tax treatment, accountant approval, latency target,
tenant volume or GCP cost. No benchmark or schema experiment ran in this session.
Go/Cloud Run and sqlc remain engineering choices; these readings are not proof
that alternatives cannot work. Spanner, Kafka, sharding, a full event store and
DuckDB require a concrete workload before reconsideration. Full currency conversion,
consolidation/eliminations, legal retention and multi-book accounting need their own
policy/source review rather than being inferred from this foundational pass.

The existing [migration tooling PR #14](https://github.com/justin-napolitano/jayburd-bookkeeping/pull/14)
was verified merged on 2026-10-11 during this research session. That delivers
tooling, not an approved ledger schema. Read this review before advancing product
schema decisions.
