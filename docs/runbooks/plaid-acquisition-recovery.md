<!-- stage-runbook: true -->
# Plaid acquisition recovery delivery

## Status and boundary

Issue #220, release/acquisition-recovery-runtime, stacked after release PR #222.
Development Sandbox only.
Existing enrollment retained; no raw provider access or live-account approval.

## Stage ledger

1. **Source and failure review — complete.** Owner supplied fixed SQL status
   codes. Official Plaid sync sections reviewed; exact rejected field is unknown.
2. **Contract — complete.** [Recovery contract](../contracts/plaid-acquisition-recovery.md)
   fixes append-only recovery, explicit authorization and closed diagnostics.
3. **Implementation — complete.** Migration 24 appends diagnostic and recovery
   receipts. Manual request releases known failed slots; status distinguishes
   quarantine/retry from demand. Terminal attempts are preserved.
4. **Validation — complete.** PR #221 checks passed at source `2d640267`.
   Seed run 39451988284 verified pinned Go 1.26.9 and all SQL/browser gates.
5. **Candidate images — complete.** Build run 39454157150 passed all validation,
   exact-image runtime proofs, SBOM generation and HIGH/CRITICAL image scans.
6. **Migration/worker release — staged.** Exact same-build migration and worker
   manifests are pinned. Apply migration 24 before promoting the worker. Scheduler
   is verified paused; exact scheduler/task-invoker service grants are present.
7. **Application release — staged.** The matching API/web candidate from build
   39454157150 uses the reviewed finite Sandbox configuration and validates
   migration status before promotion. Dispatch only after migration 24 succeeds;
   this Sandbox application operation checks status rather than applying SQL.
8. **Hosted acceptance — pending.** Classify and repair the precise failure, then
   prove publication and transaction review on the existing enrollment.

## Current evidence

Migration 24 fresh/replay/down/up passes. Dispatch SQL tests prove fixed diagnostic
redaction, immutable known-failure recovery, distinct replacement claim, old task
fencing, successful replacement publication and unknown-outcome refusal. Frontend
286 unit tests, production build and 17 browser tests pass. `make check test build`,
`make ledger-test`, `make acquisition-test sync-dispatch-test`, `make app-integration`
and `make connection-browser-test` pass. The full connection journey includes
enrollment, acquisition, explicit posting and lost-response recovery; new manual
quarantine recovery is separately proved with real restricted SQL sessions.

Local Go checks used a cached, network-isolated Go 1.27.2 container and cached
public modules because the Go 1.26.9 download was blocked. The repository toolchain
pin is unchanged and both PR/build CI verified Go 1.26.9. No tests accessed
cloud/provider data.

First CI pass reached the provider-publication privilege inventory and correctly
rejected the additional diagnostic function until its exact signature was added
to the expected count. The test still denies table access and direct recovery.
This is an intentional one-function grant; no role or direct DML grant is added.

Source review also found SQL's empty-account initialization rule disagreed with
the reviewed Go decoder. Migration 24 now permits all known Item readiness states
for that strictly empty first/final page; TestSyncEmptyAccountItemReadiness proves
zero changes and an unchanged null account cursor for each state. This is a
reproduced valid-response fix; hosted decoder classifications remain pending.

## Remaining work

Release diagnostics/recovery in the reviewed order, then reproduce and repair the precise
hosted compatibility failure without another enrollment.
No successful acquisition or complete history claim follows from existing 204s.

## PR handoff

Link #220, record migration 24 impact and named checks. Preserve failed history;
unknown outcome recovery remains excluded.
