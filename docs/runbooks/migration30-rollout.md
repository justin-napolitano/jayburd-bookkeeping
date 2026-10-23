# Migration 30 development rollout
<!-- stage-runbook: true -->

## Status and boundary

Issue #272; owner requested applying merged migration30. Dedicated lease
MIGRATE30-20261023T0407Z covers migration-only release and schema verification.
Source edb778ed4377f6763cb6c618052ed2680068a6f5 (merged PR277).
No application rollout or hosted verifier acceptance is included.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Recheck | Main and released predecessor leases checked; dedicated worktree clean | Build trusted images |
| Build | Run39898191502 started from merged main | Require terminal successful receipts |
| Bootstrap | Restricted migrator cannot create missing NOLOGIN capability | Check approved role via SQL operator |
| Release | Not prepared or dispatched | Review exact manifest and provenance |
| Verify | Not run | Require clean latest30 status |

## Current evidence

Existing dedicated job uses migration29 image with status-only action. Selected
metadata confirms fixed migrator workload identity. No migration30 execution yet.
Bootstrap prerequisite follows merged db/bootstrap/cloud-sql.sql. SQL operator
must only create/validate the safe NOLOGIN capability, not run product migration.

Cloud Shell SSH is reachable, and installed proxy/psql are available. Its GCP
automatic authentication is disabled and no active account was returned. Owner
sign-in requested before SQL access; no authorization-session transfer attempted.
Twenty release-manifest/migration-receipt tests passed. Local core checks, race
tests and build use cached Go1.26.5 with a temporary modfile, leaving tracked
Go1.26.10 unchanged; canonical pinned CI remains authoritative. The initial
default local toolchain download failed with a connection reset.

## Remaining work

Complete role prerequisite, trusted image build, guarded migration-only release
and independent status execution. Issue272/88 hosted acceptance remains open.

## PR handoff

Release metadata only; merged migrations remain immutable. Follow deploy/README.md
and docs/contracts/migration-release.md. Record exact execution receipts, validation
and any remaining risk before handoff.
