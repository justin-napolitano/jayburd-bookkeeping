# Plaid ingestion sources

Reviewed 2026-10-13 for [#70](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/70).
Only public documentation was read; no provider endpoint, credentials, Sandbox or
bank account was accessed. The [contract](../contracts/provider-ingestion.md)
labels project proposals separately from provider facts.

| ID / official source / reviewed sections | Evidence and relevance | Review limit |
| --- | --- | --- |
| PI01 — Plaid [Transactions API](https://plaid.com/docs/api/products/transactions/), `/transactions/sync` request, pagination, response amount/currency/date fields | Cursor streams return added/modified/removed changes; options.account_id defines an independent account-filtered cursor. Amount is described as a JSON number/double, positive for money leaving an account, negative for money entering. ISO and unofficial currency identifiers differ; authorization and posted dates differ. | Relevant sections reviewed, not every product field. Exact-number decoding is our requirement; generated SDK numeric behavior must be inspected before choosing it. API version alone does not freeze every additive response field. |
| PI02 — Plaid [Sync migration guide](https://plaid.com/docs/transactions/sync-migration/), pagination, initial trigger, webhook and Item-data sections | Apply complete updates; restart pagination from its original cursor on mutation. Initial sync enables sync notifications; empty initial results need not mean history is ready. | Relevant sections reviewed. Guide examples are not atomic local persistence or permission proofs. Our staging/fencing design is an inference. |
| PI03 — Plaid [Transaction states](https://plaid.com/docs/transactions/transactions-data/), pending/posted section | Posting can remove a pending ID and add a different posted ID across pages; pending linkage may be absent. Amount/details may change; even provider-posted items can be modified/removed. | Full relevant section reviewed. Provider “posted” is not our journal state. No source-to-ledger correction authority follows from it. |
| PI04 — Plaid [Webhook verification](https://plaid.com/docs/api/webhooks/webhook-verification/), algorithm/key retrieval, validation and body hashing | Verify ES256 signature, obtain key by kid, enforce five-minute age and compare authenticated SHA-256 against exact body bytes. Key metadata includes expiry. | Relevant procedure reviewed, not a security endorsement of copied sample code. Our future-skew, caching, size and ingress policies require tests. |
| PI05 — Plaid [Link overview](https://plaid.com/docs/link/), flow overview, initialization and errors | Link token initializes consent flow; temporary public token is exchanged server-side for Item/access token. Duplicate Items need handling. | Relevant sections reviewed. Our account ownership, session-binding, secret storage and uncertain-exchange handling remain project responsibilities. |
| PI06 — Plaid [Link update mode](https://plaid.com/docs/link/update-mode/), overview and login/consent recovery | ITEM_LOGIN_REQUIRED and expiration/disconnection notifications can require user reauthentication/update mode. | Relevant lifecycle sections reviewed; no live consent configuration validated. Product/country/account coverage remains qualification work. |
| PI07 — Plaid [Items API](https://plaid.com/docs/api/items/), public-token exchange, access-token invalidation, item removal | Rotation returns a replacement access token and invalidates the previous one. Item removal is a provider lifecycle operation. | Selected endpoint descriptions reviewed; do not infer idempotency or recoverability of a lost rotation response. No token value copied. |
| PI08 — Plaid [API versioning](https://plaid.com/docs/api/versioning/), version header and 2020-09-14 section | Requests can explicitly select Plaid-Version; 2020-09-14 is documented. | Relevant sections reviewed. Proposed pin must be revalidated with contract fixtures and the selected provider environment before implementation. |
| PI09 — Plaid [Link API](https://plaid.com/docs/api/link/), token creation, token/get and session results | Session results can associate a public token with its Link session. Hosted Link and ordinary Link availability differ; callback metadata alone cannot authorize our connection. | Selected sections refreshed 2026-10-13. Concrete new-session verification requires supported results; ordinary update-mode association is not guessed. Hosted qualification must verify the configured flow. |
| PI10 — Plaid [Accounts API](https://plaid.com/docs/api/accounts/), accounts/get and Item metadata | Server-returned account IDs and types drive explicit source selection, not browser-provided names or masks. | Selected fields reviewed by enrollment lane; no institution coverage or ownership certification. Local authorization expiry is a configured policy bound when provider consent expiry is absent, not a fabricated provider fact. |

## Standards and alternatives

Reuse ISO 4217/date source limits in [SI02–SI04](statement-import-sources.md) and
[the standards contract](../contracts/standards.md). Plaid JSON is a proprietary
exchange; explicit semantic mapping is necessary, not an invented camt designation.
Official ISO message/profile verification in #17 stays unresolved.

Compare: date-window `/transactions/get` requires additional change reconciliation;
`/transactions/sync` gives a clearer change/cursor boundary. Item-wide sync keeps
one checkpoint but couples account authorization; account-filtered streams could
isolate subsets, at the cost of separate checkpoints and a reviewed migration.
Our selected profile uses account-filtered connections; account revocation stops
that stream while Item-level consent revocation stops all. Filter/mode changes
require null-cursor new generations and overlap review.
CSV remains an independent adapter into the same proposed source-observation core.
No benchmark or bank coverage was measured. Revisit limits and stream selection
when #13 qualifies an actual institution and #8 implements ingestion.

## Connection operations review — 2026-10-13

#103/#104 refreshed PI04 verification/key metadata and PI01 sync notification
sections before implementation. Source requirements remain signature verification,
five-minute freshness and exact-body digest. Our selected limits (64 KiB body,
8 KiB token, 60-second future skew, bounded key cache and lookup admission),
receipt identity and transaction/outbox design are local policies, not provider
guarantees. Readiness notifications do not establish the time of the provider's
last data update; public provider_updated_at remains null. Database publication
time is separate. Signed fixtures verify protocol behavior without contacting Plaid.

The chosen alternative to direct acquisition in a webhook request is durable
generation intent with separate acquisition. It bounds the public request and
retains a wakeup arriving during an in-flight fetch. Stream revision is separate
from source binding revision so pausing acquisition does not change accounting
mapping authority. SQL and full-stack acceptance are recorded in the
[operations runbook](../runbooks/connection-operations.md); hosted routing and
measured capacity remain qualification work.

## PI01 compatibility refresh — 2026-10-14

Reviewed the official Transactions API transaction_code response description:
nullable classification, populated for European and some US institutions and
null where unavailable. No provider payload was inspected. Accepting this as
bounded inert metadata is a project interpretation; it cannot assign accounting
meaning. See the [decoder contract](../contracts/plaid-decoder.md) and
[regression evidence](../runbooks/sandbox-decoder.md). This selected-field review
is not a complete response-schema audit or proof of the hosted failure cause.
