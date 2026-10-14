<!-- stage-runbook: true -->
# Connection workspace UI

## Status and boundary

Issue #104 bounded frontend slice. Synthetic injected Link only; no live providers.
Own Connections component/tests and connection-workspace contract/runbook.

## Stage ledger

Read issue acceptance, AGENTS, index and connection HTTP/service/SQL contracts.
Audited missing pause, manual sync, attempt discovery and status timestamp APIs.
Recorded bounded component contract before implementation; root owns composition.

## Current evidence

Implemented new enrollment with injected Link, explicit provider-first binding,
known attempt inspect/reconcile, scoped list and same-key account disconnect recovery.
Peer review checked scope pins, late effects, transient tokens and one-shot handoff.

`make check test build` PASS. Web type check, 196 unit tests across 16 files,
production build, and 13 existing browser regressions PASS. Twelve new component
tests cover missing deployment, scope/revocation, Link cancellation by lock, unknown
Begin/exchange/disconnect, identity mismatch, StrictMode stale read, exact binding,
credential reconciliation, confirmed conflict and malformed list state.

Default and browser commands initially could not bind localhost under sandbox;
authorized reruns passed. Logs: `/private/tmp/jayburd-connection-ui-default.log`
and `/private/tmp/jayburd-connection-ui-browser.log`. Existing browser regressions
do not constitute the new connection browser journey; root owns that added proof.

## Remaining work

Local enrollment latch disappears on remount; there is no durable attempt discovery.
Full #104 needs additional reviewed backend projections/operations and hosted
qualification. No browser persistence, automatic provider retry or posting.

## PR handoff

Root integrates navigation, selected-source routing and browser journey evidence.
