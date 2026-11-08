<!-- stage-runbook: true -->
# Private filters and rules Sandbox rollout

## Status and boundary

Runtime/schema rollout complete; authenticated owner review remains pending.

PR457 merged f7a23a49b91fede297071070a117c7813a4e1258 after five passing
checks on06c12d1 and independent concrete reviews. User standing436 deployment
scope applies to existing jayburd-dev API/web/migrator/bookkeeping,
jayburd-bookkeeping-dev/us-central1 only. GitHub65 root lease renewed through
2026-11-08T06:00Z after checking main/claims; no intervening writer.
No Production, provider call, worker rollout/execution/schedule, new IAM/resource,
consent/cap renewal, financial-record, secret/environment/state/plan read,
owner grant or Item adoption.453 actual verified UID/operator prerequisite stays.
Merged1–46 immutable;47 unallocated. Root owns all cloud mutations.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Source acceptance | Seed42351309445 all frontend/application/backend success; Infrastructure42351309461 and GitGuardian success. Actual connected filters/detail/rules, historical recovery/old journeys, role/atomicity/replay/race and607 units/82 component browsers proved |
| Trusted candidates | Owner/main Build42440020895 attempt1 dispatched2026-11-08T02:47:41Z at exactf7a23a4. Independent numeric owner/repository/main/manual/workflow checks pass; all validation/runtime/scan/publication gates SUCCESS; independent full receipt recomputation and scan/SBOM/source identity pass for all four; HIGH0/CRITICAL0 |
| Predecessor | API00023-6sp /45bc930c… and web00023-6wv /e8f2a258… Ready/sole100percent. Existing migrator a2bdc64d… defaults status, one task/parallelism1, retry0, timeout660s, restricted identity. Named nonsecret settings retained |
| Recovery metadata | RUNNABLE PostgreSQL17/db-g1-small, backup/PITR enabled with7-day log retention. Latest automated backup1794031200000 SUCCESSFUL ended2026-11-07T07:14:31.264Z. SDK first backup projection had a local credential-cache write permission error; unchanged read-only retry succeeded, no credential/config copy or auth change |
| Clean predecessor | Fresh default status jayburd-dev-migrate-59q9j completed2026-11-08T03:07:37.870266Z, succeeded1/no failures or retries, clean44/full checksums. |
| Forward execution | Existing migrator image-only updated to cde714ae…; selected metadata confirms original status args/identity/one task/parallelism1/retry0/660s. Explicit per-execution up jayburd-dev-migrate-7pzlh completed2026-11-08T03:16:06.023948Z, succeeded1/no failures/retries. Existing migration-receipt.mjs accepted exact image/action/identity/task/timeout/completion. Separate unchanged default status jayburd-dev-migrate-kvvz5 completed2026-11-08T03:17:42.776851Z, succeeded1/no failures/retries; exact migration-receipt accepted, clean46/full embedded checksums. |
| Image-only stage | API00024-r9d/web00024-cf5 exact selected spec equals predecessor except image; old00023 sole100 before promotion. API direct revision staged Ready/Retired, ActiveFalse, ResourcesUnknown: import evidence only. |
| API promotion | Exact API00024-r9d sole100/latestReady, service Ready/ConfigurationsReady/RoutesReady true. Direct revision Ready/Active/ContainerHealthy/ResourcesAvailable true03:19Z and exact imageDigest. Selected spec remains identical to accepted stage. |
| Web promotion | Exact web00024-cf5 sole100/latestReady, all service and direct active/startup health conditions true03:19Z, exact imageDigest. Selected spec unchanged from accepted stage. Existing validateRevisions accepts both exact immutable images/latestReady/sole100. |
| Transport | Anonymous numeric web root/auth-config200; web workspace/collection/recovery/v3 taxonomy401; direct API workspace403. curl IPv4, no supplied credentials, redirects not followed, all response bodies discarded. |

## Current evidence

Fresh service/job projected metadata matches the previous accepted rollout.
Only the accepted immutable images/traffic changed. Existing API argument list,
including collection-read-enabled=true and consent expiry2026-11-02T04Z, remains
exact. Activation plan file is absent; no private grant/history activation occurs.
Existing SDK operation/source review in gcp-sources.md applies: no-traffic can
show Ready/Retired (import only); require active Ready/ContainerHealthy after
exact revision activation. Never treat a retired stage as running health proof.

## Remaining work

Managed runtime/schema deployment acceptance is complete. Owner reviews the
[existing UI](https://jayburd-dev-web-9088794002.us-central1.run.app).
No authenticated financial UI inspection was performed. Institution/legal-owner
metadata foundation242 and actual verified-owner/operator activation453 remain
open; source-only optional inline delivery is separate draft459, not deployed.
PFC integration413 remains open/map revision0. No live bookkeeping approval.
Reconcile these exact image pins with the next reviewed Terraform inputs;
no Terraform state/plan was read or stale input applied. Default migrator remains
status. No down/force/reset/retry, owner provisioning/adoption or worker rollout.

## PR handoff

Operational evidence only; GitHub436/65 own state.242 missing institution/legal
owner metadata and453 verified-owner/operator activation stay explicit. No hosted
owner identity, financial correctness or live bookkeeping acceptance inferred.

## Accepted immutable candidates

All use us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/.

- api@sha256:88a946acc77363c4eb8e9c9d43133e4a556bff9c5a6cd7816de8cf06246a7404
- migrate@sha256:cde714aed27e71fd92e3b2e547b09c23d21de6090a65f90e10d58613137a220d
- web@sha256:53e0757f4990614919cec13e0fb633abb8d9182d805da5852e26528dbbfa68af
- worker@sha256:c526bbbae1ce2faaf80f9de72779466369b6738a0220f6dafc3ad876f5363d16 (build evidence only; no rollout/execution).

Independent [build evidence](rules-build-evidence.md) records complete identities.

## Predecessor immutable pins and preserved settings

- api@sha256:45bc930c4853edf171e8b4c1a7f1684524248c4d03a59f1f4b48e8b1094dfd7c, revision jayburd-dev-api-00023-6sp.
- web@sha256:e8f2a258a8454a1c455044691ea89cf5b84320e967e04c7c521e69969847796c, revision jayburd-dev-web-00023-6wv.
- migrate@sha256:a2bdc64d39a852237d01d2dde5e884cba575fed96c64b30dfe6b074a63c7bf12, embedded44.

API identity jayburd-dev-api, concurrency16/timeout15s/CPU1/512Mi/port8080;
web identity jayburd-dev-frontend, concurrency32/timeout20s/CPU1/512Mi/port8080.
API startup /readyz and web /api/auth-config retain failure12/period5/timeout3.
All20 API arguments compare byte-for-byte with predecessor, including numeric
web origin, database/Infisical/credential-version references, private-read true,
connections-enabled, false Link qualification and unchanged expired consent.
Web command/args remain absent. No environment values were retrieved.
Restricted migrator identity/1task/parallelism1/retry0/660s/defaultstatus retained.
Selected safe metadata and existing migration receipts remain in private tmp;
public evidence records bounded outcomes without secret or plan payloads.

Any reviewed API/web image rollback retains schema46 and the current default-status
migrator. The old embedded44 migrator pin is historical; it does not authorize a
schema downgrade, down/force/reset, or replay.

Independent final documentation review checked full candidate/predecessor pins,
execution/revision times and probe arithmetic. Corrected split table/stale wording
and made schema-retaining rollback explicit. make check PASS using installed
Go1.26.5/external exact-dependency modfile; no tracked dependency/toolchain change.
Documentation-only delta over the exact f7 source whose canonical full build passed.
