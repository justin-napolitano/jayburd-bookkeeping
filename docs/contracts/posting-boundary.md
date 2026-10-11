# First posting operation boundary

## Decision and implementation scope

This is the implementation target for the next synthetic PostgreSQL posting PR,
read with the [SQL dictionary](ledger-sql.md) and
[development policy](../decisions/0005-development-accounting-policy.md).
It resolves the earlier enforcement alternatives: restricted database functions
own accounting invariants; Go owns trusted identity, HTTP and whole-transaction
retry. No production implementation exists. Evidence/alternatives are in
[posting source notes](../research/posting-sources.md).

Deliver manual identity/spot posting, journal retrieval, exact reversal and a
trial balance. Controlled test setup creates books, policies, accounts, periods,
principals, memberships and synthetic rates. No runtime setup grant, public
identity provider, production bootstrap, period-close endpoint or worker is part
of this proof. Keep #1/#2's broader gates open; this bounded target does not
resolve their real-entity, close-evidence or export requirements.

## SQL call contract

These signatures are specifications, not a migration. All names are schema
qualified; routines run as `bookkeeping_owner` with the SQL contract's trusted
search path and explicit grants. No overloads or defaulted identity arguments.

| Function | Parameters, in order | Result |
| --- | --- | --- |
| `ledger.post_journal` | actor UUID, entity UUID, book UUID, key text, request UUID, body text | One row: `http_status smallint`, `body jsonb`, `replayed boolean` |
| `ledger.reverse_journal` | actor UUID, entity UUID, book UUID, original journal UUID, key text, request UUID, body text | Same receipt result |
| `reporting.get_journal` | actor UUID, entity UUID, book UUID, journal UUID | `jsonb` matching OpenAPI Journal |
| `reporting.trial_balance` | actor UUID, entity UUID, book UUID, through-date text | `jsonb` matching OpenAPI TrialBalance |

Actor and request UUID come from trusted Go context, never user JSON. No request
hash, actor claim, precomputed total or client-selected SQL function is accepted.
Go binds parameters. Mutation body is the original UTF-8 JSON text so the database
can reject duplicate keys with `IS JSON OBJECT WITH UNIQUE KEYS` before
converting to jsonb. This also protects direct
SQL callers of the granted operation. SQL rejects SQL NULL arguments, unknown or
missing fields, JSON null, wrong types and unsupported operation shapes.

For reads, SQL validates the date just as for writes. Business dates use strictly
`YYYY-MM-DD`, real Gregorian dates in years 0001–9999; no infinity or flexible SQL
date coercion. UUID text inside JSON uses standard hyphenated form, case-insensitive,
then normalizes to lowercase. Reject invalid UTF-8/U+0000 and dates/amounts that
cannot round-trip. Go returns no success until its COMMIT succeeds.

## One authority per rule

| Rule | Authority and check |
| --- | --- |
| Network body bound and token verification | Go: 1 MiB bytes, issuer/audience/expiry/subject; no database connection for invalid token |
| Durable input shape/bounds | SQL: same byte bound, 2–1000 lines, duplicate/unknown keys rejected, required nonnull strings, positive exact amounts; Go may reject earlier but never authorize a bypass |
| Principal, entity and permission | SQL reloads current rows; trusted Go principal mapping is the documented trust boundary |
| Policy, account dates and period | SQL resolves exactly one effective policy/period and scoped active accounts |
| Currency and conversion | SQL resolves immutable currency versions by accounting date; identity prohibits a rate ID; spot recomputes exact rounded amount from an eligible stored rate |
| Balance, immutability, reversal | SQL validates full journal; no API role table DML; reversal copies historical values, bypassing fresh-rate selection only |
| Replay, audit and outbox intent | SQL inserts them atomically with the journal; no separately committed receipt |
| Retry and response | Go rolls back/retries the entire transaction on 40001/40P01; emits safe mapped errors only |

A replay checks input syntax and current permission before looking up its receipt;
it does not revalidate rates, account activity, policy/period or expected revision
against today's mutable state. An original historical reversal preserves valuation
but still checks the target period and active accounts. Direct owner/superuser
access is outside runtime guarantees and remains controlled migration authority.

## Exact rounding and amount checks

SQL accepts original decimal **strings**, validates lexical bounds before any
numeric typmod cast, then performs exact unrestricted-numeric multiplication.
Round positive product `x` at scale `s`: let `u = x * 10^s`, `n = floor(u)` and
`f = u - n`; use `n` when `f < 0.5`, `n+1` when `f > 0.5`; at equality choose the
even integer. Divide that integer by `10^s`. Check range/positive result and exact
agreement with the client assertion before storing. Do not substitute PostgreSQL
`round(numeric, s)`, which uses another tie rule [P02]. No float intermediary.

Trailing zeroes up to the lexical storage bound are accepted: `1.000` satisfies
USD two-decimal value precision; `1.001` does not. Decimal syntax with more than
nine fractional digits fails before normalization even if extra digits are zero.
Currency minor units constrain transaction amounts; functional policy scale
constrains functional amounts. Missing metadata or an unsupported scale fails.
The engine must reject unsupported policies rather than applying a USD default.

## Canonical replay version 1

SQL creates and stores `normalized_request jsonb` from validated input, including
operation, path scope, reversal target when applicable, and every accepted body
field. Normalize UUID case and insignificant fractional zeroes; preserve line
order and description/reason Unicode content (no trimming or Unicode normalization).
Revision stays a canonical unsigned decimal string, capped at 9223372036854775807.
Optional absent rate stays absent; explicit null fails. Unknown fields cannot be
silently omitted. The idempotency key is scoped by entity/book/operation, bound to
actor, and excluded from payload; transport request UUID is also excluded.

