# Connection HTTP operations

## Decision

Expose the reviewed connection service through an optional explicitly injected
`server.Dependencies.Connections`. Normal `Managed` startup leaves it absent;
there is no environment switch that silently activates Plaid or secret writes.
The same authenticated principal, exact origin boundary and 12-second request
budget as the accounting API apply. SQL rechecks current connection authority.

This implements #101's owned HTTP boundary. It does not add the #104 Link UI or
configure the hosted provider/Infisical writer. Proprietary Plaid JSON and internal
connection commands are not ISO 20022 messages. Currency/date semantics follow the
existing standards profile. See [service](connection-service.md) and
[SQL](connection-sql.md) contracts for state, authority and credential recovery.

## Routes

Issue #155 adds [creator onboarding](self-service-connections.md) in the explicitly
configured managed connection runtime. GET discovery remains read-only and can
return empty lists for eligible unconfigured creators. Begin(new) prepares the
book and persists the attempt atomically before Link work. Existing configured
or delegated books retain current grants; revoked grants are never recreated.
No request can supply worker logins or service principals.

Prefix `/v1/entities/{entity}/books/{book}`; IDs are canonical UUID strings.

| Method / suffix | Service operation |
| --- | --- |
| POST `/connection-attempts` | Begin with mode and explicit affected-stream approval. |
| GET `/connection-attempts` | Actor-scoped attempt discovery; optional limit 1–100 and before UUID. |
| GET `/connections/streams/{stream}/status` | Safe current publication/status pins; unknown provider update stays null. |
| POST `/connections/streams/{stream}/operations` | Explicit pause/resume/request with Item, binding and stream revision pins. |
| GET `/connection-attempts/{attempt}` | Safe current attempt projection. |
| POST `/connection-attempts/{attempt}/execute` | Memory-only nonce/Link handoff; omit public token for Hosted Link completion. One-shot provider mutation. |
| POST `/connection-attempts/{attempt}/reconcile` | Empty object; resolve known credential state without repeating exchange. |
| POST `/connection-attempts/{attempt}/bindings` | Explicit verified account/source/ledger/currency selection. |
| GET `/connections` | Bounded current connection list. |
| POST `/connections/streams/{stream}/disconnect` | Expected binding revision; account-only local fencing. |

Begin, bindings, stream operations and disconnect require one `Idempotency-Key`, 1–128 characters in
`[A-Za-z0-9._:-]`. Every POST requires exactly `application/json`, no content encoding,
one UTF-8 object at most 32 KiB, no duplicate or unknown fields and nesting at most
16. Only attempt discovery accepts the named limit/before query parameters; duplicate, unknown or malformed values are rejected. Encoded alternative paths and GET bodies are rejected. HEAD
cannot invoke reads. Authentication precedes body parsing or service access.

Success is JSON with status 200 and `Cache-Control: no-store`. Begin alone returns
an ephemeral nonce/Link token; browser consumers must keep these only in memory.
Status/list never return credential references, token hashes, operation claims or
provider diagnostics. No request body, underlying error or token is logged here.
Fixed error codes map invalid input to 400, denied/not-found to 403/404, conflict,
stale or credential recovery to 409, limits/unsupported modes to 422, and uncertain commits/temporary
failures to 503. Unknown failures become generic 500. No response acknowledges an
uncertain remote exchange or local commit as success.

## Evidence and limits

`TestConnectionHTTPBoundary` and `TestConnectionHTTPFailures` cover routing,
identity, origin, malformed requests and redaction. These injected tests prove
transport only. Named service/SQL proofs establish lifecycle behavior; the
[delivery runbook](../runbooks/provider-acquisition.md) records actual results.
A canceled request cannot undo remote exchange; durable recovery state must outlive
its connection. No automatic posting, hosted availability or provider qualification.

## Recovery and stream controls

Migration 14 adds actor-scoped attempt discovery, an authoritative unfinished count,
and a fresh-new-attempt guard while preserving exact-key replay. Expired unclaimed
attempts remain inspectable without authority to execute their old handoff.
Account-stream pause/resume has an independent revision so stale browser controls
cannot undo a later lifecycle transition. Manual request queues durable intent; it
does not call a provider or report completed acquisition. The exact returned
status may be a replayed receipt; refresh status before issuing a new operation.
See [SQL operations](connection-operations-sql.md) for current-grant checks.

Hosted Sandbox Begin also returns a validated memory-only URL; execute may return
unchanged link_ready while session completion is pending. See the
[Hosted Link contract](hosted-link-enrollment.md) for completion and recovery.
