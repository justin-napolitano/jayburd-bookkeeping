# Source review, category drafts and explicit posting

## Decision and limits

Migration11 implements #96: a readable source review inbox, immutable two-line
category drafts and recoverable `source_post`. A draft creates **no journal**.
Only explicit execution of its reviewed command creates the journal and immutable
source link in the same transaction. Existing posting SQL remains the authority
for policy, periods, exact money, accounts, balance, receipts and outbox intent.
[Durable import](durable-import.md), [import SQL](import-sql.md),
[posting boundary](posting-boundary.md) and [command recovery](command-recovery.md)
remain the underlying contracts.

Coordinator reviewed signatures before DDL and approved these bounded refinements:
one source observation per review, one selected offset account, original booked
date as accounting date, and same-currency identity valuation only. Source USD,
JPY and KWD units remain visible and exact. A foreign source against the book's
functional currency returns explicit `fx_required`; it never guesses a rate or
converts the source amount. Full split/spot category drafts, overlap matching,
correction intake and source-linked reversal are separate contracts. The existing
synthetic posting policy supports USD functional books; this migration does not
silently broaden that accounting policy.

Journal description is an explicit 1..500-character draft memo, separate from
immutable original source description. Categories come from the selected book's
active chart of accounts, excluding the bound source account. Categories are
account choices, not automatic accounting classifications. Outflow credits the
source asset and debits the chosen offset; inflow debits the source and credits
the chosen offset. A reviewer chooses the economic classification and must review
the complete resulting posting lines before confirming.

## Storage, permissions and revisions

New immutable `banking.review_draft` is keyed by scope/draft UUID, uniquely by
scope/review/review revision and scope/creator/idempotency key. It stores the
review/source/binding/book pins, exact generated posting body, complete normalized
review request, creator and timestamp. New immutable `banking.review_posting` has
one row per scoped review, unique draft/command/journal, scoped draft and journal
FKs and the original result. No history cascade or deletion is allowed.

`review_item` now permits only monotonic needs_review→draft_ready,
draft_ready→draft_ready and draft_ready→posted, with exactly one review revision
increment per transition. Its identity/source/created_at stay immutable.
Overlap_hold cannot draft/post. `review_decision` retains initialization and adds
immutable explicit-category draft decisions. Existing source observations,
revisions, assignments and evidence remain immutable.

Read/list/detail/categories require current entity read and explicit source read.
Draft mutation additionally requires current entity prepare and source review;
source_post registration/discovery/execute/cancel/acknowledge additionally requires
current entity post. No import grant substitutes for review/prepare/post. Current
permissions are checked before replay, including already committed commands.
Revoked unacknowledged work continues to block the principal gate until authority
is restored; this is not bypassed for convenience.

Owner-only `banking.set_source_review_permission(actor,e,b,source,key,raw)` accepts
`{expected_binding_revision,principal_id,enabled}`. It explicitly attests only the
source review grant, requires active current read membership when granting, and
permits dormant-grantee revocation. Existing entity prepare/post grants remain
separate. It records authenticated operator `session_user` and attested approver in
the existing immutable registry receipt, and increments binding revision. No API
self-grant or new runtime owner membership is introduced.

Each draft and source post increments source_revision as well as review_revision:
source revision now snapshots both source evidence and its review projection.
This makes state-filtered pagination consistent across requests. It conservatively
invalidates other previews/drafts on the same source. Drafts return the **new**
source revision they pin; source posting checks it before mutation. Drafting does
not change the book revision; successful posting increments it once through the
existing ledger function. A new draft supersedes but never overwrites an old draft.

Mutations require SERIALIZABLE, reads REPEATABLE READ or SERIALIZABLE with READ WRITE
transactions for authorization locks. Lock order starts with principal (strongest
lock for commands), entity/membership/book, source, then command/review as needed.
Source/book locks serialize snapshot changes. Functions own no commit and perform
no object I/O. Owner remains NOLOGIN bookkeeping_owner; API has narrow EXECUTE only,
no table DML or helper/operator EXECUTE. Observer/publisher grants are unchanged.

## Exact SQL and wire boundaries

All IDs are UUIDs; revision and money values are decimal strings, never floating
point. `actor,e,b` are trusted actor and entity/book UUID arguments. All functions
return jsonb. Bodies reject duplicate/unknown keys and malformed types before writes.

| Function | Input and result |
| --- | --- |
| `banking.list_sources(actor,e,b)` | `{profile:"source-list-v1",items:[Source]}`. Only readable sources, including paused/retired history; maximum100. Source: source_account_id, display_name (ledger account name), currency, currency_version_id, ledger_account_id, state, binding_revision, source_revision. Account labels are not ownership proof. |
| `banking.list_review_items(actor,e,b,raw text)` | Exact `{source_account_id,state,limit,cursor}`; state null or needs_review/overlap_hold/draft_ready/posted, limit integer1..100, cursor null or opaque string. Returns `{profile:"source-review-page-v1",source_account_id,source_revision,items:[Summary],next_cursor}`. |
| `banking.get_review_item(actor,e,b,review uuid)` | Existing signature, expanded detail below. Existing P2401 denied/inaccessible behavior is preserved. |
| `banking.review_categories(actor,e,b,review uuid)` | `{profile:"source-review-categories-v1",review_id,source_account_id,book_revision,policy_id,accounting_date,source_currency,functional_currency,fx_required,accounts}`. Accounts have account_id, code, name, category, normal_side, monetary. Maximum100 active accounts excluding source, ordered by code/UUID. |
| `banking.create_review_draft(actor,e,b,review uuid,key text,raw text)` | Exact `{expected_review_revision,source_revision,binding_revision,expected_book_revision,category_account_id,policy_id,accounting_date,description}`; returns Draft below. Body <=8KiB; key ASCII1..128. Accounting date must equal the immutable source booked date. |