Compare the full normalized jsonb for equality under `canonical_version = 1`.
Object order is immaterial, array order is material. `request_hash` is computed in
SQL as SHA-256 over UTF-8 `normalized_request::text`; it is an integrity/diagnostic
field, **not** the equality authority. No pgcrypto extension is needed for core
`sha256(bytea)`. Never accept a caller hash. Persist the normalized request and
version for the book lifetime with the receipt, under financial-data access and
retention rules; never log the payload. Preserve version-1 interpretation for old
receipts across upgrades; a new serializer/hash cannot authorize a new effect.

Alternatives: hashing raw HTTP bytes spuriously conflicts on formatting;
Go-only canonicalization trusts a bypassable SQL parameter. Exact stored semantic
comparison costs up to one bounded request per successful command but avoids
using digest equality as the sole financial replay decision.

Same key/actor/normalized payload returns original status/body without writes;
same key with changed body, operation target or actor conflicts. Different
operation names have separate keys by design. New key plus stale expected revision
conflicts; a failed transaction stores no receipt. No TTL, arbitrary replay reset
or retry with a silently changed expected revision.

## Locking and failure protocol

Mutation starts SERIALIZABLE. Lock principal FOR SHARE, entity FOR SHARE,
membership FOR SHARE, then book FOR UPDATE; subsequent order follows the SQL
contract (receipt, period, accounts in UUID order, original journal). Entity
archive, principal disable and membership revoke update the corresponding locked
row. Book disable/setup/account changes use the book lock. This closes the entity
archive race without serializing all posting across an owner's different books.
Missing membership is denied; a concurrent grant may require a fresh request.

The database routines reject an incorrect transaction isolation level; they do
not begin/commit transactions. Each successful mutation increments revision once
and writes one journal, its lines, audit action, receipt and outbox intent. Revision
exhaustion fails before writes; never wrap. No external side effects occur inside
the transaction. At most three attempts within a five-second total request budget;
transaction-local timeouts respect the remaining budget. An uncertain COMMIT is
resolved only by retrying the same operation/key/body in a fresh transaction.

Read functions require REPEATABLE READ or SERIALIZABLE, with a READ WRITE
transaction to permit authorization row locks. They take principal, entity and
membership FOR SHARE, then book FOR SHARE, so disable/revoke operations use the
same ordering; short report transactions can temporarily block that book's writes.
Reports include inactive accounts
with history and zero-balance configured accounts, sort by account code then UUID,
and count both originals and reversals. Return a safe `result_out_of_range` 422
if aggregate amounts exceed the documented response bounds; do not truncate.

SQL business errors use fixed SQLSTATEs P2100–P2110 with no financial identifiers
in message/detail: invalid_request/400, not_found/404, permission_denied/403,
revision_conflict/409, idempotency_conflict/409, period_closed/409,
already_reversed/409, invalid_amount/422, invalid_journal/422,
unsupported_policy/422, result_out_of_range/422, respectively. P2111 maps oversized SQL body to request_too_large/413; Go
usually rejects it first. Rate/currency
failures in this slice map to invalid_journal; malformed amount to invalid_amount.
Go never exposes raw database errors. Unexpected failures return generic 500;
serialization/deadlock exhaustion or uncertain commit returns retryable 503.

## Required PostgreSQL acceptance (not yet executed)

| Named test / evidence | Required observation |
| --- | --- |
| `TestSchemaPrivileges` | Actual API role cannot CREATE, SET ROLE owner, read/DML base tables or call ungranted routines; temporary-object/search-path attacks fail |
| `TestPostingInvariants` | Reject empty/unbalanced/mixed-book input; fail after each durable write and at COMMIT; journal/lines/audit/receipt/outbox/revision all roll back |
| `TestMoneyInputBounds` | Half-even vectors above, zero/2/3 minor units, overflow, nonfinite, excess scale, wrong rate direction/date/source, false functional assertion; exact zero balance |
| `TestPostingReplayConcurrency` | Concurrent same key: one effect, one increment; equivalent decimal/object/UUID spelling replays; changed line order/body/actor conflicts; old receipt survives later close/archival |
| `TestEntityReferencesAndPrivileges` / `TestRevocationRace` | Cross-scope references denied; current read/post permission checked; principal/entity/membership revocation ordering proven with barriers, not sleeps |
| `TestJournalCorrectionHistory` | Exact original values/rates reversed; original unchanged; duplicate/full-second reversal rejected; archived target account fails new correction |
| `TestPostingCloseRace` | Controlled test-only closer acquires the production book lock; closing first rejects posting; posting first invalidates stale close revision; no claim of implemented close evidence |
| `TestGoldenMonth` | Load AC-01–10 into actual PostgreSQL, assert fixed balances, original-unit totals and one audit/receipt/outbox per journal; second book isolated |
| `TestManualSpotPosting` | EUR 100 at USD/EUR 1.10 posts USD 110 with preserved rate/currency provenance; full FX lifecycle remains a later gate |
| `TestAPIContract` | Strict shape, denial/redaction, replay headers, COMMIT-before-201, unknown-commit 503; no auth claim from a synthetic actor adapter |
| `TestSchemaCompatibility` / `TestLedgerRestoreReplay` | Fresh/forward migrations and backup restore retain journals, policy/reference versions, receipts and intent; restored retry cannot duplicate |

Use deterministic goroutine/database barriers and capture ordered outcomes. Fake
SQL or fixture arithmetic alone cannot satisfy this matrix. Migration tests use
only disposable PostgreSQL with synthetic seed identities; no live DSN. Record
actual evidence before declaring the posting PR ready.
