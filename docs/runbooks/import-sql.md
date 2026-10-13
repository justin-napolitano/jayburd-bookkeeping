# Durable import SQL delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #92, child #8/#84/#85; lease IMPORT-SQL-2100. Migration10 synthetic durable
registration, preview and recoverable import only. Contract: [import SQL](../contracts/import-sql.md).
No provider/cloud/HTTP/upload screen or financial posting is added by this lane.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Review | Read AGENTS, index, bibliography, database/standards/import/registry/command contracts; exact signatures and publication trust boundary reviewed with coordinator and Go author before DDL. |
| Implementation | Scoped immutable evidence and provenance, owner-controlled entity/source import grants, current authorization and existing principal command recovery extended. Migration1–9 unchanged. |
| Review | Independent reviewer checked grants, immutable identity, locks, receipts and cancellation. Context now blocks expired/inactive first use before storage I/O. Coordinator approved refresh-before-cancel refinement and exact registry EXECUTE fixture extension. |
| Verification | `bash scripts/test-source-registry.sh` PASS: 21 top-level groups (8 durable import), 47 subtests; `make check test build` PASS, 2026-10-13. |

## Current evidence

Focused command: `bash scripts/test-source-registry.sh`, local pinned Go1.26.9,
PostgreSQL17.11 and golang-migrate4.20.1 images; disposable internal Docker network,
fictional principals, actual runtime/operator logins, no inherited DSN or cloud.
Logs: `/private/tmp/jayburd-import-sql-proof.log` and
`/private/tmp/jayburd-import-sql-default.log`. Both commands exited 0.

New named groups: TestDurableImportIdentity, Scope, Validation, Recovery, Atomicity,
Concurrency, MoneyAndBounds and Lifecycle. They cover exact/cross-file replay,
distinct identical purchases, whole-batch correction rejection, equal-hash field
conflicts, strict malformed/duplicate-key/huge-exponent rejection, USD/JPY/KWD exact
units, 1000-row import and 1001 rejection, permission/revocation/disabled/mixed scope,
role denials, stale-refresh-cancel-reconfirm and cancel-current-first ordering,
concurrent same-command execution, expiry/paused-history recovery, overlap_hold,
and eight late injected failure families. Independent SQL assertions inspect
journal_entry/journal_line/outbox_message counts and book revision.

Initial proof found two test harness expectations: financial tables use
journal_entry/outbox_message names, and the existing registry privilege allowlist
needed the five explicitly reviewed import entrypoints. Both were corrected; the
allowlist remains exact and observer/publisher grants remain empty. This is not
provider, end-to-end accounting month or deployment evidence.

## Remaining work

Coordinator owns integrated migration/ledger/Cloud SQL/schema-version/API/Go-service
checks. Object staging/orchestration has separate proof; this lane does not claim
storage-byte parsing, object cleanup or power-loss behavior. Review matching,
draft/source posting, reconciliation, Plaid, encrypted hosted storage, retention
and live policies remain gated. Source lifetime identities and normalized history
cannot be reset through this slice.

## PR handoff

Own migration10 pair/checksum, durable import integration tests, exact registry
privilege fixture extension and this contract/runbook. Required `make check test
build` and staged runbook guard precede commit. Coordinator integrates and publishes;
no lane push, deployment or provider access.
