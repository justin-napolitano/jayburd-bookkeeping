# Background work contract

## Decision and implementation boundary

**Proposed #63 design, 2026-10-13; implementation belongs to #7 under #57.**
Use PostgreSQL transactions for accounting and durable work state, Cloud Tasks
for directed HTTP wakeups, and bounded Go handlers. Pub/Sub remains deferred
until independent subscribers justify it. Sources BW01–BW05 and alternatives are
recorded in [background-work sources](../research/background-work-sources.md).
Provider acquisition details belong to #70; this document owns generic delivery,
identity, fencing and recovery requirements. No worker is implemented here.
The first #7 consumer is strictly `journal.observe` version1: reload one committed
journal/event pair and record durable consumption, with **no new posting**. This
is not a generic workflow engine; SQL allows only explicitly reviewed handler
versions, with provider acquisition a separately gated #70 extension.

Current migrations implement immutable `integration.outbox_message` rows:
`entity_id, book_id, message_id, event_type='journal.posted', event_version=1,
target_id, book_revision, created_at`; unique target per book, inserted in the
same transaction as journal, posting receipt and audit. No publisher, delivery
status, consumer receipt or lease exists. Do not UPDATE these rows or broaden
their event-type check to simulate jobs. `integration.operation_receipt` only
covers immutable post/reverse receipts. `ledger.interactive_command` covers
user setup/post recovery and one unacknowledged command per principal; neither
is a generic job table. New objects/functions require a reviewed forward migration.

## Facts, commands and applicability

`journal.posted` version 1 is a fact that journal `target_id` committed at
`book_revision` in its scope. Reversal journals also emit it; consumers reload
the immutable journal to learn its kind/reversal link. It contains no permission,
posting instruction, amount or source document. Reordering is allowed. The ledger
revision also advances for setup; consumers cannot assume contiguous event revisions
or wait for a missing event for every revision integer.

A directed command requests a named, versioned handler to perform work. Proposed
transport JSON `work-wakeup-v1` has exactly:
`{"schema_version":1,"operation_id":"<uuid>","dispatch_generation":"<positive integer>"}`.
Maximum 1024 UTF-8 bytes; reject unknown/duplicate keys, null, wrong versions and
noncanonical UUID/integer strings. It contains no amount, actor, entity, provider
token, cursor, account identifier, URL or arbitrary handler name. Load those
references from authorized durable records. Queue metadata is never business
identity. Command UUID is immutable across automatic retries and redispatch;
queue task IDs change only with persisted dispatch generation. A handler/version
and canonical request digest are fixed on the durable command at creation.

| Message/practice | Standards assessment and mapping |
| --- | --- |
| Internal `journal.posted` v1 | ISO 20022 external message encoding not selected/applicable for this internal committed-ledger notification: no bank counterparty or registered profile claimed. Preserve scoped journal UUID, book revision, event version and timestamp; do not call it a camt statement. BW05 reviews FAQ/methodology, not every message definition. This is a proposed bounded applicability interpretation requiring STD-01 review. |
| `work-wakeup-v1` | Internal scheduling envelope, not a payment/report instruction. No financial ISO message identifier/namespace invented. Validate exact version/IDs; money/currency mapping occurs only in the owned financial operation, never this envelope. |
| Financial effects or provider messages | Existing accounting, multicurrency and standards contracts remain authority. ISO 4217 versioned currencies and applicable ISO 20022 profile mappings are required in those operation contracts. Generic retries do not approve missing mappings or automated posting. |
| Worker/audit receipts | Internal execution evidence. ISO 21378:2019 / ISO 21926:2026 audit exchange mapping unresolved; no audit-export conformance claim. Preserve stable provenance for future mapping without inventing normative fields. |

STD-01 review of the above scope and named version/replay tests are implementation
gates. Normative accounting/ISO export gaps block the corresponding financial
adapter, not safe internal contract review.

## Proposed durable records and uniqueness

These are logical SQL contracts, **not approved physical DDL**. All scope keys
must participate in foreign keys/uniqueness; UUIDs alone never authorize access.

