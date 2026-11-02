# Authenticated sync notification projection — #385

Decision candidate2026-11-02; human security-sensitive review before release.
Approved388 mergedb5dacea; build41368842038 passed all validation/runtime/scans
and exact trusted receipts. Only existing Sandbox receiver updated to worker87594fa,
revision00002-492 Ready; unsigned401 retained. One-shotd8kr9 succeeded01:39:42Z,
queue restored RUNNING. Actual callback01:39:36Z returned401, fixed reason
sync_unexpected_field. This stage follows successful signature and exact-body hash
validation. No raw field name/value, payload, token or key was inspected.

## Reviewed sources and inference

Selected2026-11-02: official [Plaid versioning](https://plaid.com/docs/api/versioning/),
API-versioning/backwards-compatible changes section: adding elements to existing
response or webhook schemas does not require a new API version. Selected
[Transactions](https://plaid.com/docs/api/products/transactions/#sync_updates_available)
notification semantics/properties and [verification](https://plaid.com/docs/api/webhooks/webhook-verification/)
original-byte authentication were rechecked. Public sections only; no personalized
payload inspected. The extra field's identity remains unknown and is unnecessary
for this projection. The documentation does not guarantee that our prior closed
seven-field whitelist is exhaustive.

Inference/decision candidate: after unchanged authentication, validate bounded,
unambiguous JSON and project only exact reviewed sync keys before typed decoding.
Discard additive fields without logging/persisting/interpreting them. Required
type/code/Item/environment/readiness and optional user remain strict. Explicit
legacy error/count fields remain denied; LINK/JWT/JWK parsers remain closed.
This changes the old event-whitelist policy; it does not relax cryptography or
grant any new event, Item, environment, SQL identity or financial authority.

Alternatives: guessing one extra field is unsupported by evidence; another raw
diagnostic/logging release exposes unnecessary data; retaining the whitelist
conflicts with provider evolution. Broad permissive decoding risks case-insensitive
aliases overriding routing fields, duplicate keys and malformed JSON; reject those
ambiguities and project exact names before decoding. Depth32 and64KiB are local
availability bounds, not provider guarantees. Duplicate object keys at every depth,
invalid UTF8/trailing JSON and malformed known fields still deny before SQL.

STD-01 unchanged: proprietary wakeup notification, no applicable ISO20022 message,
amount/currency/accounting mapping change or conformance claim. Existing SQL current
authority, replay identity and atomic receipt/demand behavior remain decisive.

Acceptance: TestSignedSyncAdditiveProjection (both profiles, exact metadata equality,
ignored alias/nested/null/scalar fields, body-hash tamper denial and no reflection),
TestSyncProjectionRejectsAmbiguity (duplicates/depth/types/missing/null/legacy/event
denials), existing verifier/socket/profile/SQL tests, core and release gates.
Hosted signed204/durable receipt/new generation/automatic completion remains OPEN;
this source candidate alone cannot clear it or activate Production.
