# Automatic ledger accounts for provider binding

## Decision record

**Question and scope.** When a verified new Plaid Item contains several selected
depository accounts, how can a creator bind all of them without first editing the
chart of accounts? This contract covers new depository bindings only. Credit and
loan policy is deferred to #210; investments and unknown types remain unbound.

**Status and owner.** Accepted for #209 by owner direction on 2026-10-19.

**Sources.** The existing [connection SQL contract](connection-sql.md) establishes
one provider account, source account and ledger account per binding. The
[accounting contract](accounting.md) establishes asset/debit/monetary semantics.
Plaid's [Accounts API](https://plaid.com/docs/api/accounts/) account `type`,
`subtype`, `account_id` and currency fields were reviewed on 2026-10-19. The
[Sandbox guide](https://plaid.com/docs/sandbox/) was reviewed for fictional
multi-account behavior and default test credentials. PostgreSQL 17 documentation
for [transaction isolation](https://www.postgresql.org/docs/17/transaction-iso.html)
and [function security](https://www.postgresql.org/docs/17/perm-functions.html)
remains registered as S08/S12 in the database source register.

**Review depth and evidence.** Only the named Plaid field definitions and Sandbox
credential/multi-account sections were reviewed. Plaid establishes provider
metadata, not ledger classification. PostgreSQL establishes transaction and
privilege mechanisms; Jayburd owns the atomic command and retry policy.
The detailed source record and product-profile interpretation are in
[automatic account binding sources](../research/automatic-account-binding-sources.md).

## Provider support profile

Plaid's current account-type schema recognizes six top-level types:
`depository`, `credit`, `loan`, `investment`, `other` and `payroll`. Plaid also
publishes type/subtype combinations and a product support matrix. Those describe
provider data availability; they do not authorize a Jayburd ledger category.

Issue #209 permits automatic creation only when the verified manifest contains:

- `type: depository`;
- `subtype` equal to `checking`, `savings`, `money market` or
  `cash management`;
- exactly one ISO 4217 currency identifier that resolves to an installed currency
  version on the binding date.

Plaid depository subtypes `cd`, `ebt`, `hsa`, `limited purpose checking`,
`paypal` and `prepaid` remain visible and unbound in this slice because their
ownership, restriction or presentation semantics require an explicit policy.
Missing and unrecognized subtypes also remain unbound. `credit` and `loan` are
deferred to #210. `investment`, `other` and `payroll` require separate contracts.
The allowlist is versioned Jayburd policy and must not expand merely because Plaid
adds or renames a type or subtype.

**Interpretation.** Each selected depository account needs a separate monetary
asset ledger account so its balance, observations and reconciliation remain
separate. The creator must review the proposed code, name, currency and active
date. Provider type/subtype may seed a neutral label but never authorizes an
accounting policy or opening balance.

**Alternatives.** Requiring pre-created accounts preserves the existing API but
breaks the ordinary Plaid journey. A separate create-accounts command exposes a
partial state and a second retry boundary. Creating accounts in the existing
binding command preserves explicit review and makes account creation, source
binding, permissions, stream activation and receipt one atomic operation.

**Uncertainty.** Provider subtype can be absent or change. Code suggestions can
race another chart edit. Unknown types, missing currency and conflicting codes
must fail closed or remain unbound. The first slice does not create credit, loan,
investment or opening-balance entries.

## Request and invariants

`ConnectionBindingRequest.bindings[]` retains `ledger_account_id`. It adds an
optional `new_ledger_account` object with exactly:

- `account_id`, equal to `ledger_account_id`, generated before submission;
- owner-reviewed `code` and `name`;
- fixed `category: asset`, `normal_side: debit`, `monetary: true`;
- `active_from`, equal to the reviewed binding date.

An existing-account binding omits `new_ledger_account`. A new-account binding
includes it. The same ledger account ID, code or provider account cannot appear
twice in one command. Codes and names use the existing ledger bounds. Every new
account's provider manifest entry must satisfy the exact provider support profile
above and have a supported ISO 4217 currency version valid on the binding date.
The server validates both `type` and `subtype`; browser eligibility is
presentation only and cannot broaden the server allowlist.

The browser proposes one new account per eligible provider account. Suggested
codes are derived from the current reviewed book snapshot in stable increments;
the creator can edit them. Suggested names use bounded provider subtype plus a
stable ordinal, never a provider account ID or balance. A conflicting or stale
proposal returns for review; neither SQL nor Go silently renumbers it.

## Atomicity and recovery

`integration.bind_connection` remains the single serializable command. Under the
current authority and attempt locks it checks the immutable manifest and complete
request, creates all requested ledger accounts, creates source accounts and
streams, grants capabilities, activates the Item/attempt and writes the existing
receipt. Any error rolls back the whole transaction.

The existing actor/scope/operation/idempotency-key receipt owns retry. A repeat
with the exact normalized body returns the recorded result. A different body for
the same key conflicts. Fixed client-generated ledger account IDs plus the receipt
make a lost acknowledgement recoverable without duplicate accounts. Binding makes
no Plaid call, creates no journal and records no opening balance.

## Standards applicability

This is an internal chart-setup command, not an ISO 20022 message. Currency
selection uses the installed ISO 4217 identifier/version and minor-unit record.
ISO 21378/21926 export mappings are unaffected. Provider JSON remains a documented
proprietary adapter; no ISO conformance is claimed.

## Acceptance

- `TestConnectionAutoAccountBinding`: three verified depository accounts create
  three distinct monetary asset ledger/source accounts and active streams.
- `TestConnectionAutoAccountAtomicity`: conflict and injected failure leave no
  partial ledger accounts, sources, grants, streams or activation.
- `TestConnectionAutoAccountReplay`: exact same-key replay returns the same account
  and stream identifiers; changed body and concurrent code collisions reject.
- Go and HTTP tests reject malformed account semantics, missing or unsupported
  provider type/subtype combinations, invalid currency/date and ledger-account ID
  mismatch before mutation.
- Browser tests default all eligible accounts to distinct proposals, permit
  existing-account selection and opt-out, and keep unsupported accounts visible.
- The full synthetic browser/API/PostgreSQL journey reaches review with zero
  automatically posted journals.

Follow-up #210 owns reviewed liability support. #88 owns the fresh hosted Sandbox
proof after deployment; #90 continues to gate live institutions.
