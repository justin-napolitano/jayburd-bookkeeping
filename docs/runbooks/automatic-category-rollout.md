<!-- stage-runbook: true -->
# Synthetic automatic categorization rollout —485/488

## Status and boundary

Owner explicitly approved merging489, deploying and enabling automatic analytic
categories for synthetic workspace verification on2026-11-12. Source489 merged
normally at eb1828caab584064aa076edefccbb2bb413b84a7. Exact source8a78d04 passed
all frontend, restrictedSQL, composed application, backend, infrastructure and
secret-scanning checks. Trusted main Build43158591303 attempt1 is in progress.
No image or managed rollout is accepted merely because the build was dispatched.

Root holds the sole finite485 existing Sandbox release lane in GitHub65 through
2026-11-12T09:00Z. Dedicated branch chore/485-automatic-category-rollout. Scope:
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

Implement and prove the bounded tooling; qualify the build; inspect current
selected metadata; execute reviewed phases; verify health/transport; then prove
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
