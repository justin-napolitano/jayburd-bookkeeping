# Provider synchronization SQL

Migration 25 [dated-currency history](plaid-history-currency.md) replaces singleton
source-version equality with bound monetary identity plus exact source-date version
validation. Complete-set atomicity and all posting gates remain unchanged.

Migration 13 extends this merged foundation through the
[connection lifecycle contract](connection-sql.md): provider-first binding,
credential references, individual account disconnect, and scoped worker context
and receipt recovery. Read that extension with this record; the original migration
12 enrollment remains an owner-only synthetic fixture boundary.

## Decision and applicability

Migration 12 implements the approved **synthetic acquisition foundation** for
#70/#100. A complete account-filtered change set publishes source evidence and its
cursor atomically. Acquisition never posts a journal. Dedicated `bookkeeping_sync`
logins cannot call interactive, publisher or observer operations. No provider calls,
credential retrieval, Link, webhook ingress, hosted dispatch or cloud deployment is
implemented here. This does not complete the provider epic or approve live books.

Reviewed against migrations 9–11 and official Plaid sections on 2026-10-13.
`jayburd-plaid-transactions/1`, API version `2020-09-14`, is proprietary provider JSON,
not an ISO 20022 message. PI01–PI08/STD-01 and existing ISO 4217 currency references
apply; dates/timestamps retain the [decoder](plaid-decoder.md) ISO 8601/RFC3339 profile.
No normative ISO conformance, ownership proof or new accounting policy is claimed.
The coordinator approved these interfaces before DDL. Existing migrations remain
unchanged; downgrade intentionally fails because evidence is forward-only.

## Identity and authority

`integration.sync_item` has a globally unique case-sensitive
(provider, environment, Item) tuple bound to one entity/book. It retains an opaque
credential-reference UUID/version, explicit consent principal/evidence/expiry,
generation and active/paused/disconnected state. References contain no token and
grant no secret access. This slice permits only plaid/sandbox/synthetic enrollment.
Credential exchange/rotation and real enrollment remain #101.

`integration.sync_stream` binds one Item/account filter to an existing approved
source and a typed provider namespace, a service principal and an actual worker
login. Namespace identity includes provider/environment/Item/account, never a
lease, credential version or connection UUID. Mapping and enrolled identities are
immutable, including through ordinary owner DML. No cursor adoption across filters.
Each claim captures the current source binding revision; every stage/publication
revalidates it. Later permission edits therefore invalidate old claims without
permanently stranding the stream.

Enrollment is **existing-source-only**. Its new namespace always carries
`overlap_review_required=true`; booked positive additions enter `overlap_hold`.
There is no overlap-release implementation here. Provider-first registration and
reviewed overlap release are separate work; this is not end-to-end provider posting.
The posted-correction test uses an explicitly documented synthetic owner fixture to
represent already-approved provider evidence, then invokes the real posting boundary.

Worker functions have no actor argument. SECURITY DEFINER entrypoints require
SERIALIZABLE, `session_user` equal to the enrolled login, current membership in the
NOLOGIN sync role and no direct/inherited privileged or API/owner/observer/publisher/
identity/migrator role. Each operation, including receipt replay, rechecks active
service and consenting principals, current entity/source read+import, active
source/namespace/Item and unexpired consent. SQL cannot retrieve credential refs
through the worker API. No table DML or private helper EXECUTE is granted.

Authority locks precede financial scope locks: UUID-sorted principals, entity and
memberships, book, source, Item, stream, set. Item pause/disconnect uses current owner
approval and the common book lock, so it can revoke even disabled consent/service
subjects without a reverse principal lock. Existing API/observer authorization also
rejects mixed sync membership. SERIALIZABLE failures retry the complete transaction.

## Operator contracts

Both functions below are SECURITY INVOKER, executable only by `bookkeeping_owner`.
The approving principal is an explicit attestation, distinct from authenticated
`session_user`; immutable operator receipts record both, body and result. Current
authority is checked before idempotent replay. Keys are 1–128 characters in
`[A-Za-z0-9._:-]`; JSON rejects duplicate/unknown/null required fields before casts.

`integration.enroll_sync_stream(actor uuid,e uuid,b uuid,key text,raw text)` takes:

```json
{
  "source_account_id": "uuid", "expected_binding_revision": "1",
  "provider": "plaid", "environment": "sandbox", "synthetic": true,
  "item_id": "opaque", "account_id": "opaque",
  "credential_ref": "uuid", "credential_version": "1",
  "consent_principal_id": "uuid", "consent_evidence_ref": "uuid",
  "consent_expires_at": "2099-01-01T00:00:00Z",
  "service_principal_id": "uuid", "worker_login": "synthetic_sync"
}
```

