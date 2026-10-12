# Migration-only development release

## Decision

Add `operation: migrate` to the existing exact-hash deployment workflow. It applies
only the migration Terraform root, executes the dedicated job, validates its
immutable execution metadata and exits before runtime service changes. Owner
reviews the manifest and dispatch hash. The existing deployer identity and
concurrency group remain; no new federation trust or runtime privilege is added.

This resolves the ordering dependency: #26 needs migration evidence before full
application promotion can require #26 closed. Alternative owner-terminal migration
apply is possible, but would duplicate release validation and execution tracking.
This is infrastructure configuration, not a new ISO financial message or SQL policy.

## Contract

The migration manifest contains exactly schemaVersion, project, projectNumber,
operation, build, images, evidence and expiresAt. Keep all three same-build image
receipts; only images.migrate can execute. Evidence has bootstrap and compatibility
issue-comment links; no browser configuration or rollback field is accepted.
Hash, expiry, owner/main successful build and exact image checks remain unchanged.

Require foundation #5 closed and bootstrap PR #44 merged and ancestral to both
the workflow source and image source. Owner bootstrap evidence approves the SQL
preconditions. #26 remains open; #6/#11 are unrelated to this IAM-only job and
remain mandatory for promote/rollback. A migration-only manifest cannot reach
either service root or anonymous application smoke.

## Evidence and recovery

Selected execution metadata must match the approved digest, migrator account,
exact forward-only arguments, one task, zero retries and timeout. Require completed
success, one successful task, no failed/cancelled/running/retried tasks. Record only
execution identity, approved build/image and completion time. No environment, logs
or state is uploaded. A missing/mismatched receipt blocks acceptance even if SQL
may already have committed; investigate before rerunning. Never force or down.

Use explicit `[]` projections for container and condition array fields. TaskSpec
serializes its int64 timeout as a decimal string; accept exactly `"660"` or numeric
`660`, with no general coercion. Source G29 and the first real execution exposed
this distinction; [recovery evidence](../runbooks/migration-execution.md) records
read-only verification after a successful job and failed hosted receipt parsing.

Sources G19 and G27 establish job execution and immutable metadata; synthetic
release/orchestration tests prove rejection/order only. Actual job and SQL version,
checksums and replay remain live #26 evidence. Operator ownership cleanup remains
required before runtime release.
