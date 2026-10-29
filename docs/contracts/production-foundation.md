# Isolated Production foundation and bootstrap — #316

Coordinator source decision2026-10-28 before implementation; owner/security review
pending. Source baselinea572b31. This root is a candidate, not an apply approval.
337 still owns actual targets, billing, cost, retention and isolation acceptance.

## Scope and prior evidence

Reviewed existing foundation/application Terraform, cloud-sql.md and generic
cloud-sql.sql bootstrap. PP08–11 in production-pilot-sources.md and GCP source
register establish existing identity/backup/price mechanisms; this slice reuses
those pinned Google8.6.0/Terraform1.15.7 resources, not a new provider capability.
STD-01: operational infrastructure/SQL-role metadata is not a financial exchange;
existing ISO4217 and ISO21378 applicability/access limits remain unchanged.

A separate production-foundation root owns named Production service accounts,
private versioned state buckets, immutable-tag image registry, disabled-by-default
owner/main/exact-production-workflow build federation, one PostgreSQL17 instance
and IAM SQL logins. No deployment federation, Run services, runtime keys/secrets,
raw evidence buckets or scheduled jobs are introduced. Builder gets registry writer
only; runtime roles get SQL connector/login rights, not owner/admin rights in IAM.
No service-account keys or project-wide Editor/Owner role.

SQL candidate: db-f1-micro, ZONAL,10GiB SSD (growth capped20), max_connections50,
encrypted connector only, no authorized networks, daily backups retained8 with
7-day PITR and explicit US location. Deletion protections and prevent_destroy
remain. This shared-core tier has no Cloud SQL SLA and needs measured capacity/
restore proof. It does not guarantee the USD25/month GCP-only target, especially
while development stays on. Production billing/alerts and total Sandbox costs
must be reviewed before apply. No running resource is resized by this change.

Names align with355: jayburd-prod, jayburd-prod-migrator/api/sync/publisher/webhook,
jayburd-prod-builder and jayburd-prod-github/owner-build. Other proposed service
accounts remain unprivileged until their own runtime slice. Exact public project
ID/number are mandatory; reject known dev project and number. Bootstrap human
operator is null by default; explicit opt-in only creates the existing bounded
Cloud SQL operator mechanism. Actual SQL admin capability remains unproved.

## SQL bootstrap wrapper

production-cloud-sql.sql verifies explicit project plus exact api/migrator IAM
names belong to that project before invoking the unchanged generic bootstrap.
Use a private owner-controlled connection to the reviewed Production instance;
a login string alone cannot prove server/project identity. Wrapper rejects wrong
project, dev names, same/foreign roles and missing inputs; ON_ERROR_STOP prevents
continuation. Existing bootstrap atomically checks role safety, creates restricted
NOLOGIN capabilities and grants migration owner SET authority without CREATEROLE.
SQL does not enroll provider scope, workers, or webhook ingress. Worker/ingress
bootstrap and actual role readback remain distinct later gates.

Alternative: parameterize existing deployed dev roots. Rejected for this first
pilot because changing their addresses/defaults risks dev state and authority.
Copying a small explicit baseline adds maintenance cost; common factoring may
follow only with unchanged dev synthetic expectations. Combining deployment rights
with builder or enabling federation by default is rejected.

STRIDE: elevation/spoofing(high) -> distinct identities, owner/ref/workflow WIF
conditions and disabled default; tampering/disclosure(high) -> private versioned
state, connector enforcement and restricted SQL roles; cost/availability(medium)
-> bounded disk/connections, deletion protection, retained backups and pending
capacity/budget/recovery acceptance. Actual effective IAM and restore are unproved.
Acceptance: mock-provider resource/negative input tests; disposable PostgreSQL
bootstrap positive/replay/foreign-name denial and runtime privilege checks; core,
migration and release regressions. No real cloud plan/state/data inspected.
