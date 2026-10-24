# Transaction detail when categorization is unavailable

## Decision and scope

#293, 2026-10-24. Authorized source detail is independently readable from its
category choices. Preserve the exact already fetched detail when a non-auth
category read fails, show an explicit read-only explanation and retry action,
and clear category/draft/retry state. Retrying reloads current detail and choices;
it never invents accounts, changes dates or executes a financial command.

401/403/404 still erase all scoped snapshots and invoke the existing denial
boundary. Book/session/source changes and obsolete request generations must never
restore old detail or choices. Existing revision/permission checks govern any
later successful draft. No raw API/provider diagnostics displayed for category
failure. No financial request or token enters browser persistence.

## Evidence, alternatives and standards

Reviewed source-review-sql.md and command-recovery.md plus migration11's
banking.review_categories: it selects policy effective at the original booked
date and raises P2109 if none applies. This is a source fact, not confirmation
of the actual older-row rejection observed in the hosted browser; that response
was not captured. Preserve historical observations without changing posting policy.
Existing research/standards/accounting mappings remain authoritative.

Erasing the whole detail loses independently authorized review evidence. Guessing
another date/account would bypass policy. Choose a separate unavailable category
state, and keep genuine authority denials destructive to sensitive UI state.
This is a read-only UI availability change, not a new financial message, API,
SQL schema, accounting rule or ISO20022 mapping. Existing ISO4217/original units
and exact money rendering are unchanged; no new conformance claim.

## Acceptance

Component tests cover delayed lookup, policy/transport failure, explicit retry,
401/403/404, changed scope and obsolete category responses. Browser fixture uses
history before a book's period plus an unavailable category endpoint, on desktop
and mobile: detail/exact amount/date remain visible, no draft or command POST,
and keyboard retry remains available. Existing valid draft flow and required
core/web/full-stack regressions pass. Hosted historical-case response and owner
review remain separate evidence; a source correlation is not a policy diagnosis.
