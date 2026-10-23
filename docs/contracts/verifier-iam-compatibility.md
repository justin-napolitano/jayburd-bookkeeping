# Verifier IAM authentication compatibility

Issue272, owner-directed implementation2026-10-23; proposed sensitive-source
review before migration/application. Migration30 remains immutable.

## Evidence and decision

Read-only hosted role metadata showed the verifier login and NOLOGIN capability
have safe attributes and the operator can SET bookkeeping_owner. The login's only
membership is cloudsqliamserviceaccount, ADMIN false, INHERIT true, SET true.
The blanket SET-membership rejection therefore prevents bootstrap before grants.
No financial records were read and no authority persisted from the rejection.

Selected official sections reviewed2026-10-23:

- [Google Cloud SQL users, System users and roles](https://docs.cloud.google.com/sql/docs/postgres/users#cloud_sql_system_users_and_roles):
  cloudsqliamserviceaccount designates IAM service-account authentication;
  system roles cannot be modified/deleted or assigned by customers (except the
  documented cloudsqlsuperuser assignment). Vendor implementation guidance,
  not an accounting standard; actual instance metadata separately establishes
  this membership's options.
- [PostgreSQL17 SET ROLE, Description and Notes](https://www.postgresql.org/docs/17/sql-set-role.html):
  SET membership permits assuming the target role; INHERIT grants its effective
  privileges without switching. Selected sections, not the entire manual.

Decision: allow SET only for the exact login-to-cloudsqliamserviceaccount edge,
ADMIN false and INHERIT true, with a safe NOLOGIN marker having no parent-role
memberships. Every other SET/ADMIN edge remains rejected, including capability
memberships and alternate grantors. The existing effective database/schema/
table/column/sequence/function checks still cover the marker through inheritance.
Marker ownership and unsafe attributes must not open an authority path. Explicit
SET marker execution still cannot invoke verification: existing entrypoints
require the original session identity and role NONE. No marker role is created,
modified or granted by bootstrap; local synthetic tests supply the simulation.

Alternative: alter/remove Google's marker membership. Rejected because it changes
managed authentication and conflicts with the documented system-role boundary.
Alternative: allow all cloudsql roles or all SET edges. Rejected: broadens authority.
Inference: this narrowly inspected marker is compatible with restricted execution;
its name alone is insufficient. Unexpected privileges or role topology still fail.

## Standards and acceptance

This is internal database authorization, with no financial message/data mapping.
ISO20022/4217 and ISO21378 mappings are unchanged by this slice; no applicable
financial message identifier is invented. Existing verifier financial semantics
and multicurrency requirements remain unchanged; no ISO conformance claim.

Forward migration31 replaces only integration.verifier_login; signature,
ownership and existing EXECUTE ACL remain intact. Down restores the previous
predicate; live down-migration is not an approved rollout. Bootstrap mirrors the
narrow preliminary exception and retains the complete final predicate check.

Acceptance: migration-test, ledger-test, sandbox-verification-test and
verifier-bootstrap-test on disposable PostgreSQL17; safe marker bootstrap/replay
and actual login succeed, while marker ADMIN, NOINHERIT, unsafe attributes,
parent roles, effective financial/column/function authority, unrelated SET edges,
and direct capability SET remain denied. SET marker does not authorize comparison.
Core check/test/build plus exact-head CI remain required. Human security review,
trusted rebuilt migration image, approved finite release and hosted retry are
separate gates; no hosted PASS or mandate follows from source tests.
