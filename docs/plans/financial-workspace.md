# Financial workspace design

## Decision

Propose a financial review workspace for #52 under epic #50. Owner review is
pending. Open the [self-contained prototype](../prototypes/financial-workspace.html)
locally; it makes no requests and adds no application route. The existing
Next.js authentication and durable-command boundaries remain authoritative.

## Work

Navigation covers overview, accounts, review inbox, reports and reconciliation.
Entity/book, period and currency context stay visible. All-LLC cards retain
separate books; no consolidation or currency pooling. Reports distinguish an
as-of balance sheet from period income. Exact fixed golden-month expectations
are copied independently of posting code; the prototype does not compute an
accounting report or claim backend pagination support.

Journey: select Fictional A → Reports → Checking → AC-02 → account journals →
Reports. Period and currency selectors survive return. Switching entity, period,
currency or simulated response state clears detail immediately. A changed-book
state requires a fresh snapshot; future runtime requests must enforce current
authorization and revision consistency under #51.

Review-inbox states are explicitly illustrative. Reconciliation, exports and
close actions remain pending and cannot be executed here. USD original and
functional units match only because this fixture is USD; multicurrency and FX
policies are still required. February is missing fixture data, never verified
zero activity. Loading, error, denied and changed-book states hide all totals.

## Done when

[FW-DOM evidence](../runbooks/financial-workspace.md) verifies fixture equality,
drill/return, scope clearing and display states. Native keyboard controls,
visible focus, captions and responsive layouts follow
[FW01–FW02](../research/financial-workspace-sources.md). Semantic tables were
chosen over card-only reports for comparative review.

Owner design approval, real browser keyboard/mobile review and runtime journeys
remain gates; this partial prototype does not close #52. Implement reviewed
#51 read contracts before connecting live report views. Add named browser tests
for authorized report → journal → return, denied/revoked scope, revision refresh,
empty/error/loading and entity switches. Production accessibility and runtime
correctness cannot be inferred from the static prototype.
