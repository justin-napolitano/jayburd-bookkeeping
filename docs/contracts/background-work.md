# Background work contract

## Decision

Use database transactions for accounting and a transactional outbox for external
dispatch. Cloud Tasks carries directed commands; Pub/Sub is deferred until there
are multiple independent event subscribers. A command requests work; an event
reports a committed fact. Neither message grants user/entity authority.

## Lifecycle

1. Commit business state, a durable operation ID and outbox intent together.
2. Publisher claims a bounded batch using short transactions and expiring leases.
3. Publish outside the database transaction; record delivery only after enqueue
   acknowledgement. A crash here can duplicate delivery, so recipients deduplicate.
4. Handler verifies the service caller, reloads authorized scope, claims work,
   executes a bounded step and commits effects plus its completion receipt.
5. Acknowledge the task after durable completion. Replayed completed work has no
   additional effects. An expired worker cannot commit using a stale fence.

Delivered, running, completed, retryable and permanently failed are distinct.
An outbox row marked delivered does not mean the accounting operation finished.
Durable operation records preserve terminal failures even after queue retention
expires; operators can explicitly retry with controlled identity.

## Concurrency and recovery

Serialize sync per provider connection, not per message arrival. Recheck current
connection permissions/revocation before external calls. Commit data and the
associated checkpoint atomically after a complete bounded acquisition unit.
Retries use backoff/jitter, attempt limits and visible quarantine. External
effects need provider idempotency or reconciliation of uncertain outcomes.
Never keep a database transaction open while waiting on a network provider.

Scheduled bounded publisher runs are the initial activation proposal. Dispatch
latency and operating cost must be measured before choosing continuous polling.
Cloud Run HTTP tasks and longer Cloud Run Jobs have different lifecycles;
enqueueing a Job is not evidence of Job completion.

## Required tests

Crash before/after commit; publish-success/receipt-failure; duplicate and reordered
delivery; overlapping claims; lease expiry/stale fences; revocation while queued;
transient/permanent errors; poison work; pagination correction/replay;
redaction; cancellation; durable completion before acknowledgement.

## Source

[Cloud Tasks delivery limitations](https://docs.cloud.google.com/tasks/docs/common-pitfalls),
checked 2026-10-11: do not depend on task order or absence of duplicates.