Existing source mapping/currency and explicit grants remain authoritative; this
function grants no permissions. It increments binding revision for the new namespace,
with at most 25 namespaces. Same Item reuse requires exact scope/consent/reference
agreement; duplicate account enrollment under a new key conflicts. Response:
`{stream_id,sync_item_id,namespace_id,source_account_id,binding_revision,item_generation}`.
UUIDs are strings; revisions are canonical decimal strings.

`integration.set_sync_item_state(actor,e,b,item uuid,key text,raw text)` takes
`{expected_generation:string,state:active|paused|disconnected}`. It increments Item
generation, abandons staged sibling attempts, clears their leases and retains the
committed cursors/evidence. Disconnected is terminal in this profile. Response:
`{sync_item_id,item_generation,state}`. Reactivation does not bypass current consent.

## Worker functions and transactions

All scope/stream/set arguments below are UUIDs; fences are bigint; keys are text
except the UUID claim key. Each call is one short transaction; network/decoder work
runs outside it. Worker EXECUTE is limited to these five functions:

| Function | Mutation and replay |
| --- | --- |
| `integration.request_sync(e,b,stream,key)` | Idempotently increments requested generation. Returns `{stream_id,requested_generation}`; same key returns its original generation. |
| `integration.claim_sync(e,b,stream,claim_key uuid)` | Captures C0, current Item/binding revisions and requested target; allocates monotonic fence and 90-second lease. Returns `{stream_id,set_id,fence,starting_cursor,next_page_index,request_cursor,target_generation,binding_revision,lease_until}`. Same live key revalidates all pins; expired keys reject. A new claim abandons an expired attempt and restarts C0. |
| `integration.stage_sync_page(e,b,stream,set,fence,raw)` | Requires contiguous page index and exact cursor chain; persists private immutable normalized page. Exact full-page replay succeeds; changed replay conflicts. Returns `{set_id,page_count,change_count,final_page}`. |
| `integration.complete_sync(e,b,stream,set,fence)` | Reauthorizes before stored-receipt replay; otherwise requires live pins, final page and full-set validation. Atomically publishes evidence/reviews/cursor/receipt. Returns `{stream_id,set_id,cursor,completed_generation,source_revision,changed_count,readiness}`; count/revision fields are strings. |
| `integration.abandon_sync(e,b,stream,set,fence,reason)` | Requires current live pins. Reason is pagination_mutation/provider_failure/decoder_rejected/cancelled. Clears attempt lease without advancing cursor; returns `{set_id,state:"abandoned"}`. |

Sets are staging/completed/abandoned (quarantined is reserved). Pages, completion,
signal and operator receipts are immutable. A signal racing completion increments
requested generation under the same lock; completion records only its captured
target. `requested_generation > completed_generation` remains durable pending work.
This is a future dispatcher input, not an implemented task transport/outbox handler.
There is no partial-set adoption, lease extension or autonomous retry scheduler.

Unknown commit response requires reconnecting and replaying the exact operation;
never assume rollback. A completion receipt is returned after current authorization
before checking expired/replaced leases or changed pins. Different/stale set/fence
cannot publish. No partial evidence survives an exception or transaction rollback.

## Page, money and bounded publication

The page JSON is exactly the [decoder Page](plaid-decoder.md) plus zero-based integer
`page_index` and nullable `request_cursor`. Fields are next_cursor,has_more,
transactions_update_status,request_id,sha256,added,modified,removed. Transaction rows
are exactly account_id,transaction_id,pending,pending_transaction_id,date,
authorized_date,datetime,authorized_datetime,iso_currency_code,currency_version_id,
amount_lexeme,amount,direction,eligible_for_review,name,merchant_name,
original_description. The decoder's nullable fields remain JSON null; removed rows
contain only account_id and transaction_id. There is no public HTTP row-ingestion API.

SQL revalidates account, installed source currency/version/date/minor units, exact
signed numeric lexeme and canonical unsigned magnitude without float or rounding.
Lexemes are at most 128 bytes, at most 38 significant coefficient digits and exponent
[-100,100]; magnitude fits numeric(38,9). Positive provider amounts mean outflow,
negative inflow, zero direction zero and ineligible. Original signed lexeme is stored
in `provider_amount_lexeme`; existing unsigned `amount_lexeme` semantics remain intact.
Optional text preserves empty strings; descriptions are inert text, not executable
content. Timestamps are validated and preserved without timezone conversion.

