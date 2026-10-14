# Bounded connection workspace

## Decision and sources

Issue #104's first UI slice uses the frozen [HTTP](connection-http.md),
[service](connection-service.md) and [SQL](connection-sql.md) contracts. Selected
sections reviewed: routes, authority, one-shot claims, verified manifest binding,
credential recovery and concrete adapter limitations. These reuse PI05–PI08's
recorded source review; this UI adds no provider behavior or ISO message claim.
Internal connection commands follow the existing standards applicability profile;
currency versions and dates remain explicit. No provider SDK or live qualification.

## Component contract

`Connections` accepts api, entity, book, currencies, locked, denied, onReview and an
optional injected `link` adapter. `link.open({linkToken,signal})` returns a
memory-only `{publicToken}` or null for cancellation. Root owns composition; an
absent adapter means new bank setup is unavailable, while list reads remain usable.
New scope, lock and unmount discard sensitive state and abort the Link adapter.
Late results cannot execute an exchange, update another scope or start binding.

Supported operations are list, new connection, explicit first-provider binding,
known attempt inspection and reconciliation, and account-only disconnect.
Connection attempts are a separate durable lifecycle from posting commands.
Unknown exchange outcomes are inspected, never automatically exchanged again.
Binding/disconnect retries preserve the exact request and idempotency key in
memory; page loss requires reading durable status before any new action.

Each selected verified depository account maps to an explicitly chosen active
monetary debit asset account, matching currency version and binding date. No
account is preselected and no permission is inferred from the workspace import
flag. A generated binding evidence UUID identifies the explicit local confirmation,
not a fabricated bank document. SQL remains authoritative for grants and mappings.

## Presentation and limits

Current Item status and historical attempt status are labeled separately. Active
means enrolled, not complete acquisition or a balance. Unavailable reads never
render as empty connections. Pending/changed/removed source items use the existing
review/posting workflow; `onReview(sourceID)` opens its explicit source selector.
No automatic posting or financial totals originate here.

No public attempt discovery, pause, acquisition retry or timestamp projection exists.
Known attempt inspection needs its UUID; reload cannot rediscover a lost Begin
response. The local unfinished-enrollment latch also disappears on scope/lock/remount;
it is not durable cross-session prevention of a second enrollment. Reconnect is unavailable in the concrete provider profile. Item rotation,
Item removal, existing-source overlap binding and real Link hosting are deferred.
These limits keep full #104 acceptance open. Required evidence: component scope,
revocation, interrupted flow and single-flight tests, injected enrollment/binding;
root owns keyboard/mobile and browser→Go→PostgreSQL journey evidence.
