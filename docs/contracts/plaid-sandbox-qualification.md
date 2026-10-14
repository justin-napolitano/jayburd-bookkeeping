# Plaid Sandbox qualification and bounded diagnostics

## Approved first executable slice

Issue #88 remains an open qualification gate. Coordinator approved a read-only
Sandbox sync diagnostic, not enrollment or a completed application qualification.
The default `sandbox-qualify -mode offline` reads no credential input and makes no
network calls. Explicit `-mode sandbox-sync -allow-sandbox -input-fd N` requires
an owner-injected pipe descriptor N >= 3, never stdin, a TTY or a regular file.
No environment/ADC/credential discovery, URL option or command-line secret exists.
Agents must not execute the provider mode or inspect its input without separately
reviewed execution readiness. Owner supplies an already linked Sandbox access token.
The harness does not create/exchange tokens or impersonate Link association proof.

Input is one bounded JSON object (16KiB, unknown/duplicate keys rejected):
`environment` exactly `sandbox`, `client_id`, `secret`, `access_token` with Sandbox
prefix, selected `account_id`, `currencies` (installed ISO metadata using decoder
Currency fields Code/VersionID/MinorUnit/ValidFrom/ValidTo), and nullable `cursor`.
The cursor may be omitted for an initial snapshot. Inputs are memory-only and
never included in receipts, errors or debug formatting. Metadata selects a probe;
it does not prove SQL scope authorization. Owner must use only fictional accounts.

The harness uses production `acquisition.PlaidHTTP` and `plaid.DecodePage`, with
fixed Sandbox HTTPS host and API version 2020-09-14. Each of two sequential sync
rounds is bounded to ten pages, 1,000 changes and 2MiB aggregate raw bytes. Total
execution deadline is 60s; no automatic retry or pagination restart. First round
checks exact decoding and cursor progression; second starts from the final cursor
and records whether an empty repeat delta was observed. Concurrent provider changes
make that observation unavailable, not proof of an application defect. Historical
readiness is separate from has_more=false. Raw pages are discarded after counting;
no payload, transaction identifier, amount, name, cursor, credential or request ID
is persisted/output. Go memory disposal is not guaranteed physical zeroization.

Receipt profile `plaid-sandbox-diagnostic-v1` exposes only mode, overall status,
fixed named checks/reasons and bounded integer page/change counts. Overall status
is `blocked` before successful first-round evidence, otherwise `partial`; this
executable cannot issue overall `pass`. Individual checks can pass. Counts describe
provider changes seen by the probe, never booked transactions or accounting totals.
The repeat check is not SQL idempotency evidence. The executable opens no database,
so it cannot assert application journal counts or provider-to-review consistency.

## Full #88 acceptance still required

1. Real interactive Link session, verified server session association, one-shot
   exchange and current manifest, explicit binding and credential publication.
2. Actual application runner imports an initial snapshot and an incremental update;
   compare fixture-known exact amounts, identities/currencies and cursor publication
   against source-review SQL. Replay must preserve revision/observation counts and
   assert no journals/outbox posting effects in that same application database.
3. Dynamic Sandbox pending-to-posted update, including removed pending identity and
   added posted identity across pages, complete-set publication, source lineage and
   unchanged already-posted accounting history.
4. Real signed Sandbox webhook through the deployed verifier and SQL receipt into
   acquisition demand. A fire-webhook response alone does not prove delivery,
   signature acceptance, replay protection, dispatch or eventual review publication.
5. Owner reviews a redacted receipt tied to deployed revision and all test stages;
   failures/unfinished stages remain explicit. Sandbox never approves live books.

## Real Link readiness blocker

Existing `connections.HTTPProvider.CreateLink` omits `hosted_link` and discards
`hosted_link_url`, but VerifySession requires server `link_sessions` results.
Plaid normally provides complete `/link/token/get` session data for Hosted Link;
ordinary Link needs account-specific Link-events enablement. Before real execution,
verify that enablement or implement/review the Hosted Link request/result/UI profile.
Neither is enabled or claimed by this diagnostic. `/sandbox/public_token/create`
bypasses interactive Link and cannot satisfy association/UI acceptance. Reconnect
remains unqualified; a successful accounts/get is not reauthorization evidence.

## Annotated primary sources and review depth

Selected public sections reviewed 2026-10-14; no real API calls or dashboard access:

- SQ01 [Link API](https://plaid.com/docs/api/link/): `/link/token/create` hosted_link,
  hosted_link_url; `/link/token/get` availability and session results. Supports the
  readiness blocker above; no account-specific enablement was verified.
- SQ02 [Hosted Link](https://plaid.com/docs/link/hosted-link/): creation/completion
  workflow and Sandbox delivery limits. Sandbox does not send Hosted Link email/SMS.
- SQ03 [Transaction states](https://plaid.com/docs/transactions/transactions-data/):
  pending lineage and Testing pending and posted transactions. Dynamic test user
  `user_transactions_dynamic` can use non-OAuth `ins_109508`; refresh advances states
  and emits transaction webhooks. Removed/added replacements may span pages, and
  their amounts/names may differ. These are provider records, not posted journals.
- SQ04 [Sandbox API](https://plaid.com/docs/api/sandbox/): public_token/create bypass,
  transactions/create and item/fire_webhook. Custom creation supports up to ten
  transactions per call; explicitly pin account and ISO currency instead of defaults.
  SYNC_UPDATES_AVAILABLE firing is distinct from downstream processing evidence.
- SQ05 [Sandbox overview](https://plaid.com/docs/sandbox/): Differences between
  Sandbox and Production. Institution-specific history limits, pending behavior and
  OAuth flows are not faithfully simulated; do not derive production capacity,
  balance correctness, historical completeness or consent behavior from this probe.

Shared bibliography/index links are coordinator-owned. ISO 4217/date and exact
amount mapping follow the existing [decoder contract](plaid-decoder.md) and
[standards applicability](standards.md). Plaid API is proprietary, not an ISO 20022
message; this operational receipt is not a financial interchange or conformance
certificate. No new accounting policy, SQL schema or dependency is introduced.
