# First ledger SQL contract

## Decision and scope

JB-002 physical/transaction specification. Migrations 000001/000002 now implement
the synthetic manual posting subset; see [evidence](../runbooks/posting-proof.md). The first manual
posting target is finalized for implementation review in [posting boundary](posting-boundary.md)
and [ADR 0005](../decisions/0005-development-accounting-policy.md); their narrowed
scope and decisions supersede earlier proposals below. Broader close/FX workflows
remain gated. Accepted namespace direction is [ADR 0004](../decisions/0004-schema-ownership.md).
The first proof uses controlled synthetic book/account setup and supports manual
identity/spot journal posting, exact reversal and authorized reads. Period close
remains a later workflow; closed-state rejection is required immediately. Position accounting,
provider import and reconciliation are logical contracts below, not complete
physical schemas. [Money rules](multicurrency.md) and [scenarios](accounting-scenarios.md)
are prerequisites. An API cannot bypass these requirements.

Select a narrow database function boundary over granting API roles table DML.
Go validates transport/identity and orchestrates one transaction; owned database
functions enforce durable invariants. Alternatives: Go with deferred triggers
requires broader write privileges and a proven complete mutation protocol; raw
row CHECKs cannot enforce full-journal balance. Evidence: S07/S08/S11/S12 in the
[source register](../research/database-sources.md). The selected design still requires privilege and concurrency tests before
claiming implemented enforcement.

## Model and scope keys

An entity has memberships and books; each book has policy revisions, accounts,
periods and posted journals; each journal has at least two lines. A line references
an account and journal in the same entity/book. Books support separate reporting
bases but this slice does not automatically maintain parallel books.
Currency/rate reference data is shared; source observations never become ledger
entries without an authorized posting operation.

For every book-owned row, use `(entity_id, book_id, <row_id>)` as the primary key
and preserve those columns in every foreign key. Book key is `(entity_id, book_id)`;
entity key is `entity_id`. Globally generated UUIDs do not replace scope constraints.
Unique business keys include entity/book. No cross-entity journal or implicit
cross-book line. All foreign keys use RESTRICT/NO ACTION, never deletion cascades
into financial history. Use ordinary exact text codes with CHECKs initially,
not PostgreSQL enums that complicate later vocabulary changes.

## Column dictionary

All columns below are NOT NULL with no default unless marked `?` (nullable) or
`= value` (default). IDs are `uuid`, minted server-side. `instant` means
`timestamptz`; `money` means the proposed validated `numeric(38,9)` domain;
`rate` means validated `numeric(38,18)`. Codes are `text` with explicit checks;
no float types. Exact request inputs must be checked before typmod coercion.
The first migrations implement amount/rate domains; input text is validated
before typmod coercion. Tables for future adjacent workflows remain contracts.

