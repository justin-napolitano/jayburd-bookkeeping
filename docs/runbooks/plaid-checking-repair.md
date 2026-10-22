<!-- stage-runbook: true -->
# Shared Plaid money repair — #258

## Status and boundary

Owner requested a checking repair that applies consistently across all accounts.
Lease `PLAID-258-ROLLOUT-20261022T0102Z`; base `7737991`. Separate repair worktree
and branch `fix/plaid-checking-258`; release manifests remain on the release
branch. Synthetic fixtures only; no private source amount or payload inspected.

## Stage ledger

1. **Baseline — complete.** PR #259 merged safe classification and additional
   loan registration. It deliberately did not change accepted money quantities.
2. **Cross-account regression — prepared.** Named fixed-expected money matrix
   covers all seven supported account pins and JPY/USD/KWD. It passed through an
   external Go overlay before being copied into this dedicated repair branch.
3. **Hosted diagnosis — complete.** Candidate rollout succeeded and the single
   checking retry (generation 2, completed 0) returned
   `currency_precision_exceeded`, decoder rejected, no active lease and no
   publication. Only fixed status fields were inspected. This proves the branch,
   not the intended source value or a floating point artifact.
4. **Repair and validation — synthetic checks passed.** The source precision decision was
   recorded before changes; it preserves bounded exact source quantities and
   holds sub-minor-unit rows while permitting atomic publication of other rows.
   Preserve quantity, original lexeme, currency and
   source/account identity. Never silently round, drop a transaction or substitute
   a zero quantity.
5. **Hosted acceptance — pending.** Complete all seven account-type sync/review
   checks after the source-backed repair; preserve the broader #88 gates.

## Current evidence

`TestPlaidAccountMoneyMatrix` passes baseline checks for checking, credit card,
student, mortgage, auto, consumer and line of credit. Each uses six independently
fixed ordinary examples, three precision-held examples across three currencies
and two fixed storage rejections. It asserts exact lexical retention, rational quantity equality,
fixed diagnostic codes and zero partial page on rejection. This does not prove
hosted completeness or ledger approval.

`TestSourcePrecisionHold` and `TestPrecisionEligibilityForgery` pass against real
disposable PostgreSQL: mixed exact publication, one held and one reviewable row,
original signed lexeme, exact replay, zero journals, direct draft refusal,
forged-eligibility/rounded-value rejection and retained-evidence down refusal.
`make check test build`, `make migration-test ledger-test`, `make acquisition-test`,
`make web-check web-test web-build` and `make app-integration` passed. Web unit
tests: 290; component browser: 18, including the new desktop/mobile precision
hold case. Independent rational fuzz: 1,682,846 executions
in 30 seconds, no quantity/readiness failure. Core local toolchain is Go 1.26.5
with the external temporary modfile; tracked pin remains unchanged for CI.
Docker registry returned EOF; exact cached digests were verified and their
missing local version tags restored. A temporary adapter skips only the two
exact pinned image pulls, preserving all other Docker operations and real SQL.
Full-stack result proves synthetic browser/API/SQL behavior, not verified hosted
identity or source provider behavior.

## Remaining work

CI found migration 29 omitted the transaction-local `bookkeeping_owner` role
selection required by the restricted migrator. The disposable owner-level suites
did not exercise that path. Added the same explicit role selection used by
existing migrations to both directions; no privilege grant or runtime change.
`make cloud-sql-test worker-bootstrap-test` now passes with the restricted
synthetic migrator, including its exact replay and status checks. CI must pass
again before any hosted migration.

Submit reviewed migration 29 and shared decoder/frontend repair, complete CI,
publish a pinned candidate and roll it out under the approved Sandbox scope.
Retest checking and all additional fictional loans. No source data was rounded.

## PR handoff

Keep #258 open until actual checking repair and additional fictional loan
registration are verified. No hosted fix, complete history, automatic posting,
webhook, reconciliation or live-use approval is claimed from this baseline.
