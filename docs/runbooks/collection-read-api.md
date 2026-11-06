<!-- stage-runbook: true -->
# Private source API stages

## Status and boundary

build/daily-classification; root owns read adapter/server composition. Refs239,
304,305,412. Explicit feature defaults off. No private data/provider/cloud work,
taxonomy, journal, provisioning or default browser activation.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | AGENTS, index, issue dependency/approval state and registry contract reviewed; separate agent lanes own SQL42/PFC/UI |
| Design | collection-read-api contract, frozen registry signatures, source-first authority and ISO applicability reviewed |
| Implementation | Three fixed read functions, bounded repeatable-read, safe retry/commit handling, strict HTTP framing and flag-gated managed composition |
| Focused verification | Race-enabled TestCollection groups PASS in collectionread/server; fake transactions do not prove SQL grants |
| Core/web | make check/build and generated API check PASS; web-check/test/build PASS,417 frontend tests |
| Integration attempts | Full make test attempted: existing acquisition/webhookhost/workerhost/workerruntime socket bind denied. web-browser built fixtures but55tests could not start (EPERM ::1:3100). app-integration stopped at synthetic listener probe; none bypassed |
| Browser follow-up | Direct unchanged npm run test:browser from web PASS55/55; earlier make-wrapper listener failure retained above. Go/app listener restrictions persist |
| Approval reconciliation | ADR0006, source-first and private-categorization status now link accepted GitHub238/242 design; implementation/source/security gates retained |
| Independent review | Separate agent reviewed HTTP/service/server/OpenAPI; nullable currency numeric_code mismatch corrected in schema/types and proved against real42 fixture |
| Real adapter proof | New integration-tagged TestCollectionReadHTTPPostgres PASS with actual synthetic_runtime login: hugeUSD/datedmetadata, nullableEURnumericcode, foreignactor/closedJSONdeny, correction/revision, stale page-one snapshot, workspace revoke, zero books/journals; runtime proof depends on agent42 and is not yet a default CI step |

## Current evidence

Focused synthetic race tests passed with installed Go1.26.5 and external modfile
because tracked1.26.10 download is inaccessible locally. Tracked pins unchanged.
Canonical full checks and real restricted-role integration remain required. No migration42
activation or hosted operation has occurred in this branch.

## Remaining work

Review and integrate migration42, prove adapter against its actual runtime role,
complete default and browser gates, and publish reviewable PR/evidence. Full304
analytic queries and305 command persistence need taxonomy/revision SQL after42.

## PR handoff

Refs239/304/412. This branch adds an optional transport only; SQL42 is a distinct
dependency. No default behavior or cloud/runtime flags changed. Require exact-head
CI and source review before activation. Do not close parent issues from unit tests.
