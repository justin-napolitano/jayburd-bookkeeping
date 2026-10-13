# Synthetic durable import SQL

## Decision and boundary

Migration 10 implements the reviewed SQL subset of [durable import](durable-import.md)
for #92 under #8/#84/#85. Registration and preview create no observations. An explicit
`import` interactive command atomically creates immutable source evidence and
uncategorized review items. It creates no journal, line, posting receipt, outbox
message, valuation, reconciliation or book revision change. Go owns verified actor
identity, private byte storage, strict CSV decoding and transaction retries.

Coordinator reviewed signatures and publication trust before implementation. The
API database login is a trusted backend boundary: it already supplies verified
actor IDs. Its narrow publication function accepts server-produced decoded rows;
there is **no public HTTP row-publication endpoint**. SQL independently validates
all row keys/types, ordinal/ID uniqueness, source/currency/date equality, original
amount grammar and exact canonical scale. SQL cannot prove bytes it cannot read:
Go must read back the immutable private object, compare exact bytes on retry and
invoke the strict decoder. A digest or caller-supplied row array is not that proof.

Only synthetic `jayburd-synthetic-statement-csv/1` is accepted. The public projection
profile is `durable-import-v1`. No provider worker, bank connection, automatic
posting or live-account authority is introduced.

## Physical contract and privileges

All new book-owned tables have `(entity_id,book_id,...)` primary/foreign keys,
NO ACTION history references and owner `bookkeeping_owner` (NOLOGIN). Migration1–9
remain unchanged. `banking.source_namespace` gains a composite source/namespace
unique key. Source revision becomes nonnegative; binding edits increment only
binding revision, confirmed imports increment only source revision. Existing
immutable mapping and terminal retirement remain enforced.

| Object | Grain and invariant |
| --- | --- |
| `banking.import_batch` | Unique scoped namespace/statement reference; immutable source, binding revision, interval, profile/parser, server object UUID, actor and expiry. Original complete publication JSON, digest and byte count cannot be replaced. State received/preview_ready/rejected/imported/cancelled; imported body and result persist together. |
| `integration.import_registration` | Immutable actor/scope/idempotency-key receipt naming exact metadata and batch. An independently keyed request for the same statement reuses the batch only when metadata agrees. |
| `banking.import_preview` | Immutable scoped batch/revision result with pinned source/binding revisions, validated rows or bounded rejection diagnostics, counts and timestamp. Latest revision is selected by descending revision. |
| `banking.source_observation` | Immutable scoped source/namespace/external-entry identity; unique also across entity/source/namespace/entry. Current revision is fixed at 1 in this profile and has a deferred FK to its immutable revision. |
| `banking.observation_revision` | Booked evidence: date, currency version, direction, positive exact `numeric(38,9)`, original amount lexeme, description and full canonical equality object. No pending, modified or removed profile yet. |
| `banking.import_row` | Immutable scoped batch/physical-record occurrence linked to observation revision, retaining original decoded fields and money lexeme even when an earlier file already supplied the observation. |
| `banking.review_item`, `review_decision`, `review_evidence`, `review_assignment` | Immutable first-profile projection and initialization decision. One assignment per observation. First namespace initializes needs_review; later overlapping namespaces initialize overlap_hold. No matching/drafting/posting transition is implemented. |

Runtime API receives EXECUTE only on the five new functions below and existing
`get_source`. Owner helpers, operator grants and every table remain inaccessible
to API/observer/publisher roles. Functions use a fixed trusted search path. Existing
setup/post command permissions and worker boundaries are preserved.

`ledger.membership` explicitly permits new `import` authority. Operator-only
`banking.set_source_import_permission(actor,e,b,source,key,raw)` accepts exactly
`{expected_binding_revision,principal_id,enabled,entity_import_enabled}`. Revisions
are canonical decimal strings; booleans independently attest source and entity
import grants. Existing entity/source read grants are not created implicitly.
The function requires explicit owner role, current active approving principal and
entity read, active grantee/read membership for any grant, and a matching binding
revision. Pure revocation can remove dormant grants for disabled/revoked readers.
Receipt records authenticated `session_user` separately from attested `approved_by`.
No self-grant through an API function or initial book setup exists. Changing entity
import authority affects all sources in that entity; source grants remain separate.

