<!-- stage-runbook: true -->
# Final shared Production wiring — #316/#337

## Status and boundary

FINAL316-FOUNDATION; build/316-final-wiring from059f90c. Owner requested completion
after362. Source preparation only, no cloud mutation, provider call, credential,
Terraform state/plan or personal financial record accessed.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |360/361/362 and SQL operations merged;358 closed with exact patched API scan/SBOM proof |
| Foundation | Exact shared project; separate database and Production resources; no instance creation, resize or operator elevation |
| Import boundary | Existing Production API/migrator/database/logins/grants must be imported, never recreated |
| Build/release | Exact shared project pair admitted only with Production registry/keys/secrets and bookkeeping_prod; canonical hashes, finite deadlines and paused scheduling retained |
| Validation | Three clean Terraform mock cases, positive/negative shared build/release proof and core check/test/build/release-test PASS |

## Current evidence

Shared resource access follows existing reviewed worker permissions with distinct
Production custom-role IDs and object-prefix conditions. API writes immutable
registry objects; sync reads exact registry objects, writes/deletes raw evidence
and writes immutable tombstones. Publisher/webhook have no bucket/key access.
Webhook reads only the two static Plaid secret destinations. Terraform creates
no secret values/versions. Source authority and accepted alternative are recorded
in production-pilot-sources.md PP12 and shared-production-sql.md. This changes
operational metadata/IAM, not a financial message or accounting/ISO policy.
Elevation/tampering threats: matched project pair, distinct identities/resources,
no broad bucket roles, disabled build federation, explicit import and no operator.
Availability tradeoff: existing shared capacity/admin/restore scope, no resize.
Local Go1.26.5/external temporary modfile used; tracked1.26.10 retained. Separate
Production secret-sync identity may add/list destination versions but cannot read
payloads; no external impersonation/federation grant is created for it here.

## Remaining work

Human review before new IAM/deployment source is used. Apply/import only reviewed
root using private backend; actual metadata readback, worker SQL bootstrap and
isolated IAM sessions, secret sync and exact-version freshness, paused hosted
web/API/workers/webhook/queue, independent release receipts, identity/recovery/
qualification and bounded personal-account reconciliation remain outstanding.
The already-supplied Production Plaid secret must not be retrieved or re-entered.

## PR handoff

Refs #316/#337. No new migration; managed schema35 already clean. No source-only
PASS establishes actual resource authority, cost target or live bookkeeping.
