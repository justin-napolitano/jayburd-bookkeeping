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

Password-only success enters enrollment, never bookkeeping. Generate and display a
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
with credential-free fixtures. Live Identity Platform configuration, verified user
provisioning, enrollment/recovery, hosted origin/TLS and end-to-end MFA remain release
gates. Identity flows have no applicable ISO 20022 financial message; no ISO
conformance is claimed. Financial requests follow existing API/currency contracts.
