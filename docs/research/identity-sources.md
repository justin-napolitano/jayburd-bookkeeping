# Identity sources

Reviewed 2026-10-11; relevant documentation sections only, not a compliance review.

## I01 — Google, Verify ID Tokens

[Firebase Admin verification guide](https://firebase.google.com/docs/auth/admin/verify-id-tokens),
sections “Verify ID tokens using the Firebase Admin SDK” and claim validation.
Evidence: ID-token verification checks signed identity; ordinary verification does
not check revocation. Project-specific audience/issuer and token times matter.
Inference: use the Go SDK, with explicit project binding; do not build JWT crypto.

## I02 — Google, Manage User Sessions

[Session guide](https://firebase.google.com/docs/auth/admin/manage-sessions),
“Detect ID token revocation”, including SDK examples.
Evidence: revocation checking adds a backend request; refresh tokens expire after
deletion/disabling and certain account changes. Inference: check every request,
fail closed on provider failure; do not promise revocation atomic with SQL commit.

## I03 — Google, Identity Platform TOTP MFA

[TOTP guide](https://docs.cloud.google.com/identity-platform/docs/admin/enabling-totp-mfa),
enablement and enrollment/sign-in sections. Managed TOTP requires Identity Platform
configuration and user enrollment. Inference: require TOTP-completed tokens for
bookkeeping; enrollment/recovery must be proven before a remote release.

## I04 — Firebase Admin Go API and source

[Auth API](https://pkg.go.dev/firebase.google.com/go/v4/auth), `Token`,
`VerifyIDTokenAndCheckRevoked`; implementation pinned to v4.22.0 for this slice.
Reviewed downloaded v4.22.0 `auth/auth.go` Token/verification methods and
`auth/token_verifier.go` verifyHeaderAndBody: the reserved firebase claim remains
in Claims, while emulator mode bypasses signature verification. SDK owns signature/key rotation; our tests prove the adapter boundary,
not Google's service behavior. Default commands must not initialize provider
credentials or make provider requests.

Acceptance and alternatives: [identity/setup contract](../contracts/identity-setup.md).

## I05 — Go release and vulnerability evidence

[Official Go downloads](https://go.dev/dl/?mode=json) lists supported Go 1.26.9.
[Firebase releases](https://github.com/firebase/firebase-admin-go/releases/tag/v4.22.0)
records the selected SDK release and dependency maintenance. Reviewed release
metadata and local govulncheck v1.1.4 results. The initial scan found reachable
Go 1.26.5 and x/net v0.55.0 vulnerabilities; select patched 1.26.9/x/net v0.60.0
and refresh affected transitive pins. These are software maintenance decisions,
not an accounting-standard change. Final scan outcome belongs in the runbook.
