# Reporting SQL foundation (#72)

## Scope and review

Implementation contract refining [financial reporting](financial-reporting.md),
using its FR01–FR05 sources and standards assessment. No HTTP/OpenAPI/browser/CSV
surface in this slice. Existing trial-balance function is unchanged. New migration6
adds only restricted reporting functions. Coordinator review is required before DDL.

## Request and response

`reporting.financial_report(actor uuid,e uuid,b uuid,raw text) returns jsonb` accepts
strict JSON (duplicate keys rejected before jsonb conversion), at most 8,192 bytes.
Required `report` is trial-balance/general-ledger/balance-sheet/profit-and-loss.
Point reports require `through_date`; range reports require `starts_on,ends_before`
(at most 366 days); GL also requires `account_id`. Optional `snapshot_revision` is canonical
nonnegative bigint text; `limit` is JSON integer 1..500 (default 100). Optional comparison
is `compare_through_date` or paired `compare_starts_on,compare_ends_before` for P&L.
GL comparisons reject. Unknown/null/wrong-type/incompatible fields reject P2100.

Foundation cursor is a **JSON object**, not the future HTTP base64url encoding:
`{revision:string,filter_hash:string,last_key:array of strings}`. It must match explicit
snapshot_revision and SHA256 of normalized request filters excluding limit/cursor/
snapshot_revision; SQL recomputes the hash. Account sort key is `[account_code,account_uuid]`, compared as text arrays under
C collation; GL is `[accounting_date,journal_uuid,padded_line_number]`. Cursor is untrusted; validate full
shape and last-key membership in the selected row set. Changed book revision
raises P2114; authorization is always checked first. Cursor follows keyset order,
never offset; next cursor is null at exhaustion. Changing page size is permitted.

Envelope fields: contract_version=`financial-reporting-sql-v1`, report, entity_id,
book_id, snapshot_revision(string), generated_at(UTC timestamp), basis, timezone,
functional_currency/code version/minor units, policy_ids, filters, totals,
comparison(null or `{totals,difference}`), rows, next_cursor. All amounts are
canonical decimal strings, magnitude<10^29, scale<=9. Totals cover full selection,
not current page. Each row has stable `id`, `amounts` object and `selector` with
scope/revision/date/account or category filters. Account rows also carry
account_id/code/name/category/normal_side. GL row includes journal_id,line_number,
accounting_date,posted_at,kind,reverses_journal_id,policy_id,transaction currency
code/version, original amount,functional amount,side,valuation_basis,rate ID and
running_balance. Comparison account rows include `{amounts,difference}` aligned
by stable rowid and zero-filled. No percentages. BS derived unclosed_earnings row
is cumulative residual with revenue/expense selector, not current period profit.

## SQL ownership and policy

Repeatable-read transaction authorizes using existing locked membership/book
function before aggregation; no READ ONLY mode because authorization locks rows.
Only wrapper EXECUTE granted to bookkeeping_api, helpers owner-only. Current
synthetic accrual/UTC/USD-scale2/half-even profile is supported, original foreign
identity/spot amounts retained. Validate contiguous policy coverage through both
comparison/current history and reject currency/version/basis/scale changes.
No new non-USD functional book policy, annual close, translation or automation.

SQL calculates exact debit-positive sums. TB totals use max(net,0)/max(-net,0).
GL opening is all history before start; running totals include rows before page.
P&L revenue=-revenue-net,expense=expense-net,profit=revenue-expense. BS assets=
asset-net,liabilities=-liability-net,directequity=-equity-net,earnings=-(revenue-net+
expense-net),totalequity=directequity+earnings. No balancing plug or absolute-value
statement rendering. Original+reversal count on own dates. Empty valid periods
return zero activity and real opening. Dates are explicit, pointinclusive/rangehalfopen.

## Bounds and Go service

`Service.FinancialReport(ctx,Command)` passes trusted Actor/Entity/Book plus unchanged
Body. Total budget 9 seconds plus bounded rollback cleanup, at most 2 transaction attempts,
statement timeout 5 seconds / lock timeout 1 second. Commit acknowledgement required before returning
Result; cancellation/uncertain commit returns safe failure. SQL P2114 maps to
409 report_snapshot_changed locally without altering old error vocabulary.
Response at most 2 MiB; oversized/aggregate overflow raises result_out_of_range. No
financial arithmetic or cursor trust in Go. SQL has no ledger/receipt/outbox mutation.

## Acceptance and limits

Actual disposable PostgreSQL tests cover golden1300TB/70profit/1145BS, GL openings,
reversal/provenance, exact cents, comparison, deterministic pagination/stale revision,
unauthorized entity/prepare-only/revoked membership, malformed requests/cursors,
restricted helper grants and unchanged state. Unit tests cover adapter error/retry/
commit boundary. Full 100,000-row performance/hosted contention, FX functional transitions,
all future report matrix cases and CSV remain separate #51/#53 acceptance gates.
No new indexes without measured need; full scoped aggregates may timeout explicitly.

## Reviewed field refinements

Coordinator reviewed this foundation on 2026-10-13 before DDL. Coverage starts at
min(requested start/asof, comparison start/asof, earliest posted journal in the scoped book)
and ends at max(selected last date); it does not start at year 0001. Requests before
first policy fail unsupported_policy. All policies spanning that interval must cover
it contiguously and match supported profile/currency. Existing unused policy gaps
outside that interval do not block reports.

Exact `amounts` keys: TB `net,debit_balance,credit_balance`; GL
`debit,credit,running_balance`; P&L account/BS account/derived row `balance`.
Totals: TB `total_debits,total_credits`; GL
`opening,period_debits,period_credits,closing`; P&L `revenue,expense,net_income`;
BS `assets,liabilities,direct_equity,unclosed_earnings,total_equity`.
Comparison amounts/difference use exactly the current row keys; comparison totals/
difference use exactly current totals keys. Selector fields are entity_id,book_id,
snapshot_revision,starts_on(optional),ends_before(optional),through_date(optional),
account_id for account/GL or categories array for derived earnings. Envelope filters
exclude cursor/limit/snapshot_revision and retain the validated report/date/account/
comparison keys. Currency fields: functional_currency, functional_currency_version_id,
minor_units. GL provenance fields use transaction_currency and
transaction_currency_version_id plus transaction_amount and functional_amount.
`line_number` and minor_units remain bounded JSON integers; all money/revisions text.
Acceptance above is required/planned until the runbook names observed outcomes.

## Explicit foundation materialization limit

This implementation uses complete scoped row materialization before keyset page
selection; it is **not** a streaming large-ledger query engine. A preflight LIMIT 4097
count rejects more than 4,096 GL posting lines in the requested range/account, or
more than 4,096 accounts in a book, with result_out_of_range even when page limit 1.
Point reports aggregate the book's full historical journal facts into those bounded
account rows; GL opening history is also fully aggregated, never truncated. Scoped
account names/codes above 1,024 UTF-8 bytes reject account reports before materialization.
The BS derived row adds at most one row. JSON response remains capped at 2 MiB.
Comparison builds at most two bounded sets. Aggregate scans still need the 5-second
deadline; there is no 100,000-row performance claim. Replacing bounded materialization with
measured SQL page/window selection is explicit follow-up before raising these caps.
Cursor hash binds contract_version/entity_id/book_id/filters, not filters alone.
Policy provenance additionally caps 1,024 intersecting policy revisions across the
required history/comparison interval, checked before JSON accumulation; 1,025 fails
result_out_of_range. This is independent of workspace UI limits and cannot be
bypassed by directly provisioned policy rows.
