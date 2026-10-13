# Posting acceptance follow-up

<!-- stage-runbook: true -->

## Status and boundary

Decision: prove the existing principal-deletion and retention boundaries for
[#4](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/4), without
inventing an erasure policy or changing SQL/runtime behavior. This follows the
[original acceptance audit](core-acceptance-audit.md). #4 remains open pending
review of the acceptance wording and unresolved lifecycle scope.

## Stage ledger

| Stage | Evidence / result | Next action |
| --- | --- | --- |
| Review | Read live #4, posting/identity contracts, migrations and existing invariant tests | Add only missing deletion proofs |
| Implementation | Two disposable PostgreSQL tests distinguish physical deletion, historical retention and disable | Run default and ledger gates |
| Validation | Default checks and full disposable ledger suite passed on 2026-10-13 | Review deletion interpretation; keep #4 open |

## Current evidence

New tests in [posting gate integration](../../internal/ledger/posting_gate_integration_test.go):

- `TestDeletedUnusedPrincipalDenied`: create then physically delete an unused
  principal; posting returns `not_found` and all durable posting state is unchanged.
- `TestHistoricalPrincipalRetentionAndDisable`: a separate poster with no policy
  approval reference posts, then membership removal leaves only historical
  references. Administrative deletion fails with exact SQLSTATE `23503`; principal
  and full ledger snapshots remain unchanged. With memberships restored, disabling
  denies fresh posting, historical receipt replay and journal read. An active,
  authorized actor still reads that history.

Observed PASS on 2026-10-13: `make check test build`, then `make ledger-test`,
using Go 1.26.9 and the script-pinned PostgreSQL/migrate images. Both new tests
passed, as did the existing invariant, replay, close-race, historical reversal,
forward-upgrade, restore-replay and runtime-readiness tests. The initial ledger
invocation hit the sandbox Docker image-helper error; the approved outside-sandbox
rerun passed without inspecting credentials. Logs: local
`/private/tmp/jayburd-posting-gate-tests.log` and
`/private/tmp/jayburd-posting-gate-ledger.log` (not required repository inputs).

Source authority is the existing
[posting boundary](../contracts/posting-boundary.md),
[identity contract](../contracts/identity-setup.md), immutable migrations 000001/2
and [P03–P06 source review](../research/posting-sources.md). No new accounting
policy, financial message, external-source interpretation or ISO claim is added.

Existing tests already cover the remaining narrow posting criteria:
`TestPostingInvariants` injects every write-stage/deferred-COMMIT failure;
`TestPostingReplayConcurrency`, `TestCommitAcknowledgementLoss` and
`TestPostingCloseRace` cover replay/uncertain commit/lock ordering;
`TestSpotReversalHistoricalProvenance` compares exact original/reversal lines,
rate/currency/policy identities with an independently posted newer rate;
`TestReversalOfReversalDenied` rejects second-order correction unchanged.
These are retained, not duplicated.

## Remaining work

The phrase “deleted” in #4 does not define a user-facing deletion/erasure,
retention, identity-provider deletion propagation or anonymization lifecycle.
The tests establish only existing database authorization and foreign-key behavior.
Owner review must settle that scope before claiming the complete criterion or
using issue closure to unblock #7. Live policy, full FX, imports, close evidence,
worker delivery and hosted reliability remain separate gates.

## PR handoff

Review the named assertions and deletion qualification. No migrations, runtime
code, provider calls or live data are changed. Default and ledger gates passed; reviewers can rerun `make check test build
ledger-test` with the pinned Go toolchain. No browser/API/migration behavior changed.
