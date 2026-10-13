# Private observer transport

## Scope and decision

Implemented foundation for #7: wrap the existing [runtime adapter](observer-runtime.md) in
separate publisher/consumer HTTP handlers and a Cloud Tasks REST v2 transport.
SQL remains authoritative for scope, leases, fencing, recovery and receipts.
There is no financial payload or posting operation. The STD-01 internal-wakeup
assessment in [background work](background-work.md) applies unchanged; this is
not an ISO 20022 financial exchange or a conformance claim.

## Implemented boundaries

`NewPublisherHTTP(Ticker, TokenVerifier, CallerPolicy)` and
`NewConsumerHTTP(WakeupHandler, TokenVerifier, CallerPolicy)` return independent
`http.Handler` instances. Separate constructors receive an already configured Publisher or Consumer, an
injected Google ID-token verifier, and an exact audience/email/subject policy.
POST `/internal/observer/publish` accepts only `{}`; POST
`/internal/observer/consume` accepts the existing 1024-byte `work-wakeup-v1`.
Identity is verified before reading a body or invoking SQL. Caller headers never
supply an actor, book, binding, queue, URL or credential. One request per handler
can be active, with a 60-second overall budget and five-second verification.
A consumer acknowledges only successful committed Handle; unknown/busy outcomes
return 503. Protocol failures return 400/404/405; identity failure returns 401.
Publisher 200 exposes discovered/candidates/enqueue_calls/recorded/skipped only;
those counts describe this tick, never backlog completion. Consumer success is
204 only after Handle succeeds. No error response contains runtime diagnostics.

The queue adapter fixes project/location/queue, consumer URL, audience and
invocation service account at construction. It receives context-aware access
credentials and an injected HTTP RoundTripper; construction discovers no secrets.
One ten-second create attempt sends a deterministic task name and the unchanged
identifier envelope, with an OIDC token and 65-second dispatch deadline. It
never follows redirects or retries a create call. Queue acknowledgement is not
observation completion; uncertain responses retain durable SQL recovery.

`NewCloudTasksTransport(QueueConfig, AccessTokenProvider, http.RoundTripper)`
implements the existing Transport. AccessToken(ctx) must return a context-bound
short-lived OAuth access token with Cloud Tasks scope. Target URL must be the
fixed consumer path on HTTPS *.run.app; audience is the same origin, optional
root slash. No ports, userinfo, query, fragment or encoded path. The invocation
service account belongs to the queue project. Each enqueue validates the exact
existing envelope and recomputes its deterministic SHA-256 task identity before
credentials. Only 200/201 plus matching full task name yields accepted; 409 is
already_exists, 408/504 timeout, 429/5xx transient, other 4xx rejected. Redirects,
malformed/oversized responses and network failure are transient uncertainty.
No response body or token is logged. Successful create bodies are capped at
16 KiB; token strings at 8 KiB. No application-level create retry.

`NewGoogleTokenVerifier(http.RoundTripper)` uses the already pinned Google
validator with public Google JWKS only, five-second HTTP timeout, no redirects,
and a strict 64 KiB key-response limit checked using one sentinel byte. Only RS256 is accepted; the handler
requires exact issuer, audience, subject, email, boolean verified-email and
current issue/expiry timestamps with at most one-hour lifetime. One Authorization
Bearer value is required; X-Serverless-Authorization is rejected. Cloud Tasks
headers carry no authority.

Composition must supply separate restricted database logins and workload
identities, plus a worker http.Server with ReadHeaderTimeout5s, ReadTimeout5s,
WriteTimeout70s, IdleTimeout30s and MaxHeaderBytes16KiB. The normal API server
15-second budget is incompatible and is not reused. A five-second request-body
read deadline supplements server timeouts. Injected verifiers, credential
providers and transports must honor context; no claim is made that context
can interrupt arbitrary blocked io.Reader implementations. Constructors do not
load ADC or start a listener. No worker executable is delivered in this slice.

## Source review and alternatives

Reviewed 2026-10-13, starting with bibliography BW02/BW03. These are implementation
mechanisms, not hosted acceptance evidence.

