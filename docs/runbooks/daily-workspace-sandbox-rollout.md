<!-- stage-runbook: true -->
# Daily workspace Sandbox rollout —436

## Status and boundary

RELEASE436, ops/436-sandbox-review from2315cc4 (application stack46e67ea). Owner explicitly requested merge
and deployment for actual review. Existing Sandbox jayburd-dev-api/web and
restricted jayburd-dev-migrate/bookkeeping only, in jayburd-bookkeeping-dev,
9088794002/us-central1. No Production service/database, private records, provider
requests, credentials/environments, Terraform state/plans, consent/cap changes,
worker execution/scheduling, IAM or new resources. SQL1–41 immutable;42+unclaimed.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume/authorization | User directed merge/deploy after PR431/434/435 review handoff; live436/412/65, managed release/identity/migration/source contracts read; dedicated worktree and seed packet |
| Merge |421–427/431/434/435 each five-green exact head, merged in dependency order with normal merge and no admin bypass. Main46e67ea tree exactly equals tested925c1da |
| Target/predecessor | SDK metadata confirms separate dev/prod services. Existing Sandbox API00021-r59/web00021-dnb; old migrator23397c80…; read-only predecessor execution2hrtl succeeded. No private rows read |
| Recovery metadata | Existing shared PostgreSQL17/db-g1-small RUNNABLE; backup/PITR enabled with7day logs; automated backup1793685600000 SUCCESSFUL2026-11-03T08:32:39Z. No dump or restore claim |
| Initial build |41765752855 attempts1/2 passed source validation and candidate runtime but scanner installer failed DNS before scanning/auth/publication; no runtime/schema change.437/PR438 repairs same-version install from independently checksum-verified official release; five-green source merged |
| Build | Trusted owner/main41771886164 attempt1/source2315cc4 SUCCESS all7 jobs. All four candidate receipts independently recomputed against build metadata, tested image IDs and repository digests; exact image scans contain zero HIGH/CRITICAL findings |
| Migration |2026-11-06: predecessor status9jc4r PASS; restricted exact-digest up7bcs5 PASS at18:11:54Z; separate default statusb2xgk PASS at18:13:31Z, proving clean41 and complete embedded artifact checksums. Exact identity/task/timeout/retry/args/completion accepted by migrationReceipt |
| Rollout | API00022-2sq and web00022-mzn Ready/latest-created/100percent. Named non-secret service identity/args/command/resources/ports/probes/timeout/concurrency projections exactly equal predecessor after substituting image. No worker or Production rollout |
| Smoke | Anonymous GET: API/v1/workspace403, web/v1/workspace401, web/api/auth-config200, web root200. No token, authenticated DOM, provider call or financial record retrieval. Owner visual review remains |
| Validation | Exact build hosted application/frontend/SQL validation and all4 candidate runtime/scan/publication jobs PASS. Current local make check/build/release-test PASS using installed Go1.26.5/external modfile/cache; full make test attempted but socket-binding tests blocked by local sandbox. No test skip or tracked toolchain change |
| Commit checkpoint | Owner requested commit/push after interruption. At2026-11-04T04:21Z, repaired build41771886164 has frontend PASS and remaining validation in progress. No forward migration or API/web rollout executed; predecessor status-only proof is the only managed job execution in this stage |

## Operator procedure and evidence boundary

Use the already-authenticated official Cloud SDK and existing restricted workload
identities. Build images uses existing numeric owner/repository/main/manual WIF;
its validation, actual candidate runtime and HIGH/CRITICAL scans must all pass.
Independently recompute all component candidate receipts and verify source/run/
attempt before using immutable digests. Unsigned metadata is not an attestation.

The current Sandbox enrollment deadline is2026-11-02T04Z. Retain it exactly:
cold-expired-startup tests preserve SQL-only inspection without provider/credential
I/O. Replaying a stale Terraform input or extending consent solely for UI review
would change this boundary. Therefore this owner-authorized operator rollout uses
existing job/service image-only SDK updates, with exact pins retained below for
the next operator Terraform input; no Terraform apply/state/plan inspection.
This is an explicit operational deviation from the normal manifest-driven
Terraform rollout, not a change to default local/CI commands or release validators.

