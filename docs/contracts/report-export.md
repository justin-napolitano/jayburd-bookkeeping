# Internal report review CSV

## Decision and approval gate

Implemented bounded `financial-reporting-csv-v1` wire profile for #51/#53.
Coordinator reviewed and approved this exact long measure schema on 2026-10-13
before implementation. This is an internal review download,
not an ISO 21378/21926 audit exchange or accountant-approved export. Reuse the
[financial reporting applicability assessment](financial-reporting.md#standards-applicability-and-open-policy)
and [SQL semantics](reporting-sql.md). No SQL migrations or new accounting rules.

## Ordered wire schema

Every cell is quoted, embedded quotes doubled, records end CRLF including the last.
Header has the same fixed field count as every row. UTF-8 without BOM. Columns:

1. `profile,report,layout,entity_id,book_id,snapshot_revision,functional_currency,functional_currency_version_id,minor_units,basis,timezone,generated_at,policy_ids,through_date,starts_on,ends_before,compare_through_date,compare_starts_on,compare_ends_before,selected_account_id`
2. `row_kind,row_id,account_id,category,normal_side,journal_id,line_number,accounting_date,posted_at,journal_kind,reverses_journal_id,policy_id,side,transaction_currency,transaction_currency_version_id,valuation_basis,rate_observation_id`
3. `measure,current_sign,current_magnitude,prior_sign,prior_magnitude,difference_sign,difference_magnitude`.

Each logical SQL row emits one physical CSV record per applicable measure, ordered
by this fixed enum: `net,debit_balance,credit_balance,balance,debit,credit,running_balance,transaction_amount,functional_amount,total_debits,total_credits,opening,period_debits,period_credits,closing,revenue,expense,net_income,assets,liabilities,direct_equity,unclosed_earnings,total_equity`.

These three groups concatenate to the complete ordered header. Inapplicable cells are empty quoted
strings, never zero substitutes. A present zero has `zero,0`; nonzero values use
`positive|negative` and unsigned canonical magnitude. At most 29 integer and nine
fractional digits; no leading integer zeros or trailing fractional zeros (0.01 is valid), exponent or float coercion.

## Row and provenance matrix

All rows repeat profile, report, layout (`account-v1` for TB/P&L, `posting-v1` for GL, `basic-unclassified-v1` for BS), exact scope,
revision, functional currency/version/scale, accrual/UTC, first-page generation
time, semicolon-separated policy UUIDs and applicable normalized date/account filters.
Policy order and all shared provenance must agree across pages (generation time
may differ; only the first is exported). Comparison fields are present only when
requested and supported by SQL; GL comparisons are rejected.

| Kind | Required row fields | Current measures | Prior/difference |
| --- | --- | --- | --- |
| account | row_id,account_id,category,normal_side | SQL row amounts: TB net/debit_balance/credit_balance; BS/P&L balance | SQL comparison amounts/difference when requested |
| posting | row_id,account_id,journal_id,line_number,accounting_date,posted_at,journal_kind,policy_id,side,transaction_currency/version,valuation_basis | SQL debit/credit/running_balance plus original transaction_amount/functional_amount | Empty |
| subtotal | row_id=`unclosed_earnings` only | SQL derived balance | SQL comparison amounts/difference when requested |
| summary | row_id=`summary` only | Complete report totals | Complete comparison totals/difference when requested |

Row IDs are `account:<UUID>`, `line:<journal UUID>:<line number>`, or the fixed
`unclosed_earnings`/`summary` labels. Journal kind is `manual|reversal`, side and
normal side are `debit|credit`, category is `asset|liability|equity|revenue|expense`,
and valuation basis is `identity|spot`. Line number is an integer from 1 through
1,000. Policy IDs use UUIDs separated by literal semicolons; dates use valid
YYYY-MM-DD and timestamps use database UTC microsecond spelling.

All other row fields are empty. Posting reverses_journal_id and rate_observation_id
are nullable according to original SQL provenance. Exactly one logical summary (one record per total measure) follows all
selected data rows, including an empty selection; its opening/closing may be nonzero.
No account code/name, description, reason or arbitrary source text is serialized.
UUIDs, calendar dates, UTC timestamps, numeric strings and closed enums are validated
before serialization; malformed generated fields abort the entire download.

## Execution and bounds

`Service.FinancialReportCSV(ctx,Command)` returns status 200 and CSV bytes only when
complete. Caller supplies trusted actor/scope and explicit snapshot_revision.
Client cursor/limit are rejected. All SQL pages use the same filters/revision with
an initial default 100-row page followed by 500-row pages and a shared ten-second deadline. Every page repeats SQL authorization;
a final fresh report read rechecks authority/revision before release. Revocation or
revision change discards the buffer. No point-in-time authorization promise extends
beyond the last authorized database transaction.

Maximum 10,000 physical data records including summary measures and 10 MiB serialized bytes including
header. This counts measures, not journals: 2,000 GL lines produce 10,004
records including four summary measures and therefore exceed the cap. Comparisons
occupy additional columns, not additional records. Excess returns 422 `report_export_limit`; invalid request returns existing
400 invalid_request; SQL stale/revoked/unsupported errors remain unchanged. Malformed
server output returns safe internal_error. Context timeout/cancellation returns
503 temporarily_unavailable. Existing SQL 4,096 account/GL materialization caps and
2 MiB page cap remain stricter independent limits. No partial output, spool, storage,
background runner or publication. No implementation of arbitrary functional FX.

## Sources and alternatives

Reused FR03/FR05 in [financial-reporting sources](../research/financial-reporting-sources.md).
Re-reviewed 2026-10-13: [RFC4180 §2](https://www.rfc-editor.org/rfc/rfc4180), complete
record/header/quoting clauses; [OWASP CSV Injection](https://community.owasp.org/attacks/CSV_Injection),
hazard characters and escaping/reopen limitations. These support CSV structure and
why quoting alone is insufficient. Our inference is strict generated grammars plus
omission of arbitrary labels. Alternative descriptive CSV requires separate target
spreadsheet tests; JSON remains the lossless rich-text contract. Neither source
settles accounting policy, ISO audit mappings or spreadsheet numeric precision.
