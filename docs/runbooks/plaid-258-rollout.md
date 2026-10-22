<!-- stage-runbook: true -->
# Plaid #258 approved Sandbox rollout

## Status and boundary

Owner approved migration 28, matching API/web/worker rollout, one held-checking
retry and registration of the three fictional additional loan types. Lease
`PLAID-258-ROLLOUT-20261022T0102Z`, expiry 2026-10-22T06:00Z. No production bank,
posting, IAM expansion, credential inspection, raw payload retrieval or SQL
force/down operation. Prior acquisition scheduling was ENABLED; pause, check
quiescence and restore that same schedule only as bounded rollout controls.

## Stage ledger

1. **Implementation acceptance — complete.** PR #259 merged as
   `7737991e386d3ddfcee6385289ffc52f4091693a` after all checks passed. CI run
   39703004273 passed core validation, frontend and both full-stack/demo paths;
   Infrastructure/GitGuardian also passed. #258 and #88 stay open.
2. **Candidate publication — complete.** Build-images run 39704183980, attempt1,
   completed successfully on the exact source commit above. All validation,
   runtime/container proofs and four image-security jobs passed. Independent
   strict validators matched every candidate receipt to the trusted run and all
   four scan reports contained zero HIGH/CRITICAL vulnerabilities.
   Duplicate push validation 39704176243 was canceled to release the shared
   main-branch concurrency slot. It is not passing evidence; the candidate's own
   complete validation remains mandatory. Position CI was left running.
3. **Migration — prepared.** Exact-image migration-only manifest hash
   `5b6a88a2c6d3c0a85c393e884c2cdeb6ee0a84d8a919d995ef15aca7d3f5946d`
   passes strict metadata and same-build receipt validation. Worker manifest hash
   `579254e64c77244927bbcabfdf50d304133556d4931f60e596d0ca64bed9feb8`
   is prepared for the later paused rollout. Both expire 2026-10-22T12:00Z.
   Confirm immutable successful execution
   before runtime promotion; never infer success from dispatch.
4. **Application and paused workers — pending.** Reuse unchanged approved
   identity/secret-version metadata. Review additive schema/API compatibility,
   exact digests and expiry. Pause scheduling before application promotion;
   check queue empty and known account leases inactive before worker replacement.
   This prevents new UI-supported loans being acquired by an older worker.
   A paused Scheduler alone is not
   drain evidence. Restore its prior state only after successful readback.
5. **Diagnosis and registration — pending.** Retry checking once, inspect only
   fixed diagnostic/status fields, and repair based on evidence. Register only
   auto, consumer and line of credit in a fresh reviewed fictional attempt; keep
   the existing four accounts out of the new bindings. Verify all seven account
   types and source publication outcomes without posting.

## Current evidence

The [rollout source review](../research/plaid-rollout-sources.md) records selected
Scheduler v1 deadline/retry/state sections and local handler budgets. Queue and
lease observations will be bounded evidence, not a universal drain guarantee.

PR #259: validate 11m15s; application integration 8m52s; frontend 2m04s;
foundation 42s; secret scanning passed. Release tooling tests also pass locally.
Additional seven-account, three-currency synthetic money matrix passes using an
external Go test overlay, leaving the candidate source unchanged; retain it in
the checking-repair PR. This is baseline parser evidence, not a fixed hosted
checking cause. Edge owner session is still signed in to the fictional Sandbox
book. Current Scheduler state readback remains ENABLED.

[Compatibility review](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/258#issuecomment-6330452107)
binds migration28 checksums, exact source/build, current revisions, preserved
old-worker semantics and forward-repair limitations. Current app/worker/public
secret-version policy is unchanged. Strict release CLI evidence checks pass for
both manifests. `make check test build release-test`, diff validation and the
staged runbook guard pass locally; candidate CI provides the pinned-toolchain
and full-stack/runtime evidence for the unchanged product source.

## Remaining work

Wait for complete candidate evidence, review/merge exact release manifests,
execute migration, application and paused-worker workflows, then diagnose and
repair checking and register/qualify the additional fictional loans. Preserve
every unproved #88 webhook/lifecycle/reconciliation/no-journal requirement.

## PR handoff

Issue #258. Runtime source is the merged implementation above; migration 28 is
additive and must precede its new fixed decoder codes. New web must understand
codes before new worker emits them. No money limit or posting policy changed.
The same exact tested candidate images must be used without rebuilding between
migration and runtime stages. Keep receipts bounded; never upload state/plans,
private provider diagnostics or financial records.
