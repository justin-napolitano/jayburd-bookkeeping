# Sync foundations delivery

<!-- stage-runbook: true -->

## Status and boundary

Build from merged main `29f861a` after #99. Deliver independently testable sync
foundations for #70/#100/#102 and private transport for #7. No provider calls,
secret retrieval, hosted deployment or new account connection is authorized.

## Stage ledger

- Rechecked GitHub issues and merged baseline; claimed three bounded leases in
  #65. SQL, observer transport and exact Plaid decoding have separate worktrees.
- Contracts precede implementation. Coordinator owns cross-contract reconciliation,
  shared verification/composition and this delivery record. UI/enrollment remain
  dependent work; this branch does not fabricate those product flows.
- Integrated decoder `90264ed` and observer transport `cb7f39c`; coordinator added
  the isolated SQL proof target/CI, restricted-role bootstrap/readiness checks and
  read-only UI compatibility for correction holds and posted change references.
- Integrated SQL `8ce437e` as `a463d76`; independent reviews resolved nullable
  status validation, exact-money/timestamp parity and empty initialization bounds.
  Migration 12 is the only new migration; merged migrations 1–11 are unchanged.

## Current evidence

- Decoder defaults/race and two 60-second fuzz campaigns passed: 4,353,639 money
  executions and 2,483,899 page executions, with independent rational-value checks.
- Observer defaults/race and 44 disposable PostgreSQL proof groups passed. The
  composed HTTP→publisher→injected Tasks transport→HTTP→consumer proof records one
  receipt after lost commit/queue acknowledgements and duplicate delivery, with no
  financial mutation. Public-key response-size review finding fixed and retested.
- Shared frontend check/build, 184 unit tests and 13 browser journeys passed
  (12 regression journeys plus the new posted-source exception journey). Inspected
  the mobile exception screenshot: original amount retained, change references
  separate, no category or draft action. Go observer/decoder/ledger race checks pass.
- Combined `make check test build migration-test cloud-sql-test ledger-test
  observer-test source-registry-test import-service-test sync-state-test` passed.
  Sync proof includes 15 PostgreSQL groups and 10 subtests: current authority,
  concurrent wakeups, rollback, limits, multicurrency, decoder-to-SQL publication
  and repeated corrections after actual source-linked posting.
- `make app-integration` passed the production Next bridge → Go → PostgreSQL
  recovery journey. Workflow actionlint and generated API type checks passed.
- `make demo-test` passed all four journeys: 3,000 generated imports, one explicit
  source-linked posting, cents/recovery checks and 1,000 committed deposits whose
  PostgreSQL total is exactly USD 495.55. This is not provider acquisition proof.
  Temporary loopback ports 39781–39783 were restored after tests; existing owner
  previews were left running. No test-port changes are included in the delivery.

## Remaining work

Account connection/credential lifecycle (#101), acquisition runner (#102), verified
webhooks (#103), connected UI (#104) and hosted worker composition (#7) remain.
Synthetic enrollment attaches existing sources and holds CSV/provider overlap;
overlap release and correction resolution are not implemented. Exception UI exposes
revision references, not a full comparison of changed financial values. Publication
is capped at 10 pages / 1,000 changes / 2 MiB; no linear scaling claim. #85/#87–#90
retain independent fault, security, Sandbox, measured scaling and live-use gates.

## PR handoff

Ready for PR review against main. Broader #7/#70/#100/#102 acceptance remains
open for merge/review or remaining scope; partial foundations are labeled as such.
No provider calls, cloud deployment or live accounts were used in this delivery.
