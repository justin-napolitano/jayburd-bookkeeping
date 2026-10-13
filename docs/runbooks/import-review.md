# Import review prototype

<!-- stage-runbook: true -->

## Status and boundary

Synthetic interactive design for #52/#55 alongside #8 durable-contract work.
Root owns this lane because the session could not allocate a third agent thread.
No real upload, storage, provider or posting behavior.

## Stage ledger

- Reviewed statement profile, decoder and existing workspace/accessibility sources.
- Coordinated exact review states with import-contract agent before implementation.
- Added standalone review prototype with scope clearing, duplicate/hold distinctions,
  exact original-unit strings and explicitly simulated USD draft preparation.

## Current evidence

PASS: local Chromium interaction review (2026-10-13) exercised two distinct
USD 12.50 purchases, repeat suppression, overlap holds and changed-CSV identity rejection, explicit draft
preparation, scope reset, malformed/denied clearing, JPY 125 and KWD 1.005 strings,
and foreign-draft gating. Desktop (1440px) and mobile (390px) screenshots inspected;
mobile table scrolls inside its panel, with no document-width overflow.
Local evidence: `/private/tmp/jayburd-import-{desktop,mobile}.png`; these are
review artifacts, not CI dependencies. [Journey contract](../plans/import-review.md) records
expected behaviors and deferred production gates.

## Remaining work

Future runtime must enforce every authority boundary. Production keyboard,
screen-reader and upload/recovery journeys remain unimplemented.

## PR handoff

Design artifact only; no API or DDL. Root combines validation in the branch runbook.
