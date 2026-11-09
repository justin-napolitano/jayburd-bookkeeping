<!-- stage-runbook: true -->
# Artifact-backed accounting Sandbox rollout — 473/436

## Status and boundary

Owner requires supporting artifacts for every new application accounting entry.
PR472 contains the first enforcement slice and the preceding responsive UI work.
Exact source b6cc38e2 passed all five checks and merged natively as
1d8cae6f25eabd87159ba3e88bd962e2b04732bf at2026-11-09T00:43:19Z.
Trusted main build42541738958 attempt1 completed SUCCESS. All four image
receipts/runtime/scans/source/SBOM/provenance were independently accepted,
with zero HIGH/CRITICAL findings; see [build evidence](artifact-build-evidence.md).
Existing Sandbox API/web image-only rollout and anonymous transport acceptance
are complete. Owner visual/private-use review remains pending.
Root held the ARTIFACT-473 sole existing Sandbox release lane in GitHub65.
No migration up, migrator image change,
financial posting, private record access, provider call, connection adoption,
consent renewal, worker rollout/execution/scheduling, IAM/new resource or
Production mutation. SQL1–47 remain immutable;48 is unallocated.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Policy/source | Governing AGENTS.md and docs/contracts/artifact-backed-accounting.md. Public app rejects direct manual journals, unbound reversals, manual post registration and prepared legacy manual execution. Safe cancel/ack/history retained. Blank journal UI removed; Posted entries is a read view. |
| Canonical acceptance | Exact b6cc38e2: Seed42539662402 frontend/backend/application SUCCESS, Infrastructure42539662581 foundation SUCCESS and GitGuardian SUCCESS. Backend39 successful steps; actual application and all4 demo journeys pass. Independent API/source/proof and final test-fix reviews have no outstanding findings. |
| Canonical fixes | cd2bc01 exposed integration-tag fixture constant collision and a browser expectation of a diagnostic deliberately sanitized by Next. b301d3b fixes both test defects without changing bridge/error exposure or financial assertions. Fresh exact-head acceptance required. |
| Setup receipt proof | b6cc38e additionally replaces the stale manual-post receipt expectation with exact setup_receipt1/operation_receipt0, retaining journal0/outbox0 and one acknowledged setup/execute. Actual default browser and SQL guards pass. |
| Actual source posting | Exact b6cc38e source connection step passed00:27:17Z in job128911607048: actual source0.10 lost-execute/remount/ack/replay, exact source/journal backlink, all four reports/snapshot/comparison, exact CSV and account/journal drill. SQL requires journal1/review_posting1/outbox1/two observations. Fictional Production and all4 demo journeys also pass; all source gates are green. |
| Predecessor | Selected metadata read near00:04Z: API00025-2j2 and web00025-5vt Ready and sole100, exact predecessor images below. No environment values read. |
| Recovery metadata | Fresh instance RUNNABLE PostgreSQL17/db-g1-small, backups/PITR enabled, seven-day logs. Selected official automated-backup audit metadata reports STATUS_SUCCEEDED, started2026-11-08T06:27:51.324Z and completed06:29:22.678Z, matching owner-supplied backup1794117600000. This is successful backup metadata, not a restore rehearsal. |
| Clean schema | Existing accepted schema47 migrator default status execution jayburd-dev-migrate-s5fwv completed2026-11-09T00:14:37.066394Z with one success and no failures/retries. Exact action/image/restricted identity/task1/timeout660 accepted by migration-receipt.mjs. Status validates clean47/full embedded hashes. No up or migrator image update. |
| Trusted build | Run42541738958 attempt1 exact merged1d8cae6f completed SUCCESS; final run metadata updated01:02:03Z. All three reusable prerequisites and all four runtime/scan/inventory/publication jobs passed. Independent complete receipt recomputation, source/architecture/provenance and SBOM binding pass for all four; HIGH0/CRITICAL0 each. Root reran the independent verifier successfully. |
| Fresh preflight | Near01:00Z both predecessor revisions still healthy/sole100, selected specs exactly match original baseline. No competing writer or configuration change observed. Existing builder pool is active and provider condition is already restricted to exact owner/repository/manual-main workflow; no IAM/auth configuration changed. |
| API stage/promotion | API00026-5nt staged at01:03:12Z, exact accepted digest/settings with old00025 sole100. Inactive Retired/import evidence was not counted as startup health. Direct revision Ready transitioned01:03:54.794754Z; service Ready01:03:56.149479Z. Subsequent readback confirmed exact latestReady/sole100, service Ready/ConfigurationsReady/RoutesReady and direct revision Ready/Active/ContainerHealthy/ResourcesAvailable all TRUE. |
| Web stage/promotion | Only after API health acceptance: web00026-4mq staged01:04:47Z, exact accepted digest/settings with old00025 sole100. Direct revision Ready transitioned01:05:28.609170Z; service Ready01:05:30.114854Z. Subsequent readback confirmed exact latestReady/sole100 and all service/direct revision startup/active health conditions TRUE. Existing validateRevisions independently accepts both exact pins and traffic targets. |
| Transport | Nine IPv4 anonymous probes near01:06Z: web root/auth-config200; workspace/collection/recovery/onboarding/onboarding pending/v3 taxonomy401; direct API workspace403. No credentials, redirects or response-body retention. All expected statuses passed. |

