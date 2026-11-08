<!-- stage-runbook: true -->
# Private filters and rules Sandbox rollout

## Status and boundary

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
| Forward execution | Existing migrator image-only updated to cde714ae…; selected metadata confirms original status args/identity/one task/parallelism1/retry0/660s. Explicit per-execution up jayburd-dev-migrate-7pzlh started03:14Z; terminal outcome pending. |

## Current evidence

Fresh service/job projected metadata matches the previous accepted rollout.
Only accepted immutable images/traffic will change. Existing API argument list,
including collection-read-enabled=true and consent expiry2026-11-02T04Z, remains
exact. Activation plan file is absent; no private grant/history activation occurs.
Existing SDK operation/source review in gcp-sources.md applies: no-traffic can
show Ready/Retired (import only); require active Ready/ContainerHealthy after
exact revision activation. Never treat a retired stage as running health proof.

## Remaining work

Require four clean tested published candidate receipts with matching source/image
and zero HIGH/CRITICAL scans, current backup/PITR metadata, fresh clean44 status.
Update existing migrator image only, execute one explicit up45–46, then independent
default status proves clean46/full embedded hashes. No down/force/reset/retry.
Stage exact API/web images with no traffic, compare all named selected settings,
activate exact API then web and verify active healthy/100percent. Anonymous
root/auth-config200, web private401/direct API403 only; no financial UI inspection.
Record immutable pins for next reviewed Terraform inputs; never replay stale ones.

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
