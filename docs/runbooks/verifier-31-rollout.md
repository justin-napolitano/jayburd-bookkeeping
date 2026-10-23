# Verifier migration31 rollout
<!-- stage-runbook: true -->

## Status and boundary

Issue272; lease VERIFY272-RELEASE31-20261023T1836Z. Owner approved PR284 source
and requested immediate rollout preparation. Merged source fb4ac0411f8f5673ab0659825bcafbdb4e64253e.
The exact immutable release artifact still requires review before dispatch under
deploy/README.md. This stage introduces no SQL, grants, provider calls or mandate.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Source | PR284 merged; all five exact-head checks SUCCESS, Seed40004028278/Infrastructure40004028330 | Preserve evidence |
| Trusted build | Owner workflow40004622277 on mainfb4ac04, attempt1; all seven jobs SUCCESS, four tested/scanned immutable images published | Same-run receipts validated |
| Candidate | Exact migrate/status manifests validated with trusted metadata and three component receipts | Owner exact digest/hash review |
| Apply/status | Not executed | Separate reviewed release and immutable receipts |
| SQL login | Prior30 bootstrap rejected managed IAM marker and rolled back | Retry revised bootstrap only after31 status proof |

## Current evidence

Migration30 was previously applied and independently status-verified. Migration31
replaces only integration.verifier_login and preserves signature, owner, OID and
ACL; db/bootstrap/verifier-login.sql is separate owner-operated login wiring.
Existing API/web/scheduler/worker roots are not part of migration-only execution.
Actual metadata diagnosed only the Google-managed marker SET edge; no financial
data was read. See the [IAM contract](../contracts/verifier-iam-compatibility.md)
for source/test details.
Fresh read-only metadata confirms the actual marker is NOLOGIN, has no unsafe
role attributes or parent memberships, and its login edge is ADMIN false,
INHERIT true, SET true. These are preliminary compatibility checks, not a full
effective-ACL or mandate/comparison PASS. Operator can SET owner, and the login
and capability attributes pass. The diagnostic transaction made no mutations.

Release branch make check/test/build/release-test PASS with documented cached
Go1.26.5 temporary modfile; tracked1.26.10 unchanged. Local HTTP port sandbox
denials used approved unrestricted retry. Public build receipt download had a
sandbox network failure and succeeded on approved retry. No private logs,
plans/state or credentials inspected/copied. Trusted build supplies canonical
Go1.26.10 and complete browser/API/SQL/image-runtime/security-scan evidence.

Exact candidates (not dispatched), expire2026-10-24T02:00Z:

| Operation/path | SHA256 |
| --- | --- |
| migrate, deploy/dev-release.json | 83976b7dd1e43aebca23eb66869facc9d69e0aea74687320924b2c84d44dda8f |
| verify, deploy/migration31-status.json | de934df315f60ae0969357934345b79c220dfc7d5cd100ff8a68343bf22ec140 |

Only the existing bounded migration job/root executes: one task, zero retries,
660-second timeout,1CPU/512Mi, fixed IAM migrator and action-specific arguments.
The forward migration changes only the permission predicate and is not a grant.
Status checks clean latest31 and full embedded immutable checksums. Existing
reviewed bootstrap and new compatibility evidence are explicit manifest links.
Existing dev database/backend/registry scope remains; bounded job compute is the
execution scope added. Neither candidate deploys services, provider enrollment/
consent or account connections. The secondary status candidate is inert; the
current workflow reads only dev-release.json.

## Remaining work

Owner review of both exact manifests/digests, native merge/dispatch, actual
immutable execution receipt, separate read-only
status receipt/checksum proof, then revised bounded bootstrap. Reconcile saved
browser command after separately approved app recovery. Never force/down or
blindly retry an uncertain execution. Old30-only images cannot validate31.
Prepare the exact status candidate alongside the migration candidate, for one
review of both immutable byte sequences. Deploy reads only dev-release.json:
after actual migration acceptance, a separate native PR must copy the already
reviewed status bytes there unchanged before status dispatch. Recheck both hashes,
expiry and current main; changed bytes or expired approval require fresh review.

## PR handoff

Links272/88. Migration-only release adds no app promotion or new-source consent;
issue281 remains unresolved. No hosted equality/zero-journal claim until actual
finite mandate/comparison and cleanup. Preserve original UNKNOWN registration.
