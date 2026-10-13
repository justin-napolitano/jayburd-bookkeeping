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

## Evidence and next action

The SQL authority is the existing migration library's version and artifact tables;
the execution contract and serialization sources are G27/G29. Existing local
restricted-role tests establish the synthetic baseline. Add status mismatch and
read-only checks, release isolation/argument tests and mocked Terraform action
tests before review. Actual cloud status requires a fresh build and separately
reviewed `verify` manifest. Only a clean status permits the planned replay stage.
