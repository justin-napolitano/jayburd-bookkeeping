# Shared acquisition publisher

## Decision record

**Question.** How can every enrolled Sandbox book receive initial and requested
Plaid Transactions sync work without configuring one Cloud Run publisher per book?
Issue #156 extends the existing publisher, not the financial source authority.
The publisher login sees only bindings enrolled for that exact login and each
binding still uses its existing authorization, consent, principal, stream,
lease and generation checks before dispatch. No journal is posted.

**Status and owner.** Accepted for implementation in the development project by
the owner's 2026-10-20 request to make acquisition automatic. Hosted activation
requires migration-first delivery and fresh Sandbox evidence. Live institutions
remain gated by #90.

**Sources and review depth.** The reviewed
[acquisition SQL](acquisition-dispatch-sql.md), [runtime](acquisition-dispatch-runtime.md),
[background work](background-work.md) and [self-service book](self-service-connections.md)
contracts define the existing invariants. The [Plaid Transactions webhook guide](https://plaid.com/docs/transactions/webhooks/)
and [Transactions API](https://plaid.com/docs/api/products/transactions/) were
reviewed 2026-10-20 for initial `/transactions/sync`, cursor updates, webhook
wakeup and provider polling cadence; the exact source record is
[shared acquisition sources](../research/shared-acquisition-sources.md).
These are provider facts; shared scheduling, fairness and SQL cursor design are
Jayburd inferences. No tenant webhook configuration or hosted result was observed.

## SQL and runtime contract

`integration.sync_delivery_bindings(maximum)` is a publisher-only,
SECURITY DEFINER, serializable function. It accepts only a limit (1–5), never
book/actor identity. It checks `session_user=current_user` at the login boundary,
the approved publisher role and a durable per-login cursor. An index on enabled
`(publisher_login,binding_id)` supports keyset pagination with wraparound. The
function returns up to five binding IDs for that login and advances its cursor
atomically. A crash after cursor commit delays a binding only until a later
rotation; it does not lose its SQL demand. Missing, disabled and foreign bindings
are not selected. All other binding authority is rechecked inside the existing
`sync_delivery_discover` and subsequent dispatch functions.

`shared-v1` publisher mode visits at most five bindings per tick and at most
five candidate operations per binding, within the existing 60-second request
budget. A revoked binding can fail with the fixed `not_authorized` code and be
skipped; it cannot stop a healthy binding in the same batch. Unexpected database
or transport failures are reported after the remaining bindings have been
attempted. The durable cursor gives eventual round-robin progress beyond one
page, with no in-memory state. Work is still idempotent and at-least-once.

`binding-v1` remains the fixed-binding rollback mode. The runtime and release
manifest explicitly choose `publisher_policy`. `shared-v1` requires an empty
binding ID; `binding-v1` requires the pinned UUID. The consumer remains unchanged.
The Cloud Scheduler tick is enabled only for the Sandbox development deployment,
with OIDC and Cloud Run invoker IAM owned by the existing Terraform stack.
Automatic sync uses the already configured webhook wakeups and periodic SQL
catch-up. It does not call paid `/transactions/refresh` or claim that Plaid has
newer data than the provider makes available.

## Standards, alternatives and acceptance

The scheduler body and Cloud Tasks work-wakeup-v1 are internal control messages,
not ISO 20022 financial messages. Source currency remains ISO 4217 by the
existing decoder contract. ISO 21378/21926 export mappings and accounting
policy do not change. No ISO conformance is claimed.

One publisher per book would make self-service enrollment depend on deployment.
Scanning all bindings every minute would be unbounded; scanning by client
supplied book would cross the scheduler trust boundary. A login-scoped SQL
keyset cursor preserves bounded fairness and existing per-book checks.

Synthetic SQL and Go tests must prove two enrolled books, distinct principals,
five-plus-one pagination, exact-login scope, revoked/disabled skips, concurrent
ticks and bounded work. Release tests must prove `shared-v1` has no binding UUID
and `binding-v1` rollback still works. Hosted acceptance checks scheduler HTTP
200, consumer HTTP204, completed generations and no unauthorized provider effect
for a newly enrolled Sandbox book.
