# Financial reporting contract

## Decision and status

**Proposed for #51; design only, 2026-10-13.** Go serves scoped reports from the
existing PostgreSQL ledger. No reporting ledger, new service or client-side
accounting engine. Owner review of this contract precedes SQL/OpenAPI work in
#53. [Issue #50](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/50)
owns the application; #1 owns live accounting policy, #10 close/export acceptance,
and #59 advanced FX/consolidation. [Sources](../research/financial-reporting-sources.md)
FR01–FR05 distinguish evidence from project choices.

The existing `reporting.trial_balance(actor,e,b,through_date)` in migration 2
returns cumulative net debit/credit balances, authorized book revision and
functional currency. It includes originals and reversals and limits totals below
10^29. This contract adds proposed report behavior; it does not reinterpret that
endpoint or claim the new routes/functions/tests exist. All merged SQL stays
immutable; implementation requires a new migration and reviewed OpenAPI types.

## Scope, dates and policy

Every request names exactly one entity and book. No arrays, implicit all-LLC sum,
caller-supplied principal or mixed-scope partial success. Verified identity maps
to the existing principal; each report requires current `read` membership.
The first profile is `dev-accrual-usd-v1`: synthetic accrual, UTC accounting
calendar. Show basis, timezone, policy IDs and currency on every report.
Cash-basis conversion, tax statements, consolidated statements, presentation
translation and GAAP/IFRS compliance are not implied.

Inputs are real Gregorian `YYYY-MM-DD` dates, years 0001–9999. Range reports use
`[starts_on, ends_before)`; point reports use inclusive `through_date`. Compare
accounting dates, never browser timezone or `posted_at`. No default date from a
client locale. Point reports compare `<= through_date` directly, avoiding
9999-12-31 overflow. Reject inverted/empty ranges. Future dates are allowed for
review and do not manufacture future activity. Period state is informational;
closed does not mean audited. Unposted drafts never contribute. Original postings
continue contributing after reversal; a reversal offsets them on its own
accounting date. Opening values come from actual earlier postings, never a
balancing plug or inferred starting cash.

Select one supported basis/functional currency across all relevant policy
revisions, including the opening history. Missing policy coverage or a basis,
functional currency version/scale change without an approved transition mapping
returns `unsupported_policy`. Inactive accounts with history remain visible.
Multicurrency original units and immutable rate IDs remain in drill-down;
functional amounts are the booked amounts, never retranslated at today's rate.
Aggregate original units only within the same currency version; no cross-currency
unit total. Current SQL permits only positive original/functional amounts and
identity/spot valuation. Zero-unit revaluation, settlement and presentation FX
remain blocked by their own contracts.

## Amounts, signs and statement definitions

Use exact PostgreSQL numeric aggregates without coercing intermediate sums into
`numeric(38,9)`. Serialize amounts as canonical decimal **strings**, no exponent,
group separators, plus sign or negative zero. At most 29 integer and 9 fractional
digits; signed net fields may start with `-`; debit/credit fields are nonnegative.
Normalize zero to `"0"`, remove insignificant trailing zeros, reject overflow
with `result_out_of_range` before response. UUIDs and revisions are strings;
revision is a canonical nonnegative decimal integer string in the existing SQL
bigint range (including `"0"`), never a JavaScript number. Display pads
from currency metadata without discarding nonzero precision. All totals,
comparisons and running balances are checked against the same bounds.

For an account let `N(S)` be sum(debit functional amount minus credit functional
amount) for eligible lines in set S. Account normal side describes presentation;
it never changes posting arithmetic. Contra balances stay negative in statement
sections; do not take absolute values or suppress unexpected signs.

