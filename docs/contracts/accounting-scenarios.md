# Accounting scenarios and policy decisions

## Status and assumptions

Draft for JB-001/JB-002. Synthetic examples are proposed acceptance oracles, not
approved policies for actual LLCs. Basis remains unselected. Examples involving
receivables use an illustrative accrual book with USD functional currency;
January/February 2026 dates and UTC accounting timezone are fixture assumptions.
Entity A and Entity B have separate books even under one owner. No tax is inferred.
[S01/S02](../research/database-sources.md) support double-entry and reconciliation;
[contract sources](../research/contract-sources.md) distinguish FX authority limits.

## Proposed chart and sign interpretation

Asset/expense accounts normally have debit balances; liability/equity/revenue
accounts normally have credit balances. Contra accounts reverse the presentation
convention. Account type/normal side do not prohibit opposite-side movements.
For each line store a positive magnitude and explicit debit/credit side; reports
calculate debit minus credit before applying their presentation convention.
Control accounts require source/subledger linkage once a subledger is enabled.

## Worked cases

Amounts below are USD unless marked EUR. Each row names a future test case.

| ID | Trigger and expected postings | Required observation |
| --- | --- | --- |
| AC-01 | Approved opening balances: Dr bank 1,000; Cr owner capital 1,000 | Opening evidence explains both sides; never invent equity to hide an unexplained difference. |
| AC-02 | Service earned and paid: Dr bank 200; Cr service revenue 200 | One journal despite repeated import. |
| AC-03 | Supplies consumed: Dr supplies expense 30; Cr bank 30 | Account classification is reviewed, not inferred from debit direction. |
| AC-04 | Card purchase: Dr expense 40; Cr card payable 40. Payment: Dr card payable 40; Cr bank 40 | Payment is not a second expense. |
| AC-05 | Split purchase: Dr expense A 20; Dr expense B 10; Cr bank 30 | Three lines, one balanced journal. |
| AC-06 | Refund of AC-03: Dr bank 30; Cr supplies expense 30 | Linked refund; actual tax/revenue treatment depends on source facts. |
| AC-07 | Bank transfer: Dr savings 100; Cr checking 100 | Both feed observations link to one transfer; no income/expense. |
| AC-08 | Contribution: Dr bank 100; Cr capital 100. Draw: Dr distributions 25; Cr bank 25 | Equity classification assumes owner activity, not payroll or a loan. |
| AC-09 | Reclassify AC-05: exact reversal, then Dr expense A 15 / Dr expense B 15 / Cr bank 30 | Original unchanged; correction/reversal links retained. |
| AC-10 | Accrual: Dr expense 60; Cr payable 60. Settlement: Dr payable 60; Cr bank 60 | Expense recognized once; cash-basis alternative requires its own policy. |
| AC-11 | A lends B 50: A Dr due-from-B 50 / Cr bank 50; B Dr bank 50 / Cr due-to-A 50 | Two separately authorized journals; common transfer reference, no cross-entity lines. |
| AC-12 | Book bank 950; statement 970; outstanding payment 20 | Adjusted statement 950 matches books; timing item needs no new journal. A missing bank fee requires an explicit draft adjustment. |
| AC-13 | Closing races a posting into January | Exactly one serialization order; a close that wins prevents a new January posting. |
| FX-01 | EUR receivable 100 at 1.10 USD/EUR: Dr receivable 110; Cr revenue 110 | Retain EUR 100 and the rate provenance. |
| FX-02 | Same exposure at January close, 1.20 USD/EUR: Dr receivable 10; Cr FX gain 10 | EUR quantity unchanged; carrying value becomes USD 120. |
| FX-03 | February settlement of FX-02 at 1.15: Dr USD bank 115; Dr FX loss 5; Cr receivable 120 | Exposure becomes zero; cumulative FX gain is 5, not 15. |
| FX-04 | Alternative to FX-03: settle EUR 50 at 1.15: Dr USD bank 57.50; Dr FX loss 2.50; Cr receivable 60 | Remaining EUR 50 has USD carrying value 60; do not execute both alternatives in one fixture. |
| FX-05 | Historical-cost asset acquired for EUR 100 at 1.10 | Preserve USD cost 110 on FX-only period review; other valuation/depreciation policies are separate. |
| FX-06 | Exchange EUR cash 100 carried at USD 110 for USD 115, fee USD 2 | Dr USD bank 113; Dr fees 2; Cr EUR cash 110; Cr FX gain 5. Original currency buckets need not individually balance. |

AC-01 through AC-10, sequentially, give bank 1,145 (checking 1,045, savings 100),
capital 1,100, distributions debit 25, expenses 130 and revenue 200. AC-09 is a
net-zero reclassification. Net income 70; assets 1,145 equal equity 1,145 after
including income and distributions. Trial-balance debits and credits each total
1,300. AC-11 onward are independent branches, not additions to this oracle.
A balanced trial balance alone does not prove completeness or proper classification.

## Policy decision register

| Question | Draft position | Acceptance owner / blocking scope |
| --- | --- | --- |
| Reporting framework and accounting basis | US GAAP accrual suggested; not selected | Owner/accountant, JB-001; recognition/FX rules |
| Functional currency and accounting timezone per book | Explicit, effective-dated approved policy; fixtures use USD/UTC | Owner/accountant; book activation |
| Rounding and precision | See money contract; half-even suggested, explicit residual lines | Owner/accountant; money DDL/posting |
| Opening source and cutover | Reviewed complete opening trial balance, evidence and reconciliation | Owner/accountant; opening/live use |
| Permissions | Reader, preparer, poster, closer; trusted service actor distinct | Owner/schema reviewer; runtime grants |
| Close/reopen and errors in prior periods | Locked periods; correction in approved open period; no reopen endpoint in first slice | Owner/accountant; close policy |
| Rate source, dates, staleness, exceptions | Versioned policy and immutable observations | Owner/accountant; automated FX |

## Coverage and sequencing

Core mechanics cover double-entry, corrections, entity separation and provenance.
Accruals, deferrals, depreciation, revenue recognition, impairment, inventory,
leases, payroll, taxes and consolidation need topic-specific policy/contracts
before automation. A manual journal is not permission to claim those rules are
implemented. Unsupported automated treatments return a typed error or review state.
Presentation-currency translation and consolidation require separate statements,
OCI/elimination rules and contracts; aggregation must not be labelled consolidation.

First implementation gate: approve JB-001 for the selected slice, review
AC-01–AC-10 and FX-01–FX-06, then implement named tests from the SQL contract.
No scenario above is currently an executed product test.
