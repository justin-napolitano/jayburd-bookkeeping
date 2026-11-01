# Webhook managed IAM compatibility — #378

Observed2026-11-01: approved Production bootstrap rolled back with
webhook_bootstrap_rejected. Read-only catalog metadata showed safe intended
login/capability flags, denied Sandbox CONNECT and owner SET; the login's
cloudsqliamserviceaccount edge had ADMIN=false/SET=true. No grant persisted.

Re-reviewed Google Cloud SQL users, System users and roles:
https://docs.cloud.google.com/sql/docs/postgres/users#cloud_sql_system_users_and_roles
and PostgreSQL17 SET ROLE Description/Notes:
https://www.postgresql.org/docs/17/sql-set-role.html . Selected sections only,
2026-11-01; the vendor describes the managed marker and prohibits customer
modification; hosted catalog metadata supplies actual membership options.
The prior [verifier decision](verifier-iam-compatibility.md) and migration31 are the local
precedent, not a blanket exception for all Cloud SQL roles.

Decision: both operator webhook bootstraps permit SET only for the exact login's
inherited, ADMIN=false, safe NOLOGIN cloudsqliamserviceaccount marker with no
parent memberships. Every other SET/ADMIN edge still denies. Effective database,
schema, table, column, sequence and function checks cover inherited authority;
column checks are explicit so a marker cannot hide column-only access. Direct
ACL, predefined role, exact target/profile and Production cross-database checks
remain. No managed role/membership is modified or granted. No migration or runtime
image change; independently qualified image source remains1eec1f0.

Alternatives: removing Google's membership violates its managed-role boundary;
allowing all SET edges would admit privilege escalation. A named role alone is
insufficient. STRIDE elevation(high): exact topology/attributes plus effective ACL
denials; replay(high): existing immutable ingress profile/receipt/current-state
checks; disclosure(high): metadata-only diagnostics, no payload/secret inspection.

Internal authorization only: ISO20022/4217/21378 mapping and financial semantics
unchanged; no new financial message identifier or conformance claim. Synthetic
PG17 tests must prove safe marker initial/replay and runtime capability while
rejecting ADMIN, NOINHERIT, LOGIN/unsafe attributes, parents, unrelated/capability
SET and inherited table/column/function authority. Existing bootstrap/SQL/core
checks and exact-head CI required. Human security review before merge/apply;
actual fixed bootstrap/readback remains375, not a synthetic success claim.
