<!-- stage-runbook: true -->
# Transaction review sorting and frontend audit —406

## Status and boundary

SORT406; feat/406-transaction-sort from merged405/52088ff. One bounded sole SQL39,
API/ReviewInbox/styles/audit lease. Synthetic source only; no Production DOM/data,
provider/cloud/managed apply, existing-account date repair or consent/cap change.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live406/65 and304/305/226/250 read;405 five-green then merged; dedicated worktree/JB-001 packet |
| Design | transaction-review-sort contract records sources, alternatives, standards, private cursor, compatibility and STRIDE before39 DDL |
| Audit | Six steps captured/inspected in reviews/transaction-review-406.md; desktop/mobile list, detail, posted Ledger and explicitly limited demo Connections, plus separate synthetic picker fixture.407/408 opened;226/290/291/404 retained |
| Implementation | Private sorted SQL helper+legacy dispatch, closed paired selectors/cursor, exact amount/currency order, current-fact guard, scoped lookup index and readiness denial; Go/bridge/OpenAPI/types; stable keyboard headers, counts/end/first-page, compact actions/alignment/mobile scroller and clarified empty reports |
| Volume correction | Five-row fixed oracles passed;3000-row cold stats exposed5s timeout. EXPLAIN showed scope-prefix nested scans. Full-key bounded lateral lookups/index passed the same3000-row/30-page oracle in3.51s total, with unchanged5s request bound and no ANALYZE requirement |
| Validation | make check/test/build; source-registry-test; migration-test/ledger-test; locked web-check/web-test/web-build/web-browser (367units/30browser); app-integration, connection-browser-test and production-journey-test all PASS |

## Current evidence

Default UI order is original booked date descending. Every data column toggles a
global SQL order with a unique UUID tie-breaker; old callers without sort/order
retain12's exact legacy result. Cursors contain only scope/snapshot/UUID/selectors,
not money, descriptions or transaction dates. Empty/filter/foreign/revoked/stale/
tampered selectors, direct helper privilege, real differing review states,
JPY/KWD exact originals, ambiguous facts and populated39 down/up are tested.
Reads create0 journals. Existing posted financial functions/history are unchanged.

Local Go1.26.5 uses only the external temporary modfile/cache; tracked1.26.10 stays
unchanged. Official Node24.18.0, locked npm dependencies/browser; no new dependency
or auth adapter in application code. Source review began with bibliography and
database foundations; primary PG17/APG selected sections/limits and STD01–05/
STRIDE live in the contract. Core checks confirm1–38/checksums unchanged.

## Remaining work

Exact-head CI and human review of concrete SQL/API39 before merge/apply. No
managed39 migration or deployment authorized by this source lease. API readiness
requires the new private helper: apply39 before matching runtime promotion; do
not roll frontend alone. Product audit explicitly limits Connections/provider/
accessibility coverage;407/408 are planning only.404 remains separate. No bank
source cap, consent, credential or journal change is part of this work.

## PR handoff

Refs406/226/250/291/304/305/404. Forward39 only;1–38 immutable. Source sorting does
not prove provider coverage, post journals or complete the broader All Transactions epic.