Engineering bounds: **10 pages, 1000 changes, 2 MiB normalized complete set;
500 changes/page, 90-second lease, 24-hour staged expiry**. Exceeding a bound rejects
without cursor advancement. Page index 10 is malformed, whole-set/page size overflow
is bounded overflow. These are project bounds, not provider guarantees or a claimed
50,000-change proof. Empty initialization is exactly one empty first/final page with
any known Item update status; it retains C0=null. Item readiness does not establish
account-specific history completeness. Migration 24 aligns this with the reviewed
[decoder correction](plaid-decoder.md#account-readiness-correction--2026-10-14).
An empty initialization response cannot terminate an earlier nonempty page sequence.

Private normalized pages retain raw SHA256 and request ID. Exact raw page bytes are
retained by the pure decoder in memory only; SQL has not seen or persisted those
bytes. Durable raw object storage, access, retention/purge and composition with the
future provider adapter remain separate work. Neither a digest nor normalized JSON
is claimed to reconstruct original evidence or satisfy a live retention policy.

## History, corrections and read projection

Duplicate transaction IDs across all arrays/pages reject the complete set. Existing
added IDs require full normalized-row equality; changed added IDs conflict. Unknown
modified/removed IDs reject. Unchanged modified/repeated removed facts are no-ops;
different IDs with identical money remain different observations. No value-based
CSV/Plaid matching or deduplication occurs; typed namespace identity plus overlap
holds protects against silent duplicate posting.

Immutable provider revisions reference their predecessor and set/page/operation.
Only the source observation's monotonic current pointer changes. Removal retains
previous money/date as historical context and explicitly sets removed state. Pending
replacement uses explicit same-namespace pending_transaction_id, resolved after all
pages; unknown/nonpending/self/reused lineage rejects atomically. IDs stay distinct.

Pending, zero and changed/removed unposted evidence are `correction_hold`; no draft
can be created. Source snapshot increments once when a set changes evidence, making
old drafts/approval pins stale. Book revision, journal, lines, posting receipts and
ledger outbox are never mutated by acquisition.

Posted changes retain posted state and the immutable journal/link. The first change
writes a direct immutable observation-revision/journal anchor; every later read keeps
that original posted evidence even after successive changes/removals. Exceptions
are separate references, not automatically corrected journals or substituted amounts.
Review detail optionally includes:

```text
provider_exceptions: {
  total_count: string, truncated: boolean,
  items: [{exception_id,kind:modified|removed,previous_revision:string,
           current_revision:string,observation_id,created_at}]
}
```

Items are latest 100 by current_revision descending then exception_id. Total count
and truncation are explicit. Full inspection of changed amounts/dates and correction
resolution are deferred; this projection exposes references only. List filters accept
correction_hold; zero source money remains a decimal string and cannot post.

`banking.get_source` retains its exact CSV namespace branch. Provider namespaces are
`{namespace_id,profile,provider,environment,item_id,provider_account_id,state,
overlap_review_required}`; no fabricated null CSV identifiers or credential refs.
This existing SQL read has no new HTTP route in this slice.

## Errors, sources and evidence

Fixed message `sync_rejected`: P2600 malformed input/isolation; P2601 current authority;
P2602 identity/replay conflict; P2603 stale attempt/pin; P2604 invalid/incomplete set;
P2605 bounded overflow. PostgreSQL serialization/deadlock/lock timeout is retried by
transaction owners; SQL does not swallow it or report an uncommitted receipt.

Reviewed primary [Plaid Transactions API](https://plaid.com/docs/api/products/transactions/)
sections transactions/sync, account filtering, amount/pending fields and cursor bounds;
[sync migration guide](https://plaid.com/docs/transactions/sync-migration/) complete-loop
restarts, 2026-10-13. Fact: filtered streams need separate cursors and failed pagination
restarts from the original cursor. Inference: private fenced staging plus atomic
publication is safer than partial cursor advancement or inferred source matching.
Existing PostgreSQL 17 DI01/function/locking references and the [database research](../research/database-foundations.md)
support scoped keys, immutable history and complete SERIALIZABLE retry. Review covers
selected sections, not provider qualification or normative ISO certification.

Actual tests and limits are recorded in the [delivery runbook](../runbooks/provider-sync-sql.md).