| Proposed record | Required meaning and key |
| --- | --- |
| `integration.work_operation` | UUID, handler/version, canonical request digest, authority reference, scoped resource/connection reference, creation time, state, business attempt count, next eligible time, deadline, current worker/fence/lease and terminal outcome reference. Financial request bodies stay in their owned protected store. Unique semantic key prevents duplicate operation creation. |
| `integration.outbox_delivery` | Sidecar linking event UUID + consumer/version to exactly one work operation; uniqueness on that tuple. A configured consumer is mandatory before fanout. Unconsumed historical events remain intact; do not mark all old rows delivered implicitly. |
| `integration.work_dispatch` | Operation UUID + dispatch generation; queue/target configuration version, deterministic task-name digest, pending/publishing/enqueued/uncertain/retired state, enqueue attempts, last attempt/result time, publisher fence/lease. No completion semantics. |
| `integration.work_receipt` | Immutable outcome per `(operation_id, execution_generation)` with request digest, handler/version, fence, scoped result references, fixed outcome code, timestamps and authorization evidence version. Successful terminal effect has a unique operation completion guard, regardless of queue generation. |
| `integration.work_attempt` | Append-only bounded metadata: operation, attempt UUID, fence, start/end, classified outcome, duration, no raw exception/provider body/credential/cursor. |

A stable semantic key for event work is `(scope,message_id,consumer_id,handler_version)`;
for connection work it is `(connection_id,connection_generation,target_generation,handler_version)`.
Same key/different digest rejects conflict. Existing receipts are never overwritten.
New handler versions do not automatically replay historical financial effects:
consumer rollout/backfill needs an explicit reviewed replay scope and effect key.

Work state transitions:
`ready -> running -> succeeded | retry_wait | blocked | cancelled | quarantined`;
`retry_wait -> running` when eligible; expired `running` may be reclaimed only
under a new fence. `succeeded` and `cancelled` are terminal. `blocked` requires a
specific corrective action (for example renewed consent), not timer-only retry.
`quarantined` requires explicit authorized retry/replacement with reason and audit.
Do not delete terminal receipts or reset counters to conceal failures.

## Identity and authority

Separate publisher identity (read/claim/ack dispatch and enqueue configured queue),
Cloud Tasks invocation identity (invoke exact handler/audience), and consumer
identity (owned claim/commit functions and required provider access). No worker
inherits database owner/migrator, unrestricted ledger tables or arbitrary URL
invocation. Runtime IAM/SQL grants remain a reviewed #7/#26 deployment gate.
OIDC verification pins issuer, audience and permitted service principal; Cloud Run
invoker IAM plus application binding to the configured handler protects invocation.
Task headers/name alone are not authentication. Invalid caller/envelope gets
401/403/400 and no business claim; bounded telemetry alerts on repeated rejects.

A service token authenticates workload, not authority over an LLC. Reload current
entity/book enablement and the operation's authority before claim, before each
external call and in the commit transaction. User-delegated operations require
current principal/membership; scheduled connection work uses a separately approved,
revocable connection mandate and permitted account scope. An event's original
poster is provenance, not perpetual delegated authority. Do not invent membership
from an OIDC subject or allow the service to post merely because it can fetch data.
The first consumer needs an explicitly provisioned service principal with scoped
read authority, separate from the original poster; provisioning/grants remain
implementation gates. No authorized mandate means no dispatch for that work type.

Revocation updates the same authority/resource rows locked by claim/commit and
increments connection generation/fence where applicable. A call authorized before
revocation may already be in flight; cancellation is best effort. Recheck after
return and discard/stage-clean according to #70 policy, never publish protected
results under revoked scope. A commit authorized/locked first may finish before
revocation commits; later work fails. Existing completed receipts may suppress
reexecution after revocation but return only a fixed empty task acknowledgement,
never old financial output to the caller. Terminal receipt lookup itself is scoped
to the authenticated configured consumer.

## Publisher and fast-consumer race

1. A business producer commits owned state and work intent together. Journal
   consumers derive unique sidecars/operations from immutable outbox facts in a
   bounded repeatable scan; absent sidecar is discoverable after a crash.
2. Claim due dispatch rows ordered by eligible time/UUID using short transactions
   and `FOR UPDATE SKIP LOCKED`. Increment publisher fence, set database-clock
   lease, commit. Never keep the transaction open during enqueue.
3. Create task with a deterministic hash of operation UUID + dispatch generation
   + configured target version. Resolve target from allowlisted config, never data.