| Table / row grain | Columns beyond the scope key | Keys, checks and lifecycle |
| --- | --- | --- |
| `ledger.entity` / legal or personal accounting owner | `entity_id uuid`, `name text`, `kind text`, `created_at instant`, `disabled_at instant?` | kind legal/personal; archive, no deletion of referenced owners |
| `ledger.principal` / trusted authenticated identity | `principal_id uuid`, `issuer text`, `subject text`, `disabled_at instant?`, `created_at instant` | Unique issuer/subject; subject is not an email lookup; never store bearer credentials |
| `ledger.membership` / entity permission for principal | `entity_id uuid`, `principal_id uuid`, `permission text`, `revoked_at instant?` | PK entity/principal/permission; permissions read/prepare/post/close; FKs entity/principal; regrant is audited |
| `ledger.book` / accounting book | scope key, `name text`, `revision bigint = 0`, `created_at instant`, `disabled_at instant?` | Unique entity/name; revision >= 0; explicit book row is serialization lock |
| `ledger.book_policy` / immutable approved policy revision | scope plus `policy_id uuid`, `effective_from date`, `effective_to date?`, `functional_currency_version_id uuid`, `basis text`, `timezone text`, `posting_scale smallint`, `rounding_mode text`, `rate_policy_code text`, `approved_by uuid`, `approved_at instant` | Unique book/effective_from; disjoint half-open intervals under book lock; approved complete policies only; timezone recognized; scale 0–9; basis/rounding/rate codes from reviewed policy; FK currency/principal |
| `ledger.account` / classification within book | scope plus `account_id uuid`, `code text`, `name text`, `category text`, `normal_side text`, `monetary boolean`, `active_from date`, `active_to date?` | Unique book/code; categories asset/liability/equity/revenue/expense; side debit/credit; dates half-open; semantic changes after use require new account or explicit migration contract |
| `ledger.period` / non-overlapping posting interval | scope plus `period_id uuid`, `starts_on date`, `ends_before date`, `state text = open`, `closed_at instant?`, `closed_by uuid?`, `close_revision bigint?` | starts < end; no overlap within book under setup/book lock; state open/closed; closed metadata all present iff closed; FK principal; never reopen by direct update |
| `ledger.journal_entry` / one committed journal | scope plus `journal_id uuid`, `period_id uuid`, `policy_id uuid`, `accounting_date date`, `kind text`, `description text`, `posted_at instant`, `posted_by uuid`, `book_revision bigint`, `reverses_journal_id uuid?`, `reason text?` | Kinds manual/reversal in first slice; FK same-book period/policy/reversed journal; reversal needs original+reason; unique book/reversed-journal for nonnull original; only posted rows, no mutable status; description 1–500 characters |
| `ledger.journal_line` / one side/amount/account | scope plus `journal_id uuid`, `line_number integer`, `account_id uuid`, `side text`, `transaction_currency_version_id uuid`, `transaction_amount money`, `functional_amount money`, `valuation_basis text`, `rate_observation_id uuid?` | PK scope/journal/line_number; FK same-book journal/account; line_number 1–1000; side debit/credit; positive amounts in first slice; basis identity/spot or carrying/functional_adjustment reserved for future operations; cross-field money checks per contract |
| `reference.currency` / one currency definition version | `currency_version_id uuid`, `alphabetic_code text`, `numeric_code text?`, `minor_unit smallint?`, `valid_from date`, `valid_to date?`, `source_release text` | Code `[A-Z]{3}`, numeric code three digits if assigned; unique code/valid_from; disjoint validity intervals; no assumed minor unit when absent; referenced versions immutable |
| `reference.rate_observation` / one immutable sourced quote | `rate_observation_id uuid`, `base_currency_version_id uuid`, `quote_currency_version_id uuid`, `quote_per_base rate`, `effective_at instant`, `observed_at instant`, `source text`, `source_reference text`, `purpose text`, `policy_code text`, `supersedes_id uuid?` | Currency/self FKs; positive finite rate; purpose spot/closing/valuation; provenance retained; ingestion identity unique source/source_reference; conflicting redelivery is reviewed, never overwritten |
| `integration.operation_receipt` / one successful command | scope plus `operation text`, `idempotency_key text`, `principal_id uuid`, `request_hash bytea`, `normalized_request jsonb`, `canonical_version smallint`, `result jsonb`, `http_status smallint`, `created_at instant` | PK scope/operation/key; key 1–128 safe ASCII; hash exactly 32 bytes; full normalized request equality is authoritative per posting boundary; result schema versioned; retain for book lifetime pending retention policy; no unsafe TTL that enables a duplicate financial effect |
| `audit.action` / one committed actor action | scope plus `action_id uuid`, `principal_id uuid`, `operation text`, `target_id uuid`, `book_revision bigint`, `request_id uuid`, `recorded_at instant`, `reason text?` | Append-only; no raw financial payloads/tokens in diagnostics; target checked by operation; external audit durability requires separate design |
| `integration.outbox_message` / durable committed notification intent | scope plus `message_id uuid`, `event_type text`, `event_version integer`, `target_id uuid`, `book_revision bigint`, `created_at instant` | Append-only financial intent with identifiers only; event type journal.posted/period.closed; future dispatch state is separate; creation alone does not implement delivery |

Journal reversal keeps original policy and valuations to preserve exact historical
amounts, but its accounting date belongs to a currently open period. If functional
currency changed, reject pending a currency-transition contract. The first slice
rejects reversal of a reversal and reversal of position-affecting entries until
their correction semantics are specified.

## Roles, grants and trusted identity

| Role | Allowed | Explicitly denied |
| --- | --- | --- |
| `bookkeeping_owner` (NOLOGIN) | Own schemas, functions and tables | Runtime login/use by application |
| Migration identity | Controlled SET ROLE to owner during reviewed migrations | Application use or default CI access to live database |
| `bookkeeping_api` | Schema USAGE and EXECUTE on explicitly granted ledger/read operations | Schema CREATE, ownership, direct base-table SELECT/DML, role escalation |
| Future worker role | Only individually reviewed operation grants | Implicit API privileges or arbitrary actor impersonation |
| Human reporting identity | Reviewed entity-authorized reporting interface | Unrestricted shared reporting views/base tables |

