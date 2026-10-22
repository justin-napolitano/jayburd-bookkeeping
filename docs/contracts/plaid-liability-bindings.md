# Reviewed Plaid liability bindings

## Decision and scope

Issue #210 extends the atomic binding command from #209 in the synthetic
development book. The owner requested separate ledger accounts for selected
credit and loan accounts. This decision permits `credit / credit card`,
`loan / student`, and `loan / mortgage` from a verified Plaid Transactions
manifest. Each creates or selects one distinct monetary `liability / credit`
ledger account, source account and sync stream after explicit owner review.
All other credit/loan subtypes remain visible and unbound. The book owner must
review the proposed category, normal side, code, name, ISO 4217 currency and
binding date; Plaid metadata alone is not accounting authority.

## Additional registration profile — #258, 2026-10-22

Owner requested `loan / auto`, `loan / consumer`, and `loan / line of credit`
registration in the fictional Sandbox. Extend the existing distinct
`liability / credit / monetary` mapping to those three verified subtypes.
The Accounts API defines them under loan; its product matrix only promises
Transactions for some loan subtypes. Therefore registration permits attempted
acquisition, never inferred history, balances or product coverage. Unknown
subtypes remain unbound; the existing liability posting guard remains mandatory.
Alternative: keep these visible but unbound, which prevents the requested
registration even where Sandbox supplies transactions. Enabling every loan type
would exceed the reviewed scope. Acceptance: mixed seven-account SQL binding,
exact replay, unsupported-type rollback, trusted type/subtype decoder pins and
browser binding tests; hosted registration remains separate #88 evidence.

This extends the existing proprietary Plaid API `2020-09-14` mapping, not a new
ISO 20022 exchange. ISO 4217 dated identifiers/minor units remain enforced;
ISO 21378 audit-export mapping is unchanged and outside this acquisition slice.
No normative ISO conformance is claimed. Refreshed sources are PL04 in the
source register, selected public sections only.

The [Plaid Accounts type and product matrix](https://plaid.com/docs/api/accounts/)
and [Transactions API](https://plaid.com/docs/api/products/transactions/) were
reviewed on 2026-10-20. Transactions coverage for loans is narrower than the
Accounts schema. This profile does not enable the separate Liabilities product.
Provider coverage varies by institution; the manifest and sync response remain
subject to strict account identity checks. See the
[source record](../research/plaid-liability-sources.md).

## Invariants

- `new_ledger_account` has the existing reviewed shape and fixed
  `liability / credit / monetary` semantics for this profile. Server and SQL
  both validate the type/subtype and category pairing. Existing-account
  bindings require the same category and normal side.
- The owner selects an installed, dated ISO 4217 currency version and binding
  date. Every provider account maps to a distinct ledger and source account.
- The current serializable binding command owns account, source, permission,
  stream and receipt creation. Exact idempotent replay returns the same result;
  conflicts and partial failures roll back. No opening balance or journal is
  inferred from Plaid balances or transactions.
- Sync compares the returned account type with the trusted manifest for the
  bound stream. Unknown or changed types fail closed.
- Liability observations may be acquired, but the current source-review
  posting formula is approved only for asset/debit sources. Liability source
  drafts remain blocked until a separately reviewed transaction-direction,
  principal/interest/fee and offset-account policy is implemented. This also
  prevents a credit-card payment or loan payment from being posted as a simple
  expense from the wrong side.

## Standards, alternatives and evidence

This is an internal setup command and proprietary Plaid JSON adapter, not an
ISO 20022 financial message. ISO 4217 code, version, valid interval and minor
unit validation remain mandatory. ISO 21378/21926 export semantics are not
changed. No ISO conformance is claimed.

Treating all Plaid `credit` or `loan` subtypes as supported would exceed the
Transactions product matrix and infer accounting policy from a label. Requiring
manual account pre-creation would restore the broken multi-account journey.
The narrow reviewed allowlist preserves one atomic enrollment and makes the
coverage limit explicit. Provider documentation does not prove institution
coverage or source sign semantics. Fresh Sandbox acceptance must verify the
actual selected accounts and zero automatic journals.

Acceptance: synthetic SQL, Go and browser tests cover one credit card, one
student or mortgage loan, unsupported subtype, distinct accounts, exact replay,
conflict/rollback, manifest type mismatch, and blocked liability draft; the
browser/API/PostgreSQL gate must pass. Hosted qualification belongs to #88.
