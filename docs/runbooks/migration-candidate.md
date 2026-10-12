# First migration candidate
<!-- stage-runbook: true -->

## Status and boundary

Prepare #26/#27's first migration-only release from merged #45. Manifest:
[deploy/dev-release.json](../../deploy/dev-release.json). No new SQL or service
configuration. Owner review is pending; no cloud mutation or dispatch in this stage.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Hosted build | [38296754686, attempt 1](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/38296754686/attempts/1), all jobs successful | Preserve same-run receipts |
| Candidate | Source `32526e5`; exact manifest/build validation passes; all three retained scans have zero HIGH/CRITICAL findings | Owner reviews release PR |
| Compatibility | [Evidence](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/26#issuecomment-6131373086); SQL/bootstrap/migrator/job unchanged since #44 | Apply forward only after approval |
| Checks | `make check test build release-test` passed, including 14 release tests and synthetic orchestration | Final doc/hash/staged checks |

## Current evidence

CLI validation against downloaded build metadata and all candidate receipts passed.
Manifest hash below binds the exact release bytes; no runtime Terraform input was
generated for this migration-only operation. Bootstrap evidence is linked from
the manifest and remains bounded to the earlier administrator session.

## PR handoff

1. Review and merge the candidate PR. Execution applies five merged forward SQL
   migrations to the development database through the dedicated IAM migrator.
   Job bounds: one task, zero retries, 660 seconds, 1 CPU/512Mi. API/web remain
   undeployed. Existing database/billing scope is unchanged; job execution adds
   bounded compute/log usage. This is not a full cost or live-bookkeeping approval.
2. Before dispatch, review and apply the existing foundation Terraform with
   `enable_github_deploy_identity=true`, preserving
   `enable_github_build_identity=true` and existing project/billing/backend inputs.
   Federation was last observed disabled; this stage does not enable it. Follow
   [foundation operations](../../infra/gcp/foundation/README.md). Never apply the
   database root or remove the retained SQL operator for this access change.
3. Recheck merged manifest bytes and current main, then run **Deploy development**
   on **main** with `release_sha256`:

   ```text
   3887a5442604a35948a92a10ec2de82a74543776547923bf9bbd38ba2ef1c1bc
   ```

   Approval expires **2026-10-15 20:00 UTC**; changed bytes/expiry require renewed
   review and hash. Confirm locally with `shasum -a 256 deploy/dev-release.json`.

## Remaining work

Expect a successful immutable execution receipt before claiming migration success.
Actual IAM login, SQL version/checksums/replay, API DDL denial, operator ownership
cleanup, restore and pool/load remain #26 work. Never force dirty state, down-migrate
or blindly rerun a failure: SQL may have committed before readback failed.
Keep #26/#27 open. This manifest deploys no identity or finance service.
