# Local observer adapter
<!-- stage-runbook: true -->

## Status and boundary

Implemented the [reviewed adapter contract](../contracts/observer-runtime.md):
role-separated Go calls, a bounded publisher tick and receipt-only consumer.
Transport is injected; no HTTP/OIDC, Cloud Tasks SDK or hosted activation exists.

## Stage ledger

- Reviewed SQL/background-work contracts and proposed the API before code.
- Root approved role separation, deadlines, committed-result gates and fake queue.
- Implemented all nine SQL primitives, strict wakeup parsing and task-name checks.
- Independent review found ambiguous COMMIT SQLSTATEs were classified too broadly.
  Corrected 08007/40003 and other unclassified outcomes to `commit_unknown`;
  explicit rollback/constraint outcomes remain known. No unknown outcome retries.

## Current evidence

2026-10-13:

- `make check test build`: PASS.
- `make observer-test`: PASS against disposable PostgreSQL 17.11, including the
  existing 19 SQL fault tests, seven adapter unit groups and six adapter integration
  groups. All nine production adapter primitives execute under actual role-bound
  sessions; no inherited DSN, credential or provider is used.
- `TestAdapterTransactionFaults` and `TestPublisherUnknownSQLStateNeverEnqueues`:
  full SERIALIZABLE retries, stable holder, safe errors, bounded cleanup and no
  replay/enqueue/metadata on ambiguous commit.
- `TestWakeupBoundary`, `TestAdapterRoleAndMetadata`,
  `TestPublisherCommitBeforeTransport`, `TestTransportContradiction`:
  strict input/output, digest and deadline checks; queue calls outside transactions.
- `TestObserverAdapterFastConsumer`, `TestObserverAdapterQueueLoss`:
  fast completion, late acknowledgement, stable uncertain task identity and
  duplicate/reordered old/new dispatches preserve one observed receipt.
- `TestObserverAdapterUnknownCommit`: actual successful PostgreSQL COMMIT with an
  injected lost application reply at discovery, dispatch claim/result and consumer
  claim/finish. Scoped retries recover committed state. This is not TCP fault injection.
- `TestObserverAdapterAuthority`, `TestObserverAdapterExpiryAndDeadline`,
  `TestObserverAdapterFailure`: wrong-role rejection, current revocation, expiry,
  real query cancellation and retryable failure retain durable SQL authority.

## Remaining work

No command/service composes this adapter yet. Add separately reviewed authenticated
HTTP/Cloud Tasks transport, caller identity, configured endpoint allowlist, bounded
pool composition and hosted deployment evidence. Runtime objects never retrieve
secrets. The fake transport cannot establish provider delivery or OIDC guarantees.

## PR handoff

Issue #7, additive Go implementation and docs; no migration or cloud mutation.
Root owns integration/CI/status documentation. Tick counts report partial relay
progress, never total backlog completion or financial effects.
