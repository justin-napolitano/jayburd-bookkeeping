<!-- stage-runbook: true -->
# Acquisition dispatch runtime delivery

## Status and boundary

Issue #7, DISPATCH-GO-7: local Go publisher/consumer foundation implemented and tested with provisional SQL15.
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
external provider was called. Logs: `/private/tmp/jayburd-dispatch-default.log`. Actual SQL15/Runner composition now passes: eleven groups, nine subtests including
three real PostgreSQL groups. Real acquisition uses restricted logins, pinned HTTP
with injected synthetic responses and private durable staging. Evidence covers:

- Exact `9007199254740993.01`, cursor C1, one source observation, no journals/outbox.
- Actual completion and settlement commits with lost replies; duplicate delivery
  makes no second provider request. Current revoked authority denies receipt replay.
- Actual unknown delivery claim performs zero provider calls and creates no sync set.
- HTTP429 Retry-After3600 creates retry_wait and generation2 atomically before ack;
  old delivery cannot bypass the delay. Fixture-controlled eligibility advance lets
  generation2 complete with exactly one source observation and no journal.

Proof log: `/private/tmp/jayburd-sync-dispatch-go-proof.log`. Root owns the final
SQL15 integration rerun and private HTTP/Cloud Tasks composition proof. No hosted
acceptance is claimed.

## Remaining work

Root repeats the integrated proof against final SQL15 and validates fixed HTTP/task
profiles. Actual cloud delivery/OIDC and provider qualification remain separate gates.

## PR handoff

Initial implementation handoff: unit/race/default gates passed; actual PostgreSQL
proof passed against provisional migration15; root must repeat against final integrated SQL. Owned paths are internal/syncdispatch (excluding SQL author's
SQL integration file), this runbook and the corresponding contract.

- Follow-up stage: real delayed retry and redispatch proof passed without changing
  production Go; initial core handoff was `6985353`.
