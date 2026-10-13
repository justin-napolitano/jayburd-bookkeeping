# Workload-owned migration status
<!-- stage-runbook: true -->

## Status and boundary

Continue #26/#27 from merged #48 (`5cb958b`). Owner restored Cloud Shell access.
The IAM proxy now starts; the human operator's metadata read is denied with 42501,
and its existing migrator SET ROLE capability is false. No grants were added and
no replay ran. Proxies stopped through cleanup traps.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Cloud retry | Authentication restored; metadata SELECT denied; existing role-switch unavailable | Keep operator separation |
| Decision | [Status contract](../contracts/migration-status.md): same job/identity, read-only SQL snapshot | Implement without widening grants |
| Implementation | `status` command and `verify` release operation; exact completion/argument receipt | Review PR |
| Validation | Go/race/build, 18 release tests, isolated SQL status/migration suites, 24 mocked Terraform runs, actionlint and shell syntax pass | Build fresh images after merge |

## Current evidence

No cloud version/hash acceptance is claimed. The first execution remains successful
as documented in [execution evidence](migration-execution.md); its metadata is not
accessible through the human operator. Cloud query/binary hashes and temporary
directory remain in [replay evidence](migration-replay.md).

Local status checks require one clean current version and exact registered file
hashes in repeatable-read/read-only mode. A forced RLS test fixture exposes metadata
only when `transaction_read_only=on`; malformed/missing/dirty/drifted metadata and
elevated/API identities must fail. The initial fixture exceeded the migrator's
two-connection limit; its preliminary reader now closes before migration setup.
No production connection limit was changed.

PASS: `make check test build release-test cloud-sql-test migration-test infra-test`;
actionlint for the changed workflow, shell syntax and diff checks. Named
`TestMigrationStatus` proves missing/dirty/behind/ahead version, missing/extra/changed
artifact, elevated/API identity rejection and forced read-only visibility.
Release CLI tests prove `verify` emits `action=status` with no service variables;
orchestration rejects an `up` execution and never reaches runtime roots.

## Remaining work

Merge, build fresh images, then prepare a separate exact-digest `verify` manifest
for owner review. The currently committed manifest still requests `migrate` with
the old image: do not dispatch it for inspection. A successful status snapshot
permits a separately reviewed replay, followed by another status check.
Actual API IAM denial, operator ownership cleanup, restore and load remain open.

## PR handoff

No new SQL, data access grants, identity or cloud resources are applied here.
Verification updates only the existing job's reviewed action, never service roots.
Current old manifest bytes are unchanged. Keep #26/#27 open; implementation and
local evidence do not substitute for the new image's hosted acceptance.
