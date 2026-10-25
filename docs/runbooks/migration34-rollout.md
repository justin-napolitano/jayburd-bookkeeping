<!-- stage-runbook: true -->
# Schema34 development release — #26/#90

## Status and boundary

MIGRATE34-DEV through2026-10-25T07:00Z. Dedicated
release/34-development-migration from merged main
fbafb2ad9b5020187c22fd5b10e3e04828671957 (owner-approved PR340).
Owner requests deployment. This package applies only the existing development
migration root/job and then performs independent read-only exact schema/hash
verification. No new infrastructure, runtime/worker/webhook rollout, provider
call, credential retrieval, Production scope configuration or consent renewal.

## Stage ledger

| Stage | Evidence / limit |
| --- | --- |
| Resume | Current main/26/65, PR340 merge/all five checks, source leases released and fixed migration/verification contracts reviewed |
| Build | Trusted owner-dispatched main build40158617244 attempt1 from fbafb2a completed SUCCESS all seven jobs; exact API/migrate/web candidate receipts independently validated with run identity/source/branch/event/project and digest fields |
| Job readback | Bounded metadata only: jayburd-dev-migrate uses schema31 migrator92f4e18b..., fixed jayburd-dev-migrator identity, taskCount1/maxRetries0/timeout660. Latest selected execution jayburd-dev-migrate-vghhf succeeded2026-10-23T19:54:46.665929Z. This is job metadata, not fresh schema-version/hash acceptance |
| Preconditions | Foundation5 closed; bootstrap44 merged and ancestral; existing26 owner bootstrap evidence retained. No new role/bootstrap privilege is required by32–34 and no administrator product SQL is planned |
| Compatibility |34 is additive and private connection JSON uses additional fields; reviewed old deployed service decoder uses json.Unmarshal and ignores those fields, retains Sandbox references. Public function signatures/projections and existing Sandbox rows stay compatible. Current synthetic service/recovery gates passed34; no actual old-binary/managed-data rehearsal claimed |
| Database metadata | Existing jayburd-dev is RUNNABLE/POSTGRES_17 with backup and PITR flags enabled. This is configuration metadata, not a restore drill or fresh schema/hash proof. Diff from deployed31 source fb4ac04 adds only32–34 SQL pairs; existing1–31 unchanged |
| Local release gates | make check/test/build PASS;34 release/receipt/smoke tests and migration-only failure/isolation shell checks PASS; locked provider mirror, migration root backend-disabled initialization and four mocked Terraform cases PASS. No plan/state/provider-private diagnostic content read |
| Owner approval / source | Owner approved both exact hashes and sequential metadata merges/executions in-session. PR341873e7f9 passed all five checks and merged26ff59ff3dc9cae6de3b9a9e887a41088c8623a1. No scope or expiry changed |
| Apply | Guarded release40160644093 SUCCESS; retained receipt migration-execution-passed, selected execution jayburd-dev-migrate-zvq8b completed2026-10-25T04:08:02.087094Z with the approved image/source. Independent bounded Cloud Run readback validated by the existing receipt parser: fixed migrator/task1/retry0/660s, exact up arguments and one completed success. No financial rows/logs/environment/state read |
| Verify preparation | Dedicated release/34-development-verify from merged26ff59f; copied already-approved candidate byte-for-byte to fixed release path, hash a84c87e... unchanged. Separate status execution still required before clean34/full-checksum acceptance |

## Current evidence

Latest successful historical development deployment is40014820903; latest bounded
job metadata above retains the old image. The old committed release/consent window
is expired and must not be reused. Successful new image publication is not a release.
Forward32–34 preserve immutable existing financial data, posted journals and token
envelopes; migration34 installs no Production scope rows or new grants automatically.
The historical preparation stage above was database-inert. Apply40160644093 has
now completed successfully with independently checked execution metadata; exact
schema34/full-artifact status remains the next distinct acceptance step.

## Remaining work

Prepared exact operation=migrate and separate verify candidates with the fresh
same-build receipts and current compatibility evidence. Both release validators
PASS against the retained real build metadata and all three component receipts.
Required local core/release/mocked infrastructure gates PASS; PR341 all five
canonical checks passed. Status metadata copy uses the already-approved bytes;
its required local/core/staged and canonical checks precede its merge/dispatch.
deploy/README.md requires owner review of exact source/digests/hash before dispatch;
an updated manifest or expired approval needs a new hash review. Both present
hashes were explicitly approved; no further approval is needed for their unchanged
sequential execution before expiry.

After exact approval, merge/dispatch migration-only manifest, validate selected
immutable success receipt, then promote the byte-identical separately approved
verify candidate and require clean latest34 plus the full checksum set. On ambiguous
failure investigate bounded metadata; never force/down/retry a possibly committed
migration blindly. Do not inspect provider logs, environments, plans or state.

## PR handoff

Apply manifest deploy/dev-release.json SHA256:
473b4e0445a8a833645200ba9954671770d9ed3b488965240bfcaff27082a7dd.
Subsequent read-only deploy/migration34-verify.json SHA256:
a84c87eaa2d0f21f0304efa430296e6d942ec09ecf5ca9155b7e72a73c7f3d63.
Both expire2026-10-25T12:00:00Z; changing bytes/expiry requires new review.
Trusted build40158617244 attempt1/source fbafb2ad9b5020187c22fd5b10e3e04828671957.
Selected migrator digest199213ef1b299534dfbcc069cec71d351f20813a8cb22b90e53e99d057e00d1b.
Only this migrator executes; API/web references establish same-build provenance.
After successful migration receipt, copy the separately reviewed verify candidate
byte-for-byte to the fixed dispatch path in a separate reviewed metadata commit.

Refs26/90/311/313/314/315/316/337. Source311 is merged; source313/314 and actual
Sandbox/recovery/policy/resource/release gates still precede real account activation.
This database update does not establish Production credentials/runtime, a signed
Production receiver, actual independent hosted reconciliation or user MFA evidence.
Marked runbook in every implementation commit; staged check required before commit.
