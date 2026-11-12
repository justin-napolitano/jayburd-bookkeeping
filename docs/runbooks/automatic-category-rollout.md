<!-- stage-runbook: true -->
# Synthetic automatic categorization rollout —485/488

## Status and boundary

Owner explicitly approved merging489, deploying and enabling automatic analytic
categories for synthetic workspace verification on2026-11-12. Source489 merged
normally at eb1828caab584064aa076edefccbb2bb413b84a7. Exact source8a78d04 passed
all frontend, restrictedSQL, composed application, backend, infrastructure and
secret-scanning checks. Trusted main Build43158591303 attempt1 and all four independent candidate
checks passed. Release tooling490 merged at0568a736365db2a3ef0aa879280ec0c6c2384d5a
after exact239b066 Seed43162349952 and Infrastructure43162349982 passed. First
read-only inspection43165186172 authenticated successfully but failed closed
before an accepted configuration snapshot. No migration/runtime/activation was
attempted. The next source stage adds safe structural diagnostics; the initial
baseline-null manifest still permits inspection only.

Root holds the sole finite485 existing Sandbox release lane in GitHub65 through
2026-11-12T09:00Z. Initial branch chore/485-automatic-category-rollout is merged; diagnostic follow-up491 is merged; branch
chore/485-approved-runtime-baseline is merged; branch
chore/485-synthetic-release-baseline is merged; branch
chore/485-observed-runtime-compatibility is merged; branch
chore/485-pinned-observed-baseline owns the exact observed baseline stage. Scope:
all four same-build patched artifacts, existing migrator forward48/status,
configuration-preserving API/web stages and explicit synthetic analytic activation.
No ad hoc SQL, migration edits, down/force/reset, provider calls or consent renewal,
real-data use, accounting autopost, IAM/new resources, worker/PFC activation or
Production mutation. SQL1–48 remain immutable. Existing workspace policies remain
off until the authorized in-app setting enables the selected synthetic workspace.

## Stage ledger

| Stage | Evidence / disposition |
| --- | --- |
| Source | PR489 merged after exact-head Seed43154917364, Infrastructure43154917211 and GitGuardian success. Root observed native merge and clean worktree. |
| Build | Existing trusted main-only workflow43158591303 dispatched as owner on exacteb1828c; complete receipts/scans/runtime qualification pending. |
| Backup | Selected CloudSQL backup metadata:1794376800000 SUCCESSFUL, completed2026-11-11T08:41:55.672Z. This is backup metadata, not a restore rehearsal. |
| Local operator | gcloud auth inventory identifies existing owner. Run metadata requests fail because the session cannot write its credential-cache path. No credentials copied/exported or filesystem permissions changed. |
| Tooling | Legacy Terraform release regenerates arguments/settings and cannot represent automatic categories; a finite path in the existing owner-deploy workflow is being prepared and tested before managed effects. No federation/IAM change. |
| Migration/runtime | Not started. Retain fresh selected baseline and phase receipts before claiming success. |
| Product acceptance | Pending normal verified-owner synthetic flow. Anonymous transport is separate. |

## Delivery design

Use the existing Deploy development workflow and existing owner-deploy identity,
concurrency group and manual main-only owner gate. A new closed synthetic profile
must validate the exact manifest hash, fixed project, source/build/four images,
expiry, evidence and phase before cloud authentication. Existing legacy profiles
remain unchanged. Never refresh an expired historical release or provider consent.

Run distinct inspect, migrate, stage, activate, web and explicit pause phases. Each has selected
readback and a finite receipt; an unknown mutation outcome requires observation,
not blind redispatch. Avoid full Terraform runtime apply because it would reset
current arguments and include IAM work outside this release. No environment
values, tokens, provider payloads, private financial records, state or plans are
read or uploaded. Only approved resource identifiers/settings/hashes and outcomes
belong in evidence.

Require clean47 baseline, explicit forward48 through the existing migrator,
then clean48 candidate status. Keep the job's ordinary action status. Publish no
manual SQL path. Preserve all existing API arguments, identity, origin, connector,
credential-version references, probes, resources and connection settings, including
expired consent and false qualification. Stage/promote the patched API with
categorization off first, providing a schema48-compatible patched rollback anchor.
Only then may the explicitly authorized activation revision enable analytic work.
No-traffic API staging alone is not effect-free when its background flag is on.

For synthetic activation, API instance-based CPU allocation with minimum0 and
maximum1 permits the existing background runner while an instance exists. A
normal application request wakes a zero-instance service; durable work resumes.
This is not continuous processing while the app is closed or an availability SLA.
Web retains its current request-based CPU setting. Keep source/provider workers
and their schedules unchanged. The runner cancels its parent on shutdown and SQL
recovers the same durable work after unknown outcomes; CloudRun's termination
window does not establish a15second application shutdown guarantee.