4. On enqueue acknowledgement, compare-and-set the matching publisher fence and
   dispatch generation only. **Never set work state to running/succeeded or back
   to ready here.** A fast consumer may finish before the publisher records enqueue.
   Its durable receipt wins; a delayed publisher acknowledgement cannot undo it.
5. Enqueue timeout is uncertain, not proof of absence. Retry the same task identity
   first. `ALREADY_EXISTS` may describe an active or recently deleted task: it is
   deduplication evidence, not business success or proof a future delivery exists.
   Record that distinction. If work remains due, recovery can issue a new persisted
   dispatch generation under its lock; old/new deliveries still deduplicate by
   operation/effect identity.

Publisher leases/fences protect dispatch metadata only. They are distinct from
consumer fences and must never authorize business completion. Lease expiry is
eligibility for takeover, not proof the old network request stopped.

## Consumer fencing and receipts

Claim by operation UUID under current authorization. A successful receipt returns
204 without effect; terminal denial returns 204 only after durable outcome recording.
An active unexpired claim returns 503/Retry-After without consuming a business
attempt. An eligible claim increments the resource fence and business attempt,
sets `running`, increments the work `execution_generation` (starts at0, first
claim1; monotonic for every accepted claim), and sets the database-clock lease,
then commits. One immutable receipt per execution generation records a bounded
step/terminal outcome; the separate unique successful-operation guard prevents
two generations from completing the same effect. Resource-level serialization
is required for connection work: two
operations on one connection cannot each hold an effective fence.

Database time is authority; use a fresh server time when checking lease, not a
worker clock or the transaction-start timestamp after a long wait. Heartbeat
requires matching holder, fence, generation and unexpired lease; expired holders
cannot revive themselves. After a failed heartbeat or request cancellation, stop
new external calls. Commit always validates the fence even if cancellation was lost.

For a local effect, one transaction locks authority/resource/work rows in a
consistent reviewed order, checks holder/fence/unexpired lease/current generation,
request digest and eligibility, applies the **owned** operation, writes its receipt
and marks completion. Any failed predicate rolls back all domain state/checkpoints.
When an effect uses ledger posting, it must invoke its owned SQL operation and
idempotency contract with explicitly approved authorization; never duplicate
posting rules in Go. No automated financial mutation consumer is approved by this task.
For `journal.observe` v1, verify event scope, target UUID, type/version and book
revision equal the immutable journal header, then insert the unique consumption
receipt in the same authorized transaction. A mismatch quarantines without any
ledger write. Its only effect is the receipt; no additional outbox or posting.

External side effects cannot become atomic with SQL by writing a receipt afterward.
Require provider idempotency key based on stable operation identity, or a durable
`uncertain` attempt and status reconciliation before retry. When neither exists,
quarantine; never blindly repeat an irreversible call. Initial #70 acquisition is
read-only and cannot initiate payments or change provider accounts.

204 acknowledgement follows committed completion, durable terminal outcome, or a
committed continuation handoff. A committed continuation means the bounded step
ended but the operation may remain incomplete; work status must say so. If receipt
commit fails after a response was prepared, return non-2xx. If commit result is
uncertain, reread receipt under fresh authorization before reexecuting. Crashes
between effect/receipt/ack are named fault-test boundaries, not exactly-once claims.

## Per-connection catch-up interface with #70

In #70 a connection is one approved provider account stream (an Item/account
binding), not automatically an entire provider Item. Shared credential grouping
does not widen authority. One effective writer/fence per connection stream.
Distinct counters: `connection_generation` changes on reconnect/revocation/scope
reset; `fence_token` changes on worker takeover; `requested_generation` increments
for accepted wakeups/catch-up requests; `target_generation` is captured by a run;
`completed_generation` advances only after complete accepted acquisition. None is
`ledger.book.revision`. Provider `source_revision` advances only on published
source changes and is not a bookkeeping revision or posting approval.

Webhook/scheduler commits `requested_generation += 1` and ensures durable eligible
work in one transaction. Coalescing reduces queue tasks, not this durable demand.
A run captures the target then follows #70's staging/cursor contract. A partial
page can persist a protected staging continuation plus next-step work atomically;
it does not advance committed provider cursor or completed generation.

