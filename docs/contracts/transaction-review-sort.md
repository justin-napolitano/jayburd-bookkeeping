# Transaction review sorting —406

2026-11-02 source proposal; concrete security-sensitive SQL/API patch requires
human review before merge/apply. Existing source-scoped review only;304/305's
pre-book/cross-book taxonomy dependencies remain incomplete. No live reads or DDL.

Sources reviewed: PostgreSQL17 sections7.5 ORDER BY and7.6 LIMIT/OFFSET
(https://www.postgresql.org/docs/17/queries-order.html and
https://www.postgresql.org/docs/17/queries-limit.html), full short sections;
W3C APG sortable-table example, explanation/semantics/keyboard/limits sections
(https://www.w3.org/WAI/ARIA/apg/patterns/table/examples/sortable-table/), selected
2026-11-02. They establish deterministic multi-key ordering and native button/
aria-sort semantics, not authorization, pagination consistency or accessibility
conformance. Existing11/12 review_fact, list_review_items, import_authorize and
snapshot increments, source-review-sql, standards/database-foundations reviewed.

Decision: optional paired query/body sort=date|description|direction|amount|status
and order=asc|desc. Omitted pair preserves exact legacy review-ID pagination and
old response. New UI explicitly chooses date/desc. Invalid/partial pairs denied.
Sort applies before limit to the whole authorized source/filter snapshot. Private
SQL helper sorted_review_page owns sort derivation, validation and current source
authorization. Existing definer list_review_items dispatches sorted requests to it;
helper has no runtime/PUBLIC EXECUTE. Managed readiness requires the helper present
and inaccessible directly. No new table, DML privilege or financial mutation.

Date uses original booked_date; text uses deterministic C collation; description
sort is case-folded then review UUID; status/direction use their closed raw codes.
Amount groups ISO currency then exact original positive numeric amount (reverse
both for descending), never float/FX or cross-currency value ranking. Every order
ties on unique review UUID ascending. Ranked query computes one authorized ordered
set, locates the cursor's prior UUID and returns at most100 summaries after it.
No unbounded result transport; full-sort cost is acknowledged, not a scale claim.
The first3000-row run exposed per-row review_fact overhead during ranking.
Use the same assignment→observation→current-revision joins in one set-based
query after a scoped grouped cardinality guard (exactly one current fact per
review). Selected summaries still use the existing authoritative renderer.
Missing/ambiguous facts fail closed; duplicate joins cannot become extra rows.
Physical refinement before index DDL: the3000-row set-based variant reached the
existing5s statement bound with cold statistics. Existing assignment primary key
is(entity_id,book_id,observation_id), while review lookup joins by review_id.
Add nonunique B-tree banking.review_assignment_scope_review on
(entity_id,book_id,review_id); it changes neither cardinality nor evidence and
also supports the existing authoritative renderer. Down39 drops only this new
index/helper and restores12. PostgreSQL17 section11.3
(https://www.postgresql.org/docs/17/indexes-multicolumn.html), full short section
selected2026-11-02, explains leading equality-column access; performance benefit
is our hypothesis until the same independent volume test passes. No timeout
increase or disabled accounting constraint is accepted as a workaround.
Cold-statistics EXPLAIN showed scope-prefix nested scans with join keys applied
as filters, rather than complete-key index conditions. Use bounded correlated
lookups: at most two assignments per review (two is enough to reject ambiguity),
then at most one observation/current revision by their full existing unique keys.
Validate cardinality first; committed foreign keys guarantee those referenced
facts exist. The snapshot/source lock prevents changes between validation and
ranking. Limits never discard accepted source rows: ambiguity rejects the read.
This query shape is independently tested without ANALYZE or raised timeouts.

Cursor retains scope/source revision/filter/last UUID and adds only sort/order;
no amount, description or transaction date in URLs. Changing source/filter/sort
restarts from page one. Wrong sort/scope/filter or missing anchor denied; stale
source revision returns existing refresh-required error. SQL derives all ordering
values from currently authorized rows. Sorted response adds optional sort/order,
matching_count and start_index decimal strings (start_index0 for empty pages),
so the UI can say which rows are visible and explicitly identify the end. Legacy
unsorted projection unchanged, no history import completeness inference.

Alternatives: client sorting only the loaded page creates false global order;
loading all rows violates transport bounds; value-bearing cursors disclose private
sort values in URLs; OFFSET without snapshot/order binding permits misleading
continuation. Ranked scoped query trades full-sort work for a small private cursor
and exact unique order. Benchmark3000 synthetic rows and retain hardware/limits.

Standards STD01: internal read ordering has no applicable ISO20022 message/version.
Existing ISO4217 currencies, original units and approved proprietary source mapping
unchanged; no new ISO21378 export or normative text/conformance claim. STD02 closed
sort/cursor profiles; STD03 original exact values/dates; STD04 snapshot/replay/ties;
STD05 legacy compatibility and forward39 only.1–38 immutable; down39 restores12's
exact list function and removes only the new helper/index; no data/history erased.

STRIDE: private financial reads(high): retain current principal/entity/source locks
before ranking and validate anchors within the same scope; spoofed cursor(high):
strict closed keys/sort/filter/source revision; injection(high): static CASE order,
no interpolated SQL; disclosure(high): no financial cursor values, persistence or
logs; DoS(medium): max100 rows/2KiB cursor, bounded server timeouts, disclosed full
sort cost. Journals/commands/grants/providers unchanged. Required review applies
to the generated SQL/API boundary, not a new bank-account authorization flow.

Named acceptance: TestSourceReviewSortedPages (all keys/directions/ties/multiple
pages/exact2versus10/legacy/invalid/stale/scope/no-effects); Go/bridge closed queries;
browser sort/page reset/aria-sort/keyboard/long labels/390px; synthetic app-integration,
migration-test/ledger-test, all locked web gates and core checks. Screenshots and
broader findings recorded in the stage runbook, without private live records.

UI direction: retain existing paper white #fff, ink #202124, muted #555b63,
line #dce0e3 and green #286044; inherit the system typeface. Use a dense reading
table with left-aligned descriptions/status, right-aligned original amounts,
nonwrapping dates, compact Review actions and44px controls. Keep real table
semantics, one aria-sort header, stable header DOM/focus during read refresh,
and an explicitly labeled keyboard-focusable horizontal scroller at390px.
Avoid new marketing/card layouts or palette changes. Review the scope against
the existing workspace rather than redesigning unrelated components.
