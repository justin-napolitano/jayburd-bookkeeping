<!-- stage-runbook: true -->
# Hosted v15 replay proof

## Status and boundary

Issue #130: replay the exact previously successful v15 migration image, then
collect a separate clean-version/hash status receipt. This follow-up release changes only
`operation` from `migrate` to `verify`; it does not deploy application services,
change SQL, grant permissions, or connect a bank. Existing expiry remains binding.

## Stage ledger

- Owner authorized the hosted proof after PRs #137/#138 merged.
- New build 38758756744 passed for the independent API permission probe.
- Replay deliberately retains build 38654097056 and its migration digest, identical
  to successful execution `jayburd-dev-migrate-g4pj4` and subsequent clean15 proof
  `jayburd-dev-migrate-9vsmm`. The new API build is not substituted into this replay.
- PR #139 merged; replay workflow 38764164995 succeeded. Execution
  `jayburd-dev-migrate-6sm8n` completed 2026-10-15T15:55:17.700317Z.
  Independent allowlisted readback passed `scripts/migration-receipt.mjs`.
- Prepared the separate status-only manifest for owner review/merge.

## Current evidence

Prior execution and verification evidence is recorded in
[issue #26](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6194069357).
PASS: exact manifest validation and `make check test build release-test`.
Manifest SHA256: `b22eae58a60e3d2f0aa6304a19cab35bc9954f0841d275122ec9dd38e431a004`.
PASS: hosted API permission probe `jayburd-dev-database-probe-9fg4r`, completed
2026-10-15T15:42:03.088618Z. Exact image/identity/arguments/task/retry metadata was
accepted by `scripts/database-probe-receipt.mjs`; receipt status is
`api-database-permission-proof-passed`. Build 38758756744 supplies the API image.
This does not substitute for replay/status receipts; #130 remains open.

## Remaining work

After owner merge, dispatch Deploy dev with this exact verify manifest hash.
Validate the status execution receipt proving clean version 15 and all migration
hashes. #130 remains open until this final hosted receipt passes; #26 remains
open for its broader acceptance criteria. Application services are untouched.

## PR handoff

No migration or application code changes. Review the single manifest operation
change and retained digest/build/expiry before merging. Hosted replay passed; the post-replay status execution is still pending.
