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
| Build | Trusted owner/main41771886164 from2315cc4 running; product/SQL/web/Docker inputs equal46e67ea, only installer/ops source changed. Exact receipt/runtime/scan qualification pending |
| Migration | Pending exact matching build, restricted forward up through41 and independent read-only status |
| Rollout | Pending exact API/web digest-only update and Ready/traffic/settings comparison |
| Smoke | Pending anonymous web/auth-config and denied private API; owner visual review after deployment |
| Validation | Source stack core/SQL/API/web/full-stack proofs passed; ops core make check/test/build/release-test PASS with installed Go/external modfile adapter. Final receipt/link/diff checks pending |
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

Predecessor projected operational metadata retained outside source; no full
service environment/state/plan or financial records inspected. Current build
source46e67ea and both latest PR tips were verified before initial dispatch;
repaired source2315cc4 preserves every application/SQL/web/Docker input. Existing
build/deploy pools ACTIVE; no federation or permissions changed by this operation.

## Remaining work

Finish trusted build/receipts, required forward migration/status, API/web rollout,
anonymous transport and exact operational readback, evidence PR and owner review.
Categorization/rules/recognized Spending/savings remain proposed/gated under433/
413/414/417/418/419; deployment does not implement them. Production pilot consent
and owner-private reconciliation remain separate and unchanged.

## PR handoff

Refs436/412/432/429/433. Operations evidence only; no new product DDL/source or
grant. Existing merged40/41 may be applied to Sandbox as required. No live bank
connection, enrollment, posting, consent renewal or Production rollout authorized.
