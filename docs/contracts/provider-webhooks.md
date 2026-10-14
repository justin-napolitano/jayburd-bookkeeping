# Verified provider webhook ingress

## Reviewed scope

Issue #103 implements a separately injected handler for POST `/webhooks/plaid`.
It is not mounted on the bookkeeping API and does not deploy public ingress.
Only Sandbox `TRANSACTIONS` / `SYNC_UPDATES_AVAILABLE` is accepted. This proprietary
notification is not an ISO20022 business message; it carries no accounting amount.
Existing currency/accounting contracts remain unchanged.

## Verification and durability

Require one Plaid-Verification JWT <=8KiB and exact JSON body <=64KiB. Verify only
ES256 using the pinned maintained golang-jwt implementation, strict duplicate-free
JSON, canonical base64url framing, integer issued-at no older than300seconds and
no more than60seconds in the future, and constant-time comparison of the signed
SHA256 with original bytes. Token header URLs/embedded keys are unsupported.

Keys come from an injected provider fixed to Sandbox webhook_verification_key/get,
with no redirects/retries and3second/16KiB limits. Metadata must identify the exact
kid, EC/P-256/ES256/signing use, valid curve coordinates, and creation/expiry times.
Cached expiry is checked on every verification. Local policy: four keys,60second
TTL, four cache misses per minute and one lookup at a time; no stale fallback.
These caps are project choices, not provider guarantees. Unknown or rotated keys
can temporarily receive503 until the bounded lookup budget resets.

After signature/hash verification, strictly validate the approved body fields.
Initial/historical booleans describe the notification, not local sync completion.
The receiver commits a durable receipt keyed by SHA256 of the canonical signed delivery
plus currently authorized requested-generation/wakeup intents atomically before204.
Unknown/disabled/unbound Items receive a uniform denial. Invalid signatures never
reach SQL. COMMIT uncertainty returns503; retrying the same verified delivery
resolves its receipt without repeating generation increments. After normal library
verification, replay identity retains exact header/payload segments and R, but uses
S=min(S,N-S) in the signature segment. This local identity normalization prevents
ECDSA high/low-S equivalents creating separate receipts; it is not new signature
verification. A new signed delivery
with identical body is not permanently suppressed by body-hash deduplication.

Ingress has16 concurrent requests and10second context budget. Hosting must set
bounded socket read/write timeouts; arbitrary injected blocked readers cannot be
made context-cancelable by this package. No token/body/request/provider error logging,
credentials in SQL, transaction acquisition, posting or financial readiness claims.

## Catch-up and SQL ownership

Separate restricted webhook and sync pools are injected. Short serializable
transactions retry at most three confirmed40001/40P01 rollbacks within4seconds;
cleanup gets an independent second. Unknown COMMIT exposes no result and triggers
no blind retry. Catch-up accepts trusted entity/book scope, stable tick UUID,
maximum1–100 and optional canonical stream cursor. SQL reauthorizes each enrolled
stream and commits the same wakeup intent. No scheduler or provider call is started.

## Source evidence

Reviewed official [Plaid webhook verification](https://plaid.com/docs/api/webhooks/webhook-verification/)
key metadata, ES256 procedure, issued-at age and exact-body digest sections (PI04).
Reviewed [Transactions webhooks](https://plaid.com/docs/api/products/transactions/#sync_updates_available)
SYNC_UPDATES_AVAILABLE fields, optional user identifier and Sandbox environment
(PI01). Pinned golang-jwt/v4.5.2 `ecdsa.go` and `parser.go` were read locally:
SigningMethodES256.Verify uses standard-library ECDSA with JWS R/S framing.
Strict parsing and explicit project clock policy surround that maintained verifier.
Alternative public full-API ingress or JWT-selected key URLs is rejected because
neither is required for this narrow notification authority.

The [JWA ES256 signature representation](https://www.rfc-editor.org/rfc/rfc7518#section-3.4)
(section3.4) defines fixed-width R/S encoding. Replay identity normalization is our
local inference for ECDSA's equivalent S values, not an RFC-mandated rejection rule.
Original high-S and low-S signatures remain accepted after library verification.
