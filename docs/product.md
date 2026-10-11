# Product scope

## Decision

Build separate books for multiple LLCs and personal finances. First prove one
complete month for one LLC; enforce entity isolation with two fictional entities
from the first database test. Account aggregation is an input, not the ledger.

## First usable bookkeeping release

Chart of accounts; reviewed opening balances; bank/card statement import;
drafts and splits; balanced posting and reversals; transfer matching;
statement reconciliation; trial balance, general ledger, P&L and balance sheet;
period closing; evidence attachments; accountant-readable exports.

Default workflow: import -> review -> post -> reconcile -> close -> export.
Bank feed corrections cannot silently rewrite posted or closed books.
Account ownership is per legal entity, not per provider login or account mask.

## Scope decisions still open

- Cash versus accrual basis and required adjusting-entry workflow.
- Functional and transaction currencies, FX policy, reporting period/timezone,
  rounding policy and precision.
- LLC/account counts, providers and accountant export format.
- Whether initial invoicing and bills are manual entries or dedicated workflows.
- Retention/deletion policy, access roles, initial opening balances and cutover.
- Recovery objectives, monthly cloud budget, and acceptable dispatch latency.

Multicurrency is required from the ledger design onward, including conversion,
settlement and period-end measurement contracts. Resolve the accounting policy
in JB-001 before coding the ledger. An accounting basis choice does not remove
double-entry requirements. Apply the [standards contract](contracts/standards.md)
to financial messages and practices.

## Deferred

Payroll calculation, tax filing, inventory costing, brokerage tax lots,
public SaaS signup, payments/money movement and
complex consolidated financial statements. Basic intercompany transfers still
need explicit entity-specific entries and reconciliation; dashboard aggregation
must not present itself as statutory consolidation.

## Done when

A reviewed representative month reconciles to statements, trial balance and
reports agree, every figure traces to journal/source evidence, corrections and
period locks work, and an accountant reviews the export and cutover evidence.
Passing software tests alone does not approve official-bookkeeping use.
