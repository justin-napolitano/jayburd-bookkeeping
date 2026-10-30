<!-- stage-runbook: true -->
# Production candidate build operations — #365/#316

## Status and boundary

PROD365-EVIDENCE; docs/365-hosted-build-lookup fromed9a2fc. Owner requests scanned
Production candidate images followed by keyless metadata lookup. This branch
supplies exact public target bytes for the existing reviewed manual workflow;
no application, dependency, migration, IAM policy or financial message changes.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | PR366 merged6fa5210 after five successful checks; issues365/316/337 remain open |
| Owner injection | Two masked static Plaid entries confirmed in isolated Production project |
| Identity | Existing delivery identity repurposed with action-time owner approval; metadata below |
| Candidate | Owner reviewed367; mergeded9a2fc after five successful checks; expiry2026-10-31T20:00:00Z |
| Validation | make check/test/build and10 build-target/proof cases PASS; exact finite target accepted |
| Hosted build | Canonical40932411401 succeeded all four images; independently matched source/run/receipts/inventory |
| Hosted lookup | Exact scanned worker, dedicated GCP identity; both source version1 metadata lookups succeeded; values hidden |
| Hosted delivery | Both exact source version1 entries delivered to enabled destination version1; acknowledged client-ID replay left version count1 |
| Cleanup | Builder provider/pool disabled; lookup mode restored and direct destination grants removed |

## Current evidence

The Infisical plan rejected a fourth machine identity. Owner explicitly approved
retiring Sandbox delivery in place instead of paying for an upgrade. Identity
8d510558-6b96-44c0-b25f-ce73ff7e4d74 is now jayburd-prod-delivery. Main bookkeeping
and canary memberships were removed before adding Viewer only on
jayburd-production-credentials (cfd371a3-1229-48d8-a9e2-9c2ea37540cc).
Organization role remains no-access and delete protection enabled. Sole GCP ID
Token Auth binding is jayburd-prod-delivery@jayburd-bookkeeping-dev.iam.gserviceaccount.com;
TTL/max TTL900 seconds; no allowed-project wildcard. Saved UI views independently
confirmed these settings. This is configuration evidence, not successful workload
authentication or old-binding denial. No values were opened/copied by agent.

The reviewed production-delivery Terraform root was applied with delivery=null:
dedicated GCP identity only, no job, registry access or destination version-adder
grant. Its private backend prefix is production-static-delivery. State, plans and
owner credentials are never copied into this repository.

Read-only GCP checks confirm the existing builder provider is disabled and its
condition pins numeric repository/owner/actor, workflow_dispatch, main and the
exact Production workflow. Builder service-account policy has the reviewed
repository principalSet only; existing Production registry policy grants its
builder Writer only, and no project-level builder grants were returned.
Target SHA256: 4cae02942bd27c422f3bc2c679c588a1dfc815f3fd10c71000733e7ad16b2a38.
Core gates used local Go1.26.5/external temporary modfile, tracked1.26.10 unchanged.
Initial synthetic tests were blocked by sandbox loopback denial; authorized
rerun with local sockets passed. No provider calls occurred in those tests.

## Actual hosted build and lookup

Owner reviewed exact target and existing builder opt-in in session. PR367 merged
ed9a2fcb20584d3e318ef073c246a9541e9149a3 after all five checks passed. Applied the
reviewed foundation flag true, then dispatched
[Production run40932411401](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/40932411401),
attempt1, with the exact target SHA256 above. All validation and four image jobs
passed. Each exact local image passed runtime tests and the HIGH/CRITICAL gate
before authentication/push. Separately authenticated GitHub run metadata matched
owner/repository/main/workflow_dispatch/path/source/run/attempt. Recomputed all
four component receipts from downloaded build metadata, tested image IDs, scans,
published digests and target binding; byte equality passed. Main comparison was
identical at collection. API SBOM contains go-jose/v4 v4.1.6; migrator/worker
inventories do not report that module. No HIGH/CRITICAL findings in these scans;
this is not a claim about every severity or future vulnerability disclosures.

| Component | Published digest in us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-prod |
| --- | --- |
| api | sha256:106c5822d2daca963128e6fed3b2e46c05de3e06a91b852d2f7525ed66cae4b8 |
| migrate | sha256:afeebd5c93ebec50fdd648f3c5b32406854c567258fd3d4ffcaf759a27c25295 |
| web | sha256:7bbafc868fdb07cb70ab9fe7dade484c103c6b457d6765aaf0be6ffc5967b65b |
| worker | sha256:93f106f19ec4836def7a0eaa9b76c238ed94f275503cd883da6efd9e78907e73 |

Registry metadata independently confirms the worker digest. After terminal build,
applied enable_github_build_identity=false; provider and pool both report disabled.
Workflow artifacts retain scan/SBOM/build/component receipts for30days; selected
contents reviewed here, not a full source or supply-chain audit. Candidate receipts
remain deployable=false. Full application release-bundle qualification is separate.

Applied the reviewed production-delivery root in lookup mode using that worker,
source project cfd371a3-1229-48d8-a9e2-9c2ea37540cc, identity
8d510558-6b96-44c0-b25f-ce73ff7e4d74, prod root and exact source version1. Independently
described the job: /deliver, -lookup, Production profile, matching project pair,
deadline2026-10-31T20:00:00Z, dedicated Production service account, one task,
maxRetries0, timeout60s, CPU1/memory512Mi. No scheduler or invocation grant added.

