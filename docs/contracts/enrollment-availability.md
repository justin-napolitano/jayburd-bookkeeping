# Optional enrollment expiry and bookkeeping availability

## Decision and boundary

Implementation proposal for #281, 2026-10-24. Keep structurally valid expired
Sandbox enrollment configuration as a disabled provider capability rather than
an invalid bookkeeping server configuration. Partial/unsafe metadata and future
expiry beyond the existing thirty-day maximum still reject before I/O. Release
authorization remains strict/future under connection-release; this change does
not make an expired release eligible for promotion or renew source consent.

Runtime checks an injected clock (production uses time.Now) at construction and
each provider-dependent service entry. At the exact deadline authorization has
expired. Cold startup after expiry returns the real SQL-authorized connection
service with no provider/credential adapters and performs no managed token,
secret, KMS or provider call. Expiry during static credential initialization
falls back to that same service. A live approved runtime gates every outgoing
dependency HTTP request at the deadline as well; no retry or grace is added.

## Authority matrix

| Operation | Expired optional enrollment policy |
| --- | --- |
| Identity, ledger, workspace and durable setup/post/import recovery | Unchanged verified identity/current SQL authority; no dependency on enrollment |
| List connections, list/get attempts, stream status | SQL-authorized metadata reads remain available; no provider credential required |
| Stream disconnect, pause/resume/request | Existing local SQL authority and per-source consent/revision fences remain authoritative; no source fetch in API |
| Begin, execute, complete Hosted Link, reconcile credentials, bind, abandon Item | Fail existing temporarily_unavailable/HTTP503 before SQL claim/secret/provider work; metadata discovery remains available |
| Existing workers | Unchanged independent source consent/generation/permission fences; optional new-enrollment policy does not renew or revoke an Item |

Explicit Item-wide provider removal/abandonment is not a local stream disconnect;
it remains unavailable without current provider authority. Preserve unknown
results and exact attempts; never replace a recovery action with new enrollment.
Request-time permission validation and immutable journal/outbox rules are unchanged.
No financial data enters browser persistence or operational logs.

## Sources, applicability and alternatives

Reviewed selected official sections on 2026-10-24:
[Go Time.After](https://pkg.go.dev/time#Time.After) (displayed go1.27.3 public
reference; strict ordering semantics; pinned1.26.10 page inaccessible, local
Go1.26.5 tests and canonical pinned CI are separate implementation evidence)
and [Cloud Run container lifecycle/startup](https://docs.cloud.google.com/run/docs/container-contract#lifecycle)
(instances must start a listener within the startup bound; failed startup cannot
serve requests). Public sections only; these sources establish mechanisms, not
permission to make bank calls. The existing #281 incident and
[app recovery evidence](../runbooks/app-recovery31.md) establish the project failure.

Separating configuration validity from optional capability authority is our
inference. Rejected alternatives: extending the deadline silently changes owner
approval; disabling all connection routes loses SQL-only status/disconnect;
continuing to load credentials after cold expiry adds unnecessary dependency
failure and secret access. Keeping the entire API down is fail-closed but blocks
unrelated authorized command recovery. This design denies provider operations
while preserving their durable status and independent bookkeeping authority.

Standards applicability: this internal lifecycle/authorization rule is not a
financial exchange and has no ISO20022 message mapping. Existing financial
payload, ISO4217 dated currency/original-unit mappings and accounting policy
are unchanged. No new data schema, migration, message version or conformance
claim. Follow standards.md and the annotated bibliography for those mappings.
Security review applies to authorization/secret boundary changes before release;
synthetic proof is not AppSec, hosted, or live-bookkeeping approval.

## Acceptance

Deterministic runtime tests cover before/at/after deadline, cold restart, expiry
during static reads, malformed expired metadata, zero expired dependency I/O,
and outgoing-request fences. Service tests cover every provider entry denied
without SQL/secret/provider effects and current SQL-only reads/controls including
permission denial. The actual synthetic browser/Next/Go/PostgreSQL setup and
lost-commit recovery journey runs with a real expired enrollment composition;
bank begin returns unavailable and dependency transports are tripwires.
Core and web/app-integration gates remain required. Actual hosted revision and
saved-command recovery evidence remains open until a separately scoped release.
