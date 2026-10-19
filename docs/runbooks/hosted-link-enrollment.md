<!-- stage-runbook: true -->
# Hosted Link enrollment delivery

## Status and boundary

Implementation and synthetic validation complete for #125. Synthetic validation only; no deployment,
credential retrieval or live-bank connection. Contract: ../contracts/hosted-link-enrollment.md.

## Issue #187 recovery delivery (2026-10-18)

The new migration 19 retains each Hosted Link token as immutable AEAD ciphertext
wrapped by the existing bank-credentials KMS key, bound to authenticated actor,
entity, book and attempt. The public attempt and discovery responses still omit
tokens and ciphertext. `POST /connection-attempts/{attempt}/complete` rechecks
current SQL authority, decrypts the original token server-side, checks the hash,
and queries Plaid's completed session result before using the existing one-shot
exchange claim. The UI offers completion from a discovered `link_ready` attempt
after remount. Same-tab checks use the same route. The Next bridge and OpenAPI
allowlists include only this new empty-body operation.

Synthetic evidence: migration 19 fresh install/replay passes. Unit tests prove
per-actor/attempt envelope separation, pending completion without claim, reload
completion, one exchange, and public route behavior. The full synthetic
Connections browser/API/PostgreSQL proof passes with a remount between the
pending and completed checks, then covers account binding, acquisition and
explicit posting. `make app-integration` also passes. These tests use a fake
Plaid provider and synthetic KMS wrapper; deployed Sandbox acceptance remains.
CI's restricted Cloud SQL migration-status proof was updated to expect migration
19 as current, with 18 and 20 as invalid metadata; the old hard-coded current
version made CI reject the new migration even though the product migration passed.

The completed-session recovery window follows Plaid's six-hour result window.
The Hosted URL still expires at Plaid's reported time. A finished exit or lost
provider result does not yet have an immediate safe **Start over** path;
`link_ready` abandonment requires a separate provider-absence and concurrency
contract. Immutable ciphertext retention needs a reviewed cleanup policy.
Verified webhook delivery is still tracked under #103/#87. This change does not
deploy, connect a live bank, or activate a worker.

## Stage ledger

- Reviewed official Link creation/session/completion documentation and current service/SQL boundaries.
- Selected explicit Sandbox Hosted Link completion; retained one-shot exchange and review boundaries.
- For #160, reviewed the owner-observed attempt-review reset against current Connections
  and SQL discovery/inspection paths. Attempt discovery is a separate authorized read;
  a denied historical detail must be rechecked before clearing the selected book.

## Attempt-review recovery patch

The Connections UI now rechecks the current connection list and attempt discovery
when a historical attempt detail returns 403/404. If those scoped reads succeed,
it keeps the book and reports that the attempt is unavailable; new enrollment
remains blocked while an unresolved attempt exists. If current scope revalidation
also fails, it clears financial state as before. This is a presentation and
authorization-recheck fix, not a credential reset, provider exchange or proof
that the old attempt is recoverable.

The development web service also has a second Cloud Run URL. Its bridge accepts
the configured public origin (`jayburd-dev-web-9088794002.us-central1.run.app`),
while the service-reported `*.a.run.app` alias can lead to rejected requests.
The application landing page now redirects that alias to the configured origin
before sign-in. The owner should use the configured URL for the next test.

`Connections.test.tsx` covers both the attempt-only denial and actual scope
revocation; `page.test.tsx` covers canonical-host routing. Fresh Sandbox acceptance still requires an owner-authorized Hosted
Link session to reach verified binding review, followed by ciphertext/worker-read
evidence under #164. Keep Scheduler paused until those checks pass.

Validation for this patch: `make web-check web-test web-build web-browser`
passed on the final patch (273 unit and 17 browser tests).
`make check test build` could not start:
the pinned Go 1.26.9 toolchain download returned HTTP 403 even after a permitted
network retry; local Go 1.26.5 is below the module requirement. The existing
synthetic browser/API/PostgreSQL Hosted Link journey is unchanged but must pass
in PR CI before merge. Hosted owner acceptance remains unproven.

## Current evidence

PASS: `make check test build release-test`; `make web-check web-test web-build
web-browser` (258 unit tests, 15 browser tests). Final focused Go/type/unit and generated-contract checks also pass.
`bash scripts/test-infra.sh` passes 68 mocked Terraform tests across eight roots.

PASS: `make app-integration connection-browser-test` in an isolated copy with
only test ports changed from 39081–39083 to 41881–41883, preserving the running
demo. `TestSyncReviewPostJourney` covers a pending completion, lost committed
exchange reply, reload discovery, explicit account binding, acquisition and
reviewed exact 0.10 posting. SQL assertions confirm no automatic journals and
one final journal/source-link/outbox. Synthetic provider and identity only.

`TestHostedLinkCompletion`, `TestHostedLinkNavigation` and
`TestHostedCompletionClaimBoundary` cover ambiguous/stale results, navigation,
pre-claim authority, pending checks and replay. UI tests cover unsafe URL,
uncertain reply, cancel and scope change. Release/Go configuration tests accept
explicit false SDK enablement while retaining other gates.

## Remaining work

After merge, build the pinned API/web images
and promote reviewed connection metadata with link_events_qualified=false.
Actual hosted interactive qualification remains separate under #88.

## Operator steps after deployment

1. Sign in, select the intended development entity/book, then open Connections.
2. Choose Connect account, then Open Plaid Sandbox. Keep the original tab open;
   complete only a fictional Sandbox bank session in the new tab.
3. Return and choose Check bank setup completion. Pending means check later;
   unknown outcome means inspect the durable attempt, never repeat exchange.