Primary sources reviewed for this operational choice:
[CloudRun billing](https://docs.cloud.google.com/run/docs/configuring/billing-settings),
[autoscaling](https://docs.cloud.google.com/run/docs/about-instance-autoscaling),
[runtime contract](https://docs.cloud.google.com/run/docs/container-contract),
[pricing](https://cloud.google.com/run/pricing). At published us-central1 rates,
1CPU/512Mi is approximately$0.0684 per live instance-hour before free tier and
other charges. Continuous use can consume about$49.93/month for API alone; minimum0
is not a project spending cap. No minimum1/always-warm commitment is introduced.

## Acceptance and rollback

Require all four candidate receipts, tested/scanned image identity, complete
SBOM/provenance and fresh high/critical gates from one source/run/attempt. Stage
exact images, preserve old traffic, observe actual revision readiness before
promotion, then require exact latestReady and sole100 traffic plus service/direct
revision health. Deep-compare every selected setting and unchanged components.

Anonymous smoke discards bodies and checks web/auth-config200, private bridge
routes including automation401, and direct privateAPI403. Normal verified-owner
acceptance must separately observe automatic persisted categories, protected owner
choice, explainable exact exceptions and repeat-import/recovery behavior on only
synthetic sources. Compare unchanged existing accounting anchors/counts; zero new
journals/outbox is required, not an assumption that historical books are empty.

If activation fails, prefer holding traffic or the same patched schema48-compatible
API with automation off and request-based CPU allocation, minimum0/maximum1.
The explicit pause phase creates a new patched revision without deleting retained
history. Existing instances drain under platform lifecycle; do not claim an
instantaneous global stop. The owner can pause the selected workspace policy
through the app to fence SQL work immediately. Never roll back SQL or use
old vulnerable/pre-artifact images as an automatic fallback. A rollback still
needs exact selected configuration and durable-state compatibility review.

## Remaining work

Pass exact-head checks for the diagnostic follow-up, inspect fresh selected
metadata and pin its approved public baseline before mutation phases. Then
execute reviewed phases, verify health/transport and prove
normal owner synthetic use. Record exact images, operations, revisions and results
in GitHub485/488 and this delivery evidence. Hosted real-data/provider qualification
and live bookkeeping policy remain separate.

## PR handoff

Pending implementation and actual execution receipts. Every implementation commit
includes this marked runbook and the staged guard. No managed deployment/migration
or analytic activation has occurred at this stage.

## Current evidence

Tooling source stage: six closed phases implemented in the existing deployment
workflow with unchanged legacy default/trust/concurrency.45 credential-free tests
and independent source review pass; current script SHA256
`eb83808952e6706e5d59dbc4f9fbaea33dfa85155581b4f734284021027bb346`.
Tests cover four-image/source/architecture/scanner/SBOM bindings, metadata drift,
observed tag-free traffic and health, forward/status ordering, explicit pause and
unknown outcomes without mutation retries. Existing34 release tests and actual
fake-CLI legacy orchestration pass. Local Go check reaches public dependency-cache
write denial (x/net cache lock); no tracked dependency/toolchain was changed.
Canonical source checks and workflow lint remain required before merge. Local
infrastructure mocks are running; no managed Terraform/state/plan/apply occurred.

The complete trusted build43158591303 attempt1 is SUCCESS on exacteb1828c:
all three reusable gates and four runtime/scan/publication jobs passed. Independent
receipt regeneration, Dockerfile/VCS/OCI/amd64 bindings and nonempty Trivy0.75.0
CycloneDX inventories passed for every component, with HIGH0/CRITICAL0. Root
reran the independent verifier and the new offline CLI against all four actual
artifact sets. Permanent selected summary, file hashes and complete candidate
receipts: [build evidence](../reviews/automatic-category-build.json).

The initial exact inspection manifest has baseline=null, so it cannot execute any
mutation phase. SHA256871c1f5ba80b3ef48f8b1015881758e09b01d2d66670b681ae824449d2e08f96;
expires2026-11-12T09:00Z. Current tooling privacy fixes reject unapproved argument
values/commands/references before writing inspection receipts, while retaining
known unhealthy/status-or-up observations for recovery.57 guards pass; all local
infrastructure mocks also pass. Review fresh selected baseline bytes under the owner’s existing rollout
authorization before a later non-inspection manifest is committed/hashed. No managed
workflow or migration/traffic/activation has been dispatched yet.

Independent final privacy review passes at script SHA256
`c80a508e83bdadadf4d282acb0e2ab8c8428e38fe7128f5cfdd2558e0ce106a9`;57/57 tests.

A pre-dispatch contract cross-check corrected the fixture/validator webhook URL to
retain the existing `/webhooks/plaid` receiver path (existing provider constant and
Terraform contract). It does not renew consent or enable the provider. A separate
regression binds the retained endpoint to that existing contract and rejects the
pathless URL;58 tooling tests pass before final exact-head CI.

Follow-up inspection stage: first hosted run43165186172 passed all owner/source/
artifact/hash/expiry checks and existing WIF authentication, then its read-only
execution failed with a generic configuration/CLI rejection. Its artifact is
explicitly offline-validation-only and contains no accepted before snapshot.
No mutation path was available. Do not replay a mutation or infer the failure cause.

The follow-up emits separate fixed-stage diagnostics with only known structural
presence/type/counts and bounded child-exit/timeout codes. Failed metadata never
becomes a raw receipt. Selected desired LATEST traffic is normalized only with
healthy exact observed100/latestReady agreement; ambiguous/named/tagged/mismatched
cases still fail. Only known zero/HTTP/empty-header probe defaults are normalized.
CPU1000m maps exactly to1 for the same one-CPU service/job profile;999m/2 and all
other resource differences remain rejected. These are supported representations,
not claims about the unobserved hosted mismatch.82 credential-free tests pass.

Historical47 preflight evidence now keeps build=null and records targetBuild
separately; it must not attribute the old199ed image to the patched candidate
build. Exact old-image/action/execution checks are unchanged. Candidate up/status
receipts retain the actual new build. This fix is required before any migrate
manifest/dispatch, and was not reachable through the first inspection manifest.

Independent installed-SDK formatter checks on synthetic objects confirm the
quoted allowlisted annotation keys, repeated containers and observed traffic
projection while excluding unselected last-applied/headers/messages. No cloud,
user credentials or financial records were used in these formatter checks.

Independent final follow-up review passes at script SHA256
`6fc6558d0ad17ecec3ab6d12774f17a1509548326658e0fb647f580a97c7f2b6`;
82/82 tests pass. Offline validation of the actual four qualified artifact sets
also passes. The committed manifest remains unchanged, baseline=null; no mutation
phase is dispatchable. Fresh canonical checks and another safe inspection are
required; no actual mismatch has yet been inferred from the generic failure.

Read-only inspection43169572531 on491 mergea552863 authenticated and obtained
the API's selected metadata. Safe diagnostics show the expected object structure,
20 arguments, HTTP1/8080, custom probe fields and no secret headers, but exact
public-api compatibility still fails. No raw snapshot or runtime acceptance was
emitted. This confirms that privacy checking must precede and remain separate
from comparison against historical deployment assumptions. No schema, service,
traffic, policy, provider or IAM mutation was issued.

The next source stage introduces a closed inspection-only public minimizer. It
checks every argument/reference/command/annotation and public data type before
export, while bounded operational numeric settings and standard Go flag spellings
can be observed as requires-owner-review. Unapproved strings, credentials, unknown
fields, commands and headers remain rejected. Exact manifest, migration, service,
job, execution and mutation checks are mechanically unchanged; a drifted read-only
snapshot cannot authorize deployment.118 automatic tests and independent privacy
review pass at script SHA256
`17f1211c5bedd7322fcc0b0e1490d4d1d1c48868edffa4842ad50ac13525d2e7`.
The manifest remains baseline=null/inspection-only. No actual value mismatch is
asserted until the next validated public snapshot is obtained and reviewed.

PR492 merged at698eee1364b566a0ab3599c84c6dd7a1c1b6277c after all five
exact fb893dc checks passed (Seed43171287938, Infrastructure43171288002 and
GitGuardian). Read-only inspect43174384070 authenticated, read selected API
metadata, and again stopped at public-api before any snapshot export. No
managed job/schema/service/traffic/policy/provider/IAM mutation occurred.
The offending constraint remains unknown; do not equate structural presence
with exact compatibility or infer a current argument/name/resource mismatch.

The next diagnostic-only stage covers every known public constraint with fixed
labels and booleans, including normalization and per-known-flag checks. It
collects API, web, migrator and recent-execution diagnostics before aggregate
inspection failure, without publishing invalid values or a partial baseline.
No failure diagnostic authorizes a mutation or substitutes a guessed value.
Mutation guards, candidate images and SQL1–48 remain unchanged. Actual
inspection, manifest approval, rollout and hosted synthetic acceptance stay open.

The earlier09:00Z root claim and inspection manifest expired during the ongoing
release work. Fresh register/open-PR audit at2026-11-12T15:05Z found no competing
release writer. Root renewed the same bounded scope through18:00Z on65
comment6827330358 and485 comment6827331551 under the continuing owner approval.
The new inspection-only manifest references that renewal and expires18:00Z;
baseline stays null. Every dispatch still rechecks the independent24h image-scan
freshness; renewal never refreshes scan evidence or provider consent.

Frozen diagnostic sourceSHA256
`d17364164ad0c8194ec52950a4bf45945080d8d7008d361dad1c6ffdc4057f96`
and testSHA256
`895da894581f336aed830e67bf428e20e75d2b9e362c3ffd3962e71ea97bb60d`
pass148 automatic checks and182 combined release/deployment checks. Context
links/stage/diff checks pass. Actual complete four-image offline validation
passes at15:07Z using the renewed inspection-only manifestSHA256
`97dd688b37e665b09b1e6d81fd1e52d1e21d1b0ad4befefdb6e6bfe64a665b1d`.
Missing template names and each latest-target resolution conjunct are tested
independently; no current metadata value or healthy baseline is inferred.
Canonical exact-head CI and fresh managed inspection remain required.

PR493 merged at28c7d83f92e8d5c05a5e548dcb1b3284c3b0840c after all five
exact0559603 checks passed (Seed43245859539, Infrastructure43245858761 and
GitGuardian). Read-only inspect43250769965 now identifies the actual sole
public-profile mismatch: API and web both omit template.metadata.name.
Diagnostics report templateNamePresent=false/type=undefined; every other
API/web public predicate and every migrator/recent-execution predicate passes.
The inspection exports no partial baseline and performs no managed mutation.

The next compatibility stage preserves this observed optional-field absence as
explicit null instead of inventing a template name from revision status. Require
the metadata object and reject raw null/empty/nonstring/foreign names. Initial
baseline identity still requires the exact historical healthy latestCreated/Ready
and sole100 desired/observed traffic; staging may advance latest revision while
retaining old traffic. New candidate identity remains bound by fixed suffix,
status, image/configuration and direct revision health. The existing finite
gcloud updates never send null or a replacement receipt to CloudRun.
All20 API arguments, provider consent/qualification, SQL1–48 and qualified images
remain unchanged. Root's18:00Z claim persists; actual rollout is still pending.

[G32](../research/gcp-sources.md#g32--omitted-revision-template-name-2026-11-12)
records the selected official APIv1 RevisionTemplate metadata semantics, actual
read-only observation, alternatives and independent tests. Vendor-generated
revision names do not imply an observed template-name property.

Frozen omitted-name sourceSHA256
`3e9752f7e3cf6a722998fcb79917f892b5f8a09429d617ea5956f6a840a1bd38`
and testSHA256
`e1be95aa134828bf9db0021ffebe2d2426978e204f60340f7450cb1a5db07208`
pass169 automatic and203 combined release/deployment checks plus context links.
Full simulated inspect/JSON-baseline/migrate/stage/activate/web/pause retains
actual omission and20 arguments; malformed names/status/traffic/direct-health
failures remain blocked. No API/web image, runtime or managed schema has changed.
Exact-head canonical CI, normal merge and fresh read-only inspection remain gates.

PR494 merged normally via the standard merge API at
`f42dbca5918b5f21eececeb6d847548118420049` after all five b03fe05
checks passed (Seed43252848147, Infrastructure43252848195, GitGuardian).
The initial GraphQL merge gateway error left the PR open/main unchanged;
subsequent standard exact-head merge succeeded and was independently read back.
Read-only inspection43258761537 succeeded on that main commit, with complete
API/web/migrator baseline and all three healthy-selected-profile results.
Actual API00026-5nt and web00027-6jn retain sole100 traffic and Ready/
ConfigurationsReady/RoutesReady. Template-name omissions remain explicit null;
all20 API arguments and expired Nov2 consent/false qualification remain intact.
Migrator remains historical199ed337/defaultstatus. This is public metadata
acceptance, not schema47/48, candidate runtime or hosted product acceptance.

The next manifest stage pins the complete validated before object byte-for-byte
by value; selected source/hash/evidence is retained in
[inspection receipt](../reviews/automatic-category-inspection.json).
Pinned manifestSHA256: `9258dbf5a14d6c2d426d9325e36e1d8e2b31cb77c9cb5b804cdd588d0687d939`.
No runtime settings are reconstructed from assumptions, and no managed mutation
has occurred. Each phase still requires exact current baseline/phase state,
expiry, four-image freshness, owner/main gates and selected post-readback.
Workspace policy remains off pending complete synthetic-workspace scope
confirmation: book filters/account-scoped rules do not scope the runner.