## Current evidence

Current supported flow is imported bank/statement artifact → review/category →
immutable draft → explicit source_post → linked posted entry. Existing SQL owns
current authority, revision checks, exact money and atomic journal/source link/
receipt. Documents, matching multiple artifacts to one event and artifact-backed
corrections remain issue476; no fake Documents page is exposed.

## Remaining work

Technical source/build/runtime acceptance is complete. Owner reviews the new
flow privately in the [Sandbox](https://jayburd-dev-web-9088794002.us-central1.run.app).
The blank New journal form is gone; use imported transaction review, category,
reviewed draft and explicit posting confirmation. Posted entries is the read view.
Prepared legacy manual requests can be cancelled; historical receipts remain
readable and acknowledgeable. No authenticated financial response was inspected
for rollout verification, and the release itself created no journal or adoption.

Issue476 remains document retention, matching and evidence-linked corrections.
Owner review/live bookkeeping policy remain in473/467/436; deployment does not
complete them. Worker/migrator candidates are build evidence only. Retain exact
pins for future reviewed deployment inputs; no Terraform state/plan/apply occurred.

## Accepted runtime pins and preserved settings

- API00026-5nt: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:9747304373c1c9923518b0675cb85a2fcdde1ed1f397244df59208683fc790e0.
- Web00026-4mq: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:31873a8552222fabf12963c704e943f1559d9f1f6663a3f67b0778cf52334a34.

Selected service specs compare exactly with predecessor except image. API retains
jayburd-dev-api identity, concurrency16, timeout15s, CPU1/512Mi/port8080,
startup /readyz. Web retains jayburd-dev-frontend identity, concurrency32,
timeout20s, CPU1/512Mi/port8080, startup /api/auth-config. Both retain startup
failure12/period5/timeout3. All20 API arguments remain unchanged, including
numeric web origin, database/Infisical/credential-version references,
collection-read-enabled=true, connections-enabled, false Link qualification and
expired consent2026-11-02T04:00Z. No consent was renewed. Web command/args remain
absent. No environment values were retrieved. The existing migrator pin/default
status/restricted identity/task/parallelism/retry/timeout remain unchanged.

## Predecessor pins and rollback boundary

- API00025-2j2: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:971a9381b4e82c69a0398623a4fce85f679dab6e7a274f1fa9619b98fee2b74e.
- Web00025-5vt: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:def2c3f5f23a4a509f9af4e2ed5b662e2747316b2c2c17db24ae920c7de9ccfa.
- Existing migrator: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/migrate@sha256:199ed337de73e622c7398516a90ef196620b100701cb3f07924e6cd86df5f31a; embedded47/default status.

Pre-policy API images permit manual posting and are not an acceptable automatic
rollback across this policy boundary. Prefer reviewed forward repair preserving
artifact enforcement, retained command recovery and schema47. Never use down,
force/reset or ad hoc SQL to resolve saved work.

## PR handoff

Source, trusted build, selected settings, active runtime and anonymous transport
acceptance are complete. Source policy and future-agent instructions are merged;
this operational record documents an image-only release and no SQL mutation.
Owner application review, receipt/invoice ingestion and live bookkeeping policy
remain separate acceptance steps. No manual entry, journal or connection adoption
was executed by this release.

Independent final review recomputed before→stage→active selected specs, exact
pins/traffic/health and the complete read-only migration receipt with no functional
findings. Timestamp wording was clarified to distinguish final image completion,
run metadata update and direct revision/service Ready transitions. The reviewer
checked the nine root-reported probe records without repeating HTTP probes;
backup/audit observations are root-reported. Local make check passes using
installed Go1.26.5 with the existing external exact-dependency modfile; no tracked
toolchain or dependency changed. Source and trusted release used canonical
Go1.26.10 checks. This final PR is documentation only.
