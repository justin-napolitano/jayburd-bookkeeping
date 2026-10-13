# Private synthetic staging
<!-- stage-runbook: true -->

## Status and boundary

Implemented the [reviewed local contract](../contracts/import-staging.md), #82.
Synthetic Linux/macOS evidence storage only; no upload UI, SQL or cloud storage.

## Stage ledger

- Root approved generated scope/object identity, permanent tombstones, context-aware
  source ownership and exact-byte/no-replace publication before implementation.
- Added private root confinement, nofollow/type/owner/mode validation, cooperative
  cross-process locking and file/directory sync publication.
- Proved retry after actual child-process exit before/after publication, cancellation
  before/after the visibility boundary and interrupted deletion recovery.

## Current evidence

2026-10-13:

- `go test -race ./internal/ingestion/staging`: PASS on macOS; 13 test groups.
- The same compiled suite: PASS on Linux arm64 in an isolated `--network none`
  disposable container; no database/provider/credentials. Container removed.
- `make check test build`: PASS.
- `TestStagingImmutableReplay`, `TestStagingConcurrentStores`,
  `TestStagingConcurrentProcesses`: full bytes, unchanged receipt, changed-byte
  conflict and competing writers through independent handles/processes.
- `TestStagingSizeAndSourceBoundary`, `TestStagingCancellationAndDeleteRace`:
  exact 1MiB/sentinel limits, owned close, cooperative context cancellation, safe
  reader errors and deletion winning an in-flight upload without resurrection.
- `TestStagingPublicationFaults`, `TestStagingProcessExitRecovery`,
  `TestStagingCancellationPublicationBoundary`, `TestStagingDeleteReplay`:
  prepublication rollback, uncertain publication acknowledgement, process-crash
  private orphan, permanent tombstones and convergent deletion/retry.
- `TestStagingPathsAndModes`, `TestStagingCorruptDiskAndLockDeadline`,
  `TestStagingCloseAndPrivateOrphan`: malformed scope, symlink/outside sentinel,
  restrictive permissions, corrupt-size rejection, real lock cancellation,
  store shutdown and unaddressable temporary files.

The subprocess exits test process failure, not hardware power loss. Local fsync
calls and reopen tests do not establish encrypted storage or durable-device health.
No arbitrary blocked reader or filesystem syscall cancellation is claimed.

## Remaining work

Future caller must authorize scope and retain generated references. Hosted
encryption/Cloud Storage, SQL preview/import publication, retention/holds and
orphan/tombstone lifecycle management remain separate integration gates. Delete
removes the published name and permanently blocks key reuse; returned byte copies,
crash orphans, backups and filesystem remanence are not erased by this adapter.

## PR handoff

Issue #82 under #8/#55. Package/contract/runbook only; no SQL/API/cloud edits.
No new dependency; unsupported platforms fail closed. Root owns shared status,
source-index links, CI and integration. No live data or working-upload claim.
