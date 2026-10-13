# Migration metadata and replay
<!-- stage-runbook: true -->

## Status and boundary

Continue #26/#27 after merged #47 (`10a50b1`). Owner authorized continuing cloud
acceptance. Read migration-control metadata only; replay the approved migrator
only after a clean version/hash check. No new SQL, privileges or service deployment.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Prerequisite | #47 merged, hosted checks passed | Verify database metadata |
| Metadata query | Read-only transaction and pinned proxy transferred with hash checks; proxy could not start, no SQL ran | Owner restores Cloud Shell authorization |
| Replay | Not dispatched | Requires clean baseline |
| Denial tests | `make cloud-sql-test` passes with SQLSTATE 42501 and repeatable-read P2101 checks | Use these exact outcomes for the future cloud API proof |

## Current evidence

The first migration succeeded; its original hosted receipt failed and was recovered
read-only as recorded in [execution evidence](migration-execution.md). This stage
does not reinterpret that failed workflow as successful.

Metadata-query SHA256:
`0dfb84cc83db9cede248a3295a039da321b3def08fa8bed4a11cab0dbe0fdcac`.
Query compares `deployment.schema_migrations` and `migration_artifact` only and
returns a fixed PASS/BLOCKED result; it reads no journals or identity rows.
Expected hashes come from the approved source's `db/migrations/SHA256SUMS`.
Cloud Shell temporary directory: `/tmp/jb-replay.m8xFAg`; private query/proxy logs
remain excluded from Git. Existing operator retained; no new grants are authorized.

Cloud Shell restarted and lost the earlier temporary files. Direct proxy download
timed out; local official v2.26.0 download and transfer succeeded, SHA256
`a38fe97690a27490e60a7945a8a178186ff9464abb81fe2bc652a3f773443387`.
The verified binary reports v2.26.0. Its private socket did not start; fixed-category
diagnostics found a connection-refresh failure. A separate existing-CLI project
number lookup also failed. No raw error, token, credential or financial data was
printed. Owner was asked to reauthorize Cloud Shell; no authorization transfer or
new privilege grant was attempted. Proxy cleanup ran, and replay was not dispatched.

The local permission tests previously accepted any error. They now require
PostgreSQL `42501` for DDL/direct-table/SET ROLE denial; zero-row reads prevent
payload retrieval. The workspace unknown-principal test now uses repeatable read
and requires `P2101`, avoiding a false pass on its `P2100` isolation precondition.
These are synthetic checks, not proof of Cloud SQL API identity authorization.
`make cloud-sql-test` passed restricted migration/replay, permission denial,
contention, artifact integrity and dirty-failure checks in disposable PostgreSQL.
`make check test build` also passed (vet/race, source integrity, docs and binaries).

## Remaining work

Verify clean version and all artifact hashes before and after replay. Retain the
successful hosted execution receipt if emitted. Actual API IAM login and forbidden
DDL/direct-table access tests, operator ownership cleanup, restore and connection
load/recovery remain separate #26 gates.

Owner recovery: open Cloud Shell for `jayburd-bookkeeping-dev`, run
`gcloud projects describe jayburd-bookkeeping-dev --format='value(projectNumber)'`,
and approve its authorization prompt. Expected public output: `9088794002`.
Do not send credentials. Then rerun only the bounded read-only check first;
successful CLI authentication alone does not prove the proxy or SQL session works.

## PR handoff

Do not dispatch or repeat a job after a failed metadata check. Investigate without
force, down migration or privilege expansion. Keep #26/#27 open until their named
acceptance criteria pass; metadata and replay alone do not approve live bookkeeping.
