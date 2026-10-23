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
| Build | Run39898191502 attempt1 SUCCESS all7 jobs from merged main; all three receipts validated | Use exact image digests |
| Bootstrap | Minimal role transaction committed; all six unsafe attributes false | No verifier login/mandate provisioning |
| Release | PR279 merged e809024; owner approved both exact hashes; run39967657376 SUCCESS | Preserve immutable receipt |
| Verify | Approved status bytes copied to fixed release path; not dispatched | Require clean latest30 status |

## Current evidence

Existing dedicated job uses migration29 image with status-only action. Selected
metadata confirms fixed migrator workload identity. No migration30 execution yet.
Bootstrap prerequisite follows merged db/bootstrap/cloud-sql.sql. SQL operator
must only create/validate the safe NOLOGIN capability, not run product migration.

The paragraph above records the initial state. Actual migration-only
[release39967657376](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/39967657376)
is terminal SUCCESS. Its bounded deployment receipt reports
`migration-execution-passed`, exact build39898191502 attempt1/image digest below,
execution `jayburd-dev-migrate-ld68l`, completed2026-10-23T14:14:44.108669Z.
This establishes successful execution; independent latest-version/checksum status
remains the following release. No product SQL was run by the operator.
PR279 all five exact-head checks passed before native squash merge; an initial
merge/dispatch attempt under the unexpectedly switched managed GitHub profile was
denied without effects. Restoring the owner profile allowed native merge and the
guarded exact-hash dispatch. No branch gate bypass was used.

Initially Cloud Shell lacked an active account; owner then authorized the existing
operator session. Pinned proxy2.26.0 automatic IAM/private socket succeeded. First
preflight attempted migration metadata SELECT, which the operator correctly lacked;
the transaction rolled back before capability creation. No privilege was widened.
The minimal follow-up only created/validated bookkeeping_verifier using merged
bootstrap semantics. LOGIN/SUPERUSER/CREATEDB/CREATEROLE/REPLICATION/BYPASSRLS are
all false; no login, membership or mandate was added. Trap stopped each proxy.
[Actual bootstrap evidence](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/272#issuecomment-6366462888).

Thirty release/receipt tests and migration-only isolation script passed. Local core checks, race
tests and build use cached Go1.26.5 with a temporary modfile, leaving tracked
Go1.26.10 unchanged; canonical pinned CI remains authoritative. The initial
default local toolchain download failed with a connection reset.

## Remaining work

Owner approved both exact candidate hashes before dispatch. Migration application
and validated receipt are complete. This follow-up copies deploy/migration30-verify.json
byte-for-byte to deploy/dev-release.json; merge and dispatch its already approved
hash for independent clean-latest30/checksum evidence. Core check/test/build passed
again with the same documented local toolchain limits. Do not blindly retry failed
receipts, force migration state or run administrator product SQL. Issue272/88 hosted
acceptance remains open after this schema-only work.

## PR handoff

Exact migration manifest SHA256:
`279a35ca1a05836cc3f439544db1345402e41ec4989f735396c85ff3d0a6afdb`.
Exact following status manifest SHA256:
`d11f04988dde3ecd7c72cfc23d9b7689f26c1cabab003a0d36ca2f1f4b59def4`.
Both expire2026-10-24T13:00:00Z. Trusted build39898191502 attempt1;
migrator digest `da41aa09a13e8652228697ff90180eb20cc69ad913b0d35e2fbc2058c843258e`.

Release metadata only; merged migrations remain immutable. Follow deploy/README.md
and docs/contracts/migration-release.md. Record exact execution receipts, validation
and any remaining risk before handoff.
