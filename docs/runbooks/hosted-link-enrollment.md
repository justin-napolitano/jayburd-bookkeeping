<!-- stage-runbook: true -->
# Hosted Link enrollment delivery

## Status and boundary

Implementation and synthetic validation complete for #125. Synthetic validation only; no deployment,
credential retrieval or live-bank connection. Contract: ../contracts/hosted-link-enrollment.md.

## Stage ledger

- Reviewed official Link creation/session/completion documentation and current service/SQL boundaries.
- Selected explicit Sandbox Hosted Link completion; retained one-shot exchange and review boundaries.

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
