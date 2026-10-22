# Sandbox Hosted Link enrollment

## Decision and sources

Issue #125 implements the owner's requested next Sandbox application slice
(2026-10-14). New enrollments in the Sandbox HTTP adapter request Hosted Link;
existing synthetic SDK fixtures remain supported. No live-bank activation.

Reviewed relevant creation/completion sections of Plaid's
[Hosted Link guide](https://plaid.com/docs/link/hosted-link/) and
[Link API](https://plaid.com/docs/api/link/) on 2026-10-14, API version 2020-09-14.
Hosted Link exposes completed session results without account-specific enablement.
Ordinary SDK session results require Link-events enablement from Plaid support.
Plaid recommends the SDK for owned frontends. Our temporary qualification choice
avoids that external prerequisite; retain the SDK as the product alternative and
revisit after enablement. This does not qualify institution-specific behavior.

## Contract

Begin includes an optional ephemeral `hosted_link_url`, limited to HTTPS
secure.plaid.com/hl/ with an opaque alphanumeric/hyphen/underscore path segment,
no query, fragment, userinfo or port. Both server and UI validate it. No
SMS/email, redirects or webhook integration is added. The URL and Link token
are transient browser handoff material; the server also stores an encrypted,
attempt-scoped Link token for completion after reload. SQL retains the hash.
A new-tab link uses noreferrer/noopener.

Explicit completion posts `{}` through the new complete route. Before claiming
exchange, SQL context must match the retained token hash, unexpired new attempt
and current authority. The server reads one
bounded /link/token/get result. Zero completed public tokens returns the unchanged
link_ready projection; the user can check again. Multiple tokens, malformed,
stale or cross-session responses fail closed. Exactly one completed result provides
the public token in server memory, then the existing one-shot claim, session
verification, exchange, manifest and immutable credential publication run.
No automatic polling/retry of exchange. An uncertain reply requires existing
attempt inspection/reconciliation. Scope changes clear browser handoff material;
the server retains its encrypted attempt-scoped copy for authorized recovery.

## Source-backed recovery implementation (#187)

[Plaid's Hosted Link guide](https://plaid.com/docs/link/hosted-link/)
recommends retaining the Link token associated with the application's user,
strongly recommends a completion webhook, and documents `/link/token/get` as a
way to obtain completed-session results for up to six hours after the session
ends. A redirect or Plaid success screen alone does not prove local exchange,
storage or binding. The reviewed sections and limits are recorded as PI12 in
[the Plaid source register](../research/plaid-ingestion-sources.md).

Migration 19 stores an immutable encrypted Link handoff bound by AEAD/KMS AAD
to the exact actor/entity/book/attempt. Only scoped SQL functions can read its
ciphertext, after current grants and worker revision checks. The authenticated
`POST /connection-attempts/{id}/complete` accepts only `{}`; the server decrypts
and verifies the original Link-token hash, queries `/link/token/get`, then uses
the existing one-shot claim, exchange and encrypted credential publication.
The browser never receives a recovered token. The attempt remains claimable
until the earlier of seven hours after Begin or six hours after Plaid's Hosted
URL expiry; the URL itself still expires at the provider time. Provider results
older than six hours are rejected. Encrypted handoff rows remain immutable
audit history after expiry and are never used to start a new Plaid session.
Revoked actor/book authority blocks reads and completion. This bounded **use**
window is not a deletion policy; physical ciphertext retention needs a
separately reviewed lifecycle.

Evaluate verified `SESSION_FINISHED` webhook delivery as the recommended
provider path with `/link/token/get` as a bounded recovery/read path. This work
depends on the separate webhook authenticity and durable processing gates in
#103 and #87; an unverified webhook must not authorize exchange. Until that
design is accepted, continue explicit user-driven completion for an intact
same-tab handoff, showing **Open Plaid Sandbox** and **Check bank setup
completion** prominently.

## Explicit cancellation and retry (#200)

Owner direction now permits discarding an unfinished **new** setup even if the
user completed Plaid. This supersedes the earlier absence-of-completion restart
proposal. The application does not claim that no provider session exists.
The existing one-shot exchange claim is the authoritative local boundary:
`abandon` locks the same attempt as `claim_connection`, rechecks initiating
actor authority, and permits `link_pending`/`link_ready` only without a claim,
sync Item, stored credential or publication. A winning cancellation makes the
attempt immutable `abandoned`; a winning exchange claim rejects cancellation.
No provider calls occur during cancellation, and no credential or audit history
is deleted. This can leave an unused provider session; it is not bank-access
revocation or Plaid Item removal. Existing claimed lost-credential abandonment
keeps its original guards.

The UI offers **Try again** and **Cancel setup** beside the blocking setup after
reload. Both use authenticated POST `/abandon` with the existing explicit
confirmation body. Retry creates a new session only after confirmed abandoned
state and refreshed zero unfinished count. A lost reply reads that exact attempt;
failed reads or a competing claim never authorize a new enrollment. Cancel does
not create a replacement. Terminal attempts remain audit history.

Issue #215 extends this boundary after a credential has been published. A
`binding_review` attempt cannot be abandoned by a local state update alone: the
application follows the provider-backed prepare/remove/finalize contract in
[binding review recovery](binding-review-recovery.md). The earlier unclaimed and
unrecoverable-credential paths remain unchanged.

This is an internal operational state change, not an ISO 20022 financial
message. Existing currency and ledger contracts are unchanged. Migration 21
updates the function through release tooling; no operator record edits occur.
Sources: existing PI12 review and local claim/publication contracts. An attempted
2026-10-19 refresh of Plaid's tokens page was unavailable; no new provider
behavior is assumed. Acceptance is synthetic SQL/service and browser evidence,
not a claim of fresh hosted enrollment success.

Acceptance: synthetic browser/API/PostgreSQL fixtures cover same-tab completion,
reload, completed session after reload, no completed result, duplicate results,
concurrent claim, transport uncertainty, denied scope and repeated clicks.
One old attempt can yield at most one exchange and one stored credential. A
fresh Hosted Sandbox test must reach binding review; #160 and #164 remain open
for separate hosted credential and worker evidence.

Explicit account binding and existing acquisition/review remain separate. This
change cannot post a journal. Migration 19 adds encrypted Link handoffs and
extends the unclaimed completion window. Internal operational commands and
Plaid proprietary JSON are not ISO 20022 messages; existing ISO 4217 mapping applies.

## Fresh-enrollment recovery correction (#187, wave 2)

Coordinator reviewed 2026-10-22; existing migration 19 supplies encrypted
actor/entity/book/attempt retention and current authority. No new migration or
API field is required for this correction. This does not enable ordinary
reconnect, SDK OAuth return or LINK webhook acceptance.

The first server-side `CompletedPublicToken` lookup checks the exact retained
Link token against the provider response, one completed modern Item-add result,
bounded token shape, completion freshness and no ambiguity. `VerifySession`
currently repeats those same checks for new enrollment and also rejects an
expected existing Item. Repeating that lookup after the one-shot SQL claim can
strand a valid setup on a second-read outage without ever exchanging its token.

For a server-resolved new enrollment only, `Execute` retains an unexported local
typed proof of both token hashes. Before exchange, the SQL claim must still
match the exact new mode, nonce, Link hash, public hash and absence of an expected
existing Item. The proof comes only from the trusted lookup, never browser input,
request/context metadata or an exported verified flag. Caller-supplied public
tokens continue through `VerifySession`. SQL claim/expiry/current scope checks
and immutable credential publication remain authoritative. No provider exchange
is retried on uncertain SQL/provider acknowledgement; replay inspects state.

`CompleteHosted` uses one ten-second aggregate context and bounds the retained
ciphertext/KMS read by the earlier recovery expiry. Expired, canceled or denied
attempts do not start a provider lookup. `Execute` obtains fresh SQL context after
the retained-token read; revocation during that read therefore blocks completion.
The encrypted row is not deleted: immutable historical ciphertext retention and
physical deletion policy remain unresolved. These use bounds are not a TTL.

The UI distinguishes the short Hosted URL lifetime from the durable setup
recovery deadline. URL expiry removes navigation but preserves explicit checking
of an already completed session. A completion conflict refreshes authorized
discovery and updates expired-unclaimed state; an expired recovery window hides
stale completion/cancel guidance. It does not create another attempt automatically.

Final validation exposed an existing default-date initialization race: the form
could accept a user's binding date before the passive initializer replaced it
with today. The existing synchronous date/proposal initializer now runs before
paint through a client layout effect, retaining its `!date` guard. This changes
initialization timing, not the default date, subsequent user edits, proposal rules
or keyed scope reset. `TestConnectionsEnrollmentBinding` asserts the chosen
2026-01-02 date reaches the submitted binding request. LR06 records the React
mechanism; a retry/timeout increase is not used to hide the observed failure.

Alternatives: keep the redundant post-claim lookup (rejected because transient
read failure strands a verified session), trust browser success (rejected), or
exchange before SQL claim (rejected duplicate-effect risk). Signed
`SESSION_FINISHED` is a separate future profile, not a shortcut in this change.
STD-01 proprietary Plaid bootstrap applicability and existing currency/accounting
mapping are unchanged; no ISO 20022 message or conformance claim is invented.

Acceptance: `TestHostedRecoverySingleAssociation` uses the actual HTTP adapter
and fails any second lookup while completing exactly once; caller-token mismatch,
pending/ambiguous/outage/lost-claim and replay tests retain the original safety
boundaries. Deadline and revocation fixtures cover retained-token access. UI
expiry tests preserve completion after URL expiry and stop the discovered 409
expiry loop. Full-stack journeys remain synthetic provider/identity evidence.

Reviewed current official sections and limits: LR05 in
[the recovery source register](../research/plaid-link-recovery-sources.md).

## Acceptance and limits

[Qualification recovery](plaid-link-recovery-qualification.md) adds fixed guidance
for a unique fresh modern finished exit before claiming exchange. Legacy,
unfinished and ambiguous exits do not authorize completion. Explicit cancellation
still uses the existing SQL race guards; no provider-absence assertion is made.

`TestHostedLinkNavigation` and `TestHostedLinkCompletion` reject unsafe URLs,
ambiguous and stale results. `TestHostedCompletionClaimBoundary` proves pending
and invalid authority never consume a claim and exchange occurs once. UI tests
“Hosted completion stays pending, then exposes verified binding review” and
“Hosted handoff clears on …” cover completion, uncertainty, cancel and scope changes.
`TestSyncReviewPostJourney` checks explicit binding, acquisition, exact cents and
zero automatic journals through PostgreSQL.
Actual interactive Hosted Link, deployed acquisition, historical completion and
signed webhook qualification remain open in #88 until observed separately.

## Release compatibility

The pinned Sandbox adapter now always creates Hosted Link. Retain the existing
schemaVersion2 `link_events_qualified` boolean as ordinary-SDK account metadata,
but accept false: Hosted Link does not depend on that enablement. Terraform must
pass its actual value, never force true. The field remains required and included
in release/rollback digests; this does not silently enable connections=null.
Older binaries still reject false and are not eligible for this profile. Build
and promotion must pin this implementation. New Go/release/Terraform tests prove
false propagates without weakening credential, expiry, identity or disabled gates.
