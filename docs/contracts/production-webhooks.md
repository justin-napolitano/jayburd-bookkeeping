# Production signed webhook isolation — #314

Coordinator engineering decision2026-10-26, before implementation. Source baseline
merged313/d0b141b; no deployment/resources, public exposure, credentials, provider
calls or activation. Human sensitive-source review required before merge. Actual
Sandbox signed/reconnect qualification remains303/320/275;315 independently tests
the integrated Production profile,337/316 approve resources and finite release.

## Profile and transport

Legacy constructors/omitted runtime metadata remain Sandbox. Explicit profile
constructors require a valid private plaidprofile value. Key HTTP client, verifier,
SQL adapter and managed receiver must agree before construction. Interface values
used for tests are injected capabilities; no caller body can choose a key URL or
environment. Production key fetch uses only production.plaid.com at fixed POST
/webhook_verification_key/get with API2020-09-14, bounded three-second/no-redirect/
no-retry transport and existing key-ID limits. Credentials resolve lazily through
two exact numeric jayburd-prod-bank-sync-client-* versions, not request paths.

Each verifier owns its profile-local cache; equal key IDs in different environments
share no state. ES256/P-256, canonical header/claims/signature, five-minute age,
future skew, key metadata, raw-body SHA256, duplicate JSON and size/concurrency/
rate budgets remain unchanged. Classify Transactions or modern LINK/SESSION_FINISHED
only after authentication; then require event.environment equals the runtime.
Unverified Item/Link/session identifiers never select tenant, credential or profile.

Managed Production receiver uses the separate jayburd-prod-webhook IAM SQL identity,
nondevelopment instance/project, exact Production static references and finite
approval. Invalid/mixed/expired config rejects before dependency construction.
Each outbound identity/secret/key request rechecks the same deadline; cached-key
events also deny after expiry. No bank-token/KMS/sync/post capability is added to
the receiver. Same fixed route/body/header/time limits; no infrastructure exposure.

Production Link notification configuration is optional for source construction,
but release requires a reviewed receiver. A nonempty URL must be exactly
https://jayburd-prod-webhook-PROJECT_NUMBER.us-central1.run.app/webhooks/plaid,
derived from explicit canonical nondevelopment project-number metadata. No arbitrary
HTTPS URL or browser selector. Hosted repair is a separate false-by-default opt-in
requiring this matching destination. Empty receiver is never release readiness.

## Migration35 SQL design

Keep one PostgreSQL authority and existing restricted bookkeeping_webhook role.
Extend webhook_ingress/webhook_receipt/link_session_receipt closed environment
constraints to sandbox/production. Add immutable webhook_login_profile(login_name
name PRIMARY KEY,environment text NOT NULL in that enum), backfilled from existing
ingress and immutable operator receipts, including retired pre35 Sandbox logins.
No runtime table grants. Operator-only existing set_webhook_ingress
signature binds a safe login permanently to one environment; serialize with the
existing advisory lock, preserve exact operator replay, deny reuse across profiles
even after disabling/replacing ingress. No automatic new ingress row or grant.
The active ingress row still selects one separately provisioned login per profile.

Existing verified receipt signatures and roles remain. Both accept functions must
check login safety, immutable profile and currently enabled ingress before mapping
anything. Transactions resolve Item only within verified environment, retain current
consent/source/principal/worker/namespace checks and bounded atomic wakeups. Production
additionally requires current enabled provider_scope with matching environment and
worker login, under the existing principal/book lock order. Receipt/demand commits
atomically; replay still reauthorizes, cannot duplicate demand or post a journal.
Login safety denies verifier/predefined-role membership and direct application-table
privileges in SQL as well as the Go adapter; role drift is not hidden by a prior
successful bootstrap. Disabling a known unsafe ingress remains possible.

Session lookup counts only exact stored Link hash AND environment. No match,
ambiguity, new-mode, claimed or terminal session is ignored without creating authority.
Only an owned reconnect uses existing receipt-lock/principal/book/worker/sibling/
source/consent checks. Require current provider_scope for Production. Authenticate
the attempt's exact environment throughout scope/claim/credential/handoff; derive
Item from the existing attempt, never a callback or generic login timestamp.
Production owned repair reuses the existing protocol, replacing Sandbox literals
only where reviewed: owned_repair_scope, proof-pinned claim environment, addressed
repair read and reconnect handoff key validation. Bank/Link AEAD uses313's explicit
Production profile/key preflight; no credential exchange/rotation/new version.
All existing ambiguity/expiry/revocation/paused-sibling/replay controls remain.

Owned scope internals stay private; API gets no receipt-ingress/table authority,
webhook gets no credential/claim/confirm/catchup/post authority. SECURITY DEFINER
search_path and PUBLIC revoke/selective grant remain in one transaction. Merged
SQL1–34 immutable.35 down refuses when Production ingress/login-profile/receipt or
repair-claim history exists; it preserves provenance and never deletes to downgrade.
Empty/Sandbox-only disposable down restores exact prior functions/constraints/ACLs.

## Sources, standards, alternatives and evidence

PP09 selected Plaid verification/header/key/age/raw-body and modern session fields,
plus PG17 safe-definer sections266–292 reviewed2026-10-26. Vendor facts establish
signature/session mechanisms, not our tenant/role policy or hosted qualification.
API2020-09-14/SDK49 unchanged. Operational proprietary receipt/AAD/routing has no
registered ISO20022 financial message; no money/currency/recognition mapping changes.
ISO4217 and ISO21378 applicability/access limits remain as reviewed. No conformance.

Rejected global environment fallback or body-selected key service: confused deputy
and profile mixing. Rejected a new SQL role hierarchy that would require duplicating
all role-safety checks; immutable environment-bound existing dedicated logins retain
the reviewed role. Rejected global cross-environment Link-hash lookup; it could
associate one environment's event with the other's attempt. No added database/service.

STRIDE: spoofing/tampering(high) -> exact signature/profile/raw bytes and matched
key transport; cross-tenant escalation(high) -> current SQL scope, owned hash and
dedicated immutable login profile; replay(high) -> atomic receipt/demand and immutable
claim; disclosure(high) -> projection-only SQL, no raw secrets/body logs; outage/cost
(medium/high) -> bounded cache/request/expiry, unknown commit retry semantics unchanged.
Residual actual delivery/IAM/isolation/retention remains unproved until hosted gates.

Acceptance: TestProductionWebhookEnvironmentDeny; explicit-key-host/cache mismatch
tests; real restricted SQL signed Transactions duplicate/reordered/current-consent/
unknown-Item/revoked-scope cases; owned Production Link proof/claim/credential and
wrong-profile/session replay; API/worker/receiver cross-role denial;35 fresh/replay/
down/forward and existing Sandbox regressions. Core, migration, ledger, webhook,
bootstrap, connection/credential/recovery, locked web and app-integration gates.
