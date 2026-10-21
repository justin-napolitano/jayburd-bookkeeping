# Shared acquisition source review

Reviewed 2026-10-20 for issue #156. Only the named public sections and
repository contracts were reviewed; no live institution, provider payload or
tenant webhook setting was inspected.

## SA01 — Plaid Transactions webhooks

Plaid, [Transactions webhooks](https://plaid.com/docs/transactions/webhooks/),
sections "Configuring Link for transactions webhooks" and "Integrating the
update notification webhook". Plaid says Transactions should be initialized
with Link to start acquisition, and `SYNC_UPDATES_AVAILABLE` follows the first
`/transactions/sync` call. The webhook announces available changes; it does not
replace the cursor pull. Inference: the publisher must dispatch initial demand
without waiting for a webhook, then process durable webhook/manual demand.

## SA02 — Plaid Transactions API

Plaid, [Transactions API `/transactions/sync`](https://plaid.com/docs/api/products/transactions/),
endpoint overview and availability notes. Cursor sync retrieves incremental
updates; empty first results are possible while Item data readies. Plaid
typically checks for new data one to four times daily, varying by institution.
Inference: a one-minute SQL publisher tick reduces application queue delay but
does not force a new bank pull or prove complete provider history.

## SA03 — existing Jayburd dispatch contracts

[Acquisition dispatch SQL](../contracts/acquisition-dispatch-sql.md) and
[runtime](../contracts/acquisition-dispatch-runtime.md), reviewed for function
authority, 25-candidate bound, replay and lease behavior. The fixed binding
is a deployment choice, not a SQL requirement for financial correctness.
Changing selection must preserve per-binding SQL authorization.

## SA04 — Go patch release and container provenance

The [official Go downloads](https://go.dev/dl/) list 1.26.10 for supported
platforms. The PR #230 `govulncheck` run identified GO-2026-6809 in the pinned
1.26.9 standard library and named 1.26.10 as fixed. Reviewed 2026-10-20 at
release listing and CI symbol-result depth; the vulnerability page was not
accessible through the source browser. `docker buildx imagetools inspect`
resolved the official `golang:1.26.10-alpine` multi-platform index to
`sha256:9ce231b28998b32e116927c61aa5c406183061d4f3f2d61b292c88e7f6bdf7ca`.
Inference: update all build/test toolchain pins together, then require the
repository security and container gates to prove the new pin.
