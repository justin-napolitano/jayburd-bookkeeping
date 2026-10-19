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
no query, fragment, userinfo or port. Both server and UI validate it. Request
lifetime is 900 seconds. No SMS/email, redirects or webhook integration added.
The URL and Link token stay in component memory; SQL retains the existing hash.
A new-tab link uses noreferrer/noopener. Keep the originating tab open.

Explicit completion posts nonce and Link token through the existing execute route,
omitting public_token. Before claiming exchange, SQL context must match nonce,
Link hash, unexpired new attempt and current authority. The server reads one
bounded /link/token/get result. Zero completed public tokens returns the unchanged
link_ready projection; the user can check again. Multiple tokens, malformed,
stale or cross-session responses fail closed. Exactly one completed result provides
the public token in server memory, then the existing one-shot claim, session
verification, exchange, manifest and immutable credential publication run.
No automatic polling/retry of exchange. An uncertain reply clears the browser
handoff and requires existing attempt inspection/reconciliation. Reload drops
unclaimed handoff; wait for expiry before restarting. Scope changes destroy it.

## Proposed lost-handoff recovery (#187)

The expiry wait above is the implemented behavior, not the desired self-service
journey. A fresh `link_ready` attempt in discovery contains public status only;
GET inspection must not be presented as a way to recover its URL, nonce or Link
token. A successful Plaid screen also does not establish that Jayburd exchanged,
stored or bound a credential.

Proposed replacement: while the original tab retains a valid handoff, show
**Open Plaid Sandbox** and **Check bank setup completion** as the next actions.
If that handoff is lost, offer **Start over** only after an authoritative,
actor-scoped server check proves this exact attempt is still unclaimed, has no
recorded credential or exchange, and can be made terminal atomically against a
concurrent claim. Re-read durable discovery and require zero unfinished attempts
before a new Begin. A claim, stored credential, uncertain check, changed grant or
concurrent transition must stop restart and direct the user to safe status review.
The old `/execute` is never replayed. The new handoff remains memory-only; no
browser persistence or token-reconstruction route is permitted.

This is a proposed contract, not an implemented route. The existing
`/abandon` operation is limited to `credential_recovery_required`; extending it
to `link_ready` requires a reviewed SQL/API transition and tests. Waiting for
expiry remains the current fallback until that transition ships. This proposal
favors a guarded restart over persisting the Link token or allowing an unverified
second Begin, which could lose authority or race a provider claim.

Acceptance: a synthetic browser/API/PostgreSQL journey covers same-tab completion,
reload or tab loss, guarded restart, concurrent claim, uncertain status, denied
scope and repeated clicks. One old attempt becomes terminal and at most one new
Begin occurs; no second exchange runs for the old attempt. Hosted owner
acceptance still requires binding review and the separate encrypted-credential
and acquisition gates in #160 and #164.

Explicit account binding and existing acquisition/review remain separate. This
change cannot post a journal. No SQL migration. Internal operational commands and
Plaid proprietary JSON are not ISO 20022 messages; existing ISO 4217 mapping applies.

## Acceptance and limits

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
