# First bookkeeping interface

## Decision and scope

Implement #9's manual slice: authorized entity/book selection, controlled setup,
manual split journals, explicit review and posting, and trial balances. The browser
uses generated OpenAPI types and same-origin bearer requests; tokens and financial
state remain in memory. [Source review](../research/frontend-sources.md) records
framework decisions. Statement import, persisted drafts, journal browsing,
reversal UI and bank synchronization remain later work.

## Request and session boundary

Review freezes one idempotency key and exact request body. Only confirmation sends
it. Timeout, transport failure or server uncertainty retain both for unchanged
retry; editing and scope switching are locked. Only confirmed success or a
definitive rejection permits new work. Request keys are visible for recovery.
A denied retry after an uncertain result does not establish that nothing committed.

Session changes clear all financial views. Tokens, drafts and retry bodies are
never persisted. Signing out or closing the tab can therefore lose retry material;
the UI warns users to verify the earlier request with an authorized operator before
entering it again. This recovery limitation blocks live-use readiness. The browser
cannot override denied membership or substitute a fake successful posting.

The server-only `BOOKKEEPING_API_ORIGIN` chooses a trusted HTTPS backend or explicit
loopback HTTP endpoint. The browser never supplies that target. An unavailable
backend produces a visible failure. Financial responses use no-store; cookies
provide no authority. Authentication follows [browser auth](browser-auth.md).

## Accounting and standards

Amounts and revisions remain strings. No browser sums, currency conversions or
rounding determine posting acceptance. Setup explicitly uses the reviewed
accrual/UTC/USD development policy, selecting installed currency metadata. Manual
spot journals preserve original currency and require an operator-reviewed rate ID.
The UI is an internal application interface, not an ISO 20022 financial message;
ISO 4217 currency semantics follow the existing API and standards contracts.

## Acceptance

Playwright component journeys cover setup review, split/review/post, identical
unknown-outcome retry, and denied entity switching. Unit tests cover safe transport
errors and session invalidation. TypeScript, generated-contract drift and production
build gates complement the database suite; none proves live provider/deployment
behavior. Observed results belong to [the branch runbook](../runbooks/first-app.md).
