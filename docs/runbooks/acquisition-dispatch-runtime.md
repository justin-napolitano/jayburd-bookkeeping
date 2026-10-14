<!-- stage-runbook: true -->
# Acquisition dispatch runtime delivery

## Status and boundary

Issue #7, DISPATCH-GO-7: local Go publisher/consumer foundation under implementation.
No live provider, cloud deployment, scheduler or financial posting is enabled.
Root owns HTTP/transport composition and proof harness; the SQL lane owns migration15.

## Stage ledger

- Read merged observer/acquisition contracts, implementation and source index.
- Coordinator approved target2, strict existing wakeup, separate authority and bounded
  request/settlement budgets. SQL author froze seven JSON function signatures.
- Implemented transaction adapter, bounded publisher and stable-claim consumer.
- Eight focused unit groups pass: ambiguous commit, confirmed rollback retries,
  queue/runner outside transactions, retry delay/error redaction, boundaries and
  cancellation, unknown settle/conflicting transport outcomes, returned metadata/profile isolation,
  and preserving a full retry delay through durable handoff.

## Current evidence

`go test -race ./internal/syncdispatch` and `make check test build` pass using pinned
Go1.26.9. The default suite required loopback-listener sandbox escalation; no
external provider was called. Logs: `/private/tmp/jayburd-dispatch-default.log`. Actual SQL15/Runner
composition proof remains pending; no hosted acceptance is claimed.

## Remaining work

Compose disposable PostgreSQL and the real injected acquisition runner after SQL15
is available. Validate receipt-loss replay, current revocation, duplicate delivery,
retry handoff and no journals. Root separately validates fixed HTTP/task profiles.

## PR handoff

Initial implementation handoff: unit/race/default gates passed; actual PostgreSQL
proof explicitly remains pending migration15. Owned paths are internal/syncdispatch (excluding SQL author's
SQL integration file), this runbook and the corresponding contract.
