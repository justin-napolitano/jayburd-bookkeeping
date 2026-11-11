<!-- stage-runbook: true -->
# Customer banking web-only release — 479/436

## Status and boundary

Source PR480 exact49ceac6 passed all five checks and merged natively as
9b2a6568ffb05ed4e66c77857caa0ad07cc5509b at2026-11-09T04:24:19Z.
Trusted main Build images42561705439 attempt1 completed SUCCESS. Complete
independent four-image acceptance passed with zero HIGH/CRITICAL findings;
see [build evidence](customer-ui-build-evidence.md). Web00027-6jn is active,
healthy, exact accepted image/latestReady/sole100; nine transport probes passed.
API00026-5nt and all its selected settings/image/traffic remain unchanged.

Root holds the sole PRODUCT-479 release lane in GitHub65 through09:00Z.
Scope is trusted existing image publication and existing Sandbox WEB-ONLY
immutable image rollout. API00026-5nt/97473043 artifact enforcement stays fixed.
No API/worker/migrator deployment, SQL/operator action or migration, financial
posting, private record access, owner grant/adoption, provider/consent renewal,
IAM/new-resource or Production change. SQL1–47 immutable;48 unallocated.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source acceptance | Seed42559995647 frontend/application/backend SUCCESS; Infrastructure42559995377 foundation SUCCESS; GitGuardian SUCCESS, all exact49ceac6. Backend39 successful steps, zero failures; frontend673units/98browser journeys; actual source0.10/report/replay/CSV/drill, fictional Production and all4demo cases passed. |
| Reviewed UI | Technical inspectors/raw-ID lookup, idle history/command/JSON diagnostics removed. Compact Accounts actions and contextual safe recovery; same authority, keys, payloads, revisions, exact money and artifact policy. Five-width Accounts and prepared/denied captures inspected. Source stages in customer-banking-ui.md. |
| Compatibility | Root git diff --exit-code against accepted artifact-policy source1d8cae6 is empty for cmd/internal/db/api, Go dependencies, API/migrator/worker Dockerfiles, web server and auth modules. Existing API/protocol/schema remain compatible; no backend image change required. |
| Predecessor | Selected read-only metadata near04:09Z confirms API00026-5nt and web00026-4mq Ready/latestReady/sole100 with expected immutable pins. No environment values or financial records read. |
| Trusted build | Run42561705439 attempt1 exact merged9b2a656: all3 reusable prerequisites and all4 image jobs SUCCESS. Independent complete receipt deep equality, tested/scanned IDs, amd64, OCI/BuildKit source/entrypoint and matching CycloneDX inventories pass; HIGH0/CRITICAL0 each. Root reran the verifier. Last image job completed04:51:30Z; SUCCESS observed with final run metadata updated04:51:31Z, not an inferred exact overall completion time. Only web is a rollout target. |
| Fresh preflight | Near04:50Z selected API/web specs, exact images/revisions and predecessor sole100 match initial04:09Z baselines; all three service readiness conditions TRUE. Fresh65 register has no competing writer. |
| Web staging | Web00027-6jn staged04:52:46Z with exact e4521574 image and selected settings identical except image; old00026 retained sole100. Direct inactive Retired/import status was treated only as staging evidence, not container startup health. |
| Web promotion | Direct revision Ready transitioned04:53:51.121679Z; service Ready04:53:52.714228Z. Subsequent readback confirms exact latestReady/sole100, service Ready/ConfigurationsReady/RoutesReady and direct Ready/Active/ContainerHealthy/ResourcesAvailable all TRUE. |
| API preservation | Post-promotion API selected spec deep-equals initial baseline, including all20 arguments, image97473043, latestCreated/latestReady00026-5nt and sole100. Service readiness conditions remain TRUE. No API update was issued. Existing validateRevisions accepts the exact old API/new web pair. |
| Transport | Nine anonymous IPv4 discard-body probes near04:55Z: web root/auth-config200; workspace/collection/recovery/onboarding/onboarding pending/v3 taxonomy401; direct API workspace403. Authorization/Cookie/serverless-auth headers explicitly empty; no user/service credentials or redirects supplied and no response body retained. All expected statuses pass. |

## Current evidence

The governing source policy is merged. Actual source-backed posting/recovery and
all original exact-cent/report/export/drill assertions passed. Generic manual
posting remains forbidden by the existing API. This release changes customer
presentation, not financial semantics, source-provider authority or account data.

## Remaining work

Technical source/build/web-runtime/transport acceptance is complete. Owner
refreshes the [existing Sandbox](https://jayburd-dev-web-9088794002.us-central1.run.app)
and reviews Accounts, transaction review and contextual recovery privately.
Customer acceptance remains in479/467/436. No live bookkeeping or document
ingestion approval is inferred, and476 remains document retention/matching/
corrections. Keep the exact image pins for future reviewed deployment inputs;
no Terraform state/plan/apply occurred.

## Accepted runtime pins and selected settings

- Web00027-6jn: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:e4521574edecc6a5901c7f590341112484370e6cf93d63a3f13e30c79f3ba16d.
- Unchanged API00026-5nt: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:9747304373c1c9923518b0675cb85a2fcdde1ed1f397244df59208683fc790e0.

Web retains jayburd-dev-frontend identity, concurrency32, timeout20s, CPU1/512Mi,
port8080, startup /api/auth-config with failure12/period5/timeout3 and no command/
args. Selected spec differs only in image. API retains jayburd-dev-api identity,
concurrency16, timeout15s, CPU1/512Mi/port8080, /readyz and all20 arguments,
including existing origin/database/credential references, collection-read flag,
connections-enabled, false Link qualification and expired2026-11-02T04Z consent.
No environment values retrieved or consent renewed. API/migrator/worker candidate
images from this new build were verified but not deployed. Schema47 unchanged;
no SQL/status/operator job or financial action was executed for this UI release.

## Predecessor pins and rollback boundary

- API00026-5nt: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/api@sha256:9747304373c1c9923518b0675cb85a2fcdde1ed1f397244df59208683fc790e0.
- Web00026-4mq: us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/web@sha256:31873a8552222fabf12963c704e943f1559d9f1f6663a3f67b0778cf52334a34.

A separately reviewed web rollback may use the accepted prior web image while
retaining API00026 artifact enforcement and schema47. It reintroduces older
presentation and is not owner acceptance of that design. Prefer forward repair.
Never roll the API back to a pre-policy image, clear saved work with ad hoc SQL,
or downgrade/force/reset the schema to resolve a frontend issue.

## PR handoff

Source, trusted image, managed web and transport acceptance are complete. Root
owns all cloud writes and commits; independent build evidence is recorded
separately. Customer review and document retention/matching/corrections476 remain
open. This operational PR changes documentation only; no API, schema, private
financial record or provider state was changed by the web release.

Independent final review found no findings: web before→stage→active selected
spec recomputation, exact image/revision/traffic/health, entire unchanged selected
API metadata and documented backend/auth/schema compatibility diff all match.
Build pins, precise timestamps, rollback boundaries and scope limits agree.
Nine HTTP probes are root-reported numeric observations, not repeated by the
reviewer. Local make check passes with installedGo1.26.5/external exact-dependency
modfile; no tracked toolchain/dependency change. Canonical source/release checks
used Go1.26.10. Final documentation-only staged/diff checks pass before commit.
