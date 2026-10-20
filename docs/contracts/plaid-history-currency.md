# Plaid history and dated currencies

## Decision and source evidence

Issue #225; owner authorized all available existing-connection history on
2026-10-20. Implementation claim: fix/plaid-currency-history. Product/capacity
follow-up #226 is separate. This is source acquisition, not accounting recognition.

Reviewed selected official sections on 2026-10-20:

| Source | Fact | Application and limits |
| --- | --- | --- |
| [Plaid Transactions API](https://plaid.com/docs/api/products/transactions/), sync cursor, days_requested, account_id, has_more | Initial history defaults to 90 days, maximum 730; initialized Items ignore changed lookback. Initial null cursor reads available updates; all pagination must be consumed. Each account filter has its own cursor. | Preserve existing enrollments/cursors and all returned updates. Do not use cursor now or switch endpoints to discard older history. No guarantee of complete bank history. |
| [Plaid Transactions overview](https://plaid.com/docs/transactions/), history availability | History accumulates while connected and may exceed the initial window. | Lookback is not a permanent rolling retention cutoff. Larger history capacity remains #226; current complete-set bounds fail closed without partial publication. |
| [ISO 4217 MA amendment 115](https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/amendments/dl_currency_iso_amendment_115.pdf), page 1, dated 2002-05-13 | USD/840/2 recorded for El Salvador effective 2001-01-01. | Historical corroboration, not USD's inception or a complete audit of every subsequent amendment. |
| [SIX List One](https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/lists/list-one.xml), USD record, existing pinned 2026-09-17 release | Current USD/840/2 was reviewed in sandbox-book-bootstrap.md with publication/hash provenance. | A publication date is not the currency's inception. For this bounded profile install an additional local coverage interval [2001-01-01,2026-09-17), corroborated by both source records. It is local supported coverage, not a claim of universal historical ISO conformance. |

No paid ISO normative text, private provider page or bank record was read. The
owner's safe diagnostics prove three currency rejections, not their exact input
field. A synthetic pre-window USD fixture establishes the defect repaired here.

Alternative: change the original immutable USD interval, discard older transactions,
or re-enroll with shorter history. Rejected: these respectively rewrite provenance,
lose source changes, or change enrollment without fixing currency semantics.
Another alternative is to ignore reference dates; rejected because actual currency
changes and minor-unit policies must retain dated metadata.

## SQL and runtime contract

The source binding pins monetary identity. For provider acquisition, resolve each
transaction date against installed currency rows sharing the binding's alphabetic
and numeric codes, with a non-null minor unit. Different code/numeric identity is
never inferred as an FX conversion. Return at most 32 ordered, nonoverlapping
versions in sync_context.currency_versions, retaining legacy currency metadata.
The decoder stores each row's exact selected version/minor-unit interpretation.
SQL independently verifies the selected version, identity, date and exact amount.
Missing coverage, duplicate intervals, unofficial currency or excess precision
reject the whole set; no date rewrite, skipping or automatic reference installation.
The existing singleton context remains an accepted compatibility fallback.

Migration 25 appends a fixed historical USD reference only when the exact reviewed
bootstrap reference is installed; it never changes existing references, bindings,
observations, journals or cursors. The owner-only installer is replay/conflict safe
and is also called by future owner bootstrap. Other currency histories are installed
through separately reviewed reference operations, not hardcoded into the worker.
No runtime DML or new runtime execute grant. Down refuses to delete installed
historical reference evidence. Pause dispatch before migration/runtime promotion;
old workers reject the new context shape, so deploy matching workers before retry.

## Acquisition versus posting

Acquisition publishes all supported source history, independently of source binding
date and book start. Acquisition readiness is not permission to post. Existing
source review and posting still require a book policy, active accounts, valid dated
currencies, an open period, current permissions and an explicit reviewed command.
No opening balance, date substitution or automatic journal is inferred. Historical
version identity remains on observations; no change to journal currency resolution.
Out-of-policy review explanations and initial lookback settings remain #226.

## Standards and acceptance

ISO 4217 applies to code/numeric/minor-unit/date mapping; reuse the project standard
source limits above. Plaid API 2020-09-14 JSON stays the documented proprietary
adapter mapping, not an ISO 20022 message. ISO 21378 export conformance is outside
this source-acquisition change. Multicurrency remains supported through installed
dated references; no rates, FX policy or jurisdictional accounting approval added.

Named evidence: TestHistoricalCurrencyAcquisition, TestCurrencyVersionContext,
TestSandboxUSDHistory, TestSyncHistoricalCurrencyValidation, existing currency/FX
ledger tests and source-review dates. Verify two versions in one set, interval
boundaries/gaps, identity/precision denials, zero journals, replay, out-of-policy
post denial and usable in-policy review. Run migration, ledger, acquisition,
sync-state and repository gates; run app integration before handoff. Hosted owner
retry/publication/review remains required after matching promotion; HTTP 204 alone
does not complete #225 or #88.
