<!-- stage-runbook: true -->
# Production foundation candidate — #316

## Status and boundary

PROD316-FOUNDATION; build/316-production-foundation froma572b31.
Source/mock/disposable SQL tests only; no live cloud plans/state, apply, resource
creation, secrets, provider access or real database operations.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |355 merged;316/337 open, no conflicting writer; existing source/contracts reviewed |
| Design | Separate micro SQL/build root; disabled builder federation and null human admin; exact Production bootstrap wrapper |
| Implementation | New root/closed identities; original dev roots and generic SQL unchanged |
| Validation | 10 mock-provider tests PASS; synthetic bootstrap, core check/test/build, migration-test and release-test PASS |

## Current evidence

Local Go1.26.5 with external temporary modfile/cache; tracked1.26.10 unchanged.
Terraform1.15.7/Google8.6.0 locked provider, clean mock-only copy, no live backend.
Initial sandbox DNS failure was retried with approved provider download access.
Target is proposed under337, not approved or provisioned. Build publication workflow
still has no target file. SQL bootstrap wrapper checks names only; actual server
identity/administrator capability and resulting grants require private readback.
USD25/month is GCP-only and includes retained Sandbox cost; this root alone does
not enforce that total. Shared-core capacity and restore rehearsal remain unproved.

## Remaining work

Human security review, actual337 project/cost/retention approval, isolated backend
bootstrap, Production runtime keys/secrets/evidence, Jobs/services/queues and
reviewed finite deployment. No deployment federation or runtime permission added.

## PR handoff

Refs #316, infrastructure/bootstrap source slice; parent remains open. No product
migration or financial API change. Tests cannot establish effective live IAM.

## CI portability correction

Initial validate run40717990840 failed because the hosted runner lacks ripgrep.
The two fixed-string assertion checks now use standard grep -Fq; no bootstrap
behavior or acceptance criteria changed. Rerun the disposable bootstrap proof
and require green CI on the corrected head before merging.
