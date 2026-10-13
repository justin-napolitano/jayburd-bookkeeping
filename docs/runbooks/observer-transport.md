# Observer service transport
<!-- stage-runbook: true -->

## Status and boundary

Implemented the [transport foundation](../contracts/observer-transport.md) for
#7 under lease WORKER-HTTP-7. Separate authenticated HTTP handlers wrap the
existing publisher/consumer. Cloud Tasks REST creation and Google signature
verification receive injected transports/credentials. No cloud/provider calls,
deployment, credential discovery, SQL migration or accounting mutation. No
worker executable or hosted authentication acceptance is claimed.

## Stage ledger

- Read current worker/runtime contracts, issue #7 and background-work sources.
- Reviewed Google service authentication, token types and REST v2 task creation;
  OT01–OT06 record reviewed sections and alternatives in the contract.
- Coordinator approved exact routes/caller policy, run.app-only destinations,
  60s handler/65s dispatch budget, injected credentials and no runtime activation.
- Implemented strict auth-before-body, one request admission, fixed response
  vocabulary, exact envelope/digest validation and no application create retry.
- Added bookkeeping_sync to the existing adapter's mixed-capability rejection
  under coordinator's explicit scope extension; no SQL grant changed.
- Independent review found a valid-JWKS-plus-padding response could accept a
  truncated prefix. Replaced truncation with strict max-plus-one rejection and
  an exact65536/65537-byte regression; reviewer verified the fix with no blockers.
- Added synthetic cryptographic, HTTP, queue/credential fault and cancellation
  tests, followed by actual PostgreSQL HTTP-to-HTTP composition proof.

## Current evidence

2026-10-13:

- `make check test build`: PASS, including default synthetic checks.
- `go test -race ./internal/observer -count=1`: PASS after transport fault cases.
- `make observer-test`: PASS against disposable PostgreSQL 17.11 (first transport
  run:42 groups; final expanded suite:44 top-level groups). The Docker
  sandbox credential-helper failure was rerun through approved escalation.
- `TestObserverHTTPComposition`: real restricted-login publisher and consumer
  run through their HTTP handlers and fake Cloud Tasks REST. Actual receipt
  COMMIT reply is dropped: first consumer response503, two duplicates204.
  Queue create reply is also dropped. Result remains one observed receipt and
  one attempt, original one journal/outbox, unchanged book revision; second
  publisher tick makes no additional queue call.
- `TestObserverHTTPMixedSyncRole`: mixed sync capability returns503 before
  discovery and creates no operation. Fixture-only role creation supports both
  pre-sync and integrated migration sets.
- Unit tests use generated synthetic RSA keys and fake JWKS/REST responses;
  they do not fetch public keys or call Google. No token/body/error text escapes.

## Remaining work

A separate executable/composition must inject scoped workload credentials,
restricted independent pools, exact caller subject/email/audience and required
HTTP server deadlines. The existing public API is unchanged. Hosted Cloud Run
IAM/OIDC/header behavior, queue overrides/limits, scheduled activation, actual
service role enrollment, key rotation and cost/latency proof remain separate
acceptance gates. Identity tokens do not replace current SQL scope authority.
No user/provider ingestion or automatic posting is introduced.

## PR handoff

Coordinator owns shared composition, bibliography/index and CI. New tests are
picked up by existing observer-test; no script/Makefile/go.mod changes. No
migration. Independent review resolved the key-response limit finding and reported no
remaining blockers. `git diff --check` and staged runbook guard passed.
