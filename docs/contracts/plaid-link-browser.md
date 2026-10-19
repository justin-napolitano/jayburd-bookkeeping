# Plaid Link browser boundary

The managed page supplies an inert `ConnectionLink` adapter. Only the explicit
connect flow calls `open`, after SQL-backed Begin returns a Link token. Demo and
synthetic entry points continue to inject fictional adapters; they do not load
Plaid or inherit the managed adapter. Default server-disabled enrollment remains
unavailable; browser code does not bypass Link Events qualification or server
verification of Link/public-token association.

Reviewed [Plaid Web Link SDK](https://plaid.com/docs/link/web/) on 2026-10-14:
Plaid requires its script to load directly from its CDN and says that the Web SDK
is not versioned. `create` provides lifecycle callbacks and a handler with open,
exit and destroy. Success may contain a null public token for other products;
this Item enrollment profile rejects it. These are vendor mechanisms, not evidence
that a particular account has the required server Link Events capability.

The adapter loads only
`https://cdn.plaid.com/link/v2/stable/link-initialize.js`, with no bundled SDK,
arbitrary script location or fabricated integrity/version pin. SDK load and
initialization share a ten-second bound; an opened session has a five-minute bound.
Only one session per adapter may run. Exit/cancellation returns no token; errors
are fixed text and discard provider metadata/diagnostics. Completion, timeout and
abort destroy the handler. Abort also attempts forced exit, with terminal fencing
set before cleanup can invoke callbacks.

Link/public tokens remain in memory only. No URL, storage, analytics, console or
provider metadata retention is introduced. Scope/session/lock changes abort the
existing Connections lifetime; late callbacks cannot trigger its execute request.
Loading a third-party script cannot promise to undo downloaded JavaScript or erase
vendor memory; cleanup removes our handlers and the SDK iframe, not the browser's
entire execution history. A canceled attempt stays inspectable; it is not silently
replaced or retried. No OAuth redirect-resume storage or reconnect support is added.

This is a proprietary provider bootstrap protocol, not an ISO20022 financial
message. Source/currency/accounting standards are unchanged. Focused fake-SDK unit
tests and intercepted-CDN browser tests validate local lifecycle behavior; they do
not qualify real institutions, provider consent or a live Plaid environment.

## Official Go SDK adapter (#195)

Reviewed 2026-10-19: official [plaid-go v49.0.0 source](https://github.com/plaid/plaid-go/tree/v49.0.0),
commit `0d83b050ea325adda35c6d88bdf24b46fe2f7044`, specifically Link/Item/Accounts
clients, request options, retry caller, and response models. The tagged README
still shows older generated API names; implementation follows the pinned source.
The generated client supports Hosted Link results, explicit credentials and HTTP
transport injection, and retry disabling. Its account balance model uses floats.

Use generated request/response types for connection operations, explicitly select
Sandbox, inject credentials/version, disable all retries, and bound HTTP latency
and response size. Keep duplicate-key/UTF8/JSON rejection at the transport boundary
and replace provider errors with existing static errors. Account responses are
projected to identity/currency metadata before SDK decoding; balance amounts never
enter float models or bookkeeping. Transaction ingestion retains its exact-decimal
decoder. This is transport replacement, not a new financial message or ISO claim.

Compared with retaining handwritten payloads, generated types reduce drift while
our SQL authority, one-shot exchange, encrypted persistence and recovery semantics
remain application-owned. Acceptance: existing wire/denial/session/consent tests,
explicit 429/500/transport-error no-retry cases, poisoned environment overrides,
and the synthetic browser/API/SQL enrollment/acquisition/posting journey. Hosted
provider acceptance remains a separate deployed test.
