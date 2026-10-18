<!-- stage-runbook: true -->
# Hosted Link enrollment delivery

## Status and boundary

Implementation and synthetic validation complete for #125. Synthetic validation only; no deployment,
credential retrieval or live-bank connection. Contract: ../contracts/hosted-link-enrollment.md.

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
The browser now handles a confirmed 409 by reading the exact attempt again. If
it remains in recovery, it explains that no credential was confirmed and points
to the existing affirmative abandonment review. It never retries the provider
exchange, infers a stored token from Plaid success, or automatically abandons
the attempt. Other failures retain the generic uncertain outcome. Synthetic
coverage names the 409 path. Fresh Hosted Link completion, encrypted persistence,
binding review, and worker read remain separate hosted acceptance gates.