The complete-update commit validates connection generation/fence/lease, current
mandate/scope, expected committed checkpoint and batch completeness; publishes
normalized changes, checkpoint, source revision, operation receipt and completed
generation atomically. While holding the connection lock, if requested generation
exceeds completed generation, create/ensure the next eligible operation. A wakeup
arriving afterward takes the same lock and creates work itself. Thus neither race
can silently erase a wakeup. Duplicate external notifications may cause an empty
catch-up, never duplicate accounting effects. Provider-specific changed/removed
records, cursor invalidation and mutation-during-pagination belong to #70.

## Bounds, retry, queue loss and operator recovery

Proposed initial settings require synthetic measurement and pool-budget review:
publisher batch <=25, at most one enqueue call at a time, 30-second publisher
lease; handler work budget 60 seconds/<=10 provider pages, each provider call <=20
seconds (also bounded by remaining handler budget); consumer lease 90 seconds,
heartbeat <=30 seconds; Cloud Tasks deadline 90 seconds. SQL transactions use
5-second statement and 1-second lock timeout. Enqueue RPC <=10 seconds. No lease
renewal permits exceeding the handler budget. Durable continuation splits longer
acquisition; #70 owns complete-update size/age caps. Consumer pool max1 per instance,
max1 instance, queue concurrency1 preserve the accepted #7 baseline and must fit the existing
shared database budget including overlapping revisions. No deployment tuning claim.

Accepted claims increment `attempt_count`; failed attempts separately increment
`failed_attempt_count` exactly once when ending retryably/permanently or when
an expired running attempt is atomically abandoned during takeover. Delivery
duplicates, authorization rejects and active-lease contention increment neither.
Successful continuations increment attempts but not failures. Append-only attempt
identity prevents two recovery scanners counting the same crash twice.

Application retry is durable: jittered exponential delay starting 5 seconds,
cap15 minutes, honor longer provider Retry-After, maximum8 failed business attempts
or24 hours elapsed since first claim, **whichever comes first**, then quarantine.
Successful bounded continuation does not count as a failed attempt; provider batch
limits still prevent endless work. Distinguish transient timeout/429/5xx from
permanent malformed/version/semantic errors and blocked consent. Administrative
retry records reason and an explicit additional attempt/age budget, rechecks
authority and allocates a new execution generation on the next claim. Cumulative
counters/history and semantic effect key remain unchanged; completed operations
are never retried as new work. A reviewed budget grant, not a counter reset,
allows an operator to resume quarantined work.

Cloud Tasks retry is delivery assistance, not this attempt budget. Google currently
documents attempt/duration stopping conditions jointly; do not assume an OR cutoff
from queue configuration. Hard application limits are checked in SQL even if the
queue keeps delivering. Use bounded backoff settings reviewed with the runtime;
no reliance on unlimited task lifetime or task-name deduplication retention.

A scheduled bounded recovery scan checks durable due/retry/expired-running work,
missing sidecars, and enqueued work lacking a terminal receipt/active lease. It
can atomically retire a stale dispatch generation and create a new one, retaining
the same operation. Initial stale-dispatch threshold15 minutes; sweep cadence1
minute is a proposal to measure against #12 costs, not a latency guarantee. Queue
purge, retention expiry, max retries, missed scheduler ticks and publisher downtime
cannot delete durable business intent. Duplicate old task deliveries remain safe.
The accepted #7 activation is every-minute Scheduler to a bounded HTTP publisher,
maximum one publisher instance, with private OIDC consumer; use this baseline
until measured evidence supports a reviewed change. Alert on oldest eligible work
age, recovery redispatches, lost heartbeats, poison
counts and attempt saturation; queue depth alone is not completion evidence.

Quarantine stores fixed reason code, safe attempt metadata and authorized resource
references, not raw provider payload/exception. Operator actions pause, cancel,
retry or replace only through scoped reviewed commands with audit/reason. Cancelling
increments/invalidate fences atomically; it does not undo posted journals. An
irreversible uncertain external effect must be reconciled before cancellation is
reported settled. Deleting queue tasks is optional cleanup, never cancellation
authority. Retention/deletion periods and redacted operator UI are unresolved #7
release gates; retain synthetic evidence only until approved lifecycle is specified.

