<!-- stage-runbook: true -->
# Rule workspace web release —500

## Status and boundary

Owner says “do it” after accepting the compact YAML rule UI and normal merge499.
Scope: existing development WEB-ONLY immutable image rollout. No API, migrator,
worker, schema48, database/SQL, IAM, provider/consent, private record, automatic
category/rule or accounting mutation. Root owns serialized cloud operations;
GitHub500/65 own the finite claim and results. Branch ops/500-rule-workspace-release
from merged ddddfbd220c2a8ddd513c2a1259f96dfe2ae9751 retains evidence.

Existing APIas-43158591303-activate and webas-43158591303-web are healthy, sole100
at baseline. API image1c9c318e / web06486c8c / migrator7a70083f remain rollback and
preservation references from the accepted runtime receipt, not new candidates.
No old/expired automatic-synthetic manifest is reused: its source/build pins are
for a different release. No broad Terraform runtime apply is part of500.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source/owner |499 merged normally to ddddfbd at2026-11-14T00:56:02Z; exact e173cae all5checksPASS including actual application integration. Owner visual approval “looks good” recorded in498. |
| Claim/build |500 created; sole root500 claim65#6868941436 through05:00Z. Trusted main Build images43537686978 attempt1 dispatched at exact ddddfbd using existing owner-build trust. Full reusable Seed precedes image tests/scans/publication. |
| Compatibility | `git diff eb1828c HEAD -- cmd internal db api go.mod go.sum Dockerfile.api Dockerfile.migrate Dockerfile.worker web/src/server web/src/bookkeeping/api.ts` empty. This changes frontend authoring/presentation; current API/schema remain compatible. |
| Baseline | Selected public gcloud projections captured near01:02Z. Existing constraint audit: API145/145, web78/78, migrator30/30PASS. Both services Ready/ConfigurationsReady/RoutesReady true and sole observed/desired100 at intended current revisions. No environment/token/response body read. |
| Build qualification | Pending full new four-image evidence; api/migrate/worker are evidence only and never deployment targets. Independent reviewer prepared receipt/provenance/scan/SBOM verifier pinned to actual new source/run/attempt. |
| Runtime/transport | Pending successful candidate qualification. No cloud mutation issued yet. |
| Privacy | Owner separately requested complete repository privacy; GitHub PATCH/readback confirms private:true,visibility:private,has_pages:false. No collaborator or runtime access changes. Private artifacts/issues remain the task authority. |
| Transport tooling | Local run.app DNS is unavailable; prepared a separate manual, owner/main-only credential-free fixed-nine-endpoint check. No cloud authentication, URL inputs, user credentials, body retention or mutation.17mock tests cover exact statuses, redirects, and one15s deadline shared by fetch/body cancellation. Added to standard Node tests; existing release pins/gates remain unchanged. |
| Independent tooling review | Fixed endpoint/helper/workflow review has no material findings;17focused tests and247combined Node testsPASS. One-off runtime readback verifier was corrected to require exactly one True condition for all3service and4direct-revision health types, rejecting duplicate/contradictory states. Baseline audit and exact non-image comparison remain. |
| Local core gate | `make check test build` attempted; trackedGo1.26.11 download is blocked403 before execution. No Go/product source changes in this tooling; canonical exact-head CI remains required before merge. |

## Current evidence

Local API metadata reads with shell redirection hit credential-cache write
restrictions. A direct selected-field gcloud describe succeeds without copying
credentials, changing authentication or reading private values. Public projections
are retained as local evidence; they contain only reviewed metadata fields.

Before mutation, require completed successful trusted43537686978/attempt1 with
all4same-source runtime tests, receipt regeneration, tested/scanned image equality,
amd64/source-bound provenance, fresh zeroHIGH/CRITICAL scan and matching nonempty
CycloneDX inventory. Build metadata is unsigned; do not claim SLSA attestation.

## Bounded rollout

Use existing gcloud service update with ONLY image, explicit revision suffix,
no-traffic and fixed project/region. No command/args, environment, identity,
network, scaling, probe or security flag is supplied. Recheck current selected
metadata immediately before stage. Preserve the old web at100 during staging;
compare all selected non-image configuration and unchanged API/migrator. A
no-traffic revision may be import-only/Retired, which is not active container
health. Promote only the exact staged name using update-traffic; inspect unknown
outcomes instead of repeating a mutation. Require latestCreated=latestReady,
sole desired/observed100 and Ready/Active/ContainerHealthy/ResourcesAvailable
direct revision conditions after promotion. Preserve the previous web for a
separately inspected rollback; never downgrade/reset SQL or touch API to repair UI.

Selected primary sources reviewed2026-11-14:
[gcloud services update](https://docs.cloud.google.com/sdk/gcloud/reference/run/services/update),
image/revision-suffix/no-traffic flags; [Cloud Run traffic migration](https://docs.cloud.google.com/run/docs/rollouts-rollbacks-traffic-migration),
revision percentages, no-traffic revisions and non-instantaneous transitions.
These establish the CLI mechanisms; selected before/after equality and health
are our required acceptance, not a vendor guarantee of bookkeeping correctness.

After runtime readback, use bounded anonymous GET status checks with no credentials,
redirect following or retained bodies: web root/auth-config200, private workspace/
collection/recovery/onboarding401 and direct API readiness403. This proves
transport only. Normal owner-category coverage/manual-choice/conflict/persistence
remains488; owner acceptance of local UI is not hosted data-processing proof.

## Remaining work

Finish trusted build, qualify candidates, recheck baseline, stage/promote exact web,
retain selected runtime/anonymous evidence, and report the running URL. No new
permission is needed for the owner's explicitly requested web release. If any
gate fails, retain its concrete result and repair within the authorized scope.

## PR handoff

500 records actual execution; docs/selected public receipts will be submitted
through normal source review. The only tooling addition is the credential-free
read-only transport helper/workflow, added to standard tests. No deployment
policy, financial wire or runtime product source changes are proposed.
Existing SQL1–48 immutable and current API/provider behavior preserved.
