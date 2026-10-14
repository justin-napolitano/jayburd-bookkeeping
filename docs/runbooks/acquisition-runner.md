# Acquisition runner delivery
<!-- stage-runbook: true -->

## Status and boundary

Reviewed implementation in progress for #102, lease ACQUISITION-GO-102. Scope is
[runner contract](../contracts/acquisition-runner.md), new internal/acquisition
files and tests. No live provider, credentials, cloud or migration changes.

## Stage ledger

- Read issue102, AGENTS, merged sync/decoder/runtime contracts and source index.
- Refreshed official Plaid account-filter/pagination sections.
- Coordinator approved bounded runner and durable raw evidence design.
- Agreed sync_context/sync_resume wire with the sole SQL author; connection lane
  confirmed opaque credential binding and memory-only resolver expectations.

## Current evidence

- `make check test build` passed: vet, race tests, migration file checks and builds.
- `bash scripts/test-acquisition.sh` passed against final SQL13 from3590875:
  13groups and9subtests, including five actual PostgreSQL groups. Real restricted
  sessions commit before injected claim/page/completion response loss; replay reads
  existing evidence/receipts without duplicate fetch or observations. Unknown claim
  reaches no provider. Complete mutation restart and original claim replay passed.
- PostgreSQL assertions cover large exact USD cents, revoked completion access,
  post-fetch revocation, lease expiry, cancelled caller, API/mixed-role denial,
  no journal/outbox effects. Network and credential callbacks assert no active SQL
  transaction. Original HTTP fixtures separately exercise JPY/KWD/exponent amounts.
- SQL unit faults cover08007/40003 ambiguity without replay and bounded40001/40P01
  retries. Filesystem tests cover two-chunk2MiB exact bytes, concurrency, interruption,
  cross-scope reads and deletion tombstones. Redaction and Retry-After tests passed.
- Temporary SQL13 and harness copies were used for proof only; sole SQL author and
  coordinator own those files. No provider credentials, cloud or real data used.

## Remaining work

Hosted encrypted storage/retention, durable retry scheduling, real provider
qualification, HTTP wakeup/service deployment and operational load tests remain
separate gates. This runner is callable and composition-tested locally; it is not a
hosted bank connector. Cancellation depends on injected components honoring context.

## PR handoff

Initial implementation handoff is followed by the actual PostgreSQL proof. Coordinator owns shared composition/scripts/CI; sole SQL author owns
migration13 and context/resume functions. No journals from acquisition.
