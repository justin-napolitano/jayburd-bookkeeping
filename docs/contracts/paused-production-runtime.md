# Paused Production runtime and credential readback — #370

Decision candidate2026-10-30, before implementation. Reuse the existing Go/
PostgreSQL/Next architecture, owner-selected shared instance and bookkeeping_prod.
Read production-runtime.md, shared-production-sql.md, private-api-bridge.md,
identity-setup.md, existing runtime Terraform, connectionruntime.validateProduction/
New, providersecrets.GCPReader.Read and workerruntime.MetadataTokens. Selected
repository sections and existing GCP/identity source records in the bibliography;
no new vendor protocol or complete SDK audit. The current runtime templates derive
Sandbox resources, and disabled connection configuration cannot carry a Production
database environment. An unqualified rename could route to bookkeeping instead.

Use a separate production-runtime root, null by default, exact known project pair,
separate services/identities/backend, immutable Production api/web/worker images,
numeric static versions and public Firebase browser identifiers only. Production
API revocation verification needs firebaseauth.users.get, matching the reviewed
Sandbox lookup role; add a separate one-permission Production role/binding in the
foundation rather than broad Firebase administration or cross-state role reuse.

The API uses the existing explicit Production expired-recovery composition:
connections-enabled plus all matching Production metadata, with fixed
2020-01-01T00:00:00Z provider expiry. This selects the isolated database and keeps
authorized SQL bookkeeping/recovery available, but constructs no secret/KMS/provider
dependencies. It is deliberately expired authority, not a false pilot approval.
There is no activation-date input in this root. Webhook, workers, queues, schedulers
and live enrollment remain outside this slice. Alternative: modify the legacy root
or add a new Go pause mechanism; rejected to preserve Sandbox state and reuse the
already tested expired-recovery behavior. Initial rollback remains stop/forward
repair; there is no accepted previous Production application revision.

Private API permits only the Production frontend service account at resource IAM;
public frontend is the existing sign-in shell, with verified MFA/current database
authority still required for finance. Min0/max1, CPU1/memory512Mi, idle CPU billing
and no startup boost bound capacity without claiming a USD25 spending cap. No
database instance/DDL or authentication user provisioning is introduced. Firebase
public identifiers must come from reviewed public SDK configuration, never an
environment/credential dump. Existing Go revocation/MFA/origin checks remain intact.

Independent /credential-check job uses the existing Production API identity and
GCP read grants, exact two static names/numeric versions and metadata-only Google
authentication. The job inherits that identity's existing SQL/KMS/identity authority,
although its fixed command constructs none of those clients; no broader grants
are added for it. Explicit execute, canonical future UTC deadline within7days and
known project pair are mandatory before I/O. It reuses GCPReader name/CRC32C/payload
bounds, clears returned byte buffers and prints only fixed status. No source read,
secret write, SQL, Plaid call, payload/length/hash diagnostics or static credential.
Null runtime creates no job; configured job has one task, no retries/scheduler or
new invocation/impersonation grant. Success proves exact runtime readback/integrity,
not source equality, freshness, accepted Plaid credentials or pilot readiness.

ISO20022 has no applicable message for operational runtime/readback metadata;
ISO4217/financial contracts unchanged. STRIDE: elevation/disclosure(high) -> fixed
project/resources, private API, single lookup permission, fixed output and memory-only
payloads; tampering/replay(high) -> image digests/numeric pins and existing checksums;
availability/cost(medium) -> inert default, bounded capacity, finite job, no workers;
expired authority(high) -> permanently expired bank composition and pre-I/O job
deadline. Human security review before applying IAM, public shell, runtime or job.

Acceptance: TestCredentialCheckInvocation and TestReadbackPins; offline exact-image
checks; mocked inert/default, Production args/identity/origin, private API/public
shell, capacity, immutable images/numeric versions and single-permission role;
core and infra gates. Actual source/scan/schema/IAM/SQL/auth/readiness/private owner
enrollment remain required measured evidence after reviewed apply. No default test
performs cloud/provider calls, and no bank/accounting mandate is silently waived.
