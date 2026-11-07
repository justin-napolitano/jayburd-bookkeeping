<!-- stage-runbook: true -->
# Private workspace Sandbox rollout —436

## Status and boundary

PRIVATE-ROLLOUT436-20261107T0505Z; ops/436-private-rollout from merged452/f8d9e06.
Owner explicitly requested controlled deployment after the verified source build.
GitHub436/65 own execution and approval state. Existing Sandbox API/web and
restricted jayburd-dev-migrate/bookkeeping in jayburd-bookkeeping-dev,
9088794002/us-central1 only. No Production change, provider call, consent renewal,
source-cap increase, worker execution/scheduling, IAM or new cloud resource.
No credential/environment/state/plan, financial-record or provider-payload reads.
Existing source/journal identities and immutable migrations1–44 are preserved.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Mainf8d9e06 verified,452 source passed all five gates,436/439 previous rollout evidence and managed contracts read; dedicated worktree/live claim |
| Trusted build | Owner/main Build images42341240949 at exactf8d9e06, attempt1: provenance, all three validation jobs and all four candidate runtime proofs pass. All four scans reject fixed-version HIGH findings; publication/authentication skipped. See build-evidence436.md. Isolated dependency repair required before a new trusted build |
| Sandbox predecessor | API00022-2sq/e15d8c01… and web00022-mzn/b525fe62… Ready100percent; migrator e5060c3a… defaults status, existing restricted identity, one task, retries0,660s |
| Recovery metadata | Existing instance RUNNABLE PostgreSQL17/db-g1-small; backup/PITR enabled,7-day logs. Backup1793944800000 SUCCESSFUL ended2026-11-06T06:48:01.670Z. No restore or data-equality claim |
| Prior schema verification | Started predecessor status-only execution jayburd-dev-migrate-mcr7q with unchanged image/arguments; SUCCESS at2026-11-07T05:10:46.292855Z,1success/no failed/running/retried task; old embedded status proves clean41/seals |
| Activation review |42 does not implicitly provision an owner or adopt history. Existing migration job accepts only verify/status/up. No managed owner-provision/adoption execution profile exists; separate bounded activation work is tracked in453. No grant or old Item transition performed |

| Local operation gates | Core check/build and34 release/smoke/migration-receipt cases PASS; full make test attempted, known host socket bind EPERM only. Tracked Go1.26.10 unchanged; local external1.26.5 modfile/cache used |
| Staging decision | Selected official SDK args/no-traffic/update-traffic sections reviewed; stage ready API/web revisions without traffic, compare named projections, then promote exact revisions. Only API addition is collection-read-enabled=true after clean44; all old acquisition/consent args retained |
| Scan repair | Issue454 / PR456 (ff3f7d80ea57efe643a3b2e9ad93df581ea2b892): minimal shared x/net0.62.0 and existing APK minimum bounds, independently reviewed by root/private_registry with no source findings. Canonical checks and new trusted build remain required; no scanner waiver |
| Repair merge/build | All five canonical checks passed on ff3f7d80. PR456 merged2026-11-07T05:44:07Z as a4ad6c1dd2a6b257981c4d19513ad0aebbeb0daf. New trusted main run42343986400/attempt1 dispatched05:44:19Z; independent numeric owner/repository/main/manual/source checks pass. Runtime/scans/publication/receipts pending |
| Anonymous predecessor smoke | Numeric Sandbox web root/auth-config200, web workspace401, private API workspace403. New collection-recovery route404 on predecessor as expected. curl IPv4 discards bodies; Node host network unavailable. No identity or financial-data acceptance inferred |

## Current evidence

Old images remain the serving predecessor. The failed scan did not produce any
published candidate receipts, and its images cannot be deployed. The selected prior status job completed read-only verification; no managed forward
migration has run in this stage. GitHub436/65 hold exact current execution state.

Named operational projections exclude all environments/secret values. API arguments
are documented public resource/version references; existing consent expiry remains
2026-11-02T04:00:00Z. Private endpoint activation may add only the explicit
collection-read-enabled flag after verified44 readiness; this never grants an
owner or copies history. Existing book-based reads remain separate.

Immutable predecessor pins (retain for compatibility review; never roll SQL down):

| Component | Revision/image |
| --- | --- |
| API | jayburd-dev-api-00022-2sq; us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:e15d8c01bd7ff8a891adf01a183cccd163afa83dde3e42193605ddd4ff9e90e6 |
| Web | jayburd-dev-web-00022-mzn; us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:b525fe62714d62fe94d8f7083bc44fa0d747e78ddc146008ae40247e4e80dd8a |
| Migrator | us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/migrate@sha256:e5060c3a62fcf2c6b7f04d838560ec226a48cb7a4470ce1b6211650a88cde691; default action=status, instance=jayburd-bookkeeping-dev:us-central1:jayburd-dev |

SDK staging must preserve service accounts, timeout, concurrency, resources, ports,
startup probes and all existing API arguments. It must preserve job one-task,
parallelism1, retry0, timeout660s, restricted migrator identity and default status
arguments. Per-execution up is separate. No environment or identity setting is
read or rewritten. Traffic remains on each exact predecessor until ready-image
comparison passes. New flag and immutable pins must be reconciled into the next
reviewed operator Terraform inputs; applying the current stale configuration is
not a safe substitute for this bounded SDK procedure.

## Remaining work

Require successful trusted build, exact four candidate receipt/runtime/scan
verification and fresh predecessor status before updating the migrator image.
Run explicit forward42–44 once under existing workload identity; then independent
status proves clean44 and exact embedded artifact checksums. No down/force/reset.
Update only API/web immutable images and the reviewed private flag if appropriate,
preserving identities, existing args/settings and100percent ready traffic.
Compare named non-secret projections; retain predecessor and new pins for next
operator Terraform inputs. Do not replay stale Terraform or renew acquisition.
Anonymous root/auth-config and denied private endpoints are the metadata-only
smoke; owner performs authenticated financial UI review separately.

Existing private source visibility additionally needs an explicit active verified
owner grant and audited complete-Item adoption. The latter pauses legacy Item
acquisition and changes generation while retaining cursor/AAD/identities/history.
No actor is inferred from email/book membership, and no SET ROLE or generic SQL
escape substitutes for the trusted API actor boundary. Track a reviewed bounded
operator mechanism separately; do not claim an empty/unavailable workspace is
adopted history or that deploying images itself grants private capabilities.

## PR handoff

Refs436/452/65; operational evidence only, no new product migration/dependency.
Official Google Cloud SDK run jobs update/execute and services update references
reviewed2026-11-07 for immutable image, per-execution argument override and traffic
behavior; relevant command sections, not entire SDK manual. Sources:
https://docs.cloud.google.com/sdk/gcloud/reference/run/jobs/update,
https://docs.cloud.google.com/sdk/gcloud/reference/run/jobs/execute,
https://docs.cloud.google.com/sdk/gcloud/reference/run/services/update.
Existing migration-status/release/rollout contracts and prior436 operator
image-only deviation apply. Internal operation metadata adds no ISO20022 exchange,
accounting recognition, FX or ISO conformance claim. Actual runtime/SQL/owner
activation acceptance is never inferred from source tests or image publication.