Import mutations require SERIALIZABLE and current entity **read+import** and source
**read+import**, including replay, pending discovery, cancel and acknowledgement.
Public history reads require current entity/source read under REPEATABLE READ or
SERIALIZABLE. Authorization serializes with principal/entity/membership/book/source
grant changes; command operations take the strongest principal lock first. The book
lock serializes writes without changing its financial revision. Source locks serialize
namespaces and deduplication. All I/O occurs outside SQL transactions.

## Exact service functions

Every function returns `jsonb`; IDs are UUIDs, money/revisions/counts strings.
`actor,e,b` are UUID arguments supplied by trusted server identity and route scope.

| Function | Exact boundary |
| --- | --- |
| `banking.register_import_batch(actor,e,b,key text,raw text)` | Body `{source_account_id,namespace_id,binding_revision,statement_ref,starts_on,ends_before,profile}`; returns service context. Metadata/key <=8 KiB/128 ASCII characters. Existing receipt reauthorized before replay. |
| `banking.import_context(actor,e,b,batch uuid)` | Requires import authority. Returns public batch fields plus decoder profile, opaque object_id, source_account_ref, starts_on, ends_before, and currency `{code,version_id,minor_unit,valid_from,valid_to}`; absent currency upper bound is empty string. Cancelled/expired/unavailable unimported data is denied before storage I/O. Received/preview_ready additionally require active source/namespace and unchanged binding. Imported exact replay retains current authorization but does not require current active/pinned state. |
| `banking.publish_import_preview(actor,e,b,batch uuid,raw text)` | Trusted service only. Success input `{profile,sha256,byte_count,rows}`; rejection input `{profile,sha256,byte_count,diagnostics,total,truncated}`. Returns public batch. SQL-owned source metadata rechecked; complete publication equality checked on retry, even if hashes agree. |
| `banking.get_import_batch(actor,e,b,batch uuid)` | Read-authorized public projection, no private object reference or bytes. |
| `banking.get_review_item(actor,e,b,review uuid)` | Read-authorized first-profile review and immutable evidence projection. No write authority. |

Success rows are exactly `{record,source_account_ref,entry_id,booked_on,currency,
direction,amount,description,canonical_amount,currency_version_id}`. `record` and
`byte_count` are bounded JSON integers; `amount` retains the decoder lexeme and
`canonical_amount` is its exact decimal normalization, never float or rounded.
Records are 2..1001 in order, IDs follow the decoder ASCII grammar, descriptions
are 1..500 printable ASCII characters. Dates must be real Gregorian full dates
inside both batch interval and pinned currency validity. Significant fractional
scale cannot exceed pinned minor units; JPY zero and KWD three are supported.
At most 1000 rows, 1 MiB stored bytes and 2 MiB publication JSON; row-limit failure
is atomic. Rejection diagnostics are exactly `{code,record,field}`, using fixed
decoder codes, at most 100 retained entries and 10000 counted detections. SQL adds
`source_correction_required` when an existing observation differs materially.

Public batch contains profile, batch/source/namespace IDs, state, binding/source
revision, preview_id/revision, created_count/existing_count (null before preview),
diagnostics/error_count/truncated, raw_available, created_at/expires_at, and nested
`preview` (null or `{preview_id,preview_revision,binding_revision,source_revision,
created_count,existing_count}`). Top-level source_revision is current; **confirm
using the nested preview's pinned revisions**, not a newly observed top-level value.
`raw_available` means metadata published and no recorded deletion, not proof that
an object is physically retrievable. No raw URL/path is returned.

The bounded review projection contains profile, review/source IDs, state,
review/source/binding revisions, created_at, null draft_id/journal_id, blockers and
evidence. Each evidence entry contains observation/revision/namespace IDs,
external_entry_id, source_state, booked_date, currency_version_id, direction,
canonical amount and description. The broader parent contract's updated_at,
adapter/raw availability fields, pagination and mutable review operations remain
unimplemented; this SQL projection is not the final public review API.

Fixed SQL errors: P2400 malformed input; P2401 uniformly denied/inaccessible scope;
P2402 identity/state conflict; P2403 stale binding/preview/source; P2404 bound or
revision exhaustion. No source field appears in error messages. Invalid decoder
files and source correction conflicts publish a successful rejected batch with
safe diagnostics rather than partially importing rows.