| Key | Execution | Result and metadata |
| --- | --- | --- |
| PLAID_CLIENT_ID | jayburd-prod-static-delivery-9km4p | succeeded1, completed2026-10-29T19:03:14Z; source d7c0f138-e0b1-4618-bb73-4ddb04622042/version1 |
| PLAID_PRODUCTION_SECRET | jayburd-prod-static-delivery-tgm2g | succeeded1, completed2026-10-29T19:08:12Z; source8cb1d9f5-86a6-470c-82f0-47c5d6295a33/version1 |

Selected stdout contains only validated source_id/source_version metadata, with
no values. This proves actual workload authentication and the two exact-version
metadata reads. It does not prove source payload correctness, old-binding denial,
value equality/freshness or provider functionality. Bucket IAM independently
matches create/get-only custom role scoped to production-static-delivery-v1/;
neither destination secret has a direct grant for this account and no direct
project-level account bindings were returned. Destination version listings were
empty both before and after lookup. During this lookup phase there was no
addVersion attempt. No ledger mutation, Plaid call, financial data access,
real bank connection or secret disclosure was performed.

## Acknowledged static delivery and cleanup

Continued the owner's requested credential delivery through the reviewed execute
profile after inspecting both exact source IDs/version1. Selected one destination
at a time; independent IAM checks matched only its SecretVersionAdder grant.
Client-ID grant was removed before adding the Production-secret grant. Runtime
handled values in memory; agent never accessed source/destination payloads.

| Operation | Execution | Result |
| --- | --- | --- |
| Client-ID delivery | jayburd-prod-static-delivery-9jl9j | succeeded1, completed2026-10-29T19:14:51Z; fixed acknowledgement and destination version1 enabled |
| Acknowledged client-ID replay | jayburd-prod-static-delivery-4lwbq | succeeded1, completed2026-10-29T19:18:14Z; acknowledgement, still exactly version1 |
| Production-secret delivery | jayburd-prod-static-delivery-44fwh | succeeded1, completed2026-10-30T01:11:56Z; fixed acknowledgement and destination version1 enabled |

No ambiguous result was retried. The replay followed known success and independent
version metadata; it tests matching-receipt recovery, not unknown acknowledgement.
Exact receipt object metadata independently exists under
production-static-delivery-v1/delivery/9088794002/secrets/:
client-ID /jayburd-prod-bank-sync-client-id/1/receipt.json generation1793301284723840,
and secret /jayburd-prod-bank-sync-client-secret/1/receipt.json generation1793322712939962.
Only names/generation/size were read, not receipt or secret contents.
Destination pins are projects/9088794002/secrets/jayburd-prod-bank-sync-client-id/versions/1
and projects/9088794002/secrets/jayburd-prod-bank-sync-client-secret/versions/1.
These acknowledgements/numeric pins do not establish independent payload equality
or working Production Plaid authentication.

Restored the same job to lookup mode for the Production secret/version1, removing
execute/source-ID flags and the destination-adder resource. Independently verified
both destination policies return no direct grant for this delivery account;
builder provider remains disabled. No execution is scheduled. The scheduling
lease expired during the elapsed interval; live scheduling/main state was rechecked
and the source lease renewed before finishing the handoff. The job's finite
approval deadline remained2026-10-31T20:00:00Z throughout these operations.

## Build decision and acceptance

Use existing build-production-images.yml, source reviewed in PR363/366 and the
production-migrator contract. Existing shared-production-foundation owner/main
federation is disabled until explicitly enabled for this reviewed target. Builder
has writer only on the existing jayburd-prod registry; no deployment role. Exact
repository/owner/main/workflow claims and short-lived identity must be verified
before dispatch. Restore disabled federation after publication or interruption.

Target is public metadata: matched project jayburd-bookkeeping-dev/9088794002,
finite UTC expiry, and issue337 mandate reference. The reference is not itself
approval. Owner reviews exact bytes/hash before merge/publication. Existing
workflow publishes all four candidate components, including worker; each must
pass exact local runtime proof and HIGH/CRITICAL scan before authentication/push.
Retain authenticated run metadata, numeric attempt, digest, scan/SBOM and source
ancestry. A candidate receipt remains deployable=false. No build can activate
bank access, a delivery job, SQL mutation, HTTP service or scheduler.

Alternative: reuse a Development image or bypass target validation. Rejected:
it would not supply the selected Production workflow/registry evidence. This
slice uses existing reviewed mechanisms and source records in the bibliography;
no new external API or accounting policy. ISO20022 does not apply to build
metadata; existing ISO4217/financial contracts are unchanged.

Acceptance: existing production-image-build and build-proof suites, actual
exact-byte target validation, make check/test/build and all exact-head CI gates.
Hosted acceptance additionally requires successful trusted workflow and effective
federation/registry verification. No cost cap is claimed; owner target remains
approximately USD25/month GCP-only, with Plaid separate. No database resize or
additional project/instance is authorized by this target.

## Remaining work

Old-binding denial, independent runtime payload verification and remaining
unknown-outcome/revocation qualification remain open. Source values and destination
payload equality have not been independently verified by runtime yet. The current
consumer CLI still rejects the shared number; #368 records its separate source
fix and required new image qualification before rollout.
No source value or provider payload belongs in logs, artifacts or chat. Hosted
services, worker SQL grants, independent runtime verification and pilot mandate
remain separate open work. Refs #365/#316/#337; no completion claim.

## PR handoff

This handoff changes evidence only; no source/runtime/IAM policy or migration
change. Build and lookup operations above were separately owner-authorized.
Candidate images do not authorize application deployment or live bookkeeping.