Summary fields: review_id, state, review_revision, booked_date, currency,
currency_version_id, direction, amount, description, draft_id and journal_id;
last two nullable. Detail adds profile `source-review-v1`, source_account_id,
source_revision, binding_revision, book_revision, ledger_account_id, created_at,
blockers, evidence and nullable draft. Original evidence fields from import remain:
observation_id/revision, namespace_id, external_entry_id, source_state, booked_date,
currency_version_id, direction, amount and description. Detail flags overlap holds
and stale draft pins. Historical posted detail retains its original draft/journal.

Draft is `{draft_id,review_id,review_revision,source_revision,binding_revision,
book_revision,posting_body}`. Posting body is the complete existing PostingRequest
with two exact source/offset lines. Identity valuation has equal transaction and
functional amounts and no rate ID. The draft key is creator/book scoped and compares
full stored review/body identity; exact retry returns the original immutable draft
under current authorization before mutable state checks.

Cursor is bounded canonical hex-encoded JSON containing entity, book, source,
source_revision, complete state filter and last review UUID. SQL revalidates all
fields and current authority, rejects filter/scope changes, and returns
review_snapshot_changed when the source revision changed. It is an opaque position,
not an authorization token; tampering cannot grant source access. Pages are UUID
ordered, at most100; no unbounded dataset download or application-side filtering.
Source/account selector overflow fails explicitly rather than silently truncating.

Error codes: P2500 invalid_review/400; P2501 review_denied/404; P2502
review_conflict/409; P2503 review_snapshot_changed/409; P2504 fx_required/422;
P2505 review_limit/422. Existing posting P2100-series and source P2401 semantics
remain applicable. Messages contain no financial payloads or raw SQL diagnostics.

## Recoverable source posting

`source_post` shares the existing one-unacknowledged-principal command gate.
Its body is exactly `{review_id,draft_id,review_revision,source_revision,
binding_revision,expected_book_revision,posting_body}`. Including the complete
posting body makes remount/recovery human-reviewable; it is not an alternate
posting authority. SQL compares it and every pin with the immutable referenced
draft before registration or disclosure. Execution always uses the stored draft.

First execution rechecks current permissions, active source, draft_ready state,
review/source/binding/book pins and existing source link. It calls
`ledger.post_journal` inside the same transaction and writes the source snapshot
increment, posted review projection, immutable review_posting and committed command
audit/result. Any link/projection/command failure rolls the journal, lines, audit,
posting receipt, outbox and book revision back as well. There is no HTTP-post-then-link
sequence, worker posting or automatic confirmation.

Result is `{entity_id,book_id,review_id,draft_id,journal_id,book_revision,journal,
review_revision,source_revision}`; journal is the exact existing Journal response.
A committed command replays its exact result after current authorization without
rechecking mutable active/period state. Another command for the same immutable
already-posted draft returns the existing linked result; it cannot create a second
journal. A different historical draft conflicts. Direct manual journal duplication
is not solved by this link; manual-to-source matching remains separate work.
Two feeds representing opposite sides of an inter-account transfer also require a
future transfer-pair workflow; independently categorizing both is not semantic
deduplication. This slice does not automatically identify or match transfer pairs.

Cancel affects only the prepared command, leaving immutable draft history and
review state available for later explicit confirmation. A stale draft requires a
new explicit category draft/current pins, cancellation and acknowledgement of any
old prepared command, then new registration/confirmation. A committed command cannot
be cancelled. Acknowledgement releases the gate without deleting history. Unknown
commit means recover the same command, never generate a replacement key.

## Standards, sources and remaining gates

Applicability follows [standards](standards.md), existing SI01–SI06 import sources
and [posting policy sources](../research/posting-sources.md). This is an internal
workflow, not an ISO20022 financial message. ISO4217 currency-version/minor-unit
references and the existing ISO8601/RFC3339 date profile remain unchanged. Source
units and immutable provenance are retained; no ISO conformance or live accounting
approval is claimed. Financial exchanges, exports and accountant policy selection
remain under their dedicated contracts.

Mechanism review, 2026-10-13: existing DI01 and database source records, PostgreSQL17
[constraints](https://www.postgresql.org/docs/17/ddl-constraints.html),
[serializable transactions](https://www.postgresql.org/docs/17/transaction-iso.html)
and [owned function privileges](https://www.postgresql.org/docs/17/sql-createfunction.html)
selected sections, plus repository migration2's exact posting implementation.
Inference: one database transaction can bind source review and ledger mutation
without a service bus. Separate HTTP posting/link writes were rejected because a
partial failure can orphan a journal or duplicate it. A full arbitrary-line editor
would broaden validation/UI scope; fixed two-line category drafts retain exact
source legs while explicit FX/split support is designed separately.

Actual evidence is in the [runbook](../runbooks/source-review-sql.md). Remaining work:
FX/split drafts, overlap matching, source corrections/reversals, reconciliation,
real acquisition/ownership, hosted retention and live policies. A populated synthetic
demo demonstrates flow, not accounting cutover readiness.
