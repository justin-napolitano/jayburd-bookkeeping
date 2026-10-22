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

## Bounded listener composition — #103, 2026-10-22

Status: implementation candidate for coordinator/security review; no public
exposure or deployment is authorized by this package. `internal/webhookhost`
composes only the existing verified handler. `New(keys, receiver, close)` is inert;
`NewSQL(keys, db, close)` supplies the reviewed SQL receiver and requires a separate
restricted webhook pool. `Serve(ctx, listener)` takes ownership of one explicitly
provided listener, drains on cancellation and closes dependencies once. It does
not register bookkeeping, health, catch-up, OAuth or Item-lifecycle routes, discover
credentials, select a database principal or change scheduling. Even `OPTIONS *`
is sent through exact-route denial rather than the Go server's general response.

Local profile: header read2s, full request read5s, handler context8s, response write8s,
idle connection15s, header16KiB and shutdown drain2s. These are project resource
bounds, not a guarantee of a successful response or hosted end-to-end latency.
The existing handler's16-request admission,64KiB body, key bounds and SQL4s budget
remain active. A timeout never converts uncertain durability into success. Only
cooperative injected dependencies can be cancelled; socket close cannot stop
arbitrary blocked Go code. Response security headers contain fixed values, no
CORS capability, and no reflected request/error information. TLS and HTTPS-only
public forwarding require separate deployment review; HSTS alone does not supply
TLS. Server diagnostics are discarded to prevent raw request/panic disclosure;
redacted operational outcomes/metrics remain an explicit hosted follow-up.

### Evidence and alternatives

P88-14 (official [Webhooks API](https://plaid.com/docs/api/webhooks/), configuration,
retry and best-practices sections) and P88-15 (official
[verification procedure](https://plaid.com/docs/api/webhooks/webhook-verification/),
ES256, key metadata, exact-body SHA256 and JWT age) were re-read2026-10-22. Review
was limited to relevant public sections; no personalized Launch Center, provider
payload or credential was accessed. P88-14 requires a response within ten seconds;
local deadlines leave hosted timing as a separate acceptance measurement. Existing
proprietary-notification/ISO applicability above is unchanged; no financial
message, amount or accounting policy is added.

Go's official [`http.Server`](https://pkg.go.dev/net/http#Server) ReadTimeout,
ReadHeaderTimeout, WriteTimeout, BaseContext, DisableGeneralOptionsHandler and
Shutdown descriptions were read in full for these fields2026-10-22; matching
fields were checked in the installed Go1.26.5 standard library. The online page
is mutable (thenGo1.27.3), not the repository's pinned1.26.10 execution proof.
ReadTimeout bounds body reads, while WriteTimeout limits response writes; neither
alone supplies a cancellable durable receiver context. Our explicit context and
socket bounds are the inferred composition. Reusing `workerhost` was considered,
but its90s write budget is for private acquisition work and would weaken this
notification boundary. A fresh API mux was rejected because it risks exposing
private routes. Existing verification and SQL adapters remain the single business
implementation. `TestSignedSocketDelivery`, `TestSocketIsolation`,
`TestStalledBodyRead`, `TestShutdownCancelsReceipt` and
`TestHostConfigurationAndBounds` name the synthetic acceptance evidence.

### Threat review (STRIDE)

| Threat / local risk | Controls and required evidence |
| --- | --- |
| Spoofing, high | Existing ES256/kid/key-age/body digest verification; invalid socket requests never reach receiver; real provider delivery remains open. |
| Tampering/replay, high | Original bytes verified before existing atomic SQL receipt/generation wakeup; unknown durability503 and retry; SQL replay/concurrency integration remains separate evidence. |
| Repudiation, medium | Durable receipt is authoritative; no raw request logs; redacted hosted metrics and operator traceability need release review. |
| Disclosure, high | No API routes, request reflection, CORS or diagnostic logs; fixed errors/headers; TLS/secret/IAM provisioning remains unimplemented here. |
| Denial of service, high | Socket/body/header/context/admission/key limits; real stalled socket and cancellation tests; edge/network rate controls and load qualification remain open. |
| Elevation of privilege, high | NewSQL requires separately provisioned pool and existing actual-session restricted-role checks; never reuse API/sync principal. Hosted principal/role proof remains open. |

The applied Citizen Developer security skill requires human/security review before
release for public exposure, restricted data and generated security-sensitive
code. Repository GCP/Infisical authority overrides enterprise Azure product defaults;
no policy exception or production readiness is claimed. The root coordinator owns
the concrete managed-principal configuration review before executable composition.
