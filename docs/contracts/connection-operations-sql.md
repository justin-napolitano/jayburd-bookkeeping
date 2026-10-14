# Connection operations SQL

## Decision and reviewed sources

Proposed migration 14 extends existing generation-fenced acquisition. Human operations,
verified notification intake and worker catch-up create durable generation requests;
none fetches provider data or posts a journal. The coordinator reviews interfaces
before DDL. Migrations 1–13 remain immutable.

Reviewed 2026-10-14: [Plaid verification](https://plaid.com/docs/api/webhooks/webhook-verification/)
algorithm/key/signature, issued-at and exact-body SHA-256 sections; [Transactions API](https://plaid.com/docs/api/products/transactions/)
SYNC_UPDATES_AVAILABLE and synchronization readiness; PostgreSQL 17
[transaction isolation](https://www.postgresql.org/docs/17/transaction-iso.html)
Serializable retry requirements. Review covers these public sections only, not live
provider qualification or normative ISO certification. PI02/PI04/PI08 in the
[annotated source register](../research/plaid-ingestion-sources.md) remain applicable.

Facts: verification authenticates exact bytes and a signed issue time; sync notifications
indicate available work, not locally published history. Inference: deduplicate the
canonical ES256 signed-delivery identity, not body alone (a later identical body may signal new
work); commit the receipt with generation increment and outbox intent. Alternative
of a separate queue state machine is rejected: the existing requested/completed
generations and fenced sets already own durable acquisition progress.

Plaid JSON remains proprietary, API 2020-09-14. Internal operations have no applicable
ISO 20022 business-message designation. Existing ISO 4217/RFC3339 source semantics
are unchanged; no new money conversion, accounting policy or conformance claim.

## Interactive recovery and status

All public functions use actual API role, authenticated actor, entity/book UUIDs and
current read/connect/import plus source read/import/bind where a source exists.
`list_connection_attempts(actor,e,b,maximum,after_id)` limits 1–100, ascending UUID
keyset, returning `{items:[ConnectionAttempt + expired_unclaimed:boolean + binding_expired:boolean],
next_cursor:uuid|null,unfinished_count:string}`. Only the same actor's attempts are
visible. Private nonce, fingerprints, claim and credential references never appear.
The unfinished count covers all actor attempts, independently of pagination. Active
and disconnected are terminal; an expired attempt without a provider claim is not
unfinished. Expired binding_review is a known published credential outcome and is
also excluded, with binding_expired=true; its old paused Item and reference remain
retained, and binding still rejects expiry. This permits fresh enrollment without
repeating exchange or silently removing provider access. Claimed ambiguity in other
states stays unfinished even after handoff expiry.

A forward `begin_connection` wrapper authorizes and locks the book, preserves exact
existing-key replay, and rejects a fresh new enrollment while any unfinished attempt
exists for this actor/book. Concurrent fresh requests serialize under the canonical
book lock; whole Serializable retries remain required. Expiry never grants permission
to repeat a claimed provider operation. Existing known-attempt reconciliation is
unchanged. There is no new cancellation or remote-operation recovery bypass.

`connection_stream_status(actor,e,b,stream)` returns:

```text
{stream_id,source_account_id,sync_item_id,state:active|paused|disconnected,
 stream_revision:string,item_state,item_generation:string,binding_revision:string,
 requested_generation:string,completed_generation:string,queued:boolean,
 lease_active:boolean,last_completed_at:timestamp|null,readiness,last_error,
 provider_updated_at:null}
```

Readiness is exactly TRANSACTIONS_UPDATE_STATUS_UNKNOWN, NOT_READY,
INITIAL_UPDATE_COMPLETE or HISTORICAL_UPDATE_COMPLETE from complete publication.
Queued means requested generation exceeds completed generation, not worker dispatch
or current authority to run. Lease_active reflects an unexpired local lease only.
Last complete timestamp comes from SQL publication; no provider-updated timestamp is
available. Last error is only the latest set's known abandonment reason, or null:
pagination_mutation, provider_failure, decoder_rejected, cancelled, item_changed,
lease_expired, account_disconnected, account_paused. Never echo upstream diagnostics.

`mutate_connection_stream(actor,e,b,stream,key,raw)` requires exact JSON:
`{action:pause|resume|request,expected_item_generation:string,
expected_binding_revision:string,expected_stream_revision:string}`. Keys use the
existing 1–128 ASCII idempotency profile. It returns an immutable status snapshot;
clients refresh current status after receipt recovery. Current caller authorization
precedes exact receipt replay; changed input under the same actor/book/key conflicts.
First execution verifies all three pins.

Pause affects only this stream: abandon held set, clear lease, increase fence and
stream_revision; preserve cursor, source binding and sibling acquisition. Resume
requires current consent, namespace, service/source grants and active Item; atomically
changes stream state/revision and requests a generation. Request requires the same
active acquisition authority. Paused/disconnected cannot acquire; disconnected is
terminal. Stream revision starts at 1 and advances only pause/resume; source approval
pins and financial drafts are not changed by these operations.

## Durable signal and verified ingress

Existing `request_sync(e,b,stream,key)` keeps its wire and actual enrolled worker
identity checks, now commits signal plus `integration.sync_wakeup` generation fact.
The new immutable outbox has entity/book/stream/requested_generation primary key,
globally unique message UUID, closed reason and database creation time. Reasons are
worker/manual/resume/webhook/catchup. Existing sync_signal gives stable-key replay.
The journal outbox cannot safely host this event: its target FK and uniqueness are
specific to journal entries. No observer queue tables, lease engine or dispatcher are
repurposed. Current requested/completed generations remain the work authority.

`bookkeeping_webhook` is a new NOLOGIN role with exactly one runtime function. Its
actual login may not inherit API/owner/sync/observer/publisher/identity/migrator or
privileged roles. Owner-only `set_webhook_ingress(login name,environment,key,enabled)`
binds the actual session login to sandbox. The operator receipt records session_user,
exact input and resulting revision. Disabling this configuration stops new ingress;
it does not revoke already authorized account workers. Mixed-role guards extend
sync login, ledger authorization and observer binding authorization forward.

`accept_verified_webhook(environment,receipt_key,raw)` is callable only by this
configured ingress login, in Serializable transactions. Trusted verifier JSON is:

```json
{"item_id":"provider-case-sensitive-id","body_sha256":"64-lowercase-hex",
 "issued_at":1791936000,"webhook_type":"TRANSACTIONS",
 "webhook_code":"SYNC_UPDATES_AVAILABLE"}
```

Receipt key is SHA-256 of the verified header/payload plus canonical ES256 signature
R || min(S, N-S), encoded as the signing profile specifies. This prevents equivalent
ECDSA signature representations from creating another signal. Go must verify signature, trusted
key, exact raw body, age and bounds before SQL; storing a hash does not prove ownership
or replace cryptographic verification. SQL stores only bounded validated metadata,
not body/JWT/token. Initial SQL acceptance checks age at most 300s and future skew
at most 60s. Receipt replay still requires current ingress/Item/source/consent/service
authority; original age does not defeat database commit recovery. HTTP verification
continues to enforce its own current-age policy on each delivery.

SQL resolves the provider/environment/Item tuple; the caller supplies no actor or
entity/book. Up to 25 currently eligible bound streams receive one signal each,
with the receipt and outbox facts in the same transaction. Unknown/disabled/noeligible
Item uniformly denies without a receipt. Paused or revoked sibling bindings do not
receive work. Replay rechecks every originally accepted stream before duplicate
acknowledgement; a different valid later JWT with the same body remains a new signal.
Response `{state:accepted|duplicate,receipt_id:uuid,stream_count:integer}` is private
service metadata; public HTTP acknowledgement need not disclose it.

## Catch-up and limits

`catchup_sync(e,b,tick uuid,maximum integer,after_id uuid|null)` is sync-role-only.
It scans up to 100 current eligible streams owned by the actual worker login within
trusted injected scope, returns `{items:[{stream_id,requested_generation:string}],
next_cursor:uuid|null}` and requests through the same signal/outbox helper. Ineligible
streams are skipped; no caller actor or credential reference is returned. Stable tick
plus stream identity is one durable increment across retries and page-size changes.
Newly inserted streams before a page cursor are picked up by a subsequent tick; this
is bounded discovery, not a frozen full-scope snapshot or scheduler implementation.

All affected principals are locked in UUID order before entity/book/source/Item/
stream locks. Signal/completion share those locks: completion publishes only its
captured target, preserving later requested generations. No financial row is touched
by any operation in this migration. Errors are fixed connection_operation_rejected:
P2800 malformed/isolation, P2801 authority, P2802 conflicting replay, P2803 stale
revision/time, P2804 unfinished/invalid state, P2805 bound. Existing P270x/P260x
current-authorization failures retain their classifications. Retry serialization and
deadlock as whole transactions; unknown commits must resolve by the same key.

Physical receipt/outbox tables are owner-owned, runtime DML is absent and immutable
triggers retain receipts. Raw provider evidence retention, delivery transport,
webhook deployment, scheduled execution and account-specific retry/backoff policy
remain separately qualified. A durable signal is not a dispatched task or a claim
that account balances are current.
