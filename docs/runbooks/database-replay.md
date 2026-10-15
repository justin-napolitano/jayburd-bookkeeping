<!-- stage-runbook: true -->
# Hosted v15 replay proof

## Status and boundary

Issue #130: replay the exact previously successful v15 migration image, then
collect a separate clean-version/hash status receipt. This release changes only
`operation` from `verify` to `migrate`; it does not deploy application services,
change SQL, grant permissions, or connect a bank. Existing expiry remains binding.

## Stage ledger

- Owner authorized the hosted proof after PRs #137/#138 merged.
- New build 38758756744 passed for the independent API permission probe.
- Replay deliberately retains build 38654097056 and its migration digest, identical
  to successful execution `jayburd-dev-migrate-g4pj4` and subsequent clean15 proof
  `jayburd-dev-migrate-9vsmm`. The new API build is not substituted into this replay.
- Prepared a migration-only manifest for owner review/merge before dispatch.

## Current evidence

Prior execution and verification evidence is recorded in
[issue #26](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6194069357).
PASS: exact manifest validation and `make check test build release-test`.
Manifest SHA256: `9c601f13b807fa60b881a83eb01f2e63f0b388052cf03750ee8accb1981290fd`.
The independent API probe outcome is recorded in #130; it does not substitute
for replay/status receipts.

## Remaining work

After owner merge, dispatch Deploy dev with this exact manifest hash. Validate the
migration execution receipt, then prepare and review a separate `verify` manifest.
Keep #130 open until API probe, replay and subsequent status all pass. Keep the
broader #26 open for its remaining acceptance criteria.

## PR handoff

No migration or application code changes. Review the single manifest operation
change and retained digest/build/expiry before merging. No hosted replay is claimed
by preparing or merging this PR.
