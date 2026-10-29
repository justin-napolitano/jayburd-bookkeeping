# Shared-instance Production SQL — #316/#337

Owner selected2026-10-29: reuse jayburd-bookkeeping-dev:us-central1:jayburd-dev
with a new bookkeeping_prod database; existing bookkeeping remains Sandbox.
Actual metadata readback: PostgreSQL17, ZONAL db-g1-small,10GiB; no Production DB
exists. No resize, second instance or Sandbox-data copy requested.

PostgreSQL17 roles are cluster-wide; CONNECT is checked at session startup and
PUBLIC receives it by default. See selected official role/privilege source review
in production-pilot-sources.md. Sharing the existing safe NOLOGIN capability names
is possible only with independently enforced database CONNECT boundaries and
separate login identities. Database ownership must stay with an administrator,
not the shared bookkeeping_owner capability. No dblink/FDW bridges are introduced.
Shared instance maintenance, capacity, administrator access and restore scope
remain accepted tradeoffs requiring actual configuration/recovery verification.

Initial implementation is deliberately migration-only: explicit -database=bookkeeping_prod,
-environment=production, exact existing instance and finite approval deadline.
Legacy/omitted database retains bookkeeping. Other database strings and mixed
instance/profile combinations fail before I/O. Runtime Production paths retain
separate-instance restrictions until independently updated and tested; this slice
cannot connect real bank accounts. No arbitrary DSN/login or metadata redirect.

Runtime follow-up under316: explicit production environment plus the exact
owner-selected shared instance derives bookkeeping_prod for api/sync/publisher/
webhook only, together with dedicated jayburd-prod identities. Sandbox derives
bookkeeping and jayburd-dev identities. No free-form database option is added;
migrator/verifier cannot use this runtime exception. This source change does
not provision worker grants or enable provider access. Existing catalog isolation
proof and reviewed workload activation remain required. Acceptance:
TestSharedRuntimeRouting, TestProductionIAMProfile, core gates and synthetic
application integration; hosted acceptance remains a separate release stage.

Shared bootstrap verifies current_database and exact prod api/migrator login names
in the existing project, administrator-owned database, and absence of Production
login CONNECT rights to Sandbox. It uses the existing reviewed bootstrap statements in a separate wrapper, then
checks Sandbox and shared capability roles lack Production CONNECT. All checks
and grants must be atomic: wrapper uses one transaction for all guards and grants; original bootstrap and
merged migration files remain unchanged. Failure rolls back grants. Existing
Sandbox roles/data must remain unchanged. Actual live catalog readback, isolated
login attempts and immutable artifact hashes precede managed migration execution.

Alternative: separate instance gives stronger admin/recovery isolation but adds
standing cost. Owner selected shared instance; no mere removal of environment
checks. STD-01 operational deployment metadata has no ISO20022 message mapping;
financial schema/currency semantics remain unchanged. STRIDE elevation/tampering
(high): exact targets, separate logins, CONNECT fences and atomic bootstrap;
availability(medium): shared capacity/maintenance, pending measured proof. Human
security review required before new shared-instance code is used live.