| Report | Exact required meaning |
| --- | --- |
| Trial balance | For each account through the inclusive date: `debit_balance=max(N,0)`, `credit_balance=max(-N,0)`. Include zero accounts existing in the accepted revision. Sum columns independently and require equal totals. These are **net balance** totals, not gross journal turnover. |
| General ledger | One required account filter; opening `N(date < starts_on)`, period debit/credit turnover, closing `opening + period_debits - period_credits`. Each row includes signed running balance in ascending `(accounting_date, journal_id, line_number)` order. A page's first running balance includes all prior eligible rows, not only that page. |
| P&L | Over the exact range, revenue is `-sum(N(revenue))`, expense is `sum(N(expense))`, net income is revenue minus expense. Owner capital, distributions, loan proceeds and transfers are excluded by account category. No tax/recognition adjustments inferred from labels. |
| Balance sheet | Through date: assets `sum(N(asset))`, liabilities `-sum(N(liability))`, direct equity `-sum(N(equity))`, unclosed earnings `-sum(N(revenue)+N(expense))` over **all** prior eligible history. Total equity = direct equity + unclosed earnings; assets = liabilities + total equity. Distributions reduce direct equity. The derived earnings line never creates a journal. |

The initial balance sheet is explicitly **unclassified**: the five account
categories do not establish current/noncurrent, cash-flow or tax classifications.
The derived line is called `unclosed_earnings`, not current-year profit. When
closing transfers revenue/expense to equity, residual revenue/expense naturally
falls to zero; do not add lifetime P&L again to transferred retained earnings.
Automated annual close, comparative pre/post-close earnings and fiscal-year
classification require #10's closing-journal policy before support. Unsupported
closing semantics fail rather than silently misstate P&L.

Report line identity is `account:<UUID>` for account rows and fixed versioned
section IDs (`assets`, `liabilities`, `direct_equity`, `unclosed_earnings`,
`revenue`, `expense`, `net_income`, `total_equity`). Labels are not identifiers.
First layout `basic-unclassified-v1` sorts accounts by code under bytewise `C`
collation then UUID; GL sorts by the tuple above. No caller SQL/order expression.
Every subtotal carries a server-generated selector: scope, same revision,
relevant dates, category/account IDs and sign multiplier. Account selectors lead
to GL; category/derived selectors first list contributing accounts, then GL.
Journal references resolve through the existing authorized journal read; the UI
never guesses membership or derives inclusion from a label. Original amount,
currency version, rate ID, policy ID, journal kind and reversal link accompany
line detail. Source documents remain references only when separately authorized;
missing source evidence is explicit, never fabricated.
The future report drill-through wrapper must validate the pinned revision and
call the authorized journal reader in the same transaction; the current standalone
journal endpoint has no report-revision parameter and alone cannot prove this tie.

## SQL, authorization and snapshot contract

Proposed restricted entry point:
`reporting.financial_report(actor uuid, e uuid, b uuid, request jsonb) returns jsonb`.
A strict tagged request selects TB/GL/BS/P&L, validated dates, account selector,
optional comparison and expected revision. Go validates transport and injects
trusted identity; SQL owns scope, membership, arithmetic and selection. No arbitrary
SQL or dynamic identifiers. Use the existing restricted owner/function pattern,
fixed safe search path, revoke PUBLIC execution, grant only reviewed reporting
wrappers to `bookkeeping_api`; no direct table SELECT/DML/DDL grants.

Each request runs in one short **repeatable-read** transaction, calls
`ledger.authorize(...,'read',false)` before aggregation or account lookup, then
uses the authorized book revision. This function uses `FOR SHARE`: a PostgreSQL
`READ ONLY` transaction is incompatible. Use ordinary repeatable-read with the
restricted role and query-only wrapper; do not remove authorization locks to gain
a read-only label. It must not mutate ledger, receipt or outbox state. Bound retries
of the whole transaction on serialization failure; no partial rows on failure.
Authorization locks serialize against revocation/disable: revocation committed
before authorization denies; if a report holds the lock first, it may complete
before revocation commits. No promise of retracting already delivered bytes.
Every page/export rechecks membership even when book revision is unchanged:
permission revocation need not increment the book revision to invalidate access.

For cross-request continuity choose **revision validation**, not a retained MVCC
snapshot: initial response returns `snapshot_revision=R`; subsequent pages,
comparisons, exports and drill-down require R. After fresh authorization, if current
book revision differs, return 409 `report_snapshot_changed`, no financial body.
Never silently continue at a new revision. New backdated entries and setup edits
therefore invalidate navigation too. Existing account semantics are mostly
immutable but account additions/deletions/active-to changes are not historically
versioned; filtering journals by revision alone cannot reconstruct the whole report.
No historical report request is promised. The UI offers an explicit refresh,
discards the old page set and starts again. Frequent writes may cause restarts;
measured demand can justify a separately reviewed snapshot/export mechanism later.
Do not hold database transactions open between browser requests.