## Confirmation, replay and stale work

Existing `ledger.register_command` accepts operation `import`, non-null entity/book,
and body exactly `{batch_id,preview_id,preview_revision,binding_revision,
source_revision,profile}`. Existing execute/cancel/acknowledge calls still take `{}`.
The one-unacknowledged-principal gate covers setup/post/import together. Its
immutable command body, terminal result and action audit supply operation receipt
and recovery: no separate immediate-import function is granted to runtime.

First execution checks current active source/namespace, latest preview, exact pins,
expiry and all canonical existing-row values. One transaction creates every new
observation/revision, every provenance link, every initialization decision/group,
one source revision increment, imported batch result and committed command/audit.
Even a provenance-only batch increments source revision because durable evidence
changed. Result is `{entity_id,book_id,batch_id,created_count,existing_count,
source_revision}` with counts/revision strings. An existing committed command
returns its exact result after current authorization and before mutable checks.
A new command targeting an already imported batch requires the original exact body
and returns that result without additional effects.

Same statement/metadata returns the same object reference; private storage compares
actual bytes. New files with canonical-equal source IDs create provenance only.
Canonical equality excludes only physical record position and original amount
lexeme; all remaining decoded fields, exact amount and currency version compare.
Different IDs with identical purchases remain different observations. A changed
existing ID rejects the **whole batch**, including otherwise valid new rows; changing
statement_ref does not authorize correction. A new namespace preserves both source
identities and places new evidence into overlap_hold, never similarity auto-match.

Source revision changes require publishing the same immutable decoded evidence
again to create a new preview revision, then explicit new confirmation. Binding
revision belongs to immutable registration metadata: grant, namespace or state
changes require a newly registered statement/batch with current binding, not silent
repinning of an old batch. Authenticated read history remains available.

Reviewed cancellation refinement: a prepared command always becomes terminal
cancelled; it cancels its batch only if its pinned preview UUID is still latest.
If refresh won first, cancelling/acknowledging the obsolete command preserves the
new preview for explicitly registered confirmation. If cancellation of the current
preview won first, the batch is terminal and later publication fails. Executing any
cancelled command fails. A committed batch cannot be cancelled. Source/book locks
and SERIALIZABLE retries establish these orderings without crossing object I/O.

Unknown commit must recover the same command; retry only whole transactions for
serialization/deadlock within the caller's bounded retry budget. Do not recreate
keys or repeat object side effects inside a SQL retry.

## Sources, alternatives and open gates

Standards applicability follows [STD-01](standards.md), [decoder](statement-decoder.md)
and [SI01–SI06](../research/statement-import-sources.md). This internal synthetic
profile is not an ISO 20022 financial exchange. Currency identifiers/versioned
minor units use the reviewed ISO 4217 mapping; Gregorian dates follow the existing
ISO 8601/RFC 3339 usage profile. No ISO conformance or accounting-framework approval
is claimed. Source units are preserved without inventing FX or posting policy.

Reviewed 2026-10-13: PostgreSQL17 [§5.5 keys/foreign keys](https://www.postgresql.org/docs/17/ddl-constraints.html),
[§13.2.2–3 isolation and full transaction retry](https://www.postgresql.org/docs/17/transaction-iso.html),
and [CREATE FUNCTION, SECURITY DEFINER/search_path/privileges](https://www.postgresql.org/docs/17/sql-createfunction.html).
These primary sections corroborate scoped declarative constraints, snapshot/locking
behavior and restricted owned functions (existing DI01/database source records).
Inference: immutable evidence plus source revision pins is sufficient here; a
second event store is unnecessary. Hash-only identity, similarity deduplication
and immediate import-to-journal were rejected because they lose evidence or bypass
review. Exact SQL money validation complements the existing Go decimal-string
normalizer; SQL does not duplicate the CSV byte parser. Source documents were read
in selected sections only; no additional normative ISO text was acquired.

Evidence and exact commands are in the [living runbook](../runbooks/import-sql.md).
Retention metadata persists but no cleanup/tombstone reconciler is implemented.
Seven-day unconfirmed expiry is an execution gate, not deletion. Hosted encrypted
storage, raw retention/holds, provider acquisition/corrections, review decisions,
source-to-draft/post, reconciliation and live approval remain separate gates.
