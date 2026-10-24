# Read-only migration status release

## Decision

Add `operation: verify` to the existing exact-hash release workflow. It configures
the existing migration job with `-action=status`, under the existing migrator IAM
identity. No new identity, grant, job, database or financial message is introduced.
This is internal deployment metadata; existing ISO financial assessments stand.

The retained human operator cannot read deployment tables or SET ROLE to the
migrator. Granting it broader access would undermine the current separation.
Running another `up` to discover status would mix inspection with mutation. Use
the workload's existing authority for a read-only transaction instead.

## Contract

Keep the migration-only manifest fields, all three successful same-build receipts,
owner/main/hash/expiry checks and bootstrap gates. Verification needs a new image
implementing status; the earlier approved migration image does not support it.
Terraform accepts only `up` or `status`; no arbitrary job argument is accepted.
Verification may update job configuration, but never runs application rollout.

Status verifies embedded source checksums and restricted migrator privileges, then
opens a read-only transaction. Require exactly one clean schema-version row at the
highest embedded up migration and an exact filename/hash match for every registered
artifact, with no missing/extra entries. No grants, DDL, registration, force, down,
application reads or financial payloads. Errors emit the existing fixed failure.

An exact successful `status` execution emits `migration-verification-passed`,
distinct from `migration-execution-passed`; neither qualifies for app rollback.
Missing tables, dirty/wrong versions, artifact drift and unsafe identities fail.

## Owned Hosted repair rollback verification (#328)

The separate repair-enabled Sandbox profile adds the same exact selected-image
read-only status/hash check before `sandbox-rollback` runtime rollout. It may
update/execute the existing job with `-action=status`, never up/down, using the
existing workload identity. The same-build source ancestry includes33 and the
repair runtime; the actual database must match the selected image's highest
embedded version and complete hash set. This adds no direct API table grant or
startup probe. Legacy rollback without the explicit capability stays job/DB-inert.

This operation can emit a bounded `migration-verification-passed` receipt for its
own read-only step; it is not a substitute prior application rollback receipt.
The prior successful Sandbox transport receipt and identical approved connection
digest remain required. A failed/foreign/up/retried/mismatched status execution
stops before runtime changes. Source and mocked orchestration tests establish
these mechanics only; actual job/SQL authority still needs scoped hosted evidence.

## Evidence and next action

The SQL authority is the existing migration library's version and artifact tables;
the execution contract and serialization sources are G27/G29. Existing local
restricted-role tests establish the synthetic baseline. Add status mismatch and
read-only checks, release isolation/argument tests and mocked Terraform action
tests before review. Actual cloud status requires a fresh build and separately
reviewed `verify` manifest. Only a clean status permits the planned replay stage.