Revoke PUBLIC CREATE on public and PUBLIC function execution where applicable;
set default privileges for the actual owner before function creation. Runtime
roles receive no inherited owner membership, superuser or BYPASSRLS. Use qualified
objects and trusted `search_path = pg_catalog, ledger, reference, integration,
audit, pg_temp` on privileged functions; none of these schemas is runtime-writable.
Revoke function EXECUTE from PUBLIC in the same migration transaction as creation.
See PostgreSQL function security S12; grant tests must include malicious temporary
objects and altered session search_path. Extensions need an explicit separate review.

The Go service is a trusted identity boundary: validate issuer, audience, expiry
and subject, map to principal, and pass trusted identity separately from JSON input.
A shared SQL role cannot prove which human is calling; these functions protect
against unchecked DML, not a fully compromised trusted service forging principals.
Every operation reloads current principal/membership state; no caller-set entity
session variable or RLS claim substitutes for identity. RLS is deferred pending
its own reviewed role policy; scoped read functions and composite keys are required.

## Transaction and lock protocol

All mutations use SERIALIZABLE. Lock in this order: principal FOR SHARE;
entity FOR SHARE; relevant membership FOR SHARE; book FOR UPDATE; existing receipt; period; affected
accounts in UUID order; original journal if needed. Revocation/archive updates the same
principal/entity/membership rows; book setup, account edits, posting and closing all take
the same book lock. Every change affecting posting/report interpretation or close
evidence (including configuration, reconciliation and valuation revisions) also
increments book revision; approval evidence records the revision it reviewed.
Thus revocation that wins blocks later work; work already
serialized before revocation may commit. Different books can progress independently.
No multi-book transaction is exposed in this slice. Close/post contention is a
measurable tradeoff: benchmark before replacing the simple lock protocol.

Retry SQLSTATE 40001/40P01 by rolling back and rerunning the entire transaction,
including current authorization, with bounded jitter: proposed maximum 3 attempts
within a 5-second request budget. Exhaustion returns a retryable 503; client reuses
its key. No external network work or irreversible side effects inside a transaction.
Timeout/disconnect after commit is resolved by the receipt, not a guessed failure.

### Post journal

1. Validate trusted actor, size/shape, decimal inputs and required idempotency key.
2. Begin transaction and take authorization/book locks. Reject disabled scope.
3. Resolve receipt before mutable period/account checks. Same actor, command and
   normalized request returns original status/body; different payload/actor conflicts.
   Recheck current permissions even on replay; do not leak old results after revocation.
4. For new work, validate expected book revision, one open period containing the
   accounting date, effective approved policy, account activity and currency/rate
   provenance. Book must have no unresolved configuration. Check full journal
   has 2–1000 lines and exact equality of functional debit/credit totals.
5. Increment book revision once; insert journal/lines, audit action, receipt and
   outbox intent. All succeed or all roll back. Return 201 only after COMMIT.

