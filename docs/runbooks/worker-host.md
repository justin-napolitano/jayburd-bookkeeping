<!-- stage-runbook: true -->
# Bounded worker HTTP host

## Status and boundary

Issue #7, lease HOST-7. Coordinator authorized the fixed host profile before code.
Implemented only `internal/workerhost`: injected handler and listener; no routing,
readiness, environment, files, credentials, ADC, cloud calls or deployment wiring.
## Stage ledger

`New` is inert. `Host.Serve` takes listener ownership after validation, runs once,
and returns only fixed errors. The coordinator owns acquisition transport composition.

## Lifecycle contract

Read-header timeout 5s; total read timeout 10s; write timeout 90s; idle timeout 30s;
maximum configured headers 16KiB. Go's parser has its documented implementation
allowance above MaxHeaderBytes; this is not an exact serialized-request byte cap.
Cancellation propagates through BaseContext before graceful shutdown. Shutdown has
an independent 10s context; failure triggers Close. Serve's goroutine is joined.
Unexpected listener failure follows the same cancellation and draining path.
Default HTTP panic/error logging is discarded to prevent request diagnostics from
entering logs. Handlers still own fixed responses, authentication and durable receipts.

This host cannot terminate arbitrary Go code or external I/O that ignores context.
Hijacking/WebSockets are outside the approved worker handler profile; net/http does
not drain or close hijacked connections. An injected listener must obey net.Listener's
Close contract and unblock Accept. The bounded guarantee concerns ordinary HTTP
connections and cooperative handlers, not arbitrary malicious implementations.
Host shutdown never implies that an acquisition committed or rolled back.

## Reviewed sources and decision

Reviewed official Go [Server](https://pkg.go.dev/net/http#Server) fields BaseContext,
ReadTimeout, ReadHeaderTimeout, WriteTimeout, IdleTimeout, MaxHeaderBytes and ErrorLog;
[Shutdown](https://pkg.go.dev/net/http#Server.Shutdown),
[Close](https://pkg.go.dev/net/http#Server.Close), and
[Serve](https://pkg.go.dev/net/http#Server.Serve). Also inspected the pinned Go1.26.9
`src/net/http/server.go` definitions and shutdown implementation locally.
These selected sections establish cancellation injection and graceful-drain mechanics;
the fixed timeout values and fail-closed error vocabulary are project policy.
Chosen over default http.Serve (unbounded server defaults) or TimeoutHandler (does
not supply lifecycle draining). No financial message is introduced, so ISO 20022
mapping is not applicable; existing acquisition contracts retain that responsibility.
Coordinator adds this annotated source record to shared bibliography links.

## Current evidence

`make check test build` passed, including all-package race tests. Worker host
race tests passed (five named tests; actual loopback and ten-second forced drain).
Log: `/private/tmp/jayburd-worker-host-checks.log`. Sandbox initially denied local
socket bind; the approved rerun used loopback outside the sandbox. Named tests: TestFixedProfile,
TestCancellationDrains, TestForcedClose, TestValidationAndListenerFailure,
TestCancelledBeforeServe. Tests use synthetic local loopback requests, including
an actual 10s forced shutdown and explicit release/join of the uncooperative test
handler. No provider, database, cloud, credential or business-transaction proof is
claimed by these transport tests.

## Remaining work

Coordinator wires reviewed worker handler and deployment lifecycle separately.
No hosted availability or production qualification claim.

## PR handoff

Issue #7; no schema or financial API changes. Integrate this bounded host with the
coordinator acquisition transport tests; do not mount an arbitrary public handler.

## Independent dispatch regression followup

Coordinator additionally assigned the isolated
`internal/syncdispatch/fairness_sql_integration_test.go` after peer review found
candidate starvation. `TestDeliveryCandidateFairness` creates 30 earlier UUID
operations under one binding: 15 future retries and 15 healthy running leases,
plus a later ready operation. It requires the real publisher SQL candidate call
(limit 25) to return only the ready operation. Fixtures retain constraints and
triggers; no provider calls or trigger bypasses. Separate SQL-author tests own
the exact first-claim deadline and derived-lease takeover regressions.

Reviewed SQL fix uses one timestamp for first claim/deadline and excludes future
retry waits, healthy outer/inner leases and recently accepted task deliveries from
candidate slots. Root owns the migration, harness and merged proof. Borrowed
migration15/checksum/harness are uncommitted local validation inputs only.

Followup evidence: actual PostgreSQL dispatch proof PASS including the new fairness
case (`/private/tmp/jayburd-dispatch-fairness.log`); `make check test build` PASS
again including all-package race tests. The first fixture attempt failed on an
ambiguous UUID/text SQL parameter; explicit UUID casting fixed the fixture before
the successful full proof. These results qualify synthetic SQL behavior only.
