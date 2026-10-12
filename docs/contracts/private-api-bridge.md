# Runtime identity and private API bridge

## Decision

Issue #27 application slice, 2026-10-12. One standalone frontend image reads public
identity configuration at request time and forwards a narrow set of bookkeeping
requests to an operator-selected API. Go remains the authority for signature,
revocation, MFA, principal enrollment and current entity permissions.
Sources G16/G18 in the [GCP register](../research/gcp-sources.md) support the
separate-token and runtime-configuration mechanisms. Synthetic transport tests do
not prove actual Cloud Run IAM or Identity Platform configuration.

## Runtime configuration

`GET /api/auth-config` returns only `apiKey`, `authDomain`, `projectId` from
`JAYBURD_FIREBASE_API_KEY`, `JAYBURD_FIREBASE_AUTH_DOMAIN`,
`JAYBURD_FIREBASE_PROJECT_ID`. These are public identifiers, not credentials.
Only the selected project's `<project>.firebaseapp.com` domain is supported in
this slice; custom domains require explicit review. Partial/invalid settings return
503 and leave sign-in unavailable. No environment dump or provider call occurs.
Responses use `Cache-Control: no-store`; the browser fetches once per provider
mount, omits cookies and abandons late results after unmount. Existing in-memory
MFA/session behavior remains. Remove old NEXT_PUBLIC_FIREBASE_* build settings;
they no longer configure this application. A changed configuration requires a new
page load; an existing Firebase app with different identifiers fails closed.

## Transport and authority

Set `BOOKKEEPING_API_MODE=cloud-run`, `BOOKKEEPING_API_ORIGIN` to the exact HTTPS
Cloud Run service origin and `BOOKKEEPING_PUBLIC_ORIGIN` to the exact HTTPS browser
origin. No custom API domains, URL credentials, path/query/fragment or same-target
self proxy are accepted. The backend origin must end in `.run.app`; its value is
also the ID-token audience. No trust is inferred from incoming Host/Forwarded.

The bridge preserves the end-user `Authorization` bearer and fetches a separate
service ID token from Google's fixed metadata endpoint. It sends that token in
`X-Serverless-Authorization`. Incoming service-token, actor, Cookie and forwarding
headers are discarded. Only method, reviewed path/query, user bearer, configured
origin, JSON content type and idempotency key reach the API. Redirects are never
followed. Ambient credential-file, emulator or metadata-host override presence
fails closed; no ADC or service-account key discovery is used.

Cloud Run must separately grant the frontend identity Invoker on only the private
API. The API must validate the end-user token and entity rights on every operation.
The bridge's JWT-shape check is not authentication. No token exchange, identity
claim fabrication, financial cache, write retry or new accounting logic is added.
Frontend concurrency does not replace deployment abuse/rate controls.

Explicit local use requires `BOOKKEEPING_API_MODE=loopback` and distinct HTTP
localhost/127.0.0.1 API/public origins. This still forwards a bearer to the same
Go authorization boundary; it provides no fake identity. Loopback mode is rejected
when Cloud Run's K_SERVICE is present. Missing mode is unavailable, never fallback.

## Bounds and responses

Only workspace, setup, journal posting/read/reversal, dated trial balance and
reviewed durable-command registration/pending/execute/cancel/acknowledge routes are
forwarded. Registration requires an idempotency key; command transitions use their
durable command ID as specified by the existing recovery contract. Close/admin/unknown routes, extra queries and foreign browser origins
are rejected before metadata access. Limits: 32 requests per frontend process,
1 MiB request/response, 16 KiB bearer/metadata token, 4 KiB path/query, 12-second
combined deadline and 3-second metadata deadline. Caller disconnect cancels work.
Buffers are byte-bounded even for streams of tiny chunks. No tokens are cached.

Successful bounded JSON response bytes retain exact decimal strings and receipts.
Error responses retain supported status semantics but replace provider body and
headers with a fixed problem response; cookies and credentials never return to the
browser. Uncertain/lost posting responses remain 5xx, allowing the existing UI to
retry the unchanged request key only after review. No server-side write retry occurs.
All bridge responses are no-store. No request/provider logging is introduced.

## Evidence and next gate

`private API bridge` tests cover separate tokens, header allowlist, route/origin
and configuration denials, lost responses, metadata failures, concurrency,
cancellation and bounded bodies. Public configuration and AuthProvider tests cover
allowlist, unavailable settings and late-result disposal. The container proof
restarts one image with synthetic runtime identifiers and verifies both states.
Existing Go/ledger suites remain the entity/revocation authority, not mocked 401s.

Terraform Invoker grants, runtime environment wiring, real MFA, cross-entity and
revoked-user cloud proof, frontend request logging review, load and rollout/rollback
remain #27 deployment gates. This slice performs no cloud apply or live identity
request. No financial message semantics change; existing ISO 4217/accounting
contracts apply. Runtime authentication/configuration has no applicable ISO 20022
business message and makes no new ISO conformance claim.
