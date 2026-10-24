# Owned Hosted Link repair — #275

## Decision and scope

Proposed2026-10-24 by the coordinator, for implementation review. No reconnect
capability or hosted clearance is delivered by this document. Retain the current
Hosted frontend for one bounded Sandbox repair; the SDK transition remains a
separate direction. Production requires #308 and its environment-isolation work.
This phase allocates no migration number and changes no SQL, runtime or consent.

Problem: current `HTTPProvider.SupportsMode` rejects reconnect, migration19's
encrypted handoff is new-only, and ordinary update completion is not represented
by the pinned SDK's Item-add result. A healthy account lookup, browser redirect,
deprecated callback or foreign `LOGIN_REPAIRED` notification cannot prove that our
owned attempt repaired the selected Item.

LR07 in [the source register](../research/plaid-link-recovery-sources.md) records
the reviewed modern signed notification and update-mode sections. The event
identifies a Link token/session and outcome; it has no Item identity or completion
timestamp. Our inference is to join that proof to the **server-created** Link
hash and preselected SQL Item context. JWT issuance time establishes notification
freshness only. Actual ordinary Sandbox update delivery/shape remains unproved.

| Alternative | Decision / cost |
| --- | --- |
| Owned Hosted session plus signed completion receipt | Proposed bounded extension; requires receiver, SQL proof and ACL work before enabling repair |
| Move all enrollment to Web SDK | Better vendor-recommended owned frontend; adds lifecycle/OAuth/account-event qualification before it solves server authority |
| Accept Item health, redirect or deprecated success | Rejected: none identifies our completed update session |
| Exchange another public token | Rejected: repair reuses the existing access credential; risks creating a different Item |

Revisit if the actual update session lacks the documented signed event or the
retained-token association is ambiguous. Keep repair disabled and obtain provider
clarification rather than falling back to callback/account-health evidence.

## Authority and fixed protocol

1. The verified owner explicitly selects the existing Item and complete approved
   stream set. SQL rechecks current principal/entity/book authority, live source
   consent, Item generation and all sibling streams. Never trust browser Item,
   account, credential reference, generation or user identifiers as authority.
2. The backend reads the SQL-selected existing credential through the current
   vault/AAD boundary. It creates an **explicit Hosted** update Link using that
   credential, existing redirect/webhook configuration and no extra products,
   account-selection widening or user-based multi-Item update. New enrollment
   remains unchanged. Set an explicit **proposed** reconnect deadline of15minutes
   from attempt creation, capped by provider expiry. This requires new SQL: merged
   migration19 globally changed the attempt default to seven hours and its
   `record_link_session` adds six hours for both new and reconnect. Do not claim
   that reconnect is already short-lived. Replace that behavior for reconnect
   without shortening new enrollment recovery, changing old immutable receipts or
   extending source consent. Existing unclaimed reconnect attempts need an
   explicit migration disposition: cap eligibility at the immutable original
   `connection_event` begin receipt's `recorded_at` plus15minutes in the new
   claim/proof gate; require exactly one such event, otherwise deny. Attempts
   have no `created_at` column; do not invent one or rewrite their history.
   A longer vendor URL lifetime grants no local authority.
3. Before returning the URL, atomically persist nonce, Link hash, deadline and
   encrypted handoff under current actor/entity/book/attempt authority. Extend
   migration19's put/get only for the intended reconnect states. Concurrent
   handoff writes must still yield the existing single encrypted winner. Browser
   memory may hold transient URL/initialization material; no browser persistence.
4. The dedicated receiver verifies the bounded raw request's existing ES256,
   JWK and exact body digest before interpreting or submitting an event. Preserve
   the verifier's five-minute maximum age and sixty-second future tolerance.
   Reject duplicate JSON fields, malformed identifiers/hashes, unknown outcomes,
   wrong environment and invalid key/time/signature without SQL business work.
5. Parse only `LINK/SESSION_FINISHED` with `status=success|exited`, bounded
   `link_token`, `link_session_id`, and Sandbox environment. Hash Link/session
   identifiers before SQL. Validate bounded optional public-token fields and
   discard them; do not assume an update event has an empty array, log tokens,
   or exchange any of them. `user_id` supplies no ownership. Do not fabricate
   `item_id`, `finished_at` or ordinary Item-update API results.
6. A restricted SQL receipt function resolves exactly one current reconnect
   attempt by retained Link hash, then locks and rechecks it. Zero matches is
   ignored; multiple matches is denied without choosing one. Recheck ingress
   login/revision/environment, current principal/book authority, live consent,
   Item generation, sibling stream set and deadline before retaining proof.
   Require signed `iat >= original begin receipt time - 60seconds`; later notification
   delivery does not extend the deadline. Preserve prior immutable receipts.