Migrator defaults remain action=status against the existing Sandbox instance and
bookkeeping database. Execute up only as an explicit one-off argument override
after receipt checks, then separately execute default status to prove clean41 and
all embedded checksums. No down, force, role bootstrap or application-startup DDL.
Read only execution identity, image/args/service identity and completion counters.

Update API/web by image only. Preserve their command arguments, service identity,
timeouts, concurrency, CPU/memory, ports/probes and100percent ready traffic; compare
only named non-secret projections. No environment or secret retrieval. Retain old
image pins; investigate/forward repair on mismatch, never roll back SQL. Do not
assert Terraform inputs already match the new operator pins.

Sources: [selected SDK update/execute review](../research/gcp-sources.md#operator-image-only-review--2026-11-04),
existing [migration/status](../contracts/migration-status.md),
[Sandbox release](../contracts/sandbox-release.md) and
[cloud rollout](../contracts/cloud-run-rollout.md) contracts. Existing ISO4217/
source-activity mapping remains; this operational record adds no financial wire,
recognition or conformance claim. Scope/expiry and reviewed SQL are project policy,
not inferred from the vendor CLI. Runtime image updates and migrations are not
atomic; schema proof precedes either service change.

## Current evidence

### Resumed operator rollout —2026-11-06

Successful [trusted build41771886164](https://github.com/justin-napolitano/jayburd-bookkeeping/actions/runs/41771886164)
was independently fetched, with numeric owner/repository, main/manual source,
attempt1 and all candidate receipt identities verified. Product inputs remain
the reviewed merged daily workspace. No new source or SQL was deployed.

| Component | Exact reviewed/published image digest | Runtime evidence |
| --- | --- | --- |
| migrate |`sha256:e5060c3a62fcf2c6b7f04d838560ec226a48cb7a4470ce1b6211650a88cde691`|up7bcs5 and independent statusb2xgk complete,1success/0failed/cancelled/running/retried |
| api |`sha256:e15d8c01bd7ff8a891adf01a183cccd163afa83dde3e42193605ddd4ff9e90e6`|jayburd-dev-api-00022-2sq |
| web |`sha256:b525fe62714d62fe94d8f7083bc44fa0d747e78ddc146008ae40247e4e80dd8a`|jayburd-dev-web-00022-mzn |

Repository prefix for all rows:
`us-central1-docker.pkg.dev/jayburd-bookkeeping-dev/jayburd-dev/<component>@`.
Predecessor API524d6f9b6df804926082220fd8de45f8b895e3483885263ee68d10a68a595e19,
web revision00021-dnb and migrator23397c806b985840ca2d5c54924ec7f4040439148c87699ba830c507c0cfe0ea
were retained in projected local operator evidence. No SQL rollback is authorized.
Next operator Terraform inputs must use these new pins; this SDK operation did
not reconcile Terraform inputs or inspect state/plans.

Cloud SQL is RUNNABLE PostgreSQL17 with backup/PITR enabled and7day logs;
automated backup1793944800000 SUCCESSFUL ended2026-11-06T06:48:01.670Z.
This is metadata evidence, not an executed restore. Previous rows below are the
historical checkpoint, not a claim that rollout remains pending.

Predecessor projected operational metadata retained outside source; no full
service environment/state/plan or financial records inspected. Current build
source46e67ea and both latest PR tips were verified before initial dispatch;
repaired source2315cc4 preserves every application/SQL/web/Docker input. Existing
build/deploy pools ACTIVE; no federation or permissions changed by this operation.

## Remaining work

Operator migration/status, image rollout and anonymous transport are complete.
Publish evidence PR and obtain owner visual review of the Sandbox workspace.
Categorization/rules/recognized Spending/savings remain proposed/gated under433/
413/414/417/418/419; deployment does not implement them. Production pilot consent
and owner-private reconciliation remain separate and unchanged.

## PR handoff

Refs436/412/432/429/433. Operations evidence only; no new product DDL/source or
grant. Existing merged40/41 may be applied to Sandbox as required. No live bank
connection, enrollment, posting, consent renewal or Production rollout authorized.
