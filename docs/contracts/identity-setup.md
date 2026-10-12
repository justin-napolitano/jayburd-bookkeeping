# Identity and controlled setup

## Decision and scope

Implementation-review contract for #6, 2026-10-11. Use Google Identity Platform
and Firebase Admin Go SDK for managed end-user authentication. The SDK verifies
ID tokens with revocation checking on every request. Require TOTP MFA; bind exact
issuer and subject to an explicitly enrolled ledger principal, never email,
request actor IDs, tenant claims or custom permission claims. No automatic signup.
Application tests remain synthetic. Provider configuration is owned by the
[isolated identity root](../../infra/gcp/identity/README.md); actual setup and
remaining live proof belong in the [runbook](../runbooks/identity-setup.md).

Sources and alternatives are recorded in [identity sources](../research/identity-sources.md).
Self-managed passwords would add recovery and credential-storage responsibility;
a generic JWT verifier would still need provider revocation and lifecycle work.
The official SDK is selected despite its larger dependency graph.

## Authentication boundary

Only bearer ID tokens are accepted. The adapter requires an explicitly supplied
Admin client and project ID; it never loads credentials. The caller must construct that client without emulator mode; the SDK retains
its initialization mode. The adapter also rejects emulator environment presence
at construction and on each request.
Check SDK verification, issuer, audience, subject, expiry, issued/authentication
times, non-tenant scope and `firebase.sign_in_second_factor=totp` before mapping.
Verification/mapping has a five-second deadline and fails closed without logging
tokens or SDK errors. Provider revocation is checked at request time, not atomically
with database commit. Database principal/membership disabling still serializes
with posting. Queued work must reauthorize at execution; no worker is implemented.

Browser composition uses an explicit HTTPS origin. Reject foreign/opaque/duplicate
Origin headers and cross-site Fetch Metadata. Cookies confer no authority; no
cookie/session endpoint or permissive CORS is added. Future cookie-based UI sessions
require a separately reviewed CSRF/session contract. A bearer-token client must
not persist tokens in localStorage. TLS, redacted access logs, rate limits, SDK
workload identity/IAM and MFA enrollment/recovery require deployment evidence.

## Enrollment and setup SQL

Forward migration 000003 adds a distinct NOLOGIN `bookkeeping_identity_operator`
role. It may enroll an issuer/subject through `ledger.enroll_principal`, but may
not perform table DML or accounting operations. Runtime cannot enroll users.
Enrollment records the database session identity and is retry-safe for the same
issuer/subject; disabled identities cannot be silently reactivated.

`ledger.resolve_principal(issuer,subject)` exposes only an active mapped UUID.
The trusted Go adapter alone supplies these verified values. The runtime database
identity is a trusted service boundary, not a credential issued to end users.

`POST /v1/setup` creates one entity, book, policy, open period and 2–100 accounts
in a single serializable transaction for an enrolled user. It grants that creator
read/post permissions only. Existing entities cannot be claimed or modified here.
Body fields: `entity_name`, `entity_kind` (legal/personal), `book_name`,
`starts_on`, `ends_before`, `functional_currency_version_id`, `accounts`.
Each account has `code`, `name`, `category`, `normal_side`, `monetary`.
IDs are server-generated; no actor, membership, initial balance or journal input.
Development policy remains ADR 0005: accrual, UTC, USD/2, half-even, synthetic spot.
This restriction concerns functional policy; existing foreign transaction currency
support remains. Reference currencies are installed separately by the operator.

Require Idempotency-Key; scope is principal + setup operation. Strict unique-key
JSON, bounded text and arrays; array order remains meaningful. Canonical equality
uses validated jsonb with normalized currency UUID. Successful replay returns the
original 201 response only while principal/entity/read/post grants remain active.
Atomic immutable setup receipt/audit includes creator, request ID and result IDs;
setup has no asynchronous consumer and emits no speculative outbox event.
Unknown commit returns 503; retry the identical key/body. Invalid input is 400,
unsupported policy 422, conflicting key 409, disabled/inaccessible scope 404.

## Standards applicability and acceptance

These identity/setup operations are internal application controls, not ISO 20022
financial exchanges. Currency metadata references ISO 4217 under the existing
[standards contract](standards.md); no ISO designation or audit export is claimed.

`TestIdentityBoundary` covers MFA, issuer/audience/time/scope, provider rejection,
missing/deleted/disabled mappings and safe failures. `TestBrowserBoundary` covers
origin/cookie behavior. `TestControlledSetup` covers real-role enrollment limits,
atomic creation, posting from returned IDs, replay/conflict, invalid bundles,
revocation and rollback. `TestSetupConcurrentReplay`, `TestSetupCommitAcknowledgementLoss` and
`TestSetupRevocationRace` exercise real transaction failures/locks.
Existing ledger authorization/concurrency tests remain
required. Live provider verification and browser login are release gates, not
replaced by SDK-boundary test doubles.