First-page results and their totals/comparisons are one snapshot. Subsequent pages
use a versioned base64url cursor containing last sort tuple, revision, report kind,
layout and canonical filter digest, max 2048 bytes. Treat it as untrusted input,
validate all fields and require exact filter equality; it is **not** an access
token or authorization proof. Changing a cursor cannot access another scope or
avoid current permissions. Keyset pagination only, no offsets. Order ties include
UUID and GL line number so retries yield neither duplicates nor missing rows.
HTTP buffers the bounded response before commit and emits success only after
transaction success. Set `Cache-Control: no-store`; no tokens/rows in logs,
URLs copied to analytics, browser persistence or shared caches.

## Proposed HTTP surface and bounded execution

Proposed additive GET paths under `/v1/entities/{entity_id}/books/{book_id}/reports/`:
`trial-balance`, `general-ledger`, `balance-sheet`, `profit-and-loss`. Existing
trial-balance route remains backward compatible. Point routes accept `through_date`;
range routes accept `starts_on`, `ends_before`; GL additionally requires `account_id`.
Common optional `snapshot_revision`, `cursor`, `limit` (default 100, maximum 500).
Reject unknown/duplicate query fields, malformed UUID/date/decimal revision and
incompatible filters with 400 `invalid_request`.

Comparison is one explicit `compare_through_date` for point reports or paired
`compare_starts_on`/`compare_ends_before` for P&L; GL comparisons are unsupported.
All comparisons use the same authorized revision and currency/basis/layout, align
by account/section ID, and zero-fill an absent balance. Return current, prior,
and signed `difference=current-prior` as strings. Omit percentages in v1 to avoid
undefined denominators and misleading sign changes. Reject unsupported policies;
never perform implicit currency translation to make comparison work.

Envelope: `contract_version="financial-reporting-v1"`, report/layout IDs,
entity/book IDs, functional currency code/version/minor units, basis/timezone,
policy IDs, normalized date/comparison filters, `snapshot_revision`, UTC RFC3339
`generated_at`, `next_cursor` or null, exact totals and `rows`. Each account row
includes identity/code/name/category/normal side, amount fields appropriate to
report and selector. GL rows include the provenance listed above. `generated_at`
is generation time, not an accounting cutoff or a claim of currentness after commit.
Default first-page requests capture current revision; pinned requests never auto-refresh.

Reuse API problem semantics: 401 invalid credentials; 404 absent/inaccessible
scope/account; 403 known membership lacking read; 409 changed snapshot; 422
unsupported policy or result bound; 503 deadline/retry exhaustion. Error bodies
contain request ID and fixed problem code only, no financial values or existence
hints. Scope authorization precedes snapshot mismatch disclosure. Timeouts never
produce a zero/partial report. Request cancellation cancels SQL work.

Initial engineering limits: one book, one comparison, range <=366 days, one GL
account, <=500 rows/page, 5-second database statement timeout, 1-second lock timeout,
10-second request deadline, at most 2 whole-transaction attempts within deadline,
2 MiB encoded JSON response. Point/opening history is not truncated to 366 days.
These are proposed limits, not measured capacity. Existing indexes `journal_date`
and `account_history` are candidates; implementation must EXPLAIN synthetic
1,000- and 100,000-journal scopes, prove entity/book predicates early, measure
opening/running sums and shared-core contention. Pagination does not itself bound
aggregate scan cost. Add only measured composite indexes in a new migration;
if limits fail, return explicit unavailable and revise the query design, not totals.

## CSV review export

Proposed `GET .../reports/{kind}/export.csv` uses the same scope, filters, explicit
revision and authorization; ignores no unsupported filters. Export contains all
selected rows, never just the visible page, with max 10,000 data rows, 10 MiB UTF-8
bytes and the same 10-second deadline. Detect excess before emitting headers/body;
return 422 `report_export_limit`, never a truncated success. Buffer in bounded
memory; no background job, bucket, share link or browser persistence in this slice.
A repeated download repeats authorization and may fail after revocation.

