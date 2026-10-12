# First bookkeeping interface

## Decision and scope

Implement #9's manual slice: authorized entity/book selection, controlled setup,
manual split journals, explicit review and posting, and trial balances. The browser
uses generated OpenAPI types and same-origin bearer requests; tokens and financial
state remain in memory. [Source review](../research/frontend-sources.md) records
framework decisions. Statement import, general persisted drafts, journal browsing,
reversal UI and bank synchronization remain later work.

## Durable commands and session boundary

The [command recovery contract](command-recovery.md) owns prepare, execute, cancel
and acknowledgement semantics. Load pending commands before enabling new work.
Review registers one immutable command; only explicit confirmation executes it.
Committed or cancelled outcomes remain visible until explicit acknowledgement.
New work is blocked while discovery fails or any command remains unacknowledged.

Uncertain registration retains the same key/body for retry. Uncertain execution,
cancellation or acknowledgement discovers server state; it never automatically
executes, cancels or acknowledges. A prepared command recovered after sign-in
still requires confirmation. A commit winning a cancellation race displays its
committed result. Another tab's acknowledgement may release the server-owned gate.
Current authorization applies to every discovery and transition; inaccessible
pending work blocks new work until authorized operator restoration.

Session changes clear browser financial views. Tokens and request bodies are never
persisted in browser storage. Server discovery recovers registered work after
session/tab loss; unregistered form edits are not durable. The application does not
substitute a fake success or silently create a new request after uncertainty.

The server-only `BOOKKEEPING_API_ORIGIN` chooses a trusted HTTPS backend or explicit
loopback HTTP endpoint. Browser input cannot choose that target. Authentication
follows [browser auth](browser-auth.md); cookies provide no authority.

## Accounting and standards

Amounts and revisions remain strings. No browser sums, currency conversions or
rounding determine posting acceptance. Setup explicitly uses the reviewed
accrual/UTC/USD development policy with installed currency metadata. Manual spot
journals preserve original currency and require a reviewed rate ID. These internal
operations are not ISO 20022 financial messages; ISO 4217 semantics follow the API
and standards contracts.

## Acceptance

Named Playwright journeys cover setup registration/confirmation/acknowledgement,
split review, identical uncertain registration retry, lost execute/ack responses,
prepared recovery after remount, commit/cancel race presentation and denied scope.
A separate real PostgreSQL→Go→Next→browser harness verifies durable response-loss
recovery. Unit tests cover safe transport and session invalidation. TypeScript,
generated-contract drift and production builds complement database proofs; live
Firebase/GCP and live-use policy remain separate gates. Results belong to the
[command recovery runbook](../runbooks/command-recovery.md).
