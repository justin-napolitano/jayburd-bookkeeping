# Webhook bootstrap delivery

<!-- stage-runbook: true -->

## Status and boundary

Issue #271; lease `PLAID-271-PREP-20261022T1725Z`, expiry22:00Z. Branch
`issue/271-webhook-bootstrap`, base `a2c9dee793f81ed483055b0ee916876b59265d27`.
Implementation preparation and disposable synthetic proof only. No cloud, provider,
credential, environment inspection or GitHub operations. No product migration.

## Stage ledger

1. Read issue271 export, repository index/bibliography, database foundations,
   existing Cloud SQL/bootstrap/webhook contracts and ingress function source.
   Coordinator approved interface checkpoint before implementation.
2. Recorded PostgreSQL17 source sections, applicability and alternatives in
   [bootstrap contract](../contracts/webhook-bootstrap.md). Existing ingress body
   includes enabled: changing true→false requires a new stable operation key.
3. Added fixed development-project SQL bootstrap and inert wrapper. Existing
   roles/users are validated; no IAM creation, proxy or source API operation.
4. Added disposable pinned-image proof for safe replay, enable/disable, actual
   restricted login and negative ACL/membership/delegation cases. Initial test
   harness omitted Docker stdin attachment; corrected fixture setup before proof.

## Current evidence

Synthetic and core validation results recorded after execution below. No hosted
claim. No source credentials or raw application data are involved.

## Remaining work

Coordinator integrates shared Makefile/CI/index; infrastructure lane owns IAM.
Human security review and fresh exact trusted image/release metadata precede
public deployment. Owner must independently provide an authorized IAM SQL socket
and reviewed operator scope before `--apply`. Existing source consent and earlier
release manifests expired; no extension supplied here. Hosted role authority,
bootstrap COMMIT outcome, ingress enablement and actual signed chain remain open.

## PR handoff

Links #271/#103/#87/#88. Scope is two new scripts, one new bootstrap SQL file,
contract and this runbook. Migration impact none; merged migrations unchanged.
Core/migration/ledger gates, staged runbook guard and diff check are required before
commit. No foundation test or cloud mutation is part of this work.


Final evidence PASS:

- `bash scripts/test-webhook-bootstrap.sh`: exact enable/disable replay, changed
  body collision and stale historical-enable replay rejection; actual clean-env
  wrapper apply/replay in the disposable container; unsafe prospective capability,
  mixed roles, predefined server-program role, direct table/sequence/function,
  inherited function delegation, schema delegation and alternate-grantor checks.
  Comma-separated socket host lists and wrong project metadata are rejected.
  Log `/private/tmp/jayburd-271-bootstrap.log`.
- `make check test build`: PASS with documented external local Go1.26.5 modfile;
  tracked Go1.26.10 unchanged. Log `/private/tmp/jayburd-271-bootstrap-core.log`.
- `make migration-test cloud-sql-test ledger-test`: PASS, including actual runtime
  ACL and restore replay proofs, only disposable synthetic databases. Log
  `/private/tmp/jayburd-271-bootstrap-permissions.log`.
- Independent review found a libpq comma-host fallback path; rejected comma before
  psql and added metadata regression. Coordinator review added predefined pg_*
  role rejection. Documented separately scoped emergency owner disable when unsafe
  role drift intentionally blocks the strict bootstrap. No cloud apply occurred.
- Staged runbook guard and diff check run before commit. Initial synthetic fixture
  runs also found a pre-owner regprocedure schema lookup and unsafe optimizer
  evaluation of sequence inspection; changed to catalog signature matching and
  CASE-guarded sequence checks, then reran the full bootstrap proof successfully.