Payload normalization and receipt equality follow [canonical replay version 1](posting-boundary.md#canonical-replay-version-1).
SQL owns normalization, stored normalized-request comparison and diagnostic hash;
Go supplies original body text, never a trusted hash. Reject duplicate/unknown
JSON keys before jsonb conversion. Failed validation leaves no receipt; successful
keys never become reusable.

### Reverse journal

Same protocol and `expected_revision` semantics. Authorize post; copy original
lines with sides swapped and identical amounts/rate references; set original link
and supplied reason/date. Require open target period and active permitted accounts;
if an account is archived, use a separately reviewed correction instead of bypassing
activity checks. Unique reversal link prevents two full reversals with different
keys. Return new journal 201; original remains untouched. No generic DELETE/PATCH.

### Close period

Authorize close; same locks/receipt/revision protocol. The request identifies an
immutable reviewed close-evidence record covering reconciliation, outstanding
items, required adjustments, FX completion and report review. Close validates that
record belongs to this book/period and exact expected book revision. Caller boolean
`reconciled=true` is never evidence. Increment revision, mark closed and append
audit/receipt/outbox in one transaction; return 200 with period/revision. No new
posting into a closed period. Earlier open periods must be resolved before close.
A replay of a successful close returns its receipt; another key on a closed period
conflicts. A close-evidence physical contract is outstanding: endpoint remains
non-implementable until that dependency is accepted.

Period locking differs from fiscal-year closing entries for temporary accounts.
Annual closing/retained-earnings rules and reopening are separate pending contracts.

### Read/report

Authorize each request; use one repeatable-read transaction for a
multi-query response, so its lines/totals share one snapshot. Read principal/entity/membership
and book with FOR SHARE under the posting boundary's revocation ordering. This requires a READ WRITE
transaction even though the operation performs no business writes; PostgreSQL
READ ONLY does not permit these row locks. Apply the bounded retry policy. Journal reads return complete immutable data;
trial balance derives functional totals from posted lines, filters by book/date
and includes snapshot book revision and functional currency. Reversed originals
and reversals both contribute; do not filter originals away. Never convert report
currency opportunistically. Pagination uses deterministic accounting_date/journal_id
order and a cursor bound to scope/filter/snapshot; pagination contract is pending,
so a list endpoint is deliberately absent from this first interface.

## Adjacent workflow contracts and remaining physical work

| Operation | Required durable semantics before its own implementation |
| --- | --- |
| Import | Source account belongs to entity; immutable versioned observations, provider identity/checkpoint deduplication and provenance; corrected source cannot edit posted books |
| Reconcile | Statement identity, currency, balances and covered dates; scoped line allocations, unmatched items and explicit reviewed difference; match is not an automatic journal |
| FX remeasure | Versioned valuation batch, currency exposure positions, carrying-value deltas and unique operation identity; serialize against settlement/close |
| Settle | Position/allocation references, remaining quantity/carrying amount, fees and FX journal in one transaction; no over-allocation |
| Intercompany | Separate authorized journals linked by transfer identity; unmatched side visible; never claim atomic cross-book completion without a dedicated protocol |
| Export | Authorized immutable report snapshot and mapping/version manifest; apply ISO 21378/21926 assessment; no raw base-table dump |

These need complete dictionaries and API definitions next; reserving schema names
is not completion. Preserve [background delivery contract](background-work.md);
this slice only specifies atomic outbox intent, not a publisher/worker lifecycle.
ISO applicability for these identifier-only internal notifications: they describe
local journal/period commits, not banking interchange; no registered financial
message definition is claimed. Version the local event contract. If an adapter
later creates a bank-facing message, assess and validate its ISO profile separately.

## Queries, indexes, recovery and acceptance

Proposed indexes: journal `(entity_id, book_id, accounting_date, journal_id)` for
dated reads; line `(entity_id, book_id, account_id, journal_id)` for account history;
PKs/unique business keys cover scoped lookups and receipts; rate pair/effective time
supports rate selection. Record expected row counts and EXPLAIN on synthetic data
before optimizing. No benchmark or cardinality target has yet been approved.
Migrations use reviewed golang-migrate files; no live automatic down migrations.
Before cutover, prove backup/restore retains journals, reference versions, receipts
and outbox intent together, and replay cannot duplicate restored financial effects.

| Named future evidence | Acceptance |
| --- | --- |
| `TestSchemaPrivileges`, `TestEntityReferencesAndPrivileges` | Runtime DDL/DML, cross-entity/book references, forged request scope, revoked/disabled actors, search-path attacks denied |
| `TestPostingInvariants`, `TestMoneyInputBounds` | Empty/unbalanced/invalid amounts fail; identity/spot rules hold; failure at each write leaves no partial state |
| `TestPostingReplayConcurrency` | Concurrent same keys yield one entry; changed payload conflicts; reordered object keys replay; reordered lines conflict |
| `TestJournalCorrectionHistory` | Exact reversal, duplicate reversal conflict, original immutable and reports count both |
| `TestPostingCloseRace`, `TestRevocationRace` | Documented serialization orders hold; close evidence stale after any relevant change |
| `TestGoldenMonth`, `TestFXLifecycle` | Scenario arithmetic and reconciliations agree; no unsupported automated FX claim |
| `TestSchemaCompatibility`, `TestLedgerRestoreReplay` | Forward migration compatibility and restored idempotency/history proven |

Execution status is recorded in the posting proof runbook; full FX lifecycle
and live recovery/cutover remain future gates. ADR 0005 and the posting boundary select the synthetic development policy,
precision and enforcement target. A synthetic actor adapter does not approve a
production identity provider or real-LLC setup. Close, settlement,
reconciliation and full FX automation have additional explicitly stated gates.
