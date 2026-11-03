# Source activity API —429 implementation profile

## Decision and gate

Proposed2026-11-03; refines [spending metrics](spending-metrics.md) Phase A only.
The owner requested implementation after the430 handoff.428/430 remains a real
review/merge prerequisite, not a completed predecessor. This profile can be
reviewed before DDL; it does not itself install an endpoint or expose metadata.
The implementation will use migration41, reserved under65;40 belongs426.

## Closed transport

`POST /v1/entities/{entity}/books/{book}/source-activity/query` is a private read.
It requires a fresh verified bearer and the existing same-origin bridge; no query
string, Idempotency-Key, Content-Encoding, caller actor, command or automatic
mutation retry. Content-Type exactly application/json, UTF8 body at most8192bytes.
Duplicate/unknown keys reject. Request keys are all required:

| Field | Accepted value |
| --- | --- |
| profile | source-activity-v1 |
| source_account_id | canonical nonzero UUID |
| starts_on / through_date | inclusive real YYYY-MM-DD calendar dates; ordered |
| search | literal C-locale case-folded description substring,0..100 codepoints |
| kind | summary, observations or merchants |
| currency_version_id | null for summary; required UUID for observations/merchants |
| merchant | null or exact closed object: status and label |
| snapshot | null only for initial summary; otherwise exact closed object below |
| cursor | null or bounded hex cursor, at most2048characters |
| limit | integer1..100 for observations;1..20 for merchants;20 for summary |

Merchant status is attributed/not_provided/unavailable. Attributed has a nonblank
exact case-sensitive label of at most500codepoints; the other statuses require
null label. Merchant filters apply only to observations; summary/merchants require
null. Merchant keys never derive from description and never appear in URLs.

Snapshot has exactly source_revision, binding_revision and review_revision_sum;
all canonical nonnegative decimal strings, binding positive. The last is the exact
sum of source review-item revisions, including rows outside selected dates. It is
a consistency counter, never authorization. Current reviewed-row revisions are
monotonic; changes to overlap/accounting attention invalidate old metrics even
when imported facts have not changed. Sum uses PostgreSQL numeric, not bigint/float.
Source and binding revisions catch source updates and registry/binding changes.
Existing draft/post implementations already advance source_revision. The review
sum protects future reviewed transitions, not a confirmed current snapshot bug.
Immutable observation and reference metadata need no independent mutable pin.

Every cursor pins entity/book/source, the snapshot, canonical selector digest,
kind, unit and last group/observation anchor. The digest binds profile, dates,
search and exact merchant selector; it has no authentication/privacy claim.
Reject foreign/mismatched/missing anchors; current grants are checked first.
Do not use an offset as authority or accept a client-modified anchor as a grant.
Refreshing starts a new summary; a drill must explicitly reuse its summary snapshot.

## SQL grain, eligibility and safety

Use one source_observation joined to its exact current observation_revision and
installed currency version. Left-join review_assignment/review_item so pending or
zero observations without a review row remain inventory. Fail closed for dangling,
cross-source or multiply assigned current review facts; never multiply money via
a join. Review rows with multiple observations are ambiguous rather than additive.
The current source namespace mapping must agree with the observation's source.

First reuse import_authorize(actor,entity,book,source,false) under the existing
RepeatableRead read transaction, lock timeout and four-second service deadline.
Only then compare consistency pins, render counts or expose label metadata.
The restricted SECURITY DEFINER function has pg_catalog search_path, fixed SQL,
no user-controlled identifiers and EXECUTE only for bookkeeping_api. Helper
functions have no direct API/runtime grants. Ordinary app roles retain no direct
table access. Read path never calls a source/provider adapter or financial command.

Valid current booked, nonzero source amounts qualify if original precision and
currency validity match installed metadata and identity/overlap is unambiguous.
Precision/identity holds are counted separately; totals become partial. A valid
accounting correction hold alone does not exclude activity. A posted source
correction15 counts while its immutable original journal100 is untouched.
Pending, removed and zero are distinct inventory buckets, not inferred refunds.
Bucket precedence is removed, pending, zero, quality hold, then eligible. A booked
nonzero row without an unambiguous review assignment rejects the read; pending
and zero inventory may lack a review row.
All filtering/counting/aggregation precedes paging. No date policy, journal state
or current bank balance substitutes for original source facts.

## Closed result and drill semantics

All results carry profile, entity_id, book_id, source_account_id, source_state,
snapshot and selector; generated_at is UTC RFC3339 query time, never bank sync.
Return bank_balance, functional_total, categorized_spending and last_successful_sync
as null with explicit unavailable reasons. Available source date bounds describe
observed current imported history only; history_completeness stays unknown.