The implementation contract must pin a stable header order and RFC4180 quoting/CRLF;
the proposed `financial-reporting-csv-v1` profile
repeats provenance columns (report/layout/scope/revision/currency/date filters and
generation time) on data rows. Empty exports contain header plus one explicit
`row_kind=summary` row with actual opening/closing totals and provenance (all zero
only for empty inception). Define row kinds account,
posting, subtotal, summary so totals cannot be accidentally counted as postings.
Machine reconciliation uses exact decimal fields and UUIDs, not display labels.

To avoid relying on fragile spreadsheet escaping, v1 export omits arbitrary
account codes/names/descriptions/reasons/source text; UUIDs plus fixed enum labels
provide drill-through. Magnitudes are unsigned canonical decimal; each measure
has a separate `<measure>_sign=positive|negative|zero` and `<measure>_magnitude`
pair. Posting debit/credit side remains a separate field. Dates,
UUIDs and enums follow strict generated grammars. Quote all fields, double embedded
quotes, reject control characters and any value outside its field grammar before
serialization. The approved header must be fixed. Thus no cell is user-controlled formula text;
a validator error rejects the entire file. Rich descriptive CSV requires a later
reviewed transformation and spreadsheet compatibility test; CSV is not promised
safe after arbitrary downstream editing, or exact numeric import by spreadsheets.
JSON remains the lossless contract. This is an internal **review export**, not an
approved accountant/audit exchange. ISO audit mappings below block that promotion.

**Remaining wire-schema gate:** before CSV implementation, specify and review the
ordered header, row-kind field/nullability matrix, provenance/comparison columns,
and the amount/sign pairs for opening, turnover, closing, current, prior and
difference. The profile name is reserved for this proposal, not an implemented
or fully specified wire schema. JSON report implementation and CSV delivery need
separate acceptance; scope and safety requirements above do not settle the missing
column mapping.

## Standards applicability and open policy

