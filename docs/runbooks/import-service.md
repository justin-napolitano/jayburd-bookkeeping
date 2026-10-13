# Synthetic import service delivery
<!-- stage-runbook: true -->

## Status and boundary

Issue #93, parent #8; IMPORT-SERVICE-2100. Root approved the service/repository
boundary. Synthetic local orchestration only; no migration, API, cloud, provider,
secrets, implicit import execution or posting in this lane.

## Stage ledger

- Read staging, decoder, source registry and durable import contracts plus sources.
- Coordinated SQL signatures/publication shape with the sole migration author.
- Added bounded SERIALIZABLE orchestration and conservative unknown-COMMIT handling.
- Added synthetic transaction and real PostgreSQL/filesystem fault proofs.
- Independent reviewer found no additional blocking Go issue; coordinated pre-I/O
  expiry/current-binding checks with SQL author.

## Current evidence

`go test -race ./internal/ingestion/importer`: PASS, ten unit groups.
`make check test build`: PASS. Actual disposable PostgreSQL17.11/filesystem proof:
PASS, seven integration groups plus ten unit groups. Initial test-only assertions
used an incorrect journal table name; corrected to journal_entry before PASS.
Final proof rerun passed against migration10 up SHA256
`214350cacf0b69dd0c3273f756a1d373ae016a828a5cfbf4d631ddd74319c8b8`
(including refreshed-preview cancellation refinement). Temporary SQL copies were
removed and the baseline checksum restored; no SQL files are in this lane commit.

- TestImportServicePostgresStages: actual runtime authorization, exact-byte replay,
  amount 0012.500 retains canonical 12.5 and changed evidence conflicts.
- TestImportServicePostgresRejection: invalid cents publish a rejected batch with
  zero usable rows.
- TestImportServicePostgresRevocation, TestImportServicePublicationAuthority and
  TestImportServicePostgresDeadline: real revoked permissions before I/O and after
  storage, plus actual lock cancellation, never publish unauthorized previews.
- TestImportServicePostgresUnknownCommit: actual PostgreSQL COMMIT followed by a
  lost application reply, then replay of retained bytes returns one preview.
  The test tracks both Put/Read outside SQL lifetime; no TCP fault proxy is claimed.
- TestImportServiceExplicitRecovery: explicit existing command register/execute
  and replay create one observation, review item and provenance link; no journal,
  outbox event or ledger revision change. Scoped Review then respects revocation.

Logs: /private/tmp/jayburd-import-service-proof.log and
/private/tmp/jayburd-import-service-default.log. SQL setup is a disposable fixture;
no inherited DSN, credentials or provider calls.

## Remaining work

Root owns final combined migration/harness/recovery integration. No HTTP upload
route, preview UI, hosted encrypted storage, cleanup runner or live-provider proof
is supplied. Staging retains private orphans on unknown/denied publication;
retention/deletion reconciliation is separate work. Source implementations must
honor context; arbitrary blocked readers or filesystem syscalls are not cancellable.

## PR handoff

Lane owns importer package and its contract/runbook only. Root integrates SQL,
harness, shared recovery changes and publication. No lane push/deployment.
