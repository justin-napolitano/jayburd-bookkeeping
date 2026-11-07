<!-- stage-runbook: true -->
# Private workspace Sandbox rollout —436

## Status and boundary

PRIVATE-ROLLOUT436-20261107T0505Z; ops/436-private-rollout from merged452/f8d9e06.
Owner explicitly requested controlled deployment after the verified source build.
Runtime/schema deployment is complete; authenticated owner UI review and explicit
private owner/history activation remain separate. No financial UI was inspected.
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
| Accepted candidates | Run42343986400 SUCCESS; all three prerequisites and four exact runtime/scan/publication jobs pass. Independent recomputation matches all four receipts and OCI source labels; every image0 HIGH/0 CRITICAL, nonempty SBOM. Exact pins in build-evidence436.md. Worker built only |
| Fresh predecessor | Status-only execution jayburd-dev-migrate-r876l SUCCESS at2026-11-07T05:59:15.124815Z, clean41. API/web selected projections exactly equal recorded predecessor. Migrator image-only update to a2bdc64d… verified; default status,1task/parallelism1/retry0/660s/restricted SA unchanged |
| Forward invocation | Single explicit per-execution up started as jayburd-dev-migrate-pfxzw. No down/force/reset or automatic task retry; independent terminal metadata and subsequent status required before API/web staging |
| Forward completion | pfxzw SUCCESS at2026-11-07T06:05:56.332290Z, one success/no failed/running/retried/cancelled task. Existing migration-receipt.mjs accepts exact up/image/instance/SA/task/retry/timeout metadata using a bounded SDK evidence input, not a fabricated full Terraform release manifest. Independent status still required |
| Independent status | spn2w SUCCESS at2026-11-07T06:07:35.644052Z; exact default status/image/settings accepted by existing receipt validator. Clean44 and all embedded hashes verified; no owner grant or adoption |
| No-traffic stage | API00023-6sp/web00023-6wv exact accepted immutable digests, full selected settings equal baseline except API new flag/image and web image. Old revisions100percent. Direct new revision Ready=True/Retired and ContainerReady=True prove staging/import, not active startup. Official RETIRED/explicit traffic sections reviewed; promote exact API then require active readiness/100percent before web |
| Active rollout | Exact API00023-6sp promoted first, active Ready/ConfigurationsReady/RoutesReady TRUE at06:10:05Z; then web00023-6wv active TRUE at06:10:38Z. Each latestReady equals exact new revision and sole100percent target. All selected specs equal staged expected settings. Existing deployment-smoke validateRevisions accepts both immutable images/revisions/traffic |
| Transport acceptance | Numeric web root/auth-config200; web workspace/collection/collection-recovery/v2 synthetic-UUID taxonomy401; private API workspace403. IPv4 curl discards bodies and follows no redirects, no application credentials supplied. Node/direct curl-q variant cannot resolve here; no auth/data acceptance inferred |
| Anonymous predecessor smoke | Numeric Sandbox web root/auth-config200, web workspace401, private API workspace403. New collection-recovery route404 on predecessor as expected. curl IPv4 discards bodies; Node host network unavailable. No identity or financial-data acceptance inferred |

## Current evidence

Corrected trusted main run42343986400 supplies four independently accepted
candidates. Restricted up42–44 and independent clean44/full-hash status both
succeeded. API00023-6sp/web00023-6wv now serve100percent, Ready with exact digests
and settings. Canonical anonymous transport checks and new private-route denials
pass. No owner grant, history adoption, authenticated financial review or provider
acceptance is claimed. GitHub436/65 hold execution state;453 owns activation.

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

Owner reviews the current existing Sandbox UI at
https://jayburd-dev-web-9088794002.us-central1.run.app.
Issue436 remains open for that review; deployment/transport acceptance is complete.
Reconcile the accepted exact image pins and API collection-read-enabled=true into
the next reviewed operator Terraform inputs. Do not replay stale Terraform or
renew acquisition. No SQL down/force/reset is needed or authorized.
Broader filters/detail links242 and rule/provider runtime413/414 remain separate.

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