- **OT01:** Google Cloud [service-to-service authentication](https://docs.cloud.google.com/run/docs/authenticating/service-to-service),
  service-account setup, acquiring/configuring ID tokens, and receiving requests.
  Cloud Run Invoker is separate from application caller authorization. Audience
  identifies the receiving service. X-Serverless-Authorization removes the
  signature before delivery to the container; this adapter requires verifiable
  Authorization and rejects alternate-header ambiguity. Actual hosted header
  behavior must pass before activation.
- **OT02:** Google Cloud REST v2 [tasks.create](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks/create),
  request body, deduplication, response view, OAuth scopes. Named-task conflicts
  can refer to recently removed tasks; full-name acknowledgement is never a
  receipt. The adapter requests BASIC view and never fetches financial content.
- **OT03:** Google Cloud REST v2 [Task and HttpRequest](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks),
  task identity, base64 body, OIDC authorization and dispatchDeadline. Cancellation
  does not guarantee stopped processing. The chosen 65-second queue deadline
  allows a small margin over the 60-second handler budget; SQL fences remain
  necessary. Queue-level overrides require a separate hosted configuration check.
- **OT04:** Pinned `google.golang.org/api/idtoken` v0.298.0, local `validate.go`
  constructor/Validate/RS256 implementation reviewed. It verifies signature,
  audience and expiry; application checks must additionally authorize issuer,
  service-account subject and verified email. No custom cryptographic verifier.

- **OT05:** Google Cloud [Token types, service account ID tokens](https://docs.cloud.google.com/docs/authentication/token-types#service_account_id_tokens),
  RS256 example and issuer/subject/lifetime fields reviewed. Pin exact issuer
  `https://accounts.google.com` and immutable numeric subject, not email alone.
  ID tokens cannot be individually revoked; current SQL authorization remains
  required. Email inclusion and IAM behavior still need hosted qualification.

- **OT06:** Google Cloud REST v2 [OidcToken](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/OidcToken),
  serviceAccountEmail and audience reviewed. The service account must share the
  queue project; enqueue needs actAs authority and an explicit audience avoids
  the target-path default. No IAM permission was provisioned by this work.

Inference: retain REST plus the existing verifier dependency rather than add a
Cloud Tasks SDK for one bounded create operation. An injected SDK could be a
future replacement if it preserves no automatic retry and context ownership.
Polling-only and Pub/Sub alternatives remain as reviewed in background work.

## Acceptance evidence

- `TestWorkerHTTPAuthentication`: missing/multiple/spoofed auth, wrong issuer,
  audience, subject/email, unverified email, expiry/future issue time and verifier
  failure invoke no body read or SQL; fixed errors contain no input/token text.
- `TestWorkerHTTPProtocol`: exact routes/method/query/content-type/body bounds,
  duplicate/unknown wakeup fields and publisher input; only committed success
  acknowledges; unknown/busy/configuration failures remain non-2xx.
- `TestWorkerHTTPAdmission`: one concurrent request, context cancellation and
  deadline propagation; no queue-generated header is trusted as identity.
- `TestGoogleVerifierSyntheticKeys`: real RS256 verification against synthetic
  injected JWKs, signature tamper, unknown keys/algorithm and safe failure.
- `TestCloudTasksWire`: fixed destination, deterministic name, exact base64
  envelope, explicit OIDC audience/account/deadline, no redirects/retries.
- `TestCloudTasksUncertainty`: duplicate acknowledgement, denial, throttling,
  server/transport timeout, malformed/oversized/lost response, wrong returned
  name and redaction. Each attempt obtains at most one token and makes at most
  one request. No outcome asserts durable work completion.

- `TestObserverHTTPComposition`: disposable PostgreSQL plus both HTTP handlers
  and fake REST transport; actual receipt COMMIT acknowledgement loss yields503,
  duplicate delivery yields204, then lost create acknowledgement retains one
  receipt/attempt and unchanged journal/outbox/book revision. No TCP loss or
  actual Google token service was exercised.
- `TestObserverHTTPMixedSyncRole`: added sync membership makes publisher HTTP
  return503 before discovering an operation. The adapter role guard now includes
  bookkeeping_sync alongside owner/API/other-worker capabilities.

Hosted Cloud Run IAM, actual Scheduler/Cloud Tasks identity/header behavior,
queue overrides, invocation account permissions, key rotation, deployment/pool
budgets, costs and latency remain separate acceptance gates.
