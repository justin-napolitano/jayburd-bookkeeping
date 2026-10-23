# Verifier migration31 rollout
<!-- stage-runbook: true -->

## Status and boundary

Issue272; lease VERIFY272-RELEASE31-20261023T1836Z. Owner approved PR284 source
and requested immediate rollout preparation. Merged source fb4ac0411f8f5673ab0659825bcafbdb4e64253e.
Owner approved both exact artifacts and the revised login bootstrap in the current
session. Execution lease VERIFY272-EXEC31-20261023T1857Z through02:00Z. No new SQL
source, provider calls or mandate; login wiring follows actual status acceptance.

## Stage ledger

| Stage | Evidence | Next action |
| --- | --- | --- |
| Source | PR284 merged; all five exact-head checks SUCCESS, Seed40004028278/Infrastructure40004028330 | Preserve evidence |
| Trusted build | Owner workflow40004622277 on mainfb4ac04, attempt1; all seven jobs SUCCESS, four tested/scanned immutable images published | Same-run receipts validated |
| Candidate | Both exact hashes owner-approved; PR285 merged2bef707 with all five checks now SUCCESS | Copy approved status bytes unchanged |
| Apply/status | Apply40007581006 and status40012916635 SUCCESS; exact approved images/actions validated | Preserve bounded receipts |
| SQL login | Reviewed bootstrap hash29d1245b committed; complete effective predicate accepted | Actual finite-mandate comparison remains separate |

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

Exact approved candidates, expire2026-10-24T02:00Z:

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

Actual apply receipt: execution jayburd-dev-migrate-dr5mw completed
2026-10-23T19:01:25.040547Z; approved build40004622277 attempt1/sourcefb4ac04,
migrator digest92f4e18ba938834b25a41f4181fe5db6103d1501ebf6bcb9ec079ef5bb921a18.
Independent selected execution readback confirms exact image/action/instance,
fixed IAM migrator, task1/retries0/timeout660, Completed True, success1 and zero
failure/cancel/running/retry counts; migrationReceipt accepted it. No environment,
provider logs or private state inspected. Metadata PR285's five CI checks are now
terminal SUCCESS. No second apply/replay performed. This branch copies only the
already-approved status bytes into the workflow's fixed manifest path; hash
de934df315f60ae0969357934345b79c220dfc7d5cd100ff8a68343bf22ec140 matches.
Status delivery branch make check/test/build/release-test PASS with documented
cached local toolchain; final exact status bytes and trusted receipts validate.
No source SQL, build image or expiry change in this metadata promotion.

## Actual status and bootstrap

Approved status metadata PR286 merged9395464293c99d62f5cd4fac801ace43278fbfac.
Release40012916635 SUCCESS; validated migration-verification-passed receipt:
execution jayburd-dev-migrate-jszc6 completed2026-10-23T19:39:43.640109Z, same
approved migrator digest/build and verify action. The strict workflow validates
selected immutable execution identity/image/arguments/task/retry/time/terminal
counts; successful31-aware status checks exactly one clean latest31 schema row
and the complete embedded immutable checksum set in a read-only repeatable-read
snapshot. This was not a second apply or live down-migration.

After status acceptance, the owner-operated SQL bootstrap with SHA256
29d1245ba2c788ad8f0de0c8ae09e2e15d1baec833fc979edf534144c7c79ee8
committed through the existing owner Cloud Shell/IAM proxy route at19:41Z.
Exact transferred public SQL bytes were verified before execution. Cloud Shell
had restarted and lost earlier temporary scripts: the first invocation did not
find the script and made no SQL connection; the reviewed files were transferred
again, then executed once successfully. Pinned proxy2.26.0 downloaded from the
official TLS endpoint with bounded network timeouts after that restart.

The bootstrap accepted the complete effective privilege predicate and CONNECT,
granted only the reviewed verifier capability (ADMIN false, INHERIT true, SET
false) and fixed login settings. No managed IAM marker repair, business mandate,
financial read, provider call or comparison. It is operator-confirmed SQL wiring,
not proof of a Cloud Run verifier session authenticating or financial equality.
Private SQL/proxy diagnostics remained unread; only the fixed PASS outcome was
retained. Temporary proxy stopped through exit cleanup; no key/token copied.

## Remaining work

Migration31 rollout and reviewed login bootstrap are complete. Reconcile saved
browser command after separately approved app recovery. Never force/down or
blindly retry an uncertain execution. Old30-only images cannot validate31.
The actual finite-mandate verifier execution and provider identity-only map are
still pending;272/88 remain open. Existing API optional-consent expiry outage281
is unchanged by this database-only rollout. Application recovery needs a fresh
separately reviewed31-aware release; old283 cannot validate the installed schema.
Future releases still require exact reviewed bytes, expiry and current main;
changed bytes or expired approval require fresh review. This apply/status pair
used the same already-reviewed immutable candidates without renewal.

## PR handoff

Links272/88. Migration-only release adds no app promotion or new-source consent;
issue281 remains unresolved. No hosted equality/zero-journal claim until actual
finite mandate/comparison and cleanup. Preserve original UNKNOWN registration.
