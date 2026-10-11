# Accounting correctness contract

## Status

Target contract; no ledger implementation exists. JB-001 resolves accounting
policy before JB-004 implements posting. Requirements below need named tests.
Multicurrency is required; review the proposed [money contract](multicurrency.md)
and [accounting scenarios](accounting-scenarios.md). Policy and automated FX
implementation remain gated. Apply the [standards contract](standards.md) to
messages and practices.

## Durable posting boundary

Within one PostgreSQL transaction: authenticate/authorize the trusted actor for
the entity, coordinate with period closing, validate accounts and currency,
balance the full entry, write journal and lines, audit the operation, persist
the idempotency receipt, and append required outbox intent. Commit or roll back
everything. Database grants must prevent alternate unchecked write paths.

## Invariants and acceptance evidence

| Invariant | Required test/evidence |
| --- | --- |
| Debits equal credits in the book's functional currency; original currency amounts retain their units | Reject unbalanced, empty and invalid-account entries; accept supported FX entries with explicit valuation provenance; never balance by adding unlike currency units |
| Exact finite amounts and explicit rounding | Decimal parsing, precision/scale bounds; no float, NaN or infinity |
| Entity isolation, including references | Allowed, denied, mixed-entity, revoked and deleted-principal tests |
| Atomicity | Inject failure after header/line/audit/outbox writes; no partial commit |
| Replay safety | Same key+payload returns same result; changed payload conflicts; concurrent duplicate yields one entry |
| Posted history is immutable | Direct mutation denied; reversal links to original; correction preserves provenance |
| Closing and posting are serialized | Race tests prove no posting into a period that closed first |
| Provider facts differ from bookkeeping | Pending/posted, removed and corrected source records never silently change posted journals |
| Transfers do not double-count expense | Bank/card payment matching and intercompany reconciliation fixtures |
| Reports agree with ledger | Golden trial balance, P&L, balance sheet and general-ledger fixtures |
| Reconciliation is explicit | Statement opening/closing balances, outstanding items, unmatched/duplicate rows |
| Source evidence is traceable and protected | Access-scoped attachments, audit events and redacted diagnostics |

## Operational boundary

Imports and classifications produce drafts until a reviewed policy authorizes
posting. Model-generated suggestions, if introduced later, grant no authority.
Use synthetic fixtures locally and in CI. Restore drills and accountant-reviewed
exports precede a separate live cutover decision.
