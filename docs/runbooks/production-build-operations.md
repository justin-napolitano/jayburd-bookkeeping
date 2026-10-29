<!-- stage-runbook: true -->
# Production candidate build operations — #365/#316

## Status and boundary

PROD365-BUILD; build/365-production-build from6fa5210. Owner requests scanned
Production candidate images followed by keyless metadata lookup. This branch
supplies exact public target bytes for the existing reviewed manual workflow;
no application, dependency, migration, IAM policy or financial message changes.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | PR366 merged6fa5210 after five successful checks; issues365/316/337 remain open |
| Owner injection | Two masked static Plaid entries confirmed in isolated Production project |
| Identity | Existing delivery identity repurposed with action-time owner approval; metadata below |
| Candidate | Existing shared project/Production registry, expiry2026-10-31T20:00:00Z; exact target hash required |
| Validation | make check/test/build and10 build-target/proof cases PASS; exact finite target accepted |
| Hosted | No Production build/publication, authentication lookup or delivery yet |

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

Human review of target and existing federation opt-in, trusted scanned build,
exact-image metadata-only lookup, old-binding denial, then separately reviewed
exact source versions/IDs and one-shot static delivery/recovery qualification.
No source value or provider payload belongs in logs, artifacts or chat. Hosted
services, worker SQL grants, independent runtime verification and pilot mandate
remain separate open work. Refs #365/#316/#337; no completion claim.

## PR handoff

Exact-byte target and evidence only. No source/runtime/IAM policy or migration
change. Publication remains gated on human target review and successful CI;
candidate images do not authorize deployment or live bookkeeping.
