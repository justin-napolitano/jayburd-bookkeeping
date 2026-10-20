# Automatic account binding source review

Reviewed 2026-10-19 for issue #209. This record covers only the source sections
named below. It is not a claim that every Plaid account type is supported by
Jayburd or that provider metadata determines accounting treatment.

## AB01 — Plaid Accounts API account type schema

**Publisher and source:** Plaid,
[Accounts API — account type schema](https://plaid.com/docs/api/accounts/#account-type-schema).

**Selected sections reviewed:** the `type`, `subtype`, `account_id`, balances
currency fields, account type schema, and account type/product support matrix.
The page was accessed 2026-10-19.

**Observed provider facts:** Plaid's current schema recognizes top-level account
types `depository`, `credit`, `loan`, `investment`, `other` and `payroll`. The
documented depository subtypes are `cash management`, `cd`, `checking`, `ebt`,
`hsa`, `limited purpose checking`, `money market`, `paypal`, `prepaid` and
`savings`. Plaid publishes a separate product matrix because product availability
varies by account type.

**Project interpretation:** Provider types and subtypes are external descriptive
metadata. They may gate a reviewed Jayburd policy, but they do not establish a
ledger category, normal side, ownership restriction, opening balance or financial
statement presentation. Issue #209 therefore allows only `depository` accounts
with subtype `checking`, `savings`, `money market` or `cash management` to seed a
new monetary asset/debit ledger account. All other combinations fail closed and
remain visible for explicit follow-up.

**Access and freshness limits:** This was a public web-document review. No Plaid
tenant configuration, institution behavior or real account payload was inspected.
Plaid can revise its schema and product matrix; a future change requires a new
review and an explicit Jayburd policy change rather than automatic acceptance.

## AB02 — Plaid Sandbox

**Publisher and source:** Plaid, [Sandbox](https://plaid.com/docs/sandbox/).

**Selected sections reviewed:** Sandbox institution, credential and multi-account
test behavior. The page was accessed 2026-10-19.

**Project interpretation:** Sandbox supports fictional multi-account acceptance
evidence. It does not prove production institution coverage, live credential
handling, complete transaction history or accounting correctness.

## Acceptance use

The fixed synthetic manifest for #209 must include three distinct supported
depository accounts and at least one unsupported account. Tests must prove that
the supported accounts receive distinct ledger/source accounts in one atomic
command while the unsupported account remains visible and unbound. A provider
success screen alone is not acceptance evidence.
