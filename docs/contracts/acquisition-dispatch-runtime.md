# Acquisition dispatch runtime

## Reviewed scope

Issue #7 extends the SQL-owned demand and delivery sidecars with a bounded Go
publisher and consumer. The coordinator approved the function wire with migration
15's sole SQL author before implementation. SQL remains the source of truth for
current authority, stable acquisition claim identity, fences, completion receipts,
retry timing and the eight-failure OR 24-hour budget. Queue acceptance is not
acquisition completion. Importing source observations never posts journals.

`NewPublisher(db, bindingUUID, observer.Transport)` implements `observer.Ticker`.
`NewConsumer(db, Runner)` implements `observer.WakeupHandler`. The injected Runner
has the existing acquisition `Run(ctx, Scope, claimUUID)` signature. Production
composition must supply distinct publisher and sync logins. No default credentials,
provider calls, HTTP mount, scheduler or deployment are enabled by these constructors.

## Protocol and authority

The existing strict `work-wakeup-v1` parser accepts only operation UUID, string
positive dispatch generation and numeric schema version 1. The task target version
is 2 and its name is lowercase SHA256(operation UUID + `|` + generation + `|2`).
This separates acquisition task identity from observer target 1. Root-owned explicit
acquisition HTTP/Cloud Tasks constructors pin the route, workload identity, target
version and deadline; callers cannot choose a URL or financial payload.

Publisher Tick runs at most 60 seconds and 25 candidates: discover and commit,
read candidates, recover, dispatch claim and commit, enqueue outside SQL, then
record the transport outcome in a new committed transaction. Transport has ten
seconds per call. Partial counts describe local progress, not backlog completion.
ALREADY_EXISTS is enqueue evidence only. Unknown claim COMMIT suppresses enqueue.
Contradictory transport result plus error is rejected without guessing evidence.

Consumer validates bytes before SQL, commits the delivery claim, validates its
scope/version/lease and calls Runner with the SQL-owned stable acquisition UUID.
It reserves five seconds for settlement and caps its own request at 75 seconds
inside the separate HTTP 80-second budget. Runner independently caps acquisition
at 60 seconds. Parent cancellation propagates; an arbitrary injected dependency
must honor context, and no promise of forcible termination is made.

Settlement receives only completed/retryable/permanent/unknown and a rounded-up
retry delay. SQL independently checks original and deterministic mutation-restart
completion receipts and actual acquisition leases before acknowledging or replacing
business claims. A completed Go return alone does not authorize success. Unknown
COMMIT, cancellation and unrecognized errors are unknown hints, never a new UUID.
Retry-After through 24 hours is not shortened. `retry_deferred` and a longer delay
quarantine rather than automatically retry sooner. When a crash loses a known
provider delay before settlement, SQL must quarantine the ambiguous abandoned
attempt rather than manufacture a shorter wait.

Only a committed done/quarantined result or committed recoverable retry_wait
handoff returns nil. Busy/not_due/stale and unknown settlement COMMIT remain
errors. SQL owns redispatch after acknowledged retry_wait; no in-memory timers
or background goroutines carry durable work. Outer lease expiry cannot permit
concurrent use of a still-live original or derived acquisition set.

## Transaction and failure policy

Each function uses SERIALIZABLE with four seconds active, three-second statement
and one-second lock timeouts, and an independent one-second rollback cleanup.
Only confirmed 40001/40P01 whole-transaction rollback retries, at most three times.
Other unknown COMMIT errors, including 08007 and 40003, expose no prepared result.
Role validation rejects SET ROLE impersonation, opposite worker/API/owner and
privileged memberships, as well as direct application-table grants. SQL additionally
validates enrolled login and current mandate. Metadata and external errors expose
only closed adapter-owned classifications, never provider text or credentials.

## Evidence and standards

The [background work](background-work.md) STD-01 internal-message assessment
applies: this is an internal wakeup, not an ISO 20022 financial exchange. Existing
source acquisition retains its pinned proprietary profile and ISO currency mapping.
Reviewed the existing [observer runtime](observer-runtime.md) OR01 PostgreSQL
SQLSTATE discussion, [observer transport](observer-transport.md) OT01–OT06 selected
Cloud Tasks deduplication/authentication sections, and [acquisition runner](acquisition-runner.md)
PI01/PI02/PI08 complete-update/pagination evidence. Reusing these mechanisms avoids
a second queue client or a second acquisition implementation. Delivery sidecars,
local budgets and conservative quarantine are project policy, not vendor guarantees.

Unit tests exercise unknown claim/settlement, bounded rollback retry, external calls
outside transactions, exact target identity, safe error classifications, Retry-After
rounding and cancellation. Actual PostgreSQL composition evidence is tracked in the
[stage runbook](../runbooks/acquisition-dispatch-runtime.md); synthetic proof does not
qualify hosted OIDC, Cloud Tasks delivery or live financial use.
