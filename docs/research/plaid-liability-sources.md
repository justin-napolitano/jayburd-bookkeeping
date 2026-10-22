# Plaid liability source review

Reviewed 2026-10-20 for #210. Access was limited to public documentation;
no tenant setting, institution response or provider payload was inspected.

## PL01 — Accounts API

Plaid, [Accounts API account type schema and product support matrix](https://plaid.com/docs/api/accounts/).
Reviewed `type`, `subtype`, account identity, currency fields and product
matrix. Plaid describes `credit` and `loan` account types, but that description
does not determine Jayburd ledger presentation or posting direction. The
Transactions product matrix supports credit cards and only a subset of loans,
including student and mortgage. Coverage varies by institution. Jayburd's
`credit card`, `student` and `mortgage` allowlist is a local policy inference.

## PL02 — Transactions API

Plaid, [Transactions API `/transactions/sync`](https://plaid.com/docs/api/products/transactions/).
Reviewed account-filtered cursor sync, supported account types and update
status. The API provides transaction observations, not a balanced journal,
opening liability, principal/interest allocation or audited history. The
account-filtered response must still be matched to the trusted stream account.

## PL03 — Liabilities API

Plaid, [Liabilities API](https://plaid.com/docs/api/products/liabilities/).
Reviewed product scope only. It may expose additional liability details and
loan types but is a separate product and does not expand Transactions coverage.
It is not enabled by #210.

## PL04 — Additional fictional loan registration, 2026-10-22

Refreshed the official [Accounts API](https://plaid.com/docs/api/accounts/)
loan subtype definitions for auto, consumer and line of credit, plus the
account/product matrix caveat that only some loans support Transactions.
Refreshed [Transactions sync](https://plaid.com/docs/api/products/transactions/)
account type/subtype and amount fields. Public selected sections reviewed; no
institution payload or tenant settings accessed. Fact: subtype vocabulary does
not guarantee product availability. Inference approved by the owner's requested
scope: permit these three verified Sandbox loans to register with the existing
liability mapping and attempt scoped acquisition; retain explicit failure state.
No principal/interest allocation or opening balance follows from registration.

## Limits (continued)

Documentation can change. Fresh Sandbox verification is required for the
selected types and newly bound streams. Live-bank approval remains #90.