7. Store only closed normalized hashes, outcome, signed issuance time, receipt
   time and the exact scoped attempt/Item-generation association. Never retain
   raw webhook/JWT, Link/public/access tokens or provider financial payloads.
   This receipt **does not** claim, resume, publish, create a demand or read bank
   secrets. The public receiver has no vault/KMS/Item API authority.
8. An explicit owner completion/recovery request reauthorizes SQL, reads the
   retained handoff and verifies its hash. Pending proof returns the unchanged
   attempt with fixed waiting guidance; no claim, bank credential read or provider
   request. One distinct successful session is eligible. Exited proof remains
   blocked. More than one distinct session, or conflicting outcomes for one
   session, is ambiguous and remains blocked before claim. Offer only explicit
   unclaimed local cancellation followed by a separately confirmed fresh attempt.
9. SQL claim itself requires and pins the exact eligible proof; an API-role
   caller cannot bypass it by directly invoking the older claim interface. Use
   the existing atomic Item fence/claim and full stream checks. Once claimed,
   later receipts cannot replace the selected proof or authorize another repair.
10. After claim, reuse the existing credential reference/version and read the
    current manifest. Require the expected Item and every approved bound account
    with compatible type/currency. Missing/de-selected accounts leave the Item
    paused; newly discovered accounts remain unbound. Do not silently revise the
    consent scope, binding identities or local consent deadline. Replace the
    reconnect branch of existing confirmation, which otherwise copies the
    manifest expiry: enforce no later than the old live Item approval, also
    honoring any earlier provider expiration. Never revive an expired approval.
    Confirmation
    checks the proof, claim, current generation/scope and publishes one
    repair-caused demand per still-active approved stream using existing
    `signal_sync` with reason `resume` and stable key `repair:<attempt UUID>`.
    Acquire current stream authority/locks in stable order; do not unpause an
    owner-paused stream. This is new explicit confirmation wiring, not an existing
    implicit callback. Replay of that same cause cannot add another generation;
    independent manual/webhook/scheduler signals retain their own semantics.
    Source/account/history and posted journals remain
    unchanged. No new Item, exchange, token rotation or automatic posting.

## SQL interface and locking review

Allocate a forward migration only under a fresh sole-writer lease after checking
main/#65. Merged13/14/19 and later definitions are immutable. Review current
[database foundations](../research/database-foundations.md) before DDL.

Proposed restricted interfaces (parameter names below are descriptive):

- `integration.accept_verified_link_session(environment text, receipt_key text,
  raw text)`: only `bookkeeping_webhook`. `raw` is a **closed normalized** JSON
  object containing Link/session/body SHA256, status and integer signed `iat`,
  not the provider body. Enforce exact keys/types/bounds in SQL. Return only
  accepted/duplicate/ignored and bounded counts; never disclose an attempt to
  the public caller. Receipt key/body mismatch fails atomically.
- `integration.connection_link_proof(actor uuid,e uuid,b uuid,id uuid)`: only
  `bookkeeping_api`, current owned scope. Return pending/success/exited/ambiguous
  and private receipt hashes needed for completion; no browser token or cross-
  scope discovery. Recheck authority/deadline at claim, not just this read.
- Extend `claim_connection` to require/pin proof for reconnect and retain all
  existing new/rotate/remove restrictions. Extend handoff put/get only as above;
  `abandon_connection` may cancel **unclaimed** reconnect locally. Cancellation
  does not call Item removal or reactivate the old credential. Claimed attempts
  require exact recovery, never automatic abandonment/replacement.

Use integration-owned immutable receipt/proof rows and no direct runtime table
grants or PUBLIC EXECUTE. A hash lookup identifies a candidate only. Follow the
existing principal/entity/book authority locks, source accounts ordered by
source_account_id, Item, then attempt order; never take an attempt lock and then
acquire a source or Item lock. Recheck the complete sibling stream set under that
order, then lock/write proof and receipt rows. Pin proof under the same locks
as claim, so concurrent event, cancellation, permission revocation and repair
cannot slip through a previously read eligibility result. Design exact constraints
and receipts with the single SQL implementation, not a parallel schema draft.

Dedicated-login bootstrap currently permits only
`accept_verified_webhook(text,text,text)`. Update its closed allowlist for exactly
the new Link receipt function, retaining owner/API/sync/table/vault/KMS denial,
ingress disable/replay and failed-bootstrap atomicity. Independently assess the
actual Cloud SQL IAM marker compatibility; verifier31 is not an automatic webhook
role exception. Migration, binary and bootstrap must be compatible before release.

