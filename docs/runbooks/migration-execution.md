# First migration execution
<!-- stage-runbook: true -->

## Status and boundary

Owner authorized execution after merging #46 (`e65aeeb`). Scope: enable the
existing deployment federation and dispatch the exact migration-only manifest.
No API/web deployment, identity enrollment or real financial data.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Release | Approved hash and same-build receipts revalidated | Preserve exact manifest |
| Foundation | Baseline drift 0; enablement apply passed; post-apply drift 0 | Preserve both enabled identity flags |
| Dispatch | [38300826863](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38300826863), main `e65aeeb`; authentication and job execution succeeded | Workflow failed at receipt parsing |
| Recovery | Read-only selection of completed execution passed corrected validator | Merge receipt fix; do not rerun SQL just for a green workflow |

## Current evidence

- PASS: manifest SHA256 `3887a5442604a35948a92a10ec2de82a74543776547923bf9bbd38ba2ef1c1bc`;
  build 38296754686 attempt 1 and all three receipts match.
- PASS: merged foundation source with existing backend, project and billing inputs;
  only requested input change enables deployment federation. Build enablement
  preserved. Terraform plans/state/provider diagnostics remain private.
- PASS: provider readback ACTIVE with disabled false/omitted and exact numeric
  owner/repository/actor, main, workflow_dispatch and deploy workflow restrictions.
- PASS: `jayburd-dev-migrate-7fg5g` completed at `2026-10-12T19:59:22.195422Z`:
  Completed=True, one successful task, no failed/cancelled/running/retried counts.
  Selected immutable image, IAM account, arguments, one task, zero retries and
  660-second timeout match the approved manifest. This proves the migrator process
  succeeded under its own identity; separate SQL version/readback/replay remains.
- FAIL: hosted receipt parsing omitted container/condition array fields and
  required a numeric timeout, while Cloud Run returns a decimal string. Workflow
  remains failed and emitted no deployment artifact; it did not reach services.
- PASS: explicit `[]` projections recovered only allowlisted metadata; corrected
  validator accepted that existing execution without another job or SQL call.
  Local bounded receipt is `.local/recovered-migration.json`, not a successful
  hosted artifact or full application deployment receipt.
- PASS: `make check test build release-test` (15 release tests, synthetic
  orchestration including array projection and decimal-string timeout regression),
  shell syntax. No SQL, job definition or manifest bytes changed.
- The same array-path repair covers the later service image readback; synthetic
  command assertions pass, but no hosted service rollout was performed here.
- PASS: `make migration-test` on disposable synthetic PostgreSQL: fresh/replay,
  down/up harness, rollback/dirty-state failure and all five product migrations.

## Remaining work

Review the receipt fix, then separately verify database version/checksums and plan
the replay/IAM-denial acceptance. The migration already succeeded; never force dirty migration
state or run down migrations. Keep #26/#27 open for actual runtime IAM denial,
operator ownership cleanup, restore, pool/load and application acceptance.

## PR handoff

Receipt-reader repair and execution evidence; no new SQL or application behavior. Current private
foundation directory: `.local/foundation` in `/private/tmp/jayburd-migration-execution`.
Backend: existing foundation bucket, prefix `foundation`. Future applies preserve
both `enable_github_build_identity=true` and `enable_github_deploy_identity=true`.
Do not apply the database root or remove the retained SQL operator for this stage.
