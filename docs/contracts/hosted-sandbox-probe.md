# Hosted Sandbox probe

## Decision

Owner explicitly authorized the actual Sandbox test and its missing wiring on
2026-10-14. The probe creates one disposable fictional Item through Plaid's Sandbox
API, exchanges its token, selects a USD account, runs the existing exact-decoding
sync diagnostic, and attempts Item removal. This bypasses Link deliberately; it
cannot qualify interactive Link, application persistence, webhooks or live books.

The job reads only two pinned numeric GCP versions using managed identity. It has
no source-manager access or secret writes. Credential delivery separately uses the
existing immutable intent/receipt adapter for fixed root keys PLAID_CLIENT_ID and
PLAID_SECRET. Lookup remains value-disabled; each delivery requires reviewed ID and
source version. Infisical project access must be supplied by the owner.

## Execution boundary

Explicit opt-in is required; no default networking. Only sandbox.plaid.com is used,
redirects are disabled, responses are bounded, errors are fixed codes, and tokens
and provider payloads remain in runtime memory. No database, journal, file output,
webhook or production URL. At most one Item creation/exchange per execution; no
retry after unknown acknowledgement. Cleanup gets its own bounded context after
an access token is obtained. A failed exchange may leave an orphaned Sandbox Item;
report that ambiguity without retrying creation. Cloud Run retries are zero.

## Sources and acceptance

Reviewed Plaid [Sandbox API](https://plaid.com/docs/api/sandbox/) public_token/create
and [Items API](https://plaid.com/docs/api/items/) public_token/exchange and item/remove
on 2026-10-14. API-created Items bypass interactive Link. Inference: this provides a
credential/provider smoke test without claiming application association proof.
Existing [qualification contract](plaid-sandbox-qualification.md) governs sync limits
and partial-only receipts. No financial interchange added; ISO 20022 does not apply.

Tests must demonstrate fixed Sandbox host, redacted output, cleanup after sync
failure, no retry after exchange ambiguity, no calls without opt-in, and exact
numeric credential pinning. Hosted evidence remains separate from injected tests.
