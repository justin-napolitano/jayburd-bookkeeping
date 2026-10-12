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

## BA03 — Firebase email verification and hosted actions

Reviewed 2026-10-12: [Manage users](https://firebase.google.com/docs/auth/web/manage-users),
“Send a user a verification email”; [JavaScript Auth reference](https://firebase.google.com/docs/reference/js/auth),
`sendEmailVerification` and `reload`; [email action handlers](https://firebase.google.com/docs/auth/custom-email-handler),
default-handler introduction. SDK methods send to the authenticated user and
reload provider account data; Firebase supplies a default hosted action handler.
The TOTP guide's prerequisites require verified email. Review covers these sections,
not the whole reference. Implementation remains pinned to Firebase 13.0.0.
Our inference: explicit send, provider-hosted links, and a fresh login after
verification avoid custom mail/URL handling. The 60-second UI resend interval is
our choice, not a documented provider quota. Live sender, inbox delivery and
expired-link behavior are unproven and require owner review.

## BA04 — Separate static rehearsal build

Reviewed 2026-10-12: [Vite build options](https://vite.dev/config/build-options)
(output directory, source maps) and [shared options](https://vite.dev/config/shared-options)
(root, public directory, environment directory and Oxc JSX configuration).
These configure a separate browser bundle; they do not provide authentication or
network isolation. The implementation pins the existing Vite 8.3.4 toolchain as
a direct dev dependency and disables public-file copying, environment-file loading
and source maps. Our Node server restricts serving independently. BA01–BA03 remain
the provider authority. Synthetic browser interception cannot prove real mail,
revocation or API-key restrictions; the owner performs that separate rehearsal.

BA04 operator setup also reviewed Google's [Identity Platform REST guide](https://docs.cloud.google.com/identity-platform/docs/use-rest-api),
“Before you begin”: Providers → Application setup details exposes the application's
`apiKey`. Only the owner copies that browser identifier into the rehearsal; no
Admin token/credential retrieval commands or REST login examples are executed.