| Practice/message | Assessment, mapping and acceptance |
| --- | --- |
| Internal JSON/SQL reports and selectors | ISO 20022 message encoding: not applicable to this application-owned report/query exchange (FR04); no bank/network counterparty or message profile selected. Scope is a project interpretation for review, not proof no financial-reporting message could ever apply. No ISO namespace/version is invented. Internal schema is `financial-reporting-v1`; validate exact money, side, currency/date and reversal semantics in tests below. |
| Currency identifiers | ISO 4217 applicable: reuse immutable `reference.currency` code/version/minor units and recorded source release; no live unpinned code list. FR04/public overview does not approve FX policy. `TestReportCurrencyBoundary` rejects mismatches/unmapped versions. |
| Audit/accountant CSV | ISO 21378:2019 applicable research target; ISO 21926:2026 semantic mappings unresolved (bibliography R04/R05). Abstract/preview access does not settle normative fields. Preserve ledger IDs, dates, sides, currencies and provenance in internal review CSV; no claim of mapped ISO export. Required normative mapping/profile approval blocks accountant/audit export implementation, not review of this proposal. |
| Dates and identifiers | Use existing ISO-style Gregorian full dates/RFC3339 timestamps and UUID contracts; no new ISO identifier assigned. FR05 reviews RFC3339 §5.6/§5.7 and RFC4180 §2 for transport; these IETF formats are not accounting recognition rules. |
| Financial statements | FR01 supports introductory mechanics; actual framework/jurisdiction, fiscal year, taxonomy, live opening balances and annual close need owner/accountant decisions (#1/#10). ASC/IAS review remains pending; do not relabel synthetic statements compliant. |

Any unresolved required standards mapping blocks the corresponding implementation
slice under [standards.md](standards.md). Before a release, review this table,
record exact supported versions/profile and STD-01–05 evidence for the applicable
scope. Basic review reports and accountant-ready exchange are separate gates.

## Independent expected results and planned acceptance

Use fixed [golden-month.json](../../tests/corpus/golden-month.json) values, not
expectations recomputed by report SQL, generator or a shared aggregation helper.
For fictional A in January: checking 1045, savings 100, capital credit 1100,
services credit 200, general expense 100, category A/B expense 15 each,
distributions debit 25; supplies/card/payable zero. Trial-balance totals are
**1300 debit = 1300 credit**; P&L revenue 200, expense 130, profit 70; balance
sheet assets 1145, liabilities 0, direct equity 1075, unclosed earnings 70,
total equity 1145. Fictional B stays checking 70/capital credit 70. Hand-review
these constants separately from fixture-loading account classifications.

Checking GL `[2026-01-02,2026-01-03)` has opening 1000, debit turnover 200,
credit turnover 0, closing 1200. Empty February keeps January closing values,
zero activity/P&L and checking opening/closing 1045. Empty inception has all-zero
statements. A reversal dated in February leaves January as posted; February P&L
includes its offset exactly once. For cent/scale tests add independently reviewed
USD 0.01 + 0.02 = 0.03, JPY 1 + 2 = 3 and KWD 0.001 + 0.002 = 0.003 in
separate supported-policy books; non-USD policy setup is planned, not current
posting-profile support. EUR 10 booked at USD 11 retains EUR 10 and USD 11;
a later 1.20 rate never changes that report. Max amount plus 0.01 overflow must
fail as a whole report. Do not aggregate these books/currencies together.

All following names are **planned**, not claims of current coverage:

| Named evidence | Acceptance |
| --- | --- |
| `TestFinancialReportsGoldenMonth` | Exact constants above; TB = GL closings, BS balances, P&L ties to earnings; independent B isolation; all financial outputs strings. |
| `TestReportDateBoundaries` | Opening/empty periods/inception, inclusive point vs exclusive range, UTC/date input, leap/invalid/max dates; drafts absent. |
| `TestReportCorrectionHistory` | Original plus linked reversal and replacement, different period, exact original FX; no rewritten history/double counting. |
| `TestReportCurrencyBoundary` | Original-unit buckets, functional policy transitions denied, 0/2/3-decimal fixtures, overflow, no floating/exponent serialization. |
| `TestReportStatementSigns` | Contra asset/revenue, distributions and unexpected credit expense preserve sign; unclassified labels; planned closing transfer does not double-count equity. |
| `TestReportSnapshotPagination` | Equal-date ties, page concatenation equals fixed full results/running sums, tampered cursor denied, backdated post/setup invalidates R, comparisons/drill-through share R. |
| `TestReportAuthorizationRace` | Anonymous/unknown/disabled/revoked/prepare-only/post-only denial, mixed scopes and foreign account IDs; concurrent revocation linearization and later-page denial; no leaked totals. |
| `TestReportNoMutation` | API direct SQL/DML/DDL denial, exact SQLSTATE; report success/failure leaves ledger/receipts/outbox unchanged; lock serialization retries bounded. |
| `TestReportCSVReviewExport` | Full selection vs page, header/provenance/empty shape, decimal-sign round trip, malicious label omitted, malformed field rejected, row/byte limits all-or-nothing, revoked repeat denied. |
| `TestReportResourceBounds` | Synthetic query plans/timing, full-history opening, cancellation/deadlock/timeout, limits and redacted failures; measurements recorded without a cloud-capacity claim. |
| `ReviewFinancialReportingStandardsV1` | Owner reviews applicability/profile, source limits and deferred normative audit mapping; no blanket ISO/GAAP claim. |

## Alternatives and implementation order

FR02 explains transaction snapshots; our inference is that short revision-validated
requests fit the current mutable metadata and small pool. Keeping exported MVCC
snapshots alive avoids refresh but consumes connections/locks; durable materialized
reports add retention, revocation and storage obligations. Neither is justified
before measured failures. Client aggregation cannot be the accounting authority;
a warehouse/read replica introduces lag and a second authorization surface.

First approve the contract/standards interpretation and fixed constants; then add
restricted SQL, invariant tests and measured indexes, then reviewed OpenAPI plus
Go API/CSV tests, generated clients and browser integration. No generated OpenAPI
change in this design-only task: proposed paths are deliberately not advertised
as runtime operations. Owner/accountant policy and audit-export mappings remain
explicit gates; cloud acceptance is independent of these design tests.