## Planned acceptance matrix

All names below are **planned**. Existing ledger tests prove posting/outbox
atomicity for their fixtures, not these worker state machines.

| Test/evidence | Required outcome |
| --- | --- |
| `TestWorkIntentAtomicity` | Producer rollback leaves no intent; commit then crash still discoverable; sidecar scan repeats without duplicate operation; source outbox remains immutable. |
| `TestWorkEnvelopeBoundary` | Strict version/UUID/generation/size, unknown fields, spoofed headers/issuer/audience/caller denied; source event is never executed as a posting command. |
| `TestPublisherUncertainEnqueue` | Enqueue success + receipt failure, RPC timeout, dedup tombstone, expired publisher lease and delayed acknowledgement retain correct dispatch history; fast completed consumer cannot be regressed. |
| `TestConsumerDedupOrder` | Duplicate/reordered/old-generation messages produce one authorized effect; setup gaps in book revisions do not deadlock; wrong request digest conflicts. |
| `TestWorkFenceTakeover` | Overlapping claims, expired holder, clock skew, late heartbeat, old process surviving request deadline and cancellation cannot commit; new holder succeeds once. |
| `TestWorkReceiptAtomicity` | Crash before effect, before receipt, before commit and after commit/before ack; domain state/checkpoint/receipt atomic; commit ambiguity rereads instead of repeating. |
| `TestWorkAuthorizationRace` | Disabled/revoked principal/entity/connection, mixed scope, service-only caller, scope change during provider wait; no unauthorized call after check or commit after revocation wins. Calls already in flight are not falsely claimed prevented. |
| `TestConnectionWakeupRace` | Wakeup before/during/after complete commit preserves requested demand; partial stages never move committed cursor; stale connection generation fails despite matching operation UUID. |
| `TestWorkRetryQuarantine` | Transient/backoff/Retry-After, permanent poison, consent block,8-attempt OR24h guard, bounded continuation and authorized retry/cancel retain audit and effect identity. |
| `TestWorkRecoveryWithoutQueue` | Purge/expiry/task tombstone/scheduler outage/missed ack recovered from SQL; redispatch does not duplicate effects or replace a healthy active lease. |
| `TestExternalEffectUncertainty` | Idempotent provider replay reconciles once; unknown non-idempotent effect quarantined and cannot be blindly retried. |
| `TestWorkResourceBoundary` | Concurrency/pool/SQL/HTTP/page/body caps, cancellation, contention, crash and backlog; no network wait holding SQL transaction; redacted logs and bounded metadata. |
| `TestJournalObservationReceipt` | Scoped event/journal header correspondence and unique receipt, no second posting/outbox; bad target/type/version/revision quarantined. |
| `ReviewWorkerStandardScopeV1` | STD-01 internal message assessment, version mappings and deferred financial/audit adapters reviewed; no ISO/production claim. |

## Implementation handoff and alternatives

Preserve #7 acceptance aliases: `outbox-contract-review` maps this document plus
standards review; `fault-matrix` maps the failure tests above; `completion-proof`
maps `TestJournalObservationReceipt` and receipt atomicity; `durable-failure` maps
quarantine/recovery; `transport-auth` maps envelope/authentication and revocation.
Remote acceptance remains dependent on #5/#26/#11; a canceled predecessor is
never treated as completed evidence.

Before #7 runtime work, approve concrete DDL/functions/constraints/role grants,
the fixed journal-observer handler, mandate/consent model with #70,
retention/recovery operations, queue configuration and pool-budget evidence. New
DDL must expose the CAS predicates and unique effect guard to integration tests.
Use fake clock/provider/queue failures plus disposable PostgreSQL; no live sources
or cloud calls in default CI. No OpenAPI operation is advertised by this prose.

A database-only polling worker avoids queue ambiguity and is viable; Cloud Tasks
is retained for managed dispatch/backoff, with SQL retaining recovery authority.
CDC/logical replication can relay outbox changes but adds connector/slot lifecycle
and still needs idempotent consumers. Pub/Sub fanout is deferred; per-consumer
sidecar identity leaves room for it without changing immutable facts. Scheduled
bounded scans trade latency for idle cost; continuous polling requires measured
justification. These are design choices, not guarantees supplied by an outbox
pattern or a queue product.
