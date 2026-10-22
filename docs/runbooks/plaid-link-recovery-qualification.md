<!-- stage-runbook: true -->
# Connection recovery qualification — #104 / #187

## Status and boundary

Implementation-only lane under #65 `PLAID-88-SWARM-20261022T1540Z`.
No provider/cloud calls, deployment, secrets, consent renewal or production
activation. Synthetic coverage does not clear hosted matrix rows.

## Stage ledger

- Read live #104/#187 exports, index, bibliography, existing Link/Hosted/SQL
  contracts, runbooks and #88 L/O/I/E rows.
- Reviewed official current Hosted/Update/Link API/Web SDK selected sections;
  recorded alternatives and exact unqualified ordinary update-session boundary.
- Added adversarial Hosted completion fixtures and per-code SDK exit recovery
  proofs. No relaxation of reconnect/session proof, exchange claims or storage.
- Implemented strict unique modern finished-exit classification mapped before
  claim to existing conflict response. Both UI completion paths inspect exact
  attempt and refresh authorized discovery, then show explicit cancel/retry
  guidance. No provider absence inference, autoabandon or new API shape.
- Proposed concrete SDK transition, OAuth scoped recovery, SQL/API dependencies
  and per-row hosted execution cards in the linked contract.

## Current evidence

PASS: `make check test build` with local Go 1.26.5 and external modfile; tracked
Go requirement unchanged. Local listener permission required an approved rerun.
PASS: `make web-check web-test web-build`; initial full unit run 319 tests, then
five additional conflict-state/denial/scope-race cases passed focused (55
Connections tests). Final generated API/type checks and full unit rerun: 324
tests passed across 19 files.
PASS: `make web-browser` (24 Chromium tests), including real intercepted-SDK
failed exit, destroyed handler, late-success fencing and explicit fresh session.
PASS: `make app-integration connection-browser-test`: both actual browser →
production Next bridge → Go → disposable synthetic PostgreSQL journeys. The
connection proof covers enrollment, acquisition, explicit posting and lost-reply
recovery; provider and identity remain synthetic.

Logs: `/private/tmp/jayburd-104-core-gates.log`, `jayburd-104-web-gates.log`,
`jayburd-104-browser-gates.log`, `jayburd-104-integration-gates.log` and
`jayburd-104-final-web-tests.log`; these are local artifacts, not hosted evidence.
Hosted acceptance remains open: OAuth/device flows, repair, consent changes,
duplicate Items, actual MFA/error/challenge cases and Item lifecycle.

Contract: [recovery qualification](../contracts/plaid-link-recovery-qualification.md).
Source review: [LR01–LR04](../research/plaid-link-recovery-sources.md).

## Remaining work

Qualify actual provider/device/error flows on a separately reviewed deployment.
SDK transition awaits server association proof.

## PR handoff

Links #104/#187; no migration or API schema change. Reconnect remains disabled.
Coordinator owns review, GitHub state and merge; no hosted clearance claimed.
