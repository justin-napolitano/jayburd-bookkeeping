# Browser authentication sources

Reviewed 2026-10-11, relevant official documentation sections; no live provider
requests or credential inspection. Contract: [browser authentication](../contracts/browser-auth.md).

## BA01 — Firebase TOTP for the Web

[Google TOTP guide](https://firebase.google.com/docs/auth/web/totp-mfa),
“Before you begin”, “Enroll users in TOTP MFA” and “Sign in users with a second
factor”. Identity Platform and verified email are prerequisites. Enrollment uses
a fresh first-factor session; sign-in completion uses a managed resolver and TOTP
assertion. Documentation treats enrollment and sign-in separately and does not
establish that enrollment alone produces the backend-required second-factor claim.
Our inference: after enrollment, sign out and require a new password/TOTP challenge.
Manual setup-key entry is supported; no remote QR service is used. Recovery remains
operator-owned. Multiple pre-existing TOTP factors are rejected in this first UI;
operator recovery or a future explicit factor picker is required.

## BA02 — Firebase authentication persistence

[Google persistence guide](https://firebase.google.com/docs/auth/web/auth-state-persistence),
“Supported types of Auth state persistence” and persistence modification. Memory
persistence clears state on refresh; browser default local persistence would survive
window closure. Decision: initialize Auth directly with `inMemoryPersistence`;
no cookie session or local/session storage is used. This sacrifices persistent login
for a smaller session surface. Clearing JavaScript references is not secure memory
erasure and does not protect a compromised browser.

## Acceptance and limits

`AuthController` and `FirebaseAdapter boundary` Vitest suites exercise managed-SDK
boundaries with synthetic responses: memory configuration, verified invitation,
TOTP-only tokens, enrollment isolation, cancellation, failure redaction and stale
session suppression. They do not prove project configuration, recovery controls,
provider availability or end-to-end MFA. Those require an approved synthetic
provider release test before deployment. Google Go verification remains authoritative.
