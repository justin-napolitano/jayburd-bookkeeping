# Daily source Overview —432 /418

## Decision and financial boundary

Source proposal2026-11-04, dependent on429/431 review/CI. This increment consumes
[source-activity-v1](source-activity-api.md); it adds no SQL/financial recognition,
new data collection or private/provider calls. Parent418 category/Spending and
comparison work and417 Phase B remain open. Earlier primary transaction navigation,
posted Position/reports and durable command recovery remain reachable.

One selected authorized book and source at a time. Default All imported history
sends explicit0001-01-01..9999-12-31 bounds, then shows observed source dates; it
does not infer bank completeness or use the host clock to hide imported dates.
Custom dates are visible drafts until Apply. Source/scope/lock changes destroy
the old headline, drill, cursor and async generation before fetching new data.
Unavailable balances/import times remain unavailable text, never ledger zero.

Use only the existing private POST query helper, fresh token and no-store bridge.
Extend its closed path allowlist to source-activity/query; never fall back to an
ordinary mutation POST. Filters/labels/pins/cursors remain memory-only and do not
enter URLs, storage or logs. Rendered response scope, selector and snapshot must
match the request; late/mismatched data cannot replace a current view. A stale
drill clears both headline and rows and requires an explicit refresh.

Money comes from complete server aggregates as decimal strings. Format original
units with the exact currency_version_id metadata; keep multiple versions/codes
as separate rows, no browser sums/valuation/percentage arithmetic. Each unit's
breakdown has an observation action retaining source, period/search, snapshot and unit.
Merchant groups preserve exact status/label and profile; matching amounts open an
exact merchant selector, never a description substring search. Paging retains the
same pins and exposes exact complete counts. Eligible observation order is the
wire's date/UUID ascending order; do not label its first page Recent. Latest
transaction review remains one navigation action away in Transactions.

Attention/pending/removed/quality counts are separate statuses, never added into
a fabricated combined count. Their general Review account transactions action
does not claim to be an exact-count drill. Quality exclusions keep the partial
warning visible. No category, spending, cash-available or saving claim follows
from raw outflow. The source's active/paused/retired state is described as its
availability in this book, never provider login health or last successful sync.

## Design and critique

Use the approved existing tokens: white#fff, ink#202124, muted#555b63,
line#dce0e3, green#286044, attention#9b3e38. Existing system UI font, sentence case,
tabular money; titles24px, sections20px, money22px, body16px, supporting14px.
Left-align context/labels and right-align money. No new fonts/images/dependencies.

Layout: compact book context and account/period toolbar; a currency/version table
for money in/out/net; one honest bank-balance/import-time explanation; attention
statuses and a primary transaction-review action; merchant activity and exact
transaction drill on demand. One h1 owned by the workspace. At390px, stack toolbar
controls and use responsive table/list presentation with44px targets, wrapping
merchant descriptions and contained table overflow. Native controls/disclosures,
visible focus, heading focus on drill, live read status and200percent text reflow.

Review against brief: a decorative row of generic metric cards would hide unit
differences and unknowns. Use aligned currency rows as the dominant visual element;
quiet boundaries and explicit account/period give a bookkeeping purpose. Categories
are not represented by a misleading empty chart. Keep task labels out of product
copy. Primary Overview complements the existing one-action Transactions entry;
posted tools retain their own labels and calculations.

## Applicability, sources and validation

Reuse [SM01–07](../research/spending-metrics-sources.md) and the existing source
API standards mapping: ISO4217 installed code/version/minor unit, calendar source
days distinct from RFC3339 query time. No ISO20022 exchange or audit export is
introduced; full normative ISO text/conformance remains unclaimed. Display logic
adds no recognition, rounding or date-shifting policy. Original source references
and independent oracle remain authoritative; the mock is explicitly fictional.

Threats: high disclosure/tamper→current server authorization plus response/snapshot
context checks, no persistence/URLs and stale-data destruction; medium availability
→bounded server pages, one request generation and explicit retries; high recovery
confusion→query helper read semantics, no commands/writes from opening or drilling.
Human source/security and visual review before merge/release remain required.

TestOverviewSnapshotScope, TestMetricDrillContext, TestUnknownNeverZero,
TestOverviewReviewAttention, TestOverview390Keyboard: fixed fictional large exact
strings, mixed units, source/period/snapshot changes, late responses, loading/empty/
partial/stale/denied/error, lock/unmount, keyboard/zoom/mobile screenshots and zero
mutations. Core and locked web/API/browser gates; actual browser→Next→Go→synthetic
SQL proof for the production client helper. No closure from mocks or screenshots.
