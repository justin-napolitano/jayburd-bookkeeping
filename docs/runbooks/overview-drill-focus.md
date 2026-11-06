<!-- stage-runbook: true -->
# Overview committed drill focus —442

## Status and boundary

FOCUS442, fix/overview-drill-focus from2315cc4, dedicated jayburd-ci-budget
worktree. Only existing Overview focus behavior, independent synthetic unit
regression and this evidence record. No data/API/SQL/amount/grant/provider,
dependency, cloud, Production, consent or source-scope change. PFC PR441 remains
feature-only; this separate fix addresses its unrelated existing browser failure.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume |PR441 frontend job128126597009/run42285924204:54/55 browser journeys passed;390px exact drill heading rendered but inactive. Issue442 records scope;65 sole focused UI writer claim |
| Diagnosis |readDetail scheduled frame focus immediately after async setDetail. A frame can run before React commits the new heading/ref, permanently losing focus |
| Red regression |New synthetic unit invokes the frame callback before the state commit. Original code fails with body active instead of the rendered merchant heading;6 existing Overview units pass |
| Repair |Post-commit effect focuses the rendered current detail heading; explicit close records current epoch and restores summary after commit. Old-scope restore cannot survive a changed epoch. Existing browser assertions/retries/timeouts unchanged |
| Green regression |All7 Overview unit tests PASS after repair, including merchant→transaction→back→explicit-close focus with earliest-frame scheduling |
| Validation |Locked web-check/web-test/web-build PASS with418 units; unchanged full web-browser suite55/55 PASS, including390px/1440px drills. Core make check/build PASS using installed Go1.26.5/external modfile/cache; full make test attempted but existing Go network packages cannot bind sockets in the local sandbox. No skip or false full-suite claim |

## Current evidence

Existing source Overview wire, filtering, snapshot/amount formatting, authorization
and recovery are unchanged. This fixes a keyboard-navigation cue, not a financial
operation or ISO message. No new security boundary, metadata or persistence exists.

React's official [useEffect reference](https://react.dev/reference/react/useEffect),
setup-after-commit and dependency sections reviewed2026-11-06, documents the
commit-aware mechanism. Our inference is to focus the current rendered ref there,
instead of assuming a browser frame implies a React commit. Independent red/green
evidence tests that timing assumption; it is not an arbitrary wait or test retry.

The separate PR440 integration timeout was traced to apt fonts download17.9MB
in12m37, leaving insufficient time under the20m job deadline. Its rerun installed
browser/dependencies in22s. No workflow timeout/cache change is part of this patch;
job completion and repeatability are tracked independently before proposing one.

## Remaining work

Publish exact-head source/CI evidence and obtain source review. Local full browser
checks passed; hosted canonical toolchain/SQL/application checks remain required.
No deployment or financial integration acceptance follows from this UI fix.

## PR handoff

Refs442/441/432/418/412. No migration or wire impact. Keep source review and exact
head checks; no scan/retry/timeout/permission/pin weakening. Root owns index and
final integration; no overlapping collection UI files were edited.
