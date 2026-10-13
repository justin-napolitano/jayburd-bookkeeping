# Journal observer local SQL proof
<!-- stage-runbook: true -->

## Status and boundary

Migration 000008 implements receipt-only `journal.observe` under the reviewed
[physical contract](../contracts/worker-sql.md). Local PostgreSQL fault tests pass.
No production Go worker, hosted queue, financial effect or provider access exists.

## Stage ledger

- Reviewed the contract and allocated migration 8 with the coordinator.
- Added binding, operation, sidecar, dispatch, attempt and receipt relations;
  nine restricted functions and owner-only immutable/deferred invariant guards.
- Actual restricted-role COMMIT testing found the deferred consistency trigger
  needed SECURITY DEFINER; corrected without granting runtime table access.
- Fixed expired eighth-attempt status to return `done` after quarantine.
- Coordinator approved fixed dispatch-creation age for 15-minute recovery.
  Fresh uncertain results cannot postpone replacement forever; the first expired
  publisher claim preserves task identity, repeated expired claims remain bounded.

## Current evidence

2026-10-13, disposable PostgreSQL 17.11, synthetic owner fixtures and separate
actual LOGIN sessions:

- `bash scripts/test-observer.sh`: PASS, 19 named tests. Discovery/receipt rollback,
  concurrent discovery, two recovery scanners, lease expiry during lock wait,
  cancellation, overflow, jitter/eight-failure cap and age cap passed.
- Exact runtime function/table ACLs, spoofed session/role rejection, supported
  isolation, temporary-name isolation and immutable evidence passed.
- Two-session completion-first/revocation-first races, binding/principal/entity/book
  disable, poster/service independence and fixed terminal suppression passed.
- Reversal/revision-gap discovery, owner-injected source mismatch quarantine,
  late publisher acknowledgement, stale publisher fence, old/new dispatch delivery
  and repeated-uncertainty recovery passed. Source amounts/revisions remain intact.
- `make check test build`: PASS. `bash scripts/migrate.sh check`: PASS.

`TestObserverCommitAcknowledgementLoss` injects loss at the application boundary
following actual COMMIT and reconnects through claim semantics. It is not a TCP
fault proxy or hosted transport test. Owner-only expired/corrupt synthetic fixtures
exercise impossible-to-wait clock/corruption cases; runtime has no clock override.

## Remaining work

Combined migration, ledger, restricted bootstrap and observer gates passed; see
[branch evidence](ingestion-foundations.md).
Production Go transport, queue/OIDC, hosted enrollment, role transaction/idle
limits, retention and pool budgets remain separate release gates. Deletion policy
remains open; foreign keys currently restrict principal deletion. No ISO audit
exchange or live accounting approval is claimed.

## PR handoff

Issue #7 under #57; contract #74. Forward-only schema change; migrations 1–7 intact.
Migration 8 needs pre-created NOLOGIN publisher/observer roles under the restricted
hosted migrator. Root owns that bootstrap change, Makefile/CI and version fixtures.
No cloud activation or new workload LOGIN/binding is included.
