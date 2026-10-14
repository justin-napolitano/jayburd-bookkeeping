# Connections workspace delivery

<!-- stage-runbook: true -->

## Status and boundary

Build the supported frontend slice of #104 from merged #106 `6c6b546`.
Existing connection HTTP/service contracts are authoritative. Default startup has
no provider composition; no real credentials, Link script or provider calls.

## Stage ledger

- Verified merge and reviewed #104 against merged API, SQL and workspace behavior.
- UI agent owns new connection component/tests and its contract; bridge agent owns
  exact route forwarding/tests. Root owns navigation, source handoff and validation.
- Supported slice: list, injected enrollment, explicit verified binding, known-attempt
  inspect/reconcile and confirmed account disconnect. Unknown provider effects never
  invite blind re-execution. Tokens remain transient memory only.

## Current evidence

- `make check test build`: PASS, including Go race tests and unchanged migration checksums.
- `make web-test`: 221 tests PASS; generated API check and TypeScript PASS.
- `make web-browser`: 13 existing journeys PASS; production Next build PASS.
- Actionlint PASS. `make connection-browser-test` PASS against the integrated
  implementation: one exchange/fetch across replay, zero journals before approval,
  exact 0.10 posting, lost-response recovery, one journal/link/outbox and two source
  observations. `make app-integration` PASS for existing setup/posting recovery.
- Temporary alternate ports preserved the owner preview; committed defaults restored.

`TestConnectionsScopeAndRecovery` and related component cases cover stale scope,
revocation, single-flight handoff, cancellation, memory-only tokens and same-key
binding/disconnect recovery. The new `TestSyncReviewPostJourney` composes browser,
production Next bridge, Go enrollment/acquisition and actual PostgreSQL. Its explicit
fixture-owner review grant is separate from product connection binding. Backend-only
#106 evidence is not substituted for browser acceptance.

## Remaining work

Full #104 needs safe unfinished-attempt discovery after reload, public sync status
and timestamps, pause/manual wakeup APIs, qualified reconnect and hosted Link/secret
composition. Current workspace permissions do not advertise connect; server authority
remains decisive. No synthetic callback or UI state claims provider qualification.

## PR handoff

Refs #104; original acceptance remains open. No SQL migrations or hosted deployment.

## Review findings

Independent review confirmed the source preference is checked against current
`/sources` authority, and synthetic Link/acquisition are absent from production
composition. Mobile screenshot inspected at 390px: controls and long references
wrap within the viewport. First browser attempts corrected a label selector and
an expected presentation string: draft amounts are canonical exact `0.1`, while
trial-balance display pads USD to `0.10`; neither uses floating-point arithmetic.

The real browser proof exposed an initial source-discovery race: changing the filter
advanced the shared request generation before `/sources` completed, leaving no
source options. Discovery now holds the busy boundary until it finishes or hands
off to the selected-source read; selectors remain disabled while the source list
is unavailable and an empty selection cannot supersede discovery. `locks filters until connection source discovery
finishes` proves this regression; source authority remains checked before handoff.

## Mobile CI follow-up

PR #107's Linux frontend gate exposed horizontal overflow in book navigation.
Three `1fr` tracks inherited min-content widths from non-wrapping labels. A wider
monospace test font reproduces the same 390px overflow locally before the fix.
Mobile tracks now use `minmax(0, 1fr)` and labels wrap; no document clipping or
weakened overflow assertion. The existing workspace browser journey retains the
wide-font regression. Before fix: named browser case FAIL; after fix: all 13
browser journeys PASS and mobile screenshot inspected. Default Go checks/race tests/
build, generated API/types, 221 web unit tests and production web build PASS.
GitHub Linux validation must pass on the corrected head before merge.
