# Acquisition dispatch SQL

## Reviewed decision and sources

Migration 15 proposes a narrow delivery bridge for #7. Existing requested/completed
sync generations and sync sets remain acquisition authority; no cursor or source
checkpoint is copied. Discovery includes initial generation 1 even without a wakeup
row. Queue identity, business claim identity and acquisition fence are distinct.

Reviewed official Google Cloud Tasks [overview](https://docs.cloud.google.com/tasks/docs/dual-overview)
(at-least-once delivery, task-name deduplication and task lifecycle), [HTTP task creation](https://docs.cloud.google.com/tasks/docs/creating-http-target-tasks)
(explicit target/OIDC configuration) and [create API](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks/create)
on 2026-10-14. Facts: duplicate delivery is possible and named-task deduplication can
outlive a deleted task. Inference: durable SQL completion, not enqueue acknowledgement
or ALREADY_EXISTS, proves work; replacing a lost task needs a new dispatch generation
while retaining unresolved acquisition identity. Review covers those selected public
sections, not a cloud deployment or provider qualification.

Reuse [worker SQL](worker-sql.md), [background work](background-work.md),
[acquisition runner](acquisition-runner.md), [sync SQL](provider-sync-sql.md) and
[connection operations](connection-operations-sql.md). Internal work-wakeup-v1 with
fixed acquisition target profile 2 has no ISO 20022 business-message designation.
ISO currency/date source semantics and accounting policy are unchanged.

## Frozen interfaces

All functions belong to integration; UUID parameters name server-owned identities.
Publisher role is existing bookkeeping_publisher; consumer role is bookkeeping_sync,
with exact owner-enrolled actual login/scope checks. No new role or journal-observer
handler is introduced. All calls are short Serializable transactions; network and
acquisition.Run occur only after committed claims.

- `enroll_sync_delivery(actor,e,b,key,raw)` owner-only. JSON is
  `{publisher_login,worker_login,service_principal_id,enabled}`; immutable identity
  must match existing connection_worker, response `{binding_id,binding_revision}`.
- `sync_delivery_discover(binding,maximum)` returns `{created_count}` (maximum 25).
- `sync_delivery_candidates(binding,maximum)` returns `{operation_ids:[uuid]}`.
- `sync_delivery_recover(operation)` returns status no_change/retry_wait/
  redispatched/done/quarantined.
- `sync_delivery_dispatch_claim(operation,holder)` returns status
  claimed/not_due/done/quarantined, operation_id/entity_id/book_id,
  dispatch_generation string, target_version 2, task_name_hex, dispatch_fence
  string|null and lease_until timestamp|null.
- `sync_delivery_dispatch_result(operation,generation,holder,fence,result)` takes
  accepted/already_exists/timeout/transient/rejected and returns `{recorded:boolean}`.
- `sync_delivery_claim(operation,dispatch_generation,holder)` returns status
  claimed/done/busy/not_due/quarantined, operation_id/entity_id/book_id/stream_id,
  claim_id uuid|null, fence string|null, lease_until timestamp|null,
  deadline_at timestamp|null, target_version 2.
- `sync_delivery_settle(operation,holder,fence,outcome,retry_after_seconds)` accepts
  completed/retryable/permanent/unknown and returns done/retry_wait/quarantined/stale.

The task name is lowercase SHA-256(operation UUID + `|` + decimal dispatch generation
+ `|2`). Only IDs/version belong in the wakeup. Neither queue headers nor payload
select a principal, provider account, credentials, handler or SQL scope.

## State, authority and recovery

The binding links one actual publisher login and the configured acquisition worker
login/service principal to one book. Disabling either binding or connection worker
mandate stops dispatch and consumption. Every replay rechecks role separation,
active principals/entity/book, service read/import membership and current source,
consent and Item authority. This does not revoke the worker's independent acquisition
SQL capability; revoke that source/service authority to stop all direct acquisition.

One operation captures demanded generation per stream. A partial unique index
retains quarantined operations in that stream slot. Success releases the slot;
demand arriving during acquisition therefore becomes a successor operation. Immutable
wakeup rows are intent evidence, not a second mutable checkpoint or consumed cursor.
Discovery and recovery are bounded to 25 candidates. Future retry waits, healthy
leases and accepted tasks awaiting their queue-loss interval cannot occupy the
candidate page ahead of due work.

Enqueue leases last 30 seconds; consumer leases last 120 seconds. The original and
deterministic pagination-restart sync claims, plus the stream's current inner lease,
are inspected before takeover. An expired outer lease never overrides a live inner
lease. A committed sync completion receipt independently proves success, including
when Runner or settlement acknowledgement was lost. Uncertain enqueue uses the same
task name; queue loss after 15 minutes advances dispatch identity, not business
identity. A committed retry handoff creates its next dispatch immediately and sets
its not-before time atomically, so it does not wait for queue-loss recovery.

Only known expired/failed acquisition state permits a new business claim. An
expired staging set is abandoned and fenced before replacement. A known retryable
settlement retains at least Retry-After (maximum 86,400 seconds; minimum scheduling
backoff 5 seconds). Eight failed claims or 24 hours from first execution quarantine
the operation. No new webhook resets these budgets. A lost provider-failure
classification cannot reconstruct Retry-After from an abandoned set: recovery
quarantines `outcome_unknown`, requiring future explicit operator recovery. No
automatic repair or administrative reset is included in this slice.

Permanent quarantine fences this operation's original/derived staging set so a late
worker cannot publish after termination. Immutable enqueue and run receipts preserve
exact replay results after current authorization. Queue acknowledgement never posts a
journal and never substitutes for a source completion receipt. Fixed SQL errors
P2900–P2905 denote malformed, denied, conflict, stale, state and bounds; no credential
or raw provider payload belongs in delivery tables or responses.

The 24-hour limit prohibits new claims and redispatch; it is also the cooperative
Runner deadline. Completion receipts have no immutable completion timestamp, so SQL
does not prove publication occurred before that cutoff. A real durable receipt
still wins when discovered later: elapsed delivery age cannot erase published
source evidence or falsely report it as failed.
