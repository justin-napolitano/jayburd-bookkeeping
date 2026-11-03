# Source-scoped transaction search —408

## Decision and scope

2026-11-03 source proposal, sensitive implementation requires human review before
merge/apply. One currently readable source in one book; no private pre-book or
cross-book registry/taxonomy is assumed. No financial write, provider request or
additional database. Original amounts/currencies/dates remain unchanged.
Initial entry retains the legacy sorted read. Expanding Search and dates then
explicitly applying (including empty filters) switches that source context to the
private query/coverage projection; sort/status/pages/refresh retain applied filters.
Editing fields does not apply them. Clear resets search/dates and page one; scope/
source changes discard them. Collapsed controls preserve the compact default feed.
The collapsed disclosure shares the Refresh action row at narrow widths, with
44px targets; expansion spans the toolbar. Default first-row/action viewport
acceptance remains844px at390px width across platforms, not just local macOS.

`POST /v1/entities/{entity}/books/{book}/review-items/query` is a no-store read.
No query string, caller actor, idempotency key, browser storage, request-body log
or automatic retry. Fresh bearer/origin enforcement and current entity/source
authorization match existing reads. Maximum8192-byte UTF8 body, closed exact keys:
`source_account_id`, `state` (null or existing enum), `limit`1..100, `cursor`
(null or bounded opaque hex), `sort`, `order`, `search`(0..100 characters literal),
`starts_on` and `through_date`(null or ISO calendar dates, inclusive, ordered).
Duplicate/unknown keys and malformed values are rejected by SQL's existing strict
JSON boundary; Go preserves raw JSON. Legacy GET list remains unchanged.

The client exposes a separate query helper for this reviewed POST read so its
errors have read availability copy; ordinary POST remains mutation uncertainty.
No read can release the pending-command gate. Query results never register commands.

## Query, cursor and coverage

New migration40 adds a restricted SECURITY DEFINER read function. Principal,
entity/book/source locks and current permissions precede access. Static SQL uses
the existing unique-current-fact cardinality guard and bounded correlated joins
from39, then literal `strpos(lower(description COLLATE "C"),lower(search COLLATE
"C"))`. This is C-locale case folding, no regex/wildcard/fuzzy/merchant-identity
claim; percent/underscore/SQL punctuation are literal. Dates filter immutable
booked_date, independent of posting policy/periods.

All filters apply before ranking/count/limit. Sort/UUID tie breaking retains39's
exact original-currency semantics. Cursor pins entity/book/source, source revision,
state/sort/order/search/date bounds and last UUID. It is carried only in POST body,
never URLs, and is not authority. Changed filters/foreign scope/invalid anchor
reject; source revision change requires refresh. Result at most100 rows; existing
statement timeout bounds full ranking work, benchmark3000 cold-statistics rows.
No new scale guarantee or raised timeout. No financial sum is calculated.

Response retains ReviewPage fields plus `available_from`/`available_through`
nullable original-date bounds and `source_count` decimal string over this source's
readable current review facts, independent of selected search/state/date filter.
`matching_count`/`start_index` describe the full matching snapshot, never just the
page. Zero matches keeps observed source range. Empty source has null bounds/count0.
Availability is observed imported history, not provider completeness, bank balance
or reconciliation. This contract does not expose a last-successful-sync timestamp;
UI says it is unavailable rather than reusing read-generation time.

## Sources, alternatives, standards and threats

Reviewed2026-11-03: [RFC9110 §17.9 and §9.3.1](https://www.rfc-editor.org/rfc/rfc9110.html),
full subsections: sensitive URI disclosure and body-based query tradeoff;
[PostgreSQL17 §9.4](https://www.postgresql.org/docs/17/functions-string.html),
lower/strpos/length entries: literal substring and locale folding. Existing
[sorting review](transaction-review-sort.md) and database S06/S07/S12 govern
snapshot/auth/owned functions. Fact: these mechanisms exist. Inference: a bounded
POST read avoids financial query values in URLs while preserving source authority.
Neither transport nor deterministic sorting establishes complete bank history.

Rejected: GET merchant/date parameters (private values in URI); client-only page
filtering (false completeness); LIKE wildcards (unexpected matching); separate
search index/service (unneeded duplicate financial state); cursor without filters
(wrong continuation). New function/down40 only;1–39 immutable. No new tables/DML.

STD01: internal scoped read has no applicable ISO20022 exchange/version; original
ISO4217 currency references and ISO8601 date profile unchanged. No functional FX,
new audit export/ISO21378 mapping or normative conformance claim. STD02 closed
query; STD03 exact units/date fixtures; STD04 snapshot/filter/authorization;
STD05 legacy compatibility/down40. STRIDE: high disclosure/spoof/injection risks
mitigated by no URL/body logging, strict static query, current source authorization,
closed scoped cursors; DoS bounded body/substring/page/timeouts. No new privilege
to journals, credentials or provider payloads. Human review before merge/apply.

## Acceptance

Fixed independent search/date/currency/tie/empty fixtures plus3000-row pagination,
literal punctuation, invalid dates/keys, foreign/revoked/stale/filter-changed cursor,
legacy and down/up replay. Assert zero journals/outbox/provider effects. HTTP/Next
tests reject query strings and oversize/unauthenticated requests, preserve raw body
and redact diagnostics. Browser explicit Apply/Clear, keyboard/mobile, context/
sort/page reset, no stale response, empty/error/read retry and exact count/range;
no filter/cursor in URLs or persistence. Core, migration/ledger, locked API/types/
web check/test/build/browser and actual app integration required.
