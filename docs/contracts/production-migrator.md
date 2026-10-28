# Production migration command — #316 stage C

Coordinator source decision2026-10-28 before implementation; source review pending.
Reviewed cloud-sql.md, production-runtime.md, internal/migration/migrate.go and
existing metadata-only connector at217b2ac. Existing GCP source register and
PP08–10 establish IAM/transport mechanisms; profile/expiry controls below are
project policy. No new vendor API, SQL or financial message. STD-01 operational
migration configuration is not an ISO20022 financial exchange; existing currency
and audit-standard applicability/access limits remain unchanged.

## Explicit profile and authority

cmd/migrate adds -environment=sandbox|production; omitted stays Sandbox.
Production status/up requires the exact instance suffix :us-central1:jayburd-prod,
nondevelopment project and -approval-expires-at canonical UTC within seven days.
It derives jayburd-prod-migrator@PROJECT.iam, never an arbitrary caller login.
Sandbox migration rejects that Production instance suffix; existing dev targets
and default offline verify remain compatible. Offline verify accepts no instance
or expiry and never constructs a connector. Invalid flags/profile/target/expiry
fail before SDK or SQL I/O; credential/environment override rejection is preserved.

Live command context ends at the earlier of ten minutes and Production expiry.
The connector and every migration/status operation receive that context. This is
an execution deadline, not cancellation proof for already committed DDL; actual
status/checksums must be inspected after interruption. Existing migration SQL,
restricted owner-role checks, advisory locks, dirty-state handling and status
read-only transaction are unchanged. No down/force/version-target command exists.

The flag does not prove approval: only a separately reviewed deployment workflow
may supply it alongside exact artifact/config hashes and intended IAM identity.
No workflow, resource, login or grant is created here. Owner bootstrap remains
separate and must establish the existing safe NOLOGIN owner/deployment capabilities
before managed migration. Actual337 approval and316 release gates still apply.

Alternative: reuse dev identity/automatic environment inference. Rejected because
it obscures authority and risks cross-profile DDL. A separate migration binary
would duplicate the checksummed execution engine without improving this boundary.
STRIDE: elevation/tampering(high) -> explicit profile, derived login, unchanged
restricted SQL gates; expired authority(high) -> pre-I/O checks and bounded context;
disclosure(medium) -> fixed errors, identifiers only; interruption(medium) -> status
and dirty-state repair, never blind retry. Human security review before merge.

Acceptance: command tests assert zero connector calls for offline/invalid requests,
closed profile/expiry inputs and context deadline; connector tests assert exact
Production login/pool/SQL timeouts and wrong-target denial. Core, migration-test
and existing Sandbox/Production release regressions required. These synthetic
checks do not establish live IAM/SQL bootstrap compatibility or apply migration35.
