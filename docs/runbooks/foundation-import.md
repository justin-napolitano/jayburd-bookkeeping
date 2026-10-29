<!-- stage-runbook: true -->
# Shared foundation imports — #316

## Status and boundary

fix/316-import-graph from28c36b7. Owner approved363; all five checks passed and
363 merged. Protected/versioned Production foundation state bucket created and
metadata verified; private Terraform init succeeded. No foundation apply yet.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source parity | Private operator .tf/lock files match merged363 |
| Import | First import rejected before successful completion: Invalid for_each argument in secret reader/delivery and API KMS IAM graph |
| Correction | Use static destination/key sets for graph expansion; same keys, resource addresses, roles and recipients |
| Validation | Three infrastructure mocks, static-key regression and core check/test/build PASS |
| Actual imports | All10 resources imported: backend bucket, Production DB, two identities, two SQL IAM users, four project connector/login IAM bindings |
| Foundation apply | Original approved merged363 source applied successfully; raw provider output/state private |
| Metadata readback |10 Production identities,5 IAM SQL users,10 secret destinations,2 encryption keys,7 private/versioned buckets and immutable Production image registry; build federation disabled |

## Current evidence

Terraform mock planning did not reproduce the imperative import graph restriction.
Do not claim an import succeeded from planning alone. Raw provider output/state
remain private; failure classification emitted only fixed categories and source
resource locations. No second SQL instance, Sandbox changes, new secret payload
or financial data access occurred. This correction preserves the exact approved
IAM scope rather than expanding it. Source authority is existing shared foundation
and its reviewed scoped worker permissions; financial/ISO semantics unchanged.
SQL-user import initially used a REST path and failed. Provider8.6.0 documents
project/instance/name; corrected imports succeeded. G24 records that selected
source review. Imports alone do not establish new foundation provisioning.
Static graph correction was used only for imports. Foundation apply uses the
original reviewed merged363 source; planning can resolve resource-derived keys.
Instance readback remains PostgreSQL17/us-central1/db-g1-small/10GiB/ZONAL;
this root owns no instance settings. Both static secret destinations have no
versions yet. No secret payload was retrieved. Actual worker SQL permissions,
secret propagation, identity recovery or hosted application acceptance are not
established by this provisioning success.

## Remaining work

Imports/apply and infrastructure metadata verified. Confirm actual least
privilege/runtime sessions, remaining worker SQL bootstrap, secret
propagation and hosted readiness. Keep316/337 open; no live bank activation.

## PR handoff

Refs #316. No DDL/migration or permission changes. Retain existing Production
database and identities; do not recreate them to bypass import failure.
