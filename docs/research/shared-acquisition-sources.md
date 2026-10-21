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
