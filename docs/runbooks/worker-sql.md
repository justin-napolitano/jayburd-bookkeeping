<!-- stage-runbook: true -->
# Worker SQL contract review

## Status and boundary

PROPOSED #74 physical contract for receipt-only journal.observe v1. No SQL migration,
worker runtime, provider call, grant, cloud resource or live-data change. Root owns
review; reporting lane owns migration 000006. No worker migration number reserved.

## Stage ledger

- 2026-10-13: reviewed merged worker contract, source register, immutable outbox,
  authorization locks, membership constraints and scoped posting receipts.
- Reviewed PostgreSQL 17 locking, clock and restricted-function source sections;
  WS01–03 records reside in the [physical contract](../contracts/worker-sql.md).
- Specified scoped rows, constraints, indexes, functions, service binding, separate
  publisher/consumer fences, atomic outcome counts and named future fault tests.

## Current evidence

`make check test build` PASS on 2026-10-13: project/link checks, Go vet/race
tests, immutable migration artifact checks and command builds. Contract review
only: these checks do not execute proposed SQL. Named
Observer tests in the contract are future requirements, not current PASS evidence.
Existing ledger tests do not exercise the proposed observer records or functions.

## Remaining work

Coordinator review of physical design and service enrollment; source-index link by
root coordinator; migration allocation after inspecting merged main. Implement
restricted SQL with disposable two-session failure tests, then runtime/queue fake,
then separately approved hosted rollout. Principal deletion semantics (#4), hosted
identity/secrets/database gates (#11/#26), operator recovery and retention remain
open. No passing build or source review closes these acceptance gates.

## PR handoff

Only docs/contracts/worker-sql.md and this runbook are owned by this lane. Include
both in every commit and run staged runbook validation. Commit handed to root for
integration, no push or GitHub mutation. Issue #74 design acceptance requires
coordinator review; #7 implementation remains separate.
