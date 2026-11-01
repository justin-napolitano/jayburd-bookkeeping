# One-shot Sandbox webhook probe — #385

Decision candidate2026-11-01; owner requests webhook repair before the personal
pilot. Existing375 scope authorizes the one fictional checking stream until
2026-11-02T04:00:00Z. This source adds no authority or Production activation.

## Evidence and choice before implementation

Reviewed official Plaid [verification](https://plaid.com/docs/api/webhooks/webhook-verification/)
ES256, key retrieval, original-body digest and five-minute age sections;
[Transactions](https://plaid.com/docs/api/products/transactions/#sync_updates_available)
SYNC_UPDATES_AVAILABLE fields/initialization and legacy compatibility sections;
[Sandbox](https://plaid.com/docs/api/sandbox/#sandboxitemfire_webhook) test firing
request/result sections, selected2026-11-01. These are mutable vendor API docs,
profile2020-09-14; no private payload, keys or full normative ISO text reviewed.

Facts: legacy INITIAL_UPDATE/HISTORICAL_UPDATE also fire for sync integrations;
SYNC_UPDATES_AVAILABLE requires an earlier sync and can be fired in Sandbox.
Hosted metadata: two pre-first-sync Transactions401s, key endpoint200, then three
sync200s. Inference: unsupported legacy events may explain401; signature failure
is not established. A fresh supported event is required to distinguish them.

Choose a separately invoked inert-by-default command in the existing worker image.
Reuse acquisition.Runner, SQL lease/current scope, encrypted credential resolver,
metadata identity and evidence store; never query a credential table or bypass a
lease. The one fixed Sandbox test request happens inside an authorized Fetch;
ordinary account-filtered sync follows. No change to AllowsSync or running workers.
Alternative owner-pasted tokens/Studio payload inspection violates the chosen
credential boundary; changing verification from401 observations would be speculative.
Creating another unbound Item cannot prove this book's receipt/wakeup chain.

## Closed operational profile and standards

Default actionverify is offline; actionrun additionally requires allow-sandbox,
canonical entity/book/stream/claim UUIDs, exact shared development instance/project,
two positive int63 secret versions and an explicit UTC deadline at most10minutes
ahead. Runtime pins Sandbox SQL/database/login/KMS/static names/evidence bucket.
No Production flag, URL, event selector, credential value, DSN or actor override.
The command itself neither renews source consent nor creates a scheduling request.

One attempt only, even after uncertainty; fixed HTTPS POST Sandbox
/sandbox/item/fire_webhook, TRANSACTIONS/SYNC_UPDATES_AVAILABLE, pinned API version,
no redirects/retries, bounded10s/4KiB response. Credentials exist only in normal
runtime memory and the deliberate provider request. Closed errors discard bodies,
headers, IDs and diagnostics; fixed result reports requested/completed only.
Expiration fences each provider call and its context. Existing SQL remains final
authority; successful probe output is not proof of signed receipt/publication.

STD-01: proprietary operational test notification with no amounts, currencies or
posting instruction; no applicable ISO20022 financial message selected. Existing
provider notification mapping applies unchanged; no ISO4217 or audit-export fields
changed. Existing ISO21378 full-text access limit remains. No conformance claim.

STRIDE: spoofing/elevation(high) -> existing current SQL lease/identity/Sandbox
scope and no grants; tampering/replay(high) -> unchanged verifier/receipt and stable
claim recovery; disclosure(high) -> no tokens/payloads in flags/logs/results;
availability/cost(medium) -> one process/one firing/deadline/no retries. Interrupted
firing is uncertain, not safe to blindly repeat. Human review is required by the
applied Citizen Developer security policy before releasing sensitive generated code.

Acceptance: TestProbeBoundsAndWire, TestProbeFailureRedaction,
TestProbeAtMostOnceAndExpiry, command offline/invalid/expired tests, existing SQL
authority/runner integration and exact worker image proof. Actual qualification
also needs trusted scanned source/image, scoped job, signed204/durable receipt,
demand increment/automatic worker completion and independent row-count readback.
No migration or financial semantics change. Applicable pilot gates remain337/88.
