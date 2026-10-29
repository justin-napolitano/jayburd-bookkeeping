<!-- stage-runbook: true -->
# Shared Production provider resource profile — #316

## Status and boundary

build/316-shared-profile follows merged360/361. Owner requested completion of
shared Production wiring. Source preparation only; no cloud/provider mutation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Gap | Onboarding/evidence/webhook URL guards still assumed a separate project |
| Change | Permit exact matched shared project metadata; keep dedicated Production bucket, keys, secret names and canonical Production receiver |
| Validation | Core check/test/build, targeted connectionruntime/connections/cloudevidence tests and synthetic browser/API/SQL app-integration PASS |

## Current evidence

Database isolation remains enforced by migrated bookkeeping_prod and distinct
Production IAM logins. The shared project name does not select Sandbox Plaid,
secret environment or evidence storage. Adjacent/mixed metadata remains denied.
Existing Production policy, finite deadlines and numeric secret-version checks
remain required. Financial contracts and ISO mappings are unchanged operational
metadata. Alternatives and owner choice: shared-production-sql.md; source
authority: production-pilot-sources.md PP12. Elevation/tampering risk mitigations:
matched project pair, dedicated resources and fail-closed profile validation.
Local checks use installed Go1.26.5/external temporary modfile; tracked1.26.10
unchanged. No application requests reach real provider endpoints in these tests.

## Remaining work

Human review before using this new security-sensitive profile source. Complete
shared release validators and infrastructure/secret propagation, worker grants,
hosted identity/recovery/qualification. Fresh exact-image scan remains required.

## PR handoff

Refs #316/#337. No DDL/migration changes or hosted activation in this branch.
No secret retrieval or personal financial records accessed.