Summary returns inventory counts as decimal strings: source_count (all current
observations), matching_count (date/search selection), eligible_count, pending_count,
removed_count, zero_count, quality_hold_count and correction_attention_count.
The first five selected state/eligibility buckets partition matching_count;
correction attention can overlap eligible_count. Each unit group retains
currency_version_id, alphabetic currency, numeric code, minor_unit, count, inflow,
outflow and net_inflow as strings where quantitative. Currency metadata minor_unit
is the installed bounded integer, not a money conversion. Exact SQL decimal sums
remain strings through Go/JSON/UI. Unit groups are sorted by UUID, at most100;
overflow fails explicitly rather than returning incomplete apparent totals.
An empty source reports no observed groups, exact zero counts and null date bounds;
it does not invent a currency/version total from the source binding.

Observations are eligible contributions only, selected by exact unit and optional
merchant group. Each row has observation_id, observation_revision, booked_date,
description, source_state, direction, original amount, original currency metadata,
merchant status/label/profile, optional review_id and current review_state.
Return complete matching count, inflow/outflow/net, bounded rows and next_cursor.
Original amounts are the Phase A contributions; no loan/category allocation claim.
Sort booked_date then observation_id ascending, stable under the pinned snapshot.
The complete signed drill quantities must equal the selected summary/unit/group.

Merchants use only the stored normalized merchant_name from the supported
jayburd-plaid-transactions/1 profile. Extract that named value inside SQL; do not
return provider_fields, raw object, account IDs, URLs, logos or other enrichment.
Supported null/blank is not_provided; absent/unsupported profile is unavailable.
Malformed supported values fail closed. No raw refetch or description fallback.
Exact labels use C collation; case variants stay separate, whitespace is not
silently normalized except whitespace-only absence. Profile is
provider-merchant-exact-v1. Null-label buckets remain explicit and drillable.
Sort descending outgoing amount, then status/label in C collation for stable ties.
Return total group count and complete unit totals alongside page group totals,
next_cursor and exact remaining group count. This ranks outgoing activity, not
spending; refunds/inflows remain separately visible, including negative net_outflow.
Drills include unit plus exact status/label and the summary snapshot, never contains
search pretending to be merchant identity. Unsupported-only sources return an
unavailable metadata bucket rather than an empty attributed merchant success.

## Errors, applicability and named evidence

Reuse fixed source-read error family:400 invalid_review,404 review_denied,
409 review_snapshot_changed for changed snapshot/review_conflict for ambiguity,
422 review_limit for unsupported/overflow result and
503 temporarily_unavailable for deadline/availability. No underlying SQL/body,
label, cursor or provider content appears in errors/logs. Next preserves only
bounded approved errors and no-store; read failure never becomes uncertain posting.

ISO and source applicability reuse SM01–06 in
[the reviewed source register](../research/spending-metrics-sources.md), plus the
existing source-search transport/PostgreSQL sections. ISO4217 original code/version,
ISO calendar-day profile and RFC3339 timestamp mapping remain distinct. Internal
analytics is not an ISO20022 financial exchange; no new ISO designation or audit
export conformance claim. Full normative ISO text remains unread. This refines
mechanisms from reviewed repository schemas, not a new external accounting policy.

STRIDE/high: restricted data disclosure, forged source/filter and changed grants→
current server authorization, scoped minimal fields, fixed errors/private body;
tamper/high→closed selectors, immutable current grain and checked snapshot/drill;
DoS/medium→input/page/group limits, inherited deadlines,3000-row acceptance;
repudiation/high→original revision provenance, no command/financial writes.
Human security/source review before runtime merge/apply remains required.

Runtime tests are **not yet implemented**. Required evidence remains429's fixed
oracle, TestWholeResultNotPage, TestSpendingCurrenciesSeparate,
TestMerchantNormalizationVersion, TestAggregateDrillEquality,
TestFreshnessUnknownStale, TestAggregatePrincipalDeny, TestSourceRevisionSnapshot
and TestReadNeverPosts against actual restricted SQL/API/bridge. Required mutations
cover foreign/revoked scope before pin checks, changed date/search/unit/merchant,
stale source/binding/review revisions, duplicate keys, pending/removal/replay,
valid posted correction, null-vs-zero, full3000-row totals/30pages and zero financial
journals/outbox/provider/command acknowledgements. A contract or JSON parse is not
runtime evidence. Existing financial test corpus stays independent and unchanged.

Rejected: aggregating review pages (misses inventory); client sums (partial/float
risk); substring merchant drills (unequal sets); source_revision alone (misses
review hold changes); granting runtime full provider_fields reads (excess data);
hash as authority (untrusted); new mutable cache/table/service (unneeded second
authority). Revisit only with reviewed counterexamples and named acceptance.
