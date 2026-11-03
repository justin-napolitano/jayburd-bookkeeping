# Accounting category recovery —416

## Decision and source evidence

Source implementation2026-11-03, human review pending. This is accounting draft
eligibility, separate from planned spending/topic labels. Existing migration11
`banking.review_categories` requires exactly one book policy covering the original
booked date and excludes inactive/source accounts. No policy or ambiguity raises
P2109; `internal/ledger/service.go` maps this to422 `unsupported_policy`.
`TestHistoricalCurrencyAcquisition` independently proves older detail can be
imported while its draft rejects and a later date succeeds, with no journal.
These repository facts reproduce one cause; the hosted audit's actual response,
version and dependency state remain unconfirmed. No private/provider read needed.

## Browser mapping and boundary

The shared client retains only the closed code `unsupported_policy` from GET422
`application/problem+json`, streamed at most2048bytes then parsed. Missing,
malformed, oversized, interrupted or unknown-code errors fall back to fixed copy;
401/403/404 and all POST error bodies remain unread. No raw title/detail/request ID
is exposed, persisted or logged. This revises291's blanket unread-body rule only
for this bounded projection. Status and uncertainty retain their current meaning.
The production Next bridge independently restricts this projection to the exact
GET category path and422 problem media type, with the same2048-byte cap. It emits
only fixed code JSON; no upstream detail or request identity crosses the bridge.

Policy rejection explains the book policy cannot support a draft on the original
transaction date; review policy coverage with the book owner. It never asserts a
new date is valid, repairs history or changes policy. Other category-read failures
retain readable detail, offer an explicit read retry and explain a draft cannot
be created. Empty eligible account lists explain owner chart-of-accounts review;
there is no fake create-account action. FX remains explicit with original units.
Category-read authority loss continues clearing detail and invoking the existing
denied path. Stale async responses, scope changes, command locks and unchanged-key
draft retry remain fenced. Retry rereads authorized detail and categories only.

Alternatives: infer every422 means missing policy (incorrect for limit/FX), expose
raw backend diagnostics (unbounded/private), or silently move accounting date
(changes accounting intent). Closed-code mapping adds no API wire or SQL change.

## Standards and acceptance

No new financial exchange/field, currency, accounting date, journal or export.
Existing [source-review standards/mapping](source-review-sql.md) and
[read semantics](read-error-copy.md) remain applicable; no new ISO20022 message,
ISO4217 code-list or ISO21378 audit mapping/conformance claim. Existing internal
RFC9457 problem response is consumed by closed code, never financial authority.

API tests prove allowlist, body cap/cancellation, malformed/unknown fallback,
authority and mutation non-reading. Browser TestCategorizationReadRetry,
TestCategorizationDenied, TestMissingCategoryGuidance,
TestCategorizationUnavailableKeepsDetail and TestCategorizationNoAutomaticDraft
cover correct scope, exact money, policy/transient/empty/FX, no writes and locks.
291's TestPendingCommandStillGatesWrite proof and actual app recovery integration
remain required. Existing disposable acquisition SQL history proof is rerun;
no migration or live policy approval. Hosted diagnosis remains explicitly open.
