# Browser authentication

## Decision and boundary

Implementation contract for the first app, 2026-10-11. Use modular Firebase Web SDK
with memory-only persistence and operator-provisioned, verified email/password
accounts. Source review and alternatives: [browser sources](../research/browser-auth-sources.md).
No signup, anonymous mode, emulator, demo authentication, cookies or persistent token
storage. Public runtime identifiers are `JAYBURD_FIREBASE_API_KEY`,
`JAYBURD_FIREBASE_AUTH_DOMAIN`, `JAYBURD_FIREBASE_PROJECT_ID`, delivered through
`/api/auth-config`; missing/invalid configuration leaves sign-in unavailable and
does not initialize the SDK. See the [runtime/bridge contract](private-api-bridge.md).
These public identifiers are not server credentials.

## Session flow

Verified password-only success enters enrollment, never bookkeeping; unverified
users follow the email-verification flow below. Generate and display a
manual authenticator key only after explicit action; clear it on enrollment completion,
cancellation, logout or unmount. Enrollment always ends in signout. A new password
sign-in must complete the managed TOTP challenge. Only one existing TOTP factor is
supported; multiple factors or non-TOTP factors fail closed pending operator recovery.
No provider error text, password, token, setup key or OTP enters logs or application
storage. Inputs clear on submission/phase change. The SDK necessarily retains its
refresh token in memory; browser compromise remains outside this guarantee.

`AuthProvider`, `LoginPanel`, and `useAuth()` expose `phase`, `getToken()`,
`signOut()`, `rejectSession()` and `sessionKey`. Consumers call `getToken()` per API
request, call `rejectSession()` on backend 401, and discard financial state whenever
`sessionKey` changes or phase leaves `signed-in`. SDK token expiry/refresh errors
invalidate the session. A token returned after logout is discarded; cancelled SDK
operations cannot restore UI state and finish with another SDK signout. Browser
claim inspection is a UX gate only: Go validates signature, audience, issuer,
revocation, TOTP claim and enrolled principal on every protected API request.

## Acceptance and release gaps

Named Vitest suites `AuthController` and `FirebaseAdapter boundary` cover the above
with credential-free fixtures. Identity Platform configuration is applied (see
the identity runbook); verified user provisioning, enrollment/recovery, hosted
origin/TLS and end-to-end MFA remain release
gates. Identity flows have no applicable ISO 20022 financial message; no ISO
conformance is claimed. Financial requests follow existing API/currency contracts.

## Email verification before enrollment

Owner-directed implementation, 2026-10-12; sources BA03. An unverified successful
password login enters `verifying-email`, never enrollment or `signed-in`.
Only explicit **Send verification email** calls the SDK for its current user.
There is no recipient argument, automatic send, signup or admin verification flag.
Success means the SDK accepted the request, not proven inbox delivery. Provider
errors are replaced with a generic message; an in-memory 60-second interval
between attempts reduces accidental resends, including failures. This is UX
throttling, not an abuse-control guarantee; Google quotas still apply.

Use Google's default hosted action handler without custom continuation URLs.
Jayburd never receives/parses/stores verification links or action codes. After
opening the email, the user returns and clicks **I have verified my email**.
Reload provider user data: false stays verification-only; true signs out and
requires fresh credentials before TOTP enrollment. Failed reload signs out.
Cancellation invalidates late completions; no verification operation grants
bookkeeping access or invokes ledger APIs. Refresh loses the memory-only session.

Acceptance: `FirebaseAdapter boundary` tests send/reload and absent/unverified
user denial; `AuthController` tests phase isolation, resend throttling, pending
and verified refresh, failure and cancellation; `LoginPanel` tests explicit send,
status and fresh-login transitions. Existing browser and Go verification gates
remain required. This internal identity flow has no ISO 20022 message mapping.
Alternative custom email handlers/mail infrastructure add action-code handling
and delivery responsibilities without benefit to this slice, so are deferred.
Real inbox delivery, link expiry/reuse, sender configuration and TOTP enrollment
require owner evidence; synthetic tests do not establish them.

The [local identity rehearsal](identity-rehearsal.md) reuses this component through
validated public configuration without mounting the financial application. It is
an explicit operator tool, not an application route or a substitute for Go/SQL
identity acceptance. Its browser-only success screen never grants ledger access.