4. Review account-to-ledger mappings, matching currency and binding date, then
   confirm selected bindings. View sync status and request a sync once the worker
   deployment is verified; a queued request is not evidence of completed import.
5. Review transactions after publication. Posting remains a separate explicit
   action. Record a redacted hosted receipt before closing any #88 acceptance gate.

## PR handoff

Targets main; no migration. Owner merge precedes image build and hosted acceptance.

## Hosted recovery investigation (#160)

Owner observed Hosted Link success followed by completion HTTP 409. The new
attempt's public status is `credential_recovery_required` with a recorded account
manifest. That establishes that the service got through provider verification and
manifest recording; it does not prove source storage or destination delivery.
GSM slot-00 version metadata currently lists no versions. API source creation
and separate allowlisted delivery still need to be distinguished. No credential
values or provider payloads were accessed. Storage object-name lookup was blocked
by local TLS trust; certificate verification was not bypassed.

Fixed UI recovery: independently successful durable-attempt discovery survives an
Item-list transport failure. Hosted completion 409 inspects the exact attempt by
GET, exposes credential recovery and clears the one-shot handoff without another
execute or provider exchange. `Connections.test.tsx` names both regressions.
These changes do not deliver a credential, reset attempts or activate acquisition.
The owner is checking existence only of /bank-sync/slot-00 in the isolated writer
project; no secret values are requested. Hosted acceptance remains incomplete.

Validation for #160 recovery UI: PASS `make check test build`; PASS
`make web-check web-test web-build web-browser` (265 unit, 17 browser tests);
PASS `make app-integration` against the real local API/disposable PostgreSQL.
The initial sandbox cache-access denial was rerun with approved cache access.
These tests verify recovery presentation and retained discovery, not hosted
credential creation/delivery. No migration or deployment in this patch.


PR #165 incorporates this #160/#161 recovery presentation fix together with the
owner-selected encrypted database token path. The historical source/delivery
investigation above is superseded for new runtime credentials by
[bank-token-vault](../contracts/bank-token-vault.md); no old token was retrieved
or reconstructed. Hosted acceptance remains pending coordinated deployment.

## Historical reconciliation feedback retest (#160)

After deployment 39072895377, the owner inspected attempt `e8e959ae-…` and
clicked **Reconcile recorded credential**. The attempt remained in
`credential_recovery_required`; the browser showed only its generic unconfirmed
operation message. The screenshot did not include the HTTP status, so it does not
prove whether the service returned its expected 409 or a separate failure.
The browser now offers **Try connecting again** on a failed new setup. It
reconciles the exact attempt and, after a confirmed 409, reads it again. If it
is still unresolved, the same click requests server-guarded abandonment. Only
a confirmed abandoned state plus zero unfinished attempts opens a fresh Link
session. Stored-credential conflicts and uncertain replies stop the retry.
The old provider exchange is never repeated; its audit history remains.
Synthetic UI coverage checks the successful restart and uncertainty path.
Fresh Hosted Link completion, encrypted persistence,
binding review, and worker read remain separate hosted acceptance gates.

## Fresh retry still blocked at attempt review (2026-10-18)

After release #185, the owner used **Try connecting again**. Discovery showed
new attempt `4d72d362-beae-47a9-a901-994137ba172b` as `link_ready`, one
unfinished attempt, two older attempts (one expired unclaimed, one abandoned),
and no connected Item. Clicking **Review attempt** for the new attempt showed
“This operation could not be confirmed.” The screenshot did not contain the
exact GET HTTP status, fixed problem code or response shape, so the cause is
still unconfirmed. A `link_ready` row proves Begin persisted an attempt; it
does not prove Plaid completion, exchange, ciphertext or account binding.

Current source behavior: Review attempt calls an exact read. It does not reopen
Plaid or call `/execute`. A hosted URL/nonce/token is held only in the browser
component; reload, scope change or cancel can discard it. If the original tab
still displays **Open Plaid Sandbox**, the owner can use it and then **Check bank
setup completion**. Otherwise, the implemented fallback is to wait for the
unclaimed attempt to expire, refresh discovery and start a fresh attempt. Do
not repeat an old provider exchange or infer that an unresolved attempt is safe
to abandon.

[#186](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/186)
owns the exact-read diagnosis and regression. Record only method, scoped route,
HTTP status, fixed problem code and redacted trace metadata; never capture or
post Authorization headers, Link/public tokens, provider payloads or financial
data. [#187](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/187)
owns the proposed guarded lost-handoff restart and user-facing flow. The
[Hosted Link contract](../contracts/hosted-link-enrollment.md) and
[connection status UI contract](../contracts/connection-status-ui.md) define
acceptance. Keep [#160](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/160)
and [#164](https://github.com/justin-napolitano/jayburd-bookkeeping/issues/164)
open until a fresh Sandbox session reaches binding review and the separate
credential/worker evidence passes. No production account or automatic journal
posting is implied by this plan.

Official Plaid Hosted Link and Link API documentation was refreshed and saved as
public-page snapshots for review; see PI12 in the
[Plaid source register](../research/plaid-ingestion-sources.md). It changes the
priority of #187: Plaid recommends retaining a user-associated Link token and
a completion webhook, while Jayburd's token currently disappears with the
browser component. The proposed primary recovery is a scoped encrypted
server-side token and verified completion/status path. A fast **Start over**
remains a fallback only when absence of a completed provider result and local
claim/credential can actually be proven. No provider account or token was read.

Documentation-only validation: `git diff --check` passed. `make check test
build` stopped before project checks because the pinned Go 1.26.9 toolchain
download failed (connection reset in the sandbox; HTTP 403 on approved retry).
No application code, database migration, cloud resource or hosted connection
was changed by this documentation update.
