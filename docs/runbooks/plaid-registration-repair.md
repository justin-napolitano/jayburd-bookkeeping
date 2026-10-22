<!-- stage-runbook: true -->
# Plaid registration repair

## Status and boundary

Issue #258; lease `PLAID-258-20261022T0050Z`; base `6a92186`.
Owner requested submission of #257 followed by checking repair and registration
of fictional auto, consumer and line-of-credit accounts. No production-bank,
posting or infrastructure change is authorized by this branch.

## Stage ledger

1. **Resume and claim — complete.** Rechecked #65/#88/#258 and #257; #257 is
   ready for review. Dedicated worktree; Position work is disjoint.
2. **Sources and boundary — complete.** Refreshed official Accounts subtype,
   product support and Transactions amount fields. Supplemental decision in
   the liability contract. Exact checking failure remains undiagnosed; no
   private amount or raw provider response was accessed.
3. **Implementation — complete for registration and diagnostics.** Extend the narrow verified loan allowlist
   and distinguish amount magnitude, storage scale and currency precision
   failures using fixed diagnostic codes. Money limits remain unchanged.
4. **Validation — partial.** Core, web, migration, ledger, connection-service,
   acquisition, provider-flow and dispatch gates pass. Both full-stack browser
   gates await the shared local test ports held by the Position demo.
5. **Hosted follow-up — pending.** A reviewed Sandbox migration/runtime release
   is necessary before retrying held checking and registering the additional
   types. Do not describe source changes as completed registration or a fixed
   hosted checking cause.

## Current evidence

`make check test build` passes (all-package race tests and migration/tooling
checks). `make web-check web-test web-build web-browser` passes; final unit rerun
has 289 tests, component browser suite 17. `make migration-test ledger-test
connection-service-test acquisition-test provider-flow-test sync-dispatch-test`
passes with disposable synthetic databases. Mixed seven-account registration
has six distinct liability accounts, exact replay, whole-request unsupported-loan
rollback and zero journals. Durable quarantine/recovery passes for all three
new money codes and the previous currency code. Money fuzz passes 2,098,471
executions in 30 seconds with two workers and an independent rational oracle.

Tracked dependency/toolchain files are unchanged. Local Go uses installed 1.26.5
and the external modfile described in #257; pinned 1.26.10 remains CI's gate.
Database suites use the previously documented external cache-only pull adapter
for the two exact pinned Docker images. Initial sandbox listener/socket denials
were rerun with approval. A mistaken three-account test expectation introduced
while expanding the seven-account fixture was corrected before the passing run.

`connection-browser-test` stopped before startup because ports 39081/39083 are
held by the separate Position worktree demo. Process ownership was checked;
those processes were not killed. `app-integration` did not run after that stop.
The owning session has been asked to release its demo when finished.

## Remaining work

Complete both full-stack gates when their ports are available. Review migration
28 and matching runtime for a separately scoped fictional Sandbox rollout.
Then retry checking once to obtain the precise classification and implement a
source-backed repair; do not relabel diagnostics as a fixed checking account.
Register the three additional fictional loan types after rollout and record
actual publication outcomes. Existing active attempts cannot add new bindings;
the current command requires a fresh reviewed attempt, selecting only the three
new accounts to avoid duplicating the existing four in this book.

## PR handoff

Migration 28 must precede matching runtime. Transactions coverage varies by
institution; registration does not promise transaction availability. Liability
draft/posting guards remain. API diagnostics are additive; old generic reasons
remain readable. Human review is required for these restricted-data validation
changes. No secret system, authorization, public exposure or dependency changes.
Rollback refuses new diagnostics or active additional-loan streams rather than
discarding evidence or leaving a stream unsupported. Existing merged SQL is
unchanged. Link #258; no automatic issue closure while hosted work is incomplete.