## Replay, recovery and operational limits

Exact body/receipt replay has no new effect; re-signed same session/outcome may
add an immutable receipt but never another eligible session or repair-caused wakeup. Conflicting
outcomes and distinct sessions block unclaimed completion. Late events after claim
cannot mutate pinned proof, reactivate a revoked/stale Item or undo confirmation.
Other-mode/new-enrollment events are ignored by this reconnect boundary. Existing
new enrollment keeps its modern token-get verification and one-shot exchange.

Unknown claim reply is inspected by exact attempt. For reconnect only, the same
current-authorized claim/proof/credential can repeat the read-only manifest and
confirmation after a lost response, including failure before the first manifest.
There is no fresh Link, exchange, rotation, different proof or automatic retry
loop. Keep new/rotate one-shot uncertainty rules unchanged. Revocation, expiry,
generation or stream drift denies recovery without publishing. Terminal success
replay returns its existing receipt; journals are never produced by this path.

An ingress outage yields pending recovery and keeps acquisition paused. Owner
can explicitly cancel an unclaimed attempt; no client callback bypass. Disabling
ingress, release rollback or local cancellation is not provider consent removal.
Use the existing scoped disconnect/remove contract when that action is intended.

This changes authorization/session state: human security review is required
before managed release. Do not weaken signature checks, reuse API identity for the
public receiver or grant it secret access to simplify delivery. Exact image/schema,
bootstrap/ingress receipt and separately authorized finite reset_login mandate
precede hosted testing. #271/#303 own actual signed delivery. No cloud call,
provider call, consent renewal or Production activation is authorized here.

## Standards and acceptance

Under STD-01 this is proprietary Plaid authorization and internal operational
receipt state, API2020-09-14/Go SDK49.0.0. No ISO20022 financial exchange or audit
data export is introduced; no applicable financial-message designation is invented.
Existing dated ISO4217 currency identifiers, decimal money, original currency and
accounting policy are unchanged. No ISO conformance claim. Applicable crypto and
least-privilege controls retain the existing webhook/vault profiles.

These named acceptance cases are **proposed, not implemented evidence**:

| Case | Independent expected observation |
| --- | --- |
| `TestHostedRepairSignature` | Invalid signature/body/key/age/future/duplicate JSON never reaches business SQL |
| `TestHostedRepairForeignScope` | Unknown, multiply matched, new-mode, foreign actor/book/Item/generation/hash denied or ignored; no mutation |
| `TestHostedRepairReplay` | Repeated/re-signed same session produces one claim and one repair-caused wakeup per active approved stream; raw secrets absent from receipts/logs |
| `TestHostedRepairAmbiguous` | Exited/conflicting/two-session preclaim proofs cannot be selected; explicit unclaimed cancellation only |
| `TestHostedRepairNoSecretAccess` | Pending receiver/completion has zero bank-token/vault/KMS/exchange/manifest calls; receiver role cannot acquire those rights |
| `TestHostedRepairRevoked` / `Expired` / `ReducedScope` | Current SQL denies late receipt/claim/confirm/recovery, preserves paused acquisition, never widens scope |
| `TestHostedRepairUnknownClaim` | Lost claim response recovers the same existing claim, including no stored manifest; no second exchange or credential |
| `TestHostedRepairRecovery` | Actual API/SQL/browser reload, lost confirmation and repeated click keep the exact attempt and one terminal receipt |
| `TestHostedRepairPublication` | Fixed source/account/history/credential-version IDs and journal totals unchanged; exactly one repair-caused wakeup per active approved stream after confirmed same-Item manifest, paused streams unchanged |

Implementation gates: `make check test build migration-test ledger-test
connection-state-test connection-service-test connection-ops-test webhook-test
webhook-bootstrap-test web-check web-test web-build web-browser app-integration`.
Fixtures must assert fixed outcomes independently of generated expected values.
Add actual restricted-role direct-call bypass, ingress-disable, lock-race and
shared-Item sibling tests. Browser fixtures use no real identity or financial data.

Hosted evidence separately records image/schema/bootstrap, fixture name, platform,
bounded mandate and times, induced login-required, owned update event receipt,
same Item/credential/stream continuity, one resumed demand and independent repeat
equality with absolute zero journals and `integration.outbox_message` rows of
event_type `journal.posted` in the fresh book. Other acquisition wakeup/outbox
intent is expected; do not mislabel it a posting or require every outbox empty.
Then exercise expiry,
revoked/reduced scope, cancellation and unknown response with exact receipts.
Injected events do not clear #88 O04/I01 or production qualification.
