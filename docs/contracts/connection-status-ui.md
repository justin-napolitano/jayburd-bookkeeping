# Connection recovery and acquisition status UI

## Review and decision

Coordinator-approved design for #104 after merged #107. Reviewed the existing connection
workspace, HTTP/service/SQL lifecycle contracts and provider-sync SQL sections
on Item state, request generations, leases and complete publication. These reuse
the recorded PI05–PI08 provider sources; this is presentation and scoped control,
not a new provider wire or ISO 20022 claim. Currency/date handling retains the
existing standards profile.

The previous UI's memory-only enrollment latch cannot recover an attempt after
reload. Authenticated durable discovery must complete before new enrollment is
offered; unavailable, partial or malformed discovery fails closed. Browser storage
is not an alternative. An expired unclaimed attempt must be distinguished by the
server from unresolved claimed work; expiry never authorizes another exchange.

## Proposed presentation

A recovery section lists scoped durable attempts with clear state and permitted
inspection/reconciliation and expired-unclaimed presentation. No nonce, Link/public token, provider claim,
credential reference or raw provider error is displayed or persisted. Unfinished
provider exchanges stay unresolved; reconciliation never blindly retries exchange.

Each source stream shows current readiness, requested/completed generations,
last complete publication and a separately labeled unknown provider update time.
Publication is local atomic source visibility; it is not a provider update timestamp,
a posted balance, or proof that all history exists. Initializing, paused, pending,
lease-expired and credential recovery states remain visibly incomplete.

Pause/resume applies to one account stream. The explicit confirmation names that
account and states that sibling acquisition and posted journals are unchanged.
A manual sync request only records requested work; its receipt cannot claim an
executed provider pull or a changed posted balance. Both use current server pins,
explicit confirmation and the exact original idempotency key on uncertain replies.

## Frozen wire

GET `connection-attempts?limit=100&before=UUID` (before optional) returns
`{items:[ConnectionAttempt + expired_unclaimed:boolean],next_cursor:UUID|null,
unfinished_count:string}`. SQL calculates unfinished_count across all pages;
zero permits a new Begin after successful discovery, and SQL also blocks a fresh
Begin if unfinished work exists. Each page is bounded at100 and current permissions
apply on every read. Expired unclaimed rows are inspected without replay; claimed
uncertain work is never treated as expired authorization for another exchange.

GET `connections/streams/{stream}/status` returns stream/source/Item identifiers,
stream state/revision, Item state/generation, binding revision, requested/completed generations,
queued and lease_active booleans, readiness, last_completed_at, fixed last_error
and provider_updated_at:null. Revision/generation values stay decimal strings.
The UI loads one selected stream on explicit request, avoiding an unbounded fanout
across every account. It checks all three selected identities before display.

POST `connections/streams/{stream}/operations` takes `{action:pause|resume|request,
expected_item_generation:string,expected_binding_revision:string,
expected_stream_revision:string}` and one
Idempotency-Key. It returns status. An uncertain reply retains its exact body/key
in component memory. Stream revision independently fences pause/resume without
changing source binding/review authority. After confirmation, the UI reads fresh status so an old replay
receipt cannot masquerade as current state. Rejection does not authorize a new
provider exchange. Tokens and request bodies remain outside browser persistence.

The last_error allowlist is pagination_mutation, provider_failure, decoder_rejected,
cancelled, item_changed, lease_expired, account_disconnected, account_paused or null.
These fixed outcomes are shown as text; no raw provider diagnostic is accepted.
Existing concrete reconnect stays unavailable pending qualification. No provider
SDK or cloud integration is enabled by this UI.

## Acceptance

Named tests will cover reload discovery before Begin, expired-unclaimed versus
claimed recovery, malformed/incomplete discovery, revoked and stale scope,
late mutation responses, Item-wide confirmation, exact same-key unknown-result
recovery, honest timestamps and no browser persistence. Existing explicit review
and posting continue through source review and durable command recovery.
