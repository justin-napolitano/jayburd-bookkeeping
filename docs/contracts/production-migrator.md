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

## Manual image publication companion

The same branch adds build-production-images.yml, matching354's pinned workflow
identity. Like the existing development builder, it builds once, tests exact local
images including worker/webhook commands, scans HIGH/CRITICAL vulnerabilities and
publishes without rebuilding. Dedicated jayburd-prod registry/builder/WIF names
prevent development federation reuse. No deployment is part of publication.

Owner-only main workflow_dispatch takes the SHA256 of a separately committed
public deploy/production-build-target.json. Its bounded shared-project candidate
is recorded in [operations](../runbooks/production-build-operations.md); exact
target review is separate from full337 pilot approval. Closed
schema: schemaVersion1, project, projectNumber, expiresAt, mandate (337 issue-comment
URL). Canonical compact JSON plus newline; nondevelopment target, numeric project
number, UTC expiry at most seven days. The owner must review the target and supply
its exact hash; the URL is a reference, not proof of review. No target/schema or
credential contents may be supplied via free-form workflow inputs. Verify target
before build and again before cloud auth and push, and bind its hash in artifacts.
A missing/expired/mixed target fails before auth. Existing dev workflow unchanged.

Candidate receipt matches354's closed contract. Supporting evidence separately
retains exact tested image ID/build provenance and target hash, plus original
scan/SBOM/build metadata. Worker runtimeTests means existing inert/rejection tests
for worker and webhook, not live ingress qualification. Receipt generation checks
matching Buildx/local identity, one registry digest and no HIGH/CRITICAL scan
findings; successful trusted workflow establishes runtime-test step execution.
Source ancestry, authenticated collection and four-receipt assembly remain pending.
The future Production WIF provider must independently restrict this exact workflow
and owner/main identity; YAML guards are not a substitute for effective IAM.
