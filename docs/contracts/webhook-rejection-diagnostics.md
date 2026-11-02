# Fixed webhook rejection diagnostics — #385

Decision candidate2026-11-01. The genuine supported-event test returned401;
test request and ordinary sync completed, but signed acceptance remains FAILED.
No earlier pre-sync/legacy timing explanation establishes the actual cause.

## Sources, applicability and alternatives

Reuse the reviewed [provider webhook](provider-webhooks.md) and
[one-shot probe](sandbox-webhook-probe.md) source records: official Plaid signature,
Transactions/SYNC_UPDATES_AVAILABLE and Sandbox firing sections selected2026-11-01.
New selected primary review: Go [Logger/New](https://pkg.go.dev/log#Logger) and
[select statements](https://go.dev/ref/spec#Select_statements),2026-11-01. Logger
serializes output writes; default select avoids waiting for channel availability.
Online docs identify go1.27/go1.27.3; local tests use installed1.26.5, release remains
pinned1.26.10. Selected API/semantics only, not a full standard-library source audit.

Inference: a bounded asynchronous observer can identify the rejecting stage without
viewing or retaining a protected token/header/body/key/ID. Alternative raw logging,
payload inspection/export, public diagnostic headers and speculative signature or
schema relaxation are rejected. A callback200 or successful probe is not receipt
proof; only the actual signed/durable/automatic chain qualifies it.

STD-01: internal operational classification with no financial amounts, currencies,
identifiers or posting instruction; no applicable ISO20022 message. Existing
Plaid2020-09-14/ISO4217/financial mapping and ISO21378 access limits are unchanged.
No financial/ISO conformance claim. New wire projection is exactly event
plaid_webhook_rejected and one fixed reason label; never request-derived strings.

## Boundary

Public Verify retains its exact errors/results. Internal verification additionally
returns a closed rejection enum. Existing signature, key, time, hash, environment,
strict body/profile and durable-receipt checks are unchanged. HTTP responses never
include diagnostic detail. Missing/invalid verification header is a distinct label.
Authenticated rejected schemas may report only fixed public-field shape classes
(for example absent readiness or null user); no values or unknown field names.
Classification occurs only on a rejected request and cannot authorize SQL.

Default injected handlers/hosts remain silent. Managed receiver composition fixes
one stdout reporter at construction for both profiles; deployment scope here is
Sandbox only. One writer goroutine/16-entry queue, at most20 enqueues per process
per minute; requests never wait for output. Full/closed queues and rate cap drop
diagnostics only. A blocked trusted writer can retain one bounded goroutine until
process exit, without blocking HTTP/shutdown; close is idempotent and stops further
enqueue. This is best-effort telemetry, not a complete audit trail or fleet limit.
Existing durable receipts and HTTP metadata remain authoritative evidence.

STRIDE: disclosure/high -> fixed enum labels only, forged enums sanitized, no raw
diagnostic values/reflection; availability/high -> bounded queue/rate and nonblocking
request path; spoofing/tampering/high -> original verification acceptance unchanged;
repudiation/medium -> classify failures, retain actual run/source/readback evidence,
do not treat dropped telemetry as absence of failures. No new grants/schema/routes.
Human sensitive-source/security review is required before receiver deployment.

Acceptance: TestRejectionStages, TestObservedHandlerNeverPersistsRejected,
TestRejectionReporterBounds and TestRejectionReporterBlockedWriter; existing full
verification/key/socket/profile/SQL suites and core/release checks. Exact scanned
new worker image, unchanged isolated Sandbox receiver configuration, one known
test Item and a fresh controlled event then identify the actual failing stage.
Any acceptance change needs a documented evidence-backed regression and review;
diagnostics alone do not resolve the401 or authorize Production activation.
