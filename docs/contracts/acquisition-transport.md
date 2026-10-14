# Acquisition transport and host

## Decision and sources

Reuse owned observer transport mechanisms through explicit acquisition constructors;
keep journal observation and acquisition separate fixed profiles. No caller chooses
a handler, scope, provider URL or credential. This is the #7 synthetic implementation
slice; it creates no Cloud Run service, task, identity or provider connection.

Refreshed BW02/BW03: Google Cloud [issues and limitations](https://docs.cloud.google.com/tasks/docs/common-pitfalls),
duplicate delivery/order and throttling; [HTTP tasks](https://docs.cloud.google.com/tasks/docs/creating-http-target-tasks),
OIDC service-account authentication; [Task REST v2](https://docs.cloud.google.com/tasks/docs/reference/rest/v2/projects.locations.queues.tasks),
dispatch deadline. These facts require durable deduplication and authorization;
they do not promise process cancellation or business completion. Deadline and pool
budgets below are project policy, not Cloud Tasks recommendations.

Alternative: a generic request-selected handler factory. Rejected because a valid
journal wakeup must never select provider acquisition. Duplicating authentication
and queue clients would also create two implementations of the same security rules.

## Fixed profiles

| Profile | HTTP paths | Target version / task digest | Budgets |
| --- | --- | --- | --- |
| Existing journal observer | `/internal/observer/publish`, `/internal/observer/consume` | 1; existing SHA-256 of operation UUID, dispatch generation and 1 separated by pipes | Existing 60-second HTTP / 65-second task unchanged. |
| Acquisition | `/internal/acquisition/publish`, `/internal/acquisition/consume` | 2; SHA-256 of operation UUID, dispatch generation and 2 separated by pipes | 80-second HTTP / 85-second task / 90-second server write. |

Both use the exact identifier-only `work-wakeup-v1` envelope. Acquisition SQL loads
only its own operation table under current enrolled publisher/worker authority.
A structurally valid foreign operation ID fails; authentication never grants scope.
Constructors pin HTTPS Cloud Run destination, audience, invocation email and subject;
task metadata and arbitrary headers are not credentials. No browser route is added.

Acquisition reserves time around its 60-second runner for verified identity, request
read, claim and settle. A 120-second delivery lease is not acquisition authority:
takeover must additionally wait for any original or pagination-restart set lease.
Unknown commits preserve stable claims. Only confirmed durable outcomes/handoffs
can acknowledge delivery. Queue creation/409 is never completion.

## Host and acceptance

An injected lifecycle host sets read-header/read limits, bounded header bytes,
idle/write timeouts and shutdown budget. It does not discover credentials or choose
SQL/provider dependencies. Explicit composition remains necessary for deployment.
Internal scheduling messages have no selected ISO 20022 financial designation;
STD-01 and existing currency/accounting boundaries remain unchanged.

Named acceptance: acquisition profile isolation, OIDC-before-body rejection,
deadline/overload/cancellation, exact queue identity and no cross-profile enqueue;
actual SQL/runner tests prove replay, recovery and source publication separately.
Executed evidence belongs in the [delivery runbook](../runbooks/acquisition-dispatch.md).
